package classifier

import (
	"context"
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// RequestComplexity indicates how complex a request is
type RequestComplexity int

const (
	ComplexitySimple RequestComplexity = iota // Simple queries, short answers
	ComplexityModerate                        // Normal chat, explanations
	ComplexityComplex                         // Long-form generation, analysis
)

// Classifier analyzes requests to determine appropriate routing
type Classifier struct {
	npu backends.Backend // Use NPU for classification
}

func NewClassifier(npu backends.Backend) *Classifier {
	return &Classifier{npu: npu}
}

// ClassifyPrompt determines request complexity
func (c *Classifier) ClassifyPrompt(ctx context.Context, prompt string, model string) RequestComplexity {
	// Heuristic-based classification (fast, no inference needed)

	// 1. Check prompt length
	if len(prompt) < 50 {
		return ComplexitySimple // Short prompts usually simple
	}

	// 2. Check for simple patterns
	promptLower := strings.ToLower(prompt)

	// Simple question patterns
	simplePatterns := []string{
		"what is",
		"who is",
		"when was",
		"where is",
		"how old",
		"how many",
		"yes or no",
		"true or false",
	}

	for _, pattern := range simplePatterns {
		if strings.HasPrefix(promptLower, pattern) {
			return ComplexitySimple
		}
	}

	// 3. Check for complex patterns
	complexPatterns := []string{
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

	for _, pattern := range complexPatterns {
		if strings.Contains(promptLower, pattern) {
			return ComplexityComplex
		}
	}

	// 4. Check expected output length indicators
	if strings.Contains(promptLower, "briefly") ||
	   strings.Contains(promptLower, "one sentence") ||
	   strings.Contains(promptLower, "in short") {
		return ComplexitySimple
	}

	// 5. Check for multi-step reasoning
	if strings.Contains(promptLower, "step by step") ||
	   strings.Contains(promptLower, "first,") ||
	   strings.Count(promptLower, "?") > 1 {
		return ComplexityModerate
	}

	// 6. Model size influences complexity handling
	// Smaller models can run on NPU/iGPU, larger need NVIDIA
	if strings.Contains(model, "0.5b") || strings.Contains(model, "1.5b") {
		return ComplexitySimple // Small models efficient on NPU
	}

	if strings.Contains(model, "70b") || strings.Contains(model, "33b") {
		return ComplexityComplex // Large models need NVIDIA
	}

	// Default to moderate
	return ComplexityModerate
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
		Model:  "qwen2.5:0.5b", // Use smallest model for classification
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
		if queueDepth["ollama-npu"] < 3 {
			return "ollama-npu"
		}
		return "ollama-igpu" // Fallback to Intel GPU if NPU busy

	case ComplexityModerate:
		// Moderate requests: prefer Intel GPU (balanced)
		if onBattery {
			// On battery, prefer efficient backends
			if queueDepth["ollama-igpu"] < 2 {
				return "ollama-igpu"
			}
			return "ollama-npu"
		} else {
			// On AC power, can use NVIDIA if Intel busy
			if queueDepth["ollama-igpu"] < 2 {
				return "ollama-igpu"
			}
			return "ollama-nvidia"
		}

	case ComplexityComplex:
		// Complex requests: prefer NVIDIA (performance)
		if onBattery && queueDepth["ollama-nvidia"] > 1 {
			// On battery with NVIDIA busy, downgrade to Intel
			return "ollama-igpu"
		}
		return "ollama-nvidia"
	}

	return "ollama-igpu" // Default
}

// ShouldAllowLatencyCritical checks if user's latency_critical flag is justified
func (c *Classifier) ShouldAllowLatencyCritical(ctx context.Context, prompt string, userMarkedCritical bool) bool {
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
