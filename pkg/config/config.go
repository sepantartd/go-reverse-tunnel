package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

type ClientMapping struct {
	ClientID string `json:"client_id"`
	Ports    []int  `json:"ports"`
}

type ServerConfig struct {
	ControlAddr string          `json:"control_addr"`
	Token       string          `json:"token"`
	EnableTLS   bool            `json:"enable_tls"`
	CertFile    string          `json:"cert_file"`
	KeyFile     string          `json:"key_file"`
	CAFile      string          `json:"ca_file"`
	Clients     []ClientMapping `json:"clients"`
}

type ClientConfig struct {
	ServerAddr    string `json:"server_addr"`
	LocalAddr     string `json:"local_addr"`
	ClientID      string `json:"client_id"`
	Token         string `json:"token"`
	EnableTLS     bool   `json:"enable_tls"`
	CAFile        string `json:"ca_file"`
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

func validateHostPort(addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format '%s': %w", addr, err)
	}
	_ = host
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port '%s' in address '%s'", portStr, addr)
	}
	return nil
}

func (c *ServerConfig) Validate() error {
	if c.ControlAddr == "" {
		return errors.New("server control address cannot be empty")
	}
	if err := validateHostPort(c.ControlAddr); err != nil {
		return fmt.Errorf("invalid server control_addr: %w", err)
	}
	if c.Token == "" {
		return errors.New("server token cannot be empty")
	}
	if c.EnableTLS {
		if c.CertFile == "" || c.KeyFile == "" {
			return errors.New("TLS is enabled on server, but CertFile or KeyFile is missing")
		}
	}

	seenClients := make(map[string]bool)
	seenPorts := make(map[int]string)

	for _, client := range c.Clients {
		if client.ClientID == "" {
			return errors.New("client mapping has an empty ClientID")
		}
		if seenClients[client.ClientID] {
			return fmt.Errorf("duplicate ClientID found: %s", client.ClientID)
		}
		seenClients[client.ClientID] = true

		if len(client.Ports) == 0 {
			return fmt.Errorf("client %s has no public ports mapped", client.ClientID)
		}
		for _, port := range client.Ports {
			if port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port number %d for client %s", port, client.ClientID)
			}
			if owner, exists := seenPorts[port]; exists {
				return fmt.Errorf("port collision: port %d is mapped to both '%s' and '%s'", port, owner, client.ClientID)
			}
			seenPorts[port] = client.ClientID
		}
	}
	return nil
}

func (c *ClientConfig) Validate() error {
	if c.ServerAddr == "" {
		return errors.New("client server address cannot be empty")
	}
	if err := validateHostPort(c.ServerAddr); err != nil {
		return fmt.Errorf("invalid client server_addr: %w", err)
	}
	if c.LocalAddr != "" {
		if err := validateHostPort(c.LocalAddr); err != nil {
			return fmt.Errorf("invalid client local_addr: %w", err)
		}
	}
	if c.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	if c.Token == "" {
		return errors.New("authentication token cannot be empty")
	}
	if c.EnableTLS {
		if c.CertFile != "" && c.KeyFile == "" {
			return errors.New("CertFile specified without KeyFile")
		}
		if c.KeyFile != "" && c.CertFile == "" {
			return errors.New("KeyFile specified without CertFile")
		}
	}
	return nil
}
