package pipeline

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// Integration test for complete voice assistant workflow
func TestIntegrationVoiceAssistantWorkflow(t *testing.T) {
	// Create mock backends for NPU, iGPU, and GPU
	npuBackend := &mockBackend{
		id:                "ollama-npu",
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	igpuBackend := &mockBackend{
		id: "ollama-igpu",
	}

	gpuBackend := &mockBackend{
		id:                "ollama-nvidia",
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{npuBackend, igpuBackend, gpuBackend})

	// Define complete voice assistant pipeline
	pipeline := &Pipeline{
		ID:          "voice-assistant-integration",
		Name:        "Voice Assistant Integration Test",
		Description: "Microphone → NPU STT → iGPU LLM → NPU/GPU TTS",
		Stages: []*Stage{
			{
				ID:               "mic-to-text",
				Type:             StageTypeAudioToText,
				Description:      "Speech-to-text on NPU",
				PreferredBackend: "ollama-npu",
				Model:            "whisper-tiny",
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: false,
					EscalationPath:        []string{"ollama-npu"},
				},
			},
			{
				ID:               "process-query",
				Type:             StageTypeTextGen,
				Description:      "LLM processing on iGPU",
				PreferredBackend: "ollama-igpu",
				Model:            "llama3:7b",
				ForwardingPolicy: &ForwardingPolicy{
					EnableTimeoutForward: true,
					MaxLatencyMs:         500,
					EscalationPath:       []string{"ollama-igpu", "ollama-nvidia"},
				},
			},
			{
				ID:               "text-to-voice",
				Type:             StageTypeTextToAudio,
				Description:      "Text-to-speech on NPU or GPU",
				PreferredBackend: "ollama-npu",
				Model:            "piper-tts-fast",
				ForwardingPolicy: &ForwardingPolicy{
					EscalationPath: []string{"ollama-npu", "ollama-nvidia"},
				},
			},
		},
		Options: &PipelineOptions{
			EnableStreaming: true,
			PreserveContext: true,
			CollectMetrics:  true,
		},
	}

	// Simulate microphone input
	microphoneAudio := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}

	// Execute pipeline
	startTime := time.Now()
	result, err := executor.Execute(context.Background(), pipeline, microphoneAudio)
	totalLatency := time.Since(startTime)

	// Verify results
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful pipeline execution")
	}

	if len(result.StageResults) != 3 {
		t.Errorf("Expected 3 stage results, got %d", len(result.StageResults))
	}

	// Verify stage 1: Audio-to-text
	stage1 := result.StageResults[0]
	if stage1.StageID != "mic-to-text" {
		t.Errorf("Expected stage ID 'mic-to-text', got '%s'", stage1.StageID)
	}
	if !stage1.Success {
		t.Errorf("Stage 1 (audio-to-text) failed")
	}
	if stage1.Backend != "ollama-npu" {
		t.Errorf("Expected NPU backend for stage 1, got %s", stage1.Backend)
	}

	// Verify stage 2: Text generation
	stage2 := result.StageResults[1]
	if stage2.StageID != "process-query" {
		t.Errorf("Expected stage ID 'process-query', got '%s'", stage2.StageID)
	}
	if !stage2.Success {
		t.Errorf("Stage 2 (text generation) failed")
	}
	if stage2.Backend != "ollama-igpu" {
		t.Errorf("Expected iGPU backend for stage 2, got %s", stage2.Backend)
	}

	// Verify stage 3: Text-to-audio
	stage3 := result.StageResults[2]
	if stage3.StageID != "text-to-voice" {
		t.Errorf("Expected stage ID 'text-to-voice', got '%s'", stage3.StageID)
	}
	if !stage3.Success {
		t.Errorf("Stage 3 (text-to-audio) failed")
	}
	// Stage 3 should be on NPU (first in escalation path)
	if stage3.Backend != "ollama-npu" {
		t.Errorf("Expected NPU backend for stage 3, got %s", stage3.Backend)
	}

	// Verify final output is audio
	audioOutput, ok := result.FinalOutput.([]byte)
	if !ok {
		t.Fatalf("Expected []byte final output, got %T", result.FinalOutput)
	}
	if len(audioOutput) == 0 {
		t.Error("Expected non-empty audio output")
	}

	// Log performance metrics
	t.Logf("Pipeline Performance:")
	t.Logf("  Total Latency: %dms", totalLatency.Milliseconds())
	t.Logf("  Stage 1 (STT):  %dms on %s", stage1.Metadata.DurationMs, stage1.Backend)
	t.Logf("  Stage 2 (LLM):  %dms on %s", stage2.Metadata.DurationMs, stage2.Backend)
	t.Logf("  Stage 3 (TTS):  %dms on %s", stage3.Metadata.DurationMs, stage3.Backend)
	t.Logf("  Audio Input:    %d bytes", len(microphoneAudio))
	t.Logf("  Audio Output:   %d bytes", len(audioOutput))
}

// Integration test for parallel multimedia processing
func TestIntegrationParallelMultimediaProcessing(t *testing.T) {
	backend := &mockBackend{
		id:                "test-backend",
		supportsAudioText: true,
		supportsImageText: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	// Define pipeline with parallel stages
	pipeline := &Pipeline{
		ID:          "parallel-multimedia",
		Name:        "Parallel Multimedia Processing",
		Description: "Process audio and image in parallel",
		Stages: []*Stage{
			{
				ID:               "audio-analysis",
				Type:             StageTypeAudioToText,
				PreferredBackend: "test-backend",
				Model:            "whisper-tiny",
			},
			{
				ID:               "image-analysis",
				Type:             StageTypeImageToText,
				PreferredBackend: "test-backend",
				Model:            "llava",
			},
		},
		Options: &PipelineOptions{
			ParallelStages:  true, // Enable parallel execution
			CollectMetrics:  true,
			ContinueOnError: true, // Don't fail if one stage fails
		},
	}

	// Execute pipeline with dummy input
	result, err := executor.Execute(context.Background(), pipeline, "dummy")

	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful pipeline execution")
	}

	if len(result.StageResults) != 2 {
		t.Errorf("Expected 2 stage results, got %d", len(result.StageResults))
	}

	t.Logf("Parallel execution completed in %dms", result.TotalTimeMs)
}

// Integration test for streaming voice pipeline
func TestIntegrationStreamingVoicePipeline(t *testing.T) {
	backend := &mockBackend{
		id:                "streaming-backend",
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "streaming-voice",
		Name: "Streaming Voice Pipeline",
		Stages: []*Stage{
			{
				ID:               "stt",
				Type:             StageTypeAudioToText,
				PreferredBackend: "streaming-backend",
				Model:            "whisper-tiny",
			},
			{
				ID:               "llm",
				Type:             StageTypeTextGen,
				PreferredBackend: "streaming-backend",
				Model:            "llama3",
			},
			{
				ID:               "tts",
				Type:             StageTypeTextToAudio,
				PreferredBackend: "streaming-backend",
				Model:            "piper",
			},
		},
		Options: &PipelineOptions{
			EnableStreaming: true,
			CollectMetrics:  true,
		},
	}

	// Execute with streaming enabled
	audioInput := []byte{0x00, 0x01, 0x02, 0x03}
	result, err := executor.Execute(context.Background(), pipeline, audioInput)

	if err != nil {
		t.Fatalf("Streaming pipeline failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful streaming pipeline")
	}

	// Verify all stages completed
	for i, stageResult := range result.StageResults {
		if !stageResult.Success {
			t.Errorf("Stage %d failed", i)
		}
	}
}

// Integration test for error handling and fallback
func TestIntegrationErrorHandlingAndFallback(t *testing.T) {
	// Create backend that fails first, succeeds on retry
	backend := &mockBackend{
		id:                "fallback-backend",
		supportsAudioText: true,
		transcribeFunc: func(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
			// Simulate transient error
			return nil, io.ErrUnexpectedEOF
		},
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "error-handling",
		Name: "Error Handling Test",
		Stages: []*Stage{
			{
				ID:               "audio-to-text",
				Type:             StageTypeAudioToText,
				PreferredBackend: "fallback-backend",
				Model:            "whisper-tiny",
			},
		},
		Options: &PipelineOptions{
			ContinueOnError: false,
		},
	}

	audioInput := []byte{0x00, 0x01, 0x02, 0x03}
	result, err := executor.Execute(context.Background(), pipeline, audioInput)

	// Should fail due to backend error
	if err == nil {
		t.Error("Expected error from pipeline")
	}

	if result.Success {
		t.Error("Expected unsuccessful pipeline due to error")
	}
}

// Integration test for context preservation
func TestIntegrationContextPreservation(t *testing.T) {
	backend := &mockBackend{
		id: "context-backend",
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "context-preservation",
		Name: "Context Preservation Test",
		Stages: []*Stage{
			{
				ID:               "stage1",
				Type:             StageTypeTextGen,
				PreferredBackend: "context-backend",
				Model:            "llama3",
			},
			{
				ID:               "stage2",
				Type:             StageTypeTextGen,
				PreferredBackend: "context-backend",
				Model:            "llama3",
			},
		},
		Options: &PipelineOptions{
			PreserveContext: true,
			CollectMetrics:  true,
		},
	}

	result, err := executor.Execute(context.Background(), pipeline, "test input")

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful pipeline")
	}

	// Verify output flows through stages
	if result.FinalOutput == nil {
		t.Error("Expected non-nil final output")
	}
}

// Benchmark voice assistant pipeline
func BenchmarkVoiceAssistantPipeline(b *testing.B) {
	backend := &mockBackend{
		id:                "bench-backend",
		supportsAudioText: true,
		supportsTextAudio: true,
	}

	executor := NewPipelineExecutor([]backends.Backend{backend})

	pipeline := &Pipeline{
		ID:   "benchmark",
		Name: "Benchmark Pipeline",
		Stages: []*Stage{
			{
				ID:               "stt",
				Type:             StageTypeAudioToText,
				PreferredBackend: "bench-backend",
				Model:            "whisper-tiny",
			},
			{
				ID:               "llm",
				Type:             StageTypeTextGen,
				PreferredBackend: "bench-backend",
				Model:            "llama3",
			},
			{
				ID:               "tts",
				Type:             StageTypeTextToAudio,
				PreferredBackend: "bench-backend",
				Model:            "piper",
			},
		},
		Options: &PipelineOptions{
			EnableStreaming: true,
		},
	}

	audioInput := []byte{0x00, 0x01, 0x02, 0x03}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.Execute(context.Background(), pipeline, audioInput)
		if err != nil {
			b.Fatalf("Pipeline failed: %v", err)
		}
	}
}
