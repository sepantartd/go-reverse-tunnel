package config

import (
"encoding/json"
"os"
)

type PublicBind struct {
Port int    `json:"port"`
Name string `json:"name"`
}

type ServerConfig struct {
ControlAddr    string       `json:"control_addr"`
WebPort        int          `json:"web_port"`
Token          string       `json:"token"`
Cert           string       `json:"cert"`
Key            string       `json:"key"`
ClientCA       string       `json:"client_ca,omitempty"`
EnableTLS      bool         `json:"enable_tls"`
EnableCompress bool         `json:"enable_compress"`
PublicBinds    []PublicBind `json:"public_binds"`
}

type Forward struct {
RemotePort int    `json:"remote_port"`
Local      string `json:"local"`
Proto      string `json:"proto,omitempty"` // "tcp" or "udp"
}

type ClientConfig struct {
Server         string    `json:"server"`
Token          string    `json:"token"`
ClientID       string    `json:"client_id"`
EnableTLS      bool      `json:"enable_tls"`
TLSSkipVerify  bool      `json:"tls_skip_verify"`
ServerCA       string    `json:"server_ca,omitempty"`
ClientCert     string    `json:"client_cert,omitempty"`
ClientKey      string    `json:"client_key,omitempty"`
EnableCompress bool      `json:"enable_compress"`
Socks5Addr     string    `json:"socks5_addr,omitempty"` // e.g. "127.0.0.1:1080"
Forwards       []Forward `json:"forwards"`
}

func LoadServerConfig(path string) (*ServerConfig, error) {
file, err := os.Open(path)
if err != nil {
return nil, err
}
defer file.Close()

var cfg ServerConfig
if err := json.NewDecoder(file).Decode(&cfg); err != nil {
return nil, err
}
return &cfg, nil
}

func LoadClientConfig(path string) (*ClientConfig, error) {
file, err := os.Open(path)
if err != nil {
return nil, err
}
defer file.Close()

var cfg ClientConfig
if err := json.NewDecoder(file).Decode(&cfg); err != nil {
return nil, err
}
return &cfg, nil
}
