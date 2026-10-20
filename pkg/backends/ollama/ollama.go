// Package ollama implements an Ollama backend.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/cmdexec"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/safeconv"
	"go.uber.org/zap"
)

// Backend implements Backend interface for Ollama instances
type Backend struct {
	lastCheck       time.Time
	metrics         *backends.BackendMetrics
	modelCapability *backends.ModelCapability
	client          *http.Client
	id              string
	name            string
	hardware        string
	endpoint        string
	powerWatts      float64
	priority        int
	checkTimeout    time.Duration
	mu              sync.RWMutex
	avgLatencyMs    int32
	healthy         atomic.Bool
}

// Config for Ollama backend
type Config struct {
	Endpoint string
	backends.BackendConfig
}

// NewOllamaBackend creates a new Ollama backend instance
func NewOllamaBackend(cfg Config) (*Backend, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	backend := &Backend{
		id:              cfg.ID,
		name:            cfg.Name,
		hardware:        cfg.Hardware,
		endpoint:        cfg.Endpoint,
		powerWatts:      cfg.PowerWatts,
		avgLatencyMs:    cfg.AvgLatencyMs,
		priority:        cfg.Priority,
		modelCapability: cfg.ModelCapability,
		checkTimeout:    5 * time.Second,
		metrics: &backends.BackendMetrics{
			LoadedModels: []string{},
		},
		client: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for voice/streaming workloads
			Transport: &http.Transport{
				// Connection pooling
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,

				// Performance tuning
				DisableCompression: true,  // Reduce CPU on streaming
				DisableKeepAlives:  false, // Enable keep-alive

				// Timeouts
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second, // Increased for voice LLM processing
				ExpectContinueTimeout: 1 * time.Second,

				// HTTP/2 support
				ForceAttemptHTTP2: true,
			},
		},
	}

	backend.healthy.Store(false) // Will be set by health check
	return backend, nil
}

// ID returns backend identifier
func (b *Backend) ID() string {
	return b.id
}

// Type returns backend type
func (b *Backend) Type() string {
	return "ollama"
}

// Name returns human-readable name
func (b *Backend) Name() string {
	return b.name
}

// Hardware returns hardware type
func (b *Backend) Hardware() string {
	return b.hardware
}

// IsHealthy returns current health status
func (b *Backend) IsHealthy() bool {
	return b.healthy.Load()
}

// HealthCheck performs health check against Ollama instance
func (b *Backend) HealthCheck(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, b.checkTimeout)
	defer cancel()

	// Try to list models as health check
	req, err := http.NewRequestWithContext(checkCtx, "GET", b.endpoint+"/api/tags", nil)
	if err != nil {
		b.healthy.Store(false)
		return fmt.Errorf("health check failed: %w", err)
	}

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

// PowerWatts returns estimated power consumption
func (b *Backend) PowerWatts() float64 {
	return b.powerWatts
}

// AvgLatencyMs returns average latency
func (b *Backend) AvgLatencyMs() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.metrics.RequestCount > 0 {
		return b.metrics.AvgLatencyMs
	}
	return b.avgLatencyMs // Return configured estimate
}

// Priority returns backend priority
func (b *Backend) Priority() int {
	return b.priority
}

// SupportsGenerate returns true (Ollama supports generation)
func (b *Backend) SupportsGenerate() bool {
	return true
}

// SupportsStream returns true (Ollama supports streaming)
func (b *Backend) SupportsStream() bool {
	return true
}

// SupportsEmbed returns true (Ollama supports embeddings)
func (b *Backend) SupportsEmbed() bool {
	return true
}

// ListModels fetches available models from Ollama
func (b *Backend) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

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
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// Generate performs text generation
func (b *Backend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	start := time.Now()

	// Build Ollama request
	ollamaReq := map[string]interface{}{
		strModel:  req.Model,
		strPrompt: req.Prompt,
		strStream: false,
	}

	// Build options
	options := make(map[string]interface{})

	// Voice assistant optimizations: small context, limited output
	options["num_ctx"] = 512     // Small context for voice (vs default 8192)
	options["num_predict"] = 256 // Limit response length
	options["temperature"] = 0.7 // Balanced creativity

	if req.Options != nil {
		if req.Options.Temperature > 0 {
			options["temperature"] = req.Options.Temperature
		}
		if req.Options.TopP > 0 {
			options["top_p"] = req.Options.TopP
		}
		if req.Options.TopK > 0 {
			options["top_k"] = req.Options.TopK
		}
	}

	ollamaReq["options"] = options

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("ollama error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp struct {
		Response string `json:"response"`
		Context  []int  `json:"context"`
		Done     bool   `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		b.UpdateMetrics(safeconv.Int32FromInt64(time.Since(start).Milliseconds()), false)
		return nil, err
	}

	elapsed := time.Since(start)
	latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())

	b.UpdateMetrics(latencyMs, true)

	// Calculate energy consumption
	energyWh := (b.powerWatts * elapsed.Seconds()) / 3600.0

	return &backends.GenerateResponse{
		Response: ollamaResp.Response,
		Stats: &backends.GenerationStats{
			TotalTimeMs:     latencyMs,
			TokensGenerated: safeconv.Int32FromInt(len(ollamaResp.Context)), // Approximation
			TokensPerSecond: float32(len(ollamaResp.Context)) / float32(elapsed.Seconds()),
			EnergyWh:        float32(energyWh),
		},
	}, nil
}

// ollamaStreamReader implements StreamReader for Ollama streaming
type ollamaStreamReader struct {
	start          time.Time
	lastTokenTime  time.Time
	scanner        *bufio.Scanner
	resp           *http.Response
	backend        *Backend
	firstTokenTime *time.Time
	tokenCount     int
	firstToken     bool
}

// GenerateStream performs streaming text generation
func (b *Backend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	// Build Ollama request
	ollamaReq := map[string]interface{}{
		strModel:  req.Model,
		strPrompt: req.Prompt,
		strStream: true,
	}

	// Build options
	options := make(map[string]interface{})

	// Voice assistant optimizations: small context, limited output
	options["num_ctx"] = 512     // Small context for voice (vs default 8192)
	options["num_predict"] = 256 // Limit response length
	options["temperature"] = 0.7 // Balanced creativity

	if req.Options != nil {
		if req.Options.Temperature > 0 {
			options["temperature"] = req.Options.Temperature
		}
		if req.Options.TopP > 0 {
			options["top_p"] = req.Options.TopP
		}
		if req.Options.TopK > 0 {
			options["top_k"] = req.Options.TopK
		}
	}

	ollamaReq["options"] = options

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
		return nil, fmt.Errorf("ollama error: status %d", resp.StatusCode)
	}

	// Create scanner with smaller buffer for lower latency
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 4096) // 4KB buffer instead of default 64KB
	scanner.Buffer(buf, 4096)

	return &ollamaStreamReader{
		scanner:       scanner,
		resp:          resp,
		start:         time.Now(),
		backend:       b,
		firstToken:    true,
		lastTokenTime: time.Now(),
	}, nil
}

// Recv receives next chunk from stream
func (r *ollamaStreamReader) Recv() (*backends.StreamChunk, error) {
	now := time.Now()

	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	var chunk struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.Unmarshal(r.scanner.Bytes(), &chunk); err != nil {
		return nil, err
	}

	// Track TTFT (Time To First Token)
	if r.firstToken && chunk.Response != "" {
		r.firstToken = false
		ttft := now.Sub(r.start)
		r.firstTokenTime = &now

		// Log TTFT for voice quality monitoring
		if logging.Logger != nil {
			logging.Logger.Debug("Time to first token",
				zap.String("backend", r.backend.ID()),
				zap.Int64("ttft_ms", ttft.Milliseconds()),
			)
		}
	}

	// Track inter-token latency
	if r.tokenCount > 0 {
		interTokenLatency := now.Sub(r.lastTokenTime)
		// Log if latency is unusually high (>100ms indicates issue)
		if interTokenLatency.Milliseconds() > 100 {
			if logging.Logger != nil {
				logging.Logger.Warn("High inter-token latency",
					zap.String("backend", r.backend.ID()),
					zap.Int("token", r.tokenCount),
					zap.Int64("latency_ms", interTokenLatency.Milliseconds()),
				)
			}
		}
	}

	r.lastTokenTime = now
	r.tokenCount++

	var stats *backends.GenerationStats
	if chunk.Done {
		elapsed := time.Since(r.start)
		latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
		r.backend.UpdateMetrics(latencyMs, true)

		energyWh := (r.backend.powerWatts * elapsed.Seconds()) / 3600.0

		stats = &backends.GenerationStats{
			TotalTimeMs: latencyMs,
			EnergyWh:    float32(energyWh),
		}

		// Log streaming summary
		var ttft time.Duration
		if r.firstTokenTime != nil {
			ttft = r.firstTokenTime.Sub(r.start)
		}

		avgInterToken := time.Duration(0)
		if r.tokenCount > 1 {
			avgInterToken = elapsed / time.Duration(r.tokenCount-1)
		}

		if logging.Logger != nil {
			logging.Logger.Info("Streaming summary",
				zap.String("backend", r.backend.ID()),
				zap.Int64("ttft_ms", ttft.Milliseconds()),
				zap.Int64("avg_inter_token_ms", avgInterToken.Milliseconds()),
				zap.Int64("total_ms", elapsed.Milliseconds()),
				zap.Int("tokens", r.tokenCount),
				zap.Float64("tokens_per_sec", float64(r.tokenCount)/elapsed.Seconds()),
			)
		}
	}

	return &backends.StreamChunk{
		Token: chunk.Response,
		Done:  chunk.Done,
		Stats: stats,
	}, nil
}

// Close closes the stream
func (r *ollamaStreamReader) Close() error {
	return r.resp.Body.Close()
}

// Embed generates embeddings (placeholder - Ollama supports this via /api/embeddings)
func (b *Backend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	// TODO: Implement embedding endpoint
	return nil, fmt.Errorf("embeddings not yet implemented for Ollama backend")
}

// UpdateMetrics updates backend metrics
func (b *Backend) UpdateMetrics(latencyMs int32, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	atomic.AddInt64(&b.metrics.RequestCount, 1)

	if success {
		atomic.AddInt64(&b.metrics.SuccessCount, 1)
		atomic.AddInt64(&b.metrics.TotalLatencyMs, int64(latencyMs))

		// Update rolling average
		if b.metrics.RequestCount > 0 {
			b.metrics.AvgLatencyMs = safeconv.Int32FromInt64(b.metrics.TotalLatencyMs / b.metrics.RequestCount)
		}
	} else {
		atomic.AddInt64(&b.metrics.ErrorCount, 1)
	}

	// Calculate error rate
	if b.metrics.RequestCount > 0 {
		b.metrics.ErrorRate = float32(b.metrics.ErrorCount) / float32(b.metrics.RequestCount)
	}
}

// GetMetrics returns current metrics
func (b *Backend) GetMetrics() *backends.BackendMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Return copy
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
	// Perform initial health check
	return b.HealthCheck(ctx)
}

// Stop shuts down the backend
func (b *Backend) Stop(_ context.Context) error {
	// Nothing to clean up for HTTP client
	return nil
}

// SupportsModel checks if this backend can run the specified model
func (b *Backend) SupportsModel(modelName string) bool {
	if b.modelCapability == nil {
		return true // No restrictions if not configured
	}

	// Check excluded patterns first
	for _, pattern := range b.modelCapability.ExcludedPatterns {
		if matchesPattern(modelName, pattern) {
			return false
		}
	}

	// If no supported patterns specified, allow all (except excluded)
	if len(b.modelCapability.SupportedModelPatterns) == 0 {
		return true
	}

	// Check if model matches any supported pattern
	for _, pattern := range b.modelCapability.SupportedModelPatterns {
		if matchesPattern(modelName, pattern) {
			return true
		}
	}

	// Check preferred models (exact match)
	for _, preferred := range b.modelCapability.PreferredModels {
		if modelName == preferred {
			return true
		}
	}

	return false
}

// GetMaxModelSizeGB returns maximum model size this backend can handle
func (b *Backend) GetMaxModelSizeGB() int {
	if b.modelCapability == nil {
		return 999 // No limit if not configured
	}
	return b.modelCapability.MaxModelSizeGB
}

// GetSupportedModelPatterns returns patterns of supported models
func (b *Backend) GetSupportedModelPatterns() []string {
	if b.modelCapability == nil {
		return []string{"*"} // Support all if not configured
	}
	return b.modelCapability.SupportedModelPatterns
}

// GetPreferredModels returns list of preferred models for this backend
func (b *Backend) GetPreferredModels() []string {
	if b.modelCapability == nil {
		return []string{}
	}
	return b.modelCapability.PreferredModels
}

// matchesPattern checks if model name matches a pattern (simple glob-like matching)
func matchesPattern(modelName, pattern string) bool {
	// Handle wildcard patterns
	if pattern == "*" {
		return true
	}

	// Exact match
	if modelName == pattern {
		return true
	}

	// Pattern: "*:0.5b" matches "qwen2.5:0.5b", "tinyllama:0.5b"
	if strings.HasPrefix(pattern, "*:") {
		suffix := strings.TrimPrefix(pattern, "*:")
		return strings.HasSuffix(modelName, ":"+suffix)
	}

	// Pattern: "llama3:*" matches "llama3:7b", "llama3:70b"
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(modelName, prefix+":")
	}

	// Pattern: "*70b*" matches any model with "70b" in name
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := strings.Trim(pattern, "*")
		return strings.Contains(modelName, substr)
	}

	return false
}

// ============================================================
// Multimedia Capability Methods
// ============================================================

// SupportsAudioToText returns whether backend supports speech-to-text
func (b *Backend) SupportsAudioToText() bool {
	// We use whisper.cpp binary directly, so always return true
	return true
}

// SupportsTextToAudio returns whether backend supports text-to-speech
func (b *Backend) SupportsTextToAudio() bool {
	// We use piper binary directly, so always return true
	return true
}

// SupportsImageToText returns whether backend supports image captioning/OCR
func (b *Backend) SupportsImageToText() bool {
	// Check if LLaVA or similar vision models are loaded
	return false
}

// SupportsTextToImage returns whether backend supports image generation
func (b *Backend) SupportsTextToImage() bool {
	// Ollama doesn't natively support Stable Diffusion yet
	return false
}

// SupportsVideoToText returns whether backend supports video transcription
func (b *Backend) SupportsVideoToText() bool {
	return false
}

// SupportsTextToVideo returns whether backend supports video generation
func (b *Backend) SupportsTextToVideo() bool {
	return false
}

// ============================================================
// Audio Operations - To be implemented when Whisper/TTS added
// ============================================================

// TranscribeAudio performs speech-to-text using whisper.cpp via subprocess
// This uses the OpenVINO-accelerated whisper.cpp for NPU/GPU/CPU execution
func (b *Backend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	start := time.Now()

	// Read audio data
	var audioData []byte
	var err error

	if req.AudioStream != nil {
		audioData, err = io.ReadAll(req.AudioStream)
		if err != nil {
			return nil, fmt.Errorf("failed to read audio stream: %w", err)
		}
	} else {
		audioData = req.AudioData
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("no audio data provided")
	}

	// Write audio to temporary WAV file for whisper.cpp
	tmpFile := fmt.Sprintf("/tmp/whisper_input_%d.wav", time.Now().UnixNano())
	defer func() {
		if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
			logging.Warn("remove temp file", zap.Error(err))
		}
	}()

	// Convert PCM to WAV format
	if err := writeWAVFile(tmpFile, audioData, int(req.SampleRate), int(req.Channels)); err != nil {
		return nil, fmt.Errorf("failed to write WAV file: %w", err)
	}

	// Run whisper.cpp with OpenVINO support
	// Use NPU for ultra-low power (3W), CPU fallback if NPU unavailable
	args := []string{
		"-m", "/home/daoneill/.cache/whisper/ggml-base.bin", // Base model with OpenVINO encoder
		"-f", tmpFile,
		"-nt",   // No timestamps
		"-otxt", // Output text only
	}

	// Add OpenVINO device if this is an NPU/CPU backend
	switch b.hardware {
	case "npu":
		args = append(args, "-oved", "NPU") // Intel Neural Processor for 300ms target
	case "cpu":
		args = append(args, "-oved", "CPU") // CPU with OpenVINO optimizations
	}

	cmd, err := cmdexec.CommandContext(ctx, "whisper-cpp", args...)
	if err != nil {
		return nil, fmt.Errorf("whisper-cpp: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("whisper-cpp failed: %w - %s", err, string(output))
	}

	transcription := strings.TrimSpace(string(output))

	elapsed := time.Since(start)
	latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
	energyWh := (b.powerWatts * elapsed.Seconds()) / 3600.0

	segments := []backends.TranscriptSegment{
		{
			Text:       transcription,
			StartMs:    0,
			EndMs:      elapsed.Milliseconds(),
			Confidence: 0.9,
		},
	}

	if logging.Logger != nil {
		logging.Logger.Info("Audio transcription completed (whisper.cpp)",
			zap.String("backend", b.ID()),
			zap.Int("audio_bytes", len(audioData)),
			zap.Int64("latency_ms", elapsed.Milliseconds()),
			zap.Int("transcript_length", len(transcription)),
		)
	}

	return &backends.TranscribeResponse{
		Text:       transcription,
		Language:   "en",
		Confidence: 0.9,
		Segments:   segments,
		Stats: &backends.GenerationStats{
			TotalTimeMs: latencyMs,
			EnergyWh:    float32(energyWh),
		},
	}, nil
}

// ollamaAudioStreamReader implements AudioStreamReader for streaming transcription
type ollamaAudioStreamReader struct {
	scanner *bufio.Scanner
	resp    *http.Response
	start   time.Time
	backend *Backend
	done    bool
}

// TranscribeAudioStream performs streaming speech-to-text
func (b *Backend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	// Read audio data
	var audioData []byte
	var err error

	if req.AudioStream != nil {
		audioData, err = io.ReadAll(req.AudioStream)
		if err != nil {
			return nil, fmt.Errorf("failed to read audio stream: %w", err)
		}
	} else {
		audioData = req.AudioData
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("no audio data provided")
	}

	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	ollamaReq := map[string]interface{}{
		strModel:  req.Model,
		strPrompt: fmt.Sprintf("Transcribe this audio: %s", audioBase64),
		strStream: true,
		"options": map[string]interface{}{
			"temperature": 0.0,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
		return nil, fmt.Errorf("ollama error: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 4096)
	scanner.Buffer(buf, 4096)

	return &ollamaAudioStreamReader{
		scanner: scanner,
		resp:    resp,
		start:   time.Now(),
		backend: b,
	}, nil
}

// Recv receives next transcription chunk from stream
func (r *ollamaAudioStreamReader) Recv() (*backends.TranscriptChunk, error) {
	if r.done {
		return nil, io.EOF
	}

	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		r.done = true
		return nil, io.EOF
	}

	var chunk struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.Unmarshal(r.scanner.Bytes(), &chunk); err != nil {
		return nil, err
	}

	elapsed := time.Since(r.start)

	if chunk.Done {
		r.done = true
		latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
		r.backend.UpdateMetrics(latencyMs, true)
	}

	return &backends.TranscriptChunk{
		Text:       chunk.Response,
		IsFinal:    chunk.Done,
		Confidence: 0.9,
		StartMs:    0,
		EndMs:      elapsed.Milliseconds(),
		Done:       chunk.Done,
	}, nil
}

// Close closes the stream
func (r *ollamaAudioStreamReader) Close() error {
	return r.resp.Body.Close()
}

// SynthesizeSpeech performs text-to-speech using Piper TTS via subprocess
func (b *Backend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	start := time.Now()

	if req.Text == "" {
		return nil, fmt.Errorf("no text provided")
	}

	// Output to temporary WAV file
	tmpFile := fmt.Sprintf("/tmp/piper_output_%d.wav", time.Now().UnixNano())
	defer func() {
		if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
			logging.Warn("remove temp file", zap.Error(err))
		}
	}()

	// Run piper binary (assumes it's installed and in PATH)
	// Voice model should be downloaded to ~/.cache/piper/
	cmd, err := cmdexec.CommandContext(ctx, "piper",
		"--model", "/home/daoneill/.cache/piper/en_US-lessac-medium.onnx",
		"--output_file", tmpFile,
	)
	if err != nil {
		return nil, fmt.Errorf("piper: %w", err)
	}

	// Pipe text to stdin
	cmd.Stdin = strings.NewReader(req.Text)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("piper failed: %w - %s", err, string(output))
	}

	// Read generated WAV file
	audioData, err := readWAVFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAV file: %w", err)
	}

	elapsed := time.Since(start)
	latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
	energyWh := (b.powerWatts * elapsed.Seconds()) / 3600.0

	// Calculate duration based on audio data size
	bytesPerSample := 2        // 16-bit
	channels := 1              // Mono
	sampleRate := int32(22050) // Piper outputs 22050 Hz
	durationSeconds := float64(len(audioData)) / float64(sampleRate*int32(channels*bytesPerSample))
	durationMs := int32(durationSeconds * 1000)

	if logging.Logger != nil {
		logging.Logger.Info("Speech synthesis completed (piper)",
			zap.String("backend", b.ID()),
			zap.Int("text_length", len(req.Text)),
			zap.Int("audio_bytes", len(audioData)),
			zap.Int64("latency_ms", elapsed.Milliseconds()),
			zap.Int32("audio_duration_ms", durationMs),
		)
	}

	return &backends.SynthesizeResponse{
		AudioData:  audioData,
		Format:     backends.AudioFormatPCM,
		SampleRate: sampleRate,
		Duration:   durationMs,
		Stats: &backends.GenerationStats{
			TotalTimeMs: latencyMs,
			EnergyWh:    float32(energyWh),
		},
	}, nil
}

// ollamaAudioStreamWriter implements AudioStreamWriter for streaming TTS
type ollamaAudioStreamWriter struct {
	scanner *bufio.Scanner
	resp    *http.Response
	start   time.Time
	backend *Backend
	format  backends.AudioFormat
	sample  int32
	done    bool
}

// SynthesizeSpeechStream performs streaming text-to-speech
func (b *Backend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("no text provided")
	}

	prompt := fmt.Sprintf("Generate speech audio for: %s", req.Text)

	ollamaReq := map[string]interface{}{
		strModel:  req.Model,
		strPrompt: prompt,
		strStream: true,
		"options": map[string]interface{}{
			"temperature": 0.7,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
		return nil, fmt.Errorf("ollama error: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 4096)
	scanner.Buffer(buf, 4096)

	return &ollamaAudioStreamWriter{
		scanner: scanner,
		resp:    resp,
		start:   time.Now(),
		backend: b,
		format:  req.Format,
		sample:  req.SampleRate,
	}, nil
}

// Recv receives next audio chunk from stream
func (w *ollamaAudioStreamWriter) Recv() (*backends.AudioChunk, error) {
	if w.done {
		return nil, io.EOF
	}

	if !w.scanner.Scan() {
		if err := w.scanner.Err(); err != nil {
			return nil, err
		}
		w.done = true
		return nil, io.EOF
	}

	var chunk struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.Unmarshal(w.scanner.Bytes(), &chunk); err != nil {
		return nil, err
	}

	// Decode audio chunk
	audioData, err := base64.StdEncoding.DecodeString(chunk.Response)
	if err != nil {
		// If not base64, treat as raw data
		audioData = []byte(chunk.Response)
	}

	// Estimate chunk duration
	bytesPerSample := 2
	channels := 1
	durationSeconds := float64(len(audioData)) / float64(w.sample*int32(channels*bytesPerSample))
	durationMs := int32(durationSeconds * 1000)

	if chunk.Done {
		w.done = true
		elapsed := time.Since(w.start)
		latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
		w.backend.UpdateMetrics(latencyMs, true)
	}

	return &backends.AudioChunk{
		Data:       audioData,
		Format:     w.format,
		SampleRate: w.sample,
		Done:       chunk.Done,
		DurationMs: durationMs,
	}, nil
}

// Close closes the stream
func (w *ollamaAudioStreamWriter) Close() error {
	return w.resp.Body.Close()
}

// ============================================================
// Image Operations - To be implemented when vision models added
// ============================================================

// AnalyzeImage performs image analysis (captioning, OCR, VQA)
func (b *Backend) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("image analysis not yet implemented for Ollama backend - requires vision model (LLaVA, BLIP2)")
}

// GenerateImage performs text-to-image generation
func (b *Backend) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("image generation not supported by Ollama - requires Stable Diffusion backend")
}

// GenerateImageStream performs streaming image generation
func (b *Backend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("streaming image generation not supported by Ollama")
}

// ============================================================
// Video Operations - To be implemented when video models added
// ============================================================

// AnalyzeVideo performs video analysis (transcription, captioning, tracking)
func (b *Backend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("video analysis not yet implemented for Ollama backend - requires video model")
}

// AnalyzeVideoStream performs streaming video analysis
func (b *Backend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("streaming video analysis not yet implemented for Ollama backend")
}

// GenerateVideo performs text-to-video generation
func (b *Backend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("video generation not supported by Ollama")
}

// GenerateVideoStream performs streaming video generation
func (b *Backend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("streaming video generation not supported by Ollama")
}

// PullModel downloads a model from Ollama registry
func (b *Backend) PullModel(ctx context.Context, modelName string) error {
	logging.Logger.Info("Pulling model from Ollama",
		zap.String("backend", b.id),
		zap.String(strModel, modelName),
	)

	pullReq := map[string]interface{}{
		"name":    modelName,
		strStream: true,
	}

	body, err := json.Marshal(pullReq)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", b.endpoint+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute pull request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logging.Warn(strCloseResponseBody, zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}
		return fmt.Errorf("pull request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read streaming progress
	scanner := bufio.NewScanner(resp.Body)
	var lastProgress string
	for scanner.Scan() {
		line := scanner.Bytes()
		var progress map[string]interface{}
		if err := json.Unmarshal(line, &progress); err != nil {
			continue
		}

		// Log progress updates
		if status, ok := progress["status"].(string); ok {
			if status != lastProgress {
				logging.Logger.Info("Model pull progress",
					zap.String("backend", b.id),
					zap.String(strModel, modelName),
					zap.String("status", status),
				)
				lastProgress = status
			}
		}

		// Check for completion
		if completed, ok := progress["completed"].(bool); ok && completed {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading pull stream: %w", err)
	}

	logging.Logger.Info("Model pull completed",
		zap.String("backend", b.id),
		zap.String(strModel, modelName),
	)

	return nil
}

// EnsureModel checks if a model exists and pulls it if not
func (b *Backend) EnsureModel(ctx context.Context, modelName string) error {
	// Check if model exists
	models, err := b.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Check for exact match or partial match (e.g., "whisper:tiny" or just "whisper")
	modelLower := strings.ToLower(modelName)
	for _, m := range models {
		mLower := strings.ToLower(m)
		if mLower == modelLower || strings.HasPrefix(mLower, modelLower+":") {
			logging.Logger.Debug("Model already available",
				zap.String("backend", b.id),
				zap.String(strModel, modelName),
				zap.String("found", m),
			)
			return nil
		}
	}

	// Model not found, pull it
	logging.Logger.Info("Model not found, pulling from registry",
		zap.String("backend", b.id),
		zap.String(strModel, modelName),
	)

	return b.PullModel(ctx, modelName)
}

// writeWAVFile writes PCM audio data to a WAV file
func writeWAVFile(filename string, pcmData []byte, sampleRate, channels int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logging.Warn("close file", zap.Error(err))
		}
	}()

	if err := writeWAVContents(f, pcmData, sampleRate, channels); err != nil {
		return fmt.Errorf("write wav file: %w", err)
	}
	return nil
}

// writeWAVContents writes a PCM WAV header and payload.
func writeWAVContents(w io.Writer, pcmData []byte, sampleRate, channels int) error {
	dataSize := safeconv.Uint32FromInt(len(pcmData))
	ww := &wavWriter{w: w}
	ww.bytes([]byte("RIFF"))
	ww.le(dataSize + 36)
	ww.bytes([]byte("WAVE"))
	ww.bytes([]byte("fmt "))
	ww.le(uint32(16)) // Chunk size
	ww.le(uint16(1))  // Audio format (PCM)
	ww.le(safeconv.Uint16FromInt(channels))
	ww.le(safeconv.Uint32FromInt(sampleRate))
	ww.le(safeconv.Uint32FromInt(sampleRate * channels * 2)) // Byte rate
	ww.le(safeconv.Uint16FromInt(channels * 2))              // Block align
	ww.le(uint16(16))                                        // Bits per sample
	ww.bytes([]byte("data"))
	ww.le(dataSize)
	ww.bytes(pcmData)
	return ww.err
}

// wavWriter records the first write error and skips later writes.
type wavWriter struct {
	w   io.Writer
	err error
}

func (ww *wavWriter) bytes(p []byte) {
	if ww.err != nil {
		return
	}
	_, ww.err = ww.w.Write(p)
}

func (ww *wavWriter) le(v any) {
	if ww.err != nil {
		return
	}
	ww.err = binary.Write(ww.w, binary.LittleEndian, v)
}

// readWAVFile reads a WAV file and returns PCM data
func readWAVFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logging.Warn("close file", zap.Error(err))
		}
	}()

	// Skip WAV header (44 bytes)
	header := make([]byte, 44)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}

	// Read PCM data
	return io.ReadAll(f)
}
