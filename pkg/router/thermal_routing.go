package router

import (
	"context"
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	proxyerrors "github.com/daoneill/ollama-proxy/pkg/errors"
	"github.com/daoneill/ollama-proxy/pkg/thermal"
	"github.com/daoneill/ollama-proxy/pkg/workload"
)

// ThermalRouter extends Router with thermal awareness
type ThermalRouter struct {
	*Router
	thermalMonitor   *thermal.ThermalMonitor
	workloadDetector *workload.Detector
}

// NewThermalRouter creates a router with thermal monitoring
func NewThermalRouter(cfg Config, thermalMonitor *thermal.ThermalMonitor) *ThermalRouter {
	return &ThermalRouter{
		Router:           NewRouter(cfg),
		thermalMonitor:   thermalMonitor,
		workloadDetector: workload.NewDetector(),
	}
}

// RouteRequestThermal routes with thermal awareness
func (tr *ThermalRouter) RouteRequestThermal(ctx context.Context, annotations *backends.Annotations) (*RoutingDecision, error) {
	// Backward compatibility - route without model info
	return tr.RouteRequestWithModel(ctx, "", "", annotations)
}

// RouteRequestWithModel routes with full model-aware and thermal-aware logic
func (tr *ThermalRouter) RouteRequestWithModel(
	ctx context.Context,
	prompt string,
	requestedModel string,
	annotations *backends.Annotations,
) (*RoutingDecision, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// Step 1: Detect workload type and get routing hints
	hints := tr.workloadDetector.GetRoutingHints(prompt, requestedModel, annotations)
	reasoningChain := hints.ReasoningChain

	// Step 2: Determine which model to use
	modelToUse := requestedModel
	modelSubstituted := false
	substitutionReason := ""

	// Step 3: Filter candidates by model compatibility FIRST
	modelCompatible := tr.filterByModelSupport(modelToUse)
	if len(modelCompatible) == 0 {
		// No backend supports this model - try substitution
		if hints.PreferredModel != "" && hints.PreferredModel != modelToUse {
			modelToUse = hints.PreferredModel
			modelSubstituted = true
			substitutionReason = fmt.Sprintf("Model %s not supported, using %s for %s workload",
				requestedModel, modelToUse, hints.DetectedMediaType)
			reasoningChain = append(reasoningChain, substitutionReason)

			// Retry with substituted model
			modelCompatible = tr.filterByModelSupport(modelToUse)
		}

		if len(modelCompatible) == 0 {
			reason := fmt.Sprintf("requested: %s, substitution attempted: %s", requestedModel, modelToUse)
			return nil, proxyerrors.NewBackendUnsupportedError("all", requestedModel, reason)
		}
	}

	reasoningChain = append(reasoningChain,
		fmt.Sprintf("Model compatible backends: %d", len(modelCompatible)))

	// Step 4: Filter by thermal state
	thermalHealthy := tr.filterByThermalHealth(modelCompatible)
	if len(thermalHealthy) == 0 {
		constraints := []string{fmt.Sprintf("model=%s", modelToUse), "thermal limits exceeded"}
		return nil, proxyerrors.NewNoBackendsError(len(modelCompatible), 0, constraints)
	}

	reasoningChain = append(reasoningChain,
		fmt.Sprintf("Thermally healthy backends: %d", len(thermalHealthy)))

	// Step 5: Apply constraints (latency, power)
	constrained := tr.filterByConstraints(thermalHealthy, annotations)
	if len(constrained) == 0 {
		var constraints []string
		constraints = append(constraints, fmt.Sprintf("model=%s", modelToUse))
		if annotations.MaxLatencyMs > 0 {
			constraints = append(constraints, fmt.Sprintf("latency<%dms", annotations.MaxLatencyMs))
		}
		if annotations.MaxPowerWatts > 0 {
			constraints = append(constraints, fmt.Sprintf("power<%dW", annotations.MaxPowerWatts))
		}
		return nil, proxyerrors.NewNoBackendsError(len(thermalHealthy), len(thermalHealthy), constraints)
	}

	// Step 6: Score remaining candidates
	scored := tr.scoreCandidatesWithHints(constrained, annotations, hints)

	// Select best candidate
	best := scored[0]
	thermalState := tr.thermalMonitor.GetState(best.backend.Hardware())

	thermalInfo := ""
	if thermalState != nil {
		thermalInfo = fmt.Sprintf(" [%.1f°C, fan:%d%%]",
			thermalState.Temperature, thermalState.FanPercent)
	}

	reasoningChain = append(reasoningChain,
		fmt.Sprintf("Selected: %s%s", best.backend.ID(), thermalInfo))

	return &RoutingDecision{
		Backend:            best.backend,
		Reason:             best.reason + thermalInfo,
		EstimatedPowerW:    best.backend.PowerWatts(),
		EstimatedLatencyMs: best.backend.AvgLatencyMs(),
		Alternatives:       tr.getAlternatives(best.backend.ID()),
		ModelRequested:     requestedModel,
		ModelUsed:          modelToUse,
		ModelSubstituted:   modelSubstituted,
		SubstitutionReason: substitutionReason,
		DetectedMediaType:  string(hints.DetectedMediaType),
		RoutingHints:       reasoningChain,
	}, nil
}

// filterByModelSupport filters backends that support the model
func (tr *ThermalRouter) filterByModelSupport(model string) []backends.Backend {
	if model == "" {
		// No model specified, return all
		var all []backends.Backend
		for _, b := range tr.backends {
			if b.IsHealthy() {
				all = append(all, b)
			}
		}
		return all
	}

	var compatible []backends.Backend
	for _, backend := range tr.backends {
		if !backend.IsHealthy() {
			continue
		}

		if backend.SupportsModel(model) {
			compatible = append(compatible, backend)
		}
	}

	return compatible
}

// filterByThermalHealth filters backends that are thermally healthy
func (tr *ThermalRouter) filterByThermalHealth(candidates []backends.Backend) []backends.Backend {
	var healthy []backends.Backend

	for _, backend := range candidates {
		hardware := backend.Hardware()
		if canUse, _ := tr.thermalMonitor.CanUse(hardware); canUse {
			healthy = append(healthy, backend)
		}
	}

	return healthy
}

// filterByConstraints filters by latency and power constraints
func (tr *ThermalRouter) filterByConstraints(candidates []backends.Backend, annotations *backends.Annotations) []backends.Backend {
	var filtered []backends.Backend

	for _, backend := range candidates {
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

		filtered = append(filtered, backend)
	}

	return filtered
}

// scoreCandidatesWithHints scores candidates using workload hints
func (tr *ThermalRouter) scoreCandidatesWithHints(
	candidates []backends.Backend,
	annotations *backends.Annotations,
	hints *workload.RoutingHints,
) []candidateScore {
	scored := make([]candidateScore, 0, len(candidates))

	// Check if system wants quiet operation
	preferQuiet := tr.thermalMonitor.ShouldPreferQuiet()

	for _, backend := range candidates {
		score := 0.0
		reasons := []string{}

		// Base score from backend priority
		score += float64(backend.Priority()) * 10.0

		// Workload-specific preferences
		if hints.PreferLowLatency {
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			score += latencyScore * 2.5 // Strong preference for low latency
			reasons = append(reasons, "low-latency-workload")
		}

		if hints.PreferLowPower {
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += powerScore * 2.0 // Strong preference for power efficiency
			reasons = append(reasons, "power-efficient-workload")
		}

		// Annotation overrides
		if annotations.LatencyCritical {
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			score += latencyScore * 2
			reasons = append(reasons, "latency-critical")
		}

		if annotations.PreferPowerEfficiency || tr.powerAware {
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += powerScore * 1.5
			reasons = append(reasons, "power-efficient")
		}

		// THERMAL PENALTY
		thermalPenalty := tr.thermalMonitor.GetThermalPenalty(backend.Hardware())
		score -= thermalPenalty

		if thermalPenalty > 100 {
			reasons = append(reasons, "thermal-penalty")
		}

		// Quiet mode preference
		if preferQuiet {
			thermalState := tr.thermalMonitor.GetState(backend.Hardware())
			if thermalState != nil && thermalState.FanPercent < 40 {
				score += 200 // Bonus for quiet backends
				reasons = append(reasons, "quiet-mode")
			}
		}

		// If no specific preference, use balanced scoring
		if !annotations.LatencyCritical && !annotations.PreferPowerEfficiency && !hints.PreferLowLatency && !hints.PreferLowPower {
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += (latencyScore + powerScore) / 2
			reasons = append(reasons, "balanced")
		}

		if len(reasons) == 0 {
			reasons = append(reasons, "default-scoring")
		}

		scored = append(scored, candidateScore{
			backend: backend,
			score:   score,
			reason:  fmt.Sprintf("%s", reasons[0]),
		})
	}

	// Sort by score descending
	sortScored(scored)

	return scored
}

// filterCandidatesThermal filters backends by health and thermal state
func (tr *ThermalRouter) filterCandidatesThermal(annotations *backends.Annotations) []backends.Backend {
	var candidates []backends.Backend

	for _, backend := range tr.backends {
		// Must be healthy
		if !backend.IsHealthy() {
			continue
		}

		// Check thermal state
		hardware := backend.Hardware()
		if canUse, _ := tr.thermalMonitor.CanUse(hardware); !canUse {
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

// scoreCandidatesThermal scores backends including thermal penalty
func (tr *ThermalRouter) scoreCandidatesThermal(candidates []backends.Backend, annotations *backends.Annotations) []candidateScore {
	scored := make([]candidateScore, 0, len(candidates))

	// Check if system wants quiet operation
	preferQuiet := tr.thermalMonitor.ShouldPreferQuiet()

	for _, backend := range candidates {
		score := 0.0
		reasons := []string{}

		// Base score from backend priority
		score += float64(backend.Priority()) * 10.0

		// Latency optimization
		if annotations.LatencyCritical || tr.autoOptimize {
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			score += latencyScore * 2
			if annotations.LatencyCritical {
				reasons = append(reasons, "latency-critical")
			}
		}

		// Power efficiency optimization
		if annotations.PreferPowerEfficiency || tr.powerAware {
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += powerScore * 1.5
			if annotations.PreferPowerEfficiency {
				reasons = append(reasons, "power-efficient")
			}
		}

		// THERMAL PENALTY - the new addition!
		thermalPenalty := tr.thermalMonitor.GetThermalPenalty(backend.Hardware())
		score -= thermalPenalty

		if thermalPenalty > 100 {
			reasons = append(reasons, "thermal-penalty")
		}

		// Quiet mode preference
		if preferQuiet {
			thermalState := tr.thermalMonitor.GetState(backend.Hardware())
			if thermalState != nil && thermalState.FanPercent < 40 {
				score += 200 // Bonus for quiet backends
				reasons = append(reasons, "quiet-mode")
			}
		}

		// If no specific preference, use balanced scoring
		if !annotations.LatencyCritical && !annotations.PreferPowerEfficiency {
			latencyScore := 1000.0 - float64(backend.AvgLatencyMs())
			powerScore := 1000.0 - (backend.PowerWatts() * 10)
			score += (latencyScore + powerScore) / 2
			reasons = append(reasons, "balanced")
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
	sortScored(scored)

	return scored
}

// findThermalAlternative finds alternative when preferred backend is too hot
func (tr *ThermalRouter) findThermalAlternative(
	ctx context.Context,
	annotations *backends.Annotations,
	overheatedHardware string,
	thermalReason string,
) (*RoutingDecision, error) {

	// Get coolest alternative
	var alternatives []string
	for _, backend := range tr.backends {
		hw := backend.Hardware()
		if hw != overheatedHardware {
			if canUse, _ := tr.thermalMonitor.CanUse(hw); canUse {
				alternatives = append(alternatives, hw)
			}
		}
	}

	if len(alternatives) == 0 {
		constraints := []string{fmt.Sprintf("hardware=%s is overheated", overheatedHardware), thermalReason}
		healthyCount := 0
		for _, backend := range tr.backends {
			if backend.IsHealthy() {
				healthyCount++
			}
		}
		return nil, proxyerrors.NewNoBackendsError(len(tr.backends), healthyCount, constraints)
	}

	coolest := tr.thermalMonitor.GetCoolestBackend(alternatives)

	// Find backend with this hardware
	for _, backend := range tr.backends {
		if backend.Hardware() == coolest && backend.IsHealthy() {
			thermalState := tr.thermalMonitor.GetState(coolest)
			reason := fmt.Sprintf("Thermal override: %s too hot, using %s (%.1f°C)",
				overheatedHardware, coolest, thermalState.Temperature)

			return &RoutingDecision{
				Backend:            backend,
				Reason:             reason,
				EstimatedPowerW:    backend.PowerWatts(),
				EstimatedLatencyMs: backend.AvgLatencyMs(),
				Alternatives:       []string{},
			}, nil
		}
	}

	constraints := []string{fmt.Sprintf("hardware=%s", coolest), "no healthy backend found"}
	return nil, proxyerrors.NewNoBackendsError(len(tr.backends), 0, constraints)
}

// GetThermalStatus returns current thermal status for all backends
func (tr *ThermalRouter) GetThermalStatus() map[string]*thermal.ThermalState {
	return tr.thermalMonitor.GetAllStates()
}

// Helper to sort scored candidates
func sortScored(scored []candidateScore) {
	// Simple bubble sort for clarity (use sort.Slice in production)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}
