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
	conf1, err := proxy.NewConfig(logger, "1", db, false)
	if err != nil {
		logger.Fatal("failed to create proxy-1", zap.Error(err))
	}

	// proxy for node 2
	conf2, err := proxy.NewConfig(logger, "2", db, false)
	if err != nil {
		logger.Fatal("failed to create proxy-2", zap.Error(err))
	}

	// proxy for local node 1
	localConf1, err := proxy.NewConfig(logger, "1", db, true)
	if err != nil {
		logger.Fatal("failed to create proxy-1", zap.Error(err))
	}

	// proxy for local node 2
	localConf2, err := proxy.NewConfig(logger, "2", db, true)
	if err != nil {
		logger.Fatal("failed to create proxy-2", zap.Error(err))
	}
	logger.Info("starting proxy")
	defer logger.Info("stopping proxy")

	proxyServer2 := &httputil.ReverseProxy{Director: conf2.ProxyDirector}

	localServer1 := &httputil.ReverseProxy{Director: localConf1.ProxyDirector}
	localServer2 := &httputil.ReverseProxy{Director: localConf2.ProxyDirector}

	errorChan := make(chan error, 1)

	// setup node 1 proxy to forward request to node 2 in case of error response
	proxyServer1 := &httputil.ReverseProxy{
		Director:       conf1.ProxyDirector,
		ModifyResponse: conf1.ModifyResponse,
		ErrorHandler: func(writer http.ResponseWriter, r *http.Request, err error) {
			logger.Error("node1 failed to respond", zap.Error(err))

			// known http2 error, cannot recover once this error is hit. only solution is to restart pod
			if strings.Contains(err.Error(), "after Request.Body was written; define Request.GetBody to avoid this error") {
				errorChan <- err
			}

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
		}).Handler(auth.AuthorizeProxy(proxyServer1, localServer1, localServer2, http.HandlerFunc(conf1.ProxyConfig), http.HandlerFunc(conf2.ProxyConfig), http.HandlerFunc(localConf1.ProxyConfig), http.HandlerFunc(localConf2.ProxyConfig))),
	}
	httpServer.SetKeepAlivesEnabled(false)

	// http2 error handler, force restart the pod
	go func() {
		err := <-errorChan
		panic(fmt.Errorf("irrecoverable error, restarting pod. error=%w", err))
	}()

	httpServer.ListenAndServe()
}
