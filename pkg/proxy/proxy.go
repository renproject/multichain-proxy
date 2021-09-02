package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/util"
	"go.uber.org/zap"
)

type ProxyConfig struct {
	Url string `json:"url"`
	authorization.Credentials
}

type Config struct {
	Logger   *zap.Logger
	Lock     *sync.RWMutex
	NodeURL  *url.URL
	NodeCred authorization.Credentials // credentials to authorize with node
}

// NewConfig creates a new proxy config from the given env vars
func NewConfig(logger *zap.Logger) (*Config, error) {
	nodeURL := os.Getenv("NODE_URL")
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
			JWT:      os.Getenv("NODE_TOKEN"),
			Username: os.Getenv("NODE_USER"),
			Password: os.Getenv("NODE_PASSWORD"),
		},
	}, nil
}

// ProxyDirector handles how the request is proxied to the target node and does modifications to the request payload as required
func (conf *Config) ProxyDirector(req *http.Request) {
	conf.Lock.RLock()
	defer conf.Lock.RUnlock()
	conf.Logger.Debug("proxy data at start", zap.Any("request", req.Host), zap.Any("request-url", req.URL), zap.Any("request-url", req.URL.RawPath), zap.Any("request-url", req.URL.Path), zap.Any("node", conf.NodeURL.Host), zap.Any("node-url", conf.NodeURL))

	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("X-Origin-Host", conf.NodeURL.Host)
	req.Host = conf.NodeURL.Host
	req.URL.Scheme = conf.NodeURL.Scheme
	req.URL.Host = conf.NodeURL.Host
	req.URL.Path = strings.TrimRight(conf.NodeURL.Path, "/") + strings.TrimRight(req.URL.Path, "/")

	conf.Logger.Debug("proxy data after modification", zap.Any("request", req.Host), zap.Any("request-url", req.URL), zap.Any("request-url", req.URL.RawPath), zap.Any("request-url", req.URL.Path), zap.Any("node", conf.NodeURL.Host), zap.Any("node-url", conf.NodeURL))

	req.Header.Del("Authorization")
	if conf.NodeCred.JWT != "" {
		req.Header.Set("Authorization", conf.NodeCred.JWT)
	} else if conf.NodeCred.Username != "" || conf.NodeCred.Password != "" {
		req.SetBasicAuth(conf.NodeCred.Username, conf.NodeCred.Password)
	}
}

// ProxyConfig handles admin request to update proxy configs
func (conf *Config) ProxyConfig(w http.ResponseWriter, r *http.Request) {
	conf.Lock.Lock()
	defer conf.Lock.Unlock()
	if r.Method == "GET" {
		if err := util.WriteResponse(w, 1, ProxyConfig{
			Url:         conf.NodeURL.String(),
			Credentials: conf.NodeCred,
		}); err != nil {
			conf.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}
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
		conf.Logger.Debug("empty url sent setting to default values")
		payload.Url = os.Getenv("NODE_URL")
		payload.Credentials = authorization.Credentials{
			JWT:      os.Getenv("NODE_TOKEN"),
			Username: os.Getenv("NODE_USER"),
			Password: os.Getenv("NODE_PASSWORD"),
		}
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
	conf.Logger.Debug("proxy config update", zap.Any("url", conf.NodeURL), zap.Any("cred", conf.NodeCred))
}
