package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/util"
	"go.uber.org/zap"
)

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
}

// ProxyConfig handles admin request to update proxy configs
func (conf *Config) ProxyConfig(w http.ResponseWriter, r *http.Request) {
	conf.Lock.Lock()
	defer conf.Lock.Unlock()
	var payload authorization.ProxyConfig
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		conf.Logger.Debug("payload decode failed")
		if err := util.WriteError(w, -1, fmt.Errorf("malformed payload")); err != nil {
			conf.Logger.Error("error writing response", zap.Error(err))
		}
		return
	}
	conf.Logger.Info("test", zap.Any("payload", payload.Credentials))
	nURL, err := url.Parse(payload.Path)
	if err != nil {
		conf.Logger.Debug("invalid node url")
		if err := util.WriteError(w, -1, fmt.Errorf("invalid node url, %v", payload.Path)); err != nil {
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
