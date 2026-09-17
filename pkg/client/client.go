package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
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
)

var clientBufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func RunClient(ctx context.Context, cfg *config.ClientConfig) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := connectAndServe(ctx, cfg)
		if err != nil {
			logger.Error("Client connection error", slog.String("error", err.Error()), slog.Duration("reconnect_in", backoff))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func connectAndServe(ctx context.Context, cfg *config.ClientConfig) error {
	var conn net.Conn
	var err error

	if cfg.TLSCertFile != "" || cfg.InsecureSkipVerify {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
		conn, err = tls.Dial("tcp", cfg.ServerAddr, tlsCfg)
	} else {
		conn, err = net.Dial("tcp", cfg.ServerAddr)
	}

	if err != nil {
		return fmt.Errorf("failed to dial server: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read challenge: %v", err)
	}
	challenge := buf[:n]

	mac := hmac.New(sha256.New, []byte(cfg.Token))
	mac.Write(challenge)
	response := hex.EncodeToString(mac.Sum(nil)) + "\n"

	if _, err := conn.Write([]byte(response)); err != nil {
		return fmt.Errorf("failed to send HMAC response: %v", err)
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("failed to create yamux client session: %v", err)
	}
	defer session.Close()

	logger.Info("Connected to tunnel server successfully", slog.String("server_addr", cfg.ServerAddr))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		stream, err := session.AcceptStream()
		if err != nil {
			return fmt.Errorf("session closed or failed: %v", err)
		}

		go handleStream(stream, cfg.LocalAddr)
	}
}

func handleStream(stream *yamux.Stream, localAddr string) {
	defer stream.Close()

	targetConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		logger.Error("Failed to connect to local target", slog.String("local_addr", localAddr), slog.String("error", err.Error()))
		return
	}
	defer targetConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := clientBufferPool.Get().(*[]byte)
		defer clientBufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(targetConn, stream, *bufPtr)
	}()

	go func() {
		defer wg.Done()
		bufPtr := clientBufferPool.Get().(*[]byte)
		defer clientBufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(stream, targetConn, *bufPtr)
	}()

	wg.Wait()
}
