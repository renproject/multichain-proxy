package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	proxy "github.com/renproject/multichain-proxy"
	zap "go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		logger.Fatal("mossing pod name")
	}
	temp := strings.Split(podName, "-")
	ordinality := temp[len(temp)-1]
	if _, err := strconv.Atoi(ordinality); err != nil {
		logger.Fatal("could not determine instance ordinality", zap.String("pod-name", podName))
	}
	instanceState := os.Getenv("INSTANCE_" + ordinality)
	if instanceState != "1" {
		logger.Warn("sleeping...live status != 1")
		for {
			time.Sleep(10 * time.Minute)
		}
		return
	}
	proxyURL := os.Getenv("PROXY_URL")
	authTokenProxy := os.Getenv("AUTH_TOKEN_PROXY")
	authToken := os.Getenv("AUTH_TOKEN_" + ordinality)
	proxyUser := os.Getenv("PROXY_USER")
	proxyPassword := os.Getenv("PROXY_PASSWORD")
	proxyMethods := strings.Split(os.Getenv("PROXY_METHODS"), ",")
	proxyMethodsMap := map[string]bool{}
	for i := range proxyMethods {
		proxyMethodsMap[strings.TrimSpace(proxyMethods[i])] = true
	}

	opts := proxy.WhitelistOptions{
		Logger:         logger,
		Host:           "0.0.0.0",
		Port:           8080,
		MaxReqSize:     16 * 1024,
		NumWorkers:     10,
		URL:            proxyURL,
		AuthTokenProxy: authTokenProxy,
		AuthToken:      authToken,
		User:           proxyUser,
		Password:       proxyPassword,
		Methods:        proxyMethodsMap,
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
