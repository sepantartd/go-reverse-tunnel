package obfuscate

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
)

const (
	MaxJunkSize = 256
)

// ObfuscatedConn wraps net.Conn to add traffic padding on initial handshake
type ObfuscatedConn struct {
	net.Conn
	handshakeDone bool
}

func NewObfuscatedConn(conn net.Conn) *ObfuscatedConn {
	return &ObfuscatedConn{Conn: conn}
}

// WriteJunk writes random length junk data into the connection
func (o *ObfuscatedConn) WriteJunk() error {
	var sizeBuf [1]byte
	if _, err := rand.Read(sizeBuf[:]); err != nil {
		return err
	}
	junkLen := int(sizeBuf[0])%MaxJunkSize + 1

	junk := make([]byte, 1+junkLen)
	junk[0] = byte(junkLen)
	if _, err := rand.Read(junk[1:]); err != nil {
		return err
	}

	_, err := o.Conn.Write(junk)
	return err
}

// ReadJunk discards the random junk bytes sent during handshake
func (o *ObfuscatedConn) ReadJunk() error {
	var sizeBuf [1]byte
	if _, err := io.ReadFull(o.Conn, sizeBuf[:]); err != nil {
		return fmt.Errorf("failed to read junk header size: %w", err)
	}

	junkLen := int(sizeBuf[0])
	if junkLen > 0 {
		discardBuf := make([]byte, junkLen)
		if _, err := io.ReadFull(o.Conn, discardBuf); err != nil {
			return fmt.Errorf("failed to discard junk payload: %w", err)
		}
	}
	return nil
}

// PerformObfuscatedHandshake executes bidirectional junk exchange
func PerformClientHandshake(conn net.Conn) (net.Conn, error) {
	obf := NewObfuscatedConn(conn)
	// Write client junk
	if err := obf.WriteJunk(); err != nil {
		return nil, fmt.Errorf("client junk write failed: %w", err)
	}
	// Read server junk
	if err := obf.ReadJunk(); err != nil {
		return nil, fmt.Errorf("client junk read failed: %w", err)
	}
	return obf, nil
}

func PerformServerHandshake(conn net.Conn) (net.Conn, error) {
	obf := NewObfuscatedConn(conn)
	// Read client junk first
	if err := obf.ReadJunk(); err != nil {
		return nil, fmt.Errorf("server junk read failed: %w", err)
	}
	// Reply with server junk
	if err := obf.WriteJunk(); err != nil {
		return nil, fmt.Errorf("server junk write failed: %w", err)
	}
	return obf, nil
}
