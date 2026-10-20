package device

import (
	"fmt"
	"syscall"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/safeconv"
)

// V4L2 ioctl constants
const (
	VidiocQuerycap           = 0x80685600
	VidiocEnumFmt            = 0xc0405602
	VidiocGFmt               = 0xc0d05604
	VidiocSFmt               = 0xc0d05605
	VidiocReqbufs            = 0xc0145608
	VidiocQuerybuf           = 0xc0445609
	VidiocQbuf               = 0xc058560f
	VidiocDqbuf              = 0xc0585611
	VidiocStreamon           = 0x40045612
	VidiocStreamoff          = 0x40045613
	VidiocEnumFramesizes     = 0xc02c564a
	VidiocEnumFrameintervals = 0xc034564b

	V4l2BufTypeVideoCapture = 1
	V4l2MemoryMmap          = 1
	V4l2FieldInterlaced     = 4

	// Pixel formats
	V4l2PixFmtYuyv  = 0x56595559 // YUYV 4:2:2
	V4l2PixFmtMjpeg = 0x47504a4d // Motion-JPEG
	V4l2PixFmtRgb24 = 0x33424752 // 24-bit RGB
)

// V4L2Capability represents device capabilities
type V4L2Capability struct {
	Driver       [16]byte
	Card         [32]byte
	BusInfo      [32]byte
	Version      uint32
	Capabilities uint32
	DeviceCaps   uint32
	Reserved     [3]uint32
}

// V4L2PixFormat represents pixel format
type V4L2PixFormat struct {
	Width        uint32
	Height       uint32
	PixelFormat  uint32
	Field        uint32
	BytesPerLine uint32
	SizeImage    uint32
	Colorspace   uint32
	Priv         uint32
	Flags        uint32
	YcbcrEnc     uint32
	Quantization uint32
	XferFunc     uint32
}

// V4L2Format represents format structure
type V4L2Format struct {
	Type uint32
	Fmt  [200]byte // Union of different format types
}

// V4L2RequestBuffers requests buffers for streaming
type V4L2RequestBuffers struct {
	Count        uint32
	Type         uint32
	Memory       uint32
	Capabilities uint32
	Reserved     [1]uint32
}

// V4L2Buffer matches the Linux v4l2_buffer ioctl memory layout.
// Field order is the kernel ABI and must not be rearranged for Go alignment.
//
//nolint:govet // kernel ioctl ABI; field order is load-bearing
type V4L2Buffer struct {
	Index     uint32
	Type      uint32
	BytesUsed uint32
	Flags     uint32
	Field     uint32
	Timestamp syscall.Timeval
	Timecode  [4]uint32
	Sequence  uint32
	Memory    uint32
	Offset    uint32
	Length    uint32
	Reserved2 uint32
	RequestFd int32
	Reserved  uint32
}

// V4L2Device represents a V4L2 camera device
type V4L2Device struct {
	logger    *zap.Logger
	shmRing   *SharedMemoryRing
	stopChan  chan struct{}
	path      string
	buffers   []V4L2MappedBuffer
	fd        int
	caps      V4L2Capability
	format    V4L2PixFormat
	streaming bool
}

// V4L2MappedBuffer represents a memory-mapped buffer
type V4L2MappedBuffer struct {
	data   []byte
	length uint32
	offset uint32
}

// OpenV4L2Device opens a V4L2 video device
func OpenV4L2Device(path string, logger *zap.Logger) (*V4L2Device, error) {
	// Open device
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %s: %w", path, err)
	}

	device := &V4L2Device{
		path:     path,
		fd:       fd,
		logger:   logger,
		stopChan: make(chan struct{}),
	}

	// Query capabilities
	if err := device.queryCapabilities(); err != nil {
		warnErr("failed to close v4l2 device", syscall.Close(fd), zap.String("path", path))
		return nil, fmt.Errorf("failed to query capabilities: %w", err)
	}

	logger.Info("Opened V4L2 device",
		zap.String("path", path),
		zap.String("driver", string(device.caps.Driver[:])),
		zap.String("card", string(device.caps.Card[:])),
	)

	return device, nil
}

// queryCapabilities queries device capabilities
func (d *V4L2Device) queryCapabilities() error {
	if err := ioctl(uintptr(d.fd), uintptr(VidiocQuerycap), &d.caps); err != nil {
		return fmt.Errorf("VidiocQuerycap failed: %w", err)
	}
	return nil
}

// SetFormat sets the video format
func (d *V4L2Device) SetFormat(width, height, pixelFormat uint32) error {
	var v4l2fmt V4L2Format
	v4l2fmt.Type = V4l2BufTypeVideoCapture

	// Get current format first
	if err := ioctl(uintptr(d.fd), uintptr(VidiocGFmt), &v4l2fmt); err != nil {
		return fmt.Errorf("VidiocGFmt failed: %w", err)
	}

	// Decode pixel format from union
	pixFmt := overlayStruct[V4L2PixFormat](v4l2fmt.Fmt[:])
	pixFmt.Width = width
	pixFmt.Height = height
	pixFmt.PixelFormat = pixelFormat
	pixFmt.Field = V4l2FieldInterlaced

	// Set new format
	if err := ioctl(uintptr(d.fd), uintptr(VidiocSFmt), &v4l2fmt); err != nil {
		return fmt.Errorf("VidiocSFmt failed: %w", err)
	}

	// Store format
	d.format = *pixFmt

	d.logger.Info("V4L2 format set",
		zap.Uint32("width", d.format.Width),
		zap.Uint32("height", d.format.Height),
		zap.Uint32("pixel_format", d.format.PixelFormat),
		zap.Uint32("bytes_per_line", d.format.BytesPerLine),
		zap.Uint32("size_image", d.format.SizeImage),
	)

	return nil
}

// RequestBuffers requests buffers for memory mapping
func (d *V4L2Device) RequestBuffers(count uint32) error {
	var reqbuf V4L2RequestBuffers
	reqbuf.Count = count
	reqbuf.Type = V4l2BufTypeVideoCapture
	reqbuf.Memory = V4l2MemoryMmap

	if err := ioctl(uintptr(d.fd), uintptr(VidiocReqbufs), &reqbuf); err != nil {
		return fmt.Errorf("VidiocReqbufs failed: %w", err)
	}

	if reqbuf.Count < count {
		return fmt.Errorf("insufficient buffer memory: requested %d, got %d", count, reqbuf.Count)
	}

	d.logger.Info("V4L2 buffers requested",
		zap.Uint32("count", reqbuf.Count),
	)

	return nil
}

// MapBuffers memory-maps the device buffers
func (d *V4L2Device) MapBuffers(count uint32) error {
	d.buffers = make([]V4L2MappedBuffer, count)

	for i := uint32(0); i < count; i++ {
		var buf V4L2Buffer
		buf.Type = V4l2BufTypeVideoCapture
		buf.Memory = V4l2MemoryMmap
		buf.Index = i

		// Query buffer
		if err := ioctl(uintptr(d.fd), uintptr(VidiocQuerybuf), &buf); err != nil {
			return fmt.Errorf("VidiocQuerybuf failed for buffer %d: %w", i, err)
		}

		// Memory map buffer
		data, err := syscall.Mmap(
			d.fd,
			int64(buf.Offset),
			int(buf.Length),
			syscall.PROT_READ|syscall.PROT_WRITE,
			syscall.MAP_SHARED,
		)
		if err != nil {
			return fmt.Errorf("mmap failed for buffer %d: %w", i, err)
		}

		d.buffers[i] = V4L2MappedBuffer{
			data:   data,
			length: buf.Length,
			offset: buf.Offset,
		}

		d.logger.Debug("V4L2 buffer mapped",
			zap.Uint32("index", i),
			zap.Uint32("length", buf.Length),
			zap.Uint32("offset", buf.Offset),
		)
	}

	return nil
}

// QueueBuffer queues a buffer for capture
func (d *V4L2Device) QueueBuffer(index uint32) error {
	var buf V4L2Buffer
	buf.Type = V4l2BufTypeVideoCapture
	buf.Memory = V4l2MemoryMmap
	buf.Index = index

	if err := ioctl(uintptr(d.fd), uintptr(VidiocQbuf), &buf); err != nil {
		return fmt.Errorf("VidiocQbuf failed: %w", err)
	}

	return nil
}

// DequeueBuffer dequeues a captured buffer
func (d *V4L2Device) DequeueBuffer() (*V4L2Buffer, error) {
	var buf V4L2Buffer
	buf.Type = V4l2BufTypeVideoCapture
	buf.Memory = V4l2MemoryMmap

	if err := ioctl(uintptr(d.fd), uintptr(VidiocDqbuf), &buf); err != nil {
		return nil, fmt.Errorf("VidiocDqbuf failed: %w", err)
	}

	return &buf, nil
}

// StartStreaming starts video capture
func (d *V4L2Device) StartStreaming() error {
	if d.streaming {
		return fmt.Errorf("already streaming")
	}

	// Queue all buffers
	for i := range d.buffers {
		if err := d.QueueBuffer(uint32(i)); err != nil {
			return fmt.Errorf("failed to queue buffer %d: %w", i, err)
		}
	}

	// Start streaming
	bufType := uint32(V4l2BufTypeVideoCapture)
	if err := ioctl(uintptr(d.fd), uintptr(VidiocStreamon), &bufType); err != nil {
		return fmt.Errorf("VidiocStreamon failed: %w", err)
	}

	d.streaming = true
	d.logger.Info("V4L2 streaming started")

	return nil
}

// StopStreaming stops video capture
func (d *V4L2Device) StopStreaming() error {
	if !d.streaming {
		return nil
	}

	bufType := uint32(V4l2BufTypeVideoCapture)
	if err := ioctl(uintptr(d.fd), uintptr(VidiocStreamoff), &bufType); err != nil {
		return fmt.Errorf("VidiocStreamoff failed: %w", err)
	}

	d.streaming = false
	d.logger.Info("V4L2 streaming stopped")

	return nil
}

// CaptureLoop captures frames and writes them to shared memory
func (d *V4L2Device) CaptureLoop(shmRing *SharedMemoryRing) error {
	d.shmRing = shmRing

	for {
		select {
		case <-d.stopChan:
			return nil
		default:
		}

		// Dequeue buffer
		buf, err := d.DequeueBuffer()
		if err != nil {
			d.logger.Error("Failed to dequeue buffer", zap.Error(err))
			continue
		}

		// Get buffer data
		frameData := d.buffers[buf.Index].data[:buf.BytesUsed]

		// Write to shared memory
		frameIdx, err := shmRing.Write(frameData)
		if err != nil {
			d.logger.Error("Failed to write frame to shared memory",
				zap.Error(err),
				zap.Uint32("buffer_index", buf.Index),
			)
		} else {
			d.logger.Debug("Frame captured",
				zap.Uint64("frame_index", frameIdx),
				zap.Uint32("bytes", buf.BytesUsed),
			)
		}

		// Re-queue buffer
		if err := d.QueueBuffer(buf.Index); err != nil {
			d.logger.Error("Failed to re-queue buffer", zap.Error(err))
			return err
		}
	}
}

// StopCapture stops the capture loop
func (d *V4L2Device) StopCapture() {
	close(d.stopChan)
}

// Close closes the V4L2 device
func (d *V4L2Device) Close() error {
	// Stop streaming if active
	if d.streaming {
		if err := d.StopStreaming(); err != nil {
			d.logger.Error("Failed to stop streaming", zap.Error(err))
		}
	}

	// Unmap buffers
	for i, buf := range d.buffers {
		if err := syscall.Munmap(buf.data); err != nil {
			d.logger.Error("Failed to munmap buffer",
				zap.Int("index", i),
				zap.Error(err),
			)
		}
	}
	d.buffers = nil

	// Close device
	if d.fd >= 0 {
		if err := syscall.Close(d.fd); err != nil {
			d.logger.Error("Failed to close device", zap.Error(err))
			return err
		}
		d.fd = -1
	}

	d.logger.Info("V4L2 device closed", zap.String("path", d.path))
	return nil
}

// BufferData returns mapped capture bytes for index, limited to n bytes.
// The slice aliases device memory and is invalid after the buffer is queued again.
func (d *V4L2Device) BufferData(index, n uint32) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("v4l2 device is nil")
	}
	idx := safeconv.IntFromUint64(uint64(index))
	if idx < 0 || idx >= len(d.buffers) {
		return nil, fmt.Errorf("buffer index %d out of range", index)
	}
	data := d.buffers[idx].data
	limit := safeconv.IntFromUint64(uint64(n))
	if limit > len(data) {
		limit = len(data)
	}
	if limit < 0 {
		limit = 0
	}
	return data[:limit], nil
}

// GetCapabilities returns device capabilities as a map
func (d *V4L2Device) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"driver":         string(d.caps.Driver[:]),
		"card":           string(d.caps.Card[:]),
		"bus_info":       string(d.caps.BusInfo[:]),
		"width":          d.format.Width,
		"height":         d.format.Height,
		"pixel_format":   d.format.PixelFormat,
		"bytes_per_line": d.format.BytesPerLine,
		"size_image":     d.format.SizeImage,
	}
}
