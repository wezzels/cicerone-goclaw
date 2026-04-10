// Package vm provides VM management for workspace deployment.
package vm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// VMConfigFile represents a VM configuration loaded from config file.
// This is distinct from VMConfig which is used for creating VMs.
type VMConfigFile struct {
	// Identity
	Name        string `yaml:"name" mapstructure:"name"`
	Description string `yaml:"description" mapstructure:"description"`

	// Resources
	Memory int `yaml:"memory" mapstructure:"memory"` // MB
	VCPUs  int `yaml:"vcpus" mapstructure:"vcpus"`

	// Storage
	Image    string `yaml:"image" mapstructure:"image"`       // Path to base image
	DiskSize int    `yaml:"disk_size" mapstructure:"disk_size"` // GB (for new disks)

	// Network
	Network string `yaml:"network" mapstructure:"network"` // Network name
	IP      string `yaml:"ip" mapstructure:"ip"`           // Static IP (optional)

	// SSH Access
	SSHKey string `yaml:"ssh_key" mapstructure:"ssh_key"` // Path to SSH private key
	User   string `yaml:"user" mapstructure:"user"`       // SSH user (default root)

	// Options
	AutoStart      bool `yaml:"auto_start" mapstructure:"auto_start"`
	SnapshotOnStop bool `yaml:"snapshot_on_stop" mapstructure:"snapshot_on_stop"`
}

// DeployConfig represents the deploy section of the config.
type DeployConfig struct {
	DefaultVM      string `yaml:"default_vm" mapstructure:"default_vm"`
	AutoStart      bool   `yaml:"auto_start" mapstructure:"auto_start"`
	SnapshotOnStop bool   `yaml:"snapshot_on_stop" mapstructure:"snapshot_on_stop"`
}

// Config represents VM configuration from file.
type Config struct {
	VMs    map[string]*VMConfigFile `yaml:"vms" mapstructure:"vms"`
	Deploy DeployConfig             `yaml:"deploy" mapstructure:"deploy"`
}

// LoadConfig loads VM configuration from viper.
func LoadConfig() (*Config, error) {
	var cfg Config

	// Check if vms key exists
	if !viper.IsSet("vms") {
		return &cfg, nil
	}

	// Unmarshal vms section
	if err := viper.UnmarshalKey("vms", &cfg.VMs); err != nil {
		return nil, fmt.Errorf("failed to parse vms config: %w", err)
	}

	// Unmarshal deploy section
	if viper.IsSet("deploy") {
		if err := viper.UnmarshalKey("deploy", &cfg.Deploy); err != nil {
			return nil, fmt.Errorf("failed to parse deploy config: %w", err)
		}
	}

	return &cfg, nil
}

// GetVM returns a VM configuration by name.
func (c *Config) GetVM(name string) (*VMConfigFile, error) {
	vm, ok := c.VMs[name]
	if !ok {
		return nil, fmt.Errorf("VM '%s' not found in config", name)
	}
	return vm, nil
}

// ListVMs returns all configured VMs.
func (c *Config) ListVMs() map[string]*VMConfigFile {
	return c.VMs
}

// ToVMConfig converts VMConfigFile to VMConfig for Manager operations.
func (v *VMConfigFile) ToVMConfig() *VMConfig {
	return &VMConfig{
		Name:           v.Name,
		Description:    v.Description,
		Memory:        v.Memory,
		VCPUs:         v.VCPUs,
		Image:         v.Image,
		DiskSize:      v.DiskSize,
		Network:       v.Network,
		IP:            v.IP,
		SSHKey:        v.SSHKey,
		User:          v.User,
		AutoStart:     v.AutoStart,
		SnapshotOnStop: v.SnapshotOnStop,
	}
}

// GetSSHKeyPath returns the SSH key path, expanding ~ if needed.
func (v *VMConfigFile) GetSSHKeyPath() (string, error) {
	if v.SSHKey == "" {
		// Default to standard key location
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".ssh", "id_ed25519"), nil
	}

	return ExpandPath(v.SSHKey)
}

// GetUser returns the SSH user, defaulting to root.
func (v *VMConfigFile) GetUser() string {
	if v.User == "" {
		return "root"
	}
	return v.User
}

// Validate validates the VM configuration.
func (v *VMConfigFile) Validate() error {
	if v.Name == "" {
		return fmt.Errorf("VM name is required")
	}
	if v.Image == "" {
		return fmt.Errorf("VM '%s': image path is required", v.Name)
	}
	if v.Memory > 0 && v.Memory < 256 {
		return fmt.Errorf("VM '%s': memory must be at least 256MB", v.Name)
	}
	if v.VCPUs > 0 && v.VCPUs < 1 {
		return fmt.Errorf("VM '%s': vcpus must be at least 1", v.Name)
	}
	return nil
}

// DefaultVMConfig returns default VM configuration values.
func DefaultVMConfig() *VMConfig {
	return &VMConfig{
		Memory:        2048, // 2GB
		VCPUs:         2,
		Network:       "default",
		User:          "root",
		AutoStart:     false,
		SnapshotOnStop: false,
		Headless:      true,
	}
}

// MergeWithDefaults merges the VMConfigFile with default values.
func (v *VMConfigFile) MergeWithDefaults() *VMConfig {
	defaults := DefaultVMConfig()
	cfg := v.ToVMConfig()

	// Apply defaults where not set
	if cfg.Memory == 0 {
		cfg.Memory = defaults.Memory
	}
	if cfg.VCPUs == 0 {
		cfg.VCPUs = defaults.VCPUs
	}
	if cfg.Network == "" {
		cfg.Network = defaults.Network
	}
	if cfg.User == "" {
		cfg.User = defaults.User
	}

	return cfg
}

// ExpandPath expands ~ to home directory.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(path) == 1 {
			return home, nil
		}
		return filepath.Join(home, path[1:]), nil
	}

	return path, nil
}

// GetCiceroneDir returns the cicerone config directory.
func GetCiceroneDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cicerone"), nil
}

// GetVMKeysDir returns the directory for VM-specific SSH keys.
func GetVMKeysDir() (string, error) {
	ciceroneDir, err := GetCiceroneDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(ciceroneDir, "keys"), nil
}

// GetVMKeyPath returns the path to a VM-specific SSH key.
func GetVMKeyPath(vmName string) (string, error) {
	keysDir, err := GetVMKeysDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(keysDir, fmt.Sprintf("id_ed25519_%s", vmName)), nil
}

// EnsureKeyDir ensures the key directory exists.
func EnsureKeyDir() error {
	keysDir, err := GetVMKeysDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(keysDir, 0700)
}