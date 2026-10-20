package config

import (
	"os"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

// TestMain is part of the public API.
func TestMain(m *testing.M) {
	// Initialize logger for tests
	if err := logging.InitLogger("info", false); err != nil {
		panic(err)
	}
	defer logging.Sync()

	// Run tests
	os.Exit(m.Run())
}

// TestApplyEnvOverrides_GRPCPort is part of the public API.
func TestApplyEnvOverrides_GRPCPort(t *testing.T) {
	// Set environment variable
	os.Setenv("OLLAMA_PROXY_GRPC_PORT", "9999")
	defer os.Unsetenv("OLLAMA_PROXY_GRPC_PORT")

	cfg := &Config{}
	cfg.Server.GRPCPort = 50051
	cfg.Server.HTTPPort = 8080
	cfg.Server.Host = strLocalhost

	ApplyEnvOverrides(cfg)

	if cfg.Server.GRPCPort != 9999 {
		t.Errorf("Expected GRPC port to be 9999, got %d", cfg.Server.GRPCPort)
	}
}

// TestApplyEnvOverrides_HTTPPort is part of the public API.
func TestApplyEnvOverrides_HTTPPort(t *testing.T) {
	os.Setenv("OLLAMA_PROXY_HTTP_PORT", "8888")
	defer os.Unsetenv("OLLAMA_PROXY_HTTP_PORT")

	cfg := &Config{}
	cfg.Server.GRPCPort = 50051
	cfg.Server.HTTPPort = 8080
	cfg.Server.Host = strLocalhost

	ApplyEnvOverrides(cfg)

	if cfg.Server.HTTPPort != 8888 {
		t.Errorf("Expected HTTP port to be 8888, got %d", cfg.Server.HTTPPort)
	}
}

// TestApplyEnvOverrides_Host is part of the public API.
func TestApplyEnvOverrides_Host(t *testing.T) {
	os.Setenv("OLLAMA_PROXY_HOST", "0.0.0.0")
	defer os.Unsetenv("OLLAMA_PROXY_HOST")

	cfg := &Config{}
	cfg.Server.Host = strLocalhost

	ApplyEnvOverrides(cfg)

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host to be 0.0.0.0, got %s", cfg.Server.Host)
	}
}

// TestApplyEnvOverrides_LogLevel is part of the public API.
func TestApplyEnvOverrides_LogLevel(t *testing.T) {
	os.Setenv("OLLAMA_PROXY_LOG_LEVEL", "debug")
	defer os.Unsetenv("OLLAMA_PROXY_LOG_LEVEL")

	cfg := &Config{}
	cfg.Monitoring.LogLevel = "info"

	ApplyEnvOverrides(cfg)

	if cfg.Monitoring.LogLevel != "debug" {
		t.Errorf("Expected log level to be debug, got %s", cfg.Monitoring.LogLevel)
	}
}

// TestApplyEnvOverrides_TLSEnabled is part of the public API.
func TestApplyEnvOverrides_TLSEnabled(t *testing.T) {
	os.Setenv("OLLAMA_PROXY_TLS_ENABLED", "true")
	defer os.Unsetenv("OLLAMA_PROXY_TLS_ENABLED")

	cfg := &Config{}
	cfg.Server.TLS.Enabled = false

	ApplyEnvOverrides(cfg)

	if !cfg.Server.TLS.Enabled {
		t.Errorf("Expected TLS to be enabled, got %v", cfg.Server.TLS.Enabled)
	}
}

// TestApplyEnvOverrides_InvalidPort is part of the public API.
func TestApplyEnvOverrides_InvalidPort(t *testing.T) {
	os.Setenv("OLLAMA_PROXY_GRPC_PORT", "invalid")
	defer os.Unsetenv("OLLAMA_PROXY_GRPC_PORT")

	cfg := &Config{}
	cfg.Server.GRPCPort = 50051

	ApplyEnvOverrides(cfg)

	// Should remain unchanged due to parse error
	if cfg.Server.GRPCPort != 50051 {
		t.Errorf("Expected GRPC port to remain 50051, got %d", cfg.Server.GRPCPort)
	}
}

// TestApplyEnvOverrides_BackendEndpoint is part of the public API.
func TestApplyEnvOverrides_BackendEndpoint(t *testing.T) {
	os.Setenv("OLLAMA_NPU_ENDPOINT", "http://new-npu:11434")
	defer os.Unsetenv("OLLAMA_NPU_ENDPOINT")

	cfg := &Config{}
	// Create a backend using the Config's Backends type
	backend := cfg.Backends
	backend = append(backend, struct {
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
		ID:       "ollama-npu",
		Endpoint: "http://old-npu:11434",
	})
	cfg.Backends = backend

	ApplyEnvOverrides(cfg)

	if cfg.Backends[0].Endpoint != "http://new-npu:11434" {
		t.Errorf("Expected endpoint to be http://new-npu:11434, got %s", cfg.Backends[0].Endpoint)
	}
}
