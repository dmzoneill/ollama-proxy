package virtual

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// V4L2Loopback manages v4l2loopback kernel module and virtual cameras
type V4L2Loopback struct {
	devices      map[string]*V4L2VirtualCamera // device path -> camera
	deviceLabels []string                       // Labels for each video device
	loaded       bool                           // Is module loaded?
	logger       *zap.Logger
}

// V4L2VirtualCamera represents a virtual camera device
type V4L2VirtualCamera struct {
	DevicePath string // /dev/videoN
	DeviceNum  int    // N in /dev/videoN
	Label      string // "Ollama NPU Camera"
	Backend    string // "ollama-npu"
	Width      int
	Height     int
	Format     string // "yuyv", "mjpeg", "rgb24"
	FPS        int
}

// NewV4L2Loopback creates a new v4l2loopback manager
func NewV4L2Loopback(logger *zap.Logger) *V4L2Loopback {
	return &V4L2Loopback{
		devices:      make(map[string]*V4L2VirtualCamera),
		deviceLabels: make([]string, 0),
		loaded:       false,
		logger:       logger,
	}
}

// LoadModule loads the v4l2loopback kernel module
// deviceNums: video device numbers to create (e.g., []int{20, 21, 22, 23})
// labels: device labels corresponding to each device number
func (v *V4L2Loopback) LoadModule(deviceNums []int, labels []string) error {
	if len(deviceNums) != len(labels) {
		return fmt.Errorf("device numbers and labels must have same length")
	}

	if len(deviceNums) == 0 {
		return fmt.Errorf("must specify at least one device")
	}

	// Check if module is already loaded
	if v.isModuleLoaded() {
		v.logger.Info("v4l2loopback module already loaded, unloading first")
		if err := v.UnloadModule(); err != nil {
			v.logger.Warn("Failed to unload existing module", zap.Error(err))
		}
	}

	// Build video_nr parameter (comma-separated list)
	videoNrStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(deviceNums)), ","), "[]")

	// Build card_label parameter (comma-separated list)
	cardLabelStr := strings.Join(labels, ",")

	// Load the module with parameters
	args := []string{
		"v4l2loopback",
		fmt.Sprintf("devices=%d", len(deviceNums)),
		fmt.Sprintf("video_nr=%s", videoNrStr),
		fmt.Sprintf("card_label=%s", cardLabelStr),
		"exclusive_caps=1", // Required for Chrome to see the devices
	}

	v.logger.Info("Loading v4l2loopback module",
		zap.Ints("device_numbers", deviceNums),
		zap.Strings("labels", labels),
	)

	cmd := exec.Command("modprobe", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to load v4l2loopback module: %w (output: %s)", err, string(output))
	}

	v.loaded = true
	v.deviceLabels = labels

	// Wait for devices to appear
	for i, deviceNum := range deviceNums {
		devicePath := fmt.Sprintf("/dev/video%d", deviceNum)
		if err := v.waitForDevice(devicePath); err != nil {
			return fmt.Errorf("device %s did not appear: %w", devicePath, err)
		}

		// Create camera entry
		camera := &V4L2VirtualCamera{
			DevicePath: devicePath,
			DeviceNum:  deviceNum,
			Label:      labels[i],
			Width:      640,
			Height:     480,
			Format:     "yuyv",
			FPS:        30,
		}
		v.devices[devicePath] = camera

		v.logger.Info("Virtual camera device created",
			zap.String("device", devicePath),
			zap.String("label", labels[i]),
		)
	}

	return nil
}

// UnloadModule unloads the v4l2loopback kernel module
func (v *V4L2Loopback) UnloadModule() error {
	if !v.loaded && !v.isModuleLoaded() {
		return nil // Already unloaded
	}

	v.logger.Info("Unloading v4l2loopback module")

	cmd := exec.Command("modprobe", "-r", "v4l2loopback")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unload v4l2loopback module: %w (output: %s)", err, string(output))
	}

	v.loaded = false
	v.devices = make(map[string]*V4L2VirtualCamera)

	v.logger.Info("v4l2loopback module unloaded")
	return nil
}

// GetCamera returns a virtual camera by device path
func (v *V4L2Loopback) GetCamera(devicePath string) (*V4L2VirtualCamera, error) {
	camera, exists := v.devices[devicePath]
	if !exists {
		return nil, fmt.Errorf("camera not found: %s", devicePath)
	}
	return camera, nil
}

// GetCameraByLabel returns a virtual camera by label
func (v *V4L2Loopback) GetCameraByLabel(label string) (*V4L2VirtualCamera, error) {
	for _, camera := range v.devices {
		if camera.Label == label {
			return camera, nil
		}
	}
	return nil, fmt.Errorf("camera not found with label: %s", label)
}

// ListCameras returns all virtual cameras
func (v *V4L2Loopback) ListCameras() []*V4L2VirtualCamera {
	cameras := make([]*V4L2VirtualCamera, 0, len(v.devices))
	for _, camera := range v.devices {
		cameras = append(cameras, camera)
	}
	return cameras
}

// IsLoaded returns whether the module is loaded
func (v *V4L2Loopback) IsLoaded() bool {
	return v.loaded && v.isModuleLoaded()
}

// isModuleLoaded checks if v4l2loopback module is currently loaded
func (v *V4L2Loopback) isModuleLoaded() bool {
	cmd := exec.Command("lsmod")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "v4l2loopback")
}

// waitForDevice waits for a device to appear in /dev
func (v *V4L2Loopback) waitForDevice(devicePath string) error {
	// Simple check - in production might want to poll with timeout
	if _, err := os.Stat(devicePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("device does not exist: %s", devicePath)
		}
		return err
	}
	return nil
}

// VerifyDeviceCapabilities verifies a device has V4L2 capabilities
func (v *V4L2Loopback) VerifyDeviceCapabilities(devicePath string) error {
	// Use v4l2-ctl to verify capabilities
	cmd := exec.Command("v4l2-ctl", "--device", devicePath, "--all")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to query device capabilities: %w (output: %s)", err, string(output))
	}

	// Check for video output capability (required for loopback)
	if !strings.Contains(string(output), "Video Output") {
		return fmt.Errorf("device %s does not support video output", devicePath)
	}

	return nil
}

// SetCameraFormat sets the format for a virtual camera
func (v *V4L2Loopback) SetCameraFormat(devicePath string, width, height int, format string, fps int) error {
	camera, err := v.GetCamera(devicePath)
	if err != nil {
		return err
	}

	camera.Width = width
	camera.Height = height
	camera.Format = format
	camera.FPS = fps

	v.logger.Info("Updated camera format",
		zap.String("device", devicePath),
		zap.Int("width", width),
		zap.Int("height", height),
		zap.String("format", format),
		zap.Int("fps", fps),
	)

	return nil
}

// GetAvailableDeviceNumbers returns video device numbers that are not in use
func GetAvailableDeviceNumbers(count int) ([]int, error) {
	deviceNums := make([]int, 0, count)

	// Start checking from /dev/video20 (avoid conflicts with real cameras)
	startNum := 20
	for i := startNum; i < startNum+100 && len(deviceNums) < count; i++ {
		devicePath := fmt.Sprintf("/dev/video%d", i)
		if _, err := os.Stat(devicePath); os.IsNotExist(err) {
			deviceNums = append(deviceNums, i)
		}
	}

	if len(deviceNums) < count {
		return nil, fmt.Errorf("could not find %d available video device numbers", count)
	}

	return deviceNums, nil
}

// CreateModuleConfig creates a persistent module configuration file
// This allows the module to load automatically on boot
func CreateModuleConfig(deviceNums []int, labels []string, configPath string) error {
	if len(deviceNums) != len(labels) {
		return fmt.Errorf("device numbers and labels must have same length")
	}

	videoNrStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(deviceNums)), ","), "[]")
	cardLabelStr := strings.Join(labels, ",")

	content := fmt.Sprintf(`# v4l2loopback configuration for Ollama Proxy
# Auto-generated - do not edit manually

options v4l2loopback devices=%d video_nr=%s card_label="%s" exclusive_caps=1
`,
		len(deviceNums),
		videoNrStr,
		cardLabelStr,
	)

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ParseDeviceNumber extracts the device number from a device path
// e.g., "/dev/video20" -> 20
func ParseDeviceNumber(devicePath string) (int, error) {
	base := filepath.Base(devicePath)
	if !strings.HasPrefix(base, "video") {
		return 0, fmt.Errorf("invalid video device path: %s", devicePath)
	}

	numStr := strings.TrimPrefix(base, "video")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse device number from %s: %w", devicePath, err)
	}

	return num, nil
}
