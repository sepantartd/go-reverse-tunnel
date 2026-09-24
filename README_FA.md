<p align="center">
  <img src="https://img.shields.io/badge/Go-Reverse%20Tunnel-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Reverse Tunnel">
</p>

<p align="center">
  <strong>High-performance • Secure • Lightweight reverse tunneling</strong>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/releases"><img src="https://img.shields.io/github/v/release/sepantartd/go-reverse-tunnel?style=flat-square&logo=github&color=blue"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/stargazers"><img src="https://img.shields.io/github/stars/sepantartd/go-reverse-tunnel?style=flat-square&logo=github"></a>
  <a href="https://github.com/sepantartd/go-reverse-tunnel/actions"><img src="https://img.shields.io/github/actions/workflow/status/sepantartd/go-reverse-tunnel/ci.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=CI"></a>
</p>

یک ابزار **ریورس تانلینگ (Reverse Tunneling)** فوق‌العاده سریع، امن و سبک که به زبان Go پیاده‌سازی شده است. این ابزار به شما اجازه می‌دهد سرویس‌های محلی (Local) خود را که پشت NAT یا فایروال قرار دارند، از طریق یک سرور با IP عمومی به اینترنت معرفی و دسترس‌پذیر کنید.

---

## 🌟 ویژگی‌های کلیدی

- **تخصیص پورت پویا (Dynamic Port Allocation):** مانند ngrok، در صورت تنظیم پورت روی `0` سرور به‌صورت خودکار یک پورت آزاد رزرو کرده و به کلاینت اختصاص می‌دهد.
- **مالتی‌پلی‌کسینگ قدرتمند با Yamux:** مدیریت هزاران اتصال همزمان تنها روی یک کانال TCP/TLS.
- **مخفی‌سازی ترافیک (Obfuscation):** مبهم‌سازی ترافیک TCP خام قبل از TLS جهت عبور از سیستم‌های بازرسی عمیق بسته‌ها (DPI).
- **امنیت بالا با HMAC-SHA256:** احرازهویت کلاینت‌ها با Nonce تصادفی و رمزنگاری HMAC بدون ارسال توکن اصلی در شبکه.
- **پشتیبانی از TLS و Auto-TLS (Let's Encrypt):** دریافت و تمدید خودکار گواهی TLS مجانی بدون نیاز به ابزار جانبی.
- **پشتیبانی از UDP Forwarding:** امکان تانل کردن بسته‌های UDP برای بازی‌های آنلاین، DNS و سایر پروتکل‌های مبتنی بر UDP.
- **محدودکننده نرخ اتصال (Rate Limiting):** جلوگیری از حملات Brute-force و DoS با کنترل تعداد درخواست‌ها بر اساس IP.
- **داشبورد و مانیتورینگ:** ارائه متریک‌های Prometheus و اندپوینت‌های pprof (محافظت‌شده با توکن) جهت عیب‌یابی و پایش عملکرد.
- **وب‌هوک (Webhook Notification):** اطلاع‌رسانی اتصالات، قطع اتصال‌ها و خطاهای امنیتی.

---

## 🚀 نصب سریع

### اسکریپت نصب تک‌خطی (لینوکس و Termux)

برای نصب آخرین نسخه آماده از [GitHub Releases](https://github.com/sepantartd/go-reverse-tunnel/releases):

```bash
bash <(curl -sL [https://raw.githubusercontent.com/sepantartd/go-reverse-tunnel/main/scripts/install.sh](https://raw.githubusercontent.com/sepantartd/go-reverse-tunnel/main/scripts/install.sh))
```

---

## 🛠️ نحوه استفاده

### ۱. اجرای سرور (Server)

فایل کانفیگ سرور `server.json`:

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

دستور اجرا:

```bash
go-reverse-tunnel -config server.json
```

### ۲. اجرای کلاینت (Client)

فایل کانفیگ کلاینت `client.json`:

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

دستور اجرا:

```bash
go-reverse-tunnel -config client.json
```

---

## 📊 داشبورد و پروفایلینگ (pprof)

اگر `dashboard_addr` فعال باشد، اندپوینت‌های زیر از طریق `token` قابل دسترسی خواهند بود:

- **داشبورد وضعیت:** `http://SERVER_IP:9090/dashboard?token=my-super-secret-token`
- **متریک‌های Prometheus:** `http://SERVER_IP:9090/metrics?token=my-super-secret-token`
- **پروفایلر pprof:** `http://SERVER_IP:9090/debug/pprof/?token=my-super-secret-token`
- 
