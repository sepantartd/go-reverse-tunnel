package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/server"
)

func TestDashboard_Auth(t *testing.T) {
	srvCfg := &config.ServerConfig{
		ControlAddr:   "127.0.0.1:0",
		Token:         "secret",
		DashboardUser: "admin",
		DashboardPass: "password123",
	}

	srv := server.NewTunnelServer(srvCfg)

	// Test Unauthenticated Request
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	srv.HandleDashboard(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", w.Code)
	}

	// Test Authenticated HTML Request
	reqAuth := httptest.NewRequest("GET", "/dashboard", nil)
	reqAuth.SetBasicAuth("admin", "password123")
	wAuth := httptest.NewRecorder()
	srv.HandleDashboard(wAuth, reqAuth)

	if wAuth.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wAuth.Code)
	}

	// Test Authenticated JSON Request
	reqJSON := httptest.NewRequest("GET", "/dashboard", nil)
	reqJSON.SetBasicAuth("admin", "password123")
	reqJSON.Header.Set("Accept", "application/json")
	wJSON := httptest.NewRecorder()
	srv.HandleDashboard(wJSON, reqJSON)

	if wJSON.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for JSON, got %d", wJSON.Code)
	}
}
