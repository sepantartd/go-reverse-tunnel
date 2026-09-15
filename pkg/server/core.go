package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/protocol"
	"github.com/sepantartd/go-reverse-tunnel/pkg/tunnel"
)

type TunnelServer struct {
	config    *config.ServerConfig
	clients   map[string]*ClientSession
	mu        sync.RWMutex
	listeners map[int]net.Listener
	startTime time.Time
}

type ClientSession struct {
	ID      string
	Session *tunnel.Session
}

func NewTunnelServer(cfg *config.ServerConfig) *TunnelServer {
	return &TunnelServer{
		config:    cfg,
		clients:   make(map[string]*ClientSession),
		listeners: make(map[int]net.Listener),
		startTime: time.Now(),
	}
}

func (s *TunnelServer) Start() error {
	if s.config.WebPort > 0 {
		go s.StartDashboard(s.config.WebPort, s.startTime)
		log.Printf("[Server] Web dashboard listening on http://0.0.0.0:%d", s.config.WebPort)
	}

	controlListener, err := net.Listen("tcp", s.config.ControlAddr)
	if err != nil {
		return fmt.Errorf("failed to bind control address: %w", err)
	}
	defer controlListener.Close()

	log.Printf("[Server] Control server listening on %s", s.config.ControlAddr)

	for _, bind := range s.config.PublicBinds {
		go s.startPublicBind(bind.Port, bind.Name)
	}

	for {
		conn, err := controlListener.Accept()
		if err != nil {
			log.Printf("[Server] Accept error: %v", err)
			continue
		}
		go s.handleControlConn(conn)
	}
}

func (s *TunnelServer) handleControlConn(conn net.Conn) {
	nonce, err := protocol.GenerateNonce()
	if err != nil {
		conn.Close()
		return
	}

	challenge := protocol.Challenge{NonceHex: fmt.Sprintf("%x", nonce)}
	if err := json.NewEncoder(conn).Encode(challenge); err != nil {
		conn.Close()
		return
	}

	var authReq protocol.AuthHandshake
	if err := json.NewDecoder(conn).Decode(&authReq); err != nil {
		conn.Close()
		return
	}

	if !protocol.VerifyHMAC(nonce, authReq.Response, s.config.Token) {
		log.Printf("[Server] Auth failed for client %s", authReq.ClientID)
		conn.Close()
		return
	}

	sess, err := tunnel.NewServerSession(conn)
	if err != nil {
		log.Printf("[Server] Mux session failed for %s: %v", authReq.ClientID, err)
		conn.Close()
		return
	}

	clientSess := &ClientSession{
		ID:      authReq.ClientID,
		Session: sess,
	}

	s.mu.Lock()
	s.clients[authReq.ClientID] = clientSess
	s.mu.Unlock()

	log.Printf("[Server] Client authenticated & connected successfully: %s", authReq.ClientID)
}

func (s *TunnelServer) startPublicBind(port int, name string) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[Server] Failed to bind public port %d (%s): %v", port, name, err)
		return
	}
	defer listener.Close()

	s.mu.Lock()
	s.listeners[port] = listener
	s.mu.Unlock()

	log.Printf("[Server] Public bind active on %s (%s)", addr, name)

	for {
		publicConn, err := listener.Accept()
		if err != nil {
			return
		}

		go s.routePublicConn(port, publicConn)
	}
}

func (s *TunnelServer) routePublicConn(port int, publicConn net.Conn) {
	s.mu.RLock()
	var activeSession *ClientSession
	for _, client := range s.clients {
		activeSession = client
		break
	}
	s.mu.RUnlock()

	if activeSession == nil {
		log.Printf("[Server] No connected client available for port %d", port)
		publicConn.Close()
		return
	}

	stream, err := activeSession.Session.Open()
	if err != nil {
		log.Printf("[Server] Failed to open stream for client: %v", err)
		publicConn.Close()
		return
	}

	header := fmt.Sprintf("%d\n", port)
	if _, err := stream.Write([]byte(header)); err != nil {
		stream.Close()
		publicConn.Close()
		return
	}

	tunnel.PipeWithCompress(publicConn, stream, s.config.EnableCompress)
}
