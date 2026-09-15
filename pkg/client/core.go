package client

import (
"bufio"
"encoding/hex"
"encoding/json"
"fmt"
"log"
"net"
"strconv"
"strings"
"time"

"github.com/sepantartd/go-reverse-tunnel/pkg/config"
"github.com/sepantartd/go-reverse-tunnel/pkg/protocol"
"github.com/sepantartd/go-reverse-tunnel/pkg/tunnel"
)

type TunnelClient struct {
config *config.ClientConfig
}

func NewTunnelClient(cfg *config.ClientConfig) *TunnelClient {
return &TunnelClient{config: cfg}
}

func (c *TunnelClient) Start() {
for {
log.Printf("[Client] Connecting to server %s...", c.config.Server)
err := c.connectAndServe()
if err != nil {
log.Printf("[Client] Connection lost: %v. Reconnecting in 5 seconds...", err)
}
time.Sleep(5 * time.Second)
}
}

func (c *TunnelClient) connectAndServe() error {
conn, err := net.Dial("tcp", c.config.Server)
if err != nil {
return err
}
defer conn.Close()

// Step 1: Read Server Challenge
var challenge protocol.Challenge
if err := json.NewDecoder(conn).Decode(&challenge); err != nil {
return fmt.Errorf("failed to read challenge: %w", err)
}

nonce, err := hex.DecodeString(challenge.NonceHex)
if err != nil {
return fmt.Errorf("invalid nonce: %w", err)
}

// Step 2: Respond with HMAC
hmacResp := protocol.ComputeHMAC(nonce, c.config.Token)
authReq := protocol.AuthHandshake{
ClientID: c.config.ClientID,
Response: hmacResp,
}

if err := json.NewEncoder(conn).Encode(authReq); err != nil {
return fmt.Errorf("failed to send auth: %w", err)
}

// Step 3: Establish Yamux Session
sess, err := tunnel.NewClientSession(conn)
if err != nil {
return fmt.Errorf("mux session failed: %w", err)
}
defer sess.Close()

log.Printf("[Client] Tunnel session established successfully")

for {
stream, err := sess.Accept()
if err != nil {
return err
}
go c.handleStream(stream)
}
}

func (c *TunnelClient) handleStream(stream net.Conn) {
reader := bufio.NewReader(stream)
header, err := reader.ReadString('\n')
if err != nil {
stream.Close()
return
}

remotePort, err := strconv.Atoi(strings.TrimSpace(header))
if err != nil {
stream.Close()
return
}

var localTarget string
for _, f := range c.config.Forwards {
if f.RemotePort == remotePort {
localTarget = f.Local
break
}
}

if localTarget == "" {
log.Printf("[Client] No forward mapping for remote port %d", remotePort)
stream.Close()
return
}

localConn, err := net.Dial("tcp", localTarget)
if err != nil {
log.Printf("[Client] Failed to dial local target %s: %v", localTarget, err)
stream.Close()
return
}

tunnel.Pipe(stream, localConn)
}
