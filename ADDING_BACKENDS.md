## Adding Different Backend Types

## Yes! The proxy supports multiple backend types

The `Backend` interface is designed to be extensible. You can add:

- ✅ **API backends** (OpenAI, Anthropic, Google AI, etc.)
- ✅ **SDK-based backends** (Direct SDK integration)
- ✅ **Token-authenticated services** (PAT, Bearer tokens, API keys)
- ✅ **Custom backends** (Your own inference servers)

## Currently Implemented

### 1. Ollama Backend ✅
- Type: `ollama`
- Location: `pkg/backends/ollama/`
- Use: Local Ollama instances on different hardware

### 2. OpenAI Backend ✅ (NEW!)
- Type: `openai`
- Location: `pkg/backends/openai/`
- Use: OpenAI API (GPT-4, GPT-3.5, etc.)

### 3. Anthropic Backend ✅ (NEW!)
- Type: `anthropic`
- Location: `pkg/backends/anthropic/`
- Use: Claude API (Sonnet, Opus, Haiku)

## Configuration Examples

### OpenAI API Backend

```yaml
backends:
  - id: "openai-gpt4"
    type: "openai"
    name: "OpenAI GPT-4"
    hardware: "cloud"
    enabled: true
    api_key_env: "OPENAI_API_KEY"  # Read from env var
    # OR: api_key: "sk-..."  # Direct API key (not recommended)

    characteristics:
      power_watts: 0  # Cloud service (no local power)
      avg_latency_ms: 500
      max_tokens_per_second: 50
      priority: 8

    model_capability:
      supported_model_patterns:
        - "gpt-*"
        - "o1-*"
      preferred_models:
        - "gpt-4-turbo"
        - "gpt-4"
        - "gpt-3.5-turbo"
```

### Anthropic (Claude) Backend

```yaml
backends:
  - id: "anthropic-claude"
    type: "anthropic"
    name: "Anthropic Claude"
    hardware: "cloud"
    enabled: true
    api_key_env: "ANTHROPIC_API_KEY"

    characteristics:
      power_watts: 0
      avg_latency_ms: 600
      max_tokens_per_second: 45
      priority: 9

    model_capability:
      supported_model_patterns:
        - "claude-*"
      preferred_models:
        - "claude-3-5-sonnet-20241022"
        - "claude-3-opus-20240229"
```

### Mixed Configuration Example

Combine local Ollama + cloud APIs:

```yaml
backends:
  # Local hardware backends
  - id: "ollama-npu"
    type: "ollama"
    hardware: "npu"
    endpoint: "http://localhost:11434"
    # ... (existing config)

  - id: "ollama-nvidia"
    type: "ollama"
    hardware: "nvidia"
    endpoint: "http://localhost:11436"
    # ... (existing config)

  # Cloud API backends
  - id: "openai-gpt4"
    type: "openai"
    hardware: "cloud"
    api_key_env: "OPENAI_API_KEY"
    characteristics:
      power_watts: 0
      avg_latency_ms: 500
      priority: 7  # Lower priority (costs money!)
    model_capability:
      supported_model_patterns:
        - "gpt-*"

  - id: "anthropic-claude"
    type: "anthropic"
    hardware: "cloud"
    api_key_env: "ANTHROPIC_API_KEY"
    characteristics:
      power_watts: 0
      avg_latency_ms: 600
      priority: 8
    model_capability:
      supported_model_patterns:
        - "claude-*"
```

## Routing Behavior with Mixed Backends

### Example 1: Use Local First, Cloud Fallback

```yaml
backends:
  # Priority 10: Try local NVIDIA first
  - id: "ollama-nvidia"
    type: "ollama"
    priority: 10

  # Priority 8: Cloud fallback if local unavailable
  - id: "openai-gpt4"
    type: "openai"
    priority: 8
```

**Request:**
```
Model: gpt-4-turbo
```

**Routing:**
1. Check NVIDIA: ❌ Doesn't support gpt-4-turbo
2. Check OpenAI: ✅ Supports gpt-4-turbo
3. Route to: OpenAI

### Example 2: Cloud for Complex, Local for Simple

```yaml
backends:
  # NPU for simple queries
  - id: "ollama-npu"
    priority: 5
    model_capability:
      supported_model_patterns:
        - "*:0.5b"

  # NVIDIA for complex local
  - id: "ollama-nvidia"
    priority: 8
    model_capability:
      supported_model_patterns:
        - "llama3:70b"

  # GPT-4 for when you need the best
  - id: "openai-gpt4"
    priority: 10  # Highest priority!
    model_capability:
      supported_model_patterns:
        - "gpt-4*"
```

**Smart routing:**
- Simple query → NPU (qwen2.5:0.5b)
- Complex code → NVIDIA (llama3:70b)
- Critical analysis → OpenAI GPT-4

## Integration with main.go

Update `cmd/proxy/main.go` to register new backend types:

```go
import (
    "github.com/daoneill/ollama-proxy/pkg/backends/ollama"
    "github.com/daoneill/ollama-proxy/pkg/backends/openai"
    "github.com/daoneill/ollama-proxy/pkg/backends/anthropic"
)

// In loadBackends function
switch backendCfg.Type {
case "ollama":
    // Existing Ollama code...

case "openai":
    backend, err := openai.NewOpenAIBackend(openai.Config{
        BackendConfig: backends.BackendConfig{
            ID:              backendCfg.ID,
            Name:            backendCfg.Name,
            PowerWatts:      backendCfg.Characteristics.PowerWatts,
            AvgLatencyMs:    backendCfg.Characteristics.AvgLatencyMs,
            Priority:        backendCfg.Characteristics.Priority,
            ModelCapability: modelCap,
        },
        APIKeyEnv: backendCfg.APIKeyEnv,  // Need to add this to config struct
    })

case "anthropic":
    backend, err := anthropic.NewAnthropicBackend(anthropic.Config{
        BackendConfig: backends.BackendConfig{
            ID:              backendCfg.ID,
            Name:            backendCfg.Name,
            PowerWatts:      backendCfg.Characteristics.PowerWatts,
            AvgLatencyMs:    backendCfg.Characteristics.AvgLatencyMs,
            Priority:        backendCfg.Characteristics.Priority,
            ModelCapability: modelCap,
        },
        APIKeyEnv: backendCfg.APIKeyEnv,
    })
}
```

## Creating Custom Backends

### Template for New Backend Type

```go
package mybackend

import (
    "context"
    "github.com/daoneill/ollama-proxy/pkg/backends"
)

type MyBackend struct {
    // Your implementation
}

// Must implement all Backend interface methods:
func (b *MyBackend) ID() string { ... }
func (b *MyBackend) Type() string { return "mybackend" }
func (b *MyBackend) Name() string { ... }
func (b *MyBackend) Hardware() string { ... }
func (b *MyBackend) IsHealthy() bool { ... }
func (b *MyBackend) HealthCheck(ctx context.Context) error { ... }
func (b *MyBackend) PowerWatts() float64 { ... }
func (b *MyBackend) AvgLatencyMs() int32 { ... }
func (b *MyBackend) Priority() int { ... }
func (b *MyBackend) SupportsGenerate() bool { ... }
func (b *MyBackend) SupportsStream() bool { ... }
func (b *MyBackend) SupportsEmbed() bool { ... }
func (b *MyBackend) ListModels(ctx context.Context) ([]string, error) { ... }
func (b *MyBackend) SupportsModel(modelName string) bool { ... }
func (b *MyBackend) GetMaxModelSizeGB() int { ... }
func (b *MyBackend) GetSupportedModelPatterns() []string { ... }
func (b *MyBackend) GetPreferredModels() []string { ... }
func (b *MyBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) { ... }
func (b *MyBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) { ... }
func (b *MyBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) { ... }
func (b *MyBackend) UpdateMetrics(latencyMs int32, success bool) { ... }
func (b *MyBackend) GetMetrics() *backends.BackendMetrics { ... }
func (b *MyBackend) Start(ctx context.Context) error { ... }
func (b *MyBackend) Stop(ctx context.Context) error { ... }
```

## Common Backend Types

### Google AI (Gemini)

```yaml
- id: "google-gemini"
  type: "google"  # You'd implement this
  hardware: "cloud"
  api_key_env: "GOOGLE_AI_API_KEY"
  model_capability:
    supported_model_patterns:
      - "gemini-*"
    preferred_models:
      - "gemini-1.5-pro"
      - "gemini-1.5-flash"
```

### Hugging Face Inference API

```yaml
- id: "huggingface"
  type: "huggingface"  # You'd implement this
  hardware: "cloud"
  api_key_env: "HF_TOKEN"
  model_capability:
    supported_model_patterns:
      - "*"  # Supports many models
```

### Together.ai

```yaml
- id: "together"
  type: "together"  # You'd implement this
  hardware: "cloud"
  api_key_env: "TOGETHER_API_KEY"
  model_capability:
    supported_model_patterns:
      - "meta-llama/*"
      - "mistralai/*"
```

### Custom vLLM Server

```yaml
- id: "vllm-server"
  type: "vllm"  # You'd implement this
  hardware: "custom"
  endpoint: "http://my-vllm-server:8000"
  model_capability:
    supported_model_patterns:
      - "*"
```

## Authentication Types Supported

### 1. API Key (Header)
```go
req.Header.Set("Authorization", "Bearer " + apiKey)
```

### 2. API Key (Custom Header)
```go
req.Header.Set("x-api-key", apiKey)
```

### 3. PAT (Personal Access Token)
```go
req.Header.Set("Authorization", "token " + pat)
```

### 4. OAuth2
```go
// Use oauth2 package
token, _ := config.Token(ctx)
req.Header.Set("Authorization", "Bearer " + token.AccessToken)
```

## Benefits of Multi-Backend Support

### 1. Cost Optimization
```
Simple queries → Local NPU (free, 3W)
Complex tasks → Local NVIDIA (free, 55W)
Critical work → OpenAI GPT-4 (paid, high quality)
```

### 2. Reliability
```
Primary: Local Ollama
Fallback: Cloud API (if local unavailable)
```

### 3. Model Availability
```
Local: llama3:70b (offline capable)
Cloud: gpt-4-turbo (latest features)
Cloud: claude-3-opus (best at coding)
```

### 4. Geographic Distribution
```
Local: Ollama (instant, on-premise)
Cloud US: OpenAI (low latency US)
Cloud EU: Custom API (GDPR compliant)
```

## Testing Mixed Backends

```bash
# Test local Ollama
grpcurl -d '{"model": "llama3:7b", "prompt": "Hello"}' \
  localhost:50051 compute.v1.ComputeService/Generate

# Test OpenAI (if configured)
grpcurl -d '{"model": "gpt-4", "prompt": "Hello"}' \
  localhost:50051 compute.v1.ComputeService/Generate

# Test Claude (if configured)
grpcurl -d '{"model": "claude-3-5-sonnet-20241022", "prompt": "Hello"}' \
  localhost:50051 compute.v1.ComputeService/Generate
```

## Environment Setup

```bash
# Set API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_AI_API_KEY="..."

# Start proxy (will use env vars)
./bin/ollama-proxy
```

## Summary

**What's Supported:**
- ✅ Ollama (local hardware)
- ✅ OpenAI API (GPT models)
- ✅ Anthropic API (Claude models)
- ✅ Any backend implementing the interface

**Easy to Add:**
- Google Gemini
- Hugging Face
- Together.ai
- vLLM servers
- Custom APIs

**Routing is Smart:**
- Checks model compatibility
- Considers power consumption (0 for cloud)
- Respects priority settings
- Falls back gracefully

**All backends participate in:**
- Model-aware routing
- Thermal monitoring (N/A for cloud, always healthy)
- Efficiency modes
- Workload detection
- Metrics collection

The proxy treats all backends equally - whether local Ollama or cloud API! 🚀
