package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// StageType defines the type of processing stage
type StageType string

const (
	// Text stages
	StageTypeTextGen StageType = "text_generation" // LLM inference
	StageTypeEmbed   StageType = "embedding"       // Generate embeddings

	// Audio stages
	StageTypeAudioToText    StageType = "audio_to_text"    // Speech recognition (Whisper)
	StageTypeTextToAudio    StageType = "text_to_audio"    // Text-to-speech (Piper, Bark)
	StageTypeAudioEnhance   StageType = "audio_enhance"    // Noise reduction, enhancement
	StageTypeAudioTranslate StageType = "audio_translate"  // Speech translation

	// Image stages
	StageTypeImageToText StageType = "image_to_text" // OCR, image captioning (LLaVA, BLIP)
	StageTypeTextToImage StageType = "text_to_image" // Image generation (Stable Diffusion, DALL-E)
	StageTypeImageEdit   StageType = "image_edit"    // Image editing, inpainting
	StageTypeImageEnhance StageType = "image_enhance" // Upscaling, restoration

	// Video stages
	StageTypeVideoToText     StageType = "video_to_text"     // Video transcription/captioning
	StageTypeTextToVideo     StageType = "text_to_video"     // Video generation
	StageTypeVideoAnalysis   StageType = "video_analysis"    // Object detection, tracking
	StageTypeVideoSummary    StageType = "video_summary"     // Video summarization

	// Generic
	StageTypeCustom StageType = "custom" // Custom processing
)

// Stage represents one step in a processing pipeline
type Stage struct {
	ID          string
	Type        StageType
	Description string

	// Backend selection
	PreferredBackend string   // Specific backend ID
	PreferredHardware string  // "npu", "igpu", "nvidia", etc.
	RequiredCapabilities []string // ["audio", "streaming", etc.]

	// Model selection
	Model string // Model to use for this stage

	// Forwarding policy
	ForwardingPolicy *ForwardingPolicy

	// Input/Output transformation
	InputTransform  func(interface{}) (interface{}, error)
	OutputTransform func(interface{}) (interface{}, error)
}

// ForwardingPolicy defines when and how to forward to next backend
type ForwardingPolicy struct {
	// Confidence-based forwarding
	EnableConfidenceCheck bool
	MinConfidence         float64
	MaxRetries           int
	EscalationPath       []string // Ordered list of backend IDs

	// Thermal-based forwarding
	EnableThermalCheck bool
	MaxTemperature     float64
	MaxFanPercent      int

	// Quality-based forwarding
	EnableQualityCheck bool
	QualityThreshold   float64

	// Timeout-based forwarding
	EnableTimeoutForward bool
	MaxLatencyMs        int32
}

// Pipeline represents a multi-stage processing workflow
type Pipeline struct {
	ID          string
	Name        string
	Description string
	Stages      []*Stage

	// Execution options
	Options *PipelineOptions
}

// PipelineOptions controls pipeline execution
type PipelineOptions struct {
	// Streaming
	EnableStreaming bool

	// Context preservation
	PreserveContext bool

	// Error handling
	ContinueOnError bool
	FallbackStage   *Stage

	// Performance
	ParallelStages bool // Execute independent stages in parallel

	// Monitoring
	CollectMetrics bool
}

// StageResult represents the output of a stage
type StageResult struct {
	StageID   string
	Backend   string
	Success   bool
	Output    interface{}
	Metadata  *StageMetadata
	Error     error
}

// StageMetadata contains execution metadata
type StageMetadata struct {
	StartTime      time.Time
	EndTime        time.Time
	DurationMs     int64
	Backend        string
	Model          string
	TokensIn       int32
	TokensOut      int32
	Confidence     float64
	Temperature    float64
	FanSpeed       int
	Forwarded      bool
	ForwardReason  string
	AttemptCount   int
}

// PipelineResult represents the complete pipeline execution result
type PipelineResult struct {
	PipelineID   string
	Success      bool
	StageResults []*StageResult
	FinalOutput  interface{}
	TotalTimeMs  int64
	TotalEnergyWh float64
	Error        error
}

// PipelineExecutor executes multi-stage pipelines
type PipelineExecutor struct {
	backendRegistry map[string]backends.Backend
}

// NewPipelineExecutor creates a new pipeline executor
func NewPipelineExecutor(backendList []backends.Backend) *PipelineExecutor {
	registry := make(map[string]backends.Backend)
	for _, backend := range backendList {
		registry[backend.ID()] = backend
	}

	return &PipelineExecutor{
		backendRegistry: registry,
	}
}

// Execute runs a pipeline
func (pe *PipelineExecutor) Execute(ctx context.Context, pipeline *Pipeline, input interface{}) (*PipelineResult, error) {
	startTime := time.Now()
	result := &PipelineResult{
		PipelineID:   pipeline.ID,
		StageResults: make([]*StageResult, 0),
	}

	// Check if parallel execution is enabled
	if pipeline.Options != nil && pipeline.Options.ParallelStages {
		return pe.executeParallel(ctx, pipeline, input, startTime)
	}

	// Sequential execution (default)
	currentInput := input

	// Execute each stage
	for i, stage := range pipeline.Stages {
		stageResult, err := pe.executeStage(ctx, stage, currentInput)
		result.StageResults = append(result.StageResults, stageResult)

		if err != nil && !pipeline.Options.ContinueOnError {
			result.Success = false
			result.Error = fmt.Errorf("stage %d (%s) failed: %w", i, stage.ID, err)
			return result, result.Error
		}

		if stageResult.Success {
			currentInput = stageResult.Output
		}
	}

	result.FinalOutput = currentInput
	result.Success = true
	result.TotalTimeMs = time.Since(startTime).Milliseconds()

	return result, nil
}

// executeParallel runs independent pipeline stages in parallel
func (pe *PipelineExecutor) executeParallel(ctx context.Context, pipeline *Pipeline, input interface{}, startTime time.Time) (*PipelineResult, error) {
	result := &PipelineResult{
		PipelineID:   pipeline.ID,
		StageResults: make([]*StageResult, 0),
	}

	// Group stages by dependency level
	// For now, we assume all stages at the same level can run in parallel
	// More sophisticated dependency analysis can be added later

	type stageTask struct {
		stage  *Stage
		input  interface{}
		result *StageResult
		err    error
	}

	// Determine which stages can run in parallel
	// Simple approach: stages that don't depend on previous stage output can run in parallel
	parallelGroups := make([][]*Stage, 0)
	currentGroup := make([]*Stage, 0)

	for _, stage := range pipeline.Stages {
		// If stage requires input from previous stage, start new group
		if stage.InputTransform != nil && len(currentGroup) > 0 {
			parallelGroups = append(parallelGroups, currentGroup)
			currentGroup = []*Stage{stage}
		} else {
			currentGroup = append(currentGroup, stage)
		}
	}
	if len(currentGroup) > 0 {
		parallelGroups = append(parallelGroups, currentGroup)
	}

	// Execute groups sequentially, but stages within group in parallel
	currentInput := input

	for groupIdx, group := range parallelGroups {
		if len(group) == 1 {
			// Single stage, execute normally
			stageResult, err := pe.executeStage(ctx, group[0], currentInput)
			result.StageResults = append(result.StageResults, stageResult)

			if err != nil && !pipeline.Options.ContinueOnError {
				result.Success = false
				result.Error = fmt.Errorf("stage group %d failed: %w", groupIdx, err)
				return result, result.Error
			}

			if stageResult.Success {
				currentInput = stageResult.Output
			}
		} else {
			// Multiple stages in parallel
			tasks := make([]*stageTask, len(group))
			resultChan := make(chan *stageTask, len(group))

			// Launch parallel tasks
			for i, stage := range group {
				tasks[i] = &stageTask{
					stage: stage,
					input: currentInput,
				}

				go func(task *stageTask) {
					task.result, task.err = pe.executeStage(ctx, task.stage, task.input)
					resultChan <- task
				}(tasks[i])
			}

			// Collect results
			for i := 0; i < len(group); i++ {
				task := <-resultChan
				result.StageResults = append(result.StageResults, task.result)

				if task.err != nil && !pipeline.Options.ContinueOnError {
					result.Success = false
					result.Error = fmt.Errorf("parallel stage %s failed: %w", task.stage.ID, task.err)
					return result, result.Error
				}
			}

			// For parallel stages, use the output of the last successful stage
			for _, task := range tasks {
				if task.result != nil && task.result.Success {
					currentInput = task.result.Output
					break
				}
			}
		}
	}

	result.FinalOutput = currentInput
	result.Success = true
	result.TotalTimeMs = time.Since(startTime).Milliseconds()

	return result, nil
}

// executeStage executes a single stage with forwarding support
func (pe *PipelineExecutor) executeStage(ctx context.Context, stage *Stage, input interface{}) (*StageResult, error) {
	metadata := &StageMetadata{
		StartTime: time.Now(),
	}

	// Transform input
	processedInput := input
	if stage.InputTransform != nil {
		var err error
		processedInput, err = stage.InputTransform(input)
		if err != nil {
			return &StageResult{
				StageID:  stage.ID,
				Success:  false,
				Error:    fmt.Errorf("input transform failed: %w", err),
				Metadata: metadata,
			}, err
		}
	}

	// Select backend
	backend, err := pe.selectBackend(stage)
	if err != nil {
		return &StageResult{
			StageID:  stage.ID,
			Success:  false,
			Error:    err,
			Metadata: metadata,
		}, err
	}

	metadata.Backend = backend.ID()
	metadata.Model = stage.Model

	// Execute with forwarding policy
	var output interface{}
	var execErr error

	if stage.ForwardingPolicy != nil && stage.ForwardingPolicy.EnableConfidenceCheck {
		output, metadata, execErr = pe.executeWithForwarding(ctx, stage, backend, processedInput, metadata)
	} else {
		output, execErr = pe.executeOnBackend(ctx, backend, stage, processedInput)
	}

	if execErr != nil {
		return &StageResult{
			StageID:  stage.ID,
			Success:  false,
			Error:    execErr,
			Metadata: metadata,
		}, execErr
	}

	// Transform output
	finalOutput := output
	if stage.OutputTransform != nil {
		finalOutput, execErr = stage.OutputTransform(output)
		if execErr != nil {
			return &StageResult{
				StageID:  stage.ID,
				Success:  false,
				Error:    fmt.Errorf("output transform failed: %w", execErr),
				Metadata: metadata,
			}, execErr
		}
	}

	metadata.EndTime = time.Now()
	metadata.DurationMs = metadata.EndTime.Sub(metadata.StartTime).Milliseconds()

	return &StageResult{
		StageID:  stage.ID,
		Backend:  backend.ID(),
		Success:  true,
		Output:   finalOutput,
		Metadata: metadata,
	}, nil
}

// executeWithForwarding executes with confidence-based forwarding
func (pe *PipelineExecutor) executeWithForwarding(
	ctx context.Context,
	stage *Stage,
	initialBackend backends.Backend,
	input interface{},
	metadata *StageMetadata,
) (interface{}, *StageMetadata, error) {
	policy := stage.ForwardingPolicy
	currentBackend := initialBackend
	attemptCount := 0

	for attemptCount < policy.MaxRetries {
		attemptCount++
		metadata.AttemptCount = attemptCount

		// Execute on current backend
		output, err := pe.executeOnBackend(ctx, currentBackend, stage, input)
		if err != nil {
			// Try next backend in escalation path
			if attemptCount < len(policy.EscalationPath) {
				nextBackendID := policy.EscalationPath[attemptCount]
				nextBackend, _ := pe.getBackendByID(nextBackendID)
				currentBackend = nextBackend
				metadata.Forwarded = true
				metadata.ForwardReason = fmt.Sprintf("Error: %v", err)
				continue
			}
			return nil, metadata, err
		}

		// Check confidence (if available)
		confidence := pe.estimateConfidence(output)
		metadata.Confidence = confidence

		if confidence >= policy.MinConfidence {
			// Success!
			return output, metadata, nil
		}

		// Confidence too low, try next backend
		if attemptCount < len(policy.EscalationPath) {
			nextBackendID := policy.EscalationPath[attemptCount]
			nextBackend, _ := pe.getBackendByID(nextBackendID)
			currentBackend = nextBackend
			metadata.Forwarded = true
			metadata.ForwardReason = fmt.Sprintf("Low confidence: %.2f < %.2f", confidence, policy.MinConfidence)
			metadata.Backend = nextBackend.ID()
			continue
		}

		// No more backends to try
		return output, metadata, nil
	}

	return nil, metadata, fmt.Errorf("max retries exceeded")
}

// executeOnBackend executes stage on specific backend
func (pe *PipelineExecutor) executeOnBackend(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	switch stage.Type {
	// ===== TEXT STAGES =====
	case StageTypeTextGen:
		return pe.executeTextGeneration(ctx, backend, stage, input)

	case StageTypeEmbed:
		return pe.executeEmbedding(ctx, backend, stage, input)

	// ===== AUDIO STAGES =====
	case StageTypeAudioToText:
		return pe.executeAudioToText(ctx, backend, stage, input)

	case StageTypeTextToAudio:
		return pe.executeTextToAudio(ctx, backend, stage, input)

	case StageTypeAudioEnhance:
		return pe.executeAudioEnhance(ctx, backend, stage, input)

	case StageTypeAudioTranslate:
		return pe.executeAudioTranslate(ctx, backend, stage, input)

	// ===== IMAGE STAGES =====
	case StageTypeImageToText:
		return pe.executeImageToText(ctx, backend, stage, input)

	case StageTypeTextToImage:
		return pe.executeTextToImage(ctx, backend, stage, input)

	case StageTypeImageEdit:
		return pe.executeImageEdit(ctx, backend, stage, input)

	case StageTypeImageEnhance:
		return pe.executeImageEnhance(ctx, backend, stage, input)

	// ===== VIDEO STAGES =====
	case StageTypeVideoToText:
		return pe.executeVideoToText(ctx, backend, stage, input)

	case StageTypeTextToVideo:
		return pe.executeTextToVideo(ctx, backend, stage, input)

	case StageTypeVideoAnalysis:
		return pe.executeVideoAnalysis(ctx, backend, stage, input)

	case StageTypeVideoSummary:
		return pe.executeVideoSummary(ctx, backend, stage, input)

	// ===== CUSTOM =====
	case StageTypeCustom:
		return nil, fmt.Errorf("custom stages must provide InputTransform function")

	default:
		return nil, fmt.Errorf("unsupported stage type: %s", stage.Type)
	}
}

// ============================================================
// Text Stage Implementations
// ============================================================

func (pe *PipelineExecutor) executeTextGeneration(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	prompt, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input for text generation")
	}

	req := &backends.GenerateRequest{
		Model:  stage.Model,
		Prompt: prompt,
	}
	resp, err := backend.Generate(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (pe *PipelineExecutor) executeEmbedding(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input for embedding")
	}

	req := &backends.EmbedRequest{
		Model: stage.Model,
		Text:  text,
	}
	resp, err := backend.Embed(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Embedding, nil
}

// ============================================================
// Audio Stage Implementations - Optimized for Low Latency
// ============================================================

func (pe *PipelineExecutor) executeAudioToText(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Check backend capability
	if !backend.SupportsAudioToText() {
		return nil, fmt.Errorf("backend %s does not support audio-to-text", backend.ID())
	}

	// Accept both []byte (audio data) and *backends.TranscribeRequest
	var req *backends.TranscribeRequest
	switch v := input.(type) {
	case []byte:
		// Raw audio bytes - create request with defaults
		req = &backends.TranscribeRequest{
			AudioData:        v,
			Model:            stage.Model,
			Format:           backends.AudioFormatWAV,
			SampleRate:       16000, // Common sample rate for speech
			Channels:         1,     // Mono
			EnableVAD:        true,  // Enable VAD for streaming
			EnableTimestamps: true,  // Useful for pipelines
		}
	case *backends.TranscribeRequest:
		// Pre-configured request
		req = v
		if req.Model == "" {
			req.Model = stage.Model
		}
	default:
		return nil, fmt.Errorf("expected []byte or *TranscribeRequest for audio-to-text, got %T", input)
	}

	// Use streaming transcription for lower latency when audio stream available
	if req.AudioStream != nil {
		return pe.executeAudioToTextStreaming(ctx, backend, req)
	}

	// Non-streaming transcription
	resp, err := backend.TranscribeAudio(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	return resp.Text, nil
}

func (pe *PipelineExecutor) executeAudioToTextStreaming(
	ctx context.Context,
	backend backends.Backend,
	req *backends.TranscribeRequest,
) (interface{}, error) {
	stream, err := backend.TranscribeAudioStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming transcription: %w", err)
	}
	defer stream.Close()

	// Accumulate streaming results
	var fullText string
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		if chunk.IsFinal {
			fullText += chunk.Text
		}

		if chunk.Done {
			break
		}
	}

	return fullText, nil
}

func (pe *PipelineExecutor) executeTextToAudio(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Check backend capability
	if !backend.SupportsTextToAudio() {
		return nil, fmt.Errorf("backend %s does not support text-to-audio", backend.ID())
	}

	// Accept both string (text) and *backends.SynthesizeRequest
	var req *backends.SynthesizeRequest
	switch v := input.(type) {
	case string:
		// Raw text - create request with defaults optimized for low latency
		req = &backends.SynthesizeRequest{
			Text:       v,
			Model:      stage.Model,
			Format:     backends.AudioFormatPCM, // PCM for lowest latency (no encoding)
			SampleRate: 22050,                   // Good quality/performance balance
			Speed:      1.0,
			Pitch:      0.0,
		}
	case *backends.SynthesizeRequest:
		// Pre-configured request
		req = v
		if req.Model == "" {
			req.Model = stage.Model
		}
	default:
		return nil, fmt.Errorf("expected string or *SynthesizeRequest for text-to-audio, got %T", input)
	}

	// Use streaming synthesis for lower latency
	return pe.executeTextToAudioStreaming(ctx, backend, req)
}

func (pe *PipelineExecutor) executeTextToAudioStreaming(
	ctx context.Context,
	backend backends.Backend,
	req *backends.SynthesizeRequest,
) (interface{}, error) {
	stream, err := backend.SynthesizeSpeechStream(ctx, req)
	if err != nil {
		// Fallback to non-streaming if streaming not available
		resp, err := backend.SynthesizeSpeech(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("synthesis failed: %w", err)
		}
		return resp.AudioData, nil
	}
	defer stream.Close()

	// Accumulate audio chunks with pre-allocated buffer to reduce allocations
	var audioData []byte
	estimatedSize := 100 * 1024 // Estimate 100KB for typical speech
	audioData = make([]byte, 0, estimatedSize)

	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		// Append chunk data (zero-copy when possible)
		audioData = append(audioData, chunk.Data...)

		if chunk.Done {
			break
		}
	}

	return audioData, nil
}

func (pe *PipelineExecutor) executeAudioEnhance(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Audio enhancement (noise reduction, etc.)
	// This would call a specialized model for audio processing
	return nil, fmt.Errorf("audio enhancement not yet implemented - requires specialized model")
}

func (pe *PipelineExecutor) executeAudioTranslate(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Speech translation (e.g., Whisper with translation)
	// Similar to transcription but with target language
	return nil, fmt.Errorf("audio translation not yet implemented - requires translation model")
}

// ============================================================
// Image Stage Implementations
// ============================================================

func (pe *PipelineExecutor) executeImageToText(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Check backend capability
	if !backend.SupportsImageToText() {
		return nil, fmt.Errorf("backend %s does not support image-to-text", backend.ID())
	}

	// Accept both []byte (image data) and *backends.ImageAnalysisRequest
	var req *backends.ImageAnalysisRequest
	switch v := input.(type) {
	case []byte:
		req = &backends.ImageAnalysisRequest{
			ImageData: v,
			Model:     stage.Model,
			Task:      "caption", // Default to captioning
			Format:    backends.ImageFormatJPEG,
		}
	case *backends.ImageAnalysisRequest:
		req = v
		if req.Model == "" {
			req.Model = stage.Model
		}
	default:
		return nil, fmt.Errorf("expected []byte or *ImageAnalysisRequest for image-to-text, got %T", input)
	}

	resp, err := backend.AnalyzeImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("image analysis failed: %w", err)
	}

	return resp.Text, nil
}

func (pe *PipelineExecutor) executeTextToImage(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Check backend capability
	if !backend.SupportsTextToImage() {
		return nil, fmt.Errorf("backend %s does not support text-to-image", backend.ID())
	}

	// Accept both string (prompt) and *backends.ImageGenRequest
	var req *backends.ImageGenRequest
	switch v := input.(type) {
	case string:
		req = &backends.ImageGenRequest{
			Prompt:        v,
			Model:         stage.Model,
			Width:         512,  // Default dimensions
			Height:        512,
			Format:        backends.ImageFormatPNG,
			Steps:         20,   // Balanced quality/speed
			GuidanceScale: 7.5,
			BatchSize:     1,
		}
	case *backends.ImageGenRequest:
		req = v
		if req.Model == "" {
			req.Model = stage.Model
		}
	default:
		return nil, fmt.Errorf("expected string or *ImageGenRequest for text-to-image, got %T", input)
	}

	// Use streaming generation for progressive updates
	stream, err := backend.GenerateImageStream(ctx, req)
	if err != nil {
		// Fallback to non-streaming
		resp, err := backend.GenerateImage(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("image generation failed: %w", err)
		}
		if len(resp.Images) == 0 {
			return nil, fmt.Errorf("no images generated")
		}
		return resp.Images[0].ImageData, nil
	}
	defer stream.Close()

	// Get final image from stream
	var finalImage []byte
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		finalImage = chunk.ImageData

		if chunk.Done {
			break
		}
	}

	return finalImage, nil
}

func (pe *PipelineExecutor) executeImageEdit(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Image editing/inpainting
	return nil, fmt.Errorf("image editing not yet implemented - requires inpainting model")
}

func (pe *PipelineExecutor) executeImageEnhance(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Image enhancement/upscaling
	return nil, fmt.Errorf("image enhancement not yet implemented - requires upscaling model")
}

// ============================================================
// Video Stage Implementations
// ============================================================

func (pe *PipelineExecutor) executeVideoToText(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Check backend capability
	if !backend.SupportsVideoToText() {
		return nil, fmt.Errorf("backend %s does not support video-to-text", backend.ID())
	}

	// Accept both []byte (video data) and *backends.VideoAnalysisRequest
	var req *backends.VideoAnalysisRequest
	switch v := input.(type) {
	case []byte:
		req = &backends.VideoAnalysisRequest{
			VideoData: v,
			Model:     stage.Model,
			Task:      "transcribe",
			Format:    backends.VideoFormatMP4,
		}
	case *backends.VideoAnalysisRequest:
		req = v
		if req.Model == "" {
			req.Model = stage.Model
		}
	default:
		return nil, fmt.Errorf("expected []byte or *VideoAnalysisRequest for video-to-text, got %T", input)
	}

	// Use streaming analysis for lower latency on long videos
	if req.VideoStream != nil {
		return pe.executeVideoToTextStreaming(ctx, backend, req)
	}

	resp, err := backend.AnalyzeVideo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("video analysis failed: %w", err)
	}

	return resp.Text, nil
}

func (pe *PipelineExecutor) executeVideoToTextStreaming(
	ctx context.Context,
	backend backends.Backend,
	req *backends.VideoAnalysisRequest,
) (interface{}, error) {
	stream, err := backend.AnalyzeVideoStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming analysis: %w", err)
	}
	defer stream.Close()

	var fullText string
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		fullText += chunk.Text

		if chunk.Done {
			break
		}
	}

	return fullText, nil
}

func (pe *PipelineExecutor) executeTextToVideo(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Check backend capability
	if !backend.SupportsTextToVideo() {
		return nil, fmt.Errorf("backend %s does not support text-to-video", backend.ID())
	}

	// Accept both string (prompt) and *backends.VideoGenRequest
	var req *backends.VideoGenRequest
	switch v := input.(type) {
	case string:
		req = &backends.VideoGenRequest{
			Prompt:   v,
			Model:    stage.Model,
			Width:    512,
			Height:   512,
			FPS:      24,
			Duration: 3, // 3 seconds default
			Format:   backends.VideoFormatMP4,
		}
	case *backends.VideoGenRequest:
		req = v
		if req.Model == "" {
			req.Model = stage.Model
		}
	default:
		return nil, fmt.Errorf("expected string or *VideoGenRequest for text-to-video, got %T", input)
	}

	// Video generation is compute-intensive, use streaming for progress updates
	stream, err := backend.GenerateVideoStream(ctx, req)
	if err != nil {
		// Fallback to non-streaming
		resp, err := backend.GenerateVideo(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("video generation failed: %w", err)
		}
		return resp.VideoData, nil
	}
	defer stream.Close()

	// Accumulate video data
	var videoData []byte
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		videoData = append(videoData, chunk.FrameData...)

		if chunk.Done {
			break
		}
	}

	return videoData, nil
}

func (pe *PipelineExecutor) executeVideoAnalysis(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Object detection, tracking, etc.
	// Similar to VideoToText but focuses on object detection
	return pe.executeVideoToText(ctx, backend, stage, input)
}

func (pe *PipelineExecutor) executeVideoSummary(
	ctx context.Context,
	backend backends.Backend,
	stage *Stage,
	input interface{},
) (interface{}, error) {
	// Video summarization
	return pe.executeVideoToText(ctx, backend, stage, input)
}

// selectBackend selects the best backend for a stage
func (pe *PipelineExecutor) selectBackend(stage *Stage) (backends.Backend, error) {
	// If specific backend requested, use it
	if stage.PreferredBackend != "" {
		backend, ok := pe.backendRegistry[stage.PreferredBackend]
		if !ok {
			return nil, fmt.Errorf("backend not found: %s", stage.PreferredBackend)
		}
		return backend, nil
	}

	// If hardware type specified, find first matching backend
	if stage.PreferredHardware != "" {
		for _, backend := range pe.backendRegistry {
			if backend.Hardware() == stage.PreferredHardware {
				return backend, nil
			}
		}
		return nil, fmt.Errorf("no backend found with hardware: %s", stage.PreferredHardware)
	}

	// Default: use first available backend
	for _, backend := range pe.backendRegistry {
		return backend, nil
	}

	return nil, fmt.Errorf("no backends available")
}

// getBackendByID gets backend by ID
func (pe *PipelineExecutor) getBackendByID(id string) (backends.Backend, error) {
	backend, ok := pe.backendRegistry[id]
	if !ok {
		return nil, fmt.Errorf("backend not found: %s", id)
	}
	return backend, nil
}

// estimateConfidence estimates output confidence
func (pe *PipelineExecutor) estimateConfidence(output interface{}) float64 {
	// TODO: Implement confidence estimation
	// Could use:
	// - Response length
	// - Token probabilities (if available)
	// - Pattern matching
	// - Model-specific heuristics
	return 0.8 // Placeholder
}
