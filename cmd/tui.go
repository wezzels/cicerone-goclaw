package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/llm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// Don't use cobra command - implement chat directly to avoid stdin conflicts
	// and screen clearing issues
	
	fmt.Println("\nStarting LLM chat...")
	fmt.Println()
	
	// Import the chat logic directly
	err := runChatSession(reader)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
	}
	return err
}

func runChatSession(reader *bufio.Reader) error {
	// Get model from config
	model := viper.GetString("llm.model")
	if model == "" {
		model = "gemma3:12b"
	}

	// Get provider URL from config
	baseURL := viper.GetString("llm.base_url")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Get timeout
	timeout := viper.GetInt("llm.timeout")
	if timeout == 0 {
		timeout = 120
	}

	// Create provider
	cfg := &llm.Config{
		BaseURL: baseURL,
		Model:   model,
		Timeout: timeout,
	}
	provider := llm.NewProvider(cfg)

	// Check if provider is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !provider.IsRunning(ctx) {
		fmt.Printf("Error: LLM provider not running at %s\n", baseURL)
		fmt.Println()
		fmt.Println("To start Ollama:")
		fmt.Println("  ollama serve")
		fmt.Println()
		fmt.Println("To start llama.cpp:")
		fmt.Println("  llama-server --model <model.gguf> --port 8080")
		return fmt.Errorf("LLM provider not available")
	}

	fmt.Printf("Connected to LLM at %s\n", baseURL)
	fmt.Printf("Model: %s\n", model)
	fmt.Println("Type 'exit' to quit, 'clear' to reset history, 'history' to view")
	fmt.Println()

	messages := []llm.Message{}

	// Add system prompt if configured
	systemPrompt := viper.GetString("llm.system_prompt")
	if systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		switch input {
		case "exit", "quit", "q":
			fmt.Println("\nGoodbye!")
			return nil
		case "clear":
			messages = messages[:0]
			if systemPrompt != "" {
				messages = append(messages, llm.Message{
					Role:    "system",
					Content: systemPrompt,
				})
			}
			fmt.Println("History cleared.")
			fmt.Println()
			continue
		case "history":
			if len(messages) == 0 || (len(messages) == 1 && messages[0].Role == "system") {
				fmt.Println("No history yet.")
			} else {
				fmt.Println("\nConversation History:")
				fmt.Println(strings.Repeat("-", 40))
				for _, msg := range messages {
					if msg.Role == "system" {
						continue
					}
					fmt.Printf("%s: %s\n", strings.Title(msg.Role), msg.Content)
				}
				fmt.Println(strings.Repeat("-", 40))
			}
			fmt.Println()
			continue
		}

		// Add user message
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: input,
		})

		// Send to LLM
		fmt.Print("\nAssistant: ")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

		response, respErr := tuiStreamChat(ctx, provider, messages)
		cancel()

		if respErr != nil {
			fmt.Printf("\nError: %v\n\n", respErr)
			// Remove failed message
			messages = messages[:len(messages)-1]
			continue
		}

		fmt.Println()

		// Add assistant response to history
		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: response,
		})
	}
}

func tuiStreamChat(ctx context.Context, provider llm.Provider, messages []llm.Message) (string, error) {
	stream, err := provider.ChatStream(ctx, messages)
	if err != nil {
		return "", err
	}

	var response string
	for chunk := range stream {
		if chunk.Error != nil {
			return response, chunk.Error
		}
		fmt.Print(chunk.Text)
		response += chunk.Text
		if chunk.Done {
			break
		}
	}

	return response, nil
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
	fmt.Println()
	
	// Call the status function directly instead of through cobra
	err := runGatewayStatus(nil, nil)
	
	// Wait for user to read before returning to menu
	fmt.Println()
	fmt.Print("Press Enter to continue...")
	reader.ReadString('\n')
	
	return err
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
			fmt.Println(strings.Repeat("=", 50))
			runConfigShow(nil, nil)
			fmt.Println()
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
		case "2":
			// Set config value
			fmt.Println("\nSet Configuration Value")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Print("Enter key (e.g., telegram.bot_token): ")
			key, _ := reader.ReadString('\n')
			key = strings.TrimSpace(key)
			
			if key == "" {
				fmt.Println("Cancelled.")
				fmt.Print("Press Enter to continue...")
				reader.ReadString('\n')
				continue
			}
			
			fmt.Print("Enter value: ")
			value, _ := reader.ReadString('\n')
			value = strings.TrimSpace(value)
			
			if err := runConfigSet(nil, []string{key, value}); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println()
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
		case "3":
			// Run wizard
			fmt.Println("\nConfiguration Wizard")
			fmt.Println(strings.Repeat("=", 50))
			wizardCmd.SetArgs([]string{})
			if err := wizardCmd.Execute(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println()
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
		case "4":
			// Edit config file
			fmt.Println("\nEdit Config File")
			fmt.Println(strings.Repeat("=", 50))
			configPath := os.ExpandEnv("$HOME/.cicerone/config.yaml")
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Println("Edit manually with: " + editor + " " + configPath)
			fmt.Println()
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
		case "b", "B", "back":
			return nil
		case "q", "Q", "quit", "exit":
			return nil
		default:
			fmt.Printf("\nUnknown choice: %s\n", choice)
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
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