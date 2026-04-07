package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// llmCmd represents the llm command
var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM configuration",
	Long: `Manage LLM provider configuration.

Cicerone supports:
  - Ollama (local, default)
  - llama.cpp server (OpenAI-compatible)

Configure in ~/.cicerone/config.yaml:
  llm:
    provider: ollama
    base_url: "http://localhost:11434"
    model: "gemma3:12b"`,
}

var llmShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show LLM configuration",
	Long:  `Display the current LLM provider and configuration.`,
	RunE:  runLLMShow,
}

var llmTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test LLM connection",
	Long:  `Test the connection to the configured LLM provider.`,
	RunE:  runLLMTest,
}

var llmModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	Long:  `List models available from the configured LLM provider.`,
	RunE:  runLLMModels,
}

func init() {
	rootCmd.AddCommand(llmCmd)
	llmCmd.AddCommand(llmShowCmd)
	llmCmd.AddCommand(llmTestCmd)
	llmCmd.AddCommand(llmModelsCmd)
}

func runLLMShow(cmd *cobra.Command, args []string) error {
	fmt.Println("LLM Configuration")
	fmt.Println("=================")
	fmt.Println()

	provider := viper.GetString("llm.provider")
	if provider == "" {
		provider = "ollama"
	}

	baseURL := viper.GetString("llm.base_url")
	if baseURL == "" {
		if provider == "ollama" {
			baseURL = "http://localhost:11434"
		}
	}

	model := viper.GetString("llm.model")
	if model == "" {
		model = "gemma3:12b"
	}

	fmt.Printf("Provider:  %s\n", provider)
	fmt.Printf("Base URL:  %s\n", baseURL)
	fmt.Printf("Model:     %s\n", model)
	fmt.Printf("Timeout:   %ds\n", viper.GetInt("llm.timeout"))

	return nil
}

func runLLMTest(cmd *cobra.Command, args []string) error {
	fmt.Println("Testing LLM connection...")

	baseURL := viper.GetString("llm.base_url")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Test connection
	client := http.Client{Timeout: 5 * time.Second}

	start := time.Now()
	resp, err := client.Get(baseURL + "/api/version")
	if err != nil {
		fmt.Printf("✗ Connection failed: %v\n", err)
		return fmt.Errorf("connection failed")
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	fmt.Printf("✓ Connection successful (%dms)\n", latency.Milliseconds())

	// Parse version info
	var info map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&info)

	if version, ok := info["version"].(string); ok {
		fmt.Printf("✓ Ollama version: %s\n", version)
	}

	// Test generation
	fmt.Println("\nTesting generation...")
	model := viper.GetString("llm.model")
	if model == "" {
		model = "gemma3:12b"
	}

	testPrompt := map[string]string{
		"model":  model,
		"prompt": "Say 'test' in one word",
		"stream": "false",
	}

	testJSON, _ := json.Marshal(testPrompt)
	resp2, err := client.Post(baseURL+"/api/generate", "application/json", strings.NewReader(string(testJSON)))
	if err != nil {
		fmt.Printf("⚠ Generation test failed: %v\n", err)
		return nil
	}
	defer resp2.Body.Close()

	var genResp map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&genResp)

	if genResp["response"] != nil {
		fmt.Printf("✓ Generation test passed\n")
	}

	return nil
}

func runLLMModels(cmd *cobra.Command, args []string) error {
	fmt.Println("Available Models")
	fmt.Println("================")

	baseURL := viper.GetString("llm.base_url")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + "/api/tags")
	if err != nil {
		fmt.Printf("✗ Failed to fetch models: %v\n", err)
		return fmt.Errorf("failed to fetch models")
	}
	defer resp.Body.Close()

	var tags struct {
		Models []struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Modified string `json:"modified_at"`
		} `json:"models"`
	}

	json.NewDecoder(resp.Body).Decode(&tags)

	if len(tags.Models) == 0 {
		fmt.Println("No models found. Pull with: ollama pull <model>")
		return nil
	}

	for _, m := range tags.Models {
		sizeGB := float64(m.Size) / 1024 / 1024 / 1024
		fmt.Printf("  %-30s %5.1f GB\n", m.Name, sizeGB)
	}

	fmt.Printf("\nTotal: %d models\n", len(tags.Models))

	return nil
}