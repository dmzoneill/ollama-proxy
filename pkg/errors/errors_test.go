package errors

import (
	"strings"
	"testing"
)

// TestNoBackendsError is part of the public API.
func TestNoBackendsError(t *testing.T) {
	tests := []struct {
		err          *NoBackendsError
		name         string
		wantContains []string
		wantCode     int
	}{
		{
			name: "no constraints",
			err: &NoBackendsError{
				TotalBackends:   3,
				HealthyBackends: 0,
			},
			wantCode:     CodeNoBackendsAvailable,
			wantContains: []string{"no backends available", "3 total", "0 healthy"},
		},
		{
			name: "with constraints",
			err: &NoBackendsError{
				TotalBackends:   3,
				HealthyBackends: 1,
				Constraints:     []string{strLatency100ms, "power < 50W"},
			},
			wantCode:     CodeNoBackendsAvailable,
			wantContains: []string{"no backends available", "3 total", "1 healthy", "constraints", strLatency100ms},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %v, want %v", got, tt.wantCode)
			}

			errMsg := tt.err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestBackendUnhealthyError is part of the public API.
func TestBackendUnhealthyError(t *testing.T) {
	err := &BackendUnhealthyError{
		BackendID: strOllamaGpu,
		Reason:    "connection refused",
	}

	if got := err.Code(); got != CodeBackendUnhealthy {
		t.Errorf("Code() = %v, want %v", got, CodeBackendUnhealthy)
	}

	errMsg := err.Error()
	wantContains := []string{strOllamaGpu, "unhealthy", "connection refused"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestBackendTimeoutError is part of the public API.
func TestBackendTimeoutError(t *testing.T) {
	err := &BackendTimeoutError{
		BackendID: strOllamaNpu,
		Operation: strGenerate,
		Duration:  str30s,
	}

	if got := err.Code(); got != CodeBackendTimeout {
		t.Errorf("Code() = %v, want %v", got, CodeBackendTimeout)
	}

	errMsg := err.Error()
	wantContains := []string{strOllamaNpu, "timeout", strGenerate, str30s}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestBackendCapacityError is part of the public API.
func TestBackendCapacityError(t *testing.T) {
	err := &BackendCapacityError{
		BackendID:  "ollama-cpu",
		QueueDepth: 100,
		MaxQueue:   100,
	}

	if got := err.Code(); got != CodeBackendCapacity {
		t.Errorf("Code() = %v, want %v", got, CodeBackendCapacity)
	}

	errMsg := err.Error()
	wantContains := []string{"ollama-cpu", "capacity", "100/100"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestBackendUnsupportedError is part of the public API.
func TestBackendUnsupportedError(t *testing.T) {
	err := &BackendUnsupportedError{
		BackendID: strOllamaNpu,
		Model:     strGpt4,
		Reason:    "model too large for NPU",
	}

	if got := err.Code(); got != CodeBackendUnsupported {
		t.Errorf("Code() = %v, want %v", got, CodeBackendUnsupported)
	}

	errMsg := err.Error()
	wantContains := []string{strOllamaNpu, strGpt4, "does not support", "model too large"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestCircuitBreakerOpenError is part of the public API.
func TestCircuitBreakerOpenError(t *testing.T) {
	err := &CircuitBreakerOpenError{
		BackendID: strOllamaGpu,
		Failures:  5,
	}

	if got := err.Code(); got != CodeCircuitBreakerOpen {
		t.Errorf("Code() = %v, want %v", got, CodeCircuitBreakerOpen)
	}

	errMsg := err.Error()
	wantContains := []string{strOllamaGpu, "circuit breaker", "5 failures"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestRoutingError is part of the public API.
func TestRoutingError(t *testing.T) {
	tests := []struct {
		err          *RoutingError
		name         string
		wantContains []string
		wantCode     int
	}{
		{
			name: "no constraints",
			err: &RoutingError{
				Reason:        "all backends unhealthy",
				Model:         strLlama2,
				BackendsTried: 3,
			},
			wantCode:     CodeRoutingFailed,
			wantContains: []string{strRoutingFailed, strLlama2, "all backends unhealthy", "tried 3 backends"},
		},
		{
			name: "with constraints",
			err: &RoutingError{
				Reason:        "no match",
				Model:         strLlama2,
				BackendsTried: 3,
				Constraints: map[string]interface{}{
					"max_latency": 100,
					"max_power":   50,
				},
			},
			wantCode:     CodeRoutingFailed,
			wantContains: []string{strRoutingFailed, strLlama2, "no match", "tried 3 backends", "constraints"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %v, want %v", got, tt.wantCode)
			}

			errMsg := tt.err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestThermalLimitError is part of the public API.
func TestThermalLimitError(t *testing.T) {
	err := &ThermalLimitError{
		Hardware:    "GPU",
		Temperature: 95.5,
		Limit:       90.0,
	}

	if got := err.Code(); got != CodeThermalLimitExceeded {
		t.Errorf("Code() = %v, want %v", got, CodeThermalLimitExceeded)
	}

	errMsg := err.Error()
	wantContains := []string{"thermal limit", "GPU", "95.5", "90.0"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestValidationError is part of the public API.
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:  strTemperature,
		Value:  2.5,
		Reason: "must be between 0.0 and 2.0",
	}

	if got := err.Code(); got != CodeInvalidRequest {
		t.Errorf("Code() = %v, want %v", got, CodeInvalidRequest)
	}

	errMsg := err.Error()
	wantContains := []string{"validation failed", strTemperature, "2.5", "must be between"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestInvalidModelError is part of the public API.
func TestInvalidModelError(t *testing.T) {
	err := &InvalidModelError{
		Model:  "invalid-model!",
		Reason: "contains invalid characters",
	}

	if got := err.Code(); got != CodeInvalidModel {
		t.Errorf("Code() = %v, want %v", got, CodeInvalidModel)
	}

	errMsg := err.Error()
	wantContains := []string{"invalid model", "invalid-model!", "invalid characters"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

// TestInvalidPromptError is part of the public API.
func TestInvalidPromptError(t *testing.T) {
	tests := []struct {
		err          *InvalidPromptError
		name         string
		wantContains []string
		wantCode     int
	}{
		{
			name: "with max length",
			err: &InvalidPromptError{
				Length:    150000,
				MaxLength: 100000,
				Reason:    "too long",
			},
			wantCode:     CodeInvalidPrompt,
			wantContains: []string{"invalid prompt", "too long", "150000", "100000"},
		},
		{
			name: "without max length",
			err: &InvalidPromptError{
				Reason: "empty prompt",
			},
			wantCode:     CodeInvalidPrompt,
			wantContains: []string{"invalid prompt", "empty prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %v, want %v", got, tt.wantCode)
			}

			errMsg := tt.err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestConfigError is part of the public API.
func TestConfigError(t *testing.T) {
	tests := []struct {
		err          *ConfigError
		name         string
		wantContains []string
		wantCode     int
	}{
		{
			name: "with field",
			err: &ConfigError{
				Path:   strConfigConfigYaml,
				Field:  "grpc_port",
				Reason: "must be between 1 and 65535",
			},
			wantCode:     CodeConfigInvalid,
			wantContains: []string{"config error", strConfigConfigYaml, "grpc_port", "must be between"},
		},
		{
			name: "without field",
			err: &ConfigError{
				Path:   strConfigConfigYaml,
				Reason: "file not found",
			},
			wantCode:     CodeConfigInvalid,
			wantContains: []string{"config error", strConfigConfigYaml, "file not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %v, want %v", got, tt.wantCode)
			}

			errMsg := tt.err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestPipelineError is part of the public API.
func TestPipelineError(t *testing.T) {
	tests := []struct {
		err          *PipelineError
		name         string
		wantContains []string
		wantCode     int
	}{
		{
			name: "with stage",
			err: &PipelineError{
				PipelineID: "text-processing",
				Stage:      "tokenization",
				Reason:     "model not loaded",
			},
			wantCode:     CodePipelineExecutionFailed,
			wantContains: []string{"pipeline text-processing", "stage 'tokenization'", "model not loaded"},
		},
		{
			name: "without stage",
			err: &PipelineError{
				PipelineID: "text-processing",
				Reason:     "pipeline not found",
			},
			wantCode:     CodePipelineExecutionFailed,
			wantContains: []string{"pipeline text-processing", "failed", "pipeline not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %v, want %v", got, tt.wantCode)
			}

			errMsg := tt.err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestProxyError is part of the public API.
func TestProxyError(t *testing.T) {
	tests := []struct {
		name         string
		err          *ProxyError
		wantContains []string
	}{
		{
			name: "without context",
			err: &ProxyError{
				Code:    1000,
				Message: "test error",
			},
			wantContains: []string{"[1000]", "test error"},
		},
		{
			name: "with context",
			err: &ProxyError{
				Code:    2000,
				Message: strRoutingFailed,
				Context: map[string]interface{}{
					"model":    strLlama2,
					"backends": 3,
				},
			},
			wantContains: []string{"[2000]", strRoutingFailed, "context"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

// Test helper functions
func TestNewNoBackendsError(t *testing.T) {
	err := NewNoBackendsError(3, 1, []string{strLatency100ms})
	if err.TotalBackends != 3 {
		t.Errorf("TotalBackends = %v, want 3", err.TotalBackends)
	}
	if err.HealthyBackends != 1 {
		t.Errorf("HealthyBackends = %v, want 1", err.HealthyBackends)
	}
	if len(err.Constraints) != 1 {
		t.Errorf("len(Constraints) = %v, want 1", len(err.Constraints))
	}
}

// TestNewBackendUnhealthyError is part of the public API.
func TestNewBackendUnhealthyError(t *testing.T) {
	err := NewBackendUnhealthyError(strTestBackend, "connection failed")
	if err.BackendID != strTestBackend {
		t.Errorf("BackendID = %v, want test-backend", err.BackendID)
	}
	if err.Reason != "connection failed" {
		t.Errorf("Reason = %v, want connection failed", err.Reason)
	}
}

// TestNewBackendTimeoutError is part of the public API.
func TestNewBackendTimeoutError(t *testing.T) {
	err := NewBackendTimeoutError(strTestBackend, strGenerate, str30s)
	if err.BackendID != strTestBackend {
		t.Errorf("BackendID = %v, want test-backend", err.BackendID)
	}
	if err.Operation != strGenerate {
		t.Errorf("Operation = %v, want generate", err.Operation)
	}
	if err.Duration != str30s {
		t.Errorf("Duration = %v, want 30s", err.Duration)
	}
}

// TestNewBackendUnsupportedError is part of the public API.
func TestNewBackendUnsupportedError(t *testing.T) {
	err := NewBackendUnsupportedError(strTestBackend, strGpt4, "model too large")
	if err.BackendID != strTestBackend {
		t.Errorf("BackendID = %v, want test-backend", err.BackendID)
	}
	if err.Model != strGpt4 {
		t.Errorf("Model = %v, want gpt-4", err.Model)
	}
	if err.Reason != "model too large" {
		t.Errorf("Reason = %v, want model too large", err.Reason)
	}
}

// TestNewValidationError is part of the public API.
func TestNewValidationError(t *testing.T) {
	err := NewValidationError(strTemperature, 2.5, "out of range")
	if err.Field != strTemperature {
		t.Errorf("Field = %v, want temperature", err.Field)
	}
	if err.Value != 2.5 {
		t.Errorf("Value = %v, want 2.5", err.Value)
	}
	if err.Reason != "out of range" {
		t.Errorf("Reason = %v, want out of range", err.Reason)
	}
}

// TestNewInvalidModelError is part of the public API.
func TestNewInvalidModelError(t *testing.T) {
	err := NewInvalidModelError("test-model", "invalid format")
	if err.Model != "test-model" {
		t.Errorf("Model = %v, want test-model", err.Model)
	}
	if err.Reason != "invalid format" {
		t.Errorf("Reason = %v, want invalid format", err.Reason)
	}
}

// TestNewConfigError is part of the public API.
func TestNewConfigError(t *testing.T) {
	err := NewConfigError("config.yaml", "port", "invalid port")
	if err.Path != "config.yaml" {
		t.Errorf("Path = %v, want config.yaml", err.Path)
	}
	if err.Field != "port" {
		t.Errorf("Field = %v, want port", err.Field)
	}
	if err.Reason != "invalid port" {
		t.Errorf("Reason = %v, want invalid port", err.Reason)
	}
}

// TestNewPipelineError is part of the public API.
func TestNewPipelineError(t *testing.T) {
	err := NewPipelineError("test-pipeline", "stage1", "execution failed")
	if err.PipelineID != "test-pipeline" {
		t.Errorf("PipelineID = %v, want test-pipeline", err.PipelineID)
	}
	if err.Stage != "stage1" {
		t.Errorf("Stage = %v, want stage1", err.Stage)
	}
	if err.Reason != "execution failed" {
		t.Errorf("Reason = %v, want execution failed", err.Reason)
	}
}
