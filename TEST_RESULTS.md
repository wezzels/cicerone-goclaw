# Cicerone-goclaw Test Results

**Date:** 2026-04-08  
**Test:** Build, autonomous agent, tool calling on both machines

## Machine Specifications

| Metric | thing1 (Local) | darth (Remote) |
|--------|---------------|-----------------|
| **Hostname** | thing1 | darth |
| **CPU** | AMD Ryzen 5 5600H | AMD Ryzen 7 7800X3D |
| **Cores** | 6 cores / 12 threads | 8 cores / 16 threads |
| **RAM** | 54 GB | 124 GB |
| **Go Version** | go1.25.0 | go1.25.0 |
| **Model** | mistral:latest | llama3.1:8b |
| **Tool Support** | ✅ Yes | ✅ Yes |

## Go Build Performance

| Test | thing1 | darth | Winner |
|------|--------|-------|--------|
| **Build (Cached)** | 0.454s | 0.320s | darth 30% faster |
| **Tests** | 0.447s | 0.305s | darth 32% faster |
| **Binary Size** | 15 MB | 15 MB | Tie |

## Test Results

### ✅ All Tests Passing

| Test | thing1 | darth | Notes |
|------|--------|-------|-------|
| **Go Build** | ✅ Pass | ✅ Pass | Both compile cleanly |
| **Go Test Suite** | ✅ Pass (46 tests) | ✅ Pass | All unit tests pass |
| **Hello World Compilation** | ✅ Pass | ✅ Pass | GCC compiles and runs |
| **Pi 500 Digits** | ✅ Pass | ✅ Pass | Both output correct |
| **Direct Commands** | ✅ Pass | ✅ Pass | /write, /read, /run work |
| **Tool Calling API** | ✅ Pass | ✅ Pass | Ollama returns tool_calls |
| **Autonomous Agent** | ✅ Pass | ✅ Pass | Multi-step tasks work |

### Autonomous Agent Test Results

**Test:** `/task Create a file /tmp/autonomous_test.txt with content 'Test'`

**thing1 (mistral:latest):**
```
Step 1: LLM requested tools: [create_directory write_file]
Step 1: Executing create_directory, write_file
Step 2: LLM responded: The file has been successfully created...
Task completed successfully!
```

**darth (llama3.1:8b):**
```
Step 1: LLM requested tools: [write_file]
Step 1: Executing write_file
Step 2: LLM responded: The tool call was successful...
Task completed successfully!
```

## Configuration

### thing1 (~/.cicerone/config.yaml)
```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: mistral:latest
  timeout: 300
  context_size: 0  # auto-detect
```

### darth (~/.cicerone/config.yaml)
```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.1:8b
  timeout: 300
  context_size: 0  # auto-detect
```

## Models with Tool Support

| Model | Tool Support | Used On |
|-------|-------------|---------|
| mistral:latest | ✅ Yes | thing1 |
| llama3.1:8b | ✅ Yes | darth |
| gemma3:12b | ❌ No | (deprecated) |

## Key Fix: Multi-step Task Continuation

**Problem:** Multi-step autonomous tasks failed with `400 Bad Request` when sending tool results back to Ollama.

**Root Cause:** Ollama expects tool call `arguments` as an object (`{}`), not a JSON string (`"{...}"`).

**Solution:** 
- Added `RawArguments map[string]interface{}` field with `json:"arguments"` tag
- Custom `MarshalJSON()` outputs arguments as object for Ollama
- Preserved string `Arguments` for internal use

**Commit:** `9311f80` - fix: Multi-step autonomous task continuation now works

---

*Last updated: 2026-04-08 20:42 UTC*

## Coding Challenge Results

See `agent/PROMPT_EVOLUTION.md` for detailed prompt evolution and fixes.

### Challenge 1: Project Scaffold

**Task:** Create a Python project structure with `src/__init__.py`, `src/main.py`

| Machine | Model | Status | Time | Notes |
|---------|-------|--------|------|-------|
| thing1 | mistral:latest | ✅ Pass | ~50s | Creates structure correctly |
| darth | llama3.1:8b | ✅ Pass | ~3s | Faster execution |

**Verification:**
```bash
$ ls /tmp/myproject/src/
__init__.py  main.py
$ python3 /tmp/myproject/src/main.py
Hello World
```

### Challenge 2: Data Processing

**Task:** Create numbers.txt with data, create sum.py, run it

| Machine | Model | Status | Time | Notes |
|---------|-------|--------|------|-------|
| thing1 | mistral:latest | ⚠️ Partial | 120s | Creates files but times out |
| darth | llama3.1:8b | ✅ Pass | ~5s | Full workflow works |

**Verification (darth):**
```bash
$ cat /tmp/numbers.txt
10
20
30
40
50
$ python3 /tmp/sum.py
150
```

### Challenge 3: Python Script Creation

**Task:** Create and run a Python script

| Machine | Model | Status | Time | Notes |
|---------|-------|--------|------|-------|
| thing1 | mistral:latest | ✅ Pass | ~63s | Creates and runs correctly |
| darth | llama3.1:8b | ✅ Pass | ~2s | Fast execution |

### Challenge 4: Git Project

**Task:** Initialize git repo, create files, commit

| Machine | Model | Status | Time | Notes |
|---------|-------|--------|------|-------|
| thing1 | mistral:latest | ⏱️ Timeout | 120s | Not completed |
| darth | llama3.1:8b | Not tested | - | - |

---

*Last updated: 2026-04-08 20:42 UTC*