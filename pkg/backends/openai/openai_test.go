package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

func TestNewOpenAIBackend(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		setup   func()
		cleanup func()
		wantErr bool
	}{
		{
			name: "valid config with API key",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "openai-test",
					Name: "Test OpenAI",
				},
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "valid config with API key from env",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "openai-test",
					Name: "Test OpenAI",
				},
				APIKeyEnv: "TEST_OPENAI_KEY",
			},
			setup: func() {
				os.Setenv("TEST_OPENAI_KEY", "env-api-key")
			},
			cleanup: func() {
				os.Unsetenv("TEST_OPENAI_KEY")
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "openai-test",
					Name: "Test OpenAI",
				},
			},
			wantErr: true,
		},
		{
			name: "custom endpoint",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "openai-test",
					Name: "Test OpenAI",
				},
				APIKey:   "test-api-key",
				Endpoint: "https://custom.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name: "with model capability",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:   "openai-test",
					Name: "Test OpenAI",
					ModelCapability: &backends.ModelCapability{
						SupportedModelPatterns: []string{"gpt-4*"},
						PreferredModels:        []string{"gpt-4-turbo"},
					},
				},
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "with power and latency config",
			cfg: Config{
				BackendConfig: backends.BackendConfig{
					ID:           "openai-test",
					Name:         "Test OpenAI",
					PowerWatts:   0,
					AvgLatencyMs: 200,
					Priority:     10,
				},
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			backend, err := NewOpenAIBackend(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if backend == nil {
					t.Error("Expected backend to be created, got nil")
				}
				if backend.ID() != tt.cfg.ID {
					t.Errorf("ID() = %v, want %v", backend.ID(), tt.cfg.ID)
				}
				if backend.Type() != "openai" {
					t.Errorf("Type() = %v, want openai", backend.Type())
				}
				if tt.cfg.Endpoint != "" && backend.endpoint != tt.cfg.Endpoint {
					t.Errorf("endpoint = %v, want %v", backend.endpoint, tt.cfg.Endpoint)
				}
				if tt.cfg.Endpoint == "" && backend.endpoint != "https://api.openai.com/v1" {
					t.Errorf("endpoint = %v, want default https://api.openai.com/v1", backend.endpoint)
				}
				if !backend.healthy.Load() {
					// Initially should be false
				}
			}
		})
	}
}

func TestOpenAIBackend_Getters(t *testing.T) {
	backend, err := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:           "test-openai",
			Name:         "Test GPT",
			PowerWatts:   0,
			AvgLatencyMs: 300,
			Priority:     8,
		},
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	if backend.ID() != "test-openai" {
		t.Errorf("ID() = %v, want test-openai", backend.ID())
	}
	if backend.Type() != "openai" {
		t.Errorf("Type() = %v, want openai", backend.Type())
	}
	if backend.Name() != "Test GPT" {
		t.Errorf("Name() = %v, want Test GPT", backend.Name())
	}
	if backend.Hardware() != "cloud" {
		t.Errorf("Hardware() = %v, want cloud", backend.Hardware())
	}
	if backend.PowerWatts() != 0 {
		t.Errorf("PowerWatts() = %v, want 0", backend.PowerWatts())
	}
	if backend.Priority() != 8 {
		t.Errorf("Priority() = %v, want 8", backend.Priority())
	}
	if backend.GetMaxModelSizeGB() != 999 {
		t.Errorf("GetMaxModelSizeGB() = %v, want 999", backend.GetMaxModelSizeGB())
	}
}

func TestOpenAIBackend_Capabilities(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
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
	if !backend.SupportsEmbed() {
		t.Error("SupportsEmbed() should return true")
	}
}

func TestOpenAIBackend_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		wantHealth bool
	}{
		{
			name:       "successful health check",
			statusCode: http.StatusOK,
			wantErr:    false,
			wantHealth: true,
		},
		{
			name:       "failed health check - bad status",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			wantHealth: false,
		},
		{
			name:       "failed health check - server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
			wantHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/models" {
					t.Errorf("Expected request to /models, got %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("Expected Authorization header to be Bearer test-key, got %s", r.Header.Get("Authorization"))
				}
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"data": []map[string]string{},
					})
				}
			}))
			defer mockServer.Close()

			backend, _ := NewOpenAIBackend(Config{
				BackendConfig: backends.BackendConfig{
					ID:   "test",
					Name: "Test",
				},
				APIKey:   "test-key",
				Endpoint: mockServer.URL,
			})

			ctx := context.Background()
			err := backend.HealthCheck(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
			if backend.IsHealthy() != tt.wantHealth {
				t.Errorf("IsHealthy() = %v, want %v", backend.IsHealthy(), tt.wantHealth)
			}
		})
	}
}

func TestOpenAIBackend_ListModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("Expected request to /models, got %s", r.URL.Path)
		}
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4"},
				{"id": "gpt-4-turbo"},
				{"id": "gpt-3.5-turbo"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	models, err := backend.ListModels(ctx)
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
	}
	if len(models) != 3 {
		t.Errorf("ListModels() returned %d models, want 3", len(models))
	}

	expectedModels := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"}
	for i, model := range models {
		if model != expectedModels[i] {
			t.Errorf("Model[%d] = %v, want %v", i, model, expectedModels[i])
		}
	}
}

func TestOpenAIBackend_SupportsModel(t *testing.T) {
	tests := []struct {
		name      string
		modelCap  *backends.ModelCapability
		modelName string
		want      bool
	}{
		{
			name:      "default supports gpt models",
			modelCap:  nil,
			modelName: "gpt-4",
			want:      true,
		},
		{
			name:      "default supports gpt-3.5",
			modelCap:  nil,
			modelName: "gpt-3.5-turbo",
			want:      true,
		},
		{
			name:      "default supports o1 models",
			modelCap:  nil,
			modelName: "o1-preview",
			want:      true,
		},
		{
			name:      "default doesn't support non-gpt",
			modelCap:  nil,
			modelName: "claude-3-opus",
			want:      false,
		},
		{
			name:      "default doesn't support llama",
			modelCap:  nil,
			modelName: "llama3:8b",
			want:      false,
		},
		{
			name: "with supported patterns",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"gpt-4*"},
			},
			modelName: "gpt-4-turbo",
			want:      true,
		},
		{
			name: "pattern doesn't match",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"gpt-4*"},
			},
			modelName: "gpt-3.5-turbo",
			want:      false,
		},
		{
			name: "excluded pattern",
			modelCap: &backends.ModelCapability{
				ExcludedPatterns: []string{"*turbo*"},
			},
			modelName: "gpt-4-turbo",
			want:      false,
		},
		{
			name: "wildcard pattern",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"*"},
			},
			modelName: "any-model",
			want:      true,
		},
		{
			name: "contains pattern",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"*-4-*"},
			},
			modelName: "gpt-4-turbo",
			want:      true,
		},
		{
			name: "suffix pattern",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"*-turbo"},
			},
			modelName: "gpt-3.5-turbo",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := NewOpenAIBackend(Config{
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

func TestOpenAIBackend_Generate(t *testing.T) {
	tests := []struct {
		name           string
		request        *backends.GenerateRequest
		mockResponse   map[string]interface{}
		wantResponse   string
		wantTokens     int32
		checkLatency   bool
		checkEnergy    bool
		expectedEnergy float32
	}{
		{
			name: "basic generation",
			request: &backends.GenerateRequest{
				Model:  "gpt-4",
				Prompt: "Hello",
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": "Hello! How can I help you today?",
						},
					},
				},
				"usage": map[string]int{
					"total_tokens": 25,
				},
			},
			wantResponse:   "Hello! How can I help you today?",
			wantTokens:     25,
			checkLatency:   true,
			checkEnergy:    true,
			expectedEnergy: 0, // Cloud services don't consume local power
		},
		{
			name: "generation with options",
			request: &backends.GenerateRequest{
				Model:  "gpt-4-turbo",
				Prompt: "Write a haiku",
				Options: &backends.GenerationOptions{
					Temperature: 0.7,
					TopP:        0.9,
					MaxTokens:   100,
				},
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": "Cherry blossoms fall\nSoftly on the morning dew\nSpring awakens now",
						},
					},
				},
				"usage": map[string]int{
					"total_tokens": 20,
				},
			},
			wantResponse: "Cherry blossoms fall\nSoftly on the morning dew\nSpring awakens now",
			wantTokens:   20,
			checkLatency: true,
		},
		{
			name: "empty response",
			request: &backends.GenerateRequest{
				Model:  "gpt-3.5-turbo",
				Prompt: "Test",
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{},
				"usage": map[string]int{
					"total_tokens": 5,
				},
			},
			wantResponse: "",
			wantTokens:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Expected request to /chat/completions, got %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("Expected Authorization header to be Bearer test-key, got %s", r.Header.Get("Authorization"))
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type header to be application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Verify request body
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}
				if reqBody["model"] != tt.request.Model {
					t.Errorf("Request model = %v, want %v", reqBody["model"], tt.request.Model)
				}

				// Check options if provided
				if tt.request.Options != nil {
					if tt.request.Options.Temperature > 0 {
						temp, ok := reqBody["temperature"].(float64)
						if !ok {
							temp = float64(reqBody["temperature"].(float32))
						}
						expectedTemp := float64(tt.request.Options.Temperature)
						if temp < expectedTemp-0.01 || temp > expectedTemp+0.01 {
							t.Errorf("Request temperature = %v, want %v", temp, expectedTemp)
						}
					}
					if tt.request.Options.TopP > 0 {
						topP, ok := reqBody["top_p"].(float64)
						if !ok {
							topP = float64(reqBody["top_p"].(float32))
						}
						expectedTopP := float64(tt.request.Options.TopP)
						if topP < expectedTopP-0.01 || topP > expectedTopP+0.01 {
							t.Errorf("Request top_p = %v, want %v", topP, expectedTopP)
						}
					}
					if tt.request.Options.MaxTokens > 0 {
						if int32(reqBody["max_tokens"].(float64)) != tt.request.Options.MaxTokens {
							t.Errorf("Request max_tokens = %v, want %v", reqBody["max_tokens"], tt.request.Options.MaxTokens)
						}
					}
				}

				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer mockServer.Close()

			backend, _ := NewOpenAIBackend(Config{
				BackendConfig: backends.BackendConfig{
					ID:   "test",
					Name: "Test",
				},
				APIKey:   "test-key",
				Endpoint: mockServer.URL,
			})

			ctx := context.Background()
			resp, err := backend.Generate(ctx, tt.request)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if resp.Response != tt.wantResponse {
				t.Errorf("Generate() response = %v, want %v", resp.Response, tt.wantResponse)
			}
			if resp.Stats.TokensGenerated != tt.wantTokens {
				t.Errorf("TokensGenerated = %v, want %v", resp.Stats.TokensGenerated, tt.wantTokens)
			}
			if tt.checkLatency && resp.Stats.TotalTimeMs < 0 {
				t.Errorf("TotalTimeMs should be >= 0, got %v", resp.Stats.TotalTimeMs)
			}
			if tt.checkEnergy && resp.Stats.EnergyWh != tt.expectedEnergy {
				t.Errorf("EnergyWh = %v, want %v", resp.Stats.EnergyWh, tt.expectedEnergy)
			}
			if resp.Stats.TokensPerSecond <= 0 {
				t.Errorf("TokensPerSecond should be > 0, got %v", resp.Stats.TokensPerSecond)
			}
		})
	}
}

func TestOpenAIBackend_GenerateError(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantErrMsg   string
	}{
		{
			name:         "bad request",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error": {"message": "Invalid request"}}`,
			wantErrMsg:   "OpenAI API error: 400",
		},
		{
			name:         "unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": {"message": "Invalid API key"}}`,
			wantErrMsg:   "OpenAI API error: 401",
		},
		{
			name:         "rate limit",
			statusCode:   http.StatusTooManyRequests,
			responseBody: `{"error": {"message": "Rate limit exceeded"}}`,
			wantErrMsg:   "OpenAI API error: 429",
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": {"message": "Internal server error"}}`,
			wantErrMsg:   "OpenAI API error: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer mockServer.Close()

			backend, _ := NewOpenAIBackend(Config{
				BackendConfig: backends.BackendConfig{
					ID:   "test",
					Name: "Test",
				},
				APIKey:   "test-key",
				Endpoint: mockServer.URL,
			})

			ctx := context.Background()
			req := &backends.GenerateRequest{
				Model:  "gpt-4",
				Prompt: "Hello",
			}

			_, err := backend.Generate(ctx, req)
			if err == nil {
				t.Error("Generate() should return error for bad request")
			}
			if err != nil && len(tt.wantErrMsg) > 0 {
				errStr := err.Error()
				if len(errStr) < len(tt.wantErrMsg) || errStr[:len(tt.wantErrMsg)] != tt.wantErrMsg {
					t.Errorf("Generate() error = %v, want error containing %v", err, tt.wantErrMsg)
				}
			}

			// Verify metrics updated for failed request
			metrics := backend.GetMetrics()
			if metrics.ErrorCount != 1 {
				t.Errorf("ErrorCount = %v, want 1", metrics.ErrorCount)
			}
		})
	}
}

func TestOpenAIBackend_UpdateMetrics(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
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
	if metrics.TotalLatencyMs != 100 {
		t.Errorf("TotalLatencyMs = %v, want 100", metrics.TotalLatencyMs)
	}

	// Update with another successful request
	backend.UpdateMetrics(200, true)
	metrics = backend.GetMetrics()

	if metrics.RequestCount != 2 {
		t.Errorf("RequestCount = %v, want 2", metrics.RequestCount)
	}
	if metrics.SuccessCount != 2 {
		t.Errorf("SuccessCount = %v, want 2", metrics.SuccessCount)
	}
	if metrics.AvgLatencyMs != 150 {
		t.Errorf("AvgLatencyMs = %v, want 150", metrics.AvgLatencyMs)
	}

	// Update with failed request
	backend.UpdateMetrics(300, false)
	metrics = backend.GetMetrics()

	if metrics.RequestCount != 3 {
		t.Errorf("RequestCount = %v, want 3", metrics.RequestCount)
	}
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %v, want 1", metrics.ErrorCount)
	}
	if metrics.ErrorRate < 0.33 || metrics.ErrorRate > 0.34 {
		t.Errorf("ErrorRate = %v, want ~0.333", metrics.ErrorRate)
	}
}

func TestOpenAIBackend_Embed(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected request to /embeddings, got %s", r.URL.Path)
		}

		// Verify request body
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": []float32{0.1, 0.2, 0.3, 0.4, 0.5},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.EmbedRequest{
		Model: "text-embedding-ada-002",
		Text:  "Hello world",
	}

	resp, err := backend.Embed(ctx, req)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(resp.Embedding) != 5 {
		t.Errorf("Embedding length = %v, want 5", len(resp.Embedding))
	}
	expectedEmbedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	for i, val := range resp.Embedding {
		if val != expectedEmbedding[i] {
			t.Errorf("Embedding[%d] = %v, want %v", i, val, expectedEmbedding[i])
		}
	}
	if resp.Stats.TotalTimeMs < 0 {
		t.Errorf("TotalTimeMs should be >= 0, got %v", resp.Stats.TotalTimeMs)
	}
}

func TestOpenAIBackend_EmbedEmpty(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.EmbedRequest{
		Model: "text-embedding-ada-002",
		Text:  "Test",
	}

	resp, err := backend.Embed(ctx, req)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(resp.Embedding) != 0 {
		t.Errorf("Embedding length = %v, want 0", len(resp.Embedding))
	}
}

func TestOpenAIBackend_StartStop(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{},
		})
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()

	if err := backend.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}
	if !backend.IsHealthy() {
		t.Error("Backend should be healthy after Start()")
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
		{"gpt-4", "*", true},
		{"gpt-4", "gpt-4", true},
		{"gpt-4", "gpt-*", true},
		{"gpt-4-turbo", "gpt-4*", true},
		{"gpt-4-turbo", "*-turbo", true},
		{"gpt-4-turbo", "*-4-*", true},
		{"gpt-3.5-turbo", "gpt-*", true},
		{"o1-preview", "o1-*", true},
		{"gpt-4", "claude-*", false},
		{"llama3:8b", "gpt-*", false},
		{"claude-3-opus", "*gpt*", false},
		{"text-embedding-ada-002", "text-*", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelName+"_"+tt.pattern, func(t *testing.T) {
			if got := matchesPattern(tt.modelName, tt.pattern); got != tt.want {
				t.Errorf("matchesPattern(%v, %v) = %v, want %v", tt.modelName, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestOpenAIBackend_GetSupportedModelPatterns(t *testing.T) {
	tests := []struct {
		name     string
		modelCap *backends.ModelCapability
		want     []string
	}{
		{
			name:     "default patterns",
			modelCap: nil,
			want:     []string{"gpt-*", "o1-*"},
		},
		{
			name: "custom patterns",
			modelCap: &backends.ModelCapability{
				SupportedModelPatterns: []string{"gpt-4*", "gpt-3.5*"},
			},
			want: []string{"gpt-4*", "gpt-3.5*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := NewOpenAIBackend(Config{
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
				return
			}
			for i, pattern := range patterns {
				if pattern != tt.want[i] {
					t.Errorf("Pattern[%d] = %v, want %v", i, pattern, tt.want[i])
				}
			}
		})
	}
}

func TestOpenAIBackend_GetPreferredModels(t *testing.T) {
	tests := []struct {
		name     string
		modelCap *backends.ModelCapability
		wantLen  int
	}{
		{
			name:     "default preferred models",
			modelCap: nil,
			wantLen:  3,
		},
		{
			name: "custom preferred models",
			modelCap: &backends.ModelCapability{
				PreferredModels: []string{"gpt-4-turbo"},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := NewOpenAIBackend(Config{
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

func TestOpenAIBackend_AvgLatencyMs(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:           "test",
			Name:         "Test",
			AvgLatencyMs: 300,
		},
		APIKey: "test-key",
	})

	// Before any requests, should return configured average
	if lat := backend.AvgLatencyMs(); lat != 300 {
		t.Errorf("AvgLatencyMs() = %v, want 300", lat)
	}

	// After request, should return measured average
	backend.UpdateMetrics(150, true)
	if lat := backend.AvgLatencyMs(); lat != 150 {
		t.Errorf("AvgLatencyMs() after update = %v, want 150", lat)
	}

	// After second request
	backend.UpdateMetrics(250, true)
	if lat := backend.AvgLatencyMs(); lat != 200 {
		t.Errorf("AvgLatencyMs() after second update = %v, want 200", lat)
	}
}

func TestOpenAIBackend_ConcurrentMetricsUpdate(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	// Simulate concurrent metric updates
	done := make(chan bool)
	numGoroutines := 10
	updatesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < updatesPerGoroutine; j++ {
				backend.UpdateMetrics(100, true)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	metrics := backend.GetMetrics()
	expectedCount := int64(numGoroutines * updatesPerGoroutine)
	if metrics.RequestCount != expectedCount {
		t.Errorf("RequestCount = %v, want %v", metrics.RequestCount, expectedCount)
	}
	if metrics.SuccessCount != expectedCount {
		t.Errorf("SuccessCount = %v, want %v", metrics.SuccessCount, expectedCount)
	}
}

func TestOpenAIBackend_ContextCancellation(t *testing.T) {
	// Create a slow mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "response"}},
			},
		})
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
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
		Model:  "gpt-4",
		Prompt: "Hello",
	}

	_, err := backend.Generate(ctx, req)
	if err == nil {
		t.Error("Generate() should return error when context is cancelled")
	}

	// Verify metrics updated for failed request
	metrics := backend.GetMetrics()
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %v, want 1 after context cancellation", metrics.ErrorCount)
	}
}

func TestOpenAIBackend_HealthCheckTimeout(t *testing.T) {
	// Create a slow mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	// Health check should timeout
	ctx := context.Background()
	err := backend.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck() should timeout")
	}
	if backend.IsHealthy() {
		t.Error("Backend should not be healthy after timeout")
	}
}

func TestOpenAIBackend_GenerateStream(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:  "gpt-4",
		Prompt: "Hello",
	}

	_, err := backend.GenerateStream(ctx, req)
	if err == nil {
		t.Error("GenerateStream() should return error as it's not yet implemented")
	}
}

func TestOpenAIBackend_ListModelsError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	_, err := backend.ListModels(ctx)
	if err == nil {
		t.Error("ListModels() should return error for unauthorized request")
	}
}

func TestOpenAIBackend_EmbedError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.EmbedRequest{
		Model: "text-embedding-ada-002",
		Text:  "Test",
	}

	_, err := backend.Embed(ctx, req)
	if err == nil {
		t.Error("Embed() should return error for bad request")
	}
}

func TestOpenAIBackend_MetricsAccuracy(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey: "test-key",
	})

	// Test multiple successful and failed requests
	backend.UpdateMetrics(100, true)
	backend.UpdateMetrics(200, true)
	backend.UpdateMetrics(150, false)
	backend.UpdateMetrics(300, true)
	backend.UpdateMetrics(250, false)

	metrics := backend.GetMetrics()

	if metrics.RequestCount != 5 {
		t.Errorf("RequestCount = %v, want 5", metrics.RequestCount)
	}
	if metrics.SuccessCount != 3 {
		t.Errorf("SuccessCount = %v, want 3", metrics.SuccessCount)
	}
	if metrics.ErrorCount != 2 {
		t.Errorf("ErrorCount = %v, want 2", metrics.ErrorCount)
	}

	// Error rate should be 2/5 = 0.4
	expectedErrorRate := float32(2.0 / 5.0)
	if metrics.ErrorRate != expectedErrorRate {
		t.Errorf("ErrorRate = %v, want %v", metrics.ErrorRate, expectedErrorRate)
	}

	// Average latency: UpdateMetrics only recalculates on success
	// Request 1: success, lat=100, total=100, count=1, avg=100
	// Request 2: success, lat=200, total=300, count=2, avg=150
	// Request 3: fail, lat=150, total=300, count=3, avg=150 (not recalculated)
	// Request 4: success, lat=300, total=600, count=4, avg=150
	// Request 5: fail, lat=250, total=600, count=5, avg=150 (not recalculated)
	expectedAvgLatency := int32(150)
	if metrics.AvgLatencyMs != expectedAvgLatency {
		t.Errorf("AvgLatencyMs = %v, want %v", metrics.AvgLatencyMs, expectedAvgLatency)
	}
}

func TestOpenAIBackend_NoOptionsGenerate(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Verify no options set when nil
		if _, ok := reqBody["temperature"]; ok {
			t.Error("temperature should not be set when Options is nil")
		}
		if _, ok := reqBody["top_p"]; ok {
			t.Error("top_p should not be set when Options is nil")
		}
		if _, ok := reqBody["max_tokens"]; ok {
			t.Error("max_tokens should not be set when Options is nil")
		}

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "response"}},
			},
			"usage": map[string]int{"total_tokens": 10},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:   "gpt-4",
		Prompt:  "Hello",
		Options: nil, // No options
	}

	_, err := backend.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestOpenAIBackend_SupportsModelNoRestrictions(t *testing.T) {
	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID: "test",
			Name: "Test",
			ModelCapability: &backends.ModelCapability{
				SupportedModelPatterns: []string{}, // Empty means no restrictions
			},
		},
		APIKey: "test-key",
	})

	// Should support any model when patterns list is empty
	if !backend.SupportsModel("any-random-model") {
		t.Error("SupportsModel() should return true when SupportedModelPatterns is empty")
	}
	if !backend.SupportsModel("claude-3-opus") {
		t.Error("SupportsModel() should return true when SupportedModelPatterns is empty")
	}
}

func TestOpenAIBackend_GenerateWithZeroOptions(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Verify options with zero values are not set
		if _, ok := reqBody["temperature"]; ok {
			t.Error("temperature should not be set when value is 0")
		}
		if _, ok := reqBody["top_p"]; ok {
			t.Error("top_p should not be set when value is 0")
		}
		if _, ok := reqBody["max_tokens"]; ok {
			t.Error("max_tokens should not be set when value is 0")
		}

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "response"}},
			},
			"usage": map[string]int{"total_tokens": 10},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:  "gpt-4",
		Prompt: "Hello",
		Options: &backends.GenerationOptions{
			Temperature: 0,
			TopP:        0,
			MaxTokens:   0,
		},
	}

	_, err := backend.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestOpenAIBackend_ListModelsDecodeError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	_, err := backend.ListModels(ctx)
	if err == nil {
		t.Error("ListModels() should return error when response is invalid JSON")
	}
}

func TestOpenAIBackend_GenerateDecodeError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return invalid JSON
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.GenerateRequest{
		Model:  "gpt-4",
		Prompt: "Hello",
	}

	_, err := backend.Generate(ctx, req)
	if err == nil {
		t.Error("Generate() should return error when response is invalid JSON")
	}

	// Verify metrics updated for failed request
	metrics := backend.GetMetrics()
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %v, want 1", metrics.ErrorCount)
	}
}

func TestOpenAIBackend_EmbedDecodeError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return invalid JSON
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	backend, _ := NewOpenAIBackend(Config{
		BackendConfig: backends.BackendConfig{
			ID:   "test",
			Name: "Test",
		},
		APIKey:   "test-key",
		Endpoint: mockServer.URL,
	})

	ctx := context.Background()
	req := &backends.EmbedRequest{
		Model: "text-embedding-ada-002",
		Text:  "Test",
	}

	_, err := backend.Embed(ctx, req)
	if err == nil {
		t.Error("Embed() should return error when response is invalid JSON")
	}
}

func TestMatchesPatternEdgeCases(t *testing.T) {
	tests := []struct {
		modelName string
		pattern   string
		want      bool
	}{
		// Exact match
		{"exact-model", "exact-model", true},
		// Wildcard only
		{"anything", "*", true},
		// Prefix wildcard
		{"model-suffix", "*-suffix", true},
		{"no-match", "*-suffix", false},
		// Suffix wildcard
		{"prefix-model", "prefix-*", true},
		{"no-match", "prefix-*", false},
		// Contains wildcard
		{"prefix-middle-suffix", "*-middle-*", true},
		{"prefix-suffix", "*-middle-*", false},
		// No wildcard, no match
		{"different", "pattern", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelName+"_"+tt.pattern, func(t *testing.T) {
			if got := matchesPattern(tt.modelName, tt.pattern); got != tt.want {
				t.Errorf("matchesPattern(%v, %v) = %v, want %v", tt.modelName, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestOpenAIBackend_matchesPattern_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		pattern   string
		expected  bool
	}{
		{
			name:      "Universal wildcard",
			modelName: "gpt-4-turbo",
			pattern:   "*",
			expected:  true,
		},
		{
			name:      "Exact match",
			modelName: "gpt-4-turbo",
			pattern:   "gpt-4-turbo",
			expected:  true,
		},
		{
			name:      "Prefix wildcard match",
			modelName: "gpt-4-turbo-preview",
			pattern:   "gpt-4*",
			expected:  true,
		},
		{
			name:      "Prefix wildcard no match",
			modelName: "gpt-3.5-turbo",
			pattern:   "gpt-4*",
			expected:  false,
		},
		{
			name:      "Suffix wildcard match",
			modelName: "gpt-3.5-turbo",
			pattern:   "*turbo",
			expected:  true,
		},
		{
			name:      "Suffix wildcard no match",
			modelName: "gpt-3.5-turbo-16k",
			pattern:   "*turbo",
			expected:  false,
		},
		{
			name:      "Substring wildcard match",
			modelName: "gpt-4-vision-preview",
			pattern:   "*vision*",
			expected:  true,
		},
		{
			name:      "Substring wildcard no match",
			modelName: "gpt-4-turbo",
			pattern:   "*vision*",
			expected:  false,
		},
		{
			name:      "No pattern match",
			modelName: "claude-3-opus",
			pattern:   "gpt",
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
