package protocol

import (
	"net"
	"testing"
	"time"
)

func TestTLSConfig_CipherSuites(t *testing.T) {
	cfg, err := GetClientTLSConfig("", "", "", true)
	if err != nil {
		t.Fatalf("failed to get client tls config: %v", err)
	}

	if len(cfg.CipherSuites) == 0 {
		t.Errorf("expected explicit CipherSuites in TLS config")
	}
}

func TestServerHandshakeWithTimeout_Timeout(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	srvCfg, err := GetServerTLSConfig("../../cert.pem", "../../key.pem", "")
	if err != nil {
		t.Skip("skipping test; cert.pem/key.pem not found in root")
	}

	errChan := make(chan error, 1)
	go func() {
		_, err := ServerHandshakeWithTimeout(server, srvCfg, 100*time.Millisecond)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		if err == nil {
			t.Errorf("expected handshake to time out, got nil error")
		}
	case <-time.After(1 * time.Second):
		t.Errorf("handshake did not time out within expected duration")
	}
}
