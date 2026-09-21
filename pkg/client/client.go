package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/udp"
)

type Client struct {
	config            *config.ClientConfig
	logger            *slog.Logger
	session           *yamux.Session
	mu                sync.RWMutex
	reconnectAttempts uint64
	bytesTx           uint64
	bytesRx           uint64
	activeStreams     int32
	isShutdown        bool
}

func NewClient(cfg *config.ClientConfig) *Client {
	return &Client{
		config: cfg,
		logger: slog.Default(),
	}
}

func (c *Client) Start() error {
	c.logger.Info("Starting Reverse Tunnel Client...", slog.String("client_id", c.config.ClientID))

	for {
		if c.isShutdown {
			c.logger.Info("Client shutdown requested. Stopping reconnect loop.")
			return nil
		}

		err := c.connectAndServe()
		if err != nil {
			atomic.AddUint64(&c.reconnectAttempts, 1)
			c.logger.Error("Tunnel session lost or failed to establish, reconnecting in 5s...",
				slog.String("error", err.Error()),
				slog.Uint64("attempt", atomic.LoadUint64(&c.reconnectAttempts)),
			)
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Client) Stop() {
	c.mu.Lock()
	c.isShutdown = true
	if c.session != nil {
		_ = c.session.Close()
	}
	c.mu.Unlock()
	c.logger.Info("Reverse Tunnel Client stopped")
}

func (c *Client) connectAndServe() error {
	var conn net.Conn
	var err error

	if c.config.TLSCACertFile != "" || c.config.TLSCertFile != "" {
		tlsConfig, tlsErr := c.buildTLSConfig()
		if tlsErr != nil {
			return fmt.Errorf("failed to build TLS configuration: %w", tlsErr)
		}
		c.logger.Info("Dialing control server over TLS", slog.String("addr", c.config.ServerAddr))
		conn, err = tls.Dial("tcp", c.config.ServerAddr, tlsConfig)
	} else {
		c.logger.Info("Dialing control server over raw TCP", slog.String("addr", c.config.ServerAddr))
		conn, err = net.DialTimeout("tcp", c.config.ServerAddr, 10*time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", c.config.ServerAddr, err)
	}

	if c.config.EnableObfuscation {
		conn = newObfuscatedConn(conn)
	}

	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 15 * time.Second

	session, err := yamux.Client(conn, yamuxConfig)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create yamux client session: %w", err)
	}

	c.mu.Lock()
	c.session = session
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.session == session {
			c.session = nil
		}
		c.mu.Unlock()
		session.Close()
	}()

	authStream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("failed to open authentication stream: %w", err)
	}

	if err := c.performHMACAuth(authStream); err != nil {
		authStream.Close()
		return fmt.Errorf("authentication challenge failed: %w", err)
	}
	_ = authStream.Close()

	c.logger.Info("Successfully authenticated with server", slog.String("client_id", c.config.ClientID))
	atomic.StoreUint64(&c.reconnectAttempts, 0)

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return fmt.Errorf("session stream closed by remote server: %w", err)
		}

		go c.handleIncomingStream(stream)
	}
}

func (c *Client) performHMACAuth(stream net.Conn) error {
	nonce := make([]byte, 32)
	_ = stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, err := io.ReadFull(stream, nonce)
	if err != nil {
		return fmt.Errorf("failed to read nonce from server: %w", err)
	}

	h := hmac.New(sha256.New, []byte(c.config.Token))
	h.Write(nonce)
	h.Write([]byte(c.config.ClientID))
	mac := h.Sum(nil)

	payload := fmt.Sprintf("%s:%s", c.config.ClientID, hex.EncodeToString(mac))
	_ = stream.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = stream.Write([]byte(payload))
	if err != nil {
		return fmt.Errorf("failed to send HMAC response to server: %w", err)
	}

	return nil
}

func (c *Client) handleIncomingStream(stream net.Conn) {
	defer stream.Close()

	atomic.AddInt32(&c.activeStreams, 1)
	defer atomic.AddInt32(&c.activeStreams, -1)

	headerBuf := make([]byte, 64)
	_ = stream.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := stream.Read(headerBuf)
	if err != nil {
		c.logger.Error("Failed reading stream target header", slog.String("error", err.Error()))
		return
	}
	_ = stream.SetReadDeadline(time.Time{})

	line := strings.TrimSpace(string(headerBuf[:n]))
	remotePort, _ := strconv.Atoi(line)

	targetAddr := c.config.LocalTarget
	if targetAddr == "" {
		targetAddr = "127.0.0.1:8080"
	}

	c.logger.Debug("Forwarding incoming tunnel connection",
		slog.Int("remote_port", remotePort),
		slog.String("target", targetAddr),
	)

	targetConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		c.logger.Warn("Failed TCP connection to local target, attempting UDP fallback",
			slog.String("target", targetAddr),
			slog.String("error", err.Error()),
		)

		udpProxy, udpErr := udp.NewClientUDPProxy(targetAddr, stream)
		if udpErr == nil {
			udpProxy.StartClientForwarding()
			return
		}

		c.logger.Error("Both TCP and UDP dialing failed for target", slog.String("target", targetAddr))
		return
	}
	defer targetConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(targetConn, stream)
		atomic.AddUint64(&c.bytesRx, uint64(n))
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(stream, targetConn)
		atomic.AddUint64(&c.bytesTx, uint64(n))
	}()

	wg.Wait()
}

func (c *Client) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.TLSInsecureSkipVerify,
	}

	if c.config.TLSCACertFile != "" {
		caCert, err := os.ReadFile(c.config.TLSCACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	if c.config.TLSCertFile != "" && c.config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLSCertFile, c.config.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client TLS key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func (c *Client) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"client_id":          c.config.ClientID,
		"reconnect_attempts": atomic.LoadUint64(&c.reconnectAttempts),
		"active_streams":     atomic.LoadInt32(&c.activeStreams),
		"bytes_tx":           atomic.LoadUint64(&c.bytesTx),
		"bytes_rx":           atomic.LoadUint64(&c.bytesRx),
	}
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
