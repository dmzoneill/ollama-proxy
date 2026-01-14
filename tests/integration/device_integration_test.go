package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/device"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/godbus/dbus/v5"
)

func TestMain(m *testing.M) {
	// Initialize logging
	if err := logging.InitLogger("info", false); err != nil {
		panic(err)
	}
	defer logging.Sync()

	os.Exit(m.Run())
}

// TestDeviceManager_FullLifecycle tests the complete lifecycle of device management
func TestDeviceManager_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create device manager
	dm, err := device.NewDeviceManager()
	if err != nil {
		t.Skipf("Failed to create device manager (D-Bus may not be available): %v", err)
	}
	defer dm.Stop()

	// Test 1: Register a virtual microphone
	caps := map[string]dbus.Variant{
		"sample_rate": dbus.MakeVariant(48000),
		"channels":    dbus.MakeVariant(2),
		"format":      dbus.MakeVariant("S16_LE"),
	}

	deviceID, dbusErr := dm.RegisterDevice(
		"microphone",
		"/dev/null", // Virtual device
		"Test Microphone",
		caps,
		"ie.fio.OllamaProxy.System", // System sender
	)

	if dbusErr != nil {
		t.Fatalf("Failed to register device: %v", dbusErr)
	}

	t.Logf("Registered device: %s", deviceID)

	// Test 2: List devices
	devices, dbusErr := dm.ListDevices("")
	if dbusErr != nil {
		t.Fatalf("Failed to list devices: %v", dbusErr)
	}

	if len(devices) == 0 {
		t.Error("Expected at least one device")
	}

	// Test 3: Get device info
	deviceInfo, dbusErr := dm.GetDevice(deviceID)
	if dbusErr != nil {
		t.Fatalf("Failed to get device: %v", dbusErr)
	}

	if deviceInfo["Name"].Value() != "Test Microphone" {
		t.Errorf("Expected device name 'Test Microphone', got %v", deviceInfo["Name"].Value())
	}

	// Test 4: Request device access
	grantID, shmPath, udsPath, dbusErr := dm.RequestDeviceAccess(
		deviceID,
		"test-client",
		"ie.fio.OllamaProxy.System",
	)

	if dbusErr != nil {
		t.Fatalf("Failed to request access: %v", dbusErr)
	}

	t.Logf("Access granted: %s, SHM: %s, UDS: %s", grantID, shmPath, udsPath)

	// Test 5: Release access
	dbusErr = dm.ReleaseDeviceAccess(deviceID, "test-client")
	if dbusErr != nil {
		t.Fatalf("Failed to release access: %v", dbusErr)
	}

	// Test 6: Unregister device
	dbusErr = dm.UnregisterDevice(deviceID)
	if dbusErr != nil {
		t.Fatalf("Failed to unregister device: %v", dbusErr)
	}

	t.Log("Full lifecycle test completed successfully")
}

// TestSharedMemory_VideoStreaming tests video streaming through shared memory
func TestSharedMemory_VideoStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create shared memory ring for video (640x480 RGB)
	frameSize := uint64(640 * 480 * 3)
	numFrames := uint64(8)

	shmRing, err := device.CreateSharedMemoryRing("integration-video", numFrames, frameSize, logging.Logger)
	if err != nil {
		t.Skipf("Failed to create shared memory (may not have permissions): %v", err)
	}
	defer shmRing.Destroy()

	// Open as reader
	reader, err := device.OpenSharedMemoryRing("integration-video", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open shared memory as reader: %v", err)
	}
	defer reader.Close()

	// Simulate video streaming
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	framesWritten := 0
	framesRead := 0

	// Writer goroutine
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if framesWritten >= 100 {
					return
				}

				frame := make([]byte, frameSize)
				// Simulate frame data
				for i := range frame {
					frame[i] = byte(framesWritten % 256)
				}

				if _, err := shmRing.Write(frame); err != nil {
					t.Logf("Write error: %v", err)
					return
				}
				framesWritten++
			}
		}
	}()

	// Reader goroutine
	readTicker := time.NewTicker(16 * time.Millisecond)
	defer readTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Logf("Video streaming test: wrote %d frames, read %d frames", framesWritten, framesRead)
			if framesRead < 90 {
				t.Errorf("Expected to read at least 90 frames, got %d", framesRead)
			}
			return
		case <-readTicker.C:
			frame, _, err := reader.Read()
			if err != nil {
				continue
			}
			if frame != nil {
				framesRead++
			}
		}
	}
}

// TestUDS_Metadata tests Unix Domain Socket metadata distribution
func TestUDS_Metadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create UDS server
	server, err := device.NewUDSDeviceServer("integration-test", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	// Connect client
	client, err := device.ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		t.Fatalf("Failed to connect UDS client: %v", err)
	}
	defer client.Close()

	time.Sleep(10 * time.Millisecond)

	// Broadcast metadata
	metadata := []byte(`{"fps":60,"resolution":"640x480","format":"RGB24"}`)
	server.BroadcastMetadata(metadata)

	// Receive metadata
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		msg, err := client.ReceiveMessage()
		if err != nil {
			t.Errorf("Failed to receive metadata: %v", err)
			return
		}

		if string(msg.Payload) != string(metadata) {
			t.Errorf("Metadata mismatch: expected %s, got %s", metadata, msg.Payload)
		}

		done <- true
	}()

	select {
	case <-done:
		t.Log("Metadata distribution test completed successfully")
	case <-ctx.Done():
		t.Error("Timeout waiting for metadata")
	}
}

// TestV4L2_BasicOperation tests V4L2 device basic operations (if hardware available)
func TestV4L2_BasicOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if video device exists
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	v4l2Device, err := device.OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Skipf("Failed to open V4L2 device: %v", err)
	}
	defer v4l2Device.Close()

	// Set format
	if err := v4l2Device.SetFormat(640, 480, device.V4L2_PIX_FMT_YUYV); err != nil {
		// Try MJPEG if YUYV not supported
		if err := v4l2Device.SetFormat(640, 480, device.V4L2_PIX_FMT_MJPEG); err != nil {
			t.Skipf("Failed to set format: %v", err)
		}
	}

	// Request and map buffers
	if err := v4l2Device.RequestBuffers(4); err != nil {
		t.Fatalf("Failed to request buffers: %v", err)
	}

	if err := v4l2Device.MapBuffers(4); err != nil {
		t.Fatalf("Failed to map buffers: %v", err)
	}

	// Start streaming
	if err := v4l2Device.StartStreaming(); err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}
	defer v4l2Device.StopStreaming()

	// Capture a few frames
	capturedFrames := 0
	for i := 0; i < 10; i++ {
		buf, err := v4l2Device.DequeueBuffer()
		if err != nil {
			t.Logf("Failed to dequeue buffer: %v", err)
			continue
		}

		if buf.BytesUsed > 0 {
			capturedFrames++
		}

		v4l2Device.QueueBuffer(buf.Index)
	}

	if capturedFrames == 0 {
		t.Error("Failed to capture any frames")
	}

	t.Logf("Captured %d frames from V4L2 device", capturedFrames)
}

// BenchmarkEndToEnd_Latency benchmarks end-to-end latency
func BenchmarkEndToEnd_Latency(b *testing.B) {
	// Create shared memory ring
	frameSize := uint64(1920 * 1080 * 3)
	shmRing, err := device.CreateSharedMemoryRing("bench-e2e", 16, frameSize, logging.Logger)
	if err != nil {
		b.Skipf("Failed to create shared memory: %v", err)
	}
	defer shmRing.Destroy()

	reader, err := device.OpenSharedMemoryRing("bench-e2e", logging.Logger)
	if err != nil {
		b.Fatalf("Failed to open shared memory: %v", err)
	}
	defer reader.Close()

	frame := make([]byte, frameSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Write
		shmRing.Write(frame)

		// Read
		reader.Read()

		elapsed := time.Since(start)

		// Target: <50μs
		if elapsed > 50*time.Microsecond {
			b.Logf("Warning: High latency %v (target <50μs)", elapsed)
		}
	}
}
