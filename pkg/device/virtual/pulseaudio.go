package virtual

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/cmdexec"
)

// PulseAudioSystem implements AudioSystem for PulseAudio
type PulseAudioSystem struct {
	logger    *zap.Logger
	modules   map[string]string
	pactlPath string
}

// NewPulseAudioSystem creates a new PulseAudio system manager
func NewPulseAudioSystem(logger *zap.Logger) *PulseAudioSystem {
	pactlPath, err := exec.LookPath("pactl")
	if err != nil {
		logger.Warn("pactl not found in PATH", zap.Error(err))
		pactlPath = "pactl" // Try anyway
	}

	return &PulseAudioSystem{
		pactlPath: pactlPath,
		logger:    logger,
		modules:   make(map[string]string),
	}
}

// CreateVirtualSource creates a virtual microphone using module-null-source
func (pa *PulseAudioSystem) CreateVirtualSource(name, description string, sampleRate, channels int) (string, error) {
	// Use module-null-source to create a virtual microphone
	// This creates a source (input device) that applications can read from
	args := []string{
		strLoadModule,
		"module-null-source",
		fmt.Sprintf("source_name=%s", name),
		fmt.Sprintf("source_properties=device.description='%s'", description),
		fmt.Sprintf("channels=%d", channels),
		fmt.Sprintf("rate=%d", sampleRate),
		strFormatS16le, // 16-bit signed little-endian PCM
	}

	cmd, err := cmdexec.Command(pa.pactlPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load module-null-source: %w (output: %s)", err, string(output))
	}

	// PulseAudio returns the module ID
	moduleID := strings.TrimSpace(string(output))
	pa.modules[moduleID] = name

	pa.logger.Info("Created virtual microphone",
		zap.String("name", name),
		zap.String("module_id", moduleID),
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
	)

	return moduleID, nil
}

// CreateVirtualSink creates a virtual speaker using module-null-sink
func (pa *PulseAudioSystem) CreateVirtualSink(name, description string, sampleRate, channels int) (string, error) {
	// Use module-null-sink to create a virtual speaker
	// This creates a sink (output device) that applications can write to
	args := []string{
		strLoadModule,
		strModuleNullSink,
		fmt.Sprintf("sink_name=%s", name),
		fmt.Sprintf("sink_properties=device.description='%s'", description),
		fmt.Sprintf("channels=%d", channels),
		fmt.Sprintf("rate=%d", sampleRate),
		strFormatS16le,
	}

	cmd, err := cmdexec.Command(pa.pactlPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load module-null-sink: %w (output: %s)", err, string(output))
	}

	moduleID := strings.TrimSpace(string(output))
	pa.modules[moduleID] = name

	pa.logger.Info("Created virtual speaker",
		zap.String("name", name),
		zap.String("module_id", moduleID),
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
	)

	return moduleID, nil
}

// DestroyDevice unloads a PulseAudio module by ID
func (pa *PulseAudioSystem) DestroyDevice(moduleID string) error {
	deviceName, exists := pa.modules[moduleID]
	if !exists {
		return fmt.Errorf("unknown module ID: %s", moduleID)
	}

	cmd, err := cmdexec.Command(pa.pactlPath, "unload-module", moduleID)
	if err != nil {
		return fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unload module %s: %w (output: %s)", moduleID, err, string(output))
	}

	delete(pa.modules, moduleID)

	pa.logger.Info("Destroyed virtual device",
		zap.String("name", deviceName),
		zap.String("module_id", moduleID),
	)

	return nil
}

// SetProperty sets a property on a PulseAudio device
func (pa *PulseAudioSystem) SetProperty(deviceName, key, value string) error {
	// Determine if it's a source or sink by checking if it exists
	cmd, err := cmdexec.Command(pa.pactlPath, "list", "sources", "short")
	if err != nil {
		return fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), deviceName) {
		// It's a source
		cmd, err = cmdexec.Command(pa.pactlPath, "set-source-", deviceName, key, value)
		if err != nil {
			return fmt.Errorf("failed to create pactl command: %w", err)
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set source property: %w", err)
		}
		return nil
	}

	// Try sink
	cmd, err = cmdexec.Command(pa.pactlPath, "list", "sinks", "short")
	if err != nil {
		return fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err = cmd.Output()
	if err == nil && strings.Contains(string(output), deviceName) {
		cmd, err = cmdexec.Command(pa.pactlPath, "set-sink-property", deviceName, key, value)
		if err != nil {
			return fmt.Errorf("failed to create pactl command: %w", err)
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set sink property: %w", err)
		}
		return nil
	}

	return fmt.Errorf("device not found: %s", deviceName)
}

// IsRunning checks if PulseAudio is running
func (pa *PulseAudioSystem) IsRunning() bool {
	cmd, err := cmdexec.Command(pa.pactlPath, "info")
	if err != nil {
		return false
	}
	return cmd.Run() == nil
}

// GetType returns "pulseaudio"
func (pa *PulseAudioSystem) GetType() string {
	return "pulseaudio"
}

// ListSources lists all PulseAudio sources (for debugging)
func (pa *PulseAudioSystem) ListSources() ([]string, error) {
	cmd, err := cmdexec.Command(pa.pactlPath, "list", "sources", "short")
	if err != nil {
		return nil, fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	sources := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sources = append(sources, line)
		}
	}

	return sources, nil
}

// ListSinks lists all PulseAudio sinks (for debugging)
func (pa *PulseAudioSystem) ListSinks() ([]string, error) {
	cmd, err := cmdexec.Command(pa.pactlPath, "list", "sinks", "short")
	if err != nil {
		return nil, fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	sinks := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sinks = append(sinks, line)
		}
	}

	return sinks, nil
}

// GetModuleID returns the module ID for a device name
func (pa *PulseAudioSystem) GetModuleID(deviceName string) (string, bool) {
	for moduleID, name := range pa.modules {
		if name == deviceName {
			return moduleID, true
		}
	}
	return "", false
}
