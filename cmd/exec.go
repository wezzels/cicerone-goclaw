package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/workspace"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Execute commands in workspace",
	Long: `Execute commands in the workspace environment.

Supports timeout, background execution, and process management.

Examples:
  cicerone exec "go build ./..."
  cicerone exec --timeout 30s "go test ./..."
  cicerone exec --bg "go run main.go"
  cicerone exec ps
  cicerone exec kill <pid>`,
	RunE: runExec,
}

var (
	execTimeout    time.Duration
	execBackground bool
	execWorkdir    string
	execEnv        []string
	execVerbose    bool
)

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().DurationVar(&execTimeout, "timeout", 60*time.Second, "command timeout")
	execCmd.Flags().BoolVarP(&execBackground, "background", "b", false, "run in background")
	execCmd.Flags().StringVarP(&execWorkdir, "workdir", "w", "", "workspace directory")
	execCmd.Flags().StringArrayVarP(&execEnv, "env", "e", nil, "environment variables (KEY=VALUE)")
	execCmd.Flags().BoolVarP(&execVerbose, "verbose", "v", false, "verbose output")
}

func runExec(cmd *cobra.Command, args []string) error {
	// Check for process management
	if args[0] == "ps" {
		return listProcesses()
	}
	if args[0] == "kill" && len(args) > 1 {
		return killProcess(args[1])
	}

	// Get workspace
	workdir := "."
	if execWorkdir != "" {
		workdir = execWorkdir
	}

	abs, err := filepath.Abs(workdir)
	if err != nil {
		return fmt.Errorf("invalid workdir: %w", err)
	}

	// Create workspace (doesn't need to be initialized)
	w, err := workspace.New(abs)
	if err != nil {
		return err
	}

	executor := workspace.NewExecutor(w)
	executor.SetTimeout(execTimeout)

	// Add custom env vars
	for _, env := range execEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			executor.AddEnv(parts[0], parts[1])
		}
	}

	// Parse command
	command := strings.Join(args, " ")

	// Check for shell command vs single command
	var result []byte

	if execBackground {
		// Background execution
		shellCmd := args[0]
		shellArgs := args[1:]
		pid, err := executor.RunBackground(shellCmd, shellArgs...)
		if err != nil {
			return fmt.Errorf("failed to start: %w", err)
		}
		fmt.Printf("Started process PID %d\n", pid)
		fmt.Printf("Run 'cicerone exec ps' to list processes\n")
		return nil
	}

	// Interactive mode - stream output
	if !execVerbose {
		// Stream mode
		ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
		defer cancel()

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		// Run with context
		shellCmd := args[0]
		shellArgs := args[1:]

		execCmd := exec.CommandContext(ctx, shellCmd, shellArgs...)
		execCmd.Dir = w.Path
		execCmd.Env = executor.Env()
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("command timed out after %s", execTimeout)
			}
			return fmt.Errorf("command failed: %w", err)
		}

		return nil
	}

	// Captured mode
	result, err = executor.RunShell(command)
	if err != nil {
		fmt.Println(string(result))
		return err
	}

	fmt.Println(string(result))
	return nil
}

func listProcesses() error {
	fmt.Println("Running Processes")
	fmt.Println("================")
	fmt.Println()

	// This is a simplified version - in real implementation we'd track background processes
	fmt.Println("  No tracked processes")
	fmt.Println()
	fmt.Println("Note: Use 'cicerone exec --bg' to start background processes")

	return nil
}

func killProcess(pidStr string) error {
	// Simplified - in real implementation we'd track and kill our own processes
	fmt.Printf("Kill process %s - not implemented\n", pidStr)
	return nil
}