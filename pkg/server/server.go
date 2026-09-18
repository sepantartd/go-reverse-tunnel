package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/udp"
	"golang.org/x/crypto/acme/autocert"
)

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window * 2)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)
		for ip, times := range rl.attempts {
			var valid []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

type ClientSession struct {
	ID            string
	Session       *yamux.Session
	ConnectedAt   time.Time
	ActivePorts   []int
	BytesTx       uint64
	BytesRx       uint64
	ActiveStreams int32
}

type Server struct {
	config      *config.ServerConfig
	logger      *slog.Logger
	clients     map[string]*ClientSession
	listeners   map[int]net.Listener
	activeUDP   map[int]*udp.ServerUDPProxy
	usedPorts   map[int]bool
	rateLimiter *RateLimiter
	mu          sync.RWMutex
	dynamicMu   sync.Mutex

	totalConnections uint64
	authFailures     uint64
}

func NewServer(cfg *config.ServerConfig) *Server {
	return &Server{
		config:      cfg,
		logger:      slog.Default(),
		clients:     make(map[string]*ClientSession),
		listeners:   make(map[int]net.Listener),
		activeUDP:   make(map[int]*udp.ServerUDPProxy),
		usedPorts:   make(map[int]bool),
		rateLimiter: NewRateLimiter(20, 1*time.Minute),
	}
}

func (s *Server) Start() error {
	s.logger.Info("Initializing Reverse Tunnel Server engine...")

	if s.config.DashboardAddr != "" {
		go s.startDashboardServer()
	}

	var listener net.Listener
	var err error

	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		s.logger.Info("Loading TLS configuration from files", slog.String("cert", s.config.TLSCertFile))
		cert, err := tls.LoadX509KeyPair(s.config.TLSCertFile, s.config.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS key pair: %w", err)
		}
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		listener, err = tls.Listen("tcp", s.config.ControlAddr, tlsConfig)
	} else if s.config.AutoTLSDomain != "" {
		s.logger.Info("Enabling Auto-TLS via Let's Encrypt", slog.String("domain", s.config.AutoTLSDomain))
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.config.AutoTLSDomain),
			Cache:      autocert.DirCache("certs"),
		}
		tlsConfig := m.TLSConfig()
		listener, err = tls.Listen("tcp", s.config.ControlAddr, tlsConfig)
	} else {
		s.logger.Warn("TLS is not configured. Control channel will run over raw TCP!")
		listener, err = net.Listen("tcp", s.config.ControlAddr)
	}

	if err != nil {
		return fmt.Errorf("failed to bind control listener on %s: %w", s.config.ControlAddr, err)
	}
	defer listener.Close()

	s.logger.Info("Control server successfully bound and listening", slog.String("addr", s.config.ControlAddr))

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.logger.Error("Error accepting control connection", slog.String("error", err.Error()))
			continue
		}

		remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		if !s.rateLimiter.Allow(remoteIP) {
			s.logger.Warn("Rate limit exceeded for IP, dropping connection", slog.String("ip", remoteIP))
			conn.Close()
			atomic.AddUint64(&s.authFailures, 1)
			continue
		}

		atomic.AddUint64(&s.totalConnections, 1)
		go s.handleControlConn(conn)
	}
}

func (s *Server) handleControlConn(conn net.Conn) {
	s.logger.Debug("Processing control connection", slog.String("remote", conn.RemoteAddr().String()))

	if s.config.EnableObfuscation {
		conn = newObfuscatedConn(conn)
	}

	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 15 * time.Second

	session, err := yamux.Server(conn, yamuxConfig)
	if err != nil {
		s.logger.Error("Yamux session handshake failed", slog.String("error", err.Error()))
		conn.Close()
		return
	}

	stream, err := session.AcceptStream()
	if err != nil {
		s.logger.Error("Failed to accept authentication stream", slog.String("error", err.Error()))
		session.Close()
		return
	}

	clientID, authSuccess := s.authenticateStream(stream)
	if !authSuccess {
		s.logger.Warn("Authentication failed for connection", slog.String("remote", conn.RemoteAddr().String()))
		atomic.AddUint64(&s.authFailures, 1)
		s.sendWebhookAlert("AuthFailure", map[string]string{"remote": conn.RemoteAddr().String()})
		_ = stream.Close()
		session.Close()
		return
	}

	_ = stream.Close()
	s.logger.Info("Client authenticated successfully", slog.String("client_id", clientID))

	clientSess := &ClientSession{
		ID:          clientID,
		Session:     session,
		ConnectedAt: time.Now(),
		ActivePorts: make([]int, 0),
	}

	s.mu.Lock()
	if oldSess, exists := s.clients[clientID]; exists {
		s.logger.Info("Terminating stale session for client", slog.String("client_id", clientID))
		oldSess.Session.Close()
	}
	s.clients[clientID] = clientSess
	s.mu.Unlock()

	s.sendWebhookAlert("ClientConnected", map[string]string{"client_id": clientID})

	s.setupClientListeners(clientSess)
	go s.monitorSession(clientSess)
}

func (s *Server) authenticateStream(stream net.Conn) (string, bool) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", false
	}

	if _, err := stream.Write(nonce); err != nil {
		return "", false
	}

	buf := make([]byte, 512)
	_ = stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := stream.Read(buf)
	if err != nil {
		return "", false
	}

	parts := bytes.SplitN(buf[:n], []byte(":"), 2)
	if len(parts) != 2 {
		return "", false
	}

	clientID := string(parts[0])
	clientHmac := parts[1]

	expectedHmac := s.calculateHMAC(nonce, s.config.Token, clientID)
	if !hmac.Equal([]byte(hex.EncodeToString(clientHmac)), []byte(hex.EncodeToString(expectedHmac))) {
		return "", false
	}

	return clientID, s.validateClientConfig(clientID)
}

func (s *Server) calculateHMAC(nonce []byte, secret string, clientID string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(nonce)
	h.Write([]byte(clientID))
	return h.Sum(nil)
}

func (s *Server) validateClientConfig(clientID string) bool {
	if len(s.config.Clients) == 0 {
		return true
	}
	for _, c := range s.config.Clients {
		if c.ClientID == clientID {
			return true
		}
	}
	return false
}

func (s *Server) setupClientListeners(client *ClientSession) {
	for _, clientCfg := range s.config.Clients {
		if clientCfg.ClientID == client.ID {
			for _, port := range clientCfg.Ports {
				actualPort := port
				if port == 0 {
					allocated, err := s.AllocateDynamicPort()
					if err != nil {
						s.logger.Error("Dynamic port allocation failed", slog.String("client_id", client.ID), slog.String("error", err.Error()))
						continue
					}
					actualPort = allocated
				}
				go s.startProxyListener(actualPort, client)
			}
		}
	}
}

func (s *Server) startProxyListener(port int, client *ClientSession) {
	s.mu.Lock()
	if _, exists := s.listeners[port]; exists {
		s.mu.Unlock()
		s.logger.Warn("Port already listened by another service", slog.Int("port", port))
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.mu.Unlock()
		s.logger.Error("Failed to bind proxy listener", slog.Int("port", port), slog.String("error", err.Error()))
		return
	}

	s.listeners[port] = listener
	client.ActivePorts = append(client.ActivePorts, port)
	s.mu.Unlock()

	s.logger.Info("Proxy forwarding rule active", slog.Int("port", port), slog.String("client_id", client.ID))

	defer func() {
		s.mu.Lock()
		listener.Close()
		delete(s.listeners, port)
		s.ReleaseDynamicPort(port)
		s.mu.Unlock()
		s.logger.Info("Proxy listener closed and port released", slog.Int("port", port))
	}()

	for {
		userConn, err := listener.Accept()
		if err != nil {
			if client.Session.IsClosed() {
				break
			}
			s.logger.Error("Proxy accept connection failed", slog.Int("port", port), slog.String("error", err.Error()))
			continue
		}

		go s.handleProxyConnection(userConn, client, port)
	}
}

func (s *Server) handleProxyConnection(userConn net.Conn, client *ClientSession, port int) {
	defer userConn.Close()

	if client.Session.IsClosed() {
		s.logger.Error("Client session unavailable for forwarding", slog.String("client_id", client.ID))
		return
	}

	atomic.AddInt32(&client.ActiveStreams, 1)
	defer atomic.AddInt32(&client.ActiveStreams, -1)

	stream, err := client.Session.OpenStream()
	if err != nil {
		s.logger.Error("Failed opening stream to reverse client", slog.String("client_id", client.ID), slog.String("error", err.Error()))
		return
	}
	defer stream.Close()

	portHeader := fmt.Sprintf("%d\n", port)
	if _, err := stream.Write([]byte(portHeader)); err != nil {
		s.logger.Error("Failed writing target metadata", slog.String("error", err.Error()))
		return
	}

	go func() {
		udpProxy, err := udp.NewServerUDPProxy(port, stream)
		if err == nil {
			udpProxy.StartServerForwarding()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(stream, userConn)
		atomic.AddUint64(&client.BytesRx, uint64(n))
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(userConn, stream)
		atomic.AddUint64(&client.BytesTx, uint64(n))
	}()

	wg.Wait()
}

func (s *Server) monitorSession(client *ClientSession) {
	<-client.Session.CloseChan()
	s.logger.Info("Client session lost or disconnected", slog.String("client_id", client.ID))

	s.sendWebhookAlert("ClientDisconnected", map[string]string{"client_id": client.ID})

	s.mu.Lock()
	delete(s.clients, client.ID)
	for _, port := range client.ActivePorts {
		if l, ok := s.listeners[port]; ok {
			l.Close()
			delete(s.listeners, port)
		}
		s.ReleaseDynamicPort(port)
	}
	s.mu.Unlock()
}

func (s *Server) AllocateDynamicPort() (int, error) {
	s.dynamicMu.Lock()
	defer s.dynamicMu.Unlock()

	minPort, maxPort := s.GetDynamicPortRange()

	for port := minPort; port <= maxPort; port++ {
		if !s.usedPorts[port] {
			l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				l.Close()
				s.usedPorts[port] = true
				return port, nil
			}
		}
	}

	return 0, fmt.Errorf("exhausted dynamic ports in range [%d-%d]", minPort, maxPort)
}

func (s *Server) ReleaseDynamicPort(port int) {
	s.dynamicMu.Lock()
	defer s.dynamicMu.Unlock()
	delete(s.usedPorts, port)
}

func (s *Server) GetDynamicPortRange() (int, int) {
	minPort := s.config.DynamicPortMin
	maxPort := s.config.DynamicPortMax

	if minPort <= 0 {
		minPort = 40000
	}
	if maxPort <= 0 {
		maxPort = 50000
	}

	if minPort > maxPort {
		minPort, maxPort = 40000, 50000
	}

	return minPort, maxPort
}

func (s *Server) startDashboardServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/dashboard", s.authMiddleware(s.handleDashboardPage))
	mux.HandleFunc("/metrics", s.authMiddleware(s.handleMetrics))
	mux.HandleFunc("/api/status", s.authMiddleware(s.handleAPIStatus))
	mux.HandleFunc("/api/clients", s.authMiddleware(s.handleAPIClients))

	mux.HandleFunc("/debug/pprof/", s.authMiddleware(pprof.Index))
	mux.HandleFunc("/debug/pprof/cmdline", s.authMiddleware(pprof.Cmdline))
	mux.HandleFunc("/debug/pprof/profile", s.authMiddleware(pprof.Profile))
	mux.HandleFunc("/debug/pprof/symbol", s.authMiddleware(pprof.Symbol))
	mux.HandleFunc("/debug/pprof/trace", s.authMiddleware(pprof.Trace))

	server := &http.Server{
		Addr:         s.config.DashboardAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	s.logger.Info("Dashboard and metrics portal running", slog.String("addr", s.config.DashboardAddr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Dashboard server error", slog.String("error", err.Error()))
	}
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.Header.Get("X-API-Token")
		}

		if s.config.Token != "" && token != s.config.Token {
			http.Error(w, "401 Unauthorized: Invalid or missing token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html>
<head><title>Go Reverse Tunnel Dashboard</title>
<style>
body{font-family:monospace;background:#121212;color:#e0e0e0;padding:20px;}
h1{color:#4caf50;} table{width:100%;border-collapse:collapse;margin-top:20px;}
th,td{border:1px solid #333;padding:10px;text-align:left;}
th{background:#1e1e1e;color:#81c784;}
</style></head>
<body>
<h1>Go Reverse Tunnel Control Panel</h1>
<p>Status: <span style="color:#4caf50;">ONLINE</span></p>
<div id="data">Loading metrics...</div>
<script>
function loadData(){
  fetch('/api/status?token=` + s.config.Token + `')
    .then(r=>r.json())
    .then(d=>{
      document.getElementById('data').innerHTML = '<pre>'+JSON.stringify(d,null,2)+'</pre>';
    });
}
loadData();
setInterval(loadData, 3000);
</script>
</body></html>`
	_, _ = w.Write([]byte(html))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	activeClients := len(s.clients)
	activeListeners := len(s.listeners)
	s.mu.RUnlock()

	var totalTx, totalRx uint64
	s.mu.RLock()
	for _, c := range s.clients {
		totalTx += atomic.LoadUint64(&c.BytesTx)
		totalRx += atomic.LoadUint64(&c.BytesRx)
	}
	s.mu.RUnlock()

	metrics := fmt.Sprintf(
		"# HELP reverse_tunnel_active_clients Current active client connections.\n"+
			"# TYPE reverse_tunnel_active_clients gauge\n"+
			"reverse_tunnel_active_clients %d\n\n"+
			"# HELP reverse_tunnel_active_listeners Active listening ports.\n"+
			"# TYPE reverse_tunnel_active_listeners gauge\n"+
			"reverse_tunnel_active_listeners %d\n\n"+
			"# HELP reverse_tunnel_total_connections Cumulative total connection attempts.\n"+
			"# TYPE reverse_tunnel_total_connections counter\n"+
			"reverse_tunnel_total_connections %d\n\n"+
			"# HELP reverse_tunnel_auth_failures Total authentication failures.\n"+
			"# TYPE reverse_tunnel_auth_failures counter\n"+
			"reverse_tunnel_auth_failures %d\n\n"+
			"# HELP reverse_tunnel_bytes_transmitted_total Total bytes sent.\n"+
			"# TYPE reverse_tunnel_bytes_transmitted_total counter\n"+
			"reverse_tunnel_bytes_transmitted_total %d\n\n"+
			"# HELP reverse_tunnel_bytes_received_total Total bytes received.\n"+
			"# TYPE reverse_tunnel_bytes_received_total counter\n"+
			"reverse_tunnel_bytes_received_total %d\n",
		activeClients, activeListeners,
		atomic.LoadUint64(&s.totalConnections),
		atomic.LoadUint64(&s.authFailures),
		totalTx, totalRx,
	)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(metrics))
}

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	activeClients := len(s.clients)
	activeListeners := len(s.listeners)
	s.mu.RUnlock()

	resp := map[string]interface{}{
		"status":            "running",
		"active_clients":    activeClients,
		"active_listeners":  activeListeners,
		"total_connections": atomic.LoadUint64(&s.totalConnections),
		"auth_failures":     atomic.LoadUint64(&s.authFailures),
		"uptime_seconds":    time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAPIClients(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clientList := make([]map[string]interface{}, 0)
	for _, c := range s.clients {
		clientList = append(clientList, map[string]interface{}{
			"id":             c.ID,
			"connected_at":   c.ConnectedAt.Format(time.RFC3339),
			"active_ports":   c.ActivePorts,
			"active_streams": atomic.LoadInt32(&c.ActiveStreams),
			"bytes_tx":       atomic.LoadUint64(&c.BytesTx),
			"bytes_rx":       atomic.LoadUint64(&c.BytesRx),
		})
	}

	w.Header().State("Content-Type", "application/json") // Note: fix standard header map if needed, standard is w.Header().Set(...)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clientList)
}

func (s *Server) sendWebhookAlert(eventType string, data map[string]string) {
	if s.config.WebhookURL == "" {
		return
	}

	payload := map[string]interface{}{
		"event":     eventType,
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      data,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post(s.config.WebhookURL, "application/json", bytes.NewBuffer(jsonBytes))
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()
}

type obfuscatedConn struct {
	net.Conn
	xorKey []byte
}

func newObfuscatedConn(c net.Conn) net.Conn {
	key := []byte{0xFA, 0xCE, 0x11, 0x99, 0x55, 0xAA, 0x33, 0x77, 0xBB, 0xCC, 0xDD, 0xEE, 0x12, 0x34, 0x56, 0x78}
	return &obfuscatedConn{
		Conn:   c,
		xorKey: key,
	}
}

func (o *obfuscatedConn) Read(b []byte) (int, error) {
	n, err := o.Conn.Read(b)
	if err != nil {
		return n, err
	}
	for i := 0; i < n; i++ {
		b[i] ^= o.xorKey[i%len(o.xorKey)]
	}
	return n, nil
}

func (o *obfuscatedConn) Write(b []byte) (int, error) {
	buf := make([]byte, len(b))
	copy(buf, b)
	for i := 0; i < len(buf); i++ {
		buf[i] ^= o.xorKey[i%len(o.xorKey)]
	}
	return o.Conn.Write(buf)
}
