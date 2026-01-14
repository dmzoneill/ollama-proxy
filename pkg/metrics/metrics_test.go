package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordRequest(t *testing.T) {
	// Reset metrics before test
	RequestsTotal.Reset()
	RequestDuration.Reset()

	tests := []struct {
		name        string
		backendID   string
		model       string
		status      string
		durationSec float64
	}{
		{
			name:        "successful request",
			backendID:   "backend-1",
			model:       "llama2",
			status:      "success",
			durationSec: 1.5,
		},
		{
			name:        "failed request",
			backendID:   "backend-2",
			model:       "gpt-4",
			status:      "error",
			durationSec: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordRequest(tt.backendID, tt.model, tt.status, tt.durationSec)

			// Verify counter was incremented
			counter, err := RequestsTotal.GetMetricWithLabelValues(tt.backendID, tt.model, tt.status)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(counter)
			if value < 1 {
				t.Errorf("Expected counter >= 1, got %f", value)
			}
		})
	}
}

func TestRecordTokens(t *testing.T) {
	TokensGenerated.Reset()
	TokensPerSecond.Reset()

	tests := []struct {
		name            string
		backendID       string
		model           string
		tokensGenerated int32
		tokensPerSec    float32
	}{
		{
			name:            "normal token generation",
			backendID:       "backend-1",
			model:           "llama2",
			tokensGenerated: 100,
			tokensPerSec:    25.5,
		},
		{
			name:            "zero tokens",
			backendID:       "backend-2",
			model:           "gpt-4",
			tokensGenerated: 0,
			tokensPerSec:    0,
		},
		{
			name:            "high token generation",
			backendID:       "backend-3",
			model:           "claude",
			tokensGenerated: 5000,
			tokensPerSec:    150.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordTokens(tt.backendID, tt.model, tt.tokensGenerated, tt.tokensPerSec)
			// Just verify it doesn't panic - actual metric verification is complex for histograms
		})
	}
}

func TestSetBackendHealth(t *testing.T) {
	BackendHealth.Reset()

	tests := []struct {
		name      string
		backendID string
		hardware  string
		healthy   bool
		wantValue float64
	}{
		{
			name:      "healthy backend",
			backendID: "backend-1",
			hardware:  "GPU",
			healthy:   true,
			wantValue: 1.0,
		},
		{
			name:      "unhealthy backend",
			backendID: "backend-2",
			hardware:  "CPU",
			healthy:   false,
			wantValue: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBackendHealth(tt.backendID, tt.hardware, tt.healthy)

			gauge, err := BackendHealth.GetMetricWithLabelValues(tt.backendID, tt.hardware)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(gauge)
			if value != tt.wantValue {
				t.Errorf("Expected health %f, got %f", tt.wantValue, value)
			}
		})
	}
}

func TestSetBackendQueueDepth(t *testing.T) {
	BackendQueueDepth.Reset()

	tests := []struct {
		name      string
		backendID string
		priority  string
		depth     int
	}{
		{
			name:      "empty queue",
			backendID: "backend-1",
			priority:  "high",
			depth:     0,
		},
		{
			name:      "normal queue",
			backendID: "backend-2",
			priority:  "normal",
			depth:     5,
		},
		{
			name:      "full queue",
			backendID: "backend-3",
			priority:  "low",
			depth:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBackendQueueDepth(tt.backendID, tt.priority, tt.depth)

			gauge, err := BackendQueueDepth.GetMetricWithLabelValues(tt.backendID, tt.priority)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(gauge)
			if value != float64(tt.depth) {
				t.Errorf("Expected queue depth %d, got %f", tt.depth, value)
			}
		})
	}
}

func TestSetBackendTemperature(t *testing.T) {
	BackendTemperature.Reset()

	tests := []struct {
		name        string
		backendID   string
		hardware    string
		tempCelsius float64
	}{
		{
			name:        "cool temperature",
			backendID:   "backend-1",
			hardware:    "GPU",
			tempCelsius: 45.5,
		},
		{
			name:        "warm temperature",
			backendID:   "backend-2",
			hardware:    "CPU",
			tempCelsius: 75.0,
		},
		{
			name:        "hot temperature",
			backendID:   "backend-3",
			hardware:    "NPU",
			tempCelsius: 95.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBackendTemperature(tt.backendID, tt.hardware, tt.tempCelsius)

			gauge, err := BackendTemperature.GetMetricWithLabelValues(tt.backendID, tt.hardware)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(gauge)
			if value != tt.tempCelsius {
				t.Errorf("Expected temperature %f, got %f", tt.tempCelsius, value)
			}
		})
	}
}

func TestSetBackendFanSpeed(t *testing.T) {
	BackendFanSpeed.Reset()

	tests := []struct {
		name       string
		backendID  string
		hardware   string
		fanPercent int
	}{
		{
			name:       "fan off",
			backendID:  "backend-1",
			hardware:   "GPU",
			fanPercent: 0,
		},
		{
			name:       "fan moderate",
			backendID:  "backend-2",
			hardware:   "CPU",
			fanPercent: 50,
		},
		{
			name:       "fan max",
			backendID:  "backend-3",
			hardware:   "NPU",
			fanPercent: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBackendFanSpeed(tt.backendID, tt.hardware, tt.fanPercent)

			gauge, err := BackendFanSpeed.GetMetricWithLabelValues(tt.backendID, tt.hardware)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(gauge)
			if value != float64(tt.fanPercent) {
				t.Errorf("Expected fan speed %d, got %f", tt.fanPercent, value)
			}
		})
	}
}

func TestSetBackendPower(t *testing.T) {
	BackendPower.Reset()

	tests := []struct {
		name      string
		backendID string
		hardware  string
		watts     float64
	}{
		{
			name:      "low power",
			backendID: "backend-1",
			hardware:  "CPU",
			watts:     15.5,
		},
		{
			name:      "medium power",
			backendID: "backend-2",
			hardware:  "GPU",
			watts:     150.0,
		},
		{
			name:      "high power",
			backendID: "backend-3",
			hardware:  "GPU",
			watts:     300.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBackendPower(tt.backendID, tt.hardware, tt.watts)

			gauge, err := BackendPower.GetMetricWithLabelValues(tt.backendID, tt.hardware)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(gauge)
			if value != tt.watts {
				t.Errorf("Expected power %f, got %f", tt.watts, value)
			}
		})
	}
}

func TestRecordEnergyConsumed(t *testing.T) {
	EnergyConsumed.Reset()

	tests := []struct {
		name       string
		backendID  string
		hardware   string
		wattHours  float64
		iterations int
	}{
		{
			name:       "single consumption",
			backendID:  "backend-1",
			hardware:   "GPU",
			wattHours:  10.5,
			iterations: 1,
		},
		{
			name:       "multiple consumptions",
			backendID:  "backend-2",
			hardware:   "CPU",
			wattHours:  5.0,
			iterations: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.iterations; i++ {
				RecordEnergyConsumed(tt.backendID, tt.hardware, tt.wattHours)
			}

			counter, err := EnergyConsumed.GetMetricWithLabelValues(tt.backendID, tt.hardware)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(counter)
			expectedTotal := tt.wattHours * float64(tt.iterations)
			if value != expectedTotal {
				t.Errorf("Expected energy %f, got %f", expectedTotal, value)
			}
		})
	}
}

func TestRecordRoutingDecision(t *testing.T) {
	RoutingDecisionsTotal.Reset()

	tests := []struct {
		name      string
		reason    string
		backendID string
	}{
		{
			name:      "latency routing",
			reason:    "latency",
			backendID: "backend-1",
		},
		{
			name:      "power routing",
			reason:    "power_efficient",
			backendID: "backend-2",
		},
		{
			name:      "thermal routing",
			reason:    "thermal_limit",
			backendID: "backend-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordRoutingDecision(tt.reason, tt.backendID)

			counter, err := RoutingDecisionsTotal.GetMetricWithLabelValues(tt.reason, tt.backendID)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(counter)
			if value < 1 {
				t.Errorf("Expected counter >= 1, got %f", value)
			}
		})
	}
}

func TestRecordForwardingAttempt(t *testing.T) {
	ForwardingAttemptsTotal.Reset()

	tests := []struct {
		name        string
		fromBackend string
		toBackend   string
		result      string
	}{
		{
			name:        "successful forward",
			fromBackend: "backend-1",
			toBackend:   "backend-2",
			result:      "success",
		},
		{
			name:        "failed forward",
			fromBackend: "backend-2",
			toBackend:   "backend-3",
			result:      "failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordForwardingAttempt(tt.fromBackend, tt.toBackend, tt.result)

			counter, err := ForwardingAttemptsTotal.GetMetricWithLabelValues(tt.fromBackend, tt.toBackend, tt.result)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(counter)
			if value < 1 {
				t.Errorf("Expected counter >= 1, got %f", value)
			}
		})
	}
}

func TestRecordConfidenceScore(t *testing.T) {
	ConfidenceScores.Reset()

	tests := []struct {
		name      string
		backendID string
		model     string
		score     float64
	}{
		{
			name:      "low confidence",
			backendID: "backend-1",
			model:     "llama2",
			score:     0.3,
		},
		{
			name:      "high confidence",
			backendID: "backend-2",
			model:     "gpt-4",
			score:     0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordConfidenceScore(tt.backendID, tt.model, tt.score)
			// Histograms are harder to verify, just ensure no panic
		})
	}
}

func TestSetEfficiencyMode(t *testing.T) {
	EfficiencyMode.Reset()

	tests := []struct {
		name  string
		mode  string
		value int
	}{
		{
			name:  "performance mode",
			mode:  "performance",
			value: 0,
		},
		{
			name:  "balanced mode",
			mode:  "balanced",
			value: 1,
		},
		{
			name:  "efficiency mode",
			mode:  "efficiency",
			value: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetEfficiencyMode(tt.mode, tt.value)

			gauge, err := EfficiencyMode.GetMetricWithLabelValues(tt.mode)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(gauge)
			if value != float64(tt.value) {
				t.Errorf("Expected mode value %d, got %f", tt.value, value)
			}
		})
	}
}

func TestRecordTimeToFirstToken(t *testing.T) {
	TimeToFirstToken.Reset()

	tests := []struct {
		name      string
		backendID string
		model     string
		ttftMs    int32
	}{
		{
			name:      "fast TTFT",
			backendID: "backend-1",
			model:     "llama2",
			ttftMs:    50,
		},
		{
			name:      "slow TTFT",
			backendID: "backend-2",
			model:     "gpt-4",
			ttftMs:    500,
		},
		{
			name:      "zero TTFT",
			backendID: "backend-3",
			model:     "claude",
			ttftMs:    0, // Should not record
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordTimeToFirstToken(tt.backendID, tt.model, tt.ttftMs)
			// Histograms are harder to verify, just ensure no panic
		})
	}
}

func TestRecordInterTokenLatency(t *testing.T) {
	InterTokenLatency.Reset()

	tests := []struct {
		name       string
		backendID  string
		model      string
		latencyMs  float64
	}{
		{
			name:       "low latency",
			backendID:  "backend-1",
			model:      "llama2",
			latencyMs:  10.5,
		},
		{
			name:       "high latency",
			backendID:  "backend-2",
			model:      "gpt-4",
			latencyMs:  100.0,
		},
		{
			name:       "zero latency",
			backendID:  "backend-3",
			model:      "claude",
			latencyMs:  0.0, // Should not record
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordInterTokenLatency(tt.backendID, tt.model, tt.latencyMs)
			// Histograms are harder to verify, just ensure no panic
		})
	}
}

func TestRecordCacheHit(t *testing.T) {
	CacheHits.Reset()

	tests := []struct {
		name      string
		cacheType string
	}{
		{
			name:      "prompt cache hit",
			cacheType: "prompt",
		},
		{
			name:      "model cache hit",
			cacheType: "model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordCacheHit(tt.cacheType)

			counter, err := CacheHits.GetMetricWithLabelValues(tt.cacheType)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(counter)
			if value < 1 {
				t.Errorf("Expected counter >= 1, got %f", value)
			}
		})
	}
}

func TestRecordCacheMiss(t *testing.T) {
	CacheMisses.Reset()

	tests := []struct {
		name      string
		cacheType string
	}{
		{
			name:      "prompt cache miss",
			cacheType: "prompt",
		},
		{
			name:      "model cache miss",
			cacheType: "model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordCacheMiss(tt.cacheType)

			counter, err := CacheMisses.GetMetricWithLabelValues(tt.cacheType)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			value := testutil.ToFloat64(counter)
			if value < 1 {
				t.Errorf("Expected counter >= 1, got %f", value)
			}
		})
	}
}

func TestMetricsCollectionNoPanic(t *testing.T) {
	// Test that all metric collectors are properly registered
	// This ensures they can be collected by Prometheus without panic
	prometheus.DefaultGatherer.Gather()
}
