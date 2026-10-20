// Package config loads proxy configuration and applies environment overrides.
package config

import (
	"os"
	"strconv"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// ApplyEnvOverrides applies environment variable overrides to the configuration
func ApplyEnvOverrides(cfg *Config) {
	applyIntOverride("OLLAMA_PROXY_GRPC_PORT", "Invalid GRPC port in environment variable", func(port int) {
		cfg.Server.GRPCPort = port
	})
	applyIntOverride("OLLAMA_PROXY_HTTP_PORT", "Invalid HTTP port in environment variable", func(port int) {
		cfg.Server.HTTPPort = port
	})
	applyStringOverride("OLLAMA_PROXY_HOST", func(val string) {
		cfg.Server.Host = val
	})
	applyStringOverride("OLLAMA_PROXY_LOG_LEVEL", func(val string) {
		cfg.Monitoring.LogLevel = val
	})
	applyIntOverride("OLLAMA_PROXY_PROMETHEUS_PORT", "Invalid Prometheus port in environment variable", func(port int) {
		cfg.Monitoring.PrometheusPort = port
	})
	applyBackendEndpointOverride("OLLAMA_NPU_ENDPOINT", "ollama-npu", cfg)
	applyBackendEndpointOverride("OLLAMA_GPU_ENDPOINT", "ollama-gpu", cfg)
	applyBackendEndpointOverride("OLLAMA_CPU_ENDPOINT", "ollama-cpu", cfg)
	applyBoolOverride("OLLAMA_PROXY_TLS_ENABLED", "Invalid TLS enabled value in environment variable", func(enabled bool) {
		cfg.Server.TLS.Enabled = enabled
	})
	applyStringOverride("OLLAMA_PROXY_TLS_CERT_FILE", func(val string) {
		cfg.Server.TLS.CertFile = val
	})
	applyStringOverride("OLLAMA_PROXY_TLS_KEY_FILE", func(val string) {
		cfg.Server.TLS.KeyFile = val
	})
	applyStringOverride("OLLAMA_PROXY_TLS_CLIENT_CA_FILE", func(val string) {
		cfg.Server.TLS.ClientCAFile = val
	})
	applyBoolOverride("OLLAMA_PROXY_THERMAL_ENABLED", "Invalid thermal enabled value in environment variable", func(enabled bool) {
		cfg.Thermal.Enabled = enabled
	})
	applyBoolOverride("OLLAMA_PROXY_EFFICIENCY_ENABLED", "Invalid efficiency enabled value in environment variable", func(enabled bool) {
		cfg.Efficiency.Enabled = enabled
	})
	applyStringOverride("OLLAMA_PROXY_EFFICIENCY_DEFAULT_MODE", func(val string) {
		cfg.Efficiency.DefaultMode = val
	})
	applyBoolOverride("OLLAMA_PROXY_AUTH_ENABLED", "Invalid auth enabled value in environment variable", func(enabled bool) {
		cfg.Server.Auth.Enabled = enabled
	})
	applyBoolOverride("OLLAMA_PROXY_RATE_LIMIT_ENABLED", "Invalid rate limit enabled value in environment variable", func(enabled bool) {
		cfg.Server.RateLimit.Enabled = enabled
	})
	applyFloatOverride("OLLAMA_PROXY_RATE_LIMIT_RATE", "Invalid rate limit rate value in environment variable", func(rate float64) {
		cfg.Server.RateLimit.Rate = rate
	})
	applyIntOverride("OLLAMA_PROXY_RATE_LIMIT_BURST", "Invalid rate limit burst value in environment variable", func(burst int) {
		cfg.Server.RateLimit.Burst = burst
	})
}

func applyStringOverride(name string, set func(string)) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	logging.Logger.Info("Override from environment",
		zap.String("var", name),
		zap.String("value", val),
	)
	set(val)
}

func applyIntOverride(name, invalidMsg string, set func(int)) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		logging.Logger.Warn(invalidMsg,
			zap.String("value", val),
			zap.Error(err),
		)
		return
	}
	logging.Logger.Info("Override from environment",
		zap.String("var", name),
		zap.Int("value", parsed),
	)
	set(parsed)
}

func applyBoolOverride(name, invalidMsg string, set func(bool)) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		logging.Logger.Warn(invalidMsg,
			zap.String("value", val),
			zap.Error(err),
		)
		return
	}
	logging.Logger.Info("Override from environment",
		zap.String("var", name),
		zap.Bool("value", parsed),
	)
	set(parsed)
}

func applyFloatOverride(name, invalidMsg string, set func(float64)) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		logging.Logger.Warn(invalidMsg,
			zap.String("value", val),
			zap.Error(err),
		)
		return
	}
	logging.Logger.Info("Override from environment",
		zap.String("var", name),
		zap.Float64("value", parsed),
	)
	set(parsed)
}

func applyBackendEndpointOverride(name, backendID string, cfg *Config) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	logging.Logger.Info("Override from environment",
		zap.String("var", name),
		zap.String("value", val),
	)
	for i := range cfg.Backends {
		if cfg.Backends[i].ID == backendID {
			cfg.Backends[i].Endpoint = val
		}
	}
}
