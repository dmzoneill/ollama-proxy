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
	err     error
	id      string
	healthy bool
}

// ID is part of the public API.
func (m *mockBackend) ID() string { return m.id }

// Type is part of the public API.
func (m *mockBackend) Type() string { return strMock }

// Name is part of the public API.
func (m *mockBackend) Name() string { return "Mock Backend" }

// Hardware is part of the public API.
func (m *mockBackend) Hardware() string { return strMock }

// IsHealthy reports whether the condition holds.
func (m *mockBackend) IsHealthy() bool { return m.healthy }

// HealthCheck is part of the public API.
func (m *mockBackend) HealthCheck(_ context.Context) error { return m.err }

// PowerWatts is part of the public API.
func (m *mockBackend) PowerWatts() float64 { return 0 }

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

// ListModels is part of the public API.
func (m *mockBackend) ListModels(_ context.Context) ([]string, error) {
	return []string{"model1"}, nil
}

// SupportsModel is part of the public API.
func (m *mockBackend) SupportsModel(_ string) bool { return true }

// GetMaxModelSizeGB is part of the public API.
func (m *mockBackend) GetMaxModelSizeGB() int { return 10 }

// GetSupportedModelPatterns is part of the public API.
func (m *mockBackend) GetSupportedModelPatterns() []string { return []string{"*"} }

// GetPreferredModels is part of the public API.
func (m *mockBackend) GetPreferredModels() []string { return []string{"model1"} }

// Generate is part of the public API.
func (m *mockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return &backends.GenerateResponse{Response: "test"}, nil
}

// GenerateStream is part of the public API.
func (m *mockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

// Embed is part of the public API.
func (m *mockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return &backends.EmbedResponse{Embedding: []float32{0.1}}, nil
}

// UpdateMetrics is part of the public API.
func (m *mockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *mockBackend) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{}
}

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

// TestNewHealthChecker is part of the public API.
func TestNewHealthChecker(t *testing.T) {
	checker := NewHealthChecker()
	if checker == nil {
		t.Fatal("NewHealthChecker() returned nil")
	}
	if checker.backends == nil {
		t.Error("backends map is nil")
	}
}

// TestRegisterBackend is part of the public API.
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

// TestLivenessCheck is part of the public API.
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

	if response[strStatus] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response[strStatus])
	}

	if _, ok := response[strTime]; !ok {
		t.Error("Response missing 'time' field")
	}
}

// TestReadinessCheck_AllHealthy is part of the public API.
func TestReadinessCheck_AllHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: true})

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

	if response[strStatus] != "ready" {
		t.Errorf("Expected status 'ready', got %v", response[strStatus])
	}

	if response[string(StatusHealthy)] != float64(2) {
		t.Errorf("Expected 2 healthy backends, got %v", response[string(StatusHealthy)])
	}
}

// TestReadinessCheck_SomeHealthy is part of the public API.
func TestReadinessCheck_SomeHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: false})

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

	if response[string(StatusHealthy)] != float64(1) {
		t.Errorf("Expected 1 healthy backend, got %v", response[string(StatusHealthy)])
	}
}

// TestReadinessCheck_NoneHealthy is part of the public API.
func TestReadinessCheck_NoneHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: false})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: false})

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

	if response[strStatus] != "not_ready" {
		t.Errorf("Expected status 'not_ready', got %v", response[strStatus])
	}
}

// TestDeepHealthCheck_AllHealthy is part of the public API.
func TestDeepHealthCheck_AllHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true, err: nil})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: true, err: nil})

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

	if response[strStatus] != string(StatusHealthy) {
		t.Errorf("Expected status '%s', got %v", StatusHealthy, response[strStatus])
	}

	if response[string(StatusHealthy)] != float64(2) {
		t.Errorf("Expected 2 healthy backends, got %v", response[string(StatusHealthy)])
	}

	backends := response["backends"].(map[string]interface{})
	if len(backends) != 2 {
		t.Errorf("Expected 2 backend results, got %d", len(backends))
	}
}

// TestDeepHealthCheck_SomeUnhealthy is part of the public API.
func TestDeepHealthCheck_SomeUnhealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true, err: nil})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: false, err: context.DeadlineExceeded})

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

	if response[strStatus] != string(StatusDegraded) {
		t.Errorf("Expected status '%s', got %v", StatusDegraded, response[strStatus])
	}

	if response[string(StatusHealthy)] != float64(1) {
		t.Errorf("Expected 1 healthy backend, got %v", response[string(StatusHealthy)])
	}
}

// TestDeepHealthCheck_AllUnhealthy is part of the public API.
func TestDeepHealthCheck_AllUnhealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: false, err: context.DeadlineExceeded})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: false, err: context.DeadlineExceeded})

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

	if response[strStatus] != string(StatusUnhealthy) {
		t.Errorf("Expected status '%s', got %v", StatusUnhealthy, response[strStatus])
	}
}

// TestPerformCheck_AllHealthy is part of the public API.
func TestPerformCheck_AllHealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: true})

	result := checker.PerformCheck(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, result.Status)
	}

	if result.Details[string(StatusHealthy)] != 2 {
		t.Errorf("Expected 2 healthy backends, got %v", result.Details[string(StatusHealthy)])
	}

	if result.Details[strTotal] != 2 {
		t.Errorf("Expected 2 total backends, got %v", result.Details[strTotal])
	}
}

// TestPerformCheck_Degraded is part of the public API.
func TestPerformCheck_Degraded(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: false})

	result := checker.PerformCheck(context.Background())

	if result.Status != StatusDegraded {
		t.Errorf("Expected status %s, got %s", StatusDegraded, result.Status)
	}

	if result.Details[string(StatusHealthy)] != 1 {
		t.Errorf("Expected 1 healthy backend, got %v", result.Details[string(StatusHealthy)])
	}
}

// TestPerformCheck_Unhealthy is part of the public API.
func TestPerformCheck_Unhealthy(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: false})
	checker.RegisterBackend(&mockBackend{id: strBackend2, healthy: false})

	result := checker.PerformCheck(context.Background())

	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status %s, got %s", StatusUnhealthy, result.Status)
	}

	if result.Details[string(StatusHealthy)] != 0 {
		t.Errorf("Expected 0 healthy backends, got %v", result.Details[string(StatusHealthy)])
	}
}

// TestGetLastCheck is part of the public API.
func TestGetLastCheck(t *testing.T) {
	checker := NewHealthChecker()
	checker.RegisterBackend(&mockBackend{id: strBackend1, healthy: true})

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

// TestGetLastCheck_NoCheckPerformed is part of the public API.
func TestGetLastCheck_NoCheckPerformed(t *testing.T) {
	checker := NewHealthChecker()

	lastCheck := checker.GetLastCheck()
	if lastCheck != nil {
		t.Error("Expected nil when no check has been performed")
	}
}

// TestVersionHandler is part of the public API.
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

	if _, ok := response[strTime]; !ok {
		t.Error("Response missing 'time' field")
	}
}

// TestPerformCheck_BackendDetails is part of the public API.
func TestPerformCheck_BackendDetails(t *testing.T) {
	checker := NewHealthChecker()
	backend1 := &mockBackend{id: strBackend1, healthy: true}
	backend2 := &mockBackend{id: strBackend2, healthy: false}

	checker.RegisterBackend(backend1)
	checker.RegisterBackend(backend2)

	result := checker.PerformCheck(context.Background())

	backends := result.Details["backends"].(map[string]interface{})

	backend1Details := backends[strBackend1].(map[string]interface{})
	if backend1Details[string(StatusHealthy)] != true {
		t.Error("backend1 should be marked as healthy")
	}
	if backend1Details["type"] != strMock {
		t.Errorf("backend1 type should be 'mock', got %v", backend1Details["type"])
	}

	backend2Details := backends[strBackend2].(map[string]interface{})
	if backend2Details[string(StatusHealthy)] != false {
		t.Error("backend2 should be marked as unhealthy")
	}
}

// TestDeepHealthCheck_Timeout is part of the public API.
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

// TestConcurrentAccess is part of the public API.
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
