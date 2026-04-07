package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
Type 'clear' to clear conversation history.`,
	RunE: runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)

	chatCmd.Flags().StringP("model", "m", "", "Model to use (overrides config)")
	chatCmd.Flags().Bool("system", false, "Show system prompt")
}

func runChat(cmd *cobra.Command, args []string) error {
	model, _ := cmd.Flags().GetString("model")
	if model == "" {
		model = viper.GetString("llm.model")
		if model == "" {
			model = "gemma3:12b"
		}
	}

	fmt.Printf("Starting chat with %s\n", model)
	fmt.Println("Type 'exit' to quit, 'clear' to reset history")
	fmt.Println()

	// TODO: Implement actual chat using llm/ package
	// For now, this is a placeholder

	reader := bufio.NewReader(os.Stdin)
	history := []string{}

	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("\nGoodbye!")
			return nil
		}

		if input == "clear" {
			history = nil
			fmt.Println("History cleared.")
			fmt.Println()
			continue
		}

		// Add to history
		history = append(history, "User: "+input)

		// TODO: Send to LLM
		fmt.Printf("\nAssistant: [TODO: LLM response]\n\n")
	}

	return nil
}