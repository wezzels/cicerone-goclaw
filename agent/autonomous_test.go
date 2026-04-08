package agent

import (
	"context"
	"testing"
	"time"
)

// MockLLMProvider for testing
type MockLLMProvider struct {
	responses []string
	callCount int
}

func (m *MockLLMProvider) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if m.callCount >= len(m.responses) {
		return "TASK_COMPLETE No more responses", nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func TestNewAutonomousAgent(t *testing.T) {
	ag := New(".")
	autoAgent := NewAutonomousAgent(ag)

	if autoAgent == nil {
		t.Error("NewAutonomousAgent returned nil")
	}
	if autoAgent.maxSteps != 10 {
		t.Errorf("Expected default maxSteps 10, got %d", autoAgent.maxSteps)
	}
}

func TestAutonomousAgent_SetMaxSteps(t *testing.T) {
	ag := New(".")
	autoAgent := NewAutonomousAgent(ag)

	autoAgent.SetMaxSteps(5)
	if autoAgent.maxSteps != 5 {
		t.Errorf("Expected maxSteps 5, got %d", autoAgent.maxSteps)
	}
}

func TestAutonomousAgent_ExtractFinalOutput(t *testing.T) {
	ag := New(".")
	autoAgent := NewAutonomousAgent(ag)

	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name:     "with marker",
			response: "Here is my work\nTASK_COMPLETE\nThe task is done",
			expected: "The task is done",
		},
		{
			name:     "without marker",
			response: "The task is done",
			expected: "The task is done",
		},
		{
			name:     "empty output",
			response: "TASK_COMPLETE",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := autoAgent.extractFinalOutput(tt.response)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAutonomousAgent_BuildSystemPrompt(t *testing.T) {
	ag := New(".")
	autoAgent := NewAutonomousAgent(ag)

	prompt := autoAgent.buildSystemPrompt()

	if prompt == "" {
		t.Error("buildSystemPrompt returned empty string")
	}

	// Check essential parts
	essentialParts := []string{
		"autonomous agent",
		"tools",
		"TASK_COMPLETE",
		"working directory",
	}

	for _, part := range essentialParts {
		if !containsStr(prompt, part) {
			t.Errorf("System prompt missing: %s", part)
		}
	}
}

func TestAutonomousAgent_ExecuteTask_Timeout(t *testing.T) {
	ag := New(".")
	autoAgent := NewAutonomousAgent(ag)
	autoAgent.SetMaxSteps(2)

	// Mock provider that never completes
	mockProvider := &MockLLMProvider{
		responses: []string{
			`{"name": "write_file", "arguments": {"path": "test.txt", "content": "test"}}`,
			`{"name": "read_file", "arguments": {"path": "test.txt"}}`,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Convert MockLLMProvider to chat function
	chatFn := func(ctx context.Context, msgs []ChatMessage) (string, error) {
		return mockProvider.Chat(ctx, msgs)
	}

	result, err := autoAgent.ExecuteTask(ctx, "never complete", nil, chatFn)

	if err != nil {
		t.Logf("Task returned error (expected for timeout): %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestAutonomousAgent_ExecuteTask_Completion(t *testing.T) {
	ag := New("/tmp")
	autoAgent := NewAutonomousAgent(ag)

	// Mock provider that completes immediately
	mockProvider := &MockLLMProvider{
		responses: []string{
			"TASK_COMPLETE I have completed the task successfully.",
		},
	}

	ctx := context.Background()

	chatFn := func(ctx context.Context, msgs []ChatMessage) (string, error) {
		return mockProvider.Chat(ctx, msgs)
	}

	result, err := autoAgent.ExecuteTask(ctx, "complete immediately", nil, chatFn)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Completed {
		t.Error("Expected task to be completed")
	}
	if result.FinalOutput == "" {
		t.Error("Expected final output to be non-empty")
	}
}

func TestAutonomousAgent_ExecuteTask_WithToolCalls(t *testing.T) {
	ag := New("/tmp")
	autoAgent := NewAutonomousAgent(ag)

	// Mock provider that makes tool calls then completes
	mockProvider := &MockLLMProvider{
		responses: []string{
			`{"name": "write_file", "arguments": {"path": "/tmp/test.txt", "content": "test content"}}`,
			"TASK_COMPLETE File created successfully.",
		},
	}

	ctx := context.Background()

	chatFn := func(ctx context.Context, msgs []ChatMessage) (string, error) {
		return mockProvider.Chat(ctx, msgs)
	}

	result, err := autoAgent.ExecuteTask(ctx, "create a test file", nil, chatFn)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if len(result.Steps) == 0 {
		t.Error("Expected at least one step")
	}
}

func TestAutonomousAgent_MaxStepsReached(t *testing.T) {
	ag := New("/tmp")
	autoAgent := NewAutonomousAgent(ag)
	autoAgent.SetMaxSteps(1)

	// Mock provider that always makes tool calls
	mockProvider := &MockLLMProvider{
		responses: []string{
			`{"name": "write_file", "arguments": {"path": "test.txt", "content": "test"}}`,
		},
	}

	ctx := context.Background()

	chatFn := func(ctx context.Context, msgs []ChatMessage) (string, error) {
		return mockProvider.Chat(ctx, msgs)
	}

	result, _ := autoAgent.ExecuteTask(ctx, "never complete", nil, chatFn)

	if result.Completed {
		t.Error("Task should not complete with max steps")
	}
	if result.Error == nil {
		t.Error("Expected error for max steps reached")
	}
}