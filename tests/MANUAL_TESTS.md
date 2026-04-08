# Manual Test Procedures for Cicerone-GoClaw

## Prerequisites

1. Ollama running: `ollama serve`
2. Model pulled: `ollama pull gemma3:12b` (or model of choice)
3. Binary built: `go build -o cicerone .`

## Test Categories

### A. Basic Chat Functionality

#### A1. Chat Startup
```bash
./cicerone chat
```
**Expected:** Shows connection message, model name, working directory, and prompt.

#### A2. Basic Conversation
```
You: Hello, who are you?
```
**Expected:** LLM responds with a greeting.

#### A3. History Command
```
You: /history
```
**Expected:** Shows conversation history.

#### A4. Clear History
```
You: /clear
You: /history
```
**Expected:** "No history yet." or similar.

---

### B. Agent Commands (Manual Tools)

#### B1. File Write
```
You: /write testfile.txt Hello from cicerone!
```
**Expected:** "Wrote X bytes to testfile.txt"

#### B2. File Read
```
You: /read testfile.txt
```
**Expected:** Shows "Hello from cicerone!"

#### B3. List Directory
```
You: /ls
```
**Expected:** Shows current directory contents with permissions and sizes.

#### B4. Change Directory
```
You: /cd /tmp
You: /pwd
```
**Expected:** "Changed to: /tmp"

#### B5. Shell Execute
```
You: /run echo "test output"
```
**Expected:** Shows "test output"

#### B6. Delete File
```
You: /delete testfile.txt
You: /ls
```
**Expected:** File no longer exists.

---

### C. Web Commands

#### C1. Web Search
```
You: /search golang programming
```
**Expected:** Shows search results with titles and snippets.

#### C2. Web Fetch
```
You: /fetch https://example.com
```
**Expected:** Shows page content (truncated).

#### C3. Web Context
```
You: /web what is Go programming language
```
**Expected:** LLM responds using web search results as context.

---

### D. Autonomous Agent (/task command)

#### D1. Simple File Creation
```
You: /task create a file called autonomous_test.txt with the content "Autonomous agent was here"
```
**Expected:**
1. Shows "[Agent] Step 1/X: Planning..."
2. Shows "[Agent] Step 1: Executing write_file"
3. Shows "Task completed successfully!"
4. File exists with correct content

**Verification:**
```bash
cat autonomous_test.txt
# Should show: Autonomous agent was here
```

#### D2. Multi-Step Task
```
You: /task create a directory called testdir, then create a file inside it called notes.txt with the current date
```
**Expected:**
1. Agent creates directory
2. Agent creates file
3. Task completes successfully

**Verification:**
```bash
ls -la testdir/
cat testdir/notes.txt
```

#### D3. Web Search Task
```
You: /task search the web for "Go programming language" and save a summary to golang_summary.txt
```
**Expected:**
1. Agent uses web_search tool
2. Agent uses write_file tool
3. File contains summary

#### D4. Task Decomposition
```
You: /task write a brief report about ducks and save it as ducks_report.txt
```
**Expected:**
1. Agent plans multiple steps
2. Agent may search web for information
3. Agent writes comprehensive content
4. Task completes

#### D5. Error Recovery
```
You: /task read a file that does not exist named nonexistent.txt
```
**Expected:**
1. Agent attempts to read file
2. Tool fails with error
3. Agent either retries or reports failure

---

### E. Tool-Specific Tests

#### E1. DOCX Creation
```
You: /task create a docx document called report.docx with title "Test Report" and content "This is test content."
```
**Expected:**
1. Agent uses write_docx tool
2. File created (check if valid docx)

#### E2. HTTP GET
```
You: /get https://httpbin.org/get
```
**Expected:** Shows HTTP response with status and body.

#### E3. HTTP POST
```
You: /post https://httpbin.org/post {"test": "data"}
```
**Expected:** Shows HTTP response with posted data echoed back.

---

### F. Edge Cases

#### F1. Empty Task
```
You: /task
```
**Expected:** Error or help message about task usage.

#### F2. Very Long Task
```
You: /task [paste a very long multi-paragraph task]
```
**Expected:** Agent processes without crashing, may hit max steps.

#### F3. Max Steps Limit
```
You: /task [complex task requiring 20+ steps]
```
**Expected:** Agent stops at max steps (default 10), shows incomplete status.

#### F4. Special Characters in Files
```
You: /task create a file called "test'file.txt" with content "special chars"
```
**Expected:** Handles special characters appropriately.

---

### G. Integration Tests

#### G1. Full Workflow
```
You: /task 
1. Search the web for "Go concurrency patterns"
2. Create a directory called research
3. Write a summary file to research/concurrency.txt
4. Verify the file was created
```
**Expected:** Complete multi-step workflow.

#### G2. Chat History Integration
```
You: My favorite color is blue.
You: /task create a file that mentions my favorite color
```
**Expected:** Agent may use context from conversation (depends on implementation).

---

## Test Results Template

```
## Test Run: [DATE]

### Environment
- OS: [Linux/Darwin/Windows]
- Go Version: [go version]
- Ollama Version: [ollama --version]
- Model Used: [e.g., gemma3:12b]

### Results

| Test ID | Description | Pass/Fail | Notes |
|---------|-------------|-----------|-------|
| A1 | Chat Startup | ✅/❌ | |
| A2 | Basic Conversation | ✅/❌ | |
| ... | ... | ... | |

### Issues Found
- [Description of any bugs or unexpected behavior]

### Performance Notes
- [Any performance observations]
```