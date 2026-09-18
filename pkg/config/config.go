package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

type ClientMapping struct {
	ClientID string `json:"client_id"`
	Ports    []int  `json:"ports"`
	UDPPorts []int  `json:"udp_ports"`
}

type YamuxConfig struct {
	KeepaliveIntervalSec int `json:"keepalive_interval_sec"`
	MaxStreamWindowSize  int `json:"max_stream_window_size"`
}

type ServerConfig struct {
	ControlAddr       string          `json:"control_addr"`
	Token             string          `json:"token"`
	LogLevel          string          `json:"log_level"`
	EnableObfuscation bool            `json:"enable_obfuscation"`
	DashboardAddr     string          `json:"dashboard_addr"`
	WebhookURL        string          `json:"webhook_url"`
	TLSCertFile       string          `json:"tls_cert_file"`
	TLSKeyFile        string          `json:"tls_key_file"`
	EnableAutoTLS     bool            `json:"enable_auto_tls"`
	AutoTLSDomain     string          `json:"auto_tls_domain"`
	AutoTLSCacheDir   string          `json:"auto_tls_cache_dir"`
	Yamux             YamuxConfig     `json:"yamux"`
	Clients           []ClientMapping `json:"clients"`
}

func (s *ServerConfig) Validate() error {
	if strings.TrimSpace(s.ControlAddr) == "" {
		return errors.New("server control_addr cannot be empty")
	}
	if err := validateAddr(s.ControlAddr); err != nil {
		return fmt.Errorf("invalid control_addr: %w", err)
	}

	if strings.TrimSpace(s.Token) == "" {
		return errors.New("server token cannot be empty")
	}

	if s.DashboardAddr != "" {
		if err := validateAddr(s.DashboardAddr); err != nil {
			return fmt.Errorf("invalid dashboard_addr: %w", err)
		}
	}

	if s.EnableAutoTLS && strings.TrimSpace(s.AutoTLSDomain) == "" {
		return errors.New("auto_tls_domain must be specified when enable_auto_tls is true")
	}

	for _, client := range s.Clients {
		if strings.TrimSpace(client.ClientID) == "" {
			return errors.New("client_id in clients mapping cannot be empty")
		}
		for _, port := range client.Ports {
			if err := validatePort(port); err != nil {
				return fmt.Errorf("invalid port %d for client %s: %w", port, client.ClientID, err)
			}
		}
		for _, uport := range client.UDPPorts {
			if err := validatePort(uport); err != nil {
				return fmt.Errorf("invalid udp_port %d for client %s: %w", uport, client.ClientID, err)
			}
		}
	}

	return nil
}

type ClientConfig struct {
	ServerAddr             string      `json:"server_addr"`
	ClientID               string      `json:"client_id"`
	Token                  string      `json:"token"`
	LocalAddr              string      `json:"local_addr"`
	LogLevel               string      `json:"log_level"`
	EnableObfuscation      bool        `json:"enable_obfuscation"`
	TLSCertFile            string      `json:"tls_cert_file"`
	TLSKeyFile             string      `json:"tls_key_file"`
	TLSCAFile              string      `json:"tls_ca_file"`
	InsecureSkipVerify     bool        `json:"insecure_skip_verify"`
	InsecureAllowPlaintext bool        `json:"insecure_allow_plaintext"`
	Yamux                  YamuxConfig `json:"yamux"`
}

func (c *ClientConfig) Validate() error {
	if strings.TrimSpace(c.ServerAddr) == "" {
		return errors.New("client server_addr cannot be empty")
	}
	if err := validateAddr(c.ServerAddr); err != nil {
		return fmt.Errorf("invalid server_addr: %w", err)
	}

	if strings.TrimSpace(c.ClientID) == "" {
		return errors.New("client client_id cannot be empty")
	}

	if strings.TrimSpace(c.Token) == "" {
		return errors.New("client token cannot be empty")
	}

	if strings.TrimSpace(c.LocalAddr) == "" {
		return errors.New("client local_addr cannot be empty")
	}

	return nil
}

func validateAddr(addr string) error {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port format: %w", err)
	}
	return validatePort(port)
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("server configuration validation error: %w", err)
	}

	return &cfg, nil
}

func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("client configuration validation error: %w", err)
	}

	return &cfg, nil
}

func GetYamuxConfig(cfg YamuxConfig) *yamux.Config {
	yCfg := yamux.DefaultConfig()

	if cfg.KeepaliveIntervalSec > 0 {
		yCfg.EnableKeepAlive = true
		yCfg.KeepAliveInterval = time.Duration(cfg.KeepaliveIntervalSec) * time.Second
	}

	if cfg.MaxStreamWindowSize > 0 {
		yCfg.MaxStreamWindowSize = uint32(cfg.MaxStreamWindowSize)
	}

	return yCfg
}

func SetupLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler)
}
