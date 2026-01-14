package pipeline

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// MockBackend implements the backends.Backend interface for testing
type MockBackend struct {
	id                string
	backendType       string
	name              string
	hardware          string
	healthy           bool
	powerWatts        float64
	avgLatencyMs      int32
	priority          int
	supportsGenerate  bool
	supportsStream    bool
	supportsEmbed     bool
	supportedModels   map[string]bool
	maxModelSizeGB    int
	supportedPatterns []string
	preferredModels   []string
	metrics           *backends.BackendMetrics
	generateErr       error
	embedErr          error
	healthErr         error
}

func NewMockBackend(id string) *MockBackend {
	return &MockBackend{
		id:               id,
		backendType:      "mock",
		name:             fmt.Sprintf("Mock Backend %s", id),
		hardware:         "cpu",
		healthy:          true,
		powerWatts:       5.0,
		avgLatencyMs:     100,
		priority:         1,
		supportsGenerate: true,
		supportsStream:   true,
		supportsEmbed:    true,
		supportedModels: map[string]bool{
			"llama3:7b":           true,
			"qwen2.5:0.5b":        true,
			"nomic-embed-text":    true,
			"test-model":          true,
		},
		maxModelSizeGB:    70,
		supportedPatterns: []string{"*:*"},
		preferredModels:   []string{"llama3:7b"},
		metrics: &backends.BackendMetrics{
			RequestCount: 0,
			SuccessCount: 0,
			ErrorCount:   0,
		},
	}
}

func (m *MockBackend) ID() string {
	return m.id
}

func (m *MockBackend) Type() string {
	return m.backendType
}

func (m *MockBackend) Name() string {
	return m.name
}

func (m *MockBackend) Hardware() string {
	return m.hardware
}

func (m *MockBackend) IsHealthy() bool {
	return m.healthy
}

func (m *MockBackend) HealthCheck(ctx context.Context) error {
	return m.healthErr
}

func (m *MockBackend) PowerWatts() float64 {
	return m.powerWatts
}

func (m *MockBackend) AvgLatencyMs() int32 {
	return m.avgLatencyMs
}

func (m *MockBackend) Priority() int {
	return m.priority
}

func (m *MockBackend) SupportsGenerate() bool {
	return m.supportsGenerate
}

func (m *MockBackend) SupportsStream() bool {
	return m.supportsStream
}

func (m *MockBackend) SupportsEmbed() bool {
	return m.supportsEmbed
}

func (m *MockBackend) ListModels(ctx context.Context) ([]string, error) {
	models := make([]string, 0, len(m.supportedModels))
	for model := range m.supportedModels {
		models = append(models, model)
	}
	return models, nil
}

func (m *MockBackend) SupportsModel(modelName string) bool {
	return m.supportedModels[modelName]
}

func (m *MockBackend) GetMaxModelSizeGB() int {
	return m.maxModelSizeGB
}

func (m *MockBackend) GetSupportedModelPatterns() []string {
	return m.supportedPatterns
}

func (m *MockBackend) GetPreferredModels() []string {
	return m.preferredModels
}

func (m *MockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &backends.GenerateResponse{
		Response: "Mock response to: " + req.Prompt,
		Stats: &backends.GenerationStats{
			TimeToFirstTokenMs: 10,
			TotalTimeMs:        100,
			TokensGenerated:    50,
			TokensPerSecond:    500,
			EnergyWh:           0.1,
		},
	}, nil
}

func (m *MockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		Stats: &backends.GenerationStats{
			TimeToFirstTokenMs: 5,
			TotalTimeMs:        50,
			TokensGenerated:    1,
			TokensPerSecond:    20,
			EnergyWh:           0.05,
		},
	}, nil
}

func (m *MockBackend) UpdateMetrics(latencyMs int32, success bool) {
	m.metrics.RequestCount++
	if success {
		m.metrics.SuccessCount++
	} else {
		m.metrics.ErrorCount++
	}
	m.metrics.TotalLatencyMs += int64(latencyMs)
	if m.metrics.RequestCount > 0 {
		m.metrics.AvgLatencyMs = int32(m.metrics.TotalLatencyMs / m.metrics.RequestCount)
	}
}

func (m *MockBackend) GetMetrics() *backends.BackendMetrics {
	return m.metrics
}

func (m *MockBackend) Start(ctx context.Context) error {
	return nil
}

func (m *MockBackend) Stop(ctx context.Context) error {
	return nil
}

// Multimedia capability methods
func (m *MockBackend) SupportsAudioToText() bool                    { return false }
func (m *MockBackend) SupportsTextToAudio() bool                    { return false }
func (m *MockBackend) SupportsImageToText() bool                    { return false }
func (m *MockBackend) SupportsTextToImage() bool                    { return false }
func (m *MockBackend) SupportsVideoToText() bool                    { return false }
func (m *MockBackend) SupportsTextToVideo() bool                    { return false }

func (m *MockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// Tests for PipelineExecutor

func TestNewPipelineExecutor(t *testing.T) {
	backend1 := NewMockBackend("backend1")
	backend2 := NewMockBackend("backend2")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	if len(executor.backendRegistry) != 2 {
		t.Errorf("Expected 2 backends in registry, got %d", len(executor.backendRegistry))
	}

	if _, exists := executor.backendRegistry["backend1"]; !exists {
		t.Error("Expected backend1 to be registered")
	}
	if _, exists := executor.backendRegistry["backend2"]; !exists {
		t.Error("Expected backend2 to be registered")
	}
}

func TestNewPipelineExecutorEmptyList(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	if len(executor.backendRegistry) != 0 {
		t.Errorf("Expected 0 backends, got %d", len(executor.backendRegistry))
	}
}

func TestSelectBackendPreferred(t *testing.T) {
	backend1 := NewMockBackend("preferred-backend")
	backend2 := NewMockBackend("other-backend")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		PreferredBackend: "preferred-backend",
		Model:            "test-model",
	}

	backend, err := executor.selectBackend(stage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if backend.ID() != "preferred-backend" {
		t.Errorf("Expected backend 'preferred-backend', got %s", backend.ID())
	}
}

func TestSelectBackendByHardware(t *testing.T) {
	backend1 := NewMockBackend("npu-backend")
	backend1.hardware = "npu"

	backend2 := NewMockBackend("gpu-backend")
	backend2.hardware = "nvidia"

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		PreferredHardware: "npu",
		Model:            "test-model",
	}

	backend, err := executor.selectBackend(stage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if backend.Hardware() != "npu" {
		t.Errorf("Expected hardware 'npu', got %s", backend.Hardware())
	}
}

func TestSelectBackendDefault(t *testing.T) {
	backend1 := NewMockBackend("backend1")
	backend2 := NewMockBackend("backend2")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "test-model",
	}

	backend, err := executor.selectBackend(stage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if backend == nil {
		t.Fatal("Expected non-nil backend")
	}
}

func TestSelectBackendNotFound(t *testing.T) {
	backend := NewMockBackend("backend1")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		PreferredBackend: "nonexistent",
		Model:            "test-model",
	}

	_, err := executor.selectBackend(stage)
	if err == nil {
		t.Fatal("Expected error for nonexistent backend")
	}
}

func TestSelectBackendHardwareNotFound(t *testing.T) {
	backend := NewMockBackend("backend1")
	backend.hardware = "cpu"

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		PreferredHardware: "nvidia",
		Model:            "test-model",
	}

	_, err := executor.selectBackend(stage)
	if err == nil {
		t.Fatal("Expected error for unavailable hardware")
	}
}

func TestSelectBackendNoBackends(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "test-model",
	}

	_, err := executor.selectBackend(stage)
	if err == nil {
		t.Fatal("Expected error when no backends available")
	}
}

func TestGetBackendByID(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	retrieved, err := executor.getBackendByID("test-backend")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved.ID() != "test-backend" {
		t.Errorf("Expected backend 'test-backend', got %s", retrieved.ID())
	}
}

func TestGetBackendByIDNotFound(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	_, err := executor.getBackendByID("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent backend")
	}
}

func TestEstimateConfidence(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	confidence := executor.estimateConfidence("test output")
	if confidence != 0.8 {
		t.Errorf("Expected confidence 0.8, got %f", confidence)
	}

	confidence = executor.estimateConfidence(nil)
	if confidence != 0.8 {
		t.Errorf("Expected confidence 0.8, got %f", confidence)
	}
}

func TestExecuteOnBackendTextGen(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
	}

	output, err := executor.executeOnBackend(context.Background(), backend, stage, "test prompt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	if outputStr, ok := output.(string); !ok {
		t.Error("Expected string output")
	} else if outputStr == "" {
		t.Error("Expected non-empty output")
	}
}

func TestExecuteOnBackendEmbed(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeEmbed,
		Model: "nomic-embed-text",
	}

	output, err := executor.executeOnBackend(context.Background(), backend, stage, "test text")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	if embed, ok := output.([]float32); !ok {
		t.Error("Expected float32 slice output")
	} else if len(embed) == 0 {
		t.Error("Expected non-empty embedding")
	}
}

func TestExecuteOnBackendInvalidInput(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, 12345)
	if err == nil {
		t.Fatal("Expected error for invalid input type")
	}
}

func TestExecuteOnBackendEmbedInvalidInput(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeEmbed,
		Model: "nomic-embed-text",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, 12345)
	if err == nil {
		t.Fatal("Expected error for invalid input type")
	}
}

func TestExecuteOnBackendAudioToText(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeAudioToText,
		Model: "whisper",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "audio data")
	if err == nil {
		t.Fatal("Expected error - audio-to-text not implemented")
	}
}

func TestExecuteOnBackendTextToAudio(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextToAudio,
		Model: "tts",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "text to convert")
	if err == nil {
		t.Fatal("Expected error - text-to-audio not implemented")
	}
}

func TestExecuteOnBackendUnsupportedType(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeCustom,
		Model: "custom",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "input")
	if err == nil {
		t.Fatal("Expected error for unsupported stage type")
	}
}

func TestExecuteOnBackendGenerateError(t *testing.T) {
	backend := NewMockBackend("test-backend")
	backend.generateErr = errors.New("generate failed")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "test prompt")
	if err == nil {
		t.Fatal("Expected error from backend")
	}
}

func TestExecuteOnBackendEmbedError(t *testing.T) {
	backend := NewMockBackend("test-backend")
	backend.embedErr = errors.New("embed failed")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeEmbed,
		Model: "nomic-embed-text",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "test text")
	if err == nil {
		t.Fatal("Expected error from backend")
	}
}

func TestExecuteStageBasic(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
	}

	result, err := executor.executeStage(context.Background(), stage, "test prompt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful stage execution")
	}

	if result.StageID != "test-stage" {
		t.Errorf("Expected StageID 'test-stage', got %s", result.StageID)
	}

	if result.Backend != "test-backend" {
		t.Errorf("Expected backend 'test-backend', got %s", result.Backend)
	}

	if result.Metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if result.Metadata.DurationMs < 0 {
		t.Error("Expected non-negative duration")
	}
}

func TestExecuteStageInputTransform(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
		InputTransform: func(input interface{}) (interface{}, error) {
			return "transformed: " + input.(string), nil
		},
	}

	result, err := executor.executeStage(context.Background(), stage, "original input")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful execution")
	}

	if output, ok := result.Output.(string); !ok {
		t.Error("Expected string output")
	} else if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestExecuteStageInputTransformError(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
		InputTransform: func(input interface{}) (interface{}, error) {
			return nil, errors.New("transform failed")
		},
	}

	result, err := executor.executeStage(context.Background(), stage, "input")
	if err == nil {
		t.Fatal("Expected error from input transform")
	}

	if result.Success {
		t.Error("Expected failed execution")
	}
}

func TestExecuteStageOutputTransform(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
		OutputTransform: func(output interface{}) (interface{}, error) {
			return "transformed: " + output.(string), nil
		},
	}

	result, err := executor.executeStage(context.Background(), stage, "input")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful execution")
	}

	if output, ok := result.Output.(string); !ok {
		t.Error("Expected string output")
	} else if !contains(output, "transformed:") {
		t.Error("Expected output to be transformed")
	}
}

func TestExecuteStageOutputTransformError(t *testing.T) {
	backend := NewMockBackend("test-backend")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
		OutputTransform: func(output interface{}) (interface{}, error) {
			return nil, errors.New("transform failed")
		},
	}

	result, err := executor.executeStage(context.Background(), stage, "input")
	if err == nil {
		t.Fatal("Expected error from output transform")
	}

	if result.Success {
		t.Error("Expected failed execution")
	}
}

func TestExecuteStageBackendSelectionError(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
	}

	result, err := executor.executeStage(context.Background(), stage, "input")
	if err == nil {
		t.Fatal("Expected error when no backends available")
	}

	if result.Success {
		t.Error("Expected failed execution")
	}
}

func TestExecuteWithForwarding(t *testing.T) {
	backend1 := NewMockBackend("backend1")
	backend2 := NewMockBackend("backend2")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.9,
		MaxRetries:           2,
		EscalationPath:        []string{"backend1", "backend2"},
	}

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		Model:            "llama3:7b",
		ForwardingPolicy: policy,
	}

	metadata := &StageMetadata{
		StartTime: time.Now(),
		Backend:   "backend1",
	}

	output, result, err := executor.executeWithForwarding(context.Background(), stage, backend1, "test input", metadata)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if output == nil {
		t.Error("Expected non-nil output")
	}

	if result.AttemptCount == 0 {
		t.Error("Expected attempt count > 0")
	}
}

func TestExecuteWithForwardingAllRetriesFail(t *testing.T) {
	backend1 := NewMockBackend("backend1")
	backend1.generateErr = errors.New("backend1 failed")

	backend2 := NewMockBackend("backend2")
	backend2.generateErr = errors.New("backend2 failed")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.9,
		MaxRetries:           2,
		EscalationPath:        []string{"backend1", "backend2"},
	}

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		Model:            "llama3:7b",
		ForwardingPolicy: policy,
	}

	metadata := &StageMetadata{
		StartTime: time.Now(),
		Backend:   "backend1",
	}

	_, result, err := executor.executeWithForwarding(context.Background(), stage, backend1, "test input", metadata)
	if err == nil {
		t.Fatal("Expected error when all retries fail")
	}

	if result.AttemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", result.AttemptCount)
	}
}

func TestExecuteWithForwardingLowConfidence(t *testing.T) {
	backend1 := NewMockBackend("backend1")
	backend2 := NewMockBackend("backend2")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.99, // Very high confidence requirement
		MaxRetries:           2,
		EscalationPath:        []string{"backend1", "backend2"},
	}

	stage := &Stage{
		ID:               "test-stage",
		Type:             StageTypeTextGen,
		Model:            "llama3:7b",
		ForwardingPolicy: policy,
	}

	metadata := &StageMetadata{
		StartTime: time.Now(),
		Backend:   "backend1",
	}

	output, result, err := executor.executeWithForwarding(context.Background(), stage, backend1, "test input", metadata)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return output even if confidence is low after exhausting retries
	if output == nil {
		t.Error("Expected output even with low confidence")
	}

	if result.AttemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", result.AttemptCount)
	}
}

func TestExecutePipelineSimple(t *testing.T) {
	backend := NewMockBackend("backend1")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "test-pipeline",
		Name: "Test Pipeline",
		Stages: []*Stage{
			{
				ID:    "stage1",
				Type:  StageTypeTextGen,
				Model: "llama3:7b",
			},
		},
		Options: &PipelineOptions{
			ContinueOnError: false,
		},
	}

	result, err := executor.Execute(context.Background(), pipeline, "test prompt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful pipeline execution")
	}

	if result.PipelineID != "test-pipeline" {
		t.Errorf("Expected PipelineID 'test-pipeline', got %s", result.PipelineID)
	}

	if len(result.StageResults) != 1 {
		t.Errorf("Expected 1 stage result, got %d", len(result.StageResults))
	}

	if result.FinalOutput == nil {
		t.Error("Expected non-nil final output")
	}

	if result.TotalTimeMs < 0 {
		t.Error("Expected non-negative total time")
	}
}

func TestExecutePipelineMultipleStages(t *testing.T) {
	backend := NewMockBackend("backend1")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "test-pipeline",
		Name: "Multi-Stage Pipeline",
		Stages: []*Stage{
			{
				ID:    "stage1",
				Type:  StageTypeTextGen,
				Model: "llama3:7b",
			},
			{
				ID:    "stage2",
				Type:  StageTypeEmbed,
				Model: "nomic-embed-text",
				InputTransform: func(input interface{}) (interface{}, error) {
					return input.(string), nil
				},
			},
		},
		Options: &PipelineOptions{
			ContinueOnError: false,
		},
	}

	result, err := executor.Execute(context.Background(), pipeline, "test prompt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful pipeline execution")
	}

	if len(result.StageResults) != 2 {
		t.Errorf("Expected 2 stage results, got %d", len(result.StageResults))
	}

	// Check first stage
	if !result.StageResults[0].Success {
		t.Error("Expected first stage to succeed")
	}

	// Check second stage
	if !result.StageResults[1].Success {
		t.Error("Expected second stage to succeed")
	}
}

func TestExecutePipelineStageFailure(t *testing.T) {
	backend := NewMockBackend("backend1")
	backend.generateErr = errors.New("generation failed")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "test-pipeline",
		Name: "Test Pipeline",
		Stages: []*Stage{
			{
				ID:    "stage1",
				Type:  StageTypeTextGen,
				Model: "llama3:7b",
			},
		},
		Options: &PipelineOptions{
			ContinueOnError: false,
		},
	}

	result, err := executor.Execute(context.Background(), pipeline, "test prompt")
	if err == nil {
		t.Fatal("Expected error when stage fails")
	}

	if result.Success {
		t.Error("Expected failed pipeline execution")
	}

	if result.Error == nil {
		t.Error("Expected error to be set in result")
	}
}

func TestExecutePipelineContinueOnError(t *testing.T) {
	backend := NewMockBackend("backend1")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "test-pipeline",
		Name: "Test Pipeline",
		Stages: []*Stage{
			{
				ID:    "stage1",
				Type:  StageTypeTextGen,
				Model: "llama3:7b",
				InputTransform: func(input interface{}) (interface{}, error) {
					return nil, errors.New("stage1 failed")
				},
			},
			{
				ID:    "stage2",
				Type:  StageTypeTextGen,
				Model: "llama3:7b",
			},
		},
		Options: &PipelineOptions{
			ContinueOnError: true,
		},
	}

	result, err := executor.Execute(context.Background(), pipeline, "test prompt")
	// Should not return error due to ContinueOnError
	if err != nil {
		t.Fatalf("Expected no error with ContinueOnError, got: %v", err)
	}

	if len(result.StageResults) != 2 {
		t.Errorf("Expected 2 stage results, got %d", len(result.StageResults))
	}

	// First stage should have failed
	if result.StageResults[0].Success {
		t.Error("Expected first stage to fail")
	}

	// Second stage should have succeeded or failed independently
	if result.StageResults[1].StageID != "stage2" {
		t.Error("Expected second stage to be executed")
	}
}

func TestExecutePipelineDataFlow(t *testing.T) {
	backend := NewMockBackend("backend1")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "test-pipeline",
		Name: "Data Flow Pipeline",
		Stages: []*Stage{
			{
				ID:    "stage1",
				Type:  StageTypeTextGen,
				Model: "llama3:7b",
				OutputTransform: func(output interface{}) (interface{}, error) {
					// Pass through with metadata
					return output.(string) + " [processed]", nil
				},
			},
			{
				ID:    "stage2",
				Type:  StageTypeEmbed,
				Model: "nomic-embed-text",
			},
		},
		Options: &PipelineOptions{
			ContinueOnError: false,
		},
	}

	result, err := executor.Execute(context.Background(), pipeline, "original input")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful pipeline execution")
	}

	// Final output should be from second stage
	if _, ok := result.FinalOutput.([]float32); !ok {
		t.Error("Expected final output to be embedding from stage2")
	}
}

func TestStageTypes(t *testing.T) {
	tests := []struct {
		name      string
		stageType StageType
		expected  string
	}{
		{"AudioToText", StageTypeAudioToText, "audio_to_text"},
		{"TextGen", StageTypeTextGen, "text_generation"},
		{"TextToAudio", StageTypeTextToAudio, "text_to_audio"},
		{"Embed", StageTypeEmbed, "embedding"},
		{"Custom", StageTypeCustom, "custom"},
	}

	for _, tt := range tests {
		if string(tt.stageType) != tt.expected {
			t.Errorf("Expected %s to be %s, got %s", tt.name, tt.expected, string(tt.stageType))
		}
	}
}

func TestForwardingPolicyDefaults(t *testing.T) {
	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.75,
		MaxRetries:           3,
		EscalationPath:        []string{"backend1", "backend2", "backend3"},
	}

	if !policy.EnableConfidenceCheck {
		t.Error("Expected confidence check to be enabled")
	}

	if policy.MinConfidence != 0.75 {
		t.Errorf("Expected MinConfidence 0.75, got %f", policy.MinConfidence)
	}

	if policy.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", policy.MaxRetries)
	}

	if len(policy.EscalationPath) != 3 {
		t.Errorf("Expected 3 backends in escalation path, got %d", len(policy.EscalationPath))
	}
}

func TestPipelineOptionsDefaults(t *testing.T) {
	options := &PipelineOptions{
		EnableStreaming:  true,
		PreserveContext:  true,
		ContinueOnError:  false,
		CollectMetrics:   true,
		ParallelStages:   false,
	}

	if !options.EnableStreaming {
		t.Error("Expected streaming to be enabled")
	}

	if !options.PreserveContext {
		t.Error("Expected context preservation to be enabled")
	}

	if options.ContinueOnError {
		t.Error("Expected ContinueOnError to be false")
	}

	if !options.CollectMetrics {
		t.Error("Expected metrics collection to be enabled")
	}

	if options.ParallelStages {
		t.Error("Expected parallel stages to be false")
	}
}

func TestPipelineResultStructure(t *testing.T) {
	result := &PipelineResult{
		PipelineID:   "test",
		Success:      true,
		StageResults: make([]*StageResult, 0),
		FinalOutput:  "output",
		TotalTimeMs:  100,
		TotalEnergyWh: 0.5,
	}

	if result.PipelineID != "test" {
		t.Error("Expected PipelineID to be test")
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.TotalTimeMs != 100 {
		t.Errorf("Expected TotalTimeMs 100, got %d", result.TotalTimeMs)
	}
}

func TestStageMetadataTracking(t *testing.T) {
	backend := NewMockBackend("backend1")
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    "test-stage",
		Type:  StageTypeTextGen,
		Model: "llama3:7b",
	}

	result, err := executor.executeStage(context.Background(), stage, "test input")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	metadata := result.Metadata
	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if metadata.StartTime.IsZero() {
		t.Error("Expected start time to be set")
	}

	if metadata.EndTime.IsZero() {
		t.Error("Expected end time to be set")
	}

	if metadata.DurationMs < 0 {
		t.Error("Expected non-negative duration")
	}

	if metadata.Backend == "" {
		t.Error("Expected backend to be set")
	}

	if metadata.Model != "llama3:7b" {
		t.Error("Expected model to be set")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
