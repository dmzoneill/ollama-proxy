# Forwarding and Chaining Guide

## Overview

The proxy supports **multi-stage pipelines** with intelligent forwarding between backends. This enables sophisticated workflows like:

- **Voice Assistant:** Voice → Text (NPU) → LLM (iGPU/GPU) → Text → Voice (NPU)
- **Adaptive Quality:** Start cheap (NPU), escalate to GPU if quality insufficient
- **Thermal Protection:** Switch backends mid-generation if overheating
- **Speculative Execution:** NPU generates candidates, GPU picks best

---

## Core Concepts

### 1. **Single-Stage Routing (Current)**
One request → One backend

```
Request: "Write code"
  → Router selects: ollama-nvidia
  → Returns: Code
```

### 2. **Multi-Stage Pipeline (New)**
One request → Multiple stages → Multiple backends

```
Request: "Voice question"
  → Stage 1: ollama-npu (voice → text)
  → Stage 2: ollama-intel (process text)
  → Stage 3: ollama-npu (text → voice)
  → Returns: Voice response
```

### 3. **Forwarding within Stage**
One stage tries multiple backends

```
Stage: "Generate text"
  → Try ollama-npu (confidence: 0.6 - too low)
  → Forward to ollama-intel (confidence: 0.8 - good!)
  → Returns: Text
```

---

## Forwarding Patterns

### Pattern 1: Confidence-Based Escalation

**Use Case:** Start with cheap backend, escalate if quality insufficient

**How it works:**
1. Try NPU with small model (3W, fast)
2. Estimate confidence of output
3. If confidence < threshold, forward to better backend
4. Repeat until confidence met or max retries exceeded

**Example:**
```yaml
forwarding_policy:
  enable_confidence_check: true
  min_confidence: 0.75
  max_retries: 3
  escalation_path:
    - "ollama-npu"     # 3W
    - "ollama-intel"   # 12W
    - "ollama-nvidia"  # 55W
```

**Result:**
- Simple queries: NPU handles (3W) ✅
- Medium queries: iGPU handles (12W) ✅
- Complex queries: GPU handles (55W) ✅
- Average power: ~10W instead of 55W (5.5× battery improvement)

**Confidence Estimation Methods:**
- Response length (longer = more confident)
- Token probabilities (if available from model)
- Pattern matching (detects "I'm not sure", "I don't know")
- Model-specific heuristics

---

### Pattern 2: Thermal Failover

**Use Case:** Switch backends if current one overheats

**How it works:**
1. Start generation on preferred backend (NVIDIA)
2. Monitor temperature every second
3. If temp > threshold, pause generation
4. Switch to cooler backend (Intel GPU)
5. Resume generation with context preserved

**Example:**
```yaml
forwarding_policy:
  enable_thermal_check: true
  max_temperature: 87.0
  max_fan_percent: 85
  escalation_path:
    - "ollama-nvidia"  # Start here
    - "ollama-intel"   # Switch if overheating
    - "ollama-cpu"     # Final fallback
```

**Result:**
- NVIDIA generates tokens 1-500 (fast)
- Temperature hits 87°C
- Switch to Intel GPU
- Intel GPU generates tokens 501-1000 (preserved quality)
- NVIDIA cools down for next request

**Implementation Requirements:**
- Context preservation (KV cache or prompt reconstruction)
- Seamless handoff (no user-visible interruption)
- State management (track where we left off)

---

### Pattern 3: Split Workload (Prefill/Decode)

**Use Case:** Use different backends for different phases

**How it works:**
1. **Prefill phase:** NPU processes prompt (low power, one-time)
2. **Decode phase:** GPU generates tokens (high quality, iterative)
3. Share KV cache between backends

**Example:**
```yaml
stages:
  - id: "prefill"
    type: "prefill"
    preferred_hardware: "npu"
    model: "llama3:7b"

  - id: "decode"
    type: "decode"
    preferred_hardware: "nvidia"
    model: "llama3:7b"
    reuse_kv_cache: true
```

**Result:**
- Prefill: NPU (3W, 2 seconds) = 6 Wh
- Decode: NVIDIA (55W, 5 seconds) = 76 Wh
- Total: 82 Wh

vs. All on NVIDIA:
- Prefill + Decode: NVIDIA (55W, 7 seconds) = 107 Wh
- **Savings: 23% energy**

**Challenges:**
- KV cache compatibility between backends
- Ollama doesn't expose KV cache API (yet)
- Requires model format compatibility

---

### Pattern 4: Speculative Execution

**Use Case:** Generate multiple candidates cheaply, pick best with expensive model

**How it works:**
1. NPU generates N candidates in parallel (cheap)
2. GPU evaluates all candidates and picks best (expensive but one-time)
3. Return best candidate

**Example:**
```yaml
stages:
  - id: "generate-candidates"
    type: "text_generation"
    preferred_hardware: "npu"
    model: "qwen2.5:0.5b"
    parallel_count: 5  # Generate 5 in parallel

  - id: "select-best"
    type: "text_generation"
    preferred_hardware: "nvidia"
    model: "llama3:70b"
```

**Result:**
- NPU generates 5 candidates: 5 × 2 seconds = 10 seconds @ 3W = 30 Wh
- GPU evaluates 5 candidates: 1 second @ 55W = 55 Wh
- Total: 11 seconds, 85 Wh

vs. GPU generates 5 candidates:
- 5 × 5 seconds = 25 seconds @ 55W = 382 Wh
- **Savings: 77% energy, 56% time**

**Best For:**
- Code generation (multiple solutions)
- Creative writing (multiple variations)
- A/B testing ideas

---

### Pattern 5: Quality Gate

**Use Case:** Quick check before expensive operation

**How it works:**
1. NPU does quick "sanity check" on request
2. If request is simple → NPU handles it
3. If request is complex → Forward to GPU

**Example:**
```yaml
stages:
  - id: "classify-complexity"
    type: "text_generation"
    preferred_hardware: "npu"
    model: "qwen2.5:0.5b"
    prompt_template: "Is this query simple or complex? Query: {{ .Input }}"

  - id: "handle-simple"
    type: "text_generation"
    preferred_hardware: "npu"
    model: "qwen2.5:1.5b"
    condition: "{{ .PreviousOutput == 'simple' }}"

  - id: "handle-complex"
    type: "text_generation"
    preferred_hardware: "nvidia"
    model: "llama3:70b"
    condition: "{{ .PreviousOutput == 'complex' }}"
```

**Result:**
- 80% of queries classified as "simple" → NPU handles
- 20% of queries classified as "complex" → GPU handles
- Average power: 0.8 × 3W + 0.2 × 55W = **13.4W instead of 55W**

---

## Multi-Stage Pipelines

### Voice Assistant Pipeline

**Your Use Case:** Voice → Text → LLM → Text → Voice

```yaml
pipelines:
  - id: "voice-assistant"
    stages:
      # Stage 1: Speech Recognition (NPU)
      - id: "voice-to-text"
        type: "audio_to_text"
        preferred_hardware: "npu"
        model: "whisper-tiny"
        forwarding_policy:
          enable_confidence_check: true
          min_confidence: 0.7
          escalation_path: ["ollama-npu", "ollama-intel"]

      # Stage 2: LLM Processing (iGPU/GPU)
      - id: "process-text"
        type: "text_generation"
        preferred_hardware: "igpu"
        model: "llama3:7b"
        forwarding_policy:
          enable_confidence_check: true
          min_confidence: 0.8
          enable_thermal_check: true
          escalation_path: ["ollama-intel", "ollama-nvidia"]

      # Stage 3: Text-to-Speech (NPU)
      - id: "text-to-voice"
        type: "text_to_audio"
        preferred_hardware: "npu"
        model: "piper-tts"
```

**Power Breakdown:**
- Voice-to-text: NPU (3W × 1s = 3 Wh)
- Process text: iGPU (12W × 3s = 36 Wh)
- Text-to-voice: NPU (3W × 1s = 3 Wh)
- **Total: 42 Wh**

vs. All on GPU:
- Voice-to-text: GPU (55W × 1s = 55 Wh)
- Process text: GPU (55W × 3s = 165 Wh)
- Text-to-voice: GPU (55W × 1s = 55 Wh)
- **Total: 275 Wh**

**Savings: 84% energy, 6.5× battery life improvement**

---

### RAG Pipeline

**Use Case:** Embedding → Retrieve → Generate

```yaml
pipelines:
  - id: "rag-pipeline"
    stages:
      # Stage 1: Generate embedding (NPU)
      - id: "embed-query"
        type: "embedding"
        preferred_hardware: "npu"
        model: "nomic-embed-text"

      # Stage 2: Vector DB lookup (custom)
      - id: "retrieve-context"
        type: "custom"
        handler: "vector_db.search"

      # Stage 3: Generate answer (GPU)
      - id: "generate-answer"
        type: "text_generation"
        preferred_hardware: "nvidia"
        model: "llama3:70b"
        input_transform:
          template: |
            Context: {{ .Context }}
            Question: {{ .Query }}
```

**Why This Works:**
- Embedding: NPU perfect for small, fast models
- Retrieval: No GPU needed (database operation)
- Generation: GPU for best quality with context

---

## Implementation Status

### ✅ Already Implemented
- Single-stage routing
- Thermal monitoring
- Model capability checking
- Workload detection
- Efficiency modes

### 🟡 Partially Implemented
- Basic pipeline structure (`pkg/pipeline/pipeline.go`)
- Example pipeline configs (`config/pipelines.yaml`)
- Confidence estimation (placeholder)

### ❌ Not Yet Implemented
- Streaming failover (mid-generation backend switching)
- KV cache sharing (for split workload)
- Parallel stage execution
- Audio-to-text stage type
- Text-to-audio stage type
- Custom stage handlers
- Pipeline execution engine integration with main proxy

---

## Implementation Roadmap

### Phase 1: Basic Forwarding (1-2 days)
**Goal:** Confidence-based escalation within single stage

**Tasks:**
1. Implement confidence estimation
   - Response length heuristic
   - Pattern matching for uncertainty phrases
   - Model-specific scoring
2. Add retry logic with escalation
3. Update router to support forwarding policy
4. Test: NPU → Intel → NVIDIA escalation

**Deliverable:**
```bash
# Request that's too complex for NPU
grpcurl -d '{
  "model": "qwen2.5:0.5b",
  "prompt": "Explain quantum entanglement in detail",
  "annotations": {"enable_forwarding": true}
}' localhost:50051 compute.v1.ComputeService/Generate

# Response metadata:
# - Attempted: ollama-npu (confidence: 0.4 - too low)
# - Forwarded to: ollama-intel (confidence: 0.9 - success!)
```

---

### Phase 2: Multi-Stage Pipelines (3-5 days)
**Goal:** Execute multi-stage workflows

**Tasks:**
1. Implement pipeline executor
2. Add stage-to-stage data passing
3. Implement input/output transforms
4. Add pipeline configuration loader (YAML)
5. Integrate with main proxy service
6. Test: Basic text pipeline (embed → generate)

**Deliverable:**
```bash
# Execute RAG pipeline
grpcurl -d '{
  "pipeline_id": "rag-pipeline",
  "input": {
    "query": "What is quantum computing?"
  }
}' localhost:50051 compute.v1.ComputeService/ExecutePipeline

# Response:
# - Stage 1: Generated embedding on NPU
# - Stage 2: Retrieved 3 documents from vector DB
# - Stage 3: Generated answer on NVIDIA with context
```

---

### Phase 3: Thermal Failover (3-5 days)
**Goal:** Mid-generation backend switching

**Tasks:**
1. Implement context preservation
   - Track generated tokens
   - Reconstruct prompt with partial output
2. Add streaming monitor
   - Watch thermal state during generation
   - Trigger failover when threshold exceeded
3. Implement seamless handoff
   - Stop current backend
   - Start new backend with context
   - Resume streaming to client
4. Test: Long generation with thermal switching

**Deliverable:**
```bash
# Long document generation
grpcurl -d '{
  "model": "llama3:70b",
  "prompt": "Write a 10,000 word essay on AI",
  "annotations": {"enable_thermal_failover": true}
}' localhost:50051 compute.v1.ComputeService/GenerateStream

# Streaming response:
# [Tokens 1-500 from NVIDIA]
# [Thermal threshold reached - switching to Intel GPU]
# [Tokens 501-1000 from Intel GPU]
# [NVIDIA cooled down - switching back]
# [Tokens 1001-1500 from NVIDIA]
```

---

### Phase 4: Audio Stages (1 week)
**Goal:** Voice assistant pipeline

**Tasks:**
1. Integrate speech recognition
   - Whisper model support
   - Audio input handling
   - NPU optimization
2. Integrate text-to-speech
   - Piper TTS or similar
   - Audio output generation
   - NPU optimization
3. Implement full voice pipeline
4. Test end-to-end voice interaction

**Deliverable:**
```bash
# Voice assistant
grpcurl -d '{
  "pipeline_id": "voice-assistant",
  "input": {
    "audio": "<base64-encoded-audio>"
  }
}' localhost:50051 compute.v1.ComputeService/ExecutePipeline

# Response:
# - Stage 1: Transcribed audio to text on NPU
# - Stage 2: Processed question on Intel GPU
# - Stage 3: Generated speech on NPU
# - Returns: <base64-encoded-response-audio>
```

---

### Phase 5: Advanced Patterns (1 week)
**Goal:** Speculative execution, split workload

**Tasks:**
1. Parallel stage execution
   - Run N instances of same stage
   - Aggregate results
2. KV cache sharing (if Ollama adds support)
   - Prefill on one backend
   - Decode on another
3. Custom stage handlers
   - Vector DB integration
   - External API calls
4. Pipeline optimization
   - Caching intermediate results
   - Skipping unnecessary stages

---

## Usage Examples

### Example 1: Simple Adaptive Text

**Define pipeline:**
```yaml
# config/pipelines.yaml
pipelines:
  - id: "adaptive"
    stages:
      - type: "text_generation"
        model: "qwen2.5:0.5b"
        forwarding_policy:
          escalation_path: ["ollama-npu", "ollama-intel", "ollama-nvidia"]
          min_confidence: 0.75
```

**Use pipeline:**
```bash
# gRPC
grpcurl -d '{
  "pipeline_id": "adaptive",
  "input": {"text": "Explain quantum physics"}
}' localhost:50051 compute.v1.ComputeService/ExecutePipeline

# HTTP (future)
curl -X POST http://localhost:8080/v1/pipelines/adaptive \
  -H "Content-Type: application/json" \
  -d '{"input": {"text": "Explain quantum physics"}}'
```

**Result:**
```json
{
  "pipeline_id": "adaptive",
  "success": true,
  "stages": [
    {
      "stage_id": "generate-text",
      "backend": "ollama-intel",
      "forwarded": true,
      "attempts": [
        {"backend": "ollama-npu", "confidence": 0.6, "reason": "Low confidence"},
        {"backend": "ollama-intel", "confidence": 0.85, "reason": "Success"}
      ],
      "duration_ms": 3500
    }
  ],
  "final_output": "Quantum physics explanation...",
  "total_time_ms": 3500,
  "total_energy_wh": 12
}
```

---

### Example 2: Voice Assistant

**Python client:**
```python
import grpc
from api.proto import compute_pb2, compute_pb2_grpc

# Record audio from microphone
audio_data = record_audio()

# Send to proxy
channel = grpc.insecure_channel('localhost:50051')
stub = compute_pb2_grpc.ComputeServiceStub(channel)

response = stub.ExecutePipeline(compute_pb2.ExecutePipelineRequest(
    pipeline_id="voice-assistant",
    input={"audio": audio_data}
))

# Play response audio
play_audio(response.final_output.audio)
```

---

## Configuration Reference

### Pipeline Configuration

```yaml
pipelines:
  - id: "pipeline-id"              # Unique identifier
    name: "Human-readable name"
    description: "What this pipeline does"

    stages:
      - id: "stage-id"
        type: "text_generation"    # Stage type
        description: "Stage description"

        # Backend selection
        preferred_backend: "ollama-npu"    # Specific backend
        preferred_hardware: "npu"          # Or hardware type
        model: "llama3:7b"                 # Model to use

        # Forwarding policy
        forwarding_policy:
          enable_confidence_check: true
          min_confidence: 0.75
          max_retries: 3
          escalation_path:
            - "ollama-npu"
            - "ollama-intel"
            - "ollama-nvidia"

          enable_thermal_check: true
          max_temperature: 85.0
          max_fan_percent: 85

          enable_quality_check: true
          quality_threshold: 0.8

        # Input/output transformation
        input_transform:
          template: "Process this: {{ .PreviousOutput }}"

        output_transform:
          template: "{{ .Output | trim }}"

    # Pipeline options
    options:
      enable_streaming: true
      preserve_context: true
      continue_on_error: false
      collect_metrics: true
```

### Forwarding Policy

```yaml
forwarding_policy:
  # Confidence-based
  enable_confidence_check: bool
  min_confidence: float (0.0-1.0)
  max_retries: int

  # Thermal-based
  enable_thermal_check: bool
  max_temperature: float (°C)
  max_fan_percent: int (0-100)

  # Quality-based
  enable_quality_check: bool
  quality_threshold: float (0.0-1.0)

  # Latency-based
  enable_latency_check: bool
  max_latency_ms: int

  # Power-based
  enable_power_budget: bool
  max_power_watts: float

  # Escalation path
  escalation_path:
    - "backend-1"  # Try first
    - "backend-2"  # Try second
    - "backend-3"  # Final fallback
```

---

## Benefits Summary

### Battery Life
```
Traditional (all GPU):
- Simple query: 55W × 2s = 110 Wh
- 50 queries/hour = 5,500 Wh = Battery lasts 1 hour

With adaptive forwarding:
- Simple query: 3W × 2s = 6 Wh (NPU)
- Medium query: 12W × 3s = 36 Wh (iGPU)
- Complex query: 55W × 5s = 275 Wh (GPU)
- Mix: 80% simple, 15% medium, 5% complex
- Average: 0.8×6 + 0.15×36 + 0.05×275 = 23.7 Wh per query
- 50 queries/hour = 1,185 Wh = Battery lasts 4.6 hours

5× battery life improvement!
```

### Thermal Management
```
Without failover:
- NVIDIA reaches 87°C
- Thermal throttling kicks in
- Performance drops 40%
- Fan at 95% (loud)

With thermal failover:
- NVIDIA reaches 87°C
- Switch to Intel GPU
- Performance maintained
- NVIDIA cools down
- Fan drops to 35% (quiet)
```

### Quality
```
Without escalation:
- User always specifies model
- Often picks wrong model (too small or too large)

With confidence escalation:
- Start with small model (fast, cheap)
- Automatically escalate if insufficient
- Always get good enough quality
- Minimize wasted GPU cycles
```

---

## Next Steps

Want to implement this? Here's what I recommend:

1. **Start with Phase 1** (confidence-based forwarding)
   - Easiest to implement
   - Immediate battery life benefit
   - No audio dependencies

2. **Add Phase 2** (multi-stage pipelines)
   - Enables voice assistant workflow
   - Foundation for all other patterns

3. **Choose your priority:**
   - **Voice assistant?** → Implement Phase 4 (audio stages)
   - **Long documents?** → Implement Phase 3 (thermal failover)
   - **Code generation?** → Implement speculative execution

Would you like me to start implementing Phase 1 (confidence-based forwarding)?
