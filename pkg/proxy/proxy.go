package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/renproject/multichain-proxy/pkg/authorization"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type JSONRPCResponse struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type Config struct {
	Logger   *zap.Logger
	NodeURL  *url.URL
	NodeCred authorization.Credentials // credentials to authorize with node
	Body     []byte
}

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
		NodeURL: nURL,
		NodeCred: authorization.Credentials{
			JWT:      os.Getenv(fmt.Sprintf("NODE%v_TOKEN", nodeID)),
			Username: os.Getenv(fmt.Sprintf("NODE%v_USER", nodeID)),
			Password: os.Getenv(fmt.Sprintf("NODE%v_PASSWORD", nodeID)),
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
