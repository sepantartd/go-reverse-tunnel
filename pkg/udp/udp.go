package udp

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Packet represents a framed UDP packet sent over TCP stream
// Format: [2 bytes Length][Payload...]
func WritePacket(w io.Writer, payload []byte) error {
	length := uint16(len(payload))
	var header [2]byte
	binary.BigEndian.PutUint16(header[:], length)

	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func ReadPacket(r io.Reader) ([]byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header[:])
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// UDPProxy manages listening on a local UDP port and forwarding over stream
type UDPProxy struct {
	conn       *net.UDPConn
	stream     io.ReadWriteCloser
	targetAddr *net.UDPAddr
	mu         sync.Mutex
	closed     bool
}

func NewServerUDPProxy(listenPort int, stream io.ReadWriteCloser) (*UDPProxy, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return &UDPProxy{
		conn:   conn,
		stream: stream,
	}, nil
}

func (p *UDPProxy) StartServerForwarding() {
	defer p.conn.Close()
	defer p.stream.Close()

	// Goroutine 1: Read from UDP public listener and write framed packets to TCP stream
	go func() {
		buf := make([]byte, 65535)
		for {
			n, srcAddr, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			p.mu.Lock()
			p.targetAddr = srcAddr
			p.mu.Unlock()

			if err := WritePacket(p.stream, buf[:n]); err != nil {
				return
			}
		}
	}()

	// Goroutine 2: Read framed packets from TCP stream and respond back to UDP client
	for {
		payload, err := ReadPacket(p.stream)
		if err != nil {
			return
		}
		p.mu.Lock()
		target := p.targetAddr
		p.mu.Unlock()

		if target != nil {
			_, _ = p.conn.WriteToUDP(payload, target)
		}
	}
}

func PipeClientUDP(localUDPAddr string, stream io.ReadWriteCloser) error {
	raddr, err := net.ResolveUDPAddr("udp", localUDPAddr)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer stream.Close()

	// Goroutine 1: Read from TCP stream and write to local UDP target
	go func() {
		for {
			payload, err := ReadPacket(stream)
			if err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_, _ = conn.Write(payload)
		}
	}()

	// Read response from local UDP target and frame back to TCP stream
	buf := make([]byte, 65535)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		if err := WritePacket(stream, buf[:n]); err != nil {
			return err
		}
	}
}
