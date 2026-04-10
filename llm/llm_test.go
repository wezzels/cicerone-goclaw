package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected default BaseURL, got %s", cfg.BaseURL)
	}

	if cfg.Model != "gemma3:12b" {
		t.Errorf("Expected default model gemma3:12b, got %s", cfg.Model)
	}

	if cfg.Timeout != 300 {
		t.Errorf("Expected default timeout 300 (5 min), got %d", cfg.Timeout)
	}
}

func TestNewProvider(t *testing.T) {
	// Test Ollama provider detection
	ollamaCfg := &Config{BaseURL: "http://localhost:11434", Model: "test"}
	ollamaProvider := NewProvider(ollamaCfg)
	if _, ok := ollamaProvider.(*OllamaProvider); !ok {
		t.Error("Expected OllamaProvider for port 11434")
	}

	// Test llama.cpp provider detection
	llamacppCfg := &Config{BaseURL: "http://localhost:8080", Model: "test"}
	llamaProvider := NewProvider(llamacppCfg)
	if _, ok := llamaProvider.(*LlamaCPPProvider); !ok {
		t.Error("Expected LlamaCPPProvider for port 8080")
	}
}

func TestOllamaProviderIsRunning(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "0.1.0"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewOllamaProviderWithClient(cfg, server.Client())

	if !provider.IsRunning(context.Background()) {
		t.Error("Expected IsRunning to return true")
	}
}

func TestOllamaProviderGenerate(t *testing.T) {
	// Create test server that returns streaming response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			// Send two chunks then done
			_, _ = w.Write([]byte(`{"model":"test","response":"Hello","done":false}
{"model":"test","response":" world","done":true}
`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewOllamaProviderWithClient(cfg, server.Client())

	result, err := provider.Generate(context.Background(), "test")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result != "Hello world" {
		t.Errorf("Expected 'Hello world', got %s", result)
	}
}

func TestOllamaProviderChat(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"model":"test","message":{"role":"assistant","content":"Response"},"done":true}
`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewOllamaProviderWithClient(cfg, server.Client())

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	result, err := provider.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if result != "Response" {
		t.Errorf("Expected 'Response', got %s", result)
	}
}

func TestOllamaProviderModels(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ollamaModelsResponse{
				Models: []ollamaModel{
					{Name: "llama3", Size: 4000000000, Modified: "2024-01-01"},
					{Name: "gemma3:12b", Size: 7000000000, Modified: "2024-01-02"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewOllamaProviderWithClient(cfg, server.Client())

	models, err := provider.Models(context.Background())
	if err != nil {
		t.Fatalf("Models failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].Name != "llama3" {
		t.Errorf("Expected first model 'llama3', got %s", models[0].Name)
	}
}

func TestLlamaCPPProviderIsRunning(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewLlamaCPPProviderWithClient(cfg, server.Client())

	if !provider.IsRunning(context.Background()) {
		t.Error("Expected IsRunning to return true")
	}
}

func TestLlamaCPPProviderChat(t *testing.T) {
	// Create test server that returns SSE response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "text/event-stream")
			// SSE format
			_, _ = w.Write([]byte(`data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":""}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":""}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewLlamaCPPProviderWithClient(cfg, server.Client())

	messages := []Message{
		{Role: "user", Content: "test"},
	}
	result, err := provider.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if result != "Hello world" {
		t.Errorf("Expected 'Hello world', got %s", result)
	}
}

func TestLlamaCPPProviderModels(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(openAIModelsResponse{
				Data: []struct {
					ID      string `json:"id"`
					Object  string `json:"object"`
					Created int64  `json:"created"`
					OwnedBy string `json:"owned_by"`
				}{
					{ID: "local-model"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 5,
	}
	provider := NewLlamaCPPProviderWithClient(cfg, server.Client())

	models, err := provider.Models(context.Background())
	if err != nil {
		t.Fatalf("Models failed: %v", err)
	}

	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	if models[0].Name != "local-model" {
		t.Errorf("Expected 'local-model', got %s", models[0].Name)
	}
}

func TestMessageMarshaling(t *testing.T) {
	msg := Message{Role: "user", Content: "Hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Message
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Role != msg.Role || unmarshaled.Content != msg.Content {
		t.Error("Message marshaling/unmarshaling mismatch")
	}
}

func TestStreamChunk(t *testing.T) {
	// Test error chunk
	chunk := StreamChunk{Error: context.DeadlineExceeded}
	if chunk.Error == nil {
		t.Error("Expected error in StreamChunk")
	}

	// Test done chunk
	chunk = StreamChunk{Done: true}
	if !chunk.Done {
		t.Error("Expected Done to be true")
	}

	// Test text chunk
	chunk = StreamChunk{Text: "test"}
	if chunk.Text != "test" {
		t.Errorf("Expected 'test', got %s", chunk.Text)
	}
}

func TestToolStructure(t *testing.T) {
	tool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "write_file",
			Description: "Write content to a file",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}

	if tool.Type != "function" {
		t.Errorf("Expected tool type 'function', got %s", tool.Type)
	}
	if tool.Function.Name != "write_file" {
		t.Errorf("Expected function name 'write_file', got %s", tool.Function.Name)
	}
}

func TestToolCallStructure(t *testing.T) {
	tc := ToolCall{
		ID:   "call-abc",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path": "test.txt"}`,
		},
	}

	if tc.ID != "call-abc" {
		t.Errorf("Expected ID 'call-abc', got %s", tc.ID)
	}
	if tc.Function.Name != "read_file" {
		t.Errorf("Expected function name 'read_file', got %s", tc.Function.Name)
	}
}

func TestChatResponseStructure(t *testing.T) {
	resp := ChatResponse{
		Content: "I've created the file for you.",
		ToolCalls: []ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "write_file",
					Arguments: `{"path": "test.txt", "content": "hello"}`,
				},
			},
		},
		Done: true,
	}

	if resp.Content == "" {
		t.Error("Expected non-empty content")
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if !resp.Done {
		t.Error("Expected Done to be true")
	}
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "I'll help you with that.",
		ToolCalls: []ToolCall{
			{
				ID:       "call-123",
				Type:     "function",
				Function: ToolCallFunction{Name: "write_file", Arguments: `{"path": "test.txt"}`},
			},
		},
	}

	if len(msg.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Function.Name != "write_file" {
		t.Errorf("Expected function name 'write_file', got %s", msg.ToolCalls[0].Function.Name)
	}
}

func TestMessageToolResponse(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    `{"success": true, "output": "file created"}`,
		ToolCallID: "call-123",
	}

	if msg.Role != "tool" {
		t.Errorf("Expected role 'tool', got %s", msg.Role)
	}
	if msg.ToolCallID != "call-123" {
		t.Errorf("Expected ToolCallID 'call-123', got %s", msg.ToolCallID)
	}
}

func TestGetOptimalContextSize(t *testing.T) {
	size := GetOptimalContextSize()
	
	// Should return a valid context size
	validSizes := []int{2048, 4096, 8192, 16384, 32768}
	found := false
	for _, valid := range validSizes {
		if size == valid {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected valid context size (2048, 4096, 8192, 16384, or 32768), got %d", size)
	}
	
	// Should be at least 2048
	if size < 2048 {
		t.Errorf("Expected context size >= 2048, got %d", size)
	}
}

func TestConfigContextSize(t *testing.T) {
	// Default config should have 0 (auto-detect)
	cfg := DefaultConfig()
	if cfg.ContextSize != 0 {
		t.Errorf("Expected default ContextSize 0 (auto), got %d", cfg.ContextSize)
	}
	
	// Can set custom context size
	cfg = &Config{
		BaseURL:     "http://localhost:11434",
		Model:       "gemma3:12b",
		Timeout:     60,
		ContextSize: 8192,
	}
	if cfg.ContextSize != 8192 {
		t.Errorf("Expected ContextSize 8192, got %d", cfg.ContextSize)
	}
}