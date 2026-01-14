package classifier

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

func TestNewClassifier(t *testing.T) {
	classifier := NewClassifier(nil)
	if classifier == nil {
		t.Fatal("NewClassifier returned nil")
	}
}

func TestClassifyPrompt_Simple(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{"Short prompt", "Hi", ""},
		{"What is question", "What is Go?", ""},
		{"Who is question", "Who is Alan Turing?", ""},
		{"When was question", "When was Python created?", ""},
		{"Where is question", "Where is New York?", ""},
		{"How old question", "How old is the internet?", ""},
		{"How many question", "How many days in a year?", ""},
		{"Yes or no", "Is Go compiled? Yes or no", ""},
		{"True or false", "Is Python interpreted? True or false", ""},
		{"Briefly", "Explain quantum physics briefly", ""},
		{"One sentence", "Summarize this in one sentence", ""},
		{"In short", "Tell me in short about AI", ""},
		{"Small model", "Test prompt", "qwen2.5:0.5b"},
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

func TestClassifyPrompt_Complex(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{"Write detailed", "Write a detailed analysis of climate change impacts on ecosystems", ""},
		{"Explain in depth", "Explain in depth how neural networks work and their applications", ""},
		{"Analyze", "Analyze this data set thoroughly and provide detailed insights", ""},
		{"Compare and contrast", "Compare and contrast Python and Go programming languages in detail", ""},
		{"Comprehensive", "Create a comprehensive guide to Rust programming for beginners", ""},
		{"Generate code", "Generate code for a REST API with authentication and database integration", ""},
		{"Write story", "Write a story about a robot discovering consciousness and emotions", ""},
		{"Write essay", "Write an essay on democracy and its evolution through history", ""},
		{"Compose", "Compose a long email to the team about the new project direction", ""},
		{"Develop plan", "Develop a plan for project migration from legacy systems to modern architecture", ""},
		{"Large model", "This is a test prompt for large language models like llama3", "llama3:70b"},
		{"Large model 2", "Another test prompt for very large models like mixtral", "mixtral:33b"},
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

func TestClassifyPrompt_Moderate(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{"Step by step", "Explain step by step how to install Go on different operating systems", ""},
		{"First", "First, we need to understand the basics of how this system works", ""},
		{"Multiple questions", "Can you tell me what AI is? And also how does machine learning work?", ""},
		{"Medium length", "Can you explain to me how machine learning algorithms work and what are the main types?", ""},
		{"Default", "Tell me about programming languages and their characteristics", ""},
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

func TestRecommendBackend_Simple(t *testing.T) {
	c := NewClassifier(nil)

	// Empty queue, should prefer NPU
	result := c.RecommendBackend(ComplexitySimple, false, map[string]int{})
	if result != "ollama-npu" {
		t.Errorf("Expected ollama-npu for simple with empty queue, got %s", result)
	}

	// NPU busy, should use Intel GPU
	result = c.RecommendBackend(ComplexitySimple, false, map[string]int{"ollama-npu": 5})
	if result != "ollama-igpu" {
		t.Errorf("Expected ollama-igpu when NPU busy, got %s", result)
	}
}

func TestRecommendBackend_Moderate(t *testing.T) {
	c := NewClassifier(nil)

	// On AC power, prefer Intel GPU
	result := c.RecommendBackend(ComplexityModerate, false, map[string]int{})
	if result != "ollama-igpu" {
		t.Errorf("Expected ollama-igpu for moderate on AC, got %s", result)
	}

	// On battery, prefer efficient backends
	result = c.RecommendBackend(ComplexityModerate, true, map[string]int{})
	if result != "ollama-igpu" {
		t.Errorf("Expected ollama-igpu for moderate on battery, got %s", result)
	}

	// Intel GPU busy, use NVIDIA on AC
	result = c.RecommendBackend(ComplexityModerate, false, map[string]int{"ollama-igpu": 5})
	if result != "ollama-nvidia" {
		t.Errorf("Expected ollama-nvidia when Intel busy on AC, got %s", result)
	}

	// Intel GPU busy on battery, use NPU
	result = c.RecommendBackend(ComplexityModerate, true, map[string]int{"ollama-igpu": 5})
	if result != "ollama-npu" {
		t.Errorf("Expected ollama-npu when Intel busy on battery, got %s", result)
	}
}

func TestRecommendBackend_Complex(t *testing.T) {
	c := NewClassifier(nil)

	// Prefer NVIDIA for complex
	result := c.RecommendBackend(ComplexityComplex, false, map[string]int{})
	if result != "ollama-nvidia" {
		t.Errorf("Expected ollama-nvidia for complex on AC, got %s", result)
	}

	// On battery with NVIDIA busy, downgrade to Intel
	result = c.RecommendBackend(ComplexityComplex, true, map[string]int{"ollama-nvidia": 3})
	if result != "ollama-igpu" {
		t.Errorf("Expected ollama-igpu when NVIDIA busy on battery, got %s", result)
	}

	// On battery with NVIDIA not busy, still use NVIDIA
	result = c.RecommendBackend(ComplexityComplex, true, map[string]int{"ollama-nvidia": 1})
	if result != "ollama-nvidia" {
		t.Errorf("Expected ollama-nvidia when not busy on battery, got %s", result)
	}
}

func TestShouldAllowLatencyCritical(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name           string
		prompt         string
		userMarked     bool
		expectedResult bool
	}{
		{"Not marked critical", "Test prompt", false, false},
		{"Quick request", "Quick answer please", true, true},
		{"Urgent request", "This is urgent", true, true},
		{"Immediately", "I need this immediately", true, true},
		{"Right now", "I need this right now", true, true},
		{"ASAP", "Send this asap", true, true},
		{"Real-time", "Real-time analysis needed", true, true},
		{"Live", "Live updates required", true, true},
		{"Streaming", "Streaming response needed", true, true},
		{"Long prompt", string(make([]byte, 600)), true, false},
		{"Summarize document", "Summarize this document please", true, false},
		{"Analyze log", "Analyze this log file", true, false},
		{"Process file", "Process this file", true, false},
		{"Batch", "Batch process these items", true, false},
		{"Overnight", "Run this overnight", true, false},
		{"Default trust", "Some request", true, true},
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

func TestEstimateTokenCount(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name     string
		prompt   string
		expected int
	}{
		{"One word", "Answer in one word", 5},
		{"One sentence", "Summarize in one sentence", 20},
		{"Paragraph", "Write a paragraph about AI", 100},
		{"Essay", "Write an essay on climate change", 500},
		{"Article", "Write an article about programming", 500},
		{"Default short", "Hi", 0},          // len("Hi")/4 = 0, 0*2 = 0
		{"Default medium", "Tell me about Go programming language", 18}, // ~9 chars/4 * 2
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
	generateResponse string
	generateErr      error
}

func (m *MockBackend) ID() string                                                      { return "mock" }
func (m *MockBackend) Type() string                                                    { return "mock" }
func (m *MockBackend) Name() string                                                    { return "mock-backend" }
func (m *MockBackend) Hardware() string                                                { return "mock" }
func (m *MockBackend) IsHealthy() bool                                                 { return true }
func (m *MockBackend) HealthCheck(ctx context.Context) error                           { return nil }
func (m *MockBackend) PowerWatts() float64                                             { return 0 }
func (m *MockBackend) AvgLatencyMs() int32                                             { return 0 }
func (m *MockBackend) Priority() int                                                   { return 0 }
func (m *MockBackend) SupportsGenerate() bool                                          { return true }
func (m *MockBackend) SupportsStream() bool                                            { return false }
func (m *MockBackend) SupportsEmbed() bool                                             { return false }
func (m *MockBackend) ListModels(ctx context.Context) ([]string, error)                { return nil, nil }
func (m *MockBackend) SupportsModel(modelName string) bool                             { return true }
func (m *MockBackend) GetMaxModelSizeGB() int                                          { return 0 }
func (m *MockBackend) GetSupportedModelPatterns() []string                             { return nil }
func (m *MockBackend) GetPreferredModels() []string                                    { return nil }
func (m *MockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}
func (m *MockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return nil, nil
}
func (m *MockBackend) UpdateMetrics(latencyMs int32, success bool)    {}
func (m *MockBackend) GetMetrics() *backends.BackendMetrics           { return nil }
func (m *MockBackend) Start(ctx context.Context) error                { return nil }
func (m *MockBackend) Stop(ctx context.Context) error                 { return nil }

// Multimedia capability methods
func (m *MockBackend) SupportsAudioToText() bool { return false }
func (m *MockBackend) SupportsTextToAudio() bool { return false }
func (m *MockBackend) SupportsImageToText() bool { return false }
func (m *MockBackend) SupportsTextToImage() bool { return false }
func (m *MockBackend) SupportsVideoToText() bool { return false }
func (m *MockBackend) SupportsTextToVideo() bool { return false }
func (m *MockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }

func (m *MockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &backends.GenerateResponse{
		Response: m.generateResponse,
	}, nil
}

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

func TestClassifyPrompt_EdgeCases(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name       string
		prompt     string
		model      string
		expected   RequestComplexity
		description string
	}{
		// Edge case: less than 50 chars returns simple (< 50 check)
		{"49 characters", "a" + string(make([]byte, 48)), "", ComplexitySimple, "Just under 50 char threshold"},

		// Edge case: exactly 50 chars does not trigger simple, gets moderate
		{"Exactly 50 chars", "a" + string(make([]byte, 49)), "", ComplexityModerate, "Prompt exactly at length boundary (50 is not < 50)"},

		// Edge case: 51+ chars also doesn't trigger simple
		{"51 characters", "a" + string(make([]byte, 50)), "", ComplexityModerate, "Just over 50 char threshold"},

		// Edge case: case sensitivity in patterns (>50 chars with simple pattern)
		{"Mixed case what", "WHAT IS this some more text to reach fifty chars total ok", "", ComplexitySimple, "Pattern matching should be case-insensitive"},

		// Edge case: analyze in middle of sentence (Contains check, >50 chars)
		{"Analyze contains", "I need to analyze this code more thoroughly to understand the patterns", "", ComplexityComplex, "Analyze pattern uses Contains"},

		// Edge case: model size boundaries
		{"1.5b model", "This is a test", "qwen2.5:1.5b", ComplexitySimple, "1.5b model triggers simple"},
		{"0.5b model", "This is a test", "qwen2.5:0.5b", ComplexitySimple, "0.5b model triggers simple"},
		{"2b model", "This is a test with medium length content to trigger moderate", "qwen2.5:2b", ComplexityModerate, "2b model doesn't trigger special handling"},

		// Edge case: complex pattern precedence - complex patterns checked before moderate patterns
		{"Complex precedence", "I need to analyze this code very carefully step by step to understand it better", "", ComplexityComplex, "Complex pattern takes precedence over moderate"},

		// Edge case: briefly keyword (>50 chars)
		{"Briefly keyword 50+", "Please explain quantum computing briefly and concisely to help me understand it better", "", ComplexitySimple, "Briefly keyword triggers simple even with >50 chars"},

		// Edge case: "one sentence" keyword (>50 chars)
		{"One sentence keyword", "Please summarize the entire document in one sentence to keep it brief and concise", "", ComplexitySimple, "One sentence keyword triggers simple"},

		// Edge case: "in short" keyword (>50 chars)
		{"In short keyword", "Please tell me in short what are the main points of this article or document", "", ComplexitySimple, "In short keyword triggers simple"},

		// Edge case: 0.5b large model string match
		{"0.5b exact model", "Test query", "qwen2.5:0.5b", ComplexitySimple, "Model string with 0.5b returns simple"},

		// Edge case: 1.5b large model string match
		{"1.5b exact model", "Test query longer than 50 chars to exceed the minimum threshold and test model patterns", "qwen2.5:1.5b", ComplexitySimple, "Model string with 1.5b returns simple"},

		// Edge case: 33b large model (>50 chars)
		{"33b large model", "Test query that is longer than fifty characters to exceed the initial length check ok", "mixtral:33b", ComplexityComplex, "Model string with 33b returns complex"},

		// Edge case: 70b large model (>50 chars)
		{"70b model large", "Test query that is longer than fifty characters to exceed the initial length check ok", "llama3:70b", ComplexityComplex, "Model string with 70b returns complex"},
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

func TestRecommendBackend_EdgeCases(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name      string
		complexity RequestComplexity
		onBattery bool
		queueDepth map[string]int
		expected  string
		description string
	}{
		// ComplexitySimple edge cases
		{"NPU at threshold", ComplexitySimple, false, map[string]int{"ollama-npu": 3}, "ollama-igpu", "NPU at exactly 3 should fallback"},
		{"NPU under threshold", ComplexitySimple, false, map[string]int{"ollama-npu": 2}, "ollama-npu", "NPU under threshold should be used"},

		// ComplexityModerate on battery edge cases
		{"Moderate battery, Intel at threshold", ComplexityModerate, true, map[string]int{"ollama-igpu": 2}, "ollama-npu", "Intel GPU at 2 on battery should fallback to NPU"},
		{"Moderate battery, Intel under threshold", ComplexityModerate, true, map[string]int{"ollama-igpu": 1}, "ollama-igpu", "Intel GPU under threshold on battery should be used"},

		// ComplexityModerate on AC edge cases
		{"Moderate AC, Intel at threshold", ComplexityModerate, false, map[string]int{"ollama-igpu": 2}, "ollama-nvidia", "Intel GPU at 2 on AC should use NVIDIA"},
		{"Moderate AC, Intel under threshold", ComplexityModerate, false, map[string]int{"ollama-igpu": 1}, "ollama-igpu", "Intel GPU under threshold on AC should be used"},

		// ComplexityComplex edge cases
		{"Complex on battery, NVIDIA at boundary", ComplexityComplex, true, map[string]int{"ollama-nvidia": 2}, "ollama-igpu", "NVIDIA at 2 on battery should downgrade to Intel"},
		{"Complex on battery, NVIDIA below boundary", ComplexityComplex, true, map[string]int{"ollama-nvidia": 1}, "ollama-nvidia", "NVIDIA below 2 on battery should be used"},
		{"Complex on battery, NVIDIA at 0", ComplexityComplex, true, map[string]int{"ollama-nvidia": 0}, "ollama-nvidia", "NVIDIA at 0 on battery should be used"},
		{"Complex on AC always NVIDIA", ComplexityComplex, false, map[string]int{"ollama-nvidia": 10}, "ollama-nvidia", "NVIDIA on AC even when very busy"},

		// Invalid complexity (should default to igpu)
		{"Invalid complexity", RequestComplexity(999), false, map[string]int{}, "ollama-igpu", "Invalid complexity defaults to igpu"},
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

func TestEstimateTokenCount_EdgeCases(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		name     string
		prompt   string
		expected int
		description string
	}{
		{"Empty prompt", "", 0, "Empty prompt should return 0"},
		{"Very short", "a", 0, "Single char should return 0"},
		{"Long prompt", string(make([]byte, 1000)), 500, "1000 chars / 4 * 2 = 500"},
		{"Case insensitive word", "Answer In One Word", 5, "Case insensitive pattern matching"},
		{"Multiple matching patterns", "Write a paragraph in one sentence about AI", 20, "First explicit pattern match wins - 'one sentence' returns 20, checked before 'paragraph'"},
		{"No explicit pattern", "This is a normal prompt with no special markers", 22, "Default calculation"},
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
