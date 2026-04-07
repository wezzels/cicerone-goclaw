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

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive LLM chat",
	Long: `Start an interactive chat session with the LLM.

The chat uses the configured LLM provider (Ollama or llama.cpp).
Messages are sent to the LLM and responses are streamed back.

Type 'exit' or 'quit' to end the session.
Type 'clear' to clear conversation history.
Type 'history' to show conversation history.`,
	RunE: runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)

	chatCmd.Flags().StringP("model", "m", "", "Model to use (overrides config)")
	chatCmd.Flags().Bool("system", false, "Show system prompt")
	chatCmd.Flags().Bool("stream", true, "Stream responses (default true)")
}

func runChat(cmd *cobra.Command, args []string) error {
	// Get model from flag or config
	model, _ := cmd.Flags().GetString("model")
	if model == "" {
		model = viper.GetString("llm.model")
		if model == "" {
			model = "gemma3:12b"
		}
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

	stream, _ := cmd.Flags().GetBool("stream")

	fmt.Printf("Connected to LLM at %s\n", baseURL)
	fmt.Printf("Model: %s\n", model)
	fmt.Println("Type 'exit' to quit, 'clear' to reset history, 'history' to view")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
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

		var response string
		var respErr error

		if stream {
			response, respErr = streamChat(ctx, provider, messages)
		} else {
			response, respErr = provider.Chat(ctx, messages)
			fmt.Print(response)
		}
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

// streamChat streams the response and returns the full text
func streamChat(ctx context.Context, provider llm.Provider, messages []llm.Message) (string, error) {
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