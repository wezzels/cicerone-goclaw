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

// OllamaProvider implements Provider for Ollama server.
type OllamaProvider struct {
	config   *Config
	client   *http.Client
	endpoint string
}

// Ensure OllamaProvider implements Provider
var _ Provider = (*OllamaProvider)(nil)

// ollamaGenerateRequest is the request body for /api/generate.
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Raw    bool   `json:"raw,omitempty"`
}

// ollamaGenerateResponse is the response from /api/generate.
type ollamaGenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// ollamaChatRequest is the request body for /api/chat.
type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// ollamaChatResponse is the response from /api/chat.
type ollamaChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

// ollamaModelsResponse is the response from /api/tags.
type ollamaModelsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Modified  string `json:"modified_at"`
	Digest    string `json:"digest"`
}

// NewOllamaProviderWithClient creates provider with custom HTTP client.
func NewOllamaProviderWithClient(cfg *Config, client *http.Client) *OllamaProvider {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if client == nil {
		timeout := time.Duration(cfg.Timeout) * time.Second
		client = &http.Client{Timeout: timeout}
	}
	return &OllamaProvider{
		config:   cfg,
		client:   client,
		endpoint: strings.TrimSuffix(cfg.BaseURL, "/"),
	}
}

func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	stream, err := p.GenerateStream(ctx, prompt)
	if err != nil {
		return "", err
	}
	return readAll(stream)
}

func (p *OllamaProvider) GenerateStream(ctx context.Context, prompt string) (<-chan StreamChunk, error) {
	req := ollamaGenerateRequest{
		Model:  p.config.Model,
		Prompt: prompt,
		Stream: true,
	}

	return p.streamRequest(ctx, "/api/generate", req)
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	stream, err := p.ChatStream(ctx, messages)
	if err != nil {
		return "", err
	}
	return readAll(stream)
}

func (p *OllamaProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	req := ollamaChatRequest{
		Model:    p.config.Model,
		Messages: messages,
		Stream:   true,
	}

	return p.streamRequest(ctx, "/api/chat", req)
}

func (p *OllamaProvider) Models(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/api/tags", nil)
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

	var result ollamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	models := make([]Model, len(result.Models))
	for i, m := range result.Models {
		models[i] = Model{
			Name:     m.Name,
			Size:     m.Size,
			Modified: m.Modified,
		}
	}

	return models, nil
}

func (p *OllamaProvider) IsRunning(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/api/version", nil)
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

func (p *OllamaProvider) Close() error {
	return nil
}

// streamRequest makes a streaming request to Ollama.
func (p *OllamaProvider) streamRequest(ctx context.Context, path string, body interface{}) (<-chan StreamChunk, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
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

// parseStream reads NDJSON response and sends chunks.
func (p *OllamaProvider) parseStream(body io.ReadCloser, ch chan<- StreamChunk) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Try generate response first
		var genResp ollamaGenerateResponse
		if err := json.Unmarshal(line, &genResp); err == nil && genResp.Response != "" {
			ch <- StreamChunk{Text: genResp.Response, Done: genResp.Done}
			if genResp.Done {
				return
			}
			continue
		}

		// Try chat response
		var chatResp ollamaChatResponse
		if err := json.Unmarshal(line, &chatResp); err == nil && chatResp.Message.Content != "" {
			ch <- StreamChunk{Text: chatResp.Message.Content, Done: chatResp.Done}
			if chatResp.Done {
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream error: %w", err)}
	}
}