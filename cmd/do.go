package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// doCmd represents the do command
var doCmd = &cobra.Command{
	Use:   "do [instructions]",
	Short: "Execute instructions via LLM",
	Long: `Execute instructions using the configured LLM.

The LLM will interpret natural language instructions and:
  - Generate commands to execute
  - Run them on configured nodes
  - Return results

Examples:
  cicerone do "list files in /tmp"
  cicerone do "check disk space on darth"
  cicerone do "get status of forge-c2 pod"`,
	RunE: runDo,
	Args: cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(doCmd)
}

func runDo(cmd *cobra.Command, args []string) error {
	instructions := args[0]
	if instructions == "" {
		return fmt.Errorf("instructions required")
	}

	fmt.Printf("Executing: %s\n\n", instructions)

	// TODO: Implement LLM-based execution
	// This should:
	// 1. Send instructions to LLM
	// 2. Parse command interpretation
	// 3. Execute on node (local or remote)
	// 4. Return results

	fmt.Println("TODO: LLM execution not yet implemented")
	fmt.Println("Use 'cicerone chat' for interactive LLM session")

	return nil
}