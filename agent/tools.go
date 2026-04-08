// Package agent provides agentic capabilities for the chat.
package agent

import (
	"encoding/json"
)

// ToolDefinition describes a tool the LLM can call.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]ParamSchema `json:"parameters"`
	Required    []string               `json:"required"`
}

// ParamSchema describes a parameter schema.
type ParamSchema struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Enum        []string               `json:"enum,omitempty"`
	Properties  map[string]ParamSchema `json:"properties,omitempty"`
	Items       *ParamSchema           `json:"items,omitempty"`
}

// ToolCall represents a tool call from the LLM.
type ToolCall struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult is the result of executing a tool.
type ToolResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   error  `json:"error,omitempty"`
}

// GetToolDefinitions returns all available tools for the LLM.
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "write_file",
			Description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Creates parent directories as needed.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The file path to write to (relative or absolute)",
				},
				"content": {
					Type:        "string",
					Description: "The content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The file path to read",
				},
			},
			Required: []string{"path"},
		},
		{
			Name:        "append_file",
			Description: "Append content to an existing file.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The file path to append to",
				},
				"content": {
					Type:        "string",
					Description: "The content to append",
				},
			},
			Required: []string{"path", "content"},
		},
		{
			Name:        "delete_file",
			Description: "Delete a file.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The file path to delete",
				},
			},
			Required: []string{"path"},
		},
		{
			Name:        "list_directory",
			Description: "List the contents of a directory.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The directory path to list (defaults to current directory)",
				},
			},
			Required: []string{},
		},
		{
			Name:        "create_directory",
			Description: "Create a directory and all parent directories.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The directory path to create",
				},
			},
			Required: []string{"path"},
		},
		{
			Name:        "run_shell",
			Description: "Execute a shell command. Use with caution.",
			Parameters: map[string]ParamSchema{
				"command": {
					Type:        "string",
					Description: "The shell command to execute",
				},
			},
			Required: []string{"command"},
		},
		{
			Name:        "change_directory",
			Description: "Change the current working directory.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The directory path to change to",
				},
			},
			Required: []string{"path"},
		},
		{
			Name:        "http_get",
			Description: "Make an HTTP GET request to a URL.",
			Parameters: map[string]ParamSchema{
				"url": {
					Type:        "string",
					Description: "The URL to request",
				},
			},
			Required: []string{"url"},
		},
		{
			Name:        "http_post",
			Description: "Make an HTTP POST request to a URL with JSON data.",
			Parameters: map[string]ParamSchema{
				"url": {
					Type:        "string",
					Description: "The URL to post to",
				},
				"data": {
					Type:        "object",
					Description: "The JSON data to post",
				},
			},
			Required: []string{"url", "data"},
		},
		{
			Name:        "web_search",
			Description: "Search the web for information. Returns search results with titles, URLs, and snippets.",
			Parameters: map[string]ParamSchema{
				"query": {
					Type:        "string",
					Description: "The search query",
				},
			},
			Required: []string{"query"},
		},
		{
			Name:        "web_fetch",
			Description: "Fetch content from a web URL. Returns the text content of the page.",
			Parameters: map[string]ParamSchema{
				"url": {
					Type:        "string",
					Description: "The URL to fetch",
				},
			},
			Required: []string{"url"},
		},
		{
			Name:        "write_docx",
			Description: "Write content to a DOCX (Word) document. Creates a properly formatted document.",
			Parameters: map[string]ParamSchema{
				"path": {
					Type:        "string",
					Description: "The output file path (should end with .docx)",
				},
				"title": {
					Type:        "string",
					Description: "The document title",
				},
				"content": {
					Type:        "string",
					Description: "The document content (text with newlines)",
				},
			},
			Required: []string{"path", "content"},
		},
	}
}

// ToolsToJSON converts tool definitions to JSON for LLM context.
func ToolsToJSON() (string, error) {
	tools := GetToolDefinitions()
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToolsToOllamaFormat converts tools to Ollama's tool format.
func ToolsToOllamaFormat() []map[string]interface{} {
	tools := GetToolDefinitions()
	result := make([]map[string]interface{}, len(tools))

	for i, tool := range tools {
		params := map[string]interface{}{
			"type":       "object",
			"properties": tool.Parameters,
		}
		if len(tool.Required) > 0 {
			params["required"] = tool.Required
		}

		result[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  params,
			},
		}
	}

	return result
}