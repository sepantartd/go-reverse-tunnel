package protocol

import (
"crypto/hmac"
"crypto/rand"
"crypto/sha256"
"encoding/hex"
"errors"
"fmt"
"io"
)

const NonceSize = 32

// GenerateNonce creates a cryptographically secure random challenge
func GenerateNonce() ([]byte, error) {
nonce := make([]byte, NonceSize)
_, err := io.ReadFull(rand.Reader, nonce)
if err != nil {
return nil, fmt.Errorf("failed to generate nonce: %w", err)
}
return nonce, nil
}

// ComputeHMAC calculates HMAC-SHA256 for the given message and token
func ComputeHMAC(message []byte, token string) string {
mac := hmac.New(sha256.New, []byte(token))
mac.Write(message)
return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC checks if the received HMAC matches the expected signature
func VerifyHMAC(message []byte, receivedHMAC string, token string) bool {
expectedHMAC := ComputeHMAC(message, token)
return hmac.Equal([]byte(expectedHMAC), []byte(receivedHMAC))
}

// ClientAuthRequest represents the handshake payload from client to server
type AuthHandshake struct {
ClientID string `json:"client_id"`
Response string `json:"response"`
}

// ServerChallenge represents the challenge sent from server to client
type Challenge struct {
NonceHex string `json:"nonce_hex"`
}

var (
ErrAuthFailed = errors.New("authentication failed: invalid HMAC signature")
)
