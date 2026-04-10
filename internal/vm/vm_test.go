package vm

import (
	"testing"
)

// TestVMConfig_Validate tests VM config validation
func TestVMConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *VMConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &VMConfig{
				Name:   "test-vm",
				Image:  "/path/to/image.qcow2",
				Memory: 2048,
				VCPUs:  2,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &VMConfig{
				Image:  "/path/to/image.qcow2",
				Memory: 2048,
				VCPUs:  2,
			},
			wantErr: true,
		},
		{
			name: "missing image",
			config: &VMConfig{
				Name:   "test-vm",
				Memory: 2048,
				VCPUs:  2,
			},
			wantErr: true,
		},
		{
			name: "memory too low",
			config: &VMConfig{
				Name:   "test-vm",
				Image:  "/path/to/image.qcow2",
				Memory: 128,
				VCPUs:  2,
			},
			wantErr: true,
		},
		{
			name: "vcpus too low",
			config: &VMConfig{
				Name:   "test-vm",
				Image:  "/path/to/image.qcow2",
				Memory: 2048,
				VCPUs:  0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("VMConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestVMConfig_Clone tests config cloning
func TestVMConfig_Clone(t *testing.T) {
	original := &VMConfig{
		Name:           "test-vm",
		Description:    "Test VM",
		Memory:         2048,
		VCPUs:          2,
		Image:          "/path/to/image.qcow2",
		Network:        "default",
		SSHKey:         "/path/to/key",
		AutoStart:      true,
		SnapshotOnStop: false,
	}

	cloned := original.Clone("cloned-vm")

	if cloned.Name != "cloned-vm" {
		t.Errorf("Expected cloned name 'cloned-vm', got %s", cloned.Name)
	}
	if cloned.Description != original.Description {
		t.Error("Description should be copied")
	}
	if cloned.Memory != original.Memory {
		t.Error("Memory should be copied")
	}
	if cloned.IP != "" {
		t.Error("IP should be empty in clone")
	}
	if cloned.MAC != "" {
		t.Error("MAC should be empty in clone")
	}
}

// TestDefaultVMConfig tests default configuration
func TestDefaultVMConfig(t *testing.T) {
	defaults := DefaultVMConfig()

	if defaults.Memory != 2048 {
		t.Errorf("Expected default memory 2048, got %d", defaults.Memory)
	}
	if defaults.VCPUs != 2 {
		t.Errorf("Expected default vcpus 2, got %d", defaults.VCPUs)
	}
	if defaults.Network != "default" {
		t.Errorf("Expected default network 'default', got %s", defaults.Network)
	}
	if defaults.User != "root" {
		t.Errorf("Expected default user 'root', got %s", defaults.User)
	}
}

// TestVMState tests VM state constants
func TestVMState(t *testing.T) {
	states := map[VMState]string{
		StateUnknown:  "unknown",
		StateRunning:  "running",
		StateStopped:  "stopped",
		StatePaused:   "paused",
		StateCrashed:  "crashed",
		StateCreating: "creating",
		StateDeleting: "deleting",
	}

	for state, expected := range states {
		if string(state) != expected {
			t.Errorf("Expected state %s, got %s", expected, state)
		}
	}
}

// TestVMConfigFile_MergeWithDefaults tests config merging
func TestVMConfigFile_MergeWithDefaults(t *testing.T) {
	// Minimal config
	minimal := &VMConfigFile{
		Name:  "test-vm",
		Image: "/path/to/image.qcow2",
	}

	merged := minimal.MergeWithDefaults()

	if merged.Memory != 2048 {
		t.Errorf("Expected default memory 2048, got %d", merged.Memory)
	}
	if merged.VCPUs != 2 {
		t.Errorf("Expected default vcpus 2, got %d", merged.VCPUs)
	}
	if merged.Network != "default" {
		t.Errorf("Expected default network 'default', got %s", merged.Network)
	}
	if merged.User != "root" {
		t.Errorf("Expected default user 'root', got %s", merged.User)
	}

	// Config with overrides
	withOverrides := &VMConfigFile{
		Name:    "test-vm",
		Image:   "/path/to/image.qcow2",
		Memory:  4096,
		VCPUs:   4,
		Network: "custom",
	}

	mergedOverride := withOverrides.MergeWithDefaults()

	if mergedOverride.Memory != 4096 {
		t.Errorf("Expected memory 4096, got %d", mergedOverride.Memory)
	}
	if mergedOverride.VCPUs != 4 {
		t.Errorf("Expected vcpus 4, got %d", mergedOverride.VCPUs)
	}
	if mergedOverride.Network != "custom" {
		t.Errorf("Expected network 'custom', got %s", mergedOverride.Network)
	}
}

// TestVMConfigFile_GetUser tests user default
func TestVMConfigFile_GetUser(t *testing.T) {
	// Without user set
	cfg := &VMConfigFile{Name: "test"}
	if cfg.GetUser() != "root" {
		t.Errorf("Expected default user 'root', got %s", cfg.GetUser())
	}

	// With user set
	cfg.User = "ubuntu"
	if cfg.GetUser() != "ubuntu" {
		t.Errorf("Expected user 'ubuntu', got %s", cfg.GetUser())
	}
}

// TestVMError tests error types
func TestVMError(t *testing.T) {
	err := &VMError{Op: "create", Err: ErrNotFound}

	// The error wraps ErrNotFound which is itself a VMError
	expected := "vm create: vm : vm not found"
	if err.Error() != expected {
		t.Errorf("Unexpected error message: got %s, want %s", err.Error(), expected)
	}

	if err.Unwrap() != ErrNotFound {
		t.Error("Unwrap should return ErrNotFound")
	}
}