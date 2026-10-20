package config

import (
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/device/virtual"
)

// Config structure matching config.yaml
type Config struct {
	VirtualDevices virtual.Config `yaml:"virtual_devices"`
	Pipelines      struct {
		ConfigFile string `yaml:"config_file"`
		Enabled    bool   `yaml:"enabled"`
	} `yaml:"pipelines"`
	Efficiency struct {
		DefaultMode string `yaml:"default_mode"`
		Enabled     bool   `yaml:"enabled"`
		DBusEnabled bool   `yaml:"dbus_enabled"`
	} `yaml:"efficiency"`
	Backends []struct {
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
	} `yaml:"backends"`
	Server struct {
		Auth struct {
			APIKeys map[string]struct {
				Name        string   `yaml:"name"`
				Permissions []string `yaml:"permissions"`
				Enabled     bool     `yaml:"enabled"`
			} `yaml:"api_keys"`
			Enabled bool `yaml:"enabled"`
		} `yaml:"auth"`
		Host string `yaml:"host"`
		TLS  struct {
			CertFile     string `yaml:"cert_file"`
			KeyFile      string `yaml:"key_file"`
			ClientCAFile string `yaml:"client_ca_file"`
			Enabled      bool   `yaml:"enabled"`
		} `yaml:"tls"`
		RateLimit struct {
			Enabled bool    `yaml:"enabled"`
			Rate    float64 `yaml:"rate"`
			Burst   int     `yaml:"burst"`
		} `yaml:"rate_limit"`
		GRPCPort int `yaml:"grpc_port"`
		HTTPPort int `yaml:"http_port"`
	} `yaml:"server"`
	Monitoring struct {
		LogLevel       string `yaml:"log_level"`
		PrometheusPort int    `yaml:"prometheus_port"`
		PprofPort      int    `yaml:"pprof_port"`
		Enabled        bool   `yaml:"enabled"`
		PprofEnabled   bool   `yaml:"pprof_enabled"`
	} `yaml:"monitoring"`
	Thermal struct {
		UpdateInterval string `yaml:"update_interval"`
		Temperature    struct {
			Warning  float64 `yaml:"warning"`
			Critical float64 `yaml:"critical"`
			Shutdown float64 `yaml:"shutdown"`
		} `yaml:"temperature"`
		Fan struct {
			Quiet    int `yaml:"quiet"`
			Moderate int `yaml:"moderate"`
			Loud     int `yaml:"loud"`
		} `yaml:"fan"`
		Enabled bool `yaml:"enabled"`
	} `yaml:"thermal"`
	Routing struct {
		DefaultBackend   string `yaml:"default_backend"`
		FallbackStrategy string `yaml:"fallback_strategy"`
		Forwarding       struct {
			EscalationPath       []string `yaml:"escalation_path"`
			MinConfidence        float64  `yaml:"min_confidence"`
			MaxRetries           int      `yaml:"max_retries"`
			Enabled              bool     `yaml:"enabled"`
			RespectThermalLimits bool     `yaml:"respect_thermal_limits"`
			ReturnBestAttempt    bool     `yaml:"return_best_attempt"`
		} `yaml:"forwarding"`
		Confidence struct {
			MinLengthChars int     `yaml:"min_length_chars"`
			MaxLengthChars int     `yaml:"max_length_chars"`
			LengthWeight   float64 `yaml:"length_weight"`
			PatternWeight  float64 `yaml:"pattern_weight"`
			ModelWeight    float64 `yaml:"model_weight"`
		} `yaml:"confidence"`
		PowerAware          bool `yaml:"power_aware"`
		AutoOptimizeLatency bool `yaml:"auto_optimize_latency"`
	} `yaml:"routing"`
	Devices struct {
		Enabled      bool `yaml:"enabled"`
		AutoDiscover bool `yaml:"auto_discover"`
	} `yaml:"devices"`
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	if err := validateServer(cfg); err != nil {
		return err
	}
	backendIDs, err := enabledBackendIDs(cfg)
	if err != nil {
		return err
	}
	if err := validateBackendConfigs(cfg); err != nil {
		return err
	}
	if err := validateRouting(cfg, backendIDs); err != nil {
		return err
	}
	if err := validateConfidence(cfg); err != nil {
		return err
	}
	if err := validateThermalConfig(cfg); err != nil {
		return err
	}
	if err := validateMonitoring(cfg); err != nil {
		return err
	}
	return validateEfficiency(cfg)
}

func validateServer(cfg *Config) error {
	if cfg.Server.GRPCPort < 1 || cfg.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d (must be 1-65535)", cfg.Server.GRPCPort)
	}
	if cfg.Server.HTTPPort < 1 || cfg.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d (must be 1-65535)", cfg.Server.HTTPPort)
	}
	if cfg.Server.GRPCPort == cfg.Server.HTTPPort {
		return fmt.Errorf("gRPC and HTTP ports cannot be the same: %d", cfg.Server.GRPCPort)
	}
	if !cfg.Server.TLS.Enabled {
		return nil
	}
	if cfg.Server.TLS.CertFile == "" {
		return fmt.Errorf("TLS enabled but cert_file not specified")
	}
	if cfg.Server.TLS.KeyFile == "" {
		return fmt.Errorf("TLS enabled but key_file not specified")
	}
	return nil
}

func enabledBackendIDs(cfg *Config) (map[string]bool, error) {
	enabledCount := 0
	backendIDs := make(map[string]bool)
	for _, backend := range cfg.Backends {
		if !backend.Enabled {
			continue
		}
		enabledCount++
		if backendIDs[backend.ID] {
			return nil, fmt.Errorf("duplicate backend ID: %s", backend.ID)
		}
		backendIDs[backend.ID] = true
	}
	if enabledCount == 0 {
		return nil, fmt.Errorf("no backends enabled in configuration")
	}
	return backendIDs, nil
}

func validateBackendConfigs(cfg *Config) error {
	for i := range cfg.Backends {
		if err := validateEnabledBackend(cfg, i); err != nil {
			return err
		}
	}
	return nil
}

func validateEnabledBackend(cfg *Config, idx int) error {
	backend := cfg.Backends[idx]
	if !backend.Enabled {
		return nil
	}
	if backend.ID == "" {
		return fmt.Errorf("backend missing ID")
	}
	if backend.Type == "" {
		return fmt.Errorf("backend %s missing type", backend.ID)
	}
	if err := validateBackendTypeFields(backend.ID, backend.Type, backend.Device, backend.ModelPath, backend.ModelName, backend.Endpoint); err != nil {
		return err
	}
	return validateBackendCharacteristics(backend.ID, backend.Characteristics.PowerWatts, backend.Characteristics.AvgLatencyMs, backend.Characteristics.Priority)
}

func validateBackendTypeFields(id, backendType, device, modelPath, modelName, endpoint string) error {
	switch backendType {
	case "openvino":
		return validateOpenVINOFields(id, device, modelPath, modelName)
	case "ollama", "openai", "anthropic":
		if endpoint == "" {
			return fmt.Errorf("backend %s missing endpoint", id)
		}
	default:
		// Unknown backend types are allowed so new backends can be added.
	}
	return nil
}

func validateOpenVINOFields(id, device, modelPath, modelName string) error {
	if device == "" {
		return fmt.Errorf("backend %s (type openvino) missing device field", id)
	}
	if modelPath == "" {
		return fmt.Errorf("backend %s (type openvino) missing model_path field", id)
	}
	if modelName == "" {
		return fmt.Errorf("backend %s (type openvino) missing model_name field", id)
	}
	return nil
}

func validateBackendCharacteristics(id string, powerWatts float64, avgLatencyMs int32, priority int) error {
	if powerWatts < 0 {
		return fmt.Errorf("backend %s has negative power_watts: %.2f", id, powerWatts)
	}
	if avgLatencyMs < 0 {
		return fmt.Errorf("backend %s has negative avg_latency_ms: %d", id, avgLatencyMs)
	}
	if priority < 0 {
		return fmt.Errorf("backend %s has negative priority: %d", id, priority)
	}
	return nil
}

func validateRouting(cfg *Config, backendIDs map[string]bool) error {
	if cfg.Routing.DefaultBackend != "" && !backendIDs[cfg.Routing.DefaultBackend] {
		return fmt.Errorf("default backend '%s' not found in enabled backends", cfg.Routing.DefaultBackend)
	}
	if !cfg.Routing.Forwarding.Enabled {
		return nil
	}
	if cfg.Routing.Forwarding.MinConfidence < 0 || cfg.Routing.Forwarding.MinConfidence > 1 {
		return fmt.Errorf("forwarding min_confidence %.2f out of range [0, 1]", cfg.Routing.Forwarding.MinConfidence)
	}
	if cfg.Routing.Forwarding.MaxRetries < 0 {
		return fmt.Errorf("forwarding max_retries cannot be negative: %d", cfg.Routing.Forwarding.MaxRetries)
	}
	for _, backendID := range cfg.Routing.Forwarding.EscalationPath {
		if !backendIDs[backendID] {
			return fmt.Errorf("escalation path backend '%s' not found in enabled backends", backendID)
		}
	}
	return nil
}

func validateConfidence(cfg *Config) error {
	if cfg.Routing.Confidence.LengthWeight < 0 || cfg.Routing.Confidence.LengthWeight > 1 {
		return fmt.Errorf("confidence length_weight %.2f out of range [0, 1]", cfg.Routing.Confidence.LengthWeight)
	}
	if cfg.Routing.Confidence.PatternWeight < 0 || cfg.Routing.Confidence.PatternWeight > 1 {
		return fmt.Errorf("confidence pattern_weight %.2f out of range [0, 1]", cfg.Routing.Confidence.PatternWeight)
	}
	if cfg.Routing.Confidence.ModelWeight < 0 || cfg.Routing.Confidence.ModelWeight > 1 {
		return fmt.Errorf("confidence model_weight %.2f out of range [0, 1]", cfg.Routing.Confidence.ModelWeight)
	}
	return nil
}

func validateThermalConfig(cfg *Config) error {
	if !cfg.Thermal.Enabled {
		return nil
	}
	if cfg.Thermal.Temperature.Warning >= cfg.Thermal.Temperature.Critical {
		return fmt.Errorf("thermal warning temp %.1f must be less than critical temp %.1f",
			cfg.Thermal.Temperature.Warning, cfg.Thermal.Temperature.Critical)
	}
	if cfg.Thermal.Temperature.Critical >= cfg.Thermal.Temperature.Shutdown {
		return fmt.Errorf("thermal critical temp %.1f must be less than shutdown temp %.1f",
			cfg.Thermal.Temperature.Critical, cfg.Thermal.Temperature.Shutdown)
	}
	if cfg.Thermal.Fan.Quiet > cfg.Thermal.Fan.Moderate {
		return fmt.Errorf("fan quiet %d cannot be greater than moderate %d",
			cfg.Thermal.Fan.Quiet, cfg.Thermal.Fan.Moderate)
	}
	if cfg.Thermal.Fan.Moderate > cfg.Thermal.Fan.Loud {
		return fmt.Errorf("fan moderate %d cannot be greater than loud %d",
			cfg.Thermal.Fan.Moderate, cfg.Thermal.Fan.Loud)
	}
	return nil
}

func validateMonitoring(cfg *Config) error {
	if !cfg.Monitoring.Enabled || cfg.Monitoring.PrometheusPort <= 0 {
		return nil
	}
	if cfg.Monitoring.PrometheusPort < 1 || cfg.Monitoring.PrometheusPort > 65535 {
		return fmt.Errorf("invalid Prometheus port: %d (must be 1-65535)", cfg.Monitoring.PrometheusPort)
	}
	if cfg.Monitoring.PrometheusPort == cfg.Server.GRPCPort || cfg.Monitoring.PrometheusPort == cfg.Server.HTTPPort {
		return fmt.Errorf("prometheus port %d conflicts with gRPC or HTTP port", cfg.Monitoring.PrometheusPort)
	}
	return nil
}

func validateEfficiency(cfg *Config) error {
	if !cfg.Efficiency.Enabled {
		return nil
	}
	validModes := map[string]bool{
		"Performance":     true,
		"Balanced":        true,
		"Efficiency":      true,
		"Quiet":           true,
		"Auto":            true,
		"UltraEfficiency": true,
	}
	if !validModes[cfg.Efficiency.DefaultMode] {
		return fmt.Errorf("invalid efficiency mode: %s (must be Performance, Balanced, Efficiency, Quiet, Auto, or UltraEfficiency)",
			cfg.Efficiency.DefaultMode)
	}
	return nil
}
