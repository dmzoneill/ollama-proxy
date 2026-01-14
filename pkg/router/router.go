package router

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	proxyerrors "github.com/daoneill/ollama-proxy/pkg/errors"
)

// RoutingDecision contains the result of routing logic
type RoutingDecision struct {
	Backend            backends.Backend
	Reason             string
	EstimatedPowerW    float64
	EstimatedLatencyMs int32
	Alternatives       []string

	// Model overrides
	ModelRequested     string // Model user requested
	ModelUsed          string // Model actually used
	ModelSubstituted   bool   // Was model changed for compatibility?
	SubstitutionReason string // Why was model substituted?

	// Workload detection
	DetectedMediaType  string   // Auto-detected workload type
	RoutingHints       []string // Reasoning chain for routing decision
}

// Router handles intelligent routing to backends
type Router struct {
	mu               sync.RWMutex
	backends         map[string]backends.Backend
	defaultBackendID string
	powerAware       bool
	autoOptimize     bool

	// Queue management for priority-aware routing
	queueMgr         *QueueManager
}

// Config for router initialization
type Config struct {
	DefaultBackendID string
	PowerAware       bool
	AutoOptimize     bool
}

// NewRouter creates a new router instance
func NewRouter(cfg Config) *Router {
	return &Router{
		backends:         make(map[string]backends.Backend),
		defaultBackendID: cfg.DefaultBackendID,
		powerAware:       cfg.PowerAware,
		autoOptimize:     cfg.AutoOptimize,
		queueMgr:         NewQueueManager(),
	}
}

// RegisterBackend adds a backend to the router
func (r *Router) RegisterBackend(backend backends.Backend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := backend.ID()
	if _, exists := r.backends[id]; exists {
		return fmt.Errorf("backend %s already registered", id)
	}

	r.backends[id] = backend
	return nil
}

// GetBackend retrieves a backend by ID
func (r *Router) GetBackend(id string) (backends.Backend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, exists := r.backends[id]
	return backend, exists
}

// ListBackends returns all registered backends
func (r *Router) ListBackends() []backends.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]backends.Backend, 0, len(r.backends))
	for _, b := range r.backends {
		list = append(list, b)
	}
	return list
}

// RouteRequest intelligently selects a backend based on annotations
func (r *Router) RouteRequest(ctx context.Context, annotations *backends.Annotations) (*RoutingDecision, error) {
	// Add deadline if not set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Check context before expensive operations
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("routing cancelled: %w", ctx.Err())
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var selectedBackend backends.Backend
	var reason string

	// If specific target requested, try that first
	if annotations.Target != "" && annotations.Target != "auto" {
		if backend, exists := r.backends[annotations.Target]; exists {
			if backend.IsHealthy() {
				selectedBackend = backend
				reason = fmt.Sprintf("Explicit target: %s", annotations.Target)
			}
			// Target unhealthy, fall through to auto-selection
		}
	}

	// Auto-select if no target or target unhealthy
	if selectedBackend == nil {
		candidates := r.filterCandidates(annotations)
		if len(candidates) == 0 {
			// Build detailed error message with constraints
			var constraints []string
			if annotations.MaxLatencyMs > 0 {
				constraints = append(constraints, fmt.Sprintf("latency<%dms", annotations.MaxLatencyMs))
			}
			if annotations.MaxPowerWatts > 0 {
				constraints = append(constraints, fmt.Sprintf("power<%dW", annotations.MaxPowerWatts))
			}
			if annotations.MediaType != "" {
				constraints = append(constraints, fmt.Sprintf("media=%s", annotations.MediaType))
			}
			if annotations.Target != "" {
				constraints = append(constraints, fmt.Sprintf("target=%s", annotations.Target))
			}

			// Count healthy backends
			healthyCount := 0
			for _, backend := range r.backends {
				if backend.IsHealthy() {
					healthyCount++
				}
			}

			return nil, proxyerrors.NewNoBackendsError(len(r.backends), healthyCount, constraints)
		}

		// Score and rank candidates
		scored := r.scoreCandidates(candidates, annotations)

		// Select best candidate
		best := scored[0]
		selectedBackend = best.backend
		reason = best.reason
	}

	// Mark request start in queue
	r.queueMgr.MarkRequestStart(selectedBackend.ID(), annotations.Priority)

	// Wrap backend with queue tracking
	trackedBackend := &QueueTrackingBackend{
		Backend:  selectedBackend,
		queueMgr: r.queueMgr,
		priority: annotations.Priority,
	}

	return &RoutingDecision{
		Backend:            trackedBackend,
		Reason:             reason,
		EstimatedPowerW:    selectedBackend.PowerWatts(),
		EstimatedLatencyMs: selectedBackend.AvgLatencyMs(),
		Alternatives:       r.getAlternatives(selectedBackend.ID()),
	}, nil
}

// candidateScore holds backend with its score
type candidateScore struct {
	backend backends.Backend
	score   float64
	reason  string
}

// filterCandidates returns backends that meet basic requirements
func (r *Router) filterCandidates(annotations *backends.Annotations) []backends.Backend {
	var candidates []backends.Backend

	for _, backend := range r.backends {
		// Must be healthy
		if !backend.IsHealthy() {
			continue
		}

		// Check max latency constraint
		if annotations.MaxLatencyMs > 0 {
			if backend.AvgLatencyMs() > annotations.MaxLatencyMs {
				continue
			}
		}

		// Check max power constraint
		if annotations.MaxPowerWatts > 0 {
			if backend.PowerWatts() > float64(annotations.MaxPowerWatts) {
				continue
			}
		}

		candidates = append(candidates, backend)
	}

	return candidates
}

// scoreCandidates assigns scores to candidates based on preferences
func (r *Router) scoreCandidates(candidates []backends.Backend, annotations *backends.Annotations) []candidateScore {
	scored := make([]candidateScore, 0, len(candidates))

	for _, backend := range candidates {
		score := 0.0
		reasons := []string{}

		// Base score from backend priority
		score += float64(backend.Priority()) * 10.0

		// Latency optimization
		if annotations.LatencyCritical || r.autoOptimize {
			// Lower latency = higher score
			// NVIDIA (~150ms) gets ~850 points
			// NPU (~800ms) gets ~200 points
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			score += latencyScore * 2 // Weight latency heavily
			if annotations.LatencyCritical {
				reasons = append(reasons, "latency-critical")
			}
		}

		// Power efficiency optimization
		if annotations.PreferPowerEfficiency || r.powerAware {
			// Lower power = higher score
			// NPU (3W) gets ~970 points
			// NVIDIA (55W) gets ~450 points
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += powerScore * 1.5 // Weight power efficiency
			if annotations.PreferPowerEfficiency {
				reasons = append(reasons, "power-efficient")
			}
		}

		// If no specific preference, use balanced scoring
		if !annotations.LatencyCritical && !annotations.PreferPowerEfficiency {
			// Balanced: consider both latency and power
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += (latencyScore + powerScore) / 2
			reasons = append(reasons, "balanced")
		}

		// Queue depth penalty - avoid congested backends
		queueDepth := r.queueMgr.GetQueueDepth(backend.ID(), annotations.Priority)
		queuePenalty := float64(queueDepth) * 50.0 // -50 points per pending request
		score -= queuePenalty
		if queueDepth > 0 {
			reasons = append(reasons, fmt.Sprintf("queue-depth-%d", queueDepth))
		}

		// Priority boost for critical requests
		if annotations.Priority == backends.PriorityCritical {
			score += 500.0 // Strong boost for voice/realtime
			reasons = append(reasons, "critical-priority")
		} else if annotations.Priority == backends.PriorityHigh {
			score += 200.0 // Moderate boost for high priority
			reasons = append(reasons, "high-priority")
		}

		if len(reasons) == 0 {
			reasons = append(reasons, "default-scoring")
		}

		scored = append(scored, candidateScore{
			backend: backend,
			score:   score,
			reason:  fmt.Sprintf("Selected: %s", reasons[0]),
		})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

// getAlternatives returns IDs of other backends (excluding the given one)
func (r *Router) getAlternatives(excludeID string) []string {
	alternatives := []string{}
	for id, backend := range r.backends {
		if id != excludeID && backend.IsHealthy() {
			alternatives = append(alternatives, id)
		}
	}
	return alternatives
}

// FallbackRequest tries alternative backends if primary fails
func (r *Router) FallbackRequest(ctx context.Context, excludeBackends []string, annotations *backends.Annotations) (*RoutingDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Build exclusion set
	exclude := make(map[string]bool)
	for _, id := range excludeBackends {
		exclude[id] = true
	}

	// Find healthy backends not in exclusion list
	candidates := []backends.Backend{}
	healthyCount := 0
	for id, backend := range r.backends {
		if exclude[id] {
			continue
		}
		if backend.IsHealthy() {
			healthyCount++
			candidates = append(candidates, backend)
		}
	}

	if len(candidates) == 0 {
		excludedList := make([]string, 0, len(excludeBackends))
		excludedList = append(excludedList, excludeBackends...)
		constraints := []string{fmt.Sprintf("excluded: %s", strings.Join(excludedList, ", "))}
		return nil, proxyerrors.NewNoBackendsError(len(r.backends), healthyCount, constraints)
	}

	// Score and select
	scored := r.scoreCandidates(candidates, annotations)
	best := scored[0]

	return &RoutingDecision{
		Backend:            best.backend,
		Reason:             fmt.Sprintf("Fallback: %s", best.reason),
		EstimatedPowerW:    best.backend.PowerWatts(),
		EstimatedLatencyMs: best.backend.AvgLatencyMs(),
		Alternatives:       r.getAlternatives(best.backend.ID()),
	}, nil
}

// HealthCheckAll performs health checks on all backends
func (r *Router) HealthCheckAll(ctx context.Context) map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]bool)
	for id, backend := range r.backends {
		results[id] = backend.IsHealthy()
	}
	return results
}

// Stats returns current router statistics
func (r *Router) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	healthyCount := 0
	for _, backend := range r.backends {
		if backend.IsHealthy() {
			healthyCount++
		}
	}

	return map[string]interface{}{
		"total_backends":   len(r.backends),
		"healthy_backends": healthyCount,
		"timestamp":        time.Now().Unix(),
	}
}
