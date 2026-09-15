package main

import (
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

	// Handle graceful shutdown via OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("[Client] Received signal %v, exiting...", sig)
		os.Exit(0)
	}()

	log.Println("[Client] Starting tunnel client daemon...")
	if err := client.RunClient(&cfg); err != nil {
		log.Fatalf("[Client] Client stopped with error: %v", err)
	}
}
