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
	healthErr         error
	generateErr       error
	embedErr          error
	supportedModels   map[string]bool
	metrics           *backends.BackendMetrics
	hardware          string
	id                string
	name              string
	backendType       string
	supportedPatterns []string
	preferredModels   []string
	priority          int
	powerWatts        float64
	maxModelSizeGB    int
	avgLatencyMs      int32
	healthy           bool
	supportsEmbed     bool
	supportsStream    bool
	supportsGenerate  bool
}

// NewMockBackend constructs a mockBackend.
func NewMockBackend(id string) *MockBackend {
	return &MockBackend{
		id:               id,
		backendType:      "mock",
		name:             fmt.Sprintf("Mock Backend %s", id),
		hardware:         strCPU,
		healthy:          true,
		powerWatts:       5.0,
		avgLatencyMs:     100,
		priority:         1,
		supportsGenerate: true,
		supportsStream:   true,
		supportsEmbed:    true,
		supportedModels: map[string]bool{
			strLlama37b:       true,
			strQwen2505b:      true,
			strNomicEmbedText: true,
			strTestModel:      true,
		},
		maxModelSizeGB:    70,
		supportedPatterns: []string{"*:*"},
		preferredModels:   []string{strLlama37b},
		metrics: &backends.BackendMetrics{
			RequestCount: 0,
			SuccessCount: 0,
			ErrorCount:   0,
		},
	}
}

// ID is part of the public API.
func (m *MockBackend) ID() string {
	return m.id
}

// Type is part of the public API.
func (m *MockBackend) Type() string {
	return m.backendType
}

// Name is part of the public API.
func (m *MockBackend) Name() string {
	return m.name
}

// Hardware is part of the public API.
func (m *MockBackend) Hardware() string {
	return m.hardware
}

// IsHealthy reports whether the condition holds.
func (m *MockBackend) IsHealthy() bool {
	return m.healthy
}

// HealthCheck is part of the public API.
func (m *MockBackend) HealthCheck(_ context.Context) error {
	return m.healthErr
}

// PowerWatts is part of the public API.
func (m *MockBackend) PowerWatts() float64 {
	return m.powerWatts
}

// AvgLatencyMs is part of the public API.
func (m *MockBackend) AvgLatencyMs() int32 {
	return m.avgLatencyMs
}

// Priority is part of the public API.
func (m *MockBackend) Priority() int {
	return m.priority
}

// SupportsGenerate is part of the public API.
func (m *MockBackend) SupportsGenerate() bool {
	return m.supportsGenerate
}

// SupportsStream is part of the public API.
func (m *MockBackend) SupportsStream() bool {
	return m.supportsStream
}

// SupportsEmbed is part of the public API.
func (m *MockBackend) SupportsEmbed() bool {
	return m.supportsEmbed
}

// ListModels is part of the public API.
func (m *MockBackend) ListModels(_ context.Context) ([]string, error) {
	models := make([]string, 0, len(m.supportedModels))
	for model := range m.supportedModels {
		models = append(models, model)
	}
	return models, nil
}

// SupportsModel is part of the public API.
func (m *MockBackend) SupportsModel(modelName string) bool {
	return m.supportedModels[modelName]
}

// GetMaxModelSizeGB is part of the public API.
func (m *MockBackend) GetMaxModelSizeGB() int {
	return m.maxModelSizeGB
}

// GetSupportedModelPatterns is part of the public API.
func (m *MockBackend) GetSupportedModelPatterns() []string {
	return m.supportedPatterns
}

// GetPreferredModels is part of the public API.
func (m *MockBackend) GetPreferredModels() []string {
	return m.preferredModels
}

// Generate is part of the public API.
func (m *MockBackend) Generate(_ context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
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

// GenerateStream is part of the public API.
func (m *MockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// Embed is part of the public API.
func (m *MockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
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

// UpdateMetrics is part of the public API.
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

// GetMetrics is part of the public API.
func (m *MockBackend) GetMetrics() *backends.BackendMetrics {
	return m.metrics
}

// Start is part of the public API.
func (m *MockBackend) Start(_ context.Context) error {
	return nil
}

// Stop is part of the public API.
func (m *MockBackend) Stop(_ context.Context) error {
	return nil
}

// Multimedia capability methods
func (m *MockBackend) SupportsAudioToText() bool { return false }

// SupportsTextToAudio is part of the public API.
func (m *MockBackend) SupportsTextToAudio() bool { return false }

// SupportsImageToText is part of the public API.
func (m *MockBackend) SupportsImageToText() bool { return false }

// SupportsTextToImage is part of the public API.
func (m *MockBackend) SupportsTextToImage() bool { return false }

// SupportsVideoToText is part of the public API.
func (m *MockBackend) SupportsVideoToText() bool { return false }

// SupportsTextToVideo is part of the public API.
func (m *MockBackend) SupportsTextToVideo() bool { return false }

// TranscribeAudio is part of the public API.
func (m *MockBackend) TranscribeAudio(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// TranscribeAudioStream is part of the public API.
func (m *MockBackend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeech is part of the public API.
func (m *MockBackend) SynthesizeSpeech(_ context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeechStream is part of the public API.
func (m *MockBackend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeImage is part of the public API.
func (m *MockBackend) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImage is part of the public API.
func (m *MockBackend) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImageStream is part of the public API.
func (m *MockBackend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideo is part of the public API.
func (m *MockBackend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideoStream is part of the public API.
func (m *MockBackend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideo is part of the public API.
func (m *MockBackend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideoStream is part of the public API.
func (m *MockBackend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// Tests for Executor

// TestNewPipelineExecutor is part of the public API.
func TestNewPipelineExecutor(t *testing.T) {
	backend1 := NewMockBackend(strBackend1)
	backend2 := NewMockBackend(strBackend2)

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	if len(executor.backendRegistry) != 2 {
		t.Errorf("Expected 2 backends in registry, got %d", len(executor.backendRegistry))
	}

	if _, exists := executor.backendRegistry[strBackend1]; !exists {
		t.Error("Expected backend1 to be registered")
	}
	if _, exists := executor.backendRegistry[strBackend2]; !exists {
		t.Error("Expected backend2 to be registered")
	}
}

// TestNewPipelineExecutorEmptyList is part of the public API.
func TestNewPipelineExecutorEmptyList(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	if len(executor.backendRegistry) != 0 {
		t.Errorf("Expected 0 backends, got %d", len(executor.backendRegistry))
	}
}

// TestSelectBackendPreferred is part of the public API.
func TestSelectBackendPreferred(t *testing.T) {
	backend1 := NewMockBackend("preferred-backend")
	backend2 := NewMockBackend("other-backend")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	stage := &Stage{
		ID:               strTestStage,
		Type:             StageTypeTextGen,
		PreferredBackend: "preferred-backend",
		Model:            strTestModel,
	}

	backend, err := executor.selectBackend(stage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if backend.ID() != "preferred-backend" {
		t.Errorf("Expected backend 'preferred-backend', got %s", backend.ID())
	}
}

// TestSelectBackendByHardware is part of the public API.
func TestSelectBackendByHardware(t *testing.T) {
	backend1 := NewMockBackend("npu-backend")
	backend1.hardware = strNpu

	backend2 := NewMockBackend("gpu-backend")
	backend2.hardware = strNvidia

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	stage := &Stage{
		ID:                strTestStage,
		Type:              StageTypeTextGen,
		PreferredHardware: strNpu,
		Model:             strTestModel,
	}

	backend, err := executor.selectBackend(stage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if backend.Hardware() != strNpu {
		t.Errorf("Expected hardware 'npu', got %s", backend.Hardware())
	}
}

// TestSelectBackendDefault is part of the public API.
func TestSelectBackendDefault(t *testing.T) {
	backend1 := NewMockBackend(strBackend1)
	backend2 := NewMockBackend(strBackend2)

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strTestModel,
	}

	backend, err := executor.selectBackend(stage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if backend == nil {
		t.Fatal("Expected non-nil backend")
	}
}

// TestSelectBackendNotFound is part of the public API.
func TestSelectBackendNotFound(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               strTestStage,
		Type:             StageTypeTextGen,
		PreferredBackend: "nonexistent",
		Model:            strTestModel,
	}

	_, err := executor.selectBackend(stage)
	if err == nil {
		t.Fatal("Expected error for nonexistent backend")
	}
}

// TestSelectBackendHardwareNotFound is part of the public API.
func TestSelectBackendHardwareNotFound(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	backend.hardware = strCPU

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:                strTestStage,
		Type:              StageTypeTextGen,
		PreferredHardware: strNvidia,
		Model:             strTestModel,
	}

	_, err := executor.selectBackend(stage)
	if err == nil {
		t.Fatal("Expected error for unavailable hardware")
	}
}

// TestSelectBackendNoBackends is part of the public API.
func TestSelectBackendNoBackends(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strTestModel,
	}

	_, err := executor.selectBackend(stage)
	if err == nil {
		t.Fatal("Expected error when no backends available")
	}
}

// TestGetBackendByID is part of the public API.
func TestGetBackendByID(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	retrieved, err := executor.getBackendByID(strTestBackend)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved.ID() != strTestBackend {
		t.Errorf("Expected backend 'test-backend', got %s", retrieved.ID())
	}
}

// TestGetBackendByIDNotFound is part of the public API.
func TestGetBackendByIDNotFound(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	_, err := executor.getBackendByID("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent backend")
	}
}

// TestEstimateConfidence is part of the public API.
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

// TestExecuteOnBackendTextGen is part of the public API.
func TestExecuteOnBackendTextGen(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
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

// TestExecuteOnBackendEmbed is part of the public API.
func TestExecuteOnBackendEmbed(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeEmbed,
		Model: strNomicEmbedText,
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

// TestExecuteOnBackendInvalidInput is part of the public API.
func TestExecuteOnBackendInvalidInput(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, 12345)
	if err == nil {
		t.Fatal("Expected error for invalid input type")
	}
}

// TestExecuteOnBackendEmbedInvalidInput is part of the public API.
func TestExecuteOnBackendEmbedInvalidInput(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeEmbed,
		Model: strNomicEmbedText,
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, 12345)
	if err == nil {
		t.Fatal("Expected error for invalid input type")
	}
}

// TestExecuteOnBackendAudioToText is part of the public API.
func TestExecuteOnBackendAudioToText(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeAudioToText,
		Model: "whisper",
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "audio data")
	if err == nil {
		t.Fatal("Expected error - audio-to-text not implemented")
	}
}

// TestExecuteOnBackendTextToAudio is part of the public API.
func TestExecuteOnBackendTextToAudio(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextToAudio,
		Model: strTts,
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "text to convert")
	if err == nil {
		t.Fatal("Expected error - text-to-audio not implemented")
	}
}

// TestExecuteOnBackendUnsupportedType is part of the public API.
func TestExecuteOnBackendUnsupportedType(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeCustom,
		Model: string(StageTypeCustom),
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "input")
	if err == nil {
		t.Fatal("Expected error for unsupported stage type")
	}
}

// TestExecuteOnBackendGenerateError is part of the public API.
func TestExecuteOnBackendGenerateError(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	backend.generateErr = errors.New("generate failed")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "test prompt")
	if err == nil {
		t.Fatal("Expected error from backend")
	}
}

// TestExecuteOnBackendEmbedError is part of the public API.
func TestExecuteOnBackendEmbedError(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	backend.embedErr = errors.New("embed failed")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeEmbed,
		Model: strNomicEmbedText,
	}

	_, err := executor.executeOnBackend(context.Background(), backend, stage, "test text")
	if err == nil {
		t.Fatal("Expected error from backend")
	}
}

// TestExecuteStageBasic is part of the public API.
func TestExecuteStageBasic(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
	}

	result, err := executor.executeStage(context.Background(), stage, "test prompt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful stage execution")
	}

	if result.StageID != strTestStage {
		t.Errorf("Expected StageID 'test-stage', got %s", result.StageID)
	}

	if result.Backend != strTestBackend {
		t.Errorf("Expected backend 'test-backend', got %s", result.Backend)
	}

	if result.Metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if result.Metadata.DurationMs < 0 {
		t.Error("Expected non-negative duration")
	}
}

// TestExecuteStageInputTransform is part of the public API.
func TestExecuteStageInputTransform(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
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

// TestExecuteStageInputTransformError is part of the public API.
func TestExecuteStageInputTransformError(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
		InputTransform: func(_ interface{}) (interface{}, error) {
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

// TestExecuteStageOutputTransform is part of the public API.
func TestExecuteStageOutputTransform(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
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

// TestExecuteStageOutputTransformError is part of the public API.
func TestExecuteStageOutputTransformError(t *testing.T) {
	backend := NewMockBackend(strTestBackend)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
		OutputTransform: func(_ interface{}) (interface{}, error) {
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

// TestExecuteStageBackendSelectionError is part of the public API.
func TestExecuteStageBackendSelectionError(t *testing.T) {
	executor := NewPipelineExecutor([]backends.Backend{})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
	}

	result, err := executor.executeStage(context.Background(), stage, "input")
	if err == nil {
		t.Fatal("Expected error when no backends available")
	}

	if result.Success {
		t.Error("Expected failed execution")
	}
}

// TestExecuteWithForwarding is part of the public API.
func TestExecuteWithForwarding(t *testing.T) {
	backend1 := NewMockBackend(strBackend1)
	backend2 := NewMockBackend(strBackend2)

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.9,
		MaxRetries:            2,
		EscalationPath:        []string{strBackend1, strBackend2},
	}

	stage := &Stage{
		ID:               strTestStage,
		Type:             StageTypeTextGen,
		Model:            strLlama37b,
		ForwardingPolicy: policy,
	}

	metadata := &StageMetadata{
		StartTime: time.Now(),
		Backend:   strBackend1,
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

// TestExecuteWithForwardingAllRetriesFail is part of the public API.
func TestExecuteWithForwardingAllRetriesFail(t *testing.T) {
	backend1 := NewMockBackend(strBackend1)
	backend1.generateErr = errors.New("backend1 failed")

	backend2 := NewMockBackend(strBackend2)
	backend2.generateErr = errors.New("backend2 failed")

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.9,
		MaxRetries:            2,
		EscalationPath:        []string{strBackend1, strBackend2},
	}

	stage := &Stage{
		ID:               strTestStage,
		Type:             StageTypeTextGen,
		Model:            strLlama37b,
		ForwardingPolicy: policy,
	}

	metadata := &StageMetadata{
		StartTime: time.Now(),
		Backend:   strBackend1,
	}

	_, result, err := executor.executeWithForwarding(context.Background(), stage, backend1, "test input", metadata)
	if err == nil {
		t.Fatal("Expected error when all retries fail")
	}

	if result.AttemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", result.AttemptCount)
	}
}

// TestExecuteWithForwardingLowConfidence is part of the public API.
func TestExecuteWithForwardingLowConfidence(t *testing.T) {
	backend1 := NewMockBackend(strBackend1)
	backend2 := NewMockBackend(strBackend2)

	executor := NewPipelineExecutor([]backends.Backend{backend1, backend2})

	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.99, // Very high confidence requirement
		MaxRetries:            2,
		EscalationPath:        []string{strBackend1, strBackend2},
	}

	stage := &Stage{
		ID:               strTestStage,
		Type:             StageTypeTextGen,
		Model:            strLlama37b,
		ForwardingPolicy: policy,
	}

	metadata := &StageMetadata{
		StartTime: time.Now(),
		Backend:   strBackend1,
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

// TestExecutePipelineSimple is part of the public API.
func TestExecutePipelineSimple(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   strTestPipeline,
		Name: strTestPipeline2,
		Stages: []*Stage{
			{
				ID:    strStage1,
				Type:  StageTypeTextGen,
				Model: strLlama37b,
			},
		},
		Options: &Options{
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

	if result.PipelineID != strTestPipeline {
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

// TestExecutePipelineMultipleStages is part of the public API.
func TestExecutePipelineMultipleStages(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   strTestPipeline,
		Name: "Multi-Stage Pipeline",
		Stages: []*Stage{
			{
				ID:    strStage1,
				Type:  StageTypeTextGen,
				Model: strLlama37b,
			},
			{
				ID:    strStage2,
				Type:  StageTypeEmbed,
				Model: strNomicEmbedText,
				InputTransform: func(input interface{}) (interface{}, error) {
					return input.(string), nil
				},
			},
		},
		Options: &Options{
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

// TestExecutePipelineStageFailure is part of the public API.
func TestExecutePipelineStageFailure(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	backend.generateErr = errors.New("generation failed")

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   strTestPipeline,
		Name: strTestPipeline2,
		Stages: []*Stage{
			{
				ID:    strStage1,
				Type:  StageTypeTextGen,
				Model: strLlama37b,
			},
		},
		Options: &Options{
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

// TestExecutePipelineContinueOnError is part of the public API.
func TestExecutePipelineContinueOnError(t *testing.T) {
	backend := NewMockBackend(strBackend1)

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   strTestPipeline,
		Name: strTestPipeline2,
		Stages: []*Stage{
			{
				ID:    strStage1,
				Type:  StageTypeTextGen,
				Model: strLlama37b,
				InputTransform: func(_ interface{}) (interface{}, error) {
					return nil, errors.New("stage1 failed")
				},
			},
			{
				ID:    strStage2,
				Type:  StageTypeTextGen,
				Model: strLlama37b,
			},
		},
		Options: &Options{
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
	if result.StageResults[1].StageID != strStage2 {
		t.Error("Expected second stage to be executed")
	}
}

// TestExecutePipelineDataFlow is part of the public API.
func TestExecutePipelineDataFlow(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   strTestPipeline,
		Name: "Data Flow Pipeline",
		Stages: []*Stage{
			{
				ID:    strStage1,
				Type:  StageTypeTextGen,
				Model: strLlama37b,
				OutputTransform: func(output interface{}) (interface{}, error) {
					// Pass through with metadata
					return output.(string) + " [processed]", nil
				},
			},
			{
				ID:    strStage2,
				Type:  StageTypeEmbed,
				Model: strNomicEmbedText,
			},
		},
		Options: &Options{
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

// TestStageTypes is part of the public API.
func TestStageTypes(t *testing.T) {
	tests := []struct {
		name      string
		stageType StageType
		expected  string
	}{
		{name: "AudioToText", stageType: StageTypeAudioToText, expected: "audio_to_text"},
		{name: "TextGen", stageType: StageTypeTextGen, expected: string(StageTypeTextGen)},
		{name: "TextToAudio", stageType: StageTypeTextToAudio, expected: "text_to_audio"},
		{name: "Embed", stageType: StageTypeEmbed, expected: string(StageTypeEmbed)},
		{name: "Custom", stageType: StageTypeCustom, expected: string(StageTypeCustom)},
	}

	for _, tt := range tests {
		if string(tt.stageType) != tt.expected {
			t.Errorf("Expected %s to be %s, got %s", tt.name, tt.expected, string(tt.stageType))
		}
	}
}

// TestForwardingPolicyDefaults is part of the public API.
func TestForwardingPolicyDefaults(t *testing.T) {
	policy := &ForwardingPolicy{
		EnableConfidenceCheck: true,
		MinConfidence:         0.75,
		MaxRetries:            3,
		EscalationPath:        []string{strBackend1, strBackend2, "backend3"},
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

// TestPipelineOptionsDefaults is part of the public API.
func TestPipelineOptionsDefaults(t *testing.T) {
	options := &Options{
		EnableStreaming: true,
		PreserveContext: true,
		ContinueOnError: false,
		CollectMetrics:  true,
		ParallelStages:  false,
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

// TestPipelineResultStructure is part of the public API.
func TestPipelineResultStructure(t *testing.T) {
	result := &Result{
		PipelineID:    strTest,
		Success:       true,
		StageResults:  make([]*StageResult, 0),
		FinalOutput:   "output",
		TotalTimeMs:   100,
		TotalEnergyWh: 0.5,
	}

	if result.PipelineID != strTest {
		t.Error("Expected PipelineID to be test")
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.TotalTimeMs != 100 {
		t.Errorf("Expected TotalTimeMs 100, got %d", result.TotalTimeMs)
	}
	if result.StageResults == nil {
		t.Error("Expected StageResults to be non-nil")
	}
	if result.FinalOutput != "output" {
		t.Errorf("Expected FinalOutput output, got %v", result.FinalOutput)
	}
	if result.TotalEnergyWh != 0.5 {
		t.Errorf("Expected TotalEnergyWh 0.5, got %v", result.TotalEnergyWh)
	}
}

// TestStageMetadataTracking is part of the public API.
func TestStageMetadataTracking(t *testing.T) {
	backend := NewMockBackend(strBackend1)
	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:    strTestStage,
		Type:  StageTypeTextGen,
		Model: strLlama37b,
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

	if metadata.Model != strLlama37b {
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
