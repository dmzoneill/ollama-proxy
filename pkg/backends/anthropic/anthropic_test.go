package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

func TestNewAnthropicBackend(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config with API key",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "anthropic-test",
					Name: "Test Anthropic",
				},
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "anthropic-test",
					Name: "Test Anthropic",
				},
			},
			wantErr: true,
		},
		{
			name: "custom endpoint",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "anthropic-test",
					Name: "Test Anthropic",
				},
				APIKey:   "test-api-key",
				Endpoint: "https://custom.api.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewAnthropicBackend(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if backend == nil {
					t.Error("Expected backend to be created, got nil")
				}
				if backend.ID() != tt.cfg.ID {
					t.Errorf("ID() = %v, want %v", backend.ID(), tt.cfg.ID)
				}
				if backend.Type() != "anthropic" {
					t.Errorf("Type() = %v, want anthropic", backend.Type())
				}
				if tt.cfg.Endpoint != "" && backend.endpoint != tt.cfg.Endpoint {
					t.Errorf("endpoint = %v, want %v", backend.endpoint, tt.cfg.Endpoint)
				}
			}
		})
	}
}

func TestNewAnthropicBackend_APIKeyEnv(t *testing.T) {
	// Set environment variable
	envKey := "TEST_ANTHROPIC_API_KEY"
	expectedKey := "env-test-key-12345"
	t.Setenv(envKey, expectedKey)

	cfg := Config{
		BackendConfig: backends.BackendConfig{
			ID:   "anthropic-env-test",
			Name: "Test Anthropic Env",
		},
		APIKeyEnv: envKey,
	}

	backend, err := NewAnthropicBackend(cfg)
	if err != nil {
		t.Fatalf("NewAnthropicBackend() with APIKeyEnv failed: %v", err)
	}

	if backend == nil {
		t.Fatal("Expected backend to be created, got nil")
	}

	if backend.apiKey != expectedKey {
		t.Errorf("apiKey = %v, want %v", backend.apiKey, expectedKey)
	}
}

func TestAnthropicBackend_Getters(t *testing.T) {
	backend, err := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:           "test-anthropic",
			Name:         "Test Claude",
			PowerWatts:   0,
			AvgLatencyMs: 500,
			Priority:     5,
		},
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	if backend.ID() != "test-anthropic" {
		t.Errorf("ID() = %v, want test-anthropic", backend.ID())
	}
	if backend.Type() != "anthropic" {
		t.Errorf("Type() = %v, want anthropic", backend.Type())
	}
	if backend.Name() != "Test Claude" {
		t.Errorf("Name() = %v, want Test Claude", backend.Name())
	}
	if backend.Hardware() != "cloud" {
		t.Errorf("Hardware() = %v, want cloud", backend.Hardware())
	}
	if backend.PowerWatts() != 0 {
		t.Errorf("PowerWatts() = %v, want 0", backend.PowerWatts())
	}
	if backend.Priority() != 5 {
		t.Errorf("Priority() = %v, want 5", backend.Priority())
	}
}

func TestAnthropicBackend_Capabilities(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	if !backend.SupportsGenerate() {
		t.Error("SupportsGenerate() should return true")
	}
	if !backend.SupportsStream() {
		t.Error("SupportsStream() should return true")
	}
	if backend.SupportsEmbed() {
		t.Error("SupportsEmbed() should return false")
	}
}

func TestAnthropicBackend_HealthCheck(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	ctx := context.Background()
	err := backend.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
	if !backend.IsHealthy() {
		t.Error("IsHealthy() should return true after successful health check")
	}
}

func TestAnthropicBackend_HealthCheckNoAPIKey(t *testing.T) {
	// Create backend with empty API key
	backend := &AnthropicBackend{
		id:     "test",
		apiKey: "", // Empty API key
	}

	ctx := context.Background()
	err := backend.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck() should return error when API key is not configured")
	}
	if backend.IsHealthy() {
		t.Error("IsHealthy() should return false when API key is missing")
	}
	if err.Error() != "API key not configured" {
		t.Errorf("HealthCheck() error message = %v, want 'API key not configured'", err)
	}
}

func TestAnthropicBackend_ListModels(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	ctx := context.Background()
	models, err := backend.ListModels(ctx)
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
	}
	if len(models) == 0 {
		t.Error("ListModels() should return at least one model")
	}

	// Check for expected Claude models
	foundSonnet := false
	for _, model := range models {
		if model == "claude-3-5-sonnet-20241022" {
			foundSonnet = true
			break
		}
	}
	if !foundSonnet {
		t.Error("ListModels() should include claude-3-5-sonnet-20241022")
	}
}

func TestAnthropicBackend_SupportsModel(t *testing.T) {
	tests := []struct {
		name      string
		modelCap  *backends.ModelCapability
		modelName string
		want      bool
	}{
		{
			name:      "default supports claude models",
			modelCap:  nil,
			modelName: "claude-3-opus-20240229",
			want:      true,
		},
		{
			name:      "default doesn't support non-claude",
			modelCap:  nil,
			modelName: "gpt-4",
			want:      false,
		},
		{
			name: "with supported patterns",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"claude-3-*"},
			},
			modelName: "claude-3-opus-20240229",
			want:      true,
		},
		{
			name: "excluded pattern",
			modelCap: &backends.ModelCapability{
				ExcludedPatterns: []string{"*opus*"},
			},
			modelName: "claude-3-opus-20240229",
			want:      false,
		},
		{
			name: "empty supported patterns returns true",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{},
				ExcludedPatterns:       []string{},
			},
			modelName: "any-model-name",
			want:      true,
		},
		{
			name: "no pattern match returns false",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"claude-3-*", "claude-2-*"},
			},
			modelName: "claude-instant-1",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := NewAnthropicBackend(Config{
				BackendConfig: backends.BackendConfig{
					ID:              "test",
					Name:            "Test",
					ModelCapability: tt.modelCap,
				},
				APIKey: "test-key",
			})

			if got := backend.SupportsModel(tt.modelName); got != tt.want {
				t.Errorf("SupportsModel(%v) = %v, want %v", tt.modelName, got, tt.want)
			}
		})
	}
}

func TestAnthropicBackend_Generate(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header to be test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header to be 2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}

		// Send mock response
		response := map[string]interface{}{
			"content": []map[string]string{
				{"text": "Hello! This is a test response."},
			},
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:  "claude-3-5-sonnet-20241022",
		Prompt: "Hello",
		Options: &backends.GenerationOptions{
			Temperature: 0.7,
			TopP:        0.9,
			MaxTokens:   1000,
		},
	}

	resp, err := backend.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.Response != "Hello! This is a test response." {
		t.Errorf("Generate() response = %v, want 'Hello! This is a test response.'", resp.Response)
	}
	if resp.Stats.TokensGenerated != 20 {
		t.Errorf("TokensGenerated = %v, want 20", resp.Stats.TokensGenerated)
	}
	if resp.Stats.EnergyWh != 0 {
		t.Errorf("EnergyWh = %v, want 0", resp.Stats.EnergyWh)
	}
}

func TestAnthropicBackend_GenerateError(t *testing.T) {
	// Create mock server that returns error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
	}))
	defer mockServer.Close()

	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:  "claude-3-5-sonnet-20241022",
		Prompt: "Hello",
	}

	_, err := backend.Generate(ctx, req)
	if err == nil {
		t.Error("Generate() should return error for bad request")
	}
}

func TestAnthropicBackend_UpdateMetrics(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	// Update with successful request
	backend.UpdateMetrics(100, true)
	metrics := backend.GetMetrics()

	if metrics.RequestCount != 1 {
		t.Errorf("RequestCount = %v, want 1", metrics.RequestCount)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %v, want 1", metrics.SuccessCount)
	}
	if metrics.AvgLatencyMs != 100 {
		t.Errorf("AvgLatencyMs = %v, want 100", metrics.AvgLatencyMs)
	}

	// Update with failed request
	backend.UpdateMetrics(200, false)
	metrics = backend.GetMetrics()

	if metrics.RequestCount != 2 {
		t.Errorf("RequestCount = %v, want 2", metrics.RequestCount)
	}
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %v, want 1", metrics.ErrorCount)
	}
	if metrics.ErrorRate != 0.5 {
		t.Errorf("ErrorRate = %v, want 0.5", metrics.ErrorRate)
	}
}

func TestAnthropicBackend_Embed(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	ctx := context.Background()
	_, err := backend.Embed(ctx, &backends.EmbedRequest{})
	if err == nil {
		t.Error("Embed() should return error as it's not supported")
	}
}

func TestAnthropicBackend_StartStop(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	ctx := context.Background()

	if err := backend.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if err := backend.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		modelName string
		pattern   string
		want      bool
	}{
		{"claude-3-opus", "*", true},
		{"claude-3-opus", "claude-3-opus", true},
		{"claude-3-opus", "claude-*", true},
		{"claude-3-opus", "*-opus", true},
		{"claude-3-opus", "*-3-*", true},
		{"claude-3-opus", "gpt-*", false},
		{"claude-3-opus", "*gpt*", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelName+"_"+tt.pattern, func(t *testing.T) {
			if got := matchesPattern(tt.modelName, tt.pattern); got != tt.want {
				t.Errorf("matchesPattern(%v, %v) = %v, want %v", tt.modelName, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestAnthropicBackend_GetMaxModelSizeGB(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	if size := backend.GetMaxModelSizeGB(); size != 999 {
		t.Errorf("GetMaxModelSizeGB() = %v, want 999", size)
	}
}

func TestAnthropicBackend_GetSupportedModelPatterns(t *testing.T) {
	tests := []struct {
		name     string
		modelCap *backends.ModelCapability
		want     []string
	}{
		{
			name:     "default patterns",
			modelCap: nil,
			want:     []string{"claude-*"},
		},
		{
			name: "custom patterns",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"claude-3-*", "claude-2-*"},
			},
			want: []string{"claude-3-*", "claude-2-*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := NewAnthropicBackend(Config{
				BackendConfig: backends.BackendConfig{
					ID:              "test",
					Name:            "Test",
					ModelCapability: tt.modelCap,
				},
				APIKey: "test-key",
			})

			patterns := backend.GetSupportedModelPatterns()
			if len(patterns) != len(tt.want) {
				t.Errorf("GetSupportedModelPatterns() = %v, want %v", patterns, tt.want)
			}
		})
	}
}

func TestAnthropicBackend_GetPreferredModels(t *testing.T) {
	tests := []struct {
		name     string
		modelCap *backends.ModelCapability
		wantLen  int
	}{
		{
			name:     "default preferred models",
			modelCap: nil,
			wantLen:  2,
		},
		{
			name: "custom preferred models",
			modelCap: &backends.ModelCapability{
				PreferredModels: []string{"claude-3-5-sonnet-20241022"},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := NewAnthropicBackend(Config{
				BackendConfig: backends.BackendConfig{
					ID:              "test",
					Name:            "Test",
					ModelCapability: tt.modelCap,
				},
				APIKey: "test-key",
			})

			models := backend.GetPreferredModels()
			if len(models) != tt.wantLen {
				t.Errorf("GetPreferredModels() length = %v, want %v", len(models), tt.wantLen)
			}
		})
	}
}

func TestAnthropicBackend_AvgLatencyMs(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:           "test",
			Name:         "Test",
			AvgLatencyMs: 500,
		},
		APIKey: "test-key",
	})

	// Before any requests, should return configured average
	if lat := backend.AvgLatencyMs(); lat != 500 {
		t.Errorf("AvgLatencyMs() = %v, want 500", lat)
	}

	// After request, should return measured average
	backend.UpdateMetrics(200, true)
	if lat := backend.AvgLatencyMs(); lat != 200 {
		t.Errorf("AvgLatencyMs() after update = %v, want 200", lat)
	}
}

func TestAnthropicBackend_ConcurrentMetricsUpdate(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	// Simulate concurrent metric updates
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				backend.UpdateMetrics(100, true)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := backend.GetMetrics()
	if metrics.RequestCount != 1000 {
		t.Errorf("RequestCount = %v, want 1000", metrics.RequestCount)
	}
}

func TestAnthropicBackend_ContextCancellation(t *testing.T) {
	// Create a slow mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{{"text": "response"}},
		})
	}))
	defer mockServer.Close()

	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &backends.GenerateRequest{
		Model:  "claude-3-5-sonnet-20241022",
		Prompt: "Hello",
	}

	_, err := backend.Generate(ctx, req)
	if err == nil {
		t.Error("Generate() should return error when context is cancelled")
	}
}

func TestAnthropicBackend_GenerateStream_NotImplemented(t *testing.T) {
	backend, _ := NewAnthropicBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:  "claude-3-5-sonnet-20241022",
		Prompt: "Hello",
	}

	_, err := backend.GenerateStream(ctx, req)
	if err == nil {
		t.Error("GenerateStream should return error (not implemented)")
	}

	if err.Error() != "streaming not yet implemented for Anthropic backend" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestAnthropicBackend_matchesPattern_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		pattern   string
		expected  bool
	}{
		{
			name:      "Universal wildcard",
			modelName: "claude-3-opus",
			pattern:   "*",
			expected:  true,
		},
		{
			name:      "Exact match",
			modelName: "claude-3-opus",
			pattern:   "claude-3-opus",
			expected:  true,
		},
		{
			name:      "Prefix wildcard match",
			modelName: "claude-3-opus-20240229",
			pattern:   "claude-3-opus*",
			expected:  true,
		},
		{
			name:      "Prefix wildcard no match",
			modelName: "claude-2.1",
			pattern:   "claude-3*",
			expected:  false,
		},
		{
			name:      "Suffix wildcard match",
			modelName: "claude-instant-1.2",
			pattern:   "*instant-1.2",
			expected:  true,
		},
		{
			name:      "Suffix wildcard no match",
			modelName: "claude-instant-1.1",
			pattern:   "*instant-1.2",
			expected:  false,
		},
		{
			name:      "Substring wildcard match",
			modelName: "claude-3-sonnet-20240229",
			pattern:   "*sonnet*",
			expected:  true,
		},
		{
			name:      "Substring wildcard no match",
			modelName: "claude-3-opus-20240229",
			pattern:   "*sonnet*",
			expected:  false,
		},
		{
			name:      "No pattern match",
			modelName: "gpt-4",
			pattern:   "claude",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.modelName, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v",
					tt.modelName, tt.pattern, result, tt.expected)
			}
		})
	}
}
