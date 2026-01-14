package device

import (
	"sync"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

func TestSharedMemoryRing_CreateAndOpen(t *testing.T) {
	// Create ring
	ring, err := CreateSharedMemoryRing("test-ring", 4, 1024, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	if ring.GetNumFrames() != 4 {
		t.Errorf("Expected 4 frames, got %d", ring.GetNumFrames())
	}

	if ring.GetFrameSize() != 1024 {
		t.Errorf("Expected frame size 1024, got %d", ring.GetFrameSize())
	}

	// Open existing ring
	reader, err := OpenSharedMemoryRing("test-ring", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ring: %v", err)
	}
	defer reader.Close()

	if reader.GetNumFrames() != 4 {
		t.Errorf("Reader: Expected 4 frames, got %d", reader.GetNumFrames())
	}
}

func TestSharedMemoryRing_WriteRead(t *testing.T) {
	ring, err := CreateSharedMemoryRing("test-write-read", 8, 512, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	// Open as reader
	reader, err := OpenSharedMemoryRing("test-write-read", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ring: %v", err)
	}
	defer reader.Close()

	// Write some data
	testData := []byte("Hello, shared memory!")
	frameIdx, err := ring.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	if frameIdx != 0 {
		t.Errorf("Expected frame index 0, got %d", frameIdx)
	}

	// Read data
	readData, idx, err := reader.Read()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if idx != 0 {
		t.Errorf("Expected read frame index 0, got %d", idx)
	}

	// Compare data (only up to the length of testData)
	if string(readData[:len(testData)]) != string(testData) {
		t.Errorf("Data mismatch: expected %s, got %s", testData, readData[:len(testData)])
	}

	// Try reading again (should return nil - no new data)
	readData, _, err = reader.Read()
	if err != nil {
		t.Fatalf("Failed on second read: %v", err)
	}
	if readData != nil {
		t.Error("Expected nil when no new data")
	}
}

func TestSharedMemoryRing_MultipleFrames(t *testing.T) {
	ring, err := CreateSharedMemoryRing("test-multi-frame", 4, 256, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	reader, err := OpenSharedMemoryRing("test-multi-frame", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ring: %v", err)
	}
	defer reader.Close()

	// Write multiple frames
	for i := 0; i < 3; i++ {
		data := []byte{byte(i), byte(i + 1), byte(i + 2)}
		_, err := ring.Write(data)
		if err != nil {
			t.Fatalf("Failed to write frame %d: %v", i, err)
		}
	}

	// Check available frames
	if reader.AvailableFrames() != 3 {
		t.Errorf("Expected 3 available frames, got %d", reader.AvailableFrames())
	}

	// Read all frames
	for i := 0; i < 3; i++ {
		data, _, err := reader.Read()
		if err != nil {
			t.Fatalf("Failed to read frame %d: %v", i, err)
		}

		if data[0] != byte(i) {
			t.Errorf("Frame %d: expected first byte %d, got %d", i, i, data[0])
		}
	}

	// No more data available
	if reader.AvailableFrames() != 0 {
		t.Error("Expected 0 available frames after reading all")
	}
}

func TestSharedMemoryRing_Wrap(t *testing.T) {
	ring, err := CreateSharedMemoryRing("test-wrap", 4, 128, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	reader, err := OpenSharedMemoryRing("test-wrap", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ring: %v", err)
	}
	defer reader.Close()

	// Write more frames than capacity to test wrapping
	for i := 0; i < 10; i++ {
		data := []byte{byte(i)}
		_, err := ring.Write(data)
		if err != nil {
			t.Fatalf("Failed to write frame %d: %v", i, err)
		}
	}

	// Reader should have 4 frames available (ring size)
	available := reader.AvailableFrames()
	if available != 4 {
		t.Errorf("Expected 4 available frames (ring wrapped), got %d", available)
	}

	// Read and verify we get the latest 4 frames (6, 7, 8, 9)
	for i := 6; i < 10; i++ {
		data, _, err := reader.Read()
		if err != nil {
			t.Fatalf("Failed to read frame: %v", err)
		}

		if data[0] != byte(i) {
			t.Errorf("Expected frame data %d, got %d", i, data[0])
		}
	}
}

func TestSharedMemoryRing_ConcurrentWriteRead(t *testing.T) {
	ring, err := CreateSharedMemoryRing("test-concurrent", 16, 1024, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	reader, err := OpenSharedMemoryRing("test-concurrent", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to open ring: %v", err)
	}
	defer reader.Close()

	var wg sync.WaitGroup
	numWrites := 100
	writeErrors := make(chan error, 1)
	readErrors := make(chan error, 1)

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numWrites; i++ {
			data := []byte{byte(i % 256)}
			if _, err := ring.Write(data); err != nil {
				select {
				case writeErrors <- err:
				default:
				}
				return
			}
			time.Sleep(time.Microsecond * 10)
		}
	}()

	// Reader goroutine
	readCount := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for readCount < numWrites {
			data, _, err := reader.Read()
			if err != nil {
				select {
				case readErrors <- err:
				default:
				}
				return
			}
			if data != nil {
				readCount++
			} else {
				time.Sleep(time.Microsecond * 10)
			}
		}
	}()

	wg.Wait()

	// Check for errors
	select {
	case err := <-writeErrors:
		t.Fatalf("Write error: %v", err)
	default:
	}

	select {
	case err := <-readErrors:
		t.Fatalf("Read error: %v", err)
	default:
	}

	if readCount != numWrites {
		t.Errorf("Expected to read %d frames, got %d", numWrites, readCount)
	}
}

func TestSharedMemoryRing_Stats(t *testing.T) {
	ring, err := CreateSharedMemoryRing("test-stats", 8, 512, logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	// Write some frames
	for i := 0; i < 5; i++ {
		ring.Write([]byte{byte(i)})
	}

	stats := ring.GetStats()

	if stats.NumFrames != 8 {
		t.Errorf("Expected 8 frames, got %d", stats.NumFrames)
	}

	if stats.FrameSize != 512 {
		t.Errorf("Expected frame size 512, got %d", stats.FrameSize)
	}

	if stats.WritePosition != 5 {
		t.Errorf("Expected write position 5, got %d", stats.WritePosition)
	}
}

func BenchmarkSharedMemoryRing_Write(b *testing.B) {
	ring, err := CreateSharedMemoryRing("bench-write", 16, 4096, logging.Logger)
	if err != nil {
		b.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ring.Write(data)
	}

	// Target: <500ns per write
}

func BenchmarkSharedMemoryRing_Read(b *testing.B) {
	ring, err := CreateSharedMemoryRing("bench-read", 16, 4096, logging.Logger)
	if err != nil {
		b.Fatalf("Failed to create ring: %v", err)
	}
	defer ring.Destroy()

	reader, err := OpenSharedMemoryRing("bench-read", logging.Logger)
	if err != nil {
		b.Fatalf("Failed to open ring: %v", err)
	}
	defer reader.Close()

	// Pre-fill with data
	data := make([]byte, 4096)
	for i := 0; i < b.N; i++ {
		ring.Write(data)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader.Read()
	}

	// Target: <500ns per read
}
