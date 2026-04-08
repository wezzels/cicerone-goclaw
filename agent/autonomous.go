// Package agent provides autonomous agent capabilities.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AutonomousAgent runs tasks autonomously with planning and iteration.
type AutonomousAgent struct {
	executor  *Executor
	agent     *Agent
	maxSteps  int
}

// TaskResult represents the result of autonomous task execution.
type TaskResult struct {
	Task        string
	Completed   bool
	Steps       []StepResult
	FinalOutput string
	Error      error
}

// StepResult represents a single step in task execution.
type StepResult struct {
	StepNumber  int
	Plan        string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
	Reasoning   string
}

// NewAutonomousAgent creates a new autonomous agent.
func NewAutonomousAgent(ag *Agent) *AutonomousAgent {
	return &AutonomousAgent{
		executor: NewExecutor(ag),
		agent:    ag,
		maxSteps: 10,
	}
}

// SetMaxSteps sets the maximum number of iterations.
func (a *AutonomousAgent) SetMaxSteps(max int) {
	a.maxSteps = max
}

// ExecuteTask runs a task autonomously with the LLM provider.
func (a *AutonomousAgent) ExecuteTask(ctx context.Context, task string, onProgress func(string), chatFn func(ctx context.Context, messages []ChatMessage) (string, error)) (*TaskResult, error) {
	result := &TaskResult{
		Task: task,
	}

	// Build system prompt with tools
	systemPrompt := a.buildSystemPrompt()

	// Initial messages
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Task: %s\n\nPlease complete this task. Break it down into steps, use tools as needed, and iterate until complete.", task)},
	}

	for step := 1; step <= a.maxSteps; step++ {
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			return result, ctx.Err()
		default:
		}

		if onProgress != nil {
			onProgress(fmt.Sprintf("Step %d/%d: Planning...", step, a.maxSteps))
		}

		// Get LLM response
		response, err := chatFn(ctx, messages)
		if err != nil {
			result.Error = err
			return result, err
		}

		// Parse tool calls from response
		toolCalls, reasoning := a.parseResponse(response)

		stepResult := StepResult{
			StepNumber: step,
			Reasoning:  reasoning,
			ToolCalls:  toolCalls,
		}

		// Check if task is complete
		if len(toolCalls) == 0 && strings.Contains(response, "TASK_COMPLETE") {
			result.Completed = true
			result.FinalOutput = a.extractFinalOutput(response)
			result.Steps = append(result.Steps, stepResult)
			if onProgress != nil {
				onProgress("Task completed!")
			}
			return result, nil
		}

		// No tool calls and not complete - ask for next step
		if len(toolCalls) == 0 {
			// LLM didn't make tool calls and didn't complete
			// Add assistant response and prompt for action
			messages = append(messages, ChatMessage{Role: "assistant", Content: response})
			messages = append(messages, ChatMessage{Role: "user", Content: "You haven't made any tool calls. Either use tools to make progress on the task, or if the task is truly complete, respond with TASK_COMPLETE followed by your final output."})
			result.Steps = append(result.Steps, stepResult)
			continue
		}

		// Execute tool calls
		if onProgress != nil {
			toolNames := make([]string, len(toolCalls))
			for i, tc := range toolCalls {
				toolNames[i] = tc.Name
			}
			onProgress(fmt.Sprintf("Step %d: Executing %s", step, strings.Join(toolNames, ", ")))
		}

		toolResults := a.executor.ExecuteTools(ctx, toolCalls)
		stepResult.ToolResults = toolResults
		result.Steps = append(result.Steps, stepResult)

		// Add assistant response with tool calls
		messages = append(messages, ChatMessage{Role: "assistant", Content: response})

		// Add tool results as user message
		resultsMsg := FormatToolResults(toolResults)
		messages = append(messages, ChatMessage{Role: "user", Content: resultsMsg + "\n\nContinue with the next step, or if the task is complete, respond with TASK_COMPLETE followed by your final output."})
	}

	// Max steps reached
	result.Error = fmt.Errorf("maximum steps (%d) reached", a.maxSteps)
	return result, nil
}

// parseResponse extracts tool calls and reasoning from LLM response.
func (a *AutonomousAgent) parseResponse(response string) ([]ToolCall, string) {
	// Try JSON format first
	calls, err := ParseToolCalls(response)
	if err == nil && len(calls) > 0 {
		return calls, ""
	}

	// Try embedded JSON format: {"tool_call": {...}}
	var embedded struct {
		ToolCall struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"tool_call"`
	}
	if err := jsonUnmarshal(response, &embedded); err == nil && embedded.ToolCall.Name != "" {
		return []ToolCall{{
			Name:      embedded.ToolCall.Name,
			Arguments: embedded.ToolCall.Arguments,
		}}, ""
	}

	// Try multiple tool calls format
	var multiCall struct {
		ToolCalls []struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"tool_calls"`
	}
	if err := jsonUnmarshal(response, &multiCall); err == nil && len(multiCall.ToolCalls) > 0 {
		calls := make([]ToolCall, len(multiCall.ToolCalls))
		for i, tc := range multiCall.ToolCalls {
			calls[i] = ToolCall{
				Name:      tc.Name,
				Arguments: tc.Arguments,
			}
		}
		return calls, ""
	}

	return nil, response
}

// extractFinalOutput extracts the final output from a TASK_COMPLETE response.
func (a *AutonomousAgent) extractFinalOutput(response string) string {
	idx := strings.Index(response, "TASK_COMPLETE")
	if idx == -1 {
		return response
	}
	output := strings.TrimSpace(response[idx+len("TASK_COMPLETE"):])
	return output
}

// buildSystemPrompt creates the system prompt with tools.
func (a *AutonomousAgent) buildSystemPrompt() string {
	toolsJSON, _ := ToolsToJSON()

	return fmt.Sprintf(`You are an autonomous agent with access to tools. Your job is to complete tasks by:
1. Breaking down the task into steps
2. Using tools to accomplish each step
3. Evaluating results and iterating as needed
4. Completing when the task is done

Available tools (use JSON format to call them):

%s

TOOL CALL FORMAT:
You can call tools using JSON format:
Single call: {"name": "tool_name", "arguments": {"arg1": "value1"}}
Multiple calls: [{"name": "tool1", "arguments": {...}}, {"name": "tool2", "arguments": {...}}]

WORKFLOW:
1. Analyze the task and create a plan
2. Call tools to execute each step
3. Review tool results
4. Adjust plan if needed
5. Continue until complete

When the task is complete, respond with:
TASK_COMPLETE
<your final summary or output>

IMPORTANT:
- Always use actual tool calls (JSON format) to take action
- Do not describe what you would do - actually call the tools
- After tool results, decide: continue with more tools or complete
- Be thorough but efficient
- If a tool fails, try alternative approaches

Current working directory: %s`, toolsJSON, a.agent.WorkDir())
}

// ChatMessage represents a chat message for the autonomous agent.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// jsonUnmarshal is a helper to handle JSON parsing.
func jsonUnmarshal(data string, v interface{}) error {
	// Extract JSON from text if embedded
	start := strings.Index(data, "{")
	if start == -1 {
		return fmt.Errorf("no JSON found")
	}

	// Find matching brace
	depth := 0
	end := -1
	for i, ch := range data[start:] {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				end = start + i + 1
				break
			}
		}
	}

	if end == -1 {
		return fmt.Errorf("unmatched braces")
	}

	jsonData := data[start:end]
	return jsonUnmarshalStrict(jsonData, v)
}

func jsonUnmarshalStrict(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}