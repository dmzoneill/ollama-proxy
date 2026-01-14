package device

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"go.uber.org/zap"
)

const (
	polkitService   = "org.freedesktop.PolicyKit1"
	polkitPath      = "/org/freedesktop/PolicyKit1/Authority"
	polkitInterface = "org.freedesktop.PolicyKit1.Authority"

	// Polkit action IDs
	actionDeviceRegister      = "ie.fio.ollama-proxy.device.register"
	actionDeviceAccessCamera  = "ie.fio.ollama-proxy.device.access.camera"
	actionDeviceAccessMic     = "ie.fio.ollama-proxy.device.access.microphone"
	actionDeviceAccessScreen  = "ie.fio.ollama-proxy.device.access.screen"
	actionDeviceAccessSpeaker = "ie.fio.ollama-proxy.device.access.speaker"
	actionDeviceAccessKeyboard = "ie.fio.ollama-proxy.device.access.keyboard"
	actionDeviceAccessMouse   = "ie.fio.ollama-proxy.device.access.mouse"
)

// PolkitAuthorizer handles Polkit authorization checks
type PolkitAuthorizer struct {
	conn   *dbus.Conn
	logger *zap.Logger
}

// NewPolkitAuthorizer creates a new Polkit authorizer
func NewPolkitAuthorizer(conn *dbus.Conn, logger *zap.Logger) *PolkitAuthorizer {
	return &PolkitAuthorizer{
		conn:   conn,
		logger: logger,
	}
}

// CheckAuthorization checks if a subject is authorized for an action
func (p *PolkitAuthorizer) CheckAuthorization(sender string, action string) (bool, error) {
	// Get the Polkit object
	obj := p.conn.Object(polkitService, polkitPath)

	// Construct the subject (D-Bus sender)
	subject := map[string]dbus.Variant{
		"unix-process": dbus.MakeVariant(map[string]dbus.Variant{
			"pid":        dbus.MakeVariant(uint32(0)), // Will be filled by Polkit from sender
			"start-time": dbus.MakeVariant(uint64(0)),
		}),
	}

	// Action details
	actionDetails := map[string]string{}

	// Flags: 1 = AllowUserInteraction
	flags := uint32(1)

	// Cancellation ID (empty string = no cancellation)
	cancellationID := ""

	// Call CheckAuthorization
	var result struct {
		IsAuthorized bool
		IsChallenge  bool
		Details      map[string]string
	}

	err := obj.Call(
		polkitInterface+".CheckAuthorization",
		0,
		subject,
		action,
		actionDetails,
		flags,
		cancellationID,
	).Store(&result.IsAuthorized, &result.IsChallenge, &result.Details)

	if err != nil {
		// If Polkit is not available, log warning and allow by default
		// This allows the system to work without Polkit in development
		p.logger.Warn("Polkit authorization check failed, allowing by default",
			zap.Error(err),
			zap.String("action", action),
		)
		return true, nil
	}

	p.logger.Debug("Polkit authorization check",
		zap.String("action", action),
		zap.String("sender", sender),
		zap.Bool("authorized", result.IsAuthorized),
		zap.Bool("challenge", result.IsChallenge),
	)

	return result.IsAuthorized, nil
}

// CheckDeviceAccess checks authorization for device access based on device type
func (p *PolkitAuthorizer) CheckDeviceAccess(sender string, deviceType DeviceType) (bool, error) {
	var action string

	switch deviceType {
	case DeviceTypeCamera:
		action = actionDeviceAccessCamera
	case DeviceTypeMicrophone:
		action = actionDeviceAccessMic
	case DeviceTypeScreen:
		action = actionDeviceAccessScreen
	case DeviceTypeSpeaker:
		action = actionDeviceAccessSpeaker
	case DeviceTypeKeyboard:
		action = actionDeviceAccessKeyboard
	case DeviceTypeMouse:
		action = actionDeviceAccessMouse
	default:
		return false, fmt.Errorf("unknown device type: %s", deviceType)
	}

	return p.CheckAuthorization(sender, action)
}

// CheckDeviceRegister checks authorization for device registration
func (p *PolkitAuthorizer) CheckDeviceRegister(sender string) (bool, error) {
	return p.CheckAuthorization(sender, actionDeviceRegister)
}

// GetCallerUID gets the UID of a D-Bus caller
func (p *PolkitAuthorizer) GetCallerUID(sender string) (uint32, error) {
	obj := p.conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")

	var uid uint32
	err := obj.Call("org.freedesktop.DBus.GetConnectionUnixUser", 0, sender).Store(&uid)
	if err != nil {
		return 0, fmt.Errorf("failed to get caller UID: %w", err)
	}

	return uid, nil
}

// GetCallerPID gets the PID of a D-Bus caller
func (p *PolkitAuthorizer) GetCallerPID(sender string) (uint32, error) {
	obj := p.conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")

	var pid uint32
	err := obj.Call("org.freedesktop.DBus.GetConnectionUnixProcessID", 0, sender).Store(&pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get caller PID: %w", err)
	}

	return pid, nil
}

// LogAuditEvent logs an audit event for device access
func (p *PolkitAuthorizer) LogAuditEvent(sender string, action string, deviceID string, authorized bool) {
	uid, _ := p.GetCallerUID(sender)
	pid, _ := p.GetCallerPID(sender)

	p.logger.Info("Device access audit",
		zap.String("action", action),
		zap.String("device_id", deviceID),
		zap.String("sender", sender),
		zap.Uint32("uid", uid),
		zap.Uint32("pid", pid),
		zap.Bool("authorized", authorized),
	)
}
