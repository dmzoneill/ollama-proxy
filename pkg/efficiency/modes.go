package efficiency

import (
	"fmt"
	"sync"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// EfficiencyMode defines AI inference behavior profile
type EfficiencyMode int

const (
	// ModePerformance - Maximum speed, ignore power/thermal
	ModePerformance EfficiencyMode = iota

	// ModeBalanced - Smart routing based on complexity
	ModeBalanced

	// ModeEfficiency - Minimize power consumption
	ModeEfficiency

	// ModeQuiet - Minimize fan noise
	ModeQuiet

	// ModeAuto - Automatic based on battery, thermal, time
	ModeAuto

	// ModeUltraEfficiency - Maximum battery life (NPU only when possible)
	ModeUltraEfficiency
)

// String returns mode name
func (m EfficiencyMode) String() string {
	switch m {
	case ModePerformance:
		return "Performance"
	case ModeBalanced:
		return "Balanced"
	case ModeEfficiency:
		return "Efficiency"
	case ModeQuiet:
		return "Quiet"
	case ModeAuto:
		return "Auto"
	case ModeUltraEfficiency:
		return "Ultra Efficiency"
	default:
		return "Unknown"
	}
}

// ModeConfig defines behavior for each mode
type ModeConfig struct {
	// Preferred backends in priority order
	PreferredBackends []string

	// Maximum power draw (watts)
	MaxPowerWatts int

	// Maximum fan speed (percentage)
	MaxFanPercent int

	// Maximum temperature (Celsius)
	MaxTempCelsius float64

	// Allow overriding user's critical flag
	OverrideCriticalFlag bool

	// Throttle latency-critical requests
	ThrottleLatencyCritical bool

	// Prefer classification to save power
	UseClassification bool

	// Description for UI
	Description string

	// Icon name for UI
	Icon string
}

// GetModeConfig returns configuration for a mode
func GetModeConfig(mode EfficiencyMode) *ModeConfig {
	configs := map[EfficiencyMode]*ModeConfig{
		ModePerformance: {
			PreferredBackends:       []string{"ollama-nvidia", "ollama-igpu", "ollama-npu"},
			MaxPowerWatts:           999,
			MaxFanPercent:           100,
			MaxTempCelsius:          90.0,
			OverrideCriticalFlag:    false,
			ThrottleLatencyCritical: false,
			UseClassification:       false,
			Description:             "Maximum speed. Always use fastest backend available.",
			Icon:                    "🚀",
		},

		ModeBalanced: {
			PreferredBackends:       []string{"ollama-igpu", "ollama-nvidia", "ollama-npu"},
			MaxPowerWatts:           60,
			MaxFanPercent:           80,
			MaxTempCelsius:          85.0,
			OverrideCriticalFlag:    true,
			ThrottleLatencyCritical: false,
			UseClassification:       true,
			Description:             "Smart routing based on task complexity. Good balance of speed and efficiency.",
			Icon:                    "⚖️",
		},

		ModeEfficiency: {
			PreferredBackends:       []string{"ollama-npu", "ollama-igpu", "ollama-nvidia"},
			MaxPowerWatts:           15,
			MaxFanPercent:           60,
			MaxTempCelsius:          75.0,
			OverrideCriticalFlag:    true,
			ThrottleLatencyCritical: true,
			UseClassification:       true,
			Description:             "Minimize power consumption. Prefer NPU and Intel GPU.",
			Icon:                    "🔋",
		},

		ModeQuiet: {
			PreferredBackends:       []string{"ollama-npu", "ollama-igpu"},
			MaxPowerWatts:           15,
			MaxFanPercent:           40,
			MaxTempCelsius:          70.0,
			OverrideCriticalFlag:    true,
			ThrottleLatencyCritical: true,
			UseClassification:       true,
			Description:             "Minimize fan noise. Use silent backends only.",
			Icon:                    "🔇",
		},

		ModeAuto: {
			PreferredBackends:       []string{"ollama-igpu", "ollama-npu", "ollama-nvidia"},
			MaxPowerWatts:           0, // Determined automatically
			MaxFanPercent:           0, // Determined automatically
			MaxTempCelsius:          85.0,
			OverrideCriticalFlag:    true,
			ThrottleLatencyCritical: false,
			UseClassification:       true,
			Description:             "Automatically adjust based on battery, temperature, and time of day.",
			Icon:                    "🤖",
		},

		ModeUltraEfficiency: {
			PreferredBackends:       []string{"ollama-npu"},
			MaxPowerWatts:           5,
			MaxFanPercent:           30,
			MaxTempCelsius:          65.0,
			OverrideCriticalFlag:    true,
			ThrottleLatencyCritical: true,
			UseClassification:       true,
			Description:             "Maximum battery life. NPU only, accept slower responses.",
			Icon:                    "🪫",
		},
	}

	if cfg, ok := configs[mode]; ok {
		return cfg
	}

	return configs[ModeBalanced] // Default
}

// EfficiencyManager manages efficiency mode
type EfficiencyManager struct {
	mu   sync.RWMutex
	mode EfficiencyMode

	// System state for Auto mode
	batteryPercent int
	onBattery      bool
	avgTemp        float64
	avgFanSpeed    int
	quietHours     bool
}

// NewEfficiencyManager creates manager with default mode
func NewEfficiencyManager(defaultMode EfficiencyMode) *EfficiencyManager {
	return &EfficiencyManager{
		mode: defaultMode,
	}
}

// SetMode changes efficiency mode
func (em *EfficiencyManager) SetMode(mode EfficiencyMode) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.mode = mode
}

// GetMode returns current mode
func (em *EfficiencyManager) GetMode() EfficiencyMode {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.mode
}

// GetEffectiveMode returns the actual mode to use (resolves Auto)
func (em *EfficiencyManager) GetEffectiveMode() EfficiencyMode {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.mode != ModeAuto {
		return em.mode
	}

	// Auto mode - determine based on system state
	return em.determineAutoMode()
}

// determineAutoMode intelligently selects mode based on conditions
func (em *EfficiencyManager) determineAutoMode() EfficiencyMode {
	// Critical battery - Ultra Efficiency
	if em.onBattery && em.batteryPercent < 20 {
		return ModeUltraEfficiency
	}

	// Low battery - Efficiency
	if em.onBattery && em.batteryPercent < 50 {
		return ModeEfficiency
	}

	// Quiet hours - Quiet mode
	if em.quietHours {
		return ModeQuiet
	}

	// High temperatures - Efficiency (cool down)
	if em.avgTemp > 75.0 {
		return ModeEfficiency
	}

	// Loud fans - Quiet mode
	if em.avgFanSpeed > 70 {
		return ModeQuiet
	}

	// Battery but good level - Balanced
	if em.onBattery {
		return ModeBalanced
	}

	// AC power, cool, quiet - Performance allowed
	return ModePerformance
}

// UpdateSystemState updates state for Auto mode decisions
func (em *EfficiencyManager) UpdateSystemState(
	batteryPercent int,
	onBattery bool,
	avgTemp float64,
	avgFanSpeed int,
	quietHours bool,
) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.batteryPercent = batteryPercent
	em.onBattery = onBattery
	em.avgTemp = avgTemp
	em.avgFanSpeed = avgFanSpeed
	em.quietHours = quietHours
}

// ApplyModeToAnnotations modifies annotations based on current mode
func (em *EfficiencyManager) ApplyModeToAnnotations(annotations *backends.Annotations) {
	effectiveMode := em.GetEffectiveMode()
	config := GetModeConfig(effectiveMode)

	// Set power limit
	if config.MaxPowerWatts > 0 && config.MaxPowerWatts < 999 {
		if annotations.MaxPowerWatts == 0 || annotations.MaxPowerWatts > int32(config.MaxPowerWatts) {
			annotations.MaxPowerWatts = int32(config.MaxPowerWatts)
		}
	}

	// Override critical flag if configured
	if config.OverrideCriticalFlag && annotations.LatencyCritical {
		// Mode will determine if we respect this
		// Actually throttled in router
	}

	// Apply efficiency preference
	switch effectiveMode {
	case ModeEfficiency, ModeUltraEfficiency:
		annotations.PreferPowerEfficiency = true
		annotations.LatencyCritical = false

	case ModeQuiet:
		annotations.PreferPowerEfficiency = true
		annotations.LatencyCritical = false
		// Quiet mode handled by thermal routing

	case ModePerformance:
		// Performance mode - respect user's critical flag
		// No modifications needed

	case ModeBalanced:
		// Let smart routing decide
	}
}

// ShouldUseBackend checks if backend is allowed in current mode
func (em *EfficiencyManager) ShouldUseBackend(backendID string, temp float64, fanSpeed int) bool {
	effectiveMode := em.GetEffectiveMode()
	config := GetModeConfig(effectiveMode)

	// Check temperature limit
	if temp > config.MaxTempCelsius {
		return false
	}

	// Check fan speed limit
	if fanSpeed > config.MaxFanPercent {
		return false
	}

	return true
}

// GetPreferredBackends returns backend preference for current mode
func (em *EfficiencyManager) GetPreferredBackends() []string {
	effectiveMode := em.GetEffectiveMode()
	config := GetModeConfig(effectiveMode)
	return config.PreferredBackends
}

// GetModeDescription returns human-readable mode info
func (em *EfficiencyManager) GetModeDescription() string {
	mode := em.GetMode()
	effectiveMode := em.GetEffectiveMode()
	config := GetModeConfig(effectiveMode)

	if mode == ModeAuto {
		return fmt.Sprintf("Auto (%s): %s", effectiveMode.String(), config.Description)
	}

	return fmt.Sprintf("%s: %s", mode.String(), config.Description)
}

// SystemState represents current system state
type SystemState struct {
	BatteryPercent int
	OnBattery      bool
	AvgTemp        float64
	AvgFanSpeed    int
	QuietHours     bool
}

// GetSystemState returns current system state
func (em *EfficiencyManager) GetSystemState() SystemState {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return SystemState{
		BatteryPercent: em.batteryPercent,
		OnBattery:      em.onBattery,
		AvgTemp:        em.avgTemp,
		AvgFanSpeed:    em.avgFanSpeed,
		QuietHours:     em.quietHours,
	}
}

// AllModes returns all available modes for UI
func AllModes() []EfficiencyMode {
	return []EfficiencyMode{
		ModePerformance,
		ModeBalanced,
		ModeEfficiency,
		ModeQuiet,
		ModeAuto,
		ModeUltraEfficiency,
	}
}
