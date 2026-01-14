package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ollama_proxy_requests_total",
			Help: "Total number of requests by backend, model, and status",
		},
		[]string{"backend_id", "model", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ollama_proxy_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"backend_id", "model"},
	)

	TokensGenerated = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ollama_proxy_tokens_generated",
			Help:    "Number of tokens generated per request",
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 2000, 5000, 10000},
		},
		[]string{"backend_id", "model"},
	)

	TokensPerSecond = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ollama_proxy_tokens_per_second",
			Help:    "Tokens generated per second",
			Buckets: []float64{1, 5, 10, 20, 30, 40, 50, 75, 100, 150, 200},
		},
		[]string{"backend_id", "model"},
	)

	// Backend health metrics
	BackendHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ollama_proxy_backend_health",
			Help: "Backend health status (1=healthy, 0=unhealthy)",
		},
		[]string{"backend_id", "hardware"},
	)

	BackendQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ollama_proxy_backend_queue_depth",
			Help: "Current queue depth by backend and priority",
		},
		[]string{"backend_id", "priority"},
	)

	// Thermal metrics
	BackendTemperature = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ollama_proxy_backend_temperature_celsius",
			Help: "Backend temperature in Celsius",
		},
		[]string{"backend_id", "hardware"},
	)

	BackendFanSpeed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ollama_proxy_backend_fan_speed_percent",
			Help: "Backend fan speed percentage",
		},
		[]string{"backend_id", "hardware"},
	)

	// Power metrics
	BackendPower = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ollama_proxy_backend_power_watts",
			Help: "Backend power consumption in watts",
		},
		[]string{"backend_id", "hardware"},
	)

	EnergyConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ollama_proxy_energy_consumed_wh",
			Help: "Total energy consumed in watt-hours",
		},
		[]string{"backend_id", "hardware"},
	)

	// Routing metrics
	RoutingDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ollama_proxy_routing_decisions_total",
			Help: "Total routing decisions by reason and backend",
		},
		[]string{"reason", "backend_id"},
	)

	ForwardingAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ollama_proxy_forwarding_attempts_total",
			Help: "Total forwarding attempts due to low confidence",
		},
		[]string{"from_backend", "to_backend", "result"},
	)

	ConfidenceScores = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ollama_proxy_confidence_scores",
			Help:    "Confidence scores of responses",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"backend_id", "model"},
	)

	// Efficiency mode
	EfficiencyMode = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ollama_proxy_efficiency_mode",
			Help: "Current efficiency mode (0=Performance, 1=Balanced, 2=Efficiency, 3=Quiet, 4=Auto, 5=Ultra)",
		},
		[]string{"mode"},
	)

	// Streaming metrics
	TimeToFirstToken = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ollama_proxy_time_to_first_token_ms",
			Help:    "Time to first token in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 200, 500, 1000, 2000, 5000},
		},
		[]string{"backend_id", "model"},
	)

	InterTokenLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ollama_proxy_inter_token_latency_ms",
			Help:    "Average inter-token latency in milliseconds",
			Buckets: []float64{1, 5, 10, 20, 50, 100, 200, 500},
		},
		[]string{"backend_id", "model"},
	)

	// Cache metrics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ollama_proxy_cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ollama_proxy_cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_type"},
	)
)

// RecordRequest records a completed request
func RecordRequest(backendID, model, status string, durationSec float64) {
	RequestsTotal.WithLabelValues(backendID, model, status).Inc()
	RequestDuration.WithLabelValues(backendID, model).Observe(durationSec)
}

// RecordTokens records token generation metrics
func RecordTokens(backendID, model string, tokensGenerated int32, tokensPerSec float32) {
	if tokensGenerated > 0 {
		TokensGenerated.WithLabelValues(backendID, model).Observe(float64(tokensGenerated))
	}
	if tokensPerSec > 0 {
		TokensPerSecond.WithLabelValues(backendID, model).Observe(float64(tokensPerSec))
	}
}

// SetBackendHealth sets the health status of a backend
func SetBackendHealth(backendID, hardware string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	BackendHealth.WithLabelValues(backendID, hardware).Set(value)
}

// SetBackendQueueDepth sets the queue depth for a backend
func SetBackendQueueDepth(backendID, priority string, depth int) {
	BackendQueueDepth.WithLabelValues(backendID, priority).Set(float64(depth))
}

// SetBackendTemperature sets the temperature of a backend
func SetBackendTemperature(backendID, hardware string, tempCelsius float64) {
	BackendTemperature.WithLabelValues(backendID, hardware).Set(tempCelsius)
}

// SetBackendFanSpeed sets the fan speed of a backend
func SetBackendFanSpeed(backendID, hardware string, fanPercent int) {
	BackendFanSpeed.WithLabelValues(backendID, hardware).Set(float64(fanPercent))
}

// SetBackendPower sets the power consumption of a backend
func SetBackendPower(backendID, hardware string, watts float64) {
	BackendPower.WithLabelValues(backendID, hardware).Set(watts)
}

// RecordEnergyConsumed records energy consumption
func RecordEnergyConsumed(backendID, hardware string, wattHours float64) {
	EnergyConsumed.WithLabelValues(backendID, hardware).Add(wattHours)
}

// RecordRoutingDecision records a routing decision
func RecordRoutingDecision(reason, backendID string) {
	RoutingDecisionsTotal.WithLabelValues(reason, backendID).Inc()
}

// RecordForwardingAttempt records a forwarding attempt
func RecordForwardingAttempt(fromBackend, toBackend, result string) {
	ForwardingAttemptsTotal.WithLabelValues(fromBackend, toBackend, result).Inc()
}

// RecordConfidenceScore records a confidence score
func RecordConfidenceScore(backendID, model string, score float64) {
	ConfidenceScores.WithLabelValues(backendID, model).Observe(score)
}

// SetEfficiencyMode sets the current efficiency mode
func SetEfficiencyMode(mode string, value int) {
	EfficiencyMode.WithLabelValues(mode).Set(float64(value))
}

// RecordTimeToFirstToken records time to first token
func RecordTimeToFirstToken(backendID, model string, ttftMs int32) {
	if ttftMs > 0 {
		TimeToFirstToken.WithLabelValues(backendID, model).Observe(float64(ttftMs))
	}
}

// RecordInterTokenLatency records average inter-token latency
func RecordInterTokenLatency(backendID, model string, latencyMs float64) {
	if latencyMs > 0 {
		InterTokenLatency.WithLabelValues(backendID, model).Observe(latencyMs)
	}
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheType string) {
	CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}
