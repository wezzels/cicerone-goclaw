package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config represents SSH connection configuration
type Config struct {
	Name      string        `yaml:"name" json:"name"`
	Host      string        `yaml:"host" json:"host"`
	Port      int           `yaml:"port" json:"port"`
	User      string        `yaml:"user" json:"user"`
	KeyPath   string        `yaml:"key_path" json:"key_path"`
	Password  string        `yaml:"password,omitempty" json:"password,omitempty"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
	KeepAlive time.Duration `yaml:"keepalive" json:"keepalive"`
}

// HostAlias represents a saved SSH host configuration
type HostAlias struct {
	Name        string            `yaml:"name" json:"name"`
	Host        string            `json:"host"`
	Port        int               `yaml:"port" json:"port"`
	User        string            `yaml:"user" json:"user"`
	KeyPath     string            `yaml:"key_path" json:"key_path"`
	Description string            `yaml:"description" json:"description"`
	Extra       map[string]string `yaml:"extra,omitempty" json:"extra,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Port:      22,
		Timeout:   30 * time.Second,
		KeepAlive: 10 * time.Second,
		KeyPath:   "~/.ssh/id_rsa",
	}
}

// ParseAddress parses an address string into host and port
func ParseAddress(addr string) (host string, port int, err error) {
	parts := strings.Split(addr, ":")
	if len(parts) == 1 {
		return parts[0], 22, nil
	}
	if len(parts) == 2 {
		var p int
		if _, err := fmt.Sscanf(parts[1], "%d", &p); err != nil {
			return "", 0, fmt.Errorf("invalid port: %s", parts[1])
		}
		return parts[0], p, nil
	}
	return "", 0, fmt.Errorf("invalid address format: %s", addr)
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	return nil
}

// Address returns the full address string
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// String returns a human-readable description
func (c *Config) String() string {
	return fmt.Sprintf("%s@%s:%d", c.User, c.Host, c.Port)
}

// HostAliasFromConfig creates a HostAlias from a Config
func HostAliasFromConfig(cfg *Config) *HostAlias {
	return &HostAlias{
		Name:    cfg.Name,
		Host:    cfg.Host,
		Port:    cfg.Port,
		User:    cfg.User,
		KeyPath: cfg.KeyPath,
	}
}

// ConfigFromHostAlias creates a Config from a HostAlias
func ConfigFromHostAlias(h *HostAlias) *Config {
	return &Config{
		Name:    h.Name,
		Host:    h.Host,
		Port:    h.Port,
		User:    h.User,
		KeyPath: h.KeyPath,
		Timeout: 30 * time.Second,
	}
}