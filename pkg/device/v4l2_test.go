package device

import (
	"os"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

func TestV4L2Device_OpenClose(t *testing.T) {
	// Skip if no video device available
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	if device.fd < 0 {
		t.Error("Expected valid file descriptor")
	}

	// Check capabilities were queried
	if string(device.caps.Driver[:]) == "" {
		t.Error("Expected driver name to be populated")
	}
}

func TestV4L2Device_QueryCapabilities(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	caps := device.GetCapabilities()

	if _, ok := caps["driver"]; !ok {
		t.Error("Expected 'driver' capability")
	}

	if _, ok := caps["card"]; !ok {
		t.Error("Expected 'card' capability")
	}

	t.Logf("V4L2 Device Capabilities: %+v", caps)
}

func TestV4L2Device_SetFormat(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	// Try to set 640x480 YUYV format
	err = device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV)
	if err != nil {
		// Some devices may not support YUYV, try MJPEG
		err = device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
		if err != nil {
			t.Fatalf("Failed to set format: %v", err)
		}
	}

	if device.format.Width != 640 {
		t.Errorf("Expected width 640, got %d", device.format.Width)
	}

	if device.format.Height != 480 {
		t.Errorf("Expected height 480, got %d", device.format.Height)
	}
}

func TestV4L2Device_RequestBuffers(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	// Set format first
	if err := device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV); err != nil {
		device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
	}

	// Request 4 buffers
	err = device.RequestBuffers(4)
	if err != nil {
		t.Fatalf("Failed to request buffers: %v", err)
	}
}

func TestV4L2Device_MapBuffers(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	// Set format
	if err := device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV); err != nil {
		device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
	}

	// Request buffers
	if err := device.RequestBuffers(4); err != nil {
		t.Fatalf("Failed to request buffers: %v", err)
	}

	// Map buffers
	err = device.MapBuffers(4)
	if err != nil {
		t.Fatalf("Failed to map buffers: %v", err)
	}

	if len(device.buffers) != 4 {
		t.Errorf("Expected 4 buffers, got %d", len(device.buffers))
	}

	for i, buf := range device.buffers {
		if buf.data == nil {
			t.Errorf("Buffer %d data is nil", i)
		}
		if buf.length == 0 {
			t.Errorf("Buffer %d length is 0", i)
		}
	}
}

func TestV4L2Device_StreamingLifecycle(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	// Setup
	if err := device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV); err != nil {
		device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
	}

	if err := device.RequestBuffers(4); err != nil {
		t.Fatalf("Failed to request buffers: %v", err)
	}

	if err := device.MapBuffers(4); err != nil {
		t.Fatalf("Failed to map buffers: %v", err)
	}

	// Start streaming
	err = device.StartStreaming()
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}

	if !device.streaming {
		t.Error("Expected streaming flag to be true")
	}

	// Stop streaming
	err = device.StopStreaming()
	if err != nil {
		t.Fatalf("Failed to stop streaming: %v", err)
	}

	if device.streaming {
		t.Error("Expected streaming flag to be false")
	}
}

func TestV4L2Device_CaptureFrames(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	// Setup
	if err := device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV); err != nil {
		device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
	}

	if err := device.RequestBuffers(4); err != nil {
		t.Fatalf("Failed to request buffers: %v", err)
	}

	if err := device.MapBuffers(4); err != nil {
		t.Fatalf("Failed to map buffers: %v", err)
	}

	if err := device.StartStreaming(); err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}
	defer device.StopStreaming()

	// Try to capture a few frames
	for i := 0; i < 5; i++ {
		buf, err := device.DequeueBuffer()
		if err != nil {
			t.Fatalf("Failed to dequeue buffer: %v", err)
		}

		if buf.BytesUsed == 0 {
			t.Error("Expected non-zero bytes in captured frame")
		}

		t.Logf("Captured frame %d: %d bytes", i, buf.BytesUsed)

		// Re-queue buffer
		if err := device.QueueBuffer(buf.Index); err != nil {
			t.Fatalf("Failed to re-queue buffer: %v", err)
		}
	}
}

func TestV4L2Device_IntegrationWithSHM(t *testing.T) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		t.Skip("No /dev/video0 device available")
	}

	// Create shared memory ring
	shmRing, err := CreateSharedMemoryRing("test-v4l2-shm", 8, 640*480*2, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create shared memory ring: %v", err)
	}
	defer shmRing.Destroy()

	// Open V4L2 device
	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	// Setup device
	if err := device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV); err != nil {
		device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
	}

	if err := device.RequestBuffers(4); err != nil {
		t.Fatalf("Failed to request buffers: %v", err)
	}

	if err := device.MapBuffers(4); err != nil {
		t.Fatalf("Failed to map buffers: %v", err)
	}

	if err := device.StartStreaming(); err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}
	defer device.StopStreaming()

	// Capture a few frames and write to SHM
	framesWritten := 0
	for i := 0; i < 3; i++ {
		buf, err := device.DequeueBuffer()
		if err != nil {
			t.Errorf("Failed to dequeue buffer: %v", err)
			continue
		}

		frameData := device.buffers[buf.Index].data[:buf.BytesUsed]
		_, err = shmRing.Write(frameData)
		if err != nil {
			t.Errorf("Failed to write to SHM: %v", err)
		} else {
			framesWritten++
		}

		device.QueueBuffer(buf.Index)
	}

	if framesWritten == 0 {
		t.Error("Expected to write at least one frame to SHM")
	}

	// Verify we can read from SHM
	reader, err := OpenSharedMemoryRing("test-v4l2-shm", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open SHM as reader: %v", err)
	}
	defer reader.Close()

	framesRead := 0
	for i := 0; i < framesWritten; i++ {
		frame, _, err := reader.Read()
		if err != nil {
			t.Errorf("Failed to read from SHM: %v", err)
			continue
		}
		if frame != nil {
			framesRead++
		}
	}

	if framesRead != framesWritten {
		t.Errorf("Expected to read %d frames, got %d", framesWritten, framesRead)
	}

	t.Logf("Successfully captured %d frames from V4L2 and transferred via SHM", framesWritten)
}

func BenchmarkV4L2_CaptureFrame(b *testing.B) {
	if _, err := os.Stat("/dev/video0"); os.IsNotExist(err) {
		b.Skip("No /dev/video0 device available")
	}

	device, err := OpenV4L2Device("/dev/video0", logging.Logger)
	if err != nil {
		b.Fatalf("Failed to open V4L2 device: %v", err)
	}
	defer device.Close()

	if err := device.SetFormat(640, 480, V4L2_PIX_FMT_YUYV); err != nil {
		device.SetFormat(640, 480, V4L2_PIX_FMT_MJPEG)
	}

	if err := device.RequestBuffers(4); err != nil {
		b.Fatalf("Failed to request buffers: %v", err)
	}

	if err := device.MapBuffers(4); err != nil {
		b.Fatalf("Failed to map buffers: %v", err)
	}

	if err := device.StartStreaming(); err != nil {
		b.Fatalf("Failed to start streaming: %v", err)
	}
	defer device.StopStreaming()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf, err := device.DequeueBuffer()
		if err != nil {
			b.Fatal(err)
		}
		device.QueueBuffer(buf.Index)
	}
}
