package tunnel

import (
	"net"
	"runtime"
	"testing"
	"time"
)

func TestForwardUDPOverTCP_GoroutineLeak(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to resolve udp addr: %v", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("failed to listen udp: %v", err)
	}

	server, client := net.Pipe()

	done := make(chan struct{})
	go func() {
		ForwardUDPOverTCP(udpConn, server)
		close(done)
	}()

	// Simulate unexpected TCP disconnect
	client.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ForwardUDPOverTCP blocked indefinitely and did not terminate")
	}

	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines+1 {
		t.Errorf("possible goroutine leak: initial=%d, final=%d", initialGoroutines, finalGoroutines)
	}
}
