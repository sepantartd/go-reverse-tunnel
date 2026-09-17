package client

import (
	"context"
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
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/obfuscate"
)

func RunClient(ctx context.Context, cfg *config.ClientConfig) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			logger.Info("Attempting connection to control server", slog.String("addr", cfg.ServerAddr))
			err := connectAndServe(ctx, cfg, logger)
			if err != nil {
				logger.Error("Client error, retrying in 5 seconds...", slog.String("error", err.Error()))
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func connectAndServe(ctx context.Context, cfg *config.ClientConfig, logger *slog.Logger) error {
	var conn net.Conn
	var err error

	if cfg.TLSCertFile != "" || !cfg.InsecureAllowPlaintext {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}

		if cfg.TLSCAFile != "" {
			caCert, err := os.ReadFile(cfg.TLSCAFile)
			if err != nil {
				return fmt.Errorf("failed to read CA cert: %v", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsCfg.RootCAs = caCertPool
		}

		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
			if err != nil {
				return fmt.Errorf("failed to load client certificate: %v", err)
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		conn, err = tls.Dial("tcp", cfg.ServerAddr, tlsCfg)
	} else {
		conn, err = net.Dial("tcp", cfg.ServerAddr)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	if cfg.EnableObfuscation {
		conn, err = obfuscate.PerformClientHandshake(conn)
		if err != nil {
			return fmt.Errorf("obfuscation handshake failed: %v", err)
		}
	}

	challenge := make([]byte, 1024)
	n, err := conn.Read(challenge)
	if err != nil {
		return fmt.Errorf("failed to read challenge nonce: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(cfg.Token))
	mac.Write(challenge[:n])
	response := hex.EncodeToString(mac.Sum(nil))

	if _, err := conn.Write([]byte(response + "\n")); err != nil {
		return fmt.Errorf("failed to send auth response: %v", err)
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("failed to create yamux client session: %v", err)
	}
	defer session.Close()

	logger.Info("Established multiplexed session with server")

	go func() {
		<-ctx.Done()
		_ = session.Close()
	}()

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return fmt.Errorf("session closed or failed accepting stream: %v", err)
		}
		go handleIncomingStream(stream, cfg.LocalAddr, logger)
	}
}

func handleIncomingStream(stream net.Conn, localAddr string, logger *slog.Logger) {
	defer stream.Close()

	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		logger.Error("Failed to connect to local target service", slog.String("local_addr", localAddr), slog.String("error", err.Error()))
		return
	}
	defer localConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(localConn, stream)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(stream, localConn)
	}()

	wg.Wait()
}
