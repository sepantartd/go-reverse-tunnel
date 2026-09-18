package udp

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type ServerUDPProxy struct {
	port       int
	udpConn    *net.UDPConn
	stream     io.ReadWriteCloser
	clients    map[string]*net.UDPAddr
	clientsMu  sync.RWMutex
	closeCtx   chan struct{}
	closeOnce  sync.Once
}

func NewServerUDPProxy(port int, stream io.ReadWriteCloser) (*ServerUDPProxy, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen UDP on port %d: %w", port, err)
	}

	return &ServerUDPProxy{
		port:     port,
		udpConn:  conn,
		stream:   stream,
		clients:  make(map[string]*net.UDPAddr),
		closeCtx: make(chan struct{}),
	}, nil
}

func (p *ServerUDPProxy) StartServerForwarding() {
	defer p.Close()

	// Ingress: Public UDP Clients -> Multiplex Stream
	go func() {
		buf := make([]byte, 65535)
		for {
			n, srcAddr, err := p.udpConn.ReadFromUDP(buf)
			if err != nil {
				select {
				case <-p.closeCtx:
					return
				default:
					return
				}
			}

			// Store / Update client address mapping safely
			p.clientsMu.Lock()
			p.clients[srcAddr.String()] = srcAddr
			p.clientsMu.Unlock()

			// Forward payload to multiplex stream
			_ = p.stream.SetWriteDeadline(time.Now().Add(10 * time.Second))
			_, err = p.stream.Write(buf[:n])
			if err != nil {
				return
			}
		}
	}()

	// Egress: Multiplex Stream -> Public UDP Clients
	buf := make([]byte, 65535)
	for {
		_ = p.stream.SetReadDeadline(time.Now().Add(5 * time.Minute))
		n, err := p.stream.Read(buf)
		if err != nil {
			return
		}

		p.clientsMu.RLock()
		// Forward returned response to all active mapped clients
		for _, clientAddr := range p.clients {
			_, _ = p.udpConn.WriteToUDP(buf[:n], clientAddr)
		}
		p.clientsMu.RUnlock()
	}
}

func (p *ServerUDPProxy) Close() error {
	p.closeOnce.Do(func() {
		close(p.closeCtx)
		if p.udpConn != nil {
			_ = p.udpConn.Close()
		}
		if p.stream != nil {
			_ = p.stream.Close()
		}
	})
	return nil
}
