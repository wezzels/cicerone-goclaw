package agent

import (
	"encoding/json"
	"testing"
)

func TestGetToolDefinitions(t *testing.T) {
	tools := GetToolDefinitions()

	if len(tools) == 0 {
		t.Error("GetToolDefinitions returned no tools")
	}

	// Check for essential tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	requiredTools := []string{
		"write_file",
		"read_file",
		"append_file",
		"delete_file",
		"list_directory",
		"create_directory",
		"run_shell",
		"change_directory",
		"http_get",
		"http_post",
		"web_search",
		"web_fetch",
		"write_docx",
	}

	for _, required := range requiredTools {
		if !toolNames[required] {
			t.Errorf("Missing required tool: %s", required)
		}
	}
}

func TestToolDefinitionStructure(t *testing.T) {
	tools := GetToolDefinitions()

	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("Tool has empty name")
		}
		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}
		if len(tool.Parameters) == 0 {
			t.Errorf("Tool %s has no parameters", tool.Name)
		}
	}
}

func TestToolsToJSON(t *testing.T) {
	jsonStr, err := ToolsToJSON()
	if err != nil {
		t.Fatalf("ToolsToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("ToolsToJSON returned empty string")
	}

	// Verify it's valid JSON
	var tools []ToolDefinition
	if err := json.Unmarshal([]byte(jsonStr), &tools); err != nil {
		t.Errorf("ToolsToJSON produced invalid JSON: %v", err)
	}
}

func TestToolsToOllamaFormat(t *testing.T) {
	ollamaTools := ToolsToOllamaFormat()

	if len(ollamaTools) == 0 {
		t.Error("ToolsToOllamaFormat returned no tools")
	}

	// Verify structure
	for _, tool := range ollamaTools {
		if tool.Type != "function" {
			t.Error("Tool type should be 'function'")
			continue
		}

		if tool.Function.Name == "" {
			t.Error("Tool function missing name")
		}

		if tool.Function.Description == "" {
			t.Errorf("Tool %s missing description", tool.Function.Name)

		}
	}
	// Verify structure
	for _, tool := range ollamaTools {
		if tool.Type != "function" {
			t.Error("Tool type should be 'function'")
			continue
		}

		if tool.Function.Name == "" {
			t.Error("Tool function missing name")
		}

		if tool.Function.Description == "" {
			t.Errorf("Tool %s missing description", tool.Function.Name)
		}

		// Check parameters has type
		paramType, ok := tool.Function.Parameters["type"].(string)
		if !ok || paramType != "object" {
			t.Errorf("Tool %s parameters should have type 'object'", tool.Function.Name)
		}

		// Check properties exists
		if _, ok := tool.Function.Parameters["properties"]; !ok {
			t.Errorf("Tool %s missing properties", tool.Function.Name)
		}
	}
}

func TestToolCallParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ToolCall
		wantErr  bool
	}{
		{
			name:  "single tool call",
			input: `{"name": "write_file", "arguments": {"path": "test.txt", "content": "hello"}}`,
			expected: []ToolCall{
				{Name: "write_file", Arguments: map[string]interface{}{"path": "test.txt", "content": "hello"}},
			},
		},
		{
			name:  "array of tool calls",
			input: `[{"name": "write_file", "arguments": {"path": "a.txt"}}, {"name": "read_file", "arguments": {"path": "b.txt"}}]`,
			expected: []ToolCall{
				{Name: "write_file", Arguments: map[string]interface{}{"path": "a.txt"}},
				{Name: "read_file", Arguments: map[string]interface{}{"path": "b.txt"}},
			},
		},
		{
			name:     "invalid JSON",
			input:    `not json`,
			wantErr:  false, // ParseToolCalls returns empty on error
			expected: nil,
		},
		{
			name:     "embedded tool call",
			input:    `I will use TOOL_CALL:{"name": "write_file", "arguments": {"path": "test.txt"}} to create the file`,
			expected: []ToolCall{
				{Name: "write_file", Arguments: map[string]interface{}{"path": "test.txt"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := ParseToolCalls(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(calls) != len(tt.expected) {
				t.Errorf("expected %d calls, got %d", len(tt.expected), len(calls))
				return
			}

			for i, call := range calls {
				if call.Name != tt.expected[i].Name {
					t.Errorf("call %d: expected name %s, got %s", i, tt.expected[i].Name, call.Name)
				}
			}
		})
	}
}

func TestFormatToolResults(t *testing.T) {
	results := []ToolResult{
		{
			Name:    "write_file",
			Success: true,
			Output:  "Wrote 10 bytes to test.txt",
		},
		{
			Name:    "read_file",
			Success: false,
			Error:   nil,
			Output:  "",
		},
	}

	formatted := FormatToolResults(results)

	if formatted == "" {
		t.Error("FormatToolResults returned empty string")
	}

	// Check it contains tool names
	if !containsStr(formatted, "write_file") {
		t.Error("Formatted output missing write_file")
	}
	if !containsStr(formatted, "read_file") {
		t.Error("Formatted output missing read_file")
	}
	if !containsStr(formatted, "Success") {
		t.Error("Formatted output missing Success status")
	}
}