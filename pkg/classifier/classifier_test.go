package classifier

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// TestNewClassifier is part of the public API.
func TestNewClassifier(t *testing.T) {
	classifier := NewClassifier(nil)
	if classifier == nil {
		t.Fatal("NewClassifier returned nil")
	}
}

// TestClassifyPrompt_Simple is part of the public API.
func TestClassifyPrompt_Simple(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{name: "Short prompt", prompt: "Hi", model: ""},
		{name: "What is question", prompt: "What is Go?", model: ""},
		{name: "Who is question", prompt: "Who is Alan Turing?", model: ""},
		{name: "When was question", prompt: "When was Python created?", model: ""},
		{name: "Where is question", prompt: "Where is New York?", model: ""},
		{name: "How old question", prompt: "How old is the internet?", model: ""},
		{name: "How many question", prompt: "How many days in a year?", model: ""},
		{name: "Yes or no", prompt: "Is Go compiled? Yes or no", model: ""},
		{name: "True or false", prompt: "Is Python interpreted? True or false", model: ""},
		{name: "Briefly", prompt: "Explain quantum physics briefly", model: ""},
		{name: "One sentence", prompt: "Summarize this in one sentence", model: ""},
		{name: "In short", prompt: "Tell me in short about AI", model: ""},
		{name: "Small model", prompt: "Test prompt", model: strQwen2505b},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ClassifyPrompt(context.Background(), tt.prompt, tt.model)
			if result != ComplexitySimple {
				t.Errorf("Expected ComplexitySimple for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestClassifyPrompt_Complex is part of the public API.
func TestClassifyPrompt_Complex(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{name: "Write detailed", prompt: "Write a detailed analysis of climate change impacts on ecosystems", model: ""},
		{name: "Explain in depth", prompt: "Explain in depth how neural networks work and their applications", model: ""},
		{name: "Analyze", prompt: "Analyze this data set thoroughly and provide detailed insights", model: ""},
		{name: "Compare and contrast", prompt: "Compare and contrast Python and Go programming languages in detail", model: ""},
		{name: "Comprehensive", prompt: "Create a comprehensive guide to Rust programming for beginners", model: ""},
		{name: "Generate code", prompt: "Generate code for a REST API with authentication and database integration", model: ""},
		{name: "Write story", prompt: "Write a story about a robot discovering consciousness and emotions", model: ""},
		{name: "Write essay", prompt: "Write an essay on democracy and its evolution through history", model: ""},
		{name: "Compose", prompt: "Compose a long email to the team about the new project direction", model: ""},
		{name: "Develop plan", prompt: "Develop a plan for project migration from legacy systems to modern architecture", model: ""},
		{name: "Large model", prompt: "This is a test prompt for large language models like llama3", model: "llama3:70b"},
		{name: "Large model 2", prompt: "Another test prompt for very large models like mixtral", model: "mixtral:33b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ClassifyPrompt(context.Background(), tt.prompt, tt.model)
			if result != ComplexityComplex {
				t.Errorf("Expected ComplexityComplex for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestClassifyPrompt_Moderate is part of the public API.
func TestClassifyPrompt_Moderate(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{name: "Step by step", prompt: "Explain step by step how to install Go on different operating systems", model: ""},
		{name: "First", prompt: "First, we need to understand the basics of how this system works", model: ""},
		{name: "Multiple questions", prompt: "Can you tell me what AI is? And also how does machine learning work?", model: ""},
		{name: "Medium length", prompt: "Can you explain to me how machine learning algorithms work and what are the main types?", model: ""},
		{name: "Default", prompt: "Tell me about programming languages and their characteristics", model: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ClassifyPrompt(context.Background(), tt.prompt, tt.model)
			if result != ComplexityModerate {
				t.Errorf("Expected ComplexityModerate for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestRecommendBackend_Simple is part of the public API.
func TestRecommendBackend_Simple(t *testing.T) {
	c := NewClassifier(nil)

	// Empty queue, should prefer NPU
	result := c.RecommendBackend(ComplexitySimple, false, map[string]int{})
	if result != strOllamaNpu {
		t.Errorf("Expected ollama-npu for simple with empty queue, got %s", result)
	}

	// NPU busy, should use Intel GPU
	result = c.RecommendBackend(ComplexitySimple, false, map[string]int{strOllamaNpu: 5})
	if result != strOllamaIgpu {
		t.Errorf("Expected ollama-igpu when NPU busy, got %s", result)
	}
}

// TestRecommendBackend_Moderate is part of the public API.
func TestRecommendBackend_Moderate(t *testing.T) {
	c := NewClassifier(nil)

	// On AC power, prefer Intel GPU
	result := c.RecommendBackend(ComplexityModerate, false, map[string]int{})
	if result != strOllamaIgpu {
		t.Errorf("Expected ollama-igpu for moderate on AC, got %s", result)
	}

	// On battery, prefer efficient backends
	result = c.RecommendBackend(ComplexityModerate, true, map[string]int{})
	if result != strOllamaIgpu {
		t.Errorf("Expected ollama-igpu for moderate on battery, got %s", result)
	}

	// Intel GPU busy, use NVIDIA on AC
	result = c.RecommendBackend(ComplexityModerate, false, map[string]int{strOllamaIgpu: 5})
	if result != strOllamaNvidia {
		t.Errorf("Expected ollama-nvidia when Intel busy on AC, got %s", result)
	}

	// Intel GPU busy on battery, use NPU
	result = c.RecommendBackend(ComplexityModerate, true, map[string]int{strOllamaIgpu: 5})
	if result != strOllamaNpu {
		t.Errorf("Expected ollama-npu when Intel busy on battery, got %s", result)
	}
}

// TestRecommendBackend_Complex is part of the public API.
func TestRecommendBackend_Complex(t *testing.T) {
	c := NewClassifier(nil)

	// Prefer NVIDIA for complex
	result := c.RecommendBackend(ComplexityComplex, false, map[string]int{})
	if result != strOllamaNvidia {
		t.Errorf("Expected ollama-nvidia for complex on AC, got %s", result)
	}

	// On battery with NVIDIA busy, downgrade to Intel
	result = c.RecommendBackend(ComplexityComplex, true, map[string]int{strOllamaNvidia: 3})
	if result != strOllamaIgpu {
		t.Errorf("Expected ollama-igpu when NVIDIA busy on battery, got %s", result)
	}

	// On battery with NVIDIA not busy, still use NVIDIA
	result = c.RecommendBackend(ComplexityComplex, true, map[string]int{strOllamaNvidia: 1})
	if result != strOllamaNvidia {
		t.Errorf("Expected ollama-nvidia when not busy on battery, got %s", result)
	}
}

// TestShouldAllowLatencyCritical is part of the public API.
func TestShouldAllowLatencyCritical(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name           string
		prompt         string
		userMarked     bool
		expectedResult bool
	}{
		{name: "Not marked critical", prompt: "Test prompt", userMarked: false, expectedResult: false},
		{name: "Quick request", prompt: "Quick answer please", userMarked: true, expectedResult: true},
		{name: "Urgent request", prompt: "This is urgent", userMarked: true, expectedResult: true},
		{name: "Immediately", prompt: "I need this immediately", userMarked: true, expectedResult: true},
		{name: "Right now", prompt: "I need this right now", userMarked: true, expectedResult: true},
		{name: "ASAP", prompt: "Send this asap", userMarked: true, expectedResult: true},
		{name: "Real-time", prompt: "Real-time analysis needed", userMarked: true, expectedResult: true},
		{name: "Live", prompt: "Live updates required", userMarked: true, expectedResult: true},
		{name: "Streaming", prompt: "Streaming response needed", userMarked: true, expectedResult: true},
		{name: "Long prompt", prompt: string(make([]byte, 600)), userMarked: true, expectedResult: false},
		{name: "Summarize document", prompt: "Summarize this document please", userMarked: true, expectedResult: false},
		{name: "Analyze log", prompt: "Analyze this log file", userMarked: true, expectedResult: false},
		{name: "Process file", prompt: "Process this file", userMarked: true, expectedResult: false},
		{name: "Batch", prompt: "Batch process these items", userMarked: true, expectedResult: false},
		{name: "Overnight", prompt: "Run this overnight", userMarked: true, expectedResult: false},
		{name: "Default trust", prompt: "Some request", userMarked: true, expectedResult: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ShouldAllowLatencyCritical(context.Background(), tt.prompt, tt.userMarked)
			if result != tt.expectedResult {
				t.Errorf("ShouldAllowLatencyCritical(%q, %v) = %v, want %v",
					tt.prompt, tt.userMarked, result, tt.expectedResult)
			}
		})
	}
}

// TestEstimateTokenCount is part of the public API.
func TestEstimateTokenCount(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name     string
		prompt   string
		expected int
	}{
		{name: "One word", prompt: "Answer in one word", expected: 5},
		{name: "One sentence", prompt: "Summarize in one sentence", expected: 20},
		{name: "Paragraph", prompt: "Write a paragraph about AI", expected: 100},
		{name: "Essay", prompt: "Write an essay on climate change", expected: 500},
		{name: "Article", prompt: "Write an article about programming", expected: 500},
		{name: "Default short", prompt: "Hi", expected: 0},                                      // len("Hi")/4 = 0, 0*2 = 0
		{name: "Default medium", prompt: "Tell me about Go programming language", expected: 18}, // ~9 chars/4 * 2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.EstimateTokenCount(tt.prompt)
			if result != tt.expected {
				t.Errorf("EstimateTokenCount(%q) = %d, want %d", tt.prompt, result, tt.expected)
			}
		})
	}
}

// MockBackend implements the backends.Backend interface for testing
type MockBackend struct {
	generateErr      error
	generateResponse string
}

// ID is part of the public API.
func (m *MockBackend) ID() string { return strMock }

// Type is part of the public API.
func (m *MockBackend) Type() string { return strMock }

// Name is part of the public API.
func (m *MockBackend) Name() string { return "mock-backend" }

// Hardware is part of the public API.
func (m *MockBackend) Hardware() string { return strMock }

// IsHealthy reports whether the condition holds.
func (m *MockBackend) IsHealthy() bool { return true }

// HealthCheck is part of the public API.
func (m *MockBackend) HealthCheck(_ context.Context) error { return nil }

// PowerWatts is part of the public API.
func (m *MockBackend) PowerWatts() float64 { return 0 }

// AvgLatencyMs is part of the public API.
func (m *MockBackend) AvgLatencyMs() int32 { return 0 }

// Priority is part of the public API.
func (m *MockBackend) Priority() int { return 0 }

// SupportsGenerate is part of the public API.
func (m *MockBackend) SupportsGenerate() bool { return true }

// SupportsStream is part of the public API.
func (m *MockBackend) SupportsStream() bool { return false }

// SupportsEmbed is part of the public API.
func (m *MockBackend) SupportsEmbed() bool { return false }

// ListModels is part of the public API.
func (m *MockBackend) ListModels(_ context.Context) ([]string, error) { return nil, nil }

// SupportsModel is part of the public API.
func (m *MockBackend) SupportsModel(_ string) bool { return true }

// GetMaxModelSizeGB is part of the public API.
func (m *MockBackend) GetMaxModelSizeGB() int { return 0 }

// GetSupportedModelPatterns is part of the public API.
func (m *MockBackend) GetSupportedModelPatterns() []string { return nil }

// GetPreferredModels is part of the public API.
func (m *MockBackend) GetPreferredModels() []string { return nil }

// Embed is part of the public API.
func (m *MockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// GenerateStream is part of the public API.
func (m *MockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}

// UpdateMetrics is part of the public API.
func (m *MockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *MockBackend) GetMetrics() *backends.BackendMetrics { return nil }

// Start is part of the public API.
func (m *MockBackend) Start(_ context.Context) error { return nil }

// Stop is part of the public API.
func (m *MockBackend) Stop(_ context.Context) error { return nil }

// Multimedia capability methods
func (m *MockBackend) SupportsAudioToText() bool { return false }

// SupportsTextToAudio is part of the public API.
func (m *MockBackend) SupportsTextToAudio() bool { return false }

// SupportsImageToText is part of the public API.
func (m *MockBackend) SupportsImageToText() bool { return false }

// SupportsTextToImage is part of the public API.
func (m *MockBackend) SupportsTextToImage() bool { return false }

// SupportsVideoToText is part of the public API.
func (m *MockBackend) SupportsVideoToText() bool { return false }

// SupportsTextToVideo is part of the public API.
func (m *MockBackend) SupportsTextToVideo() bool { return false }

// TranscribeAudio is part of the public API.
func (m *MockBackend) TranscribeAudio(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// TranscribeAudioStream is part of the public API.
func (m *MockBackend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeech is part of the public API.
func (m *MockBackend) SynthesizeSpeech(_ context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeechStream is part of the public API.
func (m *MockBackend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeImage is part of the public API.
func (m *MockBackend) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImage is part of the public API.
func (m *MockBackend) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImageStream is part of the public API.
func (m *MockBackend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideo is part of the public API.
func (m *MockBackend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideoStream is part of the public API.
func (m *MockBackend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideo is part of the public API.
func (m *MockBackend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideoStream is part of the public API.
func (m *MockBackend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// Generate is part of the public API.
func (m *MockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &backends.GenerateResponse{
		Response: m.generateResponse,
	}, nil
}

// TestClassifyWithModel_Simple is part of the public API.
func TestClassifyWithModel_Simple(t *testing.T) {
	mockBackend := &MockBackend{
		generateResponse: "SIMPLE",
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "What is the capital of France?")
	if err != nil {
		t.Errorf("ClassifyWithModel returned error: %v", err)
	}
	if result != ComplexitySimple {
		t.Errorf("Expected ComplexitySimple, got %v", result)
	}
}

// TestClassifyWithModel_Moderate is part of the public API.
func TestClassifyWithModel_Moderate(t *testing.T) {
	mockBackend := &MockBackend{
		generateResponse: "MODERATE",
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "Explain how photosynthesis works")
	if err != nil {
		t.Errorf("ClassifyWithModel returned error: %v", err)
	}
	if result != ComplexityModerate {
		t.Errorf("Expected ComplexityModerate, got %v", result)
	}
}

// TestClassifyWithModel_Complex is part of the public API.
func TestClassifyWithModel_Complex(t *testing.T) {
	mockBackend := &MockBackend{
		generateResponse: "COMPLEX",
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "Write a detailed analysis of quantum computing")
	if err != nil {
		t.Errorf("ClassifyWithModel returned error: %v", err)
	}
	if result != ComplexityComplex {
		t.Errorf("Expected ComplexityComplex, got %v", result)
	}
}

// TestClassifyWithModel_DefaultModerate is part of the public API.
func TestClassifyWithModel_DefaultModerate(t *testing.T) {
	mockBackend := &MockBackend{
		generateResponse: "UNKNOWN",
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "Some prompt")
	if err != nil {
		t.Errorf("ClassifyWithModel returned error: %v", err)
	}
	if result != ComplexityModerate {
		t.Errorf("Expected ComplexityModerate for unknown response, got %v", result)
	}
}

// TestClassifyWithModel_WithWhitespace is part of the public API.
func TestClassifyWithModel_WithWhitespace(t *testing.T) {
	mockBackend := &MockBackend{
		generateResponse: "  SIMPLE  \n",
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "Test prompt")
	if err != nil {
		t.Errorf("ClassifyWithModel returned error: %v", err)
	}
	if result != ComplexitySimple {
		t.Errorf("Expected ComplexitySimple after trimming, got %v", result)
	}
}

// TestClassifyWithModel_Error is part of the public API.
func TestClassifyWithModel_Error(t *testing.T) {
	mockBackend := &MockBackend{
		generateErr: errors.New("backend error"),
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "Test prompt")
	if err == nil {
		t.Errorf("Expected error from backend, got nil")
	}
	if result != ComplexityModerate {
		t.Errorf("Expected fallback to ComplexityModerate on error, got %v", result)
	}
}

// TestClassifyWithModel_CaseInsensitive is part of the public API.
func TestClassifyWithModel_CaseInsensitive(t *testing.T) {
	mockBackend := &MockBackend{
		generateResponse: "simple",
	}
	c := NewClassifier(mockBackend)

	result, err := c.ClassifyWithModel(context.Background(), "Test")
	if err != nil {
		t.Errorf("ClassifyWithModel returned error: %v", err)
	}
	if result != ComplexitySimple {
		t.Errorf("Expected ComplexitySimple with lowercase response, got %v", result)
	}
}

// TestClassifyPrompt_EdgeCases is part of the public API.
func TestClassifyPrompt_EdgeCases(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name        string
		prompt      string
		model       string
		description string
		expected    RequestComplexity
	}{
		// Edge case: less than 50 chars returns simple (< 50 check)
		{name: "49 characters", prompt: "a" + string(make([]byte, 48)), model: "", expected: ComplexitySimple, description: "Just under 50 char threshold"},

		// Edge case: exactly 50 chars does not trigger simple, gets moderate
		{name: "Exactly 50 chars", prompt: "a" + string(make([]byte, 49)), model: "", expected: ComplexityModerate, description: "Prompt exactly at length boundary (50 is not < 50)"},

		// Edge case: 51+ chars also doesn't trigger simple
		{name: "51 characters", prompt: "a" + string(make([]byte, 50)), model: "", expected: ComplexityModerate, description: "Just over 50 char threshold"},

		// Edge case: case sensitivity in patterns (>50 chars with simple pattern)
		{name: "Mixed case what", prompt: "WHAT IS this some more text to reach fifty chars total ok", model: "", expected: ComplexitySimple, description: "Pattern matching should be case-insensitive"},

		// Edge case: analyze in middle of sentence (Contains check, >50 chars)
		{name: "Analyze contains", prompt: "I need to analyze this code more thoroughly to understand the patterns", model: "", expected: ComplexityComplex, description: "Analyze pattern uses Contains"},

		// Edge case: model size boundaries
		{name: "1.5b model", prompt: "This is a test", model: "qwen2.5:1.5b", expected: ComplexitySimple, description: "1.5b model triggers simple"},
		{name: "0.5b model", prompt: "This is a test", model: strQwen2505b, expected: ComplexitySimple, description: "0.5b model triggers simple"},
		{name: "2b model", prompt: "This is a test with medium length content to trigger moderate", model: "qwen2.5:2b", expected: ComplexityModerate, description: "2b model doesn't trigger special handling"},

		// Edge case: complex pattern precedence - complex patterns checked before moderate patterns
		{name: "Complex precedence", prompt: "I need to analyze this code very carefully step by step to understand it better", model: "", expected: ComplexityComplex, description: "Complex pattern takes precedence over moderate"},

		// Edge case: briefly keyword (>50 chars)
		{name: "Briefly keyword 50+", prompt: "Please explain quantum computing briefly and concisely to help me understand it better", model: "", expected: ComplexitySimple, description: "Briefly keyword triggers simple even with >50 chars"},

		// Edge case: "one sentence" keyword (>50 chars)
		{name: "One sentence keyword", prompt: "Please summarize the entire document in one sentence to keep it brief and concise", model: "", expected: ComplexitySimple, description: "One sentence keyword triggers simple"},

		// Edge case: "in short" keyword (>50 chars)
		{name: "In short keyword", prompt: "Please tell me in short what are the main points of this article or document", model: "", expected: ComplexitySimple, description: "In short keyword triggers simple"},

		// Edge case: 0.5b large model string match
		{name: "0.5b exact model", prompt: "Test query", model: strQwen2505b, expected: ComplexitySimple, description: "Model string with 0.5b returns simple"},

		// Edge case: 1.5b large model string match
		{name: "1.5b exact model", prompt: "Test query longer than 50 chars to exceed the minimum threshold and test model patterns", model: "qwen2.5:1.5b", expected: ComplexitySimple, description: "Model string with 1.5b returns simple"},

		// Edge case: 33b large model (>50 chars)
		{name: "33b large model", prompt: "Test query that is longer than fifty characters to exceed the initial length check ok", model: "mixtral:33b", expected: ComplexityComplex, description: "Model string with 33b returns complex"},

		// Edge case: 70b large model (>50 chars)
		{name: "70b model large", prompt: "Test query that is longer than fifty characters to exceed the initial length check ok", model: "llama3:70b", expected: ComplexityComplex, description: "Model string with 70b returns complex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ClassifyPrompt(context.Background(), tt.prompt, tt.model)
			if result != tt.expected {
				t.Errorf("%s: Expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

// TestRecommendBackend_EdgeCases is part of the public API.
func TestRecommendBackend_EdgeCases(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		queueDepth  map[string]int
		name        string
		expected    string
		description string
		complexity  RequestComplexity
		onBattery   bool
	}{
		// ComplexitySimple edge cases
		{name: "NPU at threshold", complexity: ComplexitySimple, onBattery: false, queueDepth: map[string]int{strOllamaNpu: 3}, expected: strOllamaIgpu, description: "NPU at exactly 3 should fallback"},
		{name: "NPU under threshold", complexity: ComplexitySimple, onBattery: false, queueDepth: map[string]int{strOllamaNpu: 2}, expected: strOllamaNpu, description: "NPU under threshold should be used"},

		// ComplexityModerate on battery edge cases
		{name: "Moderate battery, Intel at threshold", complexity: ComplexityModerate, onBattery: true, queueDepth: map[string]int{strOllamaIgpu: 2}, expected: strOllamaNpu, description: "Intel GPU at 2 on battery should fallback to NPU"},
		{name: "Moderate battery, Intel under threshold", complexity: ComplexityModerate, onBattery: true, queueDepth: map[string]int{strOllamaIgpu: 1}, expected: strOllamaIgpu, description: "Intel GPU under threshold on battery should be used"},

		// ComplexityModerate on AC edge cases
		{name: "Moderate AC, Intel at threshold", complexity: ComplexityModerate, onBattery: false, queueDepth: map[string]int{strOllamaIgpu: 2}, expected: strOllamaNvidia, description: "Intel GPU at 2 on AC should use NVIDIA"},
		{name: "Moderate AC, Intel under threshold", complexity: ComplexityModerate, onBattery: false, queueDepth: map[string]int{strOllamaIgpu: 1}, expected: strOllamaIgpu, description: "Intel GPU under threshold on AC should be used"},

		// ComplexityComplex edge cases
		{name: "Complex on battery, NVIDIA at boundary", complexity: ComplexityComplex, onBattery: true, queueDepth: map[string]int{strOllamaNvidia: 2}, expected: strOllamaIgpu, description: "NVIDIA at 2 on battery should downgrade to Intel"},
		{name: "Complex on battery, NVIDIA below boundary", complexity: ComplexityComplex, onBattery: true, queueDepth: map[string]int{strOllamaNvidia: 1}, expected: strOllamaNvidia, description: "NVIDIA below 2 on battery should be used"},
		{name: "Complex on battery, NVIDIA at 0", complexity: ComplexityComplex, onBattery: true, queueDepth: map[string]int{strOllamaNvidia: 0}, expected: strOllamaNvidia, description: "NVIDIA at 0 on battery should be used"},
		{name: "Complex on AC always NVIDIA", complexity: ComplexityComplex, onBattery: false, queueDepth: map[string]int{strOllamaNvidia: 10}, expected: strOllamaNvidia, description: "NVIDIA on AC even when very busy"},

		// Invalid complexity (should default to igpu)
		{name: "Invalid complexity", complexity: RequestComplexity(999), onBattery: false, queueDepth: map[string]int{}, expected: strOllamaIgpu, description: "Invalid complexity defaults to igpu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.RecommendBackend(tt.complexity, tt.onBattery, tt.queueDepth)
			if result != tt.expected {
				t.Errorf("%s: Expected %s, got %s", tt.description, tt.expected, result)
			}
		})
	}
}

// TestEstimateTokenCount_EdgeCases is part of the public API.
func TestEstimateTokenCount_EdgeCases(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name        string
		prompt      string
		description string
		expected    int
	}{
		{name: "Empty prompt", prompt: "", expected: 0, description: "Empty prompt should return 0"},
		{name: "Very short", prompt: "a", expected: 0, description: "Single char should return 0"},
		{name: "Long prompt", prompt: string(make([]byte, 1000)), expected: 500, description: "1000 chars / 4 * 2 = 500"},
		{name: "Case insensitive word", prompt: "Answer In One Word", expected: 5, description: "Case insensitive pattern matching"},
		{name: "Multiple matching patterns", prompt: "Write a paragraph in one sentence about AI", expected: 20, description: "First explicit pattern match wins - 'one sentence' returns 20, checked before 'paragraph'"},
		{name: "No explicit pattern", prompt: "This is a normal prompt with no special markers", expected: 22, description: "Default calculation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.EstimateTokenCount(tt.prompt)
			if result != tt.expected {
				t.Errorf("%s: Expected %d, got %d", tt.description, tt.expected, result)
			}
		})
	}
}
