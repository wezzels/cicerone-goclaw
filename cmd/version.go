package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables (set via ldflags)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show cicerone version",
	Long:  `Display the version, commit hash, and build date.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cicerone version %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  date:   %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}