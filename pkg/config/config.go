package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

type YamuxConfig struct {
	KeepAliveInterval int `json:"keepalive_interval_sec,omitempty"` // Default: 30s
	MaxStreamWindowSize uint32 `json:"max_stream_window_size,omitempty"` // Default: 256KB
}

type ClientMapping struct {
	ClientID string `json:"client_id"`
	Ports    []int  `json:"ports"`
	UDPPorts []int  `json:"udp_ports,omitempty"`
}

type ServerConfig struct {
	ControlAddr       string          `json:"control_addr"`
	Token             string          `json:"token"`
	LogLevel          string          `json:"log_level,omitempty"` // debug, info, warn, error
	TLSCertFile       string          `json:"tls_cert_file,omitempty"`
	TLSKeyFile        string          `json:"tls_key_file,omitempty"`
	EnableAutoTLS     bool            `json:"enable_auto_tls,omitempty"`
	AutoTLSDomain     string          `json:"auto_tls_domain,omitempty"`
	AutoTLSCacheDir   string          `json:"auto_tls_cache_dir,omitempty"`
	EnableObfuscation bool            `json:"enable_obfuscation,omitempty"`
	DashboardAddr     string          `json:"dashboard_addr,omitempty"`
	WebhookURL        string          `json:"webhook_url,omitempty"`
	Clients           []ClientMapping `json:"clients"`
	Yamux             *YamuxConfig    `json:"yamux,omitempty"`
}

type ClientConfig struct {
	ServerAddr             string       `json:"server_addr"`
	ClientID               string       `json:"client_id"`
	Token                  string       `json:"token"`
	LocalAddr              string       `json:"local_addr"`
	LogLevel               string       `json:"log_level,omitempty"` // debug, info, warn, error
	TLSCertFile            string       `json:"tls_cert_file,omitempty"`
	TLSKeyFile             string       `json:"tls_key_file,omitempty"`
	TLSCAFile              string       `json:"tls_ca_file,omitempty"`
	InsecureSkipVerify     bool         `json:"insecure_skip_verify,omitempty"`
	InsecureAllowPlaintext bool         `json:"insecure_allow_plaintext,omitempty"`
	EnableObfuscation      bool         `json:"enable_obfuscation,omitempty"`
	Yamux                  *YamuxConfig `json:"yamux,omitempty"`
}

// GetYamuxConfig converts YamuxConfig struct to yamux.Config
func GetYamuxConfig(cfg *YamuxConfig) *yamux.Config {
	yamuxCfg := yamux.DefaultConfig()
	if cfg == nil {
		return yamuxCfg
	}

	if cfg.KeepAliveInterval > 0 {
		yamuxCfg.EnableKeepAlive = true
		yamuxCfg.KeepAliveInterval = time.Duration(cfg.KeepAliveInterval) * time.Second
	}
	if cfg.MaxStreamWindowSize > 0 {
		yamuxCfg.MaxStreamWindowSize = cfg.MaxStreamWindowSize
	}
	return yamuxCfg
}

// LoadServerConfig reads and parses the JSON configuration file for the server.
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read server config file: %w", err)
	}

	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse server config JSON: %w", err)
	}

	return &cfg, nil
}

// LoadClientConfig reads and parses the JSON configuration file for the client.
func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config file: %w", err)
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse client config JSON: %w", err)
	}

	return &cfg, nil
}

// SetupLogger initializes a new slog.Logger based on the log level string.
func SetupLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
