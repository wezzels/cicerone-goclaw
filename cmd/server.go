// Package cmd provides CLI commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/agent"
	"github.com/crab-meat-repos/cicerone-goclaw/llm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP server for Cicerone API",
	Long: `Start an HTTP server that provides an OpenAI-compatible API
for chat completions with tool execution capabilities.

Endpoints:
  GET  /health              - Health check
  GET  /v1/models           - List available models
  POST /v1/chat/completions - Chat completions with tool support
  POST /execute             - Execute shell command
  GET  /vm/status           - VM status (uptime, memory, disk)`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntP("port", "p", 18789, "Port to listen on")
	serverCmd.Flags().StringP("host", "H", "127.0.0.1", "Host to bind to")
}

// ChatRequest represents an OpenAI-compatible chat request.
type ChatRequest struct {
	Model    string          `json:"model"`
	Messages []llm.Message   `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	Tools    []llm.Tool      `json:"tools,omitempty"`
}

// ChatResponse represents an OpenAI-compatible chat response.
type ChatResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
}

// Choice represents a response choice.
type Choice struct {
	Index        int          `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// ResponseMessage represents a response message.
type ResponseMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []ToolCallResponse `json:"tool_calls,omitempty"`
}

// ToolCallResponse represents a tool call in the response.
type ToolCallResponse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	} `json:"function"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelsResponse represents the models endpoint response.
type ModelsResponse struct {
	Object string        `json:"object"`
	Data   []ModelData   `json:"data"`
}

// ModelData represents a model.
type ModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ExecuteRequest represents an execute request.
type ExecuteRequest struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// ExecuteResponse represents an execute response.
type ExecuteResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// VMStatusResponse represents VM status.
type VMStatusResponse struct {
	Uptime   string `json:"uptime"`
	Memory   MemoryInfo `json:"memory"`
	Disk     DiskInfo `json:"disk"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
}

// MemoryInfo represents memory information.
type MemoryInfo struct {
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
	Percent   float64 `json:"percent"`
}

// DiskInfo represents disk information.
type DiskInfo struct {
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
	Percent   float64 `json:"percent"`
}

var (
	serverProvider llm.Provider
	serverAgent    *agent.Agent
	serverTools    []llm.Tool
)

func runServer(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")

	// Get model from config
	model := viper.GetString("llm.model")
	if model == "" {
		model = "gemma3:12b"
	}

	// Get provider URL from config
	baseURL := viper.GetString("llm.base_url")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Create provider
	cfg := &llm.Config{
		BaseURL: baseURL,
		Model:   model,
		Timeout: 300,
	}
	serverProvider = llm.NewProvider(cfg)

	// Check if provider is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !serverProvider.IsRunning(ctx) {
		return fmt.Errorf("LLM provider not running at %s", baseURL)
	}

	// Create agent
	workDir, _ := os.Getwd()
	serverAgent = agent.New(workDir)

	// Get tool definitions
	serverTools = agent.ToolsToOllamaFormat()

	// Setup routes
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/v1/models", handleModels)
	http.HandleFunc("/v1/chat/completions", handleChatCompletions)
	http.HandleFunc("/execute", handleExecute)
	http.HandleFunc("/vm/status", handleVMStatus)

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Cicerone server starting on %s", addr)
	log.Printf("Model: %s", model)
	log.Printf("Provider: %s", baseURL)

	return http.ListenAndServe(addr, nil)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "2.0.0",
		"type":    "cicerone-goclaw",
	})
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get models from provider
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := serverProvider.Models(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := make([]ModelData, len(models))
	for i, m := range models {
		data[i] = ModelData{
			ID:      m.Name,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "ollama",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ModelsResponse{
		Object: "list",
		Data:   data,
	})
}

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use model from request or default
	model := req.Model
	if model == "" {
		model = viper.GetString("llm.model")
		if model == "" {
			model = "gemma3:12b"
		}
	}

	// Check for streaming
	if req.Stream {
		handleStreamingChat(w, r, &req, model)
		return
	}

	// Non-streaming response
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Use tools from request or default
	tools := req.Tools
	if len(tools) == 0 {
		tools = serverTools
	}

	// Call LLM
	resp, err := serverProvider.ChatWithTools(ctx, req.Messages, tools)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response
	response := ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index: 0,
				Message: ResponseMessage{
					Role:    "assistant",
					Content: resp.Content,
				},
				FinishReason: "stop",
			},
		},
	}

	// Add tool calls if present
	if len(resp.ToolCalls) > 0 {
		toolCalls := make([]ToolCallResponse, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			toolCalls[i] = ToolCallResponse{
				ID:   tc.ID,
				Type: "function",
			}
			toolCalls[i].Function.Name = tc.Function.Name
			toolCalls[i].Function.Arguments = tc.Function.RawArguments
		}
		response.Choices[0].Message.ToolCalls = toolCalls
		response.Choices[0].FinishReason = "tool_calls"
	}

	// Execute tools if present
	if len(resp.ToolCalls) > 0 {
		// Convert to agent tool calls
		toolCalls := make([]agent.ToolCall, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			var args map[string]interface{}
			if tc.Function.Arguments != "" {
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
			}
			toolCalls[i] = agent.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}

		// Execute tools
		exec := agent.NewExecutor(serverAgent)
		results := exec.ExecuteTools(ctx, toolCalls)

		// Build tool results message
		var resultContent strings.Builder
		for _, r := range results {
			resultContent.WriteString(fmt.Sprintf("Tool: %s\n", r.Name))
			if r.Success {
				resultContent.WriteString(fmt.Sprintf("Result: %s\n", r.Output))
			} else {
				resultContent.WriteString(fmt.Sprintf("Error: %v\n", r.Error))
			}
			resultContent.WriteString("\n")
		}

		// Add tool results to response
		response.Choices[0].Message.Content = resultContent.String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStreamingChat(w http.ResponseWriter, r *http.Request, req *ChatRequest, model string) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Use tools from request or default
	tools := req.Tools
	if len(tools) == 0 {
		tools = serverTools
	}

	// Call LLM with streaming
	stream, err := serverProvider.ChatStream(ctx, req.Messages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for chunk := range stream {
		if chunk.Error != nil {
			break
		}

		// Send SSE event
		data, _ := json.Marshal(map[string]interface{}{
			"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{
						"content": chunk.Text,
					},
					"finish_reason": nil,
				},
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if chunk.Done {
			break
		}
	}

	// Send done event
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 120
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Execute command
	output, err := serverAgent.Execute(ctx, req.Command)

	response := ExecuteResponse{
		Success: err == nil,
		Output:  output,
	}

	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleVMStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get uptime
	uptime := getUptime()

	// Get hostname
	hostname, _ := os.Hostname()

	// Get memory info
	memInfo := getMemoryInfo()

	// Get disk info
	diskInfo := getDiskInfo()

	// Get OS info
	osInfo := getOSInfo()

	response := VMStatusResponse{
		Uptime:   uptime,
		Memory:   memInfo,
		Disk:     diskInfo,
		Hostname: hostname,
		OS:       osInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getUptime() string {
	// Read /proc/uptime
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}

	var uptimeSeconds float64
	fmt.Sscanf(string(data), "%f", &uptimeSeconds)

	duration := time.Duration(uptimeSeconds * float64(time.Second))
	return duration.Round(time.Second).String()
}

func getMemoryInfo() MemoryInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Also try to read from /proc/meminfo for system memory
	total := uint64(0)
	available := uint64(0)
	used := uint64(0)

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d", &total)
				total *= 1024 // Convert from KB to bytes
			} else if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(line, "MemAvailable: %d", &available)
				available *= 1024
			} else if strings.HasPrefix(line, "MemFree:") && available == 0 {
				var free uint64
				fmt.Sscanf(line, "MemFree: %d", &free)
				free *= 1024
				// MemAvailable is better, but MemFree is a fallback
			}
		}
	}

	if total > 0 {
		used = total - available
		percent := float64(used) / float64(total) * 100
		return MemoryInfo{
			Total:     total,
			Used:      used,
			Available: available,
			Percent:   percent,
		}
	}

	// Fallback to runtime memory
	return MemoryInfo{
		Total:     m.Sys,
		Used:      m.HeapInuse,
		Available: m.Sys - m.HeapInuse,
		Percent:   float64(m.HeapInuse) / float64(m.Sys) * 100,
	}
}

func getDiskInfo() DiskInfo {
	// Use df command to get disk usage
	cmd := exec.Command("df", "-B1", "/")
	output, err := cmd.Output()
	if err != nil {
		return DiskInfo{}
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return DiskInfo{}
	}

	// Parse the second line (first line is header)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return DiskInfo{}
	}

	var total, used, available uint64
	fmt.Sscanf(fields[1], "%d", &total)
	fmt.Sscanf(fields[2], "%d", &used)
	fmt.Sscanf(fields[3], "%d", &available)

	percent := float64(0)
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return DiskInfo{
		Total:     total,
		Used:      used,
		Available: available,
		Percent:   percent,
	}
}

func getOSInfo() string {
	// Read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name := strings.TrimPrefix(line, "PRETTY_NAME=")
			name = strings.Trim(name, "\"")
			return name
		}
	}

	return "Linux"
}

// Ensure we read all of the request body
var _ io.Reader = nil