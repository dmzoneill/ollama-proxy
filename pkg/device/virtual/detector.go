package virtual

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// AudioSystem is an interface for managing virtual audio devices
// Implementations: PulseAudio, Pipewire
type AudioSystem interface {
	// CreateVirtualSource creates a virtual microphone device
	// Returns the module/node ID for later cleanup
	CreateVirtualSource(name, description string, sampleRate, channels int) (string, error)

	// CreateVirtualSink creates a virtual speaker device
	// Returns the module/node ID for later cleanup
	CreateVirtualSink(name, description string, sampleRate, channels int) (string, error)

	// DestroyDevice removes a virtual device by its ID
	DestroyDevice(id string) error

	// SetProperty sets a property on a device
	SetProperty(id, key, value string) error

	// IsRunning checks if the audio system is running
	IsRunning() bool

	// GetType returns the audio system type (pulseaudio or pipewire)
	GetType() string
}

// DetectAudioSystem auto-detects the running audio system
// Tries Pipewire first (modern, lower latency), falls back to PulseAudio
func DetectAudioSystem(logger *zap.Logger) (AudioSystem, error) {
	// Check for Pipewire first (modern systems)
	if isPipewireRunning() {
		logger.Info("Detected Pipewire audio system")
		return NewPipewireSystem(logger), nil
	}

	// Fall back to PulseAudio (most common)
	if isPulseAudioRunning() {
		logger.Info("Detected PulseAudio audio system")
		return NewPulseAudioSystem(logger), nil
	}

	return nil, fmt.Errorf("no supported audio system detected (tried Pipewire, PulseAudio)")
}

// isPipewireRunning checks if Pipewire is running
func isPipewireRunning() bool {
	// Method 1: Check if pw-cli exists and can connect
	cmd := exec.Command("pw-cli", "info", "0")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Method 2: Check for pipewire process
	cmd = exec.Command("pgrep", "-x", "pipewire")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Method 3: Check if PipeWire socket exists
	cmd = exec.Command("test", "-S", "/run/user/1000/pipewire-0")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// isPulseAudioRunning checks if PulseAudio is running
func isPulseAudioRunning() bool {
	// Method 1: Try pactl info
	cmd := exec.Command("pactl", "info")
	output, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), "Server Name:") {
		return true
	}

	// Method 2: Check for pulseaudio process
	cmd = exec.Command("pgrep", "-x", "pulseaudio")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// GetAudioSystemBinary returns the path to pactl or pw-cli
func GetAudioSystemBinary(audioType string) (string, error) {
	var binaryName string
	switch audioType {
	case "pulseaudio":
		binaryName = "pactl"
	case "pipewire":
		binaryName = "pw-cli"
	default:
		return "", fmt.Errorf("unknown audio system type: %s", audioType)
	}

	path, err := exec.LookPath(binaryName)
	if err != nil {
		return "", fmt.Errorf("%s not found in PATH: %w", binaryName, err)
	}

	return path, nil
}
