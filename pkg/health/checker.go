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

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check result
type Check struct {
	Status  Status                 `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthChecker performs health checks
type HealthChecker struct {
	mu       sync.RWMutex
	backends map[string]backends.Backend
	lastCheck time.Time
	lastResult *Check
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		backends: make(map[string]backends.Backend),
	}
}

// RegisterBackend registers a backend for health checking
func (h *HealthChecker) RegisterBackend(backend backends.Backend) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.backends[backend.ID()] = backend
}

// LivenessCheck checks if the server is running
// This is the simplest check - if we can respond, we're alive
func (h *HealthChecker) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// ReadinessCheck checks if the server can serve traffic
// Returns 200 if at least one backend is healthy
func (h *HealthChecker) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        map[bool]string{true: "ready", false: "not_ready"}[healthyCount > 0],
		"healthy":       healthyCount,
		"total":         totalCount,
		"time":          time.Now().Unix(),
	})
}

// DeepHealthCheck performs a deep health check by actually running inference
func (h *HealthChecker) DeepHealthCheck(w http.ResponseWriter, r *http.Request) {
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
			"healthy":     err == nil,
			"latency_ms":  elapsed.Milliseconds(),
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  overallStatus,
		"healthy": healthyCount,
		"total":   totalCount,
		"backends": results,
		"time":    time.Now().Unix(),
	})
}

// PerformCheck performs a health check and caches the result
func (h *HealthChecker) PerformCheck(ctx context.Context) *Check {
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
			"healthy": isHealthy,
			"type":    backend.Type(),
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
			"healthy":  healthyCount,
			"total":    totalCount,
			"backends": details,
		},
	}

	h.lastCheck = time.Now()
	h.lastResult = check

	return check
}

// GetLastCheck returns the last health check result
func (h *HealthChecker) GetLastCheck() *Check {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastResult
}

// VersionHandler returns version information
func VersionHandler(version, gitCommit, buildTime string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"version":    version,
			"git_commit": gitCommit,
			"build_time": buildTime,
			"time":       time.Now().Unix(),
		})
	}
}
