package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/vm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manage VM workspaces",
	Long: `Manage VM-based workspaces for isolated development and execution.

VMs are managed via libvirt/QEMU and accessed via SSH. The deploy command
provides lifecycle management, file transfer, and SSH access.

Examples:
  cicerone deploy list
  cicerone deploy create dev
  cicerone deploy start dev
  cicerone deploy shell dev
  cicerone deploy exec dev "ls -la /workspace"`,
}

var (
	deployForce bool
)

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployListCmd)
	deployCmd.AddCommand(deployCreateCmd)
	deployCmd.AddCommand(deployStartCmd)
	deployCmd.AddCommand(deployStopCmd)
	deployCmd.AddCommand(deployRestartCmd)
	deployCmd.AddCommand(deployShellCmd)
	deployCmd.AddCommand(deployExecCmd)
	deployCmd.AddCommand(deployPushCmd)
	deployCmd.AddCommand(deployPullCmd)
	deployCmd.AddCommand(deployKeysCmd)
	deployCmd.AddCommand(deploySnapshotCmd)
	deployCmd.AddCommand(deployWorkspaceCmd)
	deployCmd.AddCommand(deployStatusCmd)

	// Flags for create
	deployCreateCmd.Flags().IntP("memory", "m", 0, "Memory in MB (default 2048)")
	deployCreateCmd.Flags().IntP("vcpus", "c", 0, "Number of vCPUs (default 2)")
	deployCreateCmd.Flags().StringP("image", "i", "", "Path to base image")
	deployCreateCmd.Flags().StringP("network", "n", "default", "Network name")
	deployCreateCmd.Flags().Bool("autostart", false, "Start VM after creation")

	// Flags for stop
	deployStopCmd.Flags().BoolVarP(&deployForce, "force", "f", false, "Force stop (destroy)")

	// Flags for exec
	deployExecCmd.Flags().Bool("tty", false, "Allocate TTY for interactive commands")

	// Flags for keys
	deployKeysCmd.Flags().Bool("generate", false, "Generate new key pair")
	deployKeysCmd.Flags().Bool("deploy", false, "Deploy key to VM")
	deployKeysCmd.Flags().Bool("status", false, "Show key status")
	deployKeysCmd.Flags().String("key", "", "Path to existing SSH key")

	// Flags for snapshot
	deploySnapshotCmd.Flags().Bool("create", false, "Create snapshot")
	deploySnapshotCmd.Flags().Bool("list", false, "List snapshots")
	deploySnapshotCmd.Flags().Bool("revert", false, "Revert to snapshot")
	deploySnapshotCmd.Flags().Bool("delete", false, "Delete snapshot")
	deploySnapshotCmd.Flags().String("name", "", "Snapshot name")
	deploySnapshotCmd.Flags().String("description", "", "Snapshot description")
}

// List VMs
var deployListCmd = &cobra.Command{
	Use:   "list",
	Short: "List VM workspaces",
	Long:  `List all configured and running VM workspaces.`,
	RunE:  runDeployList,
}

func runDeployList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := vm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Try to connect to libvirt
	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		// Show configured VMs without runtime status
		return listConfiguredVMs(cfg)
	}
	defer mgr.Close()

	// Get runtime status for VMs
	infos, err := mgr.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	fmt.Println("VM Workspaces")
	fmt.Println("=============")
	fmt.Println()

	if len(infos) == 0 {
		fmt.Println("No VMs found.")
		fmt.Println()
		fmt.Println("Create a VM with: cicerone deploy create <name>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tIP\tMEMORY\tVCPUS")
	for _, info := range infos {
		fmt.Fprintf(w, "%s\t%s\t%s\t%dMB\t%d\n",
			info.Name,
			info.State,
			info.IP,
			info.Memory,
			info.VCPUs,
		)
	}
	w.Flush()

	// Show configured VMs not running
	if len(cfg.VMs) > len(infos) {
		fmt.Println()
		fmt.Println("Configured VMs (not running):")
		for name, vmCfg := range cfg.VMs {
			found := false
			for _, info := range infos {
				if info.Name == vmCfg.Name || info.Name == name {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("  %s (%s) - %s\n", name, vmCfg.Name, vmCfg.Image)
			}
		}
	}

	return nil
}

func listConfiguredVMs(cfg *vm.Config) error {
	fmt.Println("VM Workspaces (Configured)")
	fmt.Println("=========================")
	fmt.Println()

	if len(cfg.VMs) == 0 {
		fmt.Println("No VMs configured.")
		fmt.Println()
		fmt.Println("Add VMs to ~/.cicerone/config.yaml under 'vms:' key.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tIMAGE\tMEMORY\tVCPUS\tDESCRIPTION")
	for name, vmCfg := range cfg.VMs {
		mem := vmCfg.Memory
		if mem == 0 {
			mem = 2048
		}
		vcpus := vmCfg.VCPUs
		if vcpus == 0 {
			vcpus = 2
		}
		fmt.Fprintf(w, "%s\t%s\t%dMB\t%d\t%s\n",
			name,
			vmCfg.Image,
			mem,
			vcpus,
			vmCfg.Description,
		)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Note: libvirt connection unavailable. Install libvirt-dev for runtime status.")
	return nil
}

// Create VM
var deployCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a VM workspace",
	Long: `Create a new VM from a base image.

The VM is defined in libvirt but not started unless --autostart is specified.
Use 'cicerone deploy start' to start the VM after creation.`,
	Args: cobra.ExactArgs(1),
	RunE: runDeployCreate,
}

func runDeployCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config to check for predefined VM
	cfg, err := vm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Build VM config
	var vmCfg *vm.VMConfig

	// Check if VM is defined in config
	if vmFile, ok := cfg.VMs[name]; ok {
		vmCfg = vmFile.MergeWithDefaults()
	} else {
		// Create from command line flags
		memory, _ := cmd.Flags().GetInt("memory")
		vcpus, _ := cmd.Flags().GetInt("vcpus")
		image, _ := cmd.Flags().GetString("image")
		network, _ := cmd.Flags().GetString("network")
		autostart, _ := cmd.Flags().GetBool("autostart")

		if image == "" {
			return fmt.Errorf("image path required (use -i flag)")
		}

		vmCfg = vm.DefaultVMConfig()
		vmCfg.Name = name
		vmCfg.Image = image
		vmCfg.Network = network
		vmCfg.AutoStart = autostart

		if memory > 0 {
			vmCfg.Memory = memory
		}
		if vcpus > 0 {
			vmCfg.VCPUs = vcpus
		}
	}

	// Validate
	if err := vmCfg.Validate(); err != nil {
		return err
	}

	// Connect to libvirt
	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	// Create VM
	fmt.Printf("Creating VM '%s'...\n", vmCfg.Name)
	info, err := mgr.Create(context.Background(), vmCfg)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	fmt.Printf("✓ VM '%s' created\n", info.Name)
	if info.State == vm.StateRunning {
		fmt.Printf("  State: running\n")
		fmt.Printf("  IP: %s\n", info.IP)
	} else {
		fmt.Printf("  State: %s\n", info.State)
		fmt.Println("  Start with: cicerone deploy start", vmCfg.Name)
	}

	return nil
}

// Start VM
var deployStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a VM",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployStart,
}

func runDeployStart(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Starting VM '%s'...\n", name)
	if err := mgr.Start(context.Background(), name); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for IP
	info, err := mgr.Status(context.Background(), name)
	if err == nil && info.IP != "" {
		fmt.Printf("✓ VM '%s' started\n", name)
		fmt.Printf("  IP: %s\n", info.IP)
	} else {
		fmt.Printf("✓ VM '%s' started\n", name)
		fmt.Println("  Waiting for IP...")
	}

	return nil
}

// Stop VM
var deployStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a VM",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployStop,
}

func runDeployStop(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Stopping VM '%s'...\n", name)
	if err := mgr.Stop(context.Background(), name, deployForce); err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	fmt.Printf("✓ VM '%s' stopped\n", name)
	return nil
}

// Restart VM
var deployRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a VM",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployRestart,
}

func runDeployRestart(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Restarting VM '%s'...\n", name)
	if err := mgr.Restart(context.Background(), name); err != nil {
		return fmt.Errorf("failed to restart VM: %w", err)
	}

	fmt.Printf("✓ VM '%s' restarted\n", name)
	return nil
}

// VM Status
var deployStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show VM status",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployStatus,
}

func runDeployStatus(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	info, err := mgr.Status(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to get VM status: %w", err)
	}

	fmt.Printf("VM: %s\n", info.Name)
	fmt.Printf("State: %s\n", info.State)
	if info.IP != "" {
		fmt.Printf("IP: %s\n", info.IP)
	}
	if info.Memory > 0 {
		fmt.Printf("Memory: %d MB\n", info.Memory)
	}
	if info.VCPUs > 0 {
		fmt.Printf("vCPUs: %d\n", info.VCPUs)
	}
	if !info.CreatedAt.IsZero() {
		fmt.Printf("Created: %s\n", info.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// SSH Shell
var deployShellCmd = &cobra.Command{
	Use:   "shell <name>",
	Short: "Open SSH shell on VM",
	Long: `Open an interactive SSH shell on the VM.

Requires SSH key to be deployed. Use 'cicerone deploy keys --deploy' first.`,
	Args: cobra.ExactArgs(1),
	RunE:  runDeployShell,
}

func runDeployShell(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Connecting to VM '%s'...\n", name)
	if err := mgr.Shell(context.Background(), name); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}

// Execute command
var deployExecCmd = &cobra.Command{
	Use:   "exec <name> <command>",
	Short: "Execute command on VM",
	Long: `Execute a command on the VM via SSH.

The command runs non-interactively. Output is printed to stdout/stderr.`,
	Args: cobra.MinimumNArgs(2),
	RunE:  runDeployExec,
}

func runDeployExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := strings.Join(args[1:], " ")

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	stdout, stderr, err := mgr.Exec(context.Background(), name, command)
	if err != nil {
		if len(stderr) > 0 {
			fmt.Fprintf(os.Stderr, "%s", stderr)
		}
		return fmt.Errorf("command failed: %w", err)
	}

	if len(stdout) > 0 {
		fmt.Printf("%s", stdout)
	}
	if len(stderr) > 0 {
		fmt.Fprintf(os.Stderr, "%s", stderr)
	}

	return nil
}

// Push file
var deployPushCmd = &cobra.Command{
	Use:   "push <name> <local> <remote>",
	Short: "Push file to VM",
	Long: `Copy a local file to the VM via SFTP.

Example:
  cicerone deploy push dev ./config.yaml /etc/app/config.yaml`,
	Args: cobra.ExactArgs(3),
	RunE: runDeployPush,
}

func runDeployPush(cmd *cobra.Command, args []string) error {
	name := args[0]
	localPath := args[1]
	remotePath := args[2]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Pushing %s -> %s:%s...\n", localPath, name, remotePath)
	if err := mgr.Push(context.Background(), name, localPath, remotePath); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Println("✓ File transferred")
	return nil
}

// Pull file
var deployPullCmd = &cobra.Command{
	Use:   "pull <name> <remote> <local>",
	Short: "Pull file from VM",
	Long: `Copy a file from the VM to local via SFTP.

Example:
  cicerone deploy pull dev /var/log/app.log ./app.log`,
	Args: cobra.ExactArgs(3),
	RunE: runDeployPull,
}

func runDeployPull(cmd *cobra.Command, args []string) error {
	name := args[0]
	remotePath := args[1]
	localPath := args[2]

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Pulling %s:%s -> %s...\n", name, remotePath, localPath)
	if err := mgr.Pull(context.Background(), name, remotePath, localPath); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	fmt.Println("✓ File transferred")
	return nil
}

// SSH Keys
var deployKeysCmd = &cobra.Command{
	Use:   "keys <name>",
	Short: "Manage SSH keys for VM",
	Long: `Manage SSH keys for VM access.

Generate and deploy SSH keys for passwordless access to VMs.

Examples:
  cicerone deploy keys dev --generate     # Generate new key pair
  cicerone deploy keys dev --status       # Show key status
  cicerone deploy keys dev --deploy       # Deploy key to VM`,
	Args: cobra.ExactArgs(1),
	RunE: runDeployKeys,
}

func runDeployKeys(cmd *cobra.Command, args []string) error {
	name := args[0]

	generate, _ := cmd.Flags().GetBool("generate")
	deploy, _ := cmd.Flags().GetBool("deploy")
	status, _ := cmd.Flags().GetBool("status")
	keyPath, _ := cmd.Flags().GetString("key")

	keyMgr, err := vm.NewKeyManager()
	if err != nil {
		return fmt.Errorf("failed to create key manager: %w", err)
	}

	// Default action: show status
	if !generate && !deploy && keyPath == "" {
		status = true
	}

	if status {
		if keyMgr.KeyExists(name) {
			info, err := keyMgr.GetKeyInfo(name)
			if err != nil {
				return err
			}
			fmt.Printf("SSH Key for VM '%s':\n", name)
			fmt.Printf("  Private key: %s\n", info.PrivateKeyPath)
			fmt.Printf("  Public key:  %s\n", info.PublicKeyPath)
			fmt.Printf("  Comment:     %s\n", info.Comment)
		} else {
			fmt.Printf("No SSH key configured for VM '%s'\n", name)
			fmt.Println("Generate with: cicerone deploy keys", name, "--generate")
		}
		return nil
	}

	if generate {
		comment := fmt.Sprintf("cicerone-%s", name)
		info, err := keyMgr.GenerateKey(name, comment)
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
		fmt.Printf("✓ Generated SSH key for VM '%s'\n", name)
		fmt.Printf("  Private key: %s\n", info.PrivateKeyPath)
		fmt.Printf("  Public key:  %s\n", info.PublicKeyPath)
		fmt.Println()
		fmt.Println("Deploy with: cicerone deploy keys", name, "--deploy")
		return nil
	}

	if deploy {
		// Get VM info for deployment
		mgr, err := vm.NewLibvirtManager(nil)
		if err != nil {
			return fmt.Errorf("failed to connect to libvirt: %w", err)
		}
		defer mgr.Close()

		info, err := mgr.Status(context.Background(), name)
		if err != nil {
			return fmt.Errorf("failed to get VM status: %w", err)
		}

		if info.State != vm.StateRunning {
			return fmt.Errorf("VM '%s' is not running", name)
		}

		if info.IP == "" {
			return fmt.Errorf("VM '%s' has no IP address", name)
		}

		// Get or generate key
		keyInfo, _, err := keyMgr.SetupKeyForVM(context.Background(), name, true)
		if err != nil {
			return fmt.Errorf("failed to setup key: %w", err)
		}

		// Deploy key
		if err := keyMgr.AddToKnownHosts(info.IP, 22); err != nil {
			fmt.Printf("Warning: failed to add to known_hosts: %v\n", err)
		}

		// Add to SSH config for easy access
		if err := keyMgr.AddToSSHConfig(name, info.IP, 22, keyInfo.PrivateKeyPath); err != nil {
			fmt.Printf("Warning: failed to add to SSH config: %v\n", err)
		}

		fmt.Printf("✓ SSH key configured for VM '%s'\n", name)
		fmt.Printf("  Connect with: ssh %s\n", name)
		return nil
	}

	return nil
}

// Snapshots
var deploySnapshotCmd = &cobra.Command{
	Use:   "snapshot <name>",
	Short: "Manage VM snapshots",
	Long: `Manage VM snapshots.

Examples:
  cicerone deploy snapshot dev --create --name "before-tests"
  cicerone deploy snapshot dev --list
  cicerone deploy snapshot dev --revert --name "before-tests"`,
	Args: cobra.ExactArgs(1),
	RunE: runDeploySnapshot,
}

func runDeploySnapshot(cmd *cobra.Command, args []string) error {
	name := args[0]

	create, _ := cmd.Flags().GetBool("create")
	list, _ := cmd.Flags().GetBool("list")
	revert, _ := cmd.Flags().GetBool("revert")
	delete, _ := cmd.Flags().GetBool("delete")
	snapName, _ := cmd.Flags().GetString("name")
	snapDesc, _ := cmd.Flags().GetString("description")

	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	// Default action: list
	if !create && !revert && !delete {
		list = true
	}

	if list {
		snapshots, err := mgr.SnapshotList(context.Background(), name)
		if err != nil {
			return fmt.Errorf("failed to list snapshots: %w", err)
		}

		if len(snapshots) == 0 {
			fmt.Printf("No snapshots for VM '%s'\n", name)
			return nil
		}

		fmt.Printf("Snapshots for VM '%s':\n", name)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCREATED\tCURRENT")
		for _, snap := range snapshots {
			current := ""
			if snap.Current {
				current = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				snap.Name,
				snap.CreatedAt.Format("2006-01-02 15:04"),
				current,
			)
		}
		w.Flush()
		return nil
	}

	if create {
		if snapName == "" {
			return fmt.Errorf("snapshot name required (use --name)")
		}
		fmt.Printf("Creating snapshot '%s' for VM '%s'...\n", snapName, name)
		if err := mgr.Snapshot(context.Background(), name, snapName, snapDesc); err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
		fmt.Printf("✓ Snapshot '%s' created\n", snapName)
		return nil
	}

	if revert {
		if snapName == "" {
			return fmt.Errorf("snapshot name required (use --name)")
		}
		fmt.Printf("Reverting VM '%s' to snapshot '%s'...\n", name, snapName)
		if err := mgr.SnapshotRevert(context.Background(), name, snapName); err != nil {
			return fmt.Errorf("failed to revert snapshot: %w", err)
		}
		fmt.Printf("✓ Reverted to snapshot '%s'\n", snapName)
		return nil
	}

	if delete {
		if snapName == "" {
			return fmt.Errorf("snapshot name required (use --name)")
		}
		fmt.Printf("Deleting snapshot '%s' for VM '%s'...\n", snapName, name)
		if err := mgr.SnapshotDelete(context.Background(), name, snapName); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}
		fmt.Printf("✓ Snapshot '%s' deleted\n", snapName)
		return nil
	}

	return nil
}

// Workspace switch
var deployWorkspaceCmd = &cobra.Command{
	Use:   "workspace [name]",
	Short: "Set active VM workspace",
	Long: `Set the active VM workspace for agent execution.

When a VM workspace is active, 'cicerone do' commands run on the VM
instead of the local machine.

Use 'local' to switch back to local execution.`,
	Args: cobra.MaximumNArgs(1),
	RunE:  runDeployWorkspace,
}

func runDeployWorkspace(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show current workspace
		current := viper.GetString("active_workspace")
		if current == "" {
			fmt.Println("Active workspace: local")
		} else {
			fmt.Printf("Active workspace: %s\n", current)
		}
		return nil
	}

	name := args[0]

	if name == "local" {
		viper.Set("active_workspace", "")
		fmt.Println("✓ Switched to local workspace")
		return nil
	}

	// Verify VM exists
	mgr, err := vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer mgr.Close()

	exists, err := mgr.Exists(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to check VM: %w", err)
	}

	if !exists {
		// Check if it's configured
		cfg, _ := vm.LoadConfig()
		if _, ok := cfg.VMs[name]; !ok {
			return fmt.Errorf("VM '%s' not found", name)
		}
	}

	viper.Set("active_workspace", name)
	fmt.Printf("✓ Active workspace: %s\n", name)

	return nil
}