package thermal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ThermalState represents thermal status of a device
type ThermalState struct {
	Temperature float64   // Current temp in Celsius
	FanSpeed    int       // Fan speed in RPM (0 if not available)
	FanPercent  int       // Fan speed as percentage (0-100)
	PowerDraw   float64   // Current power draw in Watts
	Utilization int       // GPU/NPU utilization percentage
	Throttling  bool      // Is thermal throttling active?
	UpdatedAt   time.Time // Last update timestamp
}

// ThermalMonitor tracks thermal state of all backends
type ThermalMonitor struct {
	mu sync.RWMutex

	// Thermal state per hardware
	states map[string]*ThermalState // hardware ID -> state

	// Monitoring configuration
	updateInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc

	// Thresholds
	config *ThermalConfig
}

// ThermalConfig defines thermal limits
type ThermalConfig struct {
	// Temperature thresholds (Celsius)
	TempWarning    float64 // Start preferring cooler backends
	TempCritical   float64 // Stop using this backend
	TempShutdown   float64 // Emergency - system protection

	// Fan speed thresholds (percentage)
	FanQuiet       int // Prefer this backend in quiet mode
	FanModerate    int // Normal operation
	FanLoud        int // Avoid if possible

	// Cooling wait time
	CooldownTime   time.Duration // Wait before retrying hot backend
}

// NewThermalMonitor creates a new thermal monitor
func NewThermalMonitor(config *ThermalConfig, updateInterval time.Duration) *ThermalMonitor {
	if config == nil {
		config = &ThermalConfig{
			TempWarning:  70.0,
			TempCritical: 85.0,
			TempShutdown: 95.0,
			FanQuiet:     30,
			FanModerate:  60,
			FanLoud:      85,
			CooldownTime: 2 * time.Minute,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	tm := &ThermalMonitor{
		states:         make(map[string]*ThermalState),
		updateInterval: updateInterval,
		ctx:            ctx,
		cancel:         cancel,
		config:         config,
	}

	return tm
}

// Start begins monitoring
func (tm *ThermalMonitor) Start() {
	go tm.monitorLoop()
}

// Stop stops monitoring
func (tm *ThermalMonitor) Stop() {
	tm.cancel()
}

// monitorLoop continuously updates thermal state
func (tm *ThermalMonitor) monitorLoop() {
	ticker := time.NewTicker(tm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-ticker.C:
			tm.updateAll()
		}
	}
}

// updateAll updates thermal state for all hardware
func (tm *ThermalMonitor) updateAll() {
	// Update NVIDIA GPU
	if state, err := tm.getNVIDIAState(); err == nil {
		tm.mu.Lock()
		tm.states["nvidia"] = state
		tm.mu.Unlock()
	}

	// Update Intel GPU
	if state, err := tm.getIntelGPUState(); err == nil {
		tm.mu.Lock()
		tm.states["igpu"] = state
		tm.mu.Unlock()
	}

	// Update NPU (Intel)
	if state, err := tm.getIntelNPUState(); err == nil {
		tm.mu.Lock()
		tm.states["npu"] = state
		tm.mu.Unlock()
	}

	// Update CPU
	if state, err := tm.getCPUState(); err == nil {
		tm.mu.Lock()
		tm.states["cpu"] = state
		tm.mu.Unlock()
	}
}

// getNVIDIAState reads NVIDIA GPU thermal state via nvidia-smi
func (tm *ThermalMonitor) getNVIDIAState() (*ThermalState, error) {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=temperature.gpu,fan.speed,power.draw,utilization.gpu,clocks_throttle_reasons.active",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi failed: %w", err)
	}

	// Parse: temp, fan%, power, util%, throttle
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected nvidia-smi output")
	}

	temp, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	fanPercent, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	power, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	util, _ := strconv.Atoi(strings.TrimSpace(parts[3]))

	throttling := false
	if len(parts) >= 5 {
		throttleReasons := strings.TrimSpace(parts[4])
		throttling = throttleReasons != "0x0000000000000000" && throttleReasons != "Active"
	}

	return &ThermalState{
		Temperature: temp,
		FanPercent:  fanPercent,
		PowerDraw:   power,
		Utilization: util,
		Throttling:  throttling,
		UpdatedAt:   time.Now(),
	}, nil
}

// getIntelGPUState reads Intel GPU thermal state
func (tm *ThermalMonitor) getIntelGPUState() (*ThermalState, error) {
	// Try intel_gpu_top for newer systems (disabled for now)
	// TODO: Parse intel_gpu_top JSON output when available

	// Fallback: Read from sysfs
	state := &ThermalState{
		UpdatedAt: time.Now(),
	}

	// Try reading Intel GPU temp from hwmon
	tempPaths := []string{
		"/sys/class/drm/card0/device/hwmon/hwmon*/temp1_input",
		"/sys/class/hwmon/hwmon*/temp1_label", // Check if it's GPU
	}

	for _, pattern := range tempPaths {
		matches, _ := filepath.Glob(pattern)
		for _, path := range matches {
			if strings.Contains(path, "temp1_input") {
				tempData, err := os.ReadFile(path)
				if err == nil {
					tempMilliC, _ := strconv.Atoi(strings.TrimSpace(string(tempData)))
					state.Temperature = float64(tempMilliC) / 1000.0
					break
				}
			}
		}
	}

	// Intel GPUs typically don't have user-accessible fan controls
	// They share system fans
	state.FanPercent = tm.getSystemFanSpeed()

	return state, nil
}

// getIntelNPUState reads NPU thermal state
func (tm *ThermalMonitor) getIntelNPUState() (*ThermalState, error) {
	// NPU is typically part of the SoC, low power
	// Read from sysfs or estimate based on system load

	state := &ThermalState{
		Temperature: 0,  // NPU temp often not separately reported
		FanPercent:  0,  // NPU doesn't have dedicated fan
		PowerDraw:   3.0, // Estimate from config
		Utilization: 0,   // Would need NPU-specific tools
		Throttling:  false,
		UpdatedAt:   time.Now(),
	}

	// Try to read SoC temperature as proxy
	socTempPaths := []string{
		"/sys/class/thermal/thermal_zone*/temp",
	}

	for _, pattern := range socTempPaths {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			tempData, err := os.ReadFile(matches[0])
			if err == nil {
				tempMilliC, _ := strconv.Atoi(strings.TrimSpace(string(tempData)))
				state.Temperature = float64(tempMilliC) / 1000.0
				break
			}
		}
	}

	return state, nil
}

// getCPUState reads CPU thermal state
func (tm *ThermalMonitor) getCPUState() (*ThermalState, error) {
	state := &ThermalState{
		UpdatedAt: time.Now(),
	}

	// Read CPU temperature from sensors
	cmd := exec.Command("sensors", "-u")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to reading thermal zones
		return tm.getCPUStateFallback()
	}

	// Parse sensors output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	inCPU := false
	maxTemp := 0.0

	for scanner.Scan() {
		line := scanner.Text()

		// Detect CPU section
		if strings.Contains(line, "coretemp") || strings.Contains(line, "k10temp") {
			inCPU = true
			continue
		}

		if inCPU {
			// Look for temperature readings
			if strings.Contains(line, "_input:") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					temp, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
					if err == nil && temp > maxTemp {
						maxTemp = temp
					}
				}
			}

			// Exit CPU section on blank line
			if strings.TrimSpace(line) == "" {
				inCPU = false
			}
		}
	}

	state.Temperature = maxTemp
	state.FanPercent = tm.getSystemFanSpeed()

	return state, nil
}

// getCPUStateFallback reads from thermal zones directly
func (tm *ThermalMonitor) getCPUStateFallback() (*ThermalState, error) {
	state := &ThermalState{
		UpdatedAt: time.Now(),
	}

	// Read from thermal zones
	thermalPaths, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")

	maxTemp := 0.0
	for _, path := range thermalPaths {
		tempData, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		tempMilliC, _ := strconv.Atoi(strings.TrimSpace(string(tempData)))
		temp := float64(tempMilliC) / 1000.0

		if temp > maxTemp && temp < 200 { // Sanity check
			maxTemp = temp
		}
	}

	state.Temperature = maxTemp
	state.FanPercent = tm.getSystemFanSpeed()

	return state, nil
}

// getSystemFanSpeed reads system fan speed (for integrated GPUs/NPU/CPU)
func (tm *ThermalMonitor) getSystemFanSpeed() int {
	// Try reading from hwmon
	fanPaths, _ := filepath.Glob("/sys/class/hwmon/hwmon*/fan*_input")

	for _, path := range fanPaths {
		rpmData, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		rpm, _ := strconv.Atoi(strings.TrimSpace(string(rpmData)))
		if rpm > 0 {
			// Convert RPM to percentage (approximate)
			// Assume max fan speed ~5000 RPM
			percent := (rpm * 100) / 5000
			if percent > 100 {
				percent = 100
			}
			return percent
		}
	}

	return 0 // Fan speed unknown
}

// GetState returns thermal state for hardware
func (tm *ThermalMonitor) GetState(hardware string) *ThermalState {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if state, ok := tm.states[hardware]; ok {
		return state
	}

	return nil
}

// GetAllStates returns all thermal states
func (tm *ThermalMonitor) GetAllStates() map[string]*ThermalState {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	states := make(map[string]*ThermalState)
	for k, v := range tm.states {
		states[k] = v
	}
	return states
}

// IsHealthy checks if hardware is thermally healthy
func (tm *ThermalMonitor) IsHealthy(hardware string) bool {
	state := tm.GetState(hardware)
	if state == nil {
		return true // Unknown state, assume healthy
	}

	// Check temperature
	if state.Temperature >= tm.config.TempCritical {
		return false
	}

	// Check throttling
	if state.Throttling {
		return false
	}

	return true
}

// CanUse determines if backend can be used based on thermal state
func (tm *ThermalMonitor) CanUse(hardware string) (bool, string) {
	state := tm.GetState(hardware)
	if state == nil {
		return true, "" // Unknown, allow
	}

	// Critical temperature - absolutely not
	if state.Temperature >= tm.config.TempCritical {
		return false, fmt.Sprintf("temperature critical (%.1f°C >= %.1f°C)",
			state.Temperature, tm.config.TempCritical)
	}

	// Shutdown protection
	if state.Temperature >= tm.config.TempShutdown {
		return false, fmt.Sprintf("temperature shutdown threshold (%.1f°C)",
			state.Temperature)
	}

	// Throttling active
	if state.Throttling {
		return false, "thermal throttling active"
	}

	return true, ""
}

// GetThermalPenalty calculates routing penalty based on thermal state
// Higher penalty = less preferred
func (tm *ThermalMonitor) GetThermalPenalty(hardware string) float64 {
	state := tm.GetState(hardware)
	if state == nil {
		return 0.0 // No penalty if unknown
	}

	penalty := 0.0

	// Temperature penalty (exponential above warning)
	if state.Temperature > tm.config.TempWarning {
		tempOverage := state.Temperature - tm.config.TempWarning
		criticalRange := tm.config.TempCritical - tm.config.TempWarning
		tempRatio := tempOverage / criticalRange

		// Exponential penalty: 0 at warning, 1000 at critical
		penalty += tempRatio * tempRatio * 1000
	}

	// Fan noise penalty
	if state.FanPercent > tm.config.FanLoud {
		fanOverage := float64(state.FanPercent - tm.config.FanLoud)
		penalty += fanOverage * 5 // 5 points per % over loud threshold
	}

	// Throttling penalty (severe)
	if state.Throttling {
		penalty += 2000 // Heavy penalty for throttling
	}

	// High utilization penalty (backend is busy and hot)
	if state.Utilization > 80 {
		penalty += float64(state.Utilization-80) * 10
	}

	return penalty
}

// ShouldPreferQuiet checks if quiet mode should be active
func (tm *ThermalMonitor) ShouldPreferQuiet() bool {
	// Check if any backend is running loud fans
	for _, state := range tm.GetAllStates() {
		if state.FanPercent > tm.config.FanModerate {
			return true
		}
	}
	return false
}

// GetCoolestBackend returns the hardware with lowest temperature
func (tm *ThermalMonitor) GetCoolestBackend(candidates []string) string {
	coolest := ""
	lowestTemp := 999.0

	for _, hw := range candidates {
		state := tm.GetState(hw)
		if state == nil {
			continue
		}

		if state.Temperature < lowestTemp {
			lowestTemp = state.Temperature
			coolest = hw
		}
	}

	return coolest
}

// GetConfig returns the thermal configuration
func (tm *ThermalMonitor) GetConfig() *ThermalConfig {
	return tm.config
}

// String returns human-readable thermal state
func (ts *ThermalState) String() string {
	return fmt.Sprintf("%.1f°C, Fan:%d%%, Power:%.1fW, Util:%d%%, Throttle:%v",
		ts.Temperature, ts.FanPercent, ts.PowerDraw, ts.Utilization, ts.Throttling)
}
