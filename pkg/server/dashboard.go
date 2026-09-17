package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sepantartd/go-reverse-tunnel/pkg/metrics"
)

var startTime = time.Now()

func (s *TunnelServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uptime := time.Since(startTime).Round(time.Second)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Go Reverse Tunnel Dashboard</title>
    <meta style="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem; margin: 0; }
        .container { max-width: 900px; margin: 0 auto; }
        h1 { color: #38bdf8; border-bottom: 2px solid #334155; padding-bottom: 0.5rem; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
        .card { background: #1e293b; padding: 1.5rem; border-radius: 8px; border: 1px solid #334155; }
        .card h3 { margin: 0 0 0.5rem 0; font-size: 0.875rem; color: #94a3b8; text-transform: uppercase; }
        .card .value { font-size: 1.875rem; font-weight: bold; color: #f1f5f9; }
        table { width: 100%%; border-collapse: collapse; background: #1e293b; border-radius: 8px; overflow: hidden; }
        th, td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid #334155; }
        th { background: #334155; color: #cbd5e1; font-weight: 600; }
        tr:last-child td { border-bottom: none; }
        .badge { background: #0284c7; color: white; padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.8rem; margin-right: 0.2rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Go Reverse Tunnel Dashboard</h1>
        
        <div class="grid">
            <div class="card">
                <h3>System Uptime</h3>
                <div class="value">%s</div>
            </div>
            <div class="card">
                <h3>Active Clients</h3>
                <div class="value">%d</div>
            </div>
            <div class="card">
                <h3>Control Address</h3>
                <div class="value" style="font-size: 1.1rem;">%s</div>
            </div>
        </div>

        <h2>Connected Clients</h2>
        <table>
            <thead>
                <tr>
                    <th>Client ID</th>
                    <th>Remote Address</th>
                    <th>Bound TCP Ports</th>
                    <th>Bound UDP Ports</th>
                </tr>
            </thead>
            <tbody>`, uptime.String(), len(s.clients), s.config.ControlAddr)

	if len(s.clients) == 0 {
		html += `<tr><td colspan="4" style="text-align:center; color:#94a3b8;">No clients connected</td></tr>`
	} else {
		for _, client := range s.clients {
			portsHTML := ""
			for _, p := range client.Ports {
				portsHTML += fmt.Sprintf(`<span class="badge">%d</span>`, p)
			}
			udpPortsHTML := ""
			for _, p := range client.UDPPorts {
				udpPortsHTML += fmt.Sprintf(`<span class="badge" style="background:#d97706;">%d</span>`, p)
			}

			html += fmt.Sprintf(`
                <tr>
                    <td><strong>%s</strong></td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%s</td>
                </tr>`, client.ClientID, client.RemoteAddr, portsHTML, udpPortsHTML)
		}
	}

	html += fmt.Sprintf(`
            </tbody>
        </table>
        <p style="margin-top: 1.5rem; font-size: 0.875rem; color: #64748b;">Metrics endpoint available at <a href="/metrics" style="color: #38bdf8;">/metrics</a></p>
    </div>
</body>
</html>`)

	_, _ = w.Write([]byte(html))
}
