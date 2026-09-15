package tunnel_test

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

func TestEndToEndTunneling(t *testing.T) {
// 1. Start a local mock HTTP server (Client's target)
mockLocalServer := &http.Server{
Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
fmt.Fprint(w, "Hello from local backend!")
}),
}
mockListener, err := net.Listen("tcp", "127.0.0.1:0")
if err != nil {
t.Fatalf("Failed to start mock local server listener: %v", err)
}
defer mockListener.Close()
go mockLocalServer.Serve(mockListener)

localTargetAddr := mockListener.Addr().String()

// 2. Reserve free control and public ports
controlListener, err := net.Listen("tcp", "127.0.0.1:0")
if err != nil {
t.Fatalf("Failed to listen for free control port: %v", err)
}
controlAddr := controlListener.Addr().String()
controlListener.Close()

publicListener, err := net.Listen("tcp", "127.0.0.1:0")
if err != nil {
t.Fatalf("Failed to listen for free public port: %v", err)
}
publicPort := publicListener.Addr().(*net.TCPAddr).Port
publicListener.Close()

token := "e2e_secret_token"
clientID := "test_client_1"

srvCfg := &config.ServerConfig{
ControlAddr: controlAddr,
Token:       token,
Clients: []config.ClientMapping{
{
ClientID: clientID,
Ports:    []int{publicPort},
},
},
}

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 3. Start TunnelServer instance using NewTunnelServer
srv := server.NewTunnelServer(srvCfg)
defer srv.Close()

go func() {
if err := srv.Start(); err != nil {
t.Logf("Server stopped: %v", err)
}
}()

time.Sleep(100 * time.Millisecond)

// 4. Start Client
cliCfg := &config.ClientConfig{
ServerAddr: controlAddr,
LocalAddr:  localTargetAddr,
ClientID:   clientID,
Token:      token,
}

go func() {
if err := client.RunClient(ctx, cliCfg); err != nil {
t.Logf("Client stopped: %v", err)
}
}()

time.Sleep(300 * time.Millisecond)

// 5. Verify request through tunnel public port
publicURL := fmt.Sprintf("http://127.0.0.1:%d", publicPort)
httpClient := &http.Client{Timeout: 3 * time.Second}

resp, err := httpClient.Get(publicURL)
if err != nil {
t.Fatalf("Failed to reach local server through reverse tunnel: %v", err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
t.Fatalf("Failed to read response body: %v", err)
}

expected := "Hello from local backend!"
if string(body) != expected {
t.Errorf("got %q, want %q", string(body), expected)
}
}
