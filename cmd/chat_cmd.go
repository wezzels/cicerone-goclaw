package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/llm"
	"github.com/crab-meat-repos/cicerone-goclaw/web"
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

Commands:
  /search <query>  - Search the web and get results
  /fetch <url>     - Fetch content from a URL
  /web <query>     - Search and include results in context
  exit, quit, q    - End the session
  clear            - Clear conversation history
  history          - Show conversation history`,
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

	// Create web search provider
	webProvider := web.NewDuckDuckGoProvider()

	stream, _ := cmd.Flags().GetBool("stream")

	fmt.Printf("Connected to LLM at %s\n", baseURL)
	fmt.Printf("Model: %s\n", model)
	fmt.Println("Type 'exit' to quit, 'clear' to reset history, 'history' to view")
	fmt.Println("Use /search <query> to search the web")
	fmt.Println("Use /fetch <url> to fetch a webpage")
	fmt.Println("Use /web <query> to search and include in chat")
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

		// Handle commands
		switch {
		case input == "exit" || input == "quit" || input == "q":
			fmt.Println("\nGoodbye!")
			return nil
		case input == "clear":
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
		case input == "history":
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
		case strings.HasPrefix(input, "/search "):
			query := strings.TrimPrefix(input, "/search ")
			fmt.Printf("\nSearching for: %s\n", query)
			results, err := webProvider.Search(context.Background(), query)
			if err != nil {
				fmt.Printf("Search error: %v\n\n", err)
				continue
			}
			fmt.Println(web.FormatSearchResults(results))
			continue
		case strings.HasPrefix(input, "/fetch "):
			url := strings.TrimPrefix(input, "/fetch ")
			fmt.Printf("\nFetching: %s\n", url)
			content, err := webProvider.Fetch(context.Background(), url)
			if err != nil {
				fmt.Printf("Fetch error: %v\n\n", err)
				continue
			}
			fmt.Println("\nContent:")
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println(content)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/web "):
			query := strings.TrimPrefix(input, "/web ")
			fmt.Printf("\nSearching for: %s\n", query)
			results, err := webProvider.Search(context.Background(), query)
			if err != nil {
				fmt.Printf("Search error: %v\n\n", err)
				continue
			}
			
			// Build context from search results
			var contextBuilder strings.Builder
			contextBuilder.WriteString("Based on the following search results, please answer the question.\n\n")
			for i, result := range results {
				contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, result.Title))
				if result.Snippet != "" {
					contextBuilder.WriteString(fmt.Sprintf("    %s\n", result.Snippet))
				}
				if result.URL != "" {
					contextBuilder.WriteString(fmt.Sprintf("    Source: %s\n", result.URL))
				}
				contextBuilder.WriteString("\n")
			}
			contextBuilder.WriteString(fmt.Sprintf("Question: %s\n", query))
			
			// Add to messages
			input = contextBuilder.String()
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