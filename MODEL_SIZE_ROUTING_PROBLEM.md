# The Model Size Problem: Why Simple Backend Switching Doesn't Work

## 🚨 The Problem You Identified

**User is absolutely correct:** You can't just say "NVIDIA too hot, use NPU" because:

```
NVIDIA GPU (24GB VRAM):
  ✓ Can run: llama3:70b (40GB with quantization)
  ✓ Can run: llama3:7b (4GB)
  ✓ Can run: qwen2.5:0.5b (500MB)

NPU (Shared system RAM, 3W power):
  ✗ Cannot run: llama3:70b (too large, too slow)
  ✗ Cannot run: llama3:7b (maybe, but extremely slow)
  ✓ Can run: qwen2.5:0.5b (yes, this is what it's designed for)
```

## 🎯 The Real Scenario

### What Currently Happens (WRONG!)

```
User Request:
{
  "prompt": "Analyze this code",
  "model": "llama3:70b",    ← 70 BILLION parameters!
  "annotations": {
    "latency_critical": true
  }
}

Current Mode: Quiet (max 40% fan)
NVIDIA fan: 65%

Current System (naive):
✗ "NVIDIA too loud, routing to NPU"
✗ Attempts to load llama3:70b on NPU
✗ NPU: "Out of memory" or takes 5 minutes per token!

BROKEN!
```

### What SHOULD Happen (Multiple Options)

## 💡 Solution Options

### Option 1: Model-Aware Routing (Backend Capabilities)

**Each backend declares what models it can handle:**

```yaml
backends:
  - id: "ollama-npu"
    hardware: "npu"
    capabilities:
      max_model_size_gb: 2
      supported_models:
        - "qwen2.5:0.5b"
        - "qwen2.5:1.5b"
        - "tinyllama:1b"
      model_patterns:
        - "*:0.5b"
        - "*:1.5b"

  - id: "ollama-igpu"
    hardware: "igpu"
    capabilities:
      max_model_size_gb: 8
      supported_models:
        - "qwen2.5:0.5b"
        - "qwen2.5:1.5b"
        - "llama3:7b"
        - "mistral:7b"

  - id: "ollama-nvidia"
    hardware: "nvidia"
    capabilities:
      max_model_size_gb: 24
      supported_models:
        - "*"  # Can run anything
```

**Routing logic:**

```go
func RouteRequest(model string, annotations) {
    // 1. Filter backends by model capability
    capable := filterByModelSupport(model)
    // capable = [nvidia, igpu] (NPU excluded for llama3:70b)

    // 2. Apply efficiency mode to capable backends
    if mode == Quiet {
        // Prefer quiet backends from capable list
        if igpu.fanSpeed < 40% {
            return igpu  // Intel GPU can handle 7b, is quiet
        }
        return error("llama3:70b requires loud backend in Quiet mode")
    }
}
```

**User sees:**
```json
{
  "error": "Model llama3:70b requires NVIDIA GPU, but Quiet mode blocks it (fan 65%)",
  "suggestion": "Use smaller model (llama3:7b) or switch to Balanced mode",
  "available_models_on_quiet_backends": ["qwen2.5:1.5b", "llama3:7b"]
}
```

---

### Option 2: Automatic Model Substitution (Downgrade)

**Maintain model equivalency mapping:**

```yaml
model_equivalents:
  # Quality tiers for same task
  high_quality:
    - "llama3:70b"
    - "mixtral:8x7b"

  medium_quality:
    - "llama3:7b"
    - "mistral:7b"

  low_quality:
    - "qwen2.5:1.5b"
    - "phi-2:2.7b"

  ultra_efficient:
    - "qwen2.5:0.5b"
    - "tinyllama:1b"

# Substitution rules
substitution_policy:
  # When routing to NPU, use smallest model
  npu:
    substitute_to: "ultra_efficient"

  # When routing to Intel GPU, use medium models
  igpu:
    substitute_to: "medium_quality"

  # NVIDIA can use requested model
  nvidia:
    substitute_to: "requested"
```

**Routing with substitution:**

```
User Request: model="llama3:70b"
Mode: Quiet
NVIDIA: Blocked (fan too loud)

Router:
1. Check model compatibility
2. llama3:70b not supported on NPU/Intel GPU
3. Apply substitution policy
4. NPU → qwen2.5:0.5b
5. Intel GPU → llama3:7b (better quality)
6. Choose Intel GPU (best available)

Response:
{
  "backend_used": "ollama-igpu",
  "model_requested": "llama3:70b",
  "model_used": "llama3:7b",
  "model_substitution": true,
  "substitution_reason": "Quiet mode, llama3:70b requires NVIDIA",
  "quality_impact": "medium (downgraded from high)",
  "user_notification": "Using llama3:7b instead of llama3:70b due to Quiet mode"
}
```

---

### Option 3: Mode Override (Model Takes Priority)

**Large models override efficiency mode:**

```go
func RouteRequest(model string, mode EfficiencyMode) {
    // Check model size
    if isLargeModel(model) {  // e.g., >7B parameters
        if mode == Quiet || mode == Efficiency {
            log("Large model %s overrides %s mode, using NVIDIA", model, mode)
            return nvidia, "Large model requires NVIDIA despite mode"
        }
    }

    // Small models respect mode
    if isSmallModel(model) {  // e.g., <3B parameters
        // Can route to NPU/Intel GPU based on mode
    }
}
```

**User sees:**
```json
{
  "backend_used": "ollama-nvidia",
  "efficiency_mode": "Quiet",
  "mode_overridden": true,
  "override_reason": "Model llama3:70b requires NVIDIA GPU (24GB VRAM)",
  "warning": "Quiet mode bypassed due to large model requirement",
  "fan_speed": "65% (exceeds Quiet mode 40% limit)"
}
```

---

### Option 4: Tiered Fallback Strategy

**Try in order of preference:**

```
Request: llama3:70b, Mode: Quiet

Attempt 1: NPU (preferred for Quiet mode)
  ✗ Model too large (70b > 2b limit)

Attempt 2: Intel GPU (second choice)
  ✗ Model too large (70b > 8b limit)

Attempt 3: NVIDIA (last resort)
  ⚠ Model fits BUT fan is 65% (> 40% limit)

Decision:
  Option A: Use NVIDIA anyway (model requirement wins)
  Option B: Reject request (mode enforcement wins)
  Option C: Ask user to choose

Response (Option A - Model Wins):
{
  "backend_used": "ollama-nvidia",
  "mode_preference": "ollama-npu",
  "mode_overridden": true,
  "override_reason": "Model too large for Quiet mode backends",
  "fallback_chain": ["npu (rejected: model too large)",
                     "igpu (rejected: model too large)",
                     "nvidia (accepted: only option)"],
  "warning": "Using loud backend (65% fan) because model requires it"
}
```

---

## 🎯 Recommended Approach: Hybrid

**Combine multiple strategies:**

### 1. Model Registry (Per Backend)

```go
type Backend interface {
    // ... existing methods
    SupportsModel(modelName string) bool
    GetMaxModelSize() int  // GB
    GetSupportedModelPattern() []string
}

// Ollama NPU implementation
func (b *OllamaNPU) SupportsModel(model string) bool {
    // Check model size
    if strings.Contains(model, "70b") {
        return false  // Too large
    }
    if strings.Contains(model, "0.5b") || strings.Contains(model, "1.5b") {
        return true  // Perfect for NPU
    }
    return false
}
```

### 2. Smart Routing with Warnings

```go
func RouteWithModelAwareness(model string, mode EfficiencyMode) RoutingDecision {
    // Get all healthy backends
    candidates := getHealthyBackends()

    // Filter by model support
    capable := []Backend{}
    for _, b := range candidates {
        if b.SupportsModel(model) {
            capable = append(capable, b)
        }
    }

    if len(capable) == 0 {
        return error("No backend supports model %s", model)
    }

    // Apply efficiency mode to capable backends
    preferred := applyEfficiencyMode(capable, mode)

    // Check if preferred backend violates mode
    if violatesMode(preferred, mode) {
        warning := fmt.Sprintf("Model %s requires %s which violates %s mode",
            model, preferred.ID(), mode)

        // Return with warning
        return RoutingDecision{
            Backend: preferred,
            Warning: warning,
            ModeOverridden: true,
        }
    }

    return RoutingDecision{Backend: preferred}
}
```

### 3. User Configuration

```yaml
routing:
  # What to do when model doesn't fit efficiency mode
  model_mode_conflict: "prefer_model"  # or "prefer_mode" or "ask_user"

  # Model substitution
  allow_model_substitution: true
  substitution_quality_threshold: "medium"  # Don't go below medium quality

  # Model-specific overrides
  model_routing:
    "llama3:70b":
      force_backend: "ollama-nvidia"
      ignore_efficiency_mode: true

    "qwen2.5:0.5b":
      prefer_backend: "ollama-npu"
      respect_efficiency_mode: true
```

---

## 💬 The Discussion

### Your Point is Critical Because:

**1. Hardware Heterogeneity is Real**
- NPU: Tiny models only (0.5b-1.5b)
- Intel GPU: Small-medium models (up to 7b)
- NVIDIA: Large models (up to 70b+)

**2. Can't Blindly Switch Backends**
- Model might not fit in memory
- Model might run 100x slower
- Model might not be available on that backend

**3. Need Model-Aware Routing**
- Check model compatibility BEFORE thermal/efficiency
- Or substitute to compatible model
- Or override efficiency mode for large models

**4. User Needs to Know**
```json
{
  "warning": "llama3:70b requires NVIDIA, Quiet mode overridden",
  "alternative": "Use llama3:7b for Quiet mode compliance",
  "quality_tradeoff": "7b model is 10x smaller, may reduce quality"
}
```

---

## 🔧 What We Should Implement

### Phase 1: Model Capability Checking (Critical)

```go
// Add to Backend interface
type ModelCapability struct {
    MaxModelSizeGB int
    SupportedModels []string
    ModelPattern []string
}

// Check before routing
if !backend.SupportsModel(requestedModel) {
    // Find alternative or reject
}
```

### Phase 2: Smart Fallback

```go
// Try preferred backend first
if preferred.SupportsModel(model) {
    return preferred
}

// Try alternatives
for _, alt := range alternatives {
    if alt.SupportsModel(model) {
        return alt, warning("Using %s instead of %s", alt, preferred)
    }
}

// No backend supports it
return error("Model too large for available backends in current mode")
```

### Phase 3: Model Substitution (Optional)

```go
// If large model doesn't fit efficiency mode
if !fitsMode(model, mode) {
    smaller := findSmallerEquivalent(model)
    return smaller, warning("Substituted %s → %s for %s mode", model, smaller, mode)
}
```

---

## 📊 Real Example

### Scenario: User Wants Large Model in Quiet Mode

**Request:**
```json
{
  "prompt": "Analyze this codebase",
  "model": "llama3:70b",
  "annotations": {
    "prefer_power_efficiency": true
  }
}
```

**Current mode:** `Quiet` (max 40% fan)

**Backend capabilities:**
- NPU: Max 1.5b models
- Intel GPU: Max 7b models
- NVIDIA: Max 70b models (but fan at 65%)

**Smart routing decision:**

**Option A: Model Wins (Recommended)**
```json
{
  "backend_used": "ollama-nvidia",
  "model_used": "llama3:70b",
  "mode_overridden": true,
  "override_reason": "Model llama3:70b requires NVIDIA GPU (only capable backend)",
  "efficiency_mode_violated": true,
  "warning": "Quiet mode bypassed: Model too large for quiet backends",
  "fan_speed": "65% (exceeds Quiet mode 40% limit)",
  "recommendation": "Use llama3:7b on Intel GPU for Quiet mode compliance"
}
```

**Option B: Mode Wins (Strict)**
```json
{
  "error": "Model llama3:70b incompatible with Quiet mode",
  "reason": "Quiet mode limits fan to 40%, llama3:70b requires NVIDIA (65% fan)",
  "suggested_models": [
    "llama3:7b (can run on Intel GPU, 35% fan)",
    "qwen2.5:1.5b (can run on NPU, 0% fan)"
  ],
  "or": "Change to Balanced mode: ai-efficiency set Balanced"
}
```

**Option C: Model Substitution (User-Friendly)**
```json
{
  "backend_used": "ollama-igpu",
  "model_requested": "llama3:70b",
  "model_used": "llama3:7b",
  "model_substituted": true,
  "substitution_reason": "Quiet mode compliance",
  "quality_impact": "May reduce output quality (70b → 7b)",
  "fan_speed": "35% (within Quiet mode limit)",
  "energy_saved_wh": 0.042,
  "note": "Use Performance mode for llama3:70b"
}
```

---

## ✅ Summary

**You're absolutely right:** We need to add:

1. ✅ **Model capability declarations** per backend
2. ✅ **Model-aware filtering** before thermal/efficiency routing
3. ✅ **Fallback strategies** when model doesn't fit mode
4. ✅ **User warnings** about model substitution
5. ✅ **Configuration** for model vs mode priority

**This is a CRITICAL addition to make the system actually work in practice!**

Would you like me to implement the model capability checking system?
