package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

const (
	deviceManagerInterface = "ie.fio.OllamaProxy.DeviceManager"
	deviceManagerPath      = "/ie/fio/OllamaProxy/DeviceManager"

	deviceInterface = "ie.fio.OllamaProxy.Device"
	deviceBasePath  = "/ie/fio/OllamaProxy/Device"
)

// DeviceManager manages device registration and access control via D-Bus
type DeviceManager struct {
	conn         *dbus.Conn
	devices      map[string]*Device
	accessGrants map[string]*AccessGrant // key: deviceID_clientID
	mu           sync.RWMutex
	props        *prop.Properties
	logger       *zap.Logger
	udevMonitor  *UdevMonitor
	polkit       *PolkitAuthorizer
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewDeviceManager creates a new device manager and registers it on D-Bus
func NewDeviceManager() (*DeviceManager, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	dm := &DeviceManager{
		conn:         conn,
		devices:      make(map[string]*Device),
		accessGrants: make(map[string]*AccessGrant),
		logger:       logging.Logger,
		polkit:       NewPolkitAuthorizer(conn, logging.Logger),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Export the device manager on D-Bus
	if err := dm.export(); err != nil {
		conn.Close()
		cancel()
		return nil, fmt.Errorf("failed to export device manager: %w", err)
	}

	// Start udev monitoring for hotplug events
	udevMonitor, err := NewUdevMonitor()
	if err != nil {
		dm.logger.Warn("Failed to start udev monitor", zap.Error(err))
	} else {
		dm.udevMonitor = udevMonitor
		go dm.handleUdevEvents()
	}

	dm.logger.Info("Device manager started",
		zap.String("path", deviceManagerPath),
		zap.Bool("polkit_enabled", true),
	)

	return dm, nil
}

// export registers the device manager on D-Bus
func (dm *DeviceManager) export() error {
	// Create introspection data
	intro := &introspect.Node{
		Name: deviceManagerPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: deviceManagerInterface,
				Methods: []introspect.Method{
					{
						Name: "RegisterDevice",
						Args: []introspect.Arg{
							{Name: "device_type", Type: "s", Direction: "in"},
							{Name: "device_path", Type: "s", Direction: "in"},
							{Name: "device_name", Type: "s", Direction: "in"},
							{Name: "capabilities", Type: "a{sv}", Direction: "in"},
							{Name: "device_id", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "UnregisterDevice",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s", Direction: "in"},
						},
					},
					{
						Name: "ListDevices",
						Args: []introspect.Arg{
							{Name: "device_type", Type: "s", Direction: "in"},
							{Name: "devices", Type: "aa{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetDevice",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s", Direction: "in"},
							{Name: "device", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "RequestDeviceAccess",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s", Direction: "in"},
							{Name: "client_id", Type: "s", Direction: "in"},
							{Name: "grant_id", Type: "s", Direction: "out"},
							{Name: "shared_memory_path", Type: "s", Direction: "out"},
							{Name: "unix_socket_path", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "ReleaseDeviceAccess",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s", Direction: "in"},
							{Name: "client_id", Type: "s", Direction: "in"},
						},
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "DeviceAdded",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s"},
							{Name: "device_type", Type: "s"},
							{Name: "device_name", Type: "s"},
						},
					},
					{
						Name: "DeviceRemoved",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s"},
						},
					},
					{
						Name: "DeviceStateChanged",
						Args: []introspect.Arg{
							{Name: "device_id", Type: "s"},
							{Name: "old_state", Type: "s"},
							{Name: "new_state", Type: "s"},
						},
					},
				},
				Properties: []introspect.Property{
					{
						Name:   "TotalDevices",
						Type:   "i",
						Access: "read",
					},
					{
						Name:   "AvailableDevices",
						Type:   "i",
						Access: "read",
					},
				},
			},
		},
	}

	// Export introspection
	if err := dm.conn.Export(intro, deviceManagerPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Export methods
	if err := dm.conn.Export(dm, deviceManagerPath, deviceManagerInterface); err != nil {
		return fmt.Errorf("failed to export device manager: %w", err)
	}

	// Setup properties
	props, err := prop.Export(dm.conn, deviceManagerPath, dm.createPropSpec())
	if err != nil {
		return fmt.Errorf("failed to export properties: %w", err)
	}
	dm.props = props

	return nil
}

// createPropSpec creates the property specification for D-Bus
func (dm *DeviceManager) createPropSpec() map[string]map[string]*prop.Prop {
	return map[string]map[string]*prop.Prop{
		deviceManagerInterface: {
			"TotalDevices": {
				Value:    dm.getTotalDevices,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"AvailableDevices": {
				Value:    dm.getAvailableDevices,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
}

// D-Bus Methods

// RegisterDevice registers a new device with the manager
func (dm *DeviceManager) RegisterDevice(deviceType, devicePath, deviceName string, capabilities map[string]dbus.Variant, sender dbus.Sender) (string, *dbus.Error) {
	// Allow system-initiated registrations (auto-discovery)
	if string(sender) != "ie.fio.OllamaProxy.System" {
		// Check Polkit authorization for device registration
		authorized, err := dm.polkit.CheckDeviceRegister(string(sender))
		if err != nil {
			dm.logger.Error("Failed to check authorization",
				zap.String("sender", string(sender)),
				zap.Error(err),
			)
			return "", dbus.MakeFailedError(fmt.Errorf("authorization check failed: %w", err))
		}

		if !authorized {
			dm.polkit.LogAuditEvent(string(sender), "register_device", deviceType, false)
			dm.logger.Warn("Unauthorized device registration attempt",
				zap.String("sender", string(sender)),
				zap.String("device_type", deviceType),
			)
			return "", dbus.MakeFailedError(fmt.Errorf("permission denied: not authorized to register devices"))
		}

		dm.polkit.LogAuditEvent(string(sender), "register_device", deviceType, true)
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Generate unique device ID
	deviceID := fmt.Sprintf("%s-%s-%d", deviceType, deviceName, time.Now().UnixNano())

	// Convert D-Bus variants to map[string]interface{}
	caps := make(map[string]interface{})
	for k, v := range capabilities {
		caps[k] = v.Value()
	}

	device := &Device{
		ID:           deviceID,
		Type:         DeviceType(deviceType),
		Name:         deviceName,
		Path:         devicePath,
		Capabilities: caps,
		State:        DeviceStateAvailable,
		RegisteredAt: time.Now(),
	}

	dm.devices[deviceID] = device

	dm.logger.Info("Device registered",
		zap.String("device_id", deviceID),
		zap.String("type", deviceType),
		zap.String("name", deviceName),
		zap.String("path", devicePath),
		zap.String("sender", string(sender)))

	// Emit DeviceAdded signal
	if err := dm.conn.Emit(deviceManagerPath, deviceManagerInterface+".DeviceAdded", deviceID, deviceType, deviceName); err != nil {
		dm.logger.Error("Failed to emit DeviceAdded signal", zap.Error(err))
	}

	// Update properties
	dm.props.SetMust(deviceManagerInterface, "TotalDevices", dm.getTotalDevices())
	dm.props.SetMust(deviceManagerInterface, "AvailableDevices", dm.getAvailableDevices())

	return deviceID, nil
}

// UnregisterDevice removes a device from the manager
func (dm *DeviceManager) UnregisterDevice(deviceID string) *dbus.Error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device, exists := dm.devices[deviceID]
	if !exists {
		return dbus.MakeFailedError(fmt.Errorf("device not found: %s", deviceID))
	}

	delete(dm.devices, deviceID)

	dm.logger.Info("Device unregistered",
		zap.String("device_id", deviceID),
		zap.String("name", device.Name))

	// Emit DeviceRemoved signal
	if err := dm.conn.Emit(deviceManagerPath, deviceManagerInterface+".DeviceRemoved", deviceID); err != nil {
		dm.logger.Error("Failed to emit DeviceRemoved signal", zap.Error(err))
	}

	// Update properties
	dm.props.SetMust(deviceManagerInterface, "TotalDevices", dm.getTotalDevices())
	dm.props.SetMust(deviceManagerInterface, "AvailableDevices", dm.getAvailableDevices())

	return nil
}

// ListDevices returns a list of devices, optionally filtered by type
func (dm *DeviceManager) ListDevices(deviceType string) ([]map[string]dbus.Variant, *dbus.Error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var result []map[string]dbus.Variant

	for _, device := range dm.devices {
		// Filter by type if specified
		if deviceType != "" && string(device.Type) != deviceType {
			continue
		}

		result = append(result, device.ToDBusVariant())
	}

	return result, nil
}

// GetDevice returns information about a specific device
func (dm *DeviceManager) GetDevice(deviceID string) (map[string]dbus.Variant, *dbus.Error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	device, exists := dm.devices[deviceID]
	if !exists {
		return nil, dbus.MakeFailedError(fmt.Errorf("device not found: %s", deviceID))
	}

	return device.ToDBusVariant(), nil
}

// RequestDeviceAccess grants access to a device for a client
func (dm *DeviceManager) RequestDeviceAccess(deviceID, clientID string, sender dbus.Sender) (string, string, string, *dbus.Error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device, exists := dm.devices[deviceID]
	if !exists {
		return "", "", "", dbus.MakeFailedError(fmt.Errorf("device not found: %s", deviceID))
	}

	// Check Polkit authorization for device access based on type
	authorized, err := dm.polkit.CheckDeviceAccess(string(sender), device.Type)
	if err != nil {
		dm.logger.Error("Failed to check device access authorization",
			zap.String("sender", string(sender)),
			zap.String("device_id", deviceID),
			zap.Error(err),
		)
		return "", "", "", dbus.MakeFailedError(fmt.Errorf("authorization check failed: %w", err))
	}

	if !authorized {
		dm.polkit.LogAuditEvent(string(sender), "access_device", deviceID, false)
		dm.logger.Warn("Unauthorized device access attempt",
			zap.String("sender", string(sender)),
			zap.String("device_id", deviceID),
			zap.String("device_type", string(device.Type)),
		)
		return "", "", "", dbus.MakeFailedError(fmt.Errorf("permission denied: not authorized to access %s devices", device.Type))
	}

	dm.polkit.LogAuditEvent(string(sender), "access_device", deviceID, true)

	if device.GetState() == DeviceStateError || device.GetState() == DeviceStateOffline {
		return "", "", "", dbus.MakeFailedError(fmt.Errorf("device is not available: %s", device.GetState()))
	}

	// Create access grant
	grantID := fmt.Sprintf("%s_%s", deviceID, clientID)
	grant := &AccessGrant{
		DeviceID:         deviceID,
		ClientID:         clientID,
		GrantedAt:        time.Now(),
		SharedMemoryPath: fmt.Sprintf("/ollama-proxy-shm-%s", deviceID),
		UnixSocketPath:   fmt.Sprintf("/tmp/ollama-proxy-%s.sock", deviceID),
	}

	dm.accessGrants[grantID] = grant

	// Update device state
	oldState := device.GetState()
	device.SetState(DeviceStateInUse)
	device.UpdateLastUsed()

	dm.logger.Info("Device access granted",
		zap.String("device_id", deviceID),
		zap.String("client_id", clientID),
		zap.String("grant_id", grantID),
		zap.String("sender", string(sender)))

	// Emit state changed signal
	if err := dm.conn.Emit(deviceManagerPath, deviceManagerInterface+".DeviceStateChanged",
		deviceID, string(oldState), string(DeviceStateInUse)); err != nil {
		dm.logger.Error("Failed to emit DeviceStateChanged signal", zap.Error(err))
	}

	return grantID, grant.SharedMemoryPath, grant.UnixSocketPath, nil
}

// ReleaseDeviceAccess releases access to a device for a client
func (dm *DeviceManager) ReleaseDeviceAccess(deviceID, clientID string) *dbus.Error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	grantID := fmt.Sprintf("%s_%s", deviceID, clientID)
	grant, exists := dm.accessGrants[grantID]
	if !exists {
		return dbus.MakeFailedError(fmt.Errorf("access grant not found"))
	}

	delete(dm.accessGrants, grantID)

	device, exists := dm.devices[deviceID]
	if exists {
		oldState := device.GetState()
		device.SetState(DeviceStateAvailable)

		// Emit state changed signal
		if err := dm.conn.Emit(deviceManagerPath, deviceManagerInterface+".DeviceStateChanged",
			deviceID, string(oldState), string(DeviceStateAvailable)); err != nil {
			dm.logger.Error("Failed to emit DeviceStateChanged signal", zap.Error(err))
		}
	}

	dm.logger.Info("Device access released",
		zap.String("device_id", deviceID),
		zap.String("client_id", clientID),
		zap.String("grant_id", grant.DeviceID))

	return nil
}

// Property getters

func (dm *DeviceManager) getTotalDevices() int32 {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return int32(len(dm.devices))
}

func (dm *DeviceManager) getAvailableDevices() int32 {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	count := int32(0)
	for _, device := range dm.devices {
		if device.GetState() == DeviceStateAvailable {
			count++
		}
	}
	return count
}

// handleUdevEvents processes hotplug events from udev
func (dm *DeviceManager) handleUdevEvents() {
	if dm.udevMonitor == nil {
		return
	}

	dm.udevMonitor.Start()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case event := <-dm.udevMonitor.Events():
			dm.handleUdevEvent(event)
		}
	}
}

// handleUdevEvent processes a single udev event
func (dm *DeviceManager) handleUdevEvent(event UdevEvent) {
	dm.logger.Debug("Udev event received",
		zap.String("action", event.Action),
		zap.String("devpath", event.DevPath),
		zap.String("subsystem", event.Subsystem),
		zap.String("devname", event.DevName))

	// Auto-register video devices
	if event.Subsystem == "video4linux" && event.Action == "add" {
		capabilities := map[string]dbus.Variant{
			"subsystem": dbus.MakeVariant(event.Subsystem),
			"devtype":   dbus.MakeVariant(event.DevType),
		}

		if _, err := dm.RegisterDevice(
			string(DeviceTypeCamera),
			event.DevName,
			fmt.Sprintf("Camera %s", event.DevName),
			capabilities,
			"ie.fio.OllamaProxy.System", // System sender for auto-discovery
		); err != nil {
			dm.logger.Error("Failed to auto-register camera",
				zap.String("devname", event.DevName),
				zap.Error(err))
		}
	}

	// Handle device removal
	if event.Action == "remove" {
		dm.mu.Lock()
		defer dm.mu.Unlock()

		for id, device := range dm.devices {
			if device.Path == event.DevName {
				delete(dm.devices, id)
				dm.logger.Info("Auto-removed device",
					zap.String("device_id", id),
					zap.String("path", event.DevName))

				// Emit removal signal
				if err := dm.conn.Emit(deviceManagerPath, deviceManagerInterface+".DeviceRemoved", id); err != nil {
					dm.logger.Error("Failed to emit DeviceRemoved signal", zap.Error(err))
				}
				break
			}
		}
	}
}

// StartAutoDiscovery starts monitoring for device hotplug events
func (dm *DeviceManager) StartAutoDiscovery() error {
	if dm.udevMonitor != nil {
		// Already started
		return nil
	}

	monitor, err := NewUdevMonitor()
	if err != nil {
		return fmt.Errorf("failed to create udev monitor: %w", err)
	}

	dm.udevMonitor = monitor
	monitor.Start()

	// Start processing udev events
	go dm.processUdevEvents()

	dm.logger.Info("Device auto-discovery started")
	return nil
}

// processUdevEvents processes hotplug events from udev
func (dm *DeviceManager) processUdevEvents() {
	for event := range dm.udevMonitor.Events() {
		dm.logger.Debug("Udev event received",
			zap.String("action", event.Action),
			zap.String("subsystem", event.Subsystem),
			zap.String("devname", event.DevName),
		)

		switch event.Action {
		case "add":
			// Auto-register the device
			deviceType := string(event.GetDeviceType())
			deviceName := ParseDeviceName(event)
			caps := GetCapabilities(event.DevPath)

			// Convert string caps to dbus.Variant
			dbuscaps := make(map[string]dbus.Variant)
			for k, v := range caps {
				dbuscaps[k] = dbus.MakeVariant(v)
			}

			deviceID, err := dm.RegisterDevice(deviceType, event.DevName, deviceName, dbuscaps, "ie.fio.OllamaProxy.System")
			if err != nil {
				dm.logger.Error("Failed to auto-register device",
					zap.String("devname", event.DevName),
					zap.Error(err),
				)
			} else {
				dm.logger.Info("Device auto-registered",
					zap.String("device_id", deviceID),
					zap.String("type", deviceType),
					zap.String("name", deviceName),
				)
			}

		case "remove":
			// Find and unregister device by path
			dm.mu.RLock()
			var deviceID string
			for id, dev := range dm.devices {
				if dev.Path == event.DevName {
					deviceID = id
					break
				}
			}
			dm.mu.RUnlock()

			if deviceID != "" {
				if err := dm.UnregisterDevice(deviceID); err != nil {
					dm.logger.Error("Failed to auto-unregister device",
						zap.String("device_id", deviceID),
						zap.Error(err),
					)
				} else {
					dm.logger.Info("Device auto-unregistered",
						zap.String("device_id", deviceID),
					)
				}
			}
		}
	}
}

// Stop stops the device manager and cleans up resources
func (dm *DeviceManager) Stop() error {
	dm.cancel()

	if dm.udevMonitor != nil {
		if err := dm.udevMonitor.Stop(); err != nil {
			dm.logger.Error("Failed to stop udev monitor", zap.Error(err))
		}
	}

	if dm.conn != nil {
		dm.conn.Close()
	}

	dm.logger.Info("Device manager stopped")
	return nil
}
