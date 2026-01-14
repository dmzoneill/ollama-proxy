# Audio Pipeline Performance Analysis

## Current Performance vs Xenith Baseline

### Xenith Performance (Baseline)
**Total End-to-End: 1.5-2.5 seconds**

| Stage | Time | Details |
|-------|------|---------|
| Wake word detection | 500ms | Not applicable to our use case |
| **STT (Whisper "base")** | 300ms | OpenVINO Whisper on **NPU** (Intel Neural Processor) |
| **LLM (qwen2.5-1.5b INT4)** | 200ms | OpenVINO INT4 on **CPU** (12x faster than NPU!), first token, streaming |
| **TTS (MeloTTS)** | 100ms | MeloTTS with BERT on NPU, synthesis on CPU |
| **Total (STT→LLM→TTS)** | **~600ms** | Sub-second response |

**Critical Discovery:** Xenith config explicitly uses `device: "CPU"` for LLM because:
- CPU is **12x faster** than NPU for LLM inference (~200ms vs ~2.3s)
- NPU is only used for STT (Whisper) and TTS BERT preprocessing
- Config comment: "For best response time, use CPU" / "NPU is slow (~2.3s warmup per query!)"

### Our Current Performance
**Total End-to-End: FAILS (timeouts)**

| Stage | Time | Details |
|-------|------|---------|
| **STT (whisper.cpp)** | 6,500ms | whisper.cpp tiny model on CPU (no GPU) |
| **LLM (qwen2.5:0.5b)** | 160,000ms | Ollama GGUF via HTTP, 8192 context, blocking |
| **TTS (piper)** | 140ms | ✅ Working well |
| **Total** | **TIMEOUT** | LLM stage fails after 10s HTTP timeout |

### Performance Gap Analysis

| Component | Xenith | Our Current | Gap | Root Cause |
|-----------|--------|-------------|-----|------------|
| STT | 300ms | 6,500ms | **22x slower** | CPU vs GPU, whisper.cpp vs PyTorch |
| LLM | 200ms | 160,000ms | **800x slower** | INT4 vs GGUF, CPU vs iGPU, 256 vs 8192 context, streaming vs blocking |
| TTS | 100ms | 140ms | 1.4x slower | ✅ Acceptable |

## Root Causes

### 1. STT Performance (22x slower)

**Xenith Approach:**
- OpenVINO Whisper "base" model
- **NPU** acceleration (Intel Neural Processor)
- OpenVINO INT8 quantization
- ~1-3W power consumption
- Result: **300ms**

**Our Current Approach:**
- whisper.cpp "tiny" model
- **CPU-only** execution (no OpenVINO support)
- No hardware acceleration
- GGML quantization
- Result: **6,500ms**

**Why the difference:**
- whisper.cpp built without OpenVINO support (cmake flag was OFF)
- No NPU/GPU/Intel hardware acceleration
- Missing OpenVINO runtime optimizations for Intel hardware
- Smaller model but much slower without HW acceleration

### 2. LLM Performance (800x slower)

**Xenith Approach:**
- Model: `OpenVINO/Qwen2.5-1.5B-Instruct-int4-ov`
- Quantization: **INT4** (4-bit)
- Device: CPU via `openvino_genai.LLMPipeline`
- Context: Default (typically 2048 tokens)
- Max tokens: **256**
- Streaming: Yes (first token in 200ms)
- Result: **200ms to first token**

**Our Current Approach:**
- Model: `qwen2.5:0.5b` via Ollama
- Quantization: GGUF (8-bit or mixed)
- Device: Intel iGPU via oneAPI
- Context: **8192 tokens**
- Max tokens: Unlimited (generates until done)
- Streaming: No (blocking HTTP call)
- Result: **160,000ms+ (timeout)**

**Why the difference:**
1. **Context size**: 8192 vs 512 tokens = 16x more KV cache
2. **Quantization**: GGUF vs INT4 = ~4x slower matrix ops (INT4 is more optimized)
3. **Runtime**: Ollama HTTP API vs direct `openvino_genai.LLMPipeline` = HTTP overhead
4. **Device**: Intel iGPU (struggling) vs **CPU with OpenVINO** (12x faster than NPU!)
5. **Model size**: 0.5B (GGUF) vs 1.5B (INT4) - larger INT4 model is faster due to better quantization
6. **Streaming**: Blocks for full response vs streams first token in 200ms
7. **Note:** Xenith uses **CPU, not NPU**, for LLM because CPU is 12x faster (~200ms vs ~2.3s on NPU)

### 3. TTS Performance (acceptable)

**Both use similar approaches:**
- Xenith: MeloTTS (100ms)
- Ours: Piper (140ms)
- ✅ Both fast enough for real-time

## Critical Configuration Issues

### Issue 1: Ollama Context Size
```bash
# Current Ollama iGPU runner config:
--ctx-size 8192  # Way too large!
--batch-size 512
--n-gpu-layers 25
```

**Impact:** 8192 token context = 96MB KV cache + slow attention computation

**Fix:** Reduce to 2048 or even 512 for voice assistant use case

### Issue 2: No Token Limit
```go
// Current code in ollama.go
// No max_tokens specified, generates until EOS token
ollamaReq := map[string]interface{}{
    "model":  req.Model,
    "prompt": req.Prompt,
    "stream": false,
}
```

**Impact:** LLM generates unbounded text (200+ tokens)

**Fix:** Add `"num_predict": 256` to limit output

### Issue 3: Blocking API Call
```go
// Current code uses blocking HTTP POST
resp, err := b.client.Do(httpReq)
// Waits for full response before returning
```

**Impact:** 10s HTTP timeout kills pipeline

**Fix:** Use streaming API (`"stream": true`)

### Issue 4: whisper.cpp Not GPU Accelerated
```bash
# Built without OpenVINO:
cmake -B build -DWHISPER_OPENVINO=OFF
```

**Impact:** 6.5s CPU inference vs 300ms GPU

**Fix:** Rebuild with CUDA or OpenVINO support

## Recommended Solutions (Prioritized)

### Priority 1: Fix LLM Configuration (Quick Win - 5 minutes)

**Goal:** Achieve <1s LLM response time

**Changes to `/etc/systemd/system/ollama-igpu.service`:**
```bash
# Add environment variables:
Environment="OLLAMA_NUM_CTX=512"          # Reduce context from 8192
Environment="OLLAMA_NUM_PREDICT=256"      # Limit output tokens
```

**Changes to `pkg/backends/ollama/ollama.go`:**
```go
ollamaReq := map[string]interface{}{
    "model":  req.Model,
    "prompt": req.Prompt,
    "stream": false,
    "options": map[string]interface{}{
        "num_ctx":     512,   // Small context
        "num_predict": 256,   // Limit output
        "temperature": 0.7,
    },
}
```

**Expected result:** LLM time drops from 160s to <5s

### Priority 2: Add Voice Activity Detection (Medium - 1 hour)

**Goal:** Reduce transcript length by 80%

**Problem:** Currently transcribing 2700+ characters of silence/noise

**Solution:** Add VAD to filter audio before STT
- Use `github.com/go-audio/vad` or similar
- Only send audio segments with voice activity
- Expected: 2700 chars → 300 chars

**Expected result:** STT time drops from 6.5s to <1s (10x smaller audio)

### Priority 3: Switch to Streaming LLM (Medium - 2 hours)

**Goal:** Get first audio playing in <1s

**Changes:**
```go
// Use streaming API
ollamaReq["stream"] = true

// Start TTS as soon as first sentence is complete
// Don't wait for full LLM response
```

**Expected result:** User hears first sentence in 1-2s (perceived latency improvement)

### Priority 4: Rebuild whisper.cpp with OpenVINO NPU Support (Medium - 30 minutes)

**Goal:** Match Xenith's 300ms STT performance on NPU

**Option A: OpenVINO NPU (Recommended - matches Xenith)**
```bash
# Rebuild whisper.cpp with OpenVINO support
cd ~/src/whisper.cpp
cmake -B build -DWHISPER_OPENVINO=ON \
    -DOpenVINO_DIR=/home/daoneill/src/openvino-setup/openvino_genai_ubuntu24_2025.4.0.0_x86_64/runtime/cmake
cmake --build build -j
sudo cp build/bin/whisper-cli /usr/local/bin/whisper-cpp

# Test on NPU
whisper-cpp -m ~/.cache/whisper/ggml-base.bin -f test.wav --openvino-device NPU
```

**Option B: CUDA (NVIDIA GPU - higher power)**
```bash
cd ~/src/whisper.cpp
cmake -B build -DWHISPER_CUDA=ON
cmake --build build -j
sudo cp build/bin/whisper-cli /usr/local/bin/whisper-cpp
```

**Expected result:** STT time drops from 6.5s to 300ms on NPU (matching Xenith)

### Priority 5: Switch to OpenVINO GenAI Backend (High effort - 1 day)

**Goal:** Match Xenith's 200ms LLM performance on **CPU**

**Approach:** Create new backend type in `pkg/backends/openvino/`

**Critical insight from Xenith:**
- **CPU is 12x faster than NPU** for LLM inference (~200ms vs ~2.3s)
- Use `device: "CPU"` not NPU/iGPU for best performance
- Xenith config explicitly recommends CPU for fast response

**Benefits:**
- Direct `openvino_genai.LLMPipeline` (no HTTP overhead)
- INT4 quantized models (4x faster than GGUF)
- CPU with OpenVINO optimizations (12x faster than NPU)
- Native token streaming

**Implementation:**
1. Create `pkg/backends/openvino/llm.go` with cgo bindings
2. Download `OpenVINO/Qwen2.5-1.5B-Instruct-int4-ov` from HuggingFace
3. Use `device: "CPU"` (NOT NPU, NOT iGPU)
4. Enable streaming for first-token latency
5. Update config.yaml to support openvino backend

**Expected result:** LLM time drops to 200ms on CPU (matches Xenith)

## Quick Test Plan

### Test 1: Reduce Ollama Context (5 min)
```bash
# Edit service
sudo nano /etc/systemd/system/ollama-igpu.service
# Add: Environment="OLLAMA_NUM_CTX=512"

sudo systemctl daemon-reload
sudo systemctl restart ollama-igpu

# Test
curl -X POST http://localhost:11435/api/generate \
  -d '{"model":"qwen2.5:0.5b","prompt":"Hello","options":{"num_ctx":512,"num_predict":50}}'

# Should respond in <5s
```

### Test 2: Voice Activity Detection (simulation)
```bash
# Instead of transcribing full 16s of audio (512KB)
# Transcribe only 2s segments with voice (64KB)

# Expected: 6.5s → <1s
```

### Test 3: Streaming API
```bash
# Enable streaming in ollama.go
# LLM starts generating immediately
# TTS starts after first sentence

# Expected: First audio in <2s
```

## Target Performance After Fixes

| Stage | Current | After Priority 1-3 | After Priority 4-5 | Xenith Baseline |
|-------|---------|-------------------|-------------------|-----------------|
| STT | 6,500ms | 1,000ms (VAD) | **300ms** (GPU) | 300ms |
| LLM | 160,000ms | **3,000ms** (512 ctx) | **200ms** (OpenVINO) | 200ms |
| TTS | 140ms | 140ms | 140ms | 100ms |
| **Total** | TIMEOUT | **4.1s** | **640ms** | 600ms |

## Conclusion

Our current 160+ second pipeline can be reduced to **<1 second** by:

1. **Immediate fix** (5 min): Reduce Ollama context to 512, limit output to 256 tokens
2. **Short-term** (2-3 hours): Add VAD, enable streaming, rebuild whisper.cpp with GPU
3. **Long-term** (1 day): Switch to OpenVINO GenAI backend for native INT4 inference

The Xenith project demonstrates that **sub-second voice response is achievable** on the same hardware. Our current setup is misconfigured, not fundamentally limited.

**Next Steps:**
1. Apply Priority 1 fixes immediately
2. Test and measure performance
3. Iterate based on results
