# Go Reverse Tunnel

A clean, production-ready, secure, and modular reverse tunneling system written in Go, designed for personal use and managing private network access behind NAT/firewalls.

## Features
- **HMAC-SHA256 Authentication**: Secure challenge-response handshake with nonces.
- **Real TLS Support**: Enforced TLS 1.2+ for control connections with optional mTLS.
- **Multi-Client Routing**: Directs public port traffic dynamically to specific authenticated client IDs.
- **Robust Reconnection**: Exponential backoff with random jitter to prevent thundering herd.
- **Graceful Shutdown**: Signal handling (`SIGINT`, `SIGTERM`) with clean resource cleanup.
- **Secure Health Dashboard**: Token-protected status endpoint (`/status`).

## Getting Started

### Build Binaries
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
- Designed for limited personal setups (single or multi-client environments).
- Requires correct public server port forwarding/firewall configurations.
