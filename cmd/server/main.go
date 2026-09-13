package main

import (
"crypto/tls"
"encoding/json"
"flag"
"fmt"
"html/template"
"log"
"net"
"net/http"
"os"
"strings"
"sync"
"time"
"tunnel-tool/pkg/tunnel"
"github.com/hashicorp/yamux"
)

type Config struct {
Bind        string `json:"bind"`
ControlPort int    `json:"control_port"`
WebPort     int    `json:"web_port"`
Token       string `json:"token"`
CertFile    string `json:"cert"`
KeyFile     string `json:"key"`
}

type ClientInfo struct {
ID          string    `json:"id"`
RemoteAddr  string    `json:"remote_addr"`
ConnectedAt time.Time `json:"connected_at"`
}

var (
clientsMutex   sync.Mutex
clientSessions = make(map[string]*yamux.Session)
clientDetails  = make(map[string]ClientInfo)
)

func main() {
configFile := flag.String("config", "", "Path to JSON config file")
flag.Parse()

cfg := Config{
Bind:        "0.0.0.0:7000",
ControlPort: 7001,
WebPort:     8081,
Token:       "secret-token-123",
CertFile:    "cert.pem",
KeyFile:     "key.pem",
}

if *configFile != "" {
file, err := os.Open(*configFile)
if err != nil {
log.Fatalf("Failed to open config file: %v", err)
}
defer file.Close()
if err := json.NewDecoder(file).Decode(&cfg); err != nil {
log.Fatalf("Failed to parse config file: %v", err)
}
}

cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
if err != nil {
log.Fatalf("Failed to load TLS keys: %v", err)
}

tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
controlAddr := fmt.Sprintf("0.0.0.0:%d", cfg.ControlPort)

controlListener, err := tls.Listen("tcp", controlAddr, tlsCfg)
if err != nil {
log.Fatalf("Failed to start TLS Multiplex server: %v", err)
}
log.Printf("[Server] Secure TLS Server listening on %s", controlAddr)

publicListener, err := net.Listen("tcp", cfg.Bind)
if err != nil {
log.Fatalf("Failed to start public listener: %v", err)
}
log.Printf("[Server] Public proxy listening on %s", cfg.Bind)

// Start Web Dashboard Server
go startWebDashboard(cfg.WebPort)

// Route public incoming connections
go func() {
for {
publicConn, err := publicListener.Accept()
if err != nil {
continue
}

clientsMutex.Lock()
var targetSession *yamux.Session
for _, sess := range clientSessions {
targetSession = sess
break
}
clientsMutex.Unlock()

if targetSession == nil {
log.Printf("[Server] No active client connected to handle request from %s", publicConn.RemoteAddr())
publicConn.Close()
continue
}

stream, err := targetSession.OpenStream()
if err != nil {
log.Printf("[Server] Failed to open stream: %v", err)
publicConn.Close()
continue
}

log.Printf("[Server] Routing connection from %s to client...", publicConn.RemoteAddr())
go tunnel.Pipe(publicConn, stream)
}
}()

for {
clientConn, err := controlListener.Accept()
if err != nil {
continue
}
go handleClient(clientConn, cfg.Token)
}
}

func handleClient(controlConn net.Conn, expectedToken string) {
defer controlConn.Close()

controlConn.SetReadDeadline(time.Now().Add(10 * time.Second))
buf := make([]byte, 256)
n, err := controlConn.Read(buf)
if err != nil {
return
}

parts := strings.SplitN(strings.TrimSpace(string(buf[:n])), ":", 2)
if len(parts) != 2 || parts[0] != expectedToken {
log.Printf("[Server] Auth failed from %s", controlConn.RemoteAddr())
controlConn.Write([]byte("AUTH_FAILED"))
return
}

clientID := parts[1]
controlConn.Write([]byte("AUTH_OK"))
log.Printf("[Server] Client [%s] authenticated successfully", clientID)
controlConn.SetReadDeadline(time.Time{})

// Configure Yamux with Heartbeat on Server side
yamuxConfig := yamux.DefaultConfig()
yamuxConfig.EnableKeepAlive = true
yamuxConfig.KeepAliveInterval = 10 * time.Second
yamuxConfig.ConnectionWriteTimeout = 30 * time.Second

session, err := yamux.Server(controlConn, yamuxConfig)
if err != nil {
return
}
defer session.Close()

clientsMutex.Lock()
clientSessions[clientID] = session
clientDetails[clientID] = ClientInfo{
ID:          clientID,
RemoteAddr:  controlConn.RemoteAddr().String(),
ConnectedAt: time.Now(),
}
clientsMutex.Unlock()

defer func() {
clientsMutex.Lock()
delete(clientSessions, clientID)
delete(clientDetails, clientID)
clientsMutex.Unlock()
log.Printf("[Server] Client [%s] disconnected.", clientID)
}()

for {
if session.IsClosed() {
break
}
time.Sleep(2 * time.Second)
}
}

func startWebDashboard(port int) {
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
clientsMutex.Lock()
defer clientsMutex.Unlock()

tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Vexora Tunnel Dashboard</title>
<meta http-equiv="refresh" content="3">
<style>
body { font-family: Arial, sans-serif; background: #0f172a; color: #f8fafc; margin: 0; padding: 20px; }
.container { max-width: 800px; margin: auto; background: #1e293b; padding: 20px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); }
h2 { border-bottom: 2px solid #334155; padding-bottom: 10px; color: #38bdf8; }
table { width: 100%; border-collapse: collapse; margin-top: 20px; }
th, td { padding: 12px; text-align: left; border-bottom: 1px solid #334155; }
th { background: #334155; color: #38bdf8; }
.badge { background: #22c55e; color: white; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
.no-client { color: #94a3b8; font-style: italic; }
</style>
</head>
<body>
<div class="container">
<h2>🚀 Vexora Tunnel Live Dashboard</h2>
<p>Status: <strong>Active</strong> | Auto-refreshing every 3s</p>
<table>
<thead>
<tr>
<th>Client ID</th>
<th>Remote Address</th>
<th>Connected Since</th>
<th>Status</th>
</tr>
</thead>
<tbody>
{{range .}}
<tr>
<td><strong>{{.ID}}</strong></td>
<td>{{.RemoteAddr}}</td>
<td>{{.ConnectedAt.Format "2006-01-02 15:04:05"}}</td>
<td><span class="badge">Online</span></td>
</tr>
{{else}}
<tr>
<td colspan="4" class="no-client">No active clients connected.</td>
</tr>
{{end}}
</tbody>
</table>
</div>
</body>
</html>
`
t, err := template.New("dashboard").Parse(tmpl)
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
t.Execute(w, clientDetails)
})

addr := fmt.Sprintf("0.0.0.0:%d", port)
log.Printf("[Dashboard] Web panel running at http://%s", addr)
http.ListenAndServe(addr, nil)
}
