package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"bank-system/internal/httpapi"
)

const (
	defaultHTTPAddr = "127.0.0.1:8080"
	httpAddrEnv     = "BANK_SYSTEM_HTTP_ADDR"
)

type serverConfig struct {
	addr              string
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
}

func serverConfigFromEnv() serverConfig {
	addr := os.Getenv(httpAddrEnv)
	if addr == "" {
		addr = defaultHTTPAddr
	}

	return serverConfig{
		addr:              addr,
		readHeaderTimeout: 5 * time.Second,
		readTimeout:       10 * time.Second,
		writeTimeout:      10 * time.Second,
		idleTimeout:       60 * time.Second,
	}
}

func newServer(config serverConfig) *http.Server {
	return &http.Server{
		Addr:              config.addr,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: config.readHeaderTimeout,
		ReadTimeout:       config.readTimeout,
		WriteTimeout:      config.writeTimeout,
		IdleTimeout:       config.idleTimeout,
	}
}

func main() {
	server := newServer(serverConfigFromEnv())

	log.Printf("starting bank-system server on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped unexpectedly: %v", err)
	}
}
