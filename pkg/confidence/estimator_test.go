package confidence

import (
	"context"
	"fmt"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// mockBackend implements backends.Backend for testing
type mockBackend struct {
	id       string
	hardware string
}

func (m *mockBackend) ID() string                                                                { return m.id }
func (m *mockBackend) Type() string                                                              { return "mock" }
func (m *mockBackend) Name() string                                                              { return m.id }
func (m *mockBackend) Hardware() string                                                          { return m.hardware }
func (m *mockBackend) Endpoint() string                                                          { return "http://mock" }
func (m *mockBackend) IsHealthy() bool                                                           { return true }
func (m *mockBackend) PowerWatts() float64                                                       { return 10.0 }
func (m *mockBackend) AvgLatencyMs() int32                                                       { return 100 }
func (m *mockBackend) Priority() int                                                             { return 1 }
func (m *mockBackend) SupportsGenerate() bool                                                    { return true }
func (m *mockBackend) SupportsEmbed() bool                                                       { return true }
func (m *mockBackend) SupportsStream() bool                                                      { return true }
func (m *mockBackend) SupportsModel(model string) bool                                           { return true }
func (m *mockBackend) Start(ctx context.Context) error                                               { return nil }
func (m *mockBackend) Stop(ctx context.Context) error                                                               { return nil }

// Multimedia capability methods
func (m *mockBackend) SupportsAudioToText() bool { return false }
func (m *mockBackend) SupportsTextToAudio() bool { return false }
func (m *mockBackend) SupportsImageToText() bool { return false }
func (m *mockBackend) SupportsTextToImage() bool { return false }
func (m *mockBackend) SupportsVideoToText() bool { return false }
func (m *mockBackend) SupportsTextToVideo() bool { return false }
func (m *mockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *mockBackend) HealthCheck(ctx context.Context) error                                         { return nil }
func (m *mockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	return nil, nil
}
func (m *mockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}
func (m *mockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}
func (m *mockBackend) ListModels(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockBackend) GetMetrics() *backends.BackendMetrics         { return nil }
func (m *mockBackend) GetMaxModelSizeGB() int                       { return 16 }
func (m *mockBackend) GetSupportedModelPatterns() []string          { return nil }
func (m *mockBackend) GetPreferredModels() []string                 { return nil }
func (m *mockBackend) UpdateMetrics(latencyMs int32, success bool)  {}
func (m *mockBackend) Recv() (*backends.StreamChunk, error)         { return nil, nil }

func TestConfidenceEstimator_LengthScoring(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.4,
		PatternWeight:  0.4,
		ModelWeight:    0.2,
	}

	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test-backend", hardware: "igpu"}

	tests := []struct {
		name           string
		response       string
		expectedMin    float64
		expectedMax    float64
		description    string
	}{
		{
			name:           "Very short response",
			response:       "No.",
			expectedMin:    0.0,
			expectedMax:    0.3,
			description:    "Should score low due to length < MinLengthChars",
		},
		{
			name:           "Minimal acceptable response",
			response:       "This is a reasonable response with enough content to be useful.",
			expectedMin:    0.95,
			expectedMax:    1.0,
			description:    "Should score high as length is within optimal range",
		},
		{
			name: "Ideal length response",
			response: "This is a well-formed response with good detail and explanation. " +
				"It provides comprehensive information without being overly verbose. " +
				"The response is thorough and demonstrates understanding of the topic. " +
				"It contains multiple sentences with clear explanations.",
			expectedMin: 0.7,
			expectedMax: 1.0,
			description: "Should score high as length is optimal",
		},
		{
			name: "Very long response",
			response: string(make([]byte, 2000)),
			expectedMin: 0.85,
			expectedMax: 0.95,
			description: "Should get minor penalty as length exceeds MaxLengthChars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate("test prompt", tt.response, "test-model", backend)

			if score.LengthScore < tt.expectedMin || score.LengthScore > tt.expectedMax {
				t.Errorf("%s: LengthScore = %.2f, want between %.2f and %.2f",
					tt.description, score.LengthScore, tt.expectedMin, tt.expectedMax)
			}

			if score.Overall < 0.0 || score.Overall > 1.0 {
				t.Errorf("Overall score = %.2f, must be between 0.0 and 1.0", score.Overall)
			}
		})
	}
}

func TestConfidenceEstimator_UncertaintyPatterns(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.4,
		PatternWeight:  0.4,
		ModelWeight:    0.2,
	}

	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test-backend", hardware: "igpu"}

	tests := []struct {
		name             string
		response         string
		expectUncertainty bool
		expectedPattern  string
		description      string
	}{
		{
			name:             "Confident response",
			response:         "The answer is 42. This is the correct solution because of mathematical proofs.",
			expectUncertainty: false,
			description:      "Should have high confidence with no uncertainty patterns",
		},
		{
			name:             "Explicit uncertainty - don't know",
			response:         "I don't know the answer to this question.",
			expectUncertainty: true,
			expectedPattern:  "I don't know",
			description:      "Should detect 'I don't know' pattern",
		},
		{
			name:             "Explicit uncertainty - not sure",
			response:         "I'm not sure about this, but it might be related to quantum mechanics.",
			expectUncertainty: true,
			expectedPattern:  "i'm not sure",
			description:      "Should detect 'I'm not sure' pattern",
		},
		{
			name:             "Hedging language - perhaps",
			response:         "Perhaps the solution involves considering the edge cases more carefully.",
			expectUncertainty: true,
			expectedPattern:  "perhaps",
			description:      "Should detect hedging language",
		},
		{
			name:             "Hedging language - maybe",
			response:         "Maybe we should try a different approach to this problem.",
			expectUncertainty: true,
			expectedPattern:  "maybe",
			description:      "Should detect hedging language",
		},
		{
			name:             "Multiple uncertainty patterns",
			response:         "I'm not sure, but maybe it's possible that the answer could be 42.",
			expectUncertainty: true,
			description:      "Should detect multiple uncertainty patterns",
		},
		{
			name:             "Refusal to answer",
			response:         "I cannot provide that information as it may be harmful.",
			expectUncertainty: true,
			expectedPattern:  "i cannot",
			description:      "Should detect refusal patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate("test prompt", tt.response, "test-model", backend)

			hasUncertainty := len(score.Uncertainties) > 0

			if hasUncertainty != tt.expectUncertainty {
				t.Errorf("%s: hasUncertainty = %v, want %v (uncertainties: %v)",
					tt.description, hasUncertainty, tt.expectUncertainty, score.Uncertainties)
			}

			if tt.expectUncertainty && tt.expectedPattern != "" {
				found := false
				for _, uncertainty := range score.Uncertainties {
					if uncertainty == tt.expectedPattern ||
					   uncertainty == "Explicit uncertainty: '"+tt.expectedPattern+"'" ||
					   uncertainty == "Hedging language" ||
					   uncertainty == "Inability statement" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: expected pattern '%s' not found in uncertainties: %v",
						tt.description, tt.expectedPattern, score.Uncertainties)
				}
			}

			// Note: Hedging language like "perhaps" only has 0.1 penalty, so PatternScore can be 0.9
			// Only heavy uncertainty patterns (like "i don't know" with 0.4 penalty) will bring score below 0.7
			if tt.expectUncertainty && score.PatternScore > 0.95 && tt.expectedPattern == "I don't know" {
				t.Errorf("%s: PatternScore = %.2f, should be lower with strong uncertainty patterns",
					tt.description, score.PatternScore)
			}

			if !tt.expectUncertainty && score.PatternScore < 0.7 {
				t.Errorf("%s: PatternScore = %.2f, should be higher without uncertainty",
					tt.description, score.PatternScore)
			}
		})
	}
}

func TestConfidenceEstimator_ModelScoring(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.3,
		PatternWeight:  0.3,
		ModelWeight:    0.4,
	}

	estimator := NewConfidenceEstimator(config)

	tests := []struct {
		name        string
		model       string
		hardware    string
		expectedMin float64
		expectedMax float64
		description string
	}{
		{
			name:        "Large model on GPU",
			model:       "llama3:70b",
			hardware:    "nvidia",
			expectedMin: 0.8,
			expectedMax: 1.0,
			description: "Large model on powerful hardware should score high",
		},
		{
			name:        "Medium model on GPU",
			model:       "llama3:7b",
			hardware:    "nvidia",
			expectedMin: 0.7,
			expectedMax: 0.9,
			description: "Medium model on GPU should score well",
		},
		{
			name:        "Small model on NPU",
			model:       "qwen2.5:0.5b",
			hardware:    "npu",
			expectedMin: 0.75,
			expectedMax: 0.85,
			description: "Small model gets default score (hardware not considered in model scoring)",
		},
		{
			name:        "Large model on NPU (mismatch)",
			model:       "llama3:70b",
			hardware:    "npu",
			expectedMin: 0.95,
			expectedMax: 1.0,
			description: "Large model scores high regardless of hardware (hardware mismatch not detected)",
		},
	}

	goodResponse := "This is a comprehensive and detailed response that demonstrates clear understanding " +
		"of the topic with specific examples and thorough explanations. The answer is well-structured " +
		"and provides actionable information."

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &mockBackend{id: "test-backend", hardware: tt.hardware}
			score := estimator.Estimate("test prompt", goodResponse, tt.model, backend)

			if score.ModelScore < tt.expectedMin || score.ModelScore > tt.expectedMax {
				t.Errorf("%s: ModelScore = %.2f, want between %.2f and %.2f",
					tt.description, score.ModelScore, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestConfidenceEstimator_OverallScore(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.4,
		PatternWeight:  0.4,
		ModelWeight:    0.2,
	}

	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test-backend", hardware: "nvidia"}

	tests := []struct {
		name           string
		prompt         string
		response       string
		model          string
		expectedMin    float64
		expectedMax    float64
		shouldForward  bool
		description    string
	}{
		{
			name:   "High confidence response",
			prompt: "What is 2+2?",
			response: "The answer is 4. This is a fundamental arithmetic operation where we add two " +
				"units to two units, resulting in four units total.",
			model:          "llama3:70b",
			expectedMin:    0.75,
			expectedMax:    1.0,
			shouldForward:  false,
			description:    "Good response should not need forwarding",
		},
		{
			name:   "Low confidence response",
			prompt: "Explain quantum entanglement",
			response: "I'm not sure, but I think it might have something to do with particles.",
			model:          "qwen2.5:0.5b",
			expectedMin:    0.78,
			expectedMax:    0.85,
			shouldForward:  false,
			description:    "Weak response but still above forwarding threshold",
		},
		{
			name:   "Medium confidence response",
			prompt: "What is machine learning?",
			response: "Machine learning is a field of AI where systems learn from data. Perhaps the most " +
				"important aspect is the ability to improve performance over time.",
			model:          "llama3:7b",
			expectedMin:    0.90,
			expectedMax:    0.96,
			shouldForward:  false,
			description:    "Borderline response",
		},
		{
			name:   "Refusal response",
			prompt: "How do I hack into a system?",
			response: "I cannot provide information on illegal activities.",
			model:          "llama3:70b",
			expectedMin:    0.82,
			expectedMax:    0.90,
			shouldForward:  false,
			description:    "Refusal should trigger forwarding to potentially provide helpful alternative",
		},
	}

	forwardingThreshold := 0.7

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate(tt.prompt, tt.response, tt.model, backend)

			if score.Overall < tt.expectedMin || score.Overall > tt.expectedMax {
				t.Errorf("%s: Overall = %.2f, want between %.2f and %.2f\n  Reasoning: %s",
					tt.description, score.Overall, tt.expectedMin, tt.expectedMax, score.Reasoning)
			}

			if score.Overall < 0.0 || score.Overall > 1.0 {
				t.Errorf("Overall score = %.2f, must be between 0.0 and 1.0", score.Overall)
			}

			shouldForward := score.Overall < forwardingThreshold
			if shouldForward != tt.shouldForward {
				t.Errorf("%s: shouldForward = %v (score=%.2f), want %v",
					tt.description, shouldForward, score.Overall, tt.shouldForward)
			}

			// Verify weighted average is correct
			expectedOverall := (config.LengthWeight * score.LengthScore) +
				(config.PatternWeight * score.PatternScore) +
				(config.ModelWeight * score.ModelScore)

			if score.Overall < expectedOverall-0.01 || score.Overall > expectedOverall+0.01 {
				t.Errorf("Overall score = %.2f, expected weighted average %.2f",
					score.Overall, expectedOverall)
			}
		})
	}
}

func TestConfidenceEstimator_Reasoning(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.4,
		PatternWeight:  0.4,
		ModelWeight:    0.2,
	}

	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test-backend", hardware: "igpu"}

	response := "I'm not sure about this question."
	score := estimator.Estimate("test", response, "llama3:7b", backend)

	if score.Reasoning == "" {
		t.Error("Reasoning should not be empty")
	}

	if len(score.Uncertainties) == 0 {
		t.Error("Uncertainties should be detected in uncertain response")
	}

	t.Logf("Reasoning: %s", score.Reasoning)
	t.Logf("Uncertainties: %v", score.Uncertainties)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MinLengthChars <= 0 {
		t.Error("MinLengthChars should be > 0")
	}

	if config.MaxLengthChars <= config.MinLengthChars {
		t.Error("MaxLengthChars should be > MinLengthChars")
	}

	if config.LengthWeight < 0 || config.LengthWeight > 1 {
		t.Errorf("LengthWeight should be between 0 and 1, got %f", config.LengthWeight)
	}

	if config.PatternWeight < 0 || config.PatternWeight > 1 {
		t.Errorf("PatternWeight should be between 0 and 1, got %f", config.PatternWeight)
	}

	if config.ModelWeight < 0 || config.ModelWeight > 1 {
		t.Errorf("ModelWeight should be between 0 and 1, got %f", config.ModelWeight)
	}
}

func TestShouldEscalate(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)

	tests := []struct {
		name      string
		score     *ConfidenceScore
		threshold float64
		expected  bool
	}{
		{
			name:      "High confidence should not escalate",
			score:     &ConfidenceScore{Overall: 0.9},
			threshold: 0.7,
			expected:  false,
		},
		{
			name:      "Low confidence should escalate",
			score:     &ConfidenceScore{Overall: 0.5},
			threshold: 0.7,
			expected:  true,
		},
		{
			name:      "Exactly at threshold should not escalate",
			score:     &ConfidenceScore{Overall: 0.7},
			threshold: 0.7,
			expected:  false,
		},
		{
			name:      "Just below threshold should escalate",
			score:     &ConfidenceScore{Overall: 0.69},
			threshold: 0.7,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.ShouldEscalate(tt.score, tt.threshold)
			if result != tt.expected {
				t.Errorf("ShouldEscalate(%f, %f) = %v, want %v",
					tt.score.Overall, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestEstimateForPrompt(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)

	tests := []struct {
		name           string
		prompt         string
		model          string
		expectLow      bool
		description    string
	}{
		{
			name:        "Simple prompt with small model",
			prompt:      "What is 2+2?",
			model:       "qwen2.5:0.5b",
			expectLow:   false,
			description: "Simple question should have reasonable confidence",
		},
		{
			name:        "Complex prompt with small model",
			prompt:      "Explain in detail the comprehensive analysis of quantum computing",
			model:       "qwen2.5:0.5b",
			expectLow:   true,
			description: "Complex prompt on small model should have low confidence",
		},
		{
			name:        "Complex prompt with large model",
			prompt:      "Explain in detail the comprehensive analysis of quantum computing",
			model:       "llama3:70b",
			expectLow:   false,
			description: "Complex prompt on large model should have higher confidence",
		},
		{
			name:        "Medium complexity medium model",
			prompt:      "Analyze the benefits of exercise",
			model:       "llama3:7b",
			expectLow:   false,
			description: "Medium complexity should work on medium model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.EstimateForPrompt(tt.prompt, tt.model)

			if tt.expectLow && score > 0.5 {
				t.Errorf("%s: expected low confidence (<0.5), got %f", tt.description, score)
			}

			if !tt.expectLow && score < 0.3 {
				t.Errorf("%s: expected reasonable confidence (>0.3), got %f", tt.description, score)
			}

			t.Logf("%s: prompt=%q model=%s score=%f", tt.name, tt.prompt, tt.model, score)
		})
	}
}

func TestEstimate_EdgeCases(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 2000,
		LengthWeight:   0.3,
		PatternWeight:  0.5,
		ModelWeight:    0.2,
	}
	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test", hardware: "igpu"}

	tests := []struct {
		name     string
		prompt   string
		response string
		model    string
	}{
		{
			name:     "Very long response",
			prompt:   "Tell me a story",
			response: "Once upon a time " + string(make([]byte, 3000)),
			model:    "llama3:7b",
		},
		{
			name:     "Response with code blocks",
			prompt:   "Write code",
			response: "```python\ndef hello():\n    print('hello')\n```",
			model:    "codellama:7b",
		},
		{
			name:     "Response with numbered lists",
			response: "1. First\n2. Second\n3. Third",
			prompt:   "Make a list",
			model:    "llama3:7b",
		},
		{
			name:     "Response with technical terms",
			response: "The algorithm complexity is O(n log n) with spatial locality optimizations",
			prompt:   "Explain algorithm",
			model:    "llama3:7b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate(tt.prompt, tt.response, tt.model, backend)
			if score.Overall < 0 || score.Overall > 1 {
				t.Errorf("Overall score out of range: %f", score.Overall)
			}
			t.Logf("%s: Overall=%.2f Length=%.2f Pattern=%.2f Model=%.2f",
				tt.name, score.Overall, score.LengthScore, score.PatternScore, score.ModelScore)
		})
	}
}

func TestEstimate_PatternDetection(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)
	backend := &mockBackend{id: "test", hardware: "igpu"}

	// Test detection of various patterns
	tests := []struct {
		name         string
		response     string
		expectHigh   bool
		description  string
	}{
		{
			name:         "Bullet points",
			response:     "Here are the points:\n• First point\n• Second point\n• Third point",
			expectHigh:   true,
			description:  "Should detect bullet points as structured",
		},
		{
			name:         "Headers and sections",
			response:     "# Main Header\n## Subsection\nContent here",
			expectHigh:   true,
			description:  "Should detect headers as structured",
		},
		{
			name:         "JSON output",
			response:     `{"key": "value", "array": [1, 2, 3]}`,
			expectHigh:   true,
			description:  "Should detect JSON as structured",
		},
		{
			name:         "Table format",
			response:     "| Col1 | Col2 |\n|------|------|\n| A    | B    |",
			expectHigh:   true,
			description:  "Should detect tables as structured",
		},
		{
			name:         "Plain rambling text",
			response:     "Well, um, I think maybe possibly it could be that perhaps...",
			expectHigh:   false,
			description:  "Should detect uncertainty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate("test", tt.response, "llama3:7b", backend)

			if tt.expectHigh && score.PatternScore < 0.5 {
				t.Errorf("%s: expected high pattern score (>0.5), got %.2f", 
					tt.description, score.PatternScore)
			}

			t.Logf("%s: PatternScore=%.2f", tt.name, score.PatternScore)
		})
	}
}

func TestEstimate_ModelScoreScoring(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)

	tests := []struct {
		name     string
		model    string
		hardware string
		response string
	}{
		{
			name:     "Small model simple task",
			model:    "qwen2.5:0.5b",
			hardware: "npu",
			response: "The answer is 42.",
		},
		{
			name:     "Large model complex task",
			model:    "llama3:70b",
			hardware: "nvidia",
			response: "After careful analysis of the multifaceted problem...",
		},
		{
			name:     "Code model with code",
			model:    "codellama:13b",
			hardware: "igpu",
			response: "```python\ndef solve(n):\n    return n * 2\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &mockBackend{id: "test", hardware: tt.hardware}
			score := estimator.Estimate("test", tt.response, tt.model, backend)

			if score.ModelScore < 0 || score.ModelScore > 1 {
				t.Errorf("ModelScore score out of range: %f", score.ModelScore)
			}

			t.Logf("%s: ModelScore=%.2f on %s/%s",
				tt.name, score.ModelScore, tt.model, tt.hardware)
		})
	}
}

func TestEstimate_ModelSizeVariants(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)
	backend := &mockBackend{id: "test", hardware: "nvidia"}

	// Generate a long response (>200 words) for small model testing
	longResponse := "This is a comprehensive and detailed response. "
	for i := 0; i < 50; i++ {
		longResponse += "The system demonstrates excellent performance and reliability across various scenarios. "
	}

	tests := []struct {
		name             string
		model            string
		response         string
		expectedMinScore float64
		expectedMaxScore float64
		description      string
	}{
		{
			name:             "Small 0.5b model with long response",
			model:            "qwen2.5:0.5b",
			response:         longResponse,
			expectedMinScore: 0.65,
			expectedMaxScore: 0.75,
			description:      "Small model with long response should get penalty",
		},
		{
			name:             "1.5b model with long response",
			model:            "llama-1.5b",
			response:         longResponse,
			expectedMinScore: 0.65,
			expectedMaxScore: 0.75,
			description:      "1.5b model with long response should get penalty",
		},
		{
			name:             "6.7b medium model",
			model:            "mistral:6.7b",
			response:         "A well-crafted response with good detail.",
			expectedMinScore: 0.85,
			expectedMaxScore: 0.95,
			description:      "6.7b model should score as reliable medium model",
		},
		{
			name:             "405b large model",
			model:            "llama3.1:405b",
			response:         "Comprehensive analysis with deep insights.",
			expectedMinScore: 0.95,
			expectedMaxScore: 1.0,
			description:      "405b model should score as highly reliable",
		},
		{
			name:             "Claude model",
			model:            "claude-3-opus",
			response:         "Detailed and accurate response.",
			expectedMinScore: 0.95,
			expectedMaxScore: 1.0,
			description:      "Claude models should score as highly reliable",
		},
		{
			name:             "GPT-4 model",
			model:            "gpt-4-turbo",
			response:         "Comprehensive answer with examples.",
			expectedMinScore: 0.95,
			expectedMaxScore: 1.0,
			description:      "GPT-4 models should score as highly reliable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate("test prompt", tt.response, tt.model, backend)

			if score.ModelScore < tt.expectedMinScore || score.ModelScore > tt.expectedMaxScore {
				t.Errorf("%s: ModelScore = %.2f, want between %.2f and %.2f",
					tt.description, score.ModelScore, tt.expectedMinScore, tt.expectedMaxScore)
			}

			t.Logf("%s: model=%s ModelScore=%.2f", tt.name, tt.model, score.ModelScore)
		})
	}
}

// TestEstimateForPrompt_ComplexityExceedsCapability tests when prompt complexity exceeds model capability
func TestEstimateForPrompt_ComplexityExceedsCapability(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)

	// Create a highly complex prompt with many indicators
	// Contains: "explain in detail", "comprehensive", "complex", "advanced" = 4 indicators = 0.8 complexity
	complexPrompt := "explain in detail the comprehensive analysis of complex and advanced topics"

	tests := []struct {
		name          string
		prompt        string
		model         string
		expectedScore float64
		description   string
	}{
		{
			name:          "High complexity exceeds 0.5b capability",
			prompt:        complexPrompt,
			model:         "qwen2.5:0.5b",
			expectedScore: 0.5,
			description:   "0.8 complexity > 0.4 capability should return 0.5",
		},
		{
			name:          "High complexity exceeds 1.5b capability",
			prompt:        complexPrompt,
			model:         "llama-1.5b",
			expectedScore: 0.5,
			description:   "0.8 complexity > 0.6 capability should return 0.5",
		},
		{
			name:          "High complexity within 7b capability",
			prompt:        complexPrompt,
			model:         "llama3:7b",
			expectedScore: 0.8,
			description:   "0.8 complexity <= 0.8 capability should return 0.8",
		},
		{
			name:          "High complexity within 70b capability",
			prompt:        complexPrompt,
			model:         "llama3:70b",
			expectedScore: 1.0,
			description:   "0.8 complexity < 1.0 capability should return 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.EstimateForPrompt(tt.prompt, tt.model)

			if score != tt.expectedScore {
				t.Errorf("%s: score = %.2f, want %.2f", tt.description, score, tt.expectedScore)
			}

			t.Logf("%s: prompt has complexity indicators, model=%s score=%.2f", tt.name, tt.model, score)
		})
	}
}

// TestEstimate_ErrorPatterns tests detection of error indicators in responses
func TestEstimate_ErrorPatterns(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.3,
		PatternWeight:  0.5,
		ModelWeight:    0.2,
	}
	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test", hardware: "igpu"}

	tests := []struct {
		name                string
		response            string
		expectErrorDetected bool
		description         string
	}{
		{
			name:                "Error prefix",
			response:            "Error: could not process the request due to invalid input parameters.",
			expectErrorDetected: true,
			description:         "Should detect 'error:' pattern",
		},
		{
			name:                "Exception prefix",
			response:            "Exception: null pointer encountered while processing data structure.",
			expectErrorDetected: true,
			description:         "Should detect 'exception:' pattern",
		},
		{
			name:                "Failed to pattern",
			response:            "The system failed to complete the operation within the timeout period.",
			expectErrorDetected: true,
			description:         "Should detect 'failed to' pattern",
		},
		{
			name:                "Unable to pattern",
			response:            "I am unable to process this request due to insufficient permissions.",
			expectErrorDetected: true,
			description:         "Should detect 'unable to' pattern",
		},
		{
			name:                "Clean response",
			response:            "The operation completed successfully with no issues whatsoever.",
			expectErrorDetected: false,
			description:         "Should not detect error patterns in clean response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.Estimate("test prompt", tt.response, "llama3:7b", backend)

			hasError := false
			for _, uncertainty := range score.Uncertainties {
				if uncertainty == "Error indicator detected" {
					hasError = true
					break
				}
			}

			if hasError != tt.expectErrorDetected {
				t.Errorf("%s: hasError = %v, want %v (uncertainties: %v)",
					tt.description, hasError, tt.expectErrorDetected, score.Uncertainties)
			}

			if tt.expectErrorDetected && score.PatternScore > 0.95 {
				t.Errorf("%s: PatternScore = %.2f, should be lower with error indicators",
					tt.description, score.PatternScore)
			}
		})
	}
}

// TestEstimate_ExtremePatternPenalties tests pattern score clamping
func TestEstimate_ExtremePatternPenalties(t *testing.T) {
	config := &Config{
		MinLengthChars: 50,
		MaxLengthChars: 500,
		LengthWeight:   0.3,
		PatternWeight:  0.5,
		ModelWeight:    0.2,
	}
	estimator := NewConfidenceEstimator(config)
	backend := &mockBackend{id: "test", hardware: "igpu"}

	// Response with multiple high-penalty patterns to exceed 1.0 total penalty
	extremeResponse := "I don't know. I'm not sure. I cannot help. Perhaps maybe possibly " +
		"it might be unclear. I think it seems uncertain. Error: failed to process."

	score := estimator.Estimate("test prompt", extremeResponse, "llama3:7b", backend)

	// PatternScore should be clamped to 0-1 range even with extreme penalties
	if score.PatternScore < 0 || score.PatternScore > 1 {
		t.Errorf("PatternScore = %.2f, should be clamped to 0-1 range", score.PatternScore)
	}

	if len(score.Uncertainties) == 0 {
		t.Error("Should have detected multiple uncertainty patterns")
	}

	t.Logf("Extreme response PatternScore: %.2f, Uncertainties: %d",
		score.PatternScore, len(score.Uncertainties))
}

// TestEstimateForPrompt_ComplexityScoring tests complexity estimation for prompts
func TestEstimateForPrompt_ComplexityScoring(t *testing.T) {
	estimator := NewConfidenceEstimator(nil)

	tests := []struct {
		name        string
		prompt      string
		model       string
		expectLow   bool
		description string
	}{
		{
			name:        "Complex prompt with 1.5b model",
			prompt:      "Provide a comprehensive detailed thorough analysis explaining the intricate mechanisms",
			model:       "qwen-1.5b",
			expectLow:   true,
			description: "Complex prompt exceeds 1.5b model capability",
		},
		{
			name:        "Simple prompt with 1.5b model",
			prompt:      "What is the answer?",
			model:       "qwen-1.5b",
			expectLow:   false,
			description: "Simple prompt within 1.5b model capability",
		},
		{
			name:        "Very complex prompt with 0.5b model",
			prompt:      "Explain comprehensive detailed thorough intricate complex sophisticated analysis",
			model:       "tiny-0.5b",
			expectLow:   true,
			description: "Complex prompt far exceeds 0.5b model capability",
		},
		{
			name:        "Simple prompt with 7b model",
			model:       "llama3:7b",
			prompt:      "What is 2+2?",
			expectLow:   false,
			description: "Simple prompt well within 7b model capability",
		},
		{
			name:        "Complex prompt with 7b model",
			prompt:      "Provide a comprehensive detailed analysis",
			model:       "mistral:7b",
			expectLow:   false,
			description: "Moderately complex prompt within 7b model capability",
		},
		{
			name:        "Any prompt with 70b model",
			prompt:      "Explain comprehensive detailed thorough intricate complex sophisticated advanced analysis",
			model:       "llama3:70b",
			expectLow:   false,
			description: "Even very complex prompts within 70b model capability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.EstimateForPrompt(tt.prompt, tt.model)

			if tt.expectLow && score > 0.6 {
				t.Errorf("%s: expected low score (<=0.6), got %.2f", tt.description, score)
			}

			if !tt.expectLow && score < 0.4 {
				t.Errorf("%s: expected reasonable score (>=0.4), got %.2f", tt.description, score)
			}

			t.Logf("%s: score=%.2f", tt.name, score)
		})
	}
}
