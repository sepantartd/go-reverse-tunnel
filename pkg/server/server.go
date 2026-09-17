package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/metrics"
	"github.com/sepantartd/go-reverse-tunnel/pkg/notify"
	"github.com/sepantartd/go-reverse-tunnel/pkg/obfuscate"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
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
	config    *config.ServerConfig
	clients   map[string]*ClientSession
	listeners map[int]net.Listener
	ctrlLn    net.Listener
	logger    *slog.Logger
	mu        sync.RWMutex
	connSem   chan struct{}
	closeCtx  chan struct{}
	closeOnce sync.Once
}

func NewTunnelServer(cfg *config.ServerConfig) *TunnelServer {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	metrics.Register()

	return &TunnelServer{
		config:    cfg,
		clients:   make(map[string]*ClientSession),
		listeners: make(map[int]net.Listener),
		logger:    logger,
		connSem:   make(chan struct{}, 10000),
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

	s.logger.Info("Control server started", slog.String("addr", s.config.ControlAddr))

	if s.config.DashboardAddr != "" {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/dashboard", s.HandleDashboard)
			mux.Handle("/metrics", metrics.Handler())
			s.logger.Info("Dashboard and Metrics server listening", slog.String("addr", s.config.DashboardAddr))
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
				s.logger.Error("Control accept error", slog.String("error", err.Error()))
				continue
			}
		}
		go s.handleControlConnection(conn)
	}
}

func (s *TunnelServer) handleControlConnection(conn net.Conn) {
	defer conn.Close()

	var err error
	if s.config.EnableObfuscation {
		conn, err = obfuscate.PerformServerHandshake(conn)
		if err != nil {
			s.logger.Warn("Obfuscation handshake failed", slog.String("remote_addr", conn.RemoteAddr().String()), slog.String("error", err.Error()))
			return
		}
	}

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
		metrics.AuthFailures.Inc()
		s.logger.Warn("Auth failed for connection", slog.String("remote_addr", conn.RemoteAddr().String()))
		notify.SendWebhook(s.config.WebhookURL, notify.EventAuthFailed, "", conn.RemoteAddr().String(), "Authentication failed")
		return
	}

	session, err := yamux.Server(conn, nil)
	if err != nil {
		return
	}
	defer session.Close()

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
	metrics.ActiveClients.Inc()
	s.mu.Unlock()

	s.logger.Info("Client authenticated successfully", slog.String("client_id", clientID), slog.String("remote_addr", conn.RemoteAddr().String()))
	notify.SendWebhook(s.config.WebhookURL, notify.EventClientConnected, clientID, conn.RemoteAddr().String(), "Client connected successfully")

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		metrics.ActiveClients.Dec()
		s.mu.Unlock()
		s.logger.Info("Client disconnected", slog.String("client_id", clientID))
		notify.SendWebhook(s.config.WebhookURL, notify.EventClientDisconnected, clientID, conn.RemoteAddr().String(), "Client disconnected")
	}()

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
		s.logger.Error("Failed to listen on public port", slog.Int("port", port), slog.String("error", err.Error()))
		s.mu.Unlock()
		return
	}
	s.listeners[port] = ln
	s.mu.Unlock()

	s.logger.Info("Public listener bound", slog.Int("port", port))

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
			s.logger.Warn("Semaphore full, dropping connection", slog.Int("port", port))
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

	metrics.ActiveStreams.Inc()
	defer metrics.ActiveStreams.Dec()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		n, _ := io.CopyBuffer(stream, publicConn, *bufPtr)
		metrics.BytesTransferred.WithLabelValues("ingress").Add(float64(n))
	}()

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		n, _ := io.CopyBuffer(publicConn, stream, *bufPtr)
		metrics.BytesTransferred.WithLabelValues("egress").Add(float64(n))
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
