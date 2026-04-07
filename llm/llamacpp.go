package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LlamaCPPProvider implements Provider for llama.cpp server.
// llama.cpp provides an OpenAI-compatible API on port 8080 by default.
type LlamaCPPProvider struct {
	config   *Config
	client   *http.Client
	endpoint string
}

// Ensure LlamaCPPProvider implements Provider
var _ Provider = (*LlamaCPPProvider)(nil)

// openAIChatRequest is OpenAI-compatible chat request.
type openAIChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

// openAIChatResponse is OpenAI-compatible chat response.
type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta,omitempty"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// openAIModelsResponse is models list response.
type openAIModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// NewLlamaCPPProviderWithClient creates provider with custom HTTP client.
func NewLlamaCPPProviderWithClient(cfg *Config, client *http.Client) *LlamaCPPProvider {
	if cfg == nil {
		cfg = &Config{
			BaseURL: "http://localhost:8080",
			Model:   "local",
			Timeout: 60,
		}
	}
	if client == nil {
		timeout := time.Duration(cfg.Timeout) * time.Second
		client = &http.Client{Timeout: timeout}
	}
	return &LlamaCPPProvider{
		config:   cfg,
		client:   client,
		endpoint: strings.TrimSuffix(cfg.BaseURL, "/"),
	}
}

func (p *LlamaCPPProvider) Generate(ctx context.Context, prompt string) (string, error) {
	// llama.cpp uses chat completions, convert prompt to message
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	return p.Chat(ctx, messages)
}

func (p *LlamaCPPProvider) GenerateStream(ctx context.Context, prompt string) (<-chan StreamChunk, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	return p.ChatStream(ctx, messages)
}

func (p *LlamaCPPProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	stream, err := p.ChatStream(ctx, messages)
	if err != nil {
		return "", err
	}
	return readAll(stream)
}

func (p *LlamaCPPProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	req := openAIChatRequest{
		Model:    p.config.Model,
		Messages: messages,
		Stream:   true,
	}

	return p.streamChat(ctx, req)
}

func (p *LlamaCPPProvider) Models(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var result openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	models := make([]Model, len(result.Data))
	for i, m := range result.Data {
		models[i] = Model{
			Name: m.ID,
		}
	}

	return models, nil
}

func (p *LlamaCPPProvider) IsRunning(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (p *LlamaCPPProvider) Close() error {
	return nil
}

// streamChat makes a streaming chat request.
func (p *LlamaCPPProvider) streamChat(ctx context.Context, req openAIChatRequest) (<-chan StreamChunk, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/v1/chat/completions", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	ch := make(chan StreamChunk, 100)
	go p.parseStream(resp.Body, ch)

	return ch, nil
}

// parseStream reads SSE (Server-Sent Events) response from llama.cpp.
func (p *LlamaCPPProvider) parseStream(body io.ReadCloser, ch chan<- StreamChunk) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE lines start with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for done signal
		if data == "[DONE]" {
			ch <- StreamChunk{Done: true}
			return
		}

		var resp openAIChatResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if len(resp.Choices) == 0 {
			continue
		}

		choice := resp.Choices[0]

		// Handle delta content for streaming
		if choice.Delta.Content != "" {
			ch <- StreamChunk{Text: choice.Delta.Content}
		}

		// Check if done
		if choice.FinishReason == "stop" {
			ch <- StreamChunk{Done: true}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream error: %w", err)}
	}
}