package errors

import (
	"strings"
	"testing"
)

func TestNoBackendsError(t *testing.T) {
	tests := []struct {
		name        string
		err         *NoBackendsError
		wantCode    int
		wantContains []string
	}{
		{
			name: "no constraints",
			err: &NoBackendsError{
				TotalBackends:   3,
				HealthyBackends: 0,
			},
			wantCode:    CodeNoBackendsAvailable,
			wantContains: []string{"no backends available", "3 total", "0 healthy"},
		},
		{
			name: "with constraints",
			err: &NoBackendsError{
				TotalBackends:   3,
				HealthyBackends: 1,
				Constraints:     []string{"latency < 100ms", "power < 50W"},
			},
			wantCode:    CodeNoBackendsAvailable,
			wantContains: []string{"no backends available", "3 total", "1 healthy", "constraints", "latency < 100ms"},
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

func TestBackendUnhealthyError(t *testing.T) {
	err := &BackendUnhealthyError{
		BackendID: "ollama-gpu",
		Reason:    "connection refused",
	}

	if got := err.Code(); got != CodeBackendUnhealthy {
		t.Errorf("Code() = %v, want %v", got, CodeBackendUnhealthy)
	}

	errMsg := err.Error()
	wantContains := []string{"ollama-gpu", "unhealthy", "connection refused"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

func TestBackendTimeoutError(t *testing.T) {
	err := &BackendTimeoutError{
		BackendID: "ollama-npu",
		Operation: "generate",
		Duration:  "30s",
	}

	if got := err.Code(); got != CodeBackendTimeout {
		t.Errorf("Code() = %v, want %v", got, CodeBackendTimeout)
	}

	errMsg := err.Error()
	wantContains := []string{"ollama-npu", "timeout", "generate", "30s"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

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

func TestBackendUnsupportedError(t *testing.T) {
	err := &BackendUnsupportedError{
		BackendID: "ollama-npu",
		Model:     "gpt-4",
		Reason:    "model too large for NPU",
	}

	if got := err.Code(); got != CodeBackendUnsupported {
		t.Errorf("Code() = %v, want %v", got, CodeBackendUnsupported)
	}

	errMsg := err.Error()
	wantContains := []string{"ollama-npu", "gpt-4", "does not support", "model too large"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

func TestCircuitBreakerOpenError(t *testing.T) {
	err := &CircuitBreakerOpenError{
		BackendID: "ollama-gpu",
		Failures:  5,
	}

	if got := err.Code(); got != CodeCircuitBreakerOpen {
		t.Errorf("Code() = %v, want %v", got, CodeCircuitBreakerOpen)
	}

	errMsg := err.Error()
	wantContains := []string{"ollama-gpu", "circuit breaker", "5 failures"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

func TestRoutingError(t *testing.T) {
	tests := []struct {
		name        string
		err         *RoutingError
		wantCode    int
		wantContains []string
	}{
		{
			name: "no constraints",
			err: &RoutingError{
				Reason:        "all backends unhealthy",
				Model:         "llama2",
				BackendsTried: 3,
			},
			wantCode:    CodeRoutingFailed,
			wantContains: []string{"routing failed", "llama2", "all backends unhealthy", "tried 3 backends"},
		},
		{
			name: "with constraints",
			err: &RoutingError{
				Reason:        "no match",
				Model:         "llama2",
				BackendsTried: 3,
				Constraints: map[string]interface{}{
					"max_latency": 100,
					"max_power":   50,
				},
			},
			wantCode:    CodeRoutingFailed,
			wantContains: []string{"routing failed", "llama2", "no match", "tried 3 backends", "constraints"},
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

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:  "temperature",
		Value:  2.5,
		Reason: "must be between 0.0 and 2.0",
	}

	if got := err.Code(); got != CodeInvalidRequest {
		t.Errorf("Code() = %v, want %v", got, CodeInvalidRequest)
	}

	errMsg := err.Error()
	wantContains := []string{"validation failed", "temperature", "2.5", "must be between"}
	for _, want := range wantContains {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, want to contain %q", errMsg, want)
		}
	}
}

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

func TestInvalidPromptError(t *testing.T) {
	tests := []struct {
		name        string
		err         *InvalidPromptError
		wantCode    int
		wantContains []string
	}{
		{
			name: "with max length",
			err: &InvalidPromptError{
				Length:    150000,
				MaxLength: 100000,
				Reason:    "too long",
			},
			wantCode:    CodeInvalidPrompt,
			wantContains: []string{"invalid prompt", "too long", "150000", "100000"},
		},
		{
			name: "without max length",
			err: &InvalidPromptError{
				Reason: "empty prompt",
			},
			wantCode:    CodeInvalidPrompt,
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

func TestConfigError(t *testing.T) {
	tests := []struct {
		name        string
		err         *ConfigError
		wantCode    int
		wantContains []string
	}{
		{
			name: "with field",
			err: &ConfigError{
				Path:   "config/config.yaml",
				Field:  "grpc_port",
				Reason: "must be between 1 and 65535",
			},
			wantCode:    CodeConfigInvalid,
			wantContains: []string{"config error", "config/config.yaml", "grpc_port", "must be between"},
		},
		{
			name: "without field",
			err: &ConfigError{
				Path:   "config/config.yaml",
				Reason: "file not found",
			},
			wantCode:    CodeConfigInvalid,
			wantContains: []string{"config error", "config/config.yaml", "file not found"},
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

func TestPipelineError(t *testing.T) {
	tests := []struct {
		name        string
		err         *PipelineError
		wantCode    int
		wantContains []string
	}{
		{
			name: "with stage",
			err: &PipelineError{
				PipelineID: "text-processing",
				Stage:      "tokenization",
				Reason:     "model not loaded",
			},
			wantCode:    CodePipelineExecutionFailed,
			wantContains: []string{"pipeline text-processing", "stage 'tokenization'", "model not loaded"},
		},
		{
			name: "without stage",
			err: &PipelineError{
				PipelineID: "text-processing",
				Reason:     "pipeline not found",
			},
			wantCode:    CodePipelineExecutionFailed,
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

func TestProxyError(t *testing.T) {
	tests := []struct {
		name        string
		err         *ProxyError
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
				Message: "routing failed",
				Context: map[string]interface{}{
					"model":    "llama2",
					"backends": 3,
				},
			},
			wantContains: []string{"[2000]", "routing failed", "context"},
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
	err := NewNoBackendsError(3, 1, []string{"latency < 100ms"})
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

func TestNewBackendUnhealthyError(t *testing.T) {
	err := NewBackendUnhealthyError("test-backend", "connection failed")
	if err.BackendID != "test-backend" {
		t.Errorf("BackendID = %v, want test-backend", err.BackendID)
	}
	if err.Reason != "connection failed" {
		t.Errorf("Reason = %v, want connection failed", err.Reason)
	}
}

func TestNewBackendTimeoutError(t *testing.T) {
	err := NewBackendTimeoutError("test-backend", "generate", "30s")
	if err.BackendID != "test-backend" {
		t.Errorf("BackendID = %v, want test-backend", err.BackendID)
	}
	if err.Operation != "generate" {
		t.Errorf("Operation = %v, want generate", err.Operation)
	}
	if err.Duration != "30s" {
		t.Errorf("Duration = %v, want 30s", err.Duration)
	}
}

func TestNewBackendUnsupportedError(t *testing.T) {
	err := NewBackendUnsupportedError("test-backend", "gpt-4", "model too large")
	if err.BackendID != "test-backend" {
		t.Errorf("BackendID = %v, want test-backend", err.BackendID)
	}
	if err.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", err.Model)
	}
	if err.Reason != "model too large" {
		t.Errorf("Reason = %v, want model too large", err.Reason)
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("temperature", 2.5, "out of range")
	if err.Field != "temperature" {
		t.Errorf("Field = %v, want temperature", err.Field)
	}
	if err.Value != 2.5 {
		t.Errorf("Value = %v, want 2.5", err.Value)
	}
	if err.Reason != "out of range" {
		t.Errorf("Reason = %v, want out of range", err.Reason)
	}
}

func TestNewInvalidModelError(t *testing.T) {
	err := NewInvalidModelError("test-model", "invalid format")
	if err.Model != "test-model" {
		t.Errorf("Model = %v, want test-model", err.Model)
	}
	if err.Reason != "invalid format" {
		t.Errorf("Reason = %v, want invalid format", err.Reason)
	}
}

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
