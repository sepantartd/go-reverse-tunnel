package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type ClientMapping struct {
	ClientID string   `json:"client_id"`
	Ports    []int    `json:"ports,omitempty"`
	UDPPorts []int    `json:"udp_ports,omitempty"`
	Ranges   []string `json:"ranges,omitempty"`
}

type ServerConfig struct {
	ControlAddr            string          `json:"control_addr"`
	Token                  string          `json:"token"`
	Clients                []ClientMapping `json:"clients"`
	TLSCertFile            string          `json:"tls_cert_file"`
	TLSKeyFile             string          `json:"tls_key_file"`
	TLSCAFile              string          `json:"tls_ca_file"`
	InsecureAllowPlaintext bool            `json:"insecure_allow_plaintext"`
	DashboardAddr          string          `json:"dashboard_addr"`
	DashboardUser          string          `json:"dashboard_user"`
	DashboardPass          string          `json:"dashboard_pass"`
	WebhookURL             string          `json:"webhook_url,omitempty"`
	EnableObfuscation      bool            `json:"enable_obfuscation,omitempty"`
}

type ClientConfig struct {
	ServerAddr            string `json:"server_addr"`
	LocalAddr             string `json:"local_addr"`
	ClientID              string `json:"client_id"`
	Token                 string `json:"token"`
	TLSCertFile           string `json:"tls_cert_file"`
	TLSKeyFile            string `json:"tls_key_file"`
	TLSCAFile             string `json:"tls_ca_file"`
	InsecureAllowPlaintext bool   `json:"insecure_allow_plaintext"`
	InsecureSkipVerify    bool   `json:"insecure_skip_verify"`
	EnableObfuscation      bool   `json:"enable_obfuscation,omitempty"`
}

func parsePortRange(r string) ([]int, error) {
	parts := strings.Split(r, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format %s (expected start-end)", r)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid start port in range %s", r)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid end port in range %s", r)
	}
	if start > end || start <= 0 || end > 65535 {
		return nil, fmt.Errorf("out of bound range %d-%d", start, end)
	}

	var ports []int
	for p := start; p <= end; p++ {
		ports = append(ports, p)
	}
	return ports, nil
}

func (c *ServerConfig) Validate() error {
	if c.ControlAddr == "" {
		return fmt.Errorf("control_addr is required")
	}
	if _, _, err := net.SplitHostPort(c.ControlAddr); err != nil {
		return fmt.Errorf("invalid control_addr format: %v", err)
	}
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	if (c.TLSCertFile == "" || c.TLSKeyFile == "") && !c.InsecureAllowPlaintext {
		return fmt.Errorf("TLS configuration missing (TLSCertFile/TLSKeyFile); set InsecureAllowPlaintext=true to bypass explicitly")
	}

	portMap := make(map[int]string)
	clientMap := make(map[string]bool)

	for i := range c.Clients {
		client := &c.Clients[i]
		if client.ClientID == "" {
			return fmt.Errorf("client_id cannot be empty")
		}
		if clientMap[client.ClientID] {
			return fmt.Errorf("duplicate client_id found: %s", client.ClientID)
		}
		clientMap[client.ClientID] = true

		for _, r := range client.Ranges {
			expanded, err := parsePortRange(r)
			if err != nil {
				return fmt.Errorf("client %s range error: %v", client.ClientID, err)
			}
			client.Ports = append(client.Ports, expanded...)
		}

		for _, port := range client.Ports {
			if port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port %d for client %s", port, client.ClientID)
			}
			if owner, exists := portMap[port]; exists {
				return fmt.Errorf("port collision: port %d requested by %s is already assigned to %s", port, client.ClientID, owner)
			}
			portMap[port] = client.ClientID
		}
	}
	return nil
}

func (c *ClientConfig) Validate() error {
	if c.ServerAddr == "" {
		return fmt.Errorf("server_addr is required")
	}
	if c.LocalAddr == "" {
		return fmt.Errorf("local_addr is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	if (c.TLSCertFile != "" && c.TLSKeyFile == "") || (c.TLSCertFile == "" && c.TLSKeyFile != "") {
		return fmt.Errorf("both TLSCertFile and TLSKeyFile must be provided for client mTLS")
	}
	if c.TLSCertFile == "" && !c.InsecureAllowPlaintext {
		return fmt.Errorf("TLS configuration missing for client; set InsecureAllowPlaintext=true to bypass explicitly")
	}
	return nil
}
