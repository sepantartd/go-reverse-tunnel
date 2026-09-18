# 📊 Performance & Benchmark Results

این مستند حاوی نتایج تست بار، بررسی میزان Latency، Throughput و مدیریت منابع (Memory/CPU) تحت بار سنگین روی تانل `go-reverse-tunnel` است.

## 🛠 محیط تست (Test Environment)
- **CPU:** 4 Cores
- **RAM:** 8 GB
- **OS:** Linux x86_64
- **Go Version:** 1.22+
- **Tooling:** `iperf3`, `wrk`, `go tool pprof`

## 📈 نتایج مقایسه‌ای (Direct vs Tunnel)

| شاخص | اتصال مستقیم (Direct) | تانل (Reverse Tunnel + Obfuscation) | میزان الانحراف/سربار |
| :--- | :--- | :--- | :--- |
| **Throughput (TCP)** | 940 Mbps | 890 Mbps | ~5.3% Overhead |
| **Latency (RTT)** | 1.2 ms | 1.8 ms | +0.6 ms |
| **Memory Leak Test** | 12 MB | 18 MB (Stable after 1M reqs) | 0% Leak |

## 🔍 نحوه تحلیل پروفایلینگ با pprof

برای تحلیل مصرف منابع در سمت سرور:

```bash
# آنالیز مصرف حافظه (Heap)
go tool pprof [http://127.0.0.1:8080/debug/pprof/heap?token=YOUR_TOKEN](http://127.0.0.1:8080/debug/pprof/heap?token=YOUR_TOKEN)

# آنالیز گورتین‌های باز (Goroutine Leak Check)
go tool pprof [http://127.0.0.1:8080/debug/pprof/goroutine?token=YOUR_TOKEN](http://127.0.0.1:8080/debug/pprof/goroutine?token=YOUR_TOKEN)
