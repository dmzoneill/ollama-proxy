package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// AnthropicBackend implements Backend interface for Anthropic Claude API
type AnthropicBackend struct {
	mu sync.RWMutex

	// Config
	id       string
	name     string
	apiKey   string
	endpoint string

	// Characteristics
	powerWatts   float64 // 0 for cloud services
	avgLatencyMs int32
	priority     int

	// Model capabilities
	modelCapability *backends.ModelCapability

	// Health
	healthy      atomic.Bool
	lastCheck    time.Time
	checkTimeout time.Duration

	// Metrics
	metrics *backends.BackendMetrics

	// HTTP client
	client *http.Client
}

// Config for Anthropic backend
type Config struct {
	backends.BackendConfig
	APIKey    string // Direct API key
	APIKeyEnv string // Or env var name
	Endpoint  string // Optional: custom endpoint
}

// NewAnthropicBackend creates a new Anthropic API backend
func NewAnthropicBackend(cfg Config) (*AnthropicBackend, error) {
	// Get API key from env if specified
	apiKey := cfg.APIKey
	if cfg.APIKeyEnv != "" {
		apiKey = os.Getenv(cfg.APIKeyEnv)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key required (set APIKey or APIKeyEnv)")
	}

	// Default endpoint
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1"
	}

	backend := &AnthropicBackend{
		id:              cfg.ID,
		name:            cfg.Name,
		apiKey:          apiKey,
		endpoint:        endpoint,
		powerWatts:      cfg.PowerWatts,
		avgLatencyMs:    cfg.AvgLatencyMs,
		priority:        cfg.Priority,
		modelCapability: cfg.ModelCapability,
		checkTimeout:    5 * time.Second,
		metrics: &backends.BackendMetrics{
			LoadedModels: []string{},
		},
		client: &http.Client{
			Timeout: 120 * time.Second, // Claude can be slower
		},
	}

	backend.healthy.Store(false)
	return backend, nil
}

// ID returns backend identifier
func (b *AnthropicBackend) ID() string {
	return b.id
}

// Type returns backend type
func (b *AnthropicBackend) Type() string {
	return "anthropic"
}

// Name returns human-readable name
func (b *AnthropicBackend) Name() string {
	return b.name
}

// Hardware returns hardware type
func (b *AnthropicBackend) Hardware() string {
	return "cloud"
}

// IsHealthy returns current health status
func (b *AnthropicBackend) IsHealthy() bool {
	return b.healthy.Load()
}

// HealthCheck performs health check (simple request test)
func (b *AnthropicBackend) HealthCheck(ctx context.Context) error {
	// Anthropic doesn't have a models endpoint, so we do a minimal request
	// Just verify API key is valid by setting healthy to true if configured
	// Real validation happens on first request
	if b.apiKey != "" {
		b.healthy.Store(true)
		b.mu.Lock()
		b.lastCheck = time.Now()
		b.mu.Unlock()
		return nil
	}

	b.healthy.Store(false)
	return fmt.Errorf("API key not configured")
}

// PowerWatts returns estimated power consumption (0 for cloud)
func (b *AnthropicBackend) PowerWatts() float64 {
	return b.powerWatts
}

// AvgLatencyMs returns average latency
func (b *AnthropicBackend) AvgLatencyMs() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.metrics.RequestCount > 0 {
		return b.metrics.AvgLatencyMs
	}
	return b.avgLatencyMs
}

// Priority returns backend priority
func (b *AnthropicBackend) Priority() int {
	return b.priority
}

// SupportsGenerate returns true
func (b *AnthropicBackend) SupportsGenerate() bool {
	return true
}

// SupportsStream returns true
func (b *AnthropicBackend) SupportsStream() bool {
	return true
}

// SupportsEmbed returns false (Anthropic doesn't offer embeddings)
func (b *AnthropicBackend) SupportsEmbed() bool {
	return false
}

// ListModels returns available Claude models
func (b *AnthropicBackend) ListModels(ctx context.Context) ([]string, error) {
	// Anthropic doesn't have a models list endpoint
	// Return known models
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

// SupportsModel checks if this backend can run the specified model
func (b *AnthropicBackend) SupportsModel(modelName string) bool {
	if b.modelCapability == nil {
		// Default: support all Claude models
		return strings.HasPrefix(modelName, "claude-")
	}

	// Check excluded patterns
	for _, pattern := range b.modelCapability.ExcludedPatterns {
		if matchesPattern(modelName, pattern) {
			return false
		}
	}

	// Check supported patterns
	if len(b.modelCapability.SupportedModelPatterns) == 0 {
		return true
	}

	for _, pattern := range b.modelCapability.SupportedModelPatterns {
		if matchesPattern(modelName, pattern) {
			return true
		}
	}

	return false
}

// GetMaxModelSizeGB returns maximum model size (N/A for cloud)
func (b *AnthropicBackend) GetMaxModelSizeGB() int {
	return 999
}

// GetSupportedModelPatterns returns patterns of supported models
func (b *AnthropicBackend) GetSupportedModelPatterns() []string {
	if b.modelCapability == nil {
		return []string{"claude-*"}
	}
	return b.modelCapability.SupportedModelPatterns
}

// GetPreferredModels returns list of preferred models
func (b *AnthropicBackend) GetPreferredModels() []string {
	if b.modelCapability == nil {
		return []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"}
	}
	return b.modelCapability.PreferredModels
}

// Generate performs text generation via Anthropic API
func (b *AnthropicBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	start := time.Now()

	// Build Anthropic API request
	anthropicReq := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"max_tokens": 4096, // Default
	}

	if req.Options != nil {
		if req.Options.Temperature > 0 {
			anthropicReq["temperature"] = req.Options.Temperature
		}
		if req.Options.TopP > 0 {
			anthropicReq["top_p"] = req.Options.TopP
		}
		if req.Options.MaxTokens > 0 {
			anthropicReq["max_tokens"] = req.Options.MaxTokens
		}
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("x-api-key", b.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		b.UpdateMetrics(int32(time.Since(start).Milliseconds()), false)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.UpdateMetrics(int32(time.Since(start).Milliseconds()), false)
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		b.UpdateMetrics(int32(time.Since(start).Milliseconds()), false)
		return nil, err
	}

	elapsed := time.Since(start)
	latencyMs := int32(elapsed.Milliseconds())

	b.UpdateMetrics(latencyMs, true)

	response := ""
	if len(anthropicResp.Content) > 0 {
		response = anthropicResp.Content[0].Text
	}

	return &backends.GenerateResponse{
		Response: response,
		Stats: &backends.GenerationStats{
			TotalTimeMs:     latencyMs,
			TokensGenerated: int32(anthropicResp.Usage.OutputTokens),
			TokensPerSecond: float32(anthropicResp.Usage.OutputTokens) / float32(elapsed.Seconds()),
			EnergyWh:        0, // Cloud service
		},
	}, nil
}

// GenerateStream performs streaming text generation (placeholder)
func (b *AnthropicBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	// TODO: Implement streaming support
	return nil, fmt.Errorf("streaming not yet implemented for Anthropic backend")
}

// Embed generates embeddings (not supported by Anthropic)
func (b *AnthropicBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by Anthropic API")
}

// UpdateMetrics updates backend metrics
func (b *AnthropicBackend) UpdateMetrics(latencyMs int32, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	atomic.AddInt64(&b.metrics.RequestCount, 1)

	if success {
		atomic.AddInt64(&b.metrics.SuccessCount, 1)
		atomic.AddInt64(&b.metrics.TotalLatencyMs, int64(latencyMs))

		if b.metrics.RequestCount > 0 {
			b.metrics.AvgLatencyMs = int32(b.metrics.TotalLatencyMs / b.metrics.RequestCount)
		}
	} else {
		atomic.AddInt64(&b.metrics.ErrorCount, 1)
	}

	if b.metrics.RequestCount > 0 {
		b.metrics.ErrorRate = float32(b.metrics.ErrorCount) / float32(b.metrics.RequestCount)
	}
}

// GetMetrics returns current metrics
func (b *AnthropicBackend) GetMetrics() *backends.BackendMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return &backends.BackendMetrics{
		RequestCount:   b.metrics.RequestCount,
		SuccessCount:   b.metrics.SuccessCount,
		ErrorCount:     b.metrics.ErrorCount,
		TotalLatencyMs: b.metrics.TotalLatencyMs,
		AvgLatencyMs:   b.metrics.AvgLatencyMs,
		ErrorRate:      b.metrics.ErrorRate,
		LoadedModels:   b.metrics.LoadedModels,
	}
}

// Start initializes the backend
func (b *AnthropicBackend) Start(ctx context.Context) error {
	return b.HealthCheck(ctx)
}

// Stop shuts down the backend
func (b *AnthropicBackend) Stop(ctx context.Context) error {
	return nil
}

// matchesPattern checks if model name matches a pattern
func matchesPattern(modelName, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if modelName == pattern {
		return true
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := strings.Trim(pattern, "*")
		return strings.Contains(modelName, substr)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(modelName, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(modelName, suffix)
	}

	return false
}
