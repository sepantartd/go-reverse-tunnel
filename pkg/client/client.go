package client

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/obfuscate"
	"github.com/sepantartd/go-reverse-tunnel/pkg/udp"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

type AuthResponse struct {
	Status        string `json:"status"`
	AssignedPorts []int  `json:"assigned_ports"`
	Error         string `json:"error,omitempty"`
}

type TunnelClient struct {
	config *config.ClientConfig
	logger *slog.Logger
}

func NewTunnelClient(cfg *config.ClientConfig) *TunnelClient {
	logger := config.SetupLogger(cfg.LogLevel)
	return &TunnelClient{
		config: cfg,
		logger: logger,
	}
}

func (c *TunnelClient) Start() error {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		c.logger.Info("Connecting to server", slog.String("server_addr", c.config.ServerAddr))
		err := c.connectAndServe()
		if err != nil {
			c.logger.Error("Tunnel connection failed", slog.String("error", err.Error()))
		} else {
			c.logger.Warn("Tunnel connection closed, reconnecting...")
			backoff = 1 * time.Second
		}

		c.logger.Info("Waiting before reconnecting", slog.Duration("backoff", backoff))
		time.Sleep(backoff)

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *TunnelClient) connectAndServe() error {
	conn, err := net.DialTimeout("tcp", c.config.ServerAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}
	defer conn.Close()

	if c.config.EnableObfuscation {
		conn, err = obfuscate.PerformClientHandshake(conn)
		if err != nil {
			return fmt.Errorf("obfuscation handshake failed: %w", err)
		}
	}

	if c.config.EnableTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.config.InsecureSkipVerify,
			ServerName:         c.config.TLSServerName,
		}
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("TLS handshake failed: %w", err)
		}
		conn = tlsConn
	}

	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	nonce := make([]byte, 32)
	if _, err := io.ReadFull(conn, nonce); err != nil {
		return fmt.Errorf("failed to read nonce from server: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(c.config.Token))
	mac.Write(nonce)
	expectedHMAC := hex.EncodeToString(mac.Sum(nil))

	authPayload := fmt.Sprintf("%s:%s\n", c.config.ClientID, expectedHMAC)
	if _, err := conn.Write([]byte(authPayload)); err != nil {
		return fmt.Errorf("failed to send auth payload: %w", err)
	}

	reader := bufio.NewReader(conn)
	respLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read auth response from server: %w", err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal([]byte(respLine), &authResp); err != nil {
		return fmt.Errorf("invalid json response from server: %w", err)
	}

	if authResp.Status != "ok" {
		return fmt.Errorf("server authentication rejected: %s", authResp.Error)
	}

	_ = conn.SetDeadline(time.Time{})

	c.logger.Info("Authenticated successfully with server!",
		slog.String("client_id", c.config.ClientID),
		slog.Any("assigned_ports", authResp.AssignedPorts))

	yamuxCfg := config.GetYamuxConfig(c.config.Yamux)
	session, err := yamux.Client(conn, yamuxCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize yamux client: %w", err)
	}
	defer session.Close()

	if len(c.config.UDPForwards) > 0 {
		for _, uConfig := range c.config.UDPForwards {
			go c.handleUDPClientSession(session, uConfig)
		}
	}

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return fmt.Errorf("yamux session accept stream closed: %w", err)
		}

		go c.handleStream(stream)
	}
}

func (c *TunnelClient) handleUDPClientSession(session *yamux.Session, uConfig config.UDPForwardConfig) {
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			c.logger.Error("Failed to accept stream for UDP client proxy", slog.String("error", err.Error()))
			return
		}

		clientUDPProxy, err := udp.NewClientUDPProxy(uConfig.LocalTarget, stream)
		if err != nil {
			c.logger.Error("Failed to create UDP client proxy target", slog.String("target", uConfig.LocalTarget), slog.String("error", err.Error()))
			stream.Close()
			continue
		}

		clientUDPProxy.StartClientForwarding()
	}
}

func (c *TunnelClient) handleStream(stream net.Conn) {
	defer stream.Close()

	localConn, err := net.DialTimeout("tcp", c.config.LocalTarget, 10*time.Second)
	if err != nil {
		c.logger.Error("Failed to connect to local target", slog.String("target", c.config.LocalTarget), slog.String("error", err.Error()))
		return
	}
	defer localConn.Close()

	idleTimeout := 5 * time.Minute

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		_ = stream.SetReadDeadline(time.Now().Add(idleTimeout))
		_ = localConn.SetWriteDeadline(time.Now().Add(idleTimeout))

		_, _ = io.CopyBuffer(localConn, stream, *bufPtr)
	}()

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		_ = localConn.SetReadDeadline(time.Now().Add(idleTimeout))
		_ = stream.SetWriteDeadline(time.Now().Add(idleTimeout))

		_, _ = io.CopyBuffer(stream, localConn, *bufPtr)
	}()

	wg.Wait()
}
