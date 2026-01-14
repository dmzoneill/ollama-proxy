package device

import (
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// DeviceType represents the type of device
type DeviceType string

const (
	DeviceTypeMicrophone DeviceType = "microphone"
	DeviceTypeCamera     DeviceType = "camera"
	DeviceTypeScreen     DeviceType = "screen"
	DeviceTypeSpeaker    DeviceType = "speaker"
	DeviceTypeKeyboard   DeviceType = "keyboard"
	DeviceTypeMouse      DeviceType = "mouse"
)

// DeviceState represents the current state of a device
type DeviceState string

const (
	DeviceStateAvailable DeviceState = "available"
	DeviceStateInUse     DeviceState = "in-use"
	DeviceStateError     DeviceState = "error"
	DeviceStateOffline   DeviceState = "offline"
)

// Device represents a registered device in the system
type Device struct {
	ID           string                 `json:"id"`
	Type         DeviceType             `json:"type"`
	Name         string                 `json:"name"`
	Path         string                 `json:"path"` // Device path (/dev/video0, etc.)
	Capabilities map[string]interface{} `json:"capabilities"`
	State        DeviceState            `json:"state"`
	RegisteredAt time.Time              `json:"registered_at"`
	LastUsedAt   time.Time              `json:"last_used_at,omitempty"`
	mu           sync.RWMutex
}

// AccessGrant represents permission for a client to access a device
type AccessGrant struct {
	DeviceID         string    `json:"device_id"`
	ClientID         string    `json:"client_id"`
	GrantedAt        time.Time `json:"granted_at"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	SharedMemoryPath string    `json:"shared_memory_path,omitempty"`
	UnixSocketPath   string    `json:"unix_socket_path,omitempty"`
}

// SetState updates the device state (thread-safe)
func (d *Device) SetState(state DeviceState) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.State = state
}

// GetState returns the current device state (thread-safe)
func (d *Device) GetState() DeviceState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.State
}

// UpdateLastUsed updates the last used timestamp (thread-safe)
func (d *Device) UpdateLastUsed() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.LastUsedAt = time.Now()
}

// ToDBusVariant converts the device to a D-Bus variant map
func (d *Device) ToDBusVariant() map[string]dbus.Variant {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := map[string]dbus.Variant{
		"ID":           dbus.MakeVariant(d.ID),
		"Type":         dbus.MakeVariant(string(d.Type)),
		"Name":         dbus.MakeVariant(d.Name),
		"Path":         dbus.MakeVariant(d.Path),
		"State":        dbus.MakeVariant(string(d.State)),
		"RegisteredAt": dbus.MakeVariant(d.RegisteredAt.Unix()),
	}

	if !d.LastUsedAt.IsZero() {
		result["LastUsedAt"] = dbus.MakeVariant(d.LastUsedAt.Unix())
	}

	// Convert capabilities to variant
	caps := make(map[string]dbus.Variant)
	for k, v := range d.Capabilities {
		caps[k] = dbus.MakeVariant(v)
	}
	result["Capabilities"] = dbus.MakeVariant(caps)

	return result
}
