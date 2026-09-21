package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/server"
)

func main() {
	configPath := flag.String("config", "server_config.json", "Path to server configuration file")
	flag.Parse()

	cfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load server config: %v", err)
	}

	logger := slog.Default()
	logger.Info("Starting Go Reverse Tunnel Server...")

	srv := server.NewServer(cfg)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("Server execution error", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server gracefully...")
	fmt.Println("Server stopped.")
}
