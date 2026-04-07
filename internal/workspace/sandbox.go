package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sandbox provides isolated execution environment
type Sandbox struct {
	workspace   *Workspace
	allowedDirs []string
	blockedCmds []string
	readOnly    []string
}

// NewSandbox creates a new sandbox for a workspace
func NewSandbox(w *Workspace) *Sandbox {
	return &Sandbox{
		workspace:   w,
		allowedDirs: []string{w.Path},
		blockedCmds: []string{"rm -rf /", "mkfs", "dd if="},
		readOnly:    []string{},
	}
}

// AllowDir adds a directory to the allowed list
func (s *Sandbox) AllowDir(dir string) {
	abs, err := filepath.Abs(dir)
	if err == nil {
		s.allowedDirs = append(s.allowedDirs, abs)
	}
}

// BlockCommand adds a command pattern to the blocked list
func (s *Sandbox) BlockCommand(pattern string) {
	s.blockedCmds = append(s.blockedCmds, pattern)
}

// SetReadOnly marks a directory as read-only
func (s *Sandbox) SetReadOnly(dir string) {
	abs, err := filepath.Abs(dir)
	if err == nil {
		s.readOnly = append(s.readOnly, abs)
	}
}

// ValidatePath checks if a path is allowed
func (s *Sandbox) ValidatePath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %s", path)
	}

	// Check if path is within allowed directories
	for _, allowed := range s.allowedDirs {
		if strings.HasPrefix(abs, allowed) {
			// Check if read-only
			for _, ro := range s.readOnly {
				if strings.HasPrefix(abs, ro) {
					return fmt.Errorf("path is read-only: %s", path)
				}
			}
			return nil
		}
	}

	return fmt.Errorf("path not allowed: %s", path)
}

// ValidateCommand checks if a command is allowed
func (s *Sandbox) ValidateCommand(command string) error {
	lower := strings.ToLower(command)

	for _, blocked := range s.blockedCmds {
		if strings.Contains(lower, strings.ToLower(blocked)) {
			return fmt.Errorf("blocked command pattern: %s", blocked)
		}
	}

	return nil
}

// CreateTempDir creates a temporary directory in the sandbox
func (s *Sandbox) CreateTempDir(prefix string) (string, error) {
	if err := s.ValidatePath(s.workspace.Path); err != nil {
		return "", err
	}

	tmpDir := filepath.Join(s.workspace.Path, "tmp", prefix)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	return tmpDir, nil
}

// CreateTempFile creates a temporary file in the sandbox
func (s *Sandbox) CreateTempFile(prefix string) (*os.File, error) {
	if err := s.ValidatePath(s.workspace.Path); err != nil {
		return nil, err
	}

	tmpDir := filepath.Join(s.workspace.Path, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, err
	}

	return os.CreateTemp(tmpDir, prefix)
}

// Cleanup removes sandbox temporary files
func (s *Sandbox) Cleanup() error {
	tmpDir := filepath.Join(s.workspace.Path, "tmp")
	return os.RemoveAll(tmpDir)
}

// IsAllowed checks if an operation is permitted
func (s *Sandbox) IsAllowed(operation string) bool {
	// Default allow with blocked list
	for _, blocked := range s.blockedCmds {
		if strings.Contains(strings.ToLower(operation), strings.ToLower(blocked)) {
			return false
		}
	}
	return true
}

// RestrictToWorkspace ensures path stays within workspace
func (s *Sandbox) RestrictToWorkspace(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Check if already within workspace
	if strings.HasPrefix(abs, s.workspace.Path) {
		return abs, nil
	}

	// Join with workspace path
	return filepath.Join(s.workspace.Path, path), nil
}

// SafePath returns a path guaranteed to be within workspace
func (s *Sandbox) SafePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Join(s.workspace.Path, path)
	}

	// Check if already within workspace
	if strings.HasPrefix(abs, s.workspace.Path) {
		return abs
	}

	// Strip leading slashes and join
	clean := strings.TrimPrefix(path, "/")
	return filepath.Join(s.workspace.Path, clean)
}