# Cicerone-goclaw Autonomous Agent

A Go-based autonomous agent with native Ollama function calling support.

## Overview

Cicerone-goclaw implements an autonomous agent that can execute tasks using tool calls. It uses Ollama's native function calling API for reliable tool execution.

## Features

- **Native Function Calling**: Uses Ollama's `/api/chat` with `tools` parameter
- **8 Essential Tools**: write_file, read_file, append_file, run_shell, create_directory, list_directory, web_search, web_fetch
- **Multi-step Execution**: Plan → Execute → Evaluate → Iterate loop
- **Progress Callbacks**: Real-time feedback during autonomous execution
- **Context-aware**: Auto-detects context window size based on available memory

## Quick Start

```bash
# Build
go build -o cicerone .

# Chat with autonomous agent
./cicerone chat --model llama3.1:8b

# In chat, use /task command
/task Create a file /tmp/hello.txt with content 'Hello World'
```

## How It Works

### Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   /task Command │────▶│ AutonomousAgent │────▶│  Ollama Provider│
└─────────────────┘     └────────┬────────┘     └────────┬────────┘
                                 │                       │
                                 ▼                       ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │    Executor     │◀────│  Tool Calls     │
                        └────────┬────────┘     └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │   Tool Results  │
                        └────────┬────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │ Continue/Complete│
                        └─────────────────┘
```

### Execution Flow

1. **Receive Task**: User provides task via `/task` command
2. **Plan**: LLM analyzes task and decides which tools to call
3. **Execute**: Tools are executed with LLM-provided arguments
4. **Evaluate**: Results are sent back to LLM for analysis
5. **Iterate**: Steps 2-4 repeat until task is complete or max steps reached
6. **Complete**: LLM responds with `TASK_COMPLETE` marker

### Tool Call Format

Ollama expects tool calls in this format:

```json
{
  "role": "assistant",
  "tool_calls": [
    {
      "id": "call_abc123",
      "type": "function",
      "function": {
        "name": "write_file",
        "arguments": {
          "path": "/tmp/test.txt",
          "content": "Hello"
        }
      }
    }
  ]
}
```

**Important**: The `arguments` field must be an **object**, not a JSON string.

### Tool Response Format

Tool results are sent back as:

```json
{
  "role": "tool",
  "tool_call_id": "call_abc123",
  "content": "Successfully wrote 5 bytes to /tmp/test.txt"
}
```

## Available Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `write_file` | Write content to a file | `path`, `content` |
| `read_file` | Read file contents | `path` |
| `append_file` | Append to a file | `path`, `content` |
| `run_shell` | Execute shell command | `command` |
| `create_directory` | Create directory tree | `path` |
| `list_directory` | List directory contents | `path` |
| `web_search` | Search the web | `query` |
| `web_fetch` | Fetch URL content | `url` |

## Configuration

### ~/.cicerone/config.yaml

```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.1:8b    # Recommended: fast, reliable
  timeout: 60            # Seconds (reduced from 300 for llama3.1)
```

### Recommended Models

| Model | Size | Tool Support | Performance |
|-------|------|--------------|-------------|
| **llama3.1:8b** | 4.9GB | ✅ Yes | ~2s per task (recommended) |
| mistral:latest | 4.4GB | ✅ Yes | ~50s per task |
| gemma3:12b | 8.1GB | ❌ No | N/A |

## History

### Development Timeline

**2026-04-07**: Initial autonomous agent implementation
- Added tool definitions (13 tools)
- Implemented executor with tool parsing
- Basic autonomous loop

**2026-04-08**: Major fixes for multi-step tasks

#### Fix 1: Tool Arguments Marshaling (commit 9311f80)

**Problem**: Multi-step tasks failed with `400 Bad Request` when sending tool results back to Ollama.

**Root Cause**: Ollama expects `arguments` as an object `{}`, not a JSON string `"{...}"`.

**Solution**:
```go
// Before (broken)
type ToolCallFunction struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON string - WRONG
}

// After (fixed)
type ToolCallFunction struct {
    Name         string                 `json:"name"`
    Arguments    string                 `json:"-"` // Internal use
    RawArguments map[string]interface{} `json:"arguments"` // Ollama format
}

func (tcf ToolCallFunction) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        Name      string                 `json:"name"`
        Arguments map[string]interface{} `json:"arguments"`
    }{
        Name:      tcf.Name,
        Arguments: tcf.RawArguments,
    })
}
```

#### Fix 2: Force Tool Calls (commit 5dfd1b1)

**Problem**: LLM sometimes returns text descriptions instead of actual tool calls.

**Solution**: Improved prompt when LLM returns text without tool calls:
```go
// Before (weak)
messages = append(messages, llm.Message{Role: "user", 
    Content: "Use the available tools to complete the task."})

// After (forceful)
messages = append(messages, llm.Message{Role: "user", 
    Content: "You must CALL TOOLS to complete the task. Use the tools NOW."})
```

#### Fix 3: Task Completion Detection (commit 1fe7b2c)

**Problem**: Tasks marked complete prematurely after one tool execution.

**Solution**: Only mark complete with explicit `TASK_COMPLETE` marker:
```go
// Before (too aggressive)
if strings.Contains(resp.Content, "complete") || 
   strings.Contains(resp.Content, "done") {
    result.Completed = true
}

// After (explicit only)
if strings.Contains(resp.Content, "TASK_COMPLETE") {
    result.Completed = true
}
```

#### Model Upgrade (same day)

**Problem**: mistral:latest was slow (~50s per task) and struggled with multi-step tasks.

**Solution**: Installed llama3.1:8b on both machines:
```bash
ollama pull llama3.1:8b
```

Performance improvement: **26x faster** (from ~50s to ~2s per task)

## Testing

### Test Environment

| Machine | CPU | RAM | Model |
|---------|-----|-----|-------|
| thing1 | AMD Ryzen 5 5600H | 54GB | llama3.1:8b |
| darth | AMD Ryzen 7 7800X3D | 124GB | llama3.1:8b |

### Unit Tests

```bash
$ go test ./...
ok      github.com/crab-meat-repos/cicerone-goclaw/agent       0.006s
ok      github.com/crab-meat-repos/cicerone-goclaw/llm        0.008s
ok      github.com/crab-meat-repos/cicerone-goclaw/telegram   0.003s
```

### Integration Tests

#### Test 1: Write and Read File

**Task**: Create a file and read it back.

```
/task Create /tmp/test1.txt with content 'Hello llama3.1'. Read it back.

[Agent] Step 1: LLM requested tools: [write_file, read_file]
[Agent] Step 1: Executing write_file, read_file
[Agent] Step 2: LLM responded: The file '/tmp/test1.txt' contains the text 'Hello llama3.1'.

$ cat /tmp/test1.txt
Hello llama3.1
```

**Result**: ✅ Pass on both machines

#### Test 2: Shell Command

**Task**: Execute a shell command and interpret output.

```
/task Run 'date' command and tell me the day.

[Agent] Step 1: LLM requested tools: [run_shell]
[Agent] Step 1: Executing run_shell
[Agent] Step 2: LLM responded: It's Wednesday, April 8th, 2026.
```

**Result**: ✅ Pass on both machines

#### Test 3: Python Script

**Task**: Create and run a Python script.

```
/task Create a Python script /tmp/test_py.py that prints 'Hello Python'. Run it.

[Agent] Step 1: LLM requested tools: [write_file, run_shell]
[Agent] Step 1: Executing write_file, run_shell
[Agent] Step 2: LLM responded: The script executed successfully.

$ cat /tmp/test_py.py
print('Hello Python')

$ python3 /tmp/test_py.py
Hello Python
```

**Result**: ✅ Pass on both machines

#### Test 4: Data Processing

**Task**: Create files, write a script, and execute it.

```
/task Create /tmp/numbers.txt with lines 10 20 30 40 50. 
       Create /tmp/sum.py that reads the file and prints the sum. 
       Run it.

[Agent] Step 1: LLM requested tools: [write_file, write_file, run_shell]
[Agent] Step 1: Executing write_file, write_file, run_shell
[Agent] Step 2: LLM responded: The sum of the numbers is 150.

$ cat /tmp/numbers.txt
10
20
30
40
50

$ cat /tmp/sum.py
print(sum([int(x) for x in open('/tmp/numbers.txt').read().split()]))

$ python3 /tmp/sum.py
150
```

**Result**: ✅ Pass on both machines

### Performance Comparison

| Metric | mistral:latest | llama3.1:8b | Improvement |
|--------|---------------|-------------|-------------|
| Avg Time | 52s | 2s | **26x faster** |
| Tool Accuracy | 100% | 100% | Same |
| Multi-step Tasks | ⚠️ Partial | ✅ Full | Better |
| Challenge 2 | ⏱️ Timeout | ✅ Pass | Fixed |

## Example Sessions

### Example 1: Simple File Creation

```
You: /task Create /tmp/hello.txt with content 'Hello World'

[Agent] Step 1/10: Planning...
[Agent] Step 1: LLM requested tools: [write_file]
[Agent] Step 1: Executing write_file
[Agent] Step 2/10: Planning...
[Agent] Step 2: LLM responded: The file has been created successfully.
[Agent] Task completed!

==================================================
Task completed successfully!
==================================================

Output:
The file /tmp/hello.txt has been created with content 'Hello World'.

Steps taken: 1
  Step 1: write_file
```

### Example 2: Multi-Step Task

```
You: /task Create a Python project at /tmp/myproject with src/__init__.py 
     and src/main.py that prints 'Hello World'. Run it.

[Agent] Step 1/10: Planning...
[Agent] Step 1: LLM requested tools: [create_directory, create_directory, write_file, write_file]
[Agent] Step 1: Executing create_directory, create_directory, write_file, write_file
[Agent] Step 2/10: Planning...
[Agent] Step 2: LLM requested tools: [run_shell]
[Agent] Step 2: Executing run_shell
[Agent] Step 3/10: Planning...
[Agent] Task completed!

==================================================
Task completed successfully!
==================================================

Output:
The Python project has been created and tested. Running main.py outputs 'Hello World'.

Steps taken: 2
  Step 1: create_directory, create_directory, write_file, write_file
  Step 2: run_shell
```

## Troubleshooting

### Issue: LLM returns text instead of tool calls

**Solution**: Ensure you're using llama3.1:8b or another model with tool support. Check the prompt is not too long (reduces context for reasoning).

### Issue: Multi-step tasks fail

**Solution**: 
1. Check Ollama logs for errors
2. Ensure `RawArguments` is set correctly in tool call messages
3. Verify `type: "function"` is set on each tool call

### Issue: Task marked complete too early

**Solution**: The LLM should respond with `TASK_COMPLETE` when done. If not, the prompt may need adjustment.

## API Reference

### ExecuteTaskWithTools

```go
func (a *AutonomousAgent) ExecuteTaskWithTools(
    ctx context.Context,
    task string,
    onProgress func(string),
    provider llm.Provider,
) (*TaskResult, error)
```

Execute an autonomous task using native function calling.

**Parameters:**
- `ctx`: Context for cancellation
- `task`: Task description in natural language
- `onProgress`: Progress callback (optional)
- `provider`: LLM provider (Ollama)

**Returns:**
- `TaskResult`: Contains steps taken, final output, completion status
- `error`: Non-nil if task failed

### TaskResult

```go
type TaskResult struct {
    Task        string
    Completed   bool
    Steps       []StepResult
    FinalOutput string
    Error       error
}

type StepResult struct {
    StepNumber  int
    ToolCalls   []ToolCall
    ToolResults []ToolResult
    Reasoning   string
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes
4. Run tests: `go test ./...`
5. Submit a pull request

## License

MIT License - See LICENSE file for details.

## References

- [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [Function Calling Guide](https://ollama.com/blog/function-calling)
- [Llama 3.1 Release Notes](https://ai.meta.com/blog/meta-llama-3-1/)

## VM Workspace Management (Deploy)

The `deploy` command provides VM-based workspace management using libvirt/QEMU. This allows isolated execution environments for tasks.

### Prerequisites

```bash
# Install libvirt
sudo apt install libvirt-daemon-system libvirt-clients qemu-kvm

# Add user to libvirt group
sudo usermod -aG libvirt $(whoami)

# Build with libvirt support
CGO_ENABLED=1 go build -tags libvirt -o cicerone .
```

### Deploy Commands

| Command | Description |
|---------|-------------|
| `deploy list` | List all VM workspaces |
| `deploy create <name>` | Create a VM workspace |
| `deploy start <name>` | Start a VM |
| `deploy stop <name>` | Stop a VM |
| `deploy restart <name>` | Restart a VM |
| `deploy status <name>` | Show VM status |
| `deploy shell <name>` | Open SSH shell |
| `deploy exec <name> <cmd>` | Execute command on VM |
| `deploy push <name> <local> <remote>` | Push file to VM |
| `deploy pull <name> <remote> <local>` | Pull file from VM |
| `deploy snapshot <name>` | Manage VM snapshots |
| `deploy keys <name>` | Manage SSH keys |

### VM Snapshots

Snapshots allow saving and restoring VM state:

```bash
# Create snapshot
cicerone deploy snapshot myvm --create --name "before-tests" --description "Before running tests"

# List snapshots
cicerone deploy snapshot myvm --list

# Revert to snapshot
cicerone deploy snapshot myvm --revert --name "before-tests"

# Delete snapshot
cicerone deploy snapshot myvm --delete --name "before-tests"
```

### VM Configuration

Add VMs to `~/.cicerone/config.yaml`:

```yaml
vms:
  dev:
    name: wezzelos-dev
    image: /var/lib/libvirt/images/wezzelos-base.qcow2
    memory: 4096
    vcpus: 2
    network: default
    user: root
    ssh_key: ~/.cicerone/keys/dev
    
  test:
    name: wezzelos-test
    image: /var/lib/libvirt/images/wezzelos-base.qcow2
    memory: 2048
    vcpus: 1
    network: default
    user: root
```

### VM Workflow Examples

```bash
# List available VMs
cicerone deploy list
# NAME      STATE    IP               MEMORY   VCPUS
# debian11  running  192.168.122.164  8192MB   4
# fedora41  running  10.0.20.156      16384MB  4

# Execute command on VM
cicerone deploy exec debian11 "ls -la /workspace"

# Push/pull files
cicerone deploy push debian11 ./local.txt /workspace/remote.txt
cicerone deploy pull debian11 /workspace/remote.txt ./local.txt

# Open SSH shell
cicerone deploy shell debian11
```

### SSH Key Management

```bash
# Generate keys for VM access
cicerone deploy keys debian11 --generate

# Deploy existing key
cicerone deploy keys debian11 --deploy --key ~/.ssh/id_ed25519.pub

# List keys
cicerone deploy keys debian11 --list
```

---

*Last updated: 2026-04-08 21:30 UTC*