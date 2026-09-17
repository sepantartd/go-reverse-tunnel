# Go Reverse Tunnel

یک ابزار Reverse Tunneling قدرتمند، امن و سبک که به زبان Go نوشته شده است. این پروژه به شما اجازه می‌دهد سرویس‌های محلی (پشت NAT یا فایروال) را به سادگی و با امنیت بالا به اینترنت عمومی متصل کنید.

## ویژگی‌ها

- **امنیت در برابر Replay Attack:** بهره‌گیری از Challenge Nonce تصادفی ۳۲ بایتی یک‌بارمصرف برای هر نشست.
- **احراز هویت HMAC-SHA256:** اعتبارسنجی دقیق Token به همراه نگاشت سفارشی ClientID.
- **محدودسازی نرخ تلاش‌ها (Rate Limiting):** جلوگیری از حملات Brute-Force و ایجاد ترافیک نامتعارف بر روی سرور.
- **پشتیبانی از Auto-TLS (Let's Encrypt):** دریافت و تمدید خودکار گواهی TLS/HTTPS بدون نیاز به ابزار خارجی.
- **مخفی‌سازی ترافیک (Traffic Obfuscation):** ماسک‌گذاری الگوریتمی بر اساس XOR جهت عبور از سیستم‌های بازرسی عمیق بسته (DPI).
- **مالتی‌پلی‌کسینگ:** استفاده از کتابخانه Yamux جهت انتقال چند جریان داده روی یک اتصال TCP.
- **پشتیبانی از UDP Forwarding:** امکان تونل‌سازی ترافیک UDP در کنار TCP.
- **داشبورد وب و مانیتورینگ:** نمایش زنده وضعیت کلاینت‌ها، Uptime و ارائه Endpoint استاندارد Prometheus برای متریک‌ها.
- **وب‌هوک:** ارسال گزارش رویدادهای اتصال، قطع اتصال و خطاهای امنیتی.

## نصب و ساخت (Build)

### پیش‌نیازها
- نسخه Go 1.22 یا بالاتر

### ساخت فایل‌های اجرایی
```bash
# کلون کردن ریپازیتوری
git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
cd go-reverse-tunnel

# ساخت سرور و کلاینت
go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go
```

## ساخت برای ویندوز و Termux (اندروید)

### ساخت برای ویندوز (PowerShell)
```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -o bin/server.exe cmd/server/main.go
go build -o bin/client.exe cmd/client/main.go
```

### اجرای اسکریپت ساخت چندپلتفرمی
می‌توانید از اسکریپت‌های آماده پروژه استفاده کنید:
- **لینوکس / مک / ترموکس:** `bash build.sh`
- **ویندوز:** `powershell .\build.ps1`

## راهنمای اجرا روی Termux (اندروید)

برای اجرای کلاینت روی گوشی‌های اندرویدی از طریق Termux:

1. برنامه Termux را باز کرده و بسته‌های مورد نیاز را نصب کنید:
   ```bash
   pkg update && pkg install golang git
   ```
2. پروژه را کلون کرده و فایل کلاینت را بسازید:
   ```bash
   git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
   cd go-reverse-tunnel
   go build -o client cmd/client/main.go
   ```
3. فایل کانفیگ `client_config.json` را تنظیم کرده و اجرا کنید:
   ```bash
   ./client -config client_config.json
   ```

## پیکربندی (Configuration)

### تنظیمات سرور (`server_config.json`)
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

### تنظیمات کلاینت (`client_config.json`)
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

## تنظیمات پیشرفته

برای شبکه‌های با تاخیر بالا (High Latency) یا اتصالات دارای اختلال، می‌توانید پارامترهای ترافیکی Yamux را در فایل‌های کانفیگ تنظیم کنید:

### پارامترهای Yamux
- `keepalive_interval_sec`: فاصله زمانی ارسال سیگنال Ping برای زنده نگه‌داشتن اتصال (پیش‌فرض: ۳۰ ثانیه).
- `max_stream_window_size`: حداکثر حجم پنجره ارسال ترافیک روی هر استریم به بایت (پیش‌فرض: ۲۵۶ کیلوبایت).

### تست و فعال‌سازی Obfuscation
برای عبور از سیستم‌های بازرسی عمیق بسته (DPI):
1. مقدار `"enable_obfuscation": true` را در هر دو فایل تنظیمات سرور و کلاینت قرار دهید.
2. هاندشیک اولیه پیش از برقراری نشست TLS و Yamux به صورت ماسک‌شده تبادل خواهد شد.

## مجوز (License)
این پروژه تحت مجوز MIT منتشر شده است.
برای موارد بیشتر فایل LICENCE را مشاهده کنید.
