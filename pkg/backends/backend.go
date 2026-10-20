// Package backends defines the backend interface shared by model providers.
package backends

import (
	"context"
	"io"
)

// Backend represents a compute backend (Ollama, OpenAI, etc.)
type Backend interface {
	// Identification
	ID() string
	Type() string // "ollama", "openai", "vectordb", etc.
	Name() string
	Hardware() string // "npu", "igpu", "nvidia", "cpu", "cloud"

	// Health
	IsHealthy() bool
	HealthCheck(ctx context.Context) error

	// Characteristics
	PowerWatts() float64
	AvgLatencyMs() int32
	Priority() int

	// Capabilities
	SupportsGenerate() bool
	SupportsStream() bool
	SupportsEmbed() bool
	SupportsAudioToText() bool
	SupportsTextToAudio() bool
	SupportsImageToText() bool
	SupportsTextToImage() bool
	SupportsVideoToText() bool
	SupportsTextToVideo() bool
	ListModels(ctx context.Context) ([]string, error)

	// Model capabilities
	SupportsModel(modelName string) bool
	GetMaxModelSizeGB() int
	GetSupportedModelPatterns() []string
	GetPreferredModels() []string

	// Operations - Text
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	GenerateStream(ctx context.Context, req *GenerateRequest) (StreamReader, error)
	Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)

	// Operations - Audio (streaming for low latency)
	TranscribeAudio(ctx context.Context, req *TranscribeRequest) (*TranscribeResponse, error)
	TranscribeAudioStream(ctx context.Context, req *TranscribeRequest) (AudioStreamReader, error)
	SynthesizeSpeech(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error)
	SynthesizeSpeechStream(ctx context.Context, req *SynthesizeRequest) (AudioStreamWriter, error)

	// Operations - Image
	AnalyzeImage(ctx context.Context, req *ImageAnalysisRequest) (*ImageAnalysisResponse, error)
	GenerateImage(ctx context.Context, req *ImageGenRequest) (*ImageGenResponse, error)
	GenerateImageStream(ctx context.Context, req *ImageGenRequest) (ImageStreamReader, error)

	// Operations - Video
	AnalyzeVideo(ctx context.Context, req *VideoAnalysisRequest) (*VideoAnalysisResponse, error)
	AnalyzeVideoStream(ctx context.Context, req *VideoAnalysisRequest) (VideoStreamReader, error)
	GenerateVideo(ctx context.Context, req *VideoGenRequest) (*VideoGenResponse, error)
	GenerateVideoStream(ctx context.Context, req *VideoGenRequest) (VideoStreamReader, error)

	// Metrics
	UpdateMetrics(latencyMs int32, success bool)
	GetMetrics() *BackendMetrics

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MediaType represents the type of workload
type MediaType string

// MediaTypeText is an exported const.
const (
	MediaTypeText     MediaType = "text"     // Text generation/chat
	MediaTypeCode     MediaType = "code"     // Code generation
	MediaTypeAudio    MediaType = "audio"    // Audio processing (transcription, TTS)
	MediaTypeImage    MediaType = "image"    // Image generation/analysis
	MediaTypeVideo    MediaType = "video"    // Video processing (transcription, generation)
	MediaTypeRealtime MediaType = "realtime" // Real-time interactive (audio/chat)
	MediaTypeAuto     MediaType = "auto"     // Auto-detect from prompt
)

// Priority levels for request prioritization
type Priority int

// PriorityBestEffort is an exported const.
const (
	PriorityBestEffort Priority = 0 // Batch jobs, non-critical
	PriorityNormal     Priority = 1 // Default priority
	PriorityHigh       Priority = 2 // Important but not realtime
	PriorityCritical   Priority = 3 // Voice, realtime streams
)

// Annotations for job routing
type Annotations struct {
	Custom                map[string]string
	Target                string
	MediaType             MediaType
	RequestID             string
	Priority              Priority
	DeadlineMs            int64
	MaxLatencyMs          int32
	MaxPowerWatts         int32
	LatencyCritical       bool
	PreferPowerEfficiency bool
	CacheEnabled          bool
}

// GenerateRequest for text generation
type GenerateRequest struct {
	Options *GenerationOptions
	Prompt  string
	Model   string
}

// GenerationOptions for inference
type GenerationOptions struct {
	Stop          []string
	MaxTokens     int32
	Temperature   float32
	TopP          float32
	TopK          int32
	ContextLength int32
}

// GenerateResponse from backend
type GenerateResponse struct {
	Stats    *GenerationStats
	Response string
}

// GenerationStats for a generation
type GenerationStats struct {
	TimeToFirstTokenMs int32
	TotalTimeMs        int32
	TokensGenerated    int32
	TokensPerSecond    float32
	EnergyWh           float32
}

// StreamReader for streaming responses
type StreamReader interface {
	Recv() (*StreamChunk, error)
	io.Closer
}

// StreamChunk represents a chunk of streamed response
type StreamChunk struct {
	Stats *GenerationStats
	Token string
	Done  bool
}

// EmbedRequest for embeddings
type EmbedRequest struct {
	Text  string
	Model string
}

// EmbedResponse with embeddings
type EmbedResponse struct {
	Stats     *GenerationStats
	Embedding []float32
}

// BackendMetrics tracks backend performance
type BackendMetrics struct {
	LoadedModels      []string
	RequestCount      int64
	SuccessCount      int64
	ErrorCount        int64
	TotalLatencyMs    int64
	AvgLatencyMs      int32
	RequestsPerMinute int32
	ErrorRate         float32
}

// ModelCapability defines what models a backend can run
type ModelCapability struct {
	SupportedModelPatterns []string
	PreferredModels        []string
	ExcludedPatterns       []string
	MaxModelSizeGB         int
}

// BackendConfig common configuration for all backends
type BackendConfig struct {
	ModelCapability *ModelCapability
	ID              string
	Type            string
	Name            string
	Hardware        string
	PowerWatts      float64
	Priority        int
	AvgLatencyMs    int32
	MaxTokensPerSec int32
	Enabled         bool
}

// ============================================================
// Audio Types - Optimized for low-latency streaming
// ============================================================

// AudioFormat defines audio encoding format
type AudioFormat string

// AudioFormatPCM is an exported const.
const (
	AudioFormatPCM  AudioFormat = "pcm"  // Raw PCM (fastest, no encoding overhead)
	AudioFormatWAV  AudioFormat = "wav"  // WAV container
	AudioFormatMP3  AudioFormat = "mp3"  // Compressed MP3
	AudioFormatOPUS AudioFormat = "opus" // Low-latency OPUS codec
	AudioFormatFLAC AudioFormat = "flac" // Lossless compression
)

// TranscribeRequest for speech-to-text
type TranscribeRequest struct {
	AudioStream      io.Reader
	Options          map[string]string
	Model            string
	Language         string
	Format           AudioFormat
	AudioData        []byte
	SampleRate       int32
	Channels         int32
	EnableVAD        bool
	EnableTimestamps bool
}

// TranscribeResponse from speech recognition
type TranscribeResponse struct {
	Stats      *GenerationStats
	Text       string
	Language   string
	Segments   []TranscriptSegment
	Confidence float32
}

// TranscriptSegment represents a timed segment of transcription
type TranscriptSegment struct {
	Text       string  // Segment text
	StartMs    int64   // Start time in milliseconds
	EndMs      int64   // End time in milliseconds
	Confidence float32 // Segment confidence (0-1)
}

// AudioStreamReader for streaming transcription results
type AudioStreamReader interface {
	Recv() (*TranscriptChunk, error)
	io.Closer
}

// TranscriptChunk represents a chunk of streaming transcription
type TranscriptChunk struct {
	Text       string
	StartMs    int64
	EndMs      int64
	Confidence float32
	IsFinal    bool
	Done       bool
}

// SynthesizeRequest for text-to-speech
type SynthesizeRequest struct {
	Options    map[string]string
	Text       string
	Model      string
	Voice      string
	Language   string
	Format     AudioFormat
	SampleRate int32
	Speed      float32
	Pitch      float32
}

// SynthesizeResponse from TTS
type SynthesizeResponse struct {
	Stats      *GenerationStats
	Format     AudioFormat
	AudioData  []byte
	SampleRate int32
	Duration   int32
}

// AudioStreamWriter for streaming TTS output
type AudioStreamWriter interface {
	Recv() (*AudioChunk, error)
	io.Closer
}

// AudioChunk represents a chunk of streaming audio
type AudioChunk struct {
	Format     AudioFormat
	Data       []byte
	SampleRate int32
	DurationMs int32
	Done       bool
}

// ============================================================
// Image Types
// ============================================================

// ImageFormat defines image encoding format
type ImageFormat string

// ImageFormatPNG is an exported const.
const (
	ImageFormatPNG  ImageFormat = "png"  // PNG (lossless)
	ImageFormatJPEG ImageFormat = "jpeg" // JPEG (lossy)
	ImageFormatWEBP ImageFormat = "webp" // WebP (modern, efficient)
)

// ImageAnalysisRequest for image-to-text (OCR, captioning)
type ImageAnalysisRequest struct {
	Options   map[string]string
	ImageURL  string
	Model     string
	Task      string
	Prompt    string
	Format    ImageFormat
	ImageData []byte
}

// ImageAnalysisResponse from image analysis
type ImageAnalysisResponse struct {
	Stats      *GenerationStats
	Text       string
	Detections []Detection
	Confidence float32
}

// Detection represents detected object or region in image
type Detection struct {
	Label      string  // Object label
	Confidence float32 // Detection confidence
	BBoxX      int32   // Bounding box X
	BBoxY      int32   // Bounding box Y
	BBoxWidth  int32   // Bounding box width
	BBoxHeight int32   // Bounding box height
}

// ImageGenRequest for text-to-image generation
type ImageGenRequest struct {
	Options        map[string]string
	Prompt         string
	NegativePrompt string
	Model          string
	Format         ImageFormat
	Seed           int64
	Width          int32
	Height         int32
	Steps          int32
	GuidanceScale  float32
	BatchSize      int32
}

// ImageGenResponse from image generation
type ImageGenResponse struct {
	Stats  *GenerationStats
	Images []GeneratedImage
}

// GeneratedImage represents one generated image
type GeneratedImage struct {
	Format    ImageFormat
	ImageData []byte
	Seed      int64
	Width     int32
	Height    int32
}

// ImageStreamReader for progressive image generation
type ImageStreamReader interface {
	Recv() (*ImageChunk, error)
	io.Closer
}

// ImageChunk represents progressive image generation update
type ImageChunk struct {
	ImageData   []byte  // Partial or complete image
	Progress    float32 // Generation progress (0-1)
	CurrentStep int32   // Current diffusion step
	TotalSteps  int32   // Total diffusion steps
	Done        bool    // True if generation complete
}

// ============================================================
// Video Types
// ============================================================

// VideoFormat defines video encoding format
type VideoFormat string

// VideoFormatMP4 is an exported const.
const (
	VideoFormatMP4  VideoFormat = "mp4"  // MP4 container (H.264)
	VideoFormatWEBM VideoFormat = "webm" // WebM container (VP9)
	VideoFormatGIF  VideoFormat = "gif"  // Animated GIF
)

// VideoAnalysisRequest for video-to-text (transcription, captioning)
type VideoAnalysisRequest struct {
	VideoStream io.Reader
	Options     map[string]string
	VideoURL    string
	Model       string
	Task        string
	Format      VideoFormat
	VideoData   []byte
}

// VideoAnalysisResponse from video analysis
type VideoAnalysisResponse struct {
	Stats    *GenerationStats
	Text     string
	Captions []VideoCaption
	Objects  []TrackedObject
}

// VideoCaption represents time-aligned caption
type VideoCaption struct {
	Text       string  // Caption text
	StartMs    int64   // Start time in milliseconds
	EndMs      int64   // End time in milliseconds
	Confidence float32 // Caption confidence
}

// TrackedObject represents tracked object in video
type TrackedObject struct {
	Label      string
	Frames     []ObjectFrame
	TrackID    int32
	Confidence float32
}

// ObjectFrame represents object location in a specific frame
type ObjectFrame struct {
	FrameNum   int32 // Frame number
	TimeMs     int64 // Time in milliseconds
	BBoxX      int32 // Bounding box X
	BBoxY      int32 // Bounding box Y
	BBoxWidth  int32 // Bounding box width
	BBoxHeight int32 // Bounding box height
}

// VideoStreamReader for streaming video analysis
type VideoStreamReader interface {
	Recv() (*VideoChunk, error)
	io.Closer
}

// VideoChunk represents streaming video analysis or generation
type VideoChunk struct {
	Text      string
	FrameData []byte
	TimeMs    int64
	Progress  float32
	FrameNum  int32
	Done      bool
}

// VideoGenRequest for text-to-video generation
type VideoGenRequest struct {
	Options  map[string]string
	Prompt   string
	Model    string
	Format   VideoFormat
	Seed     int64
	Width    int32
	Height   int32
	FPS      int32
	Duration int32
}

// VideoGenResponse from video generation
type VideoGenResponse struct {
	Stats     *GenerationStats
	Format    VideoFormat
	VideoData []byte
	Width     int32
	Height    int32
	FPS       int32
	Duration  int32
}
