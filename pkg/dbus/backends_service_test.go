package dbus

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/testutil"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/router"
)

// mockBackend implements the backends.Backend interface for testing
type mockBackend struct {
	modelsErr         error
	metrics           *backends.BackendMetrics
	name              string
	backendType       string
	hardware          string
	id                string
	models            []string
	preferredModels   []string
	supportedPatterns []string
	priority          int
	powerWatts        float64
	maxModelSizeGB    int
	avgLatencyMs      int32
	healthy           bool
	supportsEmbed     bool
	supportsStream    bool
	supportsGenerate  bool
}

// ID is part of the public API.
func (m *mockBackend) ID() string { return m.id }

// Type is part of the public API.
func (m *mockBackend) Type() string { return m.backendType }

// Name is part of the public API.
func (m *mockBackend) Name() string { return m.name }

// Hardware is part of the public API.
func (m *mockBackend) Hardware() string { return m.hardware }

// IsHealthy reports whether the condition holds.
func (m *mockBackend) IsHealthy() bool { return m.healthy }

// HealthCheck is part of the public API.
func (m *mockBackend) HealthCheck(_ context.Context) error { return nil }

// PowerWatts is part of the public API.
func (m *mockBackend) PowerWatts() float64 { return m.powerWatts }

// AvgLatencyMs is part of the public API.
func (m *mockBackend) AvgLatencyMs() int32 { return m.avgLatencyMs }

// Priority is part of the public API.
func (m *mockBackend) Priority() int { return m.priority }

// SupportsGenerate is part of the public API.
func (m *mockBackend) SupportsGenerate() bool { return m.supportsGenerate }

// SupportsStream is part of the public API.
func (m *mockBackend) SupportsStream() bool { return m.supportsStream }

// SupportsEmbed is part of the public API.
func (m *mockBackend) SupportsEmbed() bool { return m.supportsEmbed }

// GetMaxModelSizeGB is part of the public API.
func (m *mockBackend) GetMaxModelSizeGB() int { return m.maxModelSizeGB }

// GetSupportedModelPatterns is part of the public API.
func (m *mockBackend) GetSupportedModelPatterns() []string { return m.supportedPatterns }

// GetPreferredModels is part of the public API.
func (m *mockBackend) GetPreferredModels() []string { return m.preferredModels }

// GetMetrics is part of the public API.
func (m *mockBackend) GetMetrics() *backends.BackendMetrics { return m.metrics }

// SupportsModel is part of the public API.
func (m *mockBackend) SupportsModel(_ string) bool { return true }

// UpdateMetrics is part of the public API.
func (m *mockBackend) UpdateMetrics(_ int32, _ bool) {}

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

// ListModels is part of the public API.
func (m *mockBackend) ListModels(_ context.Context) ([]string, error) {
	if m.modelsErr != nil {
		return nil, m.modelsErr
	}
	return m.models, nil
}

// Generate is part of the public API.
func (m *mockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return nil, nil
}

// GenerateStream is part of the public API.
func (m *mockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

// Embed is part of the public API.
func (m *mockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// TestBackendInfo tests the BackendInfo structure
func TestBackendInfo(t *testing.T) {
	info := BackendInfo{
		ID:           strTestBackend,
		Name:         strTestBackend2,
		Type:         strOllama,
		Hardware:     strNpu,
		Healthy:      true,
		PowerWatts:   15.5,
		AvgLatencyMs: 100,
	}

	if info.ID != strTestBackend {
		t.Errorf("Expected ID 'test-backend', got '%s'", info.ID)
	}
	if info.Name != strTestBackend2 {
		t.Errorf("Expected Name 'Test Backend', got '%s'", info.Name)
	}
	if info.Type != strOllama {
		t.Errorf("Expected Type 'ollama', got '%s'", info.Type)
	}
	if info.Hardware != strNpu {
		t.Errorf("Expected Hardware 'npu', got '%s'", info.Hardware)
	}
	if !info.Healthy {
		t.Error("Expected Healthy to be true")
	}
	if info.PowerWatts != 15.5 {
		t.Errorf("Expected PowerWatts 15.5, got %f", info.PowerWatts)
	}
	if info.AvgLatencyMs != 100 {
		t.Errorf("Expected AvgLatencyMs 100, got %d", info.AvgLatencyMs)
	}
}

// TestBackendsServiceConstants tests the package constants
func TestBackendsServiceConstants(t *testing.T) {
	if backendsInterface != "ie.fio.OllamaProxy.Backends" {
		t.Errorf("Expected backendsInterface 'ie.fio.OllamaProxy.Backends', got '%s'", backendsInterface)
	}
	if backendsPath != "/com/anthropic/OllamaProxy/Backends" {
		t.Errorf("Expected backendsPath '/com/anthropic/OllamaProxy/Backends', got '%s'", backendsPath)
	}
}

// TestNewBackendsService tests service initialization
func TestNewBackendsService(t *testing.T) {
	r := router.NewRouter(router.Config{
		DefaultBackendID: "test",
		PowerAware:       true,
		AutoOptimize:     false,
	})

	// Note: This will fail in test environments without D-Bus, but we're testing structure
	svc, err := NewBackendsService(r)

	// In test environments, we expect D-Bus connection to fail
	// This is acceptable as we're testing structure initialization
	if err == nil && svc != nil {
		// Service was created (D-Bus available)
		if svc.router != r {
			t.Error("Expected router to be set")
		}
		if svc.conn == nil {
			t.Error("Expected conn to be set when service is created")
		}
		// Clean up if service was created
		svc.Stop()
	}

	// If error occurred, it should be D-Bus connection error
	if err != nil {
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	}
}

// TestListBackendsMethod tests the ListBackends D-Bus method
func TestListBackendsMethod(t *testing.T) {
	r := router.NewRouter(router.Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	})

	// Register mock backends
	backend1 := &mockBackend{
		id:           strBackend1,
		name:         "Backend 1",
		backendType:  strOllama,
		hardware:     strNpu,
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 50,
	}
	backend2 := &mockBackend{
		id:           strBackend2,
		name:         "Backend 2",
		backendType:  "openai",
		hardware:     "cloud",
		healthy:      false,
		powerWatts:   0.5,
		avgLatencyMs: 200,
	}

	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))

	// Create service without D-Bus connection (just for method testing)
	svc := &BackendsService{
		router: r,
	}

	// Test ListBackends method
	backends, dbusErr := svc.ListBackends()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if len(backends) != 2 {
		t.Fatalf("Expected 2 backends, got %d", len(backends))
	}

	// Find backend1 in results
	var found1, found2 bool
	for _, b := range backends {
		if b.ID == strBackend1 {
			found1 = true
			if b.Name != "Backend 1" {
				t.Errorf("Expected Name 'Backend 1', got '%s'", b.Name)
			}
			if b.Type != strOllama {
				t.Errorf("Expected Type 'ollama', got '%s'", b.Type)
			}
			if b.Hardware != strNpu {
				t.Errorf("Expected Hardware 'npu', got '%s'", b.Hardware)
			}
			if !b.Healthy {
				t.Error("Expected backend1 to be healthy")
			}
			if b.PowerWatts != 10.0 {
				t.Errorf("Expected PowerWatts 10.0, got %f", b.PowerWatts)
			}
			if b.AvgLatencyMs != 50 {
				t.Errorf("Expected AvgLatencyMs 50, got %d", b.AvgLatencyMs)
			}
		}
		if b.ID == strBackend2 {
			found2 = true
			if b.Healthy {
				t.Error("Expected backend2 to be unhealthy")
			}
		}
	}

	if !found1 {
		t.Error("backend1 not found in results")
	}
	if !found2 {
		t.Error("backend2 not found in results")
	}
}

// TestGetBackendDetailsMethod tests the GetBackendDetails D-Bus method
func TestGetBackendDetailsMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	backend := &mockBackend{
		id:                strTestBackend,
		name:              strTestBackend2,
		backendType:       strOllama,
		hardware:          strNpu,
		healthy:           true,
		powerWatts:        15.5,
		avgLatencyMs:      100,
		priority:          10,
		supportsGenerate:  true,
		supportsStream:    true,
		supportsEmbed:     false,
		maxModelSizeGB:    8,
		supportedPatterns: []string{"llama*", "mistral*"},
		preferredModels:   []string{strLlama38b, strMistral7b},
	}

	testutil.Check(t, r.RegisterBackend(backend))

	svc := &BackendsService{
		router: r,
	}

	// Test successful case
	details, dbusErr := svc.GetBackendDetails(strTestBackend)

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if details == nil {
		t.Fatal("Expected details map, got nil")
	}

	// Verify all fields
	if details["id"].Value().(string) != strTestBackend {
		t.Errorf("Expected id 'test-backend', got '%v'", details["id"].Value())
	}
	if details["name"].Value().(string) != strTestBackend2 {
		t.Errorf("Expected name 'Test Backend', got '%v'", details["name"].Value())
	}
	if details["type"].Value().(string) != strOllama {
		t.Errorf("Expected type 'ollama', got '%v'", details["type"].Value())
	}
	if details[strHardware].Value().(string) != strNpu {
		t.Errorf("Expected hardware 'npu', got '%v'", details[strHardware].Value())
	}
	if details["healthy"].Value().(bool) != true {
		t.Errorf("Expected healthy true, got %v", details["healthy"].Value())
	}
	if details["power_watts"].Value().(float64) != 15.5 {
		t.Errorf("Expected power_watts 15.5, got %v", details["power_watts"].Value())
	}
	if details["avg_latency_ms"].Value().(int32) != 100 {
		t.Errorf("Expected avg_latency_ms 100, got %v", details["avg_latency_ms"].Value())
	}
	if details["priority"].Value().(int) != 10 {
		t.Errorf("Expected priority 10, got %v", details["priority"].Value())
	}
	if details["supports_generate"].Value().(bool) != true {
		t.Errorf("Expected supports_generate true, got %v", details["supports_generate"].Value())
	}
	if details["supports_stream"].Value().(bool) != true {
		t.Errorf("Expected supports_stream true, got %v", details["supports_stream"].Value())
	}
	if details["supports_embed"].Value().(bool) != false {
		t.Errorf("Expected supports_embed false, got %v", details["supports_embed"].Value())
	}

	// Test not found case
	_, dbusErr = svc.GetBackendDetails("nonexistent")
	if dbusErr == nil {
		t.Error("Expected D-Bus error for nonexistent backend")
	}
}

// TestGetBackendMetricsMethod tests the GetBackendMetrics D-Bus method
func TestGetBackendMetricsMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	metrics := &backends.BackendMetrics{
		RequestCount:   100,
		SuccessCount:   95,
		ErrorCount:     5,
		TotalLatencyMs: 10000,
		AvgLatencyMs:   100,
		ErrorRate:      0.05,
		LoadedModels:   []string{strLlama38b, strMistral7b},
	}

	backend := &mockBackend{
		id:      strTestBackend,
		metrics: metrics,
	}

	testutil.Check(t, r.RegisterBackend(backend))

	svc := &BackendsService{
		router: r,
	}

	// Test successful case
	result, dbusErr := svc.GetBackendMetrics(strTestBackend)

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if result == nil {
		t.Fatal("Expected result map, got nil")
	}

	// Verify metrics
	if result["request_count"].Value().(int64) != 100 {
		t.Errorf("Expected request_count 100, got %v", result["request_count"].Value())
	}
	if result["success_count"].Value().(int64) != 95 {
		t.Errorf("Expected success_count 95, got %v", result["success_count"].Value())
	}
	if result["error_count"].Value().(int64) != 5 {
		t.Errorf("Expected error_count 5, got %v", result["error_count"].Value())
	}
	if result["total_latency_ms"].Value().(int64) != 10000 {
		t.Errorf("Expected total_latency_ms 10000, got %v", result["total_latency_ms"].Value())
	}
	if result["avg_latency_ms"].Value().(int32) != 100 {
		t.Errorf("Expected avg_latency_ms 100, got %v", result["avg_latency_ms"].Value())
	}
	errorRate := result["error_rate"].Value().(float64)
	if errorRate < 0.04 || errorRate > 0.06 {
		t.Errorf("Expected error_rate ~0.05, got %v", errorRate)
	}

	// Test not found case
	_, dbusErr = svc.GetBackendMetrics("nonexistent")
	if dbusErr == nil {
		t.Error("Expected D-Bus error for nonexistent backend")
	}
}

// TestGetSupportedModelsMethod tests the GetSupportedModels D-Bus method
func TestGetSupportedModelsMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	backend := &mockBackend{
		id:              strTestBackend,
		models:          []string{strLlama38b, strMistral7b, "codellama:13b"},
		preferredModels: []string{strLlama38b},
	}

	testutil.Check(t, r.RegisterBackend(backend))

	svc := &BackendsService{
		router: r,
	}

	// Test successful case
	models, dbusErr := svc.GetSupportedModels(strTestBackend)

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if len(models) != 3 {
		t.Fatalf("Expected 3 models, got %d", len(models))
	}

	expectedModels := map[string]bool{
		strLlama38b:     true,
		strMistral7b:    true,
		"codellama:13b": true,
	}

	for _, model := range models {
		if !expectedModels[model] {
			t.Errorf("Unexpected model: %s", model)
		}
	}

	// Test error fallback case
	backendWithError := &mockBackend{
		id:              "error-backend",
		modelsErr:       io.ErrUnexpectedEOF,
		preferredModels: []string{"fallback:model"},
	}
	testutil.Check(t, r.RegisterBackend(backendWithError))

	models, dbusErr = svc.GetSupportedModels("error-backend")
	if dbusErr == nil {
		t.Fatal("Expected D-Bus error when listing models fails")
	}

	if len(models) != 1 || models[0] != "fallback:model" {
		t.Errorf("Expected fallback to preferred models, got %v", models)
	}

	// Test not found case
	_, dbusErr = svc.GetSupportedModels("nonexistent")
	if dbusErr == nil {
		t.Error("Expected D-Bus error for nonexistent backend")
	}
}

// TestRefreshBackendStatusMethod tests the RefreshBackendStatus D-Bus method
func TestRefreshBackendStatusMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	backend1 := &mockBackend{
		id:      strBackend1,
		healthy: true,
	}
	backend2 := &mockBackend{
		id:      strBackend2,
		healthy: false,
	}

	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))

	svc := &BackendsService{
		router: r,
	}

	// Test refresh (without actual D-Bus connection)
	dbusErr := svc.RefreshBackendStatus()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	// The method should complete without errors even without D-Bus connection
	// (signals just won't be emitted)
}

// TestMakePropertyMap tests the property map creation
func TestMakePropertyMap(t *testing.T) {
	r := router.NewRouter(router.Config{})

	backend1 := &mockBackend{id: strBackend1, healthy: true}
	backend2 := &mockBackend{id: strBackend2, healthy: false}
	backend3 := &mockBackend{id: strBackend3, healthy: true}

	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))
	testutil.Check(t, r.RegisterBackend(backend3))

	svc := &BackendsService{
		router: r,
	}

	propMap := svc.makePropertyMap()

	if propMap == nil {
		t.Fatal("Expected property map, got nil")
	}

	if _, ok := propMap[backendsInterface]; !ok {
		t.Fatal("Expected backendsInterface in property map")
	}

	props := propMap[backendsInterface]

	// Check BackendCount
	if backendCountProp, ok := props["BackendCount"]; ok {
		if backendCountProp.Value.(int32) != 3 {
			t.Errorf("Expected BackendCount 3, got %v", backendCountProp.Value)
		}
		if backendCountProp.Writable {
			t.Error("Expected BackendCount to be read-only")
		}
	} else {
		t.Error("BackendCount property not found")
	}

	// Check HealthyCount
	if healthyCountProp, ok := props["HealthyCount"]; ok {
		if healthyCountProp.Value.(int32) != 2 {
			t.Errorf("Expected HealthyCount 2, got %v", healthyCountProp.Value)
		}
		if healthyCountProp.Writable {
			t.Error("Expected HealthyCount to be read-only")
		}
	} else {
		t.Error("HealthyCount property not found")
	}
}

// TestUpdateProperties tests the property update functionality
func TestUpdateProperties(t *testing.T) {
	r := router.NewRouter(router.Config{})

	backend1 := &mockBackend{id: strBackend1, healthy: true}
	backend2 := &mockBackend{id: strBackend2, healthy: true}

	testutil.Check(t, r.RegisterBackend(backend1))
	testutil.Check(t, r.RegisterBackend(backend2))

	svc := &BackendsService{
		router: r,
		props:  nil, // No actual D-Bus properties in test
	}

	// Should not panic even with nil props
	svc.updateProperties()

	// Change health status
	backend2.healthy = false

	// Update again - should not panic
	svc.updateProperties()
}

// TestStopService tests the Stop method
func TestStopService(_ *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &BackendsService{
		router: r,
		conn:   nil, // No actual connection in test
	}

	// Should not panic with nil connection
	svc.Stop()
}

// TestBackendInfoZeroValues tests BackendInfo with zero values
func TestBackendInfoZeroValues(t *testing.T) {
	info := BackendInfo{}

	if info.ID != "" {
		t.Errorf("Expected empty ID, got '%s'", info.ID)
	}
	if info.Healthy {
		t.Error("Expected Healthy to be false by default")
	}
	if info.PowerWatts != 0 {
		t.Errorf("Expected PowerWatts 0, got %f", info.PowerWatts)
	}
	if info.AvgLatencyMs != 0 {
		t.Errorf("Expected AvgLatencyMs 0, got %d", info.AvgLatencyMs)
	}
}
