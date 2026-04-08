// Package llm provides LLM provider implementations for Ollama and llama.cpp.
//
// This package implements a simple interface for interacting with local LLM
// servers. It supports:
//   - Ollama (default, localhost:11434)
//   - llama.cpp server (OpenAI-compatible API)
//
// Example usage:
//
//	provider := llm.NewOllamaProvider("http://localhost:11434", "gemma3:12b")
//	response, err := provider.Generate(ctx, "Hello, world!")
package llm

import (
	"context"
	"net/http"
	"strings"
	"time"

	"io"
)

// Provider defines the interface for LLM providers.
type Provider interface {
	// Generate sends a prompt and returns the complete response.
	Generate(ctx context.Context, prompt string) (string, error)

	// GenerateStream sends a prompt and streams the response.
	GenerateStream(ctx context.Context, prompt string) (<-chan StreamChunk, error)

	// Chat sends a conversation and returns the response.
	Chat(ctx context.Context, messages []Message) (string, error)

	// ChatStream sends a conversation and streams the response.
	ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)

	// ChatWithTools sends a conversation with tools and returns the response.
	// The response may contain tool calls that need to be executed.
	ChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error)

	// Models returns available models.
	Models(ctx context.Context) ([]Model, error)

	// IsRunning checks if the provider is available.
	IsRunning(ctx context.Context) bool

	// Close cleans up resources.
	Close() error
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
	// Tool calls for assistant messages (Ollama format)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// Tool call ID for tool response messages
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// Tool represents a tool definition for function calling.
type Tool struct {
	Type     string      `json:"type"`     // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function tool.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call from the LLM.
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`     // "function"
	Function ToolCallFunction        `json:"function"`
}

// ToolCallFunction contains the function name and arguments.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatResponse is the response from ChatWithTools.
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Done      bool       `json:"done"`
}

// Model represents an available LLM model.
type Model struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`      // bytes
	Modified  string `json:"modified"`  // ISO timestamp
	Quantized bool   `json:"quantized"` // true for GGUF models
}

// StreamChunk represents a chunk of streamed response.
type StreamChunk struct {
	Text  string
	Done  bool
	Error error
}

// Config holds provider configuration.
type Config struct {
	BaseURL string
	Model   string
	Timeout int // seconds, default 60
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		BaseURL: "http://localhost:11434",
		Model:   "gemma3:12b",
		Timeout: 60,
	}
}

// NewProvider creates a provider based on the config URL.
// If baseURL contains ":11434", creates Ollama provider.
// Otherwise, creates llama.cpp provider.
func NewProvider(cfg *Config) Provider {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Detect provider type from URL
	if len(cfg.BaseURL) >= 10 && cfg.BaseURL[len(cfg.BaseURL)-5:] == "11434" {
		return NewOllamaProvider(cfg)
	}

	return NewLlamaCPPProvider(cfg)
}

// NewOllamaProvider creates an Ollama provider.
func NewOllamaProvider(cfg *Config) Provider {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	return &OllamaProvider{
		config:   cfg,
		client:   &http.Client{Timeout: timeout},
		endpoint: strings.TrimSuffix(cfg.BaseURL, "/"),
	}
}

// NewLlamaCPPProvider creates a llama.cpp provider.
func NewLlamaCPPProvider(cfg *Config) Provider {
	if cfg == nil {
		cfg = DefaultConfig()
		cfg.BaseURL = "http://localhost:8080"
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	return &LlamaCPPProvider{
		config:   cfg,
		client:   &http.Client{Timeout: timeout},
		endpoint: strings.TrimSuffix(cfg.BaseURL, "/"),
	}
}

// Helper to read all from stream
func readAll(stream <-chan StreamChunk) (string, error) {
	var result string
	for chunk := range stream {
		if chunk.Error != nil {
			return result, chunk.Error
		}
		result += chunk.Text
		if chunk.Done {
			break
		}
	}
	return result, nil
}

// Ensure Stdin implements io.Reader
var _ io.Reader = nil