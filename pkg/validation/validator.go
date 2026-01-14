package validation

import (
	"fmt"
	"regexp"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

var (
	modelNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.:]+$`)
	backendIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

const (
	MaxPromptLength    = 100000 // 100KB
	MaxModelNameLength = 256
	MinTemperature     = 0.0
	MaxTemperature     = 2.0
	MinTopP            = 0.0
	MaxTopP            = 1.0
	MaxTopK            = 100
	MaxMaxTokens       = 100000
)

// ValidateModelName validates a model name
func ValidateModelName(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	if len(model) > MaxModelNameLength {
		return fmt.Errorf("model name too long: %d chars (max %d)",
			len(model), MaxModelNameLength)
	}
	if !modelNameRegex.MatchString(model) {
		return fmt.Errorf("invalid model name format: %s (allowed: alphanumeric, _, -, ., :)", model)
	}
	return nil
}

// ValidatePrompt validates a prompt
func ValidatePrompt(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	if len(prompt) > MaxPromptLength {
		return fmt.Errorf("prompt too long: %d chars (max %d)",
			len(prompt), MaxPromptLength)
	}
	return nil
}

// ValidateGenerationOptions validates generation options
func ValidateGenerationOptions(opts *backends.GenerationOptions) error {
	if opts == nil {
		return nil
	}

	if opts.Temperature < MinTemperature || opts.Temperature > MaxTemperature {
		return fmt.Errorf("temperature %.2f out of range [%.1f, %.1f]",
			opts.Temperature, MinTemperature, MaxTemperature)
	}

	if opts.TopP < MinTopP || opts.TopP > MaxTopP {
		return fmt.Errorf("top_p %.2f out of range [%.1f, %.1f]",
			opts.TopP, MinTopP, MaxTopP)
	}

	if opts.TopK < 0 || opts.TopK > MaxTopK {
		return fmt.Errorf("top_k %d out of range [0, %d]",
			opts.TopK, MaxTopK)
	}

	if opts.MaxTokens < 0 || opts.MaxTokens > MaxMaxTokens {
		return fmt.Errorf("max_tokens %d out of range [0, %d]",
			opts.MaxTokens, MaxMaxTokens)
	}

	if opts.ContextLength < 0 {
		return fmt.Errorf("context_length cannot be negative: %d", opts.ContextLength)
	}

	return nil
}

// ValidateBackendID validates a backend ID
func ValidateBackendID(id string) error {
	if id == "" || id == "auto" {
		return nil // Empty or "auto" is valid
	}
	if !backendIDRegex.MatchString(id) {
		return fmt.Errorf("invalid backend ID format: %s (allowed: alphanumeric, _, -)", id)
	}
	return nil
}

// ValidateGenerateRequest validates a generate request
func ValidateGenerateRequest(req *backends.GenerateRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if err := ValidatePrompt(req.Prompt); err != nil {
		return fmt.Errorf("invalid prompt: %w", err)
	}

	if err := ValidateModelName(req.Model); err != nil {
		return fmt.Errorf("invalid model: %w", err)
	}

	if err := ValidateGenerationOptions(req.Options); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	return nil
}

// ValidateEmbedRequest validates an embed request
func ValidateEmbedRequest(req *backends.EmbedRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.Text == "" {
		return fmt.Errorf("text cannot be empty")
	}

	if len(req.Text) > MaxPromptLength {
		return fmt.Errorf("text too long: %d chars (max %d)",
			len(req.Text), MaxPromptLength)
	}

	if err := ValidateModelName(req.Model); err != nil {
		return fmt.Errorf("invalid model: %w", err)
	}

	return nil
}

// ValidateAnnotations validates routing annotations
func ValidateAnnotations(ann *backends.Annotations) error {
	if ann == nil {
		return nil
	}

	if err := ValidateBackendID(ann.Target); err != nil {
		return fmt.Errorf("invalid target backend: %w", err)
	}

	if ann.MaxLatencyMs < 0 {
		return fmt.Errorf("max_latency_ms cannot be negative: %d", ann.MaxLatencyMs)
	}

	if ann.MaxPowerWatts < 0 {
		return fmt.Errorf("max_power_watts cannot be negative: %d", ann.MaxPowerWatts)
	}

	if ann.DeadlineMs < 0 {
		return fmt.Errorf("deadline_ms cannot be negative: %d", ann.DeadlineMs)
	}

	// Validate priority
	if ann.Priority < backends.PriorityBestEffort || ann.Priority > backends.PriorityCritical {
		return fmt.Errorf("invalid priority: %d (must be 0-3)", ann.Priority)
	}

	// Validate media type
	validMediaTypes := map[backends.MediaType]bool{
		backends.MediaTypeText:     true,
		backends.MediaTypeCode:     true,
		backends.MediaTypeAudio:    true,
		backends.MediaTypeImage:    true,
		backends.MediaTypeRealtime: true,
		backends.MediaTypeAuto:     true,
		"":                         true, // Empty is valid
	}
	if !validMediaTypes[ann.MediaType] {
		return fmt.Errorf("invalid media_type: %s", ann.MediaType)
	}

	return nil
}
