package ssh

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 22 {
		t.Errorf("Default port = %d, want 22", cfg.Port)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.KeyPath != "~/.ssh/id_rsa" {
		t.Errorf("Default key path = %s, want ~/.ssh/id_rsa", cfg.KeyPath)
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		addr     string
		host     string
		port     int
		hasError bool
	}{
		{"localhost", "localhost", 22, false},
		{"localhost:22", "localhost", 22, false},
		{"10.0.0.1:2222", "10.0.0.1", 2222, false},
		{"example.com:8080", "example.com", 8080, false},
	}

	for _, tt := range tests {
		host, port, err := ParseAddress(tt.addr)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseAddress(%q) expected error", tt.addr)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAddress(%q) error = %v", tt.addr, err)
			continue
		}
		if host != tt.host {
			t.Errorf("ParseAddress(%q) host = %s, want %s", tt.addr, host, tt.host)
		}
		if port != tt.port {
			t.Errorf("ParseAddress(%q) port = %d, want %d", tt.addr, port, tt.port)
		}
	}
}

func TestExpandPath(t *testing.T) {
	// Test with ~ prefix
	expanded, err := ExpandPath("~/test")
	if err != nil {
		t.Errorf("ExpandPath error = %v", err)
	}
	if expanded == "~/test" {
		t.Error("ExpandPath did not expand ~")
	}

	// Test without ~ prefix
	expanded, err = ExpandPath("/absolute/path")
	if err != nil {
		t.Errorf("ExpandPath error = %v", err)
	}
	if expanded != "/absolute/path" {
		t.Errorf("ExpandPath = %s, want /absolute/path", expanded)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Host:    "localhost",
				Port:    22,
				User:    "test",
				KeyPath: "~/.ssh/id_rsa",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: &Config{
				Port:    22,
				User:    "test",
				KeyPath: "~/.ssh/id_rsa",
			},
			wantErr: true,
		},
		{
			name: "missing user",
			config: &Config{
				Host:    "localhost",
				Port:    22,
				KeyPath: "~/.ssh/id_rsa",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Host:    "localhost",
				Port:    -1,
				User:    "test",
				KeyPath: "~/.ssh/id_rsa",
			},
			wantErr: true,
		},
		{
			name: "port too high",
			config: &Config{
				Host:    "localhost",
				Port:    65536,
				User:    "test",
				KeyPath: "~/.ssh/id_rsa",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v", tt.wantErr)
			}
		})
	}
}

func TestConfigAddress(t *testing.T) {
	cfg := &Config{
		Host: "localhost",
		Port: 2222,
	}

	addr := cfg.Address()
	if addr != "localhost:2222" {
		t.Errorf("Address() = %s, want localhost:2222", addr)
	}
}

func TestConfigString(t *testing.T) {
	cfg := &Config{
		Host: "example.com",
		Port: 22,
		User: "testuser",
	}

	str := cfg.String()
	if str != "testuser@example.com:22" {
		t.Errorf("String() = %s, want testuser@example.com:22", str)
	}
}

// Integration tests require actual SSH server
// These are skipped in normal test runs

func TestClientIntegration(t *testing.T) {
	// Skip if no test server
	t.Skip("Integration test requires SSH server")

	cfg := &Config{
		Host:    "localhost",
		Port:    22,
		User:    "test",
		KeyPath: "~/.ssh/id_rsa",
		Timeout: 10 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	// Test exec
	ctx := context.Background()
	stdout, stderr, err := client.Exec(ctx, "echo hello")
	if err != nil {
		t.Errorf("Exec error = %v", err)
	}
	if len(stdout) == 0 {
		t.Error("Exec stdout is empty")
	}
	if len(stderr) > 0 {
		t.Errorf("Exec stderr = %s", stderr)
	}

	// Test close
	if err := client.Close(); err != nil {
		t.Errorf("Close error = %v", err)
	}
}

func TestTunnelManager(t *testing.T) {
	// Create a tunnel manager with nil client (for testing)
	tm := NewTunnelManager(nil)

	if tm == nil {
		t.Fatal("NewTunnelManager returned nil")
	}

	if tm.tunnels == nil {
		t.Error("tunnels map not initialized")
	}

	// List should be empty
	tunnels := tm.List()
	if len(tunnels) != 0 {
		t.Errorf("Expected 0 tunnels, got %d", len(tunnels))
	}
}

func TestTunnelStartStop(t *testing.T) {
	// This test requires actual network
	t.Skip("Tunnel test requires actual network")

	// Create a tunnel (without starting)
	tunnel := &Tunnel{
		LocalPort:  18080,
		RemoteHost: "localhost",
		RemotePort: 80,
	}

	if tunnel.IsActive() {
		t.Error("Tunnel should not be active initially")
	}

	// IsActive after stop
	if err := tunnel.Stop(); err != nil {
		t.Errorf("Stop error = %v", err)
	}
}