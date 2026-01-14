package settings

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/efficiency"
)

const (
	schemaID = "ie.fio.ollamaproxy"
)

// Settings provides access to GSettings configuration
type Settings struct {
	available bool
}

// NewSettings creates a new settings instance
func NewSettings() *Settings {
	s := &Settings{}
	s.available = s.checkAvailable()
	return s
}

// checkAvailable checks if GSettings is available on the system
func (s *Settings) checkAvailable() bool {
	cmd := exec.Command("gsettings", "list-schemas")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if our schema is installed
	schemas := string(output)
	return strings.Contains(schemas, schemaID)
}

// IsAvailable returns whether GSettings is available
func (s *Settings) IsAvailable() bool {
	return s.available
}

// GetDefaultMode returns the configured default efficiency mode
func (s *Settings) GetDefaultMode() (efficiency.EfficiencyMode, error) {
	if !s.available {
		return efficiency.ModeBalanced, fmt.Errorf("GSettings not available")
	}

	cmd := exec.Command("gsettings", "get", schemaID, "default-mode")
	output, err := cmd.Output()
	if err != nil {
		return efficiency.ModeBalanced, err
	}

	// Output is quoted, so remove quotes
	modeStr := strings.Trim(string(output), "'\"\n ")

	return parseMode(modeStr), nil
}

// GetLastMode returns the last used efficiency mode
func (s *Settings) GetLastMode() (efficiency.EfficiencyMode, error) {
	if !s.available {
		return efficiency.ModeBalanced, fmt.Errorf("GSettings not available")
	}

	cmd := exec.Command("gsettings", "get", schemaID, "last-mode")
	output, err := cmd.Output()
	if err != nil {
		return efficiency.ModeBalanced, err
	}

	modeStr := strings.Trim(string(output), "'\"\n ")
	return parseMode(modeStr), nil
}

// SetLastMode saves the last used efficiency mode
func (s *Settings) SetLastMode(mode efficiency.EfficiencyMode) error {
	if !s.available {
		return nil // Gracefully ignore if not available
	}

	cmd := exec.Command("gsettings", "set", schemaID, "last-mode", mode.String())
	return cmd.Run()
}

// ShouldRememberMode returns whether to remember the last used mode
func (s *Settings) ShouldRememberMode() (bool, error) {
	if !s.available {
		return false, fmt.Errorf("GSettings not available")
	}

	cmd := exec.Command("gsettings", "get", schemaID, "remember-last-mode")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	result := strings.TrimSpace(string(output))
	return result == "true", nil
}

// GetQuietHoursEnabled returns whether quiet hours are enabled for Auto mode
func (s *Settings) GetQuietHoursEnabled() (bool, error) {
	if !s.available {
		return true, nil // Default to true
	}

	cmd := exec.Command("gsettings", "get", schemaID, "auto-quiet-hours-enabled")
	output, err := cmd.Output()
	if err != nil {
		return true, nil
	}

	result := strings.TrimSpace(string(output))
	return result == "true", nil
}

// GetQuietHoursStart returns the start hour for quiet hours (0-23)
func (s *Settings) GetQuietHoursStart() (int, error) {
	if !s.available {
		return 22, nil // Default
	}

	cmd := exec.Command("gsettings", "get", schemaID, "auto-quiet-hours-start")
	output, err := cmd.Output()
	if err != nil {
		return 22, nil
	}

	var hour int
	fmt.Sscanf(string(output), "%d", &hour)
	return hour, nil
}

// GetQuietHoursEnd returns the end hour for quiet hours (0-23)
func (s *Settings) GetQuietHoursEnd() (int, error) {
	if !s.available {
		return 7, nil // Default
	}

	cmd := exec.Command("gsettings", "get", schemaID, "auto-quiet-hours-end")
	output, err := cmd.Output()
	if err != nil {
		return 7, nil
	}

	var hour int
	fmt.Sscanf(string(output), "%d", &hour)
	return hour, nil
}

// GetNotifyOnModeChange returns whether to show notifications on mode change
func (s *Settings) GetNotifyOnModeChange() bool {
	if !s.available {
		return true
	}

	cmd := exec.Command("gsettings", "get", schemaID, "notify-on-mode-change")
	output, err := cmd.Output()
	if err != nil {
		return true
	}

	result := strings.TrimSpace(string(output))
	return result == "true"
}

// GetNotifyOnBackendFailure returns whether to show notifications on backend failure
func (s *Settings) GetNotifyOnBackendFailure() bool {
	if !s.available {
		return true
	}

	cmd := exec.Command("gsettings", "get", schemaID, "notify-on-backend-failure")
	output, err := cmd.Output()
	if err != nil {
		return true
	}

	result := strings.TrimSpace(string(output))
	return result == "true"
}

// GetNotifyOnThermalThrottle returns whether to show notifications on thermal throttling
func (s *Settings) GetNotifyOnThermalThrottle() bool {
	if !s.available {
		return false
	}

	cmd := exec.Command("gsettings", "get", schemaID, "notify-on-thermal-throttle")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	result := strings.TrimSpace(string(output))
	return result == "true"
}

// parseMode converts a string to an EfficiencyMode
func parseMode(modeStr string) efficiency.EfficiencyMode {
	switch modeStr {
	case "Performance":
		return efficiency.ModePerformance
	case "Balanced":
		return efficiency.ModeBalanced
	case "Efficiency":
		return efficiency.ModeEfficiency
	case "Quiet":
		return efficiency.ModeQuiet
	case "Auto":
		return efficiency.ModeAuto
	case "UltraEfficiency":
		return efficiency.ModeUltraEfficiency
	default:
		return efficiency.ModeBalanced
	}
}

// LoadInitialMode loads the initial efficiency mode based on settings
// Returns the mode to use on startup
func (s *Settings) LoadInitialMode() efficiency.EfficiencyMode {
	if !s.available {
		return efficiency.ModeBalanced
	}

	// Check if we should remember the last mode
	remember, err := s.ShouldRememberMode()
	if err == nil && remember {
		// Try to load last mode
		if lastMode, err := s.GetLastMode(); err == nil {
			return lastMode
		}
	}

	// Fall back to default mode
	if defaultMode, err := s.GetDefaultMode(); err == nil {
		return defaultMode
	}

	// Final fallback
	return efficiency.ModeBalanced
}
