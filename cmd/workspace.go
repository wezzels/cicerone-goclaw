package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/workspace"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage code workspaces",
	Long: `Manage code workspaces for execution and testing.

Workspaces provide isolated environments for running code, tests, and commands.

Examples:
  cicerone workspace init                    # Initialize in current directory
  cicerone workspace init /path/to/project   # Initialize at specific path
  cicerone workspace status                  # Show workspace info
  cicerone workspace list                    # List all workspaces
  cicerone workspace clean                  # Clean workspace files`,
}

var (
	workspacePath string
)

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(workspaceInitCmd)
	workspaceCmd.AddCommand(workspaceStatusCmd)
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceCleanCmd)
}

var workspaceInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a workspace",
	Long: `Initialize a new workspace for code execution.

Creates the workspace directory structure:
  src/    - Source files
  build/  - Build artifacts
  logs/   - Execution logs
  tmp/    - Temporary files`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		w, err := workspace.New(abs)
		if err != nil {
			return fmt.Errorf("failed to create workspace: %w", err)
		}

		if err := w.Init(); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		fmt.Printf("✓ Workspace initialized at %s\n", abs)
		fmt.Println()
		fmt.Println("Created directories:")
		fmt.Println("  src/    - Source files")
		fmt.Println("  build/  - Build artifacts")
		fmt.Println("  logs/   - Execution logs")
		fmt.Println("  tmp/    - Temporary files")
		fmt.Println()
		fmt.Printf("Run 'cicerone exec' to execute commands\n")

		return nil
	},
}

var workspaceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show workspace status",
	Long:  `Display information about the current or specified workspace.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		if !workspace.IsWorkspace(abs) {
			fmt.Printf("Not a cicerone workspace: %s\n", abs)
			fmt.Println("Run 'cicerone workspace init' to initialize")
			return nil
		}

		w, err := workspace.New(abs)
		if err != nil {
			return err
		}

		fmt.Println("Workspace Status")
		fmt.Println("===============")
		fmt.Println()
		fmt.Printf("  Path:     %s\n", w.Path)

		// Check directories
		dirs := []string{"src", "build", "logs", "tmp"}
		fmt.Println("  Directories:")
		for _, dir := range dirs {
			fullPath := filepath.Join(w.Path, dir)
			if _, err := os.Stat(fullPath); err == nil {
				fmt.Printf("    ✓ %s/\n", dir)
			} else {
				fmt.Printf("    ✗ %s/ (missing)\n", dir)
			}
		}

		// Count files
		fmt.Println()
		fmt.Println("  Files:")
		count := countFiles(w.Path)
		fmt.Printf("    %d files total\n", count)

		return nil
	},
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces",
	Long:  `List all cicerone workspaces in a directory tree.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		searchPath := "."
		if len(args) > 0 {
			searchPath = args[0]
		}

		abs, err := filepath.Abs(searchPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		fmt.Println("Searching for workspaces in", abs)
		fmt.Println()

		var found []string
		filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.Name() == ".workspace" {
				found = append(found, filepath.Dir(path))
			}
			return nil
		})

		if len(found) == 0 {
			fmt.Println("No workspaces found")
			return nil
		}

		fmt.Printf("Found %d workspace(s):\n", len(found))
		for _, p := range found {
			fmt.Printf("  %s\n", p)
		}

		return nil
	},
}

var workspaceCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean workspace",
	Long:  `Remove all files from workspace except .workspace marker.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		if !workspace.IsWorkspace(abs) {
			return fmt.Errorf("not a workspace: %s", abs)
		}

		w, err := workspace.New(abs)
		if err != nil {
			return err
		}

		fmt.Printf("Cleaning workspace: %s\n", abs)
		if err := w.Clean(); err != nil {
			return fmt.Errorf("clean failed: %w", err)
		}

		fmt.Println("✓ Workspace cleaned")
		return nil
	},
}

func countFiles(path string) int {
	count := 0
	filepath.Walk(path, func(_ string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}