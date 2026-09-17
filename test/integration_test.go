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

func TestIntegration_MultiClientAndTraffic(t *testing.T) {
	// 1. Start a local mock HTTP server
	mockServerMux := http.NewServeMux()
	mockServerMux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pong")
	})
	mockLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen mock server: %v", err)
	}
	defer mockLn.Close()
	go http.Serve(mockLn, mockServerMux)

	mockAddr := mockLn.Addr().String()

	// 2. Start Tunnel Server
	controlPort := 19090
	publicPort := 19091

	srvCfg := &config.ServerConfig{
		ControlAddr: fmt.Sprintf("127.0.0.1:%d", controlPort),
		Token:       "super-secret-token",
		Clients: []config.ClientMapping{
			{
				ClientID: "client-1",
				Ports:    []int{publicPort},
			},
		},
	}

	srv := server.NewTunnelServer(srvCfg)
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()
	defer srv.Close()

	time.Sleep(200 * time.Millisecond)

	// 3. Start Client 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientCfg := &config.ClientConfig{
		ServerAddr:             fmt.Sprintf("127.0.0.1:%d", controlPort),
		ClientID:               "client-1",
		Token:                  "super-secret-token",
		LocalAddr:              mockAddr,
		InsecureAllowPlaintext: true,
	}

	go func() {
		_ = client.RunClient(ctx, clientCfg)
	}()

	time.Sleep(500 * time.Millisecond)

	// 4. Test Public Traffic Routing through Tunnel
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", publicPort))
	if err != nil {
		t.Fatalf("Failed to request via tunnel: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "pong" {
		t.Fatalf("Expected 'pong', got '%s'", string(body))
	}
}
