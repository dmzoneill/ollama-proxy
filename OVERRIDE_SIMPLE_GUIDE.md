# Does Efficiency Mode Override My Annotations? - Simple Answer

## 🎯 Yes, But It Depends on the Mode!

```
┌────────────────────────────────────────────────────────┐
│  Mode            Override User Requests?               │
├────────────────────────────────────────────────────────┤
│  🚀 Performance   NO  - Always respects your choices   │
│  ⚖️ Balanced      SOMETIMES - If unjustified           │
│  🔋 Efficiency    YES - Enforces 15W power limit       │
│  🔇 Quiet         YES - Enforces 40% fan limit         │
│  🤖 Auto          DEPENDS - Based on conditions        │
│  🪫 Ultra         YES - NPU only, everything overridden│
└────────────────────────────────────────────────────────┘
```

## 🔥 Thermal Safety ALWAYS Overrides (All Modes)

```
Your Request: "Use NVIDIA GPU" [latency_critical=true]
NVIDIA Temp: 87°C (too hot!)

Result: Uses Intel GPU instead
Reason: "Thermal safety (non-overridable)"

→ Even Performance mode can't override thermal safety!
```

## 📊 Visual Decision Flow

```
User Request: "Generate code" [latency_critical=true, target=nvidia]
                              │
                              ▼
                    ┌─────────────────┐
                    │ Check Mode      │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌────────────────┐   ┌────────────────┐
│ Performance   │   │ Balanced       │   │ Efficiency     │
│ Mode          │   │ Mode           │   │ Mode           │
└───────┬───────┘   └────────┬───────┘   └───────┬────────┘
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌────────────────┐   ┌────────────────┐
│ Check Thermal │   │ Check Thermal  │   │ Check Thermal  │
│ 65°C (OK)     │   │ 65°C (OK)      │   │ 65°C (OK)      │
└───────┬───────┘   └────────┬───────┘   └───────┬────────┘
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌────────────────┐   ┌────────────────┐
│ ✅ Use NVIDIA │   │ Classify Query │   │ Check Power    │
│               │   │ → COMPLEX      │   │ 55W > 15W!     │
│ Respects your │   └────────┬───────┘   └───────┬────────┘
│ choice        │            │                    │
└───────────────┘            ▼                    ▼
                    ┌────────────────┐   ┌────────────────┐
                    │ ✅ Use NVIDIA  │   │ ❌ Use Intel   │
                    │                │   │                │
                    │ Complex task   │   │ Override to    │
                    │ justified      │   │ stay in 15W    │
                    └────────────────┘   └────────────────┘
```

## 💡 Simple Examples

### Example 1: Performance Mode
```
You: "Use NVIDIA, it's critical!" [latency_critical=true]
Mode: Performance

System: ✅ "OK, using NVIDIA as requested"
```

### Example 2: Balanced Mode (Simple Query)
```
You: "What is 2+2?" [latency_critical=true]
Mode: Balanced

System: ❌ "This is a simple query, using NPU instead"
         "Override reason: Simple query doesn't need NVIDIA"
```

### Example 3: Balanced Mode (Complex Query)
```
You: "Write detailed code analysis" [latency_critical=true]
Mode: Balanced

System: ✅ "Complex task detected, using NVIDIA as requested"
```

### Example 4: Efficiency Mode
```
You: "Use NVIDIA!" [latency_critical=true]
Mode: Efficiency (15W limit)

System: ❌ "Efficiency mode active, NVIDIA (55W) exceeds limit"
         "Using Intel GPU (12W) instead"
```

### Example 5: Quiet Mode
```
You: "Use NVIDIA!" [latency_critical=true]
Mode: Quiet (40% fan limit)
NVIDIA fan: 65%

System: ❌ "Quiet mode active, NVIDIA fan too loud (65% > 40%)"
         "Using NPU (silent) instead"
```

## 🎮 Which Mode Should I Use?

### Want your annotations ALWAYS respected?
```bash
ai-efficiency set Performance
```
→ Your choices are respected (except thermal safety)

### Want smart overrides to save power?
```bash
ai-efficiency set Balanced
```
→ Overrides unjustified critical flags, respects complex tasks

### Want maximum battery life?
```bash
ai-efficiency set Efficiency
```
→ Will override to stay within 15W power budget

### Want silence above all?
```bash
ai-efficiency set Quiet
```
→ Will override to keep fans below 40%

### Want automatic decisions?
```bash
ai-efficiency set Auto
```
→ Switches modes based on battery, temperature, time

## 📋 Quick Reference

| You Want | Set This Mode | Your Annotations |
|----------|---------------|------------------|
| Maximum control | Performance | Always respected ✅ |
| Smart balance | Balanced | Mostly respected ⚖️ |
| Save battery | Efficiency | Often overridden 🔋 |
| Stay quiet | Quiet | Often overridden 🔇 |
| Hands-free | Auto | Depends on context 🤖 |

## ⚠️ What's NEVER Overridden

```
Temperature > 85°C      → Backend excluded (safety)
Thermal throttling      → Backend excluded (safety)
Hardware offline        → Backend excluded (safety)
GPU crashed            → Backend excluded (safety)
```

**Even in Performance mode with `critical=true`, these safety rules apply!**

## 🔍 How to Tell if You Were Overridden

Every response tells you:

```json
{
  "backend_used": "ollama-npu",
  "user_requested": "ollama-nvidia",        ← You asked for this
  "override_applied": true,                  ← You were overridden
  "override_reason": "Efficiency mode power limit (15W)",
  "routing": {
    "reason": "Efficiency mode, staying within power budget"
  }
}
```

## 🎯 Summary

**Simple Answer:**
- **Performance mode** = Your annotations control everything
- **Other modes** = System may override for efficiency/quiet/thermal reasons
- **Thermal safety** = Always enforced, no exceptions

**You always see:**
- What backend was used
- Why it was chosen
- If you were overridden
- The specific reason

**Pro tip:** Use Performance mode when you need guaranteed speed, let other modes optimize when you don't.
