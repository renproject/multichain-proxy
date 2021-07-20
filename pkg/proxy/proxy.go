package proxy

import (
	"errors"
	"github.com/renproject/multichain-proxy/pkg/authorization"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"os"
)

type Config struct {
	Logger   *zap.Logger
	NodeURL  *url.URL
	NodeCred authorization.Credentials // credentials to authorize with node
}

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
		NodeURL: nURL,
		NodeCred: authorization.Credentials{
			JWT:      os.Getenv("NODE_TOKEN"),
			Username: os.Getenv("NODE_USER"),
			Password: os.Getenv("NODE_PASSWORD"),
		},
	}, nil
}

func (conf *Config) ProxyDirector(req *http.Request) {
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
