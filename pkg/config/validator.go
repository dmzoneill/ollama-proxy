package config

import (
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/device/virtual"
)

// Config structure matching config.yaml
type Config struct {
	Server struct {
		GRPCPort int    `yaml:"grpc_port"`
		HTTPPort int    `yaml:"http_port"`
		Host     string `yaml:"host"`
		TLS      struct {
			Enabled      bool   `yaml:"enabled"`
			CertFile     string `yaml:"cert_file"`
			KeyFile      string `yaml:"key_file"`
			ClientCAFile string `yaml:"client_ca_file"`
		} `yaml:"tls"`
		Auth struct {
			Enabled bool `yaml:"enabled"`
			APIKeys map[string]struct {
				Name        string   `yaml:"name"`
				Permissions []string `yaml:"permissions"`
				Enabled     bool     `yaml:"enabled"`
			} `yaml:"api_keys"`
		} `yaml:"auth"`
		RateLimit struct {
			Enabled bool    `yaml:"enabled"`
			Rate    float64 `yaml:"rate"`
			Burst   int     `yaml:"burst"`
		} `yaml:"rate_limit"`
	} `yaml:"server"`

	Backends []struct {
		ID       string `yaml:"id"`
		Type     string `yaml:"type"`
		Name     string `yaml:"name"`
		Hardware string `yaml:"hardware"`
		Enabled  bool   `yaml:"enabled"`
		Endpoint string `yaml:"endpoint"`

		// OpenVINO-specific fields
		Device    string `yaml:"device"`     // "CPU", "GPU", "NPU" for OpenVINO backends
		ModelPath string `yaml:"model_path"` // Path to OpenVINO model directory
		ModelName string `yaml:"model_name"` // Model name/identifier

		Characteristics struct {
			PowerWatts         float64 `yaml:"power_watts"`
			AvgLatencyMs       int32   `yaml:"avg_latency_ms"`
			MaxTokensPerSecond int32   `yaml:"max_tokens_per_second"`
			Priority           int     `yaml:"priority"`
		} `yaml:"characteristics"`
		ModelCapability struct {
			MaxModelSizeGB         int      `yaml:"max_model_size_gb"`
			SupportedModelPatterns []string `yaml:"supported_model_patterns"`
			PreferredModels        []string `yaml:"preferred_models"`
			ExcludedPatterns       []string `yaml:"excluded_patterns"`
		} `yaml:"model_capability"`
	} `yaml:"backends"`

	Routing struct {
		DefaultBackend      string `yaml:"default_backend"`
		PowerAware          bool   `yaml:"power_aware"`
		FallbackStrategy    string `yaml:"fallback_strategy"`
		AutoOptimizeLatency bool   `yaml:"auto_optimize_latency"`
		Forwarding struct {
			Enabled              bool     `yaml:"enabled"`
			MinConfidence        float64  `yaml:"min_confidence"`
			MaxRetries           int      `yaml:"max_retries"`
			EscalationPath       []string `yaml:"escalation_path"`
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
	} `yaml:"routing"`

	Monitoring struct {
		Enabled        bool   `yaml:"enabled"`
		PrometheusPort int    `yaml:"prometheus_port"`
		LogLevel       string `yaml:"log_level"`
		PprofEnabled   bool   `yaml:"pprof_enabled"`
		PprofPort      int    `yaml:"pprof_port"`
	} `yaml:"monitoring"`

	Thermal struct {
		Enabled        bool   `yaml:"enabled"`
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
	} `yaml:"thermal"`

	Efficiency struct {
		Enabled     bool   `yaml:"enabled"`
		DefaultMode string `yaml:"default_mode"`
		DBusEnabled bool   `yaml:"dbus_enabled"`
	} `yaml:"efficiency"`

	Pipelines struct {
		Enabled    bool   `yaml:"enabled"`
		ConfigFile string `yaml:"config_file"`
	} `yaml:"pipelines"`

	Devices struct {
		Enabled      bool `yaml:"enabled"`
		AutoDiscover bool `yaml:"auto_discover"`
	} `yaml:"devices"`

	// Virtual device configuration
	VirtualDevices virtual.Config `yaml:"virtual_devices"`
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	// Validate server ports
	if cfg.Server.GRPCPort < 1 || cfg.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d (must be 1-65535)", cfg.Server.GRPCPort)
	}
	if cfg.Server.HTTPPort < 1 || cfg.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d (must be 1-65535)", cfg.Server.HTTPPort)
	}
	if cfg.Server.GRPCPort == cfg.Server.HTTPPort {
		return fmt.Errorf("gRPC and HTTP ports cannot be the same: %d", cfg.Server.GRPCPort)
	}

	// Validate TLS configuration
	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" {
			return fmt.Errorf("TLS enabled but cert_file not specified")
		}
		if cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but key_file not specified")
		}
	}

	// Validate at least one backend enabled
	enabledCount := 0
	backendIDs := make(map[string]bool)
	for _, backend := range cfg.Backends {
		if backend.Enabled {
			enabledCount++

			// Check for duplicate IDs
			if backendIDs[backend.ID] {
				return fmt.Errorf("duplicate backend ID: %s", backend.ID)
			}
			backendIDs[backend.ID] = true
		}
	}
	if enabledCount == 0 {
		return fmt.Errorf("no backends enabled in configuration")
	}

	// Validate backend configurations
	for _, backend := range cfg.Backends {
		if backend.Enabled {
			if backend.ID == "" {
				return fmt.Errorf("backend missing ID")
			}
			if backend.Type == "" {
				return fmt.Errorf("backend %s missing type", backend.ID)
			}

			// Type-specific validation
			switch backend.Type {
			case "openvino":
				// OpenVINO backends require Device, ModelPath, and ModelName
				if backend.Device == "" {
					return fmt.Errorf("backend %s (type openvino) missing device field", backend.ID)
				}
				if backend.ModelPath == "" {
					return fmt.Errorf("backend %s (type openvino) missing model_path field", backend.ID)
				}
				if backend.ModelName == "" {
					return fmt.Errorf("backend %s (type openvino) missing model_name field", backend.ID)
				}
			case "ollama", "openai", "anthropic":
				// HTTP-based backends require endpoint
				if backend.Endpoint == "" {
					return fmt.Errorf("backend %s missing endpoint", backend.ID)
				}
			default:
				// Unknown backend type - warn but don't fail
				// This allows for future extensibility
			}

			if backend.Characteristics.PowerWatts < 0 {
				return fmt.Errorf("backend %s has negative power_watts: %.2f",
					backend.ID, backend.Characteristics.PowerWatts)
			}
			if backend.Characteristics.AvgLatencyMs < 0 {
				return fmt.Errorf("backend %s has negative avg_latency_ms: %d",
					backend.ID, backend.Characteristics.AvgLatencyMs)
			}
			if backend.Characteristics.Priority < 0 {
				return fmt.Errorf("backend %s has negative priority: %d",
					backend.ID, backend.Characteristics.Priority)
			}
		}
	}

	// Validate routing configuration
	if cfg.Routing.DefaultBackend != "" {
		if !backendIDs[cfg.Routing.DefaultBackend] {
			return fmt.Errorf("default backend '%s' not found in enabled backends",
				cfg.Routing.DefaultBackend)
		}
	}

	// Validate forwarding configuration
	if cfg.Routing.Forwarding.Enabled {
		if cfg.Routing.Forwarding.MinConfidence < 0 || cfg.Routing.Forwarding.MinConfidence > 1 {
			return fmt.Errorf("forwarding min_confidence %.2f out of range [0, 1]",
				cfg.Routing.Forwarding.MinConfidence)
		}
		if cfg.Routing.Forwarding.MaxRetries < 0 {
			return fmt.Errorf("forwarding max_retries cannot be negative: %d",
				cfg.Routing.Forwarding.MaxRetries)
		}
		// Validate escalation path backends exist
		for _, backendID := range cfg.Routing.Forwarding.EscalationPath {
			if !backendIDs[backendID] {
				return fmt.Errorf("escalation path backend '%s' not found in enabled backends",
					backendID)
			}
		}
	}

	// Validate confidence weights
	if cfg.Routing.Confidence.LengthWeight < 0 || cfg.Routing.Confidence.LengthWeight > 1 {
		return fmt.Errorf("confidence length_weight %.2f out of range [0, 1]",
			cfg.Routing.Confidence.LengthWeight)
	}
	if cfg.Routing.Confidence.PatternWeight < 0 || cfg.Routing.Confidence.PatternWeight > 1 {
		return fmt.Errorf("confidence pattern_weight %.2f out of range [0, 1]",
			cfg.Routing.Confidence.PatternWeight)
	}
	if cfg.Routing.Confidence.ModelWeight < 0 || cfg.Routing.Confidence.ModelWeight > 1 {
		return fmt.Errorf("confidence model_weight %.2f out of range [0, 1]",
			cfg.Routing.Confidence.ModelWeight)
	}

	// Validate thermal thresholds
	if cfg.Thermal.Enabled {
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
	}

	// Validate monitoring configuration
	if cfg.Monitoring.Enabled && cfg.Monitoring.PrometheusPort > 0 {
		if cfg.Monitoring.PrometheusPort < 1 || cfg.Monitoring.PrometheusPort > 65535 {
			return fmt.Errorf("invalid Prometheus port: %d (must be 1-65535)",
				cfg.Monitoring.PrometheusPort)
		}
		if cfg.Monitoring.PrometheusPort == cfg.Server.GRPCPort ||
			cfg.Monitoring.PrometheusPort == cfg.Server.HTTPPort {
			return fmt.Errorf("Prometheus port %d conflicts with gRPC or HTTP port",
				cfg.Monitoring.PrometheusPort)
		}
	}

	// Validate efficiency mode
	if cfg.Efficiency.Enabled {
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
	}

	return nil
}
