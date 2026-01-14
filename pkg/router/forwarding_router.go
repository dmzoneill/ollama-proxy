package router

import (
	"context"
	"fmt"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/confidence"
)

// ForwardingRouter extends Router with confidence-based forwarding
type ForwardingRouter struct {
	baseRouter         *Router
	thermalRouter      *ThermalRouter
	confidenceEstimator *confidence.ConfidenceEstimator
	config             *ForwardingConfig
}

// ForwardingConfig configures forwarding behavior
type ForwardingConfig struct {
	// Enable forwarding
	Enabled bool

	// Confidence thresholds
	MinConfidence    float64 // Minimum acceptable confidence (0.75 = 75%)
	MaxRetries       int     // Maximum forwarding attempts (3)

	// Escalation strategy
	EscalationPath   []string // Ordered list of backend IDs to try

	// Thermal integration
	RespectThermalLimits bool // Skip backends that are unhealthy

	// Fallback behavior
	ReturnBestAttempt bool // Return best attempt even if below threshold
}

// DefaultForwardingConfig returns sensible defaults
func DefaultForwardingConfig() *ForwardingConfig {
	return &ForwardingConfig{
		Enabled:              true,
		MinConfidence:        0.75,
		MaxRetries:           3,
		EscalationPath:       []string{}, // Set dynamically
		RespectThermalLimits: true,
		ReturnBestAttempt:    true,
	}
}

// NewForwardingRouter creates a router with forwarding support
func NewForwardingRouter(
	baseRouter *Router,
	thermalRouter *ThermalRouter,
	config *ForwardingConfig,
) *ForwardingRouter {
	if config == nil {
		config = DefaultForwardingConfig()
	}

	return &ForwardingRouter{
		baseRouter:         baseRouter,
		thermalRouter:      thermalRouter,
		confidenceEstimator: confidence.NewConfidenceEstimator(nil),
		config:             config,
	}
}

// ForwardingAttempt represents one attempt in the forwarding chain
type ForwardingAttempt struct {
	Backend        backends.Backend
	BackendID      string
	Response       string
	Confidence     *confidence.ConfidenceScore
	LatencyMs      int32
	Success        bool
	Error          error
	SkipReason     string // Why this backend was skipped (if applicable)
}

// ForwardingResult represents the complete forwarding chain result
type ForwardingResult struct {
	// Final result
	FinalResponse  string
	FinalBackend   backends.Backend
	FinalConfidence *confidence.ConfidenceScore

	// Forwarding history
	Attempts       []*ForwardingAttempt
	TotalAttempts  int
	Forwarded      bool

	// Timing
	TotalLatencyMs int32

	// Decision info
	Decision       *RoutingDecision
	Reasoning      []string
}

// GenerateWithForwarding performs generation with confidence-based forwarding
func (fr *ForwardingRouter) GenerateWithForwarding(
	ctx context.Context,
	prompt string,
	model string,
	annotations *backends.Annotations,
) (*ForwardingResult, error) {

	result := &ForwardingResult{
		Attempts:  make([]*ForwardingAttempt, 0),
		Reasoning: make([]string, 0),
	}

	startTime := time.Now()

	// Build escalation path if not specified
	escalationPath := fr.config.EscalationPath
	if len(escalationPath) == 0 {
		escalationPath = fr.buildEscalationPath(model)
	}

	result.Reasoning = append(result.Reasoning,
		fmt.Sprintf("Escalation path: %v", escalationPath))

	// Try each backend in escalation path
	bestAttempt := &ForwardingAttempt{}
	bestConfidence := 0.0

	for attemptNum, backendID := range escalationPath {
		if attemptNum >= fr.config.MaxRetries {
			result.Reasoning = append(result.Reasoning,
				fmt.Sprintf("Max retries (%d) reached", fr.config.MaxRetries))
			break
		}

		// Get backend
		backend := fr.findBackend(backendID)
		if backend == nil {
			result.Reasoning = append(result.Reasoning,
				fmt.Sprintf("Backend %s not found, skipping", backendID))
			continue
		}

		// Check if backend is healthy (thermal check)
		if fr.config.RespectThermalLimits && !backend.IsHealthy() {
			attempt := &ForwardingAttempt{
				Backend:    backend,
				BackendID:  backendID,
				Success:    false,
				SkipReason: "Backend unhealthy (thermal limits exceeded)",
			}
			result.Attempts = append(result.Attempts, attempt)
			result.Reasoning = append(result.Reasoning,
				fmt.Sprintf("Skipped %s: %s", backendID, attempt.SkipReason))
			continue
		}

		// Check if backend supports the model
		if !backend.SupportsModel(model) {
			attempt := &ForwardingAttempt{
				Backend:    backend,
				BackendID:  backendID,
				Success:    false,
				SkipReason: fmt.Sprintf("Model %s not supported", model),
			}
			result.Attempts = append(result.Attempts, attempt)
			result.Reasoning = append(result.Reasoning,
				fmt.Sprintf("Skipped %s: %s", backendID, attempt.SkipReason))
			continue
		}

		// Execute generation on this backend
		attempt := fr.tryBackend(ctx, backend, backendID, prompt, model, annotations)
		result.Attempts = append(result.Attempts, attempt)
		result.TotalAttempts++

		if !attempt.Success {
			result.Reasoning = append(result.Reasoning,
				fmt.Sprintf("Attempt %d on %s failed: %v", attemptNum+1, backendID, attempt.Error))
			continue
		}

		// Track best attempt
		if attempt.Confidence.Overall > bestConfidence {
			bestAttempt = attempt
			bestConfidence = attempt.Confidence.Overall
		}

		result.Reasoning = append(result.Reasoning,
			fmt.Sprintf("Attempt %d on %s: confidence %.2f (%s)",
				attemptNum+1, backendID, attempt.Confidence.Overall, attempt.Confidence.Reasoning))

		// Check if confidence meets threshold
		if attempt.Confidence.Overall >= fr.config.MinConfidence {
			// Success! Use this response
			result.FinalResponse = attempt.Response
			result.FinalBackend = backend
			result.FinalConfidence = attempt.Confidence
			result.Forwarded = attemptNum > 0

			result.Reasoning = append(result.Reasoning,
				fmt.Sprintf("✓ Confidence threshold met (%.2f >= %.2f), using response",
					attempt.Confidence.Overall, fr.config.MinConfidence))

			break
		}

		// Confidence too low, will try next backend
		result.Reasoning = append(result.Reasoning,
			fmt.Sprintf("✗ Confidence too low (%.2f < %.2f), forwarding to next backend",
				attempt.Confidence.Overall, fr.config.MinConfidence))
	}

	// If no attempt met threshold, use best attempt if configured
	if result.FinalResponse == "" && fr.config.ReturnBestAttempt && bestAttempt.Success {
		result.FinalResponse = bestAttempt.Response
		result.FinalBackend = bestAttempt.Backend
		result.FinalConfidence = bestAttempt.Confidence
		result.Forwarded = true

		result.Reasoning = append(result.Reasoning,
			fmt.Sprintf("No attempt met threshold, returning best attempt (confidence %.2f)",
				bestConfidence))
	}

	// If still no response, return error
	if result.FinalResponse == "" {
		return result, fmt.Errorf("all backends failed or returned low confidence")
	}

	result.TotalLatencyMs = int32(time.Since(startTime).Milliseconds())

	return result, nil
}

// tryBackend attempts generation on a specific backend
func (fr *ForwardingRouter) tryBackend(
	ctx context.Context,
	backend backends.Backend,
	backendID string,
	prompt string,
	model string,
	annotations *backends.Annotations,
) *ForwardingAttempt {
	attempt := &ForwardingAttempt{
		Backend:   backend,
		BackendID: backendID,
	}

	startTime := time.Now()

	// Execute generation
	req := &backends.GenerateRequest{
		Model:  model,
		Prompt: prompt,
	}

	resp, err := backend.Generate(ctx, req)
	if err != nil {
		attempt.Success = false
		attempt.Error = err
		return attempt
	}

	attempt.Response = resp.Response
	attempt.LatencyMs = int32(time.Since(startTime).Milliseconds())
	attempt.Success = true

	// Estimate confidence
	attempt.Confidence = fr.confidenceEstimator.Estimate(
		prompt,
		resp.Response,
		model,
		backend,
	)

	return attempt
}

// buildEscalationPath creates default escalation path based on model
func (fr *ForwardingRouter) buildEscalationPath(model string) []string {
	// Default escalation: NPU → Intel GPU → NVIDIA GPU
	// This maximizes battery life by starting with most efficient

	path := []string{}

	// Small models: try NPU first
	if isSmallModel(model) {
		path = append(path, "ollama-npu")
	}

	// Medium models: try Intel GPU
	path = append(path, "ollama-intel")

	// Large models or fallback: NVIDIA GPU
	path = append(path, "ollama-nvidia")

	// Final fallback: CPU
	path = append(path, "ollama-cpu")

	return path
}

// findBackend looks up backend by ID
func (fr *ForwardingRouter) findBackend(backendID string) backends.Backend {
	// Try base router first
	for _, backend := range fr.baseRouter.backends {
		if backend.ID() == backendID {
			return backend
		}
	}

	// Try thermal router if available
	if fr.thermalRouter != nil {
		for _, backend := range fr.thermalRouter.Router.backends {
			if backend.ID() == backendID {
				return backend
			}
		}
	}

	return nil
}

// isSmallModel checks if model is small enough for NPU
func isSmallModel(model string) bool {
	smallPatterns := []string{
		"0.5b", "1.5b", "tiny", "mini",
	}

	for _, pattern := range smallPatterns {
		if contains(model, pattern) {
			return true
		}
	}

	return false
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && len(substr) > 0 &&
		(s == substr ||
			(len(s) >= len(substr) && s[:len(substr)] == substr) ||
			(len(s) >= len(substr) && s[len(s)-len(substr):] == substr) ||
			(len(s) > len(substr) && findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GenerateWithForwardingStream performs streaming generation with forwarding
// Note: Forwarding in streaming mode is limited - can't easily switch mid-stream
// For now, we select best backend upfront based on prompt analysis
func (fr *ForwardingRouter) GenerateWithForwardingStream(
	ctx context.Context,
	prompt string,
	model string,
	annotations *backends.Annotations,
) (backends.StreamReader, backends.Backend, error) {

	// Pre-analyze prompt to select best backend
	escalationPath := fr.buildEscalationPath(model)

	// Try to predict which backend will succeed
	for _, backendID := range escalationPath {
		backend := fr.findBackend(backendID)
		if backend == nil {
			continue
		}

		// Check health
		if fr.config.RespectThermalLimits && !backend.IsHealthy() {
			continue
		}

		// Check model support
		if !backend.SupportsModel(model) {
			continue
		}

		// Estimate confidence for this backend+model combo
		estimatedConfidence := fr.confidenceEstimator.EstimateForPrompt(prompt, model)

		if estimatedConfidence >= fr.config.MinConfidence {
			// This backend should be good enough
			req := &backends.GenerateRequest{
				Model:  model,
				Prompt: prompt,
			}

			stream, err := backend.GenerateStream(ctx, req)
			if err != nil {
				// Try next backend
				continue
			}

			return stream, backend, nil
		}
	}

	return nil, nil, fmt.Errorf("no suitable backend found for streaming")
}

// SetEscalationPath allows dynamic escalation path configuration
func (fr *ForwardingRouter) SetEscalationPath(path []string) {
	fr.config.EscalationPath = path
}

// SetMinConfidence allows dynamic confidence threshold
func (fr *ForwardingRouter) SetMinConfidence(threshold float64) {
	fr.config.MinConfidence = threshold
}
