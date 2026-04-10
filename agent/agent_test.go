package agent

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAgent_New(t *testing.T) {
	ag := New("/tmp")
	if ag == nil {
		t.Error("New returned nil")
	}
	if ag.WorkDir() != "/tmp" {
		t.Errorf("Expected WorkDir /tmp, got %s", ag.WorkDir())
	}

	// Test with empty path (uses current dir)
	ag2 := New("")
	if ag2 == nil {
		t.Error("New with empty path returned nil")
	}
}

func TestAgent_WorkDir(t *testing.T) {
	ag := New(".")
	
	// Test SetWorkDir
	err := ag.SetWorkDir("/tmp")
	if err != nil {
		t.Errorf("SetWorkDir failed: %v", err)
	}
	if ag.WorkDir() != "/tmp" {
		t.Errorf("Expected WorkDir /tmp, got %s", ag.WorkDir())
	}

	// Test relative path
	err = ag.SetWorkDir("relative/path")
	if err != nil {
		t.Errorf("SetWorkDir with relative path failed: %v", err)
	}
	abs, _ := filepath.Abs("relative/path")
	if ag.WorkDir() != abs {
		t.Errorf("Expected WorkDir %s, got %s", abs, ag.WorkDir())
	}
}

func TestAgent_ReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)

	// Write file
	err := ag.WriteFile("test.txt", "hello world")
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file
	content, err := ag.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if content != "hello world" {
		t.Errorf("Expected 'hello world', got %s", content)
	}

	// Read non-existent file
	_, err = ag.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("ReadFile should fail for non-existent file")
	}

	// Write to nested path
	err = ag.WriteFile("nested/path/file.txt", "nested content")
	if err != nil {
		t.Fatalf("WriteFile to nested path failed: %v", err)
	}

	content, err = ag.ReadFile("nested/path/file.txt")
	if err != nil {
		t.Fatalf("ReadFile from nested path failed: %v", err)
	}
	if content != "nested content" {
		t.Errorf("Expected 'nested content', got %s", content)
	}
}

func TestAgent_AppendFile(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)

	// Create file first
	err := ag.WriteFile("append.txt", "line1\n")
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Append to file
	err = ag.AppendFile("append.txt", "line2\n")
	if err != nil {
		t.Fatalf("AppendFile failed: %v", err)
	}

	// Read and verify
	content, err := ag.ReadFile("append.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if content != "line1\nline2\n" {
		t.Errorf("Expected 'line1\\nline2\\n', got %s", content)
	}

	// Append to non-existent file (should fail or create)
	err = ag.AppendFile("newfile.txt", "content")
	if err != nil {
		// This is acceptable - file doesn't exist
		_ = err // explicit ignore
	}
}

func TestAgent_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)

	// Create file
	_ = ag.WriteFile("delete.txt", "content")

	// Delete file
	err := ag.DeleteFile("delete.txt")
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify deleted
	_, err = ag.ReadFile("delete.txt")
	if err == nil {
		t.Error("File should be deleted")
	}

	// Delete non-existent file
	err = ag.DeleteFile("nonexistent.txt")
	if err == nil {
		t.Error("DeleteFile should fail for non-existent file")
	}
}

func TestAgent_ListDir(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)

	// Create files
	_ = ag.WriteFile("file1.txt", "content1")
	_ = ag.WriteFile("file2.txt", "content2")
	_ = ag.Mkdir("subdir")

	// List directory
	entries, err := ag.ListDir(".")
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}

	if len(entries) < 3 {
		t.Errorf("Expected at least 3 entries, got %d", len(entries))
	}

	// List non-existent directory
	_, err = ag.ListDir("nonexistent")
	if err == nil {
		t.Error("ListDir should fail for non-existent directory")
	}
}

func TestAgent_Mkdir(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)

	// Create directory
	err := ag.Mkdir("testdir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Verify it exists
	entries, _ := ag.ListDir(".")
	for _, e := range entries {
		if e.Name() == "testdir" && e.IsDir() {
			return // Success
		}
	}
	t.Error("testdir not found or not a directory")

	// Create nested directory
	err = ag.Mkdir("nested/deep/dir")
	if err != nil {
		t.Fatalf("Mkdir nested failed: %v", err)
	}
}

func TestAgent_Execute(t *testing.T) {
	ag := New(".")

	// Simple command
	output, err := ag.Execute(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "hello" {
		t.Errorf("Expected 'hello', got %s", output)
	}

	// Command with stderr
	output, err = ag.Execute(context.Background(), "echo out && echo err >&2")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Output contains both stdout and stderr
	if output == "" {
		t.Error("Expected some output")
	}

	// Failed command
	_, err = ag.Execute(context.Background(), "exit 1")
	if err == nil {
		t.Error("Execute should fail for exit 1")
	}
}

func TestAgent_ResolvePath(t *testing.T) {
	ag := New("/home/user/workspace")

	tests := []struct {
		input    string
		expected string
	}{
		{"file.txt", "/home/user/workspace/file.txt"},
		{"./file.txt", "/home/user/workspace/file.txt"},
		{"subdir/file.txt", "/home/user/workspace/subdir/file.txt"},
		{"/absolute/path", "/home/user/workspace/absolute/path"}, // Absolute redirected to workspace
	}

	for _, tt := range tests {
		result := ag.ResolvePath(tt.input)
		if result != tt.expected {
			t.Errorf("ResolvePath(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestAgent_HTTPOperations(t *testing.T) {
	ag := New(".")

	// HTTPGet
	output, err := ag.HTTPGet(context.Background(), "https://httpbin.org/get", nil)
	if err != nil {
		t.Logf("HTTPGet failed (network may be unavailable): %v", err)
	} else {
		if output == "" {
			t.Error("HTTPGet returned empty output")
		}
	}

	// HTTPPost
	_, err = ag.HTTPPost(context.Background(), "https://httpbin.org/post", map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Logf("HTTPPost failed (network may be unavailable): %v", err)
	}
}

func TestAgent_Commands(t *testing.T) {
	ag := New(".")

	// ListCommands
	commands := ag.ListCommands()
	if len(commands) == 0 {
		t.Error("ListCommands returned empty")
	}

	// RunCommand
	_, err := ag.RunCommand(context.Background(), "run", "echo test")
	if err != nil {
		t.Errorf("RunCommand failed: %v", err)
	}

	// Unknown command
	_, err = ag.RunCommand(context.Background(), "unknown", "")
	if err == nil {
		t.Error("RunCommand should fail for unknown command")
	}
}

func TestAgent_Sandboxing(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)

	// Try to escape workspace with ..
	path := ag.ResolvePath("../../../etc/passwd")
	expectedPath := filepath.Join(tmpDir, "passwd")
	if path != expectedPath {
		t.Errorf("Path escape should be blocked: got %s, want %s", path, expectedPath)
	}

	// Try absolute path (should be redirected to workspace)
	path = ag.ResolvePath("/etc/passwd")
	expectedPath = filepath.Join(tmpDir, "etc/passwd")
	if path != expectedPath {
		t.Errorf("Absolute path should be redirected: got %s, want %s", path, expectedPath)
	}
}

func TestExecutor_AllTools(t *testing.T) {
	tmpDir := t.TempDir()
	ag := New(tmpDir)
	exec := NewExecutor(ag)
	ctx := context.Background()

	// Test all tool executions
	tests := []struct {
		name string
		call ToolCall
	}{
		{
			name: "write_file",
			call: ToolCall{Name: "write_file", Arguments: map[string]interface{}{"path": "test.txt", "content": "hello"}},
		},
		{
			name: "read_file",
			call: ToolCall{Name: "read_file", Arguments: map[string]interface{}{"path": "test.txt"}},
		},
		{
			name: "list_directory",
			call: ToolCall{Name: "list_directory", Arguments: map[string]interface{}{"path": "."}},
		},
		{
			name: "create_directory",
			call: ToolCall{Name: "create_directory", Arguments: map[string]interface{}{"path": "newdir"}},
		},
		{
			name: "run_shell",
			call: ToolCall{Name: "run_shell", Arguments: map[string]interface{}{"command": "echo test"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exec.ExecuteTool(ctx, tt.call)
			if result.Name != tt.call.Name {
				t.Errorf("Expected name %s, got %s", tt.call.Name, result.Name)
			}
			// Error is acceptable for some operations
		})
	}
}

func TestParseToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of calls
	}{
		{
			name:     "single_json",
			input:    `{"name": "write_file", "arguments": {"path": "test.txt"}}`,
			expected: 1,
		},
		{
			name:     "json_array",
			input:    `[{"name": "write_file", "arguments": {"path": "test.txt"}}, {"name": "read_file", "arguments": {"path": "test.txt"}}]`,
			expected: 2,
		},
		{
			name:     "embedded_marker",
			input:    `Some text TOOL_CALL:{"name": "write_file", "arguments": {"path": "test.txt"}} more text`,
			expected: 1,
		},
		{
			name:     "invalid_json",
			input:    `not json at all`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := ParseToolCalls(tt.input)
			if err != nil && tt.expected > 0 {
				t.Errorf("ParseToolCalls unexpected error: %v", err)
			}
			if len(calls) != tt.expected {
				t.Errorf("Expected %d calls, got %d", tt.expected, len(calls))
			}
		})
	}
}

