package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	proxy "github.com/renproject/multichain-proxy"
	zap "go.uber.org/zap"
)

func main() {
	proxyURL1 := os.Getenv("PROXY_URL1")
	proxyURL2 := os.Getenv("PROXY_URL2")
	authTokenProxy := os.Getenv("AUTH_TOKEN_PROXY")
	authToken1 := os.Getenv("AUTH_TOKEN1")
	authToken2 := os.Getenv("AUTH_TOKEN2")
	proxyUser := os.Getenv("PROXY_USER")
	proxyPassword := os.Getenv("PROXY_PASSWORD")
	proxyMethods := strings.Split(os.Getenv("PROXY_METHODS"), ",")
	proxyMethodsMap := map[string]bool{}
	for i := range proxyMethods {
		proxyMethodsMap[strings.TrimSpace(proxyMethods[i])] = true
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	opts := proxy.WhitelistOptions{
		Logger: logger,

		Host:       "0.0.0.0",
		Port:       8080,
		MaxReqSize: 16 * 1024,
		NumWorkers: 10,

		URL1:     proxyURL1,
		URL2:     proxyURL2,
		AuthTokenProxy: authTokenProxy,
		AuthToken1: authToken1,
		AuthToken2: authToken2,
		User:     proxyUser,
		Password: proxyPassword,
		Methods:  proxyMethodsMap,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var sig os.Signal
	defer func() {
		if sig != nil {
			logger.Warn("exit", zap.String("signal", sig.String()))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer cancel()
		sig = <-signals
	}()

	logger.Info("starting proxy")
	defer logger.Info("stopping proxy")
	proxy.NewWhitelist(opts).Run(ctx)
}
