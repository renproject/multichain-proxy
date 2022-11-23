package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/renproject/multichain-proxy/pkg/database"

	"github.com/renproject/multichain-proxy/pkg/authorization"
	"github.com/renproject/multichain-proxy/pkg/proxy"
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

	// create node proxy
	conf, err := proxy.NewConfig(logger, db, false)
	if err != nil {
		logger.Fatal("failed to create proxy", zap.Error(err))
	}

	// create local node proxy
	localConf, err := proxy.NewConfig(logger, db, true)
	if err != nil {
		logger.Fatal("failed to create proxy", zap.Error(err))
	}

	logger.Info("starting proxy", zap.String("port", port))
	defer logger.Info("stopping proxy")

	// setup reverse proxy for the node
	proxyServer := &httputil.ReverseProxy{Director: conf.ProxyDirector}

	// setup reverse proxy for the local node
	lcoalServer := &httputil.ReverseProxy{Director: localConf.ProxyDirector}

	httpServer := http.Server{
		Addr:              fmt.Sprintf("%v:%v", host, port),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		Handler:           CORSMiddleware(auth.AuthorizeProxy(proxyServer, lcoalServer, http.HandlerFunc(conf.ProxyConfig), http.HandlerFunc(localConf.ProxyConfig))),
	}
	httpServer.SetKeepAlivesEnabled(false)

	httpServer.ListenAndServe()
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // change this later
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}

		next.ServeHTTP(w, r)
	})
}
