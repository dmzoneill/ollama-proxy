package pipeline

import (
	"context"
	"io"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// mockBackend implements backends.Backend for testing
type mockBackend struct {
	id                 string
	supportsAudioText  bool
	supportsTextAudio  bool
	supportsImageText  bool
	supportsTextImage  bool
	transcribeFunc     func(context.Context, *backends.TranscribeRequest) (*backends.TranscribeResponse, error)
	synthesizeFunc     func(context.Context, *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error)
	analyzeImageFunc   func(context.Context, *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error)
	generateImageFunc  func(context.Context, *backends.ImageGenRequest) (*backends.ImageGenResponse, error)
}

func (m *mockBackend) ID() string                                   { return m.id }
func (m *mockBackend) Type() string                                 { return "mock" }
func (m *mockBackend) Name() string                                 { return "Mock Backend" }
func (m *mockBackend) Hardware() string                             { return "test" }
func (m *mockBackend) IsHealthy() bool                              { return true }
func (m *mockBackend) HealthCheck(ctx context.Context) error        { return nil }
func (m *mockBackend) PowerWatts() float64                          { return 10.0 }
func (m *mockBackend) AvgLatencyMs() int32                           { return 100 }
func (m *mockBackend) Priority() int                                { return 5 }
func (m *mockBackend) SupportsGenerate() bool                       { return true }
func (m *mockBackend) SupportsStream() bool                         { return true }
func (m *mockBackend) SupportsEmbed() bool                          { return true }
func (m *mockBackend) SupportsAudioToText() bool                    { return m.supportsAudioText }
func (m *mockBackend) SupportsTextToAudio() bool                    { return m.supportsTextAudio }
func (m *mockBackend) SupportsImageToText() bool                    { return m.supportsImageText }
func (m *mockBackend) SupportsTextToImage() bool                    { return m.supportsTextImage }
func (m *mockBackend) SupportsVideoToText() bool                    { return false }
func (m *mockBackend) SupportsTextToVideo() bool                    { return false }
func (m *mockBackend) ListModels(ctx context.Context) ([]string, error)      { return []string{"test-model"}, nil }
func (m *mockBackend) SupportsModel(modelName string) bool          { return true }
func (m *mockBackend) GetMaxModelSizeGB() int                       { return 10 }
func (m *mockBackend) GetSupportedModelPatterns() []string          { return []string{"*"} }
func (m *mockBackend) GetPreferredModels() []string                 { return []string{} }
func (m *mockBackend) UpdateMetrics(latencyMs int32, success bool)  {}
func (m *mockBackend) GetMetrics() *backends.BackendMetrics         { return &backends.BackendMetrics{} }
func (m *mockBackend) Start(ctx context.Context) error              { return nil }
func (m *mockBackend) Stop(ctx context.Context) error               { return nil }

func (m *mockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return &backends.GenerateResponse{
		Response: "Generated text",
		Stats:    &backends.GenerationStats{},
	}, nil
}

func (m *mockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

func (m *mockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
		Stats:     &backends.GenerationStats{},
	}, nil
}

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

func (m *mockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, nil
}

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

func (m *mockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, io.ErrUnexpectedEOF // Force fallback to non-streaming
}

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

func (m *mockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, io.ErrUnexpectedEOF // Force fallback
}

func (m *mockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, nil
}

func (m *mockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, nil
}

func (m *mockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, nil
}

func (m *mockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, nil
}

// Test audio-to-text execution
func TestExecuteAudioToText(t *testing.T) {
	backend := &mockBackend{
		id:                "test-backend",
		supportsAudioText: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:                "audio-to-text",
		Type:              StageTypeAudioToText,
		Model:             "whisper-tiny",
		PreferredBackend:  "test-backend",
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
		id:                "test-backend",
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "text-to-audio",
		Type:             StageTypeTextToAudio,
		Model:            "piper-tts",
		PreferredBackend: "test-backend",
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
		id:                "test-backend",
		supportsImageText: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "image-to-text",
		Type:             StageTypeImageToText,
		Model:            "llava",
		PreferredBackend: "test-backend",
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
		id:                "test-backend",
		supportsTextImage: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "text-to-image",
		Type:             StageTypeTextToImage,
		Model:            "stable-diffusion",
		PreferredBackend: "test-backend",
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
		id:                "test-backend",
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "voice-assistant",
		Name: "Voice Assistant Test",
		Stages: []*Stage{
			{
				ID:               "audio-to-text",
				Type:             StageTypeAudioToText,
				Model:            "whisper-tiny",
				PreferredBackend: "test-backend",
			},
			{
				ID:               "llm-response",
				Type:             StageTypeTextGen,
				Model:            "llama3",
				PreferredBackend: "test-backend",
			},
			{
				ID:               "text-to-audio",
				Type:             StageTypeTextToAudio,
				Model:            "piper-tts",
				PreferredBackend: "test-backend",
			},
		},
		Options: &PipelineOptions{
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
		id:                "test-backend",
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "parallel-test",
		Name: "Parallel Test",
		Stages: []*Stage{
			{
				ID:               "stage1",
				Type:             StageTypeTextGen,
				Model:            "llama3",
				PreferredBackend: "test-backend",
			},
			{
				ID:               "stage2",
				Type:             StageTypeEmbed,
				Model:            "nomic-embed",
				PreferredBackend: "test-backend",
			},
		},
		Options: &PipelineOptions{
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
		id:                "test-backend",
		supportsAudioText: false, // Doesn't support audio
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	stage := &Stage{
		ID:               "audio-to-text",
		Type:             StageTypeAudioToText,
		Model:            "whisper-tiny",
		PreferredBackend: "test-backend",
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
		id:                "test-backend",
		supportsAudioText: true,
		transcribeFunc: func(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
			return nil, io.ErrUnexpectedEOF // Simulate error
		},
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "error-test",
		Name: "Error Test",
		Stages: []*Stage{
			{
				ID:               "audio-to-text",
				Type:             StageTypeAudioToText,
				Model:            "whisper-tiny",
				PreferredBackend: "test-backend",
			},
		},
		Options: &PipelineOptions{
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
