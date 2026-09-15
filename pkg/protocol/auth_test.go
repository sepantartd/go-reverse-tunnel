package protocol

import (
"bytes"
"testing"
)

func TestGenerateNonce(t *testing.T) {
nonce1, err := GenerateNonce()
if err != nil {
t.Fatalf("Failed to generate nonce: %v", err)
}
if len(nonce1) != 32 {
t.Errorf("Expected nonce length 32, got %d", len(nonce1))
}

nonce2, err := GenerateNonce()
if err != nil {
t.Fatalf("Failed to generate second nonce: %v", err)
}

if bytes.Equal(nonce1, nonce2) {
t.Error("Nonces should be unique and random")
}
}

func TestVerifyHMAC_Success(t *testing.T) {
token := "my-secret-token-123"
nonce, err := GenerateNonce()
if err != nil {
t.Fatalf("Failed to generate nonce: %v", err)
}

resp := ComputeHMAC(nonce, token)
if !VerifyHMAC(nonce, resp, token) {
t.Error("HMAC verification failed for valid token and nonce")
}
}

func TestVerifyHMAC_InvalidToken(t *testing.T) {
correctToken := "correct-token"
wrongToken := "wrong-token"

nonce, err := GenerateNonce()
if err != nil {
t.Fatalf("Failed to generate nonce: %v", err)
}

resp := ComputeHMAC(nonce, wrongToken)
if VerifyHMAC(nonce, resp, correctToken) {
t.Error("HMAC verification should have failed for invalid token")
}
}
