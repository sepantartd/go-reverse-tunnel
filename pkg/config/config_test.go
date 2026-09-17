package config_test

import (
	"testing"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ServerConfig
		wantErr bool
	}{
		{
			name: "Valid Server Config",
			cfg: config.ServerConfig{
				ControlAddr:           "127.0.0.1:8080",
				Token:                 "secret",
				InsecureAllowPlaintext: true,
				Clients: []config.ClientMapping{
					{ClientID: "client1", Ports: []int{9001, 9002}},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing TLS And Insecure Flag",
			cfg: config.ServerConfig{
				ControlAddr: "127.0.0.1:8080",
				Token:       "secret",
			},
			wantErr: true,
		},
		{
			name: "Invalid Control Address",
			cfg: config.ServerConfig{
				ControlAddr:           "invalid-addr",
				Token:                 "secret",
				InsecureAllowPlaintext: true,
			},
			wantErr: true,
		},
		{
			name: "Port Collision Between Clients",
			cfg: config.ServerConfig{
				ControlAddr:           "127.0.0.1:8080",
				Token:                 "secret",
				InsecureAllowPlaintext: true,
				Clients: []config.ClientMapping{
					{ClientID: "client1", Ports: []int{8000}},
					{ClientID: "client2", Ports: []int{8000}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ServerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ClientConfig
		wantErr bool
	}{
		{
			name: "Valid Client Config",
			cfg: config.ClientConfig{
				ServerAddr:            "127.0.0.1:8080",
				LocalAddr:             "127.0.0.1:3000",
				ClientID:              "client1",
				Token:                 "secret",
				InsecureAllowPlaintext: true,
			},
			wantErr: false,
		},
		{
			name: "Missing TLS and Plaintext Flag",
			cfg: config.ClientConfig{
				ServerAddr: "127.0.0.1:8080",
				LocalAddr:  "127.0.0.1:3000",
				ClientID:   "client1",
				Token:      "secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
