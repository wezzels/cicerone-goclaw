// Package agent provides agentic capabilities for the chat.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Agent provides agentic capabilities.
type Agent struct {
	workDir    string
	httpClient *http.Client
	commands   map[string]CommandFunc
}

// CommandFunc is a command handler function.
type CommandFunc func(ctx context.Context, args string) (string, error)

// New creates a new agent.
func New(workDir string) *Agent {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	a := &Agent{
		workDir: workDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		commands: make(map[string]CommandFunc),
	}

	// Register built-in commands
	a.registerCommands()

	return a
}

// SetWorkDir sets the working directory.
func (a *Agent) SetWorkDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	a.workDir = abs
	return nil
}

// WorkDir returns the current working directory.
func (a *Agent) WorkDir() string {
	return a.workDir
}

// Register registers a custom command.
func (a *Agent) Register(name string, fn CommandFunc) {
	a.commands[name] = fn
}

// Execute runs a shell command.
func (a *Agent) Execute(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = a.workDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[stderr]\n" + stderr.String()
	}

	return strings.TrimSpace(output), err
}

// ReadFile reads a file.
func (a *Agent) ReadFile(path string) (string, error) {
	fullPath := a.resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes to a file.
func (a *Agent) WriteFile(path, content string) error {
	fullPath := a.resolvePath(path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// AppendFile appends to a file.
func (a *Agent) AppendFile(path, content string) error {
	fullPath := a.resolvePath(path)
	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// DeleteFile deletes a file.
func (a *Agent) DeleteFile(path string) error {
	return os.Remove(a.resolvePath(path))
}

// ListDir lists directory contents.
func (a *Agent) ListDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(a.resolvePath(path))
}

// Mkdir creates a directory.
func (a *Agent) Mkdir(path string) error {
	return os.MkdirAll(a.resolvePath(path), 0755)
}

// HTTPGet performs a GET request.
func (a *Agent) HTTPGet(ctx context.Context, url string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Status: %s\n\n%s", resp.Status, string(body))
	return result, nil
}

// HTTPPost performs a POST request.
func (a *Agent) HTTPPost(ctx context.Context, url string, data interface{}, headers map[string]string) (string, error) {
	var body io.Reader
	var contentType string

	switch v := data.(type) {
	case string:
		body = strings.NewReader(v)
		contentType = "text/plain"
	case []byte:
		body = bytes.NewReader(v)
		contentType = "application/octet-stream"
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(jsonData)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", contentType)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Status: %s\n\n%s", resp.Status, string(respBody))
	return result, nil
}

// HTTPRequest performs a custom HTTP request.
func (a *Agent) HTTPRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Status: %s\nHeaders:\n", resp.Status)
	for k, v := range resp.Header {
		result += fmt.Sprintf("  %s: %s\n", k, strings.Join(v, ", "))
	}
	result += fmt.Sprintf("\n%s", string(respBody))
	return result, nil
}

// RunCommand executes a registered command.
func (a *Agent) RunCommand(ctx context.Context, name, args string) (string, error) {
	fn, ok := a.commands[name]
	if !ok {
		return "", fmt.Errorf("unknown command: %s", name)
	}
	return fn(ctx, args)
}

// ListCommands returns available commands.
func (a *Agent) ListCommands() []string {
	var cmds []string
	for cmd := range a.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// resolvePath resolves a path relative to workdir.
func (a *Agent) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(a.workDir, path)
}

// registerCommands registers built-in commands.
func (a *Agent) registerCommands() {
	// Shell execution
	a.commands["run"] = func(ctx context.Context, args string) (string, error) {
		return a.Execute(ctx, args)
	}

	// File operations
	a.commands["read"] = func(ctx context.Context, args string) (string, error) {
		return a.ReadFile(strings.TrimSpace(args))
	}

	a.commands["write"] = func(ctx context.Context, args string) (string, error) {
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("usage: write <file> <content>")
		}
		return "", a.WriteFile(parts[0], parts[1])
	}

	a.commands["append"] = func(ctx context.Context, args string) (string, error) {
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("usage: append <file> <content>")
		}
		return "", a.AppendFile(parts[0], parts[1])
	}

	a.commands["delete"] = func(ctx context.Context, args string) (string, error) {
		return "", a.DeleteFile(strings.TrimSpace(args))
	}

	a.commands["ls"] = func(ctx context.Context, args string) (string, error) {
		path := strings.TrimSpace(args)
		if path == "" {
			path = "."
		}
		entries, err := a.ListDir(path)
		if err != nil {
			return "", err
		}
		var result strings.Builder
		for _, entry := range entries {
			info, _ := entry.Info()
			result.WriteString(fmt.Sprintf("%s %s\n", info.Mode(), entry.Name()))
		}
		return result.String(), nil
	}

	a.commands["mkdir"] = func(ctx context.Context, args string) (string, error) {
		return "", a.Mkdir(strings.TrimSpace(args))
	}

	// Directory navigation
	a.commands["cd"] = func(ctx context.Context, args string) (string, error) {
		return "", a.SetWorkDir(strings.TrimSpace(args))
	}

	a.commands["pwd"] = func(ctx context.Context, args string) (string, error) {
		return a.WorkDir(), nil
	}

	// HTTP operations
	a.commands["get"] = func(ctx context.Context, args string) (string, error) {
		return a.HTTPGet(ctx, strings.TrimSpace(args), nil)
	}

	a.commands["post"] = func(ctx context.Context, args string) (string, error) {
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("usage: post <url> <json>")
		}
		var data interface{}
		if err := json.Unmarshal([]byte(parts[1]), &data); err != nil {
			data = parts[1]
		}
		return a.HTTPPost(ctx, parts[0], data, nil)
	}
}

// Help returns help text for commands.
func (a *Agent) Help() string {
	return `Agent Commands:

Shell:
  /run <command>     - Execute shell command
  /cd <path>         - Change directory
  /pwd               - Show current directory

Files:
  /read <file>       - Read file contents
  /write <file> <content> - Write to file
  /append <file> <content> - Append to file
  /delete <file>      - Delete file
  /ls [path]         - List directory
  /mkdir <dir>       - Create directory

HTTP:
  /get <url>         - HTTP GET request
  /post <url> <json> - HTTP POST request

Other:
  /help              - Show this help
  /commands          - List available commands`
}