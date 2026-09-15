package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type StatusResponse struct {
	Status    string   `json:"status"`
	Uptime    string   `json:"uptime"`
	Clients   []string `json:"connected_clients"`
}

func (s *TunnelServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// Simple token protection via query parameter or header
	token := r.URL.Query().Get("token")
	if token != s.config.Token {
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
		Status:    "running",
		Uptime:    time.Since(s.startTime).String(),
		Clients:   clientIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(encodeWriter{w}).Encode(response)
}

type encodeWriter struct {
	http.ResponseWriter
}
