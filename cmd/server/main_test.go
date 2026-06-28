package main

import (
	"testing"
	"time"
)

func TestServerConfigFromEnvUsesDefaultAddr(t *testing.T) {
	t.Setenv(httpAddrEnv, "")

	config := serverConfigFromEnv()

	if config.addr != defaultHTTPAddr {
		t.Fatalf("expected default addr %q, got %q", defaultHTTPAddr, config.addr)
	}
}

func TestServerConfigFromEnvUsesConfiguredAddr(t *testing.T) {
	const configuredAddr = ":8080"
	t.Setenv(httpAddrEnv, configuredAddr)

	config := serverConfigFromEnv()

	if config.addr != configuredAddr {
		t.Fatalf("expected configured addr %q, got %q", configuredAddr, config.addr)
	}
}

func TestNewServerAppliesConfig(t *testing.T) {
	config := serverConfig{
		addr:              "127.0.0.1:0",
		readHeaderTimeout: 5 * time.Second,
		readTimeout:       10 * time.Second,
		writeTimeout:      10 * time.Second,
		idleTimeout:       60 * time.Second,
	}

	server := newServer(config)

	if server.Addr != config.addr {
		t.Fatalf("expected addr %q, got %q", config.addr, server.Addr)
	}
	if server.Handler == nil {
		t.Fatal("expected server handler to be set")
	}
	if server.ReadHeaderTimeout != config.readHeaderTimeout {
		t.Fatalf("expected ReadHeaderTimeout %s, got %s", config.readHeaderTimeout, server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != config.readTimeout {
		t.Fatalf("expected ReadTimeout %s, got %s", config.readTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != config.writeTimeout {
		t.Fatalf("expected WriteTimeout %s, got %s", config.writeTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != config.idleTimeout {
		t.Fatalf("expected IdleTimeout %s, got %s", config.idleTimeout, server.IdleTimeout)
	}
}

func TestDefaultTimeoutsAreNonZero(t *testing.T) {
	config := serverConfigFromEnv()

	if config.readHeaderTimeout == 0 {
		t.Fatal("expected ReadHeaderTimeout to be non-zero")
	}
	if config.readTimeout == 0 {
		t.Fatal("expected ReadTimeout to be non-zero")
	}
	if config.writeTimeout == 0 {
		t.Fatal("expected WriteTimeout to be non-zero")
	}
	if config.idleTimeout == 0 {
		t.Fatal("expected IdleTimeout to be non-zero")
	}
}
