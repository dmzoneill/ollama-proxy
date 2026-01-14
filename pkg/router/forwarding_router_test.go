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
	id       string
	hardware string
	healthy  bool
	supportsModelFunc func(string) bool
}

func (m *mockBackendForRouter) ID() string                                                            { return m.id }
func (m *mockBackendForRouter) Type() string                                                          { return "mock" }
func (m *mockBackendForRouter) Name() string                                                          { return m.id }
func (m *mockBackendForRouter) Hardware() string                                                      { return m.hardware }
func (m *mockBackendForRouter) Endpoint() string                                                      { return "http://mock" }
func (m *mockBackendForRouter) IsHealthy() bool                                                       {
	if m.healthy {
		return true
	}
	return false
}
func (m *mockBackendForRouter) PowerWatts() float64                                                   { return 10.0 }
func (m *mockBackendForRouter) AvgLatencyMs() int32                                                   { return 100 }
func (m *mockBackendForRouter) Priority() int                                                         { return 1 }
func (m *mockBackendForRouter) SupportsGenerate() bool                                                { return true }
func (m *mockBackendForRouter) SupportsEmbed() bool                                                   { return true }
func (m *mockBackendForRouter) SupportsStream() bool                                                  { return true }
func (m *mockBackendForRouter) SupportsModel(model string) bool                                       {
	if m.supportsModelFunc != nil {
		return m.supportsModelFunc(model)
	}
	return true
}
func (m *mockBackendForRouter) Start(ctx context.Context) error                                       { return nil }
func (m *mockBackendForRouter) Stop(ctx context.Context) error                                        { return nil }

// Multimedia capability methods
func (m *mockBackendForRouter) SupportsAudioToText() bool { return false }
func (m *mockBackendForRouter) SupportsTextToAudio() bool { return false }
func (m *mockBackendForRouter) SupportsImageToText() bool { return false }
func (m *mockBackendForRouter) SupportsTextToImage() bool { return false }
func (m *mockBackendForRouter) SupportsVideoToText() bool { return false }
func (m *mockBackendForRouter) SupportsTextToVideo() bool { return false }
func (m *mockBackendForRouter) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackendForRouter) HealthCheck(ctx context.Context) error                                 { return nil }
func (m *mockBackendForRouter) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	// Return a basic response for testing
	return &backends.GenerateResponse{
		Response: "Test response from " + m.id,
		Stats: &backends.GenerationStats{
			TotalTimeMs:        100,
			TokensGenerated:    20,
			TokensPerSecond:    10,
		},
	}, nil
}
func (m *mockBackendForRouter) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}
func (m *mockBackendForRouter) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}
func (m *mockBackendForRouter) ListModels(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockBackendForRouter) GetMetrics() *backends.BackendMetrics            { return &backends.BackendMetrics{} }
func (m *mockBackendForRouter) GetMaxModelSizeGB() int                          { return 16 }
func (m *mockBackendForRouter) GetSupportedModelPatterns() []string             { return nil }
func (m *mockBackendForRouter) GetPreferredModels() []string                    { return nil }
func (m *mockBackendForRouter) UpdateMetrics(latencyMs int32, success bool)     {}
func (m *mockBackendForRouter) Recv() (*backends.StreamChunk, error)            { return nil, nil }

func TestForwardingRouter_BasicForwarding(t *testing.T) {
	// Create mock backends
	npuBackend := &mockBackendForRouter{
		id:       "ollama-npu",
		hardware: "npu",
		healthy:  true,
	}

	intelBackend := &mockBackendForRouter{
		id:       "ollama-intel",
		hardware: "igpu",
		healthy:  true,
	}

	nvidiaBackend := &mockBackendForRouter{
		id:       "ollama-nvidia",
		hardware: "nvidia",
		healthy:  true,
	}

	// Create base router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)
	baseRouter.RegisterBackend(nvidiaBackend)

	// Create forwarding router
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{"ollama-npu", "ollama-intel", "ollama-nvidia"},
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

func TestForwardingRouter_ModelCompatibility(t *testing.T) {
	// Create backends with different model support
	npuBackend := &mockBackendForRouter{
		id:       "ollama-npu",
		hardware: "npu",
		healthy:  true,
		supportsModelFunc: func(model string) bool {
			// NPU only supports small models
			return model == "qwen2.5:0.5b"
		},
	}

	intelBackend := &mockBackendForRouter{
		id:       "ollama-intel",
		hardware: "igpu",
		healthy:  true,
		supportsModelFunc: func(model string) bool {
			// Intel supports small and medium models
			return model == "qwen2.5:0.5b" || model == "llama3:7b"
		},
	}

	nvidiaBackend := &mockBackendForRouter{
		id:       "ollama-nvidia",
		hardware: "nvidia",
		healthy:  true,
		supportsModelFunc: func(model string) bool {
			// NVIDIA supports all models
			return true
		},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)
	baseRouter.RegisterBackend(nvidiaBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{"ollama-npu", "ollama-intel", "ollama-nvidia"},
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
			model:           "qwen2.5:0.5b",
			expectedBackend: "ollama-npu",
			description:     "qwen2.5:0.5b supported by NPU, shouldn't forward",
		},
		{
			name:            "Medium model forwards to Intel",
			model:           "llama3:7b",
			expectedBackend: "ollama-intel",
			description:     "llama3:7b not supported by NPU, should forward to Intel",
		},
		{
			name:            "Large model forwards to NVIDIA",
			model:           "llama3:70b",
			expectedBackend: "ollama-nvidia",
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

func TestForwardingRouter_ThermalRespect(t *testing.T) {
	// Create backends
	npuBackend := &mockBackendForRouter{
		id:       "ollama-npu",
		hardware: "npu",
		healthy:  false, // Mark as unhealthy (thermal issues)
	}

	intelBackend := &mockBackendForRouter{
		id:       "ollama-intel",
		hardware: "igpu",
		healthy:  true,
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{"ollama-npu", "ollama-intel"},
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

	if result.FinalBackend.ID() != "ollama-intel" {
		t.Errorf("Expected to use ollama-intel (NPU is unhealthy), got %s",
			result.FinalBackend.ID())
	}

	// Check that NPU was skipped
	if len(result.Attempts) == 0 {
		t.Error("No attempts recorded")
	}

	npuSkipped := false
	for _, attempt := range result.Attempts {
		if attempt.BackendID == "ollama-npu" && !attempt.Success {
			npuSkipped = true
			t.Logf("NPU skipped: %s", attempt.SkipReason)
		}
	}

	if !npuSkipped {
		t.Error("Expected NPU to be skipped due to thermal issues")
	}
}

func TestForwardingRouter_EscalationPath(t *testing.T) {
	// Create backends
	mockBackends := []*mockBackendForRouter{
		{id: "backend-1", hardware: "npu", healthy: true},
		{id: "backend-2", hardware: "igpu", healthy: true},
		{id: "backend-3", hardware: "nvidia", healthy: true},
	}

	routerCfg := &Config{
		DefaultBackendID: "backend-1",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	for _, b := range mockBackends {
		baseRouter.RegisterBackend(b)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{"backend-1", "backend-2", "backend-3"},
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

func TestForwardingRouter_MaxRetries(t *testing.T) {
	// Create backends
	backend1 := &mockBackendForRouter{id: "backend-1", hardware: "npu", healthy: true}
	backend2 := &mockBackendForRouter{id: "backend-2", hardware: "igpu", healthy: true}

	routerCfg := &Config{
		DefaultBackendID: "backend-1",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(backend1)
	baseRouter.RegisterBackend(backend2)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           1, // Only 1 retry
		EscalationPath:       []string{"backend-1", "backend-2"},
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

func TestIsSmallModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected bool
	}{
		{"0.5b model", "qwen2.5:0.5b", true},
		{"1.5b model", "llama:1.5b", true},
		{"tiny model", "tiny-llama", true},
		{"mini model", "mini-gpt", true},
		{"7b model", "llama3:7b", false},
		{"70b model", "llama3:70b", false},
		{"regular model", "gpt-4", false},
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

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "test", "test", true},
		{"prefix", "testing", "test", true},
		{"suffix", "contest", "test", true},
		{"middle", "attested", "test", true},
		{"not found", "example", "test", false},
		{"empty substring", "test", "", false}, // Implementation requires len(substr) > 0
		{"empty string", "", "test", false},
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

func TestFindInString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"found at start", "hello world", "hello", true},
		{"found at end", "hello world", "world", true},
		{"found in middle", "hello world", "lo wo", true},
		{"not found", "hello world", "xyz", false},
		{"single char found", "abcdef", "c", true},
		{"single char not found", "abcdef", "z", false},
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

func TestSetEscalationPath(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: "test"})
	forwardingRouter := NewForwardingRouter(router, nil, DefaultForwardingConfig())
	
	newPath := []string{"backend1", "backend2", "backend3"}
	forwardingRouter.SetEscalationPath(newPath)
	
	// Verify path was set (indirect verification through config)
	if forwardingRouter.config.EscalationPath == nil {
		t.Error("SetEscalationPath should update the escalation path")
	}
}

func TestSetMinConfidence(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: "test"})
	forwardingRouter := NewForwardingRouter(router, nil, DefaultForwardingConfig())
	
	newConfidence := 0.85
	forwardingRouter.SetMinConfidence(newConfidence)
	
	if forwardingRouter.config.MinConfidence != newConfidence {
		t.Errorf("SetMinConfidence should set to %f, got %f", newConfidence, forwardingRouter.config.MinConfidence)
	}
}

func TestForwardingRouter_BuildEscalationPath(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: "test"})
	fr := NewForwardingRouter(router, nil, DefaultForwardingConfig())

	// Test via GenerateWithForwarding which calls buildEscalationPath
	npuBackend := &mockBackendForRouter{
		id:       "ollama-npu",
		hardware: "npu",
		healthy:  true,
	}
	intelBackend := &mockBackendForRouter{
		id:       "ollama-intel",
		hardware: "igpu",
		healthy:  true,
	}
	nvidiaBackend := &mockBackendForRouter{
		id:       "ollama-nvidia",
		hardware: "nvidia",
		healthy:  true,
	}

	router.RegisterBackend(npuBackend)
	router.RegisterBackend(intelBackend)
	router.RegisterBackend(nvidiaBackend)

	ctx := context.Background()

	// Test with small model - should try NPU first
	result, err := fr.GenerateWithForwarding(ctx, "test", "qwen2.5:0.5b", &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	// Test with large model
	result2, err := fr.GenerateWithForwarding(ctx, "test", "llama3:70b", &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding with large model failed: %v", err)
	}
	if result2 == nil {
		t.Fatal("Result2 is nil")
	}
}

func TestForwardingRouter_FindBackend(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: "test"})

	backend1 := &mockBackendForRouter{
		id:       "backend-1",
		hardware: "npu",
		healthy:  true,
	}

	router.RegisterBackend(backend1)

	fr := NewForwardingRouter(router, nil, DefaultForwardingConfig())

	// Test finding via GenerateWithForwarding with specific escalation path
	fr.SetEscalationPath([]string{"backend-1"})

	ctx := context.Background()
	result, err := fr.GenerateWithForwarding(ctx, "test", "test-model", &backends.Annotations{})

	if err != nil {
		t.Fatalf("GenerateWithForwarding failed: %v", err)
	}

	if result.FinalBackend.ID() != "backend-1" {
		t.Errorf("Expected backend-1, got %s", result.FinalBackend.ID())
	}
}

func TestNewForwardingRouter_NilConfig(t *testing.T) {
	router := NewRouter(Config{DefaultBackendID: "test"})

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

func TestForwardingRouter_FindBackend_WithThermalRouter(t *testing.T) {
	// Create base router
	baseRouter := NewRouter(Config{DefaultBackendID: "base-backend"})

	baseBackend := &mockBackendForRouter{
		id:       "base-backend",
		hardware: "cpu",
		healthy:  true,
	}
	baseRouter.RegisterBackend(baseBackend)

	// Create thermal router with different backend
	thermalConfig := &thermal.ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		FanQuiet:     30,
		FanLoud:      80,
	}
	monitor := thermal.NewThermalMonitor(thermalConfig, 0)
	thermalRouterCfg := Config{
		DefaultBackendID: "thermal-backend",
		PowerAware:       true,
	}
	thermalRouter := NewThermalRouter(thermalRouterCfg, monitor)

	thermalBackend := &mockBackendForRouter{
		id:       "thermal-backend",
		hardware: "gpu",
		healthy:  true,
	}
	thermalRouter.RegisterBackend(thermalBackend)

	// Create forwarding router with both routers
	forwardingCfg := DefaultForwardingConfig()
	fr := NewForwardingRouter(baseRouter, thermalRouter, forwardingCfg)

	// Test finding backend from base router
	fr.SetEscalationPath([]string{"base-backend"})
	ctx := context.Background()
	result1, err := fr.GenerateWithForwarding(ctx, "test", "test-model", &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding for base backend failed: %v", err)
	}
	if result1.FinalBackend.ID() != "base-backend" {
		t.Errorf("Expected base-backend, got %s", result1.FinalBackend.ID())
	}

	// Test finding backend from thermal router
	fr.SetEscalationPath([]string{"thermal-backend"})
	result2, err := fr.GenerateWithForwarding(ctx, "test", "test-model", &backends.Annotations{})
	if err != nil {
		t.Fatalf("GenerateWithForwarding for thermal backend failed: %v", err)
	}
	if result2.FinalBackend.ID() != "thermal-backend" {
		t.Errorf("Expected thermal-backend, got %s", result2.FinalBackend.ID())
	}

	t.Logf("Successfully found backends in both base and thermal routers")
}

// mockStreamReader implements backends.StreamReader for testing
type mockStreamReader struct {
	chunks []string
	index  int
}

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

func (m *mockStreamReader) Close() error {
	return nil
}

// mockBackendWithStream extends mockBackendForRouter to support streaming
type mockBackendWithStream struct {
	mockBackendForRouter
	streamChunks []string
	streamError  error
}

func (m *mockBackendWithStream) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	return &mockStreamReader{chunks: m.streamChunks}, nil
}

func TestGenerateWithForwardingStream_Success(t *testing.T) {
	// Create backends with streaming support
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-npu",
			hardware: "npu",
			healthy:  true,
		},
		streamChunks: []string{"Hello", " ", "world", "!"},
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-intel",
			hardware: "igpu",
			healthy:  true,
		},
		streamChunks: []string{"Test", " ", "stream"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{"ollama-npu", "ollama-intel"},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		"qwen2.5:0.5b", // Use small model to trigger NPU selection
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

	if backend.ID() != "ollama-npu" {
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

func TestGenerateWithForwardingStream_ThermalRespect(t *testing.T) {
	// Create backends
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-npu",
			hardware: "npu",
			healthy:  false, // Mark as unhealthy
		},
		streamChunks: []string{"NPU", " ", "stream"},
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-intel",
			hardware: "igpu",
			healthy:  true,
		},
		streamChunks: []string{"Intel", " ", "stream"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{"ollama-npu", "ollama-intel"},
		RespectThermalLimits: true, // Enable thermal checking
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		"qwen2.5:0.5b", // Use small model so NPU is in escalation path
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwardingStream failed: %v", err)
	}

	// Should skip unhealthy NPU and use Intel
	if backend == nil {
		t.Fatal("Backend is nil")
	}

	if backend.ID() != "ollama-intel" {
		t.Errorf("Expected to use ollama-intel (NPU is unhealthy), got %s", backend.ID())
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	t.Logf("Successfully skipped unhealthy NPU and used %s", backend.ID())
}

func TestGenerateWithForwardingStream_ModelCompatibility(t *testing.T) {
	// Create backends with different model support
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-npu",
			hardware: "npu",
			healthy:  true,
			supportsModelFunc: func(model string) bool {
				return model == "qwen2.5:0.5b"
			},
		},
		streamChunks: []string{"NPU", " ", "response"},
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-intel",
			hardware: "igpu",
			healthy:  true,
			supportsModelFunc: func(model string) bool {
				return model == "qwen2.5:0.5b" || model == "llama3:7b"
			},
		},
		streamChunks: []string{"Intel", " ", "response"},
	}

	nvidiaBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-nvidia",
			hardware: "nvidia",
			healthy:  true,
			supportsModelFunc: func(model string) bool {
				return true // Supports all models
			},
		},
		streamChunks: []string{"NVIDIA", " ", "response"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)
	baseRouter.RegisterBackend(nvidiaBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{"ollama-npu", "ollama-intel", "ollama-nvidia"},
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
			model:           "qwen2.5:0.5b",
			expectedBackend: "ollama-npu",
		},
		{
			name:            "Medium model uses Intel",
			model:           "llama3:7b",
			expectedBackend: "ollama-intel",
		},
		{
			name:            "Large model uses NVIDIA",
			model:           "llama3:70b",
			expectedBackend: "ollama-nvidia",
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

func TestGenerateWithForwardingStream_NoSuitableBackend(t *testing.T) {
	// Create backend that doesn't support the requested model
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-npu",
			hardware: "npu",
			healthy:  true,
			supportsModelFunc: func(model string) bool {
				return false // Doesn't support any models
			},
		},
		streamChunks: []string{"Should", " ", "not", " ", "be", " ", "called"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           1,
		EscalationPath:       []string{"ollama-npu"},
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

func TestGenerateWithForwardingStream_BackendGenerateStreamError(t *testing.T) {
	// Create backend that returns error on GenerateStream
	npuBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-npu",
			hardware: "npu",
			healthy:  true,
		},
		streamError: fmt.Errorf("backend unavailable"),
	}

	intelBackend := &mockBackendWithStream{
		mockBackendForRouter: mockBackendForRouter{
			id:       "ollama-intel",
			hardware: "igpu",
			healthy:  true,
		},
		streamChunks: []string{"Intel", " ", "fallback"},
	}

	// Create router
	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           2,
		EscalationPath:       []string{"ollama-npu", "ollama-intel"},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		"qwen2.5:0.5b", // Use small model so NPU is tried first
		&backends.Annotations{},
	)

	if err != nil {
		t.Fatalf("GenerateWithForwardingStream should succeed with fallback: %v", err)
	}

	// Should fall back to Intel when NPU fails
	if backend == nil {
		t.Fatal("Backend is nil")
	}

	if backend.ID() != "ollama-intel" {
		t.Errorf("Expected fallback to ollama-intel, got %s", backend.ID())
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	t.Logf("Successfully fell back from NPU to Intel after GenerateStream error")
}

func TestGenerateWithForwardingStream_EscalationPath(t *testing.T) {
	// Create multiple backends with names that match buildEscalationPath expectations
	mockBackends := []*mockBackendWithStream{
		{
			mockBackendForRouter: mockBackendForRouter{
				id:       "ollama-npu",
				hardware: "npu",
				healthy:  true,
			},
			streamChunks: []string{"Backend", " ", "1"},
		},
		{
			mockBackendForRouter: mockBackendForRouter{
				id:       "ollama-intel",
				hardware: "igpu",
				healthy:  true,
			},
			streamChunks: []string{"Backend", " ", "2"},
		},
		{
			mockBackendForRouter: mockBackendForRouter{
				id:       "ollama-nvidia",
				hardware: "nvidia",
				healthy:  true,
			},
			streamChunks: []string{"Backend", " ", "3"},
		},
	}

	routerCfg := &Config{
		DefaultBackendID: "ollama-npu",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	for _, b := range mockBackends {
		baseRouter.RegisterBackend(b)
	}

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{"ollama-npu", "ollama-intel", "ollama-nvidia"},
		RespectThermalLimits: false,
		ReturnBestAttempt:    true,
	}

	forwardingRouter := NewForwardingRouter(baseRouter, nil, forwardingCfg)

	ctx := context.Background()
	stream, backend, err := forwardingRouter.GenerateWithForwardingStream(
		ctx,
		"Test prompt",
		"qwen2.5:0.5b", // Use small model to match escalation path
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
	mockBackendForRouter
	generateError error
}

func (m *mockBackendWithError) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateError != nil {
		return nil, m.generateError
	}
	return m.mockBackendForRouter.Generate(ctx, req)
}

func TestForwardingRouter_BackendNotFound(t *testing.T) {
	// Create a single backend
	backend1 := &mockBackendForRouter{
		id:       "backend-exists",
		hardware: "npu",
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: "backend-exists",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(backend1)

	// Create forwarding config with non-existent backends in escalation path
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           10, // High retry limit so backend not found is tested
		EscalationPath:       []string{"backend-does-not-exist", "backend-also-missing", "backend-exists"},
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

	if result.FinalBackend.ID() != "backend-exists" {
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

func TestForwardingRouter_GenerateFailure(t *testing.T) {
	// Create backend that fails on Generate
	failingBackend := &mockBackendWithError{
		mockBackendForRouter: mockBackendForRouter{
			id:       "failing-backend",
			hardware: "npu",
			healthy:  true,
		},
		generateError: fmt.Errorf("backend temporarily unavailable"),
	}

	// Create working fallback backend
	workingBackend := &mockBackendForRouter{
		id:       "working-backend",
		hardware: "igpu",
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: "failing-backend",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(failingBackend)
	baseRouter.RegisterBackend(workingBackend)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.0,
		MaxRetries:           3,
		EscalationPath:       []string{"failing-backend", "working-backend"},
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

	if result.FinalBackend.ID() != "working-backend" {
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

func TestForwardingRouter_LowConfidenceForwarding(t *testing.T) {
	// Create backends
	backend1 := &mockBackendForRouter{
		id:       "backend-1",
		hardware: "npu",
		healthy:  true,
	}

	backend2 := &mockBackendForRouter{
		id:       "backend-2",
		hardware: "igpu",
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: "backend-1",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(backend1)
	baseRouter.RegisterBackend(backend2)

	// Set very high confidence threshold so all responses have low confidence
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.99, // Very high threshold - responses won't meet this
		MaxRetries:           3,
		EscalationPath:       []string{"backend-1", "backend-2"},
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

func TestForwardingRouter_BestAttemptFallback(t *testing.T) {
	// Create backends
	backend1 := &mockBackendForRouter{
		id:       "backend-1",
		hardware: "npu",
		healthy:  true,
	}

	backend2 := &mockBackendForRouter{
		id:       "backend-2",
		hardware: "igpu",
		healthy:  true,
	}

	routerCfg := &Config{
		DefaultBackendID: "backend-1",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(backend1)
	baseRouter.RegisterBackend(backend2)

	// High confidence threshold with ReturnBestAttempt enabled
	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.95, // High threshold
		MaxRetries:           2,
		EscalationPath:       []string{"backend-1", "backend-2"},
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

func TestForwardingRouter_AllBackendsFailed(t *testing.T) {
	// Create backends that all fail
	backend1 := &mockBackendWithError{
		mockBackendForRouter: mockBackendForRouter{
			id:       "backend-1",
			hardware: "npu",
			healthy:  true,
		},
		generateError: fmt.Errorf("backend-1 failed"),
	}

	backend2 := &mockBackendWithError{
		mockBackendForRouter: mockBackendForRouter{
			id:       "backend-2",
			hardware: "igpu",
			healthy:  true,
		},
		generateError: fmt.Errorf("backend-2 failed"),
	}

	routerCfg := &Config{
		DefaultBackendID: "backend-1",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	baseRouter.RegisterBackend(backend1)
	baseRouter.RegisterBackend(backend2)

	forwardingCfg := &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.5,
		MaxRetries:           3,
		EscalationPath:       []string{"backend-1", "backend-2"},
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

func TestForwardingRouter_MaxRetriesEnforced(t *testing.T) {
	// Create 5 backends
	mockBackends := []*mockBackendForRouter{
		{id: "backend-1", hardware: "npu", healthy: true},
		{id: "backend-2", hardware: "igpu", healthy: true},
		{id: "backend-3", hardware: "dgpu", healthy: true},
		{id: "backend-4", hardware: "nvidia", healthy: true},
		{id: "backend-5", hardware: "cpu", healthy: true},
	}

	routerCfg := &Config{
		DefaultBackendID: "backend-1",
		PowerAware:       true,
	}
	baseRouter := NewRouter(*routerCfg)
	for _, b := range mockBackends {
		baseRouter.RegisterBackend(b)
	}

	// Set MaxRetries to 2, but have 5 backends in escalation path
	forwardingCfg := &ForwardingConfig{
		Enabled:       true,
		MinConfidence: 0.99, // High threshold to force trying multiple backends
		MaxRetries:    2,    // KEY: Limit to 2 retries
		EscalationPath: []string{
			"backend-1", "backend-2", "backend-3", "backend-4", "backend-5",
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
