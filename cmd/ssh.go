package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/ssh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH connection management",
	Long: `Manage SSH connections to remote hosts.

Execute commands, transfer files, and create tunnels.

Examples:
  cicerone ssh add darth 10.0.0.117 wez
  cicerone ssh list
  cicerone ssh test darth
  cicerone ssh exec darth "ls -la"
  cicerone ssh shell darth
  cicerone ssh push darth local.txt /remote/path.txt
  cicerone ssh pull darth /remote/path.txt local.txt`,
}

var (
	sshKeyPath  string
	sshPort     int
	sshTimeout  time.Duration
	sshPassword string
)

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.AddCommand(sshAddCmd)
	sshCmd.AddCommand(sshListCmd)
	sshCmd.AddCommand(sshTestCmd)
	sshCmd.AddCommand(sshExecCmd)
	sshCmd.AddCommand(sshShellCmd)
	sshCmd.AddCommand(sshPushCmd)
	sshCmd.AddCommand(sshPullCmd)
	sshCmd.AddCommand(sshRemoveCmd)

	// Global SSH flags
	sshCmd.PersistentFlags().StringVarP(&sshKeyPath, "key", "k", "~/.ssh/id_rsa", "SSH key path")
	sshCmd.PersistentFlags().IntVarP(&sshPort, "port", "p", 22, "SSH port")
	sshCmd.PersistentFlags().DurationVar(&sshTimeout, "timeout", 30*time.Second, "connection timeout")
}

var sshAddCmd = &cobra.Command{
	Use:   "add <name> <host> <user>",
	Short: "Add SSH host configuration",
	Long: `Add a new SSH host to the configuration.

The host configuration is saved to ~/.cicerone/ssh_hosts.yaml`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		host := args[1]
		user := args[2]

		// Load existing hosts
		hosts, err := loadSSHHosts()
		if err != nil {
			hosts = make(map[string]ssh.HostAlias)
		}

		// Check if already exists
		if _, exists := hosts[name]; exists {
			fmt.Printf("Host '%s' already exists. Overwrite? [y/N]: ", name)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(response)) != "y" {
				return nil
			}
		}

		// Add new host
		hosts[name] = ssh.HostAlias{
			Name:    name,
			Host:    host,
			Port:    sshPort,
			User:    user,
			KeyPath: sshKeyPath,
		}

		// Save
		if err := saveSSHHosts(hosts); err != nil {
			return fmt.Errorf("failed to save hosts: %w", err)
		}

		fmt.Printf("✓ Added SSH host '%s'\n", name)
		fmt.Printf("  %s@%s:%d\n", user, host, sshPort)
		fmt.Printf("  Key: %s\n", sshKeyPath)

		return nil
	},
}

var sshListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSH hosts",
	Long:  `List all configured SSH hosts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		if len(hosts) == 0 {
			fmt.Println("No SSH hosts configured")
			fmt.Println()
			fmt.Println("Add a host with: cicerone ssh add <name> <host> <user>")
			return nil
		}

		fmt.Println("SSH Hosts")
		fmt.Println("=========")
		fmt.Println()

		for name, h := range hosts {
			fmt.Printf("  %s\n", name)
			fmt.Printf("    Host: %s:%d\n", h.Host, h.Port)
			fmt.Printf("    User: %s\n", h.User)
			fmt.Printf("    Key:  %s\n", h.KeyPath)
			if h.Description != "" {
				fmt.Printf("    Desc: %s\n", h.Description)
			}
			fmt.Println()
		}

		return nil
	},
}

var sshTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test SSH connection",
	Long:  `Test connection to an SSH host.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		h, exists := hosts[name]
		if !exists {
			return fmt.Errorf("host '%s' not found", name)
		}

		fmt.Printf("Testing connection to %s...\n", name)
		fmt.Printf("  Host: %s:%d\n", h.Host, h.Port)
		fmt.Printf("  User: %s\n", h.User)

		// Create client
		cfg := ssh.ConfigFromHostAlias(&h)
		cfg.Timeout = sshTimeout

		client, err := ssh.NewClient(cfg)
		if err != nil {
			fmt.Printf("  ✗ Connection failed: %v\n", err)
			return err
		}
		defer client.Close()

		fmt.Println("  ✓ Connected successfully")

		// Test command execution
		ctx := context.Background()
		stdout, _, err := client.Exec(ctx, "echo 'Hello from cicerone'")
		if err != nil {
			fmt.Printf("  ✗ Command test failed: %v\n", err)
			return err
		}

		fmt.Printf("  ✓ Command test passed: %s", string(stdout))

		return nil
	},
}

var sshExecCmd = &cobra.Command{
	Use:   "exec <name> <command>",
	Short: "Execute command on remote host",
	Long: `Execute a command on a remote SSH host.

Examples:
  cicerone ssh exec darth "ls -la"
  cicerone ssh exec darth "cat /etc/os-release"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		command := strings.Join(args[1:], " ")

		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		h, exists := hosts[name]
		if !exists {
			return fmt.Errorf("host '%s' not found", name)
		}

		// Create client
		cfg := ssh.ConfigFromHostAlias(&h)
		cfg.Timeout = sshTimeout

		client, err := ssh.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer client.Close()

		// Execute
		ctx := context.Background()
		stdout, stderr, err := client.Exec(ctx, command)

		// Print output
		if len(stdout) > 0 {
			fmt.Print(string(stdout))
		}
		if len(stderr) > 0 {
			fmt.Fprint(os.Stderr, string(stderr))
		}

		if err != nil {
			return fmt.Errorf("command failed: %w", err)
		}

		return nil
	},
}

var sshShellCmd = &cobra.Command{
	Use:   "shell <name>",
	Short: "Start interactive shell",
	Long:  `Start an interactive SSH shell on a remote host.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		h, exists := hosts[name]
		if !exists {
			return fmt.Errorf("host '%s' not found", name)
		}

		// Create client
		cfg := ssh.ConfigFromHostAlias(&h)
		cfg.Timeout = sshTimeout

		client, err := ssh.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer client.Close()

		fmt.Printf("Connected to %s (%s)\n", name, h.Host)
		fmt.Println("Press Ctrl+D or type 'exit' to disconnect")
		fmt.Println()

		// Start shell
		return client.Shell()
	},
}

var sshPushCmd = &cobra.Command{
	Use:   "push <name> <local> <remote>",
	Short: "Push file to remote host",
	Long: `Push a local file to a remote SSH host.

Example:
  cicerone ssh push darth ./local.txt /home/wez/remote.txt`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		localPath := args[1]
		remotePath := args[2]

		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		h, exists := hosts[name]
		if !exists {
			return fmt.Errorf("host '%s' not found", name)
		}

		// Create client
		cfg := ssh.ConfigFromHostAlias(&h)
		cfg.Timeout = sshTimeout

		client, err := ssh.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer client.Close()

		fmt.Printf("Pushing %s -> %s:%s\n", localPath, name, remotePath)

		if err := client.CopyFile(localPath, remotePath, true); err != nil {
			return fmt.Errorf("push failed: %w", err)
		}

		fmt.Println("✓ File transferred successfully")
		return nil
	},
}

var sshPullCmd = &cobra.Command{
	Use:   "pull <name> <remote> <local>",
	Short: "Pull file from remote host",
	Long: `Pull a file from a remote SSH host.

Example:
  cicerone ssh pull darth /home/wez/remote.txt ./local.txt`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		remotePath := args[1]
		localPath := args[2]

		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		h, exists := hosts[name]
		if !exists {
			return fmt.Errorf("host '%s' not found", name)
		}

		// Create client
		cfg := ssh.ConfigFromHostAlias(&h)
		cfg.Timeout = sshTimeout

		client, err := ssh.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer client.Close()

		fmt.Printf("Pulling %s:%s -> %s\n", name, remotePath, localPath)

		if err := client.CopyFile(localPath, remotePath, false); err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}

		fmt.Println("✓ File transferred successfully")
		return nil
	},
}

var sshRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove SSH host",
	Long:  `Remove an SSH host from configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		hosts, err := loadSSHHosts()
		if err != nil {
			return fmt.Errorf("failed to load hosts: %w", err)
		}

		if _, exists := hosts[name]; !exists {
			return fmt.Errorf("host '%s' not found", name)
		}

		delete(hosts, name)

		if err := saveSSHHosts(hosts); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}

		fmt.Printf("✓ Removed SSH host '%s'\n", name)
		return nil
	},
}

// SSH hosts file path
func sshHostsPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.cicerone/ssh_hosts.yaml"
}

// Load SSH hosts from file
func loadSSHHosts() (map[string]ssh.HostAlias, error) {
	hosts := make(map[string]ssh.HostAlias)

	path := sshHostsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hosts, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}

// Save SSH hosts to file
func saveSSHHosts(hosts map[string]ssh.HostAlias) error {
	path := sshHostsPath()

	// Ensure directory exists
	dir := path[:strings.LastIndex(path, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(hosts)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}