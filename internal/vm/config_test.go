package vm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestLoadConfig tests config loading
func TestLoadConfig(t *testing.T) {
	// Reset viper
	viper.Reset()

	// Test empty config
	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if cfg == nil {
		t.Error("LoadConfig() should not return nil")
	}

	// Test with VMs config
	viper.Set("vms.dev.name", "dev-vm")
	viper.Set("vms.dev.memory", 4096)
	viper.Set("vms.dev.vcpus", 4)

	cfg, err = LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}

	if cfg.VMs == nil {
		t.Error("VMs should not be nil")
	}

	if cfg.VMs["dev"] == nil {
		t.Error("dev VM config should not be nil")
		return
	}

	if cfg.VMs["dev"].Name != "dev-vm" {
		t.Errorf("Expected VM name 'dev-vm', got %s", cfg.VMs["dev"].Name)
	}

	viper.Reset()
}

// TestConfig_GetVM tests VM retrieval
func TestConfig_GetVM(t *testing.T) {
	cfg := &Config{
		VMs: map[string]*VMConfigFile{
			"dev": {
				Name:   "dev-vm",
				Memory: 2048,
			},
		},
	}

	// Existing VM
	vm, err := cfg.GetVM("dev")
	if err != nil {
		t.Errorf("GetVM() error = %v", err)
	}
	if vm.Name != "dev-vm" {
		t.Errorf("Expected name 'dev-vm', got %s", vm.Name)
	}

	// Non-existing VM
	_, err = cfg.GetVM("nonexistent")
	if err == nil {
		t.Error("GetVM() should return error for nonexistent VM")
	}
}

// TestConfig_ListVMs tests VM listing
func TestConfig_ListVMs(t *testing.T) {
	cfg := &Config{
		VMs: map[string]*VMConfigFile{
			"dev":  {Name: "dev-vm"},
			"prod": {Name: "prod-vm"},
		},
	}

	vms := cfg.ListVMs()
	if len(vms) != 2 {
		t.Errorf("Expected 2 VMs, got %d", len(vms))
	}
}

// TestExpandPath tests path expansion
func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"", ""},
	}

	for _, tt := range tests {
		result, err := ExpandPath(tt.input)
		if err != nil {
			t.Errorf("ExpandPath(%s) error = %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("ExpandPath(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// TestGetCiceroneDir tests cicerone directory path
func TestGetCiceroneDir(t *testing.T) {
	dir, err := GetCiceroneDir()
	if err != nil {
		t.Errorf("GetCiceroneDir() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cicerone")

	if dir != expected {
		t.Errorf("GetCiceroneDir() = %s, want %s", dir, expected)
	}
}

// TestGetVMKeysDir tests keys directory path
func TestGetVMKeysDir(t *testing.T) {
	dir, err := GetVMKeysDir()
	if err != nil {
		t.Errorf("GetVMKeysDir() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cicerone", "keys")

	if dir != expected {
		t.Errorf("GetVMKeysDir() = %s, want %s", dir, expected)
	}
}

// TestGetVMKeyPath tests VM key path generation
func TestGetVMKeyPath(t *testing.T) {
	path, err := GetVMKeyPath("test-vm")
	if err != nil {
		t.Errorf("GetVMKeyPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cicerone", "keys", "id_ed25519_test-vm")

	if path != expected {
		t.Errorf("GetVMKeyPath() = %s, want %s", path, expected)
	}
}

// TestVMConfigFile_ToVMConfig tests config conversion
func TestVMConfigFile_ToVMConfig(t *testing.T) {
	file := &VMConfigFile{
		Name:        "test-vm",
		Description: "Test VM",
		Memory:      4096,
		VCPUs:       4,
		Image:       "/path/to/image.qcow2",
		DiskSize:    20,
		Network:     "default",
		IP:          "192.168.122.100",
		SSHKey:      "/path/to/key",
		User:        "root",
		AutoStart:   true,
	}

	cfg := file.ToVMConfig()

	if cfg.Name != "test-vm" {
		t.Errorf("Expected name 'test-vm', got %s", cfg.Name)
	}
	if cfg.Memory != 4096 {
		t.Errorf("Expected memory 4096, got %d", cfg.Memory)
	}
	if cfg.VCPUs != 4 {
		t.Errorf("Expected vcpus 4, got %d", cfg.VCPUs)
	}
	if cfg.Network != "default" {
		t.Errorf("Expected network 'default', got %s", cfg.Network)
	}
}

// TestDeployConfig tests deploy config
func TestDeployConfig(t *testing.T) {
	cfg := &Config{
		Deploy: DeployConfig{
			DefaultVM:      "dev",
			AutoStart:      true,
			SnapshotOnStop: false,
		},
	}

	if cfg.Deploy.DefaultVM != "dev" {
		t.Errorf("Expected DefaultVM 'dev', got %s", cfg.Deploy.DefaultVM)
	}
	if !cfg.Deploy.AutoStart {
		t.Error("AutoStart should be true")
	}
}