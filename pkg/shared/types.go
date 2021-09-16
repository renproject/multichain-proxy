package shared

import "github.com/renproject/multichain-proxy/pkg/authorization"

type ProxyConfig struct {
	Url string `json:"url"`
	authorization.Credentials
}

type ProxyConfigDB struct {
	Key   string      `bson:"key"`
	Value ProxyConfig `bson:"value"`
}
