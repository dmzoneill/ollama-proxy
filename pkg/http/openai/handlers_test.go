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
	id              string
	supportsModel   bool
	supportsStream  bool
	generateErr     error
	generateResp    *backends.GenerateResponse
	streamErr       error
}

func (m *mockBackend) ID() string                                      { return m.id }
func (m *mockBackend) Type() string                                    { return "mock" }
func (m *mockBackend) Name() string                                    { return "Mock Backend" }
func (m *mockBackend) Hardware() string                                { return "mock-hw" }
func (m *mockBackend) IsHealthy() bool                                 { return true }
func (m *mockBackend) PowerWatts() float64                             { return 10.0 }
func (m *mockBackend) AvgLatencyMs() int32                             { return 50 }
func (m *mockBackend) Priority() int                                   { return 1 }
func (m *mockBackend) Start(ctx context.Context) error                 { return nil }
func (m *mockBackend) Stop(ctx context.Context) error                  { return nil }

// Multimedia capability methods
func (m *mockBackend) SupportsAudioToText() bool { return false }
func (m *mockBackend) SupportsTextToAudio() bool { return false }
func (m *mockBackend) SupportsImageToText() bool { return false }
func (m *mockBackend) SupportsTextToImage() bool { return false }
func (m *mockBackend) SupportsVideoToText() bool { return false }
func (m *mockBackend) SupportsTextToVideo() bool { return false }
func (m *mockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) HealthCheck(ctx context.Context) error           { return nil }
func (m *mockBackend) SupportsModel(model string) bool                { return m.supportsModel }
func (m *mockBackend) SupportsStream() bool                           { return m.supportsStream }
func (m *mockBackend) SupportsGenerate() bool                         { return true }
func (m *mockBackend) SupportsEmbed() bool                            { return true }
func (m *mockBackend) ListModels(ctx context.Context) ([]string, error) { return []string{"test-model"}, nil }
func (m *mockBackend) GetMaxModelSizeGB() int                         { return 10 }
func (m *mockBackend) GetSupportedModelPatterns() []string            { return []string{"*"} }
func (m *mockBackend) GetPreferredModels() []string                   { return []string{"test-model"} }

func (m *mockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
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

func (m *mockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, m.streamErr
}

func (m *mockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
	}, nil
}

func (m *mockBackend) UpdateMetrics(latencyMs int32, success bool) {}
func (m *mockBackend) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{
		RequestCount: 10,
		SuccessCount: 9,
		AvgLatencyMs: 50,
	}
}

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

func TestHandleChatCompletion_MissingModel(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChatCompletion_MissingMessages(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "test-model",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChatCompletion_Success(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
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

	if resp.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got %s", resp.Model)
	}
}

func TestHandleCompletion_Success(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleEmbedding_Success(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: "Hello world",
	}

	body, _ := json.Marshal(reqBody)
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

func TestHandleModels_Success(t *testing.T) {
	backend := &mockBackend{
		id: "test-backend",
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleModels(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestParseRoutingHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Target-Backend", "test-backend")
	req.Header.Set("X-Priority", "high")
	req.Header.Set("X-Latency-Critical", "true")

	annotations := ParseRoutingHeaders(req)

	if annotations.Target != "test-backend" {
		t.Errorf("Expected target backend 'test-backend', got %s", annotations.Target)
	}

	if !annotations.LatencyCritical {
		t.Error("Expected LatencyCritical to be true")
	}
}

func TestParseRoutingHeaders_AllHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Target-Backend", "ollama-nvidia")
	req.Header.Set("X-Latency-Critical", "true")
	req.Header.Set("X-Power-Efficient", "false")
	req.Header.Set("X-Max-Latency-Ms", "100")
	req.Header.Set("X-Max-Power-Watts", "50")
	req.Header.Set("X-Cache-Enabled", "yes")
	req.Header.Set("X-Media-Type", "code")
	req.Header.Set("X-Priority", "critical")
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

func TestParseRoutingHeaders_MediaTypes(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  backends.MediaType
	}{
		{"text", "text", backends.MediaTypeText},
		{"code", "code", backends.MediaTypeCode},
		{"image", "image", backends.MediaTypeImage},
		{"audio", "audio", backends.MediaTypeAudio},
		{"realtime", "realtime", backends.MediaTypeRealtime},
		{"auto", "auto", backends.MediaTypeAuto},
		{"uppercase", "TEXT", backends.MediaTypeText},
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

func TestParseRoutingHeaders_Priorities(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected backends.Priority
	}{
		{"best-effort", "best-effort", backends.PriorityBestEffort},
		{"low", "low", backends.PriorityBestEffort},
		{"normal", "normal", backends.PriorityNormal},
		{"high", "high", backends.PriorityHigh},
		{"critical", "critical", backends.PriorityCritical},
		{"realtime", "realtime", backends.PriorityCritical},
		{"uppercase", "HIGH", backends.PriorityHigh},
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

func TestParseRoutingHeaders_AutoPriority(t *testing.T) {
	// Latency-critical should set critical priority
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Latency-Critical", "true")

	annotations := ParseRoutingHeaders(req)

	if annotations.Priority != backends.PriorityCritical {
		t.Errorf("Expected PriorityCritical for latency-critical, got %v", annotations.Priority)
	}

	// Realtime media should set critical priority
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Media-Type", "realtime")

	annotations = ParseRoutingHeaders(req)

	if annotations.Priority != backends.PriorityCritical {
		t.Errorf("Expected PriorityCritical for realtime media, got %v", annotations.Priority)
	}

	// Audio media should set high priority
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Media-Type", "audio")

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

func TestParseRoutingHeaders_BoolParsing(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"yes", "yes", true},
		{"on", "on", true},
		{"false", "false", false},
		{"0", "0", false},
		{"no", "no", false},
		{"off", "off", false},
		{"uppercase", "TRUE", true},
		{"whitespace", " yes ", true},
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

func TestHandleChatCompletion_ModelNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "unsupported-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleChatCompletion_GenerationError(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: true,
		generateErr:   fmt.Errorf("generation failed"),
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleChatCompletion_StreamingNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:             "test-backend",
		supportsModel:  true,
		supportsStream: false,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:  "test-model",
		Stream: true,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChatCompletion_StreamingError(t *testing.T) {
	backend := &mockBackend{
		id:             "test-backend",
		supportsModel:  true,
		supportsStream: true,
		streamErr:      fmt.Errorf("streaming failed"),
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:  "test-model",
		Stream: true,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

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

func TestHandleCompletion_MissingModel(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCompletion_MissingPrompt(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model: "test-model",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCompletion_ModelNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "unsupported-model",
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCompletion_GenerationError(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: true,
		generateErr:   fmt.Errorf("generation failed"),
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleCompletion_StreamingNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:             "test-backend",
		supportsModel:  true,
		supportsStream: false,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
		Stream: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCompletion_StreamingError(t *testing.T) {
	backend := &mockBackend{
		id:             "test-backend",
		supportsModel:  true,
		supportsStream: true,
		streamErr:      fmt.Errorf("streaming failed"),
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
		Stream: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

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

func TestHandleEmbedding_MissingModel(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Input: "Hello",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleEmbedding_MissingInput(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: "test-model",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleEmbedding_ModelNotSupported(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: "unsupported-model",
		Input: "Hello",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

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

func TestWriteRoutingHeaders(t *testing.T) {
	backend := &mockBackend{
		id: "test-backend",
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

	if w.Header().Get("X-Backend-Used") != "test-backend" {
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

func TestWriteRoutingHeaders_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	WriteRoutingHeaders(w, nil)

	// Should not panic and headers should be empty
	if w.Header().Get("X-Backend-Used") != "" {
		t.Error("Expected no headers when decision is nil")
	}
}

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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
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
		Model:  "test-model",
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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
		Model: "test-model",
		Input: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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

	if resp.Object != "list" {
		t.Errorf("Expected object 'list', got %s", resp.Object)
	}
}

// Test chat completion streaming success
func TestHandleChatCompletion_StreamingSuccess(t *testing.T) {
	// Create a mock StreamReader
	mockReader := &mockStreamReader{
		chunks: []backends.StreamChunk{
			{Token: "Hello", Done: false},
			{Token: " world", Done: false},
			{Token: "!", Done: true},
		},
	}

	backend := &mockBackendWithStream{
		mockBackend: &mockBackend{
			id:             "test-backend",
			supportsModel:  true,
			supportsStream: true,
		},
		reader: mockReader,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model:  "test-model",
		Stream: true,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got %s", ct)
	}
}

// Test completion streaming success
func TestHandleCompletion_StreamingSuccess(t *testing.T) {
	mockReader := &mockStreamReader{
		chunks: []backends.StreamChunk{
			{Token: "Hello", Done: false},
			{Token: " world", Done: true},
		},
	}

	backend := &mockBackendWithStream{
		mockBackend: &mockBackend{
			id:             "test-backend",
			supportsModel:  true,
			supportsStream: true,
		},
		reader: mockReader,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
		Stream: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// Check Content-Type header
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", ct)
	}

	// Check routing headers
	if backend := w.Header().Get("X-Backend-Used"); backend != "test-backend" {
		t.Errorf("Expected X-Backend-Used 'test-backend', got %s", backend)
	}
}

// Test completion with no backends returns service unavailable
func TestHandleCompletion_NoBackends(t *testing.T) {
	r := router.NewRouter(router.Config{})
	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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
		Model:    "test-model",
		Messages: []ChatCompletionMessage{},
	}

	body, _ := json.Marshal(reqBody)
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
		Model: "test-model",
		Input: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
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
		if resp.Choices[0].FinishReason != "stop" {
			t.Errorf("Expected finish_reason 'stop', got %s", resp.Choices[0].FinishReason)
		}
		if resp.Choices[0].Message.Role != "assistant" {
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	reqBody := CompletionRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	body, _ := json.Marshal(reqBody)
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
		if resp.Choices[0].FinishReason != "stop" {
			t.Errorf("Expected finish_reason 'stop', got %s", resp.Choices[0].FinishReason)
		}
	}
}

// Test embedding response structure
func TestHandleEmbedding_ResponseStructure(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleEmbedding(r)

	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: "Hello world",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp EmbeddingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Object != "list" {
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

func (m *mockStreamReader) Recv() (*backends.StreamChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, fmt.Errorf("EOF")
	}
	chunk := m.chunks[m.index]
	m.index++
	return &chunk, nil
}

func (m *mockStreamReader) Close() error {
	return nil
}

// Mock backend with streaming support
type mockBackendWithStream struct {
	*mockBackend
	reader backends.StreamReader
}

func (m *mockBackendWithStream) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return m.reader, nil
}

// Test chat completion with model that doesn't support the model
func TestHandleChatCompletion_ModelNotFound(t *testing.T) {
	backend := &mockBackend{
		id:            "test-backend",
		supportsModel: false,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "unknown-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
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
	req.Body.Close() // Close body to cause read error
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	reqBody := ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
			{Role: "user", Content: "How are you?"},
		},
	}

	body, _ := json.Marshal(reqBody)
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	// Send array of prompts
	reqBody := map[string]interface{}{
		"model":  "test-model",
		"prompt": []string{"Hello", "Hi"},
	}

	body, _ := json.Marshal(reqBody)
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleEmbedding(r)

	// Send array of inputs
	reqBody := map[string]interface{}{
		"model": "test-model",
		"input": []string{"Hello", "World"},
	}

	body, _ := json.Marshal(reqBody)
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
	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleChatCompletion(r)

	temp := float32(0.7)
	topP := float32(0.9)
	maxTokens := int32(100)

	reqBody := ChatCompletionRequest{
		Model:       "test-model",
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
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
		id:            "test-backend",
		supportsModel: true,
	}

	r := router.NewRouter(router.Config{})
	r.RegisterBackend(backend)

	handler := HandleCompletion(r)

	temp := float32(0.5)
	topP := float32(0.8)

	reqBody := CompletionRequest{
		Model:       "test-model",
		Prompt:      "Hello",
		Temperature: &temp,
		TopP:        &topP,
	}

	body, _ := json.Marshal(reqBody)
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
		id: "test-backend",
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

	if w.Header().Get("X-Backend-Used") != "test-backend" {
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
		id: "test-backend",
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
	if w.Header().Get("X-Backend-Used") != "test-backend" {
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
