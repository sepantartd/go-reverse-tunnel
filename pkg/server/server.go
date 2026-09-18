package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/metrics"
	"github.com/sepantartd/go-reverse-tunnel/pkg/notify"
	"github.com/sepantartd/go-reverse-tunnel/pkg/obfuscate"
	"github.com/sepantartd/go-reverse-tunnel/pkg/udp"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/time/rate"
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
	UDPPorts   []int
	Session    *yamux.Session
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (i *ipRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.limiters[ip] = limiter
	}
	return limiter
}

type TunnelServer struct {
	config      *config.ServerConfig
	clients     map[string]*ClientSession
	listeners   map[int]net.Listener
	ctrlLn      net.Listener
	logger      *slog.Logger
	mu          sync.RWMutex
	connSem     chan struct{}
	closeCtx    chan struct{}
	closeOnce   sync.Once
	rateLimiter *ipRateLimiter
}

func NewTunnelServer(cfg *config.ServerConfig) *TunnelServer {
	logger := config.SetupLogger(cfg.LogLevel)
	metrics.Register()

	// 5 requests per minute limit per IP with a burst capacity of 5
	limiter := newIPRateLimiter(rate.Every(12*time.Second), 5)

	return &TunnelServer{
		config:      cfg,
		clients:     make(map[string]*ClientSession),
		listeners:   make(map[int]net.Listener),
		logger:      logger,
		connSem:     make(chan struct{}, 10000),
		closeCtx:    make(chan struct{}),
		rateLimiter: limiter,
	}
}

func (s *TunnelServer) Start() error {
	var err error

	if s.config.EnableAutoTLS {
		cacheDir := s.config.AutoTLSCacheDir
		if cacheDir == "" {
			cacheDir = "certs"
		}
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.config.AutoTLSDomain),
			Cache:      autocert.DirCache(cacheDir),
		}
		tlsCfg := m.TLSConfig()
		tlsCfg.MinVersion = tls.VersionTLS12

		s.ctrlLn, err = tls.Listen("tcp", s.config.ControlAddr, tlsCfg)
		s.logger.Info("Auto-TLS enabled with Let's Encrypt", slog.String("domain", s.config.AutoTLSDomain))
	} else if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
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

			mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
				authToken := r.URL.Query().Get("token")
				if authToken == "" {
					authHeader := r.Header.Get("Authorization")
					if strings.HasPrefix(authHeader, "Bearer ") {
						authToken = strings.TrimPrefix(authHeader, "Bearer ")
					}
				}

				if authToken == "" || authToken != s.config.Token {
					http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
					return
				}

				metrics.Handler().ServeHTTP(w, r)
			})

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

		// Rate Limiting directly at the Accept loop level
		ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err == nil {
			limiter := s.rateLimiter.GetLimiter(ip)
			if !limiter.Allow() {
				s.logger.Warn("Rate limit exceeded at control accept loop", slog.String("ip", ip))
				metrics.AuthFailures.Inc()
				metrics.RateLimitTriggers.Inc()
				conn.Close()
				continue
			}
		}

		go s.handleControlConnection(conn)
	}
}

func (s *TunnelServer) handleControlConnection(conn net.Conn) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	if s.config.EnableObfuscation {
		var err error
		conn, err = obfuscate.PerformServerHandshake(conn)
		if err != nil {
			s.logger.Warn("Obfuscation handshake failed", slog.String("remote_addr", conn.RemoteAddr().String()), slog.String("error", err.Error()))
			return
		}
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		s.logger.Error("Failed to generate random nonce", slog.String("error", err.Error()))
		return
	}

	if _, err := conn.Write(nonce); err != nil {
		s.logger.Debug("Failed to send nonce to client", slog.String("error", err.Error()))
		return
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		s.logger.Debug("Failed to read auth response", slog.String("error", err.Error()))
		return
	}

	payload := stringsTrim(string(buf[:n]))
	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 {
		metrics.AuthFailures.Inc()
		s.logger.Warn("Invalid auth payload format", slog.String("remote_addr", conn.RemoteAddr().String()))
		return
	}

	clientID := parts[0]
	clientHMAC := parts[1]

	mac := hmac.New(sha256.New, []byte(s.config.Token))
	mac.Write(nonce)
	expectedHMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(clientHMAC), []byte(expectedHMAC)) {
		metrics.AuthFailures.Inc()
		s.logger.Warn("Auth failed for connection", slog.String("client_id", clientID), slog.String("remote_addr", conn.RemoteAddr().String()))
		notify.SendWebhook(s.config.WebhookURL, notify.EventAuthFailed, clientID, conn.RemoteAddr().String(), "Authentication failed")
		return
	}

	_ = conn.SetDeadline(time.Time{})

	s.mu.Lock()
	var assignedClient *config.ClientMapping
	for i := range s.config.Clients {
		if s.config.Clients[i].ClientID == clientID {
			assignedClient = &s.config.Clients[i]
			break
		}
	}

	if assignedClient == nil && len(s.config.Clients) > 0 {
		s.logger.Warn("Client ID not allowed or found in server configuration", slog.String("client_id", clientID))
		s.mu.Unlock()
		return
	}

	var ports []int
	var udpPorts []int
	if assignedClient != nil {
		ports = assignedClient.Ports
		udpPorts = assignedClient.UDPPorts
	}

	yamuxCfg := config.GetYamuxConfig(s.config.Yamux)
	session, err := yamux.Server(conn, yamuxCfg)
	if err != nil {
		s.logger.Error("Failed to initialize yamux server session", slog.String("error", err.Error()))
		s.mu.Unlock()
		return
	}
	defer session.Close()

	clientSession := &ClientSession{
		ClientID:   clientID,
		RemoteAddr: conn.RemoteAddr().String(),
		Ports:      ports,
		UDPPorts:   udpPorts,
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

	for _, uPort := range udpPorts {
		go s.listenPublicUDPPort(uPort, session)
	}

	<-session.CloseChan()
}

func (s *TunnelServer) listenPublicUDPPort(port int, session *yamux.Session) {
	stream, err := session.OpenStream()
	if err != nil {
		s.logger.Error("Failed to open stream for UDP proxy", slog.Int("port", port))
		return
	}

	proxy, err := udp.NewServerUDPProxy(port, stream)
	if err != nil {
		s.logger.Error("Failed to start UDP listener", slog.Int("port", port), slog.String("error", err.Error()))
		stream.Close()
		return
	}

	s.logger.Info("Public UDP listener bound", slog.Int("port", port))
	proxy.StartServerForwarding()
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
			select {
			case <-s.closeCtx:
				return
			default:
				s.logger.Debug("Accept error on public port", slog.Int("port", port), slog.String("error", err.Error()))
				return
			}
		}

		select {
		case s.connSem <- struct{}{}:
			go func(c net.Conn) {
				defer func() { <-s.connSem }()
				s.routePublicTraffic(c, session)
			}(publicConn)
		default:
			s.logger.Warn("Connection limit reached, dropping public connection", slog.Int("port", port), slog.String("remote_addr", publicConn.RemoteAddr().String()))
			publicConn.Close()
		}
	}
}

func (s *TunnelServer) routePublicTraffic(publicConn net.Conn, session *yamux.Session) {
	defer publicConn.Close()

	stream, err := session.OpenStream()
	if err != nil {
		s.logger.Error("Failed to open multiplex stream to client", slog.String("error", err.Error()))
		return
	}
	defer stream.Close()

	metrics.ActiveStreams.Inc()
	defer metrics.ActiveStreams.Dec()

	idleTimeout := 5 * time.Minute

	var wg sync.WaitGroup
	wg.Add(2)

	// Ingress: Public -> Tunnel Stream
	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		_ = publicConn.SetReadDeadline(time.Now().Add(idleTimeout))
		_ = stream.SetWriteDeadline(time.Now().Add(idleTimeout))

		n, err := io.CopyBuffer(stream, publicConn, *bufPtr)
		if err != nil && err != io.EOF {
			metrics.ConnectionDrops.Inc()
			s.logger.Debug("Ingress stream copy finished with notice", slog.String("error", err.Error()))
		}
		metrics.BytesTransferred.WithLabelValues("ingress").Add(float64(n))
	}()

	// Egress: Tunnel Stream -> Public
	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		_ = stream.SetReadDeadline(time.Now().Add(idleTimeout))
		_ = publicConn.SetWriteDeadline(time.Now().Add(idleTimeout))

		n, err := io.CopyBuffer(publicConn, stream, *bufPtr)
		if err != nil && err != io.EOF {
			metrics.ConnectionDrops.Inc()
			s.logger.Debug("Egress stream copy finished with notice", slog.String("error", err.Error()))
		}
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
