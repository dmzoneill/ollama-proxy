// Package openai implements an OpenAI-compatible backend.
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
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/safeconv"
	"go.uber.org/zap"
)

// Backend implements Backend interface for OpenAI API
type Backend struct {
	lastCheck       time.Time
	metrics         *backends.BackendMetrics
	modelCapability *backends.ModelCapability
	client          *http.Client
	id              string
	name            string
	apiKey          string
	endpoint        string
	powerWatts      float64
	priority        int
	checkTimeout    time.Duration
	mu              sync.RWMutex
	avgLatencyMs    int32
	healthy         atomic.Bool
}

// Config for OpenAI backend
type Config struct {
	APIKey    string
	APIKeyEnv string
	Endpoint  string
	OrgID     string
	backends.BackendConfig
}

// NewOpenAIBackend creates a new OpenAI API backend
func NewOpenAIBackend(cfg Config) (*Backend, error) {
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

	backend := &Backend{
		id:              cfg.ID,
		name:            cfg.Name,
		apiKey:          apiKey,
		endpoint:        endpoint,
		powerWatts:      cfg.PowerWatts, // 0 for cloud
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
func (b *Backend) ID() string {
	return b.id
}

// Type returns backend type
func (b *Backend) Type() string {
	return strOpenai
}

// Name returns human-readable name
func (b *Backend) Name() string {
	return b.name
}

// Hardware returns hardware type (cloud for API services)
func (b *Backend) Hardware() string {
	return "cloud"
}

// IsHealthy returns current health status
func (b *Backend) IsHealthy() bool {
	return b.healthy.Load()
}

// HealthCheck performs health check against OpenAI API
func (b *Backend) HealthCheck(ctx context.Context) error {
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
	}()

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
func (b *Backend) PowerWatts() float64 {
	return b.powerWatts // 0 for cloud services
}

// AvgLatencyMs returns average latency
func (b *Backend) AvgLatencyMs() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.metrics.RequestCount > 0 {
		return b.metrics.AvgLatencyMs
	}
	return b.avgLatencyMs
}

// Priority returns backend priority
func (b *Backend) Priority() int {
	return b.priority
}

// SupportsGenerate returns true
func (b *Backend) SupportsGenerate() bool {
	return true
}

// SupportsStream returns true
func (b *Backend) SupportsStream() bool {
	return true
}

// SupportsEmbed returns true
func (b *Backend) SupportsEmbed() bool {
	return true
}

// ListModels fetches available models from OpenAI
func (b *Backend) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.endpoint+"/models", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
	}()

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
func (b *Backend) SupportsModel(modelName string) bool {
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
func (b *Backend) GetMaxModelSizeGB() int {
	return 999 // Cloud services have no size limit
}

// GetSupportedModelPatterns returns patterns of supported models
func (b *Backend) GetSupportedModelPatterns() []string {
	if b.modelCapability == nil {
		return []string{strGpt, strO1}
	}
	return b.modelCapability.SupportedModelPatterns
}

// GetPreferredModels returns list of preferred models
func (b *Backend) GetPreferredModels() []string {
	if b.modelCapability == nil {
		return []string{strGpt42, strGpt4Turbo, strGpt35Turbo}
	}
	return b.modelCapability.PreferredModels
}

// Generate performs text generation via OpenAI API
func (b *Backend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	start := time.Now()

	// Build OpenAI API request
	openaiReq := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", strContent: req.Prompt},
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
		b.UpdateMetrics(safeconv.Int32FromInt64(time.Since(start).Milliseconds()), false)
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		b.UpdateMetrics(safeconv.Int32FromInt64(time.Since(start).Milliseconds()), false)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}
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
		b.UpdateMetrics(safeconv.Int32FromInt64(time.Since(start).Milliseconds()), false)
		return nil, err
	}

	elapsed := time.Since(start)
	latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())

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
			TokensGenerated: safeconv.Int32FromInt(openaiResp.Usage.TotalTokens),
			TokensPerSecond: float32(openaiResp.Usage.TotalTokens) / float32(elapsed.Seconds()),
			EnergyWh:        energyWh,
		},
	}, nil
}

// GenerateStream performs streaming text generation (placeholder)
func (b *Backend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	// TODO: Implement streaming support
	return nil, fmt.Errorf("streaming not yet implemented for OpenAI backend")
}

// Embed generates embeddings via OpenAI API
func (b *Backend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
	}()

	var openaiResp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, err
	}

	latencyMs := safeconv.Int32FromInt64(time.Since(start).Milliseconds())
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
func (b *Backend) UpdateMetrics(latencyMs int32, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	atomic.AddInt64(&b.metrics.RequestCount, 1)

	if success {
		atomic.AddInt64(&b.metrics.SuccessCount, 1)
		atomic.AddInt64(&b.metrics.TotalLatencyMs, int64(latencyMs))

		if b.metrics.RequestCount > 0 {
			b.metrics.AvgLatencyMs = safeconv.Int32FromInt64(b.metrics.TotalLatencyMs / b.metrics.RequestCount)
		}
	} else {
		atomic.AddInt64(&b.metrics.ErrorCount, 1)
	}

	if b.metrics.RequestCount > 0 {
		b.metrics.ErrorRate = float32(b.metrics.ErrorCount) / float32(b.metrics.RequestCount)
	}
}

// GetMetrics returns current metrics
func (b *Backend) GetMetrics() *backends.BackendMetrics {
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
func (b *Backend) Start(ctx context.Context) error {
	return b.HealthCheck(ctx)
}

// Stop shuts down the backend
func (b *Backend) Stop(_ context.Context) error {
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
