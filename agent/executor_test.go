package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutor_WriteFile(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "executor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ag := New(tmpDir)
	exec := NewExecutor(ag)

	// Test write file
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name: "write_file",
		Arguments: map[string]interface{}{
			"path":    "test.txt",
			"content": "Hello, World!",
		},
	})

	if !result.Success {
		t.Errorf("Write file failed: %s", result.Error)
	}

	// Verify file exists
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("File content mismatch: got %q, want %q", string(content), "Hello, World!")
	}
}

func TestExecutor_ReadFile(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "executor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ag := New(tmpDir)
	exec := NewExecutor(ag)

	// Test read file
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name: "read_file",
		Arguments: map[string]interface{}{
			"path": "test.txt",
		},
	})

	if !result.Success {
		t.Errorf("Read file failed: %s", result.Error)
	}
	if result.Output != "Test content" {
		t.Errorf("Content mismatch: got %q, want %q", result.Output, "Test content")
	}
}

func TestExecutor_ListDirectory(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "executor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	ag := New(tmpDir)
	exec := NewExecutor(ag)

	// Test list directory
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name:      "list_directory",
		Arguments: map[string]interface{}{},
	})

	if !result.Success {
		t.Errorf("List directory failed: %s", result.Error)
	}
	// Output should contain file names
	if !containsAll(result.Output, "file1.txt", "file2.txt") {
		t.Errorf("List output missing files: %s", result.Output)
	}
}

func TestExecutor_RunShell(t *testing.T) {
	ag := New(".")
	exec := NewExecutor(ag)

	// Test shell command
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name: "run_shell",
		Arguments: map[string]interface{}{
			"command": "echo 'shell test'",
		},
	})

	if !result.Success {
		t.Errorf("Shell command failed: %s", result.Error)
	}
	if !containsStr(result.Output, "shell test") {
		t.Errorf("Shell output missing expected text: %s", result.Output)
	}
}

func TestExecutor_CreateDirectory(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "executor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ag := New(tmpDir)
	exec := NewExecutor(ag)

	// Test create directory
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name: "create_directory",
		Arguments: map[string]interface{}{
			"path": "subdir/nested",
		},
	})

	if !result.Success {
		t.Errorf("Create directory failed: %s", result.Error)
	}

	// Verify directory exists
	if _, err := os.Stat(filepath.Join(tmpDir, "subdir", "nested")); os.IsNotExist(err) {
		t.Errorf("Directory not created")
	}
}

func TestExecutor_DeleteFile(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "executor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "delete_me.txt")
	if err := os.WriteFile(testFile, []byte("delete me"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ag := New(tmpDir)
	exec := NewExecutor(ag)

	// Test delete file
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name: "delete_file",
		Arguments: map[string]interface{}{
			"path": "delete_me.txt",
		},
	})

	if !result.Success {
		t.Errorf("Delete file failed: %s", result.Error)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("File still exists after delete")
	}
}

func TestExecutor_UnknownTool(t *testing.T) {
	ag := New(".")
	exec := NewExecutor(ag)

	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name: "unknown_tool",
		Arguments: map[string]interface{}{},
	})

	if result.Success {
		t.Errorf("Unknown tool should fail")
	}
	if result.Error == nil {
		t.Errorf("Expected error for unknown tool")
	}
}

func TestExecutor_MissingArguments(t *testing.T) {
	ag := New(".")
	exec := NewExecutor(ag)

	// Test write without content
	result := exec.ExecuteTool(context.Background(), ToolCall{
		Name:      "write_file",
		Arguments: map[string]interface{}{"path": "test.txt"},
	})

	if result.Success {
		t.Errorf("Write without content should fail")
	}
}

// Helper functions

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !containsAt(s, substr) {
			return false
		}
	}
	return true
}