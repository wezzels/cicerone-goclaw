# Autonomous Agent Prompt Evolution

This document tracks the evolution of prompts and fixes for the autonomous agent's task completion logic.

## Timeline

### 2026-04-08 Initial State

**Problem:** Multi-step autonomous tasks failed on step 2 with `400 Bad Request` when sending tool results back to Ollama.

**Root Cause:** Ollama expects tool call `arguments` as an object `{}`, not a JSON string `"{...}"`.

---

## Fix 1: Tool Call Arguments Marshaling

**Commit:** `9311f80`

**Problem:** When sending assistant messages back to Ollama after tool execution, the `arguments` field was being marshaled as a JSON string instead of an object.

**Solution:**
```go
// llm/provider.go
type ToolCallFunction struct {
    Name         string                 `json:"name"`
    Arguments    string                 `json:"-"` // Internal (JSON string)
    RawArguments map[string]interface{} `json:"arguments"` // Ollama expects object
}

// MarshalJSON ensures arguments are sent as object for Ollama
func (tcf ToolCallFunction) MarshalJSON() ([]byte, error) {
    return json.Marshal(Alias{
        Name:      tcf.Name,
        Arguments: tcf.RawArguments,
    })
}
```

**Result:** Multi-step tasks now work! The second request with tool results succeeds.

---

## Fix 2: Force Tool Calls Instead of Descriptions

**Commit:** `5dfd1b1`

**Problem:** LLM sometimes returns text descriptions instead of actual tool calls:
```
Step 1: LLM responded: To create the Python script at /tmp/test_py.py, you can use write_file function...
```

**Solution:** Improved prompt when LLM returns text without tool calls:

```go
// OLD - Weak prompt
messages = append(messages, llm.Message{Role: "user", Content: "Use the available tools to complete the task."})

// NEW - Forceful prompt
messages = append(messages, llm.Message{Role: "user", Content: "You must CALL TOOLS to complete the task. Do NOT describe what to do. Use the tools NOW. For example, to create a file: call write_file with path and content parameters."})
```

**Result:** Challenge 1 (Project Scaffold) now works on mistral:latest.

---

## Fix 3: Task Completion Detection

**Commit:** `1fe7b2c`

**Problem:** Tasks were marked complete too early. After one tool execution, any text response would end the task:
```go
// OLD - Too aggressive
if strings.Contains(resp.Content, "complete") || strings.Contains(resp.Content, "done") {
    result.Completed = true
    return result, nil
}
```

**Solution:** Only mark complete with explicit `TASK_COMPLETE` marker:

```go
// NEW - Explicit only
if strings.Contains(resp.Content, "TASK_COMPLETE") {
    result.Completed = true
    return result, nil
}
```

**Result:** Multi-step tasks continue correctly. Challenge 2 (Data Processing) now works on llama3.1:8b.

---

## System Prompts

### Version 1: Minimal (Original)
```go
systemPrompt := "Current working directory: " + a.agent.WorkDir()
```

### Version 2: Tool Guidance (Current)
```go
systemPrompt := `Current working directory: ` + a.agent.WorkDir() + `

You have access to tools. ALWAYS call tools using JSON format to accomplish tasks. 
Do NOT describe what you would do - actually CALL the tools.

When creating files: use write_file (not run_shell with echo).`
```

---

## Test Results by Model

### mistral:latest (thing1)

| Challenge | Status | Notes |
|-----------|--------|-------|
| Write & Read File | ✅ Pass | Single-step, works reliably |
| Shell Command | ✅ Pass | Executes and interprets output |
| Python Script | ✅ Pass | Creates and runs script |
| Project Scaffold | ✅ Pass | Creates directory structure |
| Data Processing | ⚠️ Partial | Creates files but times out on execution |
| Git Project | ⏱️ Timeout | Complex multi-step |

### llama3.1:8b (darth)

| Challenge | Status | Notes |
|-----------|--------|-------|
| Write & Read File | ✅ Pass | 1.5s execution |
| Shell Command | ✅ Pass | 0.85s execution |
| Python Script | ✅ Pass | Creates and runs correctly |
| Project Scaffold | ✅ Pass | Creates structure correctly |
| Data Processing | ✅ **Pass** | Creates files, runs script, outputs 150 |
| Git Project | Not tested | - |

---

## Prompts for Different Scenarios

### First Tool Call Prompt (No tools yet)
```
You must CALL TOOLS to complete the task. Use the tools NOW.
```

### Continue After Tool Execution
```
Continue with the remaining steps. Call the appropriate tools NOW.
```

### Task Completion Marker
```
When the task is complete, respond with:
TASK_COMPLETE
<your final summary or output>
```

---

## Lessons Learned

### 1. Ollama Format Matters
- Tool calls must have `arguments` as object, not string
- Tool responses need `tool_call_id` to match original call
- `type: "function"` must be set on each tool call

### 2. Model Behavior Varies
- **mistral:latest** - Verbose, over-explains, needs forceful prompts
- **llama3.1:8b** - Direct, efficient, follows instructions better

### 3. Completion Detection
- Don't infer completion from text
- Require explicit marker (`TASK_COMPLETE`)
- Let the LLM decide when it's done

### 4. Tool Calling Patterns
- Mistral tends to call `create_directory` before `write_file` (unnecessary)
- Llama3.1 is more direct with tool usage
- Both work correctly when prompted properly

---

## Future Improvements

### Potential Fixes to Try

1. **Timeout handling** - Increase for complex tasks (180s)
2. **Streaming progress** - Show tool execution in real-time
3. **Task decomposition** - Break complex tasks into subtasks
4. **Error recovery** - Retry failed tool calls with adjusted arguments
5. **Model selection** - Auto-select best model for task type

### Known Issues

1. **Mistral timeout on complex tasks** - Needs longer timeout or smaller steps
2. **Git operations** - Not well tested, shell command handling needs work
3. **Error messages** - Not surfaced to user clearly

---

## Configuration

### Recommended Settings

**For llama3.1:8b (fast, reliable):**
```yaml
llm:
  model: llama3.1:8b
  timeout: 60  # Fast execution
```

**For mistral:latest (better reasoning):**
```yaml
llm:
  model: mistral:latest
  timeout: 180  # Allow more time for complex tasks
```

---

*Last updated: 2026-04-08 20:42 UTC*
*Commits: 9311f80, 5dfd1b1, 1fe7b2c*