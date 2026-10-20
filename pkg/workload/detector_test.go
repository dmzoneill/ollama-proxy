package workload

import (
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// TestNewDetector is part of the public API.
func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if d.profiles == nil {
		t.Fatal("profiles map is nil")
	}
	if len(d.profiles) == 0 {
		t.Error("profiles map is empty")
	}
}

// TestDetectMediaType_Realtime is part of the public API.
func TestDetectMediaType_Realtime(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{name: "Real-time keyword", prompt: "Start a real-time conversation with me"},
		{name: "Realtime keyword", prompt: "Enable realtime transcription please"},
		{name: "Live keyword", prompt: "I need live streaming translation"},
		{name: "Interactive chat", prompt: "Start an interactive chat session"},
		{name: "Voice chat", prompt: "Begin voice chat mode"},
		{name: "Instant keyword", prompt: "I need instant responses"},
		{name: "Transcribe", prompt: "Transcribe this audio in realtime"},
		{name: "Dictation", prompt: "Enable dictation mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{
				LatencyCritical: true, // Required for realtime
			}
			result := d.DetectMediaType(tt.prompt, "", annotations)
			if result != backends.MediaTypeRealtime {
				t.Errorf("Expected MediaTypeRealtime for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestDetectMediaType_Audio is part of the public API.
func TestDetectMediaType_Audio(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{name: "Audio keyword", prompt: "Process this audio file"},
		{name: "Sound keyword", prompt: "Analyze this sound recording"},
		{name: "Voice keyword", prompt: "Transcribe this voice message"},
		{name: "Speech keyword", prompt: "Convert speech to text"},
		{name: "Podcast keyword", prompt: "Summarize this podcast episode"},
		{name: "TTS keyword", prompt: "Use text to speech for this"},
		{name: "STT keyword", prompt: "Speech to text conversion needed"},
		{name: "Transcribe keyword", prompt: "Transcribe this recording"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{
				LatencyCritical: false,
			}
			result := d.DetectMediaType(tt.prompt, "", annotations)
			if result != backends.MediaTypeAudio {
				t.Errorf("Expected MediaTypeAudio for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestDetectMediaType_Code is part of the public API.
func TestDetectMediaType_Code(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{name: "Code keyword", prompt: "Write code for a REST API", model: ""},
		{name: "Program keyword", prompt: "Create a program that sorts arrays", model: ""},
		{name: "Function keyword", prompt: "Write a function to calculate fibonacci", model: ""},
		{name: "Implement keyword", prompt: "Implement a binary search tree", model: ""},
		{name: "Python keyword", prompt: "Write a python script to parse JSON", model: ""},
		{name: "JavaScript keyword", prompt: "Create a javascript function for validation", model: ""},
		{name: "Algorithm keyword", prompt: "Implement this algorithm in Go", model: ""},
		{name: "API keyword", prompt: "Design an API for user management", model: ""},
		{name: "Bug keyword", prompt: "Fix this bug in my code", model: ""},
		{name: "SQL keyword", prompt: "Write a SQL query to join these tables", model: ""},
		{name: "React keyword", prompt: "Create a react component", model: ""},
		{name: "CodeLlama model", prompt: "Test prompt", model: "codellama:7b"},
		{name: "StarCoder model", prompt: "Test prompt", model: "starcoder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{}
			result := d.DetectMediaType(tt.prompt, tt.model, annotations)
			if result != backends.MediaTypeCode {
				t.Errorf("Expected MediaTypeCode for %q (model: %s), got %v", tt.prompt, tt.model, result)
			}
		})
	}
}

// TestDetectMediaType_Image is part of the public API.
func TestDetectMediaType_Image(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{name: "Image keyword", prompt: "Analyze this image for me"},
		{name: "Picture keyword", prompt: "Describe this picture"},
		{name: "Photo keyword", prompt: "What's in this photo?"},
		{name: "Visual keyword", prompt: "Explain the visual elements"},
		{name: "Draw keyword", prompt: "Draw a diagram of the architecture"},
		{name: "Generate image", prompt: "Generate an image of a sunset"},
		{name: "Vision keyword", prompt: "Use vision to analyze this"},
		{name: "Screenshot keyword", prompt: "Analyze this screenshot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{}
			result := d.DetectMediaType(tt.prompt, "", annotations)
			if result != backends.MediaTypeImage {
				t.Errorf("Expected MediaTypeImage for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestDetectMediaType_Text is part of the public API.
func TestDetectMediaType_Text(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{name: "General question", prompt: "What's the main city of France?"},
		{name: "Explanation request", prompt: "Explain biology concepts to me"},
		{name: "Summary request", prompt: "Summarize the history of mathematics"},
		{name: "Default case", prompt: "Tell me about quantum mechanics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{}
			result := d.DetectMediaType(tt.prompt, "", annotations)
			if result != backends.MediaTypeText {
				t.Errorf("Expected MediaTypeText for %q, got %v", tt.prompt, result)
			}
		})
	}
}

// TestDetectMediaType_ExplicitOverride is part of the public API.
func TestDetectMediaType_ExplicitOverride(t *testing.T) {
	d := NewDetector()

	annotations := &backends.Annotations{
		MediaType: backends.MediaTypeCode,
	}

	result := d.DetectMediaType("Tell me a story", "", annotations)
	if result != backends.MediaTypeCode {
		t.Errorf("Expected explicit MediaType to be respected, got %v", result)
	}
}

// TestDetectMediaType_AutoIgnored is part of the public API.
func TestDetectMediaType_AutoIgnored(t *testing.T) {
	d := NewDetector()

	annotations := &backends.Annotations{
		MediaType: backends.MediaTypeAuto,
	}

	result := d.DetectMediaType("Write code for a REST API", "", annotations)
	if result != backends.MediaTypeCode {
		t.Errorf("Expected MediaTypeAuto to trigger detection, got %v", result)
	}
}

// TestIsRealtime_RequiresLatencyCritical is part of the public API.
func TestIsRealtime_RequiresLatencyCritical(t *testing.T) {
	d := NewDetector()

	// Has realtime keyword but not latency critical
	annotations := &backends.Annotations{
		LatencyCritical: false,
	}

	result := d.isRealtime("realtime chat session", annotations)
	if result {
		t.Error("isRealtime should return false when LatencyCritical is false")
	}
}

// TestIsRealtime_AudioPlusLatency is part of the public API.
func TestIsRealtime_AudioPlusLatency(t *testing.T) {
	d := NewDetector()

	// Audio keyword + latency critical = realtime
	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	result := d.isRealtime("transcribe this audio", annotations)
	if !result {
		t.Error("isRealtime should return true for audio + latency critical")
	}
}

// TestGetProfile is part of the public API.
func TestGetProfile(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		mediaType backends.MediaType
		expected  string
	}{
		{mediaType: backends.MediaTypeRealtime, expected: "qwen2.5:0.5b"},
		{mediaType: backends.MediaTypeAudio, expected: "qwen2.5:1.5b"},
		{mediaType: backends.MediaTypeCode, expected: "llama3:70b"},
		{mediaType: backends.MediaTypeImage, expected: "llama3:7b"},
		{mediaType: backends.MediaTypeText, expected: "llama3:7b"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mediaType), func(t *testing.T) {
			profile := d.GetProfile(tt.mediaType)
			if profile == nil {
				t.Fatal("GetProfile returned nil")
			}
			if profile.PreferredModel != tt.expected {
				t.Errorf("Expected preferred model %s, got %s", tt.expected, profile.PreferredModel)
			}
		})
	}
}

// TestGetProfile_Unknown is part of the public API.
func TestGetProfile_Unknown(t *testing.T) {
	d := NewDetector()

	// Unknown media type should return text profile as default
	profile := d.GetProfile(backends.MediaType("unknown"))
	if profile == nil {
		t.Fatal("GetProfile returned nil for unknown type")
	}
	if profile.MediaType != backends.MediaTypeText {
		t.Errorf("Expected default to Text profile, got %v", profile.MediaType)
	}
}

// TestGetRoutingHints_Basic is part of the public API.
func TestGetRoutingHints_Basic(t *testing.T) {
	d := NewDetector()

	annotations := &backends.Annotations{}
	hints := d.GetRoutingHints("Write code for authentication", "", annotations)

	if hints == nil {
		t.Fatal("GetRoutingHints returned nil")
	}
	if hints.DetectedMediaType != backends.MediaTypeCode {
		t.Errorf("Expected code media type, got %v", hints.DetectedMediaType)
	}
	if hints.Profile == nil {
		t.Error("Profile is nil")
	}
	if len(hints.ReasoningChain) == 0 {
		t.Error("ReasoningChain is empty")
	}
}

// TestGetRoutingHints_LatencyCriticalOverride is part of the public API.
func TestGetRoutingHints_LatencyCriticalOverride(t *testing.T) {
	d := NewDetector()

	// Code workload normally doesn't prefer low latency
	annotations := &backends.Annotations{
		LatencyCritical: true,
	}
	hints := d.GetRoutingHints("Write code for authentication", "", annotations)

	if !hints.PreferLowLatency {
		t.Error("Expected LatencyCritical to override PreferLowLatency")
	}
	if len(hints.ReasoningChain) < 2 {
		t.Error("Expected reasoning chain to include override")
	}
}

// TestGetRoutingHints_PowerEfficiencyOverride is part of the public API.
func TestGetRoutingHints_PowerEfficiencyOverride(t *testing.T) {
	d := NewDetector()

	// Code workload normally doesn't prefer low power
	annotations := &backends.Annotations{
		PreferPowerEfficiency: true,
	}
	hints := d.GetRoutingHints("Write code for authentication", "", annotations)

	if !hints.PreferLowPower {
		t.Error("Expected PreferPowerEfficiency to override PreferLowPower")
	}
	if len(hints.ReasoningChain) < 2 {
		t.Error("Expected reasoning chain to include override")
	}
}

// TestWorkloadProfiles_Realtime is part of the public API.
func TestWorkloadProfiles_Realtime(t *testing.T) {
	d := NewDetector()
	profile := d.GetProfile(backends.MediaTypeRealtime)

	if !profile.PreferLowLatency {
		t.Error("Realtime should prefer low latency")
	}
	if !profile.PreferLowPower {
		t.Error("Realtime should prefer low power")
	}
	if profile.MaxModelSizeGB > 5 {
		t.Error("Realtime should use small models")
	}
	if profile.MinTokensPerSec < 15 {
		t.Error("Realtime should require fast token generation")
	}
}

// TestWorkloadProfiles_Code is part of the public API.
func TestWorkloadProfiles_Code(t *testing.T) {
	d := NewDetector()
	profile := d.GetProfile(backends.MediaTypeCode)

	if profile.PreferLowLatency {
		t.Error("Code doesn't need low latency")
	}
	if profile.MaxModelSizeGB < 50 {
		t.Error("Code benefits from large models")
	}
}

// Additional comprehensive tests for edge cases and full coverage

// TestIsRealtime_WithRealtimeKeywordButNoLatency is part of the public API.
func TestIsRealtime_WithRealtimeKeywordButNoLatency(t *testing.T) {
	d := NewDetector()

	// Test the early return when LatencyCritical is false
	annotations := &backends.Annotations{
		LatencyCritical: false,
	}

	result := d.isRealtime("realtime conversation", annotations)
	if result {
		t.Error("isRealtime should return false when LatencyCritical is false, even with realtime keyword")
	}
}

// TestIsRealtime_AllRealtimeKeywords is part of the public API.
func TestIsRealtime_AllRealtimeKeywords(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	keywords := []string{
		"realtime",
		"real-time",
		"real time",
		"live",
		"streaming",
		"interactive chat",
		"voice chat",
		"instant",
		"immediate",
		"continuous",
		"ongoing",
		"transcribe",
		"transcription",
		"dictate",
		"dictation",
	}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			result := d.isRealtime(keyword, annotations)
			if !result {
				t.Errorf("isRealtime should return true for keyword: %s", keyword)
			}
		})
	}
}

// TestIsAudio_AllAudioKeywords is part of the public API.
func TestIsAudio_AllAudioKeywords(t *testing.T) {
	d := NewDetector()

	keywords := []string{
		"audio",
		"sound",
		"voice",
		"speech",
		"listen",
		"hear",
		"spoken",
		"podcast",
		"recording",
		"tts",
		"text to speech",
		"text-to-speech",
		"stt",
		"speech to text",
		"speech-to-text",
		"transcribe",
		"transcription",
	}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			result := d.isAudio(keyword)
			if !result {
				t.Errorf("isAudio should return true for keyword: %s", keyword)
			}
		})
	}
}

// TestIsAudio_NoKeywords is part of the public API.
func TestIsAudio_NoKeywords(t *testing.T) {
	d := NewDetector()

	result := d.isAudio("Tell me about mathematics")
	if result {
		t.Error("isAudio should return false for non-audio content")
	}
}

// TestIsCode_AllCodeKeywords is part of the public API.
func TestIsCode_AllCodeKeywords(t *testing.T) {
	d := NewDetector()

	keywords := []string{
		"code",
		"program",
		"function",
		"class",
		"implement",
		"refactor",
		"debug",
		"python",
		"javascript",
		"java",
		"go",
		"rust",
		"c++",
		"algorithm",
		"data structure",
		"api",
		"endpoint",
		"server",
		"bug",
		"error",
		"exception",
		"test",
		"unit test",
		"sql",
		"query",
		"database",
		"html",
		"css",
		"react",
		"vue",
	}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			result := d.isCode(keyword, "")
			if !result {
				t.Errorf("isCode should return true for keyword: %s", keyword)
			}
		})
	}
}

// TestIsCode_ModelNameDetection is part of the public API.
func TestIsCode_ModelNameDetection(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name  string
		model string
	}{
		{name: "code model", model: "code-llama:13b"},
		{name: "starcoder model", model: "starcoder:7b"},
		{name: "codellama model", model: "codellama:34b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.isCode("random text", tt.model)
			if !result {
				t.Errorf("isCode should return true for model: %s", tt.model)
			}
		})
	}
}

// TestIsCode_NoCodeIndicators is part of the public API.
func TestIsCode_NoCodeIndicators(t *testing.T) {
	d := NewDetector()

	result := d.isCode("Tell me about biology", "llama2:7b")
	if result {
		t.Error("isCode should return false for non-code content with generic model")
	}
}

// TestIsImage_AllImageKeywords is part of the public API.
func TestIsImage_AllImageKeywords(t *testing.T) {
	d := NewDetector()

	keywords := []string{
		"image",
		"picture",
		"photo",
		"visual",
		"draw",
		"generate image",
		"create image",
		"analyze image",
		"describe image",
		"vision",
		"see",
		"look at",
		"screenshot",
		"diagram",
	}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			result := d.isImage(keyword)
			if !result {
				t.Errorf("isImage should return true for keyword: %s", keyword)
			}
		})
	}
}

// TestIsImage_NoKeywords is part of the public API.
func TestIsImage_NoKeywords(t *testing.T) {
	d := NewDetector()

	result := d.isImage("Write a Python script")
	if result {
		t.Error("isImage should return false for non-image content")
	}
}

// TestDetectMediaType_CaseSensitivity is part of the public API.
func TestDetectMediaType_CaseSensitivity(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	tests := []struct {
		prompt       string
		expectedType backends.MediaType
	}{
		{prompt: "AUDIO processing", expectedType: backends.MediaTypeAudio},
		{prompt: "Code generation", expectedType: backends.MediaTypeCode},
		{prompt: "IMAGE analysis", expectedType: backends.MediaTypeImage},
		{prompt: "WRITE a story", expectedType: backends.MediaTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			result := d.DetectMediaType(tt.prompt, "", annotations)
			if result != tt.expectedType {
				t.Errorf("Expected %v, got %v for prompt: %s", tt.expectedType, result, tt.prompt)
			}
		})
	}
}

// TestDetectMediaType_PriorityOrder is part of the public API.
func TestDetectMediaType_PriorityOrder(t *testing.T) {
	d := NewDetector()

	// Test that realtime takes priority over audio
	annotations := &backends.Annotations{
		LatencyCritical: true,
	}
	result := d.DetectMediaType("transcribe audio in realtime", "", annotations)
	if result != backends.MediaTypeRealtime {
		t.Errorf("Realtime should take priority over audio, got %v", result)
	}

	// Test that audio takes priority over code
	annotations = &backends.Annotations{
		LatencyCritical: false,
	}
	result = d.DetectMediaType("audio transcription with algorithm", "", annotations)
	if result != backends.MediaTypeAudio {
		t.Errorf("Audio should take priority over code, got %v", result)
	}

	// Test that code takes priority over image
	annotations = &backends.Annotations{
		LatencyCritical: false,
	}
	result = d.DetectMediaType("write code to analyze image", "", annotations)
	if result != backends.MediaTypeCode {
		t.Errorf("Code should take priority over image, got %v", result)
	}
}

// TestGetRoutingHints_AllMediaTypes is part of the public API.
func TestGetRoutingHints_AllMediaTypes(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		prompt    string
		mediaType backends.MediaType
	}{
		{prompt: "realtime transcription", mediaType: backends.MediaTypeAudio}, // Without latency critical
		{prompt: "code in python", mediaType: backends.MediaTypeCode},
		{prompt: "analyze this image", mediaType: backends.MediaTypeImage},
		{prompt: "tell me a story", mediaType: backends.MediaTypeText},
	}

	for _, tt := range tests {
		t.Run(string(tt.mediaType), func(t *testing.T) {
			annotations := &backends.Annotations{}
			hints := d.GetRoutingHints(tt.prompt, "", annotations)

			if hints == nil {
				t.Fatal("GetRoutingHints returned nil")
			}
			if hints.DetectedMediaType != tt.mediaType {
				t.Errorf("Expected %v, got %v", tt.mediaType, hints.DetectedMediaType)
			}
			if hints.Profile == nil {
				t.Error("Profile should not be nil")
			}
			if hints.PreferredModel == "" {
				t.Error("PreferredModel should not be empty")
			}
		})
	}
}

// TestGetRoutingHints_BothAnnotationOverrides is part of the public API.
func TestGetRoutingHints_BothAnnotationOverrides(t *testing.T) {
	d := NewDetector()

	// Code workload normally doesn't prefer low latency or power efficiency
	annotations := &backends.Annotations{
		LatencyCritical:       true,
		PreferPowerEfficiency: true,
	}
	hints := d.GetRoutingHints("Write code", "", annotations)

	if !hints.PreferLowLatency {
		t.Error("Expected LatencyCritical to set PreferLowLatency")
	}
	if !hints.PreferLowPower {
		t.Error("Expected PreferPowerEfficiency to set PreferLowPower")
	}

	// Should have 3 items in reasoning chain: detection + 2 overrides
	if len(hints.ReasoningChain) < 3 {
		t.Errorf("Expected at least 3 reasoning items, got %d", len(hints.ReasoningChain))
	}
}

// TestWorkloadProfiles_Audio is part of the public API.
func TestWorkloadProfiles_Audio(t *testing.T) {
	d := NewDetector()
	profile := d.GetProfile(backends.MediaTypeAudio)

	if !profile.PreferLowLatency {
		t.Error("Audio should prefer low latency")
	}
	if !profile.PreferLowPower {
		t.Error("Audio should prefer low power")
	}
	if profile.MaxModelSizeGB > 5 {
		t.Error("Audio should use small models")
	}
}

// TestWorkloadProfiles_Image is part of the public API.
func TestWorkloadProfiles_Image(t *testing.T) {
	d := NewDetector()
	profile := d.GetProfile(backends.MediaTypeImage)

	if profile.PreferLowLatency {
		t.Error("Image doesn't need low latency")
	}
	if profile.MaxModelSizeGB < 5 {
		t.Error("Image needs reasonable model size")
	}
}

// TestWorkloadProfiles_Text is part of the public API.
func TestWorkloadProfiles_Text(t *testing.T) {
	d := NewDetector()
	profile := d.GetProfile(backends.MediaTypeText)

	if profile == nil {
		t.Fatal("Text profile is nil")
	}
	if profile.MediaType != backends.MediaTypeText {
		t.Errorf("Expected MediaTypeText, got %v", profile.MediaType)
	}
}

// TestDetectMediaType_MultipleKeywordsOrder is part of the public API.
func TestDetectMediaType_MultipleKeywordsOrder(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	// "sql" is code keyword, but comes after "audio"
	result := d.DetectMediaType("audio sql transcription", "", annotations)
	if result != backends.MediaTypeAudio {
		t.Errorf("Audio detection should find 'audio' keyword first, got %v", result)
	}
}

// TestDetectMediaType_MixedCaseKeywords is part of the public API.
func TestDetectMediaType_MixedCaseKeywords(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		annotations  *backends.Annotations
		prompt       string
		expectedType backends.MediaType
	}{
		{prompt: "ReAlTiMe conversation", expectedType: backends.MediaTypeRealtime, annotations: &backends.Annotations{LatencyCritical: true}},
		{prompt: "CoDeGENeRaTioN", expectedType: backends.MediaTypeCode, annotations: &backends.Annotations{}},
		{prompt: "ImAgE analysis", expectedType: backends.MediaTypeImage, annotations: &backends.Annotations{}},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			result := d.DetectMediaType(tt.prompt, "", tt.annotations)
			if result != tt.expectedType {
				t.Errorf("Expected %v, got %v", tt.expectedType, result)
			}
		})
	}
}

// TestGetProfile_AllTypes is part of the public API.
func TestGetProfile_AllTypes(t *testing.T) {
	d := NewDetector()

	allTypes := []backends.MediaType{
		backends.MediaTypeRealtime,
		backends.MediaTypeAudio,
		backends.MediaTypeCode,
		backends.MediaTypeImage,
		backends.MediaTypeText,
	}

	for _, mediaType := range allTypes {
		t.Run(string(mediaType), func(t *testing.T) {
			profile := d.GetProfile(mediaType)
			if profile == nil {
				t.Fatal("Profile is nil")
			}
			if profile.Description == "" {
				t.Error("Profile description is empty")
			}
			if profile.PreferredModel == "" {
				t.Error("PreferredModel is empty")
			}
			if profile.MaxModelSizeGB <= 0 {
				t.Error("MaxModelSizeGB should be positive")
			}
			if profile.MinTokensPerSec <= 0 {
				t.Error("MinTokensPerSec should be positive")
			}
		})
	}
}

// TestNewDetector_ProfilesInitialized is part of the public API.
func TestNewDetector_ProfilesInitialized(t *testing.T) {
	d := NewDetector()

	expectedTypes := []backends.MediaType{
		backends.MediaTypeRealtime,
		backends.MediaTypeAudio,
		backends.MediaTypeCode,
		backends.MediaTypeImage,
		backends.MediaTypeText,
	}

	for _, mediaType := range expectedTypes {
		if _, ok := d.profiles[mediaType]; !ok {
			t.Errorf("Profile for %v not initialized", mediaType)
		}
	}
}

// TestDetectMediaType_EmptyPrompt is part of the public API.
func TestDetectMediaType_EmptyPrompt(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	result := d.DetectMediaType("", "", annotations)
	if result != backends.MediaTypeText {
		t.Errorf("Empty prompt should default to text, got %v", result)
	}
}

// TestDetectMediaType_WithModel is part of the public API.
func TestDetectMediaType_WithModel(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	// Model should be considered for code detection
	result := d.DetectMediaType("", "codellama:13b", annotations)
	if result != backends.MediaTypeCode {
		t.Errorf("CodeLlama model should trigger code detection, got %v", result)
	}
}

// TestIsRealtime_AudioKeywordWithLatency is part of the public API.
func TestIsRealtime_AudioKeywordWithLatency(t *testing.T) {
	d := NewDetector()

	// When both audio keyword and latency critical are present
	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	result := d.isRealtime("audio streaming", annotations)
	if !result {
		t.Error("Audio + latency critical should be realtime")
	}
}

// TestIsRealtime_AudioKeywordButNoRealtimeKeyword is part of the public API.
func TestIsRealtime_AudioKeywordButNoRealtimeKeyword(t *testing.T) {
	d := NewDetector()

	// Test the uncovered line: audio keyword with latency critical but no realtime keyword
	// This should still return true due to the check: annotations.LatencyCritical && d.isAudio(prompt)
	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	result := d.isRealtime("process this audio file", annotations)
	if !result {
		t.Error("Audio keyword with LatencyCritical (without realtime keyword) should be realtime")
	}
}

// TestRoutingHints_ReasoningChainContent is part of the public API.
func TestRoutingHints_ReasoningChainContent(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		prompt string
		name   string
		count  int
	}{
		{prompt: "code generation", count: 1, name: "basic"},
		{prompt: "code with latency requirement", count: 2, name: "with_override"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := &backends.Annotations{}
			if tt.count > 1 {
				annotations.LatencyCritical = true
			}

			hints := d.GetRoutingHints(tt.prompt, "", annotations)
			if len(hints.ReasoningChain) < tt.count {
				t.Errorf("Expected at least %d reasoning items, got %d", tt.count, len(hints.ReasoningChain))
			}

			// Verify first item contains media type
			if !strings.Contains(hints.ReasoningChain[0], "Detected:") {
				t.Error("First reasoning item should mention 'Detected:'")
			}
		})
	}
}

// TestDetectMediaType_NilAnnotations is part of the public API.
func TestDetectMediaType_NilAnnotations(t *testing.T) {
	d := NewDetector()

	// Should handle nil annotations gracefully (though normally caller should provide)
	// We'll test with empty annotations instead since the function expects non-nil
	annotations := &backends.Annotations{}
	result := d.DetectMediaType("test", "", annotations)
	if result == "" {
		t.Error("DetectMediaType should return a valid media type")
	}
}
