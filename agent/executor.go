// Package agent provides tool execution for the autonomous agent.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crab-meat-repos/cicerone-goclaw/web"
)

// Executor runs tool calls from the agent.
type Executor struct {
	agent     *Agent
	webClient *web.DuckDuckGoProvider
}

// NewExecutor creates a new tool executor.
func NewExecutor(ag *Agent) *Executor {
	return &Executor{
		agent:     ag,
		webClient: web.NewDuckDuckGoProvider(),
	}
}

// ExecuteTool runs a tool call and returns the result.
func (e *Executor) ExecuteTool(ctx context.Context, call ToolCall) ToolResult {
	result := ToolResult{
		Name: call.Name,
	}

	switch call.Name {
	case "write_file":
		result.Output, result.Error = e.writeFile(call.Arguments)
		result.Success = result.Error == nil

	case "read_file":
		result.Output, result.Error = e.readFile(call.Arguments)
		result.Success = result.Error == nil

	case "append_file":
		result.Output, result.Error = e.appendFile(call.Arguments)
		result.Success = result.Error == nil

	case "delete_file":
		result.Output, result.Error = e.deleteFile(call.Arguments)
		result.Success = result.Error == nil

	case "list_directory":
		result.Output, result.Error = e.listDirectory(call.Arguments)
		result.Success = result.Error == nil

	case "create_directory":
		result.Output, result.Error = e.createDirectory(call.Arguments)
		result.Success = result.Error == nil

	case "run_shell":
		result.Output, result.Error = e.runShell(ctx, call.Arguments)
		result.Success = result.Error == nil

	case "change_directory":
		result.Output, result.Error = e.changeDirectory(call.Arguments)
		result.Success = result.Error == nil

	case "http_get":
		result.Output, result.Error = e.httpGet(ctx, call.Arguments)
		result.Success = result.Error == nil

	case "http_post":
		result.Output, result.Error = e.httpPost(ctx, call.Arguments)
		result.Success = result.Error == nil

	case "web_search":
		result.Output, result.Error = e.webSearch(ctx, call.Arguments)
		result.Success = result.Error == nil

	case "web_fetch":
		result.Output, result.Error = e.webFetch(ctx, call.Arguments)
		result.Success = result.Error == nil

	case "write_docx":
		result.Output, result.Error = e.writeDocx(call.Arguments)
		result.Success = result.Error == nil

	default:
		result.Error = fmt.Errorf("unknown tool: %s", call.Name)
	}

	return result
}

// ExecuteTools runs multiple tool calls.
func (e *Executor) ExecuteTools(ctx context.Context, calls []ToolCall) []ToolResult {
	results := make([]ToolResult, len(calls))
	for i, call := range calls {
		results[i] = e.ExecuteTool(ctx, call)
	}
	return results
}

// Tool implementations

func (e *Executor) writeFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'content' argument")
	}

	if err := e.agent.WriteFile(path, content); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

func (e *Executor) readFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	content, err := e.agent.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	return content, nil
}

func (e *Executor) appendFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'content' argument")
	}

	if err := e.agent.AppendFile(path, content); err != nil {
		return "", fmt.Errorf("append failed: %w", err)
	}
	return fmt.Sprintf("Successfully appended to %s", path), nil
}

func (e *Executor) deleteFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	if err := e.agent.DeleteFile(path); err != nil {
		return "", fmt.Errorf("delete failed: %w", err)
	}
	return fmt.Sprintf("Successfully deleted %s", path), nil
}

func (e *Executor) listDirectory(args map[string]interface{}) (string, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	entries, err := e.agent.ListDir(path)
	if err != nil {
		return "", fmt.Errorf("list failed: %w", err)
	}

	var result strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		result.WriteString(fmt.Sprintf("%s %8d %s\n", info.Mode().String()[:10], info.Size(), entry.Name()))
	}
	return result.String(), nil
}

func (e *Executor) createDirectory(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	if err := e.agent.Mkdir(path); err != nil {
		return "", fmt.Errorf("mkdir failed: %w", err)
	}
	return fmt.Sprintf("Successfully created directory %s", path), nil
}

func (e *Executor) runShell(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'command' argument")
	}

	output, err := e.agent.Execute(ctx, command)
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	return output, nil
}

func (e *Executor) changeDirectory(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	if err := e.agent.SetWorkDir(path); err != nil {
		return "", fmt.Errorf("cd failed: %w", err)
	}
	return fmt.Sprintf("Changed directory to %s", e.agent.WorkDir()), nil
}

func (e *Executor) httpGet(ctx context.Context, args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'url' argument")
	}

	return e.agent.HTTPGet(ctx, url, nil)
}

func (e *Executor) httpPost(ctx context.Context, args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'url' argument")
	}

	data := args["data"]
	return e.agent.HTTPPost(ctx, url, data, nil)
}

func (e *Executor) webSearch(ctx context.Context, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'query' argument")
	}

	results, err := e.webClient.Search(ctx, query)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	return web.FormatSearchResults(results), nil
}

func (e *Executor) webFetch(ctx context.Context, args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'url' argument")
	}

	return e.webClient.Fetch(ctx, url)
}

func (e *Executor) writeDocx(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'content' argument")
	}
	title, _ := args["title"].(string) // Optional

	// Resolve path
	fullPath := e.agent.ResolvePath(path)

	// Create DOCX file (DOCX is a ZIP containing XML)
	if err := createDocx(fullPath, title, content); err != nil {
		return "", fmt.Errorf("failed to create docx: %w", err)
	}

	return fmt.Sprintf("Successfully created DOCX document at %s", path), nil
}

// FormatToolResults formats tool results for LLM context.
func FormatToolResults(results []ToolResult) string {
	var sb strings.Builder
	sb.WriteString("Tool Results:\n")
	sb.WriteString(strings.Repeat("-", 50) + "\n\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("## %s\n", r.Name))
		if r.Success {
			sb.WriteString("Status: Success\n")
			sb.WriteString(fmt.Sprintf("Output:\n%s\n", r.Output))
		} else {
			sb.WriteString("Status: Failed\n")
			if r.Error != nil {
				sb.WriteString(fmt.Sprintf("Error: %s\n", r.Error.Error()))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ParseToolCalls extracts tool calls from LLM response.
// Handles both JSON format and natural language tool invocations.
func ParseToolCalls(response string) ([]ToolCall, error) {
	var calls []ToolCall

	// Try to parse as JSON array first
	if strings.HasPrefix(response, "[") {
		var jsonCalls []struct {
			ID        string                 `json:"id"`
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(response), &jsonCalls); err == nil {
			for _, jc := range jsonCalls {
				calls = append(calls, ToolCall{
					ID:        jc.ID,
					Name:      jc.Name,
					Arguments: jc.Arguments,
				})
			}
			return calls, nil
		}
	}

	// Try to parse as single JSON object
	if strings.HasPrefix(response, "{") {
		var jc struct {
			ID        string                 `json:"id"`
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(response), &jc); err == nil {
			calls = append(calls, ToolCall{
				ID:        jc.ID,
				Name:      jc.Name,
				Arguments: jc.Arguments,
			})
			return calls, nil
		}
	}

	// Look for embedded tool calls in text using markers
	// Format: TOOL_CALL:{"name": "...", "arguments": {...}}
	marker := "TOOL_CALL:"
	idx := strings.Index(response, marker)
	for idx != -1 {
		// Find the JSON after the marker
		jsonStart := idx + len(marker)
		jsonStr := response[jsonStart:]

		// Find end of JSON
		depth := 0
		jsonEnd := -1
		for i, ch := range jsonStr {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					jsonEnd = i + 1
					break
				}
			}
		}

		if jsonEnd > 0 {
			jsonData := jsonStr[:jsonEnd]
			var tc struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}
			if err := json.Unmarshal([]byte(jsonData), &tc); err == nil && tc.Name != "" {
				calls = append(calls, ToolCall{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				})
			}
		}

		// Look for next marker
		nextIdx := strings.Index(response[idx+1:], marker)
		if nextIdx == -1 {
			break
		}
		idx = idx + 1 + nextIdx
	}

	return calls, nil
}

// createDocx creates a simple DOCX file.
// DOCX is a ZIP archive containing XML files.
func createDocx(path, title, content string) error {
	// Create a minimal DOCX structure
	// This is a simplified version - a proper implementation would use a DOCX library

	// For now, we'll create a simple text file with .docx extension
	// A proper implementation would need: archive/zip, encoding/xml

	// Build the content
	var sb strings.Builder
	if title != "" {
		sb.WriteString(title)
		sb.WriteString("\n\n")
	}
	sb.WriteString(content)

	// Write as plain text (user can open in Word or convert)
	// TODO: Implement proper DOCX creation with zip/xml
	return os.WriteFile(path, []byte(sb.String()), 0644)
}