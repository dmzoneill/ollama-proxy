# Annotation Override Behavior - Complete Guide

## 🔀 How Efficiency Modes and User Annotations Interact

### The Hierarchy (Top = Highest Priority)

```
1. SYSTEM SAFETY (Always enforced)
   ├─ Temperature > 85°C     → Backend excluded (cannot override)
   ├─ Thermal throttling     → Backend excluded (cannot override)
   └─ Hardware offline       → Backend excluded (cannot override)

2. EFFICIENCY MODE LIMITS
   ├─ Max power watts        → Hard limit per mode
   ├─ Max fan speed          → Hard limit per mode
   └─ Max temperature        → Hard limit per mode

3. USER ANNOTATIONS (May be overridden by mode)
   ├─ latency_critical       → Depends on mode setting
   ├─ prefer_power_efficiency → Depends on mode setting
   └─ target=specific_backend → Depends on mode setting

4. DEFAULT ROUTING
   └─ Smart routing based on complexity
```

## 📊 Override Behavior by Mode

| Mode | Overrides User's Critical Flag? | Overrides Explicit Target? | Respects User Preference? |
|------|--------------------------------|----------------------------|--------------------------|
| **Performance** | ❌ No | ❌ No | ✅ Yes (maximum freedom) |
| **Balanced** | ✅ Yes (if unjustified) | ⚠️ Thermal only | ✅ Mostly |
| **Efficiency** | ✅ Yes (always) | ⚠️ Thermal + power | ⚠️ Limited |
| **Quiet** | ✅ Yes (always) | ⚠️ Thermal + fan | ⚠️ Limited |
| **Auto** | ✅ Yes (context-dependent) | ⚠️ Context-dependent | ⚠️ Adaptive |
| **Ultra Efficiency** | ✅ Yes (always) | ✅ Yes (NPU only) | ❌ No |

## 🎮 Detailed Examples

### Scenario 1: Performance Mode (No Overrides)

```yaml
Mode: Performance
User Request:
  prompt: "Generate code"
  annotations:
    latency_critical: true
    target: "ollama-nvidia"

System Decision:
  ✓ Respects latency_critical flag
  ✓ Uses NVIDIA as requested
  ✓ No overrides applied

Result:
  Backend: ollama-nvidia
  Reason: "Performance mode, user preference respected"
  Latency: ~150ms (fast!)
```

**Exception:** Even in Performance mode, thermal safety is enforced:
```yaml
NVIDIA Temperature: 87°C (> 85°C critical!)

System Decision:
  ✗ Cannot use NVIDIA (thermal safety)
  ✓ Routes to Intel GPU instead

Result:
  Backend: ollama-igpu
  Reason: "NVIDIA too hot (87°C), thermal safety override"
  User sees: Warning that thermal limit prevented NVIDIA use
```

### Scenario 2: Balanced Mode (Smart Overrides)

```yaml
Mode: Balanced
User Request:
  prompt: "What is 2+2?"
  annotations:
    latency_critical: true  # User says it's critical

System Analysis:
  Classifier: SIMPLE query detected
  Expected tokens: ~1
  Complexity: Very low

System Decision:
  ✗ Override latency_critical flag (unjustified)
  ✓ Routes to NPU instead of NVIDIA

Result:
  Backend: ollama-npu
  Reason: "Simple query, NPU sufficient despite critical flag"
  Latency: ~800ms (slower but acceptable for simple task)
  Energy saved: 77%

User Response Metadata:
  {
    "backend_used": "ollama-npu",
    "user_requested": "ollama-nvidia",
    "override_reason": "Simple query classification",
    "latency_ms": 823,
    "energy_wh": 0.0007
  }
```

**But respects justified critical flags:**
```yaml
Mode: Balanced
User Request:
  prompt: "URGENT: Quick code completion needed for IDE"
  annotations:
    latency_critical: true

System Analysis:
  Pattern: "URGENT", "Quick", "IDE"
  Context: Real-time application
  Complexity: MODERATE (code completion)

System Decision:
  ✓ Critical flag justified
  ✓ Uses NVIDIA

Result:
  Backend: ollama-nvidia
  Reason: "Latency-critical request justified, using NVIDIA"
  Latency: ~150ms
```

### Scenario 3: Efficiency Mode (Aggressive Overrides)

```yaml
Mode: Efficiency
User Request:
  prompt: "Write detailed analysis"
  annotations:
    latency_critical: true
    target: "ollama-nvidia"

System Analysis:
  Mode max power: 15W
  NVIDIA power: 55W (exceeds limit!)
  Classification: COMPLEX

System Decision:
  ✗ Override target (power limit)
  ✗ Override latency_critical (mode policy)
  ✓ Routes to Intel GPU (12W, within limit)

Result:
  Backend: ollama-igpu
  Reason: "Efficiency mode, power budget (15W) exceeded, using Intel GPU"
  Latency: ~350ms (slower than NVIDIA but within mode limits)
  Energy saved: 78%

User Response Metadata:
  {
    "backend_used": "ollama-igpu",
    "user_requested": "ollama-nvidia",
    "override_reason": "Efficiency mode power limit (15W)",
    "latency_ms": 367,
    "requested_latency_critical": true,
    "critical_flag_overridden": true
  }
```

### Scenario 4: Quiet Mode (Fan Limit Override)

```yaml
Mode: Quiet
User Request:
  prompt: "Any request"
  annotations:
    latency_critical: true
    target: "ollama-nvidia"

System Check:
  NVIDIA fan speed: 65%
  Quiet mode limit: 40%
  NVIDIA excluded: Fan too loud!

System Decision:
  ✗ Cannot use NVIDIA (fan limit)
  ✗ Override all user preferences
  ✓ Routes to NPU (silent)

Result:
  Backend: ollama-npu
  Reason: "Quiet mode, NVIDIA fan too loud (65% > 40%), using NPU"
  Latency: ~800ms
  Fan noise: 0% (silent!)

User sees:
  "Quiet mode active: NVIDIA blocked due to fan speed.
   Using NPU for silent operation."
```

### Scenario 5: Auto Mode (Context-Aware Overrides)

**Example A: Battery Critical**
```yaml
Mode: Auto
Battery: 15%
Time: 3:00 PM
Effective Mode: Ultra Efficiency (battery critical)

User Request:
  prompt: "Write essay"
  annotations:
    latency_critical: true
    target: "ollama-nvidia"

System Decision:
  Auto mode → Ultra Efficiency (battery < 20%)
  ✗ Override ALL user preferences
  ✓ Force NPU only

Result:
  Backend: ollama-npu
  Reason: "Auto mode: Battery critical (15%), forced to NPU"
  Notification: "Battery below 20%. Using power-saving mode (NPU only).
                Plug in for higher performance."
```

**Example B: Quiet Hours**
```yaml
Mode: Auto
Battery: 75%
Time: 11:30 PM (quiet hours)
Effective Mode: Quiet

User Request:
  prompt: "Background task"
  annotations:
    latency_critical: true

System Decision:
  Auto mode → Quiet (11:30 PM)
  ✗ Override latency_critical
  ✓ Use silent backends only

Result:
  Backend: ollama-npu
  Reason: "Auto mode: Quiet hours (10pm-6am), using NPU"
```

**Example C: Normal Conditions**
```yaml
Mode: Auto
Battery: 85% on AC
Time: 2:00 PM
Temperature: All cool
Effective Mode: Performance

User Request:
  prompt: "Generate code"
  annotations:
    latency_critical: true

System Decision:
  Auto mode → Performance (good conditions)
  ✓ Respect user preferences
  ✓ Use NVIDIA

Result:
  Backend: ollama-nvidia
  Reason: "Auto mode: Optimal conditions, using NVIDIA"
```

## 🎛️ Configuration Control

You can customize override behavior in `config/config.yaml`:

```yaml
efficiency:
  modes:
    Balanced:
      override_critical_flag: true          # Allow overriding critical
      override_conditions:
        - "simple_query"                    # Override if simple
        - "battery_low"                     # Override if battery < 30%
      respect_explicit_target: true         # Respect target= unless thermal

    Efficiency:
      override_critical_flag: true
      override_explicit_target: false       # Can override target=
      force_power_limit: true               # Strictly enforce 15W

    Quiet:
      override_critical_flag: true
      override_explicit_target: true        # Override everything for silence
      force_fan_limit: true                 # Strictly enforce 40% fan

    Performance:
      override_critical_flag: false         # Never override
      override_explicit_target: false       # Never override
      respect_all_annotations: true         # Maximum user control
```

## 📋 Decision Matrix

### Will My Request Be Overridden?

| Your Request | Performance | Balanced | Efficiency | Quiet | Auto |
|-------------|-------------|----------|------------|-------|------|
| `latency_critical=true` (simple query) | ✅ Respected | ❌ Override to NPU | ❌ Override | ❌ Override | ⚠️ Depends |
| `latency_critical=true` (complex) | ✅ Respected | ✅ Respected | ⚠️ Maybe* | ❌ Override | ⚠️ Depends |
| `target=ollama-nvidia` (cool) | ✅ Respected | ✅ Respected | ⚠️ If < 15W | ❌ If loud | ⚠️ Depends |
| `target=ollama-nvidia` (87°C) | ❌ Thermal | ❌ Thermal | ❌ Thermal | ❌ Thermal | ❌ Thermal |
| `prefer_power_efficiency=true` | ✅ Respected | ✅ Respected | ✅ Respected | ✅ Respected | ✅ Respected |

*In Efficiency mode, complex tasks might still use Intel GPU instead of NVIDIA to stay within power budget

## 🚨 Non-Overridable (Safety First)

These are NEVER overridden by any mode or annotation:

```yaml
Hardware Safety:
  ✗ Temperature > 95°C        → Emergency shutdown threshold
  ✗ Temperature > 85°C        → Critical, backend excluded
  ✗ Thermal throttling active → Backend excluded
  ✗ Hardware offline/unhealthy → Backend excluded
  ✗ GPU crashed/error         → Backend excluded

Example:
  Mode: Performance (allows everything)
  User: target=ollama-nvidia, critical=true
  NVIDIA: 88°C

  Result: STILL cannot use NVIDIA
  Reason: "Thermal safety is non-overridable"
```

## 💡 Best Practices

### For Users

**Want guaranteed speed?**
```bash
# Set Performance mode
ai-efficiency set Performance

# Then your annotations are respected
{
  "latency_critical": true,
  "target": "ollama-nvidia"
}
# → Will use NVIDIA (unless thermal emergency)
```

**Want guaranteed efficiency?**
```bash
# Set Efficiency mode
ai-efficiency set Efficiency

# Your annotations may be overridden
{
  "latency_critical": true   # Ignored
}
# → Will use NPU/Intel GPU regardless
```

**Want best of both?**
```bash
# Set Auto mode
ai-efficiency set Auto

# System adapts intelligently
# - On battery low: Overrides for efficiency
# - On AC plugged: Respects your preferences
# - Temperature high: Overrides for cooling
```

### For Applications

**Critical real-time app (IDE, video, etc.):**
```bash
# Recommend Performance mode to users
ai-efficiency set Performance

# Then your critical flags are respected
```

**Background tasks:**
```bash
# Don't set critical=true
# Let system route to efficient backends
```

**User-controlled:**
```bash
# Let user choose mode in app settings
# Your app respects their efficiency preference
```

## 📊 Override Statistics

You can track how often overrides happen:

```bash
curl http://localhost:8080/metrics/overrides

{
  "total_requests": 1000,
  "overrides": {
    "critical_flag_overridden": 234,
    "target_overridden": 45,
    "thermal_safety": 12,
    "power_limit": 67,
    "fan_limit": 34
  },
  "override_reasons": {
    "simple_query": 180,
    "efficiency_mode": 67,
    "quiet_mode": 34,
    "thermal": 12,
    "battery_critical": 23
  }
}
```

## 🎯 Summary

**Override Priority:**
```
1. Thermal Safety         → ALWAYS enforced (non-overridable)
2. Efficiency Mode Limits → Enforced based on mode
3. User Annotations       → Respected when compatible with mode
4. Smart Routing          → Default behavior
```

**Quick Rules:**
- **Performance mode**: Your annotations always respected (except thermal safety)
- **Balanced mode**: Annotations respected if justified
- **Efficiency/Quiet mode**: Annotations overridden to meet efficiency/noise goals
- **Auto mode**: Dynamic based on conditions
- **Ultra Efficiency**: All annotations overridden (NPU only)

**The system prioritizes:** Safety > Mode Goals > User Preference > Default Routing

You always get transparency:
```json
{
  "backend_used": "ollama-npu",
  "user_requested": "ollama-nvidia",
  "override_applied": true,
  "override_reason": "Efficiency mode power limit (15W)",
  "user_annotation_respected": false
}
```
