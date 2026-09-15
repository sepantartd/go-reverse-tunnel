package server

import (
"crypto/tls"
"fmt"
"io"
"log"
"net"
"sync"
"time"

"github.com/sepantartd/go-reverse-tunnel/pkg/config"
"github.com/sepantartd/go-reverse-tunnel/pkg/protocol"
"github.com/sepantartd/go-reverse-tunnel/pkg/tunnel"
)

type ClientSession struct {
ID      string
Session *tunnel.Session
}

type TunnelServer struct {
config    *config.ServerConfig
clients   map[string]*ClientSession
mu        sync.RWMutex
listeners map[int]net.Listener
startTime time.Time
}

func NewTunnelServer(cfg *config.ServerConfig) *TunnelServer {
return &TunnelServer{
config:    cfg,
clients:   make(map[string]*ClientSession),
listeners: make(map[int]net.Listener),
startTime: time.Now(),
}
}

// Start initiates the control listener and public port binders
func (s *TunnelServer) Start() error {
var rawListener net.Listener
var err error

rawListener, err = net.Listen("tcp", s.config.ControlAddr)
if err != nil {
return fmt.Errorf("failed to listen on control address %s: %v", s.config.ControlAddr, err)
}

if s.config.EnableTLS {
tlsConfig, err := protocol.GetServerTLSConfig(s.config.CertFile, s.config.KeyFile, s.config.CAFile)
if err != nil {
rawListener.Close()
return fmt.Errorf("failed to load server TLS config: %v", err)
}
rawListener = tls.NewListener(rawListener, tlsConfig)
log.Println("[Server] Real TLS encryption enabled on control listener")
} else {
log.Println("[Warning] TLS is disabled on server. Control connection is running in plaintext mode.")
}

s.listeners[0] = rawListener
log.Printf("[Server] Control listener started on %s", s.config.ControlAddr)

for {
conn, err := rawListener.Accept()
if err != nil {
log.Printf("[Server] Error accepting connection: %v", err)
break
}
go s.handleControlConnection(conn)
}

return nil
}

// handleControlConnection processes incoming client authentication and tunnel multiplexing
func (s *TunnelServer) handleControlConnection(conn net.Conn) {
defer conn.Close()

// Perform protocol handshake & authentication (simplified placeholder for session setup)
// In actual flow, we read auth request, verify HMAC token, and get clientID
clientID := "client-default" // Will be extracted from real auth protocol frame

sess, err := tunnel.NewServerSession(conn)
if err != nil {
log.Printf("[Server] Failed to create server session: %v", err)
return
}
defer sess.Close()

clientSess := &ClientSession{
ID:      clientID,
Session: sess,
}

s.mu.Lock()
s.clients[clientID] = clientSess
s.mu.Unlock()

// Ensure client is removed from map upon disconnection to prevent resource leaks
defer func() {
s.mu.Lock()
delete(s.clients, clientID)
s.mu.Unlock()
log.Printf("[Server] Client disconnected and removed from active sessions: %s", clientID)
}()

log.Printf("[Server] Client connected successfully: %s", clientID)

// Keep connection alive until session drops
select {}
}

// routePublicConn routes incoming traffic on a public port to the specific target client ID
func (s *TunnelServer) routePublicConn(port int, publicConn net.Conn) {
defer publicConn.Close()

var targetClientID string
for _, clientCfg := range s.config.Clients {
for _, p := range clientCfg.Ports {
if p == port {
targetClientID = clientCfg.ClientID
break
}
}
if targetClientID != "" {
break
}
}

if targetClientID == "" {
log.Printf("[Server] No client mapped for public port %d", port)
return
}

s.mu.RLock()
clientSess, exists := s.clients[targetClientID]
s.mu.RUnlock()

if !exists || clientSess == nil || clientSess.Session == nil {
log.Printf("[Server] Target client '%s' for port %d is not online", targetClientID, port)
return
}

stream, err := clientSess.Session.OpenStream()
if err != nil {
log.Printf("[Server] Failed to open tunnel stream for client %s on port %d: %v", targetClientID, port, err)
return
}
defer stream.Close()

var wg sync.WaitGroup
wg.Add(2)

go func() {
defer wg.Done()
io.Copy(stream, publicConn)
stream.Close()
}()

go func() {
defer wg.Done()
io.Copy(publicConn, stream)
publicConn.Close()
}()

wg.Wait()
}

// Close performs a graceful shutdown of all listeners and active client sessions
func (s *TunnelServer) Close() error {
s.mu.Lock()
defer s.mu.Unlock()

log.Println("[Server] Initiating graceful shutdown...")

// Close all listeners
for id, listener := range s.listeners {
if err := listener.Close(); err != nil {
log.Printf("[Server] Error closing listener %d: %v", id, err)
}
}

// Close all active client sessions
for id, clientSess := range s.clients {
if clientSess.Session != nil {
if err := clientSess.Session.Close(); err != nil {
log.Printf("[Server] Error closing session for client %s: %v", id, err)
}
}
}

log.Println("[Server] Graceful shutdown completed successfully.")
return nil
}
