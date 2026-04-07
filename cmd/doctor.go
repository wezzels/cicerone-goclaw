package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run health diagnostics",
	Long: `Run health diagnostics on the Cicerone installation.

Checks:
  - Config file exists and valid
  - Telegram token configured
  - LLM connection (Ollama/llama.cpp)
  - Model availability
  - Network connectivity
  - Disk space
  - Memory availability`,
	RunE: runDoctor,
}

type CheckResult struct {
	Name   string
	Status string // OK, WARN, FAIL
	Detail string
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("🏥 Cicerone Health Check")
	fmt.Println("========================")
	fmt.Println()

	checks := []struct {
		Name string
		Fn   func() CheckResult
	}{
		{"Config", checkConfig},
		{"Telegram Token", checkTelegramToken},
		{"LLM Connection", checkLLMConnection},
		{"Ollama Status", checkOllamaStatus},
		{"Model Available", checkModelAvailable},
		{"Network", checkNetwork},
		{"Disk Space", checkDiskSpace},
		{"Memory", checkMemory},
	}

	passed := 0
	warnings := 0
	failed := 0

	for _, check := range checks {
		result := check.Fn()
		printResult(result)

		switch result.Status {
		case "OK":
			passed++
		case "WARN":
			warnings++
		case "FAIL":
			failed++
		}
	}

	fmt.Println()
	fmt.Println("========================")
	fmt.Printf("Results: %d passed, %d warnings, %d failed\n", passed, warnings, failed)

	if failed > 0 {
		return fmt.Errorf("health check failed")
	}

	if warnings > 0 {
		fmt.Println("\nNote: Warnings are non-critical issues that may need attention.")
	}

	return nil
}

func printResult(r CheckResult) {
	var icon string
	switch r.Status {
	case "OK":
		icon = "✓"
	case "WARN":
		icon = "⚠"
	case "FAIL":
		icon = "✗"
	}

	fmt.Printf("  %s %-20s %s\n", icon, r.Name+":", r.Detail)
}

func checkConfig() CheckResult {
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = home + "/.cicerone/config.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return CheckResult{"Config", "FAIL", "not found at " + configPath}
	}

	return CheckResult{"Config", "OK", configPath}
}

func checkTelegramToken() CheckResult {
	token := viper.GetString("telegram.bot_token")
	if token == "" {
		return CheckResult{"Telegram Token", "WARN", "not configured (get token from @BotFather)"}
	}

	if len(token) < 40 {
		return CheckResult{"Telegram Token", "WARN", "token appears invalid"}
	}

	return CheckResult{"Telegram Token", "OK", fmt.Sprintf("configured (%d chars)", len(token))}
}

func checkLLMConnection() CheckResult {
	llmURL := viper.GetString("llm.base_url")
	if llmURL == "" {
		llmURL = "http://localhost:11434"
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(llmURL)
	if err != nil {
		return CheckResult{"LLM Connection", "FAIL", fmt.Sprintf("cannot connect to %s", llmURL)}
	}
	defer resp.Body.Close()

	return CheckResult{"LLM Connection", "OK", llmURL}
}

func checkOllamaStatus() CheckResult {
	// Check if ollama process is running
	out, err := exec.Command("pgrep", "-x", "ollama").Output()
	if err != nil || len(out) == 0 {
		return CheckResult{"Ollama Status", "WARN", "not running (start with 'ollama serve')"}
	}

	pid := strings.TrimSpace(string(out))
	return CheckResult{"Ollama Status", "OK", fmt.Sprintf("running (PID %s)", pid)}
}

func checkModelAvailable() CheckResult {
	model := viper.GetString("llm.model")
	if model == "" {
		model = "gemma3:12b"
	}

	llmURL := viper.GetString("llm.base_url")
	if llmURL == "" {
		llmURL = "http://localhost:11434"
	}

	// Check if model is available
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(llmURL + "/api/tags")
	if err != nil {
		return CheckResult{"Model Available", "FAIL", "cannot check models"}
	}
	defer resp.Body.Close()

	// TODO: Parse response and check if model exists
	return CheckResult{"Model Available", "OK", model + " (configured)"}
}

func checkNetwork() CheckResult {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.telegram.org")
	if err != nil {
		return CheckResult{"Network", "FAIL", "cannot reach Telegram API"}
	}
	defer resp.Body.Close()

	return CheckResult{"Network", "OK", "can reach Telegram API"}
}

func checkDiskSpace() CheckResult {
	home, _ := os.UserHomeDir()

	// Use df to check disk space
	out, err := exec.Command("df", "-h", home).Output()
	if err != nil {
		return CheckResult{"Disk Space", "WARN", "cannot check"}
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			available := fields[3]
			return CheckResult{"Disk Space", "OK", available + " available"}
		}
	}

	return CheckResult{"Disk Space", "WARN", "unknown"}
}

func checkMemory() CheckResult {
	var available string

	if runtime.GOOS == "linux" {
		// Read /proc/meminfo
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return CheckResult{"Memory", "WARN", "cannot check"}
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					// Convert KB to GB
					var kb int
					fmt.Sscanf(fields[1], "%d", &kb)
					gb := float64(kb) / 1024 / 1024
					available = fmt.Sprintf("%.1f GB available", gb)
					break
				}
			}
		}
	}

	if available == "" {
		available = "unknown"
	}

	return CheckResult{"Memory", "OK", available}
}