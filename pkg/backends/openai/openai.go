package openai

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

// OpenAIBackend implements Backend interface for OpenAI API
type OpenAIBackend struct {
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

// Config for OpenAI backend
type Config struct {
	backends.BackendConfig
	APIKey      string // Direct API key
	APIKeyEnv   string // Or env var name
	Endpoint    string // Optional: custom endpoint (default: api.openai.com)
	OrgID       string // Optional: OpenAI organization ID
}

// NewOpenAIBackend creates a new OpenAI API backend
func NewOpenAIBackend(cfg Config) (*OpenAIBackend, error) {
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
		endpoint = "https://api.openai.com/v1"
	}

	backend := &OpenAIBackend{
		id:              cfg.ID,
		name:            cfg.Name,
		apiKey:          apiKey,
		endpoint:        endpoint,
		powerWatts:      cfg.PowerWatts,      // 0 for cloud
		avgLatencyMs:    cfg.AvgLatencyMs,
		priority:        cfg.Priority,
		modelCapability: cfg.ModelCapability,
		checkTimeout:    5 * time.Second,
		metrics: &backends.BackendMetrics{
			LoadedModels: []string{},
		},
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	backend.healthy.Store(false)
	return backend, nil
}

// ID returns backend identifier
func (b *OpenAIBackend) ID() string {
	return b.id
}

// Type returns backend type
func (b *OpenAIBackend) Type() string {
	return "openai"
}

// Name returns human-readable name
func (b *OpenAIBackend) Name() string {
	return b.name
}

// Hardware returns hardware type (cloud for API services)
func (b *OpenAIBackend) Hardware() string {
	return "cloud"
}

// IsHealthy returns current health status
func (b *OpenAIBackend) IsHealthy() bool {
	return b.healthy.Load()
}

// HealthCheck performs health check against OpenAI API
func (b *OpenAIBackend) HealthCheck(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, b.checkTimeout)
	defer cancel()

	// Try to list models as health check
	req, err := http.NewRequestWithContext(checkCtx, "GET", b.endpoint+"/models", nil)
	if err != nil {
		b.healthy.Store(false)
		return fmt.Errorf("health check failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		b.healthy.Store(false)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.healthy.Store(false)
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	b.healthy.Store(true)
	b.mu.Lock()
	b.lastCheck = time.Now()
	b.mu.Unlock()

	return nil
}

// PowerWatts returns estimated power consumption (0 for cloud)
func (b *OpenAIBackend) PowerWatts() float64 {
	return b.powerWatts // 0 for cloud services
}

// AvgLatencyMs returns average latency
func (b *OpenAIBackend) AvgLatencyMs() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.metrics.RequestCount > 0 {
		return b.metrics.AvgLatencyMs
	}
	return b.avgLatencyMs
}

// Priority returns backend priority
func (b *OpenAIBackend) Priority() int {
	return b.priority
}

// SupportsGenerate returns true
func (b *OpenAIBackend) SupportsGenerate() bool {
	return true
}

// SupportsStream returns true
func (b *OpenAIBackend) SupportsStream() bool {
	return true
}

// SupportsEmbed returns true
func (b *OpenAIBackend) SupportsEmbed() bool {
	return true
}

// ListModels fetches available models from OpenAI
func (b *OpenAIBackend) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.endpoint+"/models", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}

	return models, nil
}

// SupportsModel checks if this backend can run the specified model
func (b *OpenAIBackend) SupportsModel(modelName string) bool {
	if b.modelCapability == nil {
		// Default OpenAI models
		return strings.HasPrefix(modelName, "gpt-") || strings.HasPrefix(modelName, "o1-")
	}

	// Check excluded patterns
	for _, pattern := range b.modelCapability.ExcludedPatterns {
		if matchesPattern(modelName, pattern) {
			return false
		}
	}

	// Check supported patterns
	if len(b.modelCapability.SupportedModelPatterns) == 0 {
		return true // No restrictions
	}

	for _, pattern := range b.modelCapability.SupportedModelPatterns {
		if matchesPattern(modelName, pattern) {
			return true
		}
	}

	return false
}

// GetMaxModelSizeGB returns maximum model size (N/A for cloud)
func (b *OpenAIBackend) GetMaxModelSizeGB() int {
	return 999 // Cloud services have no size limit
}

// GetSupportedModelPatterns returns patterns of supported models
func (b *OpenAIBackend) GetSupportedModelPatterns() []string {
	if b.modelCapability == nil {
		return []string{"gpt-*", "o1-*"}
	}
	return b.modelCapability.SupportedModelPatterns
}

// GetPreferredModels returns list of preferred models
func (b *OpenAIBackend) GetPreferredModels() []string {
	if b.modelCapability == nil {
		return []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"}
	}
	return b.modelCapability.PreferredModels
}

// Generate performs text generation via OpenAI API
func (b *OpenAIBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	start := time.Now()

	// Build OpenAI API request
	openaiReq := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
	}

	if req.Options != nil {
		if req.Options.Temperature > 0 {
			openaiReq["temperature"] = req.Options.Temperature
		}
		if req.Options.TopP > 0 {
			openaiReq["top_p"] = req.Options.TopP
		}
		if req.Options.MaxTokens > 0 {
			openaiReq["max_tokens"] = req.Options.MaxTokens
		}
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
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
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		b.UpdateMetrics(int32(time.Since(start).Milliseconds()), false)
		return nil, err
	}

	elapsed := time.Since(start)
	latencyMs := int32(elapsed.Milliseconds())

	b.UpdateMetrics(latencyMs, true)

	// Cloud services don't consume local power
	energyWh := float32(0)

	response := ""
	if len(openaiResp.Choices) > 0 {
		response = openaiResp.Choices[0].Message.Content
	}

	return &backends.GenerateResponse{
		Response: response,
		Stats: &backends.GenerationStats{
			TotalTimeMs:     latencyMs,
			TokensGenerated: int32(openaiResp.Usage.TotalTokens),
			TokensPerSecond: float32(openaiResp.Usage.TotalTokens) / float32(elapsed.Seconds()),
			EnergyWh:        energyWh,
		},
	}, nil
}

// GenerateStream performs streaming text generation (placeholder)
func (b *OpenAIBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	// TODO: Implement streaming support
	return nil, fmt.Errorf("streaming not yet implemented for OpenAI backend")
}

// Embed generates embeddings via OpenAI API
func (b *OpenAIBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	start := time.Now()

	openaiReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Text,
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var openaiResp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, err
	}

	latencyMs := int32(time.Since(start).Milliseconds())
	b.UpdateMetrics(latencyMs, true)

	embedding := []float32{}
	if len(openaiResp.Data) > 0 {
		embedding = openaiResp.Data[0].Embedding
	}

	return &backends.EmbedResponse{
		Embedding: embedding,
		Stats: &backends.GenerationStats{
			TotalTimeMs: latencyMs,
		},
	}, nil
}

// UpdateMetrics updates backend metrics
func (b *OpenAIBackend) UpdateMetrics(latencyMs int32, success bool) {
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
func (b *OpenAIBackend) GetMetrics() *backends.BackendMetrics {
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
func (b *OpenAIBackend) Start(ctx context.Context) error {
	return b.HealthCheck(ctx)
}

// Stop shuts down the backend
func (b *OpenAIBackend) Stop(ctx context.Context) error {
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

	// Simple wildcard matching
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
