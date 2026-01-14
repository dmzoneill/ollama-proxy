package dbus

import (
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/efficiency"
)

// TestSystemServiceConstants tests the package constants
func TestSystemServiceConstants(t *testing.T) {
	if systemInterface != "ie.fio.OllamaProxy.SystemState" {
		t.Errorf("Expected systemInterface 'ie.fio.OllamaProxy.SystemState', got '%s'", systemInterface)
	}
	if systemPath != "/com/anthropic/OllamaProxy/SystemState" {
		t.Errorf("Expected systemPath '/com/anthropic/OllamaProxy/SystemState', got '%s'", systemPath)
	}
}

// TestNewSystemService tests service initialization
func TestNewSystemService(t *testing.T) {
	// Test with nil manager
	_, err := NewSystemService(nil)
	if err == nil {
		t.Error("Expected error with nil efficiency manager")
	}
	if err != nil && err.Error() != "efficiency manager is nil" {
		t.Errorf("Expected 'efficiency manager is nil' error, got '%s'", err.Error())
	}

	// Test with valid manager
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)
	svc, err := NewSystemService(manager)

	// In test environments, we expect D-Bus connection to fail
	if err == nil && svc != nil {
		// Service was created (D-Bus available)
		if svc.manager != manager {
			t.Error("Expected manager to be set")
		}
		if svc.conn == nil {
			t.Error("Expected conn to be set when service is created")
		}
		// Clean up if service was created
		svc.Stop()
	}

	// If error occurred, it should be D-Bus connection error (not nil manager error)
	if err != nil && err.Error() == "efficiency manager is nil" {
		t.Error("Should not fail with nil manager error when manager is provided")
	}
}

// TestGetSystemStateMethod tests the GetSystemState D-Bus method
func TestGetSystemStateMethod(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	// Set system state
	manager.UpdateSystemState(75, true, 65.5, 50, false)

	svc := &SystemService{
		manager: manager,
	}

	// Test GetSystemState
	state, dbusErr := svc.GetSystemState()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if state == nil {
		t.Fatal("Expected state map, got nil")
	}

	// Verify all fields
	if state["battery_percent"].Value().(int32) != 75 {
		t.Errorf("Expected battery_percent 75, got %v", state["battery_percent"].Value())
	}
	if state["on_battery"].Value().(bool) != true {
		t.Errorf("Expected on_battery true, got %v", state["on_battery"].Value())
	}
	if state["avg_temp"].Value().(float64) != 65.5 {
		t.Errorf("Expected avg_temp 65.5, got %v", state["avg_temp"].Value())
	}
	if state["avg_fan_speed"].Value().(int32) != 50 {
		t.Errorf("Expected avg_fan_speed 50, got %v", state["avg_fan_speed"].Value())
	}
	if state["quiet_hours"].Value().(bool) != false {
		t.Errorf("Expected quiet_hours false, got %v", state["quiet_hours"].Value())
	}
}

// TestGetSystemStateZeroValues tests GetSystemState with zero values
func TestGetSystemStateZeroValues(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
	}

	state, dbusErr := svc.GetSystemState()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	// Default values should be zero/false
	if state["battery_percent"].Value().(int32) != 0 {
		t.Errorf("Expected battery_percent 0, got %v", state["battery_percent"].Value())
	}
	if state["on_battery"].Value().(bool) != false {
		t.Errorf("Expected on_battery false, got %v", state["on_battery"].Value())
	}
	if state["avg_temp"].Value().(float64) != 0 {
		t.Errorf("Expected avg_temp 0, got %v", state["avg_temp"].Value())
	}
}

// TestGetAutoModeReasoningMethod tests the GetAutoModeReasoning D-Bus method
func TestGetAutoModeReasoningMethod(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeAuto)

	svc := &SystemService{
		manager: manager,
	}

	// Test with default state (AC power, normal conditions)
	manager.UpdateSystemState(100, false, 50.0, 30, false)

	reasoning, dbusErr := svc.GetAutoModeReasoning()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if reasoning == "" {
		t.Error("Expected non-empty reasoning")
	}
	if len(reasoning) < 10 {
		t.Errorf("Expected detailed reasoning, got: %s", reasoning)
	}

	// Test with critical battery
	manager.UpdateSystemState(15, true, 50.0, 30, false)

	reasoning, dbusErr = svc.GetAutoModeReasoning()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	// Should mention battery critically low
	// (We can't check exact text as it might change, but should be substantial)
	if len(reasoning) < 20 {
		t.Errorf("Expected detailed reasoning for critical battery, got: %s", reasoning)
	}

	// Test with high temperature
	manager.UpdateSystemState(100, false, 80.0, 30, false)

	reasoning, dbusErr = svc.GetAutoModeReasoning()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if len(reasoning) < 20 {
		t.Errorf("Expected detailed reasoning for high temp, got: %s", reasoning)
	}

	// Test with quiet hours
	manager.UpdateSystemState(100, false, 50.0, 30, true)

	reasoning, dbusErr = svc.GetAutoModeReasoning()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if len(reasoning) < 20 {
		t.Errorf("Expected detailed reasoning for quiet hours, got: %s", reasoning)
	}
}

// TestGetAutoModeReasoningNotAutoMode tests GetAutoModeReasoning when not in Auto mode
func TestGetAutoModeReasoningNotAutoMode(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModePerformance)

	svc := &SystemService{
		manager: manager,
	}

	reasoning, dbusErr := svc.GetAutoModeReasoning()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	// Should return message indicating not in Auto mode
	if reasoning != "Not in Auto mode" {
		t.Errorf("Expected 'Not in Auto mode', got '%s'", reasoning)
	}
}

// TestIsQuietHoursMethod tests the IsQuietHours D-Bus method
func TestIsQuietHoursMethod(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
	}

	// Test when quiet hours is false
	manager.UpdateSystemState(0, false, 0.0, 0, false)

	quietHours, dbusErr := svc.IsQuietHours()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if quietHours {
		t.Error("Expected quiet hours to be false")
	}

	// Test when quiet hours is true
	manager.UpdateSystemState(0, false, 0.0, 0, true)

	quietHours, dbusErr = svc.IsQuietHours()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if !quietHours {
		t.Error("Expected quiet hours to be true")
	}
}

// TestEmitPowerSourceChanged tests the EmitPowerSourceChanged method
func TestEmitPowerSourceChanged(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
		conn:    nil, // No actual connection in test
		props:   nil, // No actual properties in test
	}

	// Should not panic even without D-Bus connection
	svc.EmitPowerSourceChanged(true)
	svc.EmitPowerSourceChanged(false)
}

// TestEmitBatteryLevelChanged tests the EmitBatteryLevelChanged method
func TestEmitBatteryLevelChanged(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
		conn:    nil,
		props:   nil,
	}

	// Should not panic even without D-Bus connection
	svc.EmitBatteryLevelChanged(50)
	svc.EmitBatteryLevelChanged(100)
	svc.EmitBatteryLevelChanged(0)
}

// TestEmitQuietHoursChanged tests the EmitQuietHoursChanged method
func TestEmitQuietHoursChanged(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
		conn:    nil,
		props:   nil,
	}

	// Should not panic even without D-Bus connection
	svc.EmitQuietHoursChanged(true)
	svc.EmitQuietHoursChanged(false)
}

// TestMakePropertyMapSystem tests the property map creation for system service
func TestMakePropertyMapSystem(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	// Set system state
	manager.UpdateSystemState(85, true, 60.0, 45, true)

	svc := &SystemService{
		manager: manager,
	}

	propMap := svc.makePropertyMap()

	if propMap == nil {
		t.Fatal("Expected property map, got nil")
	}

	if _, ok := propMap[systemInterface]; !ok {
		t.Fatal("Expected systemInterface in property map")
	}

	props := propMap[systemInterface]

	// Check BatteryPercent
	if batteryProp, ok := props["BatteryPercent"]; ok {
		if batteryProp.Value.(int32) != 85 {
			t.Errorf("Expected BatteryPercent 85, got %v", batteryProp.Value)
		}
		if batteryProp.Writable {
			t.Error("Expected BatteryPercent to be read-only")
		}
	} else {
		t.Error("BatteryPercent property not found")
	}

	// Check OnBattery
	if onBatteryProp, ok := props["OnBattery"]; ok {
		if onBatteryProp.Value.(bool) != true {
			t.Errorf("Expected OnBattery true, got %v", onBatteryProp.Value)
		}
		if onBatteryProp.Writable {
			t.Error("Expected OnBattery to be read-only")
		}
	} else {
		t.Error("OnBattery property not found")
	}

	// Check QuietHours
	if quietHoursProp, ok := props["QuietHours"]; ok {
		if quietHoursProp.Value.(bool) != true {
			t.Errorf("Expected QuietHours true, got %v", quietHoursProp.Value)
		}
		if quietHoursProp.Writable {
			t.Error("Expected QuietHours to be read-only")
		}
	} else {
		t.Error("QuietHours property not found")
	}
}

// TestUpdatePropertiesSystem tests the UpdateProperties method
func TestUpdatePropertiesSystem(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
		props:   nil, // No actual D-Bus properties in test
	}

	// Set initial state
	manager.UpdateSystemState(90, false, 0.0, 0, false)

	// Should not panic even with nil props
	svc.UpdateProperties()

	// Change state
	manager.UpdateSystemState(50, true, 0.0, 0, true)

	// Update again - should not panic
	svc.UpdateProperties()
}

// TestStopSystemService tests the Stop method
func TestStopSystemService(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
		conn:    nil, // No actual connection in test
	}

	// Should not panic with nil connection
	svc.Stop()
}

// TestGetAutoModeReasoningDifferentStates tests GetAutoModeReasoning with various states
func TestGetAutoModeReasoningDifferentStates(t *testing.T) {
	testCases := []struct {
		name  string
		state efficiency.SystemState
	}{
		{
			name: "Low battery",
			state: efficiency.SystemState{
				BatteryPercent: 35,
				OnBattery:      true,
			},
		},
		{
			name: "High fan speed",
			state: efficiency.SystemState{
				BatteryPercent: 100,
				OnBattery:      false,
				AvgFanSpeed:    85,
			},
		},
		{
			name: "On battery with good level",
			state: efficiency.SystemState{
				BatteryPercent: 70,
				OnBattery:      true,
				AvgTemp:        55.0,
				AvgFanSpeed:    40,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := efficiency.NewEfficiencyManager(efficiency.ModeAuto)
			manager.UpdateSystemState(
				tc.state.BatteryPercent,
				tc.state.OnBattery,
				tc.state.AvgTemp,
				tc.state.AvgFanSpeed,
				tc.state.QuietHours,
			)

			svc := &SystemService{
				manager: manager,
			}

			reasoning, dbusErr := svc.GetAutoModeReasoning()
			if dbusErr != nil {
				t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
			}
			if reasoning == "" {
				t.Error("Expected non-empty reasoning")
			}
			// Reasoning should be substantial
			if len(reasoning) < 15 {
				t.Errorf("Expected detailed reasoning, got: %s", reasoning)
			}
		})
	}
}

// TestSystemStateDifferentModes tests GetSystemState with different efficiency modes
func TestSystemStateDifferentModes(t *testing.T) {
	modes := []efficiency.EfficiencyMode{
		efficiency.ModePerformance,
		efficiency.ModeBalanced,
		efficiency.ModeEfficiency,
		efficiency.ModeQuiet,
		efficiency.ModeAuto,
		efficiency.ModeUltraEfficiency,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			manager := efficiency.NewEfficiencyManager(mode)
			manager.UpdateSystemState(60, true, 55.0, 40, false)

			svc := &SystemService{
				manager: manager,
			}

			state, dbusErr := svc.GetSystemState()
			if dbusErr != nil {
				t.Fatalf("Expected no D-Bus error for mode %s, got %v", mode.String(), dbusErr)
			}

			// State should always be returned regardless of mode
			if state == nil {
				t.Fatalf("Expected state map for mode %s, got nil", mode.String())
			}

			// Verify battery percent is correct
			if state["battery_percent"].Value().(int32) != 60 {
				t.Errorf("Mode %s: Expected battery_percent 60, got %v", mode.String(), state["battery_percent"].Value())
			}
		})
	}
}

// TestEmitMethodsWithNilProps tests emit methods handle nil props gracefully
func TestEmitMethodsWithNilProps(t *testing.T) {
	manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)

	svc := &SystemService{
		manager: manager,
		conn:    nil,
		props:   nil,
	}

	// All these should not panic
	tests := []struct {
		name string
		fn   func()
	}{
		{"EmitPowerSourceChanged true", func() { svc.EmitPowerSourceChanged(true) }},
		{"EmitPowerSourceChanged false", func() { svc.EmitPowerSourceChanged(false) }},
		{"EmitBatteryLevelChanged 0", func() { svc.EmitBatteryLevelChanged(0) }},
		{"EmitBatteryLevelChanged 50", func() { svc.EmitBatteryLevelChanged(50) }},
		{"EmitBatteryLevelChanged 100", func() { svc.EmitBatteryLevelChanged(100) }},
		{"EmitQuietHoursChanged true", func() { svc.EmitQuietHoursChanged(true) }},
		{"EmitQuietHoursChanged false", func() { svc.EmitQuietHoursChanged(false) }},
		{"UpdateProperties", func() { svc.UpdateProperties() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}

// TestSystemStateEdgeCases tests GetSystemState with edge case values
func TestSystemStateEdgeCases(t *testing.T) {
	testCases := []struct {
		name  string
		state efficiency.SystemState
	}{
		{
			name: "Zero battery",
			state: efficiency.SystemState{
				BatteryPercent: 0,
				OnBattery:      true,
			},
		},
		{
			name: "Full battery",
			state: efficiency.SystemState{
				BatteryPercent: 100,
				OnBattery:      false,
			},
		},
		{
			name: "High temperature",
			state: efficiency.SystemState{
				AvgTemp: 95.0,
			},
		},
		{
			name: "Max fan speed",
			state: efficiency.SystemState{
				AvgFanSpeed: 100,
			},
		},
		{
			name: "Zero fan speed",
			state: efficiency.SystemState{
				AvgFanSpeed: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := efficiency.NewEfficiencyManager(efficiency.ModeBalanced)
			manager.UpdateSystemState(
				tc.state.BatteryPercent,
				tc.state.OnBattery,
				tc.state.AvgTemp,
				tc.state.AvgFanSpeed,
				tc.state.QuietHours,
			)

			svc := &SystemService{
				manager: manager,
			}

			state, dbusErr := svc.GetSystemState()
			if dbusErr != nil {
				t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
			}

			if state == nil {
				t.Fatal("Expected state map, got nil")
			}

			// Verify the state contains all required fields
			requiredFields := []string{"battery_percent", "on_battery", "avg_temp", "avg_fan_speed", "quiet_hours"}
			for _, field := range requiredFields {
				if _, ok := state[field]; !ok {
					t.Errorf("Missing required field: %s", field)
				}
			}
		})
	}
}
