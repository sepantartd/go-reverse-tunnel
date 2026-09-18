#!/bin/bash

# اسکریپت تست بنچمارک و بررسی نشت حافظه
SERVER_DASHBOARD="http://127.0.0.1:8080"
TOKEN="YOUR_SECRET_TOKEN"

echo "=== ۱. دریافت وضعیت Goroutines و Heap قبل از تست بار ==="
curl -s "$SERVER_DASHBOARD/debug/pprof/goroutine?debug=1&token=$TOKEN" > goroutine_before.txt
curl -s "$SERVER_DASHBOARD/debug/pprof/heap?debug=1&token=$TOKEN" > heap_before.txt

echo "=== ۲. ثبت ۳۰ ثانیه پروفایل CPU تحت بار ==="
# در صورت اجرای تست بار با iperf3 یا wrk این دستور CPU Profile تهیه می‌کند
curl -s "$SERVER_DASHBOARD/debug/pprof/profile?seconds=30&token=$TOKEN" > cpu_profile.pprof &

echo "در حال ضبط CPU Profile به مدت ۳۰ ثانیه..."
sleep 30

echo "=== ۳. دریافت وضعیت Goroutines و Heap بعد از تست بار ==="
curl -s "$SERVER_DASHBOARD/debug/pprof/goroutine?debug=1&token=$TOKEN" > goroutine_after.txt
curl -s "$SERVER_DASHBOARD/debug/pprof/heap?debug=1&token=$TOKEN" > heap_after.txt

echo "تست به پایان رسید. تحلیل فایل‌ها:"
echo "- تحلیل CPU: go tool pprof cpu_profile.pprof"
