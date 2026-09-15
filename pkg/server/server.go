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
	controlLn net.Listener
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

// Start initiates the control listener and binds all mapped public ports
func (s *TunnelServer) Start() error {
	var rawLn net.Listener
	var err error

	rawLn, err = net.Listen("tcp", s.config.ControlAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on control address %s: %v", s.config.ControlAddr, err)
	}

	if s.config.EnableTLS {
		tlsConfig, err := protocol.GetServerTLSConfig(s.config.CertFile, s.config.KeyFile, s.config.CAFile)
		if err != nil {
			rawLn.Close()
			return fmt.Errorf("failed to load server TLS config: %v", err)
		}
		s.controlLn = tls.NewListener(rawLn, tlsConfig)
		log.Println("[Server] Real TLS encryption enabled on control listener (TLS 1.2+)")
	} else {
		s.controlLn = rawLn
		log.Println("[Warning] TLS is disabled on server. Control connection is running in plaintext mode.")
	}

	// Bind all public ports mapped in config
	for _, clientMap := range s.config.Clients {
		for _, port := range clientMap.Ports {
			pubLn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
			if err != nil {
				s.Close()
				return fmt.Errorf("failed to bind public port %d: %v", port, err)
			}
			s.listeners[port] = pubLn
			log.Printf("[Server] Bound public port %d for client ID: %s", port, clientMap.ClientID)

			go s.acceptPublicTraffic(port, pubLn)
		}
	}

	log.Printf("[Server] Control listener started on %s", s.config.ControlAddr)

	for {
		conn, err := s.controlLn.Accept()
		if err != nil {
			log.Printf("[Server] Control listener accept error: %v", err)
			break
		}
		go s.handleControlConnection(conn)
	}

	return nil
}

func (s *TunnelServer) acceptPublicTraffic(port int, ln net.Listener) {
	for {
		pubConn, err := ln.Accept()
		if err != nil {
			log.Printf("[Server] Public listener accept error on port %d: %v", port, err)
			break
		}
		go s.routePublicConn(port, pubConn)
	}
}

func (s *TunnelServer) handleControlConnection(conn net.Conn) {
	defer conn.Close()

	// Perform real HMAC-SHA256 authentication handshake
	clientID, err := protocol.ServerAuthenticate(conn, s.config.Token)
	if err != nil {
		log.Printf("[Server] Authentication failed from %s: %v", conn.RemoteAddr(), err)
		return
	}

	sess, err := tunnel.NewServerSession(conn)
	if err != nil {
		log.Printf("[Server] Failed to create multiplexed session for client %s: %v", clientID, err)
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

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		log.Printf("[Server] Client disconnected: %s", clientID)
	}()

	log.Printf("[Server] Client authenticated and connected successfully: %s", clientID)

	// Keep session alive
	select {}
}

func (s *TunnelServer) routePublicConn(port int, publicConn net.Conn) {
	defer publicConn.Close()

	var targetClientID string
	for _, clientMap := range s.config.Clients {
		for _, p := range clientMap.Ports {
			if p == port {
				targetClientID = clientMap.ClientID
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

func (s *TunnelServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controlLn != nil {
		s.controlLn.Close()
	}

	for port, ln := range s.listeners {
		ln.Close()
		log.Printf("[Server] Closed public listener on port %d", port)
	}

	for _, clientSess := range s.clients {
		if clientSess.Session != nil {
			clientSess.Session.Close()
		}
	}

	return nil
}
