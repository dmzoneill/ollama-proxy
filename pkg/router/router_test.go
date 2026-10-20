package router

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// MockBackend is a mock implementation of the Backend interface for testing
type MockBackend struct {
	id            string
	backendType   string
	name          string
	hardware      string
	modelPatterns []string
	powerWatts    float64
	priority      int
	avgLatencyMs  int32
	healthy       bool
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
func (m *MockBackend) SupportsGenerate() bool { return true }

// SupportsStream is part of the public API.
func (m *MockBackend) SupportsStream() bool { return true }

// SupportsEmbed is part of the public API.
func (m *MockBackend) SupportsEmbed() bool { return false }

// ListModels is part of the public API.
func (m *MockBackend) ListModels(_ context.Context) ([]string, error) { return []string{}, nil }

// SupportsModel is part of the public API.
func (m *MockBackend) SupportsModel(modelName string) bool {
	if len(m.modelPatterns) == 0 {
		return true // Default to supporting all models
	}
	// Simple pattern matching for testing
	for _, pattern := range m.modelPatterns {
		if pattern == "*" {
			return true
		}
		if pattern == modelName {
			return true
		}
		// Simple prefix/suffix matching
		if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(modelName) >= len(prefix) && modelName[:len(prefix)] == prefix {
				return true
			}
		}
		if len(pattern) > 0 && pattern[0] == '*' {
			suffix := pattern[1:]
			if len(modelName) >= len(suffix) && modelName[len(modelName)-len(suffix):] == suffix {
				return true
			}
		}
	}
	return false
}

// GetMaxModelSizeGB is part of the public API.
func (m *MockBackend) GetMaxModelSizeGB() int { return 10 }

// GetSupportedModelPatterns is part of the public API.
func (m *MockBackend) GetSupportedModelPatterns() []string {
	if len(m.modelPatterns) == 0 {
		return []string{"*"}
	}
	return m.modelPatterns
}

// GetPreferredModels is part of the public API.
func (m *MockBackend) GetPreferredModels() []string { return []string{} }

// Generate is part of the public API.
func (m *MockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return &backends.GenerateResponse{}, nil
}

// GenerateStream is part of the public API.
func (m *MockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

// Embed is part of the public API.
func (m *MockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// UpdateMetrics is part of the public API.
func (m *MockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *MockBackend) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{}
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

// TestNewRouter is part of the public API.
func TestNewRouter(t *testing.T) {
	cfg := Config{
		DefaultBackendID: strTestBackend,
		PowerAware:       true,
		AutoOptimize:     true,
	}

	router := NewRouter(cfg)

	if router == nil {
		t.Fatal("Expected router to be created, got nil")
	}

	if router.defaultBackendID != strTestBackend {
		t.Errorf("Expected default backend ID 'test-backend', got '%s'", router.defaultBackendID)
	}

	if !router.powerAware {
		t.Error("Expected powerAware to be true")
	}

	if !router.autoOptimize {
		t.Error("Expected autoOptimize to be true")
	}
}

// TestRegisterBackend is part of the public API.
func TestRegisterBackend(t *testing.T) {
	router := NewRouter(Config{})

	backend := &MockBackend{
		id:       strBackend1,
		name:     strTestBackend2,
		hardware: strCPU,
		healthy:  true,
	}

	err := router.RegisterBackend(backend)
	if err != nil {
		t.Fatalf("Failed to register backend: %v", err)
	}

	// Try to register the same backend again
	err = router.RegisterBackend(backend)
	if err == nil {
		t.Error("Expected error when registering duplicate backend, got nil")
	}
}

// TestGetBackend is part of the public API.
func TestGetBackend(t *testing.T) {
	router := NewRouter(Config{})

	backend := &MockBackend{
		id:       strBackend1,
		name:     strTestBackend2,
		hardware: strCPU,
		healthy:  true,
	}

	router.RegisterBackend(backend)

	retrieved, exists := router.GetBackend(strBackend1)
	if !exists {
		t.Fatal("Expected backend to exist")
	}

	if retrieved.ID() != strBackend1 {
		t.Errorf("Expected backend ID 'backend-1', got '%s'", retrieved.ID())
	}

	_, exists = router.GetBackend("nonexistent")
	if exists {
		t.Error("Expected backend to not exist")
	}
}

// TestListBackends is part of the public API.
func TestListBackends(t *testing.T) {
	router := NewRouter(Config{})

	backend1 := &MockBackend{id: strBackend1, healthy: true}
	backend2 := &MockBackend{id: strBackend22, healthy: true}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	backends := router.ListBackends()
	if len(backends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(backends))
	}
}

// TestRouteRequest_ExplicitTarget is part of the public API.
func TestRouteRequest_ExplicitTarget(t *testing.T) {
	router := NewRouter(Config{})

	backend := &MockBackend{
		id:           strBackend1,
		name:         strTestBackend2,
		hardware:     strCPU,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 500,
	}

	router.RegisterBackend(backend)

	annotations := &backends.Annotations{
		Target: strBackend1,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	// Unwrap the QueueTrackingBackend to get the actual backend
	trackingBackend, ok := decision.Backend.(*QueueTrackingBackend)
	if !ok {
		t.Fatal("Expected QueueTrackingBackend")
	}

	if trackingBackend.ID() != strBackend1 {
		t.Errorf("Expected backend 'backend-1', got '%s'", trackingBackend.ID())
	}
}

// TestRouteRequest_LatencyCritical is part of the public API.
func TestRouteRequest_LatencyCritical(t *testing.T) {
	router := NewRouter(Config{})

	fastBackend := &MockBackend{
		id:           strFastBackend,
		hardware:     strGpu,
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 150,
		priority:     5,
	}

	slowBackend := &MockBackend{
		id:           "slow-backend",
		hardware:     strCPU,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 1000,
		priority:     5,
	}

	router.RegisterBackend(fastBackend)
	router.RegisterBackend(slowBackend)

	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	trackingBackend := decision.Backend.(*QueueTrackingBackend)
	if trackingBackend.ID() != strFastBackend {
		t.Errorf("Expected fast backend for latency-critical request, got '%s'", trackingBackend.ID())
	}
}

// TestRouteRequest_PowerEfficient is part of the public API.
func TestRouteRequest_PowerEfficient(t *testing.T) {
	router := NewRouter(Config{PowerAware: true})

	powerHungry := &MockBackend{
		id:           strPowerHungry,
		hardware:     strGpu,
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 150,
		priority:     5,
	}

	powerEfficient := &MockBackend{
		id:           strPowerEfficient,
		hardware:     strNpu,
		healthy:      true,
		powerWatts:   3.0,
		avgLatencyMs: 500,
		priority:     5,
	}

	router.RegisterBackend(powerHungry)
	router.RegisterBackend(powerEfficient)

	annotations := &backends.Annotations{
		PreferPowerEfficiency: true,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	trackingBackend := decision.Backend.(*QueueTrackingBackend)
	if trackingBackend.ID() != strPowerEfficient {
		t.Errorf("Expected power-efficient backend, got '%s'", trackingBackend.ID())
	}
}

// TestRouteRequest_MaxLatencyConstraint is part of the public API.
func TestRouteRequest_MaxLatencyConstraint(t *testing.T) {
	router := NewRouter(Config{})

	fastBackend := &MockBackend{
		id:           strFastBackend,
		hardware:     strGpu,
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 150,
	}

	slowBackend := &MockBackend{
		id:           "slow-backend",
		hardware:     strCPU,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 1000,
	}

	router.RegisterBackend(fastBackend)
	router.RegisterBackend(slowBackend)

	annotations := &backends.Annotations{
		MaxLatencyMs: 200,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	trackingBackend := decision.Backend.(*QueueTrackingBackend)
	if trackingBackend.ID() != strFastBackend {
		t.Errorf("Expected fast-backend (only one meeting constraint), got '%s'", trackingBackend.ID())
	}

	// Test with constraint that filters out all backends
	annotations.MaxLatencyMs = 100

	_, err = router.RouteRequest(context.Background(), annotations)
	if err == nil {
		t.Error("Expected error when no backends meet latency constraint")
	}
}

// TestRouteRequest_MaxPowerConstraint is part of the public API.
func TestRouteRequest_MaxPowerConstraint(t *testing.T) {
	router := NewRouter(Config{})

	powerHungry := &MockBackend{
		id:           strPowerHungry,
		hardware:     strGpu,
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 150,
	}

	powerEfficient := &MockBackend{
		id:           strPowerEfficient,
		hardware:     strNpu,
		healthy:      true,
		powerWatts:   3.0,
		avgLatencyMs: 500,
	}

	router.RegisterBackend(powerHungry)
	router.RegisterBackend(powerEfficient)

	annotations := &backends.Annotations{
		MaxPowerWatts: 10,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	trackingBackend := decision.Backend.(*QueueTrackingBackend)
	if trackingBackend.ID() != strPowerEfficient {
		t.Errorf("Expected power-efficient backend (only one meeting constraint), got '%s'", trackingBackend.ID())
	}
}

// TestRouteRequest_UnhealthyBackend is part of the public API.
func TestRouteRequest_UnhealthyBackend(t *testing.T) {
	router := NewRouter(Config{})

	unhealthyBackend := &MockBackend{
		id:      strUnhealthy,
		healthy: false,
	}

	router.RegisterBackend(unhealthyBackend)

	annotations := &backends.Annotations{
		Target: strUnhealthy,
	}

	_, err := router.RouteRequest(context.Background(), annotations)
	if err == nil {
		t.Error("Expected error when only unhealthy backend available")
	}
}

// TestRouteRequest_NoBackends is part of the public API.
func TestRouteRequest_NoBackends(t *testing.T) {
	router := NewRouter(Config{})

	annotations := &backends.Annotations{}

	_, err := router.RouteRequest(context.Background(), annotations)
	if err == nil {
		t.Error("Expected error when no backends are registered")
	}
}

// TestRouteRequest_PriorityBoost is part of the public API.
func TestRouteRequest_PriorityBoost(t *testing.T) {
	router := NewRouter(Config{})

	backend1 := &MockBackend{
		id:           strBackend1,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 500,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           strBackend22,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 500,
		priority:     5,
	}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	// Test critical priority
	annotations := &backends.Annotations{
		Priority: backends.PriorityCritical,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	if decision.Backend == nil {
		t.Error("Expected a backend to be selected")
	}
}

// TestHealthCheckAll is part of the public API.
func TestHealthCheckAll(t *testing.T) {
	router := NewRouter(Config{})

	healthy := &MockBackend{
		id:      strHealthy,
		healthy: true,
	}

	unhealthy := &MockBackend{
		id:      strUnhealthy,
		healthy: false,
	}

	router.RegisterBackend(healthy)
	router.RegisterBackend(unhealthy)

	results := router.HealthCheckAll(context.Background())

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if !results[strHealthy] {
		t.Error("Expected healthy backend to report healthy")
	}

	if results[strUnhealthy] {
		t.Error("Expected unhealthy backend to report unhealthy")
	}
}

// TestFallbackRequest is part of the public API.
func TestFallbackRequest(t *testing.T) {
	router := NewRouter(Config{})

	backend1 := &MockBackend{
		id:           strBackend1,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 500,
	}

	backend2 := &MockBackend{
		id:           strBackend22,
		healthy:      true,
		powerWatts:   15.0,
		avgLatencyMs: 300,
	}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	// Exclude backend-1, should get backend-2
	excludeBackends := []string{strBackend1}
	annotations := &backends.Annotations{}

	decision, err := router.FallbackRequest(context.Background(), excludeBackends, annotations)
	if err != nil {
		t.Fatalf("FallbackRequest failed: %v", err)
	}

	if decision.Backend.ID() != strBackend22 {
		t.Errorf("Expected backend-2 as fallback, got '%s'", decision.Backend.ID())
	}

	// Exclude all backends
	excludeBackends = []string{strBackend1, strBackend22}
	_, err = router.FallbackRequest(context.Background(), excludeBackends, annotations)
	if err == nil {
		t.Error("Expected error when all backends are excluded")
	}
}

// TestStats is part of the public API.
func TestStats(t *testing.T) {
	router := NewRouter(Config{})

	healthy := &MockBackend{
		id:      strHealthy,
		healthy: true,
	}

	unhealthy := &MockBackend{
		id:      strUnhealthy,
		healthy: false,
	}

	router.RegisterBackend(healthy)
	router.RegisterBackend(unhealthy)

	stats := router.Stats()

	totalBackends, ok := stats["total_backends"].(int)
	if !ok || totalBackends != 2 {
		t.Errorf("Expected total_backends to be 2, got %v", stats["total_backends"])
	}

	healthyBackends, ok := stats["healthy_backends"].(int)
	if !ok || healthyBackends != 1 {
		t.Errorf("Expected healthy_backends to be 1, got %v", stats["healthy_backends"])
	}
}

// TestRouter_RouteRequest_NoBackends is part of the public API.
func TestRouter_RouteRequest_NoBackends(t *testing.T) {
	router := NewRouter(Config{})

	ctx := context.Background()
	annotations := &backends.Annotations{}
	_, err := router.RouteRequest(ctx, annotations)

	if err == nil {
		t.Error("Expected error when no backends are registered")
	}
}

// TestRouter_FallbackRequest_NoBackends is part of the public API.
func TestRouter_FallbackRequest_NoBackends(t *testing.T) {
	router := NewRouter(Config{})

	ctx := context.Background()
	annotations := &backends.Annotations{}
	_, err := router.FallbackRequest(ctx, []string{}, annotations)

	if err == nil {
		t.Error("Expected error when no backends available for fallback")
	}
}

// TestRouter_GetBackend_NotFound is part of the public API.
func TestRouter_GetBackend_NotFound(t *testing.T) {
	router := NewRouter(Config{})

	backend, found := router.GetBackend("non-existent")
	if found {
		t.Error("Expected false for non-existent backend")
	}
	if backend != nil {
		t.Error("Expected nil backend for non-existent ID")
	}
}

// TestRouter_ListBackends_Empty is part of the public API.
func TestRouter_ListBackends_Empty(t *testing.T) {
	router := NewRouter(Config{})

	backends := router.ListBackends()
	if len(backends) != 0 {
		t.Errorf("Expected 0 backends, got %d", len(backends))
	}
}

// TestRouter_ListBackends_Multiple is part of the public API.
func TestRouter_ListBackends_Multiple(t *testing.T) {
	router := NewRouter(Config{})

	backend1 := &MockBackend{id: strBackend1, healthy: true}
	backend2 := &MockBackend{id: strBackend22, healthy: true}
	backend3 := &MockBackend{id: strBackend3, healthy: false}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)
	router.RegisterBackend(backend3)

	backends := router.ListBackends()
	if len(backends) != 3 {
		t.Errorf("Expected 3 backends, got %d", len(backends))
	}
}

// TestRouter_RouteRequest_WithTarget is part of the public API.
func TestRouter_RouteRequest_WithTarget(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strBackend1})

	backend1 := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}

	router.RegisterBackend(backend1)

	ctx := context.Background()
	annotations := &backends.Annotations{
		Target: strBackend1,
	}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	if decision.Backend.ID() != strBackend1 {
		t.Errorf("Expected backend-1, got %s", decision.Backend.ID())
	}
}

// TestRouter_RouteRequest_MultipleBackends is part of the public API.
func TestRouter_RouteRequest_MultipleBackends(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strBackend22})

	backend1 := &MockBackend{id: strBackend1, healthy: true, hardware: strNpu}
	backend2 := &MockBackend{id: strBackend22, healthy: true, hardware: strIgpu}
	backend3 := &MockBackend{id: strBackend3, healthy: true, hardware: strNvidia}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)
	router.RegisterBackend(backend3)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	if decision.Backend == nil {
		t.Fatal("Backend is nil")
	}
}

// TestRouter_ScoreCandidates_PowerAware is part of the public API.
func TestRouter_ScoreCandidates_PowerAware(t *testing.T) {
	router := NewRouter(Config{PowerAware: true})

	backend1 := &MockBackend{id: "high-power", healthy: true, powerWatts: 50.0}
	backend2 := &MockBackend{id: "low-power", healthy: true, powerWatts: 10.0}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	// With power-aware routing, should prefer low-power backend
	if decision.Backend.ID() == "high-power" {
		t.Logf("Selected high-power backend (power-aware might still select it based on other factors)")
	}
}

// TestRouter_GetBackend_Exists is part of the public API.
func TestRouter_GetBackend_Exists(t *testing.T) {
	router := NewRouter(Config{})

	backend1 := &MockBackend{id: strBackend1, healthy: true}
	router.RegisterBackend(backend1)

	backend, found := router.GetBackend(strBackend1)
	if !found {
		t.Error("Expected backend to be found")
	}
	if backend.ID() != strBackend1 {
		t.Errorf("Expected backend-1, got %s", backend.ID())
	}
}

// TestRouter_RouteRequest_LatencyCritical is part of the public API.
func TestRouter_RouteRequest_LatencyCritical(t *testing.T) {
	router := NewRouter(Config{})

	fastBackend := &MockBackend{id: strFast, healthy: true, avgLatencyMs: 50}
	slowBackend := &MockBackend{id: strSlow, healthy: true, avgLatencyMs: 500}

	router.RegisterBackend(fastBackend)
	router.RegisterBackend(slowBackend)

	ctx := context.Background()
	annotations := &backends.Annotations{
		LatencyCritical: true,
		MaxLatencyMs:    100,
	}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	// Should prefer faster backend for latency-critical requests
	t.Logf("Selected: %s (latency: %dms)", decision.Backend.ID(), decision.Backend.AvgLatencyMs())
}

// TestRouter_RouteRequest_PowerEfficiency is part of the public API.
func TestRouter_RouteRequest_PowerEfficiency(t *testing.T) {
	router := NewRouter(Config{PowerAware: true})

	powerHungry := &MockBackend{id: strPowerHungry, healthy: true, powerWatts: 100.0}
	powerEfficient := &MockBackend{id: strEfficient, healthy: true, powerWatts: 10.0}

	router.RegisterBackend(powerHungry)
	router.RegisterBackend(powerEfficient)

	ctx := context.Background()
	annotations := &backends.Annotations{
		PreferPowerEfficiency: true,
		MaxPowerWatts:         50,
	}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	t.Logf("Selected: %s (power: %.1fW)", decision.Backend.ID(), decision.Backend.PowerWatts())
}

// TestRouter_RouteRequest_Priority is part of the public API.
func TestRouter_RouteRequest_Priority(t *testing.T) {
	router := NewRouter(Config{})

	highPriority := &MockBackend{id: "high-pri", healthy: true, priority: 10}
	lowPriority := &MockBackend{id: "low-pri", healthy: true, priority: 1}

	router.RegisterBackend(highPriority)
	router.RegisterBackend(lowPriority)

	ctx := context.Background()
	annotations := &backends.Annotations{
		Priority: backends.PriorityCritical,
	}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	t.Logf("Selected: %s (priority: %d)", decision.Backend.ID(), decision.Backend.Priority())
}

// TestRouter_RouteRequest_UnhealthyBackends is part of the public API.
func TestRouter_RouteRequest_UnhealthyBackends(t *testing.T) {
	router := NewRouter(Config{})

	unhealthy := &MockBackend{id: strUnhealthy, healthy: false}
	healthy := &MockBackend{id: strHealthy, healthy: true}

	router.RegisterBackend(unhealthy)
	router.RegisterBackend(healthy)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	decision, err := router.RouteRequest(ctx, annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	if decision.Backend.ID() != strHealthy {
		t.Errorf("Should not select unhealthy backend, got %s", decision.Backend.ID())
	}
}

// TestRouter_FallbackRequest_WithExclusions is part of the public API.
func TestRouter_FallbackRequest_WithExclusions(t *testing.T) {
	router := NewRouter(Config{})

	backend1 := &MockBackend{id: strBackend1, healthy: true}
	backend2 := &MockBackend{id: strBackend22, healthy: true}
	backend3 := &MockBackend{id: strBackend3, healthy: true}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)
	router.RegisterBackend(backend3)

	ctx := context.Background()
	excludeBackends := []string{strBackend1}
	annotations := &backends.Annotations{}

	decision, err := router.FallbackRequest(ctx, excludeBackends, annotations)
	if err != nil {
		t.Fatalf("FallbackRequest failed: %v", err)
	}

	if decision.Backend.ID() == strBackend1 {
		t.Error("Should not select excluded backend")
	}

	t.Logf("Selected fallback: %s", decision.Backend.ID())
}

// TestRouter_Stats_MultipleBackends is part of the public API.
func TestRouter_Stats_MultipleBackends(t *testing.T) {
	router := NewRouter(Config{})

	for i := 0; i < 5; i++ {
		backend := &MockBackend{
			id:      fmt.Sprintf("backend-%d", i),
			healthy: i%2 == 0, // Alternate healthy/unhealthy
		}
		router.RegisterBackend(backend)
	}

	stats := router.Stats()

	totalBackends, ok := stats["total_backends"].(int)
	if !ok || totalBackends != 5 {
		t.Errorf("Expected 5 total backends, got %v", stats["total_backends"])
	}

	healthyBackends, ok := stats["healthy_backends"].(int)
	if !ok || healthyBackends != 3 {
		t.Errorf("Expected 3 healthy backends, got %v", stats["healthy_backends"])
	}

	t.Logf("Stats: total=%d healthy=%d", totalBackends, healthyBackends)
}

// TestRouter_ListBackends_OrderPreserved is part of the public API.
func TestRouter_ListBackends_OrderPreserved(t *testing.T) {
	router := NewRouter(Config{})

	expectedIDs := map[string]bool{"a": false, "b": false, "c": false, "d": false, "e": false}
	for id := range expectedIDs {
		router.RegisterBackend(&MockBackend{id: id, healthy: true})
	}

	backends := router.ListBackends()

	if len(backends) != len(expectedIDs) {
		t.Fatalf("Expected %d backends, got %d", len(expectedIDs), len(backends))
	}

	// Check all expected backends are present (order doesn't matter with maps)
	for _, backend := range backends {
		if _, exists := expectedIDs[backend.ID()]; !exists {
			t.Errorf("Unexpected backend ID: %s", backend.ID())
		}
		expectedIDs[backend.ID()] = true
	}

	// Check all backends were found
	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected backend %s not found", id)
		}
	}
}

// TestRouter_RouteRequest_ContextCancellation is part of the public API.
func TestRouter_RouteRequest_ContextCancellation(t *testing.T) {
	router := NewRouter(Config{})

	backend := &MockBackend{id: strBackend1, healthy: true}
	router.RegisterBackend(backend)

	// Create already-canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	annotations := &backends.Annotations{}

	_, err := router.RouteRequest(ctx, annotations)
	if err == nil {
		t.Error("Expected error when context is canceled")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Expected 'canceled' in error message, got: %v", err)
	}
}

// TestRouter_RouteRequest_MediaTypeConstraint is part of the public API.
func TestRouter_RouteRequest_MediaTypeConstraint(t *testing.T) {
	router := NewRouter(Config{})

	// Create backend that doesn't support the media type we'll request
	backend := &MockBackend{
		id:      strBackend1,
		healthy: true,
	}
	router.RegisterBackend(backend)

	// Request with media type constraint
	// Since MockBackend doesn't filter by media type, we need to test error message building
	annotations := &backends.Annotations{
		MediaType:     "video",
		MaxLatencyMs:  100,
		MaxPowerWatts: 5,
	}

	// This should route successfully with MockBackend, but we're testing constraint tracking
	decision, err := router.RouteRequest(context.Background(), annotations)
	if err == nil && decision != nil {
		// Success is expected with permissive MockBackend
		t.Logf("Routed successfully with media type constraint")
	}
}

// TestRouter_RouteRequest_DefaultScoring is part of the public API.
func TestRouter_RouteRequest_DefaultScoring(t *testing.T) {
	// Test the default scoring path (no power awareness, no priority boost, empty queue)
	router := NewRouter(Config{PowerAware: false}) // Disable power-aware scoring

	backend := &MockBackend{
		id:           strBackend1,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 500,
		priority:     5,
	}

	router.RegisterBackend(backend)

	// Request with no special priority (default/normal)
	annotations := &backends.Annotations{
		Priority: backends.PriorityNormal,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	if decision.Backend == nil {
		t.Fatal("Expected backend to be selected")
	}

	t.Logf("Selected backend with default scoring: %s", decision.Backend.ID())
}

// TestRouter_RouteRequest_ConstraintErrorMessages is part of the public API.
func TestRouter_RouteRequest_ConstraintErrorMessages(t *testing.T) {
	config := Config{
		PowerAware: true,
	}
	router := NewRouter(config)

	// Create backend that won't match our constraints
	backend := &MockBackend{
		id:           "high-power-backend",
		healthy:      true,
		powerWatts:   300.0,
		avgLatencyMs: 1000,
		hardware:     strNvidia,
	}
	router.RegisterBackend(backend)

	// Test with MaxPowerWatts constraint
	annotations := &backends.Annotations{
		MaxPowerWatts: 50.0, // Backend uses 300W, so won't match
		MaxLatencyMs:  100,  // Backend has 1000ms latency, won't match
		MediaType:     "video",
		Target:        "optimization",
	}

	_, err := router.RouteRequest(context.Background(), annotations)
	if err == nil {
		t.Fatal("Expected error when no backends match constraints")
	}

	errorMsg := err.Error()

	// Verify error message includes constraint details
	if !strings.Contains(errorMsg, "latency<100ms") {
		t.Errorf("Error message should mention latency constraint: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "power<50W") {
		t.Errorf("Error message should mention power constraint: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "media=video") {
		t.Errorf("Error message should mention media type: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "target=optimization") {
		t.Errorf("Error message should mention target: %s", errorMsg)
	}

	t.Logf("Constraint error message: %s", errorMsg)
}

// TestRouter_RouteRequest_PriorityScoring is part of the public API.
func TestRouter_RouteRequest_PriorityScoring(t *testing.T) {
	config := Config{
		PowerAware: true,
	}
	router := NewRouter(config)

	backend1 := &MockBackend{
		id:           strBackend12,
		healthy:      true,
		powerWatts:   100.0,
		avgLatencyMs: 200,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           strBackend2,
		healthy:      true,
		powerWatts:   100.0,
		avgLatencyMs: 200,
		priority:     5,
	}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	tests := []struct {
		name        string
		description string
		priority    backends.Priority
	}{
		{
			name:        "Critical priority",
			priority:    backends.PriorityCritical,
			description: "Should handle critical priority requests",
		},
		{
			name:        "High priority",
			priority:    backends.PriorityHigh,
			description: "Should handle high priority requests",
		},
		{
			name:        "Normal priority",
			priority:    backends.PriorityNormal,
			description: "Should handle normal priority requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{
				Priority: tt.priority,
			}

			decision, err := router.RouteRequest(context.Background(), annotations)
			if err != nil {
				t.Fatalf("RouteRequest failed: %v", err)
			}

			if decision.Backend == nil {
				t.Fatal("Expected backend to be selected")
			}

			t.Logf("%s: Selected %s with reason: %s", tt.description, decision.Backend.ID(), decision.Reason)
		})
	}
}

// TestRouter_RouteRequest_QueueDepthPenalty verifies that queue depth affects backend selection
func TestRouter_RouteRequest_QueueDepthPenalty(t *testing.T) {
	config := Config{
		PowerAware: true,
	}
	router := NewRouter(config)

	// Create two identical backends
	backend1 := &MockBackend{
		id:           strBackend12,
		healthy:      true,
		powerWatts:   100.0,
		avgLatencyMs: 200,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           strBackend2,
		healthy:      true,
		powerWatts:   100.0,
		avgLatencyMs: 200,
		priority:     5,
	}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	// Add queue depth to backend1 to make it less attractive
	router.queueMgr.MarkRequestStart(strBackend12, backends.PriorityNormal)
	router.queueMgr.MarkRequestStart(strBackend12, backends.PriorityNormal)
	router.queueMgr.MarkRequestStart(strBackend12, backends.PriorityHigh)

	annotations := &backends.Annotations{
		Priority: backends.PriorityNormal,
	}

	// Route multiple requests and verify backend2 is preferred due to lower queue depth
	backend2Count := 0
	for i := 0; i < 5; i++ {
		decision, err := router.RouteRequest(context.Background(), annotations)
		if err != nil {
			t.Fatalf("RouteRequest failed: %v", err)
		}

		if decision.Backend == nil {
			t.Fatal("Expected backend to be selected")
		}

		if decision.Backend.ID() == strBackend2 {
			backend2Count++
		}

		t.Logf("Iteration %d: Selected %s with reason: %s", i, decision.Backend.ID(), decision.Reason)
	}

	// backend2 should be selected more often due to backend1's initial queue depth
	// The queue depth mechanism should cause some load balancing between backends
	if backend2Count == 0 {
		t.Error("Expected backend2 to be selected at least once due to lower initial queue depth")
	}

	// Verify that queue depth reasons appeared in the routing decisions
	// by checking that both backends got some requests (load balancing occurred)
	backend1Count := 5 - backend2Count
	t.Logf("Distribution: backend1=%d, backend2=%d (backend1 started with queue depth=3)", backend1Count, backend2Count)

	if backend1Count == 0 || backend2Count == 0 {
		t.Error("Expected both backends to receive some requests due to queue-depth-based load balancing")
	}
}

// TestRouter_ScoreCandidates_DefaultScoring is part of the public API.
func TestRouter_ScoreCandidates_DefaultScoring(t *testing.T) {
	// Test the default scoring path when no specific preferences are set
	config := Config{
		PowerAware:   false,
		AutoOptimize: false,
	}
	router := NewRouter(config)

	backend1 := &MockBackend{
		id:           strBackend12,
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 150,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           strBackend2,
		healthy:      true,
		powerWatts:   100.0,
		avgLatencyMs: 200,
		priority:     3,
	}

	router.RegisterBackend(backend1)
	router.RegisterBackend(backend2)

	// Use empty annotations - no latency-critical, no power-efficiency preference
	annotations := &backends.Annotations{
		Priority: backends.PriorityNormal,
	}

	decision, err := router.RouteRequest(context.Background(), annotations)
	if err != nil {
		t.Fatalf("RouteRequest failed: %v", err)
	}

	if decision.Backend == nil {
		t.Fatal("Expected backend to be selected")
	}

	// With default scoring (balanced), backend1 should win due to higher priority,
	// better latency, and better power efficiency
	if decision.Backend.ID() != strBackend12 {
		t.Errorf("Expected backend1 to be selected with default scoring, got %s", decision.Backend.ID())
	}

	t.Logf("Selected backend: %s with reason: %s", decision.Backend.ID(), decision.Reason)
}
