package main

import (
	"fmt"
	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/proxy"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("PROXY_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	// create auth middleware
	auth := authorization.NewAuthorizer(logger)

	// create node proxy
	conf, err := proxy.NewConfig(logger)
	if err != nil {
		logger.Fatal("failed to create proxy", zap.Error(err))
	}

	logger.Info("starting proxy")
	defer logger.Info("stopping proxy")

	// setup reverse proxy for the node
	proxyServer := &httputil.ReverseProxy{Director: conf.ProxyDirector}

	httpServer := http.Server{
		Addr:              fmt.Sprintf("%v:%v", host, port),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		Handler: cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedMethods:   []string{"POST", "GET"},
		}).Handler(auth.AuthorizeProxy(proxyServer)),
	}
	httpServer.SetKeepAlivesEnabled(false)

	httpServer.ListenAndServe()
}
