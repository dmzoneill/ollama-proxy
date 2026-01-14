# Request Flow: User Flags vs. Efficiency Profile Overrides

## ✅ Yes, You Understand Correctly!

**User provides annotations → Efficiency mode can override them**

## 📊 Complete Request Flow

```
┌─────────────────────────────────────────────────────┐
│  CLIENT (gRPC/HTTP Request)                         │
├─────────────────────────────────────────────────────┤
│  {                                                  │
│    "prompt": "Generate code",                       │
│    "model": "qwen2.5:0.5b",                        │
│    "annotations": {                                 │
│      "latency_critical": true,      ← USER REQUEST │
│      "target": "ollama-nvidia"      ← USER REQUEST │
│    }                                                │
│  }                                                  │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│  PROXY RECEIVES REQUEST                             │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│  CHECK EFFICIENCY MODE                              │
│  Current Mode: Quiet                                │
│  Profile Settings:                                  │
│    - max_power_watts: 15                           │
│    - max_fan_percent: 40                           │
│    - override_critical_flag: true                  │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│  APPLY EFFICIENCY MODE OVERRIDES                    │
│                                                     │
│  User wants: NVIDIA (55W, fan could be 65%)        │
│  Mode limit: 15W max, 40% fan max                  │
│                                                     │
│  ❌ OVERRIDE: NVIDIA exceeds limits                │
│  ✓  Use: NPU or Intel GPU instead                 │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│  CHECK THERMAL STATE                                │
│  NPU: 55°C, fan 0%     ✓ OK                       │
│  Intel GPU: 62°C, 35%  ✓ OK                       │
│  NVIDIA: 78°C, 65%     ❌ Blocked by mode          │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│  FINAL ROUTING DECISION                             │
│  Selected: ollama-npu                               │
│  Reason: "Quiet mode, NVIDIA blocked (fan > 40%)"  │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│  RESPONSE TO CLIENT                                 │
├─────────────────────────────────────────────────────┤
│  {                                                  │
│    "response": "Generated text...",                 │
│    "backend_used": "ollama-npu",     ← ACTUAL      │
│    "user_requested": "ollama-nvidia", ← WHAT USER WANTED │
│    "override_applied": true,                        │
│    "override_reason": "Quiet mode fan limit (40%)", │
│    "routing": {                                     │
│      "reason": "Quiet mode active, NVIDIA fan too loud" │
│    }                                                │
│  }                                                  │
└─────────────────────────────────────────────────────┘
```

## 🎮 Real Examples

### Example 1: Performance Mode (NO Override)

**Client sends:**
```json
{
  "prompt": "Write code",
  "annotations": {
    "latency_critical": true,
    "target": "ollama-nvidia"
  }
}
```

**Current mode:** `Performance`

**What happens:**
```
✓ User wants NVIDIA → Performance mode respects this
✓ Check thermal: NVIDIA 65°C (OK)
✓ Route to: NVIDIA

Response:
{
  "backend_used": "ollama-nvidia",
  "override_applied": false,
  "routing": {
    "reason": "Performance mode, user request honored"
  }
}
```

**User got what they requested!**

---

### Example 2: Quiet Mode (OVERRIDE Applied)

**Client sends:**
```json
{
  "prompt": "Write code",
  "annotations": {
    "latency_critical": true,
    "target": "ollama-nvidia"
  }
}
```

**Current mode:** `Quiet`

**What happens:**
```
✗ User wants NVIDIA
✓ Quiet mode check: NVIDIA fan at 65%
✗ Quiet mode limit: 40% max fan
✗ NVIDIA BLOCKED by profile
✓ Route to: NPU (0% fan)

Response:
{
  "backend_used": "ollama-npu",
  "user_requested": "ollama-nvidia",
  "override_applied": true,
  "override_reason": "Quiet mode, NVIDIA fan too loud (65% > 40%)",
  "routing": {
    "reason": "Quiet mode enforced, using silent backend"
  }
}
```

**User request OVERRIDDEN by Quiet mode!**

---

### Example 3: Efficiency Mode (OVERRIDE Applied)

**Client sends:**
```json
{
  "prompt": "Simple query",
  "annotations": {
    "latency_critical": true
  }
}
```

**Current mode:** `Efficiency`

**What happens:**
```
✗ User says latency_critical
✓ Efficiency mode classifies: "Simple query"
✗ Override critical flag (unjustified for simple query)
✓ Efficiency mode limit: 15W
✗ NVIDIA uses 55W (exceeds limit)
✓ Route to: NPU (3W)

Response:
{
  "backend_used": "ollama-npu",
  "override_applied": true,
  "override_reason": "Efficiency mode power limit (15W), simple query",
  "annotations_respected": false,
  "routing": {
    "reason": "Simple query + efficiency mode, using NPU"
  }
}
```

**Both target and critical flag OVERRIDDEN!**

---

### Example 4: Auto Mode (Context-Dependent)

**Client sends:**
```json
{
  "prompt": "Generate analysis",
  "annotations": {
    "latency_critical": true,
    "target": "ollama-nvidia"
  }
}
```

**Current mode:** `Auto`
**Context:** Battery 15%, Time 11:30 PM

**What happens:**
```
✓ Auto mode detects:
  - Battery 15% (critical!)
  - Time 11:30 PM (quiet hours)
✓ Auto switches effective mode to: Ultra Efficiency

✗ User wants NVIDIA
✗ Ultra Efficiency mode: NPU ONLY
✓ Route to: NPU

Response:
{
  "backend_used": "ollama-npu",
  "user_requested": "ollama-nvidia",
  "override_applied": true,
  "override_reason": "Auto mode → Ultra Efficiency (battery 15%)",
  "effective_mode": "UltraEfficiency",
  "routing": {
    "reason": "Battery critical, using power-saving mode"
  }
}
```

**OVERRIDDEN by Auto mode's context awareness!**

---

## 🔑 Key Points

### 1. **User ALWAYS Gets Transparency**

Every response includes:
```json
{
  "backend_used": "ollama-npu",           ← What actually happened
  "user_requested": "ollama-nvidia",      ← What user asked for
  "override_applied": true,               ← Was user overridden?
  "override_reason": "Quiet mode...",     ← Why?
  "annotations_respected": false          ← Were annotations honored?
}
```

### 2. **Override Hierarchy**

```
Priority 1: THERMAL SAFETY (never overridable)
  ├─ Temp > 85°C → Backend excluded
  ├─ Throttling → Backend excluded
  └─ Hardware offline → Backend excluded

Priority 2: EFFICIENCY MODE LIMITS
  ├─ Max power (e.g., 15W in Efficiency mode)
  ├─ Max fan speed (e.g., 40% in Quiet mode)
  └─ Max temperature (per mode)

Priority 3: USER ANNOTATIONS
  ├─ latency_critical
  ├─ target=backend
  └─ prefer_power_efficiency

Priority 4: DEFAULT ROUTING
  └─ Smart complexity-based
```

### 3. **Control via Efficiency Mode**

```bash
# Want user annotations ALWAYS respected?
ai-efficiency set Performance
→ User flags control routing (except thermal safety)

# Want system to optimize?
ai-efficiency set Auto
→ System overrides based on context

# Want power savings?
ai-efficiency set Efficiency
→ System enforces 15W limit, overrides as needed

# Want silence?
ai-efficiency set Quiet
→ System enforces 40% fan limit, blocks loud backends
```

## 📝 gRPC Request Example

### Client Code

```go
// Client sends request with annotations
resp, err := client.Generate(ctx, &pb.GenerateRequest{
    Prompt: "Generate code",
    Model:  "qwen2.5:0.5b",
    Annotations: &pb.JobAnnotations{
        LatencyCritical: true,           // User wants speed
        Target:         "ollama-nvidia",  // User wants NVIDIA
    },
})

// Check if overridden
if resp.Routing.Override_Applied {
    fmt.Printf("Request overridden: %s\n", resp.Routing.OverrideReason)
    fmt.Printf("Requested: %s, Got: %s\n",
        resp.UserRequested, resp.BackendUsed)
}
```

### Server Response (Quiet Mode Active)

```json
{
  "response": "Here's the generated code...",
  "backend_used": "ollama-npu",
  "user_requested": "ollama-nvidia",
  "override_applied": true,
  "override_reason": "Quiet mode, NVIDIA fan 65% > 40% limit",
  "routing": {
    "backend": "ollama-npu",
    "reason": "Quiet mode enforced, using silent NPU",
    "estimated_power_watts": 3.0,
    "estimated_latency_ms": 800,
    "alternatives": ["ollama-igpu"]
  },
  "stats": {
    "total_time_ms": 823,
    "tokens_generated": 150,
    "tokens_per_second": 18.2,
    "energy_wh": 0.0007
  },
  "annotations_respected": false
}
```

## 🎯 Summary

**Yes, your understanding is EXACTLY correct:**

1. **Client sends:** Annotations/flags (latency_critical, target, etc.)
2. **Proxy checks:** Current efficiency mode profile
3. **Profile can override:** Based on mode settings
4. **Client always knows:** Response shows what was requested vs. what happened

**The efficiency mode is like a system-wide policy that can override individual request preferences.**

Think of it like:
- **Performance mode** = "Users are in charge"
- **Other modes** = "System optimizes, may override users"
- **Thermal safety** = "Always enforced, no exceptions"

You control this with:
- **GUI:** System menu → AI Efficiency → Select mode
- **CLI:** `ai-efficiency set <mode>`
- **Config:** `config/config.yaml` default mode
