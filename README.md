# Go Reverse Tunnel

A high-performance, secure, and production-ready reverse tunneling solution written in Go. It enables secure exposure of local services behind NATs or firewalls to the public internet using multiplexed TCP connections.

## Key Features

* **Multiplexed Connections**: Powered by `yamux` for efficiently handling multiple streams over a single TCP connection.
* **Security & TLS/mTLS**: Enforces TLS 1.2+ encryption with optional mutual TLS (mTLS) authentication.
* **Replay-Resistant Auth**: Implements HMAC-SHA256 challenge-response authentication.
* **Resource Optimization**: Built-in connection limiting via semaphores and zero-allocation memory pooling with `sync.Pool`.
* **Observability**: Structured JSON logging (`log/slog`) and native Prometheus metrics (`/metrics`).
* **Web Management Dashboard**: Embedded Single Page Application (SPA) dashboard for live connection tracking.
* **Production Ready**: Fully dockerized multi-stage builds and automated GitHub Actions CI/CD pipelines.

## Architecture Overview

```
[ Public Client ] ---> [ Reverse Tunnel Server ] <=== (Multiplexed TLS Tunnel) ===> [ Tunnel Client ] ---> [ Local Service ]
```

## Installation

### Using Docker

```bash
docker pull ghcr.io/sepantartd/go-reverse-tunnel:latest
```

### From Source

```bash
git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
cd go-reverse-tunnel
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client
```

## Configuration

### Server Configuration (`server_config.json`)

```json
{
  "control_addr": "0.0.0.0:8080",
  "token": "your-strong-secret-token",
  "tls_cert_file": "/path/to/cert.pem",
  "tls_key_file": "/path/to/key.pem",
  "insecure_allow_plaintext": false,
  "dashboard_addr": "0.0.0.0:8081",
  "dashboard_user": "admin",
  "dashboard_pass": "securepassword",
  "clients": [
    {
      "client_id": "app-service-1",
      "ports": [9001, 9002]
    }
  ]
}
```

### Client Configuration (`client_config.json`)

```json
{
  "server_addr": "tunnel.yourdomain.com:8080",
  "local_addr": "127.0.0.1:3000",
  "client_id": "app-service-1",
  "token": "your-strong-secret-token",
  "tls_cert_file": "/path/to/client-cert.pem",
  "tls_key_file": "/path/to/client-key.pem",
  "insecure_allow_plaintext": false
}
```

## Quick Start

1. Start the Tunnel Server:
   ```bash
   ./bin/server -config server_config.json
   ```

2. Start the Tunnel Client:
   ```bash
   ./bin/client -config client_config.json
   ```

3. Access your local service remotely via the exposed public server port (e.g., `http://tunnel.yourdomain.com:9001`).

---

## License

Distributed under the MIT License. See `LICENSE` for more information.
