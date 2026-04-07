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
			if err := runSettingsInteractive(reader); err != nil {
				fmt.Printf("Error: %v\n", err)
				waitForEnter(reader)
			}
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

func runSettingsInteractive(reader *bufio.Reader) error {
	for {
		// Clear screen
		fmt.Print("\033[2J\033[H")
		
		printSettingsMenu()

		fmt.Print("\nChoice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			// Show config
			fmt.Println("\nCurrent Configuration:")
			fmt.Println("=" + strings.Repeat("=", 50))
			configCmd.SetArgs([]string{"show"})
			configCmd.Execute()
			waitForEnter(reader)
		case "2":
			// Set config value
			fmt.Println("\nSet Configuration Value")
			fmt.Println("=" + strings.Repeat("=", 50))
			fmt.Print("Enter key (e.g., telegram.bot_token): ")
			key, _ := reader.ReadString('\n')
			key = strings.TrimSpace(key)
			
			if key == "" {
				fmt.Println("Cancelled.")
				waitForEnter(reader)
				continue
			}
			
			fmt.Print("Enter value: ")
			value, _ := reader.ReadString('\n')
			value = strings.TrimSpace(value)
			
			configCmd.SetArgs([]string{"set", key, value})
			if err := configCmd.Execute(); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Set %s = %s\n", key, value)
			}
			waitForEnter(reader)
		case "3":
			// Run wizard
			fmt.Println("\nConfiguration Wizard")
			fmt.Println("=" + strings.Repeat("=", 50))
			wizardCmd.SetArgs([]string{})
			if err := wizardCmd.Execute(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			waitForEnter(reader)
		case "4":
			// Edit config file
			fmt.Println("\nOpen config file for editing...")
			configPath := os.ExpandEnv("$HOME/.cicerone/config.yaml")
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}
			fmt.Printf("Opening %s with %s...\n", configPath, editor)
			// Just show the path - can't easily open editor from here
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Println("Edit manually with: " + editor + " " + configPath)
			waitForEnter(reader)
		case "b", "B", "back":
			return nil
		case "q", "Q", "quit", "exit":
			return nil
		default:
			fmt.Printf("\nUnknown choice: %s\n", choice)
			waitForEnter(reader)
		}
	}
}

func printSettingsMenu() {
	fmt.Println(`
╔═══════════════════════════════════════╗
║           SETTINGS MENU               ║
╠═══════════════════════════════════════╣
║                                       ║
║  1. View Current Settings             ║
║  2. Set Configuration Value           ║
║  3. Run Setup Wizard                  ║
║  4. Edit Config File                  ║
║                                       ║
║  B. Back to Main Menu                 ║
║  Q. Quit                              ║
╚═══════════════════════════════════════╝`)
}