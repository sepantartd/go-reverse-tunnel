# Go Reverse Tunnel

A simple, secure, and production-ready reverse tunneling tool written in Go, designed for personal use and managing private network access behind NAT/firewalls.

## Features
- **Real TLS Support**: Enforced TLS 1.2+ encryption for control connections.
- **Multi-Client Routing**: Directs public port traffic to specific authenticated client IDs.
- **Robust Reconnection**: Exponential backoff with jitter to prevent server overload.
- **Config Validation**: Strict checks for tokens, ports, and addresses at startup.
- **Graceful Shutdown**: Clean resource cleanup and signal handling.
- **Secure Health Dashboard**: Token-protected status endpoint.

## Project Structure
- `cmd/server`: Entry point for the tunnel server.
- `cmd/client`: Entry point for the tunnel client.
- `pkg/server`: Server core, multi-client routing, TLS setup, and dashboard.
- `pkg/client`: Client connection and reconnect logic.
- `pkg/config`: Configuration structures and validation.
- `pkg/protocol`: TLS configurations and cryptographic helpers.
- `pkg/tunnel`: Multiplexed session management.

## Getting Started

### Build
```bash
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client
```

### Run Server
```bash
./bin/server -config configs/server.json
```

### Run Client
```bash
./bin/client -config configs/client.json
```

## Known Limitations
- Designed for limited single or multi-client personal setups, not massive enterprise deployments.
- Requires proper firewall/port forwarding configuration on the public server.
