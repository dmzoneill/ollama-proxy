package dbus

import (
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/godbus/dbus/v5"
)

// TestRoutingServiceConstants tests the package constants
func TestRoutingServiceConstants(t *testing.T) {
	if routingInterface != "ie.fio.OllamaProxy.Routing" {
		t.Errorf("Expected routingInterface 'ie.fio.OllamaProxy.Routing', got '%s'", routingInterface)
	}
	if routingPath != "/com/anthropic/OllamaProxy/Routing" {
		t.Errorf("Expected routingPath '/com/anthropic/OllamaProxy/Routing', got '%s'", routingPath)
	}
}

// TestRoutingDecisionInfo tests the RoutingDecisionInfo structure
func TestRoutingDecisionInfo(t *testing.T) {
	now := time.Now().Unix()
	info := RoutingDecisionInfo{
		Timestamp:        now,
		Backend:          "backend1",
		Reason:           "power efficient",
		EstimatedPowerW:  10.5,
		EstimatedLatency: 100,
	}

	if info.Timestamp != now {
		t.Errorf("Expected Timestamp %d, got %d", now, info.Timestamp)
	}
	if info.Backend != "backend1" {
		t.Errorf("Expected Backend 'backend1', got '%s'", info.Backend)
	}
	if info.Reason != "power efficient" {
		t.Errorf("Expected Reason 'power efficient', got '%s'", info.Reason)
	}
	if info.EstimatedPowerW != 10.5 {
		t.Errorf("Expected EstimatedPowerW 10.5, got %f", info.EstimatedPowerW)
	}
	if info.EstimatedLatency != 100 {
		t.Errorf("Expected EstimatedLatency 100, got %d", info.EstimatedLatency)
	}
}

// TestNewRoutingService tests service initialization
func TestNewRoutingService(t *testing.T) {
	r := router.NewRouter(router.Config{
		DefaultBackendID: "test",
		PowerAware:       true,
	})

	// Note: This will fail in test environments without D-Bus
	svc, err := NewRoutingService(r)

	// In test environments, we expect D-Bus connection to fail
	if err == nil && svc != nil {
		// Service was created (D-Bus available)
		if svc.router != r {
			t.Error("Expected router to be set")
		}
		if svc.conn == nil {
			t.Error("Expected conn to be set when service is created")
		}
		if svc.maxRecentDecisions != 100 {
			t.Errorf("Expected maxRecentDecisions 100, got %d", svc.maxRecentDecisions)
		}
		if len(svc.recentDecisions) != 0 {
			t.Errorf("Expected empty recentDecisions, got %d", len(svc.recentDecisions))
		}
		// Clean up if service was created
		svc.Stop()
	}

	// If error occurred, it should be D-Bus connection error
	if err != nil {
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	}
}

// TestGetRoutingStatsMethod tests the GetRoutingStats D-Bus method
func TestGetRoutingStatsMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	// Create backends with different metrics
	backend1 := &mockBackend{
		id: "backend1",
		metrics: &backends.BackendMetrics{
			RequestCount: 100,
			SuccessCount: 95,
			ErrorCount:   5,
		},
	}
	backend2 := &mockBackend{
		id: "backend2",
		metrics: &backends.BackendMetrics{
			RequestCount: 50,
			SuccessCount: 48,
			ErrorCount:   2,
		},
	}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

	svc := &RoutingService{
		router:          r,
		recentDecisions: []RoutingDecisionInfo{},
	}

	// Test GetRoutingStats
	stats, dbusErr := svc.GetRoutingStats()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if stats == nil {
		t.Fatal("Expected stats map, got nil")
	}

	// Verify total requests (100 + 50 = 150)
	if stats["total_requests"].Value().(int64) != 150 {
		t.Errorf("Expected total_requests 150, got %v", stats["total_requests"].Value())
	}

	// Verify total success (95 + 48 = 143)
	if stats["total_success"].Value().(int64) != 143 {
		t.Errorf("Expected total_success 143, got %v", stats["total_success"].Value())
	}

	// Verify total errors (5 + 2 = 7)
	if stats["total_errors"].Value().(int64) != 7 {
		t.Errorf("Expected total_errors 7, got %v", stats["total_errors"].Value())
	}

	// Verify success rate (143/150 * 100 ≈ 95.33)
	successRate := stats["success_rate"].Value().(float64)
	expectedRate := 143.0 / 150.0 * 100.0
	if successRate < expectedRate-0.1 || successRate > expectedRate+0.1 {
		t.Errorf("Expected success_rate ~%.2f, got %.2f", expectedRate, successRate)
	}

	// Verify backend count
	if stats["backend_count"].Value().(int32) != 2 {
		t.Errorf("Expected backend_count 2, got %v", stats["backend_count"].Value())
	}
}

// TestGetRoutingStatsNoBackends tests GetRoutingStats with no backends
func TestGetRoutingStatsNoBackends(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router: r,
	}

	stats, dbusErr := svc.GetRoutingStats()

	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	// With no backends, should have zero stats
	if stats["total_requests"].Value().(int64) != 0 {
		t.Errorf("Expected total_requests 0, got %v", stats["total_requests"].Value())
	}
	if stats["success_rate"].Value().(float64) != 0 {
		t.Errorf("Expected success_rate 0, got %v", stats["success_rate"].Value())
	}
}

// TestGetLastDecisionsMethod tests the GetLastDecisions D-Bus method
func TestGetLastDecisionsMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router: r,
		recentDecisions: []RoutingDecisionInfo{
			{Timestamp: 1000, Backend: "backend1", Reason: "reason1", EstimatedPowerW: 10, EstimatedLatency: 50},
			{Timestamp: 2000, Backend: "backend2", Reason: "reason2", EstimatedPowerW: 15, EstimatedLatency: 100},
			{Timestamp: 3000, Backend: "backend3", Reason: "reason3", EstimatedPowerW: 20, EstimatedLatency: 150},
		},
	}

	// Test getting all decisions
	decisions, dbusErr := svc.GetLastDecisions(10)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if len(decisions) != 3 {
		t.Fatalf("Expected 3 decisions, got %d", len(decisions))
	}

	// Test getting limited decisions
	decisions, dbusErr = svc.GetLastDecisions(2)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if len(decisions) != 2 {
		t.Fatalf("Expected 2 decisions, got %d", len(decisions))
	}

	// Should get the most recent ones (backend2 and backend3)
	if decisions[0].Backend != "backend2" {
		t.Errorf("Expected first decision backend2, got %s", decisions[0].Backend)
	}
	if decisions[1].Backend != "backend3" {
		t.Errorf("Expected second decision backend3, got %s", decisions[1].Backend)
	}

	// Test getting zero decisions
	decisions, dbusErr = svc.GetLastDecisions(0)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if len(decisions) != 0 {
		t.Fatalf("Expected 0 decisions, got %d", len(decisions))
	}
}

// TestGetLastDecisionsEmpty tests GetLastDecisions with no decisions
func TestGetLastDecisionsEmpty(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router:          r,
		recentDecisions: []RoutingDecisionInfo{},
	}

	decisions, dbusErr := svc.GetLastDecisions(10)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if len(decisions) != 0 {
		t.Fatalf("Expected 0 decisions, got %d", len(decisions))
	}
}

// TestGetBackendDistributionMethod tests the GetBackendDistribution D-Bus method
func TestGetBackendDistributionMethod(t *testing.T) {
	r := router.NewRouter(router.Config{})

	// Create backends with different request counts
	backend1 := &mockBackend{
		id: "backend1",
		metrics: &backends.BackendMetrics{
			RequestCount: 80,
		},
	}
	backend2 := &mockBackend{
		id: "backend2",
		metrics: &backends.BackendMetrics{
			RequestCount: 20,
		},
	}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

	svc := &RoutingService{
		router: r,
	}

	distribution, dbusErr := svc.GetBackendDistribution()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if distribution == nil {
		t.Fatal("Expected distribution map, got nil")
	}

	// backend1 should have 80% (80 out of 100 total)
	if dist, ok := distribution["backend1"]; !ok {
		t.Error("backend1 not found in distribution")
	} else if dist < 79.9 || dist > 80.1 {
		t.Errorf("Expected backend1 distribution ~80%%, got %.2f%%", dist)
	}

	// backend2 should have 20%
	if dist, ok := distribution["backend2"]; !ok {
		t.Error("backend2 not found in distribution")
	} else if dist < 19.9 || dist > 20.1 {
		t.Errorf("Expected backend2 distribution ~20%%, got %.2f%%", dist)
	}
}

// TestGetBackendDistributionEmpty tests GetBackendDistribution with no requests
func TestGetBackendDistributionEmpty(t *testing.T) {
	r := router.NewRouter(router.Config{})

	backend := &mockBackend{
		id: "backend1",
		metrics: &backends.BackendMetrics{
			RequestCount: 0,
		},
	}

	r.RegisterBackend(backend)

	svc := &RoutingService{
		router: r,
	}

	distribution, dbusErr := svc.GetBackendDistribution()
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	// With no requests, distribution should be empty
	if len(distribution) != 0 {
		t.Errorf("Expected empty distribution, got %d entries", len(distribution))
	}
}

// TestSimulateRoutingMethod tests the SimulateRouting D-Bus method
func TestSimulateRoutingMethod(t *testing.T) {
	r := router.NewRouter(router.Config{
		DefaultBackendID: "backend1",
	})

	backend := &mockBackend{
		id:       "backend1",
		healthy:  true,
		name:     "Test Backend",
	}

	r.RegisterBackend(backend)

	svc := &RoutingService{
		router: r,
	}

	// Test basic simulation
	annotations := map[string]dbus.Variant{
		"target": dbus.MakeVariant("backend1"),
	}

	backendID, reason, dbusErr := svc.SimulateRouting(annotations)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}

	if backendID != "backend1" {
		t.Errorf("Expected backend1, got %s", backendID)
	}
	if reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Test with latency critical annotation
	annotations = map[string]dbus.Variant{
		"latency_critical": dbus.MakeVariant(true),
	}

	backendID, reason, dbusErr = svc.SimulateRouting(annotations)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if backendID == "" {
		t.Error("Expected backend to be selected")
	}

	// Test with power efficiency annotation
	annotations = map[string]dbus.Variant{
		"prefer_power_efficiency": dbus.MakeVariant(true),
	}

	backendID, reason, dbusErr = svc.SimulateRouting(annotations)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if backendID == "" {
		t.Error("Expected backend to be selected")
	}

	// Test with numeric constraints
	annotations = map[string]dbus.Variant{
		"max_latency_ms":   dbus.MakeVariant(int32(100)),
		"max_power_watts":  dbus.MakeVariant(int32(50)),
	}

	backendID, reason, dbusErr = svc.SimulateRouting(annotations)
	if dbusErr != nil {
		t.Fatalf("Expected no D-Bus error, got %v", dbusErr)
	}
	if backendID == "" {
		t.Error("Expected backend to be selected")
	}
}

// TestSimulateRoutingNoBackends tests SimulateRouting with no backends
func TestSimulateRoutingNoBackends(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router: r,
	}

	annotations := map[string]dbus.Variant{}

	_, _, dbusErr := svc.SimulateRouting(annotations)
	// Should fail with no backends
	if dbusErr == nil {
		t.Error("Expected D-Bus error with no backends")
	}
}

// TestRecordDecision tests the RecordDecision method
func TestRecordDecision(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router:             r,
		recentDecisions:    []RoutingDecisionInfo{},
		maxRecentDecisions: 3, // Small limit for testing
		totalRequests:      0,
	}

	// Record first decision
	svc.RecordDecision("backend1", "reason1", 10.5, 50)

	if len(svc.recentDecisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(svc.recentDecisions))
	}
	if svc.totalRequests != 1 {
		t.Errorf("Expected totalRequests 1, got %d", svc.totalRequests)
	}

	decision := svc.recentDecisions[0]
	if decision.Backend != "backend1" {
		t.Errorf("Expected Backend 'backend1', got '%s'", decision.Backend)
	}
	if decision.Reason != "reason1" {
		t.Errorf("Expected Reason 'reason1', got '%s'", decision.Reason)
	}
	if decision.EstimatedPowerW != 10.5 {
		t.Errorf("Expected EstimatedPowerW 10.5, got %f", decision.EstimatedPowerW)
	}
	if decision.EstimatedLatency != 50 {
		t.Errorf("Expected EstimatedLatency 50, got %d", decision.EstimatedLatency)
	}

	// Record more decisions
	svc.RecordDecision("backend2", "reason2", 15.0, 100)
	svc.RecordDecision("backend3", "reason3", 20.0, 150)

	if len(svc.recentDecisions) != 3 {
		t.Fatalf("Expected 3 decisions, got %d", len(svc.recentDecisions))
	}
	if svc.totalRequests != 3 {
		t.Errorf("Expected totalRequests 3, got %d", svc.totalRequests)
	}

	// Record one more to trigger limit
	svc.RecordDecision("backend4", "reason4", 25.0, 200)

	// Should still have 3 (oldest removed)
	if len(svc.recentDecisions) != 3 {
		t.Fatalf("Expected 3 decisions after limit, got %d", len(svc.recentDecisions))
	}
	if svc.totalRequests != 4 {
		t.Errorf("Expected totalRequests 4, got %d", svc.totalRequests)
	}

	// First decision should now be backend2 (backend1 removed)
	if svc.recentDecisions[0].Backend != "backend2" {
		t.Errorf("Expected first decision backend2, got %s", svc.recentDecisions[0].Backend)
	}
}

// TestRecordDecisionConcurrent tests RecordDecision thread safety
func TestRecordDecisionConcurrent(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router:             r,
		recentDecisions:    []RoutingDecisionInfo{},
		maxRecentDecisions: 100,
		totalRequests:      0,
	}

	// Record decisions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				svc.RecordDecision("backend1", "reason", 10.0, 50)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have recorded 100 decisions
	if svc.totalRequests != 100 {
		t.Errorf("Expected totalRequests 100, got %d", svc.totalRequests)
	}
}

// TestMakePropertyMapRouting tests the property map creation for routing service
func TestMakePropertyMapRouting(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router:        r,
		totalRequests: 42,
	}

	propMap := svc.makePropertyMap()

	if propMap == nil {
		t.Fatal("Expected property map, got nil")
	}

	if _, ok := propMap[routingInterface]; !ok {
		t.Fatal("Expected routingInterface in property map")
	}

	props := propMap[routingInterface]

	// Check TotalRequests
	if totalRequestsProp, ok := props["TotalRequests"]; ok {
		if totalRequestsProp.Value.(int64) != 42 {
			t.Errorf("Expected TotalRequests 42, got %v", totalRequestsProp.Value)
		}
		if totalRequestsProp.Writable {
			t.Error("Expected TotalRequests to be read-only")
		}
	} else {
		t.Error("TotalRequests property not found")
	}
}

// TestStopRoutingService tests the Stop method
func TestStopRoutingService(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router: r,
		conn:   nil, // No actual connection in test
	}

	// Should not panic with nil connection
	svc.Stop()
}

// TestRoutingDecisionInfoZeroValues tests RoutingDecisionInfo with zero values
func TestRoutingDecisionInfoZeroValues(t *testing.T) {
	info := RoutingDecisionInfo{}

	if info.Timestamp != 0 {
		t.Errorf("Expected Timestamp 0, got %d", info.Timestamp)
	}
	if info.Backend != "" {
		t.Errorf("Expected empty Backend, got '%s'", info.Backend)
	}
	if info.Reason != "" {
		t.Errorf("Expected empty Reason, got '%s'", info.Reason)
	}
	if info.EstimatedPowerW != 0 {
		t.Errorf("Expected EstimatedPowerW 0, got %f", info.EstimatedPowerW)
	}
	if info.EstimatedLatency != 0 {
		t.Errorf("Expected EstimatedLatency 0, got %d", info.EstimatedLatency)
	}
}

// TestRecordDecisionTimestamp tests that RecordDecision sets timestamp
func TestRecordDecisionTimestamp(t *testing.T) {
	r := router.NewRouter(router.Config{})

	svc := &RoutingService{
		router:             r,
		recentDecisions:    []RoutingDecisionInfo{},
		maxRecentDecisions: 10,
	}

	beforeTime := time.Now().Unix()
	svc.RecordDecision("backend1", "reason1", 10.0, 50)
	afterTime := time.Now().Unix()

	if len(svc.recentDecisions) != 1 {
		t.Fatal("Expected 1 decision")
	}

	timestamp := svc.recentDecisions[0].Timestamp
	if timestamp < beforeTime || timestamp > afterTime {
		t.Errorf("Timestamp %d not in expected range [%d, %d]", timestamp, beforeTime, afterTime)
	}
}
