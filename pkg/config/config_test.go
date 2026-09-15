package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	content := `{"control_addr":"0.0.0.0:7001","token":"abc"}`
	tmp, _ := os.CreateTemp("", "cfg-*.json")
	defer os.Remove(tmp.Name())
	tmp.Write([]byte(content))
	tmp.Close()

	cfg, err := LoadServerConfig(tmp.Name())
	if err != nil || cfg.Token != "abc" {
		t.Fatal("Failed to parse config correctly")
	}
}
