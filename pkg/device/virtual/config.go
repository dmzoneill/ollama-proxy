package virtual

import "fmt"

// Config holds virtual device configuration
type Config struct {
	BackendModels map[string]BackendModelConfig `yaml:"backend_models"`
	Audio         AudioConfig                   `yaml:"audio"`
	Video         VideoConfig                   `yaml:"video"`
	Enabled       bool                          `yaml:"enabled"`
}

// AudioConfig holds audio device configuration
type AudioConfig struct {
	Processing       AudioProcessingConfig `yaml:"processing"`
	Microphone       MicrophoneConfig      `yaml:"microphone"`
	Speaker          SpeakerConfig         `yaml:"speaker"`
	Enabled          bool                  `yaml:"enabled"`
	AutoDetectSystem bool                  `yaml:"auto_detect_system"`
	AllowFallback    bool                  `yaml:"allow_fallback"`
}

// MicrophoneConfig holds microphone device configuration
type MicrophoneConfig struct {
	Format     string `yaml:"format"`
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	Enabled    bool   `yaml:"enabled"`
}

// SpeakerConfig holds speaker device configuration
type SpeakerConfig struct {
	Format     string `yaml:"format"`
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	Enabled    bool   `yaml:"enabled"`
}

// AudioProcessingConfig holds audio processing configuration
type AudioProcessingConfig struct {
	TranscriptionModel string `yaml:"transcription_model"`
	TTSModel           string `yaml:"tts_model"`
	NoiseCancellation  bool   `yaml:"noise_cancellation"`
}

// VideoConfig holds video device configuration
type VideoConfig struct {
	Camera  CameraConfig `yaml:"camera"`
	Enabled bool         `yaml:"enabled"`
}

// CameraConfig holds camera device configuration
type CameraConfig struct {
	Format        string                 `yaml:"format"`
	DeviceNumbers []int                  `yaml:"device_numbers"`
	Width         int                    `yaml:"width"`
	Height        int                    `yaml:"height"`
	FPS           int                    `yaml:"fps"`
	Processing    CameraProcessingConfig `yaml:"processing"`
	Enabled       bool                   `yaml:"enabled"`
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
				Format:     strS16le,
			},
			Speaker: SpeakerConfig{
				Enabled:    true,
				SampleRate: 22050, // 22kHz for TTS output
				Channels:   2,     // Stereo for output
				Format:     strS16le,
			},
			Processing: AudioProcessingConfig{
				NoiseCancellation:  true,
				TranscriptionModel: strWhisperTiny,
				TTSModel:           strPiperTtsFast,
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
				STTModel: strWhisperTiny,
				LLMModel: "qwen2.5:0.5b",
				TTSModel: strPiperTtsFast,
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
				STTModel: strWhisperTiny,
				LLMModel: "llama3:7b",
				TTSModel: strPiperTtsFast,
			},
		},
	}
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // Nothing to validate if disabled
	}
	if err := c.validateAudio(); err != nil {
		return err
	}
	return c.validateVideo()
}

func (c *Config) validateAudio() error {
	if !c.Audio.Enabled {
		return nil
	}
	if err := validatePCM(c.Audio.Microphone.Enabled, c.Audio.Microphone.SampleRate, c.Audio.Microphone.Channels); err != nil {
		return err
	}
	return validatePCM(c.Audio.Speaker.Enabled, c.Audio.Speaker.SampleRate, c.Audio.Speaker.Channels)
}

func validatePCM(enabled bool, sampleRate, channels int) error {
	if !enabled {
		return nil
	}
	if sampleRate <= 0 {
		return ErrInvalidSampleRate
	}
	if channels <= 0 || channels > 2 {
		return ErrInvalidChannels
	}
	return nil
}

func (c *Config) validateVideo() error {
	if !c.Video.Enabled || !c.Video.Camera.Enabled {
		return nil
	}
	if c.Video.Camera.Width <= 0 || c.Video.Camera.Height <= 0 {
		return ErrInvalidResolution
	}
	if c.Video.Camera.FPS <= 0 || c.Video.Camera.FPS > 120 {
		return ErrInvalidFPS
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
	ErrInvalidSampleRate = fmt.Errorf("invalid sample rate")
	ErrInvalidChannels   = fmt.Errorf("invalid channel count")
	ErrInvalidResolution = fmt.Errorf("invalid resolution")
	ErrInvalidFPS        = fmt.Errorf("invalid FPS")
)
