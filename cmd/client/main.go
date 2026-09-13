package main

import (
"crypto/tls"
"encoding/json"
"flag"
"log"
"net"
"os"
"strings"
"time"
"tunnel-tool/pkg/tunnel"
"github.com/hashicorp/yamux"
)

type Config struct {
ServerAddr   string `json:"server"`
Token        string `json:"token"`
ClientID     string `json:"client_id"`
LocalService string `json:"local"`
}

func main() {
configFile := flag.String("config", "", "Path to JSON config file")
flag.Parse()

cfg := Config{
ServerAddr:   "127.0.0.1:7001",
Token:        "secret-token-123",
ClientID:     "client-01",
LocalService: "127.0.0.1:8080",
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

tlsCfg := &tls.Config{
InsecureSkipVerify: true,
}

backoffSec := 2
maxBackoffSec := 30

for {
log.Printf("[Client %s] Connecting securely to server %s...", cfg.ClientID, cfg.ServerAddr)
conn, err := tls.Dial("tcp", cfg.ServerAddr, tlsCfg)
if err != nil {
log.Printf("[Client %s] Connection failed: %v. Retrying in %d seconds...", cfg.ClientID, err, backoffSec)
time.Sleep(time.Duration(backoffSec) * time.Second)
backoffSec *= 2
if backoffSec > maxBackoffSec {
backoffSec = maxBackoffSec
}
continue
}

backoffSec = 2

authPayload := cfg.Token + ":" + cfg.ClientID + "\n"
_, err = conn.Write([]byte(authPayload))
if err != nil {
log.Printf("[Client %s] Failed to send authentication payload: %v", cfg.ClientID, err)
conn.Close()
time.Sleep(3 * time.Second)
continue
}

buf := make([]byte, 64)
conn.SetReadDeadline(time.Now().Add(10 * time.Second))
n, err := conn.Read(buf)
if err != nil || strings.TrimSpace(string(buf[:n])) != "AUTH_OK" {
log.Printf("[Client %s] Authentication rejected.", cfg.ClientID)
conn.Close()
time.Sleep(3 * time.Second)
continue
}

log.Printf("[Client %s] Authenticated! Initializing Yamux with Heartbeat...", cfg.ClientID)
conn.SetReadDeadline(time.Time{}) // Zero value literal

yamuxConfig := yamux.DefaultConfig()
yamuxConfig.EnableKeepAlive = true
yamuxConfig.KeepAliveInterval = 10 * time.Second
yamuxConfig.ConnectionWriteTimeout = 30 * time.Second

session, err := yamux.Client(conn, yamuxConfig)
if err != nil {
log.Printf("[Client %s] Yamux session error: %v", cfg.ClientID, err)
conn.Close()
time.Sleep(2 * time.Second)
continue
}

for {
stream, err := session.AcceptStream()
if err != nil {
log.Printf("[Client %s] Yamux stream accept error: %v. Reconnecting...", cfg.ClientID, err)
break
}

go func(remoteStream net.Conn) {
localConn, err := net.Dial("tcp", cfg.LocalService)
if err != nil {
remoteStream.Close()
return
}
tunnel.Pipe(localConn, remoteStream)
}(stream)
}

session.Close()
time.Sleep(2 * time.Second)
}
}
