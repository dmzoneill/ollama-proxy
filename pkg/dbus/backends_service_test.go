package dbus

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/router"
)

// mockBackend implements the backends.Backend interface for testing
type mockBackend struct {
	id                    string
	name                  string
	backendType           string
	hardware              string
	healthy               bool
	powerWatts            float64
	avgLatencyMs          int32
	priority              int
	supportsGenerate      bool
	supportsStream        bool
	supportsEmbed         bool
	maxModelSizeGB        int
	supportedPatterns     []string
	preferredModels       []string
	metrics               *backends.BackendMetrics
	models                []string
	modelsErr             error
}

func (m *mockBackend) ID() string                               { return m.id }
func (m *mockBackend) Type() string                             { return m.backendType }
func (m *mockBackend) Name() string                             { return m.name }
func (m *mockBackend) Hardware() string                         { return m.hardware }
func (m *mockBackend) IsHealthy() bool                          { return m.healthy }
func (m *mockBackend) HealthCheck(ctx context.Context) error    { return nil }
func (m *mockBackend) PowerWatts() float64                      { return m.powerWatts }
func (m *mockBackend) AvgLatencyMs() int32                      { return m.avgLatencyMs }
func (m *mockBackend) Priority() int                            { return m.priority }
func (m *mockBackend) SupportsGenerate() bool                   { return m.supportsGenerate }
func (m *mockBackend) SupportsStream() bool                     { return m.supportsStream }
func (m *mockBackend) SupportsEmbed() bool                      { return m.supportsEmbed }
func (m *mockBackend) GetMaxModelSizeGB() int                   { return m.maxModelSizeGB }
func (m *mockBackend) GetSupportedModelPatterns() []string      { return m.supportedPatterns }
func (m *mockBackend) GetPreferredModels() []string             { return m.preferredModels }
func (m *mockBackend) GetMetrics() *backends.BackendMetrics     { return m.metrics }
func (m *mockBackend) SupportsModel(modelName string) bool      { return true }
func (m *mockBackend) UpdateMetrics(latencyMs int32, success bool) {}
func (m *mockBackend) Start(ctx context.Context) error          { return nil }
func (m *mockBackend) Stop(ctx context.Context) error           { return nil }

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

func (m *mockBackend) ListModels(ctx context.Context) ([]string, error) {
	if m.modelsErr != nil {
		return nil, m.modelsErr
	}
	return m.models, nil
}

func (m *mockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return nil, nil
}

func (m *mockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

func (m *mockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// TestBackendInfo tests the BackendInfo structure
func TestBackendInfo(t *testing.T) {
	info := BackendInfo{
		ID:           "test-backend",
		Name:         "Test Backend",
		Type:         "ollama",
		Hardware:     "npu",
		Healthy:      true,
		PowerWatts:   15.5,
		AvgLatencyMs: 100,
	}

	if info.ID != "test-backend" {
		t.Errorf("Expected ID 'test-backend', got '%s'", info.ID)
	}
	if info.Name != "Test Backend" {
		t.Errorf("Expected Name 'Test Backend', got '%s'", info.Name)
	}
	if info.Type != "ollama" {
		t.Errorf("Expected Type 'ollama', got '%s'", info.Type)
	}
	if info.Hardware != "npu" {
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
		DefaultBackendID: "backend1",
		PowerAware:       true,
	})

	// Register mock backends
	backend1 := &mockBackend{
		id:           "backend1",
		name:         "Backend 1",
		backendType:  "ollama",
		hardware:     "npu",
		healthy:      true,
		powerWatts:   10.0,
		avgLatencyMs: 50,
	}
	backend2 := &mockBackend{
		id:           "backend2",
		name:         "Backend 2",
		backendType:  "openai",
		hardware:     "cloud",
		healthy:      false,
		powerWatts:   0.5,
		avgLatencyMs: 200,
	}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

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
		if b.ID == "backend1" {
			found1 = true
			if b.Name != "Backend 1" {
				t.Errorf("Expected Name 'Backend 1', got '%s'", b.Name)
			}
			if b.Type != "ollama" {
				t.Errorf("Expected Type 'ollama', got '%s'", b.Type)
			}
			if b.Hardware != "npu" {
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
		if b.ID == "backend2" {
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
		id:                "test-backend",
		name:              "Test Backend",
		backendType:       "ollama",
		hardware:          "npu",
		healthy:           true,
		powerWatts:        15.5,
		avgLatencyMs:      100,
		priority:          10,
		supportsGenerate:  true,
		supportsStream:    true,
		supportsEmbed:     false,
		maxModelSizeGB:    8,
		supportedPatterns: []string{"llama*", "mistral*"},
		preferredModels:   []string{"llama3:8b", "mistral:7b"},
	}

	r.RegisterBackend(backend)

	svc := &BackendsService{
		router: r,
	}

	// Test successful case
	details, dbusErr := svc.GetBackendDetails("test-backend")

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if details == nil {
		t.Fatal("Expected details map, got nil")
	}

	// Verify all fields
	if details["id"].Value().(string) != "test-backend" {
		t.Errorf("Expected id 'test-backend', got '%v'", details["id"].Value())
	}
	if details["name"].Value().(string) != "Test Backend" {
		t.Errorf("Expected name 'Test Backend', got '%v'", details["name"].Value())
	}
	if details["type"].Value().(string) != "ollama" {
		t.Errorf("Expected type 'ollama', got '%v'", details["type"].Value())
	}
	if details["hardware"].Value().(string) != "npu" {
		t.Errorf("Expected hardware 'npu', got '%v'", details["hardware"].Value())
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
		RequestCount:      100,
		SuccessCount:      95,
		ErrorCount:        5,
		TotalLatencyMs:    10000,
		AvgLatencyMs:      100,
		ErrorRate:         0.05,
		LoadedModels:      []string{"llama3:8b", "mistral:7b"},
	}

	backend := &mockBackend{
		id:      "test-backend",
		metrics: metrics,
	}

	r.RegisterBackend(backend)

	svc := &BackendsService{
		router: r,
	}

	// Test successful case
	result, dbusErr := svc.GetBackendMetrics("test-backend")

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
		id:              "test-backend",
		models:          []string{"llama3:8b", "mistral:7b", "codellama:13b"},
		preferredModels: []string{"llama3:8b"},
	}

	r.RegisterBackend(backend)

	svc := &BackendsService{
		router: r,
	}

	// Test successful case
	models, dbusErr := svc.GetSupportedModels("test-backend")

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if len(models) != 3 {
		t.Fatalf("Expected 3 models, got %d", len(models))
	}

	expectedModels := map[string]bool{
		"llama3:8b":     true,
		"mistral:7b":    true,
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
	r.RegisterBackend(backendWithError)

	models, dbusErr = svc.GetSupportedModels("error-backend")
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error (should fallback), got %v", dbusErr)
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
		id:      "backend1",
		healthy: true,
	}
	backend2 := &mockBackend{
		id:      "backend2",
		healthy: false,
	}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

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

	backend1 := &mockBackend{id: "backend1", healthy: true}
	backend2 := &mockBackend{id: "backend2", healthy: false}
	backend3 := &mockBackend{id: "backend3", healthy: true}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)
	r.RegisterBackend(backend3)

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

	backend1 := &mockBackend{id: "backend1", healthy: true}
	backend2 := &mockBackend{id: "backend2", healthy: true}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

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
func TestStopService(t *testing.T) {
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
