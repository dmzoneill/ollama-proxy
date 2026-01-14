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

const (
	MediaTypeText     MediaType = "text"      // Text generation/chat
	MediaTypeCode     MediaType = "code"      // Code generation
	MediaTypeAudio    MediaType = "audio"     // Audio processing (transcription, TTS)
	MediaTypeImage    MediaType = "image"     // Image generation/analysis
	MediaTypeVideo    MediaType = "video"     // Video processing (transcription, generation)
	MediaTypeRealtime MediaType = "realtime"  // Real-time interactive (audio/chat)
	MediaTypeAuto     MediaType = "auto"      // Auto-detect from prompt
)

// Priority levels for request prioritization
type Priority int

const (
	PriorityBestEffort Priority = 0 // Batch jobs, non-critical
	PriorityNormal     Priority = 1 // Default priority
	PriorityHigh       Priority = 2 // Important but not realtime
	PriorityCritical   Priority = 3 // Voice, realtime streams
)

// Annotations for job routing
type Annotations struct {
	Target                 string
	LatencyCritical        bool
	PreferPowerEfficiency  bool
	CacheEnabled           bool
	MaxLatencyMs           int32
	MaxPowerWatts          int32
	MediaType              MediaType         // Type of workload

	// Priority queuing
	Priority               Priority          // Request priority level
	RequestID              string            // Unique request ID for tracking
	DeadlineMs             int64             // Absolute deadline (Unix ms)

	Custom                 map[string]string
}

// GenerateRequest for text generation
type GenerateRequest struct {
	Prompt  string
	Model   string
	Options *GenerationOptions
}

// GenerationOptions for inference
type GenerationOptions struct {
	MaxTokens     int32
	Temperature   float32
	TopP          float32
	TopK          int32
	Stop          []string
	ContextLength int32
}

// GenerateResponse from backend
type GenerateResponse struct {
	Response string
	Stats    *GenerationStats
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
	Token string
	Done  bool
	Stats *GenerationStats
}

// EmbedRequest for embeddings
type EmbedRequest struct {
	Text  string
	Model string
}

// EmbedResponse with embeddings
type EmbedResponse struct {
	Embedding []float32
	Stats     *GenerationStats
}

// BackendMetrics tracks backend performance
type BackendMetrics struct {
	RequestCount       int64
	SuccessCount       int64
	ErrorCount         int64
	TotalLatencyMs     int64
	AvgLatencyMs       int32
	RequestsPerMinute  int32
	ErrorRate          float32
	LoadedModels       []string
}

// ModelCapability defines what models a backend can run
type ModelCapability struct {
	MaxModelSizeGB         int      // Maximum model size in GB
	SupportedModelPatterns []string // Glob patterns like "*:0.5b", "llama3:*"
	PreferredModels        []string // Specific models that run well on this backend
	ExcludedPatterns       []string // Models explicitly not supported
}

// BackendConfig common configuration for all backends
type BackendConfig struct {
	ID       string
	Type     string
	Name     string
	Hardware string
	Enabled  bool

	// Characteristics
	PowerWatts       float64
	AvgLatencyMs     int32
	MaxTokensPerSec  int32
	Priority         int

	// Model capabilities
	ModelCapability *ModelCapability
}

// ============================================================
// Audio Types - Optimized for low-latency streaming
// ============================================================

// AudioFormat defines audio encoding format
type AudioFormat string

const (
	AudioFormatPCM  AudioFormat = "pcm"   // Raw PCM (fastest, no encoding overhead)
	AudioFormatWAV  AudioFormat = "wav"   // WAV container
	AudioFormatMP3  AudioFormat = "mp3"   // Compressed MP3
	AudioFormatOPUS AudioFormat = "opus"  // Low-latency OPUS codec
	AudioFormatFLAC AudioFormat = "flac"  // Lossless compression
)

// TranscribeRequest for speech-to-text
type TranscribeRequest struct {
	AudioData      []byte            // Raw audio bytes (use for non-streaming)
	AudioStream    io.Reader         // Streaming audio input (preferred for low latency)
	Model          string            // "whisper-tiny", "whisper-base", etc.
	Language       string            // Optional: "en", "es", "fr", auto-detect if empty
	Format         AudioFormat       // Audio format
	SampleRate     int32             // Sample rate in Hz (e.g., 16000, 48000)
	Channels       int32             // Number of audio channels (1=mono, 2=stereo)
	EnableVAD      bool              // Voice Activity Detection for streaming
	EnableTimestamps bool            // Return word-level timestamps
	Options        map[string]string // Model-specific options
}

// TranscribeResponse from speech recognition
type TranscribeResponse struct {
	Text       string              // Transcribed text
	Language   string              // Detected/specified language
	Confidence float32             // Overall confidence (0-1)
	Segments   []TranscriptSegment // Word/sentence segments with timestamps
	Stats      *GenerationStats    // Processing stats
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
	Text       string  // Partial or complete text
	IsFinal    bool    // True if this is final transcription for segment
	Confidence float32 // Confidence for this chunk
	StartMs    int64   // Segment start time
	EndMs      int64   // Segment end time
	Done       bool    // True if transcription complete
}

// SynthesizeRequest for text-to-speech
type SynthesizeRequest struct {
	Text       string            // Text to synthesize
	Model      string            // "piper", "bark", "coqui-tts", etc.
	Voice      string            // Voice ID or name
	Language   string            // Language code
	Format     AudioFormat       // Output audio format
	SampleRate int32             // Output sample rate (16000, 22050, 48000)
	Speed      float32           // Speech speed multiplier (0.5-2.0, default 1.0)
	Pitch      float32           // Pitch adjustment (-1.0 to 1.0, default 0)
	Options    map[string]string // Model-specific options
}

// SynthesizeResponse from TTS
type SynthesizeResponse struct {
	AudioData  []byte           // Generated audio bytes
	Format     AudioFormat      // Audio format
	SampleRate int32            // Sample rate in Hz
	Duration   int32            // Audio duration in milliseconds
	Stats      *GenerationStats // Processing stats
}

// AudioStreamWriter for streaming TTS output
type AudioStreamWriter interface {
	Recv() (*AudioChunk, error)
	io.Closer
}

// AudioChunk represents a chunk of streaming audio
type AudioChunk struct {
	Data       []byte      // Audio data chunk (optimized for zero-copy)
	Format     AudioFormat // Audio format
	SampleRate int32       // Sample rate
	Done       bool        // True if synthesis complete
	DurationMs int32       // Chunk duration in milliseconds
}

// ============================================================
// Image Types
// ============================================================

// ImageFormat defines image encoding format
type ImageFormat string

const (
	ImageFormatPNG  ImageFormat = "png"  // PNG (lossless)
	ImageFormatJPEG ImageFormat = "jpeg" // JPEG (lossy)
	ImageFormatWEBP ImageFormat = "webp" // WebP (modern, efficient)
)

// ImageAnalysisRequest for image-to-text (OCR, captioning)
type ImageAnalysisRequest struct {
	ImageData   []byte            // Image bytes
	ImageURL    string            // Alternative: image URL
	Model       string            // "llava", "blip2", "cogvlm", etc.
	Task        string            // "caption", "ocr", "vqa" (visual question answering)
	Prompt      string            // Optional prompt for VQA
	Format      ImageFormat       // Image format
	Options     map[string]string // Model-specific options
}

// ImageAnalysisResponse from image analysis
type ImageAnalysisResponse struct {
	Text        string           // Caption, OCR text, or answer
	Confidence  float32          // Overall confidence
	Detections  []Detection      // Optional: detected objects/regions
	Stats       *GenerationStats // Processing stats
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
	Prompt         string            // Text prompt
	NegativePrompt string            // What to avoid in image
	Model          string            // "stable-diffusion", "dall-e", etc.
	Width          int32             // Image width in pixels
	Height         int32             // Image height in pixels
	Format         ImageFormat       // Output format
	Steps          int32             // Diffusion steps (quality vs speed tradeoff)
	GuidanceScale  float32           // Prompt adherence (1-20, default 7.5)
	Seed           int64             // Random seed for reproducibility
	BatchSize      int32             // Number of images to generate
	Options        map[string]string // Model-specific options
}

// ImageGenResponse from image generation
type ImageGenResponse struct {
	Images []GeneratedImage // Generated images
	Stats  *GenerationStats // Processing stats
}

// GeneratedImage represents one generated image
type GeneratedImage struct {
	ImageData []byte      // Image bytes
	Format    ImageFormat // Image format
	Width     int32       // Width in pixels
	Height    int32       // Height in pixels
	Seed      int64       // Seed used for generation
}

// ImageStreamReader for progressive image generation
type ImageStreamReader interface {
	Recv() (*ImageChunk, error)
	io.Closer
}

// ImageChunk represents progressive image generation update
type ImageChunk struct {
	ImageData   []byte      // Partial or complete image
	Progress    float32     // Generation progress (0-1)
	CurrentStep int32       // Current diffusion step
	TotalSteps  int32       // Total diffusion steps
	Done        bool        // True if generation complete
}

// ============================================================
// Video Types
// ============================================================

// VideoFormat defines video encoding format
type VideoFormat string

const (
	VideoFormatMP4  VideoFormat = "mp4"  // MP4 container (H.264)
	VideoFormatWEBM VideoFormat = "webm" // WebM container (VP9)
	VideoFormatGIF  VideoFormat = "gif"  // Animated GIF
)

// VideoAnalysisRequest for video-to-text (transcription, captioning)
type VideoAnalysisRequest struct {
	VideoData   []byte            // Video bytes (non-streaming)
	VideoStream io.Reader         // Streaming video input
	VideoURL    string            // Alternative: video URL
	Model       string            // Model name
	Task        string            // "transcribe", "caption", "describe", "track"
	Format      VideoFormat       // Video format
	Options     map[string]string // Model-specific options
}

// VideoAnalysisResponse from video analysis
type VideoAnalysisResponse struct {
	Text        string              // Transcription or description
	Captions    []VideoCaption      // Time-aligned captions
	Objects     []TrackedObject     // Tracked objects
	Stats       *GenerationStats    // Processing stats
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
	Label      string        // Object label
	TrackID    int32         // Unique track ID
	Confidence float32       // Detection confidence
	Frames     []ObjectFrame // Object locations per frame
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
	FrameData  []byte      // Video frame or segment data
	Text       string      // Associated text (caption, transcription)
	Progress   float32     // Progress (0-1)
	FrameNum   int32       // Current frame number
	TimeMs     int64       // Time in milliseconds
	Done       bool        // True if complete
}

// VideoGenRequest for text-to-video generation
type VideoGenRequest struct {
	Prompt      string            // Text prompt
	Model       string            // Model name
	Width       int32             // Video width in pixels
	Height      int32             // Video height in pixels
	FPS         int32             // Frames per second
	Duration    int32             // Duration in seconds
	Format      VideoFormat       // Output format
	Seed        int64             // Random seed
	Options     map[string]string // Model-specific options
}

// VideoGenResponse from video generation
type VideoGenResponse struct {
	VideoData []byte           // Generated video bytes
	Format    VideoFormat      // Video format
	Width     int32            // Width in pixels
	Height    int32            // Height in pixels
	FPS       int32            // Frames per second
	Duration  int32            // Duration in milliseconds
	Stats     *GenerationStats // Processing stats
}
