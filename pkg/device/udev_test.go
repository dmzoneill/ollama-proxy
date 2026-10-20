package device

import (
	"strings"
	"testing"
	"time"
)

// TestUdevEvent_GetDeviceType is part of the public API.
func TestUdevEvent_GetDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		event    UdevEvent
		expected Type
	}{
		{
			name: "Video4Linux camera",
			event: UdevEvent{
				Subsystem: SubsystemVideo4linux,
				DevName:   strDevVideo0,
			},
			expected: DeviceTypeCamera,
		},
		{
			name: "Sound capture (microphone)",
			event: UdevEvent{
				Subsystem: SubsystemSound,
				DevName:   "/dev/snd/pcmC0D0c",
			},
			expected: DeviceTypeMicrophone,
		},
		{
			name: "Sound playback (speaker)",
			event: UdevEvent{
				Subsystem: SubsystemSound,
				DevName:   "/dev/snd/pcmC0D0p",
			},
			expected: DeviceTypeSpeaker,
		},
		{
			name: "Keyboard input",
			event: UdevEvent{
				Subsystem: SubsystemInput,
				DevType:   string(DeviceTypeKeyboard),
			},
			expected: DeviceTypeKeyboard,
		},
		{
			name: "Mouse input",
			event: UdevEvent{
				Subsystem: SubsystemInput,
				DevType:   string(DeviceTypeMouse),
			},
			expected: DeviceTypeMouse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.GetDeviceType()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestUdevEvent_String is part of the public API.
func TestUdevEvent_String(t *testing.T) {
	event := UdevEvent{
		Action:    strAdd,
		DevPath:   "/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/video4linux/video0",
		Subsystem: SubsystemVideo4linux,
		DevName:   strDevVideo0,
		DevType:   "video_device",
	}

	str := event.String()

	if !strings.Contains(str, strAdd) {
		t.Error("String should contain action")
	}

	if !strings.Contains(str, SubsystemVideo4linux) {
		t.Error("String should contain subsystem")
	}

	if !strings.Contains(str, strDevVideo0) {
		t.Error("String should contain device name")
	}
}

// TestParseUevent is part of the public API.
func TestParseUevent(t *testing.T) {
	// Note: This test creates a mock UdevMonitor just for parsing
	// We don't actually start it (which would require root/CAP_NET_ADMIN)

	um := &UdevMonitor{}

	// Simulate a raw uevent message
	// Format: action@devpath\x00KEY=VALUE\x00KEY=VALUE\x00...
	rawEvent := []byte("add@/devices/pci0000:00/usb1/video4linux/video0\x00" +
		"ACTION=add\x00" +
		"DEVPATH=/devices/pci0000:00/usb1/video4linux/video0\x00" +
		"SUBSYSTEM=video4linux\x00" +
		"DEVNAME=/dev/video0\x00" +
		"DEVTYPE=video_device\x00" +
		"PRODUCT=1234/5678/0001\x00" +
		"IdModel=USB_Camera\x00\x00")

	event, err := um.parseUevent(rawEvent)
	if err != nil {
		t.Fatalf("parseUevent failed: %v", err)
	}

	if event.Action != strAdd {
		t.Errorf("Expected action 'add', got '%s'", event.Action)
	}

	if event.DevPath != "/devices/pci0000:00/usb1/video4linux/video0" {
		t.Errorf("DevPath mismatch: %s", event.DevPath)
	}

	if event.Subsystem != SubsystemVideo4linux {
		t.Errorf("Expected subsystem 'video4linux', got '%s'", event.Subsystem)
	}

	if event.DevName != strDevVideo0 {
		t.Errorf("Expected DevName '/dev/video0', got '%s'", event.DevName)
	}

	if event.DevType != "video_device" {
		t.Errorf("Expected DevType 'video_device', got '%s'", event.DevType)
	}

	// Check properties
	if event.Properties["PRODUCT"] != "1234/5678/0001" {
		t.Error("PRODUCT property not parsed correctly")
	}

	if event.Properties["IdModel"] != "USB_Camera" {
		t.Error("IdModel property not parsed correctly")
	}
}

// TestIsRelevantEvent is part of the public API.
func TestIsRelevantEvent(t *testing.T) {
	um := &UdevMonitor{}

	tests := []struct {
		event    UdevEvent
		name     string
		expected bool
	}{
		{
			name: "Video4Linux add",
			event: UdevEvent{
				Action:    strAdd,
				Subsystem: SubsystemVideo4linux,
			},
			expected: true,
		},
		{
			name: "Sound remove",
			event: UdevEvent{
				Action:    strRemove,
				Subsystem: SubsystemSound,
			},
			expected: true,
		},
		{
			name: "Input change",
			event: UdevEvent{
				Action:    "change",
				Subsystem: SubsystemInput,
			},
			expected: true,
		},
		{
			name: "Irrelevant subsystem",
			event: UdevEvent{
				Action:    strAdd,
				Subsystem: "block",
			},
			expected: false,
		},
		{
			name: "Irrelevant action",
			event: UdevEvent{
				Action:    "bind",
				Subsystem: SubsystemVideo4linux,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := um.isRelevantEvent(tt.event)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestParseDeviceName is part of the public API.
func TestParseDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		event    UdevEvent
		expected string
	}{
		{
			name: "IdModel present",
			event: UdevEvent{
				Properties: map[string]string{
					"IdModel": "Logitech_Webcam",
				},
			},
			expected: "Logitech Webcam",
		},
		{
			name: "PRODUCT present",
			event: UdevEvent{
				Properties: map[string]string{
					"PRODUCT": "046d/0825/0010",
				},
				DevName: strDevVideo0,
			},
			expected: "046d/0825/0010",
		},
		{
			name: "Fallback to DevName",
			event: UdevEvent{
				DevName: "/dev/video1",
			},
			expected: "/dev/video1",
		},
		{
			name: "No information",
			event: UdevEvent{
				Properties: map[string]string{},
			},
			expected: "Unknown Device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDeviceName(tt.event)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestUdevMonitor_Creation is part of the public API.
func TestUdevMonitor_Creation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping udev monitor test in short mode")
	}
	// This test will skip if not running with appropriate permissions
	um, err := NewUdevMonitor()
	if err != nil {
		// Check if error is due to permissions
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("Skipping test (requires CAP_NET_ADMIN or root): %v", err)
			return
		}
		t.Fatalf("NewUdevMonitor failed: %v", err)
	}
	defer func() {
		if err := um.Stop(); err != nil {
			t.Error(err)
		}
	}()

	if um.socket == 0 {
		t.Error("Socket should be initialized")
	}

	if um.eventCh == nil {
		t.Error("Event channel should be initialized")
	}

	if um.stopCh == nil {
		t.Error("Stop channel should be initialized")
	}
}

// TestUdevMonitor_StartStop is part of the public API.
func TestUdevMonitor_StartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping udev monitor test in short mode")
	}
	um, err := NewUdevMonitor()
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("Skipping test (requires CAP_NET_ADMIN or root): %v", err)
			return
		}
		t.Fatalf("NewUdevMonitor failed: %v", err)
	}

	// Start monitoring
	um.Start()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop monitoring
	if err := um.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Verify channels are closed
	select {
	case _, ok := <-um.eventCh:
		if ok {
			t.Error("Event channel should be closed after Stop")
		}
	default:
		// Expected - channel is closed
	}
}

// TestUdevMonitor_EventChannel is part of the public API.
func TestUdevMonitor_EventChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping udev monitor test in short mode")
	}
	um, err := NewUdevMonitor()
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("Skipping test (requires CAP_NET_ADMIN or root): %v", err)
			return
		}
		t.Fatalf("NewUdevMonitor failed: %v", err)
	}
	defer func() {
		if err := um.Stop(); err != nil {
			t.Error(err)
		}
	}()

	um.Start()

	// Get events channel
	events := um.Events()

	// This test is passive - we just verify the channel is accessible
	// Actual events would require plugging/unplugging hardware
	select {
	case <-events:
		// If an event comes through, that's fine
	case <-time.After(100 * time.Millisecond):
		// No events in 100ms is also fine (no devices were plugged)
	}
}

// TestUdevMonitor_Integration is a manual integration test
// Run this with: go test -v -run TestUdevMonitor_Integration
// Then plug/unplug a USB camera or microphone
func TestUdevMonitor_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	um, err := NewUdevMonitor()
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("Skipping test (requires CAP_NET_ADMIN or root): %v", err)
			return
		}
		t.Fatalf("NewUdevMonitor failed: %v", err)
	}
	defer func() {
		if err := um.Stop(); err != nil {
			t.Error(err)
		}
	}()

	um.Start()

	t.Log("Monitoring udev events for 5 seconds...")
	t.Log("Plug or unplug a USB device to see events")

	timeout := time.After(5 * time.Second)
	eventCount := 0

	for {
		select {
		case event := <-um.Events():
			eventCount++
			t.Logf("Event %d: %s", eventCount, event.String())
			t.Logf("  Device Type: %s", event.GetDeviceType())
			t.Logf("  Device Name: %s", ParseDeviceName(event))
		case <-timeout:
			t.Logf("Test complete. Received %d events", eventCount)
			return
		}
	}
}

// BenchmarkParseUevent is part of the public API.
func BenchmarkParseUevent(b *testing.B) {
	um := &UdevMonitor{}

	rawEvent := []byte("add@/devices/pci0000:00/usb1/video4linux/video0\x00" +
		"ACTION=add\x00" +
		"DEVPATH=/devices/pci0000:00/usb1/video4linux/video0\x00" +
		"SUBSYSTEM=video4linux\x00" +
		"DEVNAME=/dev/video0\x00" +
		"DEVTYPE=video_device\x00\x00")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := um.parseUevent(rawEvent); err != nil {
			b.Fatal(err)
		}
	}

	// Target: <1μs per parse, 0-1 allocations
}

// BenchmarkIsRelevantEvent is part of the public API.
func BenchmarkIsRelevantEvent(b *testing.B) {
	um := &UdevMonitor{}

	event := UdevEvent{
		Action:    strAdd,
		Subsystem: SubsystemVideo4linux,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		um.isRelevantEvent(event)
	}

	// Should be extremely fast (just string comparisons)
}
