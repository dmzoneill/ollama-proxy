package dbus

import (
	"context"
	"fmt"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"
)

const (
	backendsInterface = "ie.fio.OllamaProxy.Backends"
	backendsPath      = "/com/anthropic/OllamaProxy/Backends"
)

// BackendsService exposes backend monitoring via D-Bus
type BackendsService struct {
	conn   *dbus.Conn
	router *router.Router
	props  *prop.Properties
}

// BackendInfo represents backend information for D-Bus
type BackendInfo struct {
	ID           string
	Name         string
	Type         string
	Hardware     string
	Healthy      bool
	PowerWatts   float64
	AvgLatencyMs int32
}

// NewBackendsService creates a D-Bus service for backend monitoring
func NewBackendsService(r *router.Router) (*BackendsService, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		// Try session bus if system bus fails
		conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to D-Bus: %w", err)
		}
	}

	svc := &BackendsService{
		conn:   conn,
		router: r,
	}

	return svc, nil
}

// Start registers the D-Bus service
func (bs *BackendsService) Start() error {
	// Request name
	reply, err := bs.conn.RequestName(backendsInterface,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request D-Bus name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name already taken")
	}

	// Export methods
	err = bs.conn.Export(bs, backendsPath, backendsInterface)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus object: %w", err)
	}

	// Export introspection
	intro := introspect.NewIntrospectable(&introspect.Node{
		Name: backendsPath,
		Interfaces: []introspect.Interface{
			{
				Name: backendsInterface,
				Methods: []introspect.Method{
					{
						Name: "ListBackends",
						Args: []introspect.Arg{
							{Name: "backends", Type: "a(ssssbdi)", Direction: "out"},
						},
					},
					{
						Name: "GetBackendDetails",
						Args: []introspect.Arg{
							{Name: "id", Type: "s", Direction: "in"},
							{Name: "details", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetBackendMetrics",
						Args: []introspect.Arg{
							{Name: "id", Type: "s", Direction: "in"},
							{Name: "metrics", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetSupportedModels",
						Args: []introspect.Arg{
							{Name: "id", Type: "s", Direction: "in"},
							{Name: "models", Type: "as", Direction: "out"},
						},
					},
					{
						Name: "RefreshBackendStatus",
					},
				},
				Properties: []introspect.Property{
					{
						Name:   "BackendCount",
						Type:   "i",
						Access: "read",
					},
					{
						Name:   "HealthyCount",
						Type:   "i",
						Access: "read",
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "BackendStatusChanged",
						Args: []introspect.Arg{
							{Name: "id", Type: "s"},
							{Name: "healthy", Type: "b"},
						},
					},
				},
			},
		},
	})

	err = bs.conn.Export(intro, backendsPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Setup properties
	bs.props, _ = prop.Export(bs.conn, backendsPath, bs.makePropertyMap())

	logging.Logger.Info("D-Bus Backends service started",
		zap.String("interface", backendsInterface),
	)
	return nil
}

// ListBackends returns a list of all backends (D-Bus method)
func (bs *BackendsService) ListBackends() ([]BackendInfo, *dbus.Error) {
	backends := bs.router.ListBackends()
	result := make([]BackendInfo, len(backends))

	for i, backend := range backends {
		result[i] = BackendInfo{
			ID:           backend.ID(),
			Name:         backend.Name(),
			Type:         backend.Type(),
			Hardware:     backend.Hardware(),
			Healthy:      backend.IsHealthy(),
			PowerWatts:   backend.PowerWatts(),
			AvgLatencyMs: backend.AvgLatencyMs(),
		}
	}

	return result, nil
}

// GetBackendDetails returns detailed information about a specific backend (D-Bus method)
func (bs *BackendsService) GetBackendDetails(id string) (map[string]dbus.Variant, *dbus.Error) {
	backends := bs.router.ListBackends()

	for _, backend := range backends {
		if backend.ID() == id {
			details := map[string]dbus.Variant{
				"id":                  dbus.MakeVariant(backend.ID()),
				"name":                dbus.MakeVariant(backend.Name()),
				"type":                dbus.MakeVariant(backend.Type()),
				"hardware":            dbus.MakeVariant(backend.Hardware()),
				"healthy":             dbus.MakeVariant(backend.IsHealthy()),
				"power_watts":         dbus.MakeVariant(backend.PowerWatts()),
				"avg_latency_ms":      dbus.MakeVariant(backend.AvgLatencyMs()),
				"priority":            dbus.MakeVariant(backend.Priority()),
				"supports_generate":   dbus.MakeVariant(backend.SupportsGenerate()),
				"supports_stream":     dbus.MakeVariant(backend.SupportsStream()),
				"supports_embed":      dbus.MakeVariant(backend.SupportsEmbed()),
				"max_model_size_gb":   dbus.MakeVariant(backend.GetMaxModelSizeGB()),
				"supported_patterns":  dbus.MakeVariant(backend.GetSupportedModelPatterns()),
				"preferred_models":    dbus.MakeVariant(backend.GetPreferredModels()),
			}

			return details, nil
		}
	}

	return nil, dbus.MakeFailedError(fmt.Errorf("backend not found: %s", id))
}

// GetBackendMetrics returns performance metrics for a specific backend (D-Bus method)
func (bs *BackendsService) GetBackendMetrics(id string) (map[string]dbus.Variant, *dbus.Error) {
	backends := bs.router.ListBackends()

	for _, backend := range backends {
		if backend.ID() == id {
			metrics := backend.GetMetrics()

			result := map[string]dbus.Variant{
				"request_count":    dbus.MakeVariant(metrics.RequestCount),
				"success_count":    dbus.MakeVariant(metrics.SuccessCount),
				"error_count":      dbus.MakeVariant(metrics.ErrorCount),
				"total_latency_ms": dbus.MakeVariant(metrics.TotalLatencyMs),
				"avg_latency_ms":   dbus.MakeVariant(metrics.AvgLatencyMs),
				"error_rate":       dbus.MakeVariant(float64(metrics.ErrorRate)),
				"loaded_models":    dbus.MakeVariant(metrics.LoadedModels),
			}

			return result, nil
		}
	}

	return nil, dbus.MakeFailedError(fmt.Errorf("backend not found: %s", id))
}

// GetSupportedModels returns the list of supported models for a backend (D-Bus method)
func (bs *BackendsService) GetSupportedModels(id string) ([]string, *dbus.Error) {
	backends := bs.router.ListBackends()

	for _, backend := range backends {
		if backend.ID() == id {
			models, err := backend.ListModels(context.Background())
			if err != nil {
				// Return preferred models as fallback
				return backend.GetPreferredModels(), nil
			}
			return models, nil
		}
	}

	return nil, dbus.MakeFailedError(fmt.Errorf("backend not found: %s", id))
}

// RefreshBackendStatus triggers health check on all backends (D-Bus method)
func (bs *BackendsService) RefreshBackendStatus() *dbus.Error {
	backends := bs.router.ListBackends()
	ctx := context.Background()

	for _, backend := range backends {
		oldStatus := backend.IsHealthy()
		backend.HealthCheck(ctx)
		newStatus := backend.IsHealthy()

		// Emit signal if status changed
		if oldStatus != newStatus {
			bs.conn.Emit(backendsPath, backendsInterface+".BackendStatusChanged",
				backend.ID(), newStatus)
		}
	}

	// Update properties
	bs.updateProperties()

	return nil
}

// makePropertyMap creates property map for D-Bus
func (bs *BackendsService) makePropertyMap() map[string]map[string]*prop.Prop {
	backends := bs.router.ListBackends()
	healthyCount := 0
	for _, b := range backends {
		if b.IsHealthy() {
			healthyCount++
		}
	}

	return map[string]map[string]*prop.Prop{
		backendsInterface: {
			"BackendCount": {
				Value:    int32(len(backends)),
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"HealthyCount": {
				Value:    int32(healthyCount),
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
}

// updateProperties updates the property values
func (bs *BackendsService) updateProperties() {
	backends := bs.router.ListBackends()
	healthyCount := 0
	for _, b := range backends {
		if b.IsHealthy() {
			healthyCount++
		}
	}

	if bs.props != nil {
		bs.props.SetMust(backendsInterface, "BackendCount", int32(len(backends)))
		bs.props.SetMust(backendsInterface, "HealthyCount", int32(healthyCount))
	}
}

// Stop stops the D-Bus service
func (bs *BackendsService) Stop() {
	if bs.conn != nil {
		bs.conn.Close()
	}
}
