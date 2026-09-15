package config

import "testing"

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		wantErr bool
	}{
		{
			name: "Valid Server Config",
			cfg: ServerConfig{
				ControlAddr: "0.0.0.0:8080",
				Token:       "secret",
				Clients: []ClientMapping{
					{ClientID: "client1", Ports: []int{8081, 8082}},
					{ClientID: "client2", Ports: []int{9090}},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid Control Address",
			cfg: ServerConfig{
				ControlAddr: "invalid-address",
				Token:       "secret",
			},
			wantErr: true,
		},
		{
			name: "Port Collision Between Clients",
			cfg: ServerConfig{
				ControlAddr: ":8080",
				Token:       "secret",
				Clients: []ClientMapping{
					{ClientID: "client1", Ports: []int{8081}},
					{ClientID: "client2", Ports: []int{8081}},
				},
			},
			wantErr: true,
		},
		{
			name: "Duplicate ClientID",
			cfg: ServerConfig{
				ControlAddr: ":8080",
				Token:       "secret",
				Clients: []ClientMapping{
					{ClientID: "client1", Ports: []int{8081}},
					{ClientID: "client1", Ports: []int{8082}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
	}{
		{
			name: "Valid Client Config",
			cfg: ClientConfig{
				ServerAddr: "127.0.0.1:8080",
				ClientID:   "client1",
				Token:      "secret",
			},
			wantErr: false,
		},
		{
			name: "Invalid Server Address",
			cfg: ClientConfig{
				ServerAddr: "127.0.0.1",
				ClientID:   "client1",
				Token:      "secret",
			},
			wantErr: true,
		},
		{
			name: "TLS Missing KeyFile",
			cfg: ClientConfig{
				ServerAddr: "127.0.0.1:8080",
				ClientID:   "client1",
				Token:      "secret",
				EnableTLS:  true,
				CertFile:   "cert.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
