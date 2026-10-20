// Package health reports backend health.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// Status represents the health status
type Status string

// StatusHealthy is an exported const.
const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check result
type Check struct {
	Details map[string]interface{} `json:"details,omitempty"`
	Status  Status                 `json:"status"`
}

// Checker performs health checks
type Checker struct {
	lastCheck  time.Time
	backends   map[string]backends.Backend
	lastResult *Check
	mu         sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *Checker {
	return &Checker{
		backends: make(map[string]backends.Backend),
	}
}

// RegisterBackend registers a backend for health checking
func (h *Checker) RegisterBackend(backend backends.Backend) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.backends[backend.ID()] = backend
}

// LivenessCheck checks if the server is running
// This is the simplest check - if we can respond, we're alive
func (h *Checker) LivenessCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]interface{}{
		strStatus: "ok",
		strTime:   time.Now().Unix(),
	})
}

// ReadinessCheck checks if the server can serve traffic
// Returns 200 if at least one backend is healthy
func (h *Checker) ReadinessCheck(w http.ResponseWriter, _ *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	healthyCount := 0
	totalCount := len(h.backends)

	for _, backend := range h.backends {
		if backend.IsHealthy() {
			healthyCount++
		}
	}

	status := http.StatusOK
	if healthyCount == 0 {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, map[string]interface{}{
		strStatus:             map[bool]string{true: "ready", false: "not_ready"}[healthyCount > 0],
		string(StatusHealthy): healthyCount,
		strTotal:              totalCount,
		strTime:               time.Now().Unix(),
	})
}

// DeepHealthCheck performs a deep health check by actually running inference
func (h *Checker) DeepHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	h.mu.RLock()
	backends := make(map[string]backends.Backend, len(h.backends))
	for id, backend := range h.backends {
		backends[id] = backend
	}
	h.mu.RUnlock()

	results := make(map[string]interface{})
	healthyCount := 0
	totalCount := len(backends)

	for id, backend := range backends {
		start := time.Now()
		err := backend.HealthCheck(ctx)
		elapsed := time.Since(start)

		backendStatus := map[string]interface{}{
			string(StatusHealthy): err == nil,
			"latency_ms":          elapsed.Milliseconds(),
		}

		if err != nil {
			backendStatus["error"] = err.Error()
			if logging.Logger != nil {
				logging.Logger.Debug("Deep health check failed",
					zap.String("backend", id),
					zap.Error(err),
				)
			}
		} else {
			healthyCount++
		}

		results[id] = backendStatus
	}

	status := http.StatusOK
	overallStatus := StatusHealthy

	if healthyCount == 0 {
		status = http.StatusServiceUnavailable
		overallStatus = StatusUnhealthy
	} else if healthyCount < totalCount {
		overallStatus = StatusDegraded
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, map[string]interface{}{
		strStatus:             overallStatus,
		string(StatusHealthy): healthyCount,
		strTotal:              totalCount,
		"backends":            results,
		strTime:               time.Now().Unix(),
	})
}

// PerformCheck performs a health check and caches the result
func (h *Checker) PerformCheck(_ context.Context) *Check {
	h.mu.Lock()
	defer h.mu.Unlock()

	healthyCount := 0
	totalCount := len(h.backends)
	details := make(map[string]interface{})

	for id, backend := range h.backends {
		isHealthy := backend.IsHealthy()
		if isHealthy {
			healthyCount++
		}
		details[id] = map[string]interface{}{
			string(StatusHealthy): isHealthy,
			"type":                backend.Type(),
		}
	}

	var status Status
	if healthyCount == 0 {
		status = StatusUnhealthy
	} else if healthyCount < totalCount {
		status = StatusDegraded
	} else {
		status = StatusHealthy
	}

	check := &Check{
		Status: status,
		Details: map[string]interface{}{
			string(StatusHealthy): healthyCount,
			strTotal:              totalCount,
			"backends":            details,
		},
	}

	h.lastCheck = time.Now()
	h.lastResult = check

	return check
}

// GetLastCheck returns the last health check result
func (h *Checker) GetLastCheck() *Check {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastResult
}

// VersionHandler returns version information
func VersionHandler(version, gitCommit, buildTime string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeJSON(w, map[string]interface{}{
			"version":    version,
			"git_commit": gitCommit,
			"build_time": buildTime,
			strTime:      time.Now().Unix(),
		})
	}
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logging.Warn("failed to encode health response", zap.Error(err))
	}
}
