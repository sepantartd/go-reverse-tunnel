package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/server"
)

func main() {
	configPath := flag.String("config", "configs/server.json", "Path to server configuration file")
	flag.Parse()

	file, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("[Server] Failed to read config file: %v", err)
	}

	var cfg config.ServerConfig
	if err := json.Unmarshal(file, &cfg); err != nil {
		log.Fatalf("[Server] Failed to parse config JSON: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("[Server] Configuration validation failed: %v", err)
	}

	srv := server.NewTunnelServer(&cfg)

	// Setup dashboard HTTP listener if token is configured
	if cfg.Token != "" {
		http.HandleFunc("/status", srv.HandleDashboard)
		go func() {
			log.Println("[Server] Status dashboard listening on :8080/status")
			if err := http.ListenAndServe(":8080", nil); err != nil {
				log.Printf("[Server] Dashboard server notice: %v", err)
			}
		}()
	}

	// Setup signal channel for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Println("[Server] Starting tunnel server core...")
		serverErr <- srv.Start()
	}()

	select {
	case sig := <-sigChan:
		log.Printf("[Server] Received signal %v, initiating graceful shutdown...", sig)
		if err := srv.Close(); err != nil {
			log.Printf("[Server] Error during server close: %v", err)
		}
	case err := <-serverErr:
		if err != nil {
			log.Printf("[Server] Server stopped with error: %v", err)
		}
	}

	log.Println("[Server] Shutdown complete.")
}
