package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

type StatusResponse struct {
	ActiveClients int            `json:"active_clients"`
	Clients       []ClientStatus `json:"clients"`
}

type ClientStatus struct {
	ClientID   string `json:"client_id"`
	RemoteAddr string `json:"remote_addr"`
	Ports      []int  `json:"ports"`
}

func (s *TunnelServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// Basic Auth Middleware
	if s.config.DashboardUser != "" || s.config.DashboardPass != "" {
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(s.config.DashboardUser)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(s.config.DashboardPass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted Dashboard"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := StatusResponse{
		ActiveClients: len(s.clients),
		Clients:       make([]ClientStatus, 0, len(s.clients)),
	}

	for _, session := range s.clients {
		resp.Clients = append(resp.Clients, ClientStatus{
			ClientID:   session.ClientID,
			RemoteAddr: session.RemoteAddr,
			Ports:      session.Ports,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
