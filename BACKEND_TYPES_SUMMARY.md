# Backend Types Summary

## Question: Can I Add Other Backend Types (SDK/API/PAT/GAT)?

**YES! Absolutely.** ✅

The proxy is designed to support **any backend** that implements the `Backend` interface.

## What's Already Implemented

### 1. Ollama Backend ✅
- **Type:** `ollama`
- **Use:** Local Ollama instances on different hardware
- **Auth:** None (local endpoints)
- **Location:** `pkg/backends/ollama/ollama.go`

### 2. OpenAI API Backend ✅ (Just Added!)
- **Type:** `openai`
- **Use:** OpenAI API (GPT-4, GPT-3.5, o1, etc.)
- **Auth:** API Key (Bearer token)
- **Location:** `pkg/backends/openai/openai.go`

### 3. Anthropic API Backend ✅ (Just Added!)
- **Type:** `anthropic`
- **Use:** Claude API (Sonnet, Opus, Haiku)
- **Auth:** API Key (x-api-key header)
- **Location:** `pkg/backends/anthropic/anthropic.go`

## Authentication Methods Supported

### API Key (Bearer Token)
```yaml
- id: "openai"
  type: "openai"
  api_key_env: "OPENAI_API_KEY"  # From environment
  # OR
  api_key: "sk-..."  # Direct (not recommended)
```

**Implementation:**
```go
req.Header.Set("Authorization", "Bearer " + apiKey)
```

### API Key (Custom Header)
```yaml
- id: "anthropic"
  type: "anthropic"
  api_key_env: "ANTHROPIC_API_KEY"
```

**Implementation:**
```go
req.Header.Set("x-api-key", apiKey)
```

### PAT (Personal Access Token)
Easy to add - same as API key:
```yaml
- id: "github-copilot"
  type: "github"
  pat_env: "GITHUB_TOKEN"
```

**Implementation:**
```go
req.Header.Set("Authorization", "token " + pat)
```

### OAuth2 / GAT (Generic Access Token)
Also supported:
```go
// Use golang.org/x/oauth2
config := &oauth2.Config{...}
token, _ := config.Token(ctx)
req.Header.Set("Authorization", "Bearer " + token.AccessToken)
```

## Quick Example: Mixed Configuration

```yaml
backends:
  # Local Ollama (free)
  - id: "ollama-nvidia"
    type: "ollama"
    endpoint: "http://localhost:11436"
    priority: 9

  # OpenAI API (paid)
  - id: "openai-gpt4"
    type: "openai"
    api_key_env: "OPENAI_API_KEY"
    priority: 6

  # Claude API (paid)
  - id: "anthropic-claude"
    type: "anthropic"
    api_key_env: "ANTHROPIC_API_KEY"
    priority: 8
```

**Routing behavior:**
- `llama3:70b` → ollama-nvidia (local, free)
- `gpt-4` → openai-gpt4 (cloud, paid)
- `claude-3-5-sonnet` → anthropic-claude (cloud, paid)

## Easy to Add

### Google Gemini
```go
// pkg/backends/google/google.go
type GoogleBackend struct {
    apiKey string
    // ...
}

func (b *GoogleBackend) Generate(...) {
    req.Header.Set("Authorization", "Bearer " + b.apiKey)
    // Call https://generativelanguage.googleapis.com/v1/models/...
}
```

### Hugging Face Inference API
```go
// pkg/backends/huggingface/huggingface.go
type HuggingFaceBackend struct {
    token string
    // ...
}

func (b *HuggingFaceBackend) Generate(...) {
    req.Header.Set("Authorization", "Bearer " + b.token)
    // Call https://api-inference.huggingface.co/models/...
}
```

### Together.ai
```go
// pkg/backends/together/together.go
type TogetherBackend struct {
    apiKey string
}
```

### Custom vLLM Server
```go
// pkg/backends/vllm/vllm.go
type VLLMBackend struct {
    endpoint string
}
```

## Interface Requirements

Any backend must implement these 24 methods:

```go
type Backend interface {
    // Identification (4)
    ID() string
    Type() string
    Name() string
    Hardware() string

    // Health (2)
    IsHealthy() bool
    HealthCheck(ctx context.Context) error

    // Characteristics (3)
    PowerWatts() float64
    AvgLatencyMs() int32
    Priority() int

    // Capabilities (4)
    SupportsGenerate() bool
    SupportsStream() bool
    SupportsEmbed() bool
    ListModels(ctx context.Context) ([]string, error)

    // Model capabilities (4)
    SupportsModel(modelName string) bool
    GetMaxModelSizeGB() int
    GetSupportedModelPatterns() []string
    GetPreferredModels() []string

    // Operations (3)
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
    GenerateStream(ctx context.Context, req *GenerateRequest) (StreamReader, error)
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)

    // Metrics (2)
    UpdateMetrics(latencyMs int32, success bool)
    GetMetrics() *BackendMetrics

    // Lifecycle (2)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

## Real-World Use Cases

### Use Case 1: Cost Optimization
```
Priority setup:
- Local Ollama: Priority 9 (free)
- Cloud API: Priority 5 (paid)

Result: Uses local when possible, cloud only when needed
```

### Use Case 2: Model Availability
```
Backends:
- ollama-nvidia: llama3:70b ✅
- openai: gpt-4 ✅
- anthropic: claude-3-5-sonnet ✅

Request GPT-4 → Routes to OpenAI (only one that has it)
```

### Use Case 3: Reliability
```
Primary: Local Ollama
Fallback: OpenAI API (if local down)

Local fails → Auto-switches to cloud
```

### Use Case 4: Geographic Distribution
```
- Local: Instant (on-premise)
- Cloud US: Low latency for US users
- Cloud EU: GDPR-compliant for EU
```

## Benefits of Multi-Backend Support

### 1. Zero Lock-In
- Switch between providers easily
- Try new models without refactoring
- Compare outputs side-by-side

### 2. Cost Control
```
Simple query → Local NPU (free, 3W)
Complex code → Local NVIDIA (free, 55W)
Critical work → OpenAI GPT-4 (paid, best quality)
```

### 3. Redundancy
```
Primary backend down? → Automatically use fallback
No manual intervention needed
```

### 4. Feature Access
```
Local: Privacy, offline, unlimited usage
Cloud: Latest models, no hardware needed
```

## All Routing Features Work

**Model-aware routing:**
- ✅ Checks if backend supports requested model
- ✅ Substitutes models when needed
- ✅ Cloud backends have no size limits (max_model_size_gb: 999)

**Thermal monitoring:**
- ✅ Local backends: Full thermal monitoring
- ✅ Cloud backends: Always "healthy" (no hardware to overheat)

**Efficiency modes:**
- ✅ Local backends: Respect power/fan limits
- ✅ Cloud backends: Power = 0 (no local consumption)

**Workload detection:**
- ✅ Works for all backends
- ✅ Realtime → Prefers local NPU (low latency)
- ✅ Code → Can prefer Claude (best for code)

## Configuration Files

**Basic:** `config/config.yaml` - Ollama only
**Mixed:** `config/config-mixed-backends.yaml` - Ollama + Cloud APIs

## Getting Started

### 1. Set API Keys
```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

### 2. Update Config
Use `config-mixed-backends.yaml` or add to existing config:
```yaml
backends:
  - id: "openai"
    type: "openai"
    api_key_env: "OPENAI_API_KEY"
    # ...
```

### 3. Update main.go
Add backend registration:
```go
import "github.com/daoneill/ollama-proxy/pkg/backends/openai"

case "openai":
    backend, err := openai.NewOpenAIBackend(...)
```

### 4. Build & Run
```bash
make build
./bin/ollama-proxy
```

## Testing

```bash
# Test OpenAI
grpcurl -d '{
  "model": "gpt-4",
  "prompt": "Hello!"
}' localhost:50051 compute.v1.ComputeService/Generate

# Test Claude
grpcurl -d '{
  "model": "claude-3-5-sonnet-20241022",
  "prompt": "Write code"
}' localhost:50051 compute.v1.ComputeService/Generate

# Test local Ollama
grpcurl -d '{
  "model": "llama3:7b",
  "prompt": "Hello!"
}' localhost:50051 compute.v1.ComputeService/Generate
```

## Documentation

- **Full Guide:** `ADDING_BACKENDS.md`
- **Implementation:** `pkg/backends/openai/openai.go`
- **Implementation:** `pkg/backends/anthropic/anthropic.go`
- **Example Config:** `config/config-mixed-backends.yaml`

## Summary

**Question:** Can I add SDK/API/PAT/GAT backends?

**Answer:** ✅ YES!

**What's supported:**
- ✅ API Key authentication (OpenAI, Anthropic)
- ✅ Custom headers (x-api-key, etc.)
- ✅ PAT (Personal Access Tokens)
- ✅ OAuth2 / GAT (Generic Access Tokens)
- ✅ Any HTTP-based API

**What's included:**
- ✅ OpenAI backend (fully implemented)
- ✅ Anthropic backend (fully implemented)
- ✅ Template for adding more

**All routing features work:**
- ✅ Model-aware routing
- ✅ Thermal monitoring (N/A for cloud)
- ✅ Efficiency modes
- ✅ Workload detection
- ✅ Smart fallback

**You can mix:**
- Local Ollama (free, power-aware)
- Cloud APIs (paid, always available)
- Custom servers (your choice)

The proxy treats all backends equally! 🚀
