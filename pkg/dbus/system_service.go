package dbus

import (
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/efficiency"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"
)

const (
	systemInterface = "ie.fio.OllamaProxy.SystemState"
	systemPath      = "/com/anthropic/OllamaProxy/SystemState"
)

// SystemService exposes system state via D-Bus
type SystemService struct {
	conn    *dbus.Conn
	manager *efficiency.EfficiencyManager
	props   *prop.Properties
}

// NewSystemService creates a D-Bus service for system state
func NewSystemService(manager *efficiency.EfficiencyManager) (*SystemService, error) {
	if manager == nil {
		return nil, fmt.Errorf("efficiency manager is nil")
	}

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		// Try session bus if system bus fails
		conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to D-Bus: %w", err)
		}
	}

	svc := &SystemService{
		conn:    conn,
		manager: manager,
	}

	return svc, nil
}

// Start registers the D-Bus service
func (ss *SystemService) Start() error {
	// Request name
	reply, err := ss.conn.RequestName(systemInterface,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request D-Bus name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name already taken")
	}

	// Export methods
	err = ss.conn.Export(ss, systemPath, systemInterface)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus object: %w", err)
	}

	// Export introspection
	intro := introspect.NewIntrospectable(&introspect.Node{
		Name: systemPath,
		Interfaces: []introspect.Interface{
			{
				Name: systemInterface,
				Methods: []introspect.Method{
					{
						Name: "GetSystemState",
						Args: []introspect.Arg{
							{Name: "state", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetAutoModeReasoning",
						Args: []introspect.Arg{
							{Name: "reasoning", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "IsQuietHours",
						Args: []introspect.Arg{
							{Name: "quiet_hours", Type: "b", Direction: "out"},
						},
					},
				},
				Properties: []introspect.Property{
					{
						Name:   "BatteryPercent",
						Type:   "i",
						Access: "read",
					},
					{
						Name:   "OnBattery",
						Type:   "b",
						Access: "read",
					},
					{
						Name:   "QuietHours",
						Type:   "b",
						Access: "read",
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "PowerSourceChanged",
						Args: []introspect.Arg{
							{Name: "on_battery", Type: "b"},
						},
					},
					{
						Name: "BatteryLevelChanged",
						Args: []introspect.Arg{
							{Name: "percent", Type: "i"},
						},
					},
					{
						Name: "QuietHoursChanged",
						Args: []introspect.Arg{
							{Name: "active", Type: "b"},
						},
					},
				},
			},
		},
	})

	err = ss.conn.Export(intro, systemPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Setup properties
	ss.props, _ = prop.Export(ss.conn, systemPath, ss.makePropertyMap())

	logging.Logger.Info("D-Bus System State service started",
		zap.String("interface", systemInterface),
	)
	return nil
}

// GetSystemState returns current system state (D-Bus method)
func (ss *SystemService) GetSystemState() (map[string]dbus.Variant, *dbus.Error) {
	state := ss.manager.GetSystemState()

	result := map[string]dbus.Variant{
		"battery_percent": dbus.MakeVariant(int32(state.BatteryPercent)),
		"on_battery":      dbus.MakeVariant(state.OnBattery),
		"avg_temp":        dbus.MakeVariant(state.AvgTemp),
		"avg_fan_speed":   dbus.MakeVariant(int32(state.AvgFanSpeed)),
		"quiet_hours":     dbus.MakeVariant(state.QuietHours),
	}

	return result, nil
}

// GetAutoModeReasoning returns explanation of why Auto mode chose current effective mode (D-Bus method)
func (ss *SystemService) GetAutoModeReasoning() (string, *dbus.Error) {
	// Only meaningful if current mode is Auto
	currentMode := ss.manager.GetMode()
	if currentMode != efficiency.ModeAuto {
		return "Not in Auto mode", nil
	}

	effectiveMode := ss.manager.GetEffectiveMode()
	state := ss.manager.GetSystemState()

	// Build reasoning string based on system state
	reasoning := fmt.Sprintf("Current effective mode: %s\n", effectiveMode.String())

	if state.BatteryPercent < 20 {
		reasoning += "→ Battery critically low (<20%) → Ultra Efficiency mode"
	} else if state.BatteryPercent < 50 {
		reasoning += "→ Battery below 50% → Efficiency mode"
	} else if state.QuietHours {
		reasoning += "→ During quiet hours → Quiet mode"
	} else if state.AvgTemp > 75 {
		reasoning += fmt.Sprintf("→ Temperature high (%.1f°C) → Efficiency mode to cool down", state.AvgTemp)
	} else if state.AvgFanSpeed > 70 {
		reasoning += fmt.Sprintf("→ Fan speed high (%d%%) → Quiet mode to reduce noise", state.AvgFanSpeed)
	} else if state.OnBattery {
		reasoning += fmt.Sprintf("→ On battery power (%d%%) → Balanced mode", state.BatteryPercent)
	} else {
		reasoning += "→ On AC power, normal conditions → Performance allowed"
	}

	return reasoning, nil
}

// IsQuietHours returns whether currently in quiet hours (D-Bus method)
func (ss *SystemService) IsQuietHours() (bool, *dbus.Error) {
	state := ss.manager.GetSystemState()
	return state.QuietHours, nil
}

// EmitPowerSourceChanged emits power source changed signal (called by application)
func (ss *SystemService) EmitPowerSourceChanged(onBattery bool) {
	if ss.conn != nil {
		ss.conn.Emit(systemPath, systemInterface+".PowerSourceChanged",
			onBattery)
	}

	// Update property
	if ss.props != nil {
		ss.props.SetMust(systemInterface, "OnBattery", onBattery)
	}
}

// EmitBatteryLevelChanged emits battery level changed signal (called by application)
func (ss *SystemService) EmitBatteryLevelChanged(percent int) {
	if ss.conn != nil {
		ss.conn.Emit(systemPath, systemInterface+".BatteryLevelChanged",
			int32(percent))
	}

	// Update property
	if ss.props != nil {
		ss.props.SetMust(systemInterface, "BatteryPercent", int32(percent))
	}
}

// EmitQuietHoursChanged emits quiet hours changed signal (called by application)
func (ss *SystemService) EmitQuietHoursChanged(active bool) {
	if ss.conn != nil {
		ss.conn.Emit(systemPath, systemInterface+".QuietHoursChanged",
			active)
	}

	// Update property
	if ss.props != nil {
		ss.props.SetMust(systemInterface, "QuietHours", active)
	}
}

// makePropertyMap creates property map for D-Bus
func (ss *SystemService) makePropertyMap() map[string]map[string]*prop.Prop {
	state := ss.manager.GetSystemState()

	return map[string]map[string]*prop.Prop{
		systemInterface: {
			"BatteryPercent": {
				Value:    int32(state.BatteryPercent),
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"OnBattery": {
				Value:    state.OnBattery,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"QuietHours": {
				Value:    state.QuietHours,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
}

// UpdateProperties updates the property values (called periodically by application)
func (ss *SystemService) UpdateProperties() {
	state := ss.manager.GetSystemState()

	if ss.props != nil {
		ss.props.SetMust(systemInterface, "BatteryPercent", int32(state.BatteryPercent))
		ss.props.SetMust(systemInterface, "OnBattery", state.OnBattery)
		ss.props.SetMust(systemInterface, "QuietHours", state.QuietHours)
	}
}

// Stop stops the D-Bus service
func (ss *SystemService) Stop() {
	if ss.conn != nil {
		ss.conn.Close()
	}
}
