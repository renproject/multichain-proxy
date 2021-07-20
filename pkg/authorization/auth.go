package authorization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/renproject/multichain-proxy/pkg/util"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
)

type Credentials struct {
	JWT      string `json:"jwt,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Authorizer struct {
	Credentials
	Logger     *zap.Logger     `json:"-,omitempty"`
	MaxReqSize int64           `json:"maxReqSize"`
	Methods    map[string]bool `json:"methods"`
	Paths      map[string]bool `json:"paths"`
}

type JSONRPCRequest struct {
	Version string          `json:"version"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// NewAuthorizer creates the proxy authorization object from the provided env vars
func NewAuthorizer(logger *zap.Logger) *Authorizer {
	return &Authorizer{
		MaxReqSize: 16 * 1024,
		Logger:     logger,
		Credentials: Credentials{
			JWT:      os.Getenv("PROXY_TOKEN"),
			Username: os.Getenv("PROXY_USER"),
			Password: os.Getenv("PROXY_PASSWORD"),
		},
		Methods: util.ConvertEnv2Map("PROXY_METHODS"),
		Paths:   util.ConvertEnv2Map("PROXY_PATHS"),
	}
}

// AuthorizeProxy middleware authorizes all the rpc calls
func (auth *Authorizer) AuthorizeProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if path is allowed, if path map is nil allow all paths
		if auth.Paths != nil && !auth.Paths[r.URL.EscapedPath()] {
			if err := util.WriteError(w, -1, fmt.Errorf("path not allowed %v", r.URL.EscapedPath())); err != nil {
				auth.Logger.Error("error writing response", zap.Error(err))
			}
			return
		}

		// check if valid proxy credentials
		if err := auth.credentialCheck(r); err != nil {
			if err = util.WriteError(w, -1, err); err != nil {
				auth.Logger.Error("error writing response", zap.Error(err))
			}
			return
		}

		// check if valid rpc function
		if err := auth.whitelistCheck(w, r); err != nil {
			if err = util.WriteError(w, -1, err); err != nil {
				auth.Logger.Error("error writing response", zap.Error(err))
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// credentialCheck verifies the proxy credentials if any
func (auth *Authorizer) credentialCheck(r *http.Request) error {
	if auth.JWT != "" && r.Header.Get("Authorization") != auth.JWT {
		return errors.New("request not authorized")
	} else if auth.Username != "" || auth.Password != "" {
		user, pass, ok := r.BasicAuth()
		if !ok {
			return errors.New("request not authorized")
		}
		if user != auth.Username || pass != auth.Password {
			return errors.New("request not authorized")
		}
	}
	return nil
}

// whitelistCheck verifies whether the rpc function is allowed or not
func (auth *Authorizer) whitelistCheck(w http.ResponseWriter, r *http.Request) error {

	// Restrict the maximum request body size. There should be no reason to
	// allow requests of unbounded size.
	sizeLimitedBody := http.MaxBytesReader(w, r.Body, auth.MaxReqSize)
	defer func() {
		if err := sizeLimitedBody.Close(); err != nil {
			auth.Logger.Error("error closing http request body", zap.Error(err))
		}
	}()

	raw, err := ioutil.ReadAll(sizeLimitedBody)
	if err != nil {
		auth.Logger.Error("error reading size limited body", zap.Error(err))
		return errors.New("invalid size limited body")
	}

	jrpcReq := JSONRPCRequest{}
	if err := json.Unmarshal(raw, &jrpcReq); err != nil {
		auth.Logger.Error("error unmarshaling", zap.Error(err))
		return errors.New("invalid json")
	}

	// verify if method is allowed, if method map is nil allow all methods
	if auth.Methods != nil && !auth.Methods[jrpcReq.Method] {
		auth.Logger.Error("method not allowed", zap.String("method", jrpcReq.Method))
		return fmt.Errorf("method not allow: %v", jrpcReq.Method)
	}
	buf := bytes.NewBuffer(raw)
	r.Body = ioutil.NopCloser(buf)
	return nil
}
