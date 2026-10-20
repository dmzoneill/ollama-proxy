package config

import (
	"strings"
	"testing"
)

// Helper function to create a valid baseline config for testing
func validConfig() *Config {
	cfg := &Config{}
	cfg.Server.GRPCPort = 50051
	cfg.Server.HTTPPort = 8080
	cfg.Server.Host = strLocalhost

	// Add at least one enabled backend
	cfg.Backends = []struct {
		ID              string `yaml:"id"`
		Type            string `yaml:"type"`
		Name            string `yaml:"name"`
		Hardware        string `yaml:"hardware"`
		Endpoint        string `yaml:"endpoint"`
		Device          string `yaml:"device"`
		ModelPath       string `yaml:"model_path"`
		ModelName       string `yaml:"model_name"`
		ModelCapability struct {
			SupportedModelPatterns []string `yaml:"supported_model_patterns"`
			PreferredModels        []string `yaml:"preferred_models"`
			ExcludedPatterns       []string `yaml:"excluded_patterns"`
			MaxModelSizeGB         int      `yaml:"max_model_size_gb"`
		} `yaml:"model_capability"`
		Characteristics struct {
			PowerWatts         float64 `yaml:"power_watts"`
			AvgLatencyMs       int32   `yaml:"avg_latency_ms"`
			MaxTokensPerSecond int32   `yaml:"max_tokens_per_second"`
			Priority           int     `yaml:"priority"`
		} `yaml:"characteristics"`
		Enabled bool `yaml:"enabled"`
	}{
		{
			ID:       strBackend1,
			Type:     "ollama",
			Name:     "Test Backend",
			Hardware: "cpu",
			Enabled:  true,
			Endpoint: "http://localhost:11434",
		},
	}
	cfg.Backends[0].Characteristics.PowerWatts = 10.0
	cfg.Backends[0].Characteristics.AvgLatencyMs = 100
	cfg.Backends[0].Characteristics.Priority = 5

	cfg.Routing.DefaultBackend = strBackend1
	cfg.Routing.Confidence.LengthWeight = 0.5
	cfg.Routing.Confidence.PatternWeight = 0.3
	cfg.Routing.Confidence.ModelWeight = 0.2

	return cfg
}

// TestValidateConfig_ValidConfig is part of the public API.
func TestValidateConfig_ValidConfig(t *testing.T) {
	cfg := validConfig()
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Valid config should not return error, got: %v", err)
	}
}

// TestValidateConfig_InvalidGRPCPort_TooLow is part of the public API.
func TestValidateConfig_InvalidGRPCPort_TooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Server.GRPCPort = 0

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for gRPC port < 1")
	}
	if !strings.Contains(err.Error(), "invalid gRPC port") {
		t.Errorf("Expected 'invalid gRPC port' in error, got: %v", err)
	}
}

// TestValidateConfig_InvalidGRPCPort_TooHigh is part of the public API.
func TestValidateConfig_InvalidGRPCPort_TooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Server.GRPCPort = 65536

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for gRPC port > 65535")
	}
	if !strings.Contains(err.Error(), "invalid gRPC port") {
		t.Errorf("Expected 'invalid gRPC port' in error, got: %v", err)
	}
}

// TestValidateConfig_InvalidHTTPPort_TooLow is part of the public API.
func TestValidateConfig_InvalidHTTPPort_TooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Server.HTTPPort = 0

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for HTTP port < 1")
	}
	if !strings.Contains(err.Error(), "invalid HTTP port") {
		t.Errorf("Expected 'invalid HTTP port' in error, got: %v", err)
	}
}

// TestValidateConfig_InvalidHTTPPort_TooHigh is part of the public API.
func TestValidateConfig_InvalidHTTPPort_TooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Server.HTTPPort = 70000

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for HTTP port > 65535")
	}
	if !strings.Contains(err.Error(), "invalid HTTP port") {
		t.Errorf("Expected 'invalid HTTP port' in error, got: %v", err)
	}
}

// TestValidateConfig_SameGRPCAndHTTPPort is part of the public API.
func TestValidateConfig_SameGRPCAndHTTPPort(t *testing.T) {
	cfg := validConfig()
	cfg.Server.GRPCPort = 8080
	cfg.Server.HTTPPort = 8080

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for same gRPC and HTTP port")
	}
	if !strings.Contains(err.Error(), "cannot be the same") {
		t.Errorf("Expected 'cannot be the same' in error, got: %v", err)
	}
}

// TestValidateConfig_TLSEnabled_MissingCertFile is part of the public API.
func TestValidateConfig_TLSEnabled_MissingCertFile(t *testing.T) {
	cfg := validConfig()
	cfg.Server.TLS.Enabled = true
	cfg.Server.TLS.CertFile = ""
	cfg.Server.TLS.KeyFile = "/path/to/key.pem"

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for TLS enabled without cert file")
	}
	if !strings.Contains(err.Error(), "cert_file not specified") {
		t.Errorf("Expected 'cert_file not specified' in error, got: %v", err)
	}
}

// TestValidateConfig_TLSEnabled_MissingKeyFile is part of the public API.
func TestValidateConfig_TLSEnabled_MissingKeyFile(t *testing.T) {
	cfg := validConfig()
	cfg.Server.TLS.Enabled = true
	cfg.Server.TLS.CertFile = "/path/to/cert.pem"
	cfg.Server.TLS.KeyFile = ""

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for TLS enabled without key file")
	}
	if !strings.Contains(err.Error(), "key_file not specified") {
		t.Errorf("Expected 'key_file not specified' in error, got: %v", err)
	}
}

// TestValidateConfig_TLSEnabled_ValidConfig is part of the public API.
func TestValidateConfig_TLSEnabled_ValidConfig(t *testing.T) {
	cfg := validConfig()
	cfg.Server.TLS.Enabled = true
	cfg.Server.TLS.CertFile = "/path/to/cert.pem"
	cfg.Server.TLS.KeyFile = "/path/to/key.pem"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Valid TLS config should not error, got: %v", err)
	}
}

// TestValidateConfig_NoBackendsEnabled is part of the public API.
func TestValidateConfig_NoBackendsEnabled(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].Enabled = false

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for no enabled backends")
	}
	if !strings.Contains(err.Error(), "no backends enabled") {
		t.Errorf("Expected 'no backends enabled' in error, got: %v", err)
	}
}

// TestValidateConfig_DuplicateBackendID is part of the public API.
func TestValidateConfig_DuplicateBackendID(t *testing.T) {
	cfg := validConfig()

	// Add a second backend with the same ID
	backend2 := cfg.Backends[0]
	backend2.ID = strBackend1 // Same as first
	backend2.Endpoint = "http://localhost:11435"
	cfg.Backends = append(cfg.Backends, backend2)

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for duplicate backend ID")
	}
	if !strings.Contains(err.Error(), "duplicate backend ID") {
		t.Errorf("Expected 'duplicate backend ID' in error, got: %v", err)
	}
}

// TestValidateConfig_BackendMissingID is part of the public API.
func TestValidateConfig_BackendMissingID(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].ID = ""

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for backend missing ID")
	}
	if !strings.Contains(err.Error(), "backend missing ID") {
		t.Errorf("Expected 'backend missing ID' in error, got: %v", err)
	}
}

// TestValidateConfig_BackendMissingType is part of the public API.
func TestValidateConfig_BackendMissingType(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].Type = ""

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for backend missing type")
	}
	if !strings.Contains(err.Error(), "missing type") {
		t.Errorf("Expected 'missing type' in error, got: %v", err)
	}
}

// TestValidateConfig_BackendMissingEndpoint is part of the public API.
func TestValidateConfig_BackendMissingEndpoint(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].Endpoint = ""

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for backend missing endpoint")
	}
	if !strings.Contains(err.Error(), "missing endpoint") {
		t.Errorf("Expected 'missing endpoint' in error, got: %v", err)
	}
}

// TestValidateConfig_BackendNegativePowerWatts is part of the public API.
func TestValidateConfig_BackendNegativePowerWatts(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].Characteristics.PowerWatts = -5.0

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for negative power_watts")
	}
	if !strings.Contains(err.Error(), "negative power_watts") {
		t.Errorf("Expected 'negative power_watts' in error, got: %v", err)
	}
}

// TestValidateConfig_BackendNegativeLatency is part of the public API.
func TestValidateConfig_BackendNegativeLatency(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].Characteristics.AvgLatencyMs = -100

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for negative avg_latency_ms")
	}
	if !strings.Contains(err.Error(), "negative avg_latency_ms") {
		t.Errorf("Expected 'negative avg_latency_ms' in error, got: %v", err)
	}
}

// TestValidateConfig_BackendNegativePriority is part of the public API.
func TestValidateConfig_BackendNegativePriority(t *testing.T) {
	cfg := validConfig()
	cfg.Backends[0].Characteristics.Priority = -1

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for negative priority")
	}
	if !strings.Contains(err.Error(), "negative priority") {
		t.Errorf("Expected 'negative priority' in error, got: %v", err)
	}
}

// TestValidateConfig_DefaultBackendNotInEnabled is part of the public API.
func TestValidateConfig_DefaultBackendNotInEnabled(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.DefaultBackend = "nonexistent-backend"

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for default backend not in enabled backends")
	}
	if !strings.Contains(err.Error(), "not found in enabled backends") {
		t.Errorf("Expected 'not found in enabled backends' in error, got: %v", err)
	}
}

// TestValidateConfig_ForwardingMinConfidenceTooLow is part of the public API.
func TestValidateConfig_ForwardingMinConfidenceTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Forwarding.Enabled = true
	cfg.Routing.Forwarding.MinConfidence = -0.5

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for min_confidence < 0")
	}
	if !strings.Contains(err.Error(), "min_confidence") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'min_confidence out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_ForwardingMinConfidenceTooHigh is part of the public API.
func TestValidateConfig_ForwardingMinConfidenceTooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Forwarding.Enabled = true
	cfg.Routing.Forwarding.MinConfidence = 1.5

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for min_confidence > 1")
	}
	if !strings.Contains(err.Error(), "min_confidence") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'min_confidence out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_ForwardingNegativeRetries is part of the public API.
func TestValidateConfig_ForwardingNegativeRetries(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Forwarding.Enabled = true
	cfg.Routing.Forwarding.MaxRetries = -5

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for negative max_retries")
	}
	if !strings.Contains(err.Error(), "max_retries cannot be negative") {
		t.Errorf("Expected 'max_retries cannot be negative' in error, got: %v", err)
	}
}

// TestValidateConfig_ForwardingValidConfig is part of the public API.
func TestValidateConfig_ForwardingValidConfig(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Forwarding.Enabled = true
	cfg.Routing.Forwarding.MinConfidence = 0.7
	cfg.Routing.Forwarding.MaxRetries = 3
	cfg.Routing.Forwarding.EscalationPath = []string{strBackend1}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Valid forwarding config should not error, got: %v", err)
	}
}

// TestValidateConfig_EscalationPathInvalidBackend is part of the public API.
func TestValidateConfig_EscalationPathInvalidBackend(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Forwarding.Enabled = true
	cfg.Routing.Forwarding.EscalationPath = []string{strBackend1, "nonexistent-backend"}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for escalation path backend not found")
	}
	if !strings.Contains(err.Error(), "escalation path backend") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'escalation path backend not found' in error, got: %v", err)
	}
}

// TestValidateConfig_LengthWeightOutOfRange_TooLow is part of the public API.
func TestValidateConfig_LengthWeightOutOfRange_TooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Confidence.LengthWeight = -0.1

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for length_weight < 0")
	}
	if !strings.Contains(err.Error(), "length_weight") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'length_weight out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_LengthWeightOutOfRange_TooHigh is part of the public API.
func TestValidateConfig_LengthWeightOutOfRange_TooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Confidence.LengthWeight = 1.5

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for length_weight > 1")
	}
	if !strings.Contains(err.Error(), "length_weight") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'length_weight out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_PatternWeightOutOfRange_TooLow is part of the public API.
func TestValidateConfig_PatternWeightOutOfRange_TooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Confidence.PatternWeight = -0.2

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for pattern_weight < 0")
	}
	if !strings.Contains(err.Error(), "pattern_weight") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'pattern_weight out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_PatternWeightOutOfRange_TooHigh is part of the public API.
func TestValidateConfig_PatternWeightOutOfRange_TooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Confidence.PatternWeight = 2.0

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for pattern_weight > 1")
	}
	if !strings.Contains(err.Error(), "pattern_weight") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'pattern_weight out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_ModelWeightOutOfRange_TooLow is part of the public API.
func TestValidateConfig_ModelWeightOutOfRange_TooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Confidence.ModelWeight = -0.5

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for model_weight < 0")
	}
	if !strings.Contains(err.Error(), "model_weight") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'model_weight out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_ModelWeightOutOfRange_TooHigh is part of the public API.
func TestValidateConfig_ModelWeightOutOfRange_TooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Confidence.ModelWeight = 1.1

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for model_weight > 1")
	}
	if !strings.Contains(err.Error(), "model_weight") && !strings.Contains(err.Error(), "out of range") {
		t.Errorf("Expected 'model_weight out of range' in error, got: %v", err)
	}
}

// TestValidateConfig_ThermalWarningNotLessThanCritical is part of the public API.
func TestValidateConfig_ThermalWarningNotLessThanCritical(t *testing.T) {
	cfg := validConfig()
	cfg.Thermal.Enabled = true
	cfg.Thermal.Temperature.Warning = 85.0
	cfg.Thermal.Temperature.Critical = 80.0
	cfg.Thermal.Temperature.Shutdown = 95.0

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for warning >= critical")
	}
	if !strings.Contains(err.Error(), "warning temp") && !strings.Contains(err.Error(), "must be less than critical") {
		t.Errorf("Expected 'warning temp must be less than critical' in error, got: %v", err)
	}
}

// TestValidateConfig_ThermalCriticalNotLessThanShutdown is part of the public API.
func TestValidateConfig_ThermalCriticalNotLessThanShutdown(t *testing.T) {
	cfg := validConfig()
	cfg.Thermal.Enabled = true
	cfg.Thermal.Temperature.Warning = 70.0
	cfg.Thermal.Temperature.Critical = 95.0
	cfg.Thermal.Temperature.Shutdown = 85.0

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for critical >= shutdown")
	}
	if !strings.Contains(err.Error(), "critical temp") && !strings.Contains(err.Error(), "must be less than shutdown") {
		t.Errorf("Expected 'critical temp must be less than shutdown' in error, got: %v", err)
	}
}

// TestValidateConfig_ThermalValidConfig is part of the public API.
func TestValidateConfig_ThermalValidConfig(t *testing.T) {
	cfg := validConfig()
	cfg.Thermal.Enabled = true
	cfg.Thermal.Temperature.Warning = 70.0
	cfg.Thermal.Temperature.Critical = 85.0
	cfg.Thermal.Temperature.Shutdown = 95.0
	cfg.Thermal.Fan.Quiet = 30
	cfg.Thermal.Fan.Moderate = 60
	cfg.Thermal.Fan.Loud = 80

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Valid thermal config should not error, got: %v", err)
	}
}

// TestValidateConfig_FanQuietGreaterThanModerate is part of the public API.
func TestValidateConfig_FanQuietGreaterThanModerate(t *testing.T) {
	cfg := validConfig()
	cfg.Thermal.Enabled = true
	cfg.Thermal.Temperature.Warning = 70.0
	cfg.Thermal.Temperature.Critical = 85.0
	cfg.Thermal.Temperature.Shutdown = 95.0
	cfg.Thermal.Fan.Quiet = 70
	cfg.Thermal.Fan.Moderate = 50
	cfg.Thermal.Fan.Loud = 80

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for fan quiet > moderate")
	}
	if !strings.Contains(err.Error(), "fan quiet") && !strings.Contains(err.Error(), "cannot be greater than moderate") {
		t.Errorf("Expected 'fan quiet cannot be greater than moderate' in error, got: %v", err)
	}
}

// TestValidateConfig_FanModerateGreaterThanLoud is part of the public API.
func TestValidateConfig_FanModerateGreaterThanLoud(t *testing.T) {
	cfg := validConfig()
	cfg.Thermal.Enabled = true
	cfg.Thermal.Temperature.Warning = 70.0
	cfg.Thermal.Temperature.Critical = 85.0
	cfg.Thermal.Temperature.Shutdown = 95.0
	cfg.Thermal.Fan.Quiet = 30
	cfg.Thermal.Fan.Moderate = 90
	cfg.Thermal.Fan.Loud = 70

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for fan moderate > loud")
	}
	if !strings.Contains(err.Error(), "fan moderate") && !strings.Contains(err.Error(), "cannot be greater than loud") {
		t.Errorf("Expected 'fan moderate cannot be greater than loud' in error, got: %v", err)
	}
}

// TestValidateConfig_PrometheusPortZero_Allowed is part of the public API.
func TestValidateConfig_PrometheusPortZero_Allowed(t *testing.T) {
	cfg := validConfig()
	cfg.Monitoring.Enabled = true
	cfg.Monitoring.PrometheusPort = 0

	// PrometheusPort = 0 means disabled, should be allowed
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("PrometheusPort = 0 should be allowed (disabled), got: %v", err)
	}
}

// TestValidateConfig_PrometheusPortInvalid_TooHigh is part of the public API.
func TestValidateConfig_PrometheusPortInvalid_TooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Monitoring.Enabled = true
	cfg.Monitoring.PrometheusPort = 70000

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for Prometheus port > 65535")
	}
	if !strings.Contains(err.Error(), "invalid Prometheus port") {
		t.Errorf("Expected 'invalid Prometheus port' in error, got: %v", err)
	}
}

// TestValidateConfig_PrometheusPortConflictsWithGRPC is part of the public API.
func TestValidateConfig_PrometheusPortConflictsWithGRPC(t *testing.T) {
	cfg := validConfig()
	cfg.Monitoring.Enabled = true
	cfg.Monitoring.PrometheusPort = cfg.Server.GRPCPort

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for Prometheus port conflict with gRPC")
	}
	if !strings.Contains(err.Error(), "conflicts with gRPC or HTTP port") {
		t.Errorf("Expected 'conflicts with gRPC or HTTP port' in error, got: %v", err)
	}
}

// TestValidateConfig_PrometheusPortConflictsWithHTTP is part of the public API.
func TestValidateConfig_PrometheusPortConflictsWithHTTP(t *testing.T) {
	cfg := validConfig()
	cfg.Monitoring.Enabled = true
	cfg.Monitoring.PrometheusPort = cfg.Server.HTTPPort

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for Prometheus port conflict with HTTP")
	}
	if !strings.Contains(err.Error(), "conflicts with gRPC or HTTP port") {
		t.Errorf("Expected 'conflicts with gRPC or HTTP port' in error, got: %v", err)
	}
}

// TestValidateConfig_PrometheusPortValid is part of the public API.
func TestValidateConfig_PrometheusPortValid(t *testing.T) {
	cfg := validConfig()
	cfg.Monitoring.Enabled = true
	cfg.Monitoring.PrometheusPort = 9090

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Valid Prometheus port should not error, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyModeInvalid is part of the public API.
func TestValidateConfig_EfficiencyModeInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "InvalidMode"

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid efficiency mode")
	}
	if !strings.Contains(err.Error(), "invalid efficiency mode") {
		t.Errorf("Expected 'invalid efficiency mode' in error, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyMode_Performance is part of the public API.
func TestValidateConfig_EfficiencyMode_Performance(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "Performance"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Performance mode should be valid, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyMode_Balanced is part of the public API.
func TestValidateConfig_EfficiencyMode_Balanced(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "Balanced"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Balanced mode should be valid, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyMode_Efficiency is part of the public API.
func TestValidateConfig_EfficiencyMode_Efficiency(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "Efficiency"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Efficiency mode should be valid, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyMode_Quiet is part of the public API.
func TestValidateConfig_EfficiencyMode_Quiet(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "Quiet"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Quiet mode should be valid, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyMode_Auto is part of the public API.
func TestValidateConfig_EfficiencyMode_Auto(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "Auto"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Auto mode should be valid, got: %v", err)
	}
}

// TestValidateConfig_EfficiencyMode_UltraEfficiency is part of the public API.
func TestValidateConfig_EfficiencyMode_UltraEfficiency(t *testing.T) {
	cfg := validConfig()
	cfg.Efficiency.Enabled = true
	cfg.Efficiency.DefaultMode = "UltraEfficiency"

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("UltraEfficiency mode should be valid, got: %v", err)
	}
}

// TestValidateConfig_MultipleBackends is part of the public API.
func TestValidateConfig_MultipleBackends(t *testing.T) {
	cfg := validConfig()

	// Add a second backend
	backend2 := cfg.Backends[0]
	backend2.ID = "backend-2"
	backend2.Endpoint = "http://localhost:11435"
	cfg.Backends = append(cfg.Backends, backend2)

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Multiple valid backends should not error, got: %v", err)
	}
}

// TestValidateConfig_DisabledBackendNotValidated is part of the public API.
func TestValidateConfig_DisabledBackendNotValidated(t *testing.T) {
	cfg := validConfig()

	// Add a disabled backend with invalid config
	backend2 := cfg.Backends[0]
	backend2.ID = "" // Invalid - missing ID
	backend2.Enabled = false
	cfg.Backends = append(cfg.Backends, backend2)

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Disabled backend with invalid config should not be validated, got: %v", err)
	}
}

// TestValidateConfig_EmptyDefaultBackend is part of the public API.
func TestValidateConfig_EmptyDefaultBackend(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.DefaultBackend = ""

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Empty default backend should be allowed, got: %v", err)
	}
}
