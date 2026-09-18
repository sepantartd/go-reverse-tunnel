package server

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// DashboardHandler manages the rendering and API routes for the server control panel and health metrics.
type DashboardHandler struct {
	server *Server
}

// NewDashboardHandler initializes a new instance of DashboardHandler with reference to the core Server.
func NewDashboardHandler(s *Server) *DashboardHandler {
	return &DashboardHandler{server: s}
}

// RegisterRoutes registers all dashboard, API, and monitoring endpoints onto the provided ServeMux.
func (dh *DashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", dh.server.authMiddleware(dh.handleDashboardPage))
	mux.HandleFunc("/api/status", dh.server.authMiddleware(dh.handleAPIStatus))
	mux.HandleFunc("/api/clients", dh.server.authMiddleware(dh.handleAPIClients))
	mux.HandleFunc("/api/metrics", dh.server.authMiddleware(dh.handleMetricsJSON))
}

// handleDashboardPage renders the administrative web interface panel.
func (dh *DashboardHandler) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Reverse Tunnel - Dashboard</title>
    <style>
        :root {
            background-color: #0f172a;
            color: #f8fafc;
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
        }
        body {
            margin: 0;
            padding: 24px;
            max-width: 1200px;
            margin-left: auto;
            margin-right: auto;
        }
        h1 {
            color: #38bdf8;
            font-size: 1.5rem;
            border-bottom: 2px solid #1e293b;
            padding-bottom: 12px;
            margin-bottom: 24px;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 16px;
            margin-bottom: 24px;
        }
        .card {
            background-color: #1e293b;
            border: 1px solid #334155;
            border-radius: 8px;
            padding: 20px;
        }
        .card h3 {
            margin: 0 0 10px 0;
            font-size: 0.875rem;
            color: #94a3b8;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        .card .value {
            font-size: 1.75rem;
            font-weight: bold;
            color: #34d399;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            background-color: #1e293b;
            border-radius: 8px;
            overflow: hidden;
            border: 1px solid #334155;
        }
        th, td {
            padding: 12px 16px;
            text-align: left;
            border-bottom: 1px solid #334155;
            font-size: 0.9rem;
        }
        th {
            background-color: #0f172a;
            color: #38bdf8;
            font-weight: 600;
        }
        tr:last-child td {
            border-bottom: none;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            background-color: #065f46;
            color: #6ee7b7;
            font-size: 0.75rem;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <h1>🚀 Go Reverse Tunnel Control Center</h1>
    
    <div class="grid">
        <div class="card">
            <h3>Server Status</h3>
            <div class="value"><span class="status-badge">ONLINE</span></div>
        </div>
        <div class="card">
            <h3>Active Clients</h3>
            <div class="value" id="stat-clients">-</div>
        </div>
        <div class="card">
            <h3>Active Listeners</h3>
            <div class="value" id="stat-listeners">-</div>
        </div>
        <div class="card">
            <h3>Total Connections</h3>
            <div class="value" id="stat-conns">-</div>
        </div>
    </div>

    <div class="card" style="padding: 0; overflow: hidden;">
        <div style="padding: 16px 20px; border-bottom: 1px solid #334155; font-weight: bold; background-color: #1e293b;">Connected Clients Registry</div>
        <table>
            <thead>
                <tr>
                    <th>Client ID</th>
                    <th>Connected At</th>
                    <th>Active Ports</th>
                    <th>Active Streams</th>
                    <th>Bytes Transferred (TX / RX)</th>
                </tr>
            </thead>
            <tbody id="clients-table-body">
                <tr><td colspan="5" style="text-align: center; color: #94a3b8;">Loading client sessions...</td></tr>
            </tbody>
        </table>
    </div>

    <script>
        const token = new URLSearchParams(window.location.search).get('token') || '';

        async function fetchData() {
            try {
                const statusRes = await fetch('/api/status?token=' + token);
                if (statusRes.ok) {
                    const statusData = await statusRes.json();
                    document.getElementById('stat-clients').innerText = statusData.active_clients;
                    document.getElementById('stat-listeners').innerText = statusData.active_listeners;
                    document.getElementById('stat-conns').innerText = statusData.total_connections;
                }

                const clientsRes = await fetch('/api/clients?token=' + token);
                if (clientsRes.ok) {
                    const clients = await clientsRes.json();
                    const tbody = document.getElementById('clients-table-body');
                    if (clients.length === 0) {
                        tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; color: #94a3b8;">No active client sessions connected.</td></tr>';
                        return;
                    }

                    tbody.innerHTML = clients.map(c => ` + "`" + `
                        <tr>
                            <td><strong>${c.id}</strong></td>
                            <td>${new Date(c.connected_at).toLocaleString()}</td>
                            <td>${c.active_ports.join(', ') || 'None'}</td>
                            <td>${c.active_streams}</td>
                            <td>${formatBytes(c.bytes_tx)} /${formatBytes(c.bytes_rx)}</td>
                        </tr>
                    ` + "`" + `).join('');
                }
            } catch (err) {
                console.error('Failed to update dashboard telemetry:', err);
            }
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024, sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        fetchData();
        setInterval(fetchData, 3000);
    </script>
</body>
</html>`
	_, _ = w.Write([]byte(html))
}

// handleAPIStatus provides a JSON endpoint detailing overall server runtime metrics.
func (dh *DashboardHandler) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	dh.server.mu.RLock()
	activeClients := len(dh.server.clients)
	activeListeners := len(dh.server.listeners)
	dh.server.mu.RUnlock()

	resp := map[string]interface{}{
		"status":            "running",
		"active_clients":    activeClients,
		"active_listeners":  activeListeners,
		"total_connections": atomic.LoadUint64(&dh.server.totalConnections),
		"auth_failures":     atomic.LoadUint64(&dh.server.authFailures),
		"uptime_seconds":    time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleAPIClients provides a JSON list of currently connected tunnel clients and their status.
func (dh *DashboardHandler) handleAPIClients(w http.ResponseWriter, r *http.Request) {
	dh.server.mu.RLock()
	defer dh.server.mu.RUnlock()

	clientList := make([]map[string]interface{}, 0)
	for _, c := range dh.server.clients {
		clientList = append(clientList, map[string]interface{}{
			"id":             c.ID,
			"connected_at":   c.ConnectedAt.Format(time.RFC3339),
			"active_ports":   c.ActivePorts,
			"active_streams": atomic.LoadInt32(&c.ActiveStreams),
			"bytes_tx":       atomic.LoadUint64(&c.BytesTx),
			"bytes_rx":       atomic.LoadUint64(&c.BytesRx),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clientList)
}

// handleMetricsJSON outputs consolidated metrics in JSON format for external scraping tools.
func (dh *DashboardHandler) handleMetricsJSON(w http.ResponseWriter, r *http.Request) {
	dh.server.mu.RLock()
	activeClients := len(dh.server.clients)
	activeListeners := len(dh.server.listeners)
	dh.server.mu.RUnlock()

	var totalTx, totalRx uint64
	dh.server.mu.RLock()
	for _, c := range dh.server.clients {
		totalTx += atomic.LoadUint64(&c.BytesTx)
		totalRx += atomic.LoadUint64(&c.BytesRx)
	}
	dh.server.mu.RUnlock()

	metrics := map[string]interface{}{
		"active_clients":     activeClients,
		"active_listeners":   activeListeners,
		"total_connections":  atomic.LoadUint64(&dh.server.totalConnections),
		"auth_failures":      atomic.LoadUint64(&dh.server.authFailures),
		"total_bytes_tx":     totalTx,
		"total_bytes_rx":     totalRx,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}
