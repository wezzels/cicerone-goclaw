package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/crab-meat-repos/cicerone-goclaw/llm"
)

// LLMHandler wraps an LLM provider for message handling.
type LLMHandler struct {
	provider llm.Provider
	system   string // System prompt
}

// NewLLMHandler creates a new LLM handler.
func NewLLMHandler(provider llm.Provider, systemPrompt string) *LLMHandler {
	return &LLMHandler{
		provider: provider,
		system:   systemPrompt,
	}
}

// Handle processes a message using the LLM.
func (h *LLMHandler) Handle(ctx context.Context, msg Message) (string, error) {
	// Build messages
	messages := []llm.Message{}

	// Add system prompt if set
	if h.system != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: h.system,
		})
	}

	// Add user message
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: msg.Text,
	})

	// Get response from LLM
	response, err := h.provider.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// HandleWithHistory processes a message with conversation history.
func (h *LLMHandler) HandleWithHistory(ctx context.Context, msg Message, history []llm.Message) (string, error) {
	// Build messages
	messages := []llm.Message{}

	// Add system prompt if set
	if h.system != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: h.system,
		})
	}

	// Add history
	messages = append(messages, history...)

	// Add user message
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: msg.Text,
	})

	// Get response from LLM
	response, err := h.provider.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// StreamHandler processes messages with streaming.
type StreamHandler struct {
	provider llm.Provider
	system   string
	onChunk  func(chunk string)
}

// NewStreamHandler creates a streaming handler.
func NewStreamHandler(provider llm.Provider, systemPrompt string, onChunk func(chunk string)) *StreamHandler {
	return &StreamHandler{
		provider: provider,
		system:   systemPrompt,
		onChunk: onChunk,
	}
}

// Handle processes a message with streaming.
func (h *StreamHandler) Handle(ctx context.Context, msg Message) (string, error) {
	// Build messages
	messages := []llm.Message{}

	if h.system != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: h.system,
		})
	}

	messages = append(messages, llm.Message{
		Role:    "user",
		Content: msg.Text,
	})

	// Stream response
	stream, err := h.provider.ChatStream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}

	var result string
	for chunk := range stream {
		if chunk.Error != nil {
			return result, chunk.Error
		}

		result += chunk.Text

		if h.onChunk != nil {
			h.onChunk(chunk.Text)
		}

		if chunk.Done {
			break
		}
	}

	return strings.TrimSpace(result), nil
}