package protocol

import (
	"net"
	"testing"
)

func TestAuthentication_Success(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	token := "secret_auth_token"
	expectedClientID := "client_001"

	errChan := make(chan error, 1)
	go func() {
		errChan <- ClientAuthenticate(clientConn, expectedClientID, token)
	}()

	authenticatedID, err := ServerAuthenticate(serverConn, token)
	if err != nil {
		t.Fatalf("ServerAuthenticate failed: %v", err)
	}

	if authenticatedID != expectedClientID {
		t.Errorf("got clientID %s, want %s", authenticatedID, expectedClientID)
	}

	if err := <-errChan; err != nil {
		t.Fatalf("ClientAuthenticate failed: %v", err)
	}
}

func TestAuthentication_InvalidToken(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	serverToken := "correct_token"
	clientToken := "wrong_token"

	errChan := make(chan error, 1)
	go func() {
		errChan <- ClientAuthenticate(clientConn, "client_001", clientToken)
	}()

	_, err := ServerAuthenticate(serverConn, serverToken)
	if err == nil {
		t.Error("expected server authentication to fail with invalid token, got nil")
	}

	clientErr := <-errChan
	if clientErr == nil {
		t.Error("expected client authentication to be rejected by server, got nil")
	}
}

func TestAuthentication_InvalidClientIDLength(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	err := ClientAuthenticate(clientConn, "", "token")
	if err == nil {
		t.Error("expected error for empty clientID, got nil")
	}
}
