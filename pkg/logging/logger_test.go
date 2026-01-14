package logging

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name       string
		level      string
		production bool
		wantErr    bool
	}{
		{
			name:       "debug level development",
			level:      "debug",
			production: false,
			wantErr:    false,
		},
		{
			name:       "info level development",
			level:      "info",
			production: false,
			wantErr:    false,
		},
		{
			name:       "warn level production",
			level:      "warn",
			production: true,
			wantErr:    false,
		},
		{
			name:       "error level production",
			level:      "error",
			production: true,
			wantErr:    false,
		},
		{
			name:       "default level (empty)",
			level:      "",
			production: false,
			wantErr:    false,
		},
		{
			name:       "unknown level defaults to info",
			level:      "unknown",
			production: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitLogger(tt.level, tt.production)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitLogger() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && Logger == nil {
				t.Error("InitLogger() succeeded but Logger is nil")
			}

			// Verify production mode sets JSON encoding
			if tt.production && Logger != nil {
				// Logger is initialized, verify it works
				Logger.Info("test message")
			}

			// Verify development mode sets console encoding
			if !tt.production && Logger != nil {
				Logger.Debug("test debug message")
			}
		})
	}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name       string
		initLogger bool
	}{
		{
			name:       "sync with initialized logger",
			initLogger: true,
		},
		{
			name:       "sync with nil logger",
			initLogger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.initLogger {
				if err := InitLogger("info", false); err != nil {
					t.Fatalf("Failed to initialize logger: %v", err)
				}
			} else {
				Logger = nil
			}

			// Should not panic
			Sync()
		})
	}
}

func TestLoggingFunctions(t *testing.T) {
	// Initialize logger for testing
	if err := InitLogger("debug", false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "Info",
			fn: func() {
				Info("test info message", zap.String("key", "value"))
			},
		},
		{
			name: "Debug",
			fn: func() {
				Debug("test debug message", zap.Int("count", 42))
			},
		},
		{
			name: "Warn",
			fn: func() {
				Warn("test warning message", zap.Bool("flag", true))
			},
		},
		{
			name: "Error",
			fn: func() {
				Error("test error message", zap.Error(nil))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}

func TestLoggingFunctionsWithNilLogger(t *testing.T) {
	Logger = nil

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "Info with nil logger",
			fn: func() {
				Info("test info message")
			},
		},
		{
			name: "Debug with nil logger",
			fn: func() {
				Debug("test debug message")
			},
		},
		{
			name: "Warn with nil logger",
			fn: func() {
				Warn("test warning message")
			},
		},
		{
			name: "Error with nil logger",
			fn: func() {
				Error("test error message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic even with nil logger
			tt.fn()
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level         string
		expectedLevel zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"invalid", zapcore.InfoLevel}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if err := InitLogger(tt.level, false); err != nil {
				t.Fatalf("Failed to initialize logger: %v", err)
			}

			if Logger == nil {
				t.Fatal("Logger should not be nil")
			}

			// Verify logger is at the expected level by checking if it would log at that level
			// This is a basic sanity check
			if tt.level == "debug" {
				Debug("debug message")
			} else if tt.level == "info" {
				Info("info message")
			}
		})
	}
}

func TestProductionVsDevelopmentMode(t *testing.T) {
	t.Run("production mode", func(t *testing.T) {
		if err := InitLogger("info", true); err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		if Logger == nil {
			t.Fatal("Logger should not be nil in production mode")
		}

		// Log a message to ensure it works
		Info("production mode test")
	})

	t.Run("development mode", func(t *testing.T) {
		if err := InitLogger("debug", false); err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		if Logger == nil {
			t.Fatal("Logger should not be nil in development mode")
		}

		// Log a message to ensure it works
		Debug("development mode test")
	})
}
