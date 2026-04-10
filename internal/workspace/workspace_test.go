package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)

	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if ws == nil {
		t.Fatal("New returned nil")
	}
	if ws.Path != tmpDir {
		t.Errorf("Path = %s, want %s", ws.Path, tmpDir)
	}
}

func TestWorkspace_Init(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = ws.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check that .workspace marker was created
	marker := filepath.Join(tmpDir, ".workspace")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error(".workspace marker not created")
	}

	// Init on existing workspace should succeed
	err = ws.Init()
	if err != nil {
		t.Errorf("Init on existing workspace failed: %v", err)
	}
}

func TestWorkspace_WriteReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()

	// Write file
	err = ws.WriteFile("test.txt", []byte("hello world"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file
	content, err := ws.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("ReadFile content = %s, want hello world", string(content))
	}

	// Write to nested path
	err = ws.WriteFile("nested/path/file.txt", []byte("nested content"))
	if err != nil {
		t.Fatalf("WriteFile to nested path failed: %v", err)
	}

	content, err = ws.ReadFile("nested/path/file.txt")
	if err != nil {
		t.Fatalf("ReadFile from nested path failed: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("ReadFile content = %s, want nested content", string(content))
	}
}

func TestWorkspace_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()

	// Create file
	_ = ws.WriteFile("delete.txt", []byte("content"))

	// Delete file
	err = ws.DeleteFile("delete.txt")
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify deleted
	if ws.Exists("delete.txt") {
		t.Error("File should be deleted")
	}

	// Delete non-existent file
	err = ws.DeleteFile("nonexistent.txt")
	if err == nil {
		t.Error("DeleteFile should fail for non-existent file")
	}
}

func TestWorkspace_ListFiles(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()

	// Create files
	_ = ws.WriteFile("file1.txt", []byte("1"))
	_ = ws.WriteFile("file2.txt", []byte("2"))

	// List files
	files, err := ws.ListFiles(".")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("ListFiles returned %d files, want at least 2", len(files))
	}
}

func TestWorkspace_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()

	_ = ws.WriteFile("exists.txt", []byte("content"))

	if !ws.Exists("exists.txt") {
		t.Error("Exists should return true for existing file")
	}

	if ws.Exists("nonexistent.txt") {
		t.Error("Exists should return false for non-existent file")
	}
}

func TestWorkspace_Clean(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()

	// Create files
	_ = ws.WriteFile("file1.txt", []byte("1"))
	_ = ws.WriteFile("file2.txt", []byte("2"))

	// Clean workspace
	err = ws.Clean()
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Verify files are gone
	files, _ := ws.ListFiles(".")
	for _, f := range files {
		if f != ".workspace" {
			t.Errorf("Unexpected file after clean: %s", f)
		}
	}
}

func TestIsWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a workspace initially
	if IsWorkspace(tmpDir) {
		t.Error("IsWorkspace should return false for non-workspace")
	}

	// Create workspace
	ws, _ := New(tmpDir)
	_ = ws.Init()

	// Now it is a workspace
	if !IsWorkspace(tmpDir) {
		t.Error("IsWorkspace should return true for workspace")
	}
}

func TestSandbox(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()
	sandbox := NewSandbox(ws)

	if sandbox == nil {
		t.Fatal("NewSandbox returned nil")
	}

	// Test AllowDir
	sandbox.AllowDir("/allowed")

	// Test BlockCommand
	if sandbox.IsAllowed("rm -rf /") {
		t.Error("IsAllowed should return false for blocked command")
	}

	// Test that safe commands are allowed
	if !sandbox.IsAllowed("ls -la") {
		t.Error("IsAllowed should return true for safe command")
	}
}

func TestSandbox_ValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()
	sandbox := NewSandbox(ws)

	// Path within workspace should be valid
	err = sandbox.ValidatePath(filepath.Join(tmpDir, "file.txt"))
	if err != nil {
		t.Errorf("ValidatePath failed for valid path: %v", err)
	}

	// Path outside workspace should be invalid
	err = sandbox.ValidatePath("/etc/passwd")
	if err == nil {
		t.Error("ValidatePath should fail for path outside workspace")
	}
}

func TestSandbox_SafePath(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	_ = ws.Init()
	sandbox := NewSandbox(ws)

	tests := []struct {
		input    string
		wantSafe bool
	}{
		{"file.txt", true},
		{"subdir/file.txt", true},
		{"/absolute/path", true}, // Will be redirected to workspace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res := sandbox.SafePath(tt.input)
			// Path should be within workspace
			if tt.wantSafe {
				if !strings.HasPrefix(res, tmpDir) {
					t.Errorf("SafePath(%s) = %s, should be within %s", tt.input, res, tmpDir)
				}
			}
		})
	}
}

func TestExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	exec := NewExecutor(ws)

	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	// Test SetTimeout
	exec.SetTimeout(30 * time.Second)

	// Test SetEnv
	exec.SetEnv([]string{"TEST=value"})

	// Test Run
	output, err := exec.Run("echo", "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("Run output = %s, want hello", string(output))
	}

	// Test Run with invalid command
	_, err = exec.Run("nonexistent_command_xyz")
	if err == nil {
		t.Error("Run should fail for nonexistent command")
	}
}

func TestExecutor_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	exec := NewExecutor(ws)
	exec.SetTimeout(100 * time.Millisecond)

	_, err = exec.Run("sleep", "5")
	if err == nil {
		t.Error("Run should timeout")
	}
}

func TestExecutor_WorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	exec := NewExecutor(ws)

	// Execute pwd to verify working directory
	output, err := exec.Run("pwd")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Output should be tmpDir
	result := string(output)
	if !strings.HasPrefix(result, tmpDir) {
		t.Errorf("pwd returned %s, should start with %s", result, tmpDir)
	}
}