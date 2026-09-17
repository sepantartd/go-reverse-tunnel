package server

import (
	"crypto/subtle"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
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

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Reverse Tunnel - Management Console</title>
    <style>
        :root {
            --bg-color: #0f172a;
            --card-bg: #1e293b;
            --accent-color: #38bdf8;
            --text-color: #f8fafc;
            --text-muted: #94a3b8;
            --border-color: #334155;
            --success-color: #22c55e;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
        body { background-color: var(--bg-color); color: var(--text-color); padding: 2rem; line-height: 1.5; }
        .container { max-width: 1100px; margin: 0 auto; }
        header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; border-bottom: 1px solid var(--border-color); padding-bottom: 1rem; }
        h1 { font-size: 1.5rem; font-weight: 600; color: var(--accent-color); display: flex; align-items: center; gap: 0.5rem; }
        .badge { background: rgba(56, 189, 248, 0.1); color: var(--accent-color); padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.85rem; font-weight: 500; border: 1px solid rgba(56, 189, 248, 0.2); }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.5rem; margin-bottom: 2rem; }
        .card { background-color: var(--card-bg); border: 1px solid var(--border-color); border-radius: 0.75rem; padding: 1.5rem; }
        .card-title { color: var(--text-muted); font-size: 0.875rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; margin-bottom: 0.5rem; }
        .card-value { font-size: 2rem; font-weight: 700; color: var(--text-color); }
        .table-container { background-color: var(--card-bg); border: 1px solid var(--border-color); border-radius: 0.75rem; overflow: hidden; }
        table { width: 100%; border-collapse: collapse; text-align: left; }
        th { background-color: rgba(15, 23, 42, 0.5); padding: 1rem; color: var(--text-muted); font-weight: 600; font-size: 0.85rem; border-bottom: 1px solid var(--border-color); }
        td { padding: 1rem; border-bottom: 1px solid var(--border-color); font-size: 0.9rem; }
        tr:last-child td { border-bottom: none; }
        .status-dot { display: inline-block; width: 8px; height: 8px; background-color: var(--success-color); border-radius: 50%; margin-right: 0.5rem; box-shadow: 0 0 8px var(--success-color); }
        .port-tag { background: #334155; color: #f1f5f9; padding: 0.2rem 0.5rem; border-radius: 0.375rem; font-size: 0.8rem; font-family: monospace; margin-right: 0.3rem; }
        .empty-state { padding: 3rem; text-align: center; color: var(--text-muted); }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>⚡ Reverse Tunnel Console</h1>
            <span class="badge">v1.0.0 Active</span>
        </header>

        <div class="grid">
            <div class="card">
                <div class="card-title">Active Clients</div>
                <div class="card-value" id="active-clients">{{.ActiveClients}}</div>
            </div>
            <div class="card">
                <div class="card-title">System Health</div>
                <div class="card-value" style="color: var(--success-color);">Optimal</div>
            </div>
        </div>

        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>CLIENT ID</th>
                        <th>REMOTE ADDRESS</th>
                        <th>FORWARDED PORTS</th>
                        <th>STATUS</th>
                    </tr>
                </thead>
                <tbody id="client-rows">
                    {{range .Clients}}
                    <tr>
                        <td style="font-weight: 600;">{{.ClientID}}</td>
                        <td style="font-family: monospace; color: var(--text-muted);">{{.RemoteAddr}}</td>
                        <td>
                            {{range .Ports}}
                            <span class="port-tag">:{{.}}</span>
                            {{end}}
                        </td>
                        <td><span class="status-dot"></span>Connected</td>
                    </tr>
                    {{else}}
                    <tr>
                        <td colspan="4" class="empty-state">No active client sessions connected.</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>

    <script>
        // Auto-refresh dashboard data every 3 seconds
        setInterval(async () => {
            try {
                const res = await fetch(window.location.href, { headers: { 'Accept': 'application/json' } });
                if (res.ok) {
                    const data = await res.json();
                    document.getElementById('active-clients').innerText = data.active_clients;
                    
                    const tbody = document.getElementById('client-rows');
                    if (data.clients.length === 0) {
                        tbody.innerHTML = '<tr><td colspan="4" class="empty-state">No active client sessions connected.</td></tr>';
                        return;
                    }
                    
                    tbody.innerHTML = data.clients.map(c => `
                        <tr>
                            <td style="font-weight: 600;">${c.client_id}</td>
                            <td style="font-family: monospace; color: var(--text-muted);">${c.remote_addr}</td>
                            <td>${c.ports.map(p => `<span class="port-tag">:${p}</span>`).join('')}</td>
                            <td><span class="status-dot"></span>Connected</td>
                        </tr>
                    `).join('');
                }
            } catch (e) {
                console.error("Failed to fetch status updates", e);
            }
        }, 3000);
    </script>
</body>
</html>`

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
	s.mu.RUnlock()

	// Handle JSON API Request
	if strings.Contains(r.Header.Get("Accept"), "application/json") || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Render HTML Dashboard
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, resp)
}
