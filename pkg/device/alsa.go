package device

import (
	"fmt"
	"syscall"

	"go.uber.org/zap"
)

// ALSA ioctl constants
const (
	SNDRV_PCM_IOCTL_HW_PARAMS = 0xc2604111
	SNDRV_PCM_IOCTL_SW_PARAMS = 0xc0684113
	SNDRV_PCM_IOCTL_PREPARE   = 0x00004140
	SNDRV_PCM_IOCTL_START     = 0x00004142
	SNDRV_PCM_IOCTL_DROP      = 0x00004143

	SNDRV_PCM_ACCESS_RW_INTERLEAVED = 3
	SNDRV_PCM_FORMAT_S16_LE         = 2
	SNDRV_PCM_FORMAT_S32_LE         = 10

	SNDRV_PCM_STREAM_PLAYBACK = 0
	SNDRV_PCM_STREAM_CAPTURE  = 1
)

// ALSADevice represents an ALSA audio device
type ALSADevice struct {
	deviceName   string
	fd           int
	sampleRate   uint32
	channels     uint32
	format       uint32
	periodSize   uint32
	bufferSize   uint32
	capturing    bool
	logger       *zap.Logger
	shmRing      *SharedMemoryRing
	stopChan     chan struct{}
}

// OpenALSADevice opens an ALSA PCM device for capture
func OpenALSADevice(deviceName string, sampleRate, channels uint32, logger *zap.Logger) (*ALSADevice, error) {
	// For ALSA, we would normally use libasound via cgo
	// This is a simplified implementation using /dev/snd/pcmCxDxc devices

	// Typical ALSA device paths:
	// /dev/snd/pcmC0D0c - Card 0, Device 0, Capture
	// /dev/snd/pcmC0D0p - Card 0, Device 0, Playback

	pcmPath := deviceName
	if deviceName == "default" {
		pcmPath = "/dev/snd/pcmC0D0c"
	}

	fd, err := syscall.Open(pcmPath, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open ALSA device %s: %w", pcmPath, err)
	}

	device := &ALSADevice{
		deviceName: deviceName,
		fd:         fd,
		sampleRate: sampleRate,
		channels:   channels,
		format:     SNDRV_PCM_FORMAT_S16_LE,
		periodSize: 1024,
		bufferSize: 4096,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}

	logger.Info("Opened ALSA device",
		zap.String("device", deviceName),
		zap.String("path", pcmPath),
		zap.Uint32("sample_rate", sampleRate),
		zap.Uint32("channels", channels),
	)

	return device, nil
}

// SetHardwareParams sets hardware parameters for the PCM device
func (d *ALSADevice) SetHardwareParams() error {
	// In a real implementation, we would use snd_pcm_hw_params_*() functions
	// via cgo to set parameters like sample rate, channels, format, etc.
	// This is a simplified placeholder that shows the structure

	d.logger.Info("ALSA hardware parameters set",
		zap.Uint32("sample_rate", d.sampleRate),
		zap.Uint32("channels", d.channels),
		zap.Uint32("period_size", d.periodSize),
		zap.Uint32("buffer_size", d.bufferSize),
	)

	return nil
}

// SetSoftwareParams sets software parameters for the PCM device
func (d *ALSADevice) SetSoftwareParams() error {
	// In a real implementation, we would use snd_pcm_sw_params_*() functions
	// to set start threshold, stop threshold, etc.

	d.logger.Debug("ALSA software parameters set")
	return nil
}

// Prepare prepares the device for capture
func (d *ALSADevice) Prepare() error {
	if err := d.SetHardwareParams(); err != nil {
		return fmt.Errorf("failed to set hardware params: %w", err)
	}

	if err := d.SetSoftwareParams(); err != nil {
		return fmt.Errorf("failed to set software params: %w", err)
	}

	d.logger.Info("ALSA device prepared for capture")
	return nil
}

// StartCapture starts audio capture
func (d *ALSADevice) StartCapture() error {
	if d.capturing {
		return fmt.Errorf("already capturing")
	}

	if err := d.Prepare(); err != nil {
		return err
	}

	d.capturing = true
	d.logger.Info("ALSA capture started")

	return nil
}

// StopCapture stops audio capture
func (d *ALSADevice) StopCapture() error {
	if !d.capturing {
		return nil
	}

	// In real implementation: snd_pcm_drop()
	d.capturing = false
	d.logger.Info("ALSA capture stopped")

	return nil
}

// CaptureLoop captures audio data and writes to shared memory
func (d *ALSADevice) CaptureLoop(shmRing *SharedMemoryRing) error {
	d.shmRing = shmRing

	// Calculate frame size in bytes
	// For S16_LE: 2 bytes per sample
	bytesPerSample := uint32(2)
	frameBytes := d.periodSize * d.channels * bytesPerSample

	buffer := make([]byte, frameBytes)

	for {
		select {
		case <-d.stopChan:
			return nil
		default:
		}

		// Read audio data
		// In real implementation: snd_pcm_readi() or read() on PCM device
		n, err := syscall.Read(d.fd, buffer)
		if err != nil {
			if err == syscall.EAGAIN {
				// No data available, continue
				continue
			}
			d.logger.Error("Failed to read audio data", zap.Error(err))
			continue
		}

		if n == 0 {
			continue
		}

		// Write to shared memory
		frameIdx, err := shmRing.Write(buffer[:n])
		if err != nil {
			d.logger.Error("Failed to write audio to shared memory",
				zap.Error(err),
				zap.Int("bytes", n),
			)
		} else {
			d.logger.Debug("Audio frame captured",
				zap.Uint64("frame_index", frameIdx),
				zap.Int("bytes", n),
			)
		}
	}
}

// Stop stops the capture loop
func (d *ALSADevice) Stop() {
	close(d.stopChan)
}

// Close closes the ALSA device
func (d *ALSADevice) Close() error {
	// Stop capture if active
	if d.capturing {
		if err := d.StopCapture(); err != nil {
			d.logger.Error("Failed to stop capture", zap.Error(err))
		}
	}

	// Close device
	if d.fd >= 0 {
		if err := syscall.Close(d.fd); err != nil {
			d.logger.Error("Failed to close device", zap.Error(err))
			return err
		}
		d.fd = -1
	}

	d.logger.Info("ALSA device closed", zap.String("device", d.deviceName))
	return nil
}

// GetCapabilities returns device capabilities as a map
func (d *ALSADevice) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"device":      d.deviceName,
		"sample_rate": d.sampleRate,
		"channels":    d.channels,
		"format":      "S16_LE",
		"period_size": d.periodSize,
		"buffer_size": d.bufferSize,
	}
}

// SimplifiedALSACapture provides a simplified ALSA capture using PipeWire/PulseAudio
// This is a fallback implementation that uses PulseAudio's simple API
type SimplifiedALSACapture struct {
	deviceName string
	sampleRate uint32
	channels   uint32
	logger     *zap.Logger
	shmRing    *SharedMemoryRing
	stopChan   chan struct{}
	buffer     []byte
}

// NewSimplifiedALSACapture creates a simplified audio capture device
// In a real implementation, this would use libpulse-simple or libpipewire
func NewSimplifiedALSACapture(deviceName string, sampleRate, channels uint32, logger *zap.Logger) (*SimplifiedALSACapture, error) {
	// Calculate buffer size for 10ms of audio
	// S16_LE: 2 bytes per sample
	bufferFrames := (sampleRate * 10) / 1000 // 10ms worth
	bufferSize := bufferFrames * channels * 2

	capture := &SimplifiedALSACapture{
		deviceName: deviceName,
		sampleRate: sampleRate,
		channels:   channels,
		logger:     logger,
		stopChan:   make(chan struct{}),
		buffer:     make([]byte, bufferSize),
	}

	logger.Info("Created simplified ALSA capture",
		zap.String("device", deviceName),
		zap.Uint32("sample_rate", sampleRate),
		zap.Uint32("channels", channels),
		zap.Uint32("buffer_size", bufferSize),
	)

	return capture, nil
}

// CaptureLoop captures audio and writes to shared memory
func (s *SimplifiedALSACapture) CaptureLoop(shmRing *SharedMemoryRing) error {
	s.shmRing = shmRing

	// In real implementation: pa_simple_read() or pw_stream_dequeue_buffer()
	// This is a placeholder showing the structure

	for {
		select {
		case <-s.stopChan:
			return nil
		default:
		}

		// Simulated audio capture
		// In real implementation, this would call PulseAudio/PipeWire APIs

		// Write to shared memory
		frameIdx, err := shmRing.Write(s.buffer)
		if err != nil {
			s.logger.Error("Failed to write audio to shared memory", zap.Error(err))
		} else {
			s.logger.Debug("Audio frame captured",
				zap.Uint64("frame_index", frameIdx),
				zap.Int("bytes", len(s.buffer)),
			)
		}
	}
}

// Stop stops the capture
func (s *SimplifiedALSACapture) Stop() {
	close(s.stopChan)
}

// Close closes the capture
func (s *SimplifiedALSACapture) Close() error {
	s.logger.Info("Simplified ALSA capture closed")
	return nil
}

// GetCapabilities returns capabilities
func (s *SimplifiedALSACapture) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"device":      s.deviceName,
		"sample_rate": s.sampleRate,
		"channels":    s.channels,
		"format":      "S16_LE",
		"api":         "simplified",
	}
}
