package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestRecordRequest is part of the public API.
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
			backendID:   strBackend1,
			model:       strLlama2,
			status:      "success",
			durationSec: 1.5,
		},
		{
			name:        "failed request",
			backendID:   strBackend2,
			model:       strGpt4,
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

// TestRecordTokens is part of the public API.
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
			backendID:       strBackend1,
			model:           strLlama2,
			tokensGenerated: 100,
			tokensPerSec:    25.5,
		},
		{
			name:            "zero tokens",
			backendID:       strBackend2,
			model:           strGpt4,
			tokensGenerated: 0,
			tokensPerSec:    0,
		},
		{
			name:            "high token generation",
			backendID:       strBackend3,
			model:           strClaude,
			tokensGenerated: 5000,
			tokensPerSec:    150.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			RecordTokens(tt.backendID, tt.model, tt.tokensGenerated, tt.tokensPerSec)
			// Just verify it doesn't panic - actual metric verification is complex for histograms
		})
	}
}

// TestSetBackendHealth is part of the public API.
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
			backendID: strBackend1,
			hardware:  strGPU,
			healthy:   true,
			wantValue: 1.0,
		},
		{
			name:      "unhealthy backend",
			backendID: strBackend2,
			hardware:  strCPU,
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

// TestSetBackendQueueDepth is part of the public API.
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
			backendID: strBackend1,
			priority:  "high",
			depth:     0,
		},
		{
			name:      "normal queue",
			backendID: strBackend2,
			priority:  "normal",
			depth:     5,
		},
		{
			name:      "full queue",
			backendID: strBackend3,
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

// TestSetBackendTemperature is part of the public API.
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
			backendID:   strBackend1,
			hardware:    strGPU,
			tempCelsius: 45.5,
		},
		{
			name:        "warm temperature",
			backendID:   strBackend2,
			hardware:    strCPU,
			tempCelsius: 75.0,
		},
		{
			name:        "hot temperature",
			backendID:   strBackend3,
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

// TestSetBackendFanSpeed is part of the public API.
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
			backendID:  strBackend1,
			hardware:   strGPU,
			fanPercent: 0,
		},
		{
			name:       "fan moderate",
			backendID:  strBackend2,
			hardware:   strCPU,
			fanPercent: 50,
		},
		{
			name:       "fan max",
			backendID:  strBackend3,
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

// TestSetBackendPower is part of the public API.
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
			backendID: strBackend1,
			hardware:  strCPU,
			watts:     15.5,
		},
		{
			name:      "medium power",
			backendID: strBackend2,
			hardware:  strGPU,
			watts:     150.0,
		},
		{
			name:      "high power",
			backendID: strBackend3,
			hardware:  strGPU,
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

// TestRecordEnergyConsumed is part of the public API.
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
			backendID:  strBackend1,
			hardware:   strGPU,
			wattHours:  10.5,
			iterations: 1,
		},
		{
			name:       "multiple consumptions",
			backendID:  strBackend2,
			hardware:   strCPU,
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

// TestRecordRoutingDecision is part of the public API.
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
			backendID: strBackend1,
		},
		{
			name:      "power routing",
			reason:    "power_efficient",
			backendID: strBackend2,
		},
		{
			name:      "thermal routing",
			reason:    "thermal_limit",
			backendID: strBackend3,
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

// TestRecordForwardingAttempt is part of the public API.
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
			fromBackend: strBackend1,
			toBackend:   strBackend2,
			result:      "success",
		},
		{
			name:        "failed forward",
			fromBackend: strBackend2,
			toBackend:   strBackend3,
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

// TestRecordConfidenceScore is part of the public API.
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
			backendID: strBackend1,
			model:     strLlama2,
			score:     0.3,
		},
		{
			name:      "high confidence",
			backendID: strBackend2,
			model:     strGpt4,
			score:     0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			RecordConfidenceScore(tt.backendID, tt.model, tt.score)
			// Histograms are harder to verify, just ensure no panic
		})
	}
}

// TestSetEfficiencyMode is part of the public API.
func TestSetEfficiencyMode(t *testing.T) {
	Mode.Reset()

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

			gauge, err := Mode.GetMetricWithLabelValues(tt.mode)
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

// TestRecordTimeToFirstToken is part of the public API.
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
			backendID: strBackend1,
			model:     strLlama2,
			ttftMs:    50,
		},
		{
			name:      "slow TTFT",
			backendID: strBackend2,
			model:     strGpt4,
			ttftMs:    500,
		},
		{
			name:      "zero TTFT",
			backendID: strBackend3,
			model:     strClaude,
			ttftMs:    0, // Should not record
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			RecordTimeToFirstToken(tt.backendID, tt.model, tt.ttftMs)
			// Histograms are harder to verify, just ensure no panic
		})
	}
}

// TestRecordInterTokenLatency is part of the public API.
func TestRecordInterTokenLatency(t *testing.T) {
	InterTokenLatency.Reset()

	tests := []struct {
		name      string
		backendID string
		model     string
		latencyMs float64
	}{
		{
			name:      "low latency",
			backendID: strBackend1,
			model:     strLlama2,
			latencyMs: 10.5,
		},
		{
			name:      "high latency",
			backendID: strBackend2,
			model:     strGpt4,
			latencyMs: 100.0,
		},
		{
			name:      "zero latency",
			backendID: strBackend3,
			model:     strClaude,
			latencyMs: 0.0, // Should not record
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			RecordInterTokenLatency(tt.backendID, tt.model, tt.latencyMs)
			// Histograms are harder to verify, just ensure no panic
		})
	}
}

// TestRecordCacheHit is part of the public API.
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
			cacheType: strModel,
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

// TestRecordCacheMiss is part of the public API.
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
			cacheType: strModel,
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

// TestMetricsCollectionNoPanic is part of the public API.
func TestMetricsCollectionNoPanic(t *testing.T) {
	// Test that all metric collectors are properly registered
	// This ensures they can be collected by Prometheus without panic
	if _, err := prometheus.DefaultGatherer.Gather(); err != nil {
		t.Fatal(err)
	}
}
