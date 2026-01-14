package dbus

import (
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/thermal"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"
)

const (
	thermalInterface = "ie.fio.OllamaProxy.Thermal"
	thermalPath      = "/com/anthropic/OllamaProxy/Thermal"
)

// ThermalService exposes thermal monitoring via D-Bus
type ThermalService struct {
	conn    *dbus.Conn
	monitor *thermal.ThermalMonitor
	props   *prop.Properties
}

// NewThermalService creates a D-Bus service for thermal monitoring
func NewThermalService(monitor *thermal.ThermalMonitor) (*ThermalService, error) {
	if monitor == nil {
		return nil, fmt.Errorf("thermal monitor is nil")
	}

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		// Try session bus if system bus fails
		conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to D-Bus: %w", err)
		}
	}

	svc := &ThermalService{
		conn:    conn,
		monitor: monitor,
	}

	return svc, nil
}

// Start registers the D-Bus service
func (ts *ThermalService) Start() error {
	// Request name
	reply, err := ts.conn.RequestName(thermalInterface,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request D-Bus name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name already taken")
	}

	// Export methods
	err = ts.conn.Export(ts, thermalPath, thermalInterface)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus object: %w", err)
	}

	// Export introspection
	intro := introspect.NewIntrospectable(&introspect.Node{
		Name: thermalPath,
		Interfaces: []introspect.Interface{
			{
				Name: thermalInterface,
				Methods: []introspect.Method{
					{
						Name: "GetThermalState",
						Args: []introspect.Arg{
							{Name: "state", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetHardwareState",
						Args: []introspect.Arg{
							{Name: "hardware", Type: "s", Direction: "in"},
							{Name: "state", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetThermalThresholds",
						Args: []introspect.Arg{
							{Name: "thresholds", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "IsThrottling",
						Args: []introspect.Arg{
							{Name: "throttling", Type: "b", Direction: "out"},
						},
					},
				},
				Properties: []introspect.Property{
					{
						Name:   "AverageTemperature",
						Type:   "d",
						Access: "read",
					},
					{
						Name:   "AverageFanSpeed",
						Type:   "i",
						Access: "read",
					},
					{
						Name:   "ThrottlingActive",
						Type:   "b",
						Access: "read",
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "ThermalWarning",
						Args: []introspect.Arg{
							{Name: "hardware", Type: "s"},
							{Name: "temperature", Type: "d"},
						},
					},
					{
						Name: "ThrottlingStarted",
						Args: []introspect.Arg{
							{Name: "hardware", Type: "s"},
						},
					},
					{
						Name: "ThrottlingStopped",
						Args: []introspect.Arg{
							{Name: "hardware", Type: "s"},
						},
					},
				},
			},
		},
	})

	err = ts.conn.Export(intro, thermalPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Setup properties
	ts.props, _ = prop.Export(ts.conn, thermalPath, ts.makePropertyMap())

	logging.Logger.Info("D-Bus Thermal service started",
		zap.String("interface", thermalInterface),
	)
	return nil
}

// GetThermalState returns overall thermal state (D-Bus method)
func (ts *ThermalService) GetThermalState() (map[string]dbus.Variant, *dbus.Error) {
	states := ts.monitor.GetAllStates()

	var totalTemp float64
	var totalFan int
	count := 0
	anyThrottling := false

	for _, state := range states {
		if state != nil {
			totalTemp += state.Temperature
			totalFan += state.FanPercent
			if state.Throttling {
				anyThrottling = true
			}
			count++
		}
	}

	avgTemp := 0.0
	avgFan := 0
	if count > 0 {
		avgTemp = totalTemp / float64(count)
		avgFan = totalFan / count
	}

	result := map[string]dbus.Variant{
		"average_temperature": dbus.MakeVariant(avgTemp),
		"average_fan_speed":   dbus.MakeVariant(int32(avgFan)),
		"throttling":          dbus.MakeVariant(anyThrottling),
		"monitored_count":     dbus.MakeVariant(int32(count)),
	}

	return result, nil
}

// GetHardwareState returns thermal state for specific hardware (D-Bus method)
func (ts *ThermalService) GetHardwareState(hardware string) (map[string]dbus.Variant, *dbus.Error) {
	state := ts.monitor.GetState(hardware)
	if state == nil {
		return nil, dbus.MakeFailedError(fmt.Errorf("hardware not found: %s", hardware))
	}

	result := map[string]dbus.Variant{
		"temperature":  dbus.MakeVariant(state.Temperature),
		"fan_speed":    dbus.MakeVariant(state.FanSpeed),
		"fan_percent":  dbus.MakeVariant(int32(state.FanPercent)),
		"power_draw":   dbus.MakeVariant(state.PowerDraw),
		"utilization":  dbus.MakeVariant(int32(state.Utilization)),
		"throttling":   dbus.MakeVariant(state.Throttling),
		"updated_at":   dbus.MakeVariant(state.UpdatedAt.Unix()),
	}

	return result, nil
}

// GetThermalThresholds returns configured thermal thresholds (D-Bus method)
func (ts *ThermalService) GetThermalThresholds() (map[string]dbus.Variant, *dbus.Error) {
	config := ts.monitor.GetConfig()

	result := map[string]dbus.Variant{
		"temp_warning":  dbus.MakeVariant(config.TempWarning),
		"temp_critical": dbus.MakeVariant(config.TempCritical),
		"temp_shutdown": dbus.MakeVariant(config.TempShutdown),
		"fan_quiet":     dbus.MakeVariant(int32(config.FanQuiet)),
		"fan_moderate":  dbus.MakeVariant(int32(config.FanModerate)),
		"fan_loud":      dbus.MakeVariant(int32(config.FanLoud)),
	}

	return result, nil
}

// IsThrottling returns whether any hardware is currently throttling (D-Bus method)
func (ts *ThermalService) IsThrottling() (bool, *dbus.Error) {
	states := ts.monitor.GetAllStates()

	for _, state := range states {
		if state != nil && state.Throttling {
			return true, nil
		}
	}

	return false, nil
}

// EmitThermalWarning emits a thermal warning signal (called by application)
func (ts *ThermalService) EmitThermalWarning(hardware string, temperature float64) {
	if ts.conn != nil {
		ts.conn.Emit(thermalPath, thermalInterface+".ThermalWarning",
			hardware, temperature)
	}
}

// EmitThrottlingStarted emits throttling started signal (called by application)
func (ts *ThermalService) EmitThrottlingStarted(hardware string) {
	if ts.conn != nil {
		ts.conn.Emit(thermalPath, thermalInterface+".ThrottlingStarted",
			hardware)
	}

	// Update property
	if ts.props != nil {
		ts.props.SetMust(thermalInterface, "ThrottlingActive", true)
	}
}

// EmitThrottlingStopped emits throttling stopped signal (called by application)
func (ts *ThermalService) EmitThrottlingStopped(hardware string) {
	if ts.conn != nil {
		ts.conn.Emit(thermalPath, thermalInterface+".ThrottlingStopped",
			hardware)
	}

	// Check if any hardware is still throttling
	throttling, _ := ts.IsThrottling()
	if ts.props != nil {
		ts.props.SetMust(thermalInterface, "ThrottlingActive", throttling)
	}
}

// makePropertyMap creates property map for D-Bus
func (ts *ThermalService) makePropertyMap() map[string]map[string]*prop.Prop {
	states := ts.monitor.GetAllStates()

	var totalTemp float64
	var totalFan int
	count := 0
	anyThrottling := false

	for _, state := range states {
		if state != nil {
			totalTemp += state.Temperature
			totalFan += state.FanPercent
			if state.Throttling {
				anyThrottling = true
			}
			count++
		}
	}

	avgTemp := 0.0
	avgFan := 0
	if count > 0 {
		avgTemp = totalTemp / float64(count)
		avgFan = totalFan / count
	}

	return map[string]map[string]*prop.Prop{
		thermalInterface: {
			"AverageTemperature": {
				Value:    avgTemp,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"AverageFanSpeed": {
				Value:    int32(avgFan),
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"ThrottlingActive": {
				Value:    anyThrottling,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
}

// UpdateProperties updates the property values (called periodically by application)
func (ts *ThermalService) UpdateProperties() {
	states := ts.monitor.GetAllStates()

	var totalTemp float64
	var totalFan int
	count := 0
	anyThrottling := false

	for _, state := range states {
		if state != nil {
			totalTemp += state.Temperature
			totalFan += state.FanPercent
			if state.Throttling {
				anyThrottling = true
			}
			count++
		}
	}

	avgTemp := 0.0
	avgFan := 0
	if count > 0 {
		avgTemp = totalTemp / float64(count)
		avgFan = totalFan / count
	}

	if ts.props != nil {
		ts.props.SetMust(thermalInterface, "AverageTemperature", avgTemp)
		ts.props.SetMust(thermalInterface, "AverageFanSpeed", int32(avgFan))
		ts.props.SetMust(thermalInterface, "ThrottlingActive", anyThrottling)
	}
}

// Stop stops the D-Bus service
func (ts *ThermalService) Stop() {
	if ts.conn != nil {
		ts.conn.Close()
	}
}
