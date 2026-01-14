package virtual

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/pipeline"
)

// AudioBridge bridges physical microphone to virtual speaker via processing pipeline
type AudioBridge struct {
	// Devices
	physicalMic    string // Physical microphone device name
	virtualSource  string // Virtual microphone name (what Chrome selects)
	virtualSink    string // Virtual speaker name (what proxy writes to)

	// Pipeline
	pipeline     *pipeline.Pipeline
	pipelineExec *pipeline.PipelineExecutor
	backend      backends.Backend

	// Configuration
	inputSampleRate  int
	inputChannels    int
	outputSampleRate int
	outputChannels   int

	// Lifecycle
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewAudioBridge creates a new audio bridge
func NewAudioBridge(
	physicalMic string,
	virtualSource string,
	virtualSink string,
	pipeline *pipeline.Pipeline,
	pipelineExec *pipeline.PipelineExecutor,
	backend backends.Backend,
	logger *zap.Logger,
) *AudioBridge {
	ctx, cancel := context.WithCancel(context.Background())

	return &AudioBridge{
		physicalMic:      physicalMic,
		virtualSource:    virtualSource,
		virtualSink:      virtualSink,
		pipeline:         pipeline,
		pipelineExec:     pipelineExec,
		backend:          backend,
		inputSampleRate:  16000,
		inputChannels:    1,
		outputSampleRate: 22050,
		outputChannels:   2,
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
	}
}

// Start starts the audio bridge
func (ab *AudioBridge) Start() error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.running {
		return fmt.Errorf("audio bridge already running")
	}

	ab.logger.Info("Starting audio bridge",
		zap.String("physical_mic", ab.physicalMic),
		zap.String("virtual_source", ab.virtualSource),
		zap.String("virtual_sink", ab.virtualSink),
	)

	// Start microphone input processing
	ab.wg.Add(1)
	go ab.processMicInput()

	ab.running = true
	return nil
}

// Stop stops the audio bridge
func (ab *AudioBridge) Stop() error {
	ab.mu.Lock()
	if !ab.running {
		ab.mu.Unlock()
		return nil
	}
	ab.running = false
	ab.mu.Unlock()

	ab.logger.Info("Stopping audio bridge")

	// Cancel context to stop goroutines
	ab.cancel()

	// Wait for goroutines to finish
	ab.wg.Wait()

	ab.logger.Info("Audio bridge stopped")
	return nil
}

// processMicInput processes audio from physical microphone
func (ab *AudioBridge) processMicInput() {
	defer ab.wg.Done()

	ab.logger.Info("Starting microphone input processing")

	// Use parec to capture audio from physical microphone
	// parec is PulseAudio's recording utility
	args := []string{
		"--device", ab.physicalMic,
		"--format", "s16le",
		"--rate", fmt.Sprintf("%d", ab.inputSampleRate),
		"--channels", fmt.Sprintf("%d", ab.inputChannels),
	}

	cmd := exec.CommandContext(ab.ctx, "parec", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ab.logger.Error("Failed to create stdout pipe", zap.Error(err))
		return
	}

	if err := cmd.Start(); err != nil {
		ab.logger.Error("Failed to start parec", zap.Error(err))
		return
	}

	// Process audio in chunks
	// Each sample is 2 bytes (s16le), so buffer size = samples * channels * 2
	chunkSamples := 1024 // 1024 samples = ~64ms at 16kHz
	bufferSize := chunkSamples * ab.inputChannels * 2
	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-ab.ctx.Done():
			ab.logger.Info("Microphone processing stopped (context cancelled)")
			cmd.Process.Kill()
			return
		default:
		}

		// Read audio chunk
		n, err := io.ReadFull(stdout, buffer)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				ab.logger.Info("Microphone input ended")
				return
			}
			ab.logger.Error("Failed to read audio", zap.Error(err))
			continue
		}

		if n == 0 {
			continue
		}

		// Process audio through pipeline
		audioData := buffer[:n]
		processedAudio, err := ab.processAudioChunk(audioData)
		if err != nil {
			ab.logger.Error("Failed to process audio chunk", zap.Error(err))
			continue
		}

		// Write processed audio to virtual speaker
		if processedAudio != nil && len(processedAudio) > 0 {
			if err := ab.writeToVirtualSpeaker(processedAudio); err != nil {
				ab.logger.Error("Failed to write to virtual speaker", zap.Error(err))
			}
		}
	}
}

// processAudioChunk processes an audio chunk through the pipeline
func (ab *AudioBridge) processAudioChunk(audioData []byte) ([]byte, error) {
	if ab.pipeline == nil {
		// No pipeline configured, pass through
		return audioData, nil
	}

	// Execute pipeline
	result, err := ab.pipelineExec.Execute(ab.ctx, ab.pipeline, audioData)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Extract processed audio from result
	if result.FinalOutput == nil {
		return nil, fmt.Errorf("no output from pipeline")
	}

	// Convert output to bytes
	switch output := result.FinalOutput.(type) {
	case []byte:
		return output, nil
	case string:
		// If output is text (e.g., from STT), we might want to log it or use it for TTS
		ab.logger.Debug("Pipeline output (text)", zap.String("text", output))
		// For now, return original audio
		return audioData, nil
	default:
		ab.logger.Warn("Unexpected pipeline output type", zap.Any("type", output))
		return audioData, nil
	}
}

// writeToVirtualSpeaker writes audio data to the virtual speaker
func (ab *AudioBridge) writeToVirtualSpeaker(audioData []byte) error {
	// Use pacat to play audio to virtual speaker
	// pacat is PulseAudio's playback utility
	args := []string{
		"--device", ab.virtualSink,
		"--format", "s16le",
		"--rate", fmt.Sprintf("%d", ab.outputSampleRate),
		"--channels", fmt.Sprintf("%d", ab.outputChannels),
	}

	cmd := exec.CommandContext(ab.ctx, "pacat", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pacat: %w", err)
	}

	// Write audio data
	if _, err := stdin.Write(audioData); err != nil {
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

// IsRunning returns whether the bridge is running
func (ab *AudioBridge) IsRunning() bool {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.running
}

// SetPipeline updates the processing pipeline
func (ab *AudioBridge) SetPipeline(pipeline *pipeline.Pipeline) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.pipeline = pipeline
	ab.logger.Info("Updated audio bridge pipeline")
}
