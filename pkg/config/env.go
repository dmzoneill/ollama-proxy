package config

import (
	"os"
	"strconv"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// ApplyEnvOverrides applies environment variable overrides to the configuration
func ApplyEnvOverrides(cfg *Config) {
	// Server overrides
	if val := os.Getenv("OLLAMA_PROXY_GRPC_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_GRPC_PORT"),
				zap.Int("value", port),
			)
			cfg.Server.GRPCPort = port
		} else {
			logging.Logger.Warn("Invalid GRPC port in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	if val := os.Getenv("OLLAMA_PROXY_HTTP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_HTTP_PORT"),
				zap.Int("value", port),
			)
			cfg.Server.HTTPPort = port
		} else {
			logging.Logger.Warn("Invalid HTTP port in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	if val := os.Getenv("OLLAMA_PROXY_HOST"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_PROXY_HOST"),
			zap.String("value", val),
		)
		cfg.Server.Host = val
	}

	// Monitoring overrides
	if val := os.Getenv("OLLAMA_PROXY_LOG_LEVEL"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_PROXY_LOG_LEVEL"),
			zap.String("value", val),
		)
		cfg.Monitoring.LogLevel = val
	}

	if val := os.Getenv("OLLAMA_PROXY_PROMETHEUS_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_PROMETHEUS_PORT"),
				zap.Int("value", port),
			)
			cfg.Monitoring.PrometheusPort = port
		} else {
			logging.Logger.Warn("Invalid Prometheus port in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	// Backend endpoint overrides
	if val := os.Getenv("OLLAMA_NPU_ENDPOINT"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_NPU_ENDPOINT"),
			zap.String("value", val),
		)
		for i := range cfg.Backends {
			if cfg.Backends[i].ID == "ollama-npu" {
				cfg.Backends[i].Endpoint = val
			}
		}
	}

	if val := os.Getenv("OLLAMA_GPU_ENDPOINT"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_GPU_ENDPOINT"),
			zap.String("value", val),
		)
		for i := range cfg.Backends {
			if cfg.Backends[i].ID == "ollama-gpu" {
				cfg.Backends[i].Endpoint = val
			}
		}
	}

	if val := os.Getenv("OLLAMA_CPU_ENDPOINT"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_CPU_ENDPOINT"),
			zap.String("value", val),
		)
		for i := range cfg.Backends {
			if cfg.Backends[i].ID == "ollama-cpu" {
				cfg.Backends[i].Endpoint = val
			}
		}
	}

	// TLS overrides
	if val := os.Getenv("OLLAMA_PROXY_TLS_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_TLS_ENABLED"),
				zap.Bool("value", enabled),
			)
			cfg.Server.TLS.Enabled = enabled
		} else {
			logging.Logger.Warn("Invalid TLS enabled value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	if val := os.Getenv("OLLAMA_PROXY_TLS_CERT_FILE"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_PROXY_TLS_CERT_FILE"),
			zap.String("value", val),
		)
		cfg.Server.TLS.CertFile = val
	}

	if val := os.Getenv("OLLAMA_PROXY_TLS_KEY_FILE"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_PROXY_TLS_KEY_FILE"),
			zap.String("value", val),
		)
		cfg.Server.TLS.KeyFile = val
	}

	if val := os.Getenv("OLLAMA_PROXY_TLS_CLIENT_CA_FILE"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_PROXY_TLS_CLIENT_CA_FILE"),
			zap.String("value", val),
		)
		cfg.Server.TLS.ClientCAFile = val
	}

	// Thermal monitoring overrides
	if val := os.Getenv("OLLAMA_PROXY_THERMAL_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_THERMAL_ENABLED"),
				zap.Bool("value", enabled),
			)
			cfg.Thermal.Enabled = enabled
		} else {
			logging.Logger.Warn("Invalid thermal enabled value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	// Efficiency mode overrides
	if val := os.Getenv("OLLAMA_PROXY_EFFICIENCY_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_EFFICIENCY_ENABLED"),
				zap.Bool("value", enabled),
			)
			cfg.Efficiency.Enabled = enabled
		} else {
			logging.Logger.Warn("Invalid efficiency enabled value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	if val := os.Getenv("OLLAMA_PROXY_EFFICIENCY_DEFAULT_MODE"); val != "" {
		logging.Logger.Info("Override from environment",
			zap.String("var", "OLLAMA_PROXY_EFFICIENCY_DEFAULT_MODE"),
			zap.String("value", val),
		)
		cfg.Efficiency.DefaultMode = val
	}

	// Auth overrides
	if val := os.Getenv("OLLAMA_PROXY_AUTH_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_AUTH_ENABLED"),
				zap.Bool("value", enabled),
			)
			cfg.Server.Auth.Enabled = enabled
		} else {
			logging.Logger.Warn("Invalid auth enabled value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	// Rate limiting overrides
	if val := os.Getenv("OLLAMA_PROXY_RATE_LIMIT_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_RATE_LIMIT_ENABLED"),
				zap.Bool("value", enabled),
			)
			cfg.Server.RateLimit.Enabled = enabled
		} else {
			logging.Logger.Warn("Invalid rate limit enabled value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	if val := os.Getenv("OLLAMA_PROXY_RATE_LIMIT_RATE"); val != "" {
		if rateVal, err := strconv.ParseFloat(val, 64); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_RATE_LIMIT_RATE"),
				zap.Float64("value", rateVal),
			)
			cfg.Server.RateLimit.Rate = rateVal
		} else {
			logging.Logger.Warn("Invalid rate limit rate value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}

	if val := os.Getenv("OLLAMA_PROXY_RATE_LIMIT_BURST"); val != "" {
		if burst, err := strconv.Atoi(val); err == nil {
			logging.Logger.Info("Override from environment",
				zap.String("var", "OLLAMA_PROXY_RATE_LIMIT_BURST"),
				zap.Int("value", burst),
			)
			cfg.Server.RateLimit.Burst = burst
		} else {
			logging.Logger.Warn("Invalid rate limit burst value in environment variable",
				zap.String("value", val),
				zap.Error(err),
			)
		}
	}
}
