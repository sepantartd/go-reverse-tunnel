package client

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/protocol"
	"github.com/sepantartd/go-reverse-tunnel/pkg/tunnel"
)

// RunClient starts the robust reverse tunnel client loop with exponential backoff & jitter.
// It respects context cancellation for graceful shutdown.
func RunClient(ctx context.Context, cfg *config.ClientConfig) error {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second
	factor := 2.0
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("[Client] Context canceled, stopping client loop...")
			return nil
		default:
		}

		attempts++
		log.Printf("[Client] Connecting to tunnel server at %s (Attempt %d)...", cfg.ServerAddr, attempts)

		var conn net.Conn
		var err error

		dialer := &net.Dialer{Timeout: 10 * time.Second}

		if cfg.EnableTLS {
			var tlsConfig *tls.Config
			tlsConfig, err = protocol.GetClientTLSConfig(cfg.CAFile, cfg.CertFile, cfg.KeyFile, cfg.TLSSkipVerify)
			if err != nil {
				log.Printf("[Client] Failed to build client TLS config: %v", err)
				if !sleepWithContext(ctx, 5*time.Second) {
					return nil
				}
				continue
			}
			tlsDialer := &tls.Dialer{NetDialer: dialer, Config: tlsConfig}
			conn, err = tlsDialer.DialContext(ctx, "tcp", cfg.ServerAddr)
		} else {
			conn, err = dialer.DialContext(ctx, "tcp", cfg.ServerAddr)
		}

		if err != nil {
			log.Printf("[Client] Connection failed: %v", err)
			sleepDuration := calculateBackoffWithJitter(attempts, baseDelay, maxDelay, factor)
			log.Printf("[Client] Waiting %v before next reconnection attempt...", sleepDuration)
			if !sleepWithContext(ctx, sleepDuration) {
				return nil
			}
			continue
		}

		// Reset attempts on successful connection
		attempts = 0
		log.Println("[Client] Connected to server. Performing authentication handshake...")

		// Perform secure HMAC-SHA256 authentication
		err = protocol.ClientAuthenticate(conn, cfg.ClientID, cfg.Token)
		if err != nil {
			log.Printf("[Client] Authentication failed: %v", err)
			conn.Close()
			if !sleepWithContext(ctx, 5*time.Second) {
				return nil
			}
			continue
		}

		log.Println("[Client] Authentication successful! Establishing multiplexed tunnel session...")

		clientSess, err := tunnel.NewClientSession(conn)
		if err != nil {
			log.Printf("[Client] Failed to create tunnel session: %v", err)
			conn.Close()
			if !sleepWithContext(ctx, 3*time.Second) {
				return nil
			}
			continue
		}

		// Monitor context cancellation to close active session promptly
		sessionDone := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				clientSess.Close()
			case <-sessionDone:
			}
		}()

		targetAddr := cfg.LocalAddr
		if targetAddr == "" {
			targetAddr = "127.0.0.1:80"
		}

		// Handle incoming streams from server
		for {
			stream, err := clientSess.AcceptStream()
			if err != nil {
				log.Printf("[Client] Tunnel session stream accepted error or dropped: %v", err)
				break
			}

			go handleTunnelStream(stream, targetAddr)
		}

		close(sessionDone)
		clientSess.Close()
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func handleTunnelStream(stream io.ReadWriteCloser, target string) {
	defer stream.Close()

	localConn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Printf("[Client] Failed to connect to local target %s: %v", target, err)
		return
	}
	defer localConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(localConn, stream)
		_ = localConn.Close()
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(stream, localConn)
		_ = stream.Close()
		done <- struct{}{}
	}()

	<-done
	<-done
}

func calculateBackoffWithJitter(attempt int, base, max time.Duration, factor float64) time.Duration {
	d := float64(base)
	for i := 1; i < attempt; i++ {
		d *= factor
		if d > float64(max) {
			d = float64(max)
			break
		}
	}
	jitter := (rand.Float64()*0.4 - 0.2) * d
	finalDuration := d + jitter
	if finalDuration < float64(base) {
		return base
	}
	if finalDuration > float64(max) {
		return max
	}
	return time.Duration(finalDuration)
}
