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
| **Model** | llama3.1:8b | llama3.1:8b |
| **Tool Support** | ✅ Yes | ✅ Yes |

## Go Build Performance

| Test | thing1 | darth | Winner |
|------|--------|-------|--------|
| **Build (Cached)** | 0.457s | 0.320s | darth 30% faster |
| **Tests** | 0.415s | 0.305s | darth 26% faster |
| **Binary Size** | 15 MB | 15 MB | Tie |

## Test Results

### ✅ All Tests Passing

| Test | thing1 | darth | Notes |
|------|--------|-------|-------|
| **Go Build** | ✅ Pass | ✅ Pass | Both compile cleanly |
| **Go Test Suite** | ✅ Pass (46 tests) | ✅ Pass | All unit tests pass |
| **Write & Read File** | ✅ Pass | ✅ Pass | Multi-step works |
| **Shell Command** | ✅ Pass | ✅ Pass | Executes and interprets |
| **Python Script** | ✅ Pass | ✅ Pass | Creates and runs |
| **Data Processing** | ✅ Pass | ✅ Pass | Creates files, runs script, outputs 150 |

### Autonomous Agent Test Results

**Test:** `/task Create numbers.txt with data, create sum.py, run it`

| Machine | Model | Time | Result |
|---------|-------|------|--------|
| thing1 | llama3.1:8b | ~3s | ✅ Output: 150 |
| darth | llama3.1:8b | ~2s | ✅ Output: 150 |

**Verification:**
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

## Configuration

### thing1 (~/.cicerone/config.yaml)
```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.1:8b
  timeout: 60
```

### darth (~/.cicerone/config.yaml)
```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.1:8b
  timeout: 60
```

## Key Fixes Applied

### Fix 1: Tool Arguments Marshaling (commit 9311f80)
- Ollama expects `arguments` as object `{}`, not JSON string
- Added `RawArguments map[string]interface{}` with `json:"arguments"` tag

### Fix 2: Force Tool Calls (commit 5dfd1b1)
- LLM sometimes returns text descriptions instead of tool calls
- Improved prompt: "You must CALL TOOLS. Do NOT describe what to do."

### Fix 3: Completion Detection (commit 1fe7b2c)
- Tasks marked complete prematurely after one tool execution
- Now only completes on explicit `TASK_COMPLETE` marker

## Models with Tool Support

| Model | Tool Support | Used On | Performance |
|-------|-------------|---------|-------------|
| llama3.1:8b | ✅ Yes | both thing1 & darth | ~2s per task |
| mistral:latest | ✅ Yes | thing1 (backup) | ~50s per task |
| gemma3:12b | ❌ No | (deprecated) | N/A |

## Performance Comparison

| Metric | mistral:latest | llama3.1:8b | Improvement |
|--------|---------------|-------------|-------------|
| **Avg Time** | 52s | 2s | **26x faster** |
| **Tool Accuracy** | 100% | 100% | Same |
| **Multi-step Tasks** | ⚠️ Partial | ✅ Full | Better |

---

*Last updated: 2026-04-08 20:58 UTC*
*Commits: 9311f80, 5dfd1b1, 1fe7b2c, 5ca3668*