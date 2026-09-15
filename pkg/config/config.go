package config

import (
	"errors"
	"fmt"
)

type ClientMapping struct {
	ClientID string   `json:"client_id"`
	Ports    []int    `json:"ports"`
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
	ClientID      string `json:"client_id"`
	Token         string `json:"token"`
	EnableTLS     bool   `json:"enable_tls"`
	CAFile        string `json:"ca_file"`
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

func (c *ServerConfig) Validate() error {
	if c.ControlAddr == "" {
		return errors.New("server control address cannot be empty")
	}
	if c.Token == "" {
		return errors.New("server token cannot be empty")
	}
	if c.EnableTLS {
		if c.CertFile == "" || c.KeyFile == "" {
			return errors.New("TLS is enabled on server, but CertFile or KeyFile is missing")
		}
	}
	for _, client := range c.Clients {
		if client.ClientID == "" {
			return errors.New("client mapping has an empty ClientID")
		}
		if len(client.Ports) == 0 {
			return fmt.Errorf("client %s has no public ports mapped", client.ClientID)
		}
		for _, port := range client.Ports {
			if port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port number %d for client %s", port, client.ClientID)
			}
		}
	}
	return nil
}

func (c *ClientConfig) Validate() error {
	if c.ServerAddr == "" {
		return errors.New("client server address cannot be empty")
	}
	if c.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	if c.Token == "" {
		return errors.New("authentication token cannot be empty")
	}
	if c.EnableTLS && c.TLSSkipVerify && c.CAFile == "" {
		// Soft check: valid if explicitly configured
	}
	return nil
}
