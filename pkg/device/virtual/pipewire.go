package virtual

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// PipewireSystem implements AudioSystem for Pipewire
type PipewireSystem struct {
	pwCliPath string
	logger    *zap.Logger
	nodes     map[string]string // node ID -> device name mapping
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

// createSourceViaWpctl creates a virtual source using wpctl and pw-loopback
func (pw *PipewireSystem) createSourceViaWpctl(name, description string, sampleRate, channels int) (string, error) {
	// Use pw-loopback to create a virtual null source
	// pw-loopback creates both a sink (for apps to write to) and a source (for apps to read from)
	args := []string{
		"--capture-props", fmt.Sprintf("node.name=%s media.class=Audio/Source audio.rate=%d audio.channels=%d node.description='%s'",
			name, sampleRate, channels, description),
	}

	cmd := exec.Command("pw-loopback", args...)

	// Run in background - pw-loopback needs to keep running
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start pw-loopback for source: %w", err)
	}

	// Store the process (we'll need to kill it on cleanup)
	nodeID := fmt.Sprintf("pw-loopback-%d", cmd.Process.Pid)
	pw.nodes[nodeID] = name

	pw.logger.Info("Created virtual microphone (pw-loopback)",
		zap.String("name", name),
		zap.String("node_id", nodeID),
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
		zap.Int("pid", cmd.Process.Pid),
	)

	return nodeID, nil
}

// createSourceViaPactl creates a source using PulseAudio compatibility
// Note: Creates a null sink whose .monitor becomes the source (mic)
func (pw *PipewireSystem) createSourceViaPactl(pactlPath, name, description string, sampleRate, channels int) (string, error) {
	// Use module-null-sink - its .monitor will be the source
	args := []string{
		"load-module",
		"module-null-sink",
		fmt.Sprintf("sink_name=%s", name),
		fmt.Sprintf("sink_properties=device.description=\"%s\"", description),
		fmt.Sprintf("channels=%d", channels),
		fmt.Sprintf("rate=%d", sampleRate),
		"format=s16le",
	}

	cmd := exec.Command(pactlPath, args...)
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

// createSourceNative creates a source using native pw-cli
func (pw *PipewireSystem) createSourceNative(name, description string, sampleRate, channels int) (string, error) {
	// Create a null sink source using pw-cli
	// Note: This is more complex and may require scripting
	script := fmt.Sprintf(`
create-node adapter {
	factory.name = support.null-audio-sink
	node.name = "%s"
	node.description = "%s"
	media.class = Audio/Source
	audio.channels = %d
	audio.rate = %d
	audio.format = S16LE
}
`, name, description, channels, sampleRate)

	cmd := exec.Command(pw.pwCliPath)
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create virtual source via pw-cli: %w (output: %s)", err, string(output))
	}

	// Parse node ID from output (format: "Created node <id>")
	nodeID := pw.parseNodeID(string(output))
	if nodeID == "" {
		return "", fmt.Errorf("failed to parse node ID from output: %s", string(output))
	}

	pw.nodes[nodeID] = name

	pw.logger.Info("Created virtual microphone (native pw-cli)",
		zap.String("name", name),
		zap.String("node_id", nodeID),
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

// createSinkViaWpctl creates a virtual sink using wpctl and pw-loopback
func (pw *PipewireSystem) createSinkViaWpctl(name, description string, sampleRate, channels int) (string, error) {
	// Use pw-loopback to create a virtual null sink
	args := []string{
		"--playback-props", fmt.Sprintf("node.name=%s media.class=Audio/Sink audio.rate=%d audio.channels=%d node.description='%s'",
			name, sampleRate, channels, description),
	}

	cmd := exec.Command("pw-loopback", args...)

	// Run in background
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start pw-loopback for sink: %w", err)
	}

	// Store the process
	nodeID := fmt.Sprintf("pw-loopback-%d", cmd.Process.Pid)
	pw.nodes[nodeID] = name

	pw.logger.Info("Created virtual speaker (pw-loopback)",
		zap.String("name", name),
		zap.String("node_id", nodeID),
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
		zap.Int("pid", cmd.Process.Pid),
	)

	return nodeID, nil
}

// createSinkViaPactl creates a sink using PulseAudio compatibility
func (pw *PipewireSystem) createSinkViaPactl(pactlPath, name, description string, sampleRate, channels int) (string, error) {
	args := []string{
		"load-module",
		"module-null-sink",
		fmt.Sprintf("sink_name=%s", name),
		fmt.Sprintf("sink_properties=device.description=\"%s\"", description),
		fmt.Sprintf("channels=%d", channels),
		fmt.Sprintf("rate=%d", sampleRate),
		"format=s16le",
	}

	cmd := exec.Command(pactlPath, args...)
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

// createSinkNative creates a sink using native pw-cli
func (pw *PipewireSystem) createSinkNative(name, description string, sampleRate, channels int) (string, error) {
	script := fmt.Sprintf(`
create-node adapter {
	factory.name = support.null-audio-sink
	node.name = "%s"
	node.description = "%s"
	media.class = Audio/Sink
	audio.channels = %d
	audio.rate = %d
	audio.format = S16LE
}
`, name, description, channels, sampleRate)

	cmd := exec.Command(pw.pwCliPath)
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create virtual sink via pw-cli: %w (output: %s)", err, string(output))
	}

	nodeID := pw.parseNodeID(string(output))
	if nodeID == "" {
		return "", fmt.Errorf("failed to parse node ID from output: %s", string(output))
	}

	pw.nodes[nodeID] = name

	pw.logger.Info("Created virtual speaker (native pw-cli)",
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
		// Try to unload as a pactl module
		cmd := exec.Command(pactlPath, "unload-module", nodeID)
		if err := cmd.Run(); err == nil {
			delete(pw.nodes, nodeID)
			pw.logger.Info("Destroyed virtual device (unloaded pactl module)",
				zap.String("name", deviceName),
				zap.String("module_id", nodeID),
			)
			return nil
		}
	}

	// For pw-loopback devices, kill the process
	if strings.HasPrefix(nodeID, "pw-loopback-") {
		// Extract PID from nodeID
		var pid int
		_, err := fmt.Sscanf(nodeID, "pw-loopback-%d", &pid)
		if err != nil {
			return fmt.Errorf("failed to parse PID from node ID %s: %w", nodeID, err)
		}

		// Kill the pw-loopback process
		cmd := exec.Command("kill", fmt.Sprintf("%d", pid))
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

	return fmt.Errorf("unknown node type for ID: %s", nodeID)
}

// SetProperty sets a property on a Pipewire node
func (pw *PipewireSystem) SetProperty(nodeID, key, value string) error {
	script := fmt.Sprintf(`set-param %s Props { %s = "%s" }`, nodeID, key, value)

	cmd := exec.Command(pw.pwCliPath)
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set property on node %s: %w (output: %s)", nodeID, err, string(output))
	}

	return nil
}

// IsRunning checks if Pipewire is running
func (pw *PipewireSystem) IsRunning() bool {
	cmd := exec.Command(pw.pwCliPath, "info", "0")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// GetType returns "pipewire"
func (pw *PipewireSystem) GetType() string {
	return "pipewire"
}

// parseNodeID extracts node ID from pw-cli output
func (pw *PipewireSystem) parseNodeID(output string) string {
	// Look for patterns like "Created node <id>" or "id: <id>"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Pattern 1: "Created node 123"
		if strings.HasPrefix(line, "Created node ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}

		// Pattern 2: "id: 123"
		if strings.HasPrefix(line, "id:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	return ""
}
