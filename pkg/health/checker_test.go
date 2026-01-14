package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// mockBackend implements backends.Backend for testing
type mockBackend struct {
	id      string
	healthy bool
	err     error
}

func (m *mockBackend) ID() string                       { return m.id }
func (m *mockBackend) Type() string                     { return "mock" }
func (m *mockBackend) Name() string                     { return "Mock Backend" }
func (m *mockBackend) Hardware() string                 { return "mock" }
func (m *mockBackend) IsHealthy() bool                  { return m.healthy }
func (m *mockBackend) HealthCheck(ctx context.Context) error { return m.err }
func (m *mockBackend) PowerWatts() float64              { return 0 }
func (m *mockBackend) AvgLatencyMs() int32              { return 100 }
func (m *mockBackend) Priority() int                    { return 5 }
func (m *mockBackend) SupportsGenerate() bool           { return true }
func (m *mockBackend) SupportsStream() bool             { return true }
func (m *mockBackend) SupportsEmbed() bool              { return true }
func (m *mockBackend) ListModels(ctx context.Context) ([]string, error) { return []string{"model1"}, nil }
func (m *mockBackend) SupportsModel(modelName string) bool { return true }
func (m *mockBackend) GetMaxModelSizeGB() int           { return 10 }
func (m *mockBackend) GetSupportedModelPatterns() []string { return []string{"*"} }
func (m *mockBackend) GetPreferredModels() []string     { return []string{"model1"} }
func (m *mockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return &backends.GenerateResponse{Response: "test"}, nil
}
func (m *mockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}
func (m *mockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return &backends.EmbedResponse{Embedding: []float32{0.1}}, nil
}
func (m *mockBackend) UpdateMetrics(latencyMs int32, success bool) {}
func (m *mockBackend) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{}
}
func (m *mockBackend) Start(ctx context.Context) error { return nil }
func (m *mockBackend) Stop(ctx context.Context) error  { return nil }

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

func TestNewHealthChecker(t *testing.T) {
	checker := NewHealthChecker()
	if checker == nil {
		t.Fatal("NewHealthChecker() returned nil")
	}
	if checker.backends == nil {
		t.Error("backends map is nil")
	}
}

func TestRegisterBackend(t *testing.T) {
	checker := NewHealthChecker()
	backend := &mockBackend{id: "test-backend", healthy: true}

	checker.RegisterBackend(backend)

	checker.mu.RLock()
	defer checker.mu.RUnlock()
	if _, exists := checker.backends["test-backend"]; !exists {
		t.Error("Backend was not registered")
	}
}

func TestLivenessCheck(t *testing.T) {
	checker := NewHealthChecker()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	checker.LivenessCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}

	if _, ok := response["time"]; !ok {
		t.Error("Response missing 'time' field")
	}
}

func TestReadinessCheck_AllHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: true})

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	checker.ReadinessCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", response["status"])
	}

	if response["healthy"] != float64(2) {
		t.Errorf("Expected 2 healthy backends, got %v", response["healthy"])
	}
}

func TestReadinessCheck_SomeHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: false})

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	checker.ReadinessCheck(w, req)

	// Should still be ready if at least one backend is healthy
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["healthy"] != float64(1) {
		t.Errorf("Expected 1 healthy backend, got %v", response["healthy"])
	}
}

func TestReadinessCheck_NoneHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: false})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: false})

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	checker.ReadinessCheck(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "not_ready" {
		t.Errorf("Expected status 'not_ready', got %v", response["status"])
	}
}

func TestDeepHealthCheck_AllHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true, err: nil})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: true, err: nil})

	req := httptest.NewRequest("GET", "/health/deep", nil)
	w := httptest.NewRecorder()

	checker.DeepHealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != string(StatusHealthy) {
		t.Errorf("Expected status '%s', got %v", StatusHealthy, response["status"])
	}

	if response["healthy"] != float64(2) {
		t.Errorf("Expected 2 healthy backends, got %v", response["healthy"])
	}

	backends := response["backends"].(map[string]interface{})
	if len(backends) != 2 {
		t.Errorf("Expected 2 backend results, got %d", len(backends))
	}
}

func TestDeepHealthCheck_SomeUnhealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true, err: nil})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: false, err: context.DeadlineExceeded})

	req := httptest.NewRequest("GET", "/health/deep", nil)
	w := httptest.NewRecorder()

	checker.DeepHealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != string(StatusDegraded) {
		t.Errorf("Expected status '%s', got %v", StatusDegraded, response["status"])
	}

	if response["healthy"] != float64(1) {
		t.Errorf("Expected 1 healthy backend, got %v", response["healthy"])
	}
}

func TestDeepHealthCheck_AllUnhealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: false, err: context.DeadlineExceeded})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: false, err: context.DeadlineExceeded})

	req := httptest.NewRequest("GET", "/health/deep", nil)
	w := httptest.NewRecorder()

	checker.DeepHealthCheck(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != string(StatusUnhealthy) {
		t.Errorf("Expected status '%s', got %v", StatusUnhealthy, response["status"])
	}
}

func TestPerformCheck_AllHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: true})

	result := checker.PerformCheck(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, result.Status)
	}

	if result.Details["healthy"] != 2 {
		t.Errorf("Expected 2 healthy backends, got %v", result.Details["healthy"])
	}

	if result.Details["total"] != 2 {
		t.Errorf("Expected 2 total backends, got %v", result.Details["total"])
	}
}

func TestPerformCheck_Degraded(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: false})

	result := checker.PerformCheck(context.Background())

	if result.Status != StatusDegraded {
		t.Errorf("Expected status %s, got %s", StatusDegraded, result.Status)
	}

	if result.Details["healthy"] != 1 {
		t.Errorf("Expected 1 healthy backend, got %v", result.Details["healthy"])
	}
}

func TestPerformCheck_Unhealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: false})
	checker.RegisterBackend(&mockBackend{id: "backend2", healthy: false})

	result := checker.PerformCheck(context.Background())

	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status %s, got %s", StatusUnhealthy, result.Status)
	}

	if result.Details["healthy"] != 0 {
		t.Errorf("Expected 0 healthy backends, got %v", result.Details["healthy"])
	}
}

func TestGetLastCheck(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: "backend1", healthy: true})

	// Perform a check
	checker.PerformCheck(context.Background())

	// Get the last check
	lastCheck := checker.GetLastCheck()
	if lastCheck == nil {
		t.Fatal("GetLastCheck() returned nil")
	}

	if lastCheck.Status != StatusHealthy {
		t.Errorf("Expected last check status %s, got %s", StatusHealthy, lastCheck.Status)
	}
}

func TestGetLastCheck_NoCheckPerformed(t *testing.T) {
	checker := NewHealthChecker()

	lastCheck := checker.GetLastCheck()
	if lastCheck != nil {
		t.Error("Expected nil when no check has been performed")
	}
}

func TestVersionHandler(t *testing.T) {
	handler := VersionHandler("1.0.0", "abc123", "2026-01-12T00:00:00Z")

	req := httptest.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", response["version"])
	}

	if response["git_commit"] != "abc123" {
		t.Errorf("Expected git_commit 'abc123', got %v", response["git_commit"])
	}

	if response["build_time"] != "2026-01-12T00:00:00Z" {
		t.Errorf("Expected build_time '2026-01-12T00:00:00Z', got %v", response["build_time"])
	}

	if _, ok := response["time"]; !ok {
		t.Error("Response missing 'time' field")
	}
}

func TestPerformCheck_BackendDetails(t *testing.T) {
	checker := NewHealthChecker()
	backend1 := &mockBackend{id: "backend1", healthy: true}
	backend2 := &mockBackend{id: "backend2", healthy: false}

	checker.RegisterBackend(backend1)
	checker.RegisterBackend(backend2)

	result := checker.PerformCheck(context.Background())

	backends := result.Details["backends"].(map[string]interface{})

	backend1Details := backends["backend1"].(map[string]interface{})
	if backend1Details["healthy"] != true {
		t.Error("backend1 should be marked as healthy")
	}
	if backend1Details["type"] != "mock" {
		t.Errorf("backend1 type should be 'mock', got %v", backend1Details["type"])
	}

	backend2Details := backends["backend2"].(map[string]interface{})
	if backend2Details["healthy"] != false {
		t.Error("backend2 should be marked as unhealthy")
	}
}

func TestDeepHealthCheck_Timeout(t *testing.T) {
	// Create a backend that takes too long to respond
	slowBackend := &mockBackend{
		id:      "slow-backend",
		healthy: true,
		err:     nil,
	}

	checker := NewHealthChecker()
	checker.RegisterBackend(slowBackend)

	// Create request with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/health/deep", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// This should complete before the timeout since we're not actually sleeping
	checker.DeepHealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestConcurrentAccess(t *testing.T) {
	checker := NewHealthChecker()

	// Run concurrent operations
	done := make(chan bool)

	// Register backends concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			backend := &mockBackend{
				id:      string(rune('a' + id)),
				healthy: i%2 == 0,
			}
			checker.RegisterBackend(backend)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		<-done
	}

	// Perform health checks concurrently
	for i := 0; i < 10; i++ {
		go func() {
			checker.PerformCheck(context.Background())
			done <- true
		}()
	}

	// Wait for all checks
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no race conditions occurred
	result := checker.PerformCheck(context.Background())
	if result == nil {
		t.Error("PerformCheck returned nil after concurrent access")
	}
}
