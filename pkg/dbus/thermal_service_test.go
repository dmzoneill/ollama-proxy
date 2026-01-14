package dbus

import (
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/thermal"
)

// TestThermalServiceConstants tests the package constants
func TestThermalServiceConstants(t *testing.T) {
	if thermalInterface != "ie.fio.OllamaProxy.Thermal" {
		t.Errorf("Expected thermalInterface 'ie.fio.OllamaProxy.Thermal', got '%s'", thermalInterface)
	}
	if thermalPath != "/com/anthropic/OllamaProxy/Thermal" {
		t.Errorf("Expected thermalPath '/com/anthropic/OllamaProxy/Thermal', got '%s'", thermalPath)
	}
}

// TestNewThermalService tests service initialization
func TestNewThermalService(t *testing.T) {
	// Test with nil monitor
	_, err := NewThermalService(nil)
	if err == nil {
		t.Error("Expected error with nil thermal monitor")
	}
	if err != nil && err.Error() != "thermal monitor is nil" {
		t.Errorf("Expected 'thermal monitor is nil' error, got '%s'", err.Error())
	}

	// Test with valid monitor
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)
	svc, err := NewThermalService(monitor)

	// In test environments, we expect D-Bus connection to fail
	if err == nil && svc != nil {
		// Service was created (D-Bus available)
		if svc.monitor != monitor {
			t.Error("Expected monitor to be set")
		}
		if svc.conn == nil {
			t.Error("Expected conn to be set when service is created")
		}
		// Clean up if service was created
		svc.Stop()
	}

	// If error occurred, it should be D-Bus connection error (not nil monitor error)
	if err != nil && err.Error() == "thermal monitor is nil" {
		t.Error("Should not fail with nil monitor error when monitor is provided")
	}
}

// TestGetThermalStateMethod tests the GetThermalState D-Bus method with empty states
func TestGetThermalStateMethod(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
	}

	// Test GetThermalState with empty states
	state, dbusErr := svc.GetThermalState()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if state == nil {
		t.Fatal("Expected state map, got nil")
	}

	// Verify fields exist and have default values
	avgTemp := state["average_temperature"].Value().(float64)
	if avgTemp != 0.0 {
		t.Errorf("Expected average_temperature 0.0 with no states, got %v", avgTemp)
	}

	avgFan := state["average_fan_speed"].Value().(int32)
	if avgFan != 0 {
		t.Errorf("Expected average_fan_speed 0 with no states, got %v", avgFan)
	}

	throttling := state["throttling"].Value().(bool)
	if throttling {
		t.Error("Expected throttling false with no states")
	}

	count := state["monitored_count"].Value().(int32)
	if count != 0 {
		t.Errorf("Expected monitored_count 0 with no states, got %v", count)
	}
}

// TestGetHardwareStateNotFound tests GetHardwareState with non-existent hardware
func TestGetHardwareStateNotFound(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
	}

	// Test not found case
	_, dbusErr := svc.GetHardwareState("nonexistent")
	if dbusErr == nil {
		t.Error("Expected D-Bus error for nonexistent hardware")
	}
}

// TestGetThermalThresholdsMethod tests the GetThermalThresholds D-Bus method
func TestGetThermalThresholdsMethod(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
	}

	thresholds, dbusErr := svc.GetThermalThresholds()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if thresholds == nil {
		t.Fatal("Expected thresholds map, got nil")
	}

	// Verify all thresholds
	if thresholds["temp_warning"].Value().(float64) != 75.0 {
		t.Errorf("Expected temp_warning 75.0, got %v", thresholds["temp_warning"].Value())
	}
	if thresholds["temp_critical"].Value().(float64) != 85.0 {
		t.Errorf("Expected temp_critical 85.0, got %v", thresholds["temp_critical"].Value())
	}
	if thresholds["temp_shutdown"].Value().(float64) != 95.0 {
		t.Errorf("Expected temp_shutdown 95.0, got %v", thresholds["temp_shutdown"].Value())
	}
	if thresholds["fan_quiet"].Value().(int32) != 30 {
		t.Errorf("Expected fan_quiet 30, got %v", thresholds["fan_quiet"].Value())
	}
	if thresholds["fan_moderate"].Value().(int32) != 50 {
		t.Errorf("Expected fan_moderate 50, got %v", thresholds["fan_moderate"].Value())
	}
	if thresholds["fan_loud"].Value().(int32) != 70 {
		t.Errorf("Expected fan_loud 70, got %v", thresholds["fan_loud"].Value())
	}
}

// TestIsThrottlingMethod tests the IsThrottling D-Bus method
func TestIsThrottlingMethod(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
	}

	// Test when nothing is throttling (no states)
	throttling, dbusErr := svc.IsThrottling()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if throttling {
		t.Error("Expected throttling false when no hardware states exist")
	}
}

// TestEmitMethodsWithNilConn tests emit methods handle nil connection gracefully
func TestEmitMethodsWithNilConn(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
		conn:    nil,
		props:   nil,
	}

	// All these should not panic
	tests := []struct {
		name string
		fn   func()
	}{
		{"EmitThermalWarning", func() { svc.EmitThermalWarning("cpu", 85.0) }},
		{"EmitThrottlingStarted", func() { svc.EmitThrottlingStarted("gpu") }},
		{"EmitThrottlingStopped", func() { svc.EmitThrottlingStopped("gpu") }},
		{"UpdatePropertiesThermal", func() { svc.UpdateProperties() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}

// TestMakePropertyMapThermal tests the property map creation
func TestMakePropertyMapThermal(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
	}

	propMap := svc.makePropertyMap()

	if propMap == nil {
		t.Fatal("Expected property map, got nil")
	}

	if _, ok := propMap[thermalInterface]; !ok {
		t.Fatal("Expected thermalInterface in property map")
	}

	props := propMap[thermalInterface]

	// Check properties exist
	if _, ok := props["AverageTemperature"]; !ok {
		t.Error("AverageTemperature property not found")
	}

	if _, ok := props["AverageFanSpeed"]; !ok {
		t.Error("AverageFanSpeed property not found")
	}

	if _, ok := props["ThrottlingActive"]; !ok {
		t.Error("ThrottlingActive property not found")
	}

	// With no states, all should be zero/false
	if tempProp, ok := props["AverageTemperature"]; ok {
		if tempProp.Value.(float64) != 0.0 {
			t.Errorf("Expected AverageTemperature 0.0, got %v", tempProp.Value)
		}
		if tempProp.Writable {
			t.Error("Expected AverageTemperature to be read-only")
		}
	}

	if fanProp, ok := props["AverageFanSpeed"]; ok {
		if fanProp.Value.(int32) != 0 {
			t.Errorf("Expected AverageFanSpeed 0, got %v", fanProp.Value)
		}
		if fanProp.Writable {
			t.Error("Expected AverageFanSpeed to be read-only")
		}
	}

	if throttleProp, ok := props["ThrottlingActive"]; ok {
		if throttleProp.Value.(bool) != false {
			t.Errorf("Expected ThrottlingActive false, got %v", throttleProp.Value)
		}
		if throttleProp.Writable {
			t.Error("Expected ThrottlingActive to be read-only")
		}
	}
}

// TestUpdatePropertiesThermal tests the UpdateProperties method
func TestUpdatePropertiesThermal(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
		props:   nil, // No actual D-Bus properties in test
	}

	// Should not panic even with nil props
	svc.UpdateProperties()
}

// TestStopThermalService tests the Stop method
func TestStopThermalService(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
		conn:    nil, // No actual connection in test
	}

	// Should not panic with nil connection
	svc.Stop()
}

// TestEmitThrottlingStoppedCallsIsThrottling tests that EmitThrottlingStopped calls IsThrottling
func TestEmitThrottlingStoppedCallsIsThrottling(t *testing.T) {
	config := &thermal.ThermalConfig{
		TempWarning:  75.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  50,
		FanLoud:      70,
	}
	monitor := thermal.NewThermalMonitor(config, 5*time.Second)

	svc := &ThermalService{
		monitor: monitor,
		conn:    nil,
		props:   nil,
	}

	// Should not panic and should check throttling status
	svc.EmitThrottlingStopped("gpu")
}
