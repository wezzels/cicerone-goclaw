package ssh

import (
	"testing"
	"time"
)

func TestClient_New(t *testing.T) {
	// Test with invalid config (should fail)
	_, err := NewClient(&Config{Host: "", Port: 22})
	if err == nil {
		t.Error("NewClient should fail with empty host")
	}

	// Test with missing key file
	_, err = NewClient(&Config{
		Host:    "localhost",
		Port:    22,
		User:    "test",
		KeyPath: "/nonexistent/key",
		Timeout: 1 * time.Second,
	})
	if err == nil {
		t.Error("NewClient should fail with nonexistent key file")
	}
}

func TestClient_IsConnected(t *testing.T) {
	// Test with nil client
	client := &Client{client: nil}
	if client.IsConnected() {
		t.Error("IsConnected should return false for nil client")
	}
}

func TestTunnel(t *testing.T) {
	tunnel := &Tunnel{
		LocalPort:  18080,
		RemoteHost: "localhost",
		RemotePort: 80,
	}

	if tunnel.IsActive() {
		t.Error("Tunnel should not be active initially")
	}

	// Stop should be idempotent
	if err := tunnel.Stop(); err != nil {
		t.Errorf("Stop on inactive tunnel should succeed: %v", err)
	}

	// Test String method
	if tunnel.String() == "" {
		t.Error("Tunnel.String() should not be empty")
	}

	// Test LocalAddr
	if tunnel.LocalAddr() != "127.0.0.1:18080" {
		t.Errorf("LocalAddr() = %s, want 127.0.0.1:18080", tunnel.LocalAddr())
	}

	// Test RemoteAddr
	if tunnel.RemoteAddr() != "localhost:80" {
		t.Errorf("RemoteAddr() = %s, want localhost:80", tunnel.RemoteAddr())
	}
}

func TestTunnelManager_CreateTunnel(t *testing.T) {
	// Create with nil client
	tm := NewTunnelManager(nil)

	// CreateTunnel should work without client (client needed for Start)
	tunnel, err := tm.CreateTunnel(18080, "localhost", 80)
	if err != nil {
		t.Errorf("CreateTunnel should succeed with nil client: %v", err)
	}
	if tunnel == nil {
		t.Error("CreateTunnel returned nil tunnel")
	}

	// Create duplicate should fail
	_, err = tm.CreateTunnel(18080, "localhost", 80)
	if err == nil {
		t.Error("CreateTunnel should fail for duplicate port")
	}

	// Test List
	tunnels := tm.List()
	if len(tunnels) != 1 {
		t.Errorf("List returned %d tunnels, want 1", len(tunnels))
	}

	// Test Get
	got, err := tm.Get(18080)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if got != tunnel {
		t.Error("Get returned different tunnel")
	}

	// Test Get non-existent
	_, err = tm.Get(99999)
	if err == nil {
		t.Error("Get should fail for non-existent port")
	}
}

func TestTunnelManager_StopAll(t *testing.T) {
	tm := NewTunnelManager(nil)

	// StopAll on empty should succeed
	err := tm.StopAll()
	if err != nil {
		t.Errorf("StopAll on empty failed: %v", err)
	}

	// Create tunnels
	_, _ = tm.CreateTunnel(18080, "localhost", 80)
	_, _ = tm.CreateTunnel(18081, "localhost", 81)

	// StopAll should succeed
	err = tm.StopAll()
	if err != nil {
		t.Errorf("StopAll failed: %v", err)
	}

	// Verify empty
	if len(tm.List()) != 0 {
		t.Error("StopAll should clear tunnels")
	}
}

func TestConfig_Address(t *testing.T) {
	cfg := &Config{
		Host: "192.168.1.1",
		Port: 2222,
	}

	expected := "192.168.1.1:2222"
	if cfg.Address() != expected {
		t.Errorf("Address() = %v, want %v", cfg.Address(), expected)
	}
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{
		Host: "example.com",
		Port: 22,
		User: "testuser",
	}

	expected := "testuser@example.com:22"
	if cfg.String() != expected {
		t.Errorf("String() = %v, want %v", cfg.String(), expected)
	}
}

func TestHostAliasConversion(t *testing.T) {
	cfg := &Config{
		Name:    "test",
		Host:    "example.com",
		Port:    22,
		User:    "testuser",
		KeyPath: "~/.ssh/id_rsa",
		Timeout: 30 * time.Second,
	}

	// Config to HostAlias
	alias := HostAliasFromConfig(cfg)
	if alias.Name != cfg.Name {
		t.Errorf("HostAliasFromConfig Name = %s, want %s", alias.Name, cfg.Name)
	}
	if alias.Host != cfg.Host {
		t.Errorf("HostAliasFromConfig Host = %s, want %s", alias.Host, cfg.Host)
	}

	// HostAlias to Config
	cfg2 := ConfigFromHostAlias(alias)
	if cfg2.Name != alias.Name {
		t.Errorf("ConfigFromHostAlias Name = %s, want %s", cfg2.Name, alias.Name)
	}
	if cfg2.Host != alias.Host {
		t.Errorf("ConfigFromHostAlias Host = %s, want %s", cfg2.Host, alias.Host)
	}
}

func TestClient_ExecWithTimeout(t *testing.T) {
	// Test with nil client (should fail gracefully)
	client := &Client{}
	_, err := client.ExecWithTimeout("echo test", 1*time.Second)
	if err == nil {
		t.Error("ExecWithTimeout should fail with disconnected client")
	}
}