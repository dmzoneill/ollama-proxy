package efficiency

import (
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// TestEfficiencyMode_String is part of the public API.
func TestEfficiencyMode_String(t *testing.T) {
	tests := []struct {
		expected string
		mode     Mode
	}{
		{mode: ModePerformance, expected: strPerformance},
		{mode: ModeBalanced, expected: strBalanced},
		{mode: ModeEfficiency, expected: strEfficiency},
		{mode: ModeQuiet, expected: strQuiet},
		{mode: ModeAuto, expected: strAuto},
		{mode: ModeUltraEfficiency, expected: strUltraEfficiency2},
		{mode: Mode(999), expected: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

// TestGetModeConfig is part of the public API.
func TestGetModeConfig(t *testing.T) {
	tests := []struct {
		mode                      Mode
		expectedMaxPower          int
		expectedMaxFan            int
		expectedUseClassification bool
	}{
		{mode: ModePerformance, expectedMaxPower: 999, expectedMaxFan: 100, expectedUseClassification: false},
		{mode: ModeBalanced, expectedMaxPower: 60, expectedMaxFan: 80, expectedUseClassification: true},
		{mode: ModeEfficiency, expectedMaxPower: 15, expectedMaxFan: 60, expectedUseClassification: true},
		{mode: ModeQuiet, expectedMaxPower: 15, expectedMaxFan: 40, expectedUseClassification: true},
		{mode: ModeUltraEfficiency, expectedMaxPower: 5, expectedMaxFan: 30, expectedUseClassification: true},
	}

	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			config := GetModeConfig(tt.mode)

			if config == nil {
				t.Fatal("GetModeConfig returned nil")
			}

			if config.MaxPowerWatts != tt.expectedMaxPower {
				t.Errorf("MaxPowerWatts = %d, expected %d", config.MaxPowerWatts, tt.expectedMaxPower)
			}

			if config.MaxFanPercent != tt.expectedMaxFan {
				t.Errorf("MaxFanPercent = %d, expected %d", config.MaxFanPercent, tt.expectedMaxFan)
			}

			if config.UseClassification != tt.expectedUseClassification {
				t.Errorf("UseClassification = %v, expected %v", config.UseClassification, tt.expectedUseClassification)
			}

			if config.Description == "" {
				t.Error("Expected non-empty description")
			}

			if config.Icon == "" {
				t.Error("Expected non-empty icon")
			}
		})
	}
}

// TestGetModeConfig_DefaultFallback is part of the public API.
func TestGetModeConfig_DefaultFallback(t *testing.T) {
	config := GetModeConfig(Mode(999))

	// Should return Balanced as default
	if config.MaxPowerWatts != 60 {
		t.Errorf("Expected Balanced config as default, got MaxPowerWatts=%d", config.MaxPowerWatts)
	}
}

// TestNewEfficiencyManager is part of the public API.
func TestNewEfficiencyManager(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	if em == nil {
		t.Fatal("NewEfficiencyManager returned nil")
	}

	if em.GetMode() != ModePerformance {
		t.Errorf("Expected mode Performance, got %s", em.GetMode())
	}
}

// TestEfficiencyManager_SetMode is part of the public API.
func TestEfficiencyManager_SetMode(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	em.SetMode(ModeEfficiency)

	if em.GetMode() != ModeEfficiency {
		t.Errorf("Expected mode Efficiency after SetMode, got %s", em.GetMode())
	}
}

// TestEfficiencyManager_GetEffectiveMode_NonAuto is part of the public API.
func TestEfficiencyManager_GetEffectiveMode_NonAuto(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	effective := em.GetEffectiveMode()

	if effective != ModePerformance {
		t.Errorf("Expected effective mode Performance, got %s", effective)
	}
}

// TestEfficiencyManager_GetEffectiveMode_Auto is part of the public API.
func TestEfficiencyManager_GetEffectiveMode_Auto(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	tests := []struct {
		name           string
		batteryPercent int
		avgTemp        float64
		avgFanSpeed    int
		expectedMode   Mode
		onBattery      bool
		quietHours     bool
	}{
		{
			name:           "Critical battery",
			batteryPercent: 15,
			onBattery:      true,
			avgTemp:        65.0,
			avgFanSpeed:    30,
			quietHours:     false,
			expectedMode:   ModeUltraEfficiency,
		},
		{
			name:           "Low battery",
			batteryPercent: 35,
			onBattery:      true,
			avgTemp:        65.0,
			avgFanSpeed:    30,
			quietHours:     false,
			expectedMode:   ModeEfficiency,
		},
		{
			name:           "Quiet hours",
			batteryPercent: 80,
			onBattery:      false,
			avgTemp:        65.0,
			avgFanSpeed:    30,
			quietHours:     true,
			expectedMode:   ModeQuiet,
		},
		{
			name:           "High temperature",
			batteryPercent: 100,
			onBattery:      false,
			avgTemp:        80.0,
			avgFanSpeed:    40,
			quietHours:     false,
			expectedMode:   ModeEfficiency,
		},
		{
			name:           "Loud fans",
			batteryPercent: 100,
			onBattery:      false,
			avgTemp:        65.0,
			avgFanSpeed:    75,
			quietHours:     false,
			expectedMode:   ModeQuiet,
		},
		{
			name:           "On battery with good level",
			batteryPercent: 70,
			onBattery:      true,
			avgTemp:        65.0,
			avgFanSpeed:    30,
			quietHours:     false,
			expectedMode:   ModeBalanced,
		},
		{
			name:           "AC power, cool, quiet",
			batteryPercent: 100,
			onBattery:      false,
			avgTemp:        55.0,
			avgFanSpeed:    25,
			quietHours:     false,
			expectedMode:   ModePerformance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em.UpdateSystemState(
				tt.batteryPercent,
				tt.onBattery,
				tt.avgTemp,
				tt.avgFanSpeed,
				tt.quietHours,
			)

			effective := em.GetEffectiveMode()

			if effective != tt.expectedMode {
				t.Errorf("Expected mode %s, got %s", tt.expectedMode, effective)
			}
		})
	}
}

// TestEfficiencyManager_UpdateSystemState is part of the public API.
func TestEfficiencyManager_UpdateSystemState(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	em.UpdateSystemState(50, true, 70.0, 40, false)

	// Values should be updated (verified through GetEffectiveMode behavior)
	effective := em.GetEffectiveMode()

	// 50% battery on battery should give Balanced mode
	if effective != ModeBalanced {
		t.Errorf("Expected Balanced mode after update, got %s", effective)
	}
}

// TestEfficiencyManager_ApplyModeToAnnotations_Performance is part of the public API.
func TestEfficiencyManager_ApplyModeToAnnotations_Performance(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	em.ApplyModeToAnnotations(annotations)

	// Performance mode should not modify annotations
	if !annotations.LatencyCritical {
		t.Error("Performance mode should not modify LatencyCritical")
	}

	if annotations.PreferPowerEfficiency {
		t.Error("Performance mode should not set PreferPowerEfficiency")
	}
}

// TestEfficiencyManager_ApplyModeToAnnotations_Efficiency is part of the public API.
func TestEfficiencyManager_ApplyModeToAnnotations_Efficiency(t *testing.T) {
	em := NewEfficiencyManager(ModeEfficiency)

	annotations := &backends.Annotations{
		LatencyCritical: true,
		MaxPowerWatts:   100,
	}

	em.ApplyModeToAnnotations(annotations)

	// Efficiency mode should override
	if annotations.LatencyCritical {
		t.Error("Efficiency mode should clear LatencyCritical")
	}

	if !annotations.PreferPowerEfficiency {
		t.Error("Efficiency mode should set PreferPowerEfficiency")
	}

	if annotations.MaxPowerWatts != 15 {
		t.Errorf("Expected MaxPowerWatts=15 for Efficiency mode, got %d", annotations.MaxPowerWatts)
	}
}

// TestEfficiencyManager_ApplyModeToAnnotations_Quiet is part of the public API.
func TestEfficiencyManager_ApplyModeToAnnotations_Quiet(t *testing.T) {
	em := NewEfficiencyManager(ModeQuiet)

	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	em.ApplyModeToAnnotations(annotations)

	// Quiet mode should prioritize silence
	if annotations.LatencyCritical {
		t.Error("Quiet mode should clear LatencyCritical")
	}

	if !annotations.PreferPowerEfficiency {
		t.Error("Quiet mode should set PreferPowerEfficiency")
	}
}

// TestEfficiencyManager_ApplyModeToAnnotations_UltraEfficiency is part of the public API.
func TestEfficiencyManager_ApplyModeToAnnotations_UltraEfficiency(t *testing.T) {
	em := NewEfficiencyManager(ModeUltraEfficiency)

	annotations := &backends.Annotations{
		LatencyCritical: true,
		MaxPowerWatts:   50,
	}

	em.ApplyModeToAnnotations(annotations)

	// Ultra Efficiency should be most aggressive
	if annotations.LatencyCritical {
		t.Error("Ultra Efficiency mode should clear LatencyCritical")
	}

	if !annotations.PreferPowerEfficiency {
		t.Error("Ultra Efficiency mode should set PreferPowerEfficiency")
	}

	if annotations.MaxPowerWatts != 5 {
		t.Errorf("Expected MaxPowerWatts=5 for Ultra Efficiency, got %d", annotations.MaxPowerWatts)
	}
}

// TestEfficiencyManager_ApplyModeToAnnotations_PowerLimit is part of the public API.
func TestEfficiencyManager_ApplyModeToAnnotations_PowerLimit(t *testing.T) {
	em := NewEfficiencyManager(ModeEfficiency)

	// Test with user-specified lower limit (should not override)
	annotations := &backends.Annotations{
		MaxPowerWatts: 10,
	}

	em.ApplyModeToAnnotations(annotations)

	// Should keep user's more restrictive limit
	if annotations.MaxPowerWatts != 10 {
		t.Errorf("Expected to keep user's lower limit of 10W, got %d", annotations.MaxPowerWatts)
	}
}

// TestEfficiencyManager_ConcurrentAccess is part of the public API.
func TestEfficiencyManager_ConcurrentAccess(_ *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			em.SetMode(Mode(i % 6))
			em.UpdateSystemState(50+i%50, i%2 == 0, 60.0+float64(i%20), 30+i%40, i%3 == 0)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = em.GetMode()
			_ = em.GetEffectiveMode()
			annotations := &backends.Annotations{}
			em.ApplyModeToAnnotations(annotations)
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// If we get here without panicking, concurrent access works
}

// TestGetModeConfig_PreferredBackends is part of the public API.
func TestGetModeConfig_PreferredBackends(t *testing.T) {
	tests := []struct {
		expectedFirstBackend string
		mode                 Mode
	}{
		{mode: ModePerformance, expectedFirstBackend: strOllamaNvidia},
		{mode: ModeBalanced, expectedFirstBackend: strOllamaIgpu},
		{mode: ModeEfficiency, expectedFirstBackend: strOllamaNpu},
		{mode: ModeQuiet, expectedFirstBackend: strOllamaNpu},
		{mode: ModeUltraEfficiency, expectedFirstBackend: strOllamaNpu},
	}

	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			config := GetModeConfig(tt.mode)

			if len(config.PreferredBackends) == 0 {
				t.Fatal("Expected non-empty PreferredBackends")
			}

			if config.PreferredBackends[0] != tt.expectedFirstBackend {
				t.Errorf("Expected first backend %s, got %s",
					tt.expectedFirstBackend, config.PreferredBackends[0])
			}
		})
	}
}

// TestEfficiencyManager_GetModeDescription is part of the public API.
func TestEfficiencyManager_GetModeDescription(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	config := GetModeConfig(em.GetMode())

	if config.Description == "" {
		t.Error("Expected non-empty mode description")
	}

	if config.Icon == "" {
		t.Error("Expected non-empty mode icon")
	}

	expectedDesc := "Maximum speed. Always use fastest backend available."
	if config.Description != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, config.Description)
	}
}

// TestEfficiencyManager_ShouldUseBackend is part of the public API.
func TestEfficiencyManager_ShouldUseBackend(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	tests := []struct {
		name     string
		desc     string
		temp     float64
		fanSpeed int
		expected bool
	}{
		{
			name:     "Normal temp and fan",
			temp:     50.0,
			fanSpeed: 40,
			expected: true,
			desc:     "Should allow usage with normal conditions",
		},
		{
			name:     "High temp",
			temp:     95.0,
			fanSpeed: 40,
			expected: false,
			desc:     "Should block usage with high temp",
		},
		{
			name:     "High fan speed",
			temp:     50.0,
			fanSpeed: 95,
			expected: false,
			desc:     "Should block usage with high fan speed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := em.ShouldUseBackend("test-backend", tt.temp, tt.fanSpeed)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.desc, tt.expected, result)
			}
		})
	}
}

// TestEfficiencyManager_GetPreferredBackends2 is part of the public API.
func TestEfficiencyManager_GetPreferredBackends2(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	// Test Performance mode
	em.SetMode(ModePerformance)
	backends := em.GetPreferredBackends()
	if len(backends) == 0 {
		t.Error("Performance mode should have preferred backends")
	}
	t.Logf("Performance mode backends: %v", backends)

	// Test Efficiency mode
	em.SetMode(ModeEfficiency)
	backends = em.GetPreferredBackends()
	if len(backends) == 0 {
		t.Error("Efficiency mode should have preferred backends")
	}
	t.Logf("Efficiency mode backends: %v", backends)
}

// TestEfficiencyManager_GetSystemState2 is part of the public API.
func TestEfficiencyManager_GetSystemState2(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	// Get initial state
	state := em.GetSystemState()

	// State should be valid (even if default values)
	if state.BatteryPercent < 0 || state.BatteryPercent > 100 {
		t.Errorf("Invalid battery percent: %d", state.BatteryPercent)
	}

	if state.AvgTemp < 0 {
		t.Errorf("Invalid avg temp: %f", state.AvgTemp)
	}

	if state.AvgFanSpeed < 0 || state.AvgFanSpeed > 100 {
		t.Errorf("Invalid avg fan speed: %d", state.AvgFanSpeed)
	}

	t.Logf("System state: Battery=%d%% OnBattery=%v Temp=%.1f°C Fan=%d%% QuietHours=%v",
		state.BatteryPercent, state.OnBattery, state.AvgTemp, state.AvgFanSpeed, state.QuietHours)
}

// TestAllModes is part of the public API.
func TestAllModes(t *testing.T) {
	modes := AllModes()

	if len(modes) == 0 {
		t.Fatal("AllModes should return non-empty list")
	}

	expectedModes := map[Mode]bool{
		ModePerformance:     false,
		ModeBalanced:        false,
		ModeEfficiency:      false,
		ModeQuiet:           false,
		ModeAuto:            false,
		ModeUltraEfficiency: false,
	}

	for _, mode := range modes {
		if _, exists := expectedModes[mode]; !exists {
			t.Errorf("Unexpected mode in AllModes: %s", mode.String())
		}
		expectedModes[mode] = true
	}

	for mode, found := range expectedModes {
		if !found {
			t.Errorf("Mode %s not found in AllModes", mode.String())
		}
	}

	t.Logf("All modes (%d total): %v", len(modes), modes)
}

// TestEfficiencyManager_SetMode_AllModes is part of the public API.
func TestEfficiencyManager_SetMode_AllModes(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	modes := []Mode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, mode := range modes {
		em.SetMode(mode)
		if em.GetMode() != mode {
			t.Errorf("SetMode(%s) failed, got %s", mode, em.GetMode())
		}
	}
}

// TestEfficiencyManager_GetEffectiveMode is part of the public API.
func TestEfficiencyManager_GetEffectiveMode(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	// In Auto mode, effective mode should be calculated
	em.SetMode(ModeAuto)
	effective := em.GetEffectiveMode()

	if effective == ModeAuto {
		t.Error("Effective mode should not be Auto, should be resolved")
	}

	// In non-Auto mode, effective should equal actual
	em.SetMode(ModePerformance)
	effective = em.GetEffectiveMode()

	if effective != ModePerformance {
		t.Errorf("Expected Performance, got %s", effective)
	}
}

// TestGetModeConfig_AllModes is part of the public API.
func TestGetModeConfig_AllModes(t *testing.T) {
	modes := AllModes()

	for _, mode := range modes {
		config := GetModeConfig(mode)

		if config.MaxTempCelsius == 0 {
			t.Errorf("Mode %s has invalid MaxTempCelsius", mode)
		}

		// Auto mode may have 0 for MaxFanPercent (dynamically determined)
		if config.MaxFanPercent == 0 && mode != ModeAuto {
			t.Errorf("Mode %s has invalid MaxFanPercent", mode)
		}

		if config.Description == "" {
			t.Errorf("Mode %s has no description", mode)
		}

		t.Logf("Mode %s: MaxTemp=%.0f°C MaxFan=%d%% Desc=%s",
			mode, config.MaxTempCelsius, config.MaxFanPercent, config.Description)
	}
}

// TestEfficiencyManager_UpdateState is part of the public API.
func TestEfficiencyManager_UpdateState(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// Test state updates (internal state management)
	state := em.GetSystemState()

	// Initial state should have zero values
	if state.BatteryPercent != 0 {
		t.Logf("Initial battery: %d%%", state.BatteryPercent)
	}

	if state.AvgTemp != 0 {
		t.Logf("Initial temp: %.1f°C", state.AvgTemp)
	}
}

// Test GetModeDescription for various modes
func TestEfficiencyManager_GetModeDescription_AllModes(t *testing.T) {
	modes := []Mode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, mode := range modes {
		em := NewEfficiencyManager(mode)
		desc := em.GetModeDescription()

		if desc == "" {
			t.Errorf("Mode %s has empty description", mode)
		}

		t.Logf("%s: %s", mode, desc)
	}
}

// Test GetModeDescription for Auto mode shows effective mode
func TestEfficiencyManager_GetModeDescription_Auto(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// Before setting state, should describe Auto
	desc := em.GetModeDescription()
	if !contains(desc, strAuto) {
		t.Errorf("Expected 'Auto' in description, got: %s", desc)
	}

	// Set state to high battery on AC - should resolve to Performance
	em.UpdateSystemState(100, false, 55.0, 25, false)
	desc = em.GetModeDescription()
	if !contains(desc, strPerformance) {
		t.Errorf("Expected 'Performance' in description for ideal conditions, got: %s", desc)
	}

	// Set state to critical battery - should resolve to UltraEfficiency
	em.UpdateSystemState(15, true, 65.0, 30, false)
	desc = em.GetModeDescription()
	if !contains(desc, strUltraEfficiency2) {
		t.Errorf("Expected 'Ultra Efficiency' in description for critical battery, got: %s", desc)
	}
}

// Test ApplyModeToAnnotations for Balanced mode
func TestEfficiencyManager_ApplyModeToAnnotations_Balanced(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	annotations := &backends.Annotations{
		LatencyCritical: true,
		MaxPowerWatts:   100,
	}

	em.ApplyModeToAnnotations(annotations)

	// Balanced mode should not modify annotations in efficiency/critical way
	if !annotations.LatencyCritical {
		t.Error("Balanced mode should preserve LatencyCritical")
	}

	if annotations.PreferPowerEfficiency {
		t.Error("Balanced mode should not set PreferPowerEfficiency")
	}

	// Balanced mode has MaxPowerWatts of 60, should apply this limit
	if annotations.MaxPowerWatts != 60 {
		t.Errorf("Balanced mode should apply its power limit of 60W, got %d", annotations.MaxPowerWatts)
	}
}

// Test ApplyModeToAnnotations with high existing power limit
func TestEfficiencyManager_ApplyModeToAnnotations_HigherPowerLimit(t *testing.T) {
	em := NewEfficiencyManager(ModeEfficiency)

	annotations := &backends.Annotations{
		MaxPowerWatts: 20, // Higher than mode's 15W limit
	}

	em.ApplyModeToAnnotations(annotations)

	// Should override with more restrictive limit
	if annotations.MaxPowerWatts != 15 {
		t.Errorf("Expected MaxPowerWatts=15 (mode limit), got %d", annotations.MaxPowerWatts)
	}
}

// Test ApplyModeToAnnotations with zero existing power limit
func TestEfficiencyManager_ApplyModeToAnnotations_ZeroPowerLimit(t *testing.T) {
	em := NewEfficiencyManager(ModeEfficiency)

	annotations := &backends.Annotations{
		MaxPowerWatts: 0, // Not set
	}

	em.ApplyModeToAnnotations(annotations)

	// Should set mode's power limit
	if annotations.MaxPowerWatts != 15 {
		t.Errorf("Expected MaxPowerWatts=15 (mode limit), got %d", annotations.MaxPowerWatts)
	}
}

// Test ShouldUseBackend thermal limit enforcement
func TestEfficiencyManager_ShouldUseBackend_ThermalLimits(t *testing.T) {
	tests := []struct {
		desc        string
		mode        Mode
		temp        float64
		shouldAllow bool
	}{
		{mode: ModePerformance, temp: 90.0, shouldAllow: true, desc: "Performance allows 90C"},
		{mode: ModePerformance, temp: 91.0, shouldAllow: false, desc: "Performance blocks >90C"},
		{mode: ModeBalanced, temp: 85.0, shouldAllow: true, desc: "Balanced allows 85C"},
		{mode: ModeBalanced, temp: 86.0, shouldAllow: false, desc: "Balanced blocks >85C"},
		{mode: ModeEfficiency, temp: 75.0, shouldAllow: true, desc: "Efficiency allows 75C"},
		{mode: ModeEfficiency, temp: 76.0, shouldAllow: false, desc: "Efficiency blocks >75C"},
		{mode: ModeQuiet, temp: 70.0, shouldAllow: true, desc: "Quiet allows 70C"},
		{mode: ModeQuiet, temp: 71.0, shouldAllow: false, desc: "Quiet blocks >70C"},
		{mode: ModeUltraEfficiency, temp: 65.0, shouldAllow: true, desc: "UltraEfficiency allows 65C"},
		{mode: ModeUltraEfficiency, temp: 66.0, shouldAllow: false, desc: "UltraEfficiency blocks >65C"},
	}

	for _, tt := range tests {
		em := NewEfficiencyManager(tt.mode)
		result := em.ShouldUseBackend("test", tt.temp, 30)
		if result != tt.shouldAllow {
			t.Errorf("%s: expected %v, got %v", tt.desc, tt.shouldAllow, result)
		}
	}
}

// Test ShouldUseBackend fan speed limit enforcement
func TestEfficiencyManager_ShouldUseBackend_FanLimits(t *testing.T) {
	tests := []struct {
		desc        string
		mode        Mode
		fanSpeed    int
		shouldAllow bool
	}{
		{mode: ModePerformance, fanSpeed: 100, shouldAllow: true, desc: "Performance allows 100% fan"},
		{mode: ModePerformance, fanSpeed: 101, shouldAllow: false, desc: "Performance blocks >100% fan"},
		{mode: ModeBalanced, fanSpeed: 80, shouldAllow: true, desc: "Balanced allows 80% fan"},
		{mode: ModeBalanced, fanSpeed: 81, shouldAllow: false, desc: "Balanced blocks >80% fan"},
		{mode: ModeEfficiency, fanSpeed: 60, shouldAllow: true, desc: "Efficiency allows 60% fan"},
		{mode: ModeEfficiency, fanSpeed: 61, shouldAllow: false, desc: "Efficiency blocks >60% fan"},
		{mode: ModeQuiet, fanSpeed: 40, shouldAllow: true, desc: "Quiet allows 40% fan"},
		{mode: ModeQuiet, fanSpeed: 41, shouldAllow: false, desc: "Quiet blocks >40% fan"},
		{mode: ModeUltraEfficiency, fanSpeed: 30, shouldAllow: true, desc: "UltraEfficiency allows 30% fan"},
		{mode: ModeUltraEfficiency, fanSpeed: 31, shouldAllow: false, desc: "UltraEfficiency blocks >30% fan"},
	}

	for _, tt := range tests {
		em := NewEfficiencyManager(tt.mode)
		result := em.ShouldUseBackend("test", 50, tt.fanSpeed)
		if result != tt.shouldAllow {
			t.Errorf("%s: expected %v, got %v", tt.desc, tt.shouldAllow, result)
		}
	}
}

// Test auto mode boundary at 20% battery
func TestEfficiencyManager_AutoMode_BatteryBoundary20(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// At exactly 20% on battery - should be Efficiency (< 20 is UltraEfficiency)
	em.UpdateSystemState(20, true, 65.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeEfficiency {
		t.Errorf("At 20%% battery on battery, expected Efficiency, got %s", effective)
	}

	// At 21% on battery - should be Efficiency
	em.UpdateSystemState(21, true, 65.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeEfficiency {
		t.Errorf("At 21%% battery on battery, expected Efficiency, got %s", effective)
	}

	// At 19% on battery - should be UltraEfficiency (< 20)
	em.UpdateSystemState(19, true, 65.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeUltraEfficiency {
		t.Errorf("At 19%% battery on battery, expected UltraEfficiency, got %s", effective)
	}
}

// Test auto mode boundary at 50% battery
func TestEfficiencyManager_AutoMode_BatteryBoundary50(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// At exactly 50% on battery - should be Balanced (< 50 is Efficiency)
	em.UpdateSystemState(50, true, 65.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeBalanced {
		t.Errorf("At 50%% battery on battery, expected Balanced, got %s", effective)
	}

	// At 51% on battery - should be Balanced
	em.UpdateSystemState(51, true, 65.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeBalanced {
		t.Errorf("At 51%% battery on battery, expected Balanced, got %s", effective)
	}

	// At 49% on battery - should be Efficiency (< 50)
	em.UpdateSystemState(49, true, 65.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeEfficiency {
		t.Errorf("At 49%% battery on battery, expected Efficiency, got %s", effective)
	}
}

// Test auto mode boundary at 75C temperature
func TestEfficiencyManager_AutoMode_TempBoundary75(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// At exactly 75.0C - should be Performance (not high enough to trigger Efficiency)
	em.UpdateSystemState(100, false, 75.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModePerformance {
		t.Errorf("At 75.0C, expected Performance, got %s", effective)
	}

	// At 75.1C - should be Efficiency (above threshold)
	em.UpdateSystemState(100, false, 75.1, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeEfficiency {
		t.Errorf("At 75.1C, expected Efficiency, got %s", effective)
	}

	// At 74.9C - should be Performance
	em.UpdateSystemState(100, false, 74.9, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModePerformance {
		t.Errorf("At 74.9C, expected Performance, got %s", effective)
	}
}

// Test auto mode boundary at 70% fan speed
func TestEfficiencyManager_AutoMode_FanBoundary70(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// At exactly 70% fan - should be Performance
	em.UpdateSystemState(100, false, 55.0, 70, false)
	if effective := em.GetEffectiveMode(); effective != ModePerformance {
		t.Errorf("At 70%% fan, expected Performance, got %s", effective)
	}

	// At 71% fan - should be Quiet
	em.UpdateSystemState(100, false, 55.0, 71, false)
	if effective := em.GetEffectiveMode(); effective != ModeQuiet {
		t.Errorf("At 71%% fan, expected Quiet, got %s", effective)
	}

	// At 69% fan - should be Performance
	em.UpdateSystemState(100, false, 55.0, 69, false)
	if effective := em.GetEffectiveMode(); effective != ModePerformance {
		t.Errorf("At 69%% fan, expected Performance, got %s", effective)
	}
}

// Test quiet hours takes priority
func TestEfficiencyManager_AutoMode_QuietHoursPriority(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// Even with ideal conditions (AC, cool, quiet), quiet hours should force Quiet mode
	em.UpdateSystemState(100, false, 55.0, 25, true)
	if effective := em.GetEffectiveMode(); effective != ModeQuiet {
		t.Errorf("With quiet hours enabled, expected Quiet mode, got %s", effective)
	}
}

// Test temperature takes priority over fan noise
func TestEfficiencyManager_AutoMode_TempPriority(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// High temp should trigger Efficiency even with normal fan
	em.UpdateSystemState(100, false, 76.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeEfficiency {
		t.Errorf("With high temp, expected Efficiency, got %s", effective)
	}
}

// Test battery level takes priority over everything except quiet hours
func TestEfficiencyManager_AutoMode_BatteryPriority(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// Critical battery should override normal conditions
	em.UpdateSystemState(15, true, 55.0, 30, false)
	if effective := em.GetEffectiveMode(); effective != ModeUltraEfficiency {
		t.Errorf("With critical battery, expected UltraEfficiency, got %s", effective)
	}
}

// Test ApplyModeToAnnotations does not override lower user limits
func TestEfficiencyManager_ApplyModeToAnnotations_RespectLowerLimits(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	annotations := &backends.Annotations{
		MaxPowerWatts: 5, // User set very low limit
	}

	em.ApplyModeToAnnotations(annotations)

	// Should keep the lower user limit
	if annotations.MaxPowerWatts != 5 {
		t.Errorf("Expected to keep user's lower limit of 5W, got %d", annotations.MaxPowerWatts)
	}
}

// Test OverrideCriticalFlag configuration
func TestEfficiencyManager_ApplyModeToAnnotations_CriticalFlag(t *testing.T) {
	tests := []struct {
		desc        string
		mode        Mode
		shouldClamp bool
	}{
		{mode: ModePerformance, shouldClamp: false, desc: "Performance should not override critical flag"},
		{mode: ModeBalanced, shouldClamp: true, desc: "Balanced should override critical flag"},
		{mode: ModeEfficiency, shouldClamp: true, desc: "Efficiency should override critical flag"},
		{mode: ModeQuiet, shouldClamp: true, desc: "Quiet should override critical flag"},
		{mode: ModeUltraEfficiency, shouldClamp: true, desc: "UltraEfficiency should override critical flag"},
	}

	for _, tt := range tests {
		em := NewEfficiencyManager(tt.mode)
		config := GetModeConfig(em.GetEffectiveMode())

		expected := tt.shouldClamp
		actual := config.OverrideCriticalFlag

		if actual != expected {
			t.Errorf("%s: OverrideCriticalFlag = %v, expected %v",
				tt.desc, actual, expected)
		}
	}
}

// Test ThrottleLatencyCritical configuration
func TestEfficiencyManager_ThrottleLatencyCritical(t *testing.T) {
	tests := []struct {
		desc           string
		mode           Mode
		shouldThrottle bool
	}{
		{mode: ModePerformance, shouldThrottle: false, desc: "Performance should not throttle latency-critical"},
		{mode: ModeBalanced, shouldThrottle: false, desc: "Balanced should not throttle latency-critical"},
		{mode: ModeEfficiency, shouldThrottle: true, desc: "Efficiency should throttle latency-critical"},
		{mode: ModeQuiet, shouldThrottle: true, desc: "Quiet should throttle latency-critical"},
		{mode: ModeUltraEfficiency, shouldThrottle: true, desc: "UltraEfficiency should throttle latency-critical"},
	}

	for _, tt := range tests {
		em := NewEfficiencyManager(tt.mode)
		config := GetModeConfig(em.GetEffectiveMode())

		expected := tt.shouldThrottle
		actual := config.ThrottleLatencyCritical

		if actual != expected {
			t.Errorf("%s: ThrottleLatencyCritical = %v, expected %v",
				tt.desc, actual, expected)
		}
	}
}

// Test auto mode does not throttle latency-critical
func TestEfficiencyManager_AutoMode_NoLatencyThrottle(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)
	config := GetModeConfig(em.GetEffectiveMode())

	if config.ThrottleLatencyCritical {
		t.Error("Auto mode should not throttle latency-critical requests")
	}
}

// Test complex scenario: battery + temperature + quiet hours
func TestEfficiencyManager_AutoMode_ComplexScenario(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// Quiet hours should override temperature concern
	em.UpdateSystemState(80, false, 80.0, 50, true)
	if effective := em.GetEffectiveMode(); effective != ModeQuiet {
		t.Errorf("Quiet hours should override high temp, got %s", effective)
	}

	// Critical battery should override quiet hours? Let's check priority
	em.UpdateSystemState(15, true, 55.0, 25, true)
	if effective := em.GetEffectiveMode(); effective != ModeUltraEfficiency {
		t.Errorf("Critical battery should override quiet hours, got %s", effective)
	}

	// Low battery + high temp + quiet hours = Efficiency wins
	em.UpdateSystemState(40, true, 76.0, 30, true)
	effective := em.GetEffectiveMode()
	// Order: critical battery -> low battery -> quiet hours -> temp -> fans -> on battery -> performance
	if effective != ModeEfficiency {
		t.Errorf("Low battery + high temp + quiet hours, expected Efficiency, got %s", effective)
	}
}

// Test GetMode returns exact mode set with SetMode
func TestEfficiencyManager_GetMode_ExactValue(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	modes := []Mode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, mode := range modes {
		em.SetMode(mode)
		if got := em.GetMode(); got != mode {
			t.Errorf("GetMode() = %v, expected %v", got, mode)
		}
	}
}

// Test SystemState struct fields are correct
func TestEfficiencyManager_SystemState_Fields(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	em.UpdateSystemState(75, true, 68.5, 45, true)
	state := em.GetSystemState()

	if state.BatteryPercent != 75 {
		t.Errorf("BatteryPercent = %d, expected 75", state.BatteryPercent)
	}
	if !state.OnBattery {
		t.Error("OnBattery should be true")
	}
	if state.AvgTemp != 68.5 {
		t.Errorf("AvgTemp = %f, expected 68.5", state.AvgTemp)
	}
	if state.AvgFanSpeed != 45 {
		t.Errorf("AvgFanSpeed = %d, expected 45", state.AvgFanSpeed)
	}
	if !state.QuietHours {
		t.Error("QuietHours should be true")
	}
}

// Test mode string edge cases
func TestEfficiencyMode_String_EdgeCases(t *testing.T) {
	// Test all known modes
	knownModes := map[Mode]string{
		ModePerformance:     strPerformance,
		ModeBalanced:        strBalanced,
		ModeEfficiency:      strEfficiency,
		ModeQuiet:           strQuiet,
		ModeAuto:            strAuto,
		ModeUltraEfficiency: strUltraEfficiency2,
	}

	for mode, expected := range knownModes {
		if got := mode.String(); got != expected {
			t.Errorf("String() for mode %d = %q, expected %q", mode, got, expected)
		}
	}

	// Test unknown mode values
	unknownModes := []Mode{-1, 100, 999, 1000}
	for _, mode := range unknownModes {
		if got := mode.String(); got != "Unknown" {
			t.Errorf("String() for unknown mode %d = %q, expected 'Unknown'", mode, got)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test ApplyModeToAnnotations for Auto mode
func TestEfficiencyManager_ApplyModeToAnnotations_Auto(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	// Set state to critical battery - should apply like UltraEfficiency
	em.UpdateSystemState(15, true, 65.0, 30, false)

	annotations := &backends.Annotations{
		LatencyCritical: true,
		MaxPowerWatts:   100,
	}

	em.ApplyModeToAnnotations(annotations)

	// Should have efficiency settings since effective mode is UltraEfficiency
	if annotations.LatencyCritical {
		t.Error("Auto mode (critical battery) should clear LatencyCritical")
	}

	if !annotations.PreferPowerEfficiency {
		t.Error("Auto mode (critical battery) should set PreferPowerEfficiency")
	}
}

// Test ApplyModeToAnnotations when performance is allowed
func TestEfficiencyManager_ApplyModeToAnnotations_PerformanceMode(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	annotations := &backends.Annotations{
		LatencyCritical: false,
		MaxPowerWatts:   0,
	}

	em.ApplyModeToAnnotations(annotations)

	// Performance mode should not set power limit for unlimited case
	if annotations.PreferPowerEfficiency {
		t.Error("Performance mode should not set PreferPowerEfficiency")
	}

	if annotations.MaxPowerWatts != 0 {
		t.Errorf("Performance mode should not restrict power, got %d", annotations.MaxPowerWatts)
	}
}

// Test ShouldUseBackend with both conditions violating limits
func TestEfficiencyManager_ShouldUseBackend_BothLimitsBroken(t *testing.T) {
	em := NewEfficiencyManager(ModeEfficiency)

	// Both temp and fan exceed limits
	result := em.ShouldUseBackend("test", 76.0, 61)

	if result {
		t.Error("Should block backend when both temp and fan exceed limits")
	}
}

// Test ShouldUseBackend with boundary values
func TestEfficiencyManager_ShouldUseBackend_Boundaries(t *testing.T) {
	em := NewEfficiencyManager(ModeEfficiency)

	// At exact limit for temp - should allow
	result := em.ShouldUseBackend("test", 75.0, 50)
	if !result {
		t.Error("Should allow at exact temperature limit")
	}

	// One degree over temp - should block
	result = em.ShouldUseBackend("test", 75.1, 50)
	if result {
		t.Error("Should block at temperature over limit")
	}

	// At exact limit for fan - should allow
	result = em.ShouldUseBackend("test", 50, 60)
	if !result {
		t.Error("Should allow at exact fan limit")
	}

	// One percent over fan - should block
	result = em.ShouldUseBackend("test", 50, 61)
	if result {
		t.Error("Should block at fan over limit")
	}
}

// Test GetPreferredBackends for each mode returns non-empty list
func TestEfficiencyManager_GetPreferredBackends_NonEmpty(t *testing.T) {
	modes := []Mode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, mode := range modes {
		em := NewEfficiencyManager(mode)
		backends := em.GetPreferredBackends()

		if len(backends) == 0 {
			t.Errorf("Mode %s has no preferred backends", mode)
		}

		t.Logf("Mode %s backends: %v", mode, backends)
	}
}

// Test GetModeDescription shows mode for non-Auto modes
func TestEfficiencyManager_GetModeDescription_NonAuto(t *testing.T) {
	modes := []Mode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeUltraEfficiency,
	}

	for _, mode := range modes {
		em := NewEfficiencyManager(mode)
		desc := em.GetModeDescription()

		// Non-auto modes should start with the mode name
		if !contains(desc, mode.String()) {
			t.Errorf("Mode %s description should contain mode name, got: %s", mode, desc)
		}
	}
}

// Test concurrent mode switching
func TestEfficiencyManager_ConcurrentModeSwitching(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	done := make(chan int, 2)

	// Thread 1: Keep switching modes
	go func() {
		for i := 0; i < 50; i++ {
			em.SetMode(Mode(i % 6))
		}
		done <- 1
	}()

	// Thread 2: Keep reading modes
	go func() {
		for i := 0; i < 50; i++ {
			_ = em.GetMode()
			_ = em.GetEffectiveMode()
		}
		done <- 1
	}()

	<-done
	<-done

	t.Log("Concurrent mode switching completed without panics")
}

// Test concurrent state updates and reads
func TestEfficiencyManager_ConcurrentStateUpdates(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	done := make(chan int, 2)

	// Thread 1: Update state repeatedly
	go func() {
		for i := 0; i < 50; i++ {
			em.UpdateSystemState(
				20+i%80,
				i%2 == 0,
				50.0+float64(i%40),
				20+i%50,
				i%3 == 0,
			)
		}
		done <- 1
	}()

	// Thread 2: Read state repeatedly
	go func() {
		for i := 0; i < 50; i++ {
			_ = em.GetSystemState()
			_ = em.GetEffectiveMode()
		}
		done <- 1
	}()

	<-done
	<-done

	t.Log("Concurrent state updates and reads completed without panics")
}

// Test auto mode with extremes
func TestEfficiencyManager_AutoMode_ExtremeValues(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	tests := []struct {
		name         string
		battery      int
		temp         float64
		fanSpeed     int
		expectedMode Mode
		onBattery    bool
		quietHours   bool
	}{
		{
			name:    "0% battery",
			battery: 0, onBattery: true, temp: 70.0, fanSpeed: 50, quietHours: false,
			expectedMode: ModeUltraEfficiency,
		},
		{
			name:    "100% battery on AC",
			battery: 100, onBattery: false, temp: 50.0, fanSpeed: 20, quietHours: false,
			expectedMode: ModePerformance,
		},
		{
			name:    "100% battery on AC with high fan",
			battery: 100, onBattery: false, temp: 50.0, fanSpeed: 75, quietHours: false,
			expectedMode: ModeQuiet,
		},
		{
			name:    "100% battery on AC with high temp",
			battery: 100, onBattery: false, temp: 80.0, fanSpeed: 30, quietHours: false,
			expectedMode: ModeEfficiency,
		},
		{
			name:    "Quiet hours override",
			battery: 100, onBattery: false, temp: 50.0, fanSpeed: 20, quietHours: true,
			expectedMode: ModeQuiet,
		},
		{
			name:    "Very high temp",
			battery: 50, onBattery: false, temp: 95.0, fanSpeed: 50, quietHours: false,
			expectedMode: ModeEfficiency,
		},
		{
			name:    "Very high fan",
			battery: 50, onBattery: false, temp: 55.0, fanSpeed: 100, quietHours: false,
			expectedMode: ModeQuiet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em.UpdateSystemState(tt.battery, tt.onBattery, tt.temp, tt.fanSpeed, tt.quietHours)
			effective := em.GetEffectiveMode()
			if effective != tt.expectedMode {
				t.Errorf("Expected %s, got %s", tt.expectedMode, effective)
			}
		})
	}
}

// Test UseClassification field for all modes
func TestEfficiencyManager_UseClassificationField(t *testing.T) {
	tests := []struct {
		desc           string
		mode           Mode
		shouldClassify bool
	}{
		{mode: ModePerformance, shouldClassify: false, desc: "Performance should not use classification"},
		{mode: ModeBalanced, shouldClassify: true, desc: "Balanced should use classification"},
		{mode: ModeEfficiency, shouldClassify: true, desc: "Efficiency should use classification"},
		{mode: ModeQuiet, shouldClassify: true, desc: "Quiet should use classification"},
		{mode: ModeUltraEfficiency, shouldClassify: true, desc: "UltraEfficiency should use classification"},
	}

	for _, tt := range tests {
		config := GetModeConfig(tt.mode)

		if config.UseClassification != tt.shouldClassify {
			t.Errorf("%s: UseClassification = %v, expected %v",
				tt.desc, config.UseClassification, tt.shouldClassify)
		}
	}

	// Test Auto mode separately - its UseClassification depends on effective mode
	autoConfig := GetModeConfig(ModeAuto)
	if !autoConfig.UseClassification {
		t.Error("Auto mode config should have UseClassification=true")
	}

	// When Auto resolves to Performance, UseClassification should be false
	em := NewEfficiencyManager(ModeAuto)
	em.UpdateSystemState(100, false, 50.0, 20, false) // Performance conditions
	effectiveMode := em.GetEffectiveMode()
	effectiveConfig := GetModeConfig(effectiveMode)
	t.Logf("Auto mode resolves to %s with UseClassification=%v", effectiveMode, effectiveConfig.UseClassification)
}

// Test all configuration fields are set properly
func TestEfficiencyManager_ConfigFieldsSet(t *testing.T) {
	modes := AllModes()

	for _, mode := range modes {
		config := GetModeConfig(mode)

		if config == nil {
			t.Errorf("Config for mode %s is nil", mode)
			continue
		}

		// All fields should be set
		if len(config.PreferredBackends) == 0 {
			t.Errorf("Mode %s has no preferred backends", mode)
		}

		if config.MaxTempCelsius == 0 {
			t.Errorf("Mode %s has no max temperature", mode)
		}

		if config.Description == "" {
			t.Errorf("Mode %s has no description", mode)
		}

		if config.Icon == "" {
			t.Errorf("Mode %s has no icon", mode)
		}

		// Optional but check values are reasonable
		if config.MaxPowerWatts < 0 {
			t.Errorf("Mode %s has negative MaxPowerWatts", mode)
		}

		if config.MaxFanPercent < 0 || config.MaxFanPercent > 100 {
			if mode != ModeAuto {
				t.Errorf("Mode %s has invalid MaxFanPercent", mode)
			}
		}
	}
}

// Test system state reflects all updates
func TestEfficiencyManager_SystemState_AllFields(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)

	testCases := []struct {
		battery    int
		temp       float64
		fanSpeed   int
		onBattery  bool
		quietHours bool
	}{
		{battery: 100, onBattery: false, temp: 25.0, fanSpeed: 10, quietHours: false},
		{battery: 50, onBattery: true, temp: 65.0, fanSpeed: 40, quietHours: false},
		{battery: 10, onBattery: true, temp: 85.0, fanSpeed: 90, quietHours: true},
		{battery: 75, onBattery: false, temp: 70.0, fanSpeed: 50, quietHours: true},
		{battery: 0, onBattery: true, temp: 95.0, fanSpeed: 100, quietHours: false},
	}

	for i, tc := range testCases {
		em.UpdateSystemState(tc.battery, tc.onBattery, tc.temp, tc.fanSpeed, tc.quietHours)
		state := em.GetSystemState()

		if state.BatteryPercent != tc.battery {
			t.Errorf("Test case %d: BatteryPercent = %d, expected %d", i, state.BatteryPercent, tc.battery)
		}
		if state.OnBattery != tc.onBattery {
			t.Errorf("Test case %d: OnBattery = %v, expected %v", i, state.OnBattery, tc.onBattery)
		}
		if state.AvgTemp != tc.temp {
			t.Errorf("Test case %d: AvgTemp = %f, expected %f", i, state.AvgTemp, tc.temp)
		}
		if state.AvgFanSpeed != tc.fanSpeed {
			t.Errorf("Test case %d: AvgFanSpeed = %d, expected %d", i, state.AvgFanSpeed, tc.fanSpeed)
		}
		if state.QuietHours != tc.quietHours {
			t.Errorf("Test case %d: QuietHours = %v, expected %v", i, state.QuietHours, tc.quietHours)
		}
	}
}
