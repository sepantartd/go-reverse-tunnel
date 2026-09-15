[![CI](https://github.com/sepantartd/go-reverse-tunnel/actions/workflows/ci.yml/badge.svg)](https://github.com/sepantartd/go-reverse-tunnel/actions/workflows/ci.yml)

# ⚡ go-reverse-tunnel

[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows%20%7C%20Android-lightgrey)](#پشتیبانی-از-پلتفرمها)
[![Security](https://img.shields.io/badge/security-TLS%20%7C%20mTLS%20%7C%20HMAC-red)](#مدل-امنیتی)

**go-reverse-tunnel** یک سیستم Reverse Tunnel امن، پایدار و پرسرعت است که با زبان Go توسعه داده شده است.

هدف پروژه ایجاد یک تونل معکوس بین یک سرور عمومی و یک سرور یا کلاینت موجود در شبکه خصوصی، محدود یا دارای اتصال ورودی نامطمئن است.

این پروژه برای سناریوهای زیر طراحی شده است:

- انتقال چندین پورت TCP
- پشتیبانی از چند کلاینت همزمان
- اتصال‌های طولانی‌مدت
- شبکه‌های ناپایدار و دارای Latency بالا
- انتقال امن با TLS
- Multiplexing با Yamux
- Reconnection خودکار
- کاهش مصرف پهنای باند
- سرورهای Linux
- کلاینت Windows
- محیط Android/Termux
- اجرای دائمی با systemd

> **نام پروژه:** `go-reverse-tunnel`
>
> **توسعه‌دهنده:** `sepantartd`
>
> **مجوز:** MIT

---

# 📌 معرفی

معماری معمول پروژه از دو بخش تشکیل می‌شود:

```text
شبکه خصوصی / محدود
        │
        │ اتصال خروجی
        ▼
┌──────────────────────┐
│     Tunnel Client    │
│     VPS / Android    │
│                      │
│  client_id           │
│  TLS                 │
│  HMAC Authentication │
│  Yamux               │
└──────────┬───────────┘
           │
           │ یک اتصال اصلی
           │ رمزنگاری‌شده
           ▼
┌──────────────────────┐
│    Tunnel Server     │
│      Public VPS      │
│                      │
│  TLS / mTLS          │
│  Yamux Multiplexer   │
│  Multi-client Router │
│  Public Listeners    │
└───────┬─────┬────────┘
        │     │
        │     └──────────────► :443
        │
        └────────────────────► :80
                              :2222
```

اصل اصلی معماری بسیار ساده است:

> **کلاینت اتصال را به سمت سرور برقرار می‌کند و سرور از همان اتصال موجود، ترافیک ورودی عمومی را به سمت کلاینت برمی‌گرداند.**

به همین دلیل این معماری برای شبکه‌هایی که اتصال ورودی به آن‌ها دشوار یا محدود است مناسب است.

---

# 🚀 قابلیت‌های اصلی

| قابلیت | توضیح |
|---|---|
| 🔀 TCP Multiplexing | انتقال چند Stream منطقی روی یک اتصال اصلی با Yamux |
| 🌐 Multi-Port | انتقال همزمان پورت‌هایی مانند `80`، `443` و `2222` |
| 👥 Multi-Client | اتصال چند کلاینت احراز هویت‌شده |
| 🔐 TLS | رمزنگاری لایه انتقال |
| 🛡️ mTLS | احراز هویت دوطرفه با گواهی |
| 🔑 HMAC-SHA256 | احراز هویت Challenge-Response |
| ♻️ Replay Protection | جلوگیری از استفاده مجدد از Challenge قدیمی |
| 📉 Snappy | فشرده‌سازی تطبیقی داده |
| 🧦 SOCKS5 | پروکسی SOCKS5 داخلی سمت کلاینت |
| 📡 UDP-over-TCP | انتقال برخی ترافیک‌های UDP روی TCP |
| 💓 Heartbeat | حفظ اتصال‌های طولانی‌مدت |
| 🔄 Reconnection | اتصال مجدد خودکار با Exponential Backoff |
| 🆔 Client ID | شناسایی مستقل کلاینت‌ها |
| 📊 Web Dashboard | مانیتورینگ وضعیت و Uptime |
| ⚙️ JSON Config | پیکربندی ساده با JSON |
| 🐧 systemd | اجرای دائمی و Restart خودکار |
| 📱 Termux | مناسب برای Android/Termux |
| 🪟 Windows | امکان Build کلاینت برای Windows |
| 🧩 Shared Transport | استفاده چند سرویس از یک اتصال اصلی |

---

# 🏗️ معماری شبکه

```text
                         INTERNET
                            │
                            │
                 ┌──────────▼──────────┐
                 │      Public VPS     │
                 │     Tunnel Server   │
                 │                     │
                 │  :80   ─────────┐   │
                 │  :443  ───────┐ │   │
                 │  :2222 ─────┐ │ │   │
                 │             │ │ │   │
                 │       ┌─────▼─▼─▼─┐ │
                 │       │   Router   │ │
                 │       │ Multi-Port │ │
                 │       └──────┬─────┘ │
                 │              │       │
                 │         ┌────▼────┐  │
                 │         │  Yamux  │  │
                 │         │ Session │  │
                 │         └────┬────┘  │
                 │              │       │
                 │         TLS / mTLS   │
                 └──────────────┼───────┘
                                │
                    یک اتصال دائمی رمزنگاری‌شده
                                │
                 ┌──────────────▼──────────────┐
                 │      Private / Iran VPS     │
                 │        Tunnel Client        │
                 │                             │
                 │  ┌──────────────────────┐   │
                 │  │    TLS Transport     │   │
                 │  └──────────┬───────────┘   │
                 │             │               │
                 │        ┌────▼────┐          │
                 │        │  Yamux  │          │
                 │        │ Client  │          │
                 │        └────┬────┘          │
                 │             │               │
                 │     ┌───────┴────────┐      │
                 │     │                │      │
                 │  127.0.0.1:80  127.0.0.1:443
                 │     │                │      │
                 │    HTTP           HTTPS     │
                 │                             │
                 │       127.0.0.1:22          │
                 │             SSH             │
                 └─────────────────────────────┘
```

---

# 🔀 نحوه کار Multiplexing

به جای اینکه برای هر پورت یک اتصال مستقل ایجاد شود، کلاینت یک اتصال اصلی را نگه می‌دارد و چند Stream منطقی را روی آن ایجاد می‌کند.

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
      HTTP            HTTPS             SSH
```

مزیت این معماری:

- کاهش تعداد اتصال‌های TCP
- مدیریت ساده‌تر Session
- استفاده بهتر از اتصال اصلی
- امکان انتقال چند سرویس روی یک Tunnel
- مناسب برای شبکه‌هایی که تعداد Connectionهای همزمان را محدود می‌کنند

---

# 🔐 مدل امنیتی

امنیت سیستم به‌صورت چندلایه طراحی شده است.

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

---

# 🔒 TLS

TLS وظیفه محافظت از ارتباط در برابر شنود و تغییر داده‌ها را بر عهده دارد.

در محیط Production توصیه می‌شود از موارد زیر استفاده شود:

- Certificate معتبر
- Private Key محافظت‌شده
- TLS Configuration مناسب
- CA قابل اعتماد
- فرآیند منظم برای تمدید Certificate

فایل Private Key را با Permission محدود نگه دارید:

```bash
chmod 600 key.pem
```

---

# 🛡️ mTLS

در حالت Mutual TLS هر دو طرف یکدیگر را احراز هویت می‌کنند.

```text
Client                         Server
  │                              │
  │──── Client Certificate ─────►│
  │                              │
  │◄──── Server Certificate ─────│
  │                              │
  │══════ Secure Session ════════│
```

مزیت mTLS این است که فقط داشتن آدرس سرور برای برقراری Tunnel کافی نیست و می‌توان اتصال را به Certificateهای مورد اعتماد محدود کرد.

---

# 🔑 HMAC-SHA256 Challenge-Response

در احراز هویت Challenge-Response باید برای هر Session یک Nonce تصادفی و غیرقابل پیش‌بینی تولید شود.

```text
Client                         Server
  │                              │
  │──── ClientHello ────────────►│
  │                              │
  │◄──── Random Challenge ───────│
  │                              │
  │ HMAC-SHA256(secret, nonce)   │
  │                              │
  │──── HMAC Response ──────────►│
  │                              │
  │◄──── Authentication OK ──────│
  │                              │
  │══════ Yamux Session ═════════│
```

هدف این مکانیزم جلوگیری از Replay Attack است.

مثلاً:

```text
Challenge A → Response A → ACCEPT

Challenge B → Response A → REJECT
```

پاسخ قبلی نباید برای Challenge جدید قابل استفاده باشد.

---

# 🧱 ساختار پروژه

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

## توضیح پوشه‌ها

| مسیر | کاربرد |
|---|---|
| `cmd/server` | Entry Point سرور |
| `cmd/client` | Entry Point کلاینت |
| `pkg/tunnel` | منطق اصلی Tunnel |
| `server-config.json` | تنظیمات Server |
| `client-config.json` | تنظیمات Client |
| `build.sh` | اسکریپت Build |
| `cert.pem` | Certificate |
| `key.pem` | Private Key |
| `go.mod` | تعریف Go Module |
| `go.sum` | Checksum وابستگی‌ها |
| `LICENSE` | مجوز پروژه |

---

# 📦 پیش‌نیازها

## سرور

پیشنهاد می‌شود سرور دارای موارد زیر باشد:

- Linux VPS
- IP عمومی
- Go 1.26+
- دسترسی مناسب برای Bind کردن پورت‌ها
- Firewall قابل تنظیم
- Certificate و Private Key

## کلاینت

محیط‌های قابل استفاده:

- Linux
- Android + Termux
- Windows

کلاینت باید بتواند یک اتصال خروجی به Tunnel Server برقرار کند.

---

# 🛠️ نصب

ابتدا Repository را Clone کنید:

```bash
git clone https://github.com/sepantartd/go-reverse-tunnel.git
cd go-reverse-tunnel
```

بررسی Go:

```bash
go version
```

دانلود Dependencies:

```bash
go mod download
```

مرتب‌سازی Module:

```bash
go mod tidy
```

اجرای تست‌ها:

```bash
go test ./...
```

---

# 🔨 Build

## Build سرور

```bash
go build -o tunnel-server ./cmd/server
```

## Build کلاینت

```bash
go build -o tunnel-client ./cmd/client
```

## Linux AMD64

```bash
GOOS=linux GOARCH=amd64 \
go build -o tunnel-server ./cmd/server
```

## Android / Termux ARM64

```bash
GOOS=linux GOARCH=arm64 \
go build -o tunnel-client ./cmd/client
```

## Windows AMD64

```bash
GOOS=windows GOARCH=amd64 \
go build -o tunnel-client.exe ./cmd/client
```

---

# ⚙️ پیکربندی

پیکربندی پروژه با JSON انجام می‌شود.

دو فایل اصلی:

```text
server-config.json
client-config.json
```

---

# 🖥️ کانفیگ سرور

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

## توضیح پارامترها

| پارامتر | نوع | توضیح |
|---|---|---|
| `control_addr` | string | آدرس Listener اصلی Tunnel |
| `web_port` | integer | پورت Dashboard |
| `token` | string | Secret احراز هویت |
| `cert` | string | مسیر Certificate |
| `key` | string | مسیر Private Key |
| `public_binds` | array | پورت‌های عمومی |
| `public_binds[].port` | integer | شماره پورت عمومی |
| `public_binds[].name` | string | نام خوانا برای Listener |

هرگز Secret واقعی را داخل Repository عمومی قرار ندهید.

---

# 📱 کانفیگ کلاینت

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

## توضیح پارامترها

| پارامتر | نوع | توضیح |
|---|---|---|
| `server` | string | آدرس Tunnel Server |
| `token` | string | Secret احراز هویت |
| `client_id` | string | شناسه یکتا برای کلاینت |
| `forwards` | array | لیست Forwardها |
| `forwards[].remote_port` | integer | پورت عمومی روی Server |
| `forwards[].local` | string | مقصد محلی روی Client |

---

# 🔁 مثال Port Forwarding

فرض کنید Server این پورت‌ها را دارد:

```text
Public VPS
  :80
  :443
  :2222
```

و Client این سرویس‌ها را دارد:

```text
Private VPS
  :80
  :443
  :22
```

در این حالت:

```text
Internet
   │
   ├── :80 ───────► Client :80
   │
   ├── :443 ──────► Client :443
   │
   └── :2222 ─────► Client :22
```

یعنی:

```text
SERVER:2222
      │
      ▼
Tunnel
      │
      ▼
CLIENT:22
```

---

# ▶️ اجرای Server

اجرای مستقیم Binary:

```bash
./tunnel-server -config=server-config.json
```

برای Development:

```bash
go run ./cmd/server -config=server-config.json
```

---

# ▶️ اجرای Client

اجرای مستقیم:

```bash
./tunnel-client -config=client-config.json
```

برای Development:

```bash
go run ./cmd/client -config=client-config.json
```

---

# 📊 داشبورد وب

در صورتی که:

```json
{
  "web_port": 8081
}
```

تنظیم شده باشد، Dashboard روی این پورت قرار می‌گیرد:

```text
http://SERVER_IP:8081
```

اطلاعات قابل مانیتورینگ می‌تواند شامل موارد زیر باشد:

- وضعیت کلاینت‌ها
- Client ID
- وضعیت اتصال
- Uptime
- Sessionهای فعال
- سلامت Tunnel

توصیه می‌شود Dashboard را مستقیماً برای تمام اینترنت باز نکنید.

---

# 🧦 SOCKS5

در صورت فعال بودن قابلیت SOCKS5، کلاینت می‌تواند یک Proxy محلی در اختیار برنامه‌ها قرار دهد.

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
Destination
```

بهتر است SOCKS5 در حالت عادی فقط روی Loopback قرار بگیرد:

```text
127.0.0.1
```

و نه:

```text
0.0.0.0
```

مگر اینکه دسترسی Remote عمداً مورد نیاز باشد.

---

# 📡 UDP-over-TCP

UDP-over-TCP اجازه می‌دهد برخی ترافیک‌های UDP از طریق Tunnel مبتنی بر TCP عبور کنند.

کاربردهای احتمالی:

- DNS
- برخی سرویس‌های UDP
- سناریوهای WireGuard در محیط‌هایی که UDP مستقیم در دسترس نیست

معماری:

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

### نکته مهم

UDP-over-TCP از نظر کارایی معادل UDP واقعی نیست.

TCP دارای:

- ترتیب تضمین‌شده
- Retransmission
- Head-of-Line Blocking

است.

بنابراین برای برنامه‌های بسیار حساس به Latency، UDP واقعی در صورت امکان انتخاب بهتری است.

---

# 💓 Heartbeat

اتصال‌های طولانی ممکن است توسط موارد زیر قطع شوند:

- NAT
- Firewall
- اپراتور موبایل
- Router
- Idle Timeout

Heartbeat برای زنده نگه داشتن Session استفاده می‌شود.

```text
Client                         Server
  │                              │
  │──────── PING ───────────────►│
  │◄─────── PONG ────────────────│
  │                              │
  │──────── PING ───────────────►│
  │◄─────── PONG ────────────────│
```

Interval بسیار کم باعث مصرف بیشتر منابع می‌شود.

Interval بسیار زیاد نیز ممکن است باعث شود NAT یا Firewall اتصال را Idle تشخیص دهد.

---

# ♻️ Reconnection

در صورت قطع شدن اتصال، Client نباید دائماً با فاصله صفر تلاش کند.

الگوی پیشنهادی:

```text
Connection Lost
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

این روش با نام:

```text
Exponential Backoff
```

شناخته می‌شود.

پس از برقراری موفق اتصال، Backoff باید Reset شود.

---

# 👥 Multi-Client

هر Client باید یک `client_id` یکتا داشته باشد.

مثال:

```text
iran-vps-01
iran-vps-02
office-gateway
android-client-01
```

معماری:

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

از استفاده همزمان چند Client با یک `client_id` خودداری کنید مگر اینکه این رفتار عمداً طراحی شده باشد.

---

# 🐧 راه‌اندازی با systemd

برای Production روی Linux استفاده از systemd پیشنهاد می‌شود.

ساخت مسیر:

```bash
sudo mkdir -p /opt/go-reverse-tunnel
```

کپی فایل‌ها:

```bash
sudo cp tunnel-server /opt/go-reverse-tunnel/
sudo cp server-config.json /opt/go-reverse-tunnel/
sudo cp cert.pem /opt/go-reverse-tunnel/
sudo cp key.pem /opt/go-reverse-tunnel/
```

محافظت از Private Key:

```bash
sudo chmod 600 /opt/go-reverse-tunnel/key.pem
```

---

# ⚙️ سرویس systemd سرور

فایل زیر را بسازید:

```bash
sudo nano /etc/systemd/system/go-reverse-tunnel-server.service
```

محتوا:

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

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
```

سپس:

```bash
sudo systemctl daemon-reload
```

فعال‌سازی اجرای خودکار:

```bash
sudo systemctl enable go-reverse-tunnel-server
```

شروع سرویس:

```bash
sudo systemctl start go-reverse-tunnel-server
```

بررسی:

```bash
sudo systemctl status go-reverse-tunnel-server
```

مشاهده Log:

```bash
sudo journalctl -u go-reverse-tunnel-server -f
```

---

# 📱 سرویس systemd کلاینت

ساخت مسیر:

```bash
sudo mkdir -p /opt/go-reverse-tunnel
```

کپی:

```bash
sudo cp tunnel-client /opt/go-reverse-tunnel/
sudo cp client-config.json /opt/go-reverse-tunnel/
```

ساخت Service:

```bash
sudo nano /etc/systemd/system/go-reverse-tunnel-client.service
```

محتوا:

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

فعال‌سازی:

```bash
sudo systemctl daemon-reload
sudo systemctl enable go-reverse-tunnel-client
sudo systemctl start go-reverse-tunnel-client
```

بررسی:

```bash
sudo systemctl status go-reverse-tunnel-client
```

Log:

```bash
sudo journalctl -u go-reverse-tunnel-client -f
```

---

# 🔥 تنظیم Firewall با UFW

برای باز کردن پورت کنترل:

```bash
sudo ufw allow 7001/tcp
```

Dashboard:

```bash
sudo ufw allow 8081/tcp
```

HTTP:

```bash
sudo ufw allow 80/tcp
```

HTTPS:

```bash
sudo ufw allow 443/tcp
```

SSH Forward:

```bash
sudo ufw allow 2222/tcp
```

بررسی:

```bash
sudo ufw status
```

فقط پورت‌هایی را باز کنید که واقعاً استفاده می‌شوند.

---

# 🧱 iptables

بررسی Ruleها:

```bash
sudo iptables -L -n -v
```

باز کردن Control Port:

```bash
sudo iptables -A INPUT -p tcp --dport 7001 -j ACCEPT
```

HTTP:

```bash
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
```

HTTPS:

```bash
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

SSH Forward:

```bash
sudo iptables -A INPUT -p tcp --dport 2222 -j ACCEPT
```

Ruleها را متناسب با توزیع Linux خود Persist کنید.

---

# 🧪 تست Tunnel

بررسی Portهای Listening:

```bash
sudo ss -lntp
```

بررسی Control Port از یک سیستم دیگر:

```bash
nc -vz SERVER_IP 7001
```

تست HTTP:

```bash
curl -v http://SERVER_IP/
```

تست SSH:

```bash
ssh -p 2222 USER@SERVER_IP
```

---

# 🛠️ عیب‌یابی

# 1. خطای Port Already in Use

خطا:

```text
bind: address already in use
```

بررسی:

```bash
sudo ss -lntp | grep ':7001'
```

یا:

```bash
sudo lsof -i :7001
```

برای Port 80:

```bash
sudo lsof -i :80
```

برای Port 443:

```bash
sudo lsof -i :443
```

برای Port 2222:

```bash
sudo lsof -i :2222
```

وضعیت سرویس:

```bash
sudo systemctl status go-reverse-tunnel-server
```

یکی از رایج‌ترین دلایل این خطا این است که یک Instance از طریق systemd در حال اجرا است و شما همزمان همان برنامه را به صورت دستی اجرا کرده‌اید.

```text
systemd instance
       +
manual instance
       =
Port Conflict
```

---

# 2. کلاینت به Server متصل نمی‌شود

ابتدا:

```bash
ping SERVER_IP
```

سپس:

```bash
nc -vz SERVER_IP 7001
```

روی Server:

```bash
sudo ss -lntp | grep 7001
```

باید Listener مربوط به Control Port وجود داشته باشد.

مسیر بررسی:

```text
Client Network
      │
      ├── Internet
      │
      ├── Routing
      │
      ├── Server Firewall
      │
      ├── Cloud Firewall
      │
      └── Tunnel Listener
```

---

# 3. قطع شدن مداوم اتصال

دلایل احتمالی:

- NAT Timeout
- Packet Loss
- Latency بالا
- Firewall
- Idle Timeout
- مشکل TLS
- Heartbeat نامناسب
- Reconnection نامناسب

Log کلاینت:

```bash
journalctl -u go-reverse-tunnel-client -f
```

Log سرور:

```bash
journalctl -u go-reverse-tunnel-server -f
```

Packet Loss:

```bash
ping SERVER_IP
```

مسیر:

```bash
traceroute SERVER_IP
```

یا:

```bash
mtr SERVER_IP
```

تست TCP:

```bash
nc -vz SERVER_IP 7001
```

---

# 4. خطای HMAC Authentication

خطاهای احتمالی:

```text
authentication failed
invalid HMAC
challenge verification failed
unauthorized client
```

موارد زیر را بررسی کنید:

1. Secret سمت Client و Server صحیح باشد.
2. `client_id` صحیح باشد.
3. Challenge جدید تولید شود.
4. Response قدیمی مجدداً استفاده نشود.
5. سیستم‌ها Time بسیار متفاوت نداشته باشند.
6. Secret دارای Space یا کاراکتر اضافی نباشد.

فایل‌ها را بررسی کنید:

```bash
cat server-config.json
```

و:

```bash
cat client-config.json
```

در صورت انتشار Log، Secret واقعی را نمایش ندهید.

---

# 5. مشکل Replay Attack

در Challenge-Response صحیح:

```text
Challenge A → Response A → ACCEPT

Challenge B → Response A → REJECT
```

یعنی Response مربوط به Challenge قبلی نباید برای Challenge جدید پذیرفته شود.

Nonce باید:

- تصادفی
- غیرقابل پیش‌بینی
- منحصر‌به‌فرد
- دارای طول مناسب

باشد.

---

# 6. خطای TLS Certificate

خطاهای رایج:

```text
x509: certificate signed by unknown authority
```

یا:

```text
tls: failed to verify certificate
```

بررسی Certificate:

```bash
openssl x509 -in cert.pem -text -noout
```

بررسی تاریخ:

```bash
openssl x509 -in cert.pem -noout -dates
```

بررسی Private Key:

```bash
openssl rsa -in key.pem -check
```

نام دامنه استفاده‌شده باید با Certificate سازگار باشد.

---

# 7. Certificate و Private Key یکسان نیستند

Public Key مربوط به Certificate:

```bash
openssl x509 -in cert.pem -pubkey -noout | sha256sum
```

Public Key مربوط به Private Key:

```bash
openssl pkey -in key.pem -pubout | sha256sum
```

خروجی‌ها باید یکسان باشند.

اگر متفاوت باشند، Certificate و Private Key متعلق به یک Key Pair نیستند.

---

# 8. بررسی TLS با OpenSSL

برای IP:

```bash
openssl s_client -connect SERVER_IP:7001
```

برای Domain:

```bash
openssl s_client \
  -connect example.com:7001 \
  -servername example.com
```

در خروجی به موارد زیر توجه کنید:

```text
Certificate
Certificate Chain
Verify return code
```

---

# 9. مشکل mTLS

خطاهای رایج:

```text
client certificate required
unknown ca
bad certificate
certificate verify failed
```

بررسی کنید:

- Client Certificate وجود داشته باشد.
- Private Key وجود داشته باشد.
- Certificate توسط CA مورد اعتماد Server امضا شده باشد.
- Server همان CA را Trust کند.
- Private Key با Certificate مطابقت داشته باشد.
- Certificate منقضی نشده باشد.

برای Clientهای مختلف بهتر است Credentialهای مستقل استفاده شود:

```text
client-01 → certificate-01
client-02 → certificate-02
client-03 → certificate-03
```

---

# 10. Firewall باعث قطع اتصال شده است

UFW:

```bash
sudo ufw status verbose
```

iptables:

```bash
sudo iptables -L INPUT -n -v
```

Portهای Listening:

```bash
sudo ss -lntp
```

فراموش نکنید که ممکن است سه لایه Firewall وجود داشته باشد:

```text
Cloud Firewall
      +
Linux Firewall
      +
Network ACL
```

هر سه باید اجازه لازم را بدهند.

---

# 11. Dashboard باز نمی‌شود

بررسی:

```bash
sudo ss -lntp | grep 8081
```

تست Local:

```bash
curl http://127.0.0.1:8081/
```

اگر Local کار می‌کند ولی Remote کار نمی‌کند، موارد زیر را بررسی کنید:

```text
UFW
iptables
Cloud Firewall
Security Group
Provider ACL
```

برای محیط Production بهتر است Dashboard فقط برای IPهای مدیریتی قابل دسترسی باشد.

---

# 12. Client ID تکراری

اگر دو کلاینت این مقدار را داشته باشند:

```text
client_id = client-01
```

ممکن است Routing یا شناسایی آن‌ها دچار مشکل شود.

بهتر است:

```text
client-iran-01
client-iran-02
client-office-01
```

استفاده شود.

---

# 13. سرویس Local اجرا نمی‌شود

اگر کانفیگ چنین باشد:

```json
{
  "remote_port": 2222,
  "local": "127.0.0.1:22"
}
```

بررسی کنید که SSH واقعاً در حال Listening باشد:

```bash
sudo ss -lntp | grep ':22'
```

یا:

```bash
sudo systemctl status ssh
```

اگر هیچ سرویسی روی `127.0.0.1:22` فعال نباشد، Tunnel نمی‌تواند Forward را کامل کند.

---

# 14. Permission Denied برای Portهای پایین

پورت‌هایی مانند:

```text
80
443
53
```

معمولاً نیازمند Permission مناسب هستند.

اگر خطای:

```text
bind: permission denied
```

گرفتید، بررسی کنید که Process مجوز Bind کردن پورت را دارد.

راهکارها:

- اجرای سرویس با Permission مناسب
- استفاده از `CAP_NET_BIND_SERVICE`
- استفاده از Port بالاتر
- استفاده از Reverse Proxy

از دادن Permission بیشتر از نیاز خودداری کنید.

---

# 15. مصرف CPU بالا

دلایل احتمالی:

- Compression زیاد
- تعداد Stream بالا
- Connectionهای بسیار زیاد
- Logging زیاد
- Heartbeat بیش‌ازحد
- Reconnection Loop

بررسی:

```bash
top
```

یا:

```bash
htop
```

بررسی Process:

```bash
ps aux | grep tunnel
```

توجه کنید که فشرده‌سازی داده‌هایی مانند موارد زیر ممکن است سود زیادی نداشته باشد:

```text
JPEG
ZIP
GZIP
HTTPS
```

در چنین شرایطی Compression می‌تواند CPU بیشتری مصرف کند بدون اینکه حجم داده به شکل قابل توجهی کاهش پیدا کند.

---

# 16. مصرف RAM بالا

بررسی:

```bash
free -h
```

و:

```bash
ps aux --sort=-%mem | head
```

تعداد زیاد Streamهای همزمان می‌تواند Memory مصرفی را افزایش دهد.

برای Production:

- Connection Limit منطقی تعیین کنید.
- Bufferها را کنترل کنید.
- Streamهای Idle را مدیریت کنید.
- Memory Usage را مانیتور کنید.

---

# 🔐 چک‌لیست امنیت Production

قبل از استفاده واقعی:

```text
[ ] Secretهای پیش‌فرض تغییر داده شده‌اند
[ ] Private Key محافظت شده است
[ ] TLS فعال است
[ ] mTLS در صورت نیاز فعال است
[ ] Client IDها یکتا هستند
[ ] Dashboard محدود شده است
[ ] Firewall تنظیم شده است
[ ] پورت‌های غیرضروری بسته هستند
[ ] Dependencies به‌روز هستند
[ ] سرویس با کمترین Permission اجرا می‌شود
[ ] Authentication Failureها مانیتور می‌شوند
[ ] Uptime مانیتور می‌شود
[ ] systemd Restart فعال است
[ ] Secretها داخل Git قرار نگرفته‌اند
```

---

# 🚨 مدیریت Secretها

هرگز موارد زیر را Commit نکنید:

```text
Production Token
Private Key
Client Private Certificate
Password
API Credential
```

قبل از Commit:

```bash
git diff
```

و:

```bash
git status
```

را بررسی کنید.

اگر Secret قبلاً داخل Git Commit شده است، صرفاً حذف آن از فایل فعلی کافی نیست؛ Secret باید Rotate شود و در صورت نیاز از History نیز پاک‌سازی شود.

---

# 📈 مانیتورینگ Production

حداقل موارد زیر را مانیتور کنید:

```text
Client Connection State
Tunnel Uptime
Reconnect Count
Authentication Failures
Active Streams
CPU Usage
Memory Usage
Network Throughput
TLS Certificate Expiration
```

دستورات کاربردی:

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

# 🧪 توسعه و تست

اجرای تست‌ها:

```bash
go test ./...
```

Race Detector:

```bash
go test -race ./...
```

Format کردن:

```bash
gofmt -w .
```

بررسی Dependencies:

```bash
go mod tidy
```

Build:

```bash
go build ./...
```

---

# 🌍 پشتیبانی از پلتفرم‌ها

| پلتفرم | Server | Client |
|---|---:|---:|
| Linux AMD64 | ✅ | ✅ |
| Linux ARM64 | ✅ | ✅ |
| Android / Termux | — | ✅ |
| Windows AMD64 | — | ✅ |
| macOS | قابل Build | قابل Build |

پشتیبانی نهایی به Dependencyها و Build Target انتخاب‌شده بستگی دارد.

---

# 🧩 اصول طراحی

### 1. یک Transport و چند Stream

به جای ایجاد Connection مستقل برای هر سرویس، از Multiplexing استفاده می‌شود.

### 2. امنیت پیش‌فرض

برای Production استفاده از TLS توصیه می‌شود.

### 3. هویت مشخص

هر Client باید هویت مشخص و قابل تشخیص داشته باشد.

### 4. تحمل خطا

قطع موقت شبکه نباید باعث نیاز به Restart دستی شود.

### 5. قابلیت مانیتورینگ

وضعیت Tunnel باید از طریق Log و Dashboard قابل بررسی باشد.

### 6. Configuration ساده

راه‌اندازی معمول باید با چند مقدار ساده در JSON امکان‌پذیر باشد.

---

# 📜 مجوز

این پروژه تحت مجوز:

```text
MIT License
```

منتشر شده است.

متن کامل مجوز در فایل زیر قرار دارد:

```text
LICENSE
```

---

# 🤝 مشارکت

برای مشارکت:

```bash
git checkout -b feature/my-feature
```

پس از اعمال تغییرات:

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

سپس Pull Request ایجاد کنید.

---

# ⭐ حمایت از پروژه

اگر پروژه برای شما مفید بود:

- Repository را Star کنید.
- Bugها را گزارش کنید.
- Documentation را بهتر کنید.
- تست اضافه کنید.
- Performance Report ارسال کنید.
- Pull Request بفرستید.

---

# ⚠️ Disclaimer

این پروژه یک ابزار عمومی برای Networking و Reverse Tunneling است.

مسئولیت استفاده از آن بر عهده کاربر است.

کاربر باید قوانین مربوط به:

- کشور محل استفاده
- ISP
- ارائه‌دهنده VPS
- شبکه سازمانی
- قوانین دسترسی به سیستم‌ها

را رعایت کند.

از این پروژه برای دسترسی بدون مجوز به سیستم‌ها یا شبکه‌ها استفاده نکنید.

---

# 👨‍💻 توسعه‌دهنده

**sepantartd**

نام پروژه:

```text
go-reverse-tunnel
```

GitHub:

```text
github.com/sepantartd/go-reverse-tunnel
```
