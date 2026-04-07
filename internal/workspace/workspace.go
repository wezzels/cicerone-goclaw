package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// Workspace represents a code workspace
type Workspace struct {
	Path    string
	Root    string
	Sandbox bool
}

// New creates a new workspace
func New(path string) (*Workspace, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		Path:    abs,
		Root:    abs,
		Sandbox: false,
	}, nil
}

// Init initializes the workspace directory structure
func (w *Workspace) Init() error {
	dirs := []string{
		w.Path,
		filepath.Join(w.Path, "src"),
		filepath.Join(w.Path, "build"),
		filepath.Join(w.Path, "logs"),
		filepath.Join(w.Path, "tmp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create .workspace marker
	marker := filepath.Join(w.Path, ".workspace")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		if err := os.WriteFile(marker, []byte("cicerone workspace\n"), 0644); err != nil {
			return err
		}
	}

	return nil
}

// WriteFile writes a file to the workspace
func (w *Workspace) WriteFile(name string, content []byte) error {
	path := filepath.Join(w.Path, name)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}

// ReadFile reads a file from the workspace
func (w *Workspace) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(w.Path, name))
}

// DeleteFile deletes a file from the workspace
func (w *Workspace) DeleteFile(name string) error {
	return os.Remove(filepath.Join(w.Path, name))
}

// ListFiles lists files in a directory
func (w *Workspace) ListFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(w.Path, dir))
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return files, nil
}

// Clean removes all files from the workspace
func (w *Workspace) Clean() error {
	entries, err := os.ReadDir(w.Path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == ".workspace" {
			continue
		}
		path := filepath.Join(w.Path, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

// Exists checks if a file exists
func (w *Workspace) Exists(name string) bool {
	_, err := os.Stat(filepath.Join(w.Path, name))
	return err == nil
}

// IsWorkspace checks if a directory is a cicerone workspace
func IsWorkspace(path string) bool {
	marker := filepath.Join(path, ".workspace")
	_, err := os.Stat(marker)
	return err == nil
}

// expandHome expands ~ to home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}