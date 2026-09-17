package client_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/client"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func TestClient_InvalidServerAuth(t *testing.T) {
	// Mock server that accepts connection, sends dummy challenge, and closes
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Send fake challenge
		_, _ = conn.Write([]byte("dummy_challenge_data"))
		// Read client HMAC response
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf)
		// Reject client by closing
	}()

	cliCfg := &config.ClientConfig{
		ServerAddr: ln.Addr().String(),
		LocalAddr:  "127.0.0.1:8080",
		ClientID:   "test_client",
		Token:      "wrong_token",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err = client.RunClient(ctx, cliCfg)
	if err == nil {
		t.Fatalf("Expected RunClient to return error or exit on context cancellation, got nil")
	}
}
