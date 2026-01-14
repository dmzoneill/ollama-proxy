package openvino

import (
	"bufio"
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
	"go.uber.org/zap"
)

// OpenVINOLLMBackend implements LLM generation using OpenVINO GenAI
// Optimized for Intel hardware (CPU/GPU/NPU) with INT4 quantization
type OpenVINOLLMBackend struct {
	mu sync.RWMutex

	// Config
	id       string
	name     string
	hardware string
	device   string // "CPU", "GPU", "NPU"

	// Model
	modelPath string
	modelName string

	// Characteristics
	powerWatts   float64
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

	logger *zap.Logger
}

// Config for OpenVINO LLM backend
type LLMConfig struct {
	backends.BackendConfig
	Device    string
	ModelPath string
	ModelName string
}

// NewOpenVINOLLMBackend creates a new OpenVINO LLM backend
func NewOpenVINOLLMBackend(cfg LLMConfig, logger *zap.Logger) (*OpenVINOLLMBackend, error) {
	backend := &OpenVINOLLMBackend{
		id:           cfg.ID,
		name:         cfg.Name,
		hardware:     cfg.Hardware,
		device:       cfg.Device,
		modelPath:    cfg.ModelPath,
		modelName:    cfg.ModelName,
		powerWatts:   cfg.PowerWatts,
		avgLatencyMs: cfg.AvgLatencyMs,
		priority:     cfg.Priority,
		modelCapability: cfg.ModelCapability,
		checkTimeout: 5 * time.Second,
		metrics: &backends.BackendMetrics{
			LoadedModels: []string{cfg.ModelName},
		},
		logger: logger,
	}

	backend.healthy.Store(false)
	return backend, nil
}

// ID returns backend identifier
func (b *OpenVINOLLMBackend) ID() string {
	return b.id
}

// Type returns backend type
func (b *OpenVINOLLMBackend) Type() string {
	return "openvino"
}

// Name returns human-readable name
func (b *OpenVINOLLMBackend) Name() string {
	return b.name
}

// Hardware returns hardware type
func (b *OpenVINOLLMBackend) Hardware() string {
	return b.hardware
}

// IsHealthy returns current health status
func (b *OpenVINOLLMBackend) IsHealthy() bool {
	return b.healthy.Load()
}

// HealthCheck performs health check
func (b *OpenVINOLLMBackend) HealthCheck(ctx context.Context) error {
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
func (b *OpenVINOLLMBackend) PowerWatts() float64 {
	return b.powerWatts
}

// AvgLatencyMs returns average latency
func (b *OpenVINOLLMBackend) AvgLatencyMs() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.metrics.RequestCount > 0 {
		return b.metrics.AvgLatencyMs
	}
	return b.avgLatencyMs
}

// Priority returns backend priority
func (b *OpenVINOLLMBackend) Priority() int {
	return b.priority
}

// SupportsGenerate returns true
func (b *OpenVINOLLMBackend) SupportsGenerate() bool {
	return true
}

// SupportsStream returns true
func (b *OpenVINOLLMBackend) SupportsStream() bool {
	return true
}

// SupportsEmbed returns false (not implemented)
func (b *OpenVINOLLMBackend) SupportsEmbed() bool {
	return false
}

// ListModels returns available models
func (b *OpenVINOLLMBackend) ListModels(ctx context.Context) ([]string, error) {
	return []string{b.modelName}, nil
}

// Generate performs text generation using OpenVINO GenAI
func (b *OpenVINOLLMBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
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

	// Call OpenVINO GenAI via Python wrapper
	cmd := exec.CommandContext(ctx, "python3", "-c", fmt.Sprintf(`
import sys
import json
import openvino_genai as ov_genai

request = json.loads(%q)
pipe = ov_genai.LLMPipeline(%q, request["device"])

config = ov_genai.GenerationConfig()
config.max_new_tokens = request["max_tokens"]
config.temperature = request["temperature"]

result = pipe.generate(request["prompt"], config)
print(json.dumps({"response": result}))
`, string(reqJSON), b.modelPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		b.UpdateMetrics(int32(time.Since(start).Milliseconds()), false)
		return nil, fmt.Errorf("openvino generate failed: %w - %s", err, string(output))
	}

	// Parse response
	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		b.UpdateMetrics(int32(time.Since(start).Milliseconds()), false)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	elapsed := time.Since(start)
	latencyMs := int32(elapsed.Milliseconds())
	b.UpdateMetrics(latencyMs, true)

	energyWh := (b.powerWatts * elapsed.Seconds()) / 3600.0

	return &backends.GenerateResponse{
		Response: response.Response,
		Stats: &backends.GenerationStats{
			TotalTimeMs: latencyMs,
			TokensGenerated: int32(len(strings.Fields(response.Response))), // Approximation
			EnergyWh:    float32(energyWh),
		},
	}, nil
}

// openvinoStreamReader implements StreamReader for OpenVINO streaming
type openvinoStreamReader struct {
	scanner *bufio.Scanner
	cmd     *exec.Cmd
	start   time.Time
	backend *OpenVINOLLMBackend
	done    bool

	firstToken     bool
	firstTokenTime *time.Time
	tokenCount     int
}

// GenerateStream performs streaming text generation
func (b *OpenVINOLLMBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
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

	// Start streaming Python process
	cmd := exec.CommandContext(ctx, "python3", "-c", fmt.Sprintf(`
import sys
import json
import openvino_genai as ov_genai

request = json.loads(%q)
pipe = ov_genai.LLMPipeline(%q, request["device"])

config = ov_genai.GenerationConfig()
config.max_new_tokens = request["max_tokens"]
config.temperature = request["temperature"]

class StreamPrinter(ov_genai.StreamerBase):
    def put(self, token_id):
        token = pipe.get_tokenizer().decode([token_id])
        print(json.dumps({"token": token, "done": False}), flush=True)
        return False

    def end(self):
        print(json.dumps({"token": "", "done": True}), flush=True)

streamer = StreamPrinter()
pipe.generate(request["prompt"], config, streamer)
`, string(reqJSON), b.modelPath))

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
		latencyMs := int32(elapsed.Milliseconds())
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
	r.cmd.Process.Kill()
	return r.cmd.Wait()
}

// Embed is not implemented
func (b *OpenVINOLLMBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by OpenVINO backend")
}

// UpdateMetrics updates backend metrics
func (b *OpenVINOLLMBackend) UpdateMetrics(latencyMs int32, success bool) {
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
func (b *OpenVINOLLMBackend) GetMetrics() *backends.BackendMetrics {
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
func (b *OpenVINOLLMBackend) Start(ctx context.Context) error {
	return b.HealthCheck(ctx)
}

// Stop shuts down the backend
func (b *OpenVINOLLMBackend) Stop(ctx context.Context) error {
	return nil
}

// SupportsModel checks if this backend can run the specified model
func (b *OpenVINOLLMBackend) SupportsModel(modelName string) bool {
	return modelName == b.modelName
}

// GetMaxModelSizeGB returns maximum model size
func (b *OpenVINOLLMBackend) GetMaxModelSizeGB() int {
	if b.modelCapability == nil {
		return 999
	}
	return b.modelCapability.MaxModelSizeGB
}

// GetSupportedModelPatterns returns patterns of supported models
func (b *OpenVINOLLMBackend) GetSupportedModelPatterns() []string {
	if b.modelCapability == nil {
		return []string{"*"}
	}
	return b.modelCapability.SupportedModelPatterns
}

// GetPreferredModels returns list of preferred models
func (b *OpenVINOLLMBackend) GetPreferredModels() []string {
	return []string{b.modelName}
}

// ============================================================
// Multimedia Capability Methods
// ============================================================

// SupportsAudioToText returns whether backend supports speech-to-text
func (b *OpenVINOLLMBackend) SupportsAudioToText() bool {
	return false // OpenVINO LLM backend is text-only
}

// SupportsTextToAudio returns whether backend supports text-to-speech
func (b *OpenVINOLLMBackend) SupportsTextToAudio() bool {
	return false // OpenVINO LLM backend is text-only
}

// SupportsImageToText returns whether backend supports image captioning/OCR
func (b *OpenVINOLLMBackend) SupportsImageToText() bool {
	return false
}

// SupportsTextToImage returns whether backend supports image generation
func (b *OpenVINOLLMBackend) SupportsTextToImage() bool {
	return false
}

// SupportsVideoToText returns whether backend supports video transcription
func (b *OpenVINOLLMBackend) SupportsVideoToText() bool {
	return false
}

// SupportsTextToVideo returns whether backend supports video generation
func (b *OpenVINOLLMBackend) SupportsTextToVideo() bool {
	return false
}

// ============================================================
// Audio Operations - Not implemented for LLM backend
// ============================================================

// TranscribeAudio is not implemented
func (b *OpenVINOLLMBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("audio transcription not supported by OpenVINO LLM backend")
}

// TranscribeAudioStream is not implemented
func (b *OpenVINOLLMBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("audio transcription streaming not supported by OpenVINO LLM backend")
}

// SynthesizeSpeech is not implemented
func (b *OpenVINOLLMBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("speech synthesis not supported by OpenVINO LLM backend")
}

// SynthesizeSpeechStream is not implemented
func (b *OpenVINOLLMBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("speech synthesis streaming not supported by OpenVINO LLM backend")
}

// ============================================================
// Image Operations - Not implemented for LLM backend
// ============================================================

// AnalyzeImage is not implemented
func (b *OpenVINOLLMBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("image analysis not supported by OpenVINO LLM backend")
}

// GenerateImage is not implemented
func (b *OpenVINOLLMBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("image generation not supported by OpenVINO LLM backend")
}

// GenerateImageStream is not implemented
func (b *OpenVINOLLMBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("image generation streaming not supported by OpenVINO LLM backend")
}

// ============================================================
// Video Operations - Not implemented for LLM backend
// ============================================================

// AnalyzeVideo is not implemented
func (b *OpenVINOLLMBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("video analysis not supported by OpenVINO LLM backend")
}

// AnalyzeVideoStream is not implemented
func (b *OpenVINOLLMBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("video analysis streaming not supported by OpenVINO LLM backend")
}

// GenerateVideo is not implemented
func (b *OpenVINOLLMBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("video generation not supported by OpenVINO LLM backend")
}

// GenerateVideoStream is not implemented
func (b *OpenVINOLLMBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("video generation streaming not supported by OpenVINO LLM backend")
}
