package protocol

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"net"
)

// ServerAuthenticate performs a secure HMAC-SHA256 handshake with a random nonce on the server side.
// Protocol flow:
// 1. Server sends 32-byte random nonce to client.
// 2. Client computes HMAC-SHA256(token, nonce + clientID) and sends [ClientIDLen(1)][ClientID][HMAC(32)].
// 3. Server verifies HMAC and responds with ACK (1) or NAK (0).
func ServerAuthenticate(conn net.Conn, expectedToken string) (string, error) {
	// Generate 32 bytes of secure random nonce
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	// Send nonce to client
	if _, err := conn.Write(nonce); err != nil {
		return "", err
	}

	// Read client ID length header
	header := make([]byte, 1)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}
	idLen := int(header[0])
	if idLen == 0 {
		conn.Write([]byte{0})
		return "", errors.New("authentication failed: invalid client ID length")
	}

	// Read ClientID and HMAC signature
	buf := make([]byte, idLen+32)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	clientID := string(buf[:idLen])
	clientHMAC := buf[idLen:]

	// Compute expected HMAC
	mac := hmac.New(sha256.New, []byte(expectedToken))
	mac.Write(nonce)
	mac.Write([]byte(clientID))
	expectedHMAC := mac.Sum(nil)

	// Verify signature in constant time
	if !hmac.Equal(clientHMAC, expectedHMAC) {
		conn.Write([]byte{0}) // Send failure ACK
		return "", errors.New("authentication failed: invalid HMAC signature")
	}

	// Send success ACK
	if _, err := conn.Write([]byte{1}); err != nil {
		return "", err
	}

	return clientID, nil
}

// ClientAuthenticate performs the client-side HMAC-SHA256 authentication handshake.
func ClientAuthenticate(conn net.Conn, clientID string, token string) error {
	// Read nonce from server
	nonce := make([]byte, 32)
	if _, err := io.ReadFull(conn, nonce); err != nil {
		return err
	}

	idBytes := []byte(clientID)
	if len(idBytes) == 0 || len(idBytes) > 255 {
		return errors.New("invalid clientID length (must be between 1 and 255 bytes)")
	}

	// Compute HMAC: HMAC-SHA256(token, nonce + clientID)
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write(nonce)
	mac.Write(idBytes)
	clientHMAC := mac.Sum(nil)

	// Construct packet: [ClientIDLen(1)][ClientID][HMAC(32)]
	packet := make([]byte, 1+len(idBytes)+32)
	packet[0] = byte(len(idBytes))
	copy(packet[1:], idBytes)
	copy(packet[1+len(idBytes):], clientHMAC)

	// Send authentication packet
	if _, err := conn.Write(packet); err != nil {
		return err
	}

	// Read server ACK response
	ack := make([]byte, 1)
	if _, err := io.ReadFull(conn, ack); err != nil {
		return err
	}

	if ack[0] != 1 {
		return errors.New("authentication rejected by server")
	}

	return nil
}
