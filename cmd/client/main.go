package main

import (
	"context"
	"encoding/json"
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

	file, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("[Client] Failed to read config file: %v", err)
	}

	var cfg config.ClientConfig
	if err := json.Unmarshal(file, &cfg); err != nil {
		log.Fatalf("[Client] Failed to parse config JSON: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("[Client] Configuration validation failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[Client] Starting tunnel client daemon...")
	if err := client.RunClient(ctx, &cfg); err != nil {
		log.Fatalf("[Client] Client stopped with error: %v", err)
	}

	log.Println("[Client] Shutdown complete.")
}
