<p align="center">
  <h1 align="center">Go Reverse Tunnel</h1>
  <p align="center">
    High-performance, secure & lightweight reverse tunneling tool written in Go
  </p>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge" alt="License"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/releases"><img src="https://img.shields.io/github/v/release/sepantartd/go-reverse-tunnel?style=for-the-badge&logo=github&color=blue" alt="Release"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/stargazers"><img src="https://img.shields.io/github/stars/sepantartd/go-reverse-tunnel?style=for-the-badge&logo=github" alt="Stars"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/network/members"><img src="https://img.shields.io/github/forks/sepantartd/go-reverse-tunnel?style=for-the-badge&logo=github" alt="Forks"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/actions"><img src="https://img.shields.io/github/actions/workflow/status/sepantartd/go-reverse-tunnel/ci.yml?branch=main&style=for-the-badge&logo=githubactions&logoColor=white&label=CI" alt="CI"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/releases"><img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-blue?style=for-the-badge&logo=linux&logoColor=white" alt="Platform"></a>
</p>

A high-performance, secure, and lightweight **Reverse Tunneling** tool written in Go. It enables you to expose your local servers behind NATs or firewalls to the public internet through a server with a public IP address.

---

## 🌟 Key Features

- **Dynamic Port Allocation:** Set port to `0` and the server will automatically assign an available port from a dynamic pool to the client.
- **Multiplexing with Yamux:** Tunnel thousands of connections concurrently over a single underlying TCP/TLS connection.
- **Traffic Obfuscation:** Mask raw TCP handshake data before TLS to bypass Deep Packet Inspection (DPI) systems.
- **HMAC-SHA256 Authentication:** Secure challenge-response authentication using random nonces without transferring secrets over the wire.
- **TLS & Auto-TLS (Let's Encrypt):** Built-in support for custom TLS certificates and automatic Let's Encrypt SSL provisioning.
- **UDP Traffic Forwarding:** Native tunneling support for UDP traffic (games, DNS, VoIP, etc.).
- **Built-in Rate Limiting:** Protect control connection endpoints from abuse and DoS attacks using IP-based rate limiting.
- **Dashboard, Metrics & pprof:** Prometheus metrics export, live status dashboard, and token-protected `pprof` endpoints.
- **Webhook Alerts:** Instant alerts for auth failures, client connections, and disconnections.

---

## 🚀 Quick Start

### One-Line Automated Installer (Linux & Termux)

Install the latest pre-compiled binary directly from [GitHub Releases](https://github.com/sepantartd/go-reverse-tunnel/releases):

```bash
bash <(curl -sL [https://raw.githubusercontent.com/sepantartd/go-reverse-tunnel/main/scripts/install.sh](https://raw.githubusercontent.com/sepantartd/go-reverse-tunnel/main/scripts/install.sh))
```

---

## 🛠️ Usage Guide

### 1. Running the Server

`server.json`:

```json
{
  "control_addr": ":8080",
  "token": "my-super-secret-token",
  "log_level": "info",
  "dashboard_addr": ":9090",
  "enable_obfuscation": true,
  "dynamic_port_min": 40000,
  "dynamic_port_max": 50000,
  "clients": [
    {
      "client_id": "app-1",
      "ports": [0, 8081]
    }
  ]
}
```

Run command:

```bash
go-reverse-tunnel -config server.json
```

### 2. Running the Client

`client.json`:

```json
{
  "server_addr": "SERVER_IP:8080",
  "client_id": "app-1",
  "token": "my-super-secret-token",
  "local_target": "127.0.0.1:3000",
  "enable_obfuscation": true,
  "log_level": "info"
}
```

Run command:

```bash
go-reverse-tunnel -config client.json
```

---

## 📊 Dashboard & Profiling

When `dashboard_addr` is configured, protected endpoints require authorization via `token`:

- **Dashboard:** `http://SERVER_IP:9090/dashboard?token=my-super-secret-token`
- **Prometheus Metrics:** `http://SERVER_IP:9090/metrics?token=my-super-secret-token`
- **pprof Profiler:** `http://SERVER_IP:9090/debug/pprof/?token=my-super-secret-token`

---

## 📄 License

Distributed under the [MIT License](LICENSE).
