package config_test

import (
	"testing"

	"github.com/sepantartd/go-reverse-tunnel/pkg/config"
)

func TestServerConfig_PortRanges(t *testing.T) {
	cfg := config.ServerConfig{
		ControlAddr:           "127.0.0.1:8080",
		Token:                 "secret",
		InsecureAllowPlaintext: true,
		Clients: []config.ClientMapping{
			{
				ClientID: "client1",
				Ports:    []int{8001},
				Ranges:   []string{"9000-9002"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected valid config, got error: %v", err)
	}

	expectedPorts := []int{8001, 9000, 9001, 9002}
	client := cfg.Clients[0]
	if len(client.Ports) != len(expectedPorts) {
		t.Fatalf("Expected %d ports, got %d", len(expectedPorts), len(client.Ports))
	}

	for i, p := range expectedPorts {
		if client.Ports[i] != p {
			t.Errorf("Expected port %d at index %d, got %d", p, i, client.Ports[i])
		}
	}
}

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
			name: "Invalid Port Range",
			cfg: config.ServerConfig{
				ControlAddr:           "127.0.0.1:8080",
				Token:                 "secret",
				InsecureAllowPlaintext: true,
				Clients: []config.ClientMapping{
					{ClientID: "client1", Ranges: []string{"9005-9000"}},
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
