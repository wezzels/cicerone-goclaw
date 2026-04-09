# Cicerone-goclaw Design Document

## Executive Summary

Cicerone-goclaw is an autonomous agent framework for Go that integrates with Ollama's native function calling API. It enables AI-driven task execution through a set of predefined tools, supporting complex multi-step operations with iterative planning and execution.

## Problem Statement

Traditional chat-based AI systems are limited to text responses. They cannot:
- Execute system commands
- Create, read, or modify files
- Interact with external APIs
- Perform multi-step operations autonomously

Cicerone-goclaw solves this by providing a structured tool-calling interface that allows LLMs to take real actions on the system.

## Goals

### Primary Goals
1. **Autonomous Task Execution** - Complete tasks without human intervention
2. **Tool Extensibility** - Easy to add new tools
3. **Multi-step Planning** - Break complex tasks into steps
4. **Error Recovery** - Handle failures gracefully
5. **Progress Visibility** - Real-time feedback during execution

### Non-Goals
1. **General AI** - Not a general-purpose AI system
2. **GUI Interface** - CLI only (for now)
3. **Cloud Deployment** - Local-only execution

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           CLI Layer                                  │
│                         cmd/chat_cmd.go                              │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Agent Layer                                     │
│  ┌──────────────────────┐  ┌──────────────────────┐                 │
│  │   AutonomousAgent     │  │      Executor        │                 │
│  │  - Task planning      │  │  - Tool execution    │                 │
│  │  - Step iteration     │  │  - Result formatting  │                 │
│  │  - Completion check   │  │  - Error handling     │                 │
│  └──────────┬───────────┘  └──────────┬───────────┘                 │
│             │                          │                              │
└─────────────┼──────────────────────────┼──────────────────────────────┘
              │                          │
              ▼                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Tool Layer                                   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │write_file│ │read_file│ │run_shell│ │web_search│ │web_fetch│       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        LLM Provider Layer                            │
│  ┌──────────────────────┐  ┌──────────────────────┐                 │
│  │    Ollama Provider    │  │   LlamaCPP Provider   │                 │
│  │  - ChatWithTools()    │  │  - ChatWithTools()    │                 │
│  │  - Function calling   │  │  - OpenAI-compatible  │                 │
│  └──────────────────────┘  └──────────────────────┘                 │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Overview

#### 1. AutonomousAgent (`agent/autonomous.go`)

**Responsibility**: Task decomposition and iterative execution

**Key Methods**:
```go
ExecuteTaskWithTools(ctx, task, onProgress, provider) (*TaskResult, error)
```

**Flow**:
1. Build system prompt with available tools
2. Call LLM with tools enabled
3. Parse tool calls from response
4. Execute tools
5. Send results back to LLM
6. Repeat until `TASK_COMPLETE` or max steps

#### 2. Executor (`agent/executor.go`)

**Responsibility**: Tool execution and result formatting

**Key Methods**:
```go
ExecuteTools(ctx, toolCalls) []ToolResult
ExecuteTool(ctx, toolCall) ToolResult
```

**Tools Implemented**:
- File operations: `write_file`, `read_file`, `append_file`, `delete_file`
- Directory: `create_directory`, `list_directory`
- Shell: `run_shell`
- Network: `web_search`, `web_fetch`

#### 3. LLM Provider (`llm/provider.go`, `llm/ollama.go`)

**Responsibility**: Interface with LLM backends

**Key Types**:
```go
type Provider interface {
    ChatWithTools(ctx, messages, tools) (*ChatResponse, error)
}

type Tool struct {
    Type     string
    Function ToolFunction
}

type ToolCall struct {
    ID       string
    Type     string
    Function ToolCallFunction
}
```

#### 4. Tool Definitions (`agent/tools.go`)

**Responsibility**: Define available tools and schemas

**Format**:
```go
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  map[string]ParamSchema
    Required    []string
}
```

### Data Flow

```
User Input (/task)
       │
       ▼
┌─────────────────┐
│ Parse Task      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ LLM Planning    │────▶│ Tool Selection   │
└────────┬────────┘     └────────┬────────┘
         │                         │
         │                         ▼
         │               ┌─────────────────┐
         │               │ Execute Tools   │
         │               └────────┬────────┘
         │                        │
         │                        ▼
         │               ┌─────────────────┐
         │               │ Format Results  │
         │               └────────┬────────┘
         │                        │
         ▼                        ▼
┌─────────────────────────────────────────┐
│         Continue or Complete?            │
│  - More tools needed? → Loop back        │
│  - TASK_COMPLETE? → Return result         │
│  - Max steps? → Return error             │
└─────────────────────────────────────────┘
```

### Key Design Decisions

#### Decision 1: Ollama Arguments Format

**Context**: Ollama expects `arguments` as an object, not a JSON string.

**Decision**: Use `RawArguments map[string]interface{}` with custom marshaling.

**Rationale**:
- Maintains compatibility with Ollama's API
- Preserves internal JSON string representation
- Clean separation between internal and external formats

#### Decision 2: Native Function Calling

**Context**: Could use text parsing or native API.

**Decision**: Use Ollama's native function calling API.

**Rationale**:
- More reliable than text parsing
- Structured output format
- Better error handling
- Supports parallel tool calls

#### Decision 3: Tool Result Format

**Context**: How to format tool results for LLM.

**Decision**: Use `role: "tool"` messages with tool call IDs.

**Rationale**:
- Matches Ollama's expected format
- Maintains conversation context
- Supports multiple tool calls

#### Decision 4: Completion Detection

**Context**: When to consider task complete.

**Decision**: Only on explicit `TASK_COMPLETE` marker.

**Rationale**:
- Prevents premature completion
- Clear termination condition
- LLM controls completion timing

## Security Considerations

### Current State

1. **Local Execution Only** - No network exposure
2. **User-Level Permissions** - Runs as current user
3. **No Sandboxing** - Direct system access

### Recommendations

1. **Add Tool Whitelist** - Restrict available tools
2. **Add Path Restrictions** - Limit file operations
3. **Add Command Filtering** - Block dangerous shell commands
4. **Add Rate Limiting** - Prevent runaway execution

## Performance

### Benchmarks

| Metric | Value |
|--------|-------|
| Go Build (cached) | 0.45s |
| Unit Tests | 0.41s |
| Binary Size | 15 MB |
| Avg Task Time (llama3.1:8b) | ~2s |
| Avg Task Time (mistral:latest) | ~50s |

### Optimization Opportunities

1. **Streaming Responses** - Show progress incrementally
2. **Tool Caching** - Cache repeated tool calls
3. **Parallel Execution** - Execute independent tools concurrently
4. **Context Pruning** - Remove irrelevant conversation history

## Extensibility

### Adding New Tools

```go
// 1. Define tool in tools.go
{
    Name:        "my_tool",
    Description: "Does something useful",
    Parameters: map[string]ParamSchema{
        "arg1": {Type: "string", Description: "First argument"},
    },
    Required: []string{"arg1"},
}

// 2. Implement in executor.go
case "my_tool":
    arg1 := args["arg1"].(string)
    // Do something
    return ToolResult{Name: "my_tool", Output: result}
```

### Adding New Providers

```go
type MyProvider struct {
    // ...
}

func (p *MyProvider) ChatWithTools(ctx context.Context, 
    messages []Message, tools []Tool) (*ChatResponse, error) {
    // Implement provider-specific logic
}
```

## Testing Strategy

### Unit Tests
- Tool definitions validation
- Executor argument parsing
- LLM message formatting

### Integration Tests
- Full autonomous task execution
- Multi-step workflows
- Error handling

### Benchmark Tests
- Build time
- Memory usage
- Task completion time

## Future Work

### Short Term (v1.1)
- [ ] Streaming progress updates
- [ ] Tool result caching
- [ ] Better error messages

### Medium Term (v2.0)
- [ ] Web UI interface
- [ ] Tool marketplace
- [ ] Cloud deployment

### Long Term (v3.0)
- [ ] Multi-agent collaboration
- [ ] Learning from corrections
- [ ] Custom tool creation UI

## References

1. [Ollama Function Calling](https://ollama.com/blog/function-calling)
2. [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
3. [Anthropic Tool Use](https://docs.anthropic.com/claude/docs/tool-use)

---

*Document Version: 1.0*
*Last Updated: 2026-04-08*