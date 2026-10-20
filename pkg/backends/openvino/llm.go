// Package openvino implements an OpenVINO local-inference backend.
package openvino

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// LLMBackend implements LLM generation using OpenVINO GenAI
// Optimized for Intel hardware (CPU/GPU/NPU) with INT4 quantization
type LLMBackend struct {
	lastCheck       time.Time
	modelCapability *backends.ModelCapability
	logger          *zap.Logger
	metrics         *backends.BackendMetrics
	device          string
	modelPath       string
	modelName       string
	hardware        string
	name            string
	id              string
	powerWatts      float64
	priority        int
	checkTimeout    time.Duration
	mu              sync.RWMutex
	avgLatencyMs    int32
	healthy         atomic.Bool
}

// LLMConfig configures the OpenVINO LLM backend.
type LLMConfig struct {
	Device    string
	ModelPath string
	ModelName string
	backends.BackendConfig
}

// NewOpenVINOLLMBackend creates a new OpenVINO LLM backend
func NewOpenVINOLLMBackend(cfg LLMConfig, logger *zap.Logger) (*LLMBackend, error) {
	backend := &LLMBackend{
		id:              cfg.ID,
		name:            cfg.Name,
		hardware:        cfg.Hardware,
		device:          cfg.Device,
		modelPath:       cfg.ModelPath,
		modelName:       cfg.ModelName,
		powerWatts:      cfg.PowerWatts,
		avgLatencyMs:    cfg.AvgLatencyMs,
		priority:        cfg.Priority,
		modelCapability: cfg.ModelCapability,
		checkTimeout:    5 * time.Second,
		metrics: &backends.BackendMetrics{
			LoadedModels: []string{cfg.ModelName},
		},
		logger: logger,
	}

	backend.healthy.Store(false)
	return backend, nil
}

// ID returns backend identifier
func (b *LLMBackend) ID() string {
	return b.id
}

// Type returns backend type
func (b *LLMBackend) Type() string {
	return "openvino"
}

// Name returns human-readable name
func (b *LLMBackend) Name() string {
	return b.name
}

// Hardware returns hardware type
func (b *LLMBackend) Hardware() string {
	return b.hardware
}

// IsHealthy returns current health status
func (b *LLMBackend) IsHealthy() bool {
	return b.healthy.Load()
}

// HealthCheck performs health check
func (b *LLMBackend) HealthCheck(_ context.Context) error {
	// Check if model path exists
	if _, err := os.Stat(b.modelPath); os.IsNotExist(err) {
		b.healthy.Store(false)
		return fmt.Errorf("model path does not exist: %s", b.modelPath)
	}

	b.healthy.Store(true)
	b.mu.Lock()
	b.lastCheck = time.Now()
	b.mu.Unlock()

	return nil
}

// PowerWatts returns estimated power consumption
func (b *LLMBackend) PowerWatts() float64 {
	return b.powerWatts
}

// AvgLatencyMs returns average latency
func (b *LLMBackend) AvgLatencyMs() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.metrics.RequestCount > 0 {
		return b.metrics.AvgLatencyMs
	}
	return b.avgLatencyMs
}

// Priority returns backend priority
func (b *LLMBackend) Priority() int {
	return b.priority
}

// SupportsGenerate returns true
func (b *LLMBackend) SupportsGenerate() bool {
	return true
}

// SupportsStream returns true
func (b *LLMBackend) SupportsStream() bool {
	return true
}

// SupportsEmbed returns false (not implemented)
func (b *LLMBackend) SupportsEmbed() bool {
	return false
}

// ListModels returns available models
func (b *LLMBackend) ListModels(_ context.Context) ([]string, error) {
	return []string{b.modelName}, nil
}

// Generate performs text generation using OpenVINO GenAI
func (b *LLMBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	start := time.Now()

	// Build request JSON
	reqData := map[string]interface{}{
		"prompt":      req.Prompt,
		"max_tokens":  256,
		"temperature": 0.7,
		"device":      b.device,
	}

	if req.Options != nil {
		if req.Options.Temperature > 0 {
			reqData["temperature"] = req.Options.Temperature
		}
	}

	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	// Call OpenVINO GenAI via Python wrapper. The prompt stays in the JSON
	// payload on stdin so it is never interpolated into Python source.
	cmd, err := openvinoCommand(ctx, openvinoGenerateScript, reqJSON, b.modelPath)
	if err != nil {
		return nil, fmt.Errorf("openvino generate: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		b.UpdateMetrics(safeconv.Int32FromInt64(time.Since(start).Milliseconds()), false)
		return nil, fmt.Errorf("openvino generate failed: %w - %s", err, string(output))
	}

	// Parse response
	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		b.UpdateMetrics(safeconv.Int32FromInt64(time.Since(start).Milliseconds()), false)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	elapsed := time.Since(start)
	latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
	b.UpdateMetrics(latencyMs, true)

	energyWh := (b.powerWatts * elapsed.Seconds()) / 3600.0

	return &backends.GenerateResponse{
		Response: response.Response,
		Stats: &backends.GenerationStats{
			TotalTimeMs:     latencyMs,
			TokensGenerated: safeconv.Int32FromInt(len(strings.Fields(response.Response))), // Approximation
			EnergyWh:        float32(energyWh),
		},
	}, nil
}

// openvinoStreamReader implements StreamReader for OpenVINO streaming
type openvinoStreamReader struct {
	start          time.Time
	scanner        *bufio.Scanner
	cmd            *exec.Cmd
	backend        *LLMBackend
	firstTokenTime *time.Time
	tokenCount     int
	done           bool
	firstToken     bool
}

// GenerateStream performs streaming text generation
func (b *LLMBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	// Build request JSON
	reqData := map[string]interface{}{
		"prompt":      req.Prompt,
		"max_tokens":  256,
		"temperature": 0.7,
		"device":      b.device,
		"stream":      true,
	}

	if req.Options != nil {
		if req.Options.Temperature > 0 {
			reqData["temperature"] = req.Options.Temperature
		}
	}

	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	// Start streaming Python process. The prompt is passed on stdin.
	cmd, err := openvinoCommand(ctx, openvinoStreamScript, reqJSON, b.modelPath)
	if err != nil {
		return nil, fmt.Errorf("openvino stream: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	return &openvinoStreamReader{
		scanner:    scanner,
		cmd:        cmd,
		start:      time.Now(),
		backend:    b,
		firstToken: true,
	}, nil
}

// Recv receives next chunk from stream
func (r *openvinoStreamReader) Recv() (*backends.StreamChunk, error) {
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
		Token string `json:"token"`
		Done  bool   `json:"done"`
	}

	if err := json.Unmarshal(r.scanner.Bytes(), &chunk); err != nil {
		return nil, err
	}

	now := time.Now()

	// Track TTFT
	if r.firstToken && chunk.Token != "" {
		r.firstToken = false
		ttft := now.Sub(r.start)
		r.firstTokenTime = &now
		r.backend.logger.Debug("Time to first token",
			zap.String("backend", r.backend.ID()),
			zap.Int64("ttft_ms", ttft.Milliseconds()),
		)
	}

	r.tokenCount++

	var stats *backends.GenerationStats
	if chunk.Done {
		r.done = true
		elapsed := time.Since(r.start)
		latencyMs := safeconv.Int32FromInt64(elapsed.Milliseconds())
		r.backend.UpdateMetrics(latencyMs, true)

		energyWh := (r.backend.powerWatts * elapsed.Seconds()) / 3600.0
		stats = &backends.GenerationStats{
			TotalTimeMs: latencyMs,
			EnergyWh:    float32(energyWh),
		}
	}

	return &backends.StreamChunk{
		Token: chunk.Token,
		Done:  chunk.Done,
		Stats: stats,
	}, nil
}

// Close closes the stream
func (r *openvinoStreamReader) Close() error {
	if err := r.cmd.Process.Kill(); err != nil {
		logging.Warn("kill openvino process", zap.Error(err))
	}
	return r.cmd.Wait()
}

// Embed is not implemented
func (b *LLMBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by OpenVINO backend")
}

// UpdateMetrics updates backend metrics
func (b *LLMBackend) UpdateMetrics(latencyMs int32, success bool) {
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
func (b *LLMBackend) GetMetrics() *backends.BackendMetrics {
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
func (b *LLMBackend) Start(ctx context.Context) error {
	return b.HealthCheck(ctx)
}

// Stop shuts down the backend
func (b *LLMBackend) Stop(_ context.Context) error {
	return nil
}

// openvinoCommand starts python3 with a constant script. The request JSON,
// including the user prompt, is passed on stdin and the model path is an
// environment variable.
func openvinoCommand(ctx context.Context, script string, reqJSON []byte, modelPath string) (*exec.Cmd, error) {
	cmd, err := cmdexec.CommandContext(ctx, "python3", "-c", script)
	if err != nil {
		return nil, err
	}
	cmd.Stdin = bytes.NewReader(reqJSON)
	cmd.Env = append(os.Environ(), envModelPath+"="+modelPath)
	return cmd, nil
}

// SupportsModel checks if this backend can run the specified model
func (b *LLMBackend) SupportsModel(modelName string) bool {
	return modelName == b.modelName
}

// GetMaxModelSizeGB returns maximum model size
func (b *LLMBackend) GetMaxModelSizeGB() int {
	if b.modelCapability == nil {
		return 999
	}
	return b.modelCapability.MaxModelSizeGB
}

// GetSupportedModelPatterns returns patterns of supported models
func (b *LLMBackend) GetSupportedModelPatterns() []string {
	if b.modelCapability == nil {
		return []string{"*"}
	}
	return b.modelCapability.SupportedModelPatterns
}

// GetPreferredModels returns list of preferred models
func (b *LLMBackend) GetPreferredModels() []string {
	return []string{b.modelName}
}

// ============================================================
// Multimedia Capability Methods
// ============================================================

// SupportsAudioToText returns whether backend supports speech-to-text
func (b *LLMBackend) SupportsAudioToText() bool {
	return false // OpenVINO LLM backend is text-only
}

// SupportsTextToAudio returns whether backend supports text-to-speech
func (b *LLMBackend) SupportsTextToAudio() bool {
	return false // OpenVINO LLM backend is text-only
}

// SupportsImageToText returns whether backend supports image captioning/OCR
func (b *LLMBackend) SupportsImageToText() bool {
	return false
}

// SupportsTextToImage returns whether backend supports image generation
func (b *LLMBackend) SupportsTextToImage() bool {
	return false
}

// SupportsVideoToText returns whether backend supports video transcription
func (b *LLMBackend) SupportsVideoToText() bool {
	return false
}

// SupportsTextToVideo returns whether backend supports video generation
func (b *LLMBackend) SupportsTextToVideo() bool {
	return false
}

// ============================================================
// Audio Operations - Not implemented for LLM backend
// ============================================================

// TranscribeAudio is not implemented
func (b *LLMBackend) TranscribeAudio(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("audio transcription not supported by OpenVINO LLM backend")
}

// TranscribeAudioStream is not implemented
func (b *LLMBackend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("audio transcription streaming not supported by OpenVINO LLM backend")
}

// SynthesizeSpeech is not implemented
func (b *LLMBackend) SynthesizeSpeech(_ context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("speech synthesis not supported by OpenVINO LLM backend")
}

// SynthesizeSpeechStream is not implemented
func (b *LLMBackend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("speech synthesis streaming not supported by OpenVINO LLM backend")
}

// ============================================================
// Image Operations - Not implemented for LLM backend
// ============================================================

// AnalyzeImage is not implemented
func (b *LLMBackend) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("image analysis not supported by OpenVINO LLM backend")
}

// GenerateImage is not implemented
func (b *LLMBackend) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("image generation not supported by OpenVINO LLM backend")
}

// GenerateImageStream is not implemented
func (b *LLMBackend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("image generation streaming not supported by OpenVINO LLM backend")
}

// ============================================================
// Video Operations - Not implemented for LLM backend
// ============================================================

// AnalyzeVideo is not implemented
func (b *LLMBackend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("video analysis not supported by OpenVINO LLM backend")
}

// AnalyzeVideoStream is not implemented
func (b *LLMBackend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("video analysis streaming not supported by OpenVINO LLM backend")
}

// GenerateVideo is not implemented
func (b *LLMBackend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("video generation not supported by OpenVINO LLM backend")
}

// GenerateVideoStream is not implemented
func (b *LLMBackend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("video generation streaming not supported by OpenVINO LLM backend")
}
