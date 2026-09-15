package main

import (
"flag"
"log"
"os"
"os/signal"
"syscall"

"github.com/sepantartd/go-reverse-tunnel/pkg/client"
"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func main() {
configPath := flag.String("config", "configs/client.json", "Path to client configuration file")
flag.Parse()

log.Println("Starting Go Reverse Tunnel Client...")

cfg, err := config.LoadClientConfig(*configPath)
if err != nil {
log.Fatalf("Failed to load configuration: %v", err)
}

// تنظیم هندلینگ سیگنال‌ها برای خروج تمیز
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

clientInstance := client.NewTunnelClient(cfg)

go func() {
<-sigChan
log.Println("Shutting down client gracefully...")
os.Exit(0)
}()

// اجرای کلاینت
clientInstance.Start()
}
