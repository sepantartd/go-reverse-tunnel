package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ServerConfig defines the master configuration schema for the reverse tunnel server.
type ServerConfig struct {
	ControlAddr       string       `json:"control_addr"`
	Token             string       `json:"token"`
	LogLevel          string       `json:"log_level"`
	DashboardAddr     string       `json:"dashboard_addr"`
	EnableObfuscation bool         `json:"enable_obfuscation"`
	DynamicPortMin    int          `json:"dynamic_port_min"`
	DynamicPortMax    int          `json:"dynamic_port_max"`
	Clients           []ClientItem `json:"clients"`
	TLSCertFile       string       `json:"tls_cert_file"`
	TLSKeyFile        string       `json:"tls_key_file"`
	AutoTLSDomain     string       `json:"auto_tls_domain"`
	WebhookURL        string       `json:"webhook_url"`
}

// ClientItem defines pre-authorized client mappings on the server.
type ClientItem struct {
	ClientID string `json:"client_id"`
	Ports    []int  `json:"ports"`
}

// ClientConfig defines the configuration schema for the reverse tunnel client node.
type ClientConfig struct {
	ServerAddr            string `json:"server_addr"`
	ClientID              string `json:"client_id"`
	Token                 string `json:"token"`
	LocalTarget           string `json:"local_target"`
	EnableObfuscation     bool   `json:"enable_obfuscation"`
	LogLevel              string `json:"log_level"`
	TLSCACertFile         string `json:"tls_ca_cert_file"`
	TLSCertFile           string `json:"tls_cert_file"`
	TLSKeyFile            string `json:"tls_key_file"`
	TLSInsecureSkipVerify bool   `json:"tls_insecure_skip_verify"`
}

// LoadServerConfig reads, parses, validates and applies environment overrides for server configuration.
func LoadServerConfig(filepath string) (*ServerConfig, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("server configuration file not found at path: %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open server config file: %w", err)
	}
	defer file.Close()

	cfg := &ServerConfig{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse server JSON configuration: %w", err)
	}

	SetServerDefaults(cfg)
	ApplyServerEnvOverrides(cfg)

	if err := ValidateServerConfig(cfg); err != nil {
		return nil, fmt.Errorf("server configuration validation failed: %w", err)
	}

	return cfg, nil
}

// SaveServerConfig serializes and writes the server configuration to a JSON file.
func SaveServerConfig(filepath string, cfg *ServerConfig) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create server config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode server configuration: %w", err)
	}

	return nil
}

// LoadClientConfig reads, parses, validates and applies environment overrides for client configuration.
func LoadClientConfig(filepath string) (*ClientConfig, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("client configuration file not found at path: %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open client config file: %w", err)
	}
	defer file.Close()

	cfg := &ClientConfig{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse client JSON configuration: %w", err)
	}

	SetClientDefaults(cfg)
	ApplyClientEnvOverrides(cfg)

	if err := ValidateClientConfig(cfg); err != nil {
		return nil, fmt.Errorf("client configuration validation failed: %w", err)
	}

	return cfg, nil
}

// SaveClientConfig serializes and writes the client configuration to a JSON file.
func SaveClientConfig(filepath string, cfg *ClientConfig) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create client config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode client configuration: %w", err)
	}

	return nil
}

// SetServerDefaults populates empty fields in ServerConfig with sensible fallback values.
func SetServerDefaults(cfg *ServerConfig) {
	if cfg.ControlAddr == "" {
		cfg.ControlAddr = ":7000"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.DynamicPortMin <= 0 {
		cfg.DynamicPortMin = 40000
	}
	if cfg.DynamicPortMax <= 0 {
		cfg.DynamicPortMax = 50000
	}
}

// SetClientDefaults populates empty fields in ClientConfig with sensible fallback values.
func SetClientDefaults(cfg *ClientConfig) {
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = "127.0.0.1:7000"
	}
	if cfg.LocalTarget == "" {
		cfg.LocalTarget = "127.0.0.1:8080"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
}

// ValidateServerConfig checks server configuration parameters for logical and security errors.
func ValidateServerConfig(cfg *ServerConfig) error {
	if cfg.Token == "" {
		return errors.New("security token cannot be empty in server configuration")
	}
	if cfg.ControlAddr == "" {
		return errors.New("control address is required")
	}
	if cfg.DynamicPortMin > cfg.DynamicPortMax {
		return errors.New("dynamic_port_min cannot be greater than dynamic_port_max")
	}
	return nil
}

// ValidateClientConfig checks client configuration parameters for consistency.
func ValidateClientConfig(cfg *ClientConfig) error {
	if cfg.ClientID == "" {
		return errors.New("client_id cannot be empty")
	}
	if cfg.Token == "" {
		return errors.New("authentication token cannot be empty")
	}
	if cfg.ServerAddr == "" {
		return errors.New("server address destination is required")
	}
	return nil
}

// ApplyServerEnvOverrides checks and overrides server settings using environment variables if present.
func ApplyServerEnvOverrides(cfg *ServerConfig) {
	if val := os.Getenv("TUNNEL_SERVER_ADDR"); val != "" {
		cfg.ControlAddr = val
	}
	if val := os.Getenv("TUNNEL_TOKEN"); val != "" {
		cfg.Token = val
	}
	if val := os.Getenv("TUNNEL_DASHBOARD_ADDR"); val != "" {
		cfg.DashboardAddr = val
	}
	if val := os.Getenv("TUNNEL_OBFUSCATION"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cfg.EnableObfuscation = parsed
		}
	}
	if val := os.Getenv("TUNNEL_WEBHOOK_URL"); val != "" {
		cfg.WebhookURL = val
	}
}

// ApplyClientEnvOverrides checks and overrides client settings using environment variables if present.
func ApplyClientEnvOverrides(cfg *ClientConfig) {
	if val := os.Getenv("TUNNEL_SERVER_TARGET"); val != "" {
		cfg.ServerAddr = val
	}
	if val := os.Getenv("TUNNEL_CLIENT_ID"); val != "" {
		cfg.ClientID = val
	}
	if val := os.Getenv("TUNNEL_TOKEN"); val != "" {
		cfg.Token = val
	}
	if val := os.Getenv("TUNNEL_LOCAL_TARGET"); val != "" {
		cfg.LocalTarget = val
	}
	if val := os.Getenv("TUNNEL_OBFUSCATION"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cfg.EnableObfuscation = parsed
		}
	}
}

// SummaryString returns a clean string overview of the server settings.
func (cfg *ServerConfig) SummaryString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ControlAddr: %s\n", cfg.ControlAddr))
	sb.WriteString(fmt.Sprintf("DashboardAddr: %s\n", cfg.DashboardAddr))
	sb.WriteString(fmt.Sprintf("Obfuscation: %t\n", cfg.EnableObfuscation))
	sb.WriteString(fmt.Sprintf("Dynamic Ports Range: %d-%d\n", cfg.DynamicPortMin, cfg.DynamicPortMax))
	sb.WriteString(fmt.Sprintf("Registered Clients Count: %d\n", len(cfg.Clients)))
	return sb.String()
}
