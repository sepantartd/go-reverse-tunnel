package test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/client"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/server"
)

func TestEndToEndTunnel(t *testing.T) {
	// 1. Start a local target HTTP service
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start target listener: %v", err)
	}
	defer targetLn.Close()

	targetAddr := targetLn.Addr().String()
	targetMux := http.NewServeMux()
	targetMux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong-from-target"))
	})
	targetServer := &http.Server{Handler: targetMux}
	go func() { _ = targetServer.Serve(targetLn) }()
	defer targetServer.Close()

	// 2. Configure & Start Tunnel Server
	srvCfg := &config.ServerConfig{
		ControlAddr:           "127.0.0.1:0",
		Token:                 "integration-secret-token",
		InsecureAllowPlaintext: true,
		Clients: []config.ClientMapping{
			{
				ClientID: "test-client",
				Ports:    []int{19090},
			},
		},
	}

	srv := server.NewTunnelServer(srvCfg)
	go func() {
		_ = srv.Start()
	}()
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	// Retrieve dynamic control server port
	// For integration, we use configured server address
	ctrlPort := srvCfg.ControlAddr

	// 3. Configure & Start Tunnel Client
	clientCfg := &config.ClientConfig{
		ServerAddr:            ctrlPort,
		LocalAddr:             targetAddr,
		ClientID:              "test-client",
		Token:                 "integration-secret-token",
		InsecureAllowPlaintext: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = client.RunClient(ctx, clientCfg)
	}()

	time.Sleep(300 * time.Millisecond)

	// 4. Verify Traffic Forwarding via Public Port 19090
	resp, err := http.Get("http://127.0.0.1:19090/ping")
	if err != nil {
		t.Fatalf("Failed to reach target via tunnel public port: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "pong-from-target" {
		t.Fatalf("Expected 'pong-from-target', got '%s'", string(body))
	}
}
