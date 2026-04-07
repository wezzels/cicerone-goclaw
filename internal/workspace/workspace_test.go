package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.Path != tmpDir {
		t.Errorf("Path = %s, want %s", w.Path, tmpDir)
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Check directories exist
	dirs := []string{"src", "build", "logs", "tmp"}
	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("directory %s not created", dir)
		}
	}

	// Check .workspace marker
	marker := filepath.Join(tmpDir, ".workspace")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Errorf(".workspace marker not created")
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	content := []byte("test content")
	if err := w.WriteFile("test.txt", content); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Check file exists
	path := filepath.Join(tmpDir, "test.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	if string(data) != "test content" {
		t.Errorf("content = %s, want 'test content'", data)
	}
}

func TestWriteFileNested(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	content := []byte("nested")
	if err := w.WriteFile("subdir/nested.txt", content); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Check file exists
	path := filepath.Join(tmpDir, "subdir", "nested.txt")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("nested file not created")
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	// Write a file
	content := []byte("read test")
	w.WriteFile("readme.txt", content)

	// Read it back
	data, err := w.ReadFile("readme.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != "read test" {
		t.Errorf("content = %s, want 'read test'", data)
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	w.WriteFile("todelete.txt", []byte("delete me"))

	if err := w.DeleteFile("todelete.txt"); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	// Check file is gone
	path := filepath.Join(tmpDir, "todelete.txt")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file not deleted")
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	if w.Exists("nonexistent.txt") {
		t.Error("Exists() = true for nonexistent file")
	}

	w.WriteFile("exists.txt", []byte("content"))
	if !w.Exists("exists.txt") {
		t.Error("Exists() = false for existing file")
	}
}

func TestClean(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	// Create some files
	w.WriteFile("test1.txt", []byte("1"))
	w.WriteFile("test2.txt", []byte("2"))

	if err := w.Clean(); err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// Check files are gone
	entries, _ := os.ReadDir(tmpDir)
	// Only .workspace should remain
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after clean, got %d", len(entries))
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	w.WriteFile("file1.txt", []byte("1"))
	w.WriteFile("file2.txt", []byte("2"))

	files, err := w.ListFiles(".")
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	if len(files) < 2 {
		t.Errorf("expected at least 2 files, got %d", len(files))
	}
}

func TestIsWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a workspace initially
	if IsWorkspace(tmpDir) {
		t.Error("IsWorkspace() = true for non-workspace")
	}

	// Create workspace
	w, _ := New(tmpDir)
	w.Init()

	// Now it should be
	if !IsWorkspace(tmpDir) {
		t.Error("IsWorkspace() = false for workspace")
	}
}

func TestNewExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)

	exec := NewExecutor(w)
	if exec == nil {
		t.Fatal("NewExecutor() returned nil")
	}

	if exec.Workdir() != tmpDir {
		t.Errorf("Workdir() = %s, want %s", exec.Workdir(), tmpDir)
	}
}

func TestExecutorExists(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	exec := NewExecutor(w)

	// 'echo' should exist on all systems
	if !exec.Exists("echo") {
		t.Error("Exists(echo) = false")
	}
}

func TestExecutorWhich(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	exec := NewExecutor(w)

	// 'echo' should exist
	path, err := exec.Which("echo")
	if err != nil {
		t.Fatalf("Which(echo) error = %v", err)
	}

	if path == "" {
		t.Error("Which(echo) returned empty string")
	}
}

func TestExecutorRun(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()
	exec := NewExecutor(w)
	exec.SetTimeout(5000 * time.Millisecond) // 5 seconds

	output, err := exec.Run("echo", "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if string(output) != "hello\n" {
		t.Errorf("output = %q, want 'hello\\n'", output)
	}
}

func TestExecutorRunShell(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()
	exec := NewExecutor(w)

	output, err := exec.RunShell("echo hello")
	if err != nil {
		t.Fatalf("RunShell() error = %v", err)
	}

	if string(output) != "hello\n" {
		t.Errorf("output = %q, want 'hello\\n'", output)
	}
}

func TestSandboxNew(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)

	s := NewSandbox(w)
	if s == nil {
		t.Fatal("NewSandbox() returned nil")
	}
}

func TestSandboxValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	w.Init()

	s := NewSandbox(w)

	// Path within workspace should be allowed
	if err := s.ValidatePath(filepath.Join(tmpDir, "test.txt")); err != nil {
		t.Errorf("ValidatePath() error = %v", err)
	}

	// Path outside workspace should fail
	if err := s.ValidatePath("/etc/passwd"); err == nil {
		t.Error("ValidatePath() should fail for path outside workspace")
	}
}

func TestSandboxValidateCommand(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	s := NewSandbox(w)

	// Normal command should be allowed
	if err := s.ValidateCommand("ls -la"); err != nil {
		t.Errorf("ValidateCommand() error = %v", err)
	}

	// Dangerous command should be blocked
	if err := s.ValidateCommand("rm -rf /"); err == nil {
		t.Error("ValidateCommand() should block dangerous command")
	}
}

func TestSandboxSafePath(t *testing.T) {
	tmpDir := t.TempDir()
	w, _ := New(tmpDir)
	s := NewSandbox(w)

	// Path with leading slash should be sanitized
	safe := s.SafePath("/etc/passwd")
	if safe == "/etc/passwd" {
		t.Error("SafePath() should sanitize path")
	}

	// Should be within workspace
	if safe != filepath.Join(tmpDir, "etc/passwd") {
		t.Errorf("SafePath() = %s, want %s", safe, filepath.Join(tmpDir, "etc/passwd"))
	}
}