package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
		clearScreen()
		printMenu()

		fmt.Print("\nChoice: ")
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			runTelegramInteractive()
		case "2":
			runChatInteractive()
		case "3":
			runDoctorInteractive()
		case "4":
			runSecurityInteractive()
		case "5":
			runGatewayRestartInteractive()
		case "6":
			runGatewayStatusInteractive()
		case "7":
			runSettingsInteractive()
		case "q", "Q", "quit", "exit":
			fmt.Println("\nGoodbye!")
			return nil
		default:
			fmt.Printf("\nUnknown choice: %s\n", choice)
			waitForEnter(reader)
		}
	}
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
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

func runTelegramInteractive() {
	fmt.Println("\nStarting Telegram bot...")
	exec.Command("cicerone", "telegram").Run()
}

func runChatInteractive() {
	fmt.Println("\nStarting LLM chat...")
	exec.Command("cicerone", "chat").Run()
}

func runDoctorInteractive() {
	fmt.Println("\nRunning health diagnostics...")
	exec.Command("cicerone", "doctor").Run()
}

func runSecurityInteractive() {
	fmt.Println("\nRunning security audit...")
	exec.Command("cicerone", "security").Run()
}

func runGatewayRestartInteractive() {
	fmt.Println("\nRestarting gateway...")
	exec.Command("cicerone", "gateway", "restart").Run()
}

func runGatewayStatusInteractive() {
	fmt.Println("\nGateway status...")
	exec.Command("cicerone", "gateway", "status").Run()
}

func runSettingsInteractive() {
	fmt.Println("\nSettings menu not yet implemented")
}