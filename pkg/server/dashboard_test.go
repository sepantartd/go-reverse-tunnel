package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func TestHandleDashboard_Auth(t *testing.T) {
	srv := &TunnelServer{
		config:    &config.ServerConfig{Token: "secret123"},
		startTime: time.Now(),
	}

	tests := []struct {
		name       string
		headerKey  string
		headerVal  string
		queryParam string
		wantStatus int
	}{
		{
			name:       "Valid Bearer Token Header",
			headerKey:  "Authorization",
			headerVal:  "Bearer secret123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Valid X-API-Key Header",
			headerKey:  "X-API-Key",
			headerVal:  "secret123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Valid Query Param (Fallback)",
			queryParam: "?token=secret123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid Token",
			headerKey:  "Authorization",
			headerVal:  "Bearer wrongtoken",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Missing Token",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqUrl := "/status" + tt.queryParam
			req := httptest.NewRequest("GET", reqUrl, nil)
			if tt.headerKey != "" {
				req.Header.Set(tt.headerKey, tt.headerVal)
			}

			rec := httptest.NewRecorder()
			srv.HandleDashboard(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
