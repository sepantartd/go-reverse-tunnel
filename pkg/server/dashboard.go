package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DashboardStats struct {
	Uptime        string   `json:"uptime"`
	ActiveClients int      `json:"active_clients"`
	ClientIDs     []string `json:"client_ids"`
}

func (s *TunnelServer) StartDashboard(port int, startTime time.Time) {
	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		clientIDs := make([]string, 0, len(s.clients))
		for id := range s.clients {
			clientIDs = append(clientIDs, id)
		}
		s.mu.RUnlock()

		stats := DashboardStats{
			Uptime:        time.Since(startTime).Round(time.Second).String(),
			ActiveClients: len(clientIDs),
			ClientIDs:     clientIDs,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Go-Reverse-Tunnel Dashboard</title>
<style>
body { font-family: sans-serif; background: #121212; color: #fff; padding: 2rem; }
.card { background: #1e1e1e; padding: 1.5rem; border-radius: 8px; margin-bottom: 1rem; }
h1 { color: #4caf50; }
</style>
</head>
<body>
<h1>Go Reverse Tunnel Dashboard</h1>
<div class="card">
  <p><strong>Uptime:</strong> %s</p>
  <p><strong>Active Clients:</strong> %d</p>
</div>
</body>
</html>`, time.Since(startTime).Round(time.Second).String(), len(s.clients))
		_, _ = w.Write([]byte(html))
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	_ = http.ListenAndServe(addr, nil)
}
