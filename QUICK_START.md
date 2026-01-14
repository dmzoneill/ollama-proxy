# Quick Start Guide

## TL;DR

**The proxy config tells the router what models each backend CAN run. You still need to:**

1. ✅ Start 4 Ollama instances (one per hardware)
2. ✅ Pull models on each instance
3. ✅ Start the proxy

## What's Already Configured ✅

The proxy knows:
- ✅ NPU can run 0.5b-1.5b models (max 2GB)
- ✅ Intel GPU can run up to 7b models (max 8GB)
- ✅ NVIDIA can run all models including 70b (max 24GB)
- ✅ CPU can run up to 7b models (max 16GB, fallback)

**BUT:** The proxy doesn't load models - it only routes to them!

## What You Need to Do ❌

### Step 1: Start 4 Ollama Instances

Open 4 terminals:

```bash
# Terminal 1: NPU (port 11434)
OLLAMA_HOST=0.0.0.0:11434 ollama serve

# Terminal 2: Intel GPU (port 11435)
OLLAMA_HOST=0.0.0.0:11435 OLLAMA_INTEL_GPU=1 ollama serve

# Terminal 3: NVIDIA (port 11436)
OLLAMA_HOST=0.0.0.0:11436 ollama serve

# Terminal 4: CPU (port 11437)
OLLAMA_HOST=0.0.0.0:11437 OLLAMA_NUM_GPU=0 ollama serve
```

### Step 2: Load Models

**Option A: Automated (Recommended)**
```bash
./scripts/setup-models.sh
```

**Option B: Manual**
```bash
# NPU - Tiny models
OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:0.5b
OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:1.5b

# Intel GPU - Medium models
OLLAMA_HOST=http://localhost:11435 ollama pull llama3:7b
OLLAMA_HOST=http://localhost:11435 ollama pull mistral:7b

# NVIDIA - Large models
OLLAMA_HOST=http://localhost:11436 ollama pull llama3:70b
OLLAMA_HOST=http://localhost:11436 ollama pull llama3:7b

# CPU - Fallback
OLLAMA_HOST=http://localhost:11437 ollama pull qwen2.5:1.5b
```

### Step 3: Start the Proxy

```bash
./bin/ollama-proxy
```

**Expected output:**
```
🚀 Starting Ollama Compute Proxy with Thermal Monitoring...
🌡️  Thermal monitoring started
🎛️  Efficiency mode: Balanced
🔥 Using thermal-aware routing
✅ Backend ollama-npu healthy (npu at http://localhost:11434) [55.0°C, fan:0%]
✅ Backend ollama-igpu healthy (igpu at http://localhost:11435) [62.0°C, fan:35%]
✅ Backend ollama-nvidia healthy (nvidia at http://localhost:11436) [65.0°C, fan:45%]
✅ Backend ollama-cpu healthy (cpu at http://localhost:11437) [72.0°C, fan:45%]
```

## Test It

### Test 1: Realtime Audio (Should Route to NPU)
```bash
grpcurl -plaintext -d '{
  "prompt": "Realtime voice transcription",
  "model": "qwen2.5:0.5b",
  "annotations": {
    "latency_critical": true
  }
}' localhost:50051 compute.v1.ComputeService/Generate
```

Expected: Routes to NPU (3W power, ultra-efficient)

### Test 2: Code Generation (Should Route to NVIDIA)
```bash
grpcurl -plaintext -d '{
  "prompt": "Write a complex Python algorithm",
  "model": "llama3:70b",
  "annotations": {
    "media_type": "code"
  }
}' localhost:50051 compute.v1.ComputeService/Generate
```

Expected: Routes to NVIDIA (only one that can run 70b)

### Test 3: General Chat (Should Route to Intel GPU)
```bash
grpcurl -plaintext -d '{
  "prompt": "Tell me about artificial intelligence",
  "model": "llama3:7b"
}' localhost:50051 compute.v1.ComputeService/Generate
```

Expected: Routes to Intel GPU (balanced, 12W)

## Model Distribution

```
┌─────────────┬──────────────┬───────────────────────────┐
│  Backend    │  Power       │  Models Loaded            │
├─────────────┼──────────────┼───────────────────────────┤
│  NPU        │  3W          │  qwen2.5:0.5b, 1.5b       │
│             │  (Silent)    │  tinyllama:1b             │
├─────────────┼──────────────┼───────────────────────────┤
│  Intel GPU  │  12W         │  llama3:7b, mistral:7b    │
│             │  (Quiet)     │  qwen2.5:1.5b             │
├─────────────┼──────────────┼───────────────────────────┤
│  NVIDIA     │  55W         │  llama3:70b, llama3:7b    │
│             │  (Loud)      │                           │
├─────────────┼──────────────┼───────────────────────────┤
│  CPU        │  28W         │  qwen2.5:1.5b, llama3:7b  │
│             │  (Fallback)  │                           │
└─────────────┴──────────────┴───────────────────────────┘
```

## Automatic Routing Examples

The proxy automatically routes based on:
1. Model compatibility (what backend has the model)
2. Workload type (realtime → NPU, code → NVIDIA)
3. Thermal state (avoids hot backends)
4. Efficiency mode (Quiet blocks NVIDIA if fan > 40%)

### Example Routing Decisions

**Scenario 1: Realtime Audio**
```
Prompt: "Realtime voice transcription"
Model: qwen2.5:0.5b
→ NPU (perfect match, 3W)
```

**Scenario 2: Complex Code**
```
Prompt: "Generate complex algorithm"
Model: llama3:70b
→ NVIDIA (only one that can run 70b)
```

**Scenario 3: General Chat**
```
Prompt: "What's the weather?"
Model: llama3:7b
→ Intel GPU (balanced, 12W)
```

**Scenario 4: Quiet Mode Override**
```
Prompt: "Write code"
Model: llama3:70b
Mode: Quiet (blocks NVIDIA)
→ Model substituted to llama3:7b
→ Intel GPU (complies with Quiet mode)
```

## Troubleshooting

### "No backend supports model X"
- Check model is pulled on at least one backend
- Check model name matches exactly
- Use `ollama list` on each backend

### "No thermally healthy backends"
- Check thermal monitoring output
- NVIDIA might be too hot (> 85°C)
- Try switching to Performance mode

### Backend shows as unhealthy
```bash
# Check the backend is running
curl http://localhost:11434/api/tags  # NPU
curl http://localhost:11435/api/tags  # Intel GPU
curl http://localhost:11436/api/tags  # NVIDIA
curl http://localhost:11437/api/tags  # CPU
```

## Summary

**What's Automatic:**
- ✅ Routing based on model compatibility
- ✅ Thermal monitoring
- ✅ Efficiency mode enforcement
- ✅ Workload type detection

**What You Must Do:**
- ❌ Start Ollama instances (4 terminals)
- ❌ Pull models on each instance
- ❌ Keep instances running

**Use the setup script to automate model pulling:**
```bash
./scripts/setup-models.sh
```

See `BACKEND_SETUP_GUIDE.md` for detailed instructions.
