package protocol

import (
	"testing"
)

func TestHMACAuth(t *testing.T) {
	token := "valid-secret-token"
	nonce, _ := GenerateNonce()

	// 1. Test Valid HMAC
	resp := ComputeHMAC(nonce, token)
	if !VerifyHMAC(nonce, resp, token) {
		t.Error("Valid HMAC failed verification")
	}

	// 2. Test Invalid HMAC
	if VerifyHMAC(nonce, resp, "wrong-token") {
		t.Error("Invalid HMAC passed verification")
	}
}
