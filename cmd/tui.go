package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	Long: `Launch an interactive terminal user interface.

Provides a menu-driven interface for:
  - Starting Telegram bot
  - Starting LLM chat
  - Running health checks
  - Running security audit
  - Gateway management
  - Settings`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Clear screen using ANSI escape codes
		fmt.Print("\033[2J\033[H")
		
		printMenu()

		fmt.Print("\nChoice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			if err := runTelegramInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "2":
			if err := runChatInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "3":
			if err := runDoctorInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "4":
			if err := runSecurityInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "5":
			if err := runGatewayRestartInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "6":
			if err := runGatewayStatusInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "7":
			runSettingsInteractive(reader)
		case "q", "Q", "quit", "exit":
			fmt.Println("\nGoodbye!")
			return nil
		default:
			fmt.Printf("\nUnknown choice: %s\n", choice)
			waitForEnter(reader)
		}
	}
}

func printMenu() {
	fmt.Println(`
╔═══════════════════════════════════════╗
║           CICERONE TUI                ║
╠═══════════════════════════════════════╣
║                                       ║
║  1. Start Telegram Bot                ║
║  2. Start LLM Chat                    ║
║  3. Run Doctor                        ║
║  4. Run Security Audit                ║
║  5. Restart Gateway                   ║
║  6. View Gateway Status               ║
║  7. Settings                          ║
║                                       ║
║  Q. Quit                              ║
╚═══════════════════════════════════════╝`)
}

func waitForEnter(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to continue...")
	reader.ReadString('\n')
}

func runTelegramInteractive(reader *bufio.Reader) error {
	fmt.Println("\nStarting Telegram bot...")
	fmt.Println("Press Ctrl+C to stop")
	
	// Get the telegram command and execute it
	telegramCmd.SetArgs([]string{})
	return telegramCmd.Execute()
}

func runChatInteractive(reader *bufio.Reader) error {
	fmt.Println("\nStarting LLM chat...")
	chatCmd.SetArgs([]string{})
	return chatCmd.Execute()
}

func runDoctorInteractive(reader *bufio.Reader) error {
	fmt.Println("\nRunning health diagnostics...")
	doctorCmd.SetArgs([]string{})
	return doctorCmd.Execute()
}

func runSecurityInteractive(reader *bufio.Reader) error {
	fmt.Println("\nRunning security audit...")
	securityCmd.SetArgs([]string{})
	return securityCmd.Execute()
}

func runGatewayRestartInteractive(reader *bufio.Reader) error {
	fmt.Println("\nRestarting gateway...")
	gatewayCmd.SetArgs([]string{"restart"})
	return gatewayCmd.Execute()
}

func runGatewayStatusInteractive(reader *bufio.Reader) error {
	fmt.Println("\nGateway status...")
	gatewayCmd.SetArgs([]string{"status"})
	return gatewayCmd.Execute()
}

func runSettingsInteractive(reader *bufio.Reader) {
	fmt.Println("\nSettings menu not yet implemented")
	fmt.Println("\nUse 'cicerone config show' to view settings")
	fmt.Println("Use 'cicerone config set <key> <value>' to change settings")
}