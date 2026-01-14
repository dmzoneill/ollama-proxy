package virtual

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/device"
	"github.com/daoneill/ollama-proxy/pkg/pipeline"
)

// VideoBridge bridges physical camera to virtual camera via processing pipeline
type VideoBridge struct {
	// Devices
	physicalCamera string // /dev/video0
	virtualCamera  string // /dev/video20

	// V4L2 devices
	v4l2Input  *device.V4L2Device
	v4l2Output *device.V4L2Device

	// Pipeline
	pipeline     *pipeline.Pipeline
	pipelineExec *pipeline.PipelineExecutor

	// Configuration
	width  int
	height int
	format uint32 // V4L2 pixel format
	fps    int

	// Lifecycle
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewVideoBridge creates a new video bridge
func NewVideoBridge(
	physicalCamera string,
	virtualCamera string,
	pipeline *pipeline.Pipeline,
	pipelineExec *pipeline.PipelineExecutor,
	logger *zap.Logger,
) *VideoBridge {
	ctx, cancel := context.WithCancel(context.Background())

	return &VideoBridge{
		physicalCamera: physicalCamera,
		virtualCamera:  virtualCamera,
		pipeline:       pipeline,
		pipelineExec:   pipelineExec,
		width:          640,
		height:         480,
		format:         device.V4L2_PIX_FMT_YUYV,
		fps:            30,
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
	}
}

// Start starts the video bridge
func (vb *VideoBridge) Start() error {
	vb.mu.Lock()
	defer vb.mu.Unlock()

	if vb.running {
		return fmt.Errorf("video bridge already running")
	}

	vb.logger.Info("Starting video bridge",
		zap.String("physical_camera", vb.physicalCamera),
		zap.String("virtual_camera", vb.virtualCamera),
	)

	// Open physical camera
	var err error
	vb.v4l2Input, err = device.OpenV4L2Device(vb.physicalCamera, vb.logger)
	if err != nil {
		return fmt.Errorf("failed to open physical camera: %w", err)
	}

	// Configure input format
	if err := vb.v4l2Input.SetFormat(uint32(vb.width), uint32(vb.height), vb.format); err != nil {
		vb.v4l2Input.Close()
		return fmt.Errorf("failed to set input format: %w", err)
	}

	// Request and map input buffers
	if err := vb.v4l2Input.RequestBuffers(4); err != nil {
		vb.v4l2Input.Close()
		return fmt.Errorf("failed to request input buffers: %w", err)
	}

	if err := vb.v4l2Input.MapBuffers(4); err != nil {
		vb.v4l2Input.Close()
		return fmt.Errorf("failed to map input buffers: %w", err)
	}

	// Start input streaming
	if err := vb.v4l2Input.StartStreaming(); err != nil {
		vb.v4l2Input.Close()
		return fmt.Errorf("failed to start input streaming: %w", err)
	}

	// Open virtual camera (output)
	vb.v4l2Output, err = device.OpenV4L2Device(vb.virtualCamera, vb.logger)
	if err != nil {
		vb.v4l2Input.StopStreaming()
		vb.v4l2Input.Close()
		return fmt.Errorf("failed to open virtual camera: %w", err)
	}

	// Configure output format
	if err := vb.v4l2Output.SetFormat(uint32(vb.width), uint32(vb.height), vb.format); err != nil {
		vb.cleanup()
		return fmt.Errorf("failed to set output format: %w", err)
	}

	// Start frame processing
	vb.wg.Add(1)
	go vb.processVideoFrames()

	vb.running = true
	vb.logger.Info("Video bridge started successfully")
	return nil
}

// Stop stops the video bridge
func (vb *VideoBridge) Stop() error {
	vb.mu.Lock()
	if !vb.running {
		vb.mu.Unlock()
		return nil
	}
	vb.running = false
	vb.mu.Unlock()

	vb.logger.Info("Stopping video bridge")

	// Cancel context to stop goroutines
	vb.cancel()

	// Wait for goroutines to finish
	vb.wg.Wait()

	// Cleanup devices
	vb.cleanup()

	vb.logger.Info("Video bridge stopped")
	return nil
}

// processVideoFrames processes video frames from physical camera to virtual camera
func (vb *VideoBridge) processVideoFrames() {
	defer vb.wg.Done()

	vb.logger.Info("Starting video frame processing")

	frameCount := 0

	for {
		select {
		case <-vb.ctx.Done():
			vb.logger.Info("Video processing stopped (context cancelled)")
			return
		default:
		}

		// Dequeue frame from physical camera
		buf, err := vb.v4l2Input.DequeueBuffer()
		if err != nil {
			vb.logger.Error("Failed to dequeue input buffer", zap.Error(err))
			continue
		}

		// TODO: Get frame data - need to expose V4L2MappedBuffer.data field
		// For now, skip processing to get the code to compile
		// frameData := vb.v4l2Input.buffers[buf.Index].data[:buf.BytesUsed]

		// Process frame through pipeline (if configured)
		// processedFrame, err := vb.processFrame(frameData)
		// if err != nil {
		// 	vb.logger.Error("Failed to process frame", zap.Error(err))
		// 	// Use original frame on error
		// 	processedFrame = frameData
		// }

		// TODO: Write frame to virtual camera - need WriteFrame method
		// if err := vb.v4l2Output.WriteFrame(processedFrame); err != nil {
		// 	vb.logger.Error("Failed to write frame to virtual camera", zap.Error(err))
		// }

		// Re-queue buffer for next frame
		if err := vb.v4l2Input.QueueBuffer(buf.Index); err != nil {
			vb.logger.Error("Failed to queue input buffer", zap.Error(err))
		}

		frameCount++
		if frameCount%100 == 0 {
			vb.logger.Debug("Processed frames", zap.Int("count", frameCount))
		}
	}
}

// processFrame processes a video frame through the pipeline
func (vb *VideoBridge) processFrame(frameData []byte) ([]byte, error) {
	if vb.pipeline == nil {
		// No pipeline configured, pass through
		return frameData, nil
	}

	// Execute pipeline
	result, err := vb.pipelineExec.Execute(vb.ctx, vb.pipeline, frameData)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Extract processed frame from result
	if result.FinalOutput == nil {
		return nil, fmt.Errorf("no output from pipeline")
	}

	// Convert output to bytes
	switch output := result.FinalOutput.(type) {
	case []byte:
		return output, nil
	default:
		vb.logger.Warn("Unexpected pipeline output type", zap.Any("type", output))
		return frameData, nil
	}
}

// cleanup cleans up V4L2 devices
func (vb *VideoBridge) cleanup() {
	if vb.v4l2Input != nil {
		vb.v4l2Input.StopStreaming()
		vb.v4l2Input.Close()
		vb.v4l2Input = nil
	}

	if vb.v4l2Output != nil {
		vb.v4l2Output.Close()
		vb.v4l2Output = nil
	}
}

// IsRunning returns whether the bridge is running
func (vb *VideoBridge) IsRunning() bool {
	vb.mu.RLock()
	defer vb.mu.RUnlock()
	return vb.running
}

// SetPipeline updates the processing pipeline
func (vb *VideoBridge) SetPipeline(pipeline *pipeline.Pipeline) {
	vb.mu.Lock()
	defer vb.mu.Unlock()
	vb.pipeline = pipeline
	vb.logger.Info("Updated video bridge pipeline")
}

// SetFormat updates the video format
func (vb *VideoBridge) SetFormat(width, height int, format uint32, fps int) error {
	vb.mu.Lock()
	defer vb.mu.Unlock()

	if vb.running {
		return fmt.Errorf("cannot change format while bridge is running")
	}

	vb.width = width
	vb.height = height
	vb.format = format
	vb.fps = fps

	vb.logger.Info("Updated video format",
		zap.Int("width", width),
		zap.Int("height", height),
		zap.Int("fps", fps),
	)

	return nil
}
