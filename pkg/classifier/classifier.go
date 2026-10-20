// Package classifier classifies prompts for routing decisions.
package classifier

import (
	"context"
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// RequestComplexity indicates how complex a request is
type RequestComplexity int

// ComplexitySimple is an exported const.
const (
	ComplexitySimple   RequestComplexity = iota // Simple queries, short answers
	ComplexityModerate                          // Normal chat, explanations
	ComplexityComplex                           // Long-form generation, analysis
)

// Classifier analyzes requests to determine appropriate routing
type Classifier struct {
	npu backends.Backend // Use NPU for classification
}

// NewClassifier constructs a classifier.
func NewClassifier(npu backends.Backend) *Classifier {
	return &Classifier{npu: npu}
}

var simplePromptPatterns = []string{
	"what is",
	"who is",
	"when was",
	"where is",
	"how old",
	"how many",
	"yes or no",
	"true or false",
}

var complexPromptPatterns = []string{
	"write a detailed",
	"explain in depth",
	"analyze",
	"compare and contrast",
	"create a comprehensive",
	"generate code",
	"write a story",
	"write an essay",
	"compose",
	"develop a plan",
}

// ClassifyPrompt determines request complexity
func (c *Classifier) ClassifyPrompt(_ context.Context, prompt string, model string) RequestComplexity {
	if len(prompt) < 50 {
		return ComplexitySimple
	}

	promptLower := strings.ToLower(prompt)
	if hasAnyPrefix(promptLower, simplePromptPatterns) {
		return ComplexitySimple
	}
	if hasAnyContains(promptLower, complexPromptPatterns) {
		return ComplexityComplex
	}
	if isBriefPrompt(promptLower) {
		return ComplexitySimple
	}
	if isMultiStepPrompt(promptLower) {
		return ComplexityModerate
	}
	if complexity, ok := complexityForModel(model); ok {
		return complexity
	}
	return ComplexityModerate
}

func hasAnyPrefix(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasPrefix(value, pattern) {
			return true
		}
	}
	return false
}

func hasAnyContains(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}

func isBriefPrompt(promptLower string) bool {
	return strings.Contains(promptLower, "briefly") ||
		strings.Contains(promptLower, "one sentence") ||
		strings.Contains(promptLower, "in short")
}

func isMultiStepPrompt(promptLower string) bool {
	return strings.Contains(promptLower, "step by step") ||
		strings.Contains(promptLower, "first,") ||
		strings.Count(promptLower, "?") > 1
}

func complexityForModel(model string) (RequestComplexity, bool) {
	if strings.Contains(model, "0.5b") || strings.Contains(model, "1.5b") {
		return ComplexitySimple, true
	}
	if strings.Contains(model, "70b") || strings.Contains(model, "33b") {
		return ComplexityComplex, true
	}
	return ComplexityModerate, false
}

// ClassifyWithModel uses a small LLM to classify (more accurate but slower)
func (c *Classifier) ClassifyWithModel(ctx context.Context, prompt string) (RequestComplexity, error) {
	// Use tiny model on NPU to classify the request
	classificationPrompt := `Classify this user request as SIMPLE, MODERATE, or COMPLEX:

SIMPLE: Short factual questions, basic lookups, yes/no questions
MODERATE: Normal explanations, standard chat, brief summaries
COMPLEX: Long-form writing, detailed analysis, code generation, creative writing

User request: "` + prompt + `"

Classification (respond with just one word - SIMPLE, MODERATE, or COMPLEX):`

	resp, err := c.npu.Generate(ctx, &backends.GenerateRequest{
		Prompt: classificationPrompt,
		Model:  strQwen2505b, // Use smallest model for classification
		Options: &backends.GenerationOptions{
			MaxTokens:   1,
			Temperature: 0.0, // Deterministic
		},
	})

	if err != nil {
		// Fall back to heuristic on error
		return ComplexityModerate, err
	}

	classification := strings.ToUpper(strings.TrimSpace(resp.Response))

	switch {
	case strings.Contains(classification, "SIMPLE"):
		return ComplexitySimple, nil
	case strings.Contains(classification, "COMPLEX"):
		return ComplexityComplex, nil
	default:
		return ComplexityModerate, nil
	}
}

// RecommendBackend suggests appropriate backend based on complexity
func (c *Classifier) RecommendBackend(complexity RequestComplexity, onBattery bool, queueDepth map[string]int) string {
	// Priority matrix
	switch complexity {
	case ComplexitySimple:
		// Simple requests: prefer NPU (ultra-efficient)
		if queueDepth[strOllamaNpu] < 3 {
			return strOllamaNpu
		}
		return strOllamaIgpu // Fallback to Intel GPU if NPU busy

	case ComplexityModerate:
		// Moderate requests: prefer Intel GPU (balanced)
		if onBattery {
			// On battery, prefer efficient backends
			if queueDepth[strOllamaIgpu] < 2 {
				return strOllamaIgpu
			}
			return strOllamaNpu
		}
		// On AC power, can use NVIDIA if Intel busy
		if queueDepth[strOllamaIgpu] < 2 {
			return strOllamaIgpu
		}
		return strOllamaNvidia

	case ComplexityComplex:
		// Complex requests: prefer NVIDIA (performance)
		if onBattery && queueDepth[strOllamaNvidia] > 1 {
			// On battery with NVIDIA busy, downgrade to Intel
			return strOllamaIgpu
		}
		return strOllamaNvidia
	}

	return strOllamaIgpu // Default
}

// ShouldAllowLatencyCritical checks if user's latency_critical flag is justified
func (c *Classifier) ShouldAllowLatencyCritical(_ context.Context, prompt string, userMarkedCritical bool) bool {
	// If user marked as critical, verify it's actually time-sensitive
	if !userMarkedCritical {
		return false
	}

	// Check for indicators of truly time-sensitive requests
	promptLower := strings.ToLower(prompt)

	timeSensitivePatterns := []string{
		"quick",
		"urgent",
		"immediately",
		"right now",
		"asap",
		"real-time",
		"live",
		"streaming",
	}

	for _, pattern := range timeSensitivePatterns {
		if strings.Contains(promptLower, pattern) {
			return true // User's critical flag is justified
		}
	}

	// Check prompt length - long prompts unlikely to be truly critical
	if len(prompt) > 500 {
		return false // Long prompts aren't time-critical
	}

	// Check for background task patterns
	backgroundPatterns := []string{
		"summarize this document",
		"analyze this log",
		"process this file",
		"batch",
		"overnight",
	}

	for _, pattern := range backgroundPatterns {
		if strings.Contains(promptLower, pattern) {
			return false // Background tasks not critical
		}
	}

	// Default: trust user but log for monitoring
	return true
}

// EstimateTokenCount estimates output tokens (for quota management)
func (c *Classifier) EstimateTokenCount(prompt string) int {
	// Rough heuristics
	promptLower := strings.ToLower(prompt)

	// Check for explicit length requests
	if strings.Contains(promptLower, "one word") {
		return 5
	}
	if strings.Contains(promptLower, "one sentence") {
		return 20
	}
	if strings.Contains(promptLower, "paragraph") {
		return 100
	}
	if strings.Contains(promptLower, "essay") || strings.Contains(promptLower, "article") {
		return 500
	}

	// Default based on prompt length
	// Typically output is 1-3x input length
	estimatedInputTokens := len(prompt) / 4 // Rough char->token conversion
	return estimatedInputTokens * 2
}
