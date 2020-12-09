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
	proxyURL := os.Getenv("PROXY_URL")
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

		URL:      proxyURL,
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
