# Go Reverse Tunnel

یک ابزار معکوس‌سازی تونل (Reverse Tunnel) پرقدرت، امن و آماده استفاده در محیط‌های پروداکشن که با زبان Go پیاده‌سازی شده است. این پروژه به شما امکان می‌دهد سرویس‌های داخلی خود را که پشت NAT یا فایروال هستند، به صورت امن و از طریق اتصالات TCP چندگانه‌سازی‌شده (Multiplexed) در اینترنت منتشر کنید.

## ویژگی‌های کلیدی

* **مالتی‌پلاکسینگ پیشرفته**: استفاده از `yamux` جهت مدیریت چندین جریان دیتایی روی یک اتصال واحد TCP.
* **امنیت بالا و TLS/mTLS**: رمزنگاری اتصالات با TLS 1.2+ و پشتیبانی از احرازهویت دوطرفه (mTLS).
* **احرازهویت مقاوم در برابر Replay Attack**: مکانیزم Challenge-Response بر پایه HMAC-SHA256.
* **مدیریت بهینه منابع**: محدودسازی اتصالات همزمان با Semaphore و کاهش تخصیص حافظه با `sync.Pool`.
* **پایش و لاگینگ ساختاریافته**: ارائه لاگ‌های JSON با `log/slog` و خروجی استانداردهای پرومتیوس (`/metrics`).
* **داشبورد مدیریتی وب**: پنل گرافیکی وب (Embedded) جهت مشاهده وضعیت لحظه‌ای کلاینت‌ها و پورت‌ها.
* **آماده‌سازی پروداکشن**: ایمیج چندمرحله‌ای Docker و پایپ‌لاین آماده GitHub Actions CI/CD.

## معماری سیستم

```
[ کاربر عمومی ] ---> [ سرور تونل معکوس ] <=== (تونل امن TLS) ===> [ کلاینت تونل ] ---> [ سرویس محلی ]
```

## نصب و راه‌اندازی

### استفاده از داکر

```bash
docker pull ghcr.io/sepantartd/go-reverse-tunnel:latest
```

### کامپایل از سورس‌کد

```bash
git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
cd go-reverse-tunnel
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client
```

## نمونه پیکربندی

### فایل کانفیگ سرور (`server_config.json`)

```json
{
  "control_addr": "0.0.0.0:8080",
  "token": "توکن-امنیتی-بسیار-قوی",
  "tls_cert_file": "/path/to/cert.pem",
  "tls_key_file": "/path/to/key.pem",
  "insecure_allow_plaintext": false,
  "dashboard_addr": "0.0.0.0:8081",
  "dashboard_user": "admin",
  "dashboard_pass": "رمزعبور-داشبورد",
  "clients": [
    {
      "client_id": "app-service-1",
      "ports": [9001, 9002]
    }
  ]
}
```

### فایل کانفیگ کلاینت (`client_config.json`)

```json
{
  "server_addr": "tunnel.yourdomain.com:8080",
  "local_addr": "127.0.0.1:3000",
  "client_id": "app-service-1",
  "token": "توکن-امنیتی-بسیار-قوی",
  "tls_cert_file": "/path/to/client-cert.pem",
  "tls_key_file": "/path/to/client-key.pem",
  "insecure_allow_plaintext": false
}
```

## راهنمای اجرای سریع

۱. اجرا و بالا آوردن سرور تونل:
```bash
./bin/server -config server_config.json
```

۲. اجرا و اتصال کلاینت تونل:
```bash
./bin/client -config client_config.json
```

۳. دسترسی به سرویس محلی از طریق پورت عمومی سرور (مانند `http://tunnel.yourdomain.com:9001`).

---

## لایسنس

این پروژه تحت لایسنس MIT منتشر شده است. برای اطلاعات بیشتر فایل `LICENSE` را مطالعه کنید.
