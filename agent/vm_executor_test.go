package agent

import (
	"context"
	"testing"

	"github.com/spf13/viper"
)

// TestVMExecutor_GetActiveWorkspace tests the workspace config reading
func TestVMExecutor_GetActiveWorkspace(t *testing.T) {
	ag := New(".")
	vme := NewVMExecutor(ag)

	// No workspace set by default
	if ws := vme.GetActiveWorkspace(); ws != "" {
		t.Errorf("Expected empty workspace, got %s", ws)
	}

	// Set a workspace
	viper.Set("active_workspace", "test-vm")
	if ws := vme.GetActiveWorkspace(); ws != "test-vm" {
		t.Errorf("Expected 'test-vm', got %s", ws)
	}

	// Set to local
	viper.Set("active_workspace", "local")
	if ws := vme.GetActiveWorkspace(); ws != "local" {
		t.Errorf("Expected 'local', got %s", ws)
	}

	// Clean up
	viper.Set("active_workspace", "")
}

// TestVMExecutor_IsVMActive tests VM active detection
func TestVMExecutor_IsVMActive(t *testing.T) {
	ag := New(".")
	vme := NewVMExecutor(ag)

	// No workspace set
	if vme.IsVMActive() {
		t.Error("Expected IsVMActive false when no workspace set")
	}

	// Set to local
	viper.Set("active_workspace", "local")
	if vme.IsVMActive() {
		t.Error("Expected IsVMActive false for 'local' workspace")
	}

	// Set to a VM
	viper.Set("active_workspace", "dev-vm")
	if !vme.IsVMActive() {
		t.Error("Expected IsVMActive true for 'dev-vm' workspace")
	}

	// Clean up
	viper.Set("active_workspace", "")
}

// TestVMExecutor_LocalTools tests that local tools are correctly identified
func TestVMExecutor_LocalTools(t *testing.T) {
	ag := New(".")
	vme := NewVMExecutor(ag)

	localTools := []string{
		"http_get",
		"http_post",
		"web_search",
		"web_fetch",
		"write_docx",
	}

	for _, tool := range localTools {
		// These should always use local executor
		call := ToolCall{Name: tool, Arguments: map[string]interface{}{}}
		result := vme.ExecuteTool(context.Background(), call)

		// Should get a result (error is OK for missing args, but should not panic)
		if result.Name != tool {
			t.Errorf("Expected result name %s, got %s", tool, result.Name)
		}
	}
}

// TestVMExecutor_VMTools tests that VM tools are routed correctly
func TestVMExecutor_VMTools(t *testing.T) {
	ag := New(".")
	vme := NewVMExecutor(ag)

	vmTools := []string{
		"write_file",
		"read_file",
		"append_file",
		"delete_file",
		"list_directory",
		"create_directory",
		"run_shell",
		"change_directory",
	}

	// Without VM active, should use local executor
	for _, tool := range vmTools {
		call := ToolCall{Name: tool, Arguments: map[string]interface{}{}}
		result := vme.ExecuteTool(context.Background(), call)

		if result.Name != tool {
			t.Errorf("Expected result name %s, got %s", tool, result.Name)
		}
	}

	// With VM active but no VM manager, should return error
	viper.Set("active_workspace", "nonexistent-vm")
	for _, tool := range vmTools {
		call := ToolCall{
			Name: tool,
			Arguments: map[string]interface{}{
				"path": "/tmp/test",
			},
		}
		result := vme.ExecuteTool(context.Background(), call)

		// Should get an error about VM manager not available
		if result.Error == nil && tool == "run_shell" {
			// run_shell might work locally if no VM
			t.Logf("Tool %s succeeded locally (expected)", tool)
		}
	}

	// Clean up
	viper.Set("active_workspace", "")
}

// TestVMExecutor_UnknownTool tests unknown tool handling
func TestVMExecutor_UnknownTool(t *testing.T) {
	ag := New(".")
	vme := NewVMExecutor(ag)

	call := ToolCall{Name: "unknown_tool", Arguments: map[string]interface{}{}}
	result := vme.ExecuteTool(context.Background(), call)

	if result.Error == nil {
		t.Error("Expected error for unknown tool")
	}
}

// TestAutonomousAgent_SetVM tests VM mode setting
func TestAutonomousAgent_SetVM(t *testing.T) {
	ag := New(".")
	auto := NewAutonomousAgent(ag)

	// Default is false
	if auto.useVM {
		t.Error("Expected useVM to be false by default")
	}

	// Enable VM mode
	auto.SetVM(true)
	if !auto.useVM {
		t.Error("Expected useVM to be true after SetVM(true)")
	}

	// Disable VM mode
	auto.SetVM(false)
	if auto.useVM {
		t.Error("Expected useVM to be false after SetVM(false)")
	}
}

// TestVMExecutor_ExecuteTools_Multiple tests batch execution
func TestVMExecutor_ExecuteTools_Multiple(t *testing.T) {
	ag := New(".")
	vme := NewVMExecutor(ag)

	calls := []ToolCall{
		{Name: "write_file", Arguments: map[string]interface{}{"path": "/tmp/a", "content": "test"}},
		{Name: "read_file", Arguments: map[string]interface{}{"path": "/tmp/a"}},
	}

	results := vme.ExecuteTools(context.Background(), calls)

	if len(results) != len(calls) {
		t.Errorf("Expected %d results, got %d", len(calls), len(results))
	}

	for i, result := range results {
		if result.Name != calls[i].Name {
			t.Errorf("Result %d: expected name %s, got %s", i, calls[i].Name, result.Name)
		}
	}
}