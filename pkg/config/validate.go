package config

import (
"errors"
"fmt"
)

// ServerConfig structure matching configuration requirements
type ServerConfig struct {
ControlAddr string
EnableTLS   bool
CertFile    string
KeyFile     string
CAFile      string
Clients     []ClientMapping
}

type ClientMapping struct {
ClientID string
Ports    []int
}

// ClientConfig structure for client side
type ClientConfig struct {
ServerAddr    string
ClientID      string
Token         string
EnableTLS     bool
CAFile        string
CertFile      string
KeyFile       string
TLSSkipVerify bool
}

// Validate checks server configuration parameters for correctness
func (c *ServerConfig) Validate() error {
if c.ControlAddr == "" {
return errors.New("server control address cannot be empty")
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

// Validate checks client configuration parameters for correctness
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
// Just a warning or soft check, but valid
}

return nil
}
