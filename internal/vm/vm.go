// Package vm provides VM management for workspace deployment.
package vm

import (
	"context"
	"time"
)

// VMState represents the current state of a VM.
type VMState string

const (
	StateUnknown   VMState = "unknown"
	StateRunning   VMState = "running"
	StateStopped   VMState = "stopped"
	StatePaused    VMState = "paused"
	StateCrashed   VMState = "crashed"
	StateCreating  VMState = "creating"
	StateDeleting  VMState = "deleting"
)

// VMConfig represents VM configuration.
type VMConfig struct {
	// Identity
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`

	// Resources
	Memory int `yaml:"memory" json:"memory"` // MB
	VCPUs  int `yaml:"vcpus" json:"vcpus"`

	// Storage
	Image     string `yaml:"image" json:"image"`         // Path to base image
	DiskSize  int    `yaml:"disk_size" json:"disk_size"`  // GB (for new disks)
	CloudInit string `yaml:"cloud_init" json:"cloud_init"` // Path to cloud-init config

	// Network
	Network string `yaml:"network" json:"network"` // Network name (default, nat, etc.)
	IP      string `yaml:"ip" json:"ip"`           // Static IP (optional)
	MAC     string `yaml:"mac" json:"mac"`         // MAC address (optional)

	// SSH Access
	SSHKey  string `yaml:"ssh_key" json:"ssh_key"`   // Path to SSH private key
	SSHPort int    `yaml:"ssh_port" json:"ssh_port"` // SSH port (default 22)
	User    string `yaml:"user" json:"user"`         // SSH user (default root)

	// Workspace
	WorkspacePath string `yaml:"workspace_path" json:"workspace_path"` // Path on VM

	// Options
	AutoStart       bool `yaml:"auto_start" json:"auto_start"`
	SnapshotOnStop  bool `yaml:"snapshot_on_stop" json:"snapshot_on_stop"`
	Headless        bool `yaml:"headless" json:"headless"`
}

// VMInfo contains runtime information about a VM.
type VMInfo struct {
	Name      string    `json:"name"`
	State     VMState   `json:"state"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Memory    int       `json:"memory"`
	VCPUs     int       `json:"vcpus"`
	DiskUsage int64     `json:"disk_usage"` // bytes
	Uptime    time.Duration `json:"uptime"`
	CreatedAt time.Time `json:"created_at"`
}

// SnapshotInfo contains snapshot information.
type SnapshotInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Size        int64     `json:"size"` // bytes
	Current     bool      `json:"current"`
}

// Manager provides VM management operations.
type Manager interface {
	// Lifecycle
	Create(ctx context.Context, cfg *VMConfig) (*VMInfo, error)
	Delete(ctx context.Context, name string) error
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string, force bool) error
	Restart(ctx context.Context, name string) error

	// Status
	Status(ctx context.Context, name string) (*VMInfo, error)
	List(ctx context.Context) ([]*VMInfo, error)
	Exists(ctx context.Context, name string) (bool, error)

	// Snapshots
	Snapshot(ctx context.Context, name, snapshotName string, description string) error
	SnapshotList(ctx context.Context, name string) ([]SnapshotInfo, error)
	SnapshotRevert(ctx context.Context, name, snapshotName string) error
	SnapshotDelete(ctx context.Context, name, snapshotName string) error

	// Connection
	Shell(ctx context.Context, name string) error
	Exec(ctx context.Context, name string, command string) (stdout, stderr []byte, err error)
	ExecInteractive(ctx context.Context, name string, command string) error

	// File Transfer
	Push(ctx context.Context, name string, localPath, remotePath string) error
	Pull(ctx context.Context, name string, remotePath, localPath string) error

	// SSH Keys
	DeployKey(ctx context.Context, name string, publicKey []byte) error
	GenerateKeys(ctx context.Context, name string) (privateKey, publicKey []byte, err error)

	// Network
	GetIP(ctx context.Context, name string) (string, error)

	// Resource Management
	SetMemory(ctx context.Context, name string, memoryMB int) error
	SetVCPUs(ctx context.Context, name string, vcpus int) error
	GetConsole(ctx context.Context, name string) (string, error)
}

// ManagerOptions are options for creating a VM manager.
type ManagerOptions struct {
	// Libvirt connection URI (e.g., qemu:///system, qemu+ssh://host/system)
	URI string `json:"uri"`

	// Storage pool for VM images
	StoragePool string `json:"storage_pool"`

	// Network to use for new VMs
	DefaultNetwork string `json:"default_network"`

	// Timeout for operations
	Timeout time.Duration `json:"timeout"`

	// Dry run mode (don't actually perform operations)
	DryRun bool `json:"dry_run"`
}

// DefaultManagerOptions returns default manager options.
func DefaultManagerOptions() *ManagerOptions {
	return &ManagerOptions{
		URI:            "qemu:///system",
		StoragePool:    "default",
		DefaultNetwork: "default",
		Timeout:        5 * time.Minute,
		DryRun:         false,
	}
}

// Clone creates a copy of the VM config with a new name.
func (c *VMConfig) Clone(name string) *VMConfig {
	return &VMConfig{
		Name:           name,
		Description:    c.Description,
		Memory:        c.Memory,
		VCPUs:         c.VCPUs,
		Image:         c.Image,
		DiskSize:      c.DiskSize,
		CloudInit:     c.CloudInit,
		Network:       c.Network,
		IP:            "", // New VM gets new IP
		MAC:           "", // New VM gets new MAC
		SSHKey:        c.SSHKey,
		SSHPort:       c.SSHPort,
		User:          c.User,
		WorkspacePath: c.WorkspacePath,
		AutoStart:     c.AutoStart,
		SnapshotOnStop: c.SnapshotOnStop,
		Headless:      c.Headless,
	}
}

// Validate checks if the VM config is valid.
func (c *VMConfig) Validate() error {
	if c.Name == "" {
		return &VMError{Op: "validate", Err: ErrInvalidName}
	}
	if c.Memory < 256 {
		return &VMError{Op: "validate", Err: ErrInvalidMemory}
	}
	if c.VCPUs < 1 {
		return &VMError{Op: "validate", Err: ErrInvalidVCPUs}
	}
	if c.Image == "" {
		return &VMError{Op: "validate", Err: ErrInvalidImage}
	}
	return nil
}

// VMError represents a VM operation error.
type VMError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

func (e *VMError) Error() string {
	return "vm " + e.Op + ": " + e.Err.Error()
}

func (e *VMError) Unwrap() error {
	return e.Err
}

// Common errors
var (
	ErrNotFound      = &VMError{Err: &vmError{"vm not found"}}
	ErrAlreadyExists = &VMError{Err: &vmError{"vm already exists"}}
	ErrNotRunning    = &VMError{Err: &vmError{"vm not running"}}
	ErrRunning       = &VMError{Err: &vmError{"vm is running"}}
	ErrInvalidName   = &VMError{Err: &vmError{"invalid vm name"}}
	ErrInvalidMemory = &VMError{Err: &vmError{"invalid memory (minimum 256MB)"}}
	ErrInvalidVCPUs  = &VMError{Err: &vmError{"invalid vcpus (minimum 1)"}}
	ErrInvalidImage  = &VMError{Err: &vmError{"invalid image path"}}
	ErrConnection    = &VMError{Err: &vmError{"connection failed"}}
	ErrSSHKey        = &VMError{Err: &vmError{"ssh key error"}}
	ErrSnapshot      = &VMError{Err: &vmError{"snapshot error"}}
)

type vmError struct {
	msg string
}

func (e *vmError) Error() string {
	return e.msg
}