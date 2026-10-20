// Package errors defines proxy error types and HTTP status codes.
package errors

import (
	"fmt"
)

// Error codes for client parsing and handling
const (
	// Backend errors (1xxx)
	CodeNoBackendsAvailable = 1001
	CodeBackendUnhealthy    = 1002
	CodeBackendTimeout      = 1003
	CodeBackendCapacity     = 1004
	CodeBackendUnsupported  = 1005
	CodeCircuitBreakerOpen  = 1006

	// Routing errors (2xxx)
	CodeRoutingFailed        = 2001
	CodeNoMatchingBackend    = 2002
	CodeConstraintsNotMet    = 2003
	CodeThermalLimitExceeded = 2004

	// Request errors (3xxx)
	CodeInvalidRequest    = 3001
	CodeInvalidModel      = 3002
	CodeInvalidPrompt     = 3003
	CodeInvalidBackendID  = 3004
	CodeRequestTooLarge   = 3005
	CodeInvalidParameters = 3006

	// Configuration errors (4xxx)
	CodeConfigInvalid     = 4001
	CodeConfigNotFound    = 4002
	CodeConfigParseFailed = 4003

	// Pipeline errors (5xxx)
	CodePipelineNotFound        = 5001
	CodePipelineExecutionFailed = 5002
	CodeInvalidPipeline         = 5003
)

// ProxyError is the base error type with code and context
type ProxyError struct {
	Context map[string]interface{}
	Message string
	Code    int
}

// Error is part of the public API.
func (e *ProxyError) Error() string {
	if len(e.Context) == 0 {
		return fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("[%d] %s (context: %v)", e.Code, e.Message, e.Context)
}

// NoBackendsError indicates no backends are available
type NoBackendsError struct {
	Constraints     []string
	TotalBackends   int
	HealthyBackends int
}

// Error is part of the public API.
func (e *NoBackendsError) Error() string {
	if len(e.Constraints) == 0 {
		return fmt.Sprintf("no backends available: %d total, %d healthy",
			e.TotalBackends, e.HealthyBackends)
	}
	return fmt.Sprintf("no backends available matching constraints: %d total, %d healthy, constraints: %v",
		e.TotalBackends, e.HealthyBackends, e.Constraints)
}

// Code is part of the public API.
func (e *NoBackendsError) Code() int {
	return CodeNoBackendsAvailable
}

// BackendUnhealthyError indicates a backend failed health check
type BackendUnhealthyError struct {
	BackendID string
	Reason    string
}

// Error is part of the public API.
func (e *BackendUnhealthyError) Error() string {
	return fmt.Sprintf("backend %s is unhealthy: %s", e.BackendID, e.Reason)
}

// Code is part of the public API.
func (e *BackendUnhealthyError) Code() int {
	return CodeBackendUnhealthy
}

// BackendTimeoutError indicates a backend request timed out
type BackendTimeoutError struct {
	BackendID string
	Operation string
	Duration  string
}

// Error is part of the public API.
func (e *BackendTimeoutError) Error() string {
	return fmt.Sprintf("backend %s timeout during %s after %s", e.BackendID, e.Operation, e.Duration)
}

// Code is part of the public API.
func (e *BackendTimeoutError) Code() int {
	return CodeBackendTimeout
}

// BackendCapacityError indicates a backend is at capacity
type BackendCapacityError struct {
	BackendID  string
	QueueDepth int
	MaxQueue   int
}

// Error is part of the public API.
func (e *BackendCapacityError) Error() string {
	return fmt.Sprintf("backend %s at capacity: queue depth %d/%d", e.BackendID, e.QueueDepth, e.MaxQueue)
}

// Code is part of the public API.
func (e *BackendCapacityError) Code() int {
	return CodeBackendCapacity
}

// BackendUnsupportedError indicates a backend doesn't support the operation
type BackendUnsupportedError struct {
	BackendID string
	Model     string
	Reason    string
}

// Error is part of the public API.
func (e *BackendUnsupportedError) Error() string {
	return fmt.Sprintf("backend %s does not support model %s: %s", e.BackendID, e.Model, e.Reason)
}

// Code is part of the public API.
func (e *BackendUnsupportedError) Code() int {
	return CodeBackendUnsupported
}

// CircuitBreakerOpenError indicates a circuit breaker is open
type CircuitBreakerOpenError struct {
	BackendID string
	Failures  int
}

// Error is part of the public API.
func (e *CircuitBreakerOpenError) Error() string {
	return fmt.Sprintf("backend %s circuit breaker open after %d failures", e.BackendID, e.Failures)
}

// Code is part of the public API.
func (e *CircuitBreakerOpenError) Code() int {
	return CodeCircuitBreakerOpen
}

// RoutingError indicates routing failed
type RoutingError struct {
	Constraints   map[string]interface{}
	Reason        string
	Model         string
	BackendsTried int
}

// Error is part of the public API.
func (e *RoutingError) Error() string {
	if len(e.Constraints) == 0 {
		return fmt.Sprintf("routing failed for model %s: %s (tried %d backends)",
			e.Model, e.Reason, e.BackendsTried)
	}
	return fmt.Sprintf("routing failed for model %s: %s (tried %d backends, constraints: %v)",
		e.Model, e.Reason, e.BackendsTried, e.Constraints)
}

// Code is part of the public API.
func (e *RoutingError) Code() int {
	return CodeRoutingFailed
}

// ThermalLimitError indicates thermal limits exceeded
type ThermalLimitError struct {
	Hardware    string
	Temperature float64
	Limit       float64
}

// Error is part of the public API.
func (e *ThermalLimitError) Error() string {
	return fmt.Sprintf("thermal limit exceeded on %s: %.1f°C (limit: %.1f°C)",
		e.Hardware, e.Temperature, e.Limit)
}

// Code is part of the public API.
func (e *ThermalLimitError) Code() int {
	return CodeThermalLimitExceeded
}

// ValidationError indicates invalid input
type ValidationError struct {
	Field  string
	Value  interface{}
	Reason string
}

// Error is part of the public API.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s (value: %v)", e.Field, e.Reason, e.Value)
}

// Code is part of the public API.
func (e *ValidationError) Code() int {
	return CodeInvalidRequest
}

// InvalidModelError indicates an invalid model name
type InvalidModelError struct {
	Model  string
	Reason string
}

// Error is part of the public API.
func (e *InvalidModelError) Error() string {
	return fmt.Sprintf("invalid model '%s': %s", e.Model, e.Reason)
}

// Code is part of the public API.
func (e *InvalidModelError) Code() int {
	return CodeInvalidModel
}

// InvalidPromptError indicates an invalid prompt
type InvalidPromptError struct {
	Reason    string
	Length    int
	MaxLength int
}

// Error is part of the public API.
func (e *InvalidPromptError) Error() string {
	if e.MaxLength > 0 {
		return fmt.Sprintf("invalid prompt: %s (length: %d, max: %d)", e.Reason, e.Length, e.MaxLength)
	}
	return fmt.Sprintf("invalid prompt: %s", e.Reason)
}

// Code is part of the public API.
func (e *InvalidPromptError) Code() int {
	return CodeInvalidPrompt
}

// ConfigError indicates configuration error
type ConfigError struct {
	Path   string
	Field  string
	Reason string
}

// Error is part of the public API.
func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config error in %s (field '%s'): %s", e.Path, e.Field, e.Reason)
	}
	return fmt.Sprintf("config error in %s: %s", e.Path, e.Reason)
}

// Code is part of the public API.
func (e *ConfigError) Code() int {
	return CodeConfigInvalid
}

// PipelineError indicates pipeline execution error
type PipelineError struct {
	PipelineID string
	Stage      string
	Reason     string
}

// Error is part of the public API.
func (e *PipelineError) Error() string {
	if e.Stage != "" {
		return fmt.Sprintf("pipeline %s failed at stage '%s': %s", e.PipelineID, e.Stage, e.Reason)
	}
	return fmt.Sprintf("pipeline %s failed: %s", e.PipelineID, e.Reason)
}

// Code is part of the public API.
func (e *PipelineError) Code() int {
	return CodePipelineExecutionFailed
}

// Helper functions to create common errors

// NewNoBackendsError creates a new NoBackendsError
func NewNoBackendsError(total, healthy int, constraints []string) *NoBackendsError {
	return &NoBackendsError{
		TotalBackends:   total,
		HealthyBackends: healthy,
		Constraints:     constraints,
	}
}

// NewBackendUnhealthyError creates a new BackendUnhealthyError
func NewBackendUnhealthyError(backendID, reason string) *BackendUnhealthyError {
	return &BackendUnhealthyError{
		BackendID: backendID,
		Reason:    reason,
	}
}

// NewBackendTimeoutError creates a new BackendTimeoutError
func NewBackendTimeoutError(backendID, operation, duration string) *BackendTimeoutError {
	return &BackendTimeoutError{
		BackendID: backendID,
		Operation: operation,
		Duration:  duration,
	}
}

// NewBackendUnsupportedError creates a new BackendUnsupportedError
func NewBackendUnsupportedError(backendID, model, reason string) *BackendUnsupportedError {
	return &BackendUnsupportedError{
		BackendID: backendID,
		Model:     model,
		Reason:    reason,
	}
}

// NewValidationError creates a new ValidationError
func NewValidationError(field string, value interface{}, reason string) *ValidationError {
	return &ValidationError{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

// NewInvalidModelError creates a new InvalidModelError
func NewInvalidModelError(model, reason string) *InvalidModelError {
	return &InvalidModelError{
		Model:  model,
		Reason: reason,
	}
}

// NewConfigError creates a new ConfigError
func NewConfigError(path, field, reason string) *ConfigError {
	return &ConfigError{
		Path:   path,
		Field:  field,
		Reason: reason,
	}
}

// NewPipelineError creates a new PipelineError
func NewPipelineError(pipelineID, stage, reason string) *PipelineError {
	return &PipelineError{
		PipelineID: pipelineID,
		Stage:      stage,
		Reason:     reason,
	}
}
