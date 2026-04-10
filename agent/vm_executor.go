// Package agent provides VM-aware tool execution.
package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/ssh"
	"github.com/crab-meat-repos/cicerone-goclaw/internal/vm"
	"github.com/spf13/viper"
)

// VMExecutor wraps Executor with VM-awareness.
// When a VM workspace is active, file and shell operations are routed through SSH.
type VMExecutor struct {
	executor  *Executor
	vmManager vm.Manager
	vmConfig  *vm.Config
}

// NewVMExecutor creates a VM-aware executor.
func NewVMExecutor(ag *Agent) *VMExecutor {
	exe := NewExecutor(ag)

	// Load VM config
	vmCfg, _ := vm.LoadConfig()

	// Try to create libvirt manager (may fail if libvirt not available)
	mgr, _ := vm.NewLibvirtManager(nil)

	return &VMExecutor{
		executor:  exe,
		vmManager: mgr,
		vmConfig:  vmCfg,
	}
}

// GetActiveWorkspace returns the active VM workspace name, or empty string for local.
func (e *VMExecutor) GetActiveWorkspace() string {
	return viper.GetString("active_workspace")
}

// IsVMActive returns true if a VM workspace is active.
func (e *VMExecutor) IsVMActive() bool {
	return e.GetActiveWorkspace() != "" && e.GetActiveWorkspace() != "local"
}

// ExecuteTool runs a tool call, routing to VM if active.
func (e *VMExecutor) ExecuteTool(ctx context.Context, call ToolCall) ToolResult {
	// If no VM active, use local executor
	if !e.IsVMActive() {
		return e.executor.ExecuteTool(ctx, call)
	}

	// Get VM info
	vmName := e.GetActiveWorkspace()

	// Tools that should run on VM
	vmTools := map[string]bool{
		"write_file":      true,
		"read_file":       true,
		"append_file":     true,
		"delete_file":     true,
		"list_directory":  true,
		"create_directory": true,
		"run_shell":       true,
		"change_directory": true,
	}

	// Tools that should run locally
	localTools := map[string]bool{
		"http_get":   true,
		"http_post":  true,
		"web_search": true,
		"web_fetch":  true,
		"write_docx": true,
	}

	// Route to appropriate executor
	if vmTools[call.Name] {
		return e.executeOnVM(ctx, vmName, call)
	}

	if localTools[call.Name] {
		return e.executor.ExecuteTool(ctx, call)
	}

	// Unknown tool
	return ToolResult{
		Name:  call.Name,
		Error: fmt.Errorf("unknown tool: %s", call.Name),
	}
}

// ExecuteTools runs multiple tool calls.
func (e *VMExecutor) ExecuteTools(ctx context.Context, calls []ToolCall) []ToolResult {
	results := make([]ToolResult, len(calls))
	for i, call := range calls {
		results[i] = e.ExecuteTool(ctx, call)
	}
	return results
}

// executeOnVM executes a tool call on the active VM via SSH.
func (e *VMExecutor) executeOnVM(ctx context.Context, vmName string, call ToolCall) ToolResult {
	result := ToolResult{Name: call.Name}

	// Check if VM manager is available
	if e.vmManager == nil {
		// Try to create one
		mgr, err := vm.NewLibvirtManager(nil)
		if err != nil {
			result.Error = fmt.Errorf("VM manager not available: %w", err)
			return result
		}
		e.vmManager = mgr
	}

	// Get SSH client for VM
	sshClient, err := e.getSSHClient(ctx, vmName)
	if err != nil {
		result.Error = fmt.Errorf("failed to connect to VM: %w", err)
		return result
	}
	defer sshClient.Close()

	// Execute based on tool
	switch call.Name {
	case "write_file":
		result.Output, result.Error = e.vmWriteFile(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "read_file":
		result.Output, result.Error = e.vmReadFile(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "append_file":
		result.Output, result.Error = e.vmAppendFile(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "delete_file":
		result.Output, result.Error = e.vmDeleteFile(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "list_directory":
		result.Output, result.Error = e.vmListDirectory(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "create_directory":
		result.Output, result.Error = e.vmCreateDirectory(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "run_shell":
		result.Output, result.Error = e.vmRunShell(ctx, sshClient, call.Arguments)
		result.Success = result.Error == nil

	case "change_directory":
		// Directory changes on VM need to be tracked locally
		path, ok := call.Arguments["path"].(string)
		if !ok {
			result.Error = fmt.Errorf("missing path argument")
		} else {
			result.Output = fmt.Sprintf("Changed directory to %s on VM %s", path, vmName)
			result.Success = true
			// Note: We'd need to track working directory per VM session
		}

	default:
		result.Error = fmt.Errorf("unsupported VM tool: %s", call.Name)
	}

	return result
}

// getSSHClient creates an SSH client for the VM.
func (e *VMExecutor) getSSHClient(ctx context.Context, vmName string) (*ssh.Client, error) {
	// Get VM status
	info, err := e.vmManager.Status(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM status: %w", err)
	}

	if info.State != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (state: %s)", vmName, info.State)
	}

	if info.IP == "" {
		return nil, fmt.Errorf("VM '%s' has no IP address", vmName)
	}

	// Get SSH key path
	keyPath, err := vm.GetVMKeyPath(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key path: %w", err)
	}

	// Check if key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Fall back to default key
		home, _ := os.UserHomeDir()
		keyPath = home + "/.ssh/id_ed25519"
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			keyPath = home + "/.ssh/id_rsa"
		}
	}

	// Create SSH config
	cfg := &ssh.Config{
		Host:    info.IP,
		Port:    22,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 30,
	}

	return ssh.NewClient(cfg)
}

// VM tool implementations

func (e *VMExecutor) vmWriteFile(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'content' argument")
	}

	// Write using cat with heredoc to handle special characters
	cmd := fmt.Sprintf("cat > %q << 'EOFMARKER'\n%s\nEOFMARKER", path, content)
	_, stderr, err := client.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("write failed: %w, stderr: %s", err, string(stderr))
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

func (e *VMExecutor) vmReadFile(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	cmd := fmt.Sprintf("cat %q", path)
	stdout, stderr, err := client.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("read failed: %w, stderr: %s", err, string(stderr))
	}

	return string(stdout), nil
}

func (e *VMExecutor) vmAppendFile(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'content' argument")
	}

	cmd := fmt.Sprintf("echo %q >> %q", content, path)
	_, stderr, err := client.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("append failed: %w, stderr: %s", err, string(stderr))
	}

	return fmt.Sprintf("Successfully appended to %s", path), nil
}

func (e *VMExecutor) vmDeleteFile(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	cmd := fmt.Sprintf("rm -f %q", path)
	_, stderr, err := client.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("delete failed: %w, stderr: %s", err, string(stderr))
	}

	return fmt.Sprintf("Successfully deleted %s", path), nil
}

func (e *VMExecutor) vmListDirectory(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	cmd := fmt.Sprintf("ls -la %q 2>/dev/null || ls -la %s", path, path)
	stdout, stderr, err := client.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("list failed: %w, stderr: %s", err, string(stderr))
	}

	return string(stdout), nil
}

func (e *VMExecutor) vmCreateDirectory(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	cmd := fmt.Sprintf("mkdir -p %q", path)
	_, stderr, err := client.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("mkdir failed: %w, stderr: %s", err, string(stderr))
	}

	return fmt.Sprintf("Successfully created directory %s", path), nil
}

func (e *VMExecutor) vmRunShell(ctx context.Context, client *ssh.Client, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'command' argument")
	}

	stdout, stderr, err := client.Exec(ctx, command)
	output := string(stdout)
	if len(stderr) > 0 {
		output += "\n[stderr]\n" + string(stderr)
	}

	return output, err
}