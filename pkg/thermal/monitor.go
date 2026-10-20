// Package thermal monitors device temperatures.
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

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// State represents thermal status of a device
type State struct {
	UpdatedAt   time.Time
	Temperature float64
	FanSpeed    int
	FanPercent  int
	PowerDraw   float64
	Utilization int
	Throttling  bool
}

// Monitor tracks thermal state of all backends
type Monitor struct {
	ctx            context.Context
	states         map[string]*State
	cancel         context.CancelFunc
	config         *Config
	updateInterval time.Duration
	mu             sync.RWMutex
}

// Config defines thermal limits
type Config struct {
	// Temperature thresholds (Celsius)
	TempWarning  float64 // Start preferring cooler backends
	TempCritical float64 // Stop using this backend
	TempShutdown float64 // Emergency - system protection

	// Fan speed thresholds (percentage)
	FanQuiet    int // Prefer this backend in quiet mode
	FanModerate int // Normal operation
	FanLoud     int // Avoid if possible

	// Cooling wait time
	CooldownTime time.Duration // Wait before retrying hot backend
}

// NewThermalMonitor creates a new thermal monitor
func NewThermalMonitor(config *Config, updateInterval time.Duration) *Monitor {
	if config == nil {
		config = &Config{
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

	tm := &Monitor{
		states:         make(map[string]*State),
		updateInterval: updateInterval,
		ctx:            ctx,
		cancel:         cancel,
		config:         config,
	}

	return tm
}

// Start begins monitoring
func (tm *Monitor) Start() {
	go tm.monitorLoop()
}

// Stop stops monitoring
func (tm *Monitor) Stop() {
	tm.cancel()
}

// monitorLoop continuously updates thermal state
func (tm *Monitor) monitorLoop() {
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
func (tm *Monitor) updateAll() {
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
func (tm *Monitor) getNVIDIAState() (*State, error) {
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

	temp, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse temperature: %w", err)
	}
	fanPercent, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse fan speed: %w", err)
	}
	power, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse power draw: %w", err)
	}
	util, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse utilization: %w", err)
	}

	throttling := false
	if len(parts) >= 5 {
		throttleReasons := strings.TrimSpace(parts[4])
		throttling = throttleReasons != "0x0000000000000000" && throttleReasons != "Active"
	}

	return &State{
		Temperature: temp,
		FanPercent:  fanPercent,
		PowerDraw:   power,
		Utilization: util,
		Throttling:  throttling,
		UpdatedAt:   time.Now(),
	}, nil
}

// getIntelGPUState reads Intel GPU thermal state
func (tm *Monitor) getIntelGPUState() (*State, error) {
	// Try intel_gpu_top for newer systems (disabled for now)
	// TODO: Parse intel_gpu_top JSON output when available

	// Fallback: Read from sysfs
	state := &State{
		UpdatedAt: time.Now(),
	}

	// Try reading Intel GPU temp from hwmon
	tempPaths := []string{
		"/sys/class/drm/card0/device/hwmon/hwmon*/temp1_input",
		"/sys/class/hwmon/hwmon*/temp1_label", // Check if it's GPU
	}

	for _, pattern := range tempPaths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob intel gpu temp: %w", err)
		}
		for _, path := range matches {
			if !strings.Contains(path, "temp1_input") {
				continue
			}
			temp, readErr := readCelsius(path)
			if readErr != nil {
				return nil, readErr
			}
			state.Temperature = temp
			break
		}
	}

	// Intel GPUs typically don't have user-accessible fan controls
	// They share system fans
	state.FanPercent = tm.getSystemFanSpeed()

	return state, nil
}

// getIntelNPUState reads NPU thermal state
func (tm *Monitor) getIntelNPUState() (*State, error) {
	// NPU is typically part of the SoC, low power
	// Read from sysfs or estimate based on system load

	state := &State{
		Temperature: 0,   // NPU temp often not separately reported
		FanPercent:  0,   // NPU doesn't have dedicated fan
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
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob npu temp: %w", err)
		}
		if len(matches) == 0 {
			continue
		}
		temp, readErr := readCelsius(matches[0])
		if readErr != nil {
			return nil, readErr
		}
		state.Temperature = temp
		break
	}

	return state, nil
}

// getCPUState reads CPU thermal state
func (tm *Monitor) getCPUState() (*State, error) {
	state := &State{
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
func (tm *Monitor) getCPUStateFallback() (*State, error) {
	state := &State{
		UpdatedAt: time.Now(),
	}

	// Read from thermal zones
	thermalPaths, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil {
		return nil, fmt.Errorf("glob thermal zones: %w", err)
	}

	maxTemp := 0.0
	for _, path := range thermalPaths {
		temp, readErr := readCelsius(path)
		if readErr != nil {
			logging.Warn("failed to read thermal zone", zap.String("path", path), zap.Error(readErr))
			continue
		}

		if temp > maxTemp && temp < 200 { // Sanity check
			maxTemp = temp
		}
	}

	state.Temperature = maxTemp
	state.FanPercent = tm.getSystemFanSpeed()

	return state, nil
}

// getSystemFanSpeed reads system fan speed (for integrated GPUs/NPU/CPU)
func (tm *Monitor) getSystemFanSpeed() int {
	// Try reading from hwmon
	fanPaths, err := filepath.Glob("/sys/class/hwmon/hwmon*/fan*_input")
	if err != nil {
		logging.Warn("failed to list fan sensors", zap.Error(err))
		return 0
	}

	for _, path := range fanPaths {
		rpmData, readErr := os.ReadFile(path)
		if readErr != nil {
			logging.Warn("failed to read fan sensor", zap.String("path", path), zap.Error(readErr))
			continue
		}

		rpm, convErr := strconv.Atoi(strings.TrimSpace(string(rpmData)))
		if convErr != nil {
			logging.Warn("failed to parse fan speed", zap.String("path", path), zap.Error(convErr))
			continue
		}
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

func readCelsius(path string) (float64, error) {
	tempData, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	tempMilliC, err := strconv.Atoi(strings.TrimSpace(string(tempData)))
	if err != nil {
		return 0, fmt.Errorf("parse temperature %s: %w", path, err)
	}
	return float64(tempMilliC) / 1000.0, nil
}

// GetState returns thermal state for hardware
func (tm *Monitor) GetState(hardware string) *State {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if state, ok := tm.states[hardware]; ok {
		return state
	}

	return nil
}

// GetAllStates returns all thermal states
func (tm *Monitor) GetAllStates() map[string]*State {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	states := make(map[string]*State)
	for k, v := range tm.states {
		states[k] = v
	}
	return states
}

// IsHealthy checks if hardware is thermally healthy
func (tm *Monitor) IsHealthy(hardware string) bool {
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
func (tm *Monitor) CanUse(hardware string) (bool, string) {
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
func (tm *Monitor) GetThermalPenalty(hardware string) float64 {
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
func (tm *Monitor) ShouldPreferQuiet() bool {
	// Check if any backend is running loud fans
	for _, state := range tm.GetAllStates() {
		if state.FanPercent > tm.config.FanModerate {
			return true
		}
	}
	return false
}

// GetCoolestBackend returns the hardware with lowest temperature
func (tm *Monitor) GetCoolestBackend(candidates []string) string {
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
func (tm *Monitor) GetConfig() *Config {
	return tm.config
}

// String returns human-readable thermal state
func (ts *State) String() string {
	return fmt.Sprintf("%.1f°C, Fan:%d%%, Power:%.1fW, Util:%d%%, Throttle:%v",
		ts.Temperature, ts.FanPercent, ts.PowerDraw, ts.Utilization, ts.Throttling)
}
