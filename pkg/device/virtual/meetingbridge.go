package virtual

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/pipeline"
)

// MeetingAudioBridge handles Google Meet audio processing:
// Chrome Speaker → STT (NPU) → LLM (iGPU) → TTS (NPU) → Chrome Microphone
type MeetingAudioBridge struct {
	// Device names
	speakerMonitor string // Where Chrome plays audio (e.g., "ollama-npu-speaker.monitor")
	microphoneSink string // Where we write AI responses (e.g., "ollama-npu-mic")

	// Backends for multi-stage processing (using interface{} for flexibility)
	sttBackend interface{}  // NPU for Whisper STT
	llmBackend interface{}  // iGPU for LLM
	ttsBackend interface{}  // NPU for TTS

	// Pipeline executor
	pipelineExec *pipeline.PipelineExecutor

	// Models
	sttModel string
	llmModel string
	ttsModel string

	// Configuration
	sampleRate int
	channels   int

	// State
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex

	// Audio buffering for streaming
	audioBuffer     []byte
	audioBufferMu   sync.Mutex
	minBufferSize   int // Minimum audio size before processing (for better STT accuracy)
	silenceTimeout  time.Duration
	lastAudioTime   time.Time

	logger *zap.Logger
}

// NewMeetingAudioBridge creates a new meeting audio bridge
func NewMeetingAudioBridge(
	speakerMonitor string,
	microphoneSink string,
	sttBackend interface{},
	llmBackend interface{},
	ttsBackend interface{},
	pipelineExec *pipeline.PipelineExecutor,
	logger *zap.Logger,
) *MeetingAudioBridge {
	ctx, cancel := context.WithCancel(context.Background())

	return &MeetingAudioBridge{
		speakerMonitor:  speakerMonitor,
		microphoneSink:  microphoneSink,
		sttBackend:      sttBackend,
		llmBackend:      llmBackend,
		ttsBackend:      ttsBackend,
		pipelineExec:    pipelineExec,
		sttModel:        "whisper:tiny",  // Fast STT on NPU
		llmModel:        "qwen2.5:0.5b",  // Fast LLM on iGPU
		ttsModel:        "piper:en_US-lessac-medium", // TTS on NPU
		sampleRate:      16000,
		channels:        1,
		ctx:             ctx,
		cancel:          cancel,
		audioBuffer:     make([]byte, 0, 320000), // ~10s at 16kHz mono
		minBufferSize:   32000, // ~1s of audio before processing
		silenceTimeout:  time.Second * 2,
		logger:          logger,
	}
}

// Start starts the meeting audio bridge
func (mab *MeetingAudioBridge) Start() error {
	mab.mu.Lock()
	defer mab.mu.Unlock()

	if mab.running {
		return fmt.Errorf("meeting audio bridge already running")
	}

	// Get backend IDs for logging
	var sttID, llmID, ttsID string
	if b, ok := mab.sttBackend.(interface{ ID() string }); ok {
		sttID = b.ID()
	}
	if b, ok := mab.llmBackend.(interface{ ID() string }); ok {
		llmID = b.ID()
	}
	if b, ok := mab.ttsBackend.(interface{ ID() string }); ok {
		ttsID = b.ID()
	}

	mab.logger.Info("Starting meeting audio bridge",
		zap.String("speaker_monitor", mab.speakerMonitor),
		zap.String("microphone_sink", mab.microphoneSink),
		zap.String("stt_backend", sttID),
		zap.String("llm_backend", llmID),
		zap.String("tts_backend", ttsID),
	)

	// Start audio capture from Chrome speaker
	mab.wg.Add(1)
	go mab.captureAudio()

	// Start audio processor (processes buffered audio)
	mab.wg.Add(1)
	go mab.processAudioLoop()

	mab.running = true
	mab.lastAudioTime = time.Now()

	return nil
}

// Stop stops the meeting audio bridge
func (mab *MeetingAudioBridge) Stop() error {
	mab.mu.Lock()
	if !mab.running {
		mab.mu.Unlock()
		return nil
	}
	mab.running = false
	mab.mu.Unlock()

	mab.logger.Info("Stopping meeting audio bridge")

	// Cancel context to stop goroutines
	mab.cancel()

	// Wait for goroutines to finish
	mab.wg.Wait()

	mab.logger.Info("Meeting audio bridge stopped")
	return nil
}

// captureAudio captures audio from Chrome speaker monitor
func (mab *MeetingAudioBridge) captureAudio() {
	defer mab.wg.Done()

	mab.logger.Info("Starting audio capture from Chrome speaker")

	// Use parec to capture audio from speaker monitor
	args := []string{
		"--device", mab.speakerMonitor,
		"--format", "s16le",
		"--rate", fmt.Sprintf("%d", mab.sampleRate),
		"--channels", fmt.Sprintf("%d", mab.channels),
	}

	cmd := exec.CommandContext(mab.ctx, "parec", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		mab.logger.Error("Failed to create stdout pipe", zap.Error(err))
		return
	}

	if err := cmd.Start(); err != nil {
		mab.logger.Error("Failed to start parec", zap.Error(err))
		return
	}

	// Process audio in chunks (64ms chunks for low latency)
	chunkSamples := 1024
	bufferSize := chunkSamples * mab.channels * 2 // s16le = 2 bytes per sample
	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-mab.ctx.Done():
			mab.logger.Info("Audio capture stopped (context cancelled)")
			cmd.Process.Kill()
			return
		default:
		}

		// Read audio chunk
		n, err := io.ReadFull(stdout, buffer)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				mab.logger.Info("Audio capture ended")
				return
			}
			mab.logger.Error("Failed to read audio", zap.Error(err))
			continue
		}

		if n == 0 {
			continue
		}

		// Add to buffer for processing
		mab.addToBuffer(buffer[:n])
	}
}

// addToBuffer adds audio data to the processing buffer
func (mab *MeetingAudioBridge) addToBuffer(data []byte) {
	mab.audioBufferMu.Lock()
	defer mab.audioBufferMu.Unlock()

	mab.audioBuffer = append(mab.audioBuffer, data...)
	mab.lastAudioTime = time.Now()

	// Log buffer status periodically
	if len(mab.audioBuffer)%64000 == 0 {
		mab.logger.Debug("Audio buffer",
			zap.Int("size_bytes", len(mab.audioBuffer)),
			zap.Float64("duration_seconds", float64(len(mab.audioBuffer))/(float64(mab.sampleRate)*2.0)),
		)
	}
}

// processAudioLoop processes buffered audio when ready
func (mab *MeetingAudioBridge) processAudioLoop() {
	defer mab.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

	for {
		select {
		case <-mab.ctx.Done():
			return
		case <-ticker.C:
			if mab.shouldProcess() {
				if err := mab.processBufferedAudio(); err != nil {
					mab.logger.Error("Failed to process audio", zap.Error(err))
				}
			}
		}
	}
}

// shouldProcess determines if we should process the buffered audio
func (mab *MeetingAudioBridge) shouldProcess() bool {
	mab.audioBufferMu.Lock()
	defer mab.audioBufferMu.Unlock()

	// Process if:
	// 1. Buffer has minimum amount of audio
	// 2. OR there's been silence for silenceTimeout (end of speech)
	bufferSize := len(mab.audioBuffer)
	timeSinceAudio := time.Since(mab.lastAudioTime)

	return bufferSize >= mab.minBufferSize ||
		(bufferSize > 0 && timeSinceAudio > mab.silenceTimeout)
}

// processBufferedAudio processes the current audio buffer through STT→LLM→TTS
func (mab *MeetingAudioBridge) processBufferedAudio() error {
	// Get and clear buffer
	mab.audioBufferMu.Lock()
	if len(mab.audioBuffer) == 0 {
		mab.audioBufferMu.Unlock()
		return nil
	}
	audioData := make([]byte, len(mab.audioBuffer))
	copy(audioData, mab.audioBuffer)
	mab.audioBuffer = mab.audioBuffer[:0] // Clear buffer
	mab.audioBufferMu.Unlock()

	audioDuration := float64(len(audioData)) / (float64(mab.sampleRate) * 2.0)
	mab.logger.Info("Processing audio segment",
		zap.Int("size_bytes", len(audioData)),
		zap.Float64("duration_seconds", audioDuration),
	)

	// Create STT→LLM→TTS pipeline
	pipeline := mab.createPipeline()

	// Convert audio to base64 WAV format for Ollama
	wavData, err := mab.pcmToWAV(audioData, mab.sampleRate, mab.channels)
	if err != nil {
		return fmt.Errorf("failed to convert PCM to WAV: %w", err)
	}

	// Execute pipeline
	startTime := time.Now()
	result, err := mab.pipelineExec.Execute(mab.ctx, pipeline, wavData)
	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	processingTime := time.Since(startTime)
	mab.logger.Info("Pipeline execution completed",
		zap.Duration("processing_time", processingTime),
		zap.Int64("total_time_ms", result.TotalTimeMs),
	)

	// Extract TTS audio from result
	if result.FinalOutput == nil {
		return fmt.Errorf("no output from pipeline")
	}

	ttsAudio, ok := result.FinalOutput.([]byte)
	if !ok {
		return fmt.Errorf("unexpected pipeline output type: %T", result.FinalOutput)
	}

	// Write to Chrome microphone
	if err := mab.writeToMicrophone(ttsAudio); err != nil {
		return fmt.Errorf("failed to write to microphone: %w", err)
	}

	mab.logger.Info("AI response sent to Chrome",
		zap.Int("audio_bytes", len(ttsAudio)),
	)

	return nil
}

// createPipeline creates the STT→LLM→TTS pipeline
func (mab *MeetingAudioBridge) createPipeline() *pipeline.Pipeline {
	// Get backend IDs from interface{}
	var sttBackendID, llmBackendID, ttsBackendID string

	if backend, ok := mab.sttBackend.(interface{ ID() string }); ok {
		sttBackendID = backend.ID()
	}
	if backend, ok := mab.llmBackend.(interface{ ID() string }); ok {
		llmBackendID = backend.ID()
	}
	if backend, ok := mab.ttsBackend.(interface{ ID() string }); ok {
		ttsBackendID = backend.ID()
	}

	return &pipeline.Pipeline{
		ID:          "meeting-assistant",
		Name:        "Google Meet AI Assistant",
		Description: "Process meeting audio: STT → LLM → TTS",
		Stages: []*pipeline.Stage{
			{
				ID:               "stt",
				Type:             pipeline.StageTypeAudioToText,
				Description:      "Transcribe meeting audio",
				PreferredBackend: sttBackendID,
				Model:            mab.sttModel,
			},
			{
				ID:               "llm",
				Type:             pipeline.StageTypeTextGen,
				Description:      "Generate AI response",
				PreferredBackend: llmBackendID,
				Model:            mab.llmModel,
				InputTransform:   mab.createLLMPrompt,
			},
			{
				ID:               "tts",
				Type:             pipeline.StageTypeTextToAudio,
				Description:      "Synthesize speech response",
				PreferredBackend: ttsBackendID,
				Model:            mab.ttsModel,
			},
		},
		Options: &pipeline.PipelineOptions{
			EnableStreaming:  false, // Use batch processing for better quality
			PreserveContext:  true,
			ContinueOnError:  false,
			CollectMetrics:   true,
		},
	}
}

// createLLMPrompt transforms STT output into LLM prompt
func (mab *MeetingAudioBridge) createLLMPrompt(input interface{}) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string from STT, got %T", input)
	}

	// Create a prompt for meeting assistant
	prompt := fmt.Sprintf(`You are an AI assistant in a Google Meet call. Someone just said:

"%s"

Provide a brief, helpful response. Be concise and natural. If it's a question, answer it. If it's a statement, acknowledge it appropriately. Keep your response under 2 sentences.

Response:`, text)

	mab.logger.Debug("Created LLM prompt",
		zap.String("transcription", text),
	)

	return prompt, nil
}

// writeToMicrophone writes TTS audio to the virtual microphone sink
func (mab *MeetingAudioBridge) writeToMicrophone(audioData []byte) error {
	// Use pacat to play audio to virtual microphone
	args := []string{
		"--device", mab.microphoneSink,
		"--format", "s16le",
		"--rate", "22050", // TTS output sample rate
		"--channels", "1",
	}

	cmd := exec.CommandContext(mab.ctx, "pacat", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pacat: %w", err)
	}

	// Write audio data
	if _, err := stdin.Write(audioData); err != nil {
		stdin.Close()
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	// Close stdin to signal end of data
	stdin.Close()

	// Wait for pacat to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pacat failed: %w", err)
	}

	return nil
}

// pcmToWAV converts raw PCM audio to WAV format
func (mab *MeetingAudioBridge) pcmToWAV(pcmData []byte, sampleRate, channels int) ([]byte, error) {
	// WAV header (44 bytes)
	dataSize := len(pcmData)
	fileSize := 36 + dataSize

	header := make([]byte, 44)

	// RIFF chunk
	copy(header[0:4], "RIFF")
	putUint32LE(header[4:8], uint32(fileSize))
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	putUint32LE(header[16:20], 16)                    // fmt chunk size
	putUint16LE(header[20:22], 1)                     // audio format (PCM)
	putUint16LE(header[22:24], uint16(channels))      // num channels
	putUint32LE(header[24:28], uint32(sampleRate))    // sample rate
	putUint32LE(header[28:32], uint32(sampleRate*channels*2)) // byte rate
	putUint16LE(header[32:34], uint16(channels*2))    // block align
	putUint16LE(header[34:36], 16)                    // bits per sample

	// data chunk
	copy(header[36:40], "data")
	putUint32LE(header[40:44], uint32(dataSize))

	// Combine header and data
	wavData := make([]byte, len(header)+len(pcmData))
	copy(wavData, header)
	copy(wavData[len(header):], pcmData)

	// Convert to base64 for Ollama API
	encoded := base64.StdEncoding.EncodeToString(wavData)
	return []byte(encoded), nil
}

// Helper functions for WAV header
func putUint16LE(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func putUint32LE(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// IsRunning returns whether the bridge is running
func (mab *MeetingAudioBridge) IsRunning() bool {
	mab.mu.RLock()
	defer mab.mu.RUnlock()
	return mab.running
}

// SetModels updates the models used for processing
func (mab *MeetingAudioBridge) SetModels(sttModel, llmModel, ttsModel string) {
	mab.mu.Lock()
	defer mab.mu.Unlock()

	if sttModel != "" {
		mab.sttModel = sttModel
	}
	if llmModel != "" {
		mab.llmModel = llmModel
	}
	if ttsModel != "" {
		mab.ttsModel = ttsModel
	}

	mab.logger.Info("Updated models",
		zap.String("stt", mab.sttModel),
		zap.String("llm", mab.llmModel),
		zap.String("tts", mab.ttsModel),
	)
}
