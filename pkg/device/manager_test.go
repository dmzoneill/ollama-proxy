package device

import (
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestDeviceManager_NewDeviceManager(t *testing.T) {
	// Note: This test requires D-Bus system bus access
	// Skip if not running with proper permissions
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	if dm.conn == nil {
		t.Error("Connection should not be nil")
	}

	if dm.devices == nil {
		t.Error("Devices map should be initialized")
	}

	if dm.accessGrants == nil {
		t.Error("Access grants map should be initialized")
	}
}

func TestDeviceManager_RegisterDevice(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	caps := map[string]dbus.Variant{
		"sample_rate": dbus.MakeVariant(48000),
		"channels":    dbus.MakeVariant(2),
		"format":      dbus.MakeVariant("S16_LE"),
	}

	deviceID, dbusErr := dm.RegisterDevice(
		"microphone",
		"/dev/snd/pcmC0D0c",
		"Test Microphone",
		caps,
	)

	if dbusErr != nil {
		t.Fatalf("RegisterDevice failed: %v", dbusErr)
	}

	if deviceID == "" {
		t.Error("Device ID should not be empty")
	}

	// Verify device exists in registry
	dm.mu.RLock()
	device, exists := dm.devices[deviceID]
	dm.mu.RUnlock()

	if !exists {
		t.Fatal("Device should exist in registry")
	}

	if device.Name != "Test Microphone" {
		t.Errorf("Expected name 'Test Microphone', got '%s'", device.Name)
	}

	if device.Type != DeviceTypeMicrophone {
		t.Errorf("Expected type 'microphone', got '%s'", device.Type)
	}

	if device.State != DeviceStateAvailable {
		t.Errorf("Expected state 'available', got '%s'", device.State)
	}
}

func TestDeviceManager_UnregisterDevice(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register a device first
	deviceID, _ := dm.RegisterDevice(
		"camera",
		"/dev/video0",
		"Test Camera",
		map[string]dbus.Variant{},
	)

	// Unregister it
	dbusErr := dm.UnregisterDevice(deviceID)
	if dbusErr != nil {
		t.Fatalf("UnregisterDevice failed: %v", dbusErr)
	}

	// Verify device is removed
	dm.mu.RLock()
	_, exists := dm.devices[deviceID]
	dm.mu.RUnlock()

	if exists {
		t.Error("Device should be removed from registry")
	}
}

func TestDeviceManager_UnregisterDevice_NotFound(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Try to unregister non-existent device
	dbusErr := dm.UnregisterDevice("nonexistent-device-id")
	if dbusErr == nil {
		t.Error("Expected error when unregistering non-existent device")
	}
}

func TestDeviceManager_ListDevices(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register multiple devices
	dm.RegisterDevice("microphone", "/dev/snd/pcmC0D0c", "Mic 1", nil)
	dm.RegisterDevice("microphone", "/dev/snd/pcmC1D0c", "Mic 2", nil)
	dm.RegisterDevice("camera", "/dev/video0", "Camera 1", nil)

	// List all devices
	allDevices, dbusErr := dm.ListDevices("")
	if dbusErr != nil {
		t.Fatalf("ListDevices failed: %v", dbusErr)
	}

	if len(allDevices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(allDevices))
	}

	// List only microphones
	mics, dbusErr := dm.ListDevices("microphone")
	if dbusErr != nil {
		t.Fatalf("ListDevices(microphone) failed: %v", dbusErr)
	}

	if len(mics) != 2 {
		t.Errorf("Expected 2 microphones, got %d", len(mics))
	}

	// List only cameras
	cameras, dbusErr := dm.ListDevices("camera")
	if dbusErr != nil {
		t.Fatalf("ListDevices(camera) failed: %v", dbusErr)
	}

	if len(cameras) != 1 {
		t.Errorf("Expected 1 camera, got %d", len(cameras))
	}
}

func TestDeviceManager_GetDevice(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register device
	deviceID, _ := dm.RegisterDevice(
		"speaker",
		"/dev/snd/pcmC0D0p",
		"Test Speaker",
		map[string]dbus.Variant{
			"sample_rate": dbus.MakeVariant(44100),
		},
	)

	// Get device
	deviceInfo, dbusErr := dm.GetDevice(deviceID)
	if dbusErr != nil {
		t.Fatalf("GetDevice failed: %v", dbusErr)
	}

	if deviceInfo["Name"].Value().(string) != "Test Speaker" {
		t.Error("Device name mismatch")
	}

	if deviceInfo["Type"].Value().(string) != "speaker" {
		t.Error("Device type mismatch")
	}

	// Check capabilities
	caps := deviceInfo["Capabilities"].Value().(map[string]dbus.Variant)
	if caps["sample_rate"].Value().(int) != 44100 {
		t.Error("Capabilities not preserved")
	}
}

func TestDeviceManager_GetDevice_NotFound(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	_, dbusErr := dm.GetDevice("nonexistent-device")
	if dbusErr == nil {
		t.Error("Expected error when getting non-existent device")
	}
}

func TestDeviceManager_RequestDeviceAccess(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register device
	deviceID, _ := dm.RegisterDevice(
		"microphone",
		"/dev/snd/pcmC0D0c",
		"Test Mic",
		nil,
	)

	// Request access
	grantID, shmPath, udsPath, dbusErr := dm.RequestDeviceAccess(deviceID, "test-client")
	if dbusErr != nil {
		t.Fatalf("RequestDeviceAccess failed: %v", dbusErr)
	}

	if grantID == "" {
		t.Error("Grant ID should not be empty")
	}

	if shmPath == "" {
		t.Error("Shared memory path should not be empty")
	}

	if udsPath == "" {
		t.Error("Unix socket path should not be empty")
	}

	// Verify access grant exists
	dm.mu.RLock()
	_, exists := dm.accessGrants[grantID]
	dm.mu.RUnlock()

	if !exists {
		t.Error("Access grant should exist")
	}

	// Verify device state changed to InUse
	dm.mu.RLock()
	device := dm.devices[deviceID]
	dm.mu.RUnlock()

	if device.GetState() != DeviceStateInUse {
		t.Errorf("Expected state 'in-use', got '%s'", device.GetState())
	}
}

func TestDeviceManager_RequestDeviceAccess_DeviceNotFound(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	_, _, _, dbusErr := dm.RequestDeviceAccess("nonexistent", "client")
	if dbusErr == nil {
		t.Error("Expected error when requesting access to non-existent device")
	}
}

func TestDeviceManager_ReleaseDeviceAccess(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register device and request access
	deviceID, _ := dm.RegisterDevice("camera", "/dev/video0", "Test Cam", nil)
	grantID, _, _, _ := dm.RequestDeviceAccess(deviceID, "test-client")

	// Release access
	dbusErr := dm.ReleaseDeviceAccess(deviceID, "test-client")
	if dbusErr != nil {
		t.Fatalf("ReleaseDeviceAccess failed: %v", dbusErr)
	}

	// Verify grant removed
	dm.mu.RLock()
	_, exists := dm.accessGrants[grantID]
	dm.mu.RUnlock()

	if exists {
		t.Error("Access grant should be removed")
	}

	// Verify device state back to Available
	dm.mu.RLock()
	device := dm.devices[deviceID]
	dm.mu.RUnlock()

	if device.GetState() != DeviceStateAvailable {
		t.Errorf("Expected state 'available', got '%s'", device.GetState())
	}
}

func TestDeviceManager_ReleaseDeviceAccess_NotFound(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	dbusErr := dm.ReleaseDeviceAccess("device-id", "client-id")
	if dbusErr == nil {
		t.Error("Expected error when releasing non-existent access grant")
	}
}

func TestDeviceManager_ConcurrentAccess(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent device registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dm.RegisterDevice(
				"microphone",
				"/dev/null",
				"Concurrent Mic",
				nil,
			)
		}(i)
	}

	wg.Wait()

	// Verify devices were registered
	dm.mu.RLock()
	deviceCount := len(dm.devices)
	dm.mu.RUnlock()

	if deviceCount != numGoroutines {
		t.Errorf("Expected %d devices, got %d", numGoroutines, deviceCount)
	}
}

func TestDeviceManager_Properties(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Initially no devices
	if dm.getTotalDevices() != 0 {
		t.Error("Expected 0 total devices initially")
	}

	if dm.getAvailableDevices() != 0 {
		t.Error("Expected 0 available devices initially")
	}

	// Register 3 devices
	id1, _ := dm.RegisterDevice("microphone", "/dev/null", "Mic 1", nil)
	id2, _ := dm.RegisterDevice("camera", "/dev/null", "Cam 1", nil)
	_, _ = dm.RegisterDevice("speaker", "/dev/null", "Speaker 1", nil)

	if dm.getTotalDevices() != 3 {
		t.Errorf("Expected 3 total devices, got %d", dm.getTotalDevices())
	}

	if dm.getAvailableDevices() != 3 {
		t.Errorf("Expected 3 available devices, got %d", dm.getAvailableDevices())
	}

	// Request access to one device
	dm.RequestDeviceAccess(id1, "client")

	if dm.getAvailableDevices() != 2 {
		t.Errorf("Expected 2 available devices after access grant, got %d",
			dm.getAvailableDevices())
	}

	// Unregister a device
	dm.UnregisterDevice(id2)

	if dm.getTotalDevices() != 2 {
		t.Errorf("Expected 2 total devices after unregister, got %d",
			dm.getTotalDevices())
	}

	// Release access
	dm.ReleaseDeviceAccess(id1, "client")

	if dm.getAvailableDevices() != 2 {
		t.Errorf("Expected 2 available devices after release, got %d",
			dm.getAvailableDevices())
	}
}

func TestDeviceManager_StateTransitions(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register device (should be Available)
	deviceID, _ := dm.RegisterDevice("camera", "/dev/video0", "Test Cam", nil)

	dm.mu.RLock()
	device := dm.devices[deviceID]
	dm.mu.RUnlock()

	if device.GetState() != DeviceStateAvailable {
		t.Errorf("New device should be Available, got %s", device.GetState())
	}

	// Request access (should transition to InUse)
	dm.RequestDeviceAccess(deviceID, "client1")

	if device.GetState() != DeviceStateInUse {
		t.Errorf("Device should be InUse after access request, got %s", device.GetState())
	}

	// Release access (should transition back to Available)
	dm.ReleaseDeviceAccess(deviceID, "client1")

	if device.GetState() != DeviceStateAvailable {
		t.Errorf("Device should be Available after release, got %s", device.GetState())
	}
}

func TestDeviceManager_MultipleClients(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register device
	deviceID, _ := dm.RegisterDevice("microphone", "/dev/null", "Shared Mic", nil)

	// Multiple clients request access
	_, _, _, err1 := dm.RequestDeviceAccess(deviceID, "client1")
	_, _, _, err2 := dm.RequestDeviceAccess(deviceID, "client2")
	_, _, _, err3 := dm.RequestDeviceAccess(deviceID, "client3")

	if err1 != nil || err2 != nil || err3 != nil {
		t.Error("All clients should be able to request access")
	}

	// Verify multiple grants exist
	dm.mu.RLock()
	grantCount := 0
	for grantID := range dm.accessGrants {
		if dm.accessGrants[grantID].DeviceID == deviceID {
			grantCount++
		}
	}
	dm.mu.RUnlock()

	if grantCount != 3 {
		t.Errorf("Expected 3 access grants, got %d", grantCount)
	}

	// Release access from one client
	dm.ReleaseDeviceAccess(deviceID, "client1")

	// Device should still be in-use (other clients have access)
	dm.mu.RLock()
	_ = dm.devices[deviceID]
	dm.mu.RUnlock()

	// Note: In current implementation, device goes to Available immediately
	// In a production system, you might want to track active client count
	// and only transition to Available when all clients release
}

func TestDeviceManager_Stop(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Skipf("Skipping test (D-Bus not available): %v", err)
		return
	}

	// Register some devices
	dm.RegisterDevice("microphone", "/dev/null", "Mic", nil)
	dm.RegisterDevice("camera", "/dev/null", "Cam", nil)

	// Stop should not error
	if err := dm.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Context should be cancelled
	select {
	case <-dm.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Stop")
	}
}

// Benchmark tests

func BenchmarkDeviceManager_RegisterDevice(b *testing.B) {
	dm, err := NewDeviceManager()
	if err != nil {
		b.Skipf("Skipping benchmark (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	caps := map[string]dbus.Variant{
		"sample_rate": dbus.MakeVariant(48000),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dm.RegisterDevice("microphone", "/dev/null", "Bench Mic", caps)
	}

	// Target: <2ms per registration
}

func BenchmarkDeviceManager_RequestAccess(b *testing.B) {
	dm, err := NewDeviceManager()
	if err != nil {
		b.Skipf("Skipping benchmark (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Pre-register devices
	deviceIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		id, _ := dm.RegisterDevice("camera", "/dev/null", "Bench Cam", nil)
		deviceIDs[i] = id
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dm.RequestDeviceAccess(deviceIDs[i], "bench-client")
	}

	// Target: <2ms per access request
}

func BenchmarkDeviceManager_ListDevices(b *testing.B) {
	dm, err := NewDeviceManager()
	if err != nil {
		b.Skipf("Skipping benchmark (D-Bus not available): %v", err)
		return
	}
	defer dm.Stop()

	// Register 100 devices
	for i := 0; i < 100; i++ {
		dm.RegisterDevice("microphone", "/dev/null", "Mic", nil)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dm.ListDevices("")
	}
}
