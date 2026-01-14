package virtual

import "fmt"

// Config holds virtual device configuration
type Config struct {
	Enabled bool        `yaml:"enabled"`
	Audio   AudioConfig `yaml:"audio"`
	Video   VideoConfig `yaml:"video"`

	// Backend-specific model configuration
	BackendModels map[string]BackendModelConfig `yaml:"backend_models"`
}

// AudioConfig holds audio device configuration
type AudioConfig struct {
	Enabled         bool                 `yaml:"enabled"`
	AutoDetectSystem bool                `yaml:"auto_detect_system"`
	AllowFallback   bool                 `yaml:"allow_fallback"` // Allow startup without audio system
	Microphone      MicrophoneConfig     `yaml:"microphone"`
	Speaker         SpeakerConfig        `yaml:"speaker"`
	Processing      AudioProcessingConfig `yaml:"processing"`
}

// MicrophoneConfig holds microphone device configuration
type MicrophoneConfig struct {
	Enabled    bool   `yaml:"enabled"`
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	Format     string `yaml:"format"`
}

// SpeakerConfig holds speaker device configuration
type SpeakerConfig struct {
	Enabled    bool   `yaml:"enabled"`
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	Format     string `yaml:"format"`
}

// AudioProcessingConfig holds audio processing configuration
type AudioProcessingConfig struct {
	NoiseCancellation  bool   `yaml:"noise_cancellation"`
	TranscriptionModel string `yaml:"transcription_model"` // e.g., "whisper-tiny"
	TTSModel           string `yaml:"tts_model"`           // e.g., "piper-tts-fast"
}

// VideoConfig holds video device configuration
type VideoConfig struct {
	Enabled bool         `yaml:"enabled"`
	Camera  CameraConfig `yaml:"camera"`
}

// CameraConfig holds camera device configuration
type CameraConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Width         int    `yaml:"width"`
	Height        int    `yaml:"height"`
	FPS           int    `yaml:"fps"`
	Format        string `yaml:"format"`
	DeviceNumbers []int  `yaml:"device_numbers"` // /dev/videoN numbers

	Processing CameraProcessingConfig `yaml:"processing"`
}

// CameraProcessingConfig holds camera processing configuration
type CameraProcessingConfig struct {
	BackgroundBlur        bool `yaml:"background_blur"`
	BackgroundReplacement bool `yaml:"background_replacement"`
	FaceDetection         bool `yaml:"face_detection"`
}

// BackendModelConfig holds model configuration for a specific backend
type BackendModelConfig struct {
	STTModel string `yaml:"stt"` // Speech-to-text model
	LLMModel string `yaml:"llm"` // Language model
	TTSModel string `yaml:"tts"` // Text-to-speech model
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Audio: AudioConfig{
			Enabled:          true,
			AutoDetectSystem: true,
			AllowFallback:    true, // Don't fail startup if audio unavailable
			Microphone: MicrophoneConfig{
				Enabled:    true,
				SampleRate: 16000, // 16kHz is standard for speech
				Channels:   1,     // Mono for speech
				Format:     "s16le",
			},
			Speaker: SpeakerConfig{
				Enabled:    true,
				SampleRate: 22050, // 22kHz for TTS output
				Channels:   2,     // Stereo for output
				Format:     "s16le",
			},
			Processing: AudioProcessingConfig{
				NoiseCancellation:  true,
				TranscriptionModel: "whisper-tiny",
				TTSModel:           "piper-tts-fast",
			},
		},
		Video: VideoConfig{
			Enabled: true,
			Camera: CameraConfig{
				Enabled:       true,
				Width:         640,
				Height:        480,
				FPS:           30,
				Format:        "yuyv",
				DeviceNumbers: []int{20, 21, 22, 23}, // /dev/video20-23
				Processing: CameraProcessingConfig{
					BackgroundBlur:        false,
					BackgroundReplacement: false,
					FaceDetection:         false,
				},
			},
		},
		BackendModels: map[string]BackendModelConfig{
			"ollama-npu": {
				STTModel: "whisper-tiny",
				LLMModel: "qwen2.5:0.5b",
				TTSModel: "piper-tts-fast",
			},
			"ollama-igpu": {
				STTModel: "whisper-base",
				LLMModel: "llama3:7b",
				TTSModel: "piper-tts",
			},
			"ollama-nvidia": {
				STTModel: "whisper-medium",
				LLMModel: "llama3:70b",
				TTSModel: "bark-tts",
			},
			"ollama-cpu": {
				STTModel: "whisper-tiny",
				LLMModel: "llama3:7b",
				TTSModel: "piper-tts-fast",
			},
		},
	}
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // Nothing to validate if disabled
	}

	if c.Audio.Enabled {
		if c.Audio.Microphone.Enabled {
			if c.Audio.Microphone.SampleRate <= 0 {
				return ErrInvalidSampleRate
			}
			if c.Audio.Microphone.Channels <= 0 || c.Audio.Microphone.Channels > 2 {
				return ErrInvalidChannels
			}
		}

		if c.Audio.Speaker.Enabled {
			if c.Audio.Speaker.SampleRate <= 0 {
				return ErrInvalidSampleRate
			}
			if c.Audio.Speaker.Channels <= 0 || c.Audio.Speaker.Channels > 2 {
				return ErrInvalidChannels
			}
		}
	}

	if c.Video.Enabled && c.Video.Camera.Enabled {
		if c.Video.Camera.Width <= 0 || c.Video.Camera.Height <= 0 {
			return ErrInvalidResolution
		}
		if c.Video.Camera.FPS <= 0 || c.Video.Camera.FPS > 120 {
			return ErrInvalidFPS
		}
	}

	return nil
}

// GetBackendModels returns model configuration for a backend
func (c *Config) GetBackendModels(backendID string) *BackendModelConfig {
	if models, ok := c.BackendModels[backendID]; ok {
		return &models
	}

	// Return defaults if not configured
	return &BackendModelConfig{
		STTModel: c.Audio.Processing.TranscriptionModel,
		LLMModel: "", // Use backend default
		TTSModel: c.Audio.Processing.TTSModel,
	}
}

// Configuration errors
var (
	ErrInvalidSampleRate  = fmt.Errorf("invalid sample rate")
	ErrInvalidChannels    = fmt.Errorf("invalid channel count")
	ErrInvalidResolution  = fmt.Errorf("invalid resolution")
	ErrInvalidFPS         = fmt.Errorf("invalid FPS")
)
