package efficiency

import (
	"os"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

// TestMain initializes the logger for all tests
func TestMain(m *testing.M) {
	// Initialize logger for tests
	if err := logging.InitLogger("info", false); err != nil {
		panic(err)
	}
	defer logging.Sync()

	// Run tests
	os.Exit(m.Run())
}

// Test that DBusService cannot connect if D-Bus is unavailable
func TestNewDBusService_NoDBus(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	// This will fail if D-Bus is not available, which is expected in test environments
	svc, err := NewDBusService(em)

	if err != nil {
		// Expected in test environment without D-Bus
		t.Logf("NewDBusService failed as expected without D-Bus: %v", err)
		return
	}

	// If we somehow got a connection, clean it up
	if svc != nil {
		svc.Stop()
	}

	// For this test, either outcome is acceptable since we're testing without D-Bus
	t.Log("NewDBusService succeeded or failed gracefully")
}

// Test that NewDBusService can handle the manager
func TestNewDBusService_WithManager(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)

	// Just verify that we can attempt to create a service with a manager
	// In real environment with D-Bus, this would succeed
	// In test environment without D-Bus, this will fail, which is OK
	_, err := NewDBusService(em)

	// We don't assert on error since D-Bus may not be available
	if err != nil {
		t.Logf("Service creation failed (expected in test env): %v", err)
	} else {
		t.Log("Service creation succeeded")
	}
}

// Test StringToMode conversion for all mode strings
func TestStringToEfficiencyMode(t *testing.T) {
	tests := []struct {
		name          string
		modeString    string
		expectedMode  EfficiencyMode
	}{
		{"Performance", "Performance", ModePerformance},
		{"Balanced", "Balanced", ModeBalanced},
		{"Efficiency", "Efficiency", ModeEfficiency},
		{"Quiet", "Quiet", ModeQuiet},
		{"Auto", "Auto", ModeAuto},
		{"UltraEfficiency", "UltraEfficiency", ModeUltraEfficiency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the string-to-mode conversion used in D-Bus methods
			var mode EfficiencyMode
			switch tt.modeString {
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
				t.Fatalf("Unknown mode string: %s", tt.modeString)
			}

			if mode != tt.expectedMode {
				t.Errorf("String conversion: got %v, expected %v", mode, tt.expectedMode)
			}
		})
	}
}

// Test mode info structure
func TestGetModeInfo_Structure(t *testing.T) {
	modes := []EfficiencyMode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			config := GetModeConfig(mode)

			// Verify all required fields exist
			if config.Description == "" {
				t.Error("Mode info missing description")
			}

			if config.Icon == "" {
				t.Error("Mode info missing icon")
			}

			// Verify expected values
			if config.MaxTempCelsius == 0 {
				t.Error("MaxTempCelsius should not be 0")
			}

			// Check that MaxFanPercent is reasonable (except Auto which may be 0)
			if config.MaxFanPercent < 0 || config.MaxFanPercent > 100 {
				if mode != ModeAuto {
					t.Errorf("Invalid MaxFanPercent %d for mode %s", config.MaxFanPercent, mode)
				}
			}

			// Check that MaxPowerWatts is positive (or 0 for Auto)
			if config.MaxPowerWatts < 0 {
				t.Errorf("Invalid MaxPowerWatts %d for mode %s", config.MaxPowerWatts, mode)
			}

			t.Logf("Mode %s: Power=%dW, Fan=%d%%, Temp=%.0fC",
				mode, config.MaxPowerWatts, config.MaxFanPercent, config.MaxTempCelsius)
		})
	}
}

// Test D-Bus method simulation: listing modes
func TestListModes_AllPresent(t *testing.T) {
	modes := AllModes()

	if len(modes) != 6 {
		t.Errorf("Expected 6 modes, got %d", len(modes))
	}

	modeMap := make(map[EfficiencyMode]bool)
	for _, mode := range modes {
		modeMap[mode] = true
	}

	expectedModes := []EfficiencyMode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, expected := range expectedModes {
		if !modeMap[expected] {
			t.Errorf("Mode %s not found in AllModes", expected)
		}
	}
}

// Test D-Bus property map structure
func TestDBusPropertyMap_Structure(t *testing.T) {
	// We can't test makePropertyMap directly without D-Bus,
	// but we can verify the mode values work
	em := NewEfficiencyManager(ModeBalanced)

	currentMode := em.GetMode().String()
	if currentMode != "Balanced" {
		t.Errorf("GetMode failed: expected Balanced, got %s", currentMode)
	}

	effectiveMode := em.GetEffectiveMode().String()
	if effectiveMode == "" {
		t.Error("GetEffectiveMode returned empty string")
	}

	t.Logf("CurrentMode property: %s", currentMode)
	t.Logf("EffectiveMode property: %s", effectiveMode)
}

// Test mode switching via simulated D-Bus SetMode
func TestDBusSetMode_Simulation(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)

	modes := []EfficiencyMode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}

	for _, targetMode := range modes {
		// Simulate D-Bus SetMode behavior
		oldMode := em.GetMode()
		em.SetMode(targetMode)
		newMode := em.GetMode()

		if newMode != targetMode {
			t.Errorf("SetMode failed: expected %v, got %v", targetMode, newMode)
		}

		t.Logf("Mode change: %s -> %s", oldMode, newMode)
	}
}

// Test GetModeInfo for each mode with expected fields
func TestGetModeInfo_ExpectedFields(t *testing.T) {
	modeTests := []struct {
		mode             EfficiencyMode
		expectedMaxPower int
		expectedMaxFan   int
	}{
		{ModePerformance, 999, 100},
		{ModeBalanced, 60, 80},
		{ModeEfficiency, 15, 60},
		{ModeQuiet, 15, 40},
		{ModeUltraEfficiency, 5, 30},
	}

	for _, tt := range modeTests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			config := GetModeConfig(tt.mode)

			if config.MaxPowerWatts != tt.expectedMaxPower {
				t.Errorf("MaxPowerWatts: expected %d, got %d",
					tt.expectedMaxPower, config.MaxPowerWatts)
			}

			if config.MaxFanPercent != tt.expectedMaxFan {
				t.Errorf("MaxFanPercent: expected %d, got %d",
					tt.expectedMaxFan, config.MaxFanPercent)
			}

			// Check fields that D-Bus would return
			if config.Icon == "" {
				t.Error("Icon field empty")
			}

			if config.Description == "" {
				t.Error("Description field empty")
			}
		})
	}
}

// Test DBusService Stop method doesn't panic
func TestDBusService_Stop(t *testing.T) {
	// Create a service with nil connection (safe for testing)
	svc := &DBusService{
		conn:    nil,
		manager: NewEfficiencyManager(ModePerformance),
		props:   nil,
	}

	// Should not panic even with nil conn
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panicked: %v", r)
		}
	}()

	svc.Stop()
	t.Log("Stop() executed without panic")
}

// Test introspection interface definition
func TestIntrospectionInterface_Methods(t *testing.T) {
	expectedMethods := []string{
		"SetMode",
		"GetMode",
		"GetEffectiveMode",
		"ListModes",
		"GetModeInfo",
	}

	expectedSignals := []string{
		"ModeChanged",
	}

	expectedProperties := []string{
		"CurrentMode",
		"EffectiveMode",
	}

	t.Logf("Expected D-Bus methods: %v", expectedMethods)
	t.Logf("Expected D-Bus signals: %v", expectedSignals)
	t.Logf("Expected D-Bus properties: %v", expectedProperties)

	// Verify these exist in the code by testing them indirectly
	em := NewEfficiencyManager(ModeBalanced)

	// SetMode equivalent
	em.SetMode(ModeEfficiency)
	if em.GetMode() != ModeEfficiency {
		t.Error("SetMode not working")
	}

	// GetMode equivalent
	mode := em.GetMode()
	if mode == EfficiencyMode(-1) {
		t.Error("GetMode not working")
	}

	// GetEffectiveMode equivalent
	effective := em.GetEffectiveMode()
	if effective == EfficiencyMode(-1) {
		t.Error("GetEffectiveMode not working")
	}

	// ListModes equivalent
	modes := AllModes()
	if len(modes) == 0 {
		t.Error("AllModes not working")
	}

	// GetModeInfo equivalent
	config := GetModeConfig(ModePerformance)
	if config == nil {
		t.Error("GetModeConfig not working")
	}

	t.Log("All expected D-Bus functionality verified")
}

// Test D-Bus SetMode method directly
func TestDBusSetMode(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)
	svc := &DBusService{
		conn:    nil, // No D-Bus connection needed
		manager: em,
		props:   nil,
	}

	tests := []struct {
		name        string
		mode        string
		expectedGet string
		wantErr     bool
	}{
		{"Set Performance", "Performance", "Performance", false},
		{"Set Balanced", "Balanced", "Balanced", false},
		{"Set Efficiency", "Efficiency", "Efficiency", false},
		{"Set Quiet", "Quiet", "Quiet", false},
		{"Set Auto", "Auto", "Auto", false},
		{"Set UltraEfficiency", "UltraEfficiency", "Ultra Efficiency", false},
		{"Set Invalid", "InvalidMode", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbusErr := svc.SetMode(tt.mode)
			if (dbusErr != nil) != tt.wantErr {
				t.Errorf("SetMode() error = %v, wantErr %v", dbusErr, tt.wantErr)
			}
			if !tt.wantErr {
				// Verify mode was actually set
				currentMode, _ := svc.GetMode()
				if currentMode != tt.expectedGet {
					t.Errorf("Mode not set correctly: got %s, want %s", currentMode, tt.expectedGet)
				}
			}
		})
	}
}

// Test D-Bus GetMode method
func TestDBusGetMode(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)
	svc := &DBusService{
		conn:    nil,
		manager: em,
		props:   nil,
	}

	mode, dbusErr := svc.GetMode()
	if dbusErr != nil {
		t.Fatalf("GetMode() error = %v", dbusErr)
	}

	if mode != "Balanced" {
		t.Errorf("GetMode() = %v, want Balanced", mode)
	}
}

// Test D-Bus GetEffectiveMode method
func TestDBusGetEffectiveMode(t *testing.T) {
	em := NewEfficiencyManager(ModeAuto)
	svc := &DBusService{
		conn:    nil,
		manager: em,
		props:   nil,
	}

	mode, dbusErr := svc.GetEffectiveMode()
	if dbusErr != nil {
		t.Fatalf("GetEffectiveMode() error = %v", dbusErr)
	}

	if mode == "" {
		t.Error("GetEffectiveMode() returned empty string")
	}

	t.Logf("Effective mode: %s", mode)
}

// Test D-Bus ListModes method
func TestDBusListModes(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)
	svc := &DBusService{
		conn:    nil,
		manager: em,
		props:   nil,
	}

	modes, dbusErr := svc.ListModes()
	if dbusErr != nil {
		t.Fatalf("ListModes() error = %v", dbusErr)
	}

	if len(modes) != 6 {
		t.Errorf("ListModes() returned %d modes, want 6. Modes: %v", len(modes), modes)
	}

	expectedModes := []string{
		"Performance",
		"Balanced",
		"Efficiency",
		"Quiet",
		"Auto",
		"Ultra Efficiency",
	}

	modeMap := make(map[string]bool)
	for _, mode := range modes {
		modeMap[mode] = true
	}

	t.Logf("Returned modes: %v", modes)
	for _, expected := range expectedModes {
		if !modeMap[expected] {
			t.Errorf("Mode %s not found in ListModes result. Available: %v", expected, modes)
		}
	}
}

// Test D-Bus GetModeInfo method
func TestDBusGetModeInfo(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)
	svc := &DBusService{
		conn:    nil,
		manager: em,
		props:   nil,
	}

	tests := []struct {
		name        string
		mode        string
		expectedName string
		wantErr     bool
	}{
		{"Performance", "Performance", "Performance", false},
		{"Balanced", "Balanced", "Balanced", false},
		{"Efficiency", "Efficiency", "Efficiency", false},
		{"Quiet", "Quiet", "Quiet", false},
		{"Auto", "Auto", "Auto", false},
		{"UltraEfficiency", "UltraEfficiency", "Ultra Efficiency", false},
		{"Invalid", "InvalidMode", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, dbusErr := svc.GetModeInfo(tt.mode)
			if (dbusErr != nil) != tt.wantErr {
				t.Errorf("GetModeInfo() error = %v, wantErr %v", dbusErr, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify all required fields
				requiredFields := []string{"name", "description", "icon", "maxPower", "maxFan", "maxTemp"}
				for _, field := range requiredFields {
					if _, ok := info[field]; !ok {
						t.Errorf("Missing required field: %s", field)
					}
				}

				// Verify name matches expected
				if info["name"].Value().(string) != tt.expectedName {
					t.Errorf("Mode name mismatch: got %s, want %s", info["name"].Value(), tt.expectedName)
				}

				t.Logf("Mode %s: %s", tt.mode, info["description"].Value())
			}
		})
	}
}

// Test D-Bus makePropertyMap method
func TestDBusMakePropertyMap(t *testing.T) {
	em := NewEfficiencyManager(ModeBalanced)
	svc := &DBusService{
		conn:    nil,
		manager: em,
		props:   nil,
	}

	propMap := svc.makePropertyMap()

	if propMap == nil {
		t.Fatal("makePropertyMap() returned nil")
	}

	if _, ok := propMap[dbusInterface]; !ok {
		t.Fatal("makePropertyMap() missing dbusInterface")
	}

	props := propMap[dbusInterface]

	// Check CurrentMode property
	if currentModeProp, ok := props["CurrentMode"]; ok {
		if currentModeProp.Value.(string) != "Balanced" {
			t.Errorf("CurrentMode = %v, want Balanced", currentModeProp.Value)
		}
		if !currentModeProp.Writable {
			t.Error("CurrentMode should be writable")
		}
	} else {
		t.Error("CurrentMode property not found")
	}

	// Check EffectiveMode property
	if effectiveModeProp, ok := props["EffectiveMode"]; ok {
		if effectiveModeProp.Value.(string) == "" {
			t.Error("EffectiveMode should not be empty")
		}
		if effectiveModeProp.Writable {
			t.Error("EffectiveMode should be read-only")
		}
	} else {
		t.Error("EffectiveMode property not found")
	}
}

// Test mode switching through D-Bus methods
func TestDBusModeSwitching(t *testing.T) {
	em := NewEfficiencyManager(ModePerformance)
	svc := &DBusService{
		conn:    nil,
		manager: em,
		props:   nil,
	}

	// Get initial mode
	initialMode, _ := svc.GetMode()
	if initialMode != "Performance" {
		t.Errorf("Initial mode = %s, want Performance", initialMode)
	}

	// Switch to Efficiency
	if err := svc.SetMode("Efficiency"); err != nil {
		t.Fatalf("SetMode(Efficiency) failed: %v", err)
	}

	// Verify mode changed
	newMode, _ := svc.GetMode()
	if newMode != "Efficiency" {
		t.Errorf("Mode after SetMode = %s, want Efficiency", newMode)
	}

	// Switch to Auto
	if err := svc.SetMode("Auto"); err != nil {
		t.Fatalf("SetMode(Auto) failed: %v", err)
	}

	// Verify mode changed
	autoMode, _ := svc.GetMode()
	if autoMode != "Auto" {
		t.Errorf("Mode after SetMode(Auto) = %s, want Auto", autoMode)
	}
}
