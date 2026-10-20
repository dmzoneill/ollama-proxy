package virtual

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/cmdexec"
)

// PipewireSystem implements AudioSystem for Pipewire
type PipewireSystem struct {
	logger    *zap.Logger
	nodes     map[string]string
	pwCliPath string
}

// NewPipewireSystem creates a new Pipewire system manager
func NewPipewireSystem(logger *zap.Logger) *PipewireSystem {
	pwCliPath, err := exec.LookPath("pw-cli")
	if err != nil {
		logger.Warn("pw-cli not found in PATH", zap.Error(err))
		pwCliPath = "pw-cli" // Try anyway
	}

	return &PipewireSystem{
		pwCliPath: pwCliPath,
		logger:    logger,
		nodes:     make(map[string]string),
	}
}

// CreateVirtualSource creates a virtual microphone using Pipewire
func (pw *PipewireSystem) CreateVirtualSource(name, description string, sampleRate, channels int) (string, error) {
	// Use pactl via pipewire-pulse compatibility layer
	pactlPath, err := exec.LookPath("pactl")
	if err != nil {
		return "", fmt.Errorf("pactl not found: %w", err)
	}
	return pw.createSourceViaPactl(pactlPath, name, description, sampleRate, channels)
}

// createSourceViaPactl creates a source using PulseAudio compatibility
// Note: Creates a null sink whose .monitor becomes the source (mic)
func (pw *PipewireSystem) createSourceViaPactl(pactlPath, name, description string, sampleRate, channels int) (string, error) {
	// Use module-null-sink - its .monitor will be the source
	args := []string{
		strLoadModule,
		strModuleNullSink,
		fmt.Sprintf("sink_name=%s", name),
		fmt.Sprintf("sink_properties=device.description=\"%s\"", description),
		fmt.Sprintf("channels=%d", channels),
		fmt.Sprintf("rate=%d", sampleRate),
		strFormatS16le,
	}

	cmd, err := cmdexec.Command(pactlPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create virtual source via pactl: %w (output: %s)", err, string(output))
	}

	nodeID := strings.TrimSpace(string(output))
	pw.nodes[nodeID] = name

	pw.logger.Info("Created virtual microphone (via pactl null-sink monitor)",
		zap.String("name", name),
		zap.String("node_id", nodeID),
		zap.String("monitor_source", name+".monitor"),
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
	)

	return nodeID, nil
}

// CreateVirtualSink creates a virtual speaker using Pipewire
func (pw *PipewireSystem) CreateVirtualSink(name, description string, sampleRate, channels int) (string, error) {
	// Use pactl via pipewire-pulse compatibility layer
	pactlPath, err := exec.LookPath("pactl")
	if err != nil {
		return "", fmt.Errorf("pactl not found: %w", err)
	}
	return pw.createSinkViaPactl(pactlPath, name, description, sampleRate, channels)
}

// createSinkViaPactl creates a sink using PulseAudio compatibility
func (pw *PipewireSystem) createSinkViaPactl(pactlPath, name, description string, sampleRate, channels int) (string, error) {
	args := []string{
		strLoadModule,
		strModuleNullSink,
		fmt.Sprintf("sink_name=%s", name),
		fmt.Sprintf("sink_properties=device.description=\"%s\"", description),
		fmt.Sprintf("channels=%d", channels),
		fmt.Sprintf("rate=%d", sampleRate),
		strFormatS16le,
	}

	cmd, err := cmdexec.Command(pactlPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create pactl command: %w", err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create virtual sink via pactl: %w (output: %s)", err, string(output))
	}

	nodeID := strings.TrimSpace(string(output))
	pw.nodes[nodeID] = name

	pw.logger.Info("Created virtual speaker (via pactl)",
		zap.String("name", name),
		zap.String("node_id", nodeID),
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
	)

	return nodeID, nil
}

// DestroyDevice destroys a Pipewire node
func (pw *PipewireSystem) DestroyDevice(nodeID string) error {
	deviceName, exists := pw.nodes[nodeID]
	if !exists {
		return fmt.Errorf("unknown node ID: %s", nodeID)
	}

	// For pactl modules (numeric IDs), unload the module
	pactlPath, err := exec.LookPath("pactl")
	if err == nil {
		if unloadErr := pw.unloadPactlModule(pactlPath, nodeID, deviceName); unloadErr == nil {
			return nil
		}
	}

	// For pw-loopback devices, kill the process
	if strings.HasPrefix(nodeID, "pw-loopback-") {
		return pw.killLoopback(nodeID, deviceName)
	}

	return fmt.Errorf("unknown node type for ID: %s", nodeID)
}

func (pw *PipewireSystem) unloadPactlModule(pactlPath, nodeID, deviceName string) error {
	cmd, err := cmdexec.Command(pactlPath, "unload-module", nodeID)
	if err != nil {
		return fmt.Errorf("failed to create pactl command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	delete(pw.nodes, nodeID)
	pw.logger.Info("Destroyed virtual device (unloaded pactl module)",
		zap.String("name", deviceName),
		zap.String("module_id", nodeID),
	)
	return nil
}

func (pw *PipewireSystem) killLoopback(nodeID, deviceName string) error {
	var pid int
	_, err := fmt.Sscanf(nodeID, "pw-loopback-%d", &pid)
	if err != nil {
		return fmt.Errorf("failed to parse PID from node ID %s: %w", nodeID, err)
	}

	cmd, err := cmdexec.Command("kill", fmt.Sprintf("%d", pid))
	if err != nil {
		return fmt.Errorf("failed to create kill command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		pw.logger.Warn("Failed to kill pw-loopback process",
			zap.Int("pid", pid),
			zap.Error(err),
		)
	}

	delete(pw.nodes, nodeID)

	pw.logger.Info("Destroyed virtual device (killed pw-loopback)",
		zap.String("name", deviceName),
		zap.String("node_id", nodeID),
		zap.Int("pid", pid),
	)

	return nil
}

// SetProperty sets a property on a Pipewire node
func (pw *PipewireSystem) SetProperty(nodeID, key, value string) error {
	script := fmt.Sprintf(`set-param %s Props { %s = "%s" }`, nodeID, key, value)

	cmd, err := cmdexec.Command(pw.pwCliPath)
	if err != nil {
		return fmt.Errorf("failed to create pw-cli command: %w", err)
	}
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set property on node %s: %w (output: %s)", nodeID, err, string(output))
	}

	return nil
}

// IsRunning checks if Pipewire is running
func (pw *PipewireSystem) IsRunning() bool {
	cmd, err := cmdexec.Command(pw.pwCliPath, "info", "0")
	if err != nil {
		return false
	}
	return cmd.Run() == nil
}

// GetType returns "pipewire"
func (pw *PipewireSystem) GetType() string {
	return "pipewire"
}
