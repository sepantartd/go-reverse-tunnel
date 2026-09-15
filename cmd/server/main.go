package main

import (
	"flag"
	"log"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
	"github.com/sepantartd/go-reverse-tunnel/pkg/server"
)

func main() {
	configFile := flag.String("config", "server-config.json", "Path to server configuration file")
	flag.Parse()

	cfg, err := config.LoadServerConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load server config: %v", err)
	}

	srv := server.NewTunnelServer(cfg)
	log.Println("[Server] Starting Reverse Tunnel Server...")
	if err := srv.Start(); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}
