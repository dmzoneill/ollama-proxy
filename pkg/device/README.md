# Device Registration System

This package implements a D-Bus-based device registration and management system for ollama-proxy, enabling applications to directly access hardware devices (cameras, microphones, screens, etc.) with ultra-low latency (<50μs).

## Features

- **D-Bus Device Manager** - System-wide device discovery and registration
- **Auto-discovery** - Automatic device detection via udev hotplug events
- **Multi-client Support** - Multiple applications can access the same device
- **Security** - Polkit-based permission management per device type
- **Thread-safe** - Concurrent access with RWMutex protection
- **Signal-based** - Real-time notifications for device state changes

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    D-Bus System Bus                          │
│          ie.fio.OllamaProxy.DeviceManager                    │
└─────────────────────────────────────────────────────────────┘
                           ▲
                           │
                ┌──────────┴──────────┐
                │                     │
        ┌───────▼────────┐    ┌──────▼──────┐
        │ DeviceManager  │    │ UdevMonitor │
        │ (manager.go)   │◄───┤ (udev.go)   │
        └────────────────┘    └─────────────┘
                │                     │
                │              Netlink Socket
                │             (hotplug events)
                ▼
        ┌────────────────┐
        │ Registered     │
        │ Devices Map    │
        └────────────────┘
```

## Installation

### 1. Install D-Bus Policy

```bash
sudo cp configs/dbus/ollama-proxy-devices.conf /etc/dbus-1/system.d/
sudo systemctl reload dbus
```

### 2. Install Polkit Policy

```bash
sudo cp configs/polkit/ie.fio.ollama-proxy.policy /usr/share/polkit-1/actions/
```

### 3. Add User to Groups

For camera access:
```bash
sudo usermod -aG video $USER
```

For microphone access:
```bash
sudo usermod -aG audio $USER
```

### 4. Enable in Configuration

Edit `config/config.yaml`:
```yaml
devices:
  enabled: true
  auto_discover: true
```

## Usage

### Starting the Device Manager

The device manager is automatically started when ollama-proxy starts (if enabled in config).

```go
// In main.go (already integrated)
deviceManager, err := device.NewDeviceManager()
if err != nil {
    log.Fatal(err)
}

// Start auto-discovery
deviceManager.StartAutoDiscovery()
```

### Using D-Bus CLI (busctl)

#### List All Devices

```bash
busctl --system call \
  ie.fio.OllamaProxy.DeviceManager \
  /ie/fio/OllamaProxy/DeviceManager \
  ie.fio.OllamaProxy.DeviceManager \
  ListDevices s ""
```

#### Register a Device

```bash
busctl --system call \
  ie.fio.OllamaProxy.DeviceManager \
  /ie/fio/OllamaProxy/DeviceManager \
  ie.fio.OllamaProxy.DeviceManager \
  RegisterDevice ssssa{sv} \
  "microphone" "/dev/snd/pcmC0D0c" "Built-in Microphone" 0
```

#### Request Device Access

```bash
busctl --system call \
  ie.fio.OllamaProxy.DeviceManager \
  /ie/fio/OllamaProxy/DeviceManager \
  ie.fio.OllamaProxy.DeviceManager \
  RequestDeviceAccess ss "device-id-here" "my-client-id"
```

Response includes:
- `grantID` - Unique access grant identifier
- `shmPath` - Shared memory path for data transfer
- `udsPath` - Unix socket path for metadata

### Monitor Device Events

```bash
busctl --system monitor ie.fio.OllamaProxy.DeviceManager
```

You'll see signals like:
- `DeviceAdded` - When a new device is plugged in
- `DeviceRemoved` - When a device is unplugged
- `DeviceStateChanged` - When device state changes

### Using from Go Code

```go
import "github.com/daoneill/ollama-proxy/pkg/device"
import "github.com/godbus/dbus/v5"

// Connect to system bus
conn, err := dbus.SystemBus()
if err != nil {
    return err
}

obj := conn.Object(
    "ie.fio.OllamaProxy.DeviceManager",
    "/ie/fio/OllamaProxy/DeviceManager",
)

// List all microphones
var devices []map[string]dbus.Variant
err = obj.Call(
    "ie.fio.OllamaProxy.DeviceManager.ListDevices",
    0,
    "microphone",
).Store(&devices)

// Request access to a device
var grantID, shmPath, udsPath string
err = obj.Call(
    "ie.fio.OllamaProxy.DeviceManager.RequestDeviceAccess",
    0,
    deviceID,
    "my-app",
).Store(&grantID, &shmPath, &udsPath)

// Release access when done
err = obj.Call(
    "ie.fio.OllamaProxy.DeviceManager.ReleaseDeviceAccess",
    0,
    deviceID,
    "my-app",
).Err
```

## Device Types

Supported device types:

- `microphone` - Audio capture devices (ALSA pcmC*D*c)
- `camera` - Video capture devices (V4L2 /dev/video*)
- `screen` - Screen capture (virtual devices)
- `speaker` - Audio playback devices (ALSA pcmC*D*p)
- `keyboard` - Input devices (requires admin auth)
- `mouse` - Pointing devices (requires admin auth)

## Security

### Permission Levels

Different device types have different permission requirements:

| Device Type | Permission Level | Polkit Action |
|-------------|-----------------|---------------|
| Speaker | None (yes) | `ie.fio.ollama-proxy.device.access.speaker` |
| Camera | User auth (auth_self_keep) | `ie.fio.ollama-proxy.device.access.camera` |
| Microphone | User auth (auth_self_keep) | `ie.fio.ollama-proxy.device.access.microphone` |
| Screen | User auth (auth_self_keep) | `ie.fio.ollama-proxy.device.access.screen` |
| Keyboard | Admin auth (auth_admin) | `ie.fio.ollama-proxy.device.access.keyboard` |
| Mouse | Admin auth (auth_admin) | `ie.fio.ollama-proxy.device.access.mouse` |

### Group-Based Access

- Members of `video` group can request camera access
- Members of `audio` group can request microphone/speaker access
- Members of `wheel` or `sudo` can register/unregister devices
- Console users (at_console) can request device access

## Testing

### Run Unit Tests

```bash
# Test device manager
go test ./pkg/device/manager_test.go -v

# Test udev monitor
go test ./pkg/device/udev_test.go -v

# Run all tests
go test ./pkg/device/... -v

# Run benchmarks
go test ./pkg/device/... -bench=. -benchmem
```

### Manual Testing with Real Hardware

```bash
# Run the integration test (requires USB device)
go test ./pkg/device/udev_test.go -run TestUdevMonitor_Integration -v

# Then plug/unplug a USB camera or microphone
```

### Test Permission Requirements

Some tests require elevated privileges:

```bash
# Tests that need CAP_NET_ADMIN (for netlink sockets)
sudo -E go test ./pkg/device/udev_test.go -run TestUdevMonitor_Creation -v

# Tests that need D-Bus system bus access
go test ./pkg/device/manager_test.go -v
```

Tests gracefully skip if permissions are insufficient.

## Performance Targets

Phase 1 (Current - D-Bus Control Plane):
- Device registration: <2ms
- Device lookup: <1ms
- Access grant creation: <2ms

Phase 3 (Future - Shared Memory Data Plane):
- Per-frame latency: <50μs
- Shared memory write: <1μs
- Zero-copy data transfer

## Files

- `types.go` - Core type definitions (Device, DeviceState, DeviceType)
- `manager.go` - D-Bus service implementation
- `udev.go` - Hotplug detection via netlink
- `manager_test.go` - Unit tests for DeviceManager
- `udev_test.go` - Unit tests for UdevMonitor

## Troubleshooting

### D-Bus Service Not Found

```bash
# Check if service is registered
busctl --system status ie.fio.OllamaProxy.DeviceManager

# Check if proxy is running
ps aux | grep ollama-proxy

# Check D-Bus logs
journalctl -u dbus -f
```

### Permission Denied Errors

```bash
# Check your groups
groups

# Add to video group if needed
sudo usermod -aG video $USER

# Log out and back in for group changes to take effect
```

### Udev Events Not Detected

```bash
# Check if udev monitor started
# Look for log message: "Device auto-discovery started"

# Test netlink socket creation (requires CAP_NET_ADMIN)
sudo -E go run pkg/device/udev.go

# Monitor kernel uevents directly
udevadm monitor
```

### No Devices Detected

```bash
# List video devices
ls -l /dev/video*

# List audio devices
ls -l /dev/snd/pcm*

# Check udev database
udevadm info /dev/video0
```

## Next Steps (Future Phases)

**Phase 2: gRPC Device Service**
- Protobuf definitions for device API
- Streaming RPCs for device data

**Phase 3: Shared Memory**
- Ring buffer implementation
- Zero-copy data transfer

**Phase 4: Unix Domain Sockets**
- Metadata notifications
- Low-latency control channel

**Phase 5: Device Drivers**
- V4L2 camera driver
- ALSA audio driver
- Evdev input driver

## References

- [D-Bus Specification](https://dbus.freedesktop.org/doc/dbus-specification.html)
- [Polkit Authorization](https://www.freedesktop.org/software/polkit/docs/latest/)
- [Udev Documentation](https://www.kernel.org/doc/html/latest/admin-guide/udev.html)
- [Video4Linux2 API](https://www.kernel.org/doc/html/latest/userspace-api/media/v4l/v4l2.html)
- [ALSA API](https://www.alsa-project.org/alsa-doc/alsa-lib/)
