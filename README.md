# ⚡ go-reverse-tunnel

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20Android%20(Termux)-blue?style=for-the-badge)

یک ابزار تونل‌زنی معکوس (Reverse Tunneling) امن، فوق‌سریع و در سطح سازمانی که با زبان **Go** نوشته شده است. این پروژه برای شبکه‌های سنگین طراحی شده و به‌طور ویژه برای محیط‌های موبایل (مانند **Termux/Android**) و همچنین سرورهای تولید (Production) لینوکس و ویندوز بهینه‌سازی شده است.

---

## 🔥 چرا go-reverse-tunnel؟

بیشتر ابزارهای تونل‌زنی یا بیش‌ازحد سنگین (Bloated) هستند یا در مواجهه با اتصال ناپایدار اینترنت موبایل، شکننده عمل می‌کنند. **go-reverse-tunnel** این شکاف را با ترکیب سرعت خام و تاب‌آوری فوق‌العاده پر می‌کند:

*   🔄 **TLS Multiplexing:** قدرت‌گرفته از `hashicorp/yamux` برای اجرای چندین جریان (Stream) همزمان روی یک اتصال امن TLS واحد.
*   📉 **بهینه‌سازی پهنای باند:** فشرده‌سازی یکپارچه با `golang/snappy` برای افزایش سرعت انتقال داده، ایده‌آل برای شبکه‌های موبایل با تاخیر (Latency) بالا.
*   🛣️ **مسیریابی چند-کلاینتی (Multi-Client):** معماری سرور یکپارچه برای مدیریت همزمان چندین کلاینت احراز هویت‌شده با استفاده از مسیریابی منحصر‌به‌فرد `client_id`.
*   🧠 **اتصال مجدد هوشمند (Smart Reconnection):** مکانیزم **Exponential Backoff** در سمت کلاینت که قطعی‌های شبکه را به‌صورت نرم مدیریت کرده و از ارسال درخواست‌های هرز (Flood) به سرور جلوگیری می‌کند.
*   💓 **ضربان قلب (Keep-Alive Heartbeat):** پالس‌های فعال در سطح TCP/Yamux برای جلوگیری از قطع اتصال توسط فایروال‌ها یا اپراتورهای موبایل در حالت بیکاری (Idle).
*   📊 **داشبورد وب زنده:** یک پنل مانیتورینگ HTTP مدرن و تعبیه‌شده (`/`) برای ردیابی جلسات فعال کلاینت‌ها و وضعیت اتصال به‌صورت بلادرنگ (Real-time).
*   ⚙️ **پیکربندی JSON:** ساختار پیکربندی تمیز و مبتنی بر فایل برای هر دو بخش سرور و کلاینت.

---

## 📦 پیش‌نیازها

*   نصب بودن [Go](https://go.dev/dl/) (نسخه 1.21 یا بالاتر)
*   دسترسی به یک سرور با IP عمومی (برای اجرای بخش Server)
*   گواهی‌های TLS (برای امنیت تولید پیشنهاد می‌شود، هرچند برای تست می‌توانید از self-signed استفاده کنید)

---

## ⚙️ پیکربندی

### پیکربندی سرور (`server-config.json`)
```json
{
  "bind": "0.0.0.0:7000",
  "control_port": 7001,
  "web_port": 8081,
  "token": "secret-token-123",
  "cert": "cert.pem",
  "key": "key.pem"
}
```

### پیکربندی کلاینت (`client-config.json`)
```json
{
  "server": "YOUR_SERVER_IP:7001",
  "token": "secret-token-123",
  "client_id": "client-01",
  "local": "127.0.0.1:8080"
}
```
> 💡 **نکته:** مقدار `token` در هر دو فایل باید یکسان باشد تا احراز هویت با موفقیت انجام شود.

---

## 🚀 راهنمای استفاده

### ۱. راه‌اندازی سرور
سرور را با فایل پیکربندی مربوطه اجرا کنید:
```bash
go run cmd/server/main.go -config=server-config.json
```

### ۲. راه‌اندازی کلاینت
کلاینت را (مثلاً در محیط Termux یا سیستم محلی) اجرا کنید:
```bash
go run cmd/client/main.go -config=client-config.json
```

### ۳. مانیتورینگ و نظارت
مرورگر خود را باز کنید و به آدرس زیر بروید تا داشبورد زنده را مشاهده کنید:
```text
http://YOUR_SERVER_IP:8081
```

---

## 🛠️ ساخت و کامپایل (Build)

برای کامپایل پروژه و دریافت باینری‌های بهینه‌شده برای پلتفرم‌های مختلف:

```bash
# کامپایل برای لینوکس (سرور)
GOOS=linux GOARCH=amd64 go build -o tunnel-server cmd/server/main.go

# کامپایل برای اندروید/Termux (کلاینت)
GOOS=linux GOARCH=arm64 go build -o tunnel-client cmd/client/main.go

# کامپایل برای ویندوز (کلاینت)
GOOS=windows GOARCH=amd64 go build -o tunnel-client.exe cmd/client/main.go
```

---

## 📜 مجوز (License)

این پروژه تحت مجوز **MIT** منتشر شده است. برای جزئیات بیشتر، فایل `LICENSE` را در مخزن پروژه مطالعه کنید.

---

> 🌟 **اگر این پروژه برای شما مفید بود، لطفاً با دادن یک ⭐ به ما انگیزه ادامه توسعه را بدهید!**
