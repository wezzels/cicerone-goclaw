# Chat Agentic Capabilities

The Cicerone chat supports agentic AI assistant tasks with the following capabilities:

## Web Commands

| Command | Description |
|---------|-------------|
| `/search <query>` | Search the web using DuckDuckGo |
| `/fetch <url>` | Fetch content from a URL |
| `/web <query>` | Search and include results in LLM context |

## Shell Commands (Agentic Mode)

| Command | Description |
|---------|-------------|
| `/run <command>` | Execute a shell command |
| `/cd <path>` | Change working directory |
| `/pwd` | Show current directory |
| `/ls [path]` | List directory contents |
| `/cat <file>` | Display file contents |
| `/write <file>` | Write to file (use `:wq` to save, `:q` to cancel) |
| `/edit <file>` | Edit file interactively |
| `/rm <file>` | Remove a file |
| `/mkdir <dir>` | Create directory |
| `/touch <file>` | Create empty file |

## API Commands

| Command | Description |
|---------|-------------|
| `/get <url>` | HTTP GET request |
| `/post <url> <json>` | HTTP POST request |
| `/put <url> <json>` | HTTP PUT request |
| `/delete <url>` | HTTP DELETE request |

## Code Execution

| Command | Description |
|---------|-------------|
| `/go <code>` | Execute Go code |
| `/python <code>` | Execute Python code (if installed) |
| `/shell <script>` | Execute shell script |

## Agent Mode

| Command | Description |
|---------|-------------|
| `/agent` | Enable autonomous agent mode |
| `/stop` | Stop agent mode |
| `/task <description>` | Give agent a task to complete |

## Security

- Commands run in the current directory by default
- Dangerous commands require confirmation
- File operations are sandboxed to workspace
- API calls have configurable timeouts

## Examples

```bash
# Search the web
You: /search golang tutorials

# Run a command
You: /run ls -la

# Write a file
You: /write hello.txt
Enter content (Ctrl+D to save):
Hello, World!

# HTTP GET
You: /get https://api.example.com/data

# Agent task
You: /agent
You: Create a new Go project with a main.go file
```