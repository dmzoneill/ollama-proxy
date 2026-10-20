// Package settings reads and writes desktop settings.
package settings

import (
	"errors"
	"fmt"
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/cmdexec"
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
	output, err := gsettingsOutput("list-schemas")
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
func (s *Settings) GetDefaultMode() (efficiency.Mode, error) {
	if !s.available {
		return efficiency.ModeBalanced, errors.New(strGSettingsNotAvailable)
	}

	output, err := gsettingsOutput("get", schemaID, "default-mode")
	if err != nil {
		return efficiency.ModeBalanced, err
	}

	// Output is quoted, so remove quotes
	modeStr := strings.Trim(string(output), "'\"\n ")

	return parseMode(modeStr), nil
}

// GetLastMode returns the last used efficiency mode
func (s *Settings) GetLastMode() (efficiency.Mode, error) {
	if !s.available {
		return efficiency.ModeBalanced, errors.New(strGSettingsNotAvailable)
	}

	output, err := gsettingsOutput("get", schemaID, "last-mode")
	if err != nil {
		return efficiency.ModeBalanced, err
	}

	modeStr := strings.Trim(string(output), "'\"\n ")
	return parseMode(modeStr), nil
}

// SetLastMode saves the last used efficiency mode
func (s *Settings) SetLastMode(mode efficiency.Mode) error {
	if !s.available {
		return nil // Gracefully ignore if not available
	}

	return gsettingsRun("set", schemaID, "last-mode", mode.String())
}

// ShouldRememberMode returns whether to remember the last used mode
func (s *Settings) ShouldRememberMode() (bool, error) {
	if !s.available {
		return false, errors.New(strGSettingsNotAvailable)
	}

	output, err := gsettingsOutput("get", schemaID, "remember-last-mode")
	if err != nil {
		return false, err
	}

	result := strings.TrimSpace(string(output))
	return result == strTrue, nil
}

// GetQuietHoursEnabled returns whether quiet hours are enabled for Auto mode
func (s *Settings) GetQuietHoursEnabled() (bool, error) {
	if !s.available {
		return true, nil // Default to true
	}

	output, err := gsettingsOutput("get", schemaID, "auto-quiet-hours-enabled")
	if err != nil {
		return true, err
	}

	result := strings.TrimSpace(string(output))
	return result == strTrue, nil
}

// GetQuietHoursStart returns the start hour for quiet hours (0-23)
func (s *Settings) GetQuietHoursStart() (int, error) {
	if !s.available {
		return 22, nil // Default
	}

	output, err := gsettingsOutput("get", schemaID, "auto-quiet-hours-start")
	if err != nil {
		return 22, err
	}

	hour, err := sscanfInt(string(output))
	if err != nil {
		return 22, err
	}
	return hour, nil
}

// GetQuietHoursEnd returns the end hour for quiet hours (0-23)
func (s *Settings) GetQuietHoursEnd() (int, error) {
	if !s.available {
		return 7, nil // Default
	}

	output, err := gsettingsOutput("get", schemaID, "auto-quiet-hours-end")
	if err != nil {
		return 7, err
	}

	hour, err := sscanfInt(string(output))
	if err != nil {
		return 7, err
	}
	return hour, nil
}

// GetNotifyOnModeChange returns whether to show notifications on mode change
func (s *Settings) GetNotifyOnModeChange() bool {
	if !s.available {
		return true
	}

	output, err := gsettingsOutput("get", schemaID, "notify-on-mode-change")
	if err != nil {
		return true
	}

	result := strings.TrimSpace(string(output))
	return result == strTrue
}

// GetNotifyOnBackendFailure returns whether to show notifications on backend failure
func (s *Settings) GetNotifyOnBackendFailure() bool {
	if !s.available {
		return true
	}

	output, err := gsettingsOutput("get", schemaID, "notify-on-backend-failure")
	if err != nil {
		return true
	}

	result := strings.TrimSpace(string(output))
	return result == strTrue
}

// GetNotifyOnThermalThrottle returns whether to show notifications on thermal throttling
func (s *Settings) GetNotifyOnThermalThrottle() bool {
	if !s.available {
		return false
	}

	output, err := gsettingsOutput("get", schemaID, "notify-on-thermal-throttle")
	if err != nil {
		return false
	}

	result := strings.TrimSpace(string(output))
	return result == strTrue
}

func gsettingsOutput(args ...string) ([]byte, error) {
	cmd, err := cmdexec.Command("gsettings", args...)
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

func gsettingsRun(args ...string) error {
	cmd, err := cmdexec.Command("gsettings", args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func sscanfInt(value string) (int, error) {
	var hour int
	if _, err := fmt.Sscanf(value, "%d", &hour); err != nil {
		return 0, err
	}
	return hour, nil
}

// parseMode converts a string to an Mode
func parseMode(modeStr string) efficiency.Mode {
	switch modeStr {
	case strPerformance:
		return efficiency.ModePerformance
	case strBalanced:
		return efficiency.ModeBalanced
	case strEfficiency:
		return efficiency.ModeEfficiency
	case strQuiet:
		return efficiency.ModeQuiet
	case strAuto:
		return efficiency.ModeAuto
	case strUltraEfficiency:
		return efficiency.ModeUltraEfficiency
	default:
		return efficiency.ModeBalanced
	}
}

// LoadInitialMode loads the initial efficiency mode based on settings
// Returns the mode to use on startup
func (s *Settings) LoadInitialMode() efficiency.Mode {
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
