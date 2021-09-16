package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/renproject/multichain-proxy/pkg/database"

	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/proxy"
	"github.com/rs/cors"
	"go.uber.org/zap"
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

	if strings.ToLower(os.Getenv("DEV_MODE")) == "true" {
		logger, err = zap.NewDevelopment()
		if err != nil {
			log.Fatal(err)
		}
	}

	// create auth middleware
	auth := authorization.NewAuthorizer(logger)

	// create db instance
	db, err := database.NewDBManager()
	if err != nil {
		logger.Fatal("failed to create db instance", zap.Error(err))
	}

	// proxy for node 1
	conf1, err := proxy.NewConfig(logger, "1", db)
	if err != nil {
		logger.Fatal("failed to create proxy-1", zap.Error(err))
	}

	// proxy for node 2
	conf2, err := proxy.NewConfig(logger, "2", db)
	if err != nil {
		logger.Fatal("failed to create proxy-2", zap.Error(err))
	}
	logger.Info("starting proxy")
	defer logger.Info("stopping proxy")

	proxyServer2 := &httputil.ReverseProxy{Director: conf2.ProxyDirector}

	// setup node 1 proxy to forward request to node 2 in case of error response
	proxyServer1 := &httputil.ReverseProxy{
		Director:       conf1.ProxyDirector,
		ModifyResponse: conf1.ModifyResponse,
		ErrorHandler: func(writer http.ResponseWriter, r *http.Request, err error) {
			logger.Error("node1 failed to respond", zap.Error(err))

			// update the request body for node 2 to the original request
			buf := bytes.NewBuffer(conf1.Body)
			r.Body = ioutil.NopCloser(buf)

			proxyServer2.ServeHTTP(writer, r)
		}}

	httpServer := http.Server{
		Addr:              fmt.Sprintf("%v:%v", host, port),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		Handler: cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedMethods:   []string{"POST", "GET"},
		}).Handler(auth.AuthorizeProxy(proxyServer1, http.HandlerFunc(conf1.ProxyConfig), http.HandlerFunc(conf2.ProxyConfig))),
	}
	httpServer.SetKeepAlivesEnabled(false)

	httpServer.ListenAndServe()
}
