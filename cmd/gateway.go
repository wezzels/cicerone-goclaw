package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// gatewayCmd represents the gateway command
var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Gateway management",
	Long:  `Manage the Cicerone messaging gateway.`,
}

var gatewayRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart gateway",
	Long: `Restart the Cicerone gateway process.

This will:
  1. Find the running cicerone telegram process
  2. Send SIGTERM for graceful shutdown
  3. Wait for clean exit (5s timeout)
  4. Start a new gateway process`,
	RunE: runGatewayRestart,
}

var gatewayStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check gateway status",
	Long: `Check the status of the Cicerone gateway.

Reports:
  - Process status (running/stopped)
  - PID and uptime
  - Telegram connection status
  - LLM connection status`,
	RunE: runGatewayStatus,
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
	gatewayCmd.AddCommand(gatewayRestartCmd)
	gatewayCmd.AddCommand(gatewayStatusCmd)
}

func runGatewayRestart(cmd *cobra.Command, args []string) error {
	fmt.Println("Restarting Cicerone gateway...")

	// Find running cicerone telegram processes
	pid, err := findCiceroneProcess()
	if err != nil {
		fmt.Println("No running gateway found, starting fresh...")
		return startGateway()
	}

	fmt.Printf("Found gateway process (PID %d), stopping...\n", pid)

	// Stop existing process
	if err := stopProcess(pid, 5*time.Second); err != nil {
		fmt.Printf("Warning: error stopping process: %v\n", err)
	}

	// Start new process
	return startGateway()
}

func runGatewayStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("Cicerone Gateway Status")
	fmt.Println("=======================")

	// Check process
	pid, err := findCiceroneProcess()
	if err != nil {
		fmt.Println("Status: STOPPED")
		fmt.Println("PID:    N/A")
		return nil
	}

	fmt.Printf("Status: RUNNING\n")
	fmt.Printf("PID:    %d\n", pid)

	// Get process info
	if info, err := getProcessInfo(pid); err == nil {
		fmt.Printf("Uptime: %s\n", info)
	}

	// TODO: Check Telegram connection
	// TODO: Check LLM connection

	return nil
}

func findCiceroneProcess() (int, error) {
	// Find cicerone telegram process
	out, err := exec.Command("pgrep", "-f", "cicerone telegram").Output()
	if err != nil {
		return 0, fmt.Errorf("no process found")
	}

	pid := strings.TrimSpace(string(out))
	if pid == "" {
		return 0, fmt.Errorf("no process found")
	}

	var pidInt int
	if _, err := fmt.Sscanf(pid, "%d", &pidInt); err != nil {
		return 0, fmt.Errorf("invalid pid: %s", pid)
	}
	return pidInt, nil
}

func stopProcess(pid int, timeout time.Duration) error {
	// Send SIGTERM
	kill := exec.Command("kill", "-TERM", fmt.Sprintf("%d", pid))
	if err := kill.Run(); err != nil {
		return err
	}

	// Wait for process to exit
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := findCiceroneProcess(); err != nil {
			return nil // Process gone
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if still running
	kill = exec.Command("kill", "-KILL", fmt.Sprintf("%d", pid))
	return kill.Run()
}

func startGateway() error {
	// Start cicerone telegram in background
	cmd := exec.Command("cicerone", "telegram")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	fmt.Printf("Gateway started (PID %d)\n", cmd.Process.Pid)
	return nil
}

func getProcessInfo(pid int) (string, error) {
	// Get process start time
	out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "etimes=").Output()
	if err != nil {
		return "", err
	}

	seconds := strings.TrimSpace(string(out))
	if seconds == "" {
		return "", fmt.Errorf("no info")
	}

	var secs int
	if _, err := fmt.Sscanf(seconds, "%d", &secs); err != nil {
		return "", fmt.Errorf("invalid duration: %s", seconds)
	}
	duration := time.Duration(secs) * time.Second

	return duration.String(), nil
}