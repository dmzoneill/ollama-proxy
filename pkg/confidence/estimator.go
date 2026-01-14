package confidence

import (
	"regexp"
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// ConfidenceEstimator estimates the confidence/quality of LLM outputs
type ConfidenceEstimator struct {
	config *Config
}

// Config for confidence estimation
type Config struct {
	// Length-based scoring
	MinLengthChars int     // Minimum expected response length
	MaxLengthChars int     // Maximum expected response length
	LengthWeight   float64 // Weight of length score (0-1)

	// Pattern-based scoring
	PatternWeight float64 // Weight of pattern score (0-1)

	// Model-specific scoring
	ModelWeight float64 // Weight of model-specific heuristics (0-1)
}

// DefaultConfig returns default confidence estimation config
func DefaultConfig() *Config {
	return &Config{
		MinLengthChars: 50,
		MaxLengthChars: 2000,
		LengthWeight:   0.3,
		PatternWeight:  0.5,
		ModelWeight:    0.2,
	}
}

// NewConfidenceEstimator creates a new confidence estimator
func NewConfidenceEstimator(config *Config) *ConfidenceEstimator {
	if config == nil {
		config = DefaultConfig()
	}
	return &ConfidenceEstimator{config: config}
}

// ConfidenceScore represents the confidence analysis of a response
type ConfidenceScore struct {
	Overall       float64 // Overall confidence (0-1)
	LengthScore   float64 // Score based on response length
	PatternScore  float64 // Score based on pattern matching
	ModelScore    float64 // Score based on model-specific heuristics
	Reasoning     string  // Explanation of the score
	Uncertainties []string // Detected uncertainty indicators
}

// Estimate calculates confidence score for a response
func (ce *ConfidenceEstimator) Estimate(
	prompt string,
	response string,
	model string,
	backend backends.Backend,
) *ConfidenceScore {
	score := &ConfidenceScore{
		Uncertainties: make([]string, 0),
	}

	// 1. Length-based scoring
	score.LengthScore = ce.scoreLengthQuality(response)

	// 2. Pattern-based scoring (detect uncertainty)
	score.PatternScore = ce.scorePatterns(response, score)

	// 3. Model-specific scoring
	score.ModelScore = ce.scoreModelSpecific(model, backend, response)

	// Calculate overall score (weighted average)
	score.Overall = (ce.config.LengthWeight * score.LengthScore) +
		(ce.config.PatternWeight * score.PatternScore) +
		(ce.config.ModelWeight * score.ModelScore)

	// Generate reasoning
	score.Reasoning = ce.generateReasoning(score)

	return score
}

// scoreLengthQuality scores based on response length
func (ce *ConfidenceEstimator) scoreLengthQuality(response string) float64 {
	length := len(strings.TrimSpace(response))

	// Very short responses are likely incomplete or low quality
	if length < 20 {
		return 0.1
	}

	// Score based on whether length is in expected range
	if length >= ce.config.MinLengthChars && length <= ce.config.MaxLengthChars {
		return 1.0 // Perfect length
	}

	if length < ce.config.MinLengthChars {
		// Too short - score proportionally
		ratio := float64(length) / float64(ce.config.MinLengthChars)
		return 0.5 + (ratio * 0.5) // Range: 0.5-1.0
	}

	// Too long - minor penalty (not necessarily bad)
	return 0.9
}

// scorePatterns detects uncertainty patterns in the response
func (ce *ConfidenceEstimator) scorePatterns(response string, score *ConfidenceScore) float64 {
	responseLower := strings.ToLower(response)

	// Uncertainty indicators (negative patterns)
	uncertaintyPatterns := []struct {
		pattern string
		penalty float64
		reason  string
	}{
		{"i don't know", 0.4, "Explicit uncertainty: 'I don't know'"},
		{"i'm not sure", 0.3, "Explicit uncertainty: 'I'm not sure'"},
		{"i cannot", 0.3, "Inability statement"},
		{"i can't", 0.3, "Inability statement"},
		{"unclear", 0.2, "Acknowledges lack of clarity"},
		{"uncertain", 0.2, "Explicit uncertainty"},
		{"perhaps", 0.1, "Hedging language"},
		{"maybe", 0.1, "Hedging language"},
		{"possibly", 0.1, "Hedging language"},
		{"might be", 0.1, "Hedging language"},
		{"could be", 0.1, "Hedging language"},
		{"i think", 0.05, "Weak assertion"},
		{"it seems", 0.05, "Weak assertion"},
	}

	totalPenalty := 0.0
	for _, pattern := range uncertaintyPatterns {
		if strings.Contains(responseLower, pattern.pattern) {
			totalPenalty += pattern.penalty
			score.Uncertainties = append(score.Uncertainties, pattern.reason)
		}
	}

	// Incomplete response indicators
	incompletePatterns := []string{
		"...",           // Trailing ellipsis
		"[incomplete]",
		"[truncated]",
	}

	for _, pattern := range incompletePatterns {
		if strings.Contains(responseLower, pattern) {
			totalPenalty += 0.3
			score.Uncertainties = append(score.Uncertainties, "Incomplete response detected")
			break
		}
	}

	// Error indicators
	errorPatterns := []string{
		"error:",
		"exception:",
		"failed to",
		"unable to",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(responseLower, pattern) {
			totalPenalty += 0.5
			score.Uncertainties = append(score.Uncertainties, "Error indicator detected")
			break
		}
	}

	// Positive patterns (confidence indicators)
	bonus := 0.0

	// Structured response (lists, numbered items)
	if hasStructuredContent(response) {
		bonus += 0.1
	}

	// Technical terms (suggests depth)
	if hasTechnicalContent(response) {
		bonus += 0.1
	}

	// Calculate final pattern score
	patternScore := 1.0 - totalPenalty + bonus

	// Clamp to 0-1 range
	if patternScore < 0 {
		patternScore = 0
	}
	if patternScore > 1 {
		patternScore = 1
	}

	return patternScore
}

// scoreModelSpecific applies model-specific heuristics
func (ce *ConfidenceEstimator) scoreModelSpecific(model string, backend backends.Backend, response string) float64 {
	// Small models (0.5b-1.5b) get penalty for complex responses
	if strings.Contains(model, "0.5b") || strings.Contains(model, "1.5b") {
		wordCount := len(strings.Fields(response))
		if wordCount > 200 {
			// Small model generating long response - might be repetitive
			return 0.7
		}
	}

	// Medium models (7b) are reliable for most tasks
	if strings.Contains(model, "7b") || strings.Contains(model, "6.7b") {
		return 0.9
	}

	// Large models (70b+) are highly reliable
	if strings.Contains(model, "70b") || strings.Contains(model, "405b") {
		return 1.0
	}

	// Cloud models (Claude, GPT-4) are highly reliable
	if strings.HasPrefix(model, "claude-") || strings.HasPrefix(model, "gpt-4") {
		return 1.0
	}

	// Default for unknown models
	return 0.8
}

// generateReasoning creates human-readable explanation
func (ce *ConfidenceEstimator) generateReasoning(score *ConfidenceScore) string {
	var parts []string

	// Overall assessment
	if score.Overall >= 0.9 {
		parts = append(parts, "High confidence")
	} else if score.Overall >= 0.7 {
		parts = append(parts, "Medium confidence")
	} else {
		parts = append(parts, "Low confidence")
	}

	// Length assessment
	if score.LengthScore < 0.6 {
		parts = append(parts, "response too short")
	} else if score.LengthScore >= 0.9 {
		parts = append(parts, "good length")
	}

	// Pattern assessment
	if len(score.Uncertainties) > 0 {
		parts = append(parts, "detected uncertainty indicators")
	}

	return strings.Join(parts, ", ")
}

// hasStructuredContent checks for structured formatting
func hasStructuredContent(response string) bool {
	// Check for numbered lists
	numberedList := regexp.MustCompile(`(?m)^\s*\d+\.\s`)
	if numberedList.MatchString(response) {
		return true
	}

	// Check for bullet points
	bulletList := regexp.MustCompile(`(?m)^\s*[-*•]\s`)
	if bulletList.MatchString(response) {
		return true
	}

	// Check for markdown headers
	headers := regexp.MustCompile(`(?m)^#+\s`)
	if headers.MatchString(response) {
		return true
	}

	return false
}

// hasTechnicalContent checks for technical depth
func hasTechnicalContent(response string) bool {
	// Simple heuristic: check for code blocks or technical terms
	codeBlock := regexp.MustCompile("```")
	if codeBlock.MatchString(response) {
		return true
	}

	// Check for common technical keywords
	technicalTerms := []string{
		"function", "class", "method", "algorithm", "implementation",
		"variable", "parameter", "return", "iterate", "recursive",
		"complexity", "optimize", "efficient", "architecture",
	}

	responseLower := strings.ToLower(response)
	matchCount := 0
	for _, term := range technicalTerms {
		if strings.Contains(responseLower, term) {
			matchCount++
		}
	}

	// If multiple technical terms present, likely technical content
	return matchCount >= 3
}

// ShouldEscalate determines if response quality warrants escalation
func (ce *ConfidenceEstimator) ShouldEscalate(score *ConfidenceScore, threshold float64) bool {
	return score.Overall < threshold
}

// EstimateForPrompt estimates expected quality for a prompt (before generation)
func (ce *ConfidenceEstimator) EstimateForPrompt(prompt string, model string) float64 {
	// Simple heuristic: estimate if prompt is likely too complex for model
	promptLower := strings.ToLower(prompt)

	// Complexity indicators in prompt
	complexIndicators := []string{
		"explain in detail",
		"comprehensive",
		"analyze",
		"compare and contrast",
		"evaluate",
		"in-depth",
		"complex",
		"advanced",
	}

	complexityScore := 0.0
	for _, indicator := range complexIndicators {
		if strings.Contains(promptLower, indicator) {
			complexityScore += 0.2
		}
	}

	// Check if model can handle complexity
	modelCapability := 0.8 // Default

	if strings.Contains(model, "0.5b") {
		modelCapability = 0.4
	} else if strings.Contains(model, "1.5b") {
		modelCapability = 0.6
	} else if strings.Contains(model, "7b") {
		modelCapability = 0.8
	} else if strings.Contains(model, "70b") {
		modelCapability = 1.0
	}

	// If complexity exceeds model capability, low confidence
	if complexityScore > modelCapability {
		return 0.5
	}

	return modelCapability
}
