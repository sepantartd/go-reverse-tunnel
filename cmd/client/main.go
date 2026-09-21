package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sepantartd/go-reverse-tunnel/pkg/client"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func main() {
	configPath := flag.String("config", "client_config.json", "Path to client configuration file")
	flag.Parse()

	cfg, err := config.LoadClientConfig(*configPath)
	if err != nil {
		log.Fatalf("[Client] Failed to load config: %v", err)
	}

	logger := slog.Default()
	logger.Info("Starting Go Reverse Tunnel Client...")

	c := client.NewClient(cfg)

	go func() {
		if err := c.Start(); err != nil {
			logger.Error("Client stopped with error", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	c.Stop()
	logger.Info("Client shutdown complete.")
}
