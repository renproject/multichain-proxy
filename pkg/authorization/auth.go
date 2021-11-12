package authorization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/renproject/multichain-proxy/pkg/util"
	"go.uber.org/zap"
)

type Credentials struct {
	JWT      string `json:"jwt,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Authorizer struct {
	Credentials
	Logger           *zap.Logger     `json:"-,omitempty"`
	MaxReqSize       int64           `json:"maxReqSize"`
	Methods          map[string]bool `json:"methods"`
	Paths            map[string]bool `json:"paths"`
	ConfigPath1      string          `json:"configPath1"`
	ConfigPath2      string          `json:"configPath2"`
	LocalNodePath    string          `json:"localPath"`
	ConfigCredential Credentials     `json:"configCredential"`
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
		Methods:       util.ConvertEnv2Map("PROXY_METHODS"),
		Paths:         util.ConvertEnv2Map("PROXY_PATHS"),
		ConfigPath1:   os.Getenv("CONFIG_PATH_1"),
		ConfigPath2:   os.Getenv("CONFIG_PATH_2"),
		LocalNodePath: os.Getenv("LOCAL_NODE_PATH"),
		ConfigCredential: Credentials{
			JWT:      os.Getenv("CONFIG_TOKEN"),
			Username: os.Getenv("CONFIG_USER"),
			Password: os.Getenv("CONFIG_PASSWORD"),
		},
	}
}

// AuthorizeProxy middleware authorizes all the rpc calls
func (auth *Authorizer) AuthorizeProxy(next http.Handler, def1 http.Handler, def2 http.Handler, renProxy1 http.Handler, renProxy2 http.Handler, renLocalProxy1 http.Handler, renLocalProxy2 http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.ConfigPath1 == r.URL.EscapedPath() || auth.ConfigPath2 == r.URL.EscapedPath() {
			if err := auth.credentialCheck(r, auth.ConfigCredential); err != nil {
				auth.Logger.Debug("auth[config-path] check", zap.Error(err))
				if err = util.WriteError(w, -1, err); err != nil {
					auth.Logger.Error("error writing response", zap.Error(err))
				}
				return
			}
			if auth.ConfigPath1 == r.URL.EscapedPath() {
				renProxy1.ServeHTTP(w, r)
			} else {
				renProxy2.ServeHTTP(w, r)
			}
			return
		}

		if auth.LocalNodePath+auth.ConfigPath1 == r.URL.EscapedPath() || auth.LocalNodePath+auth.ConfigPath2 == r.URL.EscapedPath() {
			if err := auth.credentialCheck(r, auth.ConfigCredential); err != nil {
				auth.Logger.Debug("auth[config-path-local] check", zap.Error(err))
				if err = util.WriteError(w, -1, err); err != nil {
					auth.Logger.Error("error writing response", zap.Error(err))
				}
				return
			}
			if auth.LocalNodePath+auth.ConfigPath1 == r.URL.EscapedPath() {
				renLocalProxy1.ServeHTTP(w, r)
			} else {
				renLocalProxy2.ServeHTTP(w, r)
			}
			return
		}

		// check if path is allowed, if path map is nil allow all paths
		if auth.Paths != nil && !auth.Paths[r.URL.EscapedPath()] {
			auth.Logger.Debug("requested path not allowed", zap.String("path", r.URL.EscapedPath()))
			if err := util.WriteError(w, -1, fmt.Errorf("path not allowed %v", r.URL.EscapedPath())); err != nil {
				auth.Logger.Error("error writing response", zap.Error(err))
			}
			return
		}

		// check if valid proxy credentials
		if err := auth.credentialCheck(r, auth.Credentials); err != nil {
			auth.Logger.Debug("auth check", zap.Error(err))
			if err = util.WriteError(w, -1, err); err != nil {
				auth.Logger.Error("error writing response", zap.Error(err))
			}
			return
		}

		// check if valid rpc method
		requestBody, err := auth.whitelistCheck(w, r)
		auth.Logger.Debug("request info", zap.Any("headers", r.Header), zap.String("requestBody", requestBody), zap.String("path", r.URL.EscapedPath()))
		if err != nil {
			auth.Logger.Debug("method check", zap.Error(err))
			if err = util.WriteError(w, -1, err); err != nil {
				auth.Logger.Error("error writing response", zap.Error(err))
			}
			return
		}

		if strings.HasPrefix(r.URL.EscapedPath(), auth.LocalNodePath+"/1") {
			r.URL.Path = strings.TrimLeft(r.URL.Path, auth.LocalNodePath+"/1")
			if !strings.HasPrefix(r.URL.Path, "/") {
				r.URL.Path = "/" + r.URL.Path
			}
			auth.Logger.Debug("proxy to local node-1")
			def1.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.EscapedPath(), auth.LocalNodePath+"/2") {
			r.URL.Path = strings.TrimLeft(r.URL.Path, auth.LocalNodePath+"/2")
			if !strings.HasPrefix(r.URL.Path, "/") {
				r.URL.Path = "/" + r.URL.Path
			}
			auth.Logger.Debug("proxy to local node-2")
			def2.ServeHTTP(w, r)
		} else {
			auth.Logger.Debug("proxy to config node")
			next.ServeHTTP(w, r)
		}
	})
}

// credentialCheck verifies the proxy credentials if any
func (auth *Authorizer) credentialCheck(r *http.Request, cred Credentials) error {
	if cred.JWT != "" {
		if r.Header.Get("Authorization") != cred.JWT {
			return errors.New("request not authorized")
		}
	} else if cred.Username != "" || cred.Password != "" {
		user, pass, ok := r.BasicAuth()
		if !ok {
			return errors.New("request not authorized")
		}
		if user != cred.Username || pass != cred.Password {
			return errors.New("request not authorized")
		}
	}
	return nil
}

// whitelistCheck verifies whether the rpc function is allowed or not
func (auth *Authorizer) whitelistCheck(w http.ResponseWriter, r *http.Request) (string, error) {

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
		return "", errors.New("invalid size limited body")
	}

	jrpcReq := JSONRPCRequest{}
	if err := json.Unmarshal(raw, &jrpcReq); err != nil {
		return string(raw), errors.New("invalid json")
	}

	// verify if method is allowed, if method map is nil allow all methods
	if auth.Methods != nil && !auth.Methods[jrpcReq.Method] {
		return string(raw), fmt.Errorf("method not allow: %v", jrpcReq.Method)
	}
	buf := bytes.NewBuffer(raw)
	r.Body = ioutil.NopCloser(buf)
	return string(raw), nil
}
