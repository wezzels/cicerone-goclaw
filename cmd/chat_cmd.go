package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/agent"
	"github.com/crab-meat-repos/cicerone-goclaw/llm"
	"github.com/crab-meat-repos/cicerone-goclaw/web"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive LLM chat with agentic capabilities",
	Long: `Start an interactive chat session with the LLM.

The chat uses the configured LLM provider (Ollama or llama.cpp).
Messages are sent to the LLM and responses are streamed back.

Commands:
  /search <query>  - Search the web and get results
  /fetch <url>     - Fetch content from a URL
  /web <query>     - Search and include results in context

  /task <task>     - Run autonomous agent task
  /agent            - Enable autonomous mode (LLM can execute commands)
  /stop              - Disable autonomous mode

  /run <command>   - Execute shell command
  /cd <path>       - Change directory
  /pwd              - Show current directory
  /ls [path]        - List directory
  /read <file>     - Read file contents
  /write <file>    - Write to file (then enter content)
  /append <file>   - Append to file
  /delete <file>   - Delete file
  /mkdir <dir>     - Create directory

  /get <url>        - HTTP GET request
  /post <url> <json> - HTTP POST request

  /help              - Show all commands
  exit, quit, q      - End the session
  clear              - Clear conversation history
  history            - Show conversation history`,
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
		timeout = 300 // 5 minutes for large models with tools
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

	// Create agent for command execution
	workDir, _ := os.Getwd()
	ag := agent.New(workDir)

	// Create autonomous agent
	autoAgent := agent.NewAutonomousAgent(ag)

	// stream flag deprecated - we use ChatWithTools now for automatic tool calling
	_, _ = cmd.Flags().GetBool("stream")

	fmt.Printf("Connected to LLM at %s\n", baseURL)
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Work Dir: %s\n", workDir)
	fmt.Println("Type 'exit' to quit, '/help' for commands")
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

	// Add tools system prompt
	toolsPrompt := `You are an AI assistant with access to tools. When the user asks you to do something that requires file operations, web searches, or running commands, you MUST call the appropriate tool to actually perform the action.

IMPORTANT: Do NOT just describe what you would do. You MUST actually call the tools to execute the requested actions. The tools are real and will be executed.

Available tools:
- write_file(path, content): Write content to a file. USE THIS when asked to create/save/write files.
- read_file(path): Read a file's contents.
- run_shell(command): Execute a shell command. USE THIS for compiling code, running programs, etc.
- web_search(query): Search the web for information.
- web_fetch(url): Fetch content from a URL.
- list_directory(path): List directory contents.

Examples:
- User: "Write a hello world C program" → Call write_file, then run_shell to compile
- User: "Search for weather" → Call web_search
- User: "Run this command" → Call run_shell

ALWAYS use the tools to perform actions. The tools work and will execute your commands.`
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: toolsPrompt,
	})

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

		// Agent commands
		case input == "/help":
			fmt.Println("\n" + ag.Help())
			fmt.Println()
			continue
		case input == "/commands":
			fmt.Println("\nAvailable commands:")
			for _, cmd := range ag.ListCommands() {
				fmt.Printf("  /%s\n", cmd)
			}
			fmt.Println()
			continue

		// Shell commands
		case strings.HasPrefix(input, "/run "):
			cmdStr := strings.TrimPrefix(input, "/run ")
			output, err := ag.Execute(context.Background(), cmdStr)
			if err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Println("\n" + output)
			}
			continue
		case strings.HasPrefix(input, "/cd "):
			dir := strings.TrimPrefix(input, "/cd ")
			if err := ag.SetWorkDir(dir); err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Printf("\nChanged to: %s\n\n", ag.WorkDir())
			}
			continue
		case input == "/pwd":
			fmt.Printf("\n%s\n\n", ag.WorkDir())
			continue
		case strings.HasPrefix(input, "/ls"):
			path := strings.TrimPrefix(input, "/ls ")
			if path == input {
				path = "."
			}
			entries, err := ag.ListDir(path)
			if err != nil {
				fmt.Printf("\nError: %v\n\n", err)
				continue
			}
			fmt.Println()
			for _, entry := range entries {
				info, _ := entry.Info()
				fmt.Printf("%s %8d %s\n", info.Mode().String()[:10], info.Size(), entry.Name())
			}
			fmt.Println()
			continue

		// File commands
		case strings.HasPrefix(input, "/read "):
			filePath := strings.TrimPrefix(input, "/read ")
			content, err := ag.ReadFile(filePath)
			if err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Println("\n" + strings.Repeat("-", 50))
				fmt.Println(content)
				fmt.Println(strings.Repeat("-", 50))
			}
			continue
		case strings.HasPrefix(input, "/write "):
			remaining := strings.TrimPrefix(input, "/write ")
			parts := strings.SplitN(remaining, " ", 2)
			if len(parts) < 1 {
				fmt.Println("\nUsage: /write <file> <content>")
				fmt.Println("       /write <file> (then enter content, Ctrl+D to save)")
				fmt.Println()
				continue
			}
			filePath := parts[0]
			var content string
			if len(parts) > 1 {
				content = parts[1]
			} else {
				fmt.Println("\nEnter content (Ctrl+D to save):")
				var lines []string
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				content = strings.Join(lines, "\n")
			}
			if err := ag.WriteFile(filePath, content); err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Printf("\nWrote %d bytes to %s\n\n", len(content), filePath)
			}
			continue
		case strings.HasPrefix(input, "/append "):
			parts := strings.SplitN(strings.TrimPrefix(input, "/append "), " ", 2)
			if len(parts) < 2 {
				fmt.Println("\nUsage: /append <file> <content>")
				continue
			}
			if err := ag.AppendFile(parts[0], parts[1]); err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Printf("\nAppended to %s\n\n", parts[0])
			}
			continue
		case strings.HasPrefix(input, "/delete "):
			filePath := strings.TrimPrefix(input, "/delete ")
			if err := ag.DeleteFile(filePath); err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Printf("\nDeleted: %s\n\n", filePath)
			}
			continue
		case strings.HasPrefix(input, "/mkdir "):
			dir := strings.TrimPrefix(input, "/mkdir ")
			if err := ag.Mkdir(dir); err != nil {
				fmt.Printf("\nError: %v\n\n", err)
			} else {
				fmt.Printf("\nCreated: %s\n\n", dir)
			}
			continue

		case strings.HasPrefix(input, "/task "):
			task := strings.TrimPrefix(input, "/task ")
			fmt.Printf("\nStarting autonomous task: %s\n", task)
			fmt.Println("Using native function calling for tool execution.")
			fmt.Println("Press Ctrl+C to interrupt.")
			fmt.Println()

			// Progress callback
			onProgress := func(status string) {
				fmt.Printf("[Agent] %s\n", status)
			}

			// Execute task with native function calling
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			result, err := autoAgent.ExecuteTaskWithTools(ctx, task, onProgress, provider)
			cancel()

			if err != nil {
				fmt.Printf("\nTask failed: %v\n\n", err)
				continue
			}

			// Print result
			fmt.Println()
			fmt.Println(strings.Repeat("=", 50))
			if result.Completed {
				fmt.Println("Task completed successfully!")
			} else {
				fmt.Printf("Task incomplete: %v\n", result.Error)
			}
			fmt.Println(strings.Repeat("=", 50))
			fmt.Println()
			if result.FinalOutput != "" {
				fmt.Println("Output:")
				fmt.Println(result.FinalOutput)
				fmt.Println()
			}
			fmt.Printf("Steps taken: %d\n", len(result.Steps))
			for _, step := range result.Steps {
				fmt.Printf("  Step %d: ", step.StepNumber)
				if len(step.ToolCalls) > 0 {
					toolNames := make([]string, len(step.ToolCalls))
					for i, tc := range step.ToolCalls {
						toolNames[i] = tc.Name
					}
					fmt.Printf("%s\n", strings.Join(toolNames, ", "))
				} else {
					fmt.Println("(no tools)")
				}
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

		// Use ChatWithTools to allow tool calling
		toolDefs := agent.ToolsToOllamaFormat()
		resp, err := provider.ChatWithTools(ctx, messages, toolDefs)
		cancel()

		if err != nil {
			fmt.Printf("\nError: %v\n\n", err)
			messages = messages[:len(messages)-1]
			continue
		}

		// Display any text response
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}

		// Add assistant response to history
		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// If LLM made tool calls, execute them and continue conversation
		if len(resp.ToolCalls) > 0 {
			fmt.Print("\n\n[Executing tools...]\n")

			// Execute each tool call
			for _, tc := range resp.ToolCalls {
				fmt.Printf("  - %s\n", tc.Function.Name)
			}
			fmt.Println()

			// Convert tool calls to agent format and execute
			toolCalls := make([]agent.ToolCall, len(resp.ToolCalls))
			for i, tc := range resp.ToolCalls {
				var args map[string]interface{}
				if tc.Function.Arguments != "" {
					json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				toolCalls[i] = agent.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: args,
				}
			}

			// Execute tools
			exec := agent.NewExecutor(ag)
			results := exec.ExecuteTools(context.Background(), toolCalls)

			// Display results
			for _, r := range results {
				if r.Success {
					fmt.Printf("[Result: %s]\n", r.Name)
					// Truncate long output
					output := r.Output
					if len(output) > 500 {
						output = output[:500] + "..."
					}
					fmt.Println(output)
				} else {
					fmt.Printf("[Error: %s] %v\n", r.Name, r.Error)
				}
			}

			// Add tool results to messages
			for _, r := range results {
				resultJSON, _ := json.Marshal(map[string]interface{}{
					"success": r.Success,
					"output":  r.Output,
				})
				messages = append(messages, llm.Message{
					Role:       "tool",
					Content:    string(resultJSON),
					ToolCallID: r.Name,
				})
			}

			// Get follow-up response from LLM
			fmt.Print("\nAssistant: ")
			ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			followUp, err := provider.Chat(ctx2, messages)
			cancel2()

			if err != nil {
				fmt.Printf("\nError: %v\n\n", err)
				continue
			}

			fmt.Println(followUp)
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: followUp,
			})
		} else {
			fmt.Println()
		}
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