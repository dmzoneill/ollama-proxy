# Model-Aware & Workload-Aware Routing

## Overview

The proxy now includes intelligent model compatibility checking and workload type detection to prevent routing failures and optimize backend selection.

## The Problem We Solved

### Before (Naive Routing):
```
User Request:
  model: "llama3:70b"
  mode: Quiet

System: "NVIDIA too loud, routing to NPU"
Result: ❌ NPU crashes (can't run 70B model)
```

### After (Model-Aware Routing):
```
User Request:
  model: "llama3:70b"
  mode: Quiet

System:
  1. Check model compatibility
     - NPU: ❌ Only supports up to 1.5B models
     - Intel GPU: ❌ Only supports up to 7B models
     - NVIDIA: ✅ Supports 70B models
  2. Use NVIDIA (only capable backend)
  3. Return warning: "Quiet mode overridden, model requires NVIDIA"
```

## Features

### 1. Model Capability Declarations

Each backend declares what models it can run:

```yaml
backends:
  - id: "ollama-npu"
    model_capability:
      max_model_size_gb: 2
      supported_model_patterns:
        - "*:0.5b"      # Any 0.5B model
        - "*:1.5b"      # Any 1.5B model
        - "qwen2.5:*"   # All Qwen models
      preferred_models:
        - "qwen2.5:0.5b"  # Best performance
      excluded_patterns:
        - "*:70b"       # Too large

  - id: "ollama-nvidia"
    model_capability:
      max_model_size_gb: 24
      supported_model_patterns:
        - "*"  # Supports everything
      preferred_models:
        - "llama3:70b"  # Best for large models
```

### 2. Workload Type Detection

System automatically detects workload type from prompt:

#### Supported Media Types:

- **text**: General text generation
- **code**: Code generation/analysis
- **audio**: Audio transcription/TTS
- **realtime**: Real-time interactive (audio/chat)
- **image**: Image analysis/generation

#### Detection Examples:

```go
Prompt: "Transcribe this audio file"
→ Detected: audio
→ Preferred: NPU with qwen2.5:0.5b (low power, low latency)

Prompt: "Write a Python function to..."
→ Detected: code
→ Preferred: NVIDIA with llama3:70b (quality matters)

Prompt: "Live voice chat assistant"
→ Detected: realtime
→ Routing: NPU (latency critical + can use small model)
```

### 3. Automatic Model Substitution

If requested model not compatible with available backends:

```
Request: model="llama3:70b", mode=Quiet

Step 1: Check compatible backends
  - NVIDIA: ✅ Supports llama3:70b
  - Intel GPU: ❌ Max 7B
  - NPU: ❌ Max 1.5B

Step 2: Apply mode constraints
  - NVIDIA: ❌ Blocked (Quiet mode, fan 65%)

Step 3: No compatible backends!

Step 4: Model substitution
  - Workload: code
  - Substitute: llama3:70b → llama3:7b
  - Retry with llama3:7b

Step 5: Route to Intel GPU

Response:
{
  "backend_used": "ollama-igpu",
  "model_requested": "llama3:70b",
  "model_used": "llama3:7b",
  "model_substituted": true,
  "substitution_reason": "Quiet mode, llama3:70b requires NVIDIA",
  "detected_media_type": "code"
}
```

## Routing Algorithm

### New Multi-Stage Filtering:

```
1. Detect Workload Type
   └─ Analyze prompt for media type
   └─ Get workload profile (latency/power preferences)

2. Filter by Model Compatibility ⭐ NEW
   └─ Check model patterns
   └─ Check model size limits
   └─ Exclude incompatible backends

3. Filter by Thermal Health
   └─ Check temperature < 85°C
   └─ Check not throttling
   └─ Exclude overheated backends

4. Filter by Constraints
   └─ Check max_latency_ms
   └─ Check max_power_watts
   └─ Exclude over-limit backends

5. Score Remaining Candidates
   └─ Workload preferences (latency/power)
   └─ Thermal penalties
   └─ User annotations
   └─ Backend priority

6. Select Best Match
```

## Configuration

### Backend Model Capability (config.yaml)

```yaml
backends:
  - id: "ollama-npu"
    model_capability:
      max_model_size_gb: 2
      supported_model_patterns:
        - "*:0.5b"
        - "*:1.5b"
      preferred_models:
        - "qwen2.5:0.5b"
      excluded_patterns:
        - "*:70b"
```

### Pattern Matching Examples:

| Pattern | Matches | Description |
|---------|---------|-------------|
| `*` | All models | Wildcard |
| `*:0.5b` | `qwen2.5:0.5b`, `tinyllama:0.5b` | Any model with 0.5B size |
| `llama3:*` | `llama3:7b`, `llama3:70b` | All Llama3 models |
| `*70b*` | `llama3:70b`, `mixtral:8x70b` | Any model with "70b" in name |
| `qwen2.5:*` | `qwen2.5:0.5b`, `qwen2.5:7b` | All Qwen 2.5 models |

## Media Type Annotations

### Option 1: Explicit Media Type

```go
req := &GenerateRequest{
  Prompt: "Transcribe this",
  Model:  "qwen2.5:0.5b",
  Annotations: &Annotations{
    MediaType: "audio",  // Explicit
  },
}
```

### Option 2: Auto-Detection (Default)

```go
req := &GenerateRequest{
  Prompt: "Write Python code to sort a list",
  Model:  "llama3:7b",
  Annotations: &Annotations{
    MediaType: "auto",  // System detects "code"
  },
}
```

## Real-World Scenarios

### Scenario 1: Realtime Audio (NPU Perfect!)

```
Request:
  Prompt: "Live voice transcription"
  Model: "qwen2.5:0.5b"
  Annotations:
    latency_critical: true

Detection:
  Media Type: realtime
  Profile:
    - Prefer low latency: ✅
    - Prefer low power: ✅ (runs continuously)
    - Preferred model: qwen2.5:0.5b
    - Max model size: 2GB

Routing:
  Filter by model:
    ✅ NPU (supports qwen2.5:0.5b)
    ✅ Intel GPU (supports it)
    ✅ NVIDIA (supports it)

  Score with workload hints:
    NPU: 2000 (low latency + low power match!)
    Intel GPU: 1200
    NVIDIA: 800 (high power penalty)

  Selected: NPU ⭐

Result: Perfect match! Realtime audio on 3W NPU.
```

### Scenario 2: Code Generation (Needs NVIDIA)

```
Request:
  Prompt: "Generate complex algorithm in Rust"
  Model: "llama3:70b"
  Annotations:
    media_type: "code"

Detection:
  Media Type: code
  Profile:
    - Prefer low latency: ❌
    - Prefer low power: ❌
    - Preferred model: llama3:70b
    - Max model size: 80GB

Routing:
  Filter by model:
    ❌ NPU (max 2GB)
    ❌ Intel GPU (max 8GB)
    ✅ NVIDIA (supports 70B)

  Filter by thermal:
    ✅ NVIDIA (65°C, OK)

  Selected: NVIDIA ⭐

Result: Only NVIDIA can run 70B model.
```

### Scenario 3: Quiet Mode Conflict

```
Request:
  Prompt: "Analyze this codebase"
  Model: "llama3:70b"
  Mode: Quiet

Detection:
  Media Type: code

Routing:
  Filter by model:
    ✅ NVIDIA (only one that supports 70B)

  Filter by thermal:
    ❌ NVIDIA (fan 65% > Quiet limit 40%)

  No backends available!

  Model Substitution:
    Substitute: llama3:70b → llama3:7b (code profile)

  Retry with llama3:7b:
    ✅ Intel GPU (supports 7B, fan 35%)

  Selected: Intel GPU

Response:
  {
    "backend_used": "ollama-igpu",
    "model_requested": "llama3:70b",
    "model_used": "llama3:7b",
    "model_substituted": true,
    "substitution_reason": "Quiet mode, llama3:70b requires NVIDIA",
    "detected_media_type": "code",
    "routing_hints": [
      "Detected: code (Code generation - benefits from larger models)",
      "Model llama3:70b not supported, using llama3:7b for code workload",
      "Model compatible backends: 3",
      "Thermally healthy backends: 2",
      "Selected: ollama-igpu [62.0°C, fan:35%]"
    ]
  }
```

## Response Fields

Every routing decision now includes:

```go
type RoutingDecision struct {
  Backend            backends.Backend
  Reason             string
  EstimatedPowerW    float64
  EstimatedLatencyMs int32
  Alternatives       []string

  // Model awareness
  ModelRequested     string   // "llama3:70b"
  ModelUsed          string   // "llama3:7b"
  ModelSubstituted   bool     // true
  SubstitutionReason string   // "Quiet mode, 70b requires NVIDIA"

  // Workload detection
  DetectedMediaType  string   // "code"
  RoutingHints       []string // Reasoning chain
}
```

## Benefits

### 1. Prevents Routing Failures
- ✅ Never routes incompatible models to backends
- ✅ Clear error messages when no backend supports model
- ✅ Automatic fallback with substitution

### 2. Optimizes for Workload
- ✅ Audio realtime → NPU (low latency + low power)
- ✅ Code generation → NVIDIA with large models (quality)
- ✅ Simple chat → Intel GPU (balanced)

### 3. Transparency
- ✅ User always knows what was requested vs. what happened
- ✅ Full reasoning chain in response
- ✅ Clear substitution explanations

### 4. Smart Defaults
- ✅ System detects workload type automatically
- ✅ Selects appropriate model size for backend
- ✅ Respects user annotations but won't break

## Testing Examples

### Test 1: Large Model in Quiet Mode
```bash
curl -X POST http://localhost:50051/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Write complex code",
    "model": "llama3:70b",
    "annotations": {
      "media_type": "code"
    }
  }'

# With Quiet mode active:
# Expected: Model substituted to llama3:7b, routed to Intel GPU
# Response shows: model_substituted=true, substitution_reason="Quiet mode..."
```

### Test 2: Realtime Audio
```bash
curl -X POST http://localhost:50051/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Realtime voice transcription",
    "model": "qwen2.5:0.5b",
    "annotations": {
      "latency_critical": true
    }
  }'

# Expected: Detected as realtime, routed to NPU
# Response shows: detected_media_type="realtime", backend="ollama-npu"
```

### Test 3: Incompatible Model
```bash
curl -X POST http://localhost:50051/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Simple query",
    "model": "some-huge-model:175b"
  }'

# Expected: Error - no backend supports this model
# Or: Substituted to compatible model with warning
```

## Summary

**Model-aware routing solves the critical problem:**
- ✅ NPU can't run 70B models → System knows this and won't try
- ✅ Realtime audio needs low latency → System routes to NPU
- ✅ Code needs quality → System prefers large models on NVIDIA
- ✅ Quiet mode conflicts → System substitutes model or warns user

**Your feedback was essential!** This makes the system actually usable in practice.
