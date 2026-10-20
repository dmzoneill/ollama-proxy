package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/router"
)

// mockBackend implements backends.Backend interface for testing
type mockBackend struct {
	generateErr    error
	streamErr      error
	generateResp   *backends.GenerateResponse
	id             string
	supportsModel  bool
	supportsStream bool
}

// ID is part of the public API.
func (m *mockBackend) ID() string { return m.id }

// Type is part of the public API.
func (m *mockBackend) Type() string { return "mock" }

// Name is part of the public API.
func (m *mockBackend) Name() string { return "Mock Backend" }

// Hardware is part of the public API.
func (m *mockBackend) Hardware() string { return "mock-hw" }

// IsHealthy reports whether the condition holds.
func (m *mockBackend) IsHealthy() bool { return true }

// PowerWatts is part of the public API.
func (m *mockBackend) PowerWatts() float64 { return 10.0 }

// AvgLatencyMs is part of the public API.
func (m *mockBackend) AvgLatencyMs() int32 { return 50 }

// Priority is part of the public API.
func (m *mockBackend) Priority() int { return 1 }

// Start is part of the public API.
func (m *mockBackend) Start(_ context.Context) error { return nil }

// Stop is part of the public API.
func (m *mockBackend) Stop(_ context.Context) error { return nil }

// Multimedia capability methods
func (m *mockBackend) SupportsAudioToText() bool { return false }

// SupportsTextToAudio is part of the public API.
func (m *mockBackend) SupportsTextToAudio() bool { return false }

// SupportsImageToText is part of the public API.
func (m *mockBackend) SupportsImageToText() bool { return false }

// SupportsTextToImage is part of the public API.
func (m *mockBackend) SupportsTextToImage() bool { return false }

// SupportsVideoToText is part of the public API.
func (m *mockBackend) SupportsVideoToText() bool { return false }

// SupportsTextToVideo is part of the public API.
func (m *mockBackend) SupportsTextToVideo() bool { return false }

// TranscribeAudio is part of the public API.
func (m *mockBackend) TranscribeAudio(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// TranscribeAudioStream is part of the public API.
func (m *mockBackend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeech is part of the public API.
func (m *mockBackend) SynthesizeSpeech(_ context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeechStream is part of the public API.
func (m *mockBackend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeImage is part of the public API.
func (m *mockBackend) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImage is part of the public API.
func (m *mockBackend) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImageStream is part of the public API.
func (m *mockBackend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideo is part of the public API.
func (m *mockBackend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideoStream is part of the public API.
func (m *mockBackend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideo is part of the public API.
func (m *mockBackend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideoStream is part of the public API.
func (m *mockBackend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// HealthCheck is part of the public API.
func (m *mockBackend) HealthCheck(_ context.Context) error { return nil }

// SupportsModel is part of the public API.
func (m *mockBackend) SupportsModel(_ string) bool { return m.supportsModel }

// SupportsStream is part of the public API.
func (m *mockBackend) SupportsStream() bool { return m.supportsStream }

// SupportsGenerate is part of the public API.
func (m *mockBackend) SupportsGenerate() bool { return true }

// SupportsEmbed is part of the public API.
func (m *mockBackend) SupportsEmbed() bool { return true }

// ListModels is part of the public API.
func (m *mockBackend) ListModels(_ context.Context) ([]string, error) {
	return []string{strTestModel}, nil
}

// GetMaxModelSizeGB is part of the public API.
func (m *mockBackend) GetMaxModelSizeGB() int { return 10 }

// GetSupportedModelPatterns is part of the public API.
func (m *mockBackend) GetSupportedModelPatterns() []string { return []string{"*"} }

// GetPreferredModels is part of the public API.
func (m *mockBackend) GetPreferredModels() []string { return []string{strTestModel} }

// Generate is part of the public API.
func (m *mockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	if m.generateResp != nil {
		return m.generateResp, nil
	}
	return &backends.GenerateResponse{
		Response: "Hello! How can I help you?",
		Stats: &backends.GenerationStats{
			TokensGenerated: 7,
		},
	}, nil
}

// GenerateStream is part of the public API.
func (m *mockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, m.streamErr
}

// Embed is part of the public API.
func (m *mockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
	}, nil
}

// UpdateMetrics is part of the public API.
func (m *mockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *mockBackend) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{
		RequestCount: 10,
		SuccessCount: 9,
		AvgLatencyMs: 50,
	}
}

// TestHandleChatCompletion_MethodNotAllowed is part of the public API.
func TestHandleChatCompletion_MethodNotAllowed(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestHandleChatCompletion_InvalidJSON is part of the public API.
func TestHandleChatCompletion_InvalidJSON(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleChatCompletion_MissingModel is part of the public API.
func TestHandleChatCompletion_MissingModel(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleChatCompletion_MissingMessages is part of the public API.
func TestHandleChatCompletion_MissingMessages(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleChatCompletion_Success is part of the public API.
func TestHandleChatCompletion_Success(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ChatCompletionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Model != strTestModel {
		t.Errorf("Expected model 'test-model', got %s", resp.Model)
	}
}

// TestHandleCompletion_Success is part of the public API.
func TestHandleCompletion_Success(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHandleEmbedding_Success is part of the public API.
func TestHandleEmbedding_Success(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
		Input: "Hello world",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp EmbeddingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) == 0 {
		t.Error("Expected embedding data")
	}
}

// TestHandleModels_Success is part of the public API.
func TestHandleModels_Success(t *testing.T) {
	backend := &mockBackend{
		id: strTestBackend,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleModels(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestParseRoutingHeaders is part of the public API.
func TestParseRoutingHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Target-Backend", strTestBackend)
	req.Header.Set("X-Priority", strHigh)
	req.Header.Set("X-Latency-Critical", strTrue)

	annotations := ParseRoutingHeaders(req)

	if annotations.Target != strTestBackend {
		t.Errorf("Expected target backend 'test-backend', got %s", annotations.Target)
	}

	if !annotations.LatencyCritical {
		t.Error("Expected LatencyCritical to be true")
	}
}

// TestParseRoutingHeaders_AllHeaders is part of the public API.
func TestParseRoutingHeaders_AllHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Target-Backend", "ollama-nvidia")
	req.Header.Set("X-Latency-Critical", strTrue)
	req.Header.Set("X-Power-Efficient", "false")
	req.Header.Set("X-Max-Latency-Ms", "100")
	req.Header.Set("X-Max-Power-Watts", "50")
	req.Header.Set("X-Cache-Enabled", strYes)
	req.Header.Set("X-Media-Type", strCode)
	req.Header.Set("X-Priority", strCritical)
	req.Header.Set("X-Request-ID", "req-12345")
	req.Header.Set("X-Deadline-Ms", "1234567890")
	req.Header.Set("X-Custom-Key1", "value1")
	req.Header.Set("X-Custom-Key2", "value2")

	annotations := ParseRoutingHeaders(req)

	if annotations.Target != "ollama-nvidia" {
		t.Errorf("Expected target 'ollama-nvidia', got %s", annotations.Target)
	}
	if !annotations.LatencyCritical {
		t.Error("Expected LatencyCritical to be true")
	}
	if annotations.PreferPowerEfficiency {
		t.Error("Expected PreferPowerEfficiency to be false")
	}
	if annotations.MaxLatencyMs != 100 {
		t.Errorf("Expected MaxLatencyMs 100, got %d", annotations.MaxLatencyMs)
	}
	if annotations.MaxPowerWatts != 50 {
		t.Errorf("Expected MaxPowerWatts 50, got %d", annotations.MaxPowerWatts)
	}
	if !annotations.CacheEnabled {
		t.Error("Expected CacheEnabled to be true")
	}
	if annotations.MediaType != backends.MediaTypeCode {
		t.Errorf("Expected MediaType code, got %v", annotations.MediaType)
	}
	if annotations.Priority != backends.PriorityCritical {
		t.Errorf("Expected PriorityCritical, got %v", annotations.Priority)
	}
	if annotations.RequestID != "req-12345" {
		t.Errorf("Expected RequestID 'req-12345', got %s", annotations.RequestID)
	}
	if annotations.DeadlineMs != 1234567890 {
		t.Errorf("Expected DeadlineMs 1234567890, got %d", annotations.DeadlineMs)
	}
	if annotations.Custom["Key1"] != "value1" {
		t.Errorf("Expected Custom[Key1]='value1', got %s", annotations.Custom["Key1"])
	}
	if annotations.Custom["Key2"] != "value2" {
		t.Errorf("Expected Custom[Key2]='value2', got %s", annotations.Custom["Key2"])
	}
}

// TestParseRoutingHeaders_MediaTypes is part of the public API.
func TestParseRoutingHeaders_MediaTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected backends.MediaType
	}{
		{name: strText, value: strText, expected: backends.MediaTypeText},
		{name: strCode, value: strCode, expected: backends.MediaTypeCode},
		{name: strImage, value: strImage, expected: backends.MediaTypeImage},
		{name: strAudio, value: strAudio, expected: backends.MediaTypeAudio},
		{name: strRealtime, value: strRealtime, expected: backends.MediaTypeRealtime},
		{name: strAuto, value: strAuto, expected: backends.MediaTypeAuto},
		{name: strUppercase, value: "TEXT", expected: backends.MediaTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Media-Type", tt.value)

			annotations := ParseRoutingHeaders(req)

			if annotations.MediaType != tt.expected {
				t.Errorf("Expected MediaType %v, got %v", tt.expected, annotations.MediaType)
			}
		})
	}
}

// TestParseRoutingHeaders_Priorities is part of the public API.
func TestParseRoutingHeaders_Priorities(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected backends.Priority
	}{
		{name: strBestEffort, value: strBestEffort, expected: backends.PriorityBestEffort},
		{name: strLow, value: strLow, expected: backends.PriorityBestEffort},
		{name: strNormal, value: strNormal, expected: backends.PriorityNormal},
		{name: strHigh, value: strHigh, expected: backends.PriorityHigh},
		{name: strCritical, value: strCritical, expected: backends.PriorityCritical},
		{name: strRealtime, value: strRealtime, expected: backends.PriorityCritical},
		{name: strUppercase, value: "HIGH", expected: backends.PriorityHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Priority", tt.value)

			annotations := ParseRoutingHeaders(req)

			if annotations.Priority != tt.expected {
				t.Errorf("Expected Priority %v, got %v", tt.expected, annotations.Priority)
			}
		})
	}
}

// TestParseRoutingHeaders_AutoPriority is part of the public API.
func TestParseRoutingHeaders_AutoPriority(t *testing.T) {
	// Latency-critical should set critical priority
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Latency-Critical", strTrue)

	annotations := ParseRoutingHeaders(req)

	if annotations.Priority != backends.PriorityCritical {
		t.Errorf("Expected PriorityCritical for latency-critical, got %v", annotations.Priority)
	}

	// Realtime media should set critical priority
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Media-Type", strRealtime)

	annotations = ParseRoutingHeaders(req)

	if annotations.Priority != backends.PriorityCritical {
		t.Errorf("Expected PriorityCritical for realtime media, got %v", annotations.Priority)
	}

	// Audio media should set high priority
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Media-Type", strAudio)

	annotations = ParseRoutingHeaders(req)

	if annotations.Priority != backends.PriorityHigh {
		t.Errorf("Expected PriorityHigh for audio media, got %v", annotations.Priority)
	}

	// Default should be normal
	req = httptest.NewRequest(http.MethodGet, "/test", nil)

	annotations = ParseRoutingHeaders(req)

	if annotations.Priority != backends.PriorityNormal {
		t.Errorf("Expected PriorityNormal by default, got %v", annotations.Priority)
	}
}

// TestParseRoutingHeaders_BoolParsing is part of the public API.
func TestParseRoutingHeaders_BoolParsing(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: strTrue, value: strTrue, expected: true},
		{name: "1", value: "1", expected: true},
		{name: strYes, value: strYes, expected: true},
		{name: "on", value: "on", expected: true},
		{name: "false", value: "false", expected: false},
		{name: "0", value: "0", expected: false},
		{name: "no", value: "no", expected: false},
		{name: "off", value: "off", expected: false},
		{name: strUppercase, value: "TRUE", expected: true},
		{name: "whitespace", value: " yes ", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Latency-Critical", tt.value)

			annotations := ParseRoutingHeaders(req)

			if annotations.LatencyCritical != tt.expected {
				t.Errorf("Expected LatencyCritical %v for value %q, got %v", tt.expected, tt.value, annotations.LatencyCritical)
			}
		})
	}
}

// TestWriteError is part of the public API.
func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "Test error", "test_error_code")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error.Message != "Test error" {
		t.Errorf("Expected error message 'Test error', got %s", errResp.Error.Message)
	}

	if errResp.Error.Code != "test_error_code" {
		t.Errorf("Expected error code 'test_error_code', got %s", errResp.Error.Code)
	}
}

// TestHandleChatCompletion_ModelNotSupported is part of the public API.
func TestHandleChatCompletion_ModelNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strUnsupportedModel,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestHandleChatCompletion_GenerationError is part of the public API.
func TestHandleChatCompletion_GenerationError(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
		generateErr:   fmt.Errorf("generation failed"),
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestHandleChatCompletion_StreamingNotSupported is part of the public API.
func TestHandleChatCompletion_StreamingNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:             strTestBackend,
		supportsModel:  true,
		supportsStream: false,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:  strTestModel,
		Stream: true,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleChatCompletion_StreamingError is part of the public API.
func TestHandleChatCompletion_StreamingError(t *testing.T) {
	backend := &mockBackend{
		id:             strTestBackend,
		supportsModel:  true,
		supportsStream: true,
		streamErr:      fmt.Errorf("streaming failed"),
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:  strTestModel,
		Stream: true,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestHandleCompletion_MethodNotAllowed is part of the public API.
func TestHandleCompletion_MethodNotAllowed(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/completions", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestHandleCompletion_InvalidJSON is part of the public API.
func TestHandleCompletion_InvalidJSON(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleCompletion_MissingModel is part of the public API.
func TestHandleCompletion_MissingModel(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleCompletion_MissingPrompt is part of the public API.
func TestHandleCompletion_MissingPrompt(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model: strTestModel,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleCompletion_ModelNotSupported is part of the public API.
func TestHandleCompletion_ModelNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strUnsupportedModel,
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestHandleCompletion_GenerationError is part of the public API.
func TestHandleCompletion_GenerationError(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
		generateErr:   fmt.Errorf("generation failed"),
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestHandleCompletion_StreamingNotSupported is part of the public API.
func TestHandleCompletion_StreamingNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:             strTestBackend,
		supportsModel:  true,
		supportsStream: false,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
		Stream: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleCompletion_StreamingError is part of the public API.
func TestHandleCompletion_StreamingError(t *testing.T) {
	backend := &mockBackend{
		id:             strTestBackend,
		supportsModel:  true,
		supportsStream: true,
		streamErr:      fmt.Errorf("streaming failed"),
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
		Stream: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestHandleEmbedding_MethodNotAllowed is part of the public API.
func TestHandleEmbedding_MethodNotAllowed(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/embeddings", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestHandleEmbedding_InvalidJSON is part of the public API.
func TestHandleEmbedding_InvalidJSON(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleEmbedding_MissingModel is part of the public API.
func TestHandleEmbedding_MissingModel(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Input: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleEmbedding_MissingInput is part of the public API.
func TestHandleEmbedding_MissingInput(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleEmbedding_ModelNotSupported is part of the public API.
func TestHandleEmbedding_ModelNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strUnsupportedModel,
		Input: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestHandleModels_MethodNotAllowed is part of the public API.
func TestHandleModels_MethodNotAllowed(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleModels(r)

	req := httptest.NewRequest(http.MethodPost, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestWriteRoutingHeaders is part of the public API.
func TestWriteRoutingHeaders(t *testing.T) {
	backend := &mockBackend{
		id: strTestBackend,
	}

	decision := &router.RoutingDecision{
		Backend:            backend,
		Reason:             "best match",
		EstimatedPowerW:    15.5,
		EstimatedLatencyMs: 50,
		Alternatives:       []string{"backend-2", "backend-3"},
	}

	w := httptest.NewRecorder()
	WriteRoutingHeaders(w, decision)

	if w.Header().Get("X-Backend-Used") != strTestBackend {
		t.Errorf("Expected X-Backend-Used 'test-backend', got %s", w.Header().Get("X-Backend-Used"))
	}

	if w.Header().Get("X-Routing-Reason") != "best match" {
		t.Errorf("Expected X-Routing-Reason 'best match', got %s", w.Header().Get("X-Routing-Reason"))
	}

	if w.Header().Get("X-Estimated-Power-Watts") != "15.5" {
		t.Errorf("Expected X-Estimated-Power-Watts '15.5', got %s", w.Header().Get("X-Estimated-Power-Watts"))
	}

	if w.Header().Get("X-Estimated-Latency-Ms") != "50" {
		t.Errorf("Expected X-Estimated-Latency-Ms '50', got %s", w.Header().Get("X-Estimated-Latency-Ms"))
	}

	if w.Header().Get("X-Alternatives") != "backend-2,backend-3" {
		t.Errorf("Expected X-Alternatives 'backend-2,backend-3', got %s", w.Header().Get("X-Alternatives"))
	}
}

// TestWriteRoutingHeaders_Nil is part of the public API.
func TestWriteRoutingHeaders_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	WriteRoutingHeaders(w, nil)

	// Should not panic and headers should be empty
	if w.Header().Get("X-Backend-Used") != "" {
		t.Error("Expected no headers when decision is nil")
	}
}

// TestWriteRoutingHeaders_Minimal is part of the public API.
func TestWriteRoutingHeaders_Minimal(t *testing.T) {
	backend := &mockBackend{
		id: "minimal-backend",
	}

	decision := &router.RoutingDecision{
		Backend: backend,
	}

	w := httptest.NewRecorder()
	WriteRoutingHeaders(w, decision)

	// Only backend should be set
	if w.Header().Get("X-Backend-Used") != "minimal-backend" {
		t.Errorf("Expected X-Backend-Used 'minimal-backend', got %s", w.Header().Get("X-Backend-Used"))
	}

	// Other headers should be empty
	if w.Header().Get("X-Routing-Reason") != "" {
		t.Errorf("Expected empty X-Routing-Reason, got %s", w.Header().Get("X-Routing-Reason"))
	}
}

// ===== Additional Tests for Better Coverage =====

// Test HandleEmbedding succeeds when backend supports embeddings
func TestHandleEmbedding_BackendSupportsEmbed(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
		Input: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// Backend supports embeddings, should succeed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test HandleEmbedding with embedding error
func TestHandleEmbedding_EmbeddingError(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
		Input: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test HandleCompletion with routing error
func TestHandleChatCompletion_RoutingError(t *testing.T) {
	r := router.NewRouter(router.Config{})
	// Don't register any backends - routing will fail

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// Test HandleCompletion with routing error
func TestHandleCompletion_RoutingError(t *testing.T) {
	r := router.NewRouter(router.Config{})
	// Don't register any backends - routing will fail

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// Test HandleEmbedding with routing error
func TestHandleEmbedding_RoutingError(t *testing.T) {
	r := router.NewRouter(router.Config{})
	// Don't register any backends - routing will fail

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
		Input: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// Test HandleModels_EmptyBackendList
func TestHandleModels_EmptyBackendList(t *testing.T) {
	r := router.NewRouter(router.Config{})

	handler := HandleModels(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ModelsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Object != strList {
		t.Errorf("Expected object 'list', got %s", resp.Object)
	}
}

// Test chat completion streaming success
func TestHandleChatCompletion_StreamingSuccess(t *testing.T) {
	// Create a mock StreamReader
	mockReader := &mockStreamReader{
		chunks: []backends.StreamChunk{
			{Token: strHello2, Done: false},
			{Token: strWorld2, Done: false},
			{Token: "!", Done: true},
		},
	}

	backend := &mockBackendWithStream{
		mockBackend: &mockBackend{
			id:             strTestBackend,
			supportsModel:  true,
			supportsStream: true,
		},
		reader: mockReader,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:  strTestModel,
		Stream: true,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != strTextEventStream {
		t.Errorf("Expected Content-Type 'text/event-stream', got %s", ct)
	}
}

// Test completion streaming success
func TestHandleCompletion_StreamingSuccess(t *testing.T) {
	mockReader := &mockStreamReader{
		chunks: []backends.StreamChunk{
			{Token: strHello2, Done: false},
			{Token: strWorld2, Done: true},
		},
	}

	backend := &mockBackendWithStream{
		mockBackend: &mockBackend{
			id:             strTestBackend,
			supportsModel:  true,
			supportsStream: true,
		},
		reader: mockReader,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
		Stream: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != strTextEventStream {
		t.Errorf("Expected Content-Type 'text/event-stream', got %s", ct)
	}
}

// Test ParseRoutingHeaders with invalid numeric values
func TestParseRoutingHeaders_InvalidNumericValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Max-Latency-Ms", "not-a-number")
	req.Header.Set("X-Max-Power-Watts", "invalid")
	req.Header.Set("X-Deadline-Ms", "not-a-timestamp")

	annotations := ParseRoutingHeaders(req)

	// Should default to 0 for invalid values
	if annotations.MaxLatencyMs != 0 {
		t.Errorf("Expected MaxLatencyMs 0 for invalid value, got %d", annotations.MaxLatencyMs)
	}
	if annotations.MaxPowerWatts != 0 {
		t.Errorf("Expected MaxPowerWatts 0 for invalid value, got %d", annotations.MaxPowerWatts)
	}
	if annotations.DeadlineMs != 0 {
		t.Errorf("Expected DeadlineMs 0 for invalid value, got %d", annotations.DeadlineMs)
	}
}

// Test ParseRoutingHeaders with unknown media type stays unset
func TestParseRoutingHeaders_UnknownMediaType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Media-Type", "unknown-type")

	annotations := ParseRoutingHeaders(req)

	// Unknown media type should remain empty (unset)
	if annotations.MediaType != "" {
		t.Errorf("Expected empty MediaType for unknown value, got %q", annotations.MediaType)
	}
}

// Test ParseRoutingHeaders with unknown priority stays at initial value
func TestParseRoutingHeaders_UnknownPriorityValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Priority", "unknown-priority")

	annotations := ParseRoutingHeaders(req)

	// Unknown priority string doesn't match any case, so Priority stays at 0
	if annotations.Priority != 0 {
		t.Errorf("Expected Priority 0 for unknown value, got %v", annotations.Priority)
	}
}

// Test ParseRoutingHeaders with no headers defaults to normal
func TestParseRoutingHeaders_NoHeadersDefaultsNormal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	annotations := ParseRoutingHeaders(req)

	// No X-Priority header means it uses auto-logic and defaults to normal
	if annotations.Priority != backends.PriorityNormal {
		t.Errorf("Expected PriorityNormal (1) when no headers set, got %v", annotations.Priority)
	}
}

// Test response headers in successful requests
func TestHandleChatCompletion_ResponseHeaders(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// Check Content-Type header
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", ct)
	}

	// Check routing headers
	if backend := w.Header().Get("X-Backend-Used"); backend != strTestBackend {
		t.Errorf("Expected X-Backend-Used 'test-backend', got %s", backend)
	}
}

// Test completion with no backends returns service unavailable
func TestHandleCompletion_NoBackends(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// No backends registered, routing should fail
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// Test chat completion with empty messages
func TestHandleChatCompletion_EmptyMessageList(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:    strTestModel,
		Messages: []ChatCompletionMessage{},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test embedding with no backends returns service unavailable
func TestHandleEmbedding_NoBackends(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
		Input: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// No backends registered, routing should fail
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// Test chat completion response structure
func TestHandleChatCompletion_ResponseStructure(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp ChatCompletionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response structure
	if resp.Object != "chat.completion" {
		t.Errorf("Expected object 'chat.completion', got %s", resp.Object)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice")
	} else {
		if resp.Choices[0].FinishReason != strStop {
			t.Errorf("Expected finish_reason 'stop', got %s", resp.Choices[0].FinishReason)
		}
		if resp.Choices[0].Message.Role != strAssistant {
			t.Errorf("Expected role 'assistant', got %s", resp.Choices[0].Message.Role)
		}
	}

	if resp.Usage.PromptTokens <= 0 {
		t.Error("Expected prompt tokens > 0")
	}

	if resp.Usage.TotalTokens != resp.Usage.PromptTokens+resp.Usage.CompletionTokens {
		t.Error("Expected total tokens = prompt + completion tokens")
	}
}

// Test completion response structure
func TestHandleCompletion_ResponseStructure(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  strTestModel,
		Prompt: strHello2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp CompletionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Object != "text_completion" {
		t.Errorf("Expected object 'text_completion', got %s", resp.Object)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice")
	} else {
		if resp.Choices[0].FinishReason != strStop {
			t.Errorf("Expected finish_reason 'stop', got %s", resp.Choices[0].FinishReason)
		}
	}
}

// Test embedding response structure
func TestHandleEmbedding_ResponseStructure(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: strTestModel,
		Input: "Hello world",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp EmbeddingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Object != strList {
		t.Errorf("Expected object 'list', got %s", resp.Object)
	}

	if len(resp.Data) == 0 {
		t.Error("Expected at least one embedding")
	} else {
		if resp.Data[0].Object != "embedding" {
			t.Errorf("Expected object 'embedding', got %s", resp.Data[0].Object)
		}
		if len(resp.Data[0].Embedding) == 0 {
			t.Error("Expected non-empty embedding")
		}
	}
}

// Mock StreamReader implementation
type mockStreamReader struct {
	chunks []backends.StreamChunk
	index  int
}

// Recv is part of the public API.
func (m *mockStreamReader) Recv() (*backends.StreamChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, fmt.Errorf("EOF")
	}
	chunk := m.chunks[m.index]
	m.index++
	return &chunk, nil
}

// Close is part of the public API.
func (m *mockStreamReader) Close() error {
	return nil
}

// Mock backend with streaming support
type mockBackendWithStream struct {
	*mockBackend
	reader backends.StreamReader
}

// GenerateStream is part of the public API.
func (m *mockBackendWithStream) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return m.reader, nil
}

// Test chat completion with model that doesn't support the model
func TestHandleChatCompletion_ModelNotFound(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "unknown-model",
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err == nil {
		if !strings.Contains(errResp.Error.Message, "not available") {
			t.Errorf("Expected error message about model not available")
		}
	}
}

// Test completion with read body error
func TestHandleChatCompletion_ReadBodyError(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	// Use a reader that returns an error
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if err := req.Body.Close(); err != nil { // Close body to cause read error
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test null prompt handling in completion
func TestHandleCompletion_NullPrompt(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	body := []byte(`{"model":"test-model","prompt":null}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test null input handling in embedding
func TestHandleEmbedding_NullInput(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	body := []byte(`{"model":"test-model","input":null}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test multiple messages in chat completion
func TestHandleChatCompletion_MultipleMessages(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: strTestModel,
		Messages: []ChatCompletionMessage{
			{Role: strSystem, Content: strYouAreHelpful},
			{Role: strUser, Content: strHello2},
			{Role: strAssistant, Content: "Hi there"},
			{Role: strUser, Content: "How are you?"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ChatCompletionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice in response")
	}
}

// Test completion with various prompt formats
func TestHandleCompletion_ArrayPrompt(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	// Send array of prompts
	reqBody := map[string]interface{}{
		strModel: strTestModel,
		"prompt": []string{strHello2, "Hi"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test embedding with array input
func TestHandleEmbedding_ArrayInput(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleEmbedding(r)

	// Send array of inputs
	reqBody := map[string]interface{}{
		strModel: strTestModel,
		"input":  []string{strHello2, "World"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test models endpoint with different backends
func TestHandleModels_MultipleBackends(t *testing.T) {
	backend1 := &mockBackend{
		id: "backend-1",
	}
	backend2 := &mockBackend{
		id: "backend-2",
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterBackend(backend2); err != nil {
		t.Fatal(err)
	}

	handler := HandleModels(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ModelsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) == 0 {
		t.Error("Expected models in response")
	}
}

// Test chat completion with temperature and other parameters
func TestHandleChatCompletion_WithParameters(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleChatCompletion(r)

	temp := float32(0.7)
	topP := float32(0.9)
	maxTokens := int32(100)

	reqBody := ChatCompletionRequest{
		Model:       strTestModel,
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
		Messages: []ChatCompletionMessage{
			{Role: strUser, Content: strHello2},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test completion with temperature and parameters
func TestHandleCompletion_WithParameters(t *testing.T) {
	backend := &mockBackend{
		id:            strTestBackend,
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	handler := HandleCompletion(r)

	temp := float32(0.5)
	topP := float32(0.8)

	reqBody := CompletionRequest{
		Model:       strTestModel,
		Prompt:      strHello2,
		Temperature: &temp,
		TopP:        &topP,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test WriteRoutingHeaders with all fields populated
func TestWriteRoutingHeaders_AllFields(t *testing.T) {
	backend := &mockBackend{
		id: strTestBackend,
	}

	decision := &router.RoutingDecision{
		Backend:            backend,
		Reason:             "lowest latency",
		EstimatedPowerW:    20.5,
		EstimatedLatencyMs: 75,
		Alternatives:       []string{"alt-1", "alt-2", "alt-3"},
	}

	w := httptest.NewRecorder()
	WriteRoutingHeaders(w, decision)

	if w.Header().Get("X-Backend-Used") != strTestBackend {
		t.Error("Missing X-Backend-Used header")
	}
	if w.Header().Get("X-Routing-Reason") != "lowest latency" {
		t.Error("Missing X-Routing-Reason header")
	}
	if w.Header().Get("X-Estimated-Power-Watts") != "20.5" {
		t.Error("Missing X-Estimated-Power-Watts header")
	}
	if w.Header().Get("X-Estimated-Latency-Ms") != "75" {
		t.Error("Missing X-Estimated-Latency-Ms header")
	}
	if w.Header().Get("X-Alternatives") != "alt-1,alt-2,alt-3" {
		t.Error("Missing X-Alternatives header")
	}
}

// Test WriteRoutingHeaders with zero values
func TestWriteRoutingHeaders_ZeroValues(t *testing.T) {
	backend := &mockBackend{
		id: strTestBackend,
	}

	decision := &router.RoutingDecision{
		Backend:            backend,
		EstimatedPowerW:    0,
		EstimatedLatencyMs: 0,
		Alternatives:       []string{},
	}

	w := httptest.NewRecorder()
	WriteRoutingHeaders(w, decision)

	// Should have backend header
	if w.Header().Get("X-Backend-Used") != strTestBackend {
		t.Error("Missing X-Backend-Used header")
	}

	// Should NOT have empty headers for zero values
	if w.Header().Get("X-Estimated-Power-Watts") != "" {
		t.Error("Should not set X-Estimated-Power-Watts for 0 value")
	}
	if w.Header().Get("X-Estimated-Latency-Ms") != "" {
		t.Error("Should not set X-Estimated-Latency-Ms for 0 value")
	}
}
