# Setting Up 4 Ollama Backend Instances

## Overview

You have **4 hardware backends**, each needs its own Ollama instance with appropriate models pulled:

| Backend | Hardware | Port | Models to Load |
|---------|----------|------|----------------|
| **ollama-npu** | Intel NPU | 11434 | Tiny models (0.5b-1.5b) |
| **ollama-igpu** | Intel Arc GPU | 11435 | Small-medium models (up to 7b) |
| **ollama-nvidia** | NVIDIA RTX 4060 | 11436 | All models (including 70b) |
| **ollama-cpu** | CPU | 11437 | Small models (fallback) |

## Current Status

❌ **Models are NOT auto-loaded by the proxy**

The proxy config tells the **routing system** what models each backend can run, but **you must manually pull models** on each Ollama instance.

## Step-by-Step Setup

### 1. Start Ollama Instances on Different Ports

Each Ollama instance needs to run on a different port and use specific hardware.

#### Terminal 1: NPU Instance (Port 11434)
```bash
# NPU uses default port 11434
OLLAMA_HOST=0.0.0.0:11434 \
OLLAMA_NUM_GPU=0 \
OLLAMA_INTEL_GPU=0 \
ollama serve
```

#### Terminal 2: Intel GPU Instance (Port 11435)
```bash
OLLAMA_HOST=0.0.0.0:11435 \
OLLAMA_NUM_GPU=1 \
OLLAMA_INTEL_GPU=1 \
ollama serve
```

#### Terminal 3: NVIDIA GPU Instance (Port 11436)
```bash
OLLAMA_HOST=0.0.0.0:11436 \
CUDA_VISIBLE_DEVICES=0 \
ollama serve
```

#### Terminal 4: CPU Instance (Port 11437)
```bash
OLLAMA_HOST=0.0.0.0:11437 \
OLLAMA_NUM_GPU=0 \
ollama serve
```

### 2. Pull Models on Each Backend

Based on the proxy's `model_capability` config, here's what to pull on each backend:

#### NPU Backend (localhost:11434) - Tiny Models Only

```bash
# Best for NPU (ultra-low power, 3W)
OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:0.5b
OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:1.5b
OLLAMA_HOST=http://localhost:11434 ollama pull tinyllama:1b

# Verify
OLLAMA_HOST=http://localhost:11434 ollama list
```

**Why these models?**
- NPU max: 2GB
- These are 0.5B-1.5B models (~500MB-1.5GB)
- Perfect for realtime audio, simple queries
- Ultra-low power consumption

#### Intel GPU Backend (localhost:11435) - Small to Medium Models

```bash
# Sweet spot for Intel Arc GPU (12W, balanced)
OLLAMA_HOST=http://localhost:11435 ollama pull llama3:7b
OLLAMA_HOST=http://localhost:11435 ollama pull mistral:7b
OLLAMA_HOST=http://localhost:11435 ollama pull qwen2.5:7b

# Also pull small models for fallback
OLLAMA_HOST=http://localhost:11435 ollama pull qwen2.5:1.5b

# Verify
OLLAMA_HOST=http://localhost:11435 ollama list
```

**Why these models?**
- Intel GPU max: 8GB
- 7B models (~4-5GB)
- Good for general text, code, balanced workloads
- Medium power consumption

#### NVIDIA GPU Backend (localhost:11436) - All Models

```bash
# Large models (NVIDIA only)
OLLAMA_HOST=http://localhost:11436 ollama pull llama3:70b
OLLAMA_HOST=http://localhost:11436 ollama pull mixtral:8x7b

# Medium models (also good on NVIDIA)
OLLAMA_HOST=http://localhost:11436 ollama pull llama3:7b
OLLAMA_HOST=http://localhost:11436 ollama pull codellama:7b

# Verify
OLLAMA_HOST=http://localhost:11436 ollama list
```

**Why these models?**
- NVIDIA max: 24GB
- Can run 70B models (~40GB with quantization)
- Best for complex code generation, analysis
- High power consumption (55W)

#### CPU Backend (localhost:11437) - Small Models (Fallback)

```bash
# CPU fallback (slow but reliable)
OLLAMA_HOST=http://localhost:11437 ollama pull qwen2.5:1.5b
OLLAMA_HOST=http://localhost:11437 ollama pull llama3:7b

# Verify
OLLAMA_HOST=http://localhost:11437 ollama list
```

**Why these models?**
- CPU max: 16GB RAM
- Keep it simple (slow inference)
- Emergency fallback only

### 3. Verify All Backends

Test each backend individually:

```bash
# NPU
curl http://localhost:11434/api/tags

# Intel GPU
curl http://localhost:11435/api/tags

# NVIDIA
curl http://localhost:11436/api/tags

# CPU
curl http://localhost:11437/api/tags
```

### 4. Start the Proxy

```bash
cd /home/daoneill/src/ollama-proxy
./bin/ollama-proxy
```

The proxy will:
- ✅ Detect all 4 backends
- ✅ Health check each one
- ✅ Show thermal state
- ✅ Route based on model compatibility

## Model Loading Summary

### What Gets Loaded Where

```
┌─────────────────────────────────────────────────────────────┐
│  NPU (Port 11434) - 3W Power                                │
├─────────────────────────────────────────────────────────────┤
│  ✓ qwen2.5:0.5b    (500MB)   - Realtime audio, simple      │
│  ✓ qwen2.5:1.5b    (1.5GB)   - Light chat                  │
│  ✓ tinyllama:1b    (1GB)     - Ultra efficient             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Intel GPU (Port 11435) - 12W Power                         │
├─────────────────────────────────────────────────────────────┤
│  ✓ llama3:7b       (~5GB)    - General text, code          │
│  ✓ mistral:7b      (~5GB)    - Alternative 7B              │
│  ✓ qwen2.5:7b      (~5GB)    - Qwen 7B variant             │
│  ✓ qwen2.5:1.5b    (1.5GB)   - Fallback to small           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  NVIDIA GPU (Port 11436) - 55W Power                        │
├─────────────────────────────────────────────────────────────┤
│  ✓ llama3:70b      (~40GB)   - Complex code, analysis      │
│  ✓ mixtral:8x7b    (~30GB)   - MoE large model             │
│  ✓ llama3:7b       (~5GB)    - Can also run smaller        │
│  ✓ codellama:7b    (~5GB)    - Code-specific               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  CPU (Port 11437) - 28W Power                               │
├─────────────────────────────────────────────────────────────┤
│  ✓ qwen2.5:1.5b    (1.5GB)   - Emergency fallback          │
│  ✓ llama3:7b       (~5GB)    - Slow but works              │
└─────────────────────────────────────────────────────────────┘
```

## Routing Examples

Once everything is set up:

### Example 1: Realtime Audio
```bash
Request: "Realtime voice transcription"
Model: qwen2.5:0.5b

Proxy routing:
1. Detects: realtime workload
2. Checks: qwen2.5:0.5b compatibility
   ✓ NPU: Has qwen2.5:0.5b loaded ✅
   ✓ Intel GPU: Compatible but not needed
   ✓ NVIDIA: Compatible but not needed
3. Scores: NPU wins (low latency + low power)
4. Routes to: NPU (localhost:11434)
```

### Example 2: Code Generation
```bash
Request: "Write Python function"
Model: llama3:70b

Proxy routing:
1. Detects: code workload
2. Checks: llama3:70b compatibility
   ✗ NPU: Max 2GB (70b too large)
   ✗ Intel GPU: Max 8GB (70b too large)
   ✓ NVIDIA: Has llama3:70b loaded ✅
   ✗ CPU: Max 16GB (70b too large)
3. Only option: NVIDIA
4. Routes to: NVIDIA (localhost:11436)
```

### Example 3: General Chat
```bash
Request: "Tell me about AI"
Model: llama3:7b

Proxy routing:
1. Detects: text workload
2. Checks: llama3:7b compatibility
   ✗ NPU: Doesn't have llama3:7b
   ✓ Intel GPU: Has llama3:7b loaded ✅
   ✓ NVIDIA: Has llama3:7b loaded ✅
   ✓ CPU: Has llama3:7b loaded ✅
3. Scores: Intel GPU wins (balanced)
4. Routes to: Intel GPU (localhost:11435)
```

## Automation Script

Create a helper script to pull all models:

```bash
#!/bin/bash
# setup-models.sh

echo "Setting up NPU models..."
OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:0.5b
OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:1.5b

echo "Setting up Intel GPU models..."
OLLAMA_HOST=http://localhost:11435 ollama pull llama3:7b
OLLAMA_HOST=http://localhost:11435 ollama pull mistral:7b
OLLAMA_HOST=http://localhost:11435 ollama pull qwen2.5:1.5b

echo "Setting up NVIDIA GPU models..."
OLLAMA_HOST=http://localhost:11436 ollama pull llama3:70b
OLLAMA_HOST=http://localhost:11436 ollama pull llama3:7b

echo "Setting up CPU models..."
OLLAMA_HOST=http://localhost:11437 ollama pull qwen2.5:1.5b
OLLAMA_HOST=http://localhost:11437 ollama pull llama3:7b

echo "✓ All models loaded!"
```

## Troubleshooting

### Backend Not Starting

```bash
# Check if port is already in use
lsof -i :11434
lsof -i :11435
lsof -i :11436
lsof -i :11437

# Kill existing Ollama instances
pkill ollama
```

### Model Pull Fails

```bash
# Check disk space
df -h

# Check Ollama version
ollama --version

# Pull with verbose output
OLLAMA_DEBUG=1 OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:0.5b
```

### Proxy Can't Connect

```bash
# Check backends are running
curl http://localhost:11434/api/tags
curl http://localhost:11435/api/tags
curl http://localhost:11436/api/tags
curl http://localhost:11437/api/tags

# Check proxy logs
./bin/ollama-proxy
# Look for "✅ Backend ... healthy" messages
```

## Summary

**What the proxy does:**
- ✅ Routes requests to appropriate backend based on model compatibility
- ✅ Monitors thermal state
- ✅ Applies efficiency modes
- ❌ **Does NOT load models** - you must do this manually

**Setup checklist:**
1. ☐ Start 4 Ollama instances on different ports
2. ☐ Configure each for specific hardware
3. ☐ Pull appropriate models on each instance
4. ☐ Verify all backends are healthy
5. ☐ Start the proxy
6. ☐ Test routing with different workloads

**Recommended minimal setup:**
- NPU: `qwen2.5:0.5b` (realtime)
- Intel GPU: `llama3:7b` (general use)
- NVIDIA: `llama3:70b` (complex tasks)
- CPU: `qwen2.5:1.5b` (fallback)

This gives you full coverage for all workload types! 🚀
