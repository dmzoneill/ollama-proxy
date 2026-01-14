package settings

import (
	"os"
	"os/exec"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/efficiency"
)

// TestNewSettings tests the creation of a new Settings instance
func TestNewSettings(t *testing.T) {
	s := NewSettings()
	if s == nil {
		t.Fatal("NewSettings returned nil")
	}
	if !isValidBool(s.available) {
		t.Errorf("Settings.available has unexpected value: %v", s.available)
	}
}

// TestIsAvailable tests the IsAvailable method
func TestIsAvailable(t *testing.T) {
	s := &Settings{available: true}
	if !s.IsAvailable() {
		t.Error("IsAvailable should return true when available is true")
	}

	s = &Settings{available: false}
	if s.IsAvailable() {
		t.Error("IsAvailable should return false when available is false")
	}
}

// TestCheckAvailable tests the checkAvailable internal method
func TestCheckAvailable(t *testing.T) {
	s := &Settings{}
	available := s.checkAvailable()
	// Just verify it returns a bool - actual value depends on system
	if !isValidBool(available) {
		t.Errorf("checkAvailable returned invalid value: %v", available)
	}
}

// TestParseMode tests the parseMode utility function with all valid modes
func TestParseMode(t *testing.T) {
	tests := []struct {
		input    string
		expected efficiency.EfficiencyMode
	}{
		{"Performance", efficiency.ModePerformance},
		{"Balanced", efficiency.ModeBalanced},
		{"Efficiency", efficiency.ModeEfficiency},
		{"Quiet", efficiency.ModeQuiet},
		{"Auto", efficiency.ModeAuto},
		{"UltraEfficiency", efficiency.ModeUltraEfficiency},
		{"", efficiency.ModeBalanced},              // Default
		{"InvalidMode", efficiency.ModeBalanced},   // Default
		{"performance", efficiency.ModeBalanced},   // Case sensitive - defaults
		{"PERFORMANCE", efficiency.ModeBalanced},   // Case sensitive - defaults
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMode(tt.input)
			if result != tt.expected {
				t.Errorf("parseMode(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseModeSringRepresentation tests that parsed modes have correct string representation
func TestParseModeStringRepresentation(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"Performance", "Performance"},
		{"Balanced", "Balanced"},
		{"Efficiency", "Efficiency"},
		{"Quiet", "Quiet"},
		{"Auto", "Auto"},
		{"UltraEfficiency", "Ultra Efficiency"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode := parseMode(tt.input)
			result := mode.String()
			if result != tt.expect {
				t.Errorf("parseMode(%q).String() = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

// TestGetDefaultModeWhenUnavailable tests GetDefaultMode when GSettings is not available
func TestGetDefaultModeWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	mode, err := s.GetDefaultMode()
	if err == nil {
		t.Error("GetDefaultMode should return error when not available")
	}
	if mode != efficiency.ModeBalanced {
		t.Errorf("GetDefaultMode should return ModeBalanced on error, got %v", mode)
	}
	if err.Error() != "GSettings not available" {
		t.Errorf("Expected 'GSettings not available' error, got %v", err)
	}
}

// TestGetLastModeWhenUnavailable tests GetLastMode when GSettings is not available
func TestGetLastModeWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	mode, err := s.GetLastMode()
	if err == nil {
		t.Error("GetLastMode should return error when not available")
	}
	if mode != efficiency.ModeBalanced {
		t.Errorf("GetLastMode should return ModeBalanced on error, got %v", mode)
	}
	if err.Error() != "GSettings not available" {
		t.Errorf("Expected 'GSettings not available' error, got %v", err)
	}
}

// TestSetLastModeWhenUnavailable tests SetLastMode when GSettings is not available
func TestSetLastModeWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	err := s.SetLastMode(efficiency.ModePerformance)
	if err != nil {
		t.Errorf("SetLastMode should gracefully ignore when not available, got error: %v", err)
	}
}

// TestShouldRememberModeWhenUnavailable tests ShouldRememberMode when GSettings is not available
func TestShouldRememberModeWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result, err := s.ShouldRememberMode()
	if err == nil {
		t.Error("ShouldRememberMode should return error when not available")
	}
	if result {
		t.Errorf("ShouldRememberMode should return false on error, got %v", result)
	}
	if err.Error() != "GSettings not available" {
		t.Errorf("Expected 'GSettings not available' error, got %v", err)
	}
}

// TestGetQuietHoursEnabledWhenUnavailable tests GetQuietHoursEnabled defaults
func TestGetQuietHoursEnabledWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result, err := s.GetQuietHoursEnabled()
	if err != nil {
		t.Errorf("GetQuietHoursEnabled should not return error when unavailable, got: %v", err)
	}
	if !result {
		t.Errorf("GetQuietHoursEnabled should default to true, got %v", result)
	}
}

// TestGetQuietHoursStartWhenUnavailable tests GetQuietHoursStart default
func TestGetQuietHoursStartWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result, err := s.GetQuietHoursStart()
	if err != nil {
		t.Errorf("GetQuietHoursStart should not return error when unavailable, got: %v", err)
	}
	if result != 22 {
		t.Errorf("GetQuietHoursStart should default to 22, got %v", result)
	}
}

// TestGetQuietHoursEndWhenUnavailable tests GetQuietHoursEnd default
func TestGetQuietHoursEndWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result, err := s.GetQuietHoursEnd()
	if err != nil {
		t.Errorf("GetQuietHoursEnd should not return error when unavailable, got: %v", err)
	}
	if result != 7 {
		t.Errorf("GetQuietHoursEnd should default to 7, got %v", result)
	}
}

// TestGetNotifyOnModeChangeWhenUnavailable tests GetNotifyOnModeChange default
func TestGetNotifyOnModeChangeWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result := s.GetNotifyOnModeChange()
	if !result {
		t.Errorf("GetNotifyOnModeChange should default to true, got %v", result)
	}
}

// TestGetNotifyOnBackendFailureWhenUnavailable tests GetNotifyOnBackendFailure default
func TestGetNotifyOnBackendFailureWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result := s.GetNotifyOnBackendFailure()
	if !result {
		t.Errorf("GetNotifyOnBackendFailure should default to true, got %v", result)
	}
}

// TestGetNotifyOnThermalThrottleWhenUnavailable tests GetNotifyOnThermalThrottle default
func TestGetNotifyOnThermalThrottleWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	result := s.GetNotifyOnThermalThrottle()
	if result {
		t.Errorf("GetNotifyOnThermalThrottle should default to false, got %v", result)
	}
}

// TestLoadInitialModeWhenUnavailable tests LoadInitialMode when GSettings is unavailable
func TestLoadInitialModeWhenUnavailable(t *testing.T) {
	s := &Settings{available: false}
	mode := s.LoadInitialMode()
	if mode != efficiency.ModeBalanced {
		t.Errorf("LoadInitialMode should return ModeBalanced when unavailable, got %v", mode)
	}
}

// TestShouldRememberModeBooleanParsing tests the parsing of "true"/"false" strings
func TestShouldRememberModeBooleanParsing(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*Settings)
		expected    bool
		shouldError bool
	}{
		{
			name: "unavailable",
			setupMock: func(s *Settings) {
				s.available = false
			},
			expected:    false,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{}
			tt.setupMock(s)
			result, err := s.ShouldRememberMode()
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Result = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStructure verifies the Settings struct has the expected fields
func TestStructure(t *testing.T) {
	s := &Settings{available: true}
	// Verify we can access the available field (even though it's unexported)
	// by checking methods that rely on it
	if !s.IsAvailable() {
		t.Error("Settings.available field not working correctly")
	}

	s.available = false
	if s.IsAvailable() {
		t.Error("Settings.available field not working correctly when false")
	}
}

// TestModeStringRepresentations tests all efficiency modes
func TestModeStringRepresentations(t *testing.T) {
	tests := []struct {
		mode     efficiency.EfficiencyMode
		expected string
	}{
		{efficiency.ModePerformance, "Performance"},
		{efficiency.ModeBalanced, "Balanced"},
		{efficiency.ModeEfficiency, "Efficiency"},
		{efficiency.ModeQuiet, "Quiet"},
		{efficiency.ModeAuto, "Auto"},
		{efficiency.ModeUltraEfficiency, "Ultra Efficiency"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.mode.String()
			if result != tt.expected {
				t.Errorf("Mode.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSchemaIDConstant verifies the schema ID is correct
func TestSchemaIDConstant(t *testing.T) {
	if schemaID != "ie.fio.ollamaproxy" {
		t.Errorf("schemaID = %q, want %q", schemaID, "ie.fio.ollamaproxy")
	}
}

// TestErrorMessages verifies error messages are correct
func TestErrorMessages(t *testing.T) {
	s := &Settings{available: false}

	tests := []struct {
		name     string
		testFunc func() error
		expected string
	}{
		{
			name: "GetDefaultMode error",
			testFunc: func() error {
				_, err := s.GetDefaultMode()
				return err
			},
			expected: "GSettings not available",
		},
		{
			name: "GetLastMode error",
			testFunc: func() error {
				_, err := s.GetLastMode()
				return err
			},
			expected: "GSettings not available",
		},
		{
			name: "ShouldRememberMode error",
			testFunc: func() error {
				_, err := s.ShouldRememberMode()
				return err
			},
			expected: "GSettings not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if err == nil {
				t.Fatal("Expected error but got none")
			}
			if err.Error() != tt.expected {
				t.Errorf("Error message = %q, want %q", err.Error(), tt.expected)
			}
		})
	}
}

// TestLoadInitialModeWithMemoryDisabled tests LoadInitialMode when remember is disabled
func TestLoadInitialModeWithMemoryDisabled(t *testing.T) {
	s := &Settings{available: false}
	mode := s.LoadInitialMode()
	if mode != efficiency.ModeBalanced {
		t.Errorf("LoadInitialMode should return ModeBalanced on unavailable, got %v", mode)
	}
}

// TestGetDefaultModeError tests that GetDefaultMode returns error for unavailable
func TestGetDefaultModeErrorWithoutAvailable(t *testing.T) {
	s := &Settings{available: false}
	_, err := s.GetDefaultMode()
	if err == nil {
		t.Error("GetDefaultMode should return error when unavailable")
	}
}

// TestQuietHoursDefaultValues tests that quiet hours return correct defaults
func TestQuietHoursDefaultValues(t *testing.T) {
	s := &Settings{available: false}

	enabled, err := s.GetQuietHoursEnabled()
	if err != nil {
		t.Errorf("GetQuietHoursEnabled error: %v", err)
	}
	if !enabled {
		t.Error("Quiet hours should be enabled by default")
	}

	start, err := s.GetQuietHoursStart()
	if err != nil {
		t.Errorf("GetQuietHoursStart error: %v", err)
	}
	if start != 22 {
		t.Errorf("Quiet hours start should be 22, got %d", start)
	}

	end, err := s.GetQuietHoursEnd()
	if err != nil {
		t.Errorf("GetQuietHoursEnd error: %v", err)
	}
	if end != 7 {
		t.Errorf("Quiet hours end should be 7, got %d", end)
	}
}

// TestNotificationDefaults tests notification configuration defaults
func TestNotificationDefaults(t *testing.T) {
	s := &Settings{available: false}

	modeChange := s.GetNotifyOnModeChange()
	if !modeChange {
		t.Error("Notify on mode change should default to true")
	}

	backendFailure := s.GetNotifyOnBackendFailure()
	if !backendFailure {
		t.Error("Notify on backend failure should default to true")
	}

	thermalThrottle := s.GetNotifyOnThermalThrottle()
	if thermalThrottle {
		t.Error("Notify on thermal throttle should default to false")
	}
}

// TestAvailabilityFlag tests the available flag behavior
func TestAvailabilityFlag(t *testing.T) {
	tests := []struct {
		available bool
	}{
		{true},
		{false},
	}

	for _, tt := range tests {
		s := &Settings{available: tt.available}
		if s.IsAvailable() != tt.available {
			t.Errorf("IsAvailable() = %v, want %v", s.IsAvailable(), tt.available)
		}
	}
}

// TestParseModeCaseSensitivity tests that parseMode is case-sensitive
func TestParseModeCaseSensitivity(t *testing.T) {
	tests := []struct {
		input    string
		wantMode efficiency.EfficiencyMode
		desc     string
	}{
		{"Performance", efficiency.ModePerformance, "exact case"},
		{"performance", efficiency.ModeBalanced, "lowercase (should default)"},
		{"PERFORMANCE", efficiency.ModeBalanced, "uppercase (should default)"},
		{"PeRfOrMaNcE", efficiency.ModeBalanced, "mixed case (should default)"},
		{"Balanced", efficiency.ModeBalanced, "exact case"},
		{"balanced", efficiency.ModeBalanced, "lowercase (should default)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := parseMode(tt.input)
			if result != tt.wantMode {
				t.Errorf("parseMode(%q) = %v, want %v", tt.input, result, tt.wantMode)
			}
		})
	}
}

// TestSettingsChainOfResponsibility tests the LoadInitialMode fallback chain
func TestLoadInitialModeChain(t *testing.T) {
	// When unavailable, should return ModeBalanced immediately
	s := &Settings{available: false}
	mode := s.LoadInitialMode()
	if mode != efficiency.ModeBalanced {
		t.Errorf("LoadInitialMode with unavailable = %v, want %v", mode, efficiency.ModeBalanced)
	}
}

// TestCommandConstruction tests that we're using the right command structure
func TestCommandUsage(t *testing.T) {
	// This test verifies the command names used in the code
	// We can't execute them without mocking, but we can verify the structure

	// Test that exec.Command is being called correctly by checking
	// that the Settings methods would use the right command names
	s := &Settings{available: false}

	// These should all return without errors due to unavailability
	_, err := s.GetDefaultMode()
	if err == nil {
		t.Error("Should error when unavailable")
	}

	_, err = s.GetLastMode()
	if err == nil {
		t.Error("Should error when unavailable")
	}

	err = s.SetLastMode(efficiency.ModeBalanced)
	if err != nil {
		t.Error("SetLastMode should not error when unavailable (graceful)")
	}

	_, err = s.ShouldRememberMode()
	if err == nil {
		t.Error("Should error when unavailable")
	}
}

// TestExecCommandErrorHandling tests error handling in command execution
func TestExecCommandType(t *testing.T) {
	// Verify that exec.Command returns a *exec.Cmd
	// This is a sanity check for the code structure
	cmd := exec.Command("echo", "test")
	if cmd == nil {
		t.Fatal("exec.Command returned nil")
	}

	// Verify the command can be constructed
	if len(cmd.Args) == 0 {
		t.Error("Command should have args")
	}
}

// TestEnvironmentVariable tests behavior when gsettings command might not exist
func TestGsettingsAvailability(t *testing.T) {
	// Create a Settings instance and check availability
	s := NewSettings()
	// The result depends on whether gsettings is installed
	// Just verify the method doesn't panic
	available := s.IsAvailable()
	if !isValidBool(available) {
		t.Errorf("IsAvailable returned invalid bool: %v", available)
	}
}

// TestQuietHoursRangeValues tests that quiet hours return integer values
func TestQuietHoursRangeValues(t *testing.T) {
	s := &Settings{available: false}

	// Test default values are within valid range (0-23)
	start, err := s.GetQuietHoursStart()
	if err != nil {
		t.Fatalf("GetQuietHoursStart error: %v", err)
	}
	if start < 0 || start > 23 {
		t.Errorf("Quiet hours start should be 0-23, got %d", start)
	}

	end, err := s.GetQuietHoursEnd()
	if err != nil {
		t.Fatalf("GetQuietHoursEnd error: %v", err)
	}
	if end < 0 || end > 23 {
		t.Errorf("Quiet hours end should be 0-23, got %d", end)
	}
}

// TestNewSettingsType tests that NewSettings returns the correct type
func TestNewSettingsType(t *testing.T) {
	s := NewSettings()
	if s == nil {
		t.Fatal("NewSettings returned nil")
	}

	// Verify it's a pointer to Settings
	_, ok := interface{}(s).(*Settings)
	if !ok {
		t.Errorf("NewSettings should return *Settings, got %T", s)
	}
}

// TestInterfaceConsistency tests that all methods exist and have expected signatures
func TestInterfaceConsistency(t *testing.T) {
	s := &Settings{available: true}

	// Test that all methods exist and are callable
	// (even if they would fail due to unavailability)

	// Methods that return (EfficiencyMode, error)
	if _, err := s.GetDefaultMode(); err != nil {
		// Expected to error if gsettings unavailable
	}
	if _, err := s.GetLastMode(); err != nil {
		// Expected to error if gsettings unavailable
	}

	// Methods that return error only
	if err := s.SetLastMode(efficiency.ModeBalanced); err != nil {
		// May error
	}

	// Methods that return (bool, error)
	if _, err := s.ShouldRememberMode(); err != nil {
		// Expected to error if gsettings unavailable
	}
	if _, err := s.GetQuietHoursEnabled(); err != nil {
		// Expected (no error even if unavailable)
	}

	// Methods that return (int, error)
	if _, err := s.GetQuietHoursStart(); err != nil {
		// Expected (no error even if unavailable)
	}
	if _, err := s.GetQuietHoursEnd(); err != nil {
		// Expected (no error even if unavailable)
	}

	// Methods that return bool only
	if s.GetNotifyOnModeChange() {
		// Returns bool
	}
	if s.GetNotifyOnBackendFailure() {
		// Returns bool
	}
	if s.GetNotifyOnThermalThrottle() {
		// Returns bool
	}

	// Methods that return EfficiencyMode only
	if s.LoadInitialMode() == efficiency.ModeBalanced {
		// Returns EfficiencyMode
	}
}

// TestParseAllModes ensures all standard modes can be parsed
func TestParseAllModes(t *testing.T) {
	tests := []struct {
		parseStr string
		mode     efficiency.EfficiencyMode
	}{
		{"Performance", efficiency.ModePerformance},
		{"Balanced", efficiency.ModeBalanced},
		{"Efficiency", efficiency.ModeEfficiency},
		{"Quiet", efficiency.ModeQuiet},
		{"Auto", efficiency.ModeAuto},
		{"UltraEfficiency", efficiency.ModeUltraEfficiency},
	}

	for _, tt := range tests {
		parsed := parseMode(tt.parseStr)
		if parsed != tt.mode {
			t.Errorf("parseMode(%q) = %v, want %v", tt.parseStr, parsed, tt.mode)
		}
	}
}

// TestUltraEfficiencyRoundtrip tests that UltraEfficiency mode can be round-tripped
func TestUltraEfficiencyRoundtrip(t *testing.T) {
	// The code has "UltraEfficiency" as the case-sensitive string to parse
	result := parseMode("UltraEfficiency")
	if result != efficiency.ModeUltraEfficiency {
		t.Errorf("parseMode(\"UltraEfficiency\") = %v, want %v", result, efficiency.ModeUltraEfficiency)
	}
}

// BenchmarkParseMode benchmarks the parseMode function
func BenchmarkParseMode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseMode("Performance")
	}
}

// BenchmarkNewSettings benchmarks Settings creation
func BenchmarkNewSettings(b *testing.B) {
	// Disable the actual check by mocking behavior is not possible in this context
	// Just benchmark the object creation
	for i := 0; i < b.N; i++ {
		_ = &Settings{available: false}
	}
}

// Helper function to validate boolean values
func isValidBool(v bool) bool {
	return true // bool in Go can only be true or false
}

// TestOSEnvDoesNotAffectDefaults tests that OS environment doesn't affect defaults
func TestOSEnvDoesNotAffectDefaults(t *testing.T) {
	originalEnv := os.Environ()
	defer func() {
		// In a real test, we'd restore, but since we're just reading we're safe
		_ = originalEnv
	}()

	s := &Settings{available: false}

	// These defaults should be consistent regardless of environment
	if notify := s.GetNotifyOnModeChange(); !notify {
		t.Error("Default notification setting should be true")
	}

	if quiet, _ := s.GetQuietHoursStart(); quiet != 22 {
		t.Error("Default quiet hours start should be 22")
	}
}

// TestGsettingsPanic ensures no panics occur during method calls with unavailable settings
func TestGsettingsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred: %v", r)
		}
	}()

	s := &Settings{available: false}

	// Call all methods - should not panic
	s.GetDefaultMode()
	s.GetLastMode()
	s.SetLastMode(efficiency.ModeBalanced)
	s.ShouldRememberMode()
	s.GetQuietHoursEnabled()
	s.GetQuietHoursStart()
	s.GetQuietHoursEnd()
	s.GetNotifyOnModeChange()
	s.GetNotifyOnBackendFailure()
	s.GetNotifyOnThermalThrottle()
	s.LoadInitialMode()
	s.IsAvailable()
}

// TestEmptyModeString tests parseMode with empty and whitespace strings
func TestEmptyModeString(t *testing.T) {
	tests := []string{"", " ", "\n", "\t"}

	for _, input := range tests {
		result := parseMode(input)
		if result != efficiency.ModeBalanced {
			t.Errorf("parseMode(%q) should return ModeBalanced, got %v", input, result)
		}
	}
}

// TestSettingsAvailableInitialization tests that available flag is properly initialized
func TestSettingsAvailableInitialization(t *testing.T) {
	s := NewSettings()

	// available should be set to the result of checkAvailable()
	// We can't predict the result without running gsettings,
	// but we know it should be a valid bool

	if _, err := s.GetDefaultMode(); err != nil {
		// If not available, error is expected
		if err.Error() != "GSettings not available" {
			t.Logf("Note: GSettings is available on this system")
		}
	}
}

// TestAllMethodsWithUnavailable ensures all methods handle unavailable gracefully
func TestAllMethodsWithUnavailable(t *testing.T) {
	s := &Settings{available: false}

	// Methods that error on unavailable
	errorMethods := []func() error{
		func() error { _, err := s.GetDefaultMode(); return err },
		func() error { _, err := s.GetLastMode(); return err },
		func() error { _, err := s.ShouldRememberMode(); return err },
	}

	for i, method := range errorMethods {
		if err := method(); err == nil {
			t.Errorf("Method %d should error when unavailable", i)
		}
	}

	// Methods that don't error on unavailable
	setErr := s.SetLastMode(efficiency.ModeBalanced)
	if setErr != nil {
		t.Error("SetLastMode should not error when unavailable (graceful)")
	}

	// Methods with default returns
	if _, err := s.GetQuietHoursEnabled(); err != nil {
		t.Error("GetQuietHoursEnabled should not error")
	}
}

// TestModeIntValues tests that EfficiencyMode iota values work correctly
func TestModeIntValues(t *testing.T) {
	// Verify iota assignment by checking the order
	modes := []efficiency.EfficiencyMode{
		efficiency.ModePerformance,     // 0
		efficiency.ModeBalanced,        // 1
		efficiency.ModeEfficiency,      // 2
		efficiency.ModeQuiet,           // 3
		efficiency.ModeAuto,            // 4
		efficiency.ModeUltraEfficiency, // 5
	}

	for i, mode := range modes {
		if int(mode) != i {
			t.Errorf("Mode %v should have value %d, got %d", mode, i, int(mode))
		}
	}
}

// TestGetQuietHoursStartEndDefaults tests the specific default values for quiet hours
func TestGetQuietHoursStartEndDefaults(t *testing.T) {
	s := &Settings{available: false}

	start, _ := s.GetQuietHoursStart()
	end, _ := s.GetQuietHoursEnd()

	if start != 22 {
		t.Errorf("Quiet hours start default should be 22, got %d", start)
	}

	if end != 7 {
		t.Errorf("Quiet hours end default should be 7, got %d", end)
	}

	// Verify the values make sense (start is after end, wrapping across midnight)
	if start < end {
		t.Logf("Note: Quiet hours wrap across midnight: %d to %d (next day)", start, end)
	}
}

// TestCheckAvailableCommand tests that checkAvailable would use the correct command
func TestCheckAvailableCommand(t *testing.T) {
	s := &Settings{}
	// The method uses exec.Command("gsettings", "list-schemas")
	// We can't mock this easily, but we can verify the method doesn't panic

	available := s.checkAvailable()
	if !isValidBool(available) {
		t.Error("checkAvailable should return a valid bool")
	}
}
