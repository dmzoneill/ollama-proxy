package openvino

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/cmdexec"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/safeconv"
	"go.uber.org/zap"
)

// Backend implements audio processing via OpenVINO on NPU
// Uses whisper.cpp with OpenVINO backend for STT
// Uses Piper ONNX with OpenVINO runtime for TTS
type Backend struct {
	logger       *zap.Logger
	id           string
	name         string
	hardware     string
	device       string
	whisperModel string
	piperModel   string
}

// NewOpenVINOBackend creates a new OpenVINO backend for NPU audio processing
func NewOpenVINOBackend(id, name, hardware, device string, logger *zap.Logger) (*Backend, error) {
	return &Backend{
		id:           id,
		name:         name,
		hardware:     hardware,
		device:       device,
		whisperModel: "/home/daoneill/.cache/xenith/models/whisper-tiny-ov",
		piperModel:   "/home/daoneill/.cache/piper/en_US-lessac-medium.onnx",
		logger:       logger,
	}, nil
}

// ID returns backend identifier
func (b *Backend) ID() string {
	return b.id
}

// Name returns human-readable name
func (b *Backend) Name() string {
	return b.name
}

// Hardware returns hardware type
func (b *Backend) Hardware() string {
	return b.hardware
}

// SupportsAudioToText returns true (Whisper on OpenVINO)
func (b *Backend) SupportsAudioToText() bool {
	return true
}

// SupportsTextToAudio returns true (Piper on OpenVINO)
func (b *Backend) SupportsTextToAudio() bool {
	return true
}

// TranscribeAudio performs STT using Whisper via OpenVINO NPU
func (b *Backend) TranscribeAudio(ctx context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	start := time.Now()

	// For now, use whisper.cpp CLI with OpenVINO backend
	// TODO: Use direct OpenVINO Go bindings when available

	// Write audio to temp file
	tmpFile := fmt.Sprintf("/tmp/whisper_input_%d.wav", time.Now().UnixNano())
	defer removeTemp(tmpFile)

	// Convert PCM to WAV format for whisper.cpp
	// TODO: Implement proper WAV writing

	// Run whisper.cpp with OpenVINO backend
	cmd, err := cmdexec.CommandContext(ctx,
		"whisper-cpp",
		"--model", b.whisperModel,
		"--device", b.device,
		"--output-txt",
		tmpFile,
	)
	if err != nil {
		return nil, fmt.Errorf("whisper-cpp: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("whisper failed: %w - %s", err, string(output))
	}

	elapsed := time.Since(start)

	// Parse output
	text := string(output)

	b.logger.Info("STT completed via OpenVINO",
		zap.String("backend", b.id),
		zap.String("device", b.device),
		zap.Int64("latency_ms", elapsed.Milliseconds()),
	)

	return &backends.TranscribeResponse{
		Text:     text,
		Language: "en",
		Stats: &backends.GenerationStats{
			TotalTimeMs: safeconv.Int32FromInt64(elapsed.Milliseconds()),
		},
	}, nil
}

// TranscribeAudioStream - not implemented
func (b *Backend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("streaming STT not implemented")
}

// SynthesizeSpeech performs TTS using Piper via OpenVINO
func (b *Backend) SynthesizeSpeech(ctx context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	start := time.Now()

	// Use piper binary with OpenVINO backend
	// TODO: Use direct ONNX runtime Go bindings

	tmpOutput := fmt.Sprintf("/tmp/piper_output_%d.wav", time.Now().UnixNano())
	defer removeTemp(tmpOutput)

	cmd, err := cmdexec.CommandContext(ctx,
		"piper",
		"--model", b.piperModel,
		"--output_file", tmpOutput,
	)
	if err != nil {
		return nil, fmt.Errorf("piper: %w", err)
	}

	cmd.Stdin = nil // TODO: Pipe text input

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("piper failed: %w", err)
	}

	elapsed := time.Since(start)

	b.logger.Info("TTS completed via OpenVINO",
		zap.String("backend", b.id),
		zap.String("device", b.device),
		zap.Int64("latency_ms", elapsed.Milliseconds()),
	)

	return &backends.SynthesizeResponse{
		AudioData:  []byte{}, // TODO: Read WAV file
		Format:     backends.AudioFormatPCM,
		SampleRate: 22050,
		Stats: &backends.GenerationStats{
			TotalTimeMs: safeconv.Int32FromInt64(elapsed.Milliseconds()),
		},
	}, nil
}

// SynthesizeSpeechStream - not implemented
func (b *Backend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("streaming TTS not implemented")
}

// removeTemp deletes a best-effort temporary file.
func removeTemp(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		logging.Warn("remove temp file", zap.Error(err))
	}
}
