package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/rs/cors"
	"go.uber.org/zap"
)

type WhitelistOptions struct {
	Logger         *zap.Logger     `json:"-,omitempty"`
	URL            string          `json:"url"`
	AuthTokenProxy string          `json:"authTokenProxy"`
	AuthToken      string          `json:"authToken"`
	User           string          `json:"user"`
	Password       string          `json:"password"`
	Methods        map[string]bool `json:"methods"`
	Host           string          `json:"host"`
	Port           uint16          `json:"port"`
	MaxReqSize     int64           `json:"maxReqSize"`
	NumWorkers     int             `json:"numWorkers"`
}

type WhitelistJob struct {
	done chan struct{}
	rw   http.ResponseWriter
	req  *http.Request
}

type Whitelist struct {
	opts WhitelistOptions
	jobs chan WhitelistJob
}

func NewWhitelist(opts WhitelistOptions) *Whitelist {
	return &Whitelist{
		opts: opts,
		jobs: make(chan WhitelistJob, opts.NumWorkers),
	}
}

func (p *Whitelist) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	done := make(chan struct{})
	job := WhitelistJob{
		done: done,
		rw:   rw,
		req:  req,
	}
	select {
	case <-req.Context().Done():
		return
	case p.jobs <- job:
		select {
		case <-req.Context().Done():
		case <-done:
		}
	}
}

func (p *Whitelist) Run(ctx context.Context) {
	bind := fmt.Sprintf("%v:%v", p.opts.Host, p.opts.Port)
	httpServer := http.Server{
		Addr:              bind,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		Handler: cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedMethods:   []string{"POST", "GET"},
		}).Handler(p),
	}
	httpServer.SetKeepAlivesEnabled(false)

	p.opts.Logger.Info("listening for http connections", zap.String("bind", bind))
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				p.opts.Logger.Error("error listening for http connections", zap.String("bind", bind), zap.Error(err))
			}
		}
	}()
	defer func() {
		if err := httpServer.Close(); err != nil {
			p.opts.Logger.Error("error closing http server", zap.String("bind", bind), zap.Error(err))
		}
	}()

	p.opts.Logger.Info("running", zap.Int("num_workers", p.opts.NumWorkers))
	var wg sync.WaitGroup
	wg.Add(p.opts.NumWorkers)
	for i := 0; i < p.opts.NumWorkers; i++ {
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-p.jobs:
					p.do(job)
				}
			}
		}(i)
	}
	wg.Wait()
}

func (p *Whitelist) do(job WhitelistJob) {
	defer close(job.done)

	// check if the auth header has been set correctly
	if job.req.Header.Get("Authorization") != p.opts.AuthTokenProxy {
		if err := WriteError(job.rw, -1, fmt.Errorf("request not authorized")); err != nil {
			p.opts.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}

	// Restrict the maximum request body size. There should be no reason to
	// allow requests of unbounded size.
	sizeLimitedBody := http.MaxBytesReader(job.rw, job.req.Body, p.opts.MaxReqSize)
	defer func() {
		if err := sizeLimitedBody.Close(); err != nil {
			p.opts.Logger.Error("error closing http request body", zap.Error(err))
		}
	}()

	raw, err := ioutil.ReadAll(sizeLimitedBody)
	if err != nil {
		p.opts.Logger.Error("error reading size limited body", zap.Error(err))
		if err := WriteError(job.rw, -1, fmt.Errorf("invalid size limited body")); err != nil {
			p.opts.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}

	jrpcReq := JSONRPCRequest{}
	if err := json.Unmarshal(raw, &jrpcReq); err != nil {
		p.opts.Logger.Error("error unmarshaling", zap.Error(err))
		if err := WriteError(job.rw, jrpcReq.ID, fmt.Errorf("invalid json")); err != nil {
			p.opts.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}

	if !p.opts.Methods[jrpcReq.Method] {
		p.opts.Logger.Error("method not allowed", zap.String("method", jrpcReq.Method))
		if err := WriteError(job.rw, jrpcReq.ID, fmt.Errorf("method not allow: %v", jrpcReq.Method)); err != nil {
			p.opts.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}

	res1, resBody1, err := p.processRequest(p.opts.URL, p.opts.AuthToken, raw)
	if err != nil {
		p.opts.Logger.Error("processing request", zap.Error(err), zap.String("url", p.opts.URL))
		if err = WriteError(job.rw, jrpcReq.ID, fmt.Errorf("bad proxy request")); err != nil {
			p.opts.Logger.Error("writing response", zap.Error(err))
		}
		return
	}

	defer res1.Body.Close()
	CopyResponse(job.rw, res1, resBody1)
}

func (p *Whitelist) processRequest(url, authToken string, raw []byte) (*http.Response, []byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("building http request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Del("Authorization")
	req.Header.Add("Authorization", authToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("sending http request: %v", err)
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response body: %v", err)
	}

	jrpcRes := JSONRPCResponse{}
	if err := json.Unmarshal(bodyBytes, &jrpcRes); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling response body: %v", err)
	}

	if jrpcRes.Error != nil {
		return nil, nil, fmt.Errorf("jsonrpc response error: %v", jrpcRes.Error)
	}

	return res, bodyBytes, nil
}
