package workload

import (
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// WorkloadProfile defines routing preferences for a workload type
type WorkloadProfile struct {
	MediaType         backends.MediaType
	PreferLowLatency  bool   // Prefer low-latency backends
	PreferLowPower    bool   // Prefer power-efficient backends
	PreferredModel    string // Recommended model for this workload
	MaxModelSizeGB    int    // Maximum model size needed
	MinTokensPerSec   int    // Minimum throughput needed
	Description       string
}

// Detector analyzes prompts to determine workload type
type Detector struct {
	profiles map[backends.MediaType]*WorkloadProfile
}

// NewDetector creates a new workload detector
func NewDetector() *Detector {
	return &Detector{
		profiles: map[backends.MediaType]*WorkloadProfile{
			backends.MediaTypeRealtime: {
				MediaType:        backends.MediaTypeRealtime,
				PreferLowLatency: true,
				PreferLowPower:   true, // Realtime often runs continuously
				PreferredModel:   "qwen2.5:0.5b",
				MaxModelSizeGB:   2,
				MinTokensPerSec:  20, // Need fast response
				Description:      "Real-time audio/chat - NPU optimized",
			},
			backends.MediaTypeAudio: {
				MediaType:        backends.MediaTypeAudio,
				PreferLowLatency: true,
				PreferLowPower:   true,
				PreferredModel:   "qwen2.5:1.5b",
				MaxModelSizeGB:   3,
				MinTokensPerSec:  15,
				Description:      "Audio processing - small models work well",
			},
			backends.MediaTypeCode: {
				MediaType:        backends.MediaTypeCode,
				PreferLowLatency: false,
				PreferLowPower:   false,
				PreferredModel:   "llama3:70b", // Code benefits from larger models
				MaxModelSizeGB:   80,
				MinTokensPerSec:  10,
				Description:      "Code generation - benefits from larger models",
			},
			backends.MediaTypeImage: {
				MediaType:        backends.MediaTypeImage,
				PreferLowLatency: false,
				PreferLowPower:   false,
				PreferredModel:   "llama3:7b",
				MaxModelSizeGB:   10,
				MinTokensPerSec:  10,
				Description:      "Image analysis - medium models",
			},
			backends.MediaTypeText: {
				MediaType:        backends.MediaTypeText,
				PreferLowLatency: false,
				PreferLowPower:   false,
				PreferredModel:   "llama3:7b",
				MaxModelSizeGB:   10,
				MinTokensPerSec:  15,
				Description:      "General text - balanced approach",
			},
		},
	}
}

// DetectMediaType analyzes prompt to determine workload type
func (d *Detector) DetectMediaType(prompt string, model string, annotations *backends.Annotations) backends.MediaType {
	// If explicitly set, use it
	if annotations.MediaType != "" && annotations.MediaType != backends.MediaTypeAuto {
		return annotations.MediaType
	}

	promptLower := strings.ToLower(prompt)

	// Realtime detection (highest priority)
	if d.isRealtime(promptLower, annotations) {
		return backends.MediaTypeRealtime
	}

	// Audio detection
	if d.isAudio(promptLower) {
		return backends.MediaTypeAudio
	}

	// Code detection
	if d.isCode(promptLower, model) {
		return backends.MediaTypeCode
	}

	// Image detection
	if d.isImage(promptLower) {
		return backends.MediaTypeImage
	}

	// Default to text
	return backends.MediaTypeText
}

// isRealtime checks if this is a realtime workload
func (d *Detector) isRealtime(prompt string, annotations *backends.Annotations) bool {
	// Realtime is characterized by:
	// 1. Low latency requirement
	// 2. Audio/voice/chat keywords
	// 3. Interactive nature

	if !annotations.LatencyCritical {
		return false // Realtime always needs low latency
	}

	realtimeKeywords := []string{
		"realtime", "real-time", "real time",
		"live", "streaming",
		"interactive chat", "voice chat",
		"instant", "immediate",
		"continuous", "ongoing",
		"transcribe", "transcription",
		"dictate", "dictation",
	}

	for _, keyword := range realtimeKeywords {
		if strings.Contains(prompt, keyword) {
			return true
		}
	}

	// Check for audio + latency critical combination
	if annotations.LatencyCritical && d.isAudio(prompt) {
		return true
	}

	return false
}

// isAudio checks if this is audio processing
func (d *Detector) isAudio(prompt string) bool {
	audioKeywords := []string{
		"audio", "sound", "voice", "speech",
		"listen", "hear", "spoken",
		"podcast", "recording",
		"tts", "text to speech", "text-to-speech",
		"stt", "speech to text", "speech-to-text",
		"transcribe", "transcription",
	}

	for _, keyword := range audioKeywords {
		if strings.Contains(prompt, keyword) {
			return true
		}
	}

	return false
}

// isCode checks if this is code generation/analysis
func (d *Detector) isCode(prompt string, model string) bool {
	// Check model name first (code-specific models)
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "code") ||
	   strings.Contains(modelLower, "starcoder") ||
	   strings.Contains(modelLower, "codellama") {
		return true
	}

	codeKeywords := []string{
		"code", "program", "function", "class",
		"implement", "refactor", "debug",
		"python", "javascript", "java", "go", "rust", "c++",
		"algorithm", "data structure",
		"api", "endpoint", "server",
		"bug", "error", "exception",
		"test", "unit test",
		"sql", "query", "database",
		"html", "css", "react", "vue",
	}

	for _, keyword := range codeKeywords {
		if strings.Contains(prompt, keyword) {
			return true
		}
	}

	return false
}

// isImage checks if this is image processing
func (d *Detector) isImage(prompt string) bool {
	imageKeywords := []string{
		"image", "picture", "photo", "visual",
		"draw", "generate image", "create image",
		"analyze image", "describe image",
		"vision", "see", "look at",
		"screenshot", "diagram",
	}

	for _, keyword := range imageKeywords {
		if strings.Contains(prompt, keyword) {
			return true
		}
	}

	return false
}

// GetProfile returns the workload profile for a media type
func (d *Detector) GetProfile(mediaType backends.MediaType) *WorkloadProfile {
	if profile, ok := d.profiles[mediaType]; ok {
		return profile
	}
	return d.profiles[backends.MediaTypeText] // Default
}

// GetRoutingHints returns routing hints based on detected workload
func (d *Detector) GetRoutingHints(prompt string, model string, annotations *backends.Annotations) *RoutingHints {
	mediaType := d.DetectMediaType(prompt, model, annotations)
	profile := d.GetProfile(mediaType)

	hints := &RoutingHints{
		DetectedMediaType:  mediaType,
		Profile:            profile,
		PreferredModel:     profile.PreferredModel,
		PreferLowLatency:   profile.PreferLowLatency,
		PreferLowPower:     profile.PreferLowPower,
		MaxModelSizeGB:     profile.MaxModelSizeGB,
		MinTokensPerSec:    profile.MinTokensPerSec,
		ReasoningChain:     []string{},
	}

	// Add reasoning
	hints.ReasoningChain = append(hints.ReasoningChain,
		"Detected: "+string(mediaType)+" ("+profile.Description+")")

	// Override from annotations if stronger signal
	if annotations.LatencyCritical && !profile.PreferLowLatency {
		hints.PreferLowLatency = true
		hints.ReasoningChain = append(hints.ReasoningChain,
			"Annotation override: latency_critical=true")
	}

	if annotations.PreferPowerEfficiency && !profile.PreferLowPower {
		hints.PreferLowPower = true
		hints.ReasoningChain = append(hints.ReasoningChain,
			"Annotation override: prefer_power_efficiency=true")
	}

	return hints
}

// RoutingHints contains recommendations for routing
type RoutingHints struct {
	DetectedMediaType backends.MediaType
	Profile           *WorkloadProfile
	PreferredModel    string
	PreferLowLatency  bool
	PreferLowPower    bool
	MaxModelSizeGB    int
	MinTokensPerSec   int
	ReasoningChain    []string // Explanation of decisions
}
