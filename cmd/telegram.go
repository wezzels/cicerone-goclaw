package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// telegramCmd represents the telegram command
var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Start Telegram bot",
	Long: `Start the Telegram bot with LLM integration.

The bot will:
  - Connect to Telegram using the configured bot token
  - Listen for messages from allowed users
  - Process messages through the configured LLM (Ollama/llama.cpp)
  - Stream responses back to users

Configuration is read from ~/.cicerone/config.yaml or --config flag.`,
	RunE: runTelegram,
}

func init() {
	rootCmd.AddCommand(telegramCmd)

	telegramCmd.Flags().String("token", "", "Telegram bot token (overrides config)")
	telegramCmd.Flags().Bool("debug", false, "Enable debug logging")

	viper.BindPFlag("telegram.bot_token", telegramCmd.Flags().Lookup("token"))
}

func runTelegram(cmd *cobra.Command, args []string) error {
	token := viper.GetString("telegram.bot_token")
	if token == "" {
		return fmt.Errorf("telegram bot token not configured. Set telegram.bot_token in config or use --token")
	}

	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Printf("Starting Telegram bot (debug=%v)", debug)

	// TODO: Initialize Telegram bot from openclaw/telegram.go
	// For now, just validate config
	log.Printf("Bot token configured: %s...", token[:20])

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v, shutting down...", sig)
		cancel()
	}()

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Telegram bot stopped")

	return nil
}