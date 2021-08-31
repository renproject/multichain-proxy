package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/util"
	"go.uber.org/zap"
)

type JSONRPCResponse struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type ProxyConfig struct {
	Url string `json:"url"`
	authorization.Credentials
}

type Config struct {
	Logger   *zap.Logger
	Lock     *sync.RWMutex
	NodeURL  *url.URL
	NodeCred authorization.Credentials // credentials to authorize with node
	Body     []byte
}

// NewConfig creates a new proxy config from the given env vars
func NewConfig(logger *zap.Logger, nodeID string) (*Config, error) {
	nodeURL := os.Getenv(fmt.Sprintf("NODE%v_URL", nodeID))
	if nodeURL == "" {
		return nil, errors.New("missing node url")
	}
	nURL, err := url.Parse(nodeURL)
	if err != nil {
		return nil, errors.New("invalid node url")
	}
	return &Config{
		Logger:  logger,
		Lock:    &sync.RWMutex{},
		NodeURL: nURL,
		NodeCred: authorization.Credentials{
			JWT:      os.Getenv(fmt.Sprintf("NODE%v_TOKEN", nodeID)),
			Username: os.Getenv(fmt.Sprintf("NODE%v_USER", nodeID)),
			Password: os.Getenv(fmt.Sprintf("NODE%v_PASSWORD", nodeID)),
		},
	}, nil
}

// ProxyDirector handles how the request is proxied to the target node and does modifications to the request payload as required
func (conf *Config) ProxyDirector(req *http.Request) {
	conf.Lock.RLock()
	defer conf.Lock.RUnlock()
	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("X-Origin-Host", conf.NodeURL.Host)
	req.Host = conf.NodeURL.Host
	req.URL.Scheme = conf.NodeURL.Scheme
	req.URL.Host = conf.NodeURL.Host

	if conf.NodeCred.JWT != "" {
		req.Header.Set("Authorization", conf.NodeCred.JWT)
	} else if conf.NodeCred.Username != "" || conf.NodeCred.Password != "" {
		req.SetBasicAuth(conf.NodeCred.Username, conf.NodeCred.Password)
	}
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		conf.Body = nil
		conf.Logger.Error("failed to get body", zap.Error(err))
		return
	}
	conf.Body = reqBody
	buf := bytes.NewBuffer(reqBody)
	req.Body = ioutil.NopCloser(buf)
}

// ModifyResponse verifies the response from the node and throws errors to invoke the proxy ErrorHandler in case error response is returned from node
func (conf *Config) ModifyResponse(r *http.Response) error {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}
	jrpcRes := JSONRPCResponse{}
	if err = json.Unmarshal(bodyBytes, &jrpcRes); err != nil {
		return fmt.Errorf("error unmarshaling response body: %v", err)
	}
	if jrpcRes.Error != nil {
		return fmt.Errorf("jsonrpc response error: %v", jrpcRes.Error)
	}
	buf := bytes.NewBuffer(bodyBytes)
	r.Body = ioutil.NopCloser(buf)
	return nil
}

// ProxyConfig handles admin request to update proxy configs
func (conf *Config) ProxyConfig(w http.ResponseWriter, r *http.Request) {
	conf.Lock.Lock()
	defer conf.Lock.Unlock()
	var payload ProxyConfig
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		conf.Logger.Debug("payload decode failed")
		if err := util.WriteError(w, -1, fmt.Errorf("malformed payload")); err != nil {
			conf.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}
	if payload.Url == "" {
		conf.Logger.Debug("empty node url")
		if err := util.WriteError(w, -1, fmt.Errorf("empty node url")); err != nil {
			conf.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}
	nURL, err := url.Parse(payload.Url)
	if err != nil {
		conf.Logger.Debug("invalid node url")
		if err := util.WriteError(w, -1, fmt.Errorf("invalid node url, url=%v, error=%w", payload.Url, err)); err != nil {
			conf.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}
	conf.NodeURL = nURL
	conf.NodeCred = payload.Credentials
	if err := util.WriteResponse(w, 1, "successfully updated"); err != nil {
		conf.Logger.Error("error writing response", zap.Error(err))
	}
}
