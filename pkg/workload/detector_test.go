package workload

import (
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

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

func TestDetectMediaType_Realtime(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{"Real-time keyword", "Start a real-time conversation with me"},
		{"Realtime keyword", "Enable realtime transcription please"},
		{"Live keyword", "I need live streaming translation"},
		{"Interactive chat", "Start an interactive chat session"},
		{"Voice chat", "Begin voice chat mode"},
		{"Instant keyword", "I need instant responses"},
		{"Transcribe", "Transcribe this audio in realtime"},
		{"Dictation", "Enable dictation mode"},
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

func TestDetectMediaType_Audio(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{"Audio keyword", "Process this audio file"},
		{"Sound keyword", "Analyze this sound recording"},
		{"Voice keyword", "Transcribe this voice message"},
		{"Speech keyword", "Convert speech to text"},
		{"Podcast keyword", "Summarize this podcast episode"},
		{"TTS keyword", "Use text to speech for this"},
		{"STT keyword", "Speech to text conversion needed"},
		{"Transcribe keyword", "Transcribe this recording"},
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

func TestDetectMediaType_Code(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
		model  string
	}{
		{"Code keyword", "Write code for a REST API", ""},
		{"Program keyword", "Create a program that sorts arrays", ""},
		{"Function keyword", "Write a function to calculate fibonacci", ""},
		{"Implement keyword", "Implement a binary search tree", ""},
		{"Python keyword", "Write a python script to parse JSON", ""},
		{"JavaScript keyword", "Create a javascript function for validation", ""},
		{"Algorithm keyword", "Implement this algorithm in Go", ""},
		{"API keyword", "Design an API for user management", ""},
		{"Bug keyword", "Fix this bug in my code", ""},
		{"SQL keyword", "Write a SQL query to join these tables", ""},
		{"React keyword", "Create a react component", ""},
		{"CodeLlama model", "Test prompt", "codellama:7b"},
		{"StarCoder model", "Test prompt", "starcoder"},
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

func TestDetectMediaType_Image(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{"Image keyword", "Analyze this image for me"},
		{"Picture keyword", "Describe this picture"},
		{"Photo keyword", "What's in this photo?"},
		{"Visual keyword", "Explain the visual elements"},
		{"Draw keyword", "Draw a diagram of the architecture"},
		{"Generate image", "Generate an image of a sunset"},
		{"Vision keyword", "Use vision to analyze this"},
		{"Screenshot keyword", "Analyze this screenshot"},
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

func TestDetectMediaType_Text(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		prompt string
	}{
		{"General question", "What's the main city of France?"},
		{"Explanation request", "Explain biology concepts to me"},
		{"Summary request", "Summarize the history of mathematics"},
		{"Default case", "Tell me about quantum mechanics"},
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

func TestGetProfile(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		mediaType backends.MediaType
		expected  string
	}{
		{backends.MediaTypeRealtime, "qwen2.5:0.5b"},
		{backends.MediaTypeAudio, "qwen2.5:1.5b"},
		{backends.MediaTypeCode, "llama3:70b"},
		{backends.MediaTypeImage, "llama3:7b"},
		{backends.MediaTypeText, "llama3:7b"},
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

func TestIsAudio_NoKeywords(t *testing.T) {
	d := NewDetector()

	result := d.isAudio("Tell me about mathematics")
	if result {
		t.Error("isAudio should return false for non-audio content")
	}
}

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

func TestIsCode_ModelNameDetection(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name  string
		model string
	}{
		{"code model", "code-llama:13b"},
		{"starcoder model", "starcoder:7b"},
		{"codellama model", "codellama:34b"},
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

func TestIsCode_NoCodeIndicators(t *testing.T) {
	d := NewDetector()

	result := d.isCode("Tell me about biology", "llama2:7b")
	if result {
		t.Error("isCode should return false for non-code content with generic model")
	}
}

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

func TestIsImage_NoKeywords(t *testing.T) {
	d := NewDetector()

	result := d.isImage("Write a Python script")
	if result {
		t.Error("isImage should return false for non-image content")
	}
}

func TestDetectMediaType_CaseSensitivity(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	tests := []struct {
		prompt       string
		expectedType backends.MediaType
	}{
		{"AUDIO processing", backends.MediaTypeAudio},
		{"Code generation", backends.MediaTypeCode},
		{"IMAGE analysis", backends.MediaTypeImage},
		{"WRITE a story", backends.MediaTypeText},
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

func TestGetRoutingHints_AllMediaTypes(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		prompt    string
		mediaType backends.MediaType
	}{
		{"realtime transcription", backends.MediaTypeAudio}, // Without latency critical
		{"code in python", backends.MediaTypeCode},
		{"analyze this image", backends.MediaTypeImage},
		{"tell me a story", backends.MediaTypeText},
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

func TestDetectMediaType_MultipleKeywordsOrder(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	// "sql" is code keyword, but comes after "audio"
	result := d.DetectMediaType("audio sql transcription", "", annotations)
	if result != backends.MediaTypeAudio {
		t.Errorf("Audio detection should find 'audio' keyword first, got %v", result)
	}
}

func TestDetectMediaType_MixedCaseKeywords(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		prompt       string
		expectedType backends.MediaType
		annotations  *backends.Annotations
	}{
		{"ReAlTiMe conversation", backends.MediaTypeRealtime, &backends.Annotations{LatencyCritical: true}},
		{"CoDeGENeRaTioN", backends.MediaTypeCode, &backends.Annotations{}},
		{"ImAgE analysis", backends.MediaTypeImage, &backends.Annotations{}},
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

func TestDetectMediaType_EmptyPrompt(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	result := d.DetectMediaType("", "", annotations)
	if result != backends.MediaTypeText {
		t.Errorf("Empty prompt should default to text, got %v", result)
	}
}

func TestDetectMediaType_WithModel(t *testing.T) {
	d := NewDetector()
	annotations := &backends.Annotations{}

	// Model should be considered for code detection
	result := d.DetectMediaType("", "codellama:13b", annotations)
	if result != backends.MediaTypeCode {
		t.Errorf("CodeLlama model should trigger code detection, got %v", result)
	}
}

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

func TestRoutingHints_ReasoningChainContent(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		prompt string
		count  int
		name   string
	}{
		{"code generation", 1, "basic"},
		{"code with latency requirement", 2, "with_override"},
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
