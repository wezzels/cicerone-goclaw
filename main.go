// Command cicerone is a Go-only messaging gateway with LLM integration.
//
// Features:
//   - Telegram bot with LLM-powered responses
//   - Interactive TUI for local management
//   - Health diagnostics (doctor)
//   - Security auditing
//   - Gateway management
//
// Usage:
//
//	cicerone [command]
//
// Available Commands:
//	telegram    Start Telegram bot
//	tui         Launch interactive TUI
//	gateway     Gateway management
//	doctor      Run health diagnostics
//	security    Run security audit
//	llm         Manage LLM configuration
//	do          Execute via LLM
//	chat        Interactive LLM chat
//	config      Manage configuration
//	version     Show version
//	help        Help about any command
package main

import (
	"github.com/crab-meat-repos/cicerone-goclaw/cmd"
)

func main() {
	cmd.Execute()
}