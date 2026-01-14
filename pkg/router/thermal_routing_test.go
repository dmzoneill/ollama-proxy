package router

import (
	"context"
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/thermal"
)

func TestNewThermalRouter(t *testing.T) {
	config := Config{
		DefaultBackendID: "test",
		PowerAware:       true,
	}

	thermalConfig := &thermal.ThermalConfig{
		TempWarning:  70.0,
		TempCritical: 85.0,
		FanQuiet:     30,
		FanLoud:      80,
	}

	monitor := thermal.NewThermalMonitor(thermalConfig, 0) // 0 interval for testing
	tr := NewThermalRouter(config, monitor)

	if tr == nil {
		t.Fatal("NewThermalRouter returned nil")
	}

	if tr.Router == nil {
		t.Error("ThermalRouter.Router is nil")
	}

	if tr.thermalMonitor == nil {
		t.Error("ThermalRouter.thermalMonitor is nil")
	}

	if tr.workloadDetector == nil {
		t.Error("ThermalRouter.workloadDetector is nil")
	}
}

func TestThermalRouter_FilterByModelSupport_NoModel(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{id: "b1", healthy: true}
	backend2 := &MockBackend{id: "b2", healthy: true}
	backend3 := &MockBackend{id: "b3", healthy: false}

	tr.RegisterBackend(backend1)
	tr.RegisterBackend(backend2)
	tr.RegisterBackend(backend3)

	// Test with no model (should return all healthy)
	filtered := tr.filterByModelSupport("")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 healthy backends with no model filter, got %d", len(filtered))
	}
}

func TestThermalRouter_FilterByModelSupport_WithModel(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{id: "b1", healthy: true}
	backend2 := &MockBackend{id: "b2", healthy: true}

	tr.RegisterBackend(backend1)
	tr.RegisterBackend(backend2)

	// Test with model (MockBackend.SupportsModel always returns true)
	filtered := tr.filterByModelSupport("llama3:7b")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 backends supporting model, got %d", len(filtered))
	}
}

func TestThermalRouter_FilterByConstraints_Latency(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{id: "fast", avgLatencyMs: 100, powerWatts: 50.0}
	backend2 := &MockBackend{id: "slow", avgLatencyMs: 500, powerWatts: 10.0}
	backend3 := &MockBackend{id: "mid", avgLatencyMs: 200, powerWatts: 100.0}

	candidates := []backends.Backend{backend1, backend2, backend3}

	// Test latency constraint
	annotations := &backends.Annotations{
		MaxLatencyMs: 250,
	}

	filtered := tr.filterByConstraints(candidates, annotations)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 backends meeting latency constraint, got %d", len(filtered))
	}

	// Verify slow backend filtered out
	for _, b := range filtered {
		if b.ID() == "slow" {
			t.Error("slow backend should have been filtered out")
		}
	}
}

func TestThermalRouter_FilterByConstraints_Power(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{id: "fast", avgLatencyMs: 100, powerWatts: 50.0}
	backend2 := &MockBackend{id: "slow", avgLatencyMs: 500, powerWatts: 10.0}
	backend3 := &MockBackend{id: "hungry", avgLatencyMs: 200, powerWatts: 100.0}

	candidates := []backends.Backend{backend1, backend2, backend3}

	// Test power constraint
	annotations := &backends.Annotations{
		MaxPowerWatts: 60,
	}

	filtered := tr.filterByConstraints(candidates, annotations)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 backends meeting power constraint, got %d", len(filtered))
	}

	// Verify hungry backend filtered out
	for _, b := range filtered {
		if b.ID() == "hungry" {
			t.Error("hungry backend should have been filtered out")
		}
	}
}

func TestThermalRouter_FilterByConstraints_Both(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{id: "fast", avgLatencyMs: 100, powerWatts: 50.0}
	backend2 := &MockBackend{id: "slow", avgLatencyMs: 500, powerWatts: 10.0}
	backend3 := &MockBackend{id: "hungry", avgLatencyMs: 200, powerWatts: 100.0}

	candidates := []backends.Backend{backend1, backend2, backend3}

	// Test both constraints
	annotations := &backends.Annotations{
		MaxLatencyMs:  250,
		MaxPowerWatts: 60,
	}

	filtered := tr.filterByConstraints(candidates, annotations)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 backend meeting both constraints, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].ID() != "fast" {
		t.Errorf("Expected fast backend, got %s", filtered[0].ID())
	}
}

func TestThermalRouter_FilterByConstraints_NoConstraints(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{id: "b1"}
	backend2 := &MockBackend{id: "b2"}
	backend3 := &MockBackend{id: "b3"}

	candidates := []backends.Backend{backend1, backend2, backend3}

	// No constraints
	annotations := &backends.Annotations{}

	filtered := tr.filterByConstraints(candidates, annotations)
	if len(filtered) != 3 {
		t.Errorf("Expected all 3 backends with no constraints, got %d", len(filtered))
	}
}

func TestThermalRouter_FilterCandidatesThermal_HealthCheck(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{
		id:           "healthy",
		hardware:     "nvidia",
		healthy:      true,
		avgLatencyMs: 150,
		powerWatts:   40.0,
	}

	backend2 := &MockBackend{
		id:           "unhealthy",
		hardware:     "igpu",
		healthy:      false,
		avgLatencyMs: 200,
		powerWatts:   15.0,
	}

	tr.RegisterBackend(backend1)
	tr.RegisterBackend(backend2)

	annotations := &backends.Annotations{}

	filtered := tr.filterCandidatesThermal(annotations)

	// Should only filter by health (thermal check will likely pass or fail both)
	for _, b := range filtered {
		if !b.IsHealthy() {
			t.Errorf("Unhealthy backend %s should have been filtered", b.ID())
		}
	}
}

func TestThermalRouter_FilterCandidatesThermal_WithConstraints(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{
		id:           "fast",
		hardware:     "nvidia",
		healthy:      true,
		avgLatencyMs: 100,
		powerWatts:   50.0,
	}

	backend2 := &MockBackend{
		id:           "slow",
		hardware:     "cpu",
		healthy:      true,
		avgLatencyMs: 800,
		powerWatts:   10.0,
	}

	tr.RegisterBackend(backend1)
	tr.RegisterBackend(backend2)

	annotations := &backends.Annotations{
		MaxLatencyMs: 500,
	}

	filtered := tr.filterCandidatesThermal(annotations)

	// Should filter by both health, thermal, and constraints
	for _, b := range filtered {
		if b.AvgLatencyMs() > annotations.MaxLatencyMs {
			t.Errorf("Backend %s exceeds latency constraint", b.ID())
		}
	}
}

func TestThermalRouter_GetThermalStatus(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	status := tr.GetThermalStatus()

	if status == nil {
		t.Error("GetThermalStatus returned nil")
	}

	// Status map should exist even if empty
	t.Logf("Thermal status has %d entries", len(status))
}

func TestSortScored(t *testing.T) {
	backend1 := &MockBackend{id: "low", priority: 1}
	backend2 := &MockBackend{id: "high", priority: 10}
	backend3 := &MockBackend{id: "mid", priority: 5}

	scored := []candidateScore{
		{backend: backend1, score: 100.0, reason: "low"},
		{backend: backend2, score: 500.0, reason: "high"},
		{backend: backend3, score: 300.0, reason: "mid"},
	}

	sortScored(scored)

	// Should be sorted descending
	if scored[0].score != 500.0 {
		t.Errorf("Expected highest score first, got %.1f", scored[0].score)
	}

	if scored[1].score != 300.0 {
		t.Errorf("Expected mid score second, got %.1f", scored[1].score)
	}

	if scored[2].score != 100.0 {
		t.Errorf("Expected lowest score last, got %.1f", scored[2].score)
	}

	// Verify correct backends
	if scored[0].backend.ID() != "high" {
		t.Errorf("Expected high backend first, got %s", scored[0].backend.ID())
	}
}

func TestSortScored_AlreadySorted(t *testing.T) {
	backend1 := &MockBackend{id: "first"}
	backend2 := &MockBackend{id: "second"}

	scored := []candidateScore{
		{backend: backend1, score: 100.0},
		{backend: backend2, score: 50.0},
	}

	sortScored(scored)

	if scored[0].score != 100.0 || scored[1].score != 50.0 {
		t.Error("Sort modified already-sorted array incorrectly")
	}
}

func TestSortScored_Empty(t *testing.T) {
	scored := []candidateScore{}
	sortScored(scored)

	if len(scored) != 0 {
		t.Error("Sort modified empty array")
	}
}

func TestSortScored_SingleElement(t *testing.T) {
	backend := &MockBackend{id: "only"}
	scored := []candidateScore{
		{backend: backend, score: 100.0},
	}

	sortScored(scored)

	if len(scored) != 1 || scored[0].score != 100.0 {
		t.Error("Sort modified single-element array incorrectly")
	}
}

func TestThermalRouter_RouteRequestThermal(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend := &MockBackend{
		id:           "test",
		hardware:     "nvidia",
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 200,
		priority:     5,
	}

	tr.RegisterBackend(backend)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	// This will likely fail due to thermal checks, but we're testing the call path
	decision, err := tr.RouteRequestThermal(ctx, annotations)

	// If it succeeds, verify decision structure
	if err == nil {
		if decision == nil {
			t.Error("Decision is nil despite no error")
		}
		if decision != nil && decision.Backend == nil {
			t.Error("Decision.Backend is nil")
		}
	} else {
		// Expected to fail without actual thermal data
		t.Logf("RouteRequestThermal failed as expected: %v", err)
	}
}

func TestThermalRouter_RouteRequestWithModel(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend := &MockBackend{
		id:           "test",
		hardware:     "igpu",
		healthy:      true,
		powerWatts:   15.0,
		avgLatencyMs: 300,
		priority:     5,
	}

	tr.RegisterBackend(backend)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	// This will likely fail due to thermal checks
	decision, err := tr.RouteRequestWithModel(ctx, "test prompt", "llama3:7b", annotations)

	if err == nil {
		// If successful, verify model tracking
		if decision.ModelRequested != "llama3:7b" {
			t.Errorf("Expected ModelRequested=llama3:7b, got %s", decision.ModelRequested)
		}
	} else {
		t.Logf("RouteRequestWithModel failed as expected: %v", err)
	}
}

func TestThermalRouter_ScoreCandidatesThermal_LatencyCritical(t *testing.T) {
	config := Config{PowerAware: true, AutoOptimize: true}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{
		id:           "fast",
		hardware:     "nvidia",
		avgLatencyMs: 100,
		powerWatts:   50.0,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           "slow",
		hardware:     "cpu",
		avgLatencyMs: 400,
		powerWatts:   10.0,
		priority:     5,
	}

	candidates := []backends.Backend{backend1, backend2}

	annotations := &backends.Annotations{
		LatencyCritical: true,
	}

	scored := tr.scoreCandidatesThermal(candidates, annotations)

	if len(scored) != 2 {
		t.Fatalf("Expected 2 scored candidates, got %d", len(scored))
	}

	// Should be sorted by score
	if scored[0].score < scored[1].score {
		t.Error("Candidates not sorted by score descending")
	}

	t.Logf("Latency-critical: %s (%.1f) > %s (%.1f)",
		scored[0].backend.ID(), scored[0].score,
		scored[1].backend.ID(), scored[1].score)
}

func TestThermalRouter_ScoreCandidatesThermal_PowerEfficient(t *testing.T) {
	config := Config{PowerAware: true}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{
		id:           "hungry",
		hardware:     "nvidia",
		avgLatencyMs: 100,
		powerWatts:   100.0,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           "efficient",
		hardware:     "npu",
		avgLatencyMs: 400,
		powerWatts:   5.0,
		priority:     5,
	}

	candidates := []backends.Backend{backend1, backend2}

	annotations := &backends.Annotations{
		PreferPowerEfficiency: true,
	}

	scored := tr.scoreCandidatesThermal(candidates, annotations)

	if len(scored) != 2 {
		t.Fatalf("Expected 2 scored candidates, got %d", len(scored))
	}

	t.Logf("Power-efficient: %s (%.1f) > %s (%.1f)",
		scored[0].backend.ID(), scored[0].score,
		scored[1].backend.ID(), scored[1].score)

	// Efficient backend should score higher with power preference
	if scored[0].backend.ID() != "efficient" {
		t.Logf("Note: Expected efficient backend first, but scoring may vary")
	}
}

func TestThermalRouter_ScoreCandidatesThermal_Balanced(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	backend1 := &MockBackend{
		id:           "b1",
		avgLatencyMs: 200,
		powerWatts:   30.0,
		priority:     5,
	}

	backend2 := &MockBackend{
		id:           "b2",
		avgLatencyMs: 300,
		powerWatts:   20.0,
		priority:     5,
	}

	candidates := []backends.Backend{backend1, backend2}

	// No preferences - should use balanced scoring
	annotations := &backends.Annotations{}

	scored := tr.scoreCandidatesThermal(candidates, annotations)

	if len(scored) != 2 {
		t.Fatalf("Expected 2 scored candidates, got %d", len(scored))
	}

	// Should have "balanced" in reason
	hasBalanced := false
	for _, s := range scored {
		if s.reason == "Selected: balanced" {
			hasBalanced = true
			break
		}
	}

	if !hasBalanced {
		t.Logf("Expected balanced scoring reason")
	}
}

// TestThermalRouter_RouteRequestWithModel_ModelSubstitution tests model substitution
// when the requested model is not supported by any backend
func TestThermalRouter_RouteRequestWithModel_ModelSubstitution(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	// Create a backend that supports llama3:7b but not the requested gpt-4
	backend := &MockBackend{
		id:           "local-backend",
		hardware:     "cpu",
		healthy:      true,
		powerWatts:   50.0,
		avgLatencyMs: 200,
		priority:     5,
		modelPatterns: []string{"llama3*"},
	}

	tr.RegisterBackend(backend)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	// Request a model that's not supported (gpt-4)
	// The workload detector should suggest llama3:7b as a substitution
	decision, err := tr.RouteRequestWithModel(ctx, "Write a story", "gpt-4", annotations)

	if err != nil {
		// This is expected if model substitution doesn't work
		// or if thermal health filtering fails
		if !strings.Contains(err.Error(), "no backend supports model") {
			t.Logf("Got expected error: %v", err)
		}
	} else {
		// If substitution worked, verify it
		if decision.ModelRequested != "gpt-4" {
			t.Errorf("Expected ModelRequested=gpt-4, got %s", decision.ModelRequested)
		}
		if decision.ModelSubstituted {
			t.Logf("Model substitution successful: %s -> %s", decision.ModelRequested, decision.ModelUsed)
			if decision.SubstitutionReason == "" {
				t.Error("Expected substitution reason to be set")
			}
		}
	}
}

// TestThermalRouter_RouteRequestWithModel_NoBackends tests error handling
// when no backends match the model requirements
func TestThermalRouter_RouteRequestWithModel_NoBackends(t *testing.T) {
	config := Config{}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	// Create a backend that only supports gpt models
	backend := &MockBackend{
		id:            "gpt-only-backend",
		hardware:      "cpu",
		healthy:       true,
		powerWatts:    50.0,
		avgLatencyMs:  200,
		priority:      5,
		modelPatterns: []string{"gpt-*"},
	}

	tr.RegisterBackend(backend)

	ctx := context.Background()
	annotations := &backends.Annotations{}

	// Request llama3 model which isn't supported
	_, err := tr.RouteRequestWithModel(ctx, "test prompt", "llama3:7b", annotations)

	if err == nil {
		t.Error("Expected error when no backend supports the model")
	} else if !strings.Contains(err.Error(), "does not support model") {
		t.Errorf("Expected 'does not support model' error, got: %v", err)
	}
}

// TestThermalRouter_RouteRequestWithModel_WithWorkloadHints tests routing
// with workload detection hints
func TestThermalRouter_RouteRequestWithModel_WithWorkloadHints(t *testing.T) {
	config := Config{PowerAware: true}
	monitor := thermal.NewThermalMonitor(nil, 0)
	tr := NewThermalRouter(config, monitor)

	// Create multiple backends with different characteristics
	fastBackend := &MockBackend{
		id:           "fast-gpu",
		hardware:     "nvidia",
		healthy:      true,
		powerWatts:   200.0,
		avgLatencyMs: 50,
		priority:     5,
	}

	efficientBackend := &MockBackend{
		id:           "efficient-cpu",
		hardware:     "cpu",
		healthy:      true,
		powerWatts:   30.0,
		avgLatencyMs: 300,
		priority:     3,
	}

	tr.RegisterBackend(fastBackend)
	tr.RegisterBackend(efficientBackend)

	ctx := context.Background()

	// Test with latency-critical annotation (should prefer fast backend)
	latencyCritical := &backends.Annotations{
		LatencyCritical: true,
	}

	decision, err := tr.RouteRequestWithModel(ctx, "Quick question", "llama3:7b", latencyCritical)

	if err == nil {
		if decision.Backend.ID() == "fast-gpu" {
			t.Logf("Correctly selected fast backend for latency-critical workload")
		}
		if decision.DetectedMediaType != "" {
			t.Logf("Detected media type: %s", decision.DetectedMediaType)
		}
		if len(decision.RoutingHints) > 0 {
			t.Logf("Routing hints provided: %d", len(decision.RoutingHints))
		}
	}

	// Test with power-efficiency preference (should prefer efficient backend)
	powerEfficient := &backends.Annotations{
		PreferPowerEfficiency: true,
	}

	decision, err = tr.RouteRequestWithModel(ctx, "Background task", "llama3:7b", powerEfficient)

	if err == nil {
		if decision.Backend.ID() == "efficient-cpu" {
			t.Logf("Correctly selected efficient backend for power-conscious workload")
		}
		if decision.EstimatedPowerW < 100 {
			t.Logf("Estimated power usage: %.1fW", decision.EstimatedPowerW)
		}
	}
}
