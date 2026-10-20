package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/thermal"
)

// mockBackendForRouter implements backends.Backend for testing
type mockBackendForRouter struct {
	supportsModelFunc func(string) bool
	id                string
	hardware          string
	healthy           bool
}

// ID is part of the public API.
func (m *mockBackendForRouter) ID() string { return m.id }

// Type is part of the public API.
func (m *mockBackendForRouter) Type() string { return "mock" }

// Name is part of the public API.
func (m *mockBackendForRouter) Name() string { return m.id }

// Hardware is part of the public API.
func (m *mockBackendForRouter) Hardware() string { return m.hardware }

// Endpoint is part of the public API.
func (m *mockBackendForRouter) Endpoint() string { return "http://mock" }

// IsHealthy reports whether the condition holds.
func (m *mockBackendForRouter) IsHealthy() bool {
	return m.healthy
}

// PowerWatts is part of the public API.
func (m *mockBackendForRouter) PowerWatts() float64 { return 10.0 }

// AvgLatencyMs is part of the public API.
func (m *mockBackendForRouter) AvgLatencyMs() int32 { return 100 }

// Priority is part of the public API.
func (m *mockBackendForRouter) Priority() int { return 1 }

// SupportsGenerate is part of the public API.
func (m *mockBackendForRouter) SupportsGenerate() bool { return true }

// SupportsEmbed is part of the public API.
func (m *mockBackendForRouter) SupportsEmbed() bool { return true }

// SupportsStream is part of the public API.
func (m *mockBackendForRouter) SupportsStream() bool { return true }

// SupportsModel is part of the public API.
func (m *mockBackendForRouter) SupportsModel(model string) bool {
	if m.supportsModelFunc != nil {
		return m.supportsModelFunc(model)
	}
	return true
}

// Start is part of the public API.
func (m *mockBackendForRouter) Start(_ context.Context) error { return nil }

// Stop is part of the public API.
func (m *mockBackendForRouter) Stop(_ context.Context) error { return nil }

// Multimedia capability methods
func (m *mockBackendForRouter) SupportsAudioToText() bool { return false }

// SupportsTextToAudio is part of the public API.
func (m *mockBackendForRouter) SupportsTextToAudio() bool { return false }

// SupportsImageToText is part of the public API.
func (m *mockBackendForRouter) SupportsImageToText() bool { return false }

// SupportsTextToImage is part of the public API.
func (m *mockBackendForRouter) SupportsTextToImage() bool { return false }

// SupportsVideoToText is part of the public API.
func (m *mockBackendForRouter) SupportsVideoToText() bool { return false }

// SupportsTextToVideo is part of the public API.
func (m *mockBackendForRouter) SupportsTextToVideo() bool { return false }

// TranscribeAudio is part of the public API.
func (m *mockBackendForRouter) TranscribeAudio(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// TranscribeAudioStream is part of the public API.
func (m *mockBackendForRouter) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeech is part of the public API.
func (m *mockBackendForRouter) SynthesizeSpeech(_ context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeechStream is part of the public API.
func (m *mockBackendForRouter) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeImage is part of the public API.
func (m *mockBackendForRouter) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImage is part of the public API.
func (m *mockBackendForRouter) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImageStream is part of the public API.
func (m *mockBackendForRouter) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideo is part of the public API.
func (m *mockBackendForRouter) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideoStream is part of the public API.
func (m *mockBackendForRouter) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideo is part of the public API.
func (m *mockBackendForRouter) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideoStream is part of the public API.
func (m *mockBackendForRouter) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// HealthCheck is part of the public API.
func (m *mockBackendForRouter) HealthCheck(_ context.Context) error { return nil }

// Generate is part of the public API.
func (m *mockBackendForRouter) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	// Return a basic response for testing
	return &backends.GenerateResponse{
		Response: "Test response from " + m.id,
		Stats: &backends.GenerationStats{
			TotalTimeMs:     100,
			TokensGenerated: 20,
			TokensPerSecond: 10,
		},
	}, nil
}

// GenerateStream is part of the public API.
func (m *mockBackendForRouter) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

// Embed is part of the public API.
func (m *mockBackendForRouter) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// ListModels is part of the public API.
func (m *mockBackendForRouter) ListModels(_ context.Context) ([]string, error) { return nil, nil }

// GetMetrics is part of the public API.
func (m *mockBackendForRouter) GetMetrics() *backends.BackendMetrics {
	return &backends.BackendMetrics{}
}

// GetMaxModelSizeGB is part of the public API.
func (m *mockBackendForRouter) GetMaxModelSizeGB() int { return 16 }

// GetSupportedModelPatterns is part of the public API.
func (m *mockBackendForRouter) GetSupportedModelPatterns() []string { return nil }

// GetPreferredModels is part of the public API.
func (m *mockBackendForRouter) GetPreferredModels() []string { return nil }

// UpdateMetrics is part of the public API.
func (m *mockBackendForRouter) UpdateMetrics(_ int32, _ bool) {}

// Recv is part of the public API.
func (m *mockBackendForRouter) Recv() (*backends.StreamChunk, error) { return nil, nil }

// TestForwardingRouter_BasicForwarding is part of the public API.
func TestForwardingRouter_BasicForwarding(t *testing.T) {
	// Create mock backends
	npuBackend := &mockBackendForRouter{
		id:       strOllamaNpu,
		hardware: strNpu,
		healthy:  true,
	}

	intelBackend := &mockBackendForRouter{
		id:       strOllamaIntel,
		hardware: strIgpu,
		healthy:  true,
	}

	nvidiaBackend := &mockBackendForRouter{
		id:       strOllamaNvidia,
		hardware: strNvidia,
		healthy:  true,
	}

	// Create base router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(nvidiaBackend); err != nil {
		t.Fatal(err)
	}

	// Create forwarding router
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel, strOllamaNvidia},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	// Test basic forwarding
	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.FinalResponse == "" {
		t.Error("FinalResponse is empty")
	}

	if result.FinalBackend == nil {
		t.Error("FinalBackend is nil")
	}

	if result.TotalAttempts == 0 {
		t.Error("TotalAttempts should be > 0")
	}

	t.Logf("Result: %d attempts, final backend: %s, forwarded: %v",
		result.TotalAttempts, result.FinalBackend.ID(), result.Forwarded)
}

// TestForwardingRouter_ModelCompatibility is part of the public API.
func TestForwardingRouter_ModelCompatibility(t *testing.T) {
	// Create backends with different model support
	npuBackend := &mockBackendForRouter{
		id:       strOllamaNpu,
		hardware: strNpu,
		healthy:  true,
		supportsModelFunc: func(model string) bool {
			// NPU only supports small models
			return model == strQwen2505b
		},
	}

	intelBackend := &mockBackendForRouter{
		id:       strOllamaIntel,
		hardware: strIgpu,
		healthy:  true,
		supportsModelFunc: func(model string) bool {
			// Intel supports small and medium models
			return model == strQwen2505b || model == strLlama37b
		},
	}

	nvidiaBackend := &mockBackendForRouter{
		id:       strOllamaNvidia,
		hardware: strNvidia,
		healthy:  true,
		supportsModelFunc: func(_ string) bool {
			// NVIDIA supports all models
			return true
		},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(nvidiaBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel, strOllamaNvidia},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	tests := []struct {
		name            string
		model           string
		expectedBackend string
		description     string
	}{
		{
			name:            "Small model stays on NPU",
			model:           strQwen2505b,
			expectedBackend: strOllamaNpu,
			description:     "qwen2.5:0.5b supported by NPU, shouldn't forward",
		},
		{
			name:            "Medium model forwards to Intel",
			model:           strLlama37b,
			expectedBackend: strOllamaIntel,
			description:     "llama3:7b not supported by NPU, should forward to Intel",
		},
		{
			name:            "Large model forwards to NVIDIA",
			model:           strLlama370b,
			expectedBackend: strOllamaNvidia,
			description:     "llama3:70b only supported by NVIDIA, should forward there",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := forwardingRouter.GenerateWithForwarding(
				ctx,
				"Test prompt",
				tt.model,
				&backends.Annotations{},
			)

			if err != nil {
				t.Fatalf("GenerateWithForwarding failed: %v", err)
			}

			if result.FinalBackend == nil {
				t.Fatal("FinalBackend is nil")
			}

			if result.FinalBackend.ID() != tt.expectedBackend {
				t.Errorf("%s: got backend %s, want %s",
					tt.description, result.FinalBackend.ID(), tt.expectedBackend)
			}

			t.Logf("%s: used %s after %d attempts",
				tt.name, result.FinalBackend.ID(), result.TotalAttempts)
		})
	}
}

// TestForwardingRouter_ThermalRespect is part of the public API.
func TestForwardingRouter_ThermalRespect(t *testing.T) {
	// Create backends
	npuBackend := &mockBackendForRouter{
		id:       strOllamaNpu,
		hardware: strNpu,
		healthy:  false, // Mark as unhealthy (thermal issues)
	}

	intelBackend := &mockBackendForRouter{
		id:       strOllamaIntel,
		hardware: strIgpu,
		healthy:  true,
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel},
		RespectThermalLimits: true, // Enable thermal checking
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should skip unhealthy NPU and use Intel
	if result.FinalBackend == nil {
		t.Fatal("FinalBackend is nil")
	}

	if result.FinalBackend.ID() != strOllamaIntel {
		t.Errorf("Expected to use ollama-intel (NPU is unhealthy), got %s",
			result.FinalBackend.ID())
	}

	// Check that NPU was skipped
	if len(result.Attempts) == 0 {
		t.Error("No attempts recorded")
	}

	npuSkipped := false
	for _, attempt := range result.Attempts {
		if attempt.BackendID == strOllamaNpu && !attempt.Success {
			npuSkipped = true
			t.Logf("NPU skipped: %s", attempt.SkipReason)
		}
	}

	if !npuSkipped {
		t.Error("Expected NPU to be skipped due to thermal issues")
	}
}

// TestForwardingRouter_EscalationPath is part of the public API.
func TestForwardingRouter_EscalationPath(t *testing.T) {
	// Create backends
	mockBackends := []*mockBackendForRouter{
		{id: strBackend1, hardware: strNpu, healthy: true},
		{id: strBackend22, hardware: strIgpu, healthy: true},
		{id: strBackend3, hardware: strNvidia, healthy: true},
	}

	routerCfg := &Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	for _, b := range mockBackends {
		if err := baseRouter.RegisterBackend(b); err != nil {
			t.Fatal(err)
		}
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{strBackend1, strBackend22, strBackend3},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	// Verify escalation path was followed
	if len(result.Attempts) == 0 {
		t.Error("No attempts recorded")
	}

	t.Logf("Total attempts: %d", result.TotalAttempts)
	for i, attempt := range result.Attempts {
		t.Logf("Attempt %d: backend=%s, success=%v, confidence=%.2f",
			i+1, attempt.BackendID, attempt.Success, attempt.Confidence.Overall)
	}

	// Should have used at least one backend
	if result.FinalBackend == nil {
		t.Error("No final backend selected")
	}
}

// TestForwardingRouter_MaxRetries is part of the public API.
func TestForwardingRouter_MaxRetries(t *testing.T) {
	// Create backends
	backend1 := &mockBackendForRouter{id: strBackend1, hardware: strNpu, healthy: true}
	backend2 := &mockBackendForRouter{id: strBackend22, hardware: strIgpu, healthy: true}

	routerCfg := &Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(backend2); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           1, // Only 1 retry
		EscalationPath:       []string{strBackend1, strBackend22},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should respect MaxRetries limit
	if result.TotalAttempts > forwardingCfg.MaxRetries {
		t.Errorf("TotalAttempts = %d, should not exceed MaxRetries = %d",
			result.TotalAttempts, forwardingCfg.MaxRetries)
	}

	t.Logf("Attempts: %d (max: %d)", result.TotalAttempts, forwardingCfg.MaxRetries)
}

// TestIsSmallModel is part of the public API.
func TestIsSmallModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected bool
	}{
		{name: "0.5b model", model: strQwen2505b, expected: true},
		{name: "1.5b model", model: "llama:1.5b", expected: true},
		{name: "tiny model", model: "tiny-llama", expected: true},
		{name: "mini model", model: "mini-gpt", expected: true},
		{name: "7b model", model: strLlama37b, expected: false},
		{name: "70b model", model: strLlama370b, expected: false},
		{name: "regular model", model: "gpt-4", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSmallModel(tt.model)
			if result != tt.expected {
				t.Errorf("isSmallModel(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

// TestContains is part of the public API.
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{name: "exact match", s: strTest, substr: strTest, expected: true},
		{name: "prefix", s: "testing", substr: strTest, expected: true},
		{name: "suffix", s: "contest", substr: strTest, expected: true},
		{name: "middle", s: "attested", substr: strTest, expected: true},
		{name: "not found", s: "example", substr: strTest, expected: false},
		{name: "empty substring", s: strTest, substr: "", expected: false}, // Implementation requires len(substr) > 0
		{name: "empty string", s: "", substr: strTest, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestFindInString is part of the public API.
func TestFindInString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{name: "found at start", s: strHelloWorld, substr: "hello", expected: true},
		{name: "found at end", s: strHelloWorld, substr: "world", expected: true},
		{name: "found in middle", s: strHelloWorld, substr: "lo wo", expected: true},
		{name: "not found", s: strHelloWorld, substr: "xyz", expected: false},
		{name: "single char found", s: "abcdef", substr: "c", expected: true},
		{name: "single char not found", s: "abcdef", substr: "z", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findInString(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("findInString(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestDefaultForwardingConfig is part of the public API.
func TestDefaultForwardingConfig(t *testing.T) {
	config := DefaultForwardingConfig()

	if config.Enabled != true {
		t.Error("DefaultForwardingConfig should have Enabled = true")
	}

	if config.MinConfidence <= 0 || config.MinConfidence > 1 {
		t.Errorf("MinConfidence should be between 0 and 1, got %f", config.MinConfidence)
	}

	if config.MaxRetries <= 0 {
		t.Errorf("MaxRetries should be > 0, got %d", config.MaxRetries)
	}

	if config.RespectThermalLimits != true {
		t.Error("RespectThermalLimits should be true")
	}

	if config.ReturnBestAttempt != true {
		t.Error("ReturnBestAttempt should be true")
	}

	// EscalationPath is empty by design (set dynamically)
	if config.EscalationPath == nil {
		t.Error("EscalationPath should not be nil")
	}
}

// TestSetEscalationPath is part of the public API.
func TestSetEscalationPath(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strTest})
	forwardingRouter := NewForwardingRouter(router, nil, DefaultForwardingConfig())

	newPath := []string{strBackend12, strBackend2, "backend3"}
	forwardingRouter.SetEscalationPath(newPath)

	// Verify path was set (indirect verification through config)
	if forwardingRouter.config.EscalationPath == nil {
		t.Error("SetEscalationPath should update the escalation path")
	}
}

// TestSetMinConfidence is part of the public API.
func TestSetMinConfidence(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strTest})
	forwardingRouter := NewForwardingRouter(router, nil, DefaultForwardingConfig())

	newConfidence := 0.85
	forwardingRouter.SetMinConfidence(newConfidence)

	if forwardingRouter.config.MinConfidence != newConfidence {
		t.Errorf("SetMinConfidence should set to %f, got %f", newConfidence, forwardingRouter.config.MinConfidence)
	}
}

// TestForwardingRouter_BuildEscalationPath is part of the public API.
func TestForwardingRouter_BuildEscalationPath(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strTest})
	fr := NewForwardingRouter(router, nil, DefaultForwardingConfig())

	// Test via GenerateWithForwarding which calls buildEscalationPath
	npuBackend := &mockBackendForRouter{
		id:       strOllamaNpu,
		hardware: strNpu,
		healthy:  true,
	}
	intelBackend := &mockBackendForRouter{
		id:       strOllamaIntel,
		hardware: strIgpu,
		healthy:  true,
	}
	nvidiaBackend := &mockBackendForRouter{
		id:       strOllamaNvidia,
		hardware: strNvidia,
		healthy:  true,
	}

	if err := router.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := router.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}
	if err := router.RegisterBackend(nvidiaBackend); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Test with small model - should try NPU first
	result, err := fr.GenerateWithForwarding(ctx, strTest, strQwen2505b, &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	// Test with large model
	result2, err := fr.GenerateWithForwarding(ctx, strTest, strLlama370b, &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding with large model failed: %v", err)
	}
	if result2 == nil {
		t.Fatal("Result2 is nil")
	}
}

// TestForwardingRouter_FindBackend is part of the public API.
func TestForwardingRouter_FindBackend(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strTest})

	backend1 := &mockBackendForRouter{
		id:       strBackend1,
		hardware: strNpu,
		healthy:  true,
	}

	if err := router.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}

	fr := NewForwardingRouter(router, nil, DefaultForwardingConfig())

	// Test finding via GenerateWithForwarding with specific escalation path
	fr.SetEscalationPath([]string{strBackend1})

	ctx := context.Background()
	result, err := fr.GenerateWithForwarding(ctx, strTest, "test-model", &backends.Annotations{})

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	if result.FinalBackend.ID() != strBackend1 {
		t.Errorf("Expected backend-1, got %s", result.FinalBackend.ID())
	}
}

// TestNewForwardingRouter_NilConfig is part of the public API.
func TestNewForwardingRouter_NilConfig(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: strTest})

	// Pass nil config - should use defaults
	fr := NewForwardingRouter(router, nil, nil)

	if fr == nil {
		t.Fatal("NewForwardingRouter returned nil")
	}

	if fr.config == nil {
		t.Error("Config should not be nil after NewForwardingRouter with nil config")
	}

	// Verify default config was applied
	if !fr.config.Enabled {
		t.Error("Default config should have Enabled=true")
	}

	if fr.config.MinConfidence <= 0 {
		t.Error("Default config should have MinConfidence > 0")
	}

	t.Logf("NewForwardingRouter with nil config created with defaults: MinConfidence=%.2f, MaxRetries=%d",
		fr.config.MinConfidence, fr.config.MaxRetries)
}

// TestForwardingRouter_FindBackend_WithThermalRouter is part of the public API.
func TestForwardingRouter_FindBackend_WithThermalRouter(t *testing.T) {
	// Create base router
	baseRouter := NewRouter(Config{DefaultBackendID: strBaseBackend})

	baseBackend := &mockBackendForRouter{
		id:       strBaseBackend,
		hardware: strCPU,
		healthy:  true,
	}
	if err := baseRouter.RegisterBackend(baseBackend); err != nil {
		t.Fatal(err)
	}

	// Create thermal router with different backend
	thermalConfig := &thermal.Config{
		TempWarning:  70.0,
		TempCritical: 85.0,
		FanQuiet:     30,
		FanLoud:      80,
	}
	monitor := thermal.NewThermalMonitor(thermalConfig, 0)
	thermalRouterCfg := Config{
		DefaultBackendID: strThermalBackend,
		PowerAware:       true,
	}
	thermalRouter := NewThermalRouter(thermalRouterCfg, monitor)

	thermalBackend := &mockBackendForRouter{
		id:       strThermalBackend,
		hardware: strGpu,
		healthy:  true,
	}
	if err := thermalRouter.RegisterBackend(thermalBackend); err != nil {
		t.Fatal(err)
	}

	// Create forwarding router with both routers
	forwardingCfg := DefaultForwardingConfig()
	fr := NewForwardingRouter(baseRouter, thermalRouter, forwardingCfg)

	// Test finding backend from base router
	fr.SetEscalationPath([]string{strBaseBackend})
	ctx := context.Background()
	result1, err := fr.GenerateWithForwarding(ctx, strTest, "test-model", &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding for base backend failed: %v", err)
	}
	if result1.FinalBackend.ID() != strBaseBackend {
		t.Errorf("Expected base-backend, got %s", result1.FinalBackend.ID())
	}

	// Test finding backend from thermal router
	fr.SetEscalationPath([]string{strThermalBackend})
	result2, err := fr.GenerateWithForwarding(ctx, strTest, "test-model", &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding for thermal backend failed: %v", err)
	}
	if result2.FinalBackend.ID() != strThermalBackend {
		t.Errorf("Expected thermal-backend, got %s", result2.FinalBackend.ID())
	}

	t.Logf("Successfully found backends in both base and thermal routers")
}

// mockStreamReader implements backends.StreamReader for testing
type mockStreamReader struct {
	chunks []string
	index  int
}

// Recv is part of the public API.
func (m *mockStreamReader) Recv() (*backends.StreamChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, nil // EOF
	}
	chunk := &backends.StreamChunk{
		Token: m.chunks[m.index],
	}
	m.index++
	return chunk, nil
}

// Close is part of the public API.
func (m *mockStreamReader) Close() error {
	return nil
}

// mockBackendWithStream extends mockBackendForRouter to support streaming
type mockBackendWithStream struct {
	mockBackendForRouter
	streamError  error
	streamChunks []string
}

// GenerateStream is part of the public API.
func (m *mockBackendWithStream) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	return &mockStreamReader{chunks: m.streamChunks}, nil
}

// TestGenerateWithForwardingStream_Success is part of the public API.
func TestGenerateWithForwardingStream_Success(t *testing.T) {
	// Create backends with streaming support
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaNpu,
			hardware: strNpu,
			healthy:  true,
		},
		streamChunks: []string{"Hello", " ", "world", "!"},
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaIntel,
			hardware: strIgpu,
			healthy:  true,
		},
		streamChunks: []string{"Test", " ", strStream},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		strQwen2505b, // Use small model to trigger NPU selection
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwardingStream failed: %v", err)
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	if backend == nil {
		t.Fatal("Backend is nil")
	}

	if backend.ID() != strOllamaNpu {
		t.Errorf("Expected backend ollama-npu, got %s", backend.ID())
	}

	// Read stream chunks
	var receivedChunks []string
	for {
		chunk, err := stream.Recv()
		if err != nil {
			t.Fatalf("Error reading stream: %v", err)
		}
		if chunk == nil {
			break // EOF
		}
		receivedChunks = append(receivedChunks, chunk.Token)
	}

	if len(receivedChunks) != 4 {
		t.Errorf("Expected 4 chunks, got %d", len(receivedChunks))
	}

	t.Logf("Received %d chunks from %s", len(receivedChunks), backend.ID())
}

// TestGenerateWithForwardingStream_ThermalRespect is part of the public API.
func TestGenerateWithForwardingStream_ThermalRespect(t *testing.T) {
	// Create backends
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaNpu,
			hardware: strNpu,
			healthy:  false, // Mark as unhealthy
		},
		streamChunks: []string{"NPU", " ", strStream},
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaIntel,
			hardware: strIgpu,
			healthy:  true,
		},
		streamChunks: []string{strIntel, " ", strStream},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel},
		RespectThermalLimits: true, // Enable thermal checking
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		strQwen2505b, // Use small model so NPU is in escalation path
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwardingStream failed: %v", err)
	}

	// Should skip unhealthy NPU and use Intel
	if backend == nil {
		t.Fatal("Backend is nil")
	}

	if backend.ID() != strOllamaIntel {
		t.Errorf("Expected to use ollama-intel (NPU is unhealthy), got %s", backend.ID())
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	t.Logf("Successfully skipped unhealthy NPU and used %s", backend.ID())
}

// TestGenerateWithForwardingStream_ModelCompatibility is part of the public API.
func TestGenerateWithForwardingStream_ModelCompatibility(t *testing.T) {
	// Create backends with different model support
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaNpu,
			hardware: strNpu,
			healthy:  true,
			supportsModelFunc: func(model string) bool {
				return model == strQwen2505b
			},
		},
		streamChunks: []string{"NPU", " ", strResponse},
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaIntel,
			hardware: strIgpu,
			healthy:  true,
			supportsModelFunc: func(model string) bool {
				return model == strQwen2505b || model == strLlama37b
			},
		},
		streamChunks: []string{strIntel, " ", strResponse},
	}

	nvidiaBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaNvidia,
			hardware: strNvidia,
			healthy:  true,
			supportsModelFunc: func(_ string) bool {
				return true // Supports all models
			},
		},
		streamChunks: []string{"NVIDIA", " ", strResponse},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(nvidiaBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel, strOllamaNvidia},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	tests := []struct {
		name            string
		model           string
		expectedBackend string
	}{
		{
			name:            "Small model uses NPU",
			model:           strQwen2505b,
			expectedBackend: strOllamaNpu,
		},
		{
			name:            "Medium model uses Intel",
			model:           strLlama37b,
			expectedBackend: strOllamaIntel,
		},
		{
			name:            "Large model uses NVIDIA",
			model:           strLlama370b,
			expectedBackend: strOllamaNvidia,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
				ctx,
				"Test prompt",
				tt.model,
				&backends.Annotations{},
			)

			if err != nil {
				t.Fatalf("GenerateWithForwardingStream failed: %v", err)
			}

			if backend == nil {
				t.Fatal("Backend is nil")
			}

			if backend.ID() != tt.expectedBackend {
				t.Errorf("Expected backend %s, got %s", tt.expectedBackend, backend.ID())
			}

			if stream == nil {
				t.Fatal("Stream is nil")
			}

			t.Logf("%s: used %s for model %s", tt.name, backend.ID(), tt.model)
		})
	}
}

// TestGenerateWithForwardingStream_NoSuitableBackend is part of the public API.
func TestGenerateWithForwardingStream_NoSuitableBackend(t *testing.T) {
	// Create backend that doesn't support the requested model
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaNpu,
			hardware: strNpu,
			healthy:  true,
			supportsModelFunc: func(_ string) bool {
				return false // Doesn't support any models
			},
		},
		streamChunks: []string{"Should", " ", "not", " ", "be", " ", "called"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           1,
		EscalationPath:       []string{strOllamaNpu},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		"unsupported-model",
		&backends.Annotations{},
	)

	// Should return error when no suitable backend found
	if err == nil {
		t.Error("Expected error when no suitable backend found, got nil")
	}

	if stream != nil {
		t.Error("Stream should be nil when no suitable backend found")
	}

	if backend != nil {
		t.Error("Backend should be nil when no suitable backend found")
	}

	if err != nil && err.Error() != "no suitable backend found for streaming" {
		t.Logf("Got expected error: %v", err)
	}
}

// TestGenerateWithForwardingStream_BackendGenerateStreamError is part of the public API.
func TestGenerateWithForwardingStream_BackendGenerateStreamError(t *testing.T) {
	// Create backend that returns error on GenerateStream
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaNpu,
			hardware: strNpu,
			healthy:  true,
		},
		streamError: fmt.Errorf("backend unavailable"),
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       strOllamaIntel,
			hardware: strIgpu,
			healthy:  true,
		},
		streamChunks: []string{strIntel, " ", "fallback"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(npuBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(intelBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		strQwen2505b, // Use small model so NPU is tried first
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwardingStream should succeed with fallback: %v", err)
	}

	// Should fall back to Intel when NPU fails
	if backend == nil {
		t.Fatal("Backend is nil")
	}

	if backend.ID() != strOllamaIntel {
		t.Errorf("Expected fallback to ollama-intel, got %s", backend.ID())
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	t.Logf("Successfully fell back from NPU to Intel after GenerateStream error")
}

// TestGenerateWithForwardingStream_EscalationPath is part of the public API.
func TestGenerateWithForwardingStream_EscalationPath(t *testing.T) {
	// Create multiple backends with names that match buildEscalationPath expectations
	mockBackends := []*mockBackendWithStream{
		{
			mockBackendForRouter: mockBackendForRouter{
				id:       strOllamaNpu,
				hardware: strNpu,
				healthy:  true,
			},
			streamChunks: []string{strBackend, " ", "1"},
		},
		{
			mockBackendForRouter: mockBackendForRouter{
				id:       strOllamaIntel,
				hardware: strIgpu,
				healthy:  true,
			},
			streamChunks: []string{strBackend, " ", "2"},
		},
		{
			mockBackendForRouter: mockBackendForRouter{
				id:       strOllamaNvidia,
				hardware: strNvidia,
				healthy:  true,
			},
			streamChunks: []string{strBackend, " ", "3"},
		},
	}

	routerCfg := &Config{
		DefaultBackendID: strOllamaNpu,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	for _, b := range mockBackends {
		if err := baseRouter.RegisterBackend(b); err != nil {
			t.Fatal(err)
		}
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{strOllamaNpu, strOllamaIntel, strOllamaNvidia},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		strQwen2505b, // Use small model to match escalation path
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwardingStream failed: %v", err)
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	if backend == nil {
		t.Fatal("Backend is nil")
	}

	// Should use first available backend in escalation path
	t.Logf("Used backend: %s", backend.ID())
}

// mockBackendWithError extends mockBackendForRouter to return errors on Generate
type mockBackendWithError struct {
	generateError error
	mockBackendForRouter
}

// Generate is part of the public API.
func (m *mockBackendWithError) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateError != nil {
		return nil, m.generateError
	}
	return m.mockBackendForRouter.Generate(ctx, req)
}

// TestForwardingRouter_BackendNotFound is part of the public API.
func TestForwardingRouter_BackendNotFound(t *testing.T) {
	// Create a single backend
	backend1 := &mockBackendForRouter{
		id:       strBackendExists,
		hardware: strNpu,
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: strBackendExists,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}

	// Create forwarding config with non-existent backends in escalation path
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           10, // High retry limit so backend not found is tested
		EscalationPath:       []string{"backend-does-not-exist", "backend-also-missing", strBackendExists},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should skip non-existent backends and use backend-exists
	if result.FinalBackend == nil {
		t.Fatal("FinalBackend is nil")
	}

	if result.FinalBackend.ID() != strBackendExists {
		t.Errorf("Expected backend-exists, got %s", result.FinalBackend.ID())
	}

	// Check reasoning includes backend not found messages
	foundNotFoundReasoning := false
	for _, reason := range result.Reasoning {
		if findInString(reason, "not found") || findInString(reason, "skipping") {
			foundNotFoundReasoning = true
			t.Logf("Backend not found reasoning: %s", reason)
		}
	}

	if !foundNotFoundReasoning {
		t.Error("Expected reasoning about backends not found")
	}
}

// TestForwardingRouter_GenerateFailure is part of the public API.
func TestForwardingRouter_GenerateFailure(t *testing.T) {
	// Create backend that fails on Generate
	failingBackend := &mockBackendWithError{
		mockBackendForRouter: mockBackendForRouter{
			id:       strFailingBackend,
			hardware: strNpu,
			healthy:  true,
		},
		generateError: fmt.Errorf("backend temporarily unavailable"),
	}

	// Create working fallback backend
	workingBackend := &mockBackendForRouter{
		id:       strWorkingBackend,
		hardware: strIgpu,
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: strFailingBackend,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(failingBackend); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(workingBackend); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{strFailingBackend, strWorkingBackend},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should fall back to working backend
	if result.FinalBackend == nil {
		t.Fatal("FinalBackend is nil")
	}

	if result.FinalBackend.ID() != strWorkingBackend {
		t.Errorf("Expected working-backend after failure, got %s", result.FinalBackend.ID())
	}

	// Check that failure was recorded in reasoning
	foundFailureReasoning := false
	for _, reason := range result.Reasoning {
		if findInString(reason, "failed") {
			foundFailureReasoning = true
			t.Logf("Failure reasoning: %s", reason)
		}
	}

	if !foundFailureReasoning {
		t.Error("Expected reasoning about backend failure")
	}

	// Verify first attempt failed
	if len(result.Attempts) < 2 {
		t.Fatalf("Expected at least 2 attempts, got %d", len(result.Attempts))
	}

	if result.Attempts[0].Success {
		t.Error("First attempt should have failed")
	}

	if result.Attempts[1].Success == false {
		t.Error("Second attempt should have succeeded")
	}
}

// TestForwardingRouter_LowConfidenceForwarding is part of the public API.
func TestForwardingRouter_LowConfidenceForwarding(t *testing.T) {
	// Create backends
	backend1 := &mockBackendForRouter{
		id:       strBackend1,
		hardware: strNpu,
		healthy:  true,
	}

	backend2 := &mockBackendForRouter{
		id:       strBackend22,
		hardware: strIgpu,
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(backend2); err != nil {
		t.Fatal(err)
	}

	// Set very high confidence threshold so all responses have low confidence
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.99, // Very high threshold - responses won't meet this
		MaxRetries:           3,
		EscalationPath:       []string{strBackend1, strBackend22},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true, // Will return best attempt even if below threshold
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should have tried multiple backends due to low confidence
	if result.TotalAttempts < 2 {
		t.Errorf("Expected at least 2 attempts due to low confidence, got %d", result.TotalAttempts)
	}

	// Check reasoning includes low confidence messages
	foundLowConfidenceReasoning := false
	for _, reason := range result.Reasoning {
		if findInString(reason, "Confidence too low") || findInString(reason, "forwarding to next") {
			foundLowConfidenceReasoning = true
			t.Logf("Low confidence reasoning: %s", reason)
		}
	}

	if !foundLowConfidenceReasoning {
		t.Error("Expected reasoning about low confidence forwarding")
	}

	// Should still get a response via best attempt fallback
	if result.FinalResponse == "" {
		t.Error("Expected FinalResponse via best attempt fallback")
	}

	t.Logf("Tried %d backends due to confidence < %.2f, used best attempt",
		result.TotalAttempts, forwardingCfg.MinConfidence)
}

// TestForwardingRouter_BestAttemptFallback is part of the public API.
func TestForwardingRouter_BestAttemptFallback(t *testing.T) {
	// Create backends
	backend1 := &mockBackendForRouter{
		id:       strBackend1,
		hardware: strNpu,
		healthy:  true,
	}

	backend2 := &mockBackendForRouter{
		id:       strBackend22,
		hardware: strIgpu,
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(backend2); err != nil {
		t.Fatal(err)
	}

	// High confidence threshold with ReturnBestAttempt enabled
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.95, // High threshold
		MaxRetries:           2,
		EscalationPath:       []string{strBackend1, strBackend22},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true, // KEY: Enable best attempt fallback
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should have tried both backends
	if result.TotalAttempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", result.TotalAttempts)
	}

	// Check reasoning includes best attempt message
	foundBestAttemptReasoning := false
	for _, reason := range result.Reasoning {
		if findInString(reason, "No attempt met threshold") || findInString(reason, "best attempt") {
			foundBestAttemptReasoning = true
			t.Logf("Best attempt reasoning: %s", reason)
		}
	}

	if !foundBestAttemptReasoning {
		t.Error("Expected reasoning about returning best attempt")
	}

	// Should have returned best attempt
	if result.FinalResponse == "" {
		t.Error("Expected FinalResponse from best attempt")
	}

	if result.FinalBackend == nil {
		t.Error("Expected FinalBackend from best attempt")
	}

	t.Logf("Successfully returned best attempt with confidence below %.2f threshold",
		forwardingCfg.MinConfidence)
}

// TestForwardingRouter_AllBackendsFailed is part of the public API.
func TestForwardingRouter_AllBackendsFailed(t *testing.T) {
	// Create backends that all fail
	backend1 := &mockBackendWithError{
		mockBackendForRouter: mockBackendForRouter{
			id:       strBackend1,
			hardware: strNpu,
			healthy:  true,
		},
		generateError: fmt.Errorf("backend-1 failed"),
	}

	backend2 := &mockBackendWithError{
		mockBackendForRouter: mockBackendForRouter{
			id:       strBackend22,
			hardware: strIgpu,
			healthy:  true,
		},
		generateError: fmt.Errorf("backend-2 failed"),
	}

	routerCfg := &Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	if err := baseRouter.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}
	if err := baseRouter.RegisterBackend(backend2); err != nil {
		t.Fatal(err)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.5,
		MaxRetries:           3,
		EscalationPath:       []string{strBackend1, strBackend22},
		RespectThermalLimits: false,
		ReturnBestAttempt:    false, // KEY: Disable best attempt fallback
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	// Should return error when all backends fail
	if err == nil {
		t.Error("Expected error when all backends failed, got nil")
	}

	if err != nil && !findInString(err.Error(), "all backends failed") {
		t.Errorf("Expected 'all backends failed' error, got: %v", err)
	}

	// Result should still be returned with attempts
	if result == nil {
		t.Fatal("Result should not be nil even on error")
	}

	// All attempts should have failed
	for i, attempt := range result.Attempts {
		if attempt.Success {
			t.Errorf("Attempt %d should have failed", i)
		}
	}

	// Should have no final response
	if result.FinalResponse != "" {
		t.Error("Expected empty FinalResponse when all backends failed")
	}

	t.Logf("Successfully returned error when all %d backends failed", len(result.Attempts))
}

// TestForwardingRouter_MaxRetriesEnforced is part of the public API.
func TestForwardingRouter_MaxRetriesEnforced(t *testing.T) {
	// Create 5 backends
	mockBackends := []*mockBackendForRouter{
		{id: strBackend1, hardware: strNpu, healthy: true},
		{id: strBackend22, hardware: strIgpu, healthy: true},
		{id: strBackend3, hardware: "dgpu", healthy: true},
		{id: "backend-4", hardware: strNvidia, healthy: true},
		{id: "backend-5", hardware: strCPU, healthy: true},
	}

	routerCfg := &Config{
		DefaultBackendID: strBackend1,
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	for _, b := range mockBackends {
		if err := baseRouter.RegisterBackend(b); err != nil {
			t.Fatal(err)
		}
	}

	// Set MaxRetries to 2, but have 5 backends in escalation path
	forwardingCfg := &ForwardingConfig{
		Enabled:       true,
		MinConfidence: 0.99, // High threshold to force trying multiple backends
		MaxRetries:    2,    // KEY: Limit to 2 retries
		EscalationPath: []string{
			strBackend1, strBackend22, strBackend3, "backend-4", "backend-5",
		},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	result, err := forwardingRouter.GenerateWithForwarding(
		ctx,
		"Test prompt",
		"test-model",
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	// Should have stopped at MaxRetries
	if result.TotalAttempts > forwardingCfg.MaxRetries {
		t.Errorf("TotalAttempts = %d, should not exceed MaxRetries = %d",
			result.TotalAttempts, forwardingCfg.MaxRetries)
	}

	// Check reasoning includes max retries message
	foundMaxRetriesReasoning := false
	for _, reason := range result.Reasoning {
		if findInString(reason, "Max retries") && findInString(reason, "reached") {
			foundMaxRetriesReasoning = true
			t.Logf("Max retries reasoning: %s", reason)
		}
	}

	if !foundMaxRetriesReasoning {
		t.Error("Expected reasoning about max retries being reached")
	}

	t.Logf("Successfully enforced MaxRetries=%d (total attempts: %d)",
		forwardingCfg.MaxRetries, result.TotalAttempts)
}
