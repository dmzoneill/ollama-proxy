package virtual

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/device"
	"github.com/daoneill/ollama-proxy/pkg/pipeline"
)

// VirtualDeviceManager manages virtual audio and video devices
type VirtualDeviceManager struct {
	audioSystem     AudioSystem
	videoSystem     *V4L2Loopback
	deviceManager   *device.DeviceManager
	pipelineExec    *pipeline.PipelineExecutor

	// Virtual devices
	audioSources    map[string]*VirtualAudioDevice  // mic devices (backend -> device)
	audioSinks      map[string]*VirtualAudioDevice  // speaker devices (backend -> device)
	cameras         map[string]*VirtualVideoDevice  // camera devices (backend -> device)

	// Data bridges
	audioBridges    map[string]*AudioBridge         // backend -> audio bridge
	videoBridges    map[string]*VideoBridge         // backend -> video bridge
	meetingBridges  map[string]*MeetingAudioBridge  // backend -> meeting bridge

	// Backend registry (for meeting bridges)
	backends        map[string]interface{} // backend ID -> backend interface

	// Configuration
	config          *Config

	// Lifecycle
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	logger          *zap.Logger
}

// VirtualAudioDevice represents a virtual microphone or speaker
type VirtualAudioDevice struct {
	ID             string
	Name           string
	Description    string
	Type           string // "source" (mic) or "sink" (speaker)
	Backend        string // "ollama-npu", "ollama-igpu", etc.
	SystemID       string // Module/node ID from audio system
	SampleRate     int
	Channels       int
	Format         string
	IsRunning      bool
}

// VirtualVideoDevice represents a virtual camera
type VirtualVideoDevice struct {
	ID             string
	Label          string
	Backend        string
	DevicePath     string
	DeviceNum      int
	Width          int
	Height         int
	Format         string
	FPS            int
	IsRunning      bool
}

// NewVirtualDeviceManager creates a new virtual device manager
func NewVirtualDeviceManager(
	deviceManager *device.DeviceManager,
	pipelineExec *pipeline.PipelineExecutor,
	config *Config,
	logger *zap.Logger,
) (*VirtualDeviceManager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	vdm := &VirtualDeviceManager{
		deviceManager:  deviceManager,
		pipelineExec:   pipelineExec,
		audioSources:   make(map[string]*VirtualAudioDevice),
		audioSinks:     make(map[string]*VirtualAudioDevice),
		cameras:        make(map[string]*VirtualVideoDevice),
		audioBridges:   make(map[string]*AudioBridge),
		videoBridges:   make(map[string]*VideoBridge),
		meetingBridges: make(map[string]*MeetingAudioBridge),
		backends:       make(map[string]interface{}),
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
	}

	// Initialize audio system
	if config.Audio.Enabled {
		audioSys, err := DetectAudioSystem(logger)
		if err != nil {
			logger.Warn("Failed to detect audio system", zap.Error(err))
			if !config.Audio.AllowFallback {
				cancel()
				return nil, fmt.Errorf("audio system required but not available: %w", err)
			}
		} else {
			vdm.audioSystem = audioSys
			logger.Info("Audio system initialized",
				zap.String("type", audioSys.GetType()),
			)
		}
	}

	// Initialize video system
	if config.Video.Enabled {
		vdm.videoSystem = NewV4L2Loopback(logger)
		logger.Info("Video system initialized (v4l2loopback)")
	}

	logger.Info("Virtual device manager created")
	return vdm, nil
}

// CreateDevicesForBackend creates all virtual devices for a specific backend
func (vdm *VirtualDeviceManager) CreateDevicesForBackend(backendID, backendName, hardware string) error {
	vdm.mu.Lock()
	defer vdm.mu.Unlock()

	vdm.logger.Info("Creating virtual devices for backend",
		zap.String("backend_id", backendID),
		zap.String("hardware", hardware),
	)

	// Create microphone
	if vdm.config.Audio.Enabled && vdm.config.Audio.Microphone.Enabled {
		if err := vdm.createAudioSource(backendID, backendName, hardware); err != nil {
			vdm.logger.Error("Failed to create microphone",
				zap.String("backend", backendID),
				zap.Error(err),
			)
			return err
		}
	}

	// Create speaker
	if vdm.config.Audio.Enabled && vdm.config.Audio.Speaker.Enabled {
		if err := vdm.createAudioSink(backendID, backendName, hardware); err != nil {
			vdm.logger.Error("Failed to create speaker",
				zap.String("backend", backendID),
				zap.Error(err),
			)
			return err
		}
	}

	return nil
}

// createAudioSource creates a virtual microphone for a backend
func (vdm *VirtualDeviceManager) createAudioSource(backendID, backendName, hardware string) error {
	if vdm.audioSystem == nil {
		return fmt.Errorf("audio system not available")
	}

	name := fmt.Sprintf("ollama-%s-mic", hardware)
	description := strings.ReplaceAll(fmt.Sprintf("Ollama-%s-Microphone", backendName), " ", "-")

	systemID, err := vdm.audioSystem.CreateVirtualSource(
		name,
		description,
		vdm.config.Audio.Microphone.SampleRate,
		vdm.config.Audio.Microphone.Channels,
	)
	if err != nil {
		return fmt.Errorf("failed to create virtual source: %w", err)
	}

	virtualDevice := &VirtualAudioDevice{
		ID:          fmt.Sprintf("vmic-%s", backendID),
		Name:        name,
		Description: description,
		Type:        "source",
		Backend:     backendID,
		SystemID:    systemID,
		SampleRate:  vdm.config.Audio.Microphone.SampleRate,
		Channels:    vdm.config.Audio.Microphone.Channels,
		Format:      vdm.config.Audio.Microphone.Format,
		IsRunning:   false,
	}

	vdm.audioSources[backendID] = virtualDevice

	vdm.logger.Info("Created virtual microphone",
		zap.String("backend", backendID),
		zap.String("name", name),
		zap.String("system_id", systemID),
	)

	return nil
}

// createAudioSink creates a virtual speaker for a backend
func (vdm *VirtualDeviceManager) createAudioSink(backendID, backendName, hardware string) error {
	if vdm.audioSystem == nil {
		return fmt.Errorf("audio system not available")
	}

	name := fmt.Sprintf("ollama-%s-speaker", hardware)
	description := strings.ReplaceAll(fmt.Sprintf("Ollama-%s-Speaker", backendName), " ", "-")

	systemID, err := vdm.audioSystem.CreateVirtualSink(
		name,
		description,
		vdm.config.Audio.Speaker.SampleRate,
		vdm.config.Audio.Speaker.Channels,
	)
	if err != nil {
		return fmt.Errorf("failed to create virtual sink: %w", err)
	}

	virtualDevice := &VirtualAudioDevice{
		ID:          fmt.Sprintf("vspk-%s", backendID),
		Name:        name,
		Description: description,
		Type:        "sink",
		Backend:     backendID,
		SystemID:    systemID,
		SampleRate:  vdm.config.Audio.Speaker.SampleRate,
		Channels:    vdm.config.Audio.Speaker.Channels,
		Format:      vdm.config.Audio.Speaker.Format,
		IsRunning:   false,
	}

	vdm.audioSinks[backendID] = virtualDevice

	vdm.logger.Info("Created virtual speaker",
		zap.String("backend", backendID),
		zap.String("name", name),
		zap.String("system_id", systemID),
	)

	return nil
}

// CreateVirtualCameras creates all virtual cameras at once
// This must be done together because v4l2loopback requires module loading
func (vdm *VirtualDeviceManager) CreateVirtualCameras(backends []BackendInfo) error {
	if !vdm.config.Video.Enabled || !vdm.config.Video.Camera.Enabled {
		return nil
	}

	if vdm.videoSystem == nil {
		return fmt.Errorf("video system not available")
	}

	vdm.mu.Lock()
	defer vdm.mu.Unlock()

	// Build device numbers and labels
	deviceNums := make([]int, 0, len(backends))
	labels := make([]string, 0, len(backends))

	for i, backend := range backends {
		if i < len(vdm.config.Video.Camera.DeviceNumbers) {
			deviceNums = append(deviceNums, vdm.config.Video.Camera.DeviceNumbers[i])
		} else {
			// Auto-assign starting from 20
			deviceNums = append(deviceNums, 20+i)
		}

		label := fmt.Sprintf("Ollama %s Camera", backend.Name)
		labels = append(labels, label)
	}

	// Load v4l2loopback module
	if err := vdm.videoSystem.LoadModule(deviceNums, labels); err != nil {
		return fmt.Errorf("failed to load v4l2loopback: %w", err)
	}

	// Create virtual device entries
	for i, backend := range backends {
		devicePath := fmt.Sprintf("/dev/video%d", deviceNums[i])

		virtualDevice := &VirtualVideoDevice{
			ID:         fmt.Sprintf("vcam-%s", backend.ID),
			Label:      labels[i],
			Backend:    backend.ID,
			DevicePath: devicePath,
			DeviceNum:  deviceNums[i],
			Width:      vdm.config.Video.Camera.Width,
			Height:     vdm.config.Video.Camera.Height,
			Format:     vdm.config.Video.Camera.Format,
			FPS:        vdm.config.Video.Camera.FPS,
			IsRunning:  false,
		}

		vdm.cameras[backend.ID] = virtualDevice

		vdm.logger.Info("Created virtual camera",
			zap.String("backend", backend.ID),
			zap.String("device", devicePath),
			zap.String("label", labels[i]),
		)
	}

	return nil
}

// Stop stops all virtual devices and cleans up
func (vdm *VirtualDeviceManager) Stop() error {
	vdm.mu.Lock()
	defer vdm.mu.Unlock()

	vdm.logger.Info("Stopping virtual device manager")

	// Stop all meeting bridges
	for backendID, bridge := range vdm.meetingBridges {
		if err := bridge.Stop(); err != nil {
			vdm.logger.Warn("Failed to stop meeting bridge",
				zap.String("backend", backendID),
				zap.Error(err),
			)
		}
	}

	// Stop all audio bridges
	for backendID, bridge := range vdm.audioBridges {
		if err := bridge.Stop(); err != nil {
			vdm.logger.Warn("Failed to stop audio bridge",
				zap.String("backend", backendID),
				zap.Error(err),
			)
		}
	}

	// Stop all video bridges
	for backendID, bridge := range vdm.videoBridges {
		if err := bridge.Stop(); err != nil {
			vdm.logger.Warn("Failed to stop video bridge",
				zap.String("backend", backendID),
				zap.Error(err),
			)
		}
	}

	// Destroy audio sources
	if vdm.audioSystem != nil {
		for backendID, device := range vdm.audioSources {
			if err := vdm.audioSystem.DestroyDevice(device.SystemID); err != nil {
				vdm.logger.Warn("Failed to destroy audio source",
					zap.String("backend", backendID),
					zap.Error(err),
				)
			}
		}

		// Destroy audio sinks
		for backendID, device := range vdm.audioSinks {
			if err := vdm.audioSystem.DestroyDevice(device.SystemID); err != nil {
				vdm.logger.Warn("Failed to destroy audio sink",
					zap.String("backend", backendID),
					zap.Error(err),
				)
			}
		}
	}

	// Unload v4l2loopback module
	if vdm.videoSystem != nil && vdm.videoSystem.IsLoaded() {
		if err := vdm.videoSystem.UnloadModule(); err != nil {
			vdm.logger.Warn("Failed to unload v4l2loopback", zap.Error(err))
		}
	}

	// Cancel context
	vdm.cancel()

	vdm.logger.Info("Virtual device manager stopped")
	return nil
}

// GetSourceCount returns the number of virtual microphones
func (vdm *VirtualDeviceManager) GetSourceCount() int {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()
	return len(vdm.audioSources)
}

// GetSinkCount returns the number of virtual speakers
func (vdm *VirtualDeviceManager) GetSinkCount() int {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()
	return len(vdm.audioSinks)
}

// GetCameraCount returns the number of virtual cameras
func (vdm *VirtualDeviceManager) GetCameraCount() int {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()
	return len(vdm.cameras)
}

// ListAudioSources returns all virtual microphones
func (vdm *VirtualDeviceManager) ListAudioSources() []*VirtualAudioDevice {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()

	devices := make([]*VirtualAudioDevice, 0, len(vdm.audioSources))
	for _, device := range vdm.audioSources {
		devices = append(devices, device)
	}
	return devices
}

// ListAudioSinks returns all virtual speakers
func (vdm *VirtualDeviceManager) ListAudioSinks() []*VirtualAudioDevice {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()

	devices := make([]*VirtualAudioDevice, 0, len(vdm.audioSinks))
	for _, device := range vdm.audioSinks {
		devices = append(devices, device)
	}
	return devices
}

// ListCameras returns all virtual cameras
func (vdm *VirtualDeviceManager) ListCameras() []*VirtualVideoDevice {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()

	cameras := make([]*VirtualVideoDevice, 0, len(vdm.cameras))
	for _, camera := range vdm.cameras {
		cameras = append(cameras, camera)
	}
	return cameras
}

// BackendInfo contains backend information for device creation
type BackendInfo struct {
	ID       string
	Name     string
	Hardware string
}

// RegisterBackend registers a backend for use in meeting bridges
func (vdm *VirtualDeviceManager) RegisterBackend(backendID string, backend interface{}) {
	vdm.mu.Lock()
	defer vdm.mu.Unlock()

	vdm.backends[backendID] = backend
	vdm.logger.Debug("Registered backend for virtual devices",
		zap.String("backend_id", backendID),
	)
}

// StartMeetingBridge starts a meeting audio bridge for a specific backend
// This enables the Google Meet AI assistant functionality
func (vdm *VirtualDeviceManager) StartMeetingBridge(backendID string) error {
	vdm.mu.Lock()
	defer vdm.mu.Unlock()

	// Check if devices exist for this backend
	micDevice, micExists := vdm.audioSources[backendID]
	speakerDevice, speakerExists := vdm.audioSinks[backendID]

	if !micExists || !speakerExists {
		return fmt.Errorf("audio devices not found for backend %s", backendID)
	}

	// Check if bridge already running
	if existing, exists := vdm.meetingBridges[backendID]; exists {
		if existing.IsRunning() {
			return fmt.Errorf("meeting bridge already running for backend %s", backendID)
		}
	}

	// Get backends from registry
	// For meeting bridge, we need: STT backend, LLM backend, TTS backend
	// Use NPU backend for STT/TTS (whisper.cpp + piper via subprocess)
	backendInterface, exists := vdm.backends[backendID]
	if !exists {
		return fmt.Errorf("backend %s not registered", backendID)
	}

	// Type assert to audio backend interface
	sttBackend, ok := backendInterface.(interface {
		ID() string
		SupportsAudioToText() bool
		SupportsTextToAudio() bool
		TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error)
		SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error)
	})
	if !ok {
		return fmt.Errorf("backend %s does not support audio processing", backendID)
	}

	// Use same backend for TTS
	ttsBackend := sttBackend

	// Find LLM backend (prefer OpenVINO CPU for best performance)
	// Based on Xenith analysis: CPU with OpenVINO INT4 is 12x faster than NPU
	var llmBackendID string

	// First preference: OpenVINO CPU (200ms, INT4 optimized)
	if _, exists := vdm.backends["openvino-cpu"]; exists {
		llmBackendID = "openvino-cpu"
	} else {
		// Second preference: iGPU (if OpenVINO not available)
		for id := range vdm.backends {
			if strings.Contains(id, "igpu") {
				llmBackendID = id
				break
			}
		}
	}

	if llmBackendID == "" {
		// Fallback to requested backend
		llmBackendID = backendID
	}

	llmBackendInterface, exists := vdm.backends[llmBackendID]
	if !exists {
		return fmt.Errorf("LLM backend %s not registered", llmBackendID)
	}

	llmBackend, ok := llmBackendInterface.(interface {
		ID() string
		SupportsGenerate() bool
		Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error)
		GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error)
	})
	if !ok {
		return fmt.Errorf("backend %s does not support text generation", llmBackendID)
	}

	// Audio models are handled via subprocess (whisper.cpp + Piper)
	// For Ollama backends: Ensure LLM model is available
	// For OpenVINO backends: Model is already on disk
	if llmBackendID != "openvino-cpu" {
		vdm.logger.Info("Ensuring LLM model available for meeting bridge",
			zap.String("llm_backend", llmBackendID),
			zap.String("model", "qwen2.5:0.5b"),
		)

		if ensurer, ok := llmBackend.(interface {
			EnsureModel(ctx context.Context, modelName string) error
		}); ok {
			ctx := context.Background()
			if err := ensurer.EnsureModel(ctx, "qwen2.5:0.5b"); err != nil {
				vdm.logger.Warn("Failed to ensure LLM model",
					zap.String("model", "qwen2.5:0.5b"),
					zap.Error(err),
				)
			}
		}
	} else {
		vdm.logger.Info("Using OpenVINO LLM backend for meeting bridge",
			zap.String("llm_backend", llmBackendID),
			zap.String("model", "qwen2.5-1.5b-int4"),
			zap.String("device", "CPU"),
			zap.String("note", "INT4 optimized, 200ms target latency"),
		)
	}

	// Create meeting bridge
	// Speaker monitor name is the speaker sink name + ".monitor"
	speakerMonitor := speakerDevice.Name + ".monitor"

	bridge := NewMeetingAudioBridge(
		speakerMonitor,
		micDevice.Name,
		sttBackend,  // NPU/CPU backend (whisper.cpp)
		llmBackend,  // OpenVINO CPU (qwen2.5-1.5b-int4) or Ollama iGPU fallback
		ttsBackend,  // NPU/CPU backend (piper)
		vdm.pipelineExec,
		vdm.logger,
	)

	// Start the bridge
	if err := bridge.Start(); err != nil {
		return fmt.Errorf("failed to start meeting bridge: %w", err)
	}

	vdm.meetingBridges[backendID] = bridge

	vdm.logger.Info("Started meeting audio bridge",
		zap.String("backend_id", backendID),
		zap.String("speaker_monitor", speakerMonitor),
		zap.String("microphone_sink", micDevice.Name),
	)

	return nil
}

// StopMeetingBridge stops a meeting audio bridge for a specific backend
func (vdm *VirtualDeviceManager) StopMeetingBridge(backendID string) error {
	vdm.mu.Lock()
	defer vdm.mu.Unlock()

	bridge, exists := vdm.meetingBridges[backendID]
	if !exists {
		return fmt.Errorf("meeting bridge not found for backend %s", backendID)
	}

	if err := bridge.Stop(); err != nil {
		return fmt.Errorf("failed to stop meeting bridge: %w", err)
	}

	delete(vdm.meetingBridges, backendID)

	vdm.logger.Info("Stopped meeting audio bridge",
		zap.String("backend_id", backendID),
	)

	return nil
}

// GetMeetingBridgeStatus returns the status of a meeting bridge
func (vdm *VirtualDeviceManager) GetMeetingBridgeStatus(backendID string) (bool, error) {
	vdm.mu.RLock()
	defer vdm.mu.RUnlock()

	bridge, exists := vdm.meetingBridges[backendID]
	if !exists {
		return false, nil
	}

	return bridge.IsRunning(), nil
}
