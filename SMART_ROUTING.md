# Smart Request Categorization - Preventing "Everything is Critical"

## The Problem

If users can just mark every request as `latency_critical=true`, the system degrades to:
- Always using NVIDIA GPU (highest power)
- Defeating the purpose of power-aware routing
- Draining battery quickly
- No actual prioritization

## Solutions

### 1. **Automatic Prompt Classification** ⭐ RECOMMENDED

The proxy automatically analyzes prompts to determine real complexity:

```go
// Example: User says "latency_critical=true" but prompt is simple
Request: "What is 2+2?"
Annotations: { latency_critical: true }

Classifier detects:
  - Short prompt (12 chars)
  - Simple pattern ("what is")
  - Expected output: 1 token

Routing decision: Override user's critical flag → Use NPU
Reason: "Simple query detected, NPU sufficient despite critical flag"
```

#### Classification Tiers

| Complexity | Characteristics | Backend | Example |
|------------|----------------|---------|---------|
| **SIMPLE** | • Short prompts (< 50 chars)<br>• Factual questions<br>• Yes/no answers<br>• Single-word responses | **NPU** (3W) | "What is the capital of France?"<br>"How many days in a week?"<br>"True or false: Earth is flat?" |
| **MODERATE** | • Normal explanations<br>• Standard chat<br>• Brief summaries<br>• Medium prompts (50-200 chars) | **Intel GPU** (12W) | "Explain photosynthesis"<br>"Summarize this paragraph"<br>"How does a car engine work?" |
| **COMPLEX** | • Long-form writing<br>• Code generation<br>• Detailed analysis<br>• Creative writing<br>• Multi-step reasoning | **NVIDIA** (55W) | "Write a 500-word essay on AI"<br>"Generate a Python web scraper"<br>"Analyze this code for bugs" |

#### Automatic Detection Patterns

```python
SIMPLE patterns:
  ✓ "what is", "who is", "when was", "where is"
  ✓ "yes or no", "true or false"
  ✓ Contains "briefly", "one sentence", "in short"
  ✓ Short prompts (< 50 chars)

COMPLEX patterns:
  ✓ "write a detailed", "explain in depth"
  ✓ "analyze", "compare and contrast"
  ✓ "generate code", "create a comprehensive"
  ✓ "write an essay", "compose", "develop a plan"

MODERATE:
  Everything else
```

### 2. **Policy-Based Quotas**

Users get limited high-power backend access based on tier:

| Tier | Daily Energy Budget | NVIDIA Quota/Hour | Cost/Day |
|------|---------------------|-------------------|----------|
| **Free** | 10 Wh | 5 requests | $0 |
| **Basic** | 50 Wh | 20 requests | $1 |
| **Premium** | 200 Wh | 100 requests | $10 |
| **Enterprise** | Unlimited | Unlimited | Custom |

**Example scenario (Free tier):**
```
User makes 6 requests/hour marked as critical:

Request 1-5: ✅ NVIDIA (within quota)
Request 6: ❌ Quota exceeded → Auto-downgraded to Intel GPU

Response: "NVIDIA quota exceeded (5/5 per hour). Using Intel GPU instead.
          Resets in 42 minutes. Or upgrade to Premium tier."
```

### 3. **Battery-Aware Policies**

System automatically adjusts based on battery state:

| Battery Level | Max Power | Allowed Backends | Behavior |
|---------------|-----------|------------------|----------|
| **< 20%** | 5W | NPU only | Critical power saving |
| **20-50%** | 15W | NPU, Intel GPU | Conservative |
| **50-80%** | 30W | NPU, Intel GPU, limited NVIDIA | Balanced |
| **> 80%** | Unlimited | All backends | Full performance |
| **On AC** | Unlimited | All backends | No restrictions |

**Auto-downgrade example:**
```
User request: { latency_critical: true }
Battery: 15%

System overrides:
  ✗ NVIDIA (55W) - exceeds 5W limit
  ✓ NPU (3W) - within limit

Response metadata:
  "backend_used": "ollama-npu"
  "reason": "Battery critical (15%), overrode latency_critical flag"
  "requested_backend": "ollama-nvidia"
```

### 4. **Time-Based Policies**

Different behaviors based on time of day:

```go
Quiet Hours (10pm - 6am):
  - Prefer silent, low-power backends (NPU, Intel GPU)
  - Deprioritize high-fan NVIDIA GPU
  - Good for overnight batch jobs

Peak Hours (9am - 5pm weekdays):
  - Allow high-performance backends
  - Users actively working

Off-Peak (evenings, weekends):
  - Balanced approach
```

### 5. **Two-Tier Inference** (Advanced)

Use NPU to classify, then route:

```
┌─────────────┐
│ User Request│
└──────┬──────┘
       │
       ▼
┌─────────────────────────────┐
│ NPU Classification          │  ← Fast, 3W, ~100ms
│ "Is this SIMPLE/COMPLEX?"   │
└──────┬──────────────────────┘
       │
       ├─── SIMPLE ───────────► NPU (3W, ~800ms)
       │
       ├─── MODERATE ─────────► Intel GPU (12W, ~350ms)
       │
       └─── COMPLEX ──────────► NVIDIA (55W, ~150ms)
```

**Energy savings:**
- Classification: 0.083 Wh (100ms @ 3W)
- Simple request on NPU: 0.67 Wh
- **Total: 0.75 Wh**

vs. always using NVIDIA:
- Direct to NVIDIA: 2.29 Wh
- **Savings: 67%**

### 6. **Model-Based Auto-Routing**

Different models have different requirements:

```yaml
Model routing rules:
  qwen2.5:0.5b  → NPU/Intel GPU    # Small, efficient
  qwen2.5:1.5b  → Intel GPU/NVIDIA # Medium
  llama3:7b     → NVIDIA/Intel GPU # Large
  llama3:70b    → NVIDIA only      # Very large

Embeddings    → NPU              # Always lightweight
```

### 7. **Application-Level Context**

Different apps have different default behaviors:

```go
Application profiles:

Email Summarizer:
  default_backend: "ollama-npu"
  reason: Background task, not time-sensitive
  allow_critical_override: false

IDE Code Completion:
  default_backend: "ollama-nvidia"
  reason: User actively typing, needs speed
  allow_critical_override: true

Chatbot:
  default_backend: "ollama-igpu"
  reason: Interactive but not ultra-time-sensitive
  allow_critical_override: true

Document Search:
  default_backend: "ollama-npu"
  reason: Embeddings are lightweight
  allow_critical_override: false
```

## Implementation Strategy

### Phase 1: Heuristic Classification (No ML)

```go
func ClassifyRequest(prompt string, model string) string {
    // Fast, deterministic, no inference needed

    if len(prompt) < 50 && containsSimplePattern(prompt) {
        return "ollama-npu"
    }

    if containsComplexPattern(prompt) || isLargeModel(model) {
        return "ollama-nvidia"
    }

    return "ollama-igpu" // Default
}
```

**Pros:** Instant, no overhead, predictable
**Cons:** Less accurate than ML

### Phase 2: NPU-Based Classification

```go
func ClassifyWithNPU(prompt string) string {
    // Use tiny model on NPU to classify
    classification := npu.Generate("Classify as SIMPLE/MODERATE/COMPLEX: " + prompt)

    // Maps to appropriate backend
    return mapClassificationToBackend(classification)
}
```

**Pros:** More accurate, still fast (~100ms), low power (3W)
**Cons:** Adds latency, uses NPU capacity

### Phase 3: Learning System

```go
func LearnFromResults(prompt string, backend string, actualLatency int, userSatisfaction bool) {
    // Track which requests actually benefited from NVIDIA
    // Downgrade future similar requests if NVIDIA wasn't needed

    if backend == "nvidia" && actualLatency > 500 && userSatisfaction {
        // Complex request, NVIDIA was needed
        recordPattern(prompt, "complex")
    } else if backend == "nvidia" && actualLatency < 200 {
        // Simple request, wasted NVIDIA capacity
        recordPattern(prompt, "simple")
        suggestDowngrade(prompt, "igpu")
    }
}
```

## Practical Example Scenarios

### Scenario 1: Aggressive User

**User:** Marks everything as `latency_critical=true`

**System response:**
```
Request 1: "What is 2+2?" [critical=true]
→ Classifier: SIMPLE
→ Override: Use NPU (3W)
→ Response: "Backend: ollama-npu, Reason: Simple query, critical flag overridden"

Request 2: "Write detailed essay on AI" [critical=true]
→ Classifier: COMPLEX
→ Allow: Use NVIDIA (55W)
→ Response: "Backend: ollama-nvidia, Reason: Complex request, critical flag justified"

Request 3-7: More critical requests
→ Quota: NVIDIA quota exceeded (5/hour)
→ Downgrade: Use Intel GPU
→ Response: "Backend: ollama-igpu, Reason: NVIDIA quota exceeded, resets in 30min"
```

**Result:** User can't abuse the system.

### Scenario 2: Battery-Constrained Mobile

**Device:** Laptop at 18% battery

**System behavior:**
```
Request 1: [latency_critical=true]
→ Battery: 18% (< 20% threshold)
→ Max power: 5W
→ Force: NPU only
→ Response: "Backend: ollama-npu, Reason: Battery critical, all requests routed to NPU"

User notification: "Battery below 20%. Using power-saving mode.
                    High-performance backends disabled until plugged in or > 50%."
```

### Scenario 3: Overnight Batch Job

**Request:** "Process 1000 documents" at 2am

**System behavior:**
```
Time: 2:00am (quiet hours)
Request complexity: HIGH
User marking: [critical=true]

Policy check:
→ Quiet hours: true
→ Time-sensitive: false (batch job)
→ Override critical flag

Route to: NPU (silent, efficient)
Response: "Backend: ollama-npu
          Reason: Quiet hours + batch job detected
          Estimated completion: 45 minutes
          Energy cost: 2.25 Wh (vs 12 Wh on NVIDIA)"
```

## Integration with Existing Proxy

Add to router decision logic:

```go
func (r *Router) RouteRequest(ctx context.Context, annotations *Annotations) (*Decision, error) {
    // 1. Classify request
    complexity := r.classifier.ClassifyPrompt(ctx, req.Prompt, req.Model)

    // 2. Check policies
    backend, err := r.policy.GetRecommendedBackend(
        userID,
        userTier,
        annotations.Target,
        estimatedTokens,
    )
    if err != nil {
        // Policy violation, use suggested alternative
        log.Printf("Policy override: %v", err)
    }

    // 3. Validate critical flag
    if annotations.LatencyCritical {
        justified := r.classifier.ShouldAllowLatencyCritical(ctx, req.Prompt, true)
        if !justified {
            annotations.LatencyCritical = false
            log.Printf("Critical flag overridden: prompt not time-sensitive")
        }
    }

    // 4. Apply battery policy
    if r.policy.ShouldThrottle() {
        maxPower := r.policy.MaxAllowedPower()
        // Filter backends by power
    }

    // 5. Continue with existing routing logic
    return r.routeWithPolicies(ctx, annotations, complexity, backend)
}
```

## Configuration

Add to `config/config.yaml`:

```yaml
policies:
  enabled: true

  # Default user tier (if no authentication)
  default_tier: "free"

  # Quotas
  quotas:
    free:
      daily_energy_wh: 10
      nvidia_per_hour: 5
    basic:
      daily_energy_wh: 50
      nvidia_per_hour: 20

  # Battery thresholds
  battery:
    critical_percent: 20     # NPU only below this
    low_percent: 50          # Limit NVIDIA below this
    conservative_percent: 80 # Balanced below this

  # Time-based
  quiet_hours:
    enabled: true
    start: "22:00"
    end: "06:00"
    prefer_backends: ["ollama-npu", "ollama-igpu"]

classification:
  enabled: true
  method: "heuristic"  # "heuristic", "npu", or "hybrid"
  override_user_critical: true  # Allow overriding user's critical flag

  # NPU classification (if method=npu)
  npu_classifier:
    backend: "ollama-npu"
    model: "qwen2.5:0.5b"
    max_classification_time_ms: 200
```

## Monitoring

Track policy enforcement:

```bash
GET /metrics/policies

Response:
{
  "critical_flags_overridden": 145,
  "quota_violations": 23,
  "battery_downgrades": 67,
  "quiet_hour_reroutes": 34,
  "energy_saved_wh": 234.5,
  "cost_saved_usd": 12.34
}
```

## Summary

**Best approach:** Combine multiple strategies

1. ✅ **Heuristic classification** (fast, no overhead)
2. ✅ **Quota system** (prevents abuse)
3. ✅ **Battery policies** (automatic power management)
4. ✅ **Time-based rules** (context-aware)
5. ⏭️ **Optional: NPU classification** (more accurate)
6. ⏭️ **Optional: Learning system** (improve over time)

This prevents users from gaming the system while still allowing genuinely critical requests to get fast backends.
