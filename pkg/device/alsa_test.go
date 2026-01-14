package device

import (
	"os"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

func TestALSADevice_OpenClose(t *testing.T) {
	// Skip if no ALSA device available
	if _, err := os.Stat("/dev/snd/pcmC0D0c"); os.IsNotExist(err) {
		t.Skip("No /dev/snd/pcmC0D0c device available")
	}

	device, err := OpenALSADevice("/dev/snd/pcmC0D0c", 48000, 2, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ALSA device: %v", err)
	}
	defer device.Close()

	if device.fd < 0 {
		t.Error("Expected valid file descriptor")
	}

	if device.sampleRate != 48000 {
		t.Errorf("Expected sample rate 48000, got %d", device.sampleRate)
	}

	if device.channels != 2 {
		t.Errorf("Expected 2 channels, got %d", device.channels)
	}
}

func TestALSADevice_GetCapabilities(t *testing.T) {
	if _, err := os.Stat("/dev/snd/pcmC0D0c"); os.IsNotExist(err) {
		t.Skip("No /dev/snd/pcmC0D0c device available")
	}

	device, err := OpenALSADevice("/dev/snd/pcmC0D0c", 44100, 1, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ALSA device: %v", err)
	}
	defer device.Close()

	caps := device.GetCapabilities()

	if caps["sample_rate"] != uint32(44100) {
		t.Errorf("Expected sample rate 44100 in caps, got %v", caps["sample_rate"])
	}

	if caps["channels"] != uint32(1) {
		t.Errorf("Expected 1 channel in caps, got %v", caps["channels"])
	}

	if caps["format"] != "S16_LE" {
		t.Errorf("Expected format S16_LE, got %v", caps["format"])
	}

	t.Logf("ALSA Device Capabilities: %+v", caps)
}

func TestALSADevice_Prepare(t *testing.T) {
	if _, err := os.Stat("/dev/snd/pcmC0D0c"); os.IsNotExist(err) {
		t.Skip("No /dev/snd/pcmC0D0c device available")
	}

	device, err := OpenALSADevice("/dev/snd/pcmC0D0c", 48000, 2, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ALSA device: %v", err)
	}
	defer device.Close()

	err = device.Prepare()
	if err != nil {
		t.Fatalf("Failed to prepare device: %v", err)
	}
}

func TestALSADevice_StartStopCapture(t *testing.T) {
	if _, err := os.Stat("/dev/snd/pcmC0D0c"); os.IsNotExist(err) {
		t.Skip("No /dev/snd/pcmC0D0c device available")
	}

	device, err := OpenALSADevice("/dev/snd/pcmC0D0c", 48000, 2, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ALSA device: %v", err)
	}
	defer device.Close()

	// Start capture
	err = device.StartCapture()
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	if !device.capturing {
		t.Error("Expected capturing flag to be true")
	}

	// Stop capture
	err = device.StopCapture()
	if err != nil {
		t.Fatalf("Failed to stop capture: %v", err)
	}

	if device.capturing {
		t.Error("Expected capturing flag to be false")
	}
}

func TestSimplifiedALSACapture_Create(t *testing.T) {
	capture, err := NewSimplifiedALSACapture("default", 48000, 2, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create simplified capture: %v", err)
	}
	defer capture.Close()

	if capture.sampleRate != 48000 {
		t.Errorf("Expected sample rate 48000, got %d", capture.sampleRate)
	}

	if capture.channels != 2 {
		t.Errorf("Expected 2 channels, got %d", capture.channels)
	}

	if len(capture.buffer) == 0 {
		t.Error("Expected non-zero buffer size")
	}
}

func TestSimplifiedALSACapture_GetCapabilities(t *testing.T) {
	capture, err := NewSimplifiedALSACapture("default", 44100, 1, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create simplified capture: %v", err)
	}
	defer capture.Close()

	caps := capture.GetCapabilities()

	if caps["sample_rate"] != uint32(44100) {
		t.Errorf("Expected sample rate 44100, got %v", caps["sample_rate"])
	}

	if caps["channels"] != uint32(1) {
		t.Errorf("Expected 1 channel, got %v", caps["channels"])
	}

	if caps["api"] != "simplified" {
		t.Errorf("Expected api=simplified, got %v", caps["api"])
	}

	t.Logf("Simplified ALSA Capabilities: %+v", caps)
}

func TestALSADevice_IntegrationWithSHM(t *testing.T) {
	// Create shared memory ring for audio
	// 48kHz stereo S16_LE = 48000 * 2 * 2 = 192,000 bytes/sec
	// Use 10ms buffers = 1920 bytes per frame
	frameSize := uint64(1920)
	shmRing, err := CreateSharedMemoryRing("test-alsa-shm", 16, frameSize, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create shared memory ring: %v", err)
	}
	defer shmRing.Destroy()

	// Use simplified capture (doesn't require actual hardware)
	capture, err := NewSimplifiedALSACapture("default", 48000, 2, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create simplified capture: %v", err)
	}
	defer capture.Close()

	// Test that we can write audio data to SHM
	testData := make([]byte, frameSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	frameIdx, err := shmRing.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write audio to SHM: %v", err)
	}

	if frameIdx != 0 {
		t.Errorf("Expected frame index 0, got %d", frameIdx)
	}

	// Verify we can read it back
	reader, err := OpenSharedMemoryRing("test-alsa-shm", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open SHM as reader: %v", err)
	}
	defer reader.Close()

	readData, idx, err := reader.Read()
	if err != nil {
		t.Fatalf("Failed to read from SHM: %v", err)
	}

	if idx != 0 {
		t.Errorf("Expected read frame index 0, got %d", idx)
	}

	// Verify data matches (up to frameSize)
	if len(readData) < len(testData) {
		t.Fatalf("Read data too short: %d < %d", len(readData), len(testData))
	}

	for i := range testData {
		if readData[i] != testData[i] {
			t.Errorf("Data mismatch at byte %d: expected %d, got %d", i, testData[i], readData[i])
			break
		}
	}

	t.Log("Successfully transferred audio data via SHM")
}

func BenchmarkSimplifiedALSACapture_Write(b *testing.B) {
	shmRing, err := CreateSharedMemoryRing("bench-alsa", 16, 1920, logging.Logger)
	if err != nil {
		b.Fatalf("Failed to create SHM: %v", err)
	}
	defer shmRing.Destroy()

	data := make([]byte, 1920)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		shmRing.Write(data)
	}
}
