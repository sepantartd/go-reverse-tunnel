# ⚡ go-reverse-tunnel

[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows%20%7C%20Android-lightgrey)](#platform-support)
[![Architecture](https://img.shields.io/badge/architecture-reverse%20tunnel-blue)](#architecture)
[![Security](https://img.shields.io/badge/security-TLS%20%7C%20mTLS%20%7C%20HMAC-red)](#security-model)

**go-reverse-tunnel** is a high-performance, production-oriented reverse tunneling system written in Go.

It is designed for scenarios where a private or restricted network needs to expose selected TCP/UDP services through a public VPS without requiring inbound connectivity to the private side.

The project is optimized for:

- Multi-port TCP forwarding
- Multiple simultaneous clients
- Long-lived connections
- Unstable or high-latency networks
- TLS-protected transport
- TCP stream multiplexing
- Automatic client reconnection
- Bandwidth optimization
- Linux production servers
- Windows clients
- Android/Termux environments
- 24/7 operation through systemd

> **Project:** `go-reverse-tunnel`
>
> **Author:** `sepantartd`
>
> **License:** MIT

---

## 📌 Overview

A typical deployment consists of two components:

```text
Private / Restricted Network
        │
        │ Outbound connection
        ▼
┌──────────────────────┐
│   Tunnel Client      │
│   VPS / Android      │
│                      │
│  client_id           │
│  TLS                 │
│  HMAC authentication │
│  Yamux               │
└──────────┬───────────┘
           │
           │ One persistent
           │ encrypted session
           ▼
┌──────────────────────┐
│   Public Tunnel      │
│   Server VPS         │
│                      │
│  TLS / mTLS          │
│  Yamux Multiplexer   │
│  Multi-client router │
│  Public listeners    │
└───────┬─────┬────────┘
        │     │
        │     └──────────────► :443
        │
        └────────────────────► :80
                              :2222
```

The fundamental design principle is simple:

> **The client establishes the connection outbound. The public server accepts external traffic and sends it back through the existing tunnel.**

This makes the architecture suitable for environments where the client side cannot reliably accept inbound connections.

---

# 🚀 Key Features

| Feature | Description |
|---|---|
| 🔀 TCP Multiplexing | Multiple logical TCP streams share a single persistent connection using Yamux |
| 🌐 Multi-Port | Forward multiple public ports such as `80`, `443`, and `2222` |
| 👥 Multi-Client | Multiple authenticated clients can connect simultaneously |
| 🔐 TLS | Encrypts the transport connection |
| 🛡️ mTLS | Optional mutual certificate authentication |
| 🔑 HMAC-SHA256 | Challenge-response authentication using cryptographic nonces |
| ♻️ Replay Protection | Nonce-based authentication prevents reuse of previous challenges |
| 📉 Snappy | Adaptive compression for suitable traffic |
| 🧦 SOCKS5 | Built-in SOCKS5 proxy support on the client side |
| 📡 UDP-over-TCP | Allows selected UDP traffic to traverse a TCP tunnel |
| 💓 Heartbeat | Keeps long-lived connections active |
| 🔄 Reconnection | Exponential backoff for unstable networks |
| 🆔 Client IDs | Unique routing and identification of clients |
| 📊 Web Dashboard | HTTP dashboard for connection and uptime monitoring |
| ⚙️ JSON Config | Simple machine-readable configuration |
| 🐧 systemd | 24/7 service operation and automatic restart |
| 📱 Termux | Designed to work on Android/Termux environments |
| 🪟 Windows | Client binaries can be built for Windows |
| 🧩 Single Transport | Multiple services can share one tunnel connection |

---

# 🏗️ Architecture

## Network Architecture

```text
                         PUBLIC INTERNET
                               │
                               │
                    ┌──────────▼──────────┐
                    │   Public VPS        │
                    │   Tunnel Server     │
                    │                     │
                    │  :80   ─────────┐   │
                    │  :443  ───────┐ │   │
                    │  :2222 ─────┐ │ │   │
                    │              │ │ │   │
                    │        ┌─────▼─▼─▼─┐ │
                    │        │  Router   │ │
                    │        │ Multi-Port│ │
                    │        └──────┬────┘ │
                    │               │      │
                    │          ┌────▼────┐ │
                    │          │  Yamux  │ │
                    │          │ Session │ │
                    │          └────┬────┘ │
                    │               │      │
                    │          TLS / mTLS  │
                    └───────────────┼──────┘
                                    │
                         ONE PERSISTENT CONNECTION
                                    │
                                    │
                    ┌───────────────▼──────────────┐
                    │       Private / Iran VPS     │
                    │          Tunnel Client       │
                    │                              │
                    │  ┌───────────────┐           │
                    │  │ TLS Transport │           │
                    │  └───────┬───────┘           │
                    │          │                   │
                    │      ┌───▼────┐              │
                    │      │ Yamux  │              │
                    │      │ Client │              │
                    │      └───┬────┘              │
                    │          │                   │
                    │     ┌────┴─────────────┐     │
                    │     │                  │     │
                    │  127.0.0.1:80    127.0.0.1:443
                    │     │                  │     │
                    │  HTTP Service      HTTPS Service
                    │                              │
                    │  127.0.0.1:22                │
                    │       SSH                    │
                    └──────────────────────────────┘
```

---

# 🔀 How Multiplexing Works

Instead of opening an independent TCP connection for every forwarded port, the client maintains one primary transport connection.

Conceptually:

```text
                    TLS Connection
                         │
                    ┌────▼────┐
                    │  Yamux  │
                    └────┬────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
     Stream 1         Stream 2         Stream 3
        │                │                │
      TCP:80          TCP:443          TCP:2222
        │                │                │
    HTTP Server      HTTPS Server       SSH
```

This approach reduces connection-management overhead and allows multiple logical streams to coexist on the same transport.

Yamux provides stream-oriented multiplexing over a reliable ordered connection.

---

# 🔐 Security Model

Security is implemented as multiple independent layers.

```text
Application Data
       │
       ▼
Optional Compression
       │
       ▼
Yamux Stream
       │
       ▼
HMAC Authentication
       │
       ▼
TLS / mTLS
       │
       ▼
TCP Transport
```

## TLS

TLS protects the tunnel against passive traffic inspection and active manipulation.

Production deployments should use:

- A valid server certificate
- A private key protected with restrictive filesystem permissions
- Strong TLS configuration
- A trusted CA where appropriate
- Certificate rotation procedures

---

## mTLS

With mutual TLS enabled:

```text
Client ───── Client Certificate ─────► Server
Client ◄──── Server Certificate ────── Server
```

Both endpoints authenticate the other side.

This is stronger than server-only TLS because possession of a trusted client certificate can be required before a tunnel session is accepted.

---

## HMAC-SHA256 Challenge-Response

A challenge-response authentication flow should use a fresh cryptographic nonce.

Conceptually:

```text
Client                         Server
  │                              │
  │──── ClientHello ────────────►│
  │                              │
  │◄──── Random Challenge ───────│
  │                              │
  │  HMAC-SHA256(secret, nonce)  │
  │                              │
  │──── HMAC Response ──────────►│
  │                              │
  │◄──── Authentication OK ──────│
  │                              │
  │══════ Yamux Session ═════════│
```

The challenge must be unpredictable and unique.

A static token should never be treated as equivalent to a properly implemented nonce-based challenge-response protocol.

---

# 🧱 Repository Structure

The project follows a Go-oriented structure:

```text
go-reverse-tunnel/
│
├── cmd/
│   ├── server/
│   │   └── main.go
│   │
│   └── client/
│       └── main.go
│
├── pkg/
│   └── tunnel/
│       └── ...
│
├── server-config.json
├── client-config.json
│
├── config-server.json
├── config-client.json
│
├── build.sh
│
├── cert.pem
├── key.pem
│
├── go.mod
├── go.sum
│
├── LICENSE
└── README.md
```

### Directory responsibilities

| Path | Purpose |
|---|---|
| `cmd/server` | Server entry point |
| `cmd/client` | Client entry point |
| `pkg/tunnel` | Core tunnel implementation |
| `server-config.json` | Server runtime configuration |
| `client-config.json` | Client runtime configuration |
| `build.sh` | Build automation |
| `cert.pem` | TLS certificate |
| `key.pem` | TLS private key |
| `go.mod` | Go module definition |
| `go.sum` | Dependency checksums |
| `LICENSE` | Project license |

---

# 📦 Requirements

## Server

Recommended:

- Linux VPS
- Public IPv4/IPv6 address
- Go 1.26+
- Root or appropriate `CAP_NET_BIND_SERVICE` privileges for ports below `1024`
- Open firewall ports
- TLS certificate and private key for production

## Client

Supported environments may include:

- Linux
- Android + Termux
- Windows

The client requires outbound connectivity to the server control endpoint.

---

# 🛠️ Installation

## Clone the Repository

```bash
git clone https://github.com/sepantartd/go-reverse-tunnel.git
cd go-reverse-tunnel
```

## Verify Go

```bash
go version
```

Recommended:

```text
go version go1.26.x linux/amd64
```

## Download Dependencies

```bash
go mod download
```

## Verify the Module

```bash
go mod tidy
```

Then:

```bash
go test ./...
```

---

# 🔨 Build

## Build Server

```bash
go build -o tunnel-server ./cmd/server
```

## Build Client

```bash
go build -o tunnel-client ./cmd/client
```

## Build for Linux AMD64

```bash
GOOS=linux GOARCH=amd64 \
go build -o tunnel-server ./cmd/server
```

## Build Client for Android/Termux ARM64

```bash
GOOS=linux GOARCH=arm64 \
go build -o tunnel-client ./cmd/client
```

## Build Client for Windows

```bash
GOOS=windows GOARCH=amd64 \
go build -o tunnel-client.exe ./cmd/client
```

---

# ⚙️ Configuration

Configuration is stored in JSON.

There are two primary configurations:

```text
server-config.json
client-config.json
```

The server defines public listeners and tunnel control parameters.

The client defines the remote server and local forwarding destinations.

---

# 🖥️ Server Configuration

## `server-config.json`

```json
{
  "control_addr": "0.0.0.0:7001",
  "web_port": 8081,
  "token": "CHANGE-THIS-TO-A-LONG-RANDOM-SECRET",
  "cert": "cert.pem",
  "key": "key.pem",
  "public_binds": [
    {
      "port": 80,
      "name": "http"
    },
    {
      "port": 443,
      "name": "https"
    },
    {
      "port": 2222,
      "name": "ssh"
    }
  ]
}
```

## Server Parameters

| Parameter | Type | Description |
|---|---|---|
| `control_addr` | string | Address where tunnel clients connect |
| `web_port` | integer | HTTP monitoring/dashboard port |
| `token` | string | Shared authentication secret |
| `cert` | string | Path to TLS certificate |
| `key` | string | Path to TLS private key |
| `public_binds` | array | Public ports exposed by the server |
| `public_binds[].port` | integer | Public TCP port |
| `public_binds[].name` | string | Human-readable listener name |

### Important

Never commit a real production secret into Git.

Use:

```text
CHANGE-THIS-TO-A-LONG-RANDOM-SECRET
```

only as a placeholder.

---

# 📱 Client Configuration

## `client-config.json`

```json
{
  "server": "YOUR_SERVER_IP:7001",
  "token": "CHANGE-THIS-TO-A-LONG-RANDOM-SECRET",
  "client_id": "client-01",
  "forwards": [
    {
      "remote_port": 80,
      "local": "127.0.0.1:80"
    },
    {
      "remote_port": 443,
      "local": "127.0.0.1:443"
    },
    {
      "remote_port": 2222,
      "local": "127.0.0.1:22"
    }
  ]
}
```

## Client Parameters

| Parameter | Type | Description |
|---|---|---|
| `server` | string | Server control address |
| `token` | string | Authentication secret matching the server |
| `client_id` | string | Unique client identifier |
| `forwards` | array | List of tunnel forwarding rules |
| `forwards[].remote_port` | integer | Public server port |
| `forwards[].local` | string | Local client-side destination |

---

# 🔁 Port Forwarding Example

Consider:

```text
Public VPS
  :80
  :443
  :2222

       │
       │ Tunnel
       ▼

Private VPS
  :80
  :443
  :22
```

Configuration:

```json
{
  "forwards": [
    {
      "remote_port": 80,
      "local": "127.0.0.1:80"
    },
    {
      "remote_port": 443,
      "local": "127.0.0.1:443"
    },
    {
      "remote_port": 2222,
      "local": "127.0.0.1:22"
    }
  ]
}
```

Traffic becomes:

```text
Internet
   │
   ├── :80 ───────► Client :80
   │
   ├── :443 ──────► Client :443
   │
   └── :2222 ─────► Client :22
```

---

# ▶️ Running the Server

Start the server with:

```bash
./tunnel-server -config=server-config.json
```

For development:

```bash
go run ./cmd/server -config=server-config.json
```

---

# ▶️ Running the Client

Start the client with:

```bash
./tunnel-client -config=client-config.json
```

For development:

```bash
go run ./cmd/client -config=client-config.json
```

---

# 📊 Web Dashboard

The server exposes a monitoring HTTP endpoint.

With:

```json
{
  "web_port": 8081
}
```

the dashboard is available on:

```text
http://SERVER_IP:8081
```

The dashboard can be used to inspect information such as:

- Connected clients
- Client IDs
- Connection state
- Uptime
- Active sessions
- Tunnel health

Do not expose the monitoring port publicly without appropriate network controls.

A safer approach is to allow access only from a trusted administration network.

---

# 🧦 SOCKS5 Proxy

When enabled, the client can provide a local SOCKS5 endpoint.

Conceptually:

```text
Application
    │
    ▼
SOCKS5
    │
    ▼
Tunnel Client
    │
    ▼
TLS + Yamux
    │
    ▼
Tunnel Server
    │
    ▼
Internet / Destination
```

This allows compatible applications to route traffic through the tunnel without configuring a separate forwarding rule for every destination.

The SOCKS5 listener should preferably bind to:

```text
127.0.0.1
```

rather than:

```text
0.0.0.0
```

unless remote access is explicitly required.

---

# 📡 UDP-over-TCP

The architecture can transport selected UDP traffic through the reliable TCP tunnel.

Example use cases include:

- DNS
- Application-specific UDP traffic
- WireGuard scenarios where UDP transport is otherwise unavailable

Conceptually:

```text
UDP Application
      │
      ▼
UDP Adapter
      │
      ▼
UDP-over-TCP
      │
      ▼
TLS
      │
      ▼
Yamux
      │
      ▼
TCP
      │
      ▼
Remote Endpoint
```

## Important Performance Consideration

UDP-over-TCP is a compatibility mechanism, not a replacement for native UDP.

Because TCP provides ordered delivery and retransmission, packet loss can introduce head-of-line blocking.

For latency-sensitive workloads, native UDP is preferable whenever the network permits it.

---

# 💓 Keep-Alive and Connection Stability

Long-lived connections can be silently removed by:

- NAT devices
- Stateful firewalls
- Mobile operators
- Idle connection timers
- Intermediate routers

The tunnel therefore benefits from heartbeat traffic.

Conceptually:

```text
Client                         Server
  │                              │
  │──────── PING ───────────────►│
  │◄─────── PONG ────────────────│
  │                              │
  │──────── PING ───────────────►│
  │◄─────── PONG ────────────────│
```

Keep-alive intervals should be selected according to the network environment.

Too frequent:

```text
More overhead
```

Too infrequent:

```text
Higher probability of idle timeout
```

---

# ♻️ Automatic Reconnection

The client should not reconnect in a tight loop after a network failure.

Recommended strategy:

```text
Connection lost
      │
      ▼
Wait 1s
      │
      ▼
Retry
      │
      ▼
Wait 2s
      │
      ▼
Retry
      │
      ▼
Wait 4s
      │
      ▼
Retry
      │
      ▼
...
      │
      ▼
Maximum Backoff
```

Exponential backoff reduces unnecessary load on the server during prolonged outages.

A successful connection resets the backoff state.

---

# 👥 Multi-Client Architecture

Each client should have a unique:

```text
client_id
```

Example:

```text
iran-vps-01
iran-vps-02
office-gateway
android-client-01
```

Conceptually:

```text
                 Tunnel Server
                      │
          ┌───────────┼───────────┐
          │           │           │
          ▼           ▼           ▼
      client-01   client-02   client-03
          │           │           │
       VPS #1       VPS #2      VPS #3
```

The server can use the client identity to route incoming public traffic to the correct tunnel session.

Avoid duplicate `client_id` values unless shared identity is explicitly intended.

---

# 🐧 systemd Deployment

For production Linux deployments, systemd is recommended.

Example installation directory:

```text
/opt/go-reverse-tunnel/
```

Create:

```bash
sudo mkdir -p /opt/go-reverse-tunnel
```

Copy binaries:

```bash
sudo cp tunnel-server /opt/go-reverse-tunnel/
sudo cp server-config.json /opt/go-reverse-tunnel/
sudo cp cert.pem /opt/go-reverse-tunnel/
sudo cp key.pem /opt/go-reverse-tunnel/
```

Protect the private key:

```bash
sudo chmod 600 /opt/go-reverse-tunnel/key.pem
```

---

# ⚙️ Server systemd Service

Create:

```bash
sudo nano /etc/systemd/system/go-reverse-tunnel-server.service
```

Use:

```ini
[Unit]
Description=go-reverse-tunnel Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/go-reverse-tunnel
ExecStart=/opt/go-reverse-tunnel/tunnel-server -config=/opt/go-reverse-tunnel/server-config.json
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
```

Reload systemd:

```bash
sudo systemctl daemon-reload
```

Enable at boot:

```bash
sudo systemctl enable go-reverse-tunnel-server
```

Start:

```bash
sudo systemctl start go-reverse-tunnel-server
```

Check status:

```bash
sudo systemctl status go-reverse-tunnel-server
```

View logs:

```bash
sudo journalctl -u go-reverse-tunnel-server -f
```

---

# 📱 Client systemd Service

On the client Linux VPS:

```bash
sudo mkdir -p /opt/go-reverse-tunnel
```

Copy:

```bash
sudo cp tunnel-client /opt/go-reverse-tunnel/
sudo cp client-config.json /opt/go-reverse-tunnel/
```

Create:

```bash
sudo nano /etc/systemd/system/go-reverse-tunnel-client.service
```

Use:

```ini
[Unit]
Description=go-reverse-tunnel Client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/go-reverse-tunnel
ExecStart=/opt/go-reverse-tunnel/tunnel-client -config=/opt/go-reverse-tunnel/client-config.json
Restart=always
RestartSec=5

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable go-reverse-tunnel-client
sudo systemctl start go-reverse-tunnel-client
```

Check:

```bash
sudo systemctl status go-reverse-tunnel-client
```

Logs:

```bash
sudo journalctl -u go-reverse-tunnel-client -f
```

---

# 🔥 Firewall Configuration

## UFW

If the server uses UFW:

```bash
sudo ufw allow 7001/tcp
sudo ufw allow 8081/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 2222/tcp
```

Then:

```bash
sudo ufw status
```

Only open ports that are actually required.

---

# iptables

Check current rules:

```bash
sudo iptables -L -n -v
```

Allow the control port:

```bash
sudo iptables -A INPUT -p tcp --dport 7001 -j ACCEPT
```

Allow HTTP:

```bash
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
```

Allow HTTPS:

```bash
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

Allow SSH tunnel port:

```bash
sudo iptables -A INPUT -p tcp --dport 2222 -j ACCEPT
```

Remember that firewall rules must be persisted using the firewall management mechanism used by the operating system.

---

# 🧪 Testing the Tunnel

## Check Listening Ports

Server:

```bash
sudo ss -lntp
```

Client:

```bash
sudo ss -lntp
```

## Test Server Control Port

From another machine:

```bash
nc -vz SERVER_IP 7001
```

## Test Public HTTP

```bash
curl -v http://SERVER_IP/
```

## Test SSH Forward

```bash
ssh -p 2222 USER@SERVER_IP
```

---

# 🛠️ Troubleshooting

## 1. Port Already in Use

### Error

```text
bind: address already in use
```

Find the process:

```bash
sudo ss -lntp | grep ':7001'
```

Or:

```bash
sudo lsof -i :7001
```

For port 80:

```bash
sudo lsof -i :80
```

For port 443:

```bash
sudo lsof -i :443
```

For port 2222:

```bash
sudo lsof -i :2222
```

If another instance of the tunnel is running:

```bash
sudo systemctl status go-reverse-tunnel-server
```

Stop it if necessary:

```bash
sudo systemctl stop go-reverse-tunnel-server
```

### Common Cause

A frequent reason is running the binary manually while systemd is already running another copy.

Bad:

```text
systemd instance
+
manual instance
=
Port conflict
```

---

# 2. Client Cannot Connect

First verify basic connectivity:

```bash
ping SERVER_IP
```

Then test the control port:

```bash
nc -vz SERVER_IP 7001
```

If this fails, check:

```text
Client network
      │
      ├── Internet connectivity
      │
      ├── VPS routing
      │
      ├── Server firewall
      │
      ├── Cloud firewall/security group
      │
      └── Tunnel server listener
```

On the server:

```bash
sudo ss -lntp | grep 7001
```

Expected:

```text
0.0.0.0:7001
```

or the specific server address.

---

# 3. Connection Drops Frequently on Restricted Networks

Possible causes:

- Aggressive NAT timeout
- Mobile carrier timeout
- Packet loss
- High latency
- Firewall inspection
- TLS interruption
- Server-side timeout
- Insufficient keep-alive frequency

Check client logs:

```bash
journalctl -u go-reverse-tunnel-client -f
```

Check server logs:

```bash
journalctl -u go-reverse-tunnel-server -f
```

Test packet loss:

```bash
ping SERVER_IP
```

Test route:

```bash
traceroute SERVER_IP
```

or:

```bash
mtr SERVER_IP
```

Check TCP connectivity:

```bash
nc -vz SERVER_IP 7001
```

If the connection repeatedly fails and recovers, verify that the client has exponential backoff and heartbeat enabled.

---

# 4. HMAC Authentication Failure

Typical symptoms:

```text
authentication failed
invalid HMAC
challenge verification failed
unauthorized client
```

Check:

1. Client secret matches server secret.
2. The client ID is correct.
3. System clocks are reasonable.
4. The challenge is generated fresh.
5. The client is not reusing an old response.
6. No configuration file contains trailing whitespace or unexpected characters.

Compare configuration carefully:

```bash
cat server-config.json
```

and:

```bash
cat client-config.json
```

For production, never print secrets into shared logs or public issue reports.

---

# 5. Replay / Stale Challenge Problems

If authentication uses challenge-response, every authentication attempt should use a fresh nonce.

The server should reject:

```text
old nonce
+
old HMAC response
```

A correct flow is:

```text
Challenge A → Response A → ACCEPT

Challenge B → Response A → REJECT
```

The response to Challenge A must not be reusable for Challenge B.

---

# 6. TLS Certificate Errors

Typical errors:

```text
x509: certificate signed by unknown authority
```

or:

```text
tls: failed to verify certificate
```

Check the certificate:

```bash
openssl x509 -in cert.pem -text -noout
```

Check expiration:

```bash
openssl x509 -in cert.pem -noout -dates
```

Check the key:

```bash
openssl rsa -in key.pem -check
```

The certificate hostname must match the hostname used by the client when hostname verification is enabled.

---

# 7. Certificate and Private Key Do Not Match

Extract the certificate public key:

```bash
openssl x509 -in cert.pem -pubkey -noout | sha256sum
```

Extract the private key public key:

```bash
openssl pkey -in key.pem -pubout | sha256sum
```

The hashes should match.

If they differ, the certificate and private key belong to different key pairs.

---

# 8. TLS Handshake Debugging

Use OpenSSL:

```bash
openssl s_client -connect SERVER_IP:7001
```

For a hostname:

```bash
openssl s_client -connect example.com:7001 -servername example.com
```

Look for:

```text
Verify return code
```

and certificate-chain errors.

---

# 9. mTLS Client Certificate Failure

Typical symptoms:

```text
client certificate required
unknown ca
bad certificate
certificate verify failed
```

Check:

- Client certificate exists
- Client private key exists
- Client certificate is signed by the expected CA
- Server trusts the correct CA
- Client private key matches its certificate
- Certificate is not expired

Never distribute the same private key to every client.

Use unique credentials:

```text
client-01 → certificate-01
client-02 → certificate-02
client-03 → certificate-03
```

This makes revocation and auditing much easier.

---

# 10. UFW / iptables Blocking Traffic

Check UFW:

```bash
sudo ufw status verbose
```

Check iptables:

```bash
sudo iptables -L INPUT -n -v
```

Check listening ports:

```bash
sudo ss -lntp
```

A port can be correctly listening but still unreachable because of:

```text
Cloud firewall
      +
Linux firewall
      +
Network ACL
```

All relevant layers must permit the connection.

---

# 11. Dashboard Is Not Accessible

Check whether the dashboard is listening:

```bash
sudo ss -lntp | grep 8081
```

Check locally:

```bash
curl http://127.0.0.1:8081/
```

If local access works but remote access fails, investigate:

```text
UFW
iptables
Cloud firewall
Security Group
Provider ACL
```

For security-sensitive deployments, prefer restricting the dashboard to trusted IP addresses instead of exposing it globally.

---

# 12. Client ID Collision

If two clients use:

```text
client_id = client-01
```

the server may not be able to uniquely identify them.

Use unique IDs:

```text
client-iran-01
client-iran-02
client-office-01
```

---

# 13. Local Destination Is Not Listening

Suppose:

```json
{
  "remote_port": 2222,
  "local": "127.0.0.1:22"
}
```

Check SSH:

```bash
sudo ss -lntp | grep ':22'
```

Or:

```bash
sudo systemctl status ssh
```

If nothing is listening on `127.0.0.1:22`, the tunnel cannot forward traffic successfully.

---

# 14. Permission Denied on Ports Below 1024

Linux normally restricts unprivileged applications from binding ports such as:

```text
80
443
53
```

If you see:

```text
bind: permission denied
```

check whether the process has sufficient privileges.

Possible solutions include:

- Run the service with an appropriate privileged account
- Use `CAP_NET_BIND_SERVICE`
- Bind to an unprivileged port and use a reverse proxy
- Use a dedicated socket-forwarding architecture

Avoid granting unnecessary privileges.

---

# 15. High CPU Usage

Possible causes:

- Excessive compression
- Too many concurrent streams
- Very high connection rate
- Excessive logging
- Very aggressive heartbeat intervals
- Reconnection loops

Inspect CPU usage:

```bash
top
```

or:

```bash
htop
```

Inspect the process:

```bash
ps aux | grep tunnel
```

If Snappy compression is enabled, evaluate whether the traffic is already compressed.

Compressing:

```text
JPEG
ZIP
GZIP
HTTPS
```

may provide little benefit while increasing CPU usage.

---

# 16. High Memory Usage

Check:

```bash
free -h
```

and:

```bash
ps aux --sort=-%mem | head
```

Large numbers of concurrent streams can increase memory usage.

Recommended production practices:

- Limit unnecessary concurrent connections
- Monitor stream counts
- Avoid unbounded buffering
- Use reasonable connection limits
- Monitor long-lived idle sessions

---

# 🔐 Production Security Checklist

Before exposing the service to the Internet:

```text
[ ] Change all default secrets
[ ] Protect private keys
[ ] Enable TLS
[ ] Prefer mTLS for trusted clients
[ ] Use unique client IDs
[ ] Restrict dashboard access
[ ] Configure firewall rules
[ ] Disable unnecessary public ports
[ ] Keep Go dependencies updated
[ ] Run with least privilege
[ ] Monitor authentication failures
[ ] Monitor connection uptime
[ ] Configure automatic restart
[ ] Back up configuration securely
[ ] Never commit secrets to Git
```

---

# 🚨 Secrets Management

Never commit:

```text
production token
private key
client private certificate
API credential
password
```

to Git.

Before committing:

```bash
git diff
```

Check the repository:

```bash
git status
```

If a secret has already been committed, simply deleting it from the latest working tree is not sufficient. Rotate the secret and remove it from repository history when appropriate.

---

# 📈 Production Monitoring

At minimum monitor:

```text
Client connection state
Tunnel uptime
Reconnect count
Authentication failures
Active streams
CPU usage
Memory usage
Network throughput
TLS certificate expiration
```

Useful Linux commands:

```bash
systemctl status go-reverse-tunnel-server
```

```bash
journalctl -u go-reverse-tunnel-server --since "1 hour ago"
```

```bash
ss -s
```

```bash
free -h
```

```bash
uptime
```

---

# 🧪 Development Workflow

Run tests:

```bash
go test ./...
```

Run with race detection:

```bash
go test -race ./...
```

Format:

```bash
gofmt -w .
```

Verify dependencies:

```bash
go mod tidy
```

Build:

```bash
go build ./...
```

---

# 🌍 Platform Support

| Platform | Server | Client |
|---|---:|---:|
| Linux AMD64 | ✅ | ✅ |
| Linux ARM64 | ✅ | ✅ |
| Android / Termux | — | ✅ |
| Windows AMD64 | — | ✅ |
| macOS | Possible | Possible |

The exact availability of a platform depends on the dependencies and build target used.

---

# 🧩 Design Principles

The project follows several important engineering principles:

### 1. One Transport, Multiple Streams

Avoid unnecessary independent TCP connections.

### 2. Secure by Default

TLS should be preferred for production deployments.

### 3. Explicit Identity

Every client should have a unique identity.

### 4. Resilient Connections

Temporary network failures should not require manual intervention.

### 5. Observable Operation

A tunnel should be diagnosable through logs and monitoring.

### 6. Minimal Configuration

The most common deployment should require only a small JSON configuration.

---

# 📜 License

MIT License.

See:

```text
LICENSE
```

for the complete license text.

---

# 🤝 Contributing

Contributions are welcome.

Recommended workflow:

```bash
git checkout -b feature/my-feature
```

Make changes.

Run:

```bash
gofmt -w .
go test ./...
go vet ./...
```

Commit:

```bash
git add .
git commit -m "feat: add my feature"
```

Push:

```bash
git push origin feature/my-feature
```

Then open a Pull Request.

---

# ⭐ Support the Project

If `go-reverse-tunnel` is useful to you:

- Star the repository
- Report bugs
- Submit improvements
- Improve documentation
- Share performance results
- Contribute tests

---

# ⚠️ Disclaimer

This software is a general-purpose networking tool.

Users are responsible for complying with:

- Local laws
- Network policies
- VPS provider terms
- ISP policies
- Organizational security policies

Do not use the project to access systems or networks without authorization.

---

# 👨‍💻 Author

**sepantartd**

Project:

```text
go-reverse-tunnel
```

GitHub:

```text
github.com/sepantartd/go-reverse-tunnel
```
