package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage Cicerone configuration.

Configuration is stored in ~/.cicerone/config.yaml.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show configuration",
	Long:  `Display the current configuration.`,
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Long: `Set a configuration value.

Keys use dot notation:
  cicerone config set telegram.bot_token "your-token"
  cicerone config set llm.model "llama3"

The config file is automatically created if it doesn't exist.`,
	RunE:  runConfigSet,
	Args:  cobra.ExactArgs(2),
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	fmt.Println("Configuration")
	fmt.Println("=============")
	fmt.Println()

	settings := []string{
		"telegram.bot_token",
		"telegram.allowed_users",
		"llm.provider",
		"llm.base_url",
		"llm.model",
		"llm.timeout",
		"workspace.path",
		"gateway.listen",
	}

	for _, key := range settings {
		value := viper.Get(key)
		if value == nil {
			value = "(not set)"
		}

		// Mask sensitive values
		if strings.Contains(key, "token") || strings.Contains(key, "password") {
			if str, ok := value.(string); ok && len(str) > 10 {
				value = str[:10] + "..."
			}
		}

		fmt.Printf("  %-25s: %v\n", key, value)
	}

	fmt.Println()
	fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	viper.Set(key, value)

	// Write config file
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = home + "/.cicerone/config.yaml"
	}

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	fmt.Printf("Written to: %s\n", configPath)

	return nil
}