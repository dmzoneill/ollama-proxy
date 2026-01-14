package efficiency

import (
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"
)

const (
	dbusInterface = "com.anthropic.OllamaProxy.Efficiency"
	dbusPath      = "/com/anthropic/OllamaProxy/Efficiency"
)

// DBusService exposes efficiency mode control via D-Bus
type DBusService struct {
	conn    *dbus.Conn
	manager *EfficiencyManager
	props   *prop.Properties
}

// NewDBusService creates a D-Bus service for mode control
func NewDBusService(manager *EfficiencyManager) (*DBusService, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		// Try session bus if system bus fails
		conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to D-Bus: %w", err)
		}
	}

	svc := &DBusService{
		conn:    conn,
		manager: manager,
	}

	return svc, nil
}

// Start registers the D-Bus service
func (ds *DBusService) Start() error {
	// Request name
	reply, err := ds.conn.RequestName(dbusInterface,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request D-Bus name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name already taken")
	}

	// Export methods
	err = ds.conn.Export(ds, dbusPath, dbusInterface)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus object: %w", err)
	}

	// Export introspection
	intro := introspect.NewIntrospectable(&introspect.Node{
		Name: dbusPath,
		Interfaces: []introspect.Interface{
			{
				Name: dbusInterface,
				Methods: []introspect.Method{
					{
						Name: "SetMode",
						Args: []introspect.Arg{
							{Name: "mode", Type: "s", Direction: "in"},
						},
					},
					{
						Name: "GetMode",
						Args: []introspect.Arg{
							{Name: "mode", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "GetEffectiveMode",
						Args: []introspect.Arg{
							{Name: "mode", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "ListModes",
						Args: []introspect.Arg{
							{Name: "modes", Type: "as", Direction: "out"},
						},
					},
					{
						Name: "GetModeInfo",
						Args: []introspect.Arg{
							{Name: "mode", Type: "s", Direction: "in"},
							{Name: "info", Type: "a{sv}", Direction: "out"},
						},
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "ModeChanged",
						Args: []introspect.Arg{
							{Name: "oldMode", Type: "s"},
							{Name: "newMode", Type: "s"},
						},
					},
				},
				Properties: []introspect.Property{
					{
						Name:   "CurrentMode",
						Type:   "s",
						Access: "readwrite",
					},
					{
						Name:   "EffectiveMode",
						Type:   "s",
						Access: "read",
					},
				},
			},
		},
	})

	err = ds.conn.Export(intro, dbusPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Setup properties
	ds.props, _ = prop.Export(ds.conn, dbusPath, ds.makePropertyMap())

	logging.Logger.Info("D-Bus Efficiency service started",
		zap.String("interface", dbusInterface),
	)
	return nil
}

// SetMode changes the efficiency mode (D-Bus method)
func (ds *DBusService) SetMode(mode string) *dbus.Error {
	oldMode := ds.manager.GetMode()

	var newMode EfficiencyMode
	switch mode {
	case "Performance":
		newMode = ModePerformance
	case "Balanced":
		newMode = ModeBalanced
	case "Efficiency":
		newMode = ModeEfficiency
	case "Quiet":
		newMode = ModeQuiet
	case "Auto":
		newMode = ModeAuto
	case "UltraEfficiency":
		newMode = ModeUltraEfficiency
	default:
		return dbus.MakeFailedError(fmt.Errorf("unknown mode: %s", mode))
	}

	ds.manager.SetMode(newMode)

	// Emit signal
	if ds.conn != nil {
		ds.conn.Emit(dbusPath, dbusInterface+".ModeChanged",
			oldMode.String(), newMode.String())
	}

	// Update property
	if ds.props != nil {
		ds.props.SetMust(dbusInterface, "CurrentMode", newMode.String())
	}

	logging.Logger.Info("Efficiency mode changed",
		zap.String("old_mode", oldMode.String()),
		zap.String("new_mode", newMode.String()),
	)

	return nil
}

// GetMode returns current mode (D-Bus method)
func (ds *DBusService) GetMode() (string, *dbus.Error) {
	mode := ds.manager.GetMode()
	return mode.String(), nil
}

// GetEffectiveMode returns effective mode (D-Bus method)
func (ds *DBusService) GetEffectiveMode() (string, *dbus.Error) {
	mode := ds.manager.GetEffectiveMode()
	return mode.String(), nil
}

// ListModes returns all available modes (D-Bus method)
func (ds *DBusService) ListModes() ([]string, *dbus.Error) {
	modes := AllModes()
	names := make([]string, len(modes))
	for i, m := range modes {
		names[i] = m.String()
	}
	return names, nil
}

// GetModeInfo returns information about a mode (D-Bus method)
func (ds *DBusService) GetModeInfo(modeName string) (map[string]dbus.Variant, *dbus.Error) {
	var mode EfficiencyMode

	switch modeName {
	case "Performance":
		mode = ModePerformance
	case "Balanced":
		mode = ModeBalanced
	case "Efficiency":
		mode = ModeEfficiency
	case "Quiet":
		mode = ModeQuiet
	case "Auto":
		mode = ModeAuto
	case "UltraEfficiency":
		mode = ModeUltraEfficiency
	default:
		return nil, dbus.MakeFailedError(fmt.Errorf("unknown mode: %s", modeName))
	}

	config := GetModeConfig(mode)

	info := map[string]dbus.Variant{
		"name":        dbus.MakeVariant(mode.String()),
		"description": dbus.MakeVariant(config.Description),
		"icon":        dbus.MakeVariant(config.Icon),
		"maxPower":    dbus.MakeVariant(config.MaxPowerWatts),
		"maxFan":      dbus.MakeVariant(config.MaxFanPercent),
		"maxTemp":     dbus.MakeVariant(config.MaxTempCelsius),
	}

	return info, nil
}

// makePropertyMap creates property map for D-Bus
func (ds *DBusService) makePropertyMap() map[string]map[string]*prop.Prop {
	return map[string]map[string]*prop.Prop{
		dbusInterface: {
			"CurrentMode": {
				Value:    ds.manager.GetMode().String(),
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					return ds.SetMode(c.Value.(string))
				},
			},
			"EffectiveMode": {
				Value:    ds.manager.GetEffectiveMode().String(),
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
}

// Stop stops the D-Bus service
func (ds *DBusService) Stop() {
	if ds.conn != nil {
		ds.conn.Close()
	}
}
