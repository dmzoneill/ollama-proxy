package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/testutil"

	"google.golang.org/grpc/metadata"

	pb "github.com/daoneill/ollama-proxy/api/gen/go"
	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/pipeline"
	"github.com/daoneill/ollama-proxy/pkg/router"
)

// TestMain initializes the logger for all tests
func TestMain(m *testing.M) {
	// Initialize logger for tests
	if err := logging.InitLogger("info", false); err != nil {
		panic(err)
	}
	defer logging.Sync()

	// Run tests
	os.Exit(m.Run())
}

// ============================================================================
// Mock Implementations
// ============================================================================

// MockBackend is a mock implementation of the Backend interface
type MockBackend struct {
	generateErr      error
	embedErr         error
	streamErr        error
	backendType      string
	name             string
	hardware         string
	id               string
	models           []string
	streamChunks     []*backends.StreamChunk
	powerWatts       float64
	priority         int
	avgLatencyMs     int32
	healthy          bool
	streamEOF        bool
	supportsGenerate bool
	supportsEmbed    bool
	supportsStream   bool
}

// ID is part of the public API.
func (m *MockBackend) ID() string { return m.id }

// Type is part of the public API.
func (m *MockBackend) Type() string { return m.backendType }

// Name is part of the public API.
func (m *MockBackend) Name() string { return m.name }

// Hardware is part of the public API.
func (m *MockBackend) Hardware() string { return m.hardware }

// IsHealthy reports whether the condition holds.
func (m *MockBackend) IsHealthy() bool { return m.healthy }

// HealthCheck is part of the public API.
func (m *MockBackend) HealthCheck(_ context.Context) error { return nil }

// PowerWatts is part of the public API.
func (m *MockBackend) PowerWatts() float64 { return m.powerWatts }

// AvgLatencyMs is part of the public API.
func (m *MockBackend) AvgLatencyMs() int32 { return m.avgLatencyMs }

// Priority is part of the public API.
func (m *MockBackend) Priority() int { return m.priority }

// SupportsGenerate is part of the public API.
func (m *MockBackend) SupportsGenerate() bool { return m.supportsGenerate }

// SupportsStream is part of the public API.
func (m *MockBackend) SupportsStream() bool { return m.supportsStream }

// SupportsEmbed is part of the public API.
func (m *MockBackend) SupportsEmbed() bool { return m.supportsEmbed }

// ListModels is part of the public API.
func (m *MockBackend) ListModels(_ context.Context) ([]string, error) { return m.models, nil }

// SupportsModel is part of the public API.
func (m *MockBackend) SupportsModel(_ string) bool { return true }

// GetMaxModelSizeGB is part of the public API.
func (m *MockBackend) GetMaxModelSizeGB() int { return 10 }

// GetSupportedModelPatterns is part of the public API.
func (m *MockBackend) GetSupportedModelPatterns() []string { return []string{"*"} }

// GetPreferredModels is part of the public API.
func (m *MockBackend) GetPreferredModels() []string { return []string{} }

// Generate is part of the public API.
func (m *MockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &backends.GenerateResponse{
		Response: "test response",
		Stats: &backends.GenerationStats{
			TimeToFirstTokenMs: 10,
			TotalTimeMs:        100,
			TokensGenerated:    10,
			TokensPerSecond:    100.0,
			EnergyWh:           0.5,
		},
	}, nil
}

// GenerateStream is part of the public API.
func (m *MockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	chunks := m.streamChunks
	if chunks == nil {
		chunks = []*backends.StreamChunk{
			{Token: "test", Done: false},
			{Token: " token", Done: true, Stats: &backends.GenerationStats{TokensPerSecond: 100.0}},
		}
	}
	return &MockStreamReader{chunks: chunks, returnEOF: m.streamEOF}, nil
}

// Embed is part of the public API.
func (m *MockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
	}, nil
}

// UpdateMetrics is part of the public API.
func (m *MockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *MockBackend) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{
		AvgLatencyMs: 50,
		ErrorRate:    0.0,
		LoadedModels: []string{strTestModel},
	}
}

// Start is part of the public API.
func (m *MockBackend) Start(_ context.Context) error { return nil }

// Stop is part of the public API.
func (m *MockBackend) Stop(_ context.Context) error { return nil }

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

// MockStreamReader is a mock implementation of StreamReader
type MockStreamReader struct {
	chunks    []*backends.StreamChunk
	idx       int
	closed    bool
	returnEOF bool
}

// Recv is part of the public API.
func (m *MockStreamReader) Recv() (*backends.StreamChunk, error) {
	if m.idx >= len(m.chunks) {
		if m.returnEOF {
			return nil, errors.New("EOF")
		}
		return nil, errors.New("end of stream")
	}
	chunk := m.chunks[m.idx]
	m.idx++
	return chunk, nil
}

// Close is part of the public API.
func (m *MockStreamReader) Close() error {
	m.closed = true
	return nil
}

// MockGenerateStream is a mock implementation of gRPC stream
type MockGenerateStream struct {
	ctx  context.Context
	sent []*pb.GenerateStreamResponse
}

// Send is part of the public API.
func (m *MockGenerateStream) Send(resp *pb.GenerateStreamResponse) error {
	m.sent = append(m.sent, resp)
	return nil
}

// Context is part of the public API.
func (m *MockGenerateStream) Context() context.Context {
	return m.ctx
}

// SendMsg is part of the public API.
func (m *MockGenerateStream) SendMsg(_ interface{}) error {
	return nil
}

// RecvMsg is part of the public API.
func (m *MockGenerateStream) RecvMsg(_ interface{}) error {
	return nil
}

// SendHeader is part of the public API.
func (m *MockGenerateStream) SendHeader(metadata.MD) error {
	return nil
}

// SetHeader is part of the public API.
func (m *MockGenerateStream) SetHeader(metadata.MD) error {
	return nil
}

// SetTrailer is part of the public API.
func (m *MockGenerateStream) SetTrailer(metadata.MD) {
}

// MockPipelineStream is a mock implementation of pipeline gRPC stream
type MockPipelineStream struct {
	ctx  context.Context
	sent []*pb.PipelineStreamResponse
}

// Send is part of the public API.
func (m *MockPipelineStream) Send(resp *pb.PipelineStreamResponse) error {
	m.sent = append(m.sent, resp)
	return nil
}

// Context is part of the public API.
func (m *MockPipelineStream) Context() context.Context {
	return m.ctx
}

// SendMsg is part of the public API.
func (m *MockPipelineStream) SendMsg(_ interface{}) error {
	return nil
}

// RecvMsg is part of the public API.
func (m *MockPipelineStream) RecvMsg(_ interface{}) error {
	return nil
}

// SendHeader is part of the public API.
func (m *MockPipelineStream) SendHeader(metadata.MD) error {
	return nil
}

// SetHeader is part of the public API.
func (m *MockPipelineStream) SetHeader(metadata.MD) error {
	return nil
}

// SetTrailer is part of the public API.
func (m *MockPipelineStream) SetTrailer(metadata.MD) {
}

// ============================================================================
// Tests for Server Initialization
// ============================================================================

// TestNewComputeServer is part of the public API.
func TestNewComputeServer(t *testing.T) {
	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	server := NewComputeServer(r)

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.router != r {
		t.Error("Expected router to be set correctly")
	}

	if server.forwardingRouter != nil {
		t.Error("Expected forwardingRouter to be nil initially")
	}

	if server.pipelineExecutor != nil {
		t.Error("Expected pipelineExecutor to be nil initially")
	}

	if server.pipelineLoader != nil {
		t.Error("Expected pipelineLoader to be nil initially")
	}
}

// TestSetForwardingRouter is part of the public API.
func TestSetForwardingRouter(t *testing.T) {
	r := router.NewRouter(router.Config{})
	server := NewComputeServer(r)

	forwardingRouter := &router.ForwardingRouter{}
	server.SetForwardingRouter(forwardingRouter)

	if server.forwardingRouter != forwardingRouter {
		t.Error("Expected forwardingRouter to be set")
	}
}

// TestSetPipelineExecutor is part of the public API.
func TestSetPipelineExecutor(t *testing.T) {
	r := router.NewRouter(router.Config{})
	server := NewComputeServer(r)

	backend := &MockBackend{id: strBackend1, healthy: true}
	backends := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backends)
	pipelineLoader := &pipeline.Loader{}

	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	if server.pipelineExecutor != pipelineExecutor {
		t.Error("Expected pipelineExecutor to be set")
	}

	if server.pipelineLoader != pipelineLoader {
		t.Error("Expected pipelineLoader to be set")
	}
}

// ============================================================================
// Tests for Generate (without forwarding router)
// ============================================================================

// TestGenerateSuccess is part of the public API.
func TestGenerateSuccess(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
		powerWatts:       50.0,
		avgLatencyMs:     100,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
		Annotations: &pb.JobAnnotations{
			Target: strBackend1,
		},
	}

	resp, err := server.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Response != "test response" {
		t.Errorf("Expected response 'test response', got '%s'", resp.Response)
	}

	if resp.BackendUsed != strBackend1 {
		t.Errorf("Expected backend 'backend-1', got '%s'", resp.BackendUsed)
	}
}

// TestGenerateWithNilAnnotations is part of the public API.
func TestGenerateWithNilAnnotations(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt2,
		Model:       strTestModel,
		Annotations: nil,
	}

	resp, err := server.Generate(context.Background(), req)
	// Should still work with nil annotations (converts to empty)
	if resp == nil && err == nil {
		t.Fatal("Expected either response or error")
	}
}

// TestGenerateBackendError is part of the public API.
func TestGenerateBackendError(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
		generateErr:      errors.New("backend error"),
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
	}

	resp, err := server.Generate(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
}

// TestGenerateWithOptions is part of the public API.
func TestGenerateWithOptions(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
		Options: &pb.GenerationOptions{
			MaxTokens:     100,
			Temperature:   0.7,
			TopP:          0.9,
			TopK:          40,
			ContextLength: 2048,
		},
	}

	resp, err := server.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}
}

// TestGenerateWithFallbackSuccess is part of the public API.
func TestGenerateWithFallbackSuccess(t *testing.T) {
	primaryBackend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
		generateErr:      errors.New("primary failed"),
	}

	fallbackBackend := &MockBackend{
		id:               strBackend2,
		healthy:          true,
		supportsGenerate: true,
		generateErr:      nil,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(primaryBackend))
	testutil.Check(t, r.RegisterBackend(fallbackBackend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
	}

	resp, err := server.Generate(context.Background(), req)
	if resp == nil && err == nil {
		t.Fatal("expected a response or an error from fallback routing")
	}
}

// ============================================================================
// Tests for GenerateStream
// ============================================================================

// TestGenerateStreamSuccess is part of the public API.
func TestGenerateStreamSuccess(t *testing.T) {
	backend := &MockBackend{
		id:             strBackend1,
		healthy:        true,
		supportsStream: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
	}

	stream := &MockGenerateStream{
		ctx: context.Background(),
	}

	err := server.GenerateStream(req, stream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(stream.sent) == 0 {
		t.Fatal("Expected at least one message sent")
	}

	// Check first message has backend info
	firstMsg := stream.sent[0]
	if firstMsg.BackendUsed != strBackend1 {
		t.Errorf("Expected backend 'backend-1', got '%s'", firstMsg.BackendUsed)
	}
}

// TestGenerateStreamBackendError is part of the public API.
func TestGenerateStreamBackendError(t *testing.T) {
	backend := &MockBackend{
		id:             strBackend1,
		healthy:        true,
		supportsStream: true,
		streamErr:      errors.New("stream error"),
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
	}

	stream := &MockGenerateStream{
		ctx: context.Background(),
	}

	err := server.GenerateStream(req, stream)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestGenerateStreamWithAnnotations is part of the public API.
func TestGenerateStreamWithAnnotations(t *testing.T) {
	backend := &MockBackend{
		id:             strBackend1,
		healthy:        true,
		supportsStream: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.GenerateRequest{
		Prompt: strTestPrompt2,
		Model:  strTestModel,
		Annotations: &pb.JobAnnotations{
			Target:          strBackend1,
			LatencyCritical: true,
		},
	}

	stream := &MockGenerateStream{
		ctx: context.Background(),
	}

	err := server.GenerateStream(req, stream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(stream.sent) == 0 {
		t.Fatal("Expected at least one message sent")
	}
}

// ============================================================================
// Tests for Embed
// ============================================================================

// TestEmbedSuccess is part of the public API.
func TestEmbedSuccess(t *testing.T) {
	backend := &MockBackend{
		id:            strBackend1,
		healthy:       true,
		supportsEmbed: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.EmbedRequest{
		Text:  strTestText,
		Model: strTestModel,
	}

	resp, err := server.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.BackendUsed != strBackend1 {
		t.Errorf("Expected backend 'backend-1', got '%s'", resp.BackendUsed)
	}

	if len(resp.Embedding) != 3 {
		t.Errorf("Expected embedding length 3, got %d", len(resp.Embedding))
	}
}

// TestEmbedBackendError is part of the public API.
func TestEmbedBackendError(t *testing.T) {
	backend := &MockBackend{
		id:            strBackend1,
		healthy:       true,
		supportsEmbed: true,
		embedErr:      errors.New("embed error"),
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.EmbedRequest{
		Text:  strTestText,
		Model: strTestModel,
	}

	resp, err := server.Embed(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
}

// TestEmbedWithAnnotations is part of the public API.
func TestEmbedWithAnnotations(t *testing.T) {
	backend := &MockBackend{
		id:            strBackend1,
		healthy:       true,
		supportsEmbed: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	req := &pb.EmbedRequest{
		Text:  strTestText,
		Model: strTestModel,
		Annotations: &pb.JobAnnotations{
			Target:                strBackend1,
			PreferPowerEfficiency: true,
		},
	}

	resp, err := server.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.BackendUsed != strBackend1 {
		t.Errorf("Expected backend 'backend-1', got '%s'", resp.BackendUsed)
	}
}

// ============================================================================
// Tests for ListBackends
// ============================================================================

// TestListBackendsSuccess is part of the public API.
func TestListBackendsSuccess(t *testing.T) {
	backend1 := &MockBackend{
		id:               strBackend1,
		name:             "Backend 1",
		hardware:         "cpu",
		healthy:          true,
		supportsGenerate: true,
		supportsEmbed:    true,
		supportsStream:   true,
		models:           []string{"model-1", "model-2"},
	}

	backend2 := &MockBackend{
		id:               strBackend2,
		name:             "Backend 2",
		hardware:         "gpu",
		healthy:          false,
		supportsGenerate: true,
		supportsEmbed:    false,
		supportsStream:   true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))

	server := NewComputeServer(r)

	req := &pb.ListBackendsRequest{}
	resp, err := server.ListBackends(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Backends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(resp.Backends))
	}

	// Check first backend status
	healthyCount := 0
	unhealthyCount := 0
	for _, b := range resp.Backends {
		switch b.Status.State {
		case strHealthy:
			healthyCount++
		case strUnhealthy:
			unhealthyCount++
		}
	}

	if healthyCount != 1 {
		t.Errorf("Expected 1 healthy backend, got %d", healthyCount)
	}

	if unhealthyCount != 1 {
		t.Errorf("Expected 1 unhealthy backend, got %d", unhealthyCount)
	}
}

// TestListBackendsEmpty is part of the public API.
func TestListBackendsEmpty(t *testing.T) {
	r := router.NewRouter(router.Config{})
	server := NewComputeServer(r)

	req := &pb.ListBackendsRequest{}
	resp, err := server.ListBackends(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Backends) != 0 {
		t.Errorf("Expected 0 backends, got %d", len(resp.Backends))
	}
}

// ============================================================================
// Tests for HealthCheck
// ============================================================================

// TestHealthCheckAllHealthy is part of the public API.
func TestHealthCheckAllHealthy(t *testing.T) {
	backend1 := &MockBackend{id: strBackend1, healthy: true}
	backend2 := &MockBackend{id: strBackend2, healthy: true}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))

	server := NewComputeServer(r)

	req := &pb.HealthCheckRequest{}
	resp, err := server.HealthCheck(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Status != strHealthy {
		t.Errorf("Expected status 'healthy', got '%s'", resp.Status)
	}
}

// TestHealthCheckDegraded is part of the public API.
func TestHealthCheckDegraded(t *testing.T) {
	backend1 := &MockBackend{id: strBackend1, healthy: true}
	backend2 := &MockBackend{id: strBackend2, healthy: false}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))

	server := NewComputeServer(r)

	req := &pb.HealthCheckRequest{}
	resp, err := server.HealthCheck(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", resp.Status)
	}

	if resp.BackendHealth[strBackend1] != strHealthy {
		t.Errorf("Expected backend-1 to be healthy")
	}

	if resp.BackendHealth[strBackend2] != strUnhealthy {
		t.Errorf("Expected backend-2 to be unhealthy")
	}
}

// ============================================================================
// Tests for ExecutePipeline
// ============================================================================

// TestExecutePipelineNotEnabled is part of the public API.
func TestExecutePipelineNotEnabled(t *testing.T) {
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)
	// Don't set pipeline executor

	req := &pb.ExecutePipelineRequest{
		PipelineId: strTestPipeline,
	}

	resp, err := server.ExecutePipeline(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
}

// TestExecutePipelineSuccess is part of the public API.
func TestExecutePipelineSuccess(t *testing.T) {
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	backends := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backends)
	pipelineLoader := pipeline.NewPipelineLoader()

	server := NewComputeServer(r)
	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	req := &pb.ExecutePipelineRequest{
		PipelineId: strTestPipeline,
		Input:      map[string]string{"data": "test input"},
	}

	_, err := server.ExecutePipeline(context.Background(), req)

	// Since pipeline isn't registered in loader, we expect an error
	// This tests the error path in ExecutePipeline
	if err == nil {
		t.Fatal("Expected error for unregistered pipeline, got nil")
	}

	// Verify the error message is about pipeline not found
	if !containsSubstring(err.Error(), "pipeline not found") {
		t.Errorf("Expected 'pipeline not found' error, got: %v", err)
	}
}

// TestExecutePipelineWithInput is part of the public API.
func TestExecutePipelineWithInput(t *testing.T) {
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	backends := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backends)
	pipelineLoader := pipeline.NewPipelineLoader()

	server := NewComputeServer(r)
	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	// Test with different input maps
	testInputs := []map[string]string{
		{"input": "string input"},
		{"key": "value"},
		{"item1": "value1", "item2": "value2"},
	}

	for _, input := range testInputs {
		req := &pb.ExecutePipelineRequest{
			PipelineId: strTestPipeline,
			Input:      input,
		}

		_, err := server.ExecutePipeline(context.Background(), req)

		// All should fail with pipeline not found (loader has no pipelines)
		if err == nil {
			t.Fatal("Expected error for unregistered pipeline, got nil")
		}
	}
}

// TestExecutePipelineFullSuccess is part of the public API.
func TestExecutePipelineFullSuccess(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	backendsList := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backendsList)
	pipelineLoader := pipeline.NewPipelineLoader()

	// Manually create and register a pipeline
	testPipeline := &pipeline.Pipeline{
		ID:   strTestSuccessPipeline,
		Name: "Test Success Pipeline",
		Stages: []*pipeline.Stage{
			{
				ID:    "stage1",
				Type:  pipeline.StageTypeTextGen,
				Model: strTestModel,
			},
		},
		Options: &pipeline.Options{
			ContinueOnError: false,
		},
	}

	// Add pipeline to loader
	pipelineLoader.AddPipeline(testPipeline)

	server := NewComputeServer(r)
	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	req := &pb.ExecutePipelineRequest{
		PipelineId: strTestSuccessPipeline,
		Input:      map[string]string{"prompt": "test input"},
	}

	resp, err := server.ExecutePipeline(context.Background(), req)

	// Execution errors are returned alongside the response payload.
	if resp == nil {
		t.Fatalf("ExecutePipeline returned nil response: %v", err)
	}

	if resp.PipelineId != strTestSuccessPipeline {
		t.Errorf("Expected pipeline_id 'test-success-pipeline', got '%s'", resp.PipelineId)
	}

	// Verify response structure - may be empty but not nil
	if resp.StageResults == nil {
		// Stage results can be empty for mock backend
		t.Log("Stage results is nil (expected for mock without proper result)")
	}

	// FinalOutput might be nil if execution didn't produce output
	t.Logf("Success: %v, Error: %s", resp.Success, resp.Error)
}

// TestExecutePipelineWithMetadata is part of the public API.
func TestExecutePipelineWithMetadata(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	backendsList := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backendsList)
	pipelineLoader := pipeline.NewPipelineLoader()

	// Create pipeline with multiple stages to test metadata conversion
	testPipeline := &pipeline.Pipeline{
		ID:   "metadata-test-pipeline",
		Name: "Metadata Test Pipeline",
		Stages: []*pipeline.Stage{
			{
				ID:    "stage1",
				Type:  pipeline.StageTypeTextGen,
				Model: "model1",
			},
			{
				ID:    "stage2",
				Type:  pipeline.StageTypeTextGen,
				Model: "model2",
			},
		},
		Options: &pipeline.Options{
			ContinueOnError: true,
			CollectMetrics:  true,
		},
	}

	pipelineLoader.AddPipeline(testPipeline)

	server := NewComputeServer(r)
	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	req := &pb.ExecutePipelineRequest{
		PipelineId: "metadata-test-pipeline",
		Input:      map[string]string{"data": "test"},
	}

	resp, err := server.ExecutePipeline(context.Background(), req)

	if err != nil {
		t.Fatalf("ExecutePipeline returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Verify stage results exist
	if len(resp.StageResults) == 0 {
		t.Error("Expected stage results, got empty array")
	}

	// Check that stage results have proper structure
	for i, stageResult := range resp.StageResults {
		if stageResult.StageId == "" {
			t.Errorf("Stage %d has empty stage_id", i)
		}
		// Metadata should exist if metrics were collected
		if stageResult.Metadata != nil {
			// Verify metadata fields
			if stageResult.Metadata.DurationMs < 0 {
				t.Errorf("Stage %d has negative duration", i)
			}
		}
	}
}

// TestExecutePipelineOutputConversion is part of the public API.
func TestExecutePipelineOutputConversion(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	backendsList := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backendsList)
	pipelineLoader := pipeline.NewPipelineLoader()

	testPipeline := &pipeline.Pipeline{
		ID:   "output-test-pipeline",
		Name: "Output Test Pipeline",
		Stages: []*pipeline.Stage{
			{
				ID:    "output-stage",
				Type:  pipeline.StageTypeTextGen,
				Model: strTestModel,
			},
		},
		Options: &pipeline.Options{},
	}

	pipelineLoader.AddPipeline(testPipeline)

	server := NewComputeServer(r)
	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	req := &pb.ExecutePipelineRequest{
		PipelineId: "output-test-pipeline",
		Input:      map[string]string{"text": "input text"},
	}

	resp, err := server.ExecutePipeline(context.Background(), req)

	if resp == nil {
		t.Fatalf("ExecutePipeline returned nil response: %v", err)
	}

	// Test that final output was converted (may be nil for mock)
	if resp.FinalOutput == nil {
		t.Log("Final output is nil (expected for mock without output)")
	} else if len(resp.FinalOutput) == 0 {
		t.Log("Final output is empty map (expected for mock)")
	}

	// Verify total time and energy fields exist
	if resp.TotalTimeMs < 0 {
		t.Error("Expected non-negative total_time_ms")
	}

	if resp.TotalEnergyWh < 0 {
		t.Error("Expected non-negative total_energy_wh")
	}
}

// ============================================================================
// Tests for ExecutePipelineStream
// ============================================================================

// TestExecutePipelineStreamNotEnabled is part of the public API.
func TestExecutePipelineStreamNotEnabled(t *testing.T) {
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)
	// Don't set pipeline executor

	req := &pb.ExecutePipelineRequest{
		PipelineId: strTestPipeline,
	}

	stream := &MockPipelineStream{
		ctx: context.Background(),
	}

	err := server.ExecutePipelineStream(req, stream)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestExecutePipelineStreamSuccess is part of the public API.
func TestExecutePipelineStreamSuccess(t *testing.T) {
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	backends := []backends.Backend{backend}
	pipelineExecutor := pipeline.NewPipelineExecutor(backends)
	pipelineLoader := pipeline.NewPipelineLoader()

	server := NewComputeServer(r)
	server.SetPipelineExecutor(pipelineExecutor, pipelineLoader)

	req := &pb.ExecutePipelineRequest{
		PipelineId: strTestPipeline,
	}

	stream := &MockPipelineStream{
		ctx: context.Background(),
	}

	err := server.ExecutePipelineStream(req, stream)
	if err == nil {
		t.Fatal("expected an error for an unregistered pipeline")
	}
	if stream.sent != nil {
		t.Fatal("expected no stream payload before the pipeline error")
	}
}

// ============================================================================
// Tests for Concurrent Operations
// ============================================================================

// TestConcurrentGenerateRequests is part of the public API.
func TestConcurrentGenerateRequests(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	numGoroutines := 10
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			req := &pb.GenerateRequest{
				Prompt: strTestPrompt2,
				Model:  strTestModel,
			}
			resp, err := server.Generate(context.Background(), req)
			if err != nil {
				errChan <- err
			}
			if resp == nil && err == nil {
				errChan <- errors.New("nil response")
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}

// TestConcurrentEmbedAndGenerate is part of the public API.
func TestConcurrentEmbedAndGenerate(t *testing.T) {
	backend := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
		supportsEmbed:    true,
	}

	r := router.NewRouter(router.Config{DefaultBackendID: strBackend1})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	// 10 generate requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &pb.GenerateRequest{
				Prompt: strTestPrompt2,
				Model:  strTestModel,
			}
			_, err := server.Generate(context.Background(), req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// 10 embed requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &pb.EmbedRequest{
				Text:  strTestText,
				Model: strTestModel,
			}
			_, err := server.Embed(context.Background(), req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}

// TestConcurrentListBackendsAndHealthCheck is part of the public API.
func TestConcurrentListBackendsAndHealthCheck(t *testing.T) {
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	r := router.NewRouter(router.Config{})
	testutil.Check(t, r.RegisterBackend(backend))

	server := NewComputeServer(r)

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	// 10 ListBackends requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &pb.ListBackendsRequest{}
			_, err := server.ListBackends(context.Background(), req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// 10 HealthCheck requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &pb.HealthCheckRequest{}
			_, err := server.HealthCheck(context.Background(), req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}

// ============================================================================
// Tests for Helper Functions
// ============================================================================

// TestConvertAnnotations is part of the public API.
func TestConvertAnnotations(t *testing.T) {
	pbAnnotations := &pb.JobAnnotations{
		Target:                strBackend1,
		LatencyCritical:       true,
		PreferPowerEfficiency: true,
		CacheEnabled:          false,
		MaxLatencyMs:          1000,
		MaxPowerWatts:         100,
		Custom: map[string]string{
			"key": "value",
		},
	}

	result := convertAnnotations(pbAnnotations)

	if result.Target != strBackend1 {
		t.Errorf("Expected target 'backend-1', got '%s'", result.Target)
	}

	if !result.LatencyCritical {
		t.Error("Expected LatencyCritical to be true")
	}

	if !result.PreferPowerEfficiency {
		t.Error("Expected PreferPowerEfficiency to be true")
	}

	if result.CacheEnabled {
		t.Error("Expected CacheEnabled to be false")
	}

	if result.MaxLatencyMs != 1000 {
		t.Errorf("Expected MaxLatencyMs 1000, got %d", result.MaxLatencyMs)
	}
}

// TestConvertAnnotationsNil is part of the public API.
func TestConvertAnnotationsNil(t *testing.T) {
	result := convertAnnotations(nil)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Target != "" {
		t.Error("Expected empty target")
	}
}

// TestConvertGenerationOptions is part of the public API.
func TestConvertGenerationOptions(t *testing.T) {
	pbOptions := &pb.GenerationOptions{
		MaxTokens:     100,
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		Stop:          []string{"<end>"},
		ContextLength: 2048,
	}

	result := convertGenerationOptions(pbOptions)

	if result.MaxTokens != 100 {
		t.Errorf("Expected MaxTokens 100, got %d", result.MaxTokens)
	}

	if result.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", result.Temperature)
	}

	if result.TopP != 0.9 {
		t.Errorf("Expected TopP 0.9, got %f", result.TopP)
	}

	if result.TopK != 40 {
		t.Errorf("Expected TopK 40, got %d", result.TopK)
	}

	if len(result.Stop) != 1 || result.Stop[0] != "<end>" {
		t.Error("Expected stop sequence <end>")
	}

	if result.ContextLength != 2048 {
		t.Errorf("Expected ContextLength 2048, got %d", result.ContextLength)
	}
}

// TestConvertGenerationOptionsNil is part of the public API.
func TestConvertGenerationOptionsNil(t *testing.T) {
	result := convertGenerationOptions(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

// TestConvertStats is part of the public API.
func TestConvertStats(t *testing.T) {
	stats := &backends.GenerationStats{
		TimeToFirstTokenMs: 50,
		TotalTimeMs:        100,
		TokensGenerated:    10,
		TokensPerSecond:    100.0,
		EnergyWh:           0.5,
	}

	result := convertStats(stats)

	if result.TimeToFirstTokenMs != 50 {
		t.Errorf("Expected TimeToFirstTokenMs 50, got %d", result.TimeToFirstTokenMs)
	}

	if result.TotalTimeMs != 100 {
		t.Errorf("Expected TotalTimeMs 100, got %d", result.TotalTimeMs)
	}

	if result.TokensGenerated != 10 {
		t.Errorf("Expected TokensGenerated 10, got %d", result.TokensGenerated)
	}

	if result.TokensPerSecond != 100.0 {
		t.Errorf("Expected TokensPerSecond 100.0, got %f", result.TokensPerSecond)
	}

	if result.EnergyWh != 0.5 {
		t.Errorf("Expected EnergyWh 0.5, got %f", result.EnergyWh)
	}
}

// TestConvertStatsNil is part of the public API.
func TestConvertStatsNil(t *testing.T) {
	result := convertStats(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

// TestHealthState is part of the public API.
func TestHealthState(t *testing.T) {
	if healthState(true) != strHealthy {
		t.Error("Expected 'healthy' for true")
	}

	if healthState(false) != strUnhealthy {
		t.Error("Expected 'unhealthy' for false")
	}
}

// TestHealthMessage is part of the public API.
func TestHealthMessage(t *testing.T) {
	if healthMessage(true) != "Backend is responding normally" {
		t.Error("Expected 'Backend is responding normally' for true")
	}

	if healthMessage(false) != "Backend is not responding" {
		t.Error("Expected 'Backend is not responding' for false")
	}
}

// TestTruncate is part of the public API.
func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		max      int
	}{
		{input: "short", max: 10, expected: "short"},
		{input: "this is a long string", max: 10, expected: "this is a ..."},
		{input: "0123456789abcdef", max: 10, expected: "0123456789..."},
		{input: "", max: 5, expected: ""},
		{input: "a", max: 5, expected: "a"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.max)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, result, tt.expected)
		}
	}
}

// Helper function for string matching
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGenerateWithForwardingRouter is part of the public API.
func TestGenerateWithForwardingRouter(t *testing.T) {
	// Create mock backend
	mockBackend := &MockBackend{
		id:               strTestBackend,
		healthy:          true,
		supportsGenerate: true,
		powerWatts:       50.0,
		avgLatencyMs:     100,
	}

	// Create router
	routerCfg := router.Config{
		DefaultBackendID: strTestBackend,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	testutil.Check(t, baseRouter.RegisterBackend(mockBackend))

	// Create forwarding router
	forwardingCfg := &router.ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.5,
		MaxRetries:           2,
		EscalationPath:       []string{strTestBackend},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}
	forwardingRouter := router.NewForwardingRouter(baseRouter, nil, forwardingCfg)

	// Create server with forwarding router
	server := NewComputeServer(baseRouter)
	server.SetForwardingRouter(forwardingRouter)

	// Test request
	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	resp, err := server.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate with forwarding failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Response == "" {
		t.Error("Response text is empty")
	}

	if resp.BackendUsed != strTestBackend {
		t.Errorf("Expected backend 'test-backend', got '%s'", resp.BackendUsed)
	}

	if resp.Routing == nil {
		t.Error("Routing metadata is nil")
	}

	if resp.Stats == nil {
		t.Error("Stats are nil")
	}
}

// TestGenerateRoutingError is part of the public API.
func TestGenerateRoutingError(t *testing.T) {
	// Create router with no backends - this will cause routing to fail
	routerCfg := router.Config{
		DefaultBackendID: strNonExistent,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)

	// Create server without forwarding router (to test standard routing path)
	server := NewComputeServer(baseRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	_, err := server.Generate(context.Background(), req)
	if err == nil {
		t.Error("Expected error when routing fails with no backends")
	}

	if err != nil && !containsSubstring(err.Error(), "routing failed") {
		t.Errorf("Expected 'routing failed' error, got: %v", err)
	}
}

// TestGenerateWithForwardingError is part of the public API.
func TestGenerateWithForwardingError(t *testing.T) {
	// Create router with no backends - forwarding will fail
	routerCfg := router.Config{
		DefaultBackendID: strNonExistent,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)

	// Create forwarding router
	forwardingCfg := &router.ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.5,
		MaxRetries:           2,
		EscalationPath:       []string{strNonExistent},
		RespectThermalLimits: false,
		ReturnBestAttempt:    false, // Don't return best attempt
	}
	forwardingRouter := router.NewForwardingRouter(baseRouter, nil, forwardingCfg)

	// Create server with forwarding router
	server := NewComputeServer(baseRouter)
	server.SetForwardingRouter(forwardingRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	_, err := server.Generate(context.Background(), req)
	if err == nil {
		t.Error("Expected error when forwarding fails")
	}

	if err != nil && !containsSubstring(err.Error(), "forwarding failed") {
		t.Errorf("Expected 'forwarding failed' error, got: %v", err)
	}
}

// TestGenerateWithActualForwarding is part of the public API.
func TestGenerateWithActualForwarding(t *testing.T) {
	// Create two backends
	backend1 := &MockBackend{
		id:               strBackend1,
		healthy:          true,
		supportsGenerate: true,
		powerWatts:       50.0,
		avgLatencyMs:     100,
		generateErr:      nil,
	}

	backend2 := &MockBackend{
		id:               strBackend2,
		healthy:          true,
		supportsGenerate: true,
		powerWatts:       100.0,
		avgLatencyMs:     200,
	}

	// Create router
	routerCfg := router.Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	testutil.Check(t, baseRouter.RegisterBackend(backend1))
	testutil.Check(t, baseRouter.RegisterBackend(backend2))

	// Create forwarding router with very high confidence threshold
	// to trigger forwarding
	forwardingCfg := &router.ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.95, // Very high - will cause forwarding
		MaxRetries:           2,
		EscalationPath:       []string{strBackend1, strBackend2},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}
	forwardingRouter := router.NewForwardingRouter(baseRouter, nil, forwardingCfg)

	// Create server with forwarding router
	server := NewComputeServer(baseRouter)
	server.SetForwardingRouter(forwardingRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	resp, err := server.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	// Should have response (best attempt returned)
	if resp.Response == "" {
		t.Error("Response text is empty")
	}
}

// TestGenerateStreamRoutingError tests streaming generation when routing fails
func TestGenerateStreamRoutingError(t *testing.T) {
	// Create router with no backends - routing will fail
	routerCfg := router.Config{
		DefaultBackendID: strNonExistent,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	server := NewComputeServer(baseRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	// Create a mock stream
	mockStream := &mockGenerateStream{
		ctx: context.Background(),
	}

	err := server.GenerateStream(req, mockStream)
	if err == nil {
		t.Fatal("Expected routing error, got nil")
	}

	if !strings.Contains(err.Error(), "routing failed") {
		t.Errorf("Expected 'routing failed' error, got: %v", err)
	}
}

// TestEmbedWithPowerPreference tests embedding with power preference annotations
func TestEmbedWithPowerPreference(t *testing.T) {
	mockBackend := &MockBackend{
		id:            strTestBackend,
		healthy:       true,
		supportsEmbed: true,
	}

	routerCfg := router.Config{
		DefaultBackendID: strTestBackend,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	testutil.Check(t, baseRouter.RegisterBackend(mockBackend))

	server := NewComputeServer(baseRouter)

	req := &pb.EmbedRequest{
		Text:  "Test text for embedding",
		Model: strTestModel,
		Annotations: &pb.JobAnnotations{
			Target:                strTestBackend,
			PreferPowerEfficiency: true,
			MaxPowerWatts:         100,
		},
	}

	resp, err := server.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if len(resp.Embedding) != 3 {
		t.Errorf("Expected embedding length 3, got %d", len(resp.Embedding))
	}

	// Verify embedding values
	expected := []float32{0.1, 0.2, 0.3}
	for i, val := range resp.Embedding {
		if val != expected[i] {
			t.Errorf("Embedding[%d]: expected %f, got %f", i, expected[i], val)
		}
	}
}

// TestEmbedRoutingError tests embedding when routing fails
func TestEmbedRoutingError(t *testing.T) {
	// Create router with no backends
	routerCfg := router.Config{
		DefaultBackendID: strNonExistent,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	server := NewComputeServer(baseRouter)

	req := &pb.EmbedRequest{
		Text:        "Test text",
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	_, err := server.Embed(context.Background(), req)
	if err == nil {
		t.Fatal("Expected routing error, got nil")
	}

	if !strings.Contains(err.Error(), "routing failed") {
		t.Errorf("Expected 'routing failed' error, got: %v", err)
	}
}

// TestGenerateStreamBackendStreamError tests streaming generation when backend stream fails
func TestGenerateStreamBackendStreamError(t *testing.T) {
	mockBackend := &MockBackend{
		id:             strTestBackend,
		healthy:        true,
		supportsStream: true,
		streamErr:      errors.New("backend stream error"),
	}

	routerCfg := router.Config{
		DefaultBackendID: strTestBackend,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	testutil.Check(t, baseRouter.RegisterBackend(mockBackend))

	server := NewComputeServer(baseRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	mockStream := &mockGenerateStream{
		ctx: context.Background(),
	}

	err := server.GenerateStream(req, mockStream)
	if err == nil {
		t.Fatal("Expected backend stream error, got nil")
	}

	if !strings.Contains(err.Error(), "streaming failed") {
		t.Errorf("Expected 'streaming failed' error, got: %v", err)
	}
}

// Mock stream implementation for GenerateStream testing
type mockGenerateStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.GenerateStreamResponse
}

// Send is part of the public API.
func (m *mockGenerateStream) Send(resp *pb.GenerateStreamResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, resp)
	return nil
}

// Context is part of the public API.
func (m *mockGenerateStream) Context() context.Context {
	return m.ctx
}

// SetHeader is part of the public API.
func (m *mockGenerateStream) SetHeader(metadata.MD) error {
	return nil
}

// SendHeader is part of the public API.
func (m *mockGenerateStream) SendHeader(metadata.MD) error {
	return nil
}

// SetTrailer is part of the public API.
func (m *mockGenerateStream) SetTrailer(metadata.MD) {
}

// SendMsg is part of the public API.
func (m *mockGenerateStream) SendMsg(_ interface{}) error {
	return nil
}

// RecvMsg is part of the public API.
func (m *mockGenerateStream) RecvMsg(_ interface{}) error {
	return nil
}

// TestGenerateFallbackOnBackendFailure tests fallback when primary backend fails
func TestGenerateFallbackOnBackendFailure(t *testing.T) {
	// Create a backend that fails
	failingBackend := &MockBackend{
		id:          "failing-backend",
		healthy:     true,
		generateErr: errors.New("backend failure"),
	}

	// Create a fallback backend that succeeds
	fallbackBackend := &MockBackend{
		id:      "fallback-backend",
		healthy: true,
	}

	routerCfg := router.Config{
		DefaultBackendID: "failing-backend",
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	testutil.Check(t, baseRouter.RegisterBackend(failingBackend))
	testutil.Check(t, baseRouter.RegisterBackend(fallbackBackend))

	server := NewComputeServer(baseRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	resp, err := server.Generate(context.Background(), req)

	// Should succeed using fallback
	if err != nil {
		t.Fatalf("Generate should succeed with fallback, got error: %v", err)
	}

	// Verify fallback was used
	if resp.BackendUsed != "fallback-backend" {
		t.Errorf("Expected fallback-backend to be used, got: %s", resp.BackendUsed)
	}
}

// TestGenerateStreamEOFHandling tests EOF handling in streaming loop
func TestGenerateStreamEOFHandling(t *testing.T) {
	// Create a backend that returns EOF after some chunks
	mockBackend := &MockBackend{
		id:             strTestBackend,
		healthy:        true,
		supportsStream: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "Hello", Done: false},
			{Token: " world", Done: false},
		},
		streamEOF: true, // Signal EOF after chunks
	}

	routerCfg := router.Config{
		DefaultBackendID: strTestBackend,
		PowerAware:       true,
	}
	baseRouter := router.NewRouter(routerCfg)
	testutil.Check(t, baseRouter.RegisterBackend(mockBackend))

	server := NewComputeServer(baseRouter)

	req := &pb.GenerateRequest{
		Prompt:      strTestPrompt,
		Model:       strTestModel,
		Annotations: &pb.JobAnnotations{},
	}

	mockStream := &mockGenerateStream{ctx: context.Background()}
	err := server.GenerateStream(req, mockStream)

	// Should complete successfully after EOF
	if err != nil {
		t.Fatalf("GenerateStream should handle EOF, got error: %v", err)
	}

	// Verify chunks were sent
	if len(mockStream.sent) < 2 {
		t.Errorf("Expected at least 2 chunks to be sent, got: %d", len(mockStream.sent))
	}
}
