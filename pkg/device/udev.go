package device

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

// UdevEvent represents a hotplug event from the kernel
type UdevEvent struct {
	Action     string // "add", "remove", "change"
	DevPath    string // /devices/pci0000:00/...
	Subsystem  string // "video4linux", "sound", "input"
	DevName    string // /dev/video0, /dev/snd/pcmC0D0c
	DevType    string // Device type (if available)
	Properties map[string]string
}

// UdevMonitor monitors kernel udev events via netlink socket
type UdevMonitor struct {
	socket  int
	eventCh chan UdevEvent
	stopCh  chan struct{}
	logger  *zap.Logger
}

const (
	// Netlink socket constants
	NETLINK_KOBJECT_UEVENT = 15
	UEVENT_BUFFER_SIZE     = 8192

	// Subsystems we care about
	SUBSYSTEM_VIDEO4LINUX = "video4linux"
	SUBSYSTEM_SOUND       = "sound"
	SUBSYSTEM_INPUT       = "input"
)

// NewUdevMonitor creates a new udev event monitor
func NewUdevMonitor() (*UdevMonitor, error) {
	// Create netlink socket
	fd, err := syscall.Socket(
		syscall.AF_NETLINK,
		syscall.SOCK_RAW,
		NETLINK_KOBJECT_UEVENT,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create netlink socket: %w", err)
	}

	// Bind to netlink groups (kernel events)
	addr := &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: 1, // Kernel event group
		Pid:    0, // Let kernel assign PID
	}

	if err := syscall.Bind(fd, addr); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to bind netlink socket: %w", err)
	}

	return &UdevMonitor{
		socket:  fd,
		eventCh: make(chan UdevEvent, 100), // Buffered to avoid dropping events
		stopCh:  make(chan struct{}),
		logger:  logging.Logger,
	}, nil
}

// Start begins monitoring udev events
func (um *UdevMonitor) Start() {
	go um.receiveLoop()
}

// Events returns the channel for receiving udev events
func (um *UdevMonitor) Events() <-chan UdevEvent {
	return um.eventCh
}

// Stop stops the udev monitor
func (um *UdevMonitor) Stop() error {
	close(um.stopCh)
	close(um.eventCh)

	if um.socket != 0 {
		return syscall.Close(um.socket)
	}
	return nil
}

// receiveLoop continuously receives events from the netlink socket
func (um *UdevMonitor) receiveLoop() {
	buffer := make([]byte, UEVENT_BUFFER_SIZE)

	for {
		select {
		case <-um.stopCh:
			return
		default:
			// Read from netlink socket
			n, _, err := syscall.Recvfrom(um.socket, buffer, 0)
			if err != nil {
				if err == syscall.EINTR {
					continue // Interrupted, retry
				}
				um.logger.Error("Failed to receive netlink message", zap.Error(err))
				continue
			}

			if n <= 0 {
				continue
			}

			// Parse the uevent message
			event, err := um.parseUevent(buffer[:n])
			if err != nil {
				um.logger.Debug("Failed to parse uevent", zap.Error(err))
				continue
			}

			// Filter for relevant subsystems
			if um.isRelevantEvent(event) {
				select {
				case um.eventCh <- event:
				case <-um.stopCh:
					return
				default:
					um.logger.Warn("Event channel full, dropping event",
						zap.String("action", event.Action),
						zap.String("subsystem", event.Subsystem))
				}
			}
		}
	}
}

// parseUevent parses a raw netlink uevent message
// Format: null-separated key=value pairs
// Example: "add@/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/video4linux/video0\x00
//           ACTION=add\x00DEVPATH=/devices/.../video0\x00SUBSYSTEM=video4linux\x00..."
func (um *UdevMonitor) parseUevent(data []byte) (UdevEvent, error) {
	event := UdevEvent{
		Properties: make(map[string]string),
	}

	// Split by null bytes
	parts := bytes.Split(data, []byte{0})
	if len(parts) < 2 {
		return event, fmt.Errorf("invalid uevent format")
	}

	// First line is the header: "action@devpath"
	header := string(parts[0])
	atIndex := strings.Index(header, "@")
	if atIndex > 0 {
		event.Action = strings.ToLower(header[:atIndex])
		event.DevPath = header[atIndex+1:]
	}

	// Parse key=value pairs
	for _, part := range parts[1:] {
		if len(part) == 0 {
			continue
		}

		line := string(part)
		eqIndex := strings.Index(line, "=")
		if eqIndex <= 0 {
			continue
		}

		key := line[:eqIndex]
		value := line[eqIndex+1:]

		event.Properties[key] = value

		// Extract important fields
		switch key {
		case "ACTION":
			if event.Action == "" {
				event.Action = strings.ToLower(value)
			}
		case "DEVPATH":
			if event.DevPath == "" {
				event.DevPath = value
			}
		case "SUBSYSTEM":
			event.Subsystem = value
		case "DEVNAME":
			event.DevName = value
		case "DEVTYPE":
			event.DevType = value
		}
	}

	return event, nil
}

// isRelevantEvent checks if we care about this event
func (um *UdevMonitor) isRelevantEvent(event UdevEvent) bool {
	// Only process add/remove/change events
	if event.Action != "add" && event.Action != "remove" && event.Action != "change" {
		return false
	}

	// Only process relevant subsystems
	switch event.Subsystem {
	case SUBSYSTEM_VIDEO4LINUX, SUBSYSTEM_SOUND, SUBSYSTEM_INPUT:
		return true
	default:
		return false
	}
}

// String returns a string representation of the event
func (e UdevEvent) String() string {
	return fmt.Sprintf("UdevEvent{Action=%s, DevPath=%s, Subsystem=%s, DevName=%s, DevType=%s}",
		e.Action, e.DevPath, e.Subsystem, e.DevName, e.DevType)
}

// GetDeviceType maps udev subsystem to our DeviceType
func (e UdevEvent) GetDeviceType() DeviceType {
	switch e.Subsystem {
	case SUBSYSTEM_VIDEO4LINUX:
		return DeviceTypeCamera
	case SUBSYSTEM_SOUND:
		// Check if it's capture or playback
		if strings.Contains(e.DevName, "pcmC") && strings.Contains(e.DevName, "c") {
			return DeviceTypeMicrophone
		}
		return DeviceTypeSpeaker
	case SUBSYSTEM_INPUT:
		// Check device type
		if strings.Contains(strings.ToLower(e.DevType), "keyboard") {
			return DeviceTypeKeyboard
		}
		if strings.Contains(strings.ToLower(e.DevType), "mouse") {
			return DeviceTypeMouse
		}
		return DeviceTypeKeyboard // Default for input devices
	default:
		return DeviceTypeCamera // Default fallback
	}
}

// MonitorDevices is a convenience function to monitor and print events
// Useful for debugging
func MonitorDevices() error {
	monitor, err := NewUdevMonitor()
	if err != nil {
		return err
	}
	defer monitor.Stop()

	monitor.Start()

	fmt.Println("Monitoring udev events (Ctrl+C to stop)...")
	fmt.Println("Plug/unplug devices to see events")
	fmt.Println()

	for event := range monitor.Events() {
		fmt.Printf("[%s] %s: %s (%s)\n",
			event.Action,
			event.Subsystem,
			event.DevName,
			event.DevType)

		// Print some properties
		for k, v := range event.Properties {
			if k == "PRODUCT" || k == "ID_MODEL" || k == "ID_VENDOR" {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}
		fmt.Println()
	}

	return nil
}

// Helper function to check if running as root (required for netlink)
func requiresRoot() bool {
	return os.Geteuid() != 0
}

// init checks if we have permission to create netlink socket
func init() {
	// Test netlink socket creation
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, NETLINK_KOBJECT_UEVENT)
	if err == nil {
		syscall.Close(fd)
	} else if err == syscall.EPERM {
		// Log warning but don't fail - let NewUdevMonitor handle it
		logging.Logger.Warn("Netlink socket creation requires elevated privileges (may need CAP_NET_ADMIN or root)")
	}
}

// GetCapabilities queries device capabilities via sysfs
func GetCapabilities(devPath string) map[string]string {
	caps := make(map[string]string)

	// Try to read sysfs attributes
	sysfsPath := "/sys" + devPath

	// Read common attributes
	attrs := []string{"vendor", "product", "model", "name", "manufacturer"}
	for _, attr := range attrs {
		attrPath := fmt.Sprintf("%s/%s", sysfsPath, attr)
		if data, err := os.ReadFile(attrPath); err == nil {
			caps[attr] = strings.TrimSpace(string(data))
		}
	}

	return caps
}

// ParseDeviceName extracts a friendly name from udev properties
func ParseDeviceName(event UdevEvent) string {
	// Try ID_MODEL first
	if model, ok := event.Properties["ID_MODEL"]; ok && model != "" {
		return strings.ReplaceAll(model, "_", " ")
	}

	// Try PRODUCT (format: vendor/product/version)
	if product, ok := event.Properties["PRODUCT"]; ok && product != "" {
		return product
	}

	// Fall back to device name
	if event.DevName != "" {
		return event.DevName
	}

	return "Unknown Device"
}

// SetNonBlocking sets the socket to non-blocking mode
func (um *UdevMonitor) SetNonBlocking() error {
	return syscall.SetNonblock(um.socket, true)
}

// SetReceiveBufferSize sets the socket receive buffer size
func (um *UdevMonitor) SetReceiveBufferSize(size int) error {
	return syscall.SetsockoptInt(um.socket, syscall.SOL_SOCKET, syscall.SO_RCVBUF, size)
}
