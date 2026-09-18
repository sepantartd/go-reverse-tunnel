package udp

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

type ServerUDPProxy struct {
	port   int
	stream net.Conn
	logger *slog.Logger
}

func NewServerUDPProxy(port int, stream net.Conn) (*ServerUDPProxy, error) {
	return &ServerUDPProxy{
		port:   port,
		stream: stream,
		logger: slog.Default(),
	}, nil
}

func (p *ServerUDPProxy) StartServerForwarding() {
	defer p.stream.Close()

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		p.logger.Error("Failed to resolve UDP address", slog.Int("port", p.port), slog.String("error", err.Error()))
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		p.logger.Error("Failed to listen on UDP port", slog.Int("port", p.port), slog.String("error", err.Error()))
		return
	}
	defer udpConn.Close()

	idleTimeout := 5 * time.Minute

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		for {
			_ = p.stream.SetReadDeadline(time.Now().Add(idleTimeout))
			n, err := p.stream.Read(*bufPtr)
			if err != nil {
				break
			}
			_, _ = udpConn.Write((*bufPtr)[:n])
		}
	}()

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		for {
			_ = udpConn.SetReadDeadline(time.Now().Add(idleTimeout))
			n, _, err := udpConn.ReadFromUDP(*bufPtr)
			if err != nil {
				break
			}
			_ = p.stream.SetWriteDeadline(time.Now().Add(idleTimeout))
			_, err = p.stream.Write((*bufPtr)[:n])
			if err != nil {
				break
			}
		}
	}()

	wg.Wait()
}

type ClientUDPProxy struct {
	target string
	stream net.Conn
	logger *slog.Logger
}

func NewClientUDPProxy(target string, stream net.Conn) (*ClientUDPProxy, error) {
	return &ClientUDPProxy{
		target: target,
		stream: stream,
		logger: slog.Default(),
	}, nil
}

func (p *ClientUDPProxy) StartClientForwarding() {
	defer p.stream.Close()

	udpAddr, err := net.ResolveUDPAddr("udp", p.target)
	if err != nil {
		p.logger.Error("Failed to resolve target UDP address", slog.String("target", p.target), slog.String("error", err.Error()))
		return
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		p.logger.Error("Failed to dial target UDP", slog.String("target", p.target), slog.String("error", err.Error()))
		return
	}
	defer udpConn.Close()

	idleTimeout := 5 * time.Minute

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		for {
			_ = p.stream.SetReadDeadline(time.Now().Add(idleTimeout))
			n, err := p.stream.Read(*bufPtr)
			if err != nil {
				break
			}
			_, _ = udpConn.Write((*bufPtr)[:n])
		}
	}()

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		for {
			_ = udpConn.SetReadDeadline(time.Now().Add(idleTimeout))
			n, err := udpConn.Read(*bufPtr)
			if err != nil {
				break
			}
			_ = p.stream.SetWriteDeadline(time.Now().Add(idleTimeout))
			_, err = p.stream.Write((*bufPtr)[:n])
			if err != nil {
				break
			}
		}
	}()

	wg.Wait()
}
