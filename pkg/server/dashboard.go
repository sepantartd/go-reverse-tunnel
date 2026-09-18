package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HandleDashboard renders an HTML interface showing real-time tunnel statistics.
// Access requires authentication via 'token' query param or 'Authorization: Bearer <token>' header.
func (s *TunnelServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	authToken := r.URL.Query().Get("token")
	if authToken == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			authToken = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if authToken == "" || authToken != s.config.Token {
		http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
		return
	}

	s.mu.RLock()
	clientCount := len(s.clients)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Go Reverse Tunnel Dashboard</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #f4f6f9; margin: 0; padding: 40px; }
        .card { background: #ffffff; max-width: 600px; margin: 0 auto; padding: 30px; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.08); }
        h1 { color: #1a202c; font-size: 24px; margin-bottom: 20px; border-bottom: 2px solid #edf2f7; padding-bottom: 10px; }
        .stat { font-size: 16px; margin: 12px 0; color: #4a5568; }
        .stat strong { color: #2d3748; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Go Reverse Tunnel Status</h1>
        <div class="stat"><strong>Active Clients:</strong> %d</div>
        <div class="stat"><strong>Server Time:</strong> %s</div>
    </div>
</body>
</html>`, clientCount, time.Now().Format(time.RFC1123))
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
