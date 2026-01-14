package thermal

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewThermalMonitor(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	if tm == nil {
		t.Fatal("NewThermalMonitor returned nil")
	}

	if tm.config.TempWarning != 70.0 {
		t.Errorf("Expected TempWarning 70.0, got %.1f", tm.config.TempWarning)
	}

	if tm.updateInterval != 5*time.Second {
		t.Errorf("Expected update interval 5s, got %v", tm.updateInterval)
	}

	if tm.states == nil {
		t.Error("states map not initialized")
	}
}

func TestNewThermalMonitor_DefaultConfig(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	if tm == nil {
		t.Fatal("NewThermalMonitor returned nil")
	}

	// Should use default config
	if tm.config.TempWarning != 70.0 {
		t.Errorf("Expected default TempWarning 70.0, got %.1f", tm.config.TempWarning)
	}

	if tm.config.TempCritical != 85.0 {
		t.Errorf("Expected default TempCritical 85.0, got %.1f", tm.config.TempCritical)
	}
}

func TestThermalMonitor_StartStop(t *testing.T) {
	tm := NewThermalMonitor(nil, 100*time.Millisecond)

	tm.Start()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	tm.Stop()

	// Verify context is cancelled
	select {
	case <-tm.ctx.Done():
		// Good - context was cancelled
	case <-time.After(1 * time.Second):
		t.Error("Stop() did not cancel context")
	}
}

func TestThermalMonitor_GetState(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	// Initially should return nil for unknown hardware
	state := tm.GetState("nvidia")
	if state != nil {
		t.Error("Expected nil state for unknown hardware")
	}

	// Add a mock state
	tm.mu.Lock()
	tm.states["nvidia"] = &ThermalState{
		Temperature: 65.0,
		FanPercent:  45,
		PowerDraw:   150.0,
		Utilization: 80,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	// Now should return the state
	state = tm.GetState("nvidia")
	if state == nil {
		t.Fatal("Expected state for nvidia, got nil")
	}

	if state.Temperature != 65.0 {
		t.Errorf("Expected temperature 65.0, got %.1f", state.Temperature)
	}

	if state.FanPercent != 45 {
		t.Errorf("Expected fan 45%%, got %d%%", state.FanPercent)
	}
}

func TestThermalMonitor_GetAllStates(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	// Initially should return empty map
	states := tm.GetAllStates()
	if states == nil {
		t.Fatal("GetAllStates returned nil")
	}
	if len(states) != 0 {
		t.Errorf("Expected 0 states, got %d", len(states))
	}

	// Add multiple mock states
	tm.mu.Lock()
	tm.states["nvidia"] = &ThermalState{Temperature: 65.0, FanPercent: 45}
	tm.states["cpu"] = &ThermalState{Temperature: 55.0, FanPercent: 30}
	tm.mu.Unlock()

	states = tm.GetAllStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	if states["nvidia"] == nil {
		t.Error("Expected nvidia state")
	}

	if states["cpu"] == nil {
		t.Error("Expected cpu state")
	}
}

func TestThermalMonitor_IsHealthy(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	tests := []struct {
		name        string
		temperature float64
		throttling  bool
		expected    bool
	}{
		{"Normal temp", 60.0, false, true},
		{"Warning temp", 75.0, false, true},
		{"Critical temp", 90.0, false, false},
		{"Shutdown temp", 95.0, false, false},
		{"Throttling", 60.0, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm.mu.Lock()
			tm.states["test"] = &ThermalState{
				Temperature: tt.temperature,
				Throttling:  tt.throttling,
				UpdatedAt:   time.Now(),
			}
			tm.mu.Unlock()

			healthy := tm.IsHealthy("test")
			if healthy != tt.expected {
				t.Errorf("IsHealthy() = %v, expected %v (temp=%.1f, throttling=%v)",
					healthy, tt.expected, tt.temperature, tt.throttling)
			}
		})
	}
}

func TestThermalMonitor_CanUse(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// No state - should be usable
	canUse, reason := tm.CanUse("unknown")
	if !canUse {
		t.Error("Expected CanUse=true for unknown hardware")
	}

	// Normal temperature - usable
	tm.mu.Lock()
	tm.states["normal"] = &ThermalState{
		Temperature: 65.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("normal")
	if !canUse {
		t.Errorf("Expected CanUse=true for normal temp, got false: %s", reason)
	}

	// Critical temperature - not usable
	tm.mu.Lock()
	tm.states["hot"] = &ThermalState{
		Temperature: 90.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("hot")
	if canUse {
		t.Errorf("Expected CanUse=false for critical temp, got true")
	}
	if reason == "" {
		t.Error("Expected reason for CanUse=false")
	}

	// Throttling - not usable
	tm.mu.Lock()
	tm.states["throttling"] = &ThermalState{
		Temperature: 65.0,
		Throttling:  true,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("throttling")
	if canUse {
		t.Errorf("Expected CanUse=false for throttling, got true")
	}
}

func TestThermalMonitor_GetThermalPenalty(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	tests := []struct {
		name           string
		temperature    float64
		expectNonZero  bool
		expectHigh     bool // penalty > 100
	}{
		{"Cool", 50.0, false, false},
		{"Normal", 65.0, false, false},
		{"Warning", 75.0, true, true},    // Exponential penalty kicks in
		{"High", 82.0, true, true},       // Higher exponential penalty
		{"Critical", 88.0, true, true},   // Very high penalty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm.mu.Lock()
			tm.states["test"] = &ThermalState{
				Temperature: tt.temperature,
				UpdatedAt:   time.Now(),
			}
			tm.mu.Unlock()

			penalty := tm.GetThermalPenalty("test")

			if tt.expectNonZero {
				if penalty == 0.0 {
					t.Errorf("Expected non-zero penalty for temp %.1f, got 0.0", tt.temperature)
				}
			} else {
				if penalty != 0.0 {
					t.Errorf("Expected zero penalty for temp %.1f, got %.2f", tt.temperature, penalty)
				}
			}

			if tt.expectHigh && penalty < 100.0 {
				t.Errorf("Expected high penalty (>100) for temp %.1f, got %.2f", tt.temperature, penalty)
			}
		})
	}

	// Test throttling penalty
	t.Run("Throttling", func(t *testing.T) {
		tm.mu.Lock()
		tm.states["test"] = &ThermalState{
			Temperature: 65.0,
			Throttling:  true,
			UpdatedAt:   time.Now(),
		}
		tm.mu.Unlock()

		penalty := tm.GetThermalPenalty("test")
		if penalty < 2000 {
			t.Errorf("Expected high penalty for throttling (>=2000), got %.2f", penalty)
		}
	})
}

func TestThermalMonitor_GetCoolestBackend(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Add states for multiple hardware
	tm.mu.Lock()
	tm.states["hot"] = &ThermalState{
		Temperature: 85.0,
		UpdatedAt:   time.Now(),
	}
	tm.states["warm"] = &ThermalState{
		Temperature: 70.0,
		UpdatedAt:   time.Now(),
	}
	tm.states["cool"] = &ThermalState{
		Temperature: 55.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	candidates := []string{"hot", "warm", "cool"}
	coolest := tm.GetCoolestBackend(candidates)

	if coolest != "cool" {
		t.Errorf("Expected 'cool' as coolest, got %s", coolest)
	}

	// Test with empty candidates
	coolest = tm.GetCoolestBackend([]string{})
	if coolest != "" {
		t.Errorf("Expected empty string for empty candidates, got %s", coolest)
	}

	// Test with unknown hardware (no states) - returns empty since all are nil
	coolest = tm.GetCoolestBackend([]string{"unknown1", "unknown2"})
	if coolest != "" {
		t.Errorf("Expected empty string for unknown hardware (no states), got %s", coolest)
	}
}

func TestThermalMonitor_ShouldPreferQuiet(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// No states - not quiet hours (no loud fans)
	if tm.ShouldPreferQuiet() {
		t.Error("Expected false with no states")
	}

	// Low fan speed only - not quiet hours (no fans > moderate)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		FanPercent: 25,
		UpdatedAt:  time.Now(),
	}
	tm.mu.Unlock()

	if tm.ShouldPreferQuiet() {
		t.Error("Expected false when fan speed < moderate (25 < 60)")
	}

	// High fan speed - triggers quiet mode (fan > moderate threshold)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		FanPercent: 70,
		UpdatedAt:  time.Now(),
	}
	tm.mu.Unlock()

	if !tm.ShouldPreferQuiet() {
		t.Error("Expected true when fan speed > moderate (70 > 60)")
	}
}

func TestThermalMonitor_GetConfig(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	retrievedConfig := tm.GetConfig()

	if retrievedConfig.TempWarning != 70.0 {
		t.Errorf("Expected TempWarning 70.0, got %.1f", retrievedConfig.TempWarning)
	}

	if retrievedConfig.TempCritical != 85.0 {
		t.Errorf("Expected TempCritical 85.0, got %.1f", retrievedConfig.TempCritical)
	}

	if retrievedConfig.CooldownTime != 2*time.Minute {
		t.Errorf("Expected CooldownTime 2m, got %v", retrievedConfig.CooldownTime)
	}
}

func TestThermalState_UpdatedAt(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	now := time.Now()

	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 65.0,
		UpdatedAt:   now,
	}
	tm.mu.Unlock()

	state := tm.GetState("test")
	if state == nil {
		t.Fatal("Expected state")
	}

	// Check UpdatedAt is close to our timestamp
	if state.UpdatedAt.Sub(now) > time.Second {
		t.Errorf("UpdatedAt timestamp mismatch: expected ~%v, got %v", now, state.UpdatedAt)
	}
}

func TestThermalMonitor_ConcurrentAccess(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	// Test concurrent read/write access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			tm.mu.Lock()
			tm.states["concurrent"] = &ThermalState{
				Temperature: float64(50 + i%30),
				UpdatedAt:   time.Now(),
			}
			tm.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = tm.GetState("concurrent")
			_ = tm.GetAllStates()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// If we get here without panicking, concurrent access works
}

// TestThermalState_String tests the String method of ThermalState
func TestThermalState_String(t *testing.T) {
	state := &ThermalState{
		Temperature: 72.5,
		FanPercent:  65,
		PowerDraw:   125.3,
		Utilization: 85,
		Throttling:  true,
		UpdatedAt:   time.Now(),
	}

	result := state.String()
	if result == "" {
		t.Error("String() returned empty string")
	}

	// Check that key components are in the string
	if !strings.Contains(result, "72.5") {
		t.Errorf("String() missing temperature: %s", result)
	}

	if !strings.Contains(result, "65%") {
		t.Errorf("String() missing fan percent: %s", result)
	}

	if !strings.Contains(result, "125.3W") {
		t.Errorf("String() missing power: %s", result)
	}

	if !strings.Contains(result, "true") {
		t.Errorf("String() missing throttling status: %s", result)
	}
}

// TestThermalMonitor_IsHealthy_NilState tests IsHealthy with nil state
func TestThermalMonitor_IsHealthy_NilState(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	// Unknown hardware should return true (assume healthy)
	healthy := tm.IsHealthy("unknown-hardware")
	if !healthy {
		t.Error("Expected IsHealthy=true for unknown hardware (nil state)")
	}
}

// TestThermalMonitor_CanUse_EdgeCases tests edge cases in CanUse
func TestThermalMonitor_CanUse_EdgeCases(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test at exactly warning temperature (should be usable)
	tm.mu.Lock()
	tm.states["warning"] = &ThermalState{
		Temperature: 70.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason := tm.CanUse("warning")
	if !canUse {
		t.Errorf("Expected CanUse=true at warning temp (70.0), got false: %s", reason)
	}

	// Test just below critical temperature
	tm.mu.Lock()
	tm.states["nearCrit"] = &ThermalState{
		Temperature: 84.9,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("nearCrit")
	if !canUse {
		t.Errorf("Expected CanUse=true just below critical, got false: %s", reason)
	}

	// Test at exactly critical temperature
	tm.mu.Lock()
	tm.states["critical"] = &ThermalState{
		Temperature: 85.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("critical")
	if canUse {
		t.Error("Expected CanUse=false at critical temp (85.0)")
	}
	if !strings.Contains(reason, "critical") {
		t.Errorf("Expected 'critical' in reason, got: %s", reason)
	}

	// Test at shutdown temperature - note: it triggers "critical" message first
	// since critical (85) is checked before shutdown (95) in CanUse()
	tm.mu.Lock()
	tm.states["shutdown"] = &ThermalState{
		Temperature: 95.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("shutdown")
	if canUse {
		t.Error("Expected CanUse=false at shutdown temp (95.0)")
	}
	if !strings.Contains(reason, "critical") {
		t.Errorf("Expected 'critical' in reason (shutdown overrides to critical check), got: %s", reason)
	}

	// Test above shutdown temperature
	tm.mu.Lock()
	tm.states["extreme"] = &ThermalState{
		Temperature: 100.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason = tm.CanUse("extreme")
	if canUse {
		t.Error("Expected CanUse=false above shutdown")
	}
}

// TestThermalMonitor_GetThermalPenalty_FanNoise tests fan noise penalty calculation
func TestThermalMonitor_GetThermalPenalty_FanNoise(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test at exactly loud threshold (no penalty)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		FanPercent:  85,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("test")
	if penalty != 0.0 {
		t.Errorf("Expected zero penalty at loud threshold (85), got %.2f", penalty)
	}

	// Test above loud threshold
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		FanPercent:  90,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty = tm.GetThermalPenalty("test")
	if penalty == 0.0 {
		t.Error("Expected non-zero penalty above loud threshold (90 > 85)")
	}

	// Penalty should be (90-85)*5 = 25
	if penalty < 20.0 || penalty > 30.0 {
		t.Errorf("Expected penalty around 25 for fan at 90%%, got %.2f", penalty)
	}

	// Test high fan speed
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		FanPercent:  100,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty = tm.GetThermalPenalty("test")
	// Expected penalty: (100-85)*5 = 75
	if penalty < 70.0 || penalty > 80.0 {
		t.Errorf("Expected penalty around 75 for fan at 100%%, got %.2f", penalty)
	}
}

// TestThermalMonitor_GetThermalPenalty_HighUtilization tests utilization penalty
func TestThermalMonitor_GetThermalPenalty_HighUtilization(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test below utilization threshold (no penalty)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		Utilization: 75,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("test")
	if penalty != 0.0 {
		t.Errorf("Expected zero penalty for 75%% utilization, got %.2f", penalty)
	}

	// Test at exactly threshold (80%)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		Utilization: 80,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty = tm.GetThermalPenalty("test")
	if penalty != 0.0 {
		t.Errorf("Expected zero penalty at 80%% utilization threshold, got %.2f", penalty)
	}

	// Test above threshold
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		Utilization: 95,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty = tm.GetThermalPenalty("test")
	// Expected penalty: (95-80)*10 = 150
	if penalty < 140.0 || penalty > 160.0 {
		t.Errorf("Expected penalty around 150 for 95%% utilization, got %.2f", penalty)
	}
}

// TestThermalMonitor_GetThermalPenalty_CombinedPenalties tests combined penalty factors
func TestThermalMonitor_GetThermalPenalty_CombinedPenalties(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test all penalties at once
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 78.0,    // Above warning
		FanPercent:  90,      // Above loud
		Utilization: 90,      // Above 80%
		Throttling:  true,    // Throttling
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("test")

	// Should have all four penalty components
	// Temperature: (78-70)/(85-70) = 8/15 = 0.533, 0.533^2 * 1000 ≈ 284
	// Fan: (90-85)*5 = 25
	// Utilization: (90-80)*10 = 100
	// Throttling: 2000
	// Total should be around 2409+

	if penalty < 2400 {
		t.Errorf("Expected high combined penalty (>2400), got %.2f", penalty)
	}
}

// TestThermalMonitor_GetThermalPenalty_NilState tests penalty with nil state
func TestThermalMonitor_GetThermalPenalty_NilState(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	penalty := tm.GetThermalPenalty("unknown")
	if penalty != 0.0 {
		t.Errorf("Expected zero penalty for unknown hardware, got %.2f", penalty)
	}
}

// TestThermalMonitor_ShouldPreferQuiet_EdgeCases tests edge cases in quiet mode
func TestThermalMonitor_ShouldPreferQuiet_EdgeCases(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test at exactly moderate threshold (should not trigger quiet mode)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		FanPercent: 60,
		UpdatedAt:  time.Now(),
	}
	tm.mu.Unlock()

	if tm.ShouldPreferQuiet() {
		t.Error("Expected false when fan speed == moderate (60 == 60)")
	}

	// Test just above moderate (should trigger quiet mode)
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		FanPercent: 61,
		UpdatedAt:  time.Now(),
	}
	tm.mu.Unlock()

	if !tm.ShouldPreferQuiet() {
		t.Error("Expected true when fan speed > moderate (61 > 60)")
	}

	// Test with multiple backends, one loud
	tm.mu.Lock()
	tm.states["quiet"] = &ThermalState{
		FanPercent: 30,
		UpdatedAt:  time.Now(),
	}
	tm.states["loud"] = &ThermalState{
		FanPercent: 90,
		UpdatedAt:  time.Now(),
	}
	tm.mu.Unlock()

	if !tm.ShouldPreferQuiet() {
		t.Error("Expected true when any backend is loud")
	}
}

// TestThermalMonitor_GetCoolestBackend_EdgeCases tests edge cases in GetCoolestBackend
func TestThermalMonitor_GetCoolestBackend_EdgeCases(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test with single candidate
	tm.mu.Lock()
	tm.states["single"] = &ThermalState{
		Temperature: 50.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	coolest := tm.GetCoolestBackend([]string{"single"})
	if coolest != "single" {
		t.Errorf("Expected 'single', got %s", coolest)
	}

	// Test with mixed known/unknown
	tm.mu.Lock()
	tm.states["known"] = &ThermalState{
		Temperature: 50.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	coolest = tm.GetCoolestBackend([]string{"known", "unknown"})
	if coolest != "known" {
		t.Errorf("Expected 'known' (skips unknown), got %s", coolest)
	}

	// Test with all candidates unknown
	coolest = tm.GetCoolestBackend([]string{"unknown1", "unknown2", "unknown3"})
	if coolest != "" {
		t.Errorf("Expected empty string for all unknown, got %s", coolest)
	}

	// Test with temperature equality - first one should win
	tm.mu.Lock()
	tm.states["cool1"] = &ThermalState{
		Temperature: 50.0,
		UpdatedAt:   time.Now(),
	}
	tm.states["cool2"] = &ThermalState{
		Temperature: 50.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	coolest = tm.GetCoolestBackend([]string{"cool1", "cool2"})
	if coolest != "cool1" {
		t.Errorf("Expected first candidate on tie, got %s", coolest)
	}

	// Test with zero temperature
	tm.mu.Lock()
	tm.states["zero"] = &ThermalState{
		Temperature: 0.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	coolest = tm.GetCoolestBackend([]string{"zero"})
	if coolest != "zero" {
		t.Errorf("Expected 'zero' temperature candidate, got %s", coolest)
	}
}

// TestThermalMonitor_TemperatureThresholdBoundaries tests exact boundary conditions
func TestThermalMonitor_TemperatureThresholdBoundaries(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	tests := []struct {
		name        string
		temperature float64
		healthy     bool
		canUse      bool
	}{
		{"Below warning", 69.9, true, true},
		{"At warning", 70.0, true, true},
		{"Above warning", 70.1, true, true},
		{"Below critical", 84.9, true, true},
		{"At critical", 85.0, false, false},
		{"Above critical", 85.1, false, false},
		{"Below shutdown", 94.9, false, false},
		{"At shutdown", 95.0, false, false},
		{"Above shutdown", 95.1, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm.mu.Lock()
			tm.states["test"] = &ThermalState{
				Temperature: tt.temperature,
				Throttling:  false,
				UpdatedAt:   time.Now(),
			}
			tm.mu.Unlock()

			healthy := tm.IsHealthy("test")
			if healthy != tt.healthy {
				t.Errorf("IsHealthy: expected %v, got %v (temp=%.1f)", tt.healthy, healthy, tt.temperature)
			}

			canUse, _ := tm.CanUse("test")
			if canUse != tt.canUse {
				t.Errorf("CanUse: expected %v, got %v (temp=%.1f)", tt.canUse, canUse, tt.temperature)
			}
		})
	}
}

// TestThermalMonitor_ConcurrentMultipleHardware tests concurrent access with multiple hardware
func TestThermalMonitor_ConcurrentMultipleHardware(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)
	done := make(chan bool, 4)

	hardwareIDs := []string{"nvidia", "igpu", "cpu", "npu"}

	// Multiple writer goroutines for different hardware
	for _, hwID := range hardwareIDs {
		go func(id string) {
			for i := 0; i < 50; i++ {
				tm.mu.Lock()
				tm.states[id] = &ThermalState{
					Temperature: float64(40 + (i % 50)),
					FanPercent:  (i % 100),
					Utilization: (i % 100),
					UpdatedAt:   time.Now(),
				}
				tm.mu.Unlock()
			}
			done <- true
		}(hwID)
	}

	// Reader goroutines
	go func() {
		for i := 0; i < 100; i++ {
			tm.GetAllStates()
			for _, hw := range hardwareIDs {
				tm.GetState(hw)
				tm.IsHealthy(hw)
				tm.CanUse(hw)
				tm.GetThermalPenalty(hw)
			}
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify final state
	states := tm.GetAllStates()
	if len(states) != 4 {
		t.Errorf("Expected 4 hardware states, got %d", len(states))
	}
}

// TestThermalMonitor_MonitorLoopExecution tests that monitor loop executes
func TestThermalMonitor_MonitorLoopExecution(t *testing.T) {
	tm := NewThermalMonitor(nil, 50*time.Millisecond)

	tm.Start()

	// Give the monitor loop time to execute
	time.Sleep(150 * time.Millisecond)

	// Verify we can still interact with the monitor
	states := tm.GetAllStates()
	if states == nil {
		t.Error("Monitor loop should not break GetAllStates()")
	}

	// Verify context is still active
	select {
	case <-tm.ctx.Done():
		t.Error("Context should not be cancelled while running")
	default:
		// Good - context is still active
	}

	tm.Stop()

	// After stop, context should be done
	select {
	case <-tm.ctx.Done():
		// Good
	case <-time.After(1 * time.Second):
		t.Error("Context should be cancelled after Stop()")
	}
}

// TestThermalMonitor_GetStateThreadSafety tests thread safety of GetState
func TestThermalMonitor_GetStateThreadSafety(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)
	results := make(chan *ThermalState, 100)

	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 50.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func() {
			state := tm.GetState("test")
			results <- state
		}()
	}

	// Collect results
	for i := 0; i < 50; i++ {
		state := <-results
		if state == nil {
			t.Error("GetState returned nil during concurrent access")
		}
		if state.Temperature != 50.0 {
			t.Errorf("Got corrupted state: %.1f", state.Temperature)
		}
	}
}

// TestThermalMonitor_ConfigNilHandling tests nil config is properly handled
func TestThermalMonitor_ConfigNilHandling(t *testing.T) {
	// Create with nil config
	tm := NewThermalMonitor(nil, 5*time.Second)

	if tm.config == nil {
		t.Fatal("Config should not be nil, should use defaults")
	}

	// Verify all defaults are set
	if tm.config.TempWarning != 70.0 {
		t.Errorf("Default TempWarning should be 70.0, got %.1f", tm.config.TempWarning)
	}

	if tm.config.TempCritical != 85.0 {
		t.Errorf("Default TempCritical should be 85.0, got %.1f", tm.config.TempCritical)
	}

	if tm.config.TempShutdown != 95.0 {
		t.Errorf("Default TempShutdown should be 95.0, got %.1f", tm.config.TempShutdown)
	}

	if tm.config.FanQuiet != 30 {
		t.Errorf("Default FanQuiet should be 30, got %d", tm.config.FanQuiet)
	}

	if tm.config.FanModerate != 60 {
		t.Errorf("Default FanModerate should be 60, got %d", tm.config.FanModerate)
	}

	if tm.config.FanLoud != 85 {
		t.Errorf("Default FanLoud should be 85, got %d", tm.config.FanLoud)
	}

	if tm.config.CooldownTime != 2*time.Minute {
		t.Errorf("Default CooldownTime should be 2m, got %v", tm.config.CooldownTime)
	}
}

// TestThermalMonitor_GetAllStatesCopy verifies GetAllStates returns a copy
func TestThermalMonitor_GetAllStatesCopy(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	tm.mu.Lock()
	tm.states["original"] = &ThermalState{
		Temperature: 50.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	// Get states copy
	statesCopy := tm.GetAllStates()

	// Modify the copy
	statesCopy["modified"] = &ThermalState{
		Temperature: 99.0,
		UpdatedAt:   time.Now(),
	}

	// Original should not have the modification
	originalStates := tm.GetAllStates()
	if len(originalStates) != 1 {
		t.Errorf("Original states were modified! Expected 1, got %d", len(originalStates))
	}

	if _, exists := originalStates["modified"]; exists {
		t.Error("Modification leaked to original states map")
	}
}

// TestThermalMonitor_TemperaturePenaltyExponential tests exponential temperature penalty
func TestThermalMonitor_TemperaturePenaltyExponential(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Test exponential nature: penalty should increase non-linearly
	temps := []float64{71.0, 75.0, 80.0, 84.0}
	var penalties []float64

	for _, temp := range temps {
		tm.mu.Lock()
		tm.states["test"] = &ThermalState{
			Temperature: temp,
			UpdatedAt:   time.Now(),
		}
		tm.mu.Unlock()

		penalty := tm.GetThermalPenalty("test")
		penalties = append(penalties, penalty)
	}

	// Each step should increase penalty more than the previous
	for i := 1; i < len(penalties); i++ {
		if penalties[i] <= penalties[i-1] {
			t.Errorf("Exponential penalty not increasing properly at index %d", i)
		}
	}

	// At critical temperature, penalty should approach or exceed 1000
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 84.9,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("test")
	if penalty < 900 {
		t.Errorf("Expected high penalty near critical temp, got %.2f", penalty)
	}
}

// TestThermalMonitor_MultipleHardwareSelection tests backend selection logic
func TestThermalMonitor_MultipleHardwareSelection(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Set up multiple backends with different thermal states
	// Use temps above warning to show clear penalty differences
	tm.mu.Lock()
	tm.states["hot"] = &ThermalState{
		Temperature: 82.0,
		FanPercent:  80,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.states["warm"] = &ThermalState{
		Temperature: 75.0,  // Above warning (70), so has temperature penalty
		FanPercent:  50,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.states["cool"] = &ThermalState{
		Temperature: 68.0,  // Below warning, so no temperature penalty
		FanPercent:  30,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	// All backends should be usable
	hotUsable, _ := tm.CanUse("hot")
	warmUsable, _ := tm.CanUse("warm")
	coolUsable, _ := tm.CanUse("cool")

	if !hotUsable {
		t.Error("Expected hot backend to be usable")
	}
	if !warmUsable {
		t.Error("Expected warm backend to be usable")
	}
	if !coolUsable {
		t.Error("Expected cool backend to be usable")
	}

	// Cool should be preferred (lowest penalty)
	hotPenalty := tm.GetThermalPenalty("hot")
	warmPenalty := tm.GetThermalPenalty("warm")
	coolPenalty := tm.GetThermalPenalty("cool")

	// Cool has no temp penalty, warm has some, hot has high temp penalty + fan penalty
	if coolPenalty >= warmPenalty || warmPenalty >= hotPenalty {
		t.Errorf("Penalty ordering wrong: cool=%.2f, warm=%.2f, hot=%.2f",
			coolPenalty, warmPenalty, hotPenalty)
	}

	// GetCoolestBackend should select cool
	coolest := tm.GetCoolestBackend([]string{"hot", "warm", "cool"})
	if coolest != "cool" {
		t.Errorf("Expected 'cool' as coolest, got %s", coolest)
	}
}

// TestThermalMonitor_CanUse_NilState tests CanUse with unknown hardware
func TestThermalMonitor_CanUse_NilState(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	// Unknown hardware should be usable
	canUse, reason := tm.CanUse("unknown-hw")
	if !canUse {
		t.Error("Expected CanUse=true for unknown hardware")
	}
	if reason != "" {
		t.Errorf("Expected empty reason for unknown hardware, got: %s", reason)
	}
}

// TestThermalMonitor_CanUse_Throttling tests CanUse when throttling is active
func TestThermalMonitor_CanUse_Throttling(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}

	tm := NewThermalMonitor(config, 5*time.Second)

	// Set hardware with normal temp but throttling
	tm.mu.Lock()
	tm.states["throttling"] = &ThermalState{
		Temperature: 60.0,
		Throttling:  true,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	canUse, reason := tm.CanUse("throttling")
	if canUse {
		t.Error("Expected CanUse=false when throttling")
	}
	if !strings.Contains(reason, "throttling") {
		t.Errorf("Expected 'throttling' in reason, got: %s", reason)
	}
}

// TestThermalMonitor_CoolestBackend_WithNilStates tests GetCoolestBackend filtering
func TestThermalMonitor_CoolestBackend_WithNilStates(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	tm.mu.Lock()
	tm.states["hw1"] = &ThermalState{
		Temperature: 55.0,
		UpdatedAt:   time.Now(),
	}
	tm.states["hw2"] = &ThermalState{
		Temperature: 65.0,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	// Include some non-existent candidates
	coolest := tm.GetCoolestBackend([]string{"unknown1", "hw1", "unknown2", "hw2"})
	if coolest != "hw1" {
		t.Errorf("Expected hw1 (lowest among existing), got %s", coolest)
	}
}

// TestThermalMonitor_ShouldPreferQuiet_MultipleBackends tests with multiple backends
func TestThermalMonitor_ShouldPreferQuiet_MultipleBackends(t *testing.T) {
	config := &ThermalConfig{
		FanModerate: 60,
	}
	tm := NewThermalMonitor(config, 5*time.Second)

	// Add many backends all below moderate
	tm.mu.Lock()
	for i := 1; i <= 5; i++ {
		tm.states[fmt.Sprintf("hw%d", i)] = &ThermalState{
			FanPercent: 40,
			UpdatedAt:  time.Now(),
		}
	}
	tm.mu.Unlock()

	if tm.ShouldPreferQuiet() {
		t.Error("Expected false when all backends below moderate")
	}

	// Add one backend above moderate
	tm.mu.Lock()
	tm.states["loud"] = &ThermalState{
		FanPercent: 70,
		UpdatedAt:  time.Now(),
	}
	tm.mu.Unlock()

	if !tm.ShouldPreferQuiet() {
		t.Error("Expected true when any backend above moderate")
	}
}

// TestThermalMonitor_IsHealthy_HealthyCases tests various healthy scenarios
func TestThermalMonitor_IsHealthy_HealthyCases(t *testing.T) {
	config := &ThermalConfig{
		TempCritical: 85.0,
	}
	tm := NewThermalMonitor(config, 5*time.Second)

	// Low temperature, no throttling
	tm.mu.Lock()
	tm.states["healthy1"] = &ThermalState{
		Temperature: 40.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	if !tm.IsHealthy("healthy1") {
		t.Error("Expected healthy for low temp, no throttle")
	}

	// Just below critical
	tm.mu.Lock()
	tm.states["healthy2"] = &ThermalState{
		Temperature: 84.5,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	if !tm.IsHealthy("healthy2") {
		t.Error("Expected healthy just below critical")
	}
}

// TestThermalMonitor_GetThermalPenalty_ZeroUtilization tests penalty with zero util
func TestThermalMonitor_GetThermalPenalty_ZeroUtilization(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	tm.mu.Lock()
	tm.states["idle"] = &ThermalState{
		Temperature: 50.0,
		Utilization: 0,
		FanPercent:  10,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("idle")
	if penalty != 0.0 {
		t.Errorf("Expected zero penalty for idle backend, got %.2f", penalty)
	}
}

// TestThermalMonitor_GetThermalPenalty_MaxFanSpeed tests maximum fan speed penalty
func TestThermalMonitor_GetThermalPenalty_MaxFanSpeed(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		TempShutdown: 95.0,
		FanQuiet:     30,
		FanModerate:  60,
		FanLoud:      85,
		CooldownTime: 2 * time.Minute,
	}
	tm := NewThermalMonitor(config, 5*time.Second)

	tm.mu.Lock()
	tm.states["maxfan"] = &ThermalState{
		Temperature: 50.0,
		FanPercent:  100,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("maxfan")
	// Expected: (100-85)*5 = 75
	if penalty < 70.0 || penalty > 80.0 {
		t.Errorf("Expected penalty around 75 at max fan (100%%), got %.2f", penalty)
	}
}

// TestThermalMonitor_GetThermalPenalty_MaxUtilization tests maximum utilization
func TestThermalMonitor_GetThermalPenalty_MaxUtilization(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	tm.mu.Lock()
	tm.states["maxutil"] = &ThermalState{
		Temperature: 50.0,
		Utilization: 100,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("maxutil")
	// Expected: (100-80)*10 = 200
	if penalty < 190.0 || penalty > 210.0 {
		t.Errorf("Expected penalty around 200 at max util (100%%), got %.2f", penalty)
	}
}

// TestThermalMonitor_IsHealthy_Unhealthy tests unhealthy scenarios
func TestThermalMonitor_IsHealthy_Unhealthy(t *testing.T) {
	config := &ThermalConfig{
		TempCritical: 85.0,
	}
	tm := NewThermalMonitor(config, 5*time.Second)

	// At critical
	tm.mu.Lock()
	tm.states["critical"] = &ThermalState{
		Temperature: 85.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	if tm.IsHealthy("critical") {
		t.Error("Expected unhealthy at critical temp")
	}

	// Above critical
	tm.mu.Lock()
	tm.states["hot"] = &ThermalState{
		Temperature: 90.0,
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	if tm.IsHealthy("hot") {
		t.Error("Expected unhealthy above critical")
	}

	// Throttling only
	tm.mu.Lock()
	tm.states["throttle"] = &ThermalState{
		Temperature: 50.0,
		Throttling:  true,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	if tm.IsHealthy("throttle") {
		t.Error("Expected unhealthy when throttling")
	}
}

// TestThermalMonitor_ConcurrentReadWrite tests mixed read/write concurrency
func TestThermalMonitor_ConcurrentReadWrite(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)
	done := make(chan bool, 10)

	// Multiple writers updating different hardware
	for hw := 0; hw < 5; hw++ {
		go func(hwNum int) {
			for i := 0; i < 30; i++ {
				tm.mu.Lock()
				tm.states[fmt.Sprintf("hw%d", hwNum)] = &ThermalState{
					Temperature: float64(50 + i%40),
					FanPercent:  (i % 100),
					Utilization: (i % 100),
					UpdatedAt:   time.Now(),
				}
				tm.mu.Unlock()
				if i%10 == 0 {
					time.Sleep(time.Microsecond)
				}
			}
			done <- true
		}(hw)
	}

	// Multiple readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 30; j++ {
				tm.GetAllStates()
				for k := 0; k < 5; k++ {
					tm.GetState(fmt.Sprintf("hw%d", k))
					tm.IsHealthy(fmt.Sprintf("hw%d", k))
				}
			}
			done <- true
		}()
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestThermalMonitor_TemperaturePenalty_Precision tests penalty calculations
func TestThermalMonitor_TemperaturePenalty_Precision(t *testing.T) {
	config := &ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
	}
	tm := NewThermalMonitor(config, 5*time.Second)

	// Test at just above warning
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 70.1,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty := tm.GetThermalPenalty("test")
	// Should have a small non-zero penalty
	if penalty <= 0.0 || penalty > 1.0 {
		t.Errorf("Expected small penalty for 70.1°C, got %.4f", penalty)
	}

	// Test just below critical
	tm.mu.Lock()
	tm.states["test"] = &ThermalState{
		Temperature: 84.5,
		UpdatedAt:   time.Now(),
	}
	tm.mu.Unlock()

	penalty = tm.GetThermalPenalty("test")
	// Should be close to 1000 but less than it
	if penalty < 900.0 || penalty >= 1000.0 {
		t.Errorf("Expected penalty 900-1000 for 84.5°C, got %.2f", penalty)
	}
}

// TestThermalMonitor_LargeScaleSimulation tests with many backends
func TestThermalMonitor_LargeScaleSimulation(t *testing.T) {
	tm := NewThermalMonitor(nil, 5*time.Second)

	// Create 100 hardware backends with varying states
	tm.mu.Lock()
	for i := 0; i < 100; i++ {
		hw := fmt.Sprintf("hw%03d", i)
		tm.states[hw] = &ThermalState{
			Temperature: float64(30 + (i % 60)),
			FanPercent:  (i % 100),
			Utilization: (i % 100),
			Throttling:  (i % 10) == 0, // Every 10th one throttles
			UpdatedAt:   time.Now(),
		}
	}
	tm.mu.Unlock()

	// Verify all states accessible
	allStates := tm.GetAllStates()
	if len(allStates) != 100 {
		t.Errorf("Expected 100 states, got %d", len(allStates))
	}

	// Check various operations still work
	for i := 0; i < 10; i++ {
		hw := fmt.Sprintf("hw%03d", i*10)
		if tm.GetState(hw) == nil {
			t.Errorf("Failed to get state for %s", hw)
		}
		tm.IsHealthy(hw)
		tm.CanUse(hw)
		tm.GetThermalPenalty(hw)
	}

	// Test GetCoolestBackend with large set
	candidates := make([]string, 10)
	for i := 0; i < 10; i++ {
		candidates[i] = fmt.Sprintf("hw%03d", i*10)
	}
	coolest := tm.GetCoolestBackend(candidates)
	if coolest == "" {
		t.Error("Failed to find coolest backend from large set")
	}
}
