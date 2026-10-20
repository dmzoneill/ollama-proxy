package pipeline

import (
	"context"
	"io"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// mockBackend implements backends.Backend for testing
type mockBackend struct {
	transcribeFunc    func(context.Context, *backends.TranscribeRequest) (*backends.TranscribeResponse, error)
	synthesizeFunc    func(context.Context, *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error)
	analyzeImageFunc  func(context.Context, *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error)
	generateImageFunc func(context.Context, *backends.ImageGenRequest) (*backends.ImageGenResponse, error)
	id                string
	supportsAudioText bool
	supportsTextAudio bool
	supportsImageText bool
	supportsTextImage bool
}

// ID is part of the public API.
func (m *mockBackend) ID() string { return m.id }

// Type is part of the public API.
func (m *mockBackend) Type() string { return "mock" }

// Name is part of the public API.
func (m *mockBackend) Name() string { return "Mock Backend" }

// Hardware is part of the public API.
func (m *mockBackend) Hardware() string { return strTest }

// IsHealthy reports whether the condition holds.
func (m *mockBackend) IsHealthy() bool { return true }

// HealthCheck is part of the public API.
func (m *mockBackend) HealthCheck(_ context.Context) error { return nil }

// PowerWatts is part of the public API.
func (m *mockBackend) PowerWatts() float64 { return 10.0 }

// AvgLatencyMs is part of the public API.
func (m *mockBackend) AvgLatencyMs() int32 { return 100 }

// Priority is part of the public API.
func (m *mockBackend) Priority() int { return 5 }

// SupportsGenerate is part of the public API.
func (m *mockBackend) SupportsGenerate() bool { return true }

// SupportsStream is part of the public API.
func (m *mockBackend) SupportsStream() bool { return true }

// SupportsEmbed is part of the public API.
func (m *mockBackend) SupportsEmbed() bool { return true }

// SupportsAudioToText is part of the public API.
func (m *mockBackend) SupportsAudioToText() bool { return m.supportsAudioText }

// SupportsTextToAudio is part of the public API.
func (m *mockBackend) SupportsTextToAudio() bool { return m.supportsTextAudio }

// SupportsImageToText is part of the public API.
func (m *mockBackend) SupportsImageToText() bool { return m.supportsImageText }

// SupportsTextToImage is part of the public API.
func (m *mockBackend) SupportsTextToImage() bool { return m.supportsTextImage }

// SupportsVideoToText is part of the public API.
func (m *mockBackend) SupportsVideoToText() bool { return false }

// SupportsTextToVideo is part of the public API.
func (m *mockBackend) SupportsTextToVideo() bool { return false }

// ListModels is part of the public API.
func (m *mockBackend) ListModels(_ context.Context) ([]string, error) {
	return []string{strTestModel}, nil
}

// SupportsModel is part of the public API.
func (m *mockBackend) SupportsModel(_ string) bool { return true }

// GetMaxModelSizeGB is part of the public API.
func (m *mockBackend) GetMaxModelSizeGB() int { return 10 }

// GetSupportedModelPatterns is part of the public API.
func (m *mockBackend) GetSupportedModelPatterns() []string { return []string{"*"} }

// GetPreferredModels is part of the public API.
func (m *mockBackend) GetPreferredModels() []string { return []string{} }

// UpdateMetrics is part of the public API.
func (m *mockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *mockBackend) GetMetrics() *backends.BackendMetrics { return &backends.BackendMetrics{} }

// Start is part of the public API.
func (m *mockBackend) Start(_ context.Context) error { return nil }

// Stop is part of the public API.
func (m *mockBackend) Stop(_ context.Context) error { return nil }

// Generate is part of the public API.
func (m *mockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return &backends.GenerateResponse{
		Response: "Generated text",
		Stats:    &backends.GenerationStats{},
	}, nil
}

// GenerateStream is part of the public API.
func (m *mockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

// Embed is part of the public API.
func (m *mockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
		Stats:     &backends.GenerationStats{},
	}, nil
}

// TranscribeAudio is part of the public API.
func (m *mockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	if m.transcribeFunc != nil {
		return m.transcribeFunc(ctx, req)
	}
	return &backends.TranscribeResponse{
		Text:       "Transcribed text",
		Language:   "en",
		Confidence: 0.95,
		Segments:   []backends.TranscriptSegment{},
		Stats:      &backends.GenerationStats{},
	}, nil
}

// TranscribeAudioStream is part of the public API.
func (m *mockBackend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, nil
}

// SynthesizeSpeech is part of the public API.
func (m *mockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, req)
	}
	return &backends.SynthesizeResponse{
		AudioData:  []byte{0x00, 0x01, 0x02, 0x03},
		Format:     backends.AudioFormatPCM,
		SampleRate: 22050,
		Duration:   1000,
		Stats:      &backends.GenerationStats{},
	}, nil
}

// SynthesizeSpeechStream is part of the public API.
func (m *mockBackend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, io.ErrUnexpectedEOF // Force fallback to non-streaming
}

// AnalyzeImage is part of the public API.
func (m *mockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	if m.analyzeImageFunc != nil {
		return m.analyzeImageFunc(ctx, req)
	}
	return &backends.ImageAnalysisResponse{
		Text:       "A cat sitting on a mat",
		Confidence: 0.92,
		Detections: []backends.Detection{},
		Stats:      &backends.GenerationStats{},
	}, nil
}

// GenerateImage is part of the public API.
func (m *mockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	if m.generateImageFunc != nil {
		return m.generateImageFunc(ctx, req)
	}
	return &backends.ImageGenResponse{
		Images: []backends.GeneratedImage{
			{
				ImageData: []byte{0xFF, 0xD8, 0xFF, 0xE0}, // JPEG header
				Format:    backends.ImageFormatJPEG,
				Width:     512,
				Height:    512,
				Seed:      12345,
			},
		},
		Stats: &backends.GenerationStats{},
	}, nil
}

// GenerateImageStream is part of the public API.
func (m *mockBackend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, io.ErrUnexpectedEOF // Force fallback
}

// AnalyzeVideo is part of the public API.
func (m *mockBackend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, nil
}

// AnalyzeVideoStream is part of the public API.
func (m *mockBackend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, nil
}

// GenerateVideo is part of the public API.
func (m *mockBackend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, nil
}

// GenerateVideoStream is part of the public API.
func (m *mockBackend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, nil
}

// Test audio-to-text execution
func TestExecuteAudioToText(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsAudioText: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               strAudioToText,
		Type:             StageTypeAudioToText,
		Model:            strWhisperTiny,
		PreferredBackend: strTestBackend,
	}

	// Test with raw audio bytes
	audioData := []byte{0x00, 0x01, 0x02, 0x03}
	result, err := executor.executeAudioToText(context.Background(), backend, stage, audioData)

	if err != nil {
		t.Fatalf("executeAudioToText failed: %v", err)
	}

	text, ok := result.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result)
	}

	if text != "Transcribed text" {
		t.Errorf("expected 'Transcribed text', got '%s'", text)
	}
}

// Test text-to-audio execution
func TestExecuteTextToAudio(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "text-to-audio",
		Type:             StageTypeTextToAudio,
		Model:            "piper-tts",
		PreferredBackend: strTestBackend,
	}

	// Test with text input
	text := "Hello, world!"
	result, err := executor.executeTextToAudio(context.Background(), backend, stage, text)

	if err != nil {
		t.Fatalf("executeTextToAudio failed: %v", err)
	}

	audioData, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte result, got %T", result)
	}

	if len(audioData) == 0 {
		t.Error("expected non-empty audio data")
	}
}

// Test image-to-text execution
func TestExecuteImageToText(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsImageText: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "image-to-text",
		Type:             StageTypeImageToText,
		Model:            "llava",
		PreferredBackend: strTestBackend,
	}

	// Test with raw image bytes
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
	result, err := executor.executeImageToText(context.Background(), backend, stage, imageData)

	if err != nil {
		t.Fatalf("executeImageToText failed: %v", err)
	}

	text, ok := result.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result)
	}

	if text != "A cat sitting on a mat" {
		t.Errorf("expected 'A cat sitting on a mat', got '%s'", text)
	}
}

// Test text-to-image execution
func TestExecuteTextToImage(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsTextImage: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "text-to-image",
		Type:             StageTypeTextToImage,
		Model:            "stable-diffusion",
		PreferredBackend: strTestBackend,
	}

	// Test with text prompt
	prompt := "A sunset over mountains"
	result, err := executor.executeTextToImage(context.Background(), backend, stage, prompt)

	if err != nil {
		t.Fatalf("executeTextToImage failed: %v", err)
	}

	imageData, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte result, got %T", result)
	}

	if len(imageData) == 0 {
		t.Error("expected non-empty image data")
	}
}

// Test full voice assistant pipeline with multimedia stages
func TestVoiceAssistantMultimediaPipeline(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "voice-assistant",
		Name: "Voice Assistant Test",
		Stages: []*Stage{
			{
				ID:               strAudioToText,
				Type:             StageTypeAudioToText,
				Model:            strWhisperTiny,
				PreferredBackend: strTestBackend,
			},
			{
				ID:               "llm-response",
				Type:             StageTypeTextGen,
				Model:            strLlama3,
				PreferredBackend: strTestBackend,
			},
			{
				ID:               "text-to-audio",
				Type:             StageTypeTextToAudio,
				Model:            "piper-tts",
				PreferredBackend: strTestBackend,
			},
		},
		Options: &Options{
			EnableStreaming: true,
			PreserveContext: true,
			CollectMetrics:  true,
		},
	}

	// Execute pipeline with audio input
	audioInput := []byte{0x00, 0x01, 0x02, 0x03}
	result, err := executor.Execute(context.Background(), pipeline, audioInput)

	if err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	if !result.Success {
		t.Error("expected successful pipeline execution")
	}

	if len(result.StageResults) != 3 {
		t.Errorf("expected 3 stage results, got %d", len(result.StageResults))
	}

	// Check final output is audio
	audioOutput, ok := result.FinalOutput.([]byte)
	if !ok {
		t.Fatalf("expected []byte final output, got %T", result.FinalOutput)
	}

	if len(audioOutput) == 0 {
		t.Error("expected non-empty audio output")
	}
}

// Test parallel pipeline execution
func TestParallelPipelineExecution(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "parallel-test",
		Name: "Parallel Test",
		Stages: []*Stage{
			{
				ID:               strStage1,
				Type:             StageTypeTextGen,
				Model:            strLlama3,
				PreferredBackend: strTestBackend,
			},
			{
				ID:               strStage2,
				Type:             StageTypeEmbed,
				Model:            "nomic-embed",
				PreferredBackend: strTestBackend,
			},
		},
		Options: &Options{
			ParallelStages: true, // Enable parallel execution
		},
	}

	// Execute pipeline
	input := "Test input"
	result, err := executor.Execute(context.Background(), pipeline, input)

	if err != nil {
		t.Fatalf("parallel pipeline execution failed: %v", err)
	}

	if !result.Success {
		t.Error("expected successful pipeline execution")
	}

	if len(result.StageResults) != 2 {
		t.Errorf("expected 2 stage results, got %d", len(result.StageResults))
	}
}

// Test backend capability check
func TestBackendCapabilityCheck(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsAudioText: false, // Doesn't support audio
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               strAudioToText,
		Type:             StageTypeAudioToText,
		Model:            strWhisperTiny,
		PreferredBackend: strTestBackend,
	}

	audioData := []byte{0x00, 0x01, 0x02, 0x03}
	_, err := executor.executeAudioToText(context.Background(), backend, stage, audioData)

	if err == nil {
		t.Error("expected error when backend doesn't support audio-to-text")
	}
}

// Test error handling in pipeline
func TestPipelineErrorHandling(t *testing.T) {
	backend := &mockBackend{
		id:                strTestBackend,
		supportsAudioText: true,
		transcribeFunc: func(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
			return nil, io.ErrUnexpectedEOF // Simulate error
		},
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "error-test",
		Name: "Error Test",
		Stages: []*Stage{
			{
				ID:               strAudioToText,
				Type:             StageTypeAudioToText,
				Model:            strWhisperTiny,
				PreferredBackend: strTestBackend,
			},
		},
		Options: &Options{
			ContinueOnError: false,
		},
	}

	audioInput := []byte{0x00, 0x01, 0x02, 0x03}
	result, err := executor.Execute(context.Background(), pipeline, audioInput)

	if err == nil {
		t.Error("expected error from pipeline execution")
	}

	if result.Success {
		t.Error("expected unsuccessful pipeline execution")
	}
}
