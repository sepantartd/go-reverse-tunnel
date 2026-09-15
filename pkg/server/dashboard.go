package server

import (
"encoding/json"
"net/http"
"time"
)

// HandleDashboard serves a secure, token-protected status and health endpoint
func (s *TunnelServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
// Simple query token authentication protection
token := r.URL.Query().Get("token")
if token == "" || (s.config != nil && token != s.config.Token) {
http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
return
}

s.mu.RLock()
clientCount := len(s.clients)
uptime := time.Since(s.startTime).String()
s.mu.RUnlock()

statusInfo := map[string]interface{}{
"status":         "healthy",
"active_clients": clientCount,
"uptime":         uptime,
"timestamp":      time.Now().Format(time.RFC3339),
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(statusInfo)
}
