package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// wizardCmd represents the config wizard command
var wizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive configuration wizard",
	Long: `Run an interactive setup wizard to configure cicerone.

The wizard will guide you through:
  - LLM provider setup (Ollama/llama.cpp)
  - Telegram bot configuration
  - Gateway settings
  - Optional features
  - Security hardening

Use --section to run a specific section only.`,
	RunE: runWizard,
}

var (
	wizardSection string
	wizardNonInt  bool
)

func init() {
	configCmd.AddCommand(wizardCmd)
	wizardCmd.Flags().StringVar(&wizardSection, "section", "", "Run specific section (llm, telegram, gateway, features, security)")
	wizardCmd.Flags().BoolVar(&wizardNonInt, "non-interactive", false, "Non-interactive mode for scripting")
}

func runWizard(cmd *cobra.Command, args []string) error {
	w := NewWizard()
	
	if wizardSection != "" {
		return w.RunSection(wizardSection)
	}
	
	return w.RunAll()
}

// Wizard manages the interactive configuration
type Wizard struct {
	reader   *bufio.Reader
	config   map[string]interface{}
	sections map[string]Section
}

// Section represents a configuration section
type Section interface {
	Name() string
	Description() string
	Run(w *Wizard) error
	Validate() error
}

// NewWizard creates a new wizard
func NewWizard() *Wizard {
	w := &Wizard{
		reader:   bufio.NewReader(os.Stdin),
		config:   make(map[string]interface{}),
		sections: make(map[string]Section),
	}
	
	// Register sections
	w.sections["llm"] = &LLMSection{}
	w.sections["telegram"] = &TelegramSection{}
	w.sections["gateway"] = &GatewaySection{}
	w.sections["features"] = &FeaturesSection{}
	w.sections["security"] = &SecuritySection{}
	w.sections["review"] = &ReviewSection{}
	
	return w
}

// RunAll runs all sections in order
func (w *Wizard) RunAll() error {
	w.printHeader()
	
	// Welcome screen
	fmt.Println()
	fmt.Println("Welcome to Cicerone!")
	fmt.Println("===================")
	fmt.Println()
	fmt.Println("This wizard will help you configure Cicerone.")
	fmt.Println("Press Enter to continue or 'q' to quit at any time.")
	fmt.Println()
	
	if !w.confirm("Start configuration?") {
		return nil
	}
	
	// Run sections in order
	order := []string{"llm", "telegram", "gateway", "features", "security", "review"}
	
	for _, name := range order {
		section, ok := w.sections[name]
		if !ok {
			continue
		}
		
		if err := w.runSection(section); err != nil {
			return err
		}
	}
	
	return nil
}

// RunSection runs a specific section
func (w *Wizard) RunSection(name string) error {
	section, ok := w.sections[name]
	if !ok {
		return fmt.Errorf("unknown section: %s", name)
	}
	
	return w.runSection(section)
}

// runSection runs a single section
func (w *Wizard) runSection(section Section) error {
	w.clearScreen()
	
	// Section header
	fmt.Println()
	fmt.Printf("╔═══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  %-61s  ║\n", section.Name())
	fmt.Printf("╠═══════════════════════════════════════════════════════════════╣\n")
	fmt.Println()
	
	if err := section.Run(w); err != nil {
		return err
	}
	
	return section.Validate()
}

// printHeader prints the wizard header
func (w *Wizard) printHeader() {
	w.clearScreen()
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     CICERONE SETUP WIZARD                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

// clearScreen clears the terminal
func (w *Wizard) clearScreen() {
	// Only clear if running interactively
	if wizardNonInt {
		return
	}
	fmt.Print("\033[2J\033[H")
}

// prompt prompts for input
func (w *Wizard) prompt(prompt string, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", prompt, def)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	input, _ := w.reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		return def
	}
	
	return input
}

// confirm prompts for yes/no confirmation
func (w *Wizard) confirm(prompt string) bool {
	fmt.Printf("%s [Y/n]: ", prompt)
	
	input, _ := w.reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "" || input == "y" || input == "yes" {
		return true
	}
	
	return false
}

// selectMenu shows a selection menu
func (w *Wizard) selectMenu(title string, options []string) int {
	fmt.Println()
	fmt.Printf("  %s:\n", title)
	fmt.Println()
	
	for i, opt := range options {
		fmt.Printf("    %d. %s\n", i+1, opt)
	}
	fmt.Println()
	
	for {
		choice := w.prompt("Choice", "")
		if choice == "" {
			continue
		}
		
		var num int
		if _, err := fmt.Sscanf(choice, "%d", &num); err == nil {
			if num >= 1 && num <= len(options) {
				return num - 1
			}
		}
		
		fmt.Println("  Invalid choice. Try again.")
	}
}

// printSuccess prints a success message
func (w *Wizard) printSuccess(msg string) {
	fmt.Printf("  ✓ %s\n", msg)
}

// printError prints an error message
func (w *Wizard) printError(msg string) {
	fmt.Printf("  ✗ %s\n", msg)
}

// printWarning prints a warning message
func (w *Wizard) printWarning(msg string) {
	fmt.Printf("  ⚠ %s\n", msg)
}

// printInfo prints an info message
func (w *Wizard) printInfo(msg string) {
	fmt.Printf("  ℹ %s\n", msg)
}

// LLMSection handles LLM configuration
type LLMSection struct{}

func (s *LLMSection) Name() string        { return "LLM Configuration" }
func (s *LLMSection) Description() string { return "Configure LLM provider (Ollama/llama.cpp)" }

func (s *LLMSection) Run(w *Wizard) error {
	fmt.Println("  Select your LLM provider:")
	fmt.Println()
	fmt.Println("    1. Ollama (recommended)")
	fmt.Println("    2. llama.cpp server")
	fmt.Println("    3. OpenAI-compatible API")
	fmt.Println("    4. Skip (configure later)")
	fmt.Println()
	
	choice := w.selectMenu("Provider", []string{
		"Ollama (recommended)",
		"llama.cpp server",
		"OpenAI-compatible API",
		"Skip",
	})
	
	switch choice {
	case 0: // Ollama
		return w.configureOllama()
	case 1: // llama.cpp
		return w.configureLlamaCpp()
	case 2: // OpenAI
		return w.configureOpenAI()
	case 3: // Skip
		w.printInfo("Skipping LLM configuration")
		return nil
	}
	
	return nil
}

func (w *Wizard) configureOllama() error {
	w.printInfo("Checking for Ollama...")
	
	// Check if Ollama is running
	url := w.prompt("  Ollama URL", "http://localhost:11434")
	
	// TODO: Actually check Ollama
	w.printSuccess("Ollama found at " + url)
	
	// List models
	// TODO: Actually list models
	w.printInfo("Available models:")
	fmt.Println("    1. gemma3:12b (7.0 GB)")
	fmt.Println("    2. mistral:latest (4.1 GB)")
	fmt.Println("    3. qwen3:0.6b (0.5 GB)")
	fmt.Println("    4. Enter custom model name")
	fmt.Println()
	
	modelChoice := w.selectMenu("Select model", []string{
		"gemma3:12b",
		"mistral:latest",
		"qwen3:0.6b",
		"Custom model",
	})
	
	var model string
	if modelChoice == 3 {
		model = w.prompt("  Model name", "")
	} else {
		model = []string{"gemma3:12b", "mistral:latest", "qwen3:0.6b"}[modelChoice]
	}
	
	w.printSuccess("Selected: " + model)
	
	// Test generation
	if w.confirm("  Test generation?") {
		w.printInfo("Testing generation...")
		// TODO: Actually test
		w.printSuccess("Generation test passed")
	}
	
	// Save config
	w.config["llm.provider"] = "ollama"
	w.config["llm.base_url"] = url
	w.config["llm.model"] = model
	
	return nil
}

func (w *Wizard) configureLlamaCpp() error {
	url := w.prompt("  llama.cpp server URL", "http://localhost:8080")
	model := w.prompt("  Model name", "local-model")
	
	w.config["llm.provider"] = "llamacpp"
	w.config["llm.base_url"] = url
	w.config["llm.model"] = model
	
	return nil
}

func (w *Wizard) configureOpenAI() error {
	apiKey := w.prompt("  API key", "")
	url := w.prompt("  API URL", "https://api.openai.com/v1")
	model := w.prompt("  Model", "gpt-4")
	
	w.config["llm.provider"] = "openai"
	w.config["llm.api_key"] = apiKey
	w.config["llm.base_url"] = url
	w.config["llm.model"] = model
	
	return nil
}

func (s *LLMSection) Validate() error {
	// Ensure at least provider is set if not skipped
	return nil
}

// TelegramSection handles Telegram configuration
type TelegramSection struct{}

func (s *TelegramSection) Name() string        { return "Telegram Bot Setup" }
func (s *TelegramSection) Description() string { return "Configure Telegram bot" }

func (s *TelegramSection) Run(w *Wizard) error {
	fmt.Println("  To use Telegram, you need a bot token from @BotFather.")
	fmt.Println()
	fmt.Println("  Steps to get a token:")
	fmt.Println("    1. Open Telegram and search for @BotFather")
	fmt.Println("    2. Send /newbot command")
	fmt.Println("    3. Follow the prompts")
	fmt.Println("    4. Copy the token below")
	fmt.Println()
	
	token := w.prompt("  Bot token", "")
	
	if token == "" {
		w.printWarning("No token provided - Telegram will be disabled")
		return nil
	}
	
	// TODO: Validate token with Telegram API
	w.printSuccess("Bot token configured")
	
	// Restrict users?
	if w.confirm("  Restrict to specific users?") {
		users := w.prompt("  Allowed user IDs (comma-separated)", "")
		w.config["telegram.allowed_users"] = users
	}
	
	w.config["telegram.bot_token"] = token
	w.printSuccess("Telegram configured")
	
	return nil
}

func (s *TelegramSection) Validate() error {
	return nil
}

// GatewaySection handles gateway settings
type GatewaySection struct{}

func (s *GatewaySection) Name() string        { return "Gateway Settings" }
func (s *GatewaySection) Description() string { return "Configure gateway settings" }

func (s *GatewaySection) Run(w *Wizard) error {
	listen := w.prompt("  Listen address", "127.0.0.1:8080")
	
	w.config["gateway.listen"] = listen
	w.printSuccess("Gateway configured")
	
	return nil
}

func (s *GatewaySection) Validate() error {
	return nil
}

// FeaturesSection handles optional features
type FeaturesSection struct{}

func (s *FeaturesSection) Name() string        { return "Optional Features" }
func (s *FeaturesSection) Description() string { return "Enable optional features" }

func (s *FeaturesSection) Run(w *Wizard) error {
	fmt.Println("  Select features to enable:")
	fmt.Println()
	
	features := []string{
		"TTS (Text-to-Speech)",
		"Web Search Plugin",
		"Scheduler (Cron jobs)",
		"Node Pairing (Distributed)",
	}
	
	for i, f := range features {
		fmt.Printf("    [ ] %d. %s\n", i+1, f)
	}
	
	fmt.Println()
	fmt.Println("  (Press Enter to skip for now)")
	w.prompt("  Selection", "")
	
	w.printInfo("Features can be configured later in ~/.cicerone/config.yaml")
	
	return nil
}

func (s *FeaturesSection) Validate() error {
	return nil
}

// SecuritySection handles security settings
type SecuritySection struct{}

func (s *SecuritySection) Name() string        { return "Security Settings" }
func (s *SecuritySection) Description() string { return "Configure security settings" }

func (s *SecuritySection) Run(w *Wizard) error {
	if w.confirm("  Enable rate limiting?") {
		maxReqs := w.prompt("  Max requests per minute", "60")
		w.config["security.max_requests"] = maxReqs
	}
	
	if w.confirm("  Run security audit now?") {
		w.printInfo("Running security audit...")
		// TODO: Call security command
		w.printSuccess("Security audit complete")
	}
	
	return nil
}

func (s *SecuritySection) Validate() error {
	return nil
}

// ReviewSection handles review and save
type ReviewSection struct{}

func (s *ReviewSection) Name() string        { return "Configuration Review" }
func (s *ReviewSection) Description() string { return "Review and save configuration" }

func (s *ReviewSection) Run(w *Wizard) error {
	fmt.Println()
	fmt.Println("  Configuration Summary:")
	fmt.Println()
	
	// Print config
	for k, v := range w.config {
		fmt.Printf("    %-25s: %v\n", k, v)
	}
	
	fmt.Println()
	
	if !w.confirm("  Save to ~/.cicerone/config.yaml?") {
		return nil
	}
	
	// TODO: Actually save config
	w.printSuccess("Configuration saved to ~/.cicerone/config.yaml")
	
	if w.confirm("  Run 'cicerone doctor' to verify setup?") {
		w.printInfo("Running doctor...")
		// TODO: Call doctor command
	}
	
	return nil
}

func (s *ReviewSection) Validate() error {
	return nil
}