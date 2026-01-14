package device

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	"go.uber.org/zap"
)

// RingHeader defines the shared memory ring buffer header
// Aligned to 64 bytes (cache line size) to avoid false sharing
type RingHeader struct {
	WritePos  uint64 // Atomic write position
	ReadPos   uint64 // Atomic read position (per-reader)
	TotalSize uint64 // Total buffer size in bytes
	FrameSize uint64 // Size of each frame in bytes
	NumFrames uint64 // Number of frames in buffer
	Padding   [24]byte // Pad to 64 bytes
}

// SharedMemoryRing represents a lock-free ring buffer in shared memory
type SharedMemoryRing struct {
	name      string
	size      int
	fd        int
	data      []byte
	header    *RingHeader
	logger    *zap.Logger
	readPos   uint64 // Local read position for this reader
}

const (
	// DefaultFrameSize is the default size for each frame (1MB)
	DefaultFrameSize = 1024 * 1024

	// DefaultNumFrames is the default number of frames in the ring
	DefaultNumFrames = 8

	// HeaderSize is the size of the ring header (64 bytes, cache-aligned)
	HeaderSize = 64
)

// CreateSharedMemoryRing creates a new shared memory ring buffer
func CreateSharedMemoryRing(name string, numFrames, frameSize uint64, logger *zap.Logger) (*SharedMemoryRing, error) {
	if numFrames == 0 {
		numFrames = DefaultNumFrames
	}
	if frameSize == 0 {
		frameSize = DefaultFrameSize
	}

	totalSize := HeaderSize + (numFrames * frameSize)

	// Create shared memory object
	shmName := "/ollama-proxy-" + name
	fd, err := syscall.Open(shmName, syscall.O_RDWR|syscall.O_CREAT|syscall.O_EXCL, 0600)
	if err != nil {
		// Try to unlink if it already exists
		syscall.Unlink(shmName)
		fd, err = syscall.Open(shmName, syscall.O_RDWR|syscall.O_CREAT|syscall.O_EXCL, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared memory: %w", err)
		}
	}

	// Set size
	if err := syscall.Ftruncate(fd, int64(totalSize)); err != nil {
		syscall.Close(fd)
		syscall.Unlink(shmName)
		return nil, fmt.Errorf("failed to set shm size: %w", err)
	}

	// Map into memory
	data, err := syscall.Mmap(fd, 0, int(totalSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)
	if err != nil {
		syscall.Close(fd)
		syscall.Unlink(shmName)
		return nil, fmt.Errorf("failed to mmap: %w", err)
	}

	// Initialize header
	header := (*RingHeader)(unsafe.Pointer(&data[0]))
	header.WritePos = 0
	header.ReadPos = 0
	header.TotalSize = totalSize
	header.FrameSize = frameSize
	header.NumFrames = numFrames

	ring := &SharedMemoryRing{
		name:    shmName,
		size:    int(totalSize),
		fd:      fd,
		data:    data,
		header:  header,
		logger:  logger,
		readPos: 0,
	}

	logger.Info("Created shared memory ring",
		zap.String("name", shmName),
		zap.Uint64("frames", numFrames),
		zap.Uint64("frame_size", frameSize),
		zap.Uint64("total_size", totalSize),
	)

	return ring, nil
}

// OpenSharedMemoryRing opens an existing shared memory ring buffer (for readers)
func OpenSharedMemoryRing(name string, logger *zap.Logger) (*SharedMemoryRing, error) {
	shmName := "/ollama-proxy-" + name

	// Open existing shared memory
	fd, err := syscall.Open(shmName, syscall.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open shared memory: %w", err)
	}

	// Get size
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to stat shm: %w", err)
	}

	// Map into memory
	data, err := syscall.Mmap(fd, 0, int(stat.Size),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)
	if err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to mmap: %w", err)
	}

	// Read header
	header := (*RingHeader)(unsafe.Pointer(&data[0]))

	ring := &SharedMemoryRing{
		name:    shmName,
		size:    int(stat.Size),
		fd:      fd,
		data:    data,
		header:  header,
		logger:  logger,
		readPos: atomic.LoadUint64(&header.ReadPos),
	}

	logger.Info("Opened shared memory ring",
		zap.String("name", shmName),
		zap.Uint64("frames", header.NumFrames),
		zap.Uint64("frame_size", header.FrameSize),
	)

	return ring, nil
}

// Write writes a frame to the ring buffer
// Returns the frame index where data was written
func (r *SharedMemoryRing) Write(frameData []byte) (uint64, error) {
	if uint64(len(frameData)) > r.header.FrameSize {
		return 0, fmt.Errorf("frame data too large: %d > %d", len(frameData), r.header.FrameSize)
	}

	// Get current write position
	writePos := atomic.LoadUint64(&r.header.WritePos)
	frameIndex := writePos % r.header.NumFrames

	// Calculate offset in shared memory
	offset := HeaderSize + (frameIndex * r.header.FrameSize)

	// Copy data (this is the only copy - zero-copy from here to reader)
	copy(r.data[offset:offset+uint64(len(frameData))], frameData)

	// Advance write position atomically
	// Use memory fence to ensure write completes before position update
	atomic.StoreUint64(&r.header.WritePos, writePos+1)

	return frameIndex, nil
}

// Read reads the next available frame
// Returns nil if no new frames are available
func (r *SharedMemoryRing) Read() ([]byte, uint64, error) {
	// Get current write position
	writePos := atomic.LoadUint64(&r.header.WritePos)

	// Check if new data is available
	if r.readPos >= writePos {
		return nil, 0, nil // No new data
	}

	// Check for overflow (writer wrapped around)
	if writePos-r.readPos > r.header.NumFrames {
		// We've fallen behind, skip to latest - NumFrames
		r.logger.Warn("Reader fell behind, skipping frames",
			zap.Uint64("read_pos", r.readPos),
			zap.Uint64("write_pos", writePos),
			zap.Uint64("skipped", writePos-r.readPos-r.header.NumFrames),
		)
		r.readPos = writePos - r.header.NumFrames
	}

	// Calculate frame index
	frameIndex := r.readPos % r.header.NumFrames
	offset := HeaderSize + (frameIndex * r.header.FrameSize)

	// Return slice pointing to shared memory (zero-copy read)
	frameData := r.data[offset : offset+r.header.FrameSize]

	// Advance read position
	r.readPos++

	return frameData, frameIndex, nil
}

// GetWritePosition returns the current write position
func (r *SharedMemoryRing) GetWritePosition() uint64 {
	return atomic.LoadUint64(&r.header.WritePos)
}

// GetReadPosition returns this reader's current read position
func (r *SharedMemoryRing) GetReadPosition() uint64 {
	return r.readPos
}

// GetNumFrames returns the number of frames in the ring
func (r *SharedMemoryRing) GetNumFrames() uint64 {
	return r.header.NumFrames
}

// GetFrameSize returns the size of each frame
func (r *SharedMemoryRing) GetFrameSize() uint64 {
	return r.header.FrameSize
}

// GetName returns the shared memory name
func (r *SharedMemoryRing) GetName() string {
	return r.name
}

// AvailableFrames returns the number of unread frames
func (r *SharedMemoryRing) AvailableFrames() uint64 {
	writePos := atomic.LoadUint64(&r.header.WritePos)
	if writePos <= r.readPos {
		return 0
	}
	available := writePos - r.readPos
	if available > r.header.NumFrames {
		return r.header.NumFrames
	}
	return available
}

// Close closes the shared memory ring
func (r *SharedMemoryRing) Close() error {
	if r.data != nil {
		if err := syscall.Munmap(r.data); err != nil {
			r.logger.Error("Failed to munmap", zap.Error(err))
		}
		r.data = nil
	}

	if r.fd >= 0 {
		if err := syscall.Close(r.fd); err != nil {
			r.logger.Error("Failed to close fd", zap.Error(err))
		}
		r.fd = -1
	}

	r.logger.Info("Closed shared memory ring", zap.String("name", r.name))
	return nil
}

// Destroy closes and unlinks the shared memory
// Should only be called by the creator
func (r *SharedMemoryRing) Destroy() error {
	if err := r.Close(); err != nil {
		return err
	}

	if err := syscall.Unlink(r.name); err != nil {
		r.logger.Error("Failed to unlink shm", zap.Error(err), zap.String("name", r.name))
		return err
	}

	r.logger.Info("Destroyed shared memory ring", zap.String("name", r.name))
	return nil
}

// Stats returns statistics about the ring buffer
type RingStats struct {
	Name          string
	TotalSize     uint64
	FrameSize     uint64
	NumFrames     uint64
	WritePosition uint64
	ReadPosition  uint64
	Available     uint64
	UsagePercent  float64
}

// GetStats returns current statistics
func (r *SharedMemoryRing) GetStats() RingStats {
	writePos := atomic.LoadUint64(&r.header.WritePos)
	available := r.AvailableFrames()

	return RingStats{
		Name:          r.name,
		TotalSize:     r.header.TotalSize,
		FrameSize:     r.header.FrameSize,
		NumFrames:     r.header.NumFrames,
		WritePosition: writePos,
		ReadPosition:  r.readPos,
		Available:     available,
		UsagePercent:  float64(available) / float64(r.header.NumFrames) * 100,
	}
}
