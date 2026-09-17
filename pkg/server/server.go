package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024) // 32KB reusable buffer
		return &b
	},
}

type ClientSession struct {
	ClientID   string
	RemoteAddr string
	Ports      []int
	Session    *yamux.Session
}

type TunnelServer struct {
	config      *config.ServerConfig
	clients     map[string]*ClientSession
	listeners   map[int]net.Listener
	ctrlLn      net.Listener
	mu          sync.RWMutex
	connSem     chan struct{} // Semaphore for controlling concurrent public connections
	closeCtx    chan struct{}
	closeOnce   sync.Once
}

func NewTunnelServer(cfg *config.ServerConfig) *TunnelServer {
	return &TunnelServer{
		config:    cfg,
		clients:   make(map[string]*ClientSession),
		listeners: make(map[int]net.Listener),
		connSem:   make(chan struct{}, 10000), // Max 10,000 active concurrent public connections
		closeCtx:  make(chan struct{}),
	}
}

func (s *TunnelServer) Start() error {
	var err error
	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(s.config.TLSCertFile, s.config.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS key pair: %v", err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		s.ctrlLn, err = tls.Listen("tcp", s.config.ControlAddr, tlsCfg)
	} else {
		s.ctrlLn, err = net.Listen("tcp", s.config.ControlAddr)
	}

	if err != nil {
		return fmt.Errorf("failed to listen on control addr %s: %v", s.config.ControlAddr, err)
	}
	defer s.ctrlLn.Close()

	if s.config.DashboardAddr != "" {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/dashboard", s.HandleDashboard)
			_ = http.ListenAndServe(s.config.DashboardAddr, mux)
		}()
	}

	for {
		conn, err := s.ctrlLn.Accept()
		if err != nil {
			select {
			case <-s.closeCtx:
				return nil
			default:
				log.Printf("[Server] Control accept error: %v", err)
				continue
			}
		}
		go s.handleControlConnection(conn)
	}
}

func (s *TunnelServer) handleControlConnection(conn net.Conn) {
	defer conn.Close()

	challenge := []byte("secret_challenge_nonce")
	if _, err := conn.Write(challenge); err != nil {
		return
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	clientResponse := string(buf[:n])
	mac := hmac.New(sha256.New, []byte(s.config.Token))
	mac.Write(challenge)
	expectedResponse := hex.EncodeToString(mac.Sum(nil))

	if stringsTrim(clientResponse) != expectedResponse {
		log.Printf("[Server] Auth failed for connection: %s", conn.RemoteAddr().String())
		return
	}

	session, err := yamux.Server(conn, nil)
	if err != nil {
		return
	}
	defer session.Close()

	// Assign Client
	s.mu.Lock()
	var assignedClient *config.ClientMapping
	if len(s.config.Clients) > 0 {
		assignedClient = &s.config.Clients[0]
	}

	clientID := "default"
	var ports []int
	if assignedClient != nil {
		clientID = assignedClient.ClientID
		ports = assignedClient.Ports
	}

	clientSession := &ClientSession{
		ClientID:   clientID,
		RemoteAddr: conn.RemoteAddr().String(),
		Ports:      ports,
		Session:    session,
	}
	s.clients[clientID] = clientSession
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
	}()

	// Open Public Ports
	for _, port := range ports {
		go s.listenPublicPort(port, session)
	}

	<-session.CloseChan()
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func (s *TunnelServer) listenPublicPort(port int, session *yamux.Session) {
	s.mu.Lock()
	if _, exists := s.listeners[port]; exists {
		s.mu.Unlock()
		return
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.mu.Unlock()
		return
	}
	s.listeners[port] = ln
	s.mu.Unlock()

	defer func() {
		ln.Close()
		s.mu.Lock()
		delete(s.listeners, port)
		s.mu.Unlock()
	}()

	for {
		publicConn, err := ln.Accept()
		if err != nil {
			return
		}

		select {
		case s.connSem <- struct{}{}:
			go func(c net.Conn) {
				defer func() { <-s.connSem }()
				s.routePublicTraffic(c, session)
			}(publicConn)
		default:
			log.Printf("[Server] Semaphore full, dropping public connection on port %d", port)
			publicConn.Close()
		}
	}
}

func (s *TunnelServer) routePublicTraffic(publicConn net.Conn, session *yamux.Session) {
	defer publicConn.Close()

	stream, err := session.OpenStream()
	if err != nil {
		return
	}
	defer stream.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(stream, publicConn, *bufPtr)
	}()

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(publicConn, stream, *bufPtr)
	}()

	wg.Wait()
}

func (s *TunnelServer) Close() error {
	s.closeOnce.Do(func() {
		close(s.closeCtx)
		if s.ctrlLn != nil {
			s.ctrlLn.Close()
		}
		s.mu.Lock()
		for _, ln := range s.listeners {
			ln.Close()
		}
		for _, client := range s.clients {
			client.Session.Close()
		}
		s.mu.Unlock()
	})
	return nil
}
