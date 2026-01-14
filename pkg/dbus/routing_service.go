package dbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"
)

const (
	routingInterface = "ie.fio.OllamaProxy.Routing"
	routingPath      = "/com/anthropic/OllamaProxy/Routing"
)

// RoutingService exposes routing statistics via D-Bus
type RoutingService struct {
	conn   *dbus.Conn
	router *router.Router
	props  *prop.Properties

	mu               sync.RWMutex
	totalRequests    int64
	recentDecisions  []RoutingDecisionInfo
	maxRecentDecisions int
}

// RoutingDecisionInfo represents a routing decision for D-Bus
type RoutingDecisionInfo struct {
	Timestamp        int64
	Backend          string
	Reason           string
	EstimatedPowerW  float64
	EstimatedLatency int32
}

// NewRoutingService creates a D-Bus service for routing statistics
func NewRoutingService(r *router.Router) (*RoutingService, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		// Try session bus if system bus fails
		conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to D-Bus: %w", err)
		}
	}

	svc := &RoutingService{
		conn:               conn,
		router:             r,
		recentDecisions:    make([]RoutingDecisionInfo, 0, 100),
		maxRecentDecisions: 100,
	}

	return svc, nil
}

// Start registers the D-Bus service
func (rs *RoutingService) Start() error {
	// Request name
	reply, err := rs.conn.RequestName(routingInterface,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request D-Bus name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name already taken")
	}

	// Export methods
	err = rs.conn.Export(rs, routingPath, routingInterface)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus object: %w", err)
	}

	// Export introspection
	intro := introspect.NewIntrospectable(&introspect.Node{
		Name: routingPath,
		Interfaces: []introspect.Interface{
			{
				Name: routingInterface,
				Methods: []introspect.Method{
					{
						Name: "GetRoutingStats",
						Args: []introspect.Arg{
							{Name: "stats", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetLastDecisions",
						Args: []introspect.Arg{
							{Name: "limit", Type: "u", Direction: "in"},
							{Name: "decisions", Type: "a(xssdi)", Direction: "out"},
						},
					},
					{
						Name: "GetBackendDistribution",
						Args: []introspect.Arg{
							{Name: "distribution", Type: "a{sd}", Direction: "out"},
						},
					},
					{
						Name: "SimulateRouting",
						Args: []introspect.Arg{
							{Name: "annotations", Type: "a{sv}", Direction: "in"},
							{Name: "backend", Type: "s", Direction: "out"},
							{Name: "reason", Type: "s", Direction: "out"},
						},
					},
				},
				Properties: []introspect.Property{
					{
						Name:   "TotalRequests",
						Type:   "x",
						Access: "read",
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "RequestRouted",
						Args: []introspect.Arg{
							{Name: "backend", Type: "s"},
							{Name: "reason", Type: "s"},
							{Name: "latency", Type: "i"},
						},
					},
				},
			},
		},
	})

	err = rs.conn.Export(intro, routingPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Setup properties
	rs.props, _ = prop.Export(rs.conn, routingPath, rs.makePropertyMap())

	logging.Logger.Info("D-Bus Routing service started",
		zap.String("interface", routingInterface),
	)
	return nil
}

// GetRoutingStats returns overall routing statistics (D-Bus method)
func (rs *RoutingService) GetRoutingStats() (map[string]dbus.Variant, *dbus.Error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	backends := rs.router.ListBackends()

	// Calculate total requests across all backends
	totalRequests := int64(0)
	totalSuccess := int64(0)
	totalErrors := int64(0)

	for _, backend := range backends {
		metrics := backend.GetMetrics()
		totalRequests += metrics.RequestCount
		totalSuccess += metrics.SuccessCount
		totalErrors += metrics.ErrorCount
	}

	successRate := float64(0)
	if totalRequests > 0 {
		successRate = float64(totalSuccess) / float64(totalRequests) * 100
	}

	stats := map[string]dbus.Variant{
		"total_requests": dbus.MakeVariant(totalRequests),
		"total_success":  dbus.MakeVariant(totalSuccess),
		"total_errors":   dbus.MakeVariant(totalErrors),
		"success_rate":   dbus.MakeVariant(successRate),
		"backend_count":  dbus.MakeVariant(int32(len(backends))),
	}

	return stats, nil
}

// GetLastDecisions returns recent routing decisions (D-Bus method)
func (rs *RoutingService) GetLastDecisions(limit uint32) ([]RoutingDecisionInfo, *dbus.Error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	count := int(limit)
	if count > len(rs.recentDecisions) {
		count = len(rs.recentDecisions)
	}

	if count == 0 {
		return []RoutingDecisionInfo{}, nil
	}

	// Return the most recent decisions
	start := len(rs.recentDecisions) - count
	result := make([]RoutingDecisionInfo, count)
	copy(result, rs.recentDecisions[start:])

	return result, nil
}

// GetBackendDistribution returns request distribution across backends (D-Bus method)
func (rs *RoutingService) GetBackendDistribution() (map[string]float64, *dbus.Error) {
	backends := rs.router.ListBackends()

	totalRequests := int64(0)
	requestCounts := make(map[string]int64)

	// Collect request counts per backend
	for _, backend := range backends {
		metrics := backend.GetMetrics()
		requestCounts[backend.ID()] = metrics.RequestCount
		totalRequests += metrics.RequestCount
	}

	// Calculate percentages
	distribution := make(map[string]float64)
	if totalRequests > 0 {
		for id, count := range requestCounts {
			distribution[id] = float64(count) / float64(totalRequests) * 100
		}
	}

	return distribution, nil
}

// SimulateRouting simulates a routing decision without executing it (D-Bus method)
func (rs *RoutingService) SimulateRouting(annotationsMap map[string]dbus.Variant) (string, string, *dbus.Error) {
	// Convert D-Bus variant map to annotations
	annotations := &backends.Annotations{
		Custom: make(map[string]string),
	}

	// Parse common annotation fields
	if target, ok := annotationsMap["target"]; ok {
		if targetStr, ok := target.Value().(string); ok {
			annotations.Target = targetStr
		}
	}

	if latencyCritical, ok := annotationsMap["latency_critical"]; ok {
		if lc, ok := latencyCritical.Value().(bool); ok {
			annotations.LatencyCritical = lc
		}
	}

	if powerEfficient, ok := annotationsMap["prefer_power_efficiency"]; ok {
		if pe, ok := powerEfficient.Value().(bool); ok {
			annotations.PreferPowerEfficiency = pe
		}
	}

	if maxLatency, ok := annotationsMap["max_latency_ms"]; ok {
		if ml, ok := maxLatency.Value().(int32); ok {
			annotations.MaxLatencyMs = ml
		}
	}

	if maxPower, ok := annotationsMap["max_power_watts"]; ok {
		if mp, ok := maxPower.Value().(int32); ok {
			annotations.MaxPowerWatts = mp
		}
	}

	// Simulate routing
	ctx := context.Background()
	decision, err := rs.router.RouteRequest(ctx, annotations)
	if err != nil {
		return "", "", dbus.MakeFailedError(fmt.Errorf("routing simulation failed: %w", err))
	}

	backendID := "unknown"
	if decision.Backend != nil {
		backendID = decision.Backend.ID()
	}

	return backendID, decision.Reason, nil
}

// RecordDecision records a routing decision (called by the application)
func (rs *RoutingService) RecordDecision(backend string, reason string, estimatedPowerW float64, estimatedLatency int32) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	decision := RoutingDecisionInfo{
		Timestamp:        time.Now().Unix(),
		Backend:          backend,
		Reason:           reason,
		EstimatedPowerW:  estimatedPowerW,
		EstimatedLatency: estimatedLatency,
	}

	rs.recentDecisions = append(rs.recentDecisions, decision)

	// Keep only the last N decisions
	if len(rs.recentDecisions) > rs.maxRecentDecisions {
		rs.recentDecisions = rs.recentDecisions[1:]
	}

	rs.totalRequests++

	// Update properties
	if rs.props != nil {
		rs.props.SetMust(routingInterface, "TotalRequests", rs.totalRequests)
	}

	// Emit signal (optional, can be high frequency)
	// Uncomment if you want real-time routing events
	// rs.conn.Emit(routingPath, routingInterface+".RequestRouted",
	//     backend, reason, estimatedLatency)
}

// makePropertyMap creates property map for D-Bus
func (rs *RoutingService) makePropertyMap() map[string]map[string]*prop.Prop {
	return map[string]map[string]*prop.Prop{
		routingInterface: {
			"TotalRequests": {
				Value:    rs.totalRequests,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
}

// Stop stops the D-Bus service
func (rs *RoutingService) Stop() {
	if rs.conn != nil {
		rs.conn.Close()
	}
}
