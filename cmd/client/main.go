package main

import (
	"flag"
	"log"

	"github.com/sepantartd/go-reverse-tunnel/pkg/client"
	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func main() {
	configFile := flag.String("config", "client-config.json", "Path to client configuration file")
	flag.Parse()

	cfg, err := config.LoadClientConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load client config: %v", err)
	}

	cli := client.NewTunnelClient(cfg)
	log.Println("[Client] Starting Reverse Tunnel Client...")
	cli.Start()
}
