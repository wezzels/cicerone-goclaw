package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Executor handles command execution in a workspace
type Executor struct {
	workspace *Workspace
	timeout   time.Duration
	env       []string
	processes sync.Map // pid -> *Process
}

// Process represents a running process
type Process struct {
	PID      int
	Command  string
	Args     []string
	StartTime time.Time
	Status   string
}

// NewExecutor creates a new executor for a workspace
func NewExecutor(w *Workspace) *Executor {
	return &Executor{
		workspace: w,
		timeout:   60 * time.Second,
		env:       os.Environ(),
	}
}

// SetTimeout sets the default timeout for commands
func (e *Executor) SetTimeout(d time.Duration) {
	e.timeout = d
}

// SetEnv sets environment variables
func (e *Executor) SetEnv(env []string) {
	e.env = env
}

// AddEnv adds environment variables
func (e *Executor) AddEnv(key, value string) {
	e.env = append(e.env, fmt.Sprintf("%s=%s", key, value))
}

// Run executes a command and returns output
func (e *Executor) Run(command string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	return e.RunWithContext(ctx, command, args...)
}

// RunWithContext executes a command with context
func (e *Executor) RunWithContext(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = e.workspace.Path
	cmd.Env = e.env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// RunStream executes a command and streams output
func (e *Executor) RunStream(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.workspace.Path
	cmd.Env = e.env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunInteractive executes a command with full terminal interaction
func (e *Executor) RunInteractive(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.workspace.Path
	cmd.Env = e.env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunBackground executes a command in the background
func (e *Executor) RunBackground(command string, args ...string) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.workspace.Path
	cmd.Env = e.env

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start command: %w", err)
	}

	pid := cmd.Process.Pid
	e.processes.Store(pid, &Process{
		PID:       pid,
		Command:   command,
		Args:      args,
		StartTime: time.Now(),
		Status:    "running",
	})

	// Wait for completion in background
	go func() {
		cmd.Wait()
		e.processes.Delete(pid)
	}()

	return pid, nil
}

// ListProcesses lists running background processes
func (e *Executor) ListProcesses() []Process {
	var processes []Process
	e.processes.Range(func(key, value interface{}) bool {
		if p, ok := value.(*Process); ok {
			processes = append(processes, *p)
		}
		return true
	})
	return processes
}

// KillProcess kills a background process by PID
func (e *Executor) KillProcess(pid int) error {
	if p, ok := e.processes.Load(pid); ok {
		if proc, ok := p.(*Process); ok {
			// Find the process
			process, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			if err := process.Kill(); err != nil {
				return err
			}
			e.processes.Delete(pid)
			proc.Status = "killed"
			return nil
		}
	}
	return fmt.Errorf("process %d not found", pid)
}

// RunShell executes a shell command string
func (e *Executor) RunShell(command string) ([]byte, error) {
	return e.Run("sh", "-c", command)
}

// RunWithTimeout executes with a specific timeout
func (e *Executor) RunWithTimeout(timeout time.Duration, command string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return e.RunWithContext(ctx, command, args...)
}

// CaptureOutput captures stdout and stderr separately
func (e *Executor) CaptureOutput(command string, args ...string) (stdout, stderr []byte, err error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.workspace.Path
	cmd.Env = e.env

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// Which finds the full path of an executable
func (e *Executor) Which(command string) (string, error) {
	return exec.LookPath(command)
}

// Exists checks if a command exists
func (e *Executor) Exists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// RunInDir runs a command in a specific directory
func (e *Executor) RunInDir(dir string, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = filepath.Join(e.workspace.Path, dir)
	cmd.Env = e.env

	return cmd.CombinedOutput()
}

// Pipe runs command1 and pipes its output to command2
func (e *Executor) Pipe(cmd1 string, args1 []string, cmd2 string, args2 []string) ([]byte, error) {
	c1 := exec.Command(cmd1, args1...)
	c1.Dir = e.workspace.Path
	c1.Env = e.env

	c2 := exec.Command(cmd2, args2...)
	c2.Dir = e.workspace.Path
	c2.Env = e.env

	pipe, err := c1.StdoutPipe()
	if err != nil {
		return nil, err
	}

	c2.Stdin = pipe

	var output bytes.Buffer
	c2.Stdout = &output

	if err := c1.Start(); err != nil {
		return nil, err
	}

	if err := c2.Start(); err != nil {
		return nil, err
	}

	if err := c1.Wait(); err != nil {
		return nil, err
	}

	if err := c2.Wait(); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// Env returns current environment
func (e *Executor) Env() []string {
	return e.env
}

// Workdir returns the workspace path
func (e *Executor) Workdir() string {
	return e.workspace.Path
}