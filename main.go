package main

import (
	"bytes"
	"fmt"
	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/proxy"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"io/ioutil"
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
	auth := authorization.NewAuthorizer(logger)
	conf1, err := proxy.NewConfig(logger, "1")
	if err != nil {
		logger.Fatal("failed to create proxy-1", zap.Error(err))
	}
	conf2, err := proxy.NewConfig(logger, "2")
	if err != nil {
		logger.Fatal("failed to create proxy-2", zap.Error(err))
	}
	logger.Info("starting proxy")
	defer logger.Info("stopping proxy")

	proxyServer2 := &httputil.ReverseProxy{Director: conf2.ProxyDirector}
	proxyServer1 := &httputil.ReverseProxy{
		Director:       conf1.ProxyDirector,
		ModifyResponse: conf1.ModifyResponse,
		ErrorHandler: func(writer http.ResponseWriter, r *http.Request, err error) {
			logger.Error("node1 failed to respond", zap.Error(err))
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
		}).Handler(auth.AuthorizeProxy(proxyServer1)),
	}
	httpServer.SetKeepAlivesEnabled(false)

	httpServer.ListenAndServe()
}
