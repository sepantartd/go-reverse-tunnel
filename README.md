# Go Reverse Tunnel

A high-performance, secure, and lightweight reverse tunneling solution written in Go. Easily expose local services behind NAT or firewalls to the public internet with built-in TLS, Let's Encrypt (Auto-TLS), Traffic Obfuscation, UDP Forwarding, Rate Limiting, and a Web Dashboard.

## Features

- **Cryptographic Replay Protection:** Single-use 32-byte random challenge nonce per session.
- **HMAC-SHA256 Authentication:** Secure token validation with custom ClientID mapping.
- **Rate Limiting:** Built-in connection rate limiting per IP to prevent brute-force attacks.
- **Auto-TLS (Let's Encrypt):** Native support for automatic HTTPS/TLS certificate management.
- **Traffic Obfuscation:** XOR-based handshake masking to bypass strict DPI inspection systems.
- **Multiplexing:** Powered by Yamux for high-throughput stream multiplexing over a single TCP connection.
- **UDP Forwarding:** Tunnel UDP traffic alongside standard TCP streams.
- **Web Dashboard & Metrics:** Real-time client session monitoring and Prometheus `/metrics` endpoint.
- **Webhook Notifications:** Instant alerts for connection events and security failures.

## Installation & Building

### Prerequisites
- Go 1.22 or higher

### Build from Source
```bash
# Clone the repository
git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
cd go-reverse-tunnel

# Build binaries
go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go
```

## Building for Windows & Termux (Android)

### Windows (PowerShell)
```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -o bin/server.exe cmd/server/main.go
go build -o bin/client.exe cmd/client/main.go
```

### Cross-Platform Build Scripts
You can use the provided build scripts for compilation:
- **Linux / macOS / Termux:** `bash build.sh`
- **Windows:** `powershell .\build.ps1`

## Running on Termux (Android)

To run the client directly on an Android device via Termux:

1. Open Termux and install the required packages:
   ```bash
   pkg update && pkg install golang git
   ```
2. Clone the repository and build the client binary:
   ```bash
   git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
   cd go-reverse-tunnel
   go build -o client cmd/client/main.go
   ```
3. Configure `client_config.json` and start the tunnel:
   ```bash
   ./client -config client_config.json
   ```

## Configuration

### Server Configuration (`server_config.json`)
```json
{
  "control_addr": ":9090",
  "token": "your-secret-token",
  "log_level": "info",
  "enable_obfuscation": true,
  "dashboard_addr": ":8080",
  "yamux": {
    "keepalive_interval_sec": 15,
    "max_stream_window_size": 524288
  },
  "clients": [
    {
      "client_id": "app-server-1",
      "ports": [8080, 9000],
      "udp_ports": [5000]
    }
  ]
}
```

### Client Configuration (`client_config.json`)
```json
{
  "server_addr": "your-server.com:9090",
  "client_id": "app-server-1",
  "token": "your-secret-token",
  "local_addr": "127.0.0.1:80",
  "log_level": "info",
  "enable_obfuscation": true,
  "yamux": {
    "keepalive_interval_sec": 15,
    "max_stream_window_size": 524288
  }
}
```

## Advanced Configuration

For high-latency networks or unstable connections, you can fine-tune the internal Yamux multiplexer settings and dashboard configuration in the options file.

### Yamux Parameters
- `keepalive_interval_sec`: Heartbeat keep-alive ping interval in seconds (default: 30s).
- `max_stream_window_size`: Stream window size in bytes for high-throughput link tuning (default: 256KB).

### Dashboard and Metrics
- `dashboard_addr`: Specifies the binding address for the web dashboard and Prometheus `/metrics` endpoint (e.g., `:8080`).

### Traffic Obfuscation Testing
To mask tunnel control traffic against Deep Packet Inspection (DPI):
1. Set `"enable_obfuscation": true` in both `server_config.json` and `client_config.json`.
2. The initial control handshake will perform a seed mask exchange before initiating the TLS/Yamux session.

## License
MIT License.
