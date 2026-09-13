<<<<<<< HEAD
# ⚡ go-reverse-tunnel

A high-performance, enterprise-grade secure reverse tunneling tool crafted in Go. Built for heavy-duty networking, it's meticulously optimized for mobile environments like **Termux/Android** as well as production Linux and Windows servers.

## 🔥 What Makes It Different?

Most tunneling tools are either bloated or fragile when faced with unstable mobile connections. **go-reverse-tunnel** bridges the gap by combining raw speed with robust resilience:

*   **TLS Multiplexing:** Powered by `hashicorp/yamux` to run multiple concurrent streams over a single secure TLS connection.
*   **Bandwidth Saver:** Integrated `golang/snappy` compression for lightning-fast data throughput, perfect for high-latency mobile networks.
*   **Multi-Client Routing:** A unified server architecture capable of managing multiple authenticated clients simultaneously using unique `client_id` routing.
*   **Smart Reconnection:** Client-side **Exponential Backoff** mechanism that gracefully handles network drops without flooding the server.
*   **Keep-Alive Heartbeat:** Active TCP/Yamux level heartbeat pulses to keep firewalls and mobile operators from dropping idle connections.
*   **Live Web Dashboard:** A sleek embedded HTTP monitoring panel (`/`) to track active client sessions and connection status in real-time.
*   **JSON Configuration:** Clean, file-based configuration structure for both server and client.

---

## ⚙️ Configuration

### Server Config (`server-config.json`)
```json
{
  "bind": "0.0.0.0:7000",
  "control_port": 7001,
  "web_port": 8081,
  "token": "secret-token-123",
  "cert": "cert.pem",
  "key": "key.pem"
}
Client Config (client-config.json)
{
  "server": "YOUR_SERVER_IP:7001",
  "token": "secret-token-123",
  "client_id": "client-01",
  "local": "127.0.0.1:8080"
}
🚀 Usage
1. Start the Server:
go run cmd/server/main.go -config=server-config.json
2. Start the Client:
go run cmd/client/main.go -config=client-config.json
3. Monitor:
Open your browser and visit http://YOUR_SERVER_IP:8081 to view the live dashboard.
📜 License
This project is licensed under the MIT License - see the LICENSE file for details.
=======
# go-reverse-tunnel
>>>>>>> ed0aa1639eb15b6a29677ae4130eb7602e29efe7
