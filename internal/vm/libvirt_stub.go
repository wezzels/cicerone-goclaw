//go:build !libvirt

package vm

import (
	"context"
	"errors"
)

// ErrLibvirtNotAvailable is returned when libvirt is not compiled in.
var ErrLibvirtNotAvailable = errors.New("libvirt support not compiled in (rebuild with -tags libvirt)")

// LibvirtManager is a stub when libvirt is not available.
type LibvirtManager struct{}

// NewLibvirtManager returns an error when libvirt is not compiled in.
func NewLibvirtManager(opts *ManagerOptions) (*LibvirtManager, error) {
	return nil, ErrLibvirtNotAvailable
}

// Close does nothing.
func (m *LibvirtManager) Close() error { return nil }

// Create returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Create(ctx context.Context, cfg *VMConfig) (*VMInfo, error) {
	return nil, ErrLibvirtNotAvailable
}

// Delete returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Delete(ctx context.Context, name string) error {
	return ErrLibvirtNotAvailable
}

// Start returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Start(ctx context.Context, name string) error {
	return ErrLibvirtNotAvailable
}

// Stop returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Stop(ctx context.Context, name string, force bool) error {
	return ErrLibvirtNotAvailable
}

// Restart returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Restart(ctx context.Context, name string) error {
	return ErrLibvirtNotAvailable
}

// Status returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Status(ctx context.Context, name string) (*VMInfo, error) {
	return nil, ErrLibvirtNotAvailable
}

// List returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) List(ctx context.Context) ([]*VMInfo, error) {
	return nil, ErrLibvirtNotAvailable
}

// Exists returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Exists(ctx context.Context, name string) (bool, error) {
	return false, ErrLibvirtNotAvailable
}

// Snapshot returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Snapshot(ctx context.Context, name, snapshotName, description string) error {
	return ErrLibvirtNotAvailable
}

// SnapshotList returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) SnapshotList(ctx context.Context, name string) ([]SnapshotInfo, error) {
	return nil, ErrLibvirtNotAvailable
}

// SnapshotRevert returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) SnapshotRevert(ctx context.Context, name, snapshotName string) error {
	return ErrLibvirtNotAvailable
}

// SnapshotDelete returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) SnapshotDelete(ctx context.Context, name, snapshotName string) error {
	return ErrLibvirtNotAvailable
}

// GetIP returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) GetIP(ctx context.Context, name string) (string, error) {
	return "", ErrLibvirtNotAvailable
}

// Shell returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Shell(ctx context.Context, name string) error {
	return ErrLibvirtNotAvailable
}

// Exec returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Exec(ctx context.Context, name string, command string) (stdout, stderr []byte, err error) {
	return nil, nil, ErrLibvirtNotAvailable
}

// ExecInteractive returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) ExecInteractive(ctx context.Context, name string, command string) error {
	return ErrLibvirtNotAvailable
}

// Push returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Push(ctx context.Context, name string, localPath, remotePath string) error {
	return ErrLibvirtNotAvailable
}

// Pull returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) Pull(ctx context.Context, name string, remotePath, localPath string) error {
	return ErrLibvirtNotAvailable
}

// DeployKey returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) DeployKey(ctx context.Context, name string, publicKey []byte) error {
	return ErrLibvirtNotAvailable
}

// GenerateKeys returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) GenerateKeys(ctx context.Context, name string) (privateKey, publicKey []byte, err error) {
	return nil, nil, ErrLibvirtNotAvailable
}

// SetMemory returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) SetMemory(ctx context.Context, name string, memoryMB int) error {
	return ErrLibvirtNotAvailable
}

// SetVCPUs returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) SetVCPUs(ctx context.Context, name string, vcpus int) error {
	return ErrLibvirtNotAvailable
}

// GetConsole returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) GetConsole(ctx context.Context, name string) (string, error) {
	return "", ErrLibvirtNotAvailable
}

// CloneVM returns ErrLibvirtNotAvailable.
func (m *LibvirtManager) CloneVM(ctx context.Context, sourceName, destName string, cfg *VMConfig) (*VMInfo, error) {
	return nil, ErrLibvirtNotAvailable
}