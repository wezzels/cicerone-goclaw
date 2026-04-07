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
	return &OllamaProvider{config: cfg}
}

// NewLlamaCPPProvider creates a llama.cpp provider.
func NewLlamaCPPProvider(cfg *Config) Provider {
	if cfg == nil {
		cfg = DefaultConfig()
		cfg.BaseURL = "http://localhost:8080"
	}
	return &LlamaCPPProvider{config: cfg}
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