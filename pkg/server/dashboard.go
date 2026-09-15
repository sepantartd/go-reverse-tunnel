package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type StatusResponse struct {
	Status  string   `json:"status"`
	Uptime  string   `json:"uptime"`
	Clients []string `json:"connected_clients"`
}

func (s *TunnelServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	var token string

	// 1. Check Authorization Bearer header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 2. Fallback to X-API-Key header
	if token == "" {
		token = r.Header.Get("X-API-Key")
	}

	// 3. Fallback to query parameter
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	// Constant-time compare to mitigate timing attacks
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.config.Token)) != 1 {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	s.mu.RLock()
	clientIDs := make([]string, 0, len(s.clients))
	for id := range s.clients {
		clientIDs = append(clientIDs, id)
	}
	s.mu.RUnlock()

	response := StatusResponse{
		Status:  "running",
		Uptime:  time.Since(s.startTime).String(),
		Clients: clientIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
