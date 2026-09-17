package server_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/server"
)

func TestServer_AuthenticationFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer ln.Close()

	srvCfg := &config.ServerConfig{
		ControlAddr: ln.Addr().String(),
		Token:       "correct_secret_token",
		Clients: []config.ClientMapping{
			{ClientID: "client_1", Ports: []int{0}},
		},
	}

	srv := server.NewTunnelServer(srvCfg)
	defer srv.Close()

	go func() {
		_ = srv.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	// Connect to control port directly
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Read challenge from server
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read challenge: %v", err)
	}
	challenge := buf[:n]

	// Respond with WRONG token HMAC
	mac := hmac.New(sha256.New, []byte("wrong_secret_token"))
	mac.Write(challenge)
	invalidResponse := hex.EncodeToString(mac.Sum(nil)) + "\n"

	_, err = conn.Write([]byte(invalidResponse))
	if err != nil {
		t.Fatalf("Failed to send response: %v", err)
	}

	// Server must close the connection on auth failure
	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	readBuf := make([]byte, 100)
	_, err = conn.Read(readBuf)
	if err == nil {
		t.Fatalf("Expected connection to be closed by server on auth failure, but read succeeded")
	}
}

func TestServer_GoroutineLeakCheck(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	srvCfg := &config.ServerConfig{
		ControlAddr: ln.Addr().String(),
		Token:       "test_token",
	}

	srv := server.NewTunnelServer(srvCfg)
	ln.Close() // Close listener immediately

	go func() {
		_ = srv.Start()
	}()

	time.Sleep(50 * time.Millisecond)
	srv.Close()

	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Potential Goroutine leak: initial %d, final %d", initialGoroutines, finalGoroutines)
	}
}
