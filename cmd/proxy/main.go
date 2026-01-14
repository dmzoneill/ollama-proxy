package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/daoneill/ollama-proxy/api/gen/go"
	devicev1 "github.com/daoneill/ollama-proxy/api/proto/device/v1"
	"github.com/daoneill/ollama-proxy/pkg/auth"
	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/backends/ollama"
	"github.com/daoneill/ollama-proxy/pkg/backends/openvino"
	"github.com/daoneill/ollama-proxy/pkg/config"
	dbusPkg "github.com/daoneill/ollama-proxy/pkg/dbus"
	"github.com/daoneill/ollama-proxy/pkg/device"
	"github.com/daoneill/ollama-proxy/pkg/device/virtual"
	"github.com/daoneill/ollama-proxy/pkg/efficiency"
	openaihttp "github.com/daoneill/ollama-proxy/pkg/http/openai"
	websockethttp "github.com/daoneill/ollama-proxy/pkg/http/websocket"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/middleware"
	"github.com/daoneill/ollama-proxy/pkg/pipeline"
	"github.com/daoneill/ollama-proxy/pkg/ratelimit"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/daoneill/ollama-proxy/pkg/server"
	"github.com/daoneill/ollama-proxy/pkg/settings"
	"github.com/daoneill/ollama-proxy/pkg/thermal"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
)

// CLI flags
var (
	configPath = flag.String("config", "config/config.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "", "Log level (debug, info, warn, error) - overrides config")
	grpcPort   = flag.Int("grpc-port", 0, "gRPC port - overrides config")
	httpPort   = flag.Int("http-port", 0, "HTTP port - overrides config")
)

// Config structure matching config.yaml

func main() {
	// Parse CLI flags
	flag.Parse()

	// Initialize basic logging first (will be reconfigured after config load)
	if err := logging.InitLogger("info", false); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logging.Logger.Info("Starting Ollama Compute Proxy",
		zap.String("component", "main"),
		zap.Bool("thermal_monitoring", true),
		zap.String("config_path", *configPath),
	)

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		logging.Logger.Fatal("Failed to load config",
			zap.Error(err),
			zap.String("config_path", *configPath),
		)
	}

	// Apply environment variable overrides (before CLI flags)
	config.ApplyEnvOverrides(cfg)

	// Apply CLI flag overrides
	if *logLevel != "" {
		cfg.Monitoring.LogLevel = *logLevel
		logging.Logger.Info("Log level overridden by CLI flag",
			zap.String("log_level", *logLevel),
		)
	}
	if *grpcPort > 0 {
		cfg.Server.GRPCPort = *grpcPort
		logging.Logger.Info("gRPC port overridden by CLI flag",
			zap.Int("grpc_port", *grpcPort),
		)
	}
	if *httpPort > 0 {
		cfg.Server.HTTPPort = *httpPort
		logging.Logger.Info("HTTP port overridden by CLI flag",
			zap.Int("http_port", *httpPort),
		)
	}

	// Validate configuration
	configValidator := cfg
	if err := config.ValidateConfig(configValidator); err != nil {
		logging.Logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// Reconfigure logger with config settings
	if err := logging.InitLogger(cfg.Monitoring.LogLevel, true); err != nil {
		logging.Logger.Error("Failed to reconfigure logger", zap.Error(err))
	}

	logging.Logger.Info("Configuration validated successfully")

	ctx := context.Background()

	// Initialize thermal monitor
	var thermalMonitor *thermal.ThermalMonitor
	if cfg.Thermal.Enabled {
		updateInterval := 5 * time.Second
		if cfg.Thermal.UpdateInterval != "" {
			var err error
			updateInterval, err = time.ParseDuration(cfg.Thermal.UpdateInterval)
			if err != nil {
				logging.Logger.Warn("Invalid thermal update interval, using default",
					zap.String("value", cfg.Thermal.UpdateInterval),
					zap.Duration("default", 5*time.Second),
					zap.Error(err),
				)
				updateInterval = 5 * time.Second
			}
		}

		thermalConfig := &thermal.ThermalConfig{
			TempWarning:  cfg.Thermal.Temperature.Warning,
			TempCritical: cfg.Thermal.Temperature.Critical,
			TempShutdown: cfg.Thermal.Temperature.Shutdown,
			FanQuiet:     cfg.Thermal.Fan.Quiet,
			FanModerate:  cfg.Thermal.Fan.Moderate,
			FanLoud:      cfg.Thermal.Fan.Loud,
			CooldownTime: 2 * time.Minute,
		}

		thermalMonitor = thermal.NewThermalMonitor(thermalConfig, updateInterval)
		thermalMonitor.Start()
		logging.Logger.Info("Thermal monitoring started",
			zap.Duration("update_interval", updateInterval),
			zap.Float64("warning_temp", cfg.Thermal.Temperature.Warning),
			zap.Float64("critical_temp", cfg.Thermal.Temperature.Critical),
		)
	} else {
		logging.Logger.Info("Thermal monitoring disabled")
	}

	// Initialize efficiency manager
	var efficiencyMgr *efficiency.EfficiencyManager
	var dbusSvc *efficiency.DBusService
	if cfg.Efficiency.Enabled {
		// Parse default mode
		defaultMode := efficiency.ModeBalanced
		switch cfg.Efficiency.DefaultMode {
		case "Performance":
			defaultMode = efficiency.ModePerformance
		case "Efficiency":
			defaultMode = efficiency.ModeEfficiency
		case "Quiet":
			defaultMode = efficiency.ModeQuiet
		case "Auto":
			defaultMode = efficiency.ModeAuto
		case "UltraEfficiency":
			defaultMode = efficiency.ModeUltraEfficiency
		}

		efficiencyMgr = efficiency.NewEfficiencyManager(defaultMode)
		logging.Logger.Info("Efficiency manager initialized",
			zap.String("mode", defaultMode.String()),
		)

		// Start D-Bus service if enabled
		if cfg.Efficiency.DBusEnabled {
			dbusSvc, err = efficiency.NewDBusService(efficiencyMgr)
			if err != nil {
				logging.Logger.Warn("D-Bus service failed to start",
					zap.Error(err),
					zap.String("note", "GNOME integration unavailable, CLI still works"),
				)
			} else {
				if err := dbusSvc.Start(); err != nil {
					logging.Logger.Warn("D-Bus service error", zap.Error(err))
				} else {
					logging.Logger.Info("D-Bus Efficiency service started",
						zap.String("integration", "GNOME"),
					)
				}
			}
		}
	} else {
		logging.Logger.Info("Efficiency modes disabled")
	}

	// Load GSettings (if available)
	gsettings := settings.NewSettings()
	if gsettings.IsAvailable() {
		logging.Logger.Info("GSettings available")

		// Load initial mode from settings if efficiency manager was created
		if efficiencyMgr != nil {
			initialMode := gsettings.LoadInitialMode()
			efficiencyMgr.SetMode(initialMode)
			logging.Logger.Info("Loaded mode from settings",
				zap.String("mode", initialMode.String()),
			)
		}
	}

	// Initialize device manager
	var deviceManager *device.DeviceManager
	if cfg.Devices.Enabled {
		dm, err := device.NewDeviceManager()
		if err != nil {
			logging.Logger.Warn("Failed to initialize device manager",
				zap.Error(err),
				zap.String("note", "Device hotplug detection unavailable"),
			)
		} else {
			deviceManager = dm

			// Start auto-discovery if enabled
			if cfg.Devices.AutoDiscover {
				deviceManager.StartAutoDiscovery()
				logging.Logger.Info("Device auto-discovery started",
					zap.Bool("hotplug_detection", true),
				)
			}

			logging.Logger.Info("Device manager initialized",
				zap.String("dbus_service", "ie.fio.OllamaProxy.DeviceManager"),
				zap.Bool("auto_discover", cfg.Devices.AutoDiscover),
			)
		}
	} else {
		logging.Logger.Info("Device management disabled")
	}

	// Initialize router (thermal-aware if enabled, with optional forwarding)
	var r interface {
		RegisterBackend(backends.Backend) error
		ListBackends() []backends.Backend
		RouteRequest(context.Context, *backends.Annotations) (*router.RoutingDecision, error)
		HealthCheckAll(context.Context) map[string]bool
	}

	routerCfg := &router.Config{
		DefaultBackendID: cfg.Routing.DefaultBackend,
		PowerAware:       cfg.Routing.PowerAware,
		AutoOptimize:     cfg.Routing.AutoOptimizeLatency,
	}

	// Create base router
	var baseRouter *router.Router
	var thermalRouter *router.ThermalRouter

	if thermalMonitor != nil {
		// Use thermal-aware router
		thermalRouter = router.NewThermalRouter(*routerCfg, thermalMonitor)
		baseRouter = thermalRouter.Router
		r = thermalRouter
		logging.Logger.Info("Using thermal-aware routing",
			zap.Bool("power_aware", routerCfg.PowerAware),
			zap.Bool("auto_optimize", routerCfg.AutoOptimize),
		)
	} else {
		// Use basic router
		baseRouter = router.NewRouter(*routerCfg)
		r = baseRouter
		logging.Logger.Info("Using basic routing",
			zap.Bool("power_aware", routerCfg.PowerAware),
		)
	}

	// Optionally wrap with forwarding router
	var forwardingRouter *router.ForwardingRouter
	if cfg.Routing.Forwarding.Enabled {
		forwardingCfg := &router.ForwardingConfig{
			Enabled:              cfg.Routing.Forwarding.Enabled,
			MinConfidence:        cfg.Routing.Forwarding.MinConfidence,
			MaxRetries:           cfg.Routing.Forwarding.MaxRetries,
			EscalationPath:       cfg.Routing.Forwarding.EscalationPath,
			RespectThermalLimits: cfg.Routing.Forwarding.RespectThermalLimits,
			ReturnBestAttempt:    cfg.Routing.Forwarding.ReturnBestAttempt,
		}

		forwardingRouter = router.NewForwardingRouter(baseRouter, thermalRouter, forwardingCfg)
		logging.Logger.Info("Confidence-based forwarding enabled",
			zap.Float64("threshold", forwardingCfg.MinConfidence),
			zap.Int("max_retries", forwardingCfg.MaxRetries),
		)
	}

	// Register backends
	for _, backendCfg := range cfg.Backends {
		if !backendCfg.Enabled {
			logging.Logger.Info("Skipping disabled backend",
				zap.String("backend_id", backendCfg.ID),
			)
			continue
		}

		switch backendCfg.Type {
		case "ollama":
			// Build model capability
			var modelCap *backends.ModelCapability
			if backendCfg.ModelCapability.MaxModelSizeGB > 0 ||
				len(backendCfg.ModelCapability.SupportedModelPatterns) > 0 {
				modelCap = &backends.ModelCapability{
					MaxModelSizeGB:         backendCfg.ModelCapability.MaxModelSizeGB,
					SupportedModelPatterns: backendCfg.ModelCapability.SupportedModelPatterns,
					PreferredModels:        backendCfg.ModelCapability.PreferredModels,
					ExcludedPatterns:       backendCfg.ModelCapability.ExcludedPatterns,
				}
			}

			backend, err := ollama.NewOllamaBackend(ollama.Config{
				BackendConfig: backends.BackendConfig{
					ID:              backendCfg.ID,
					Type:            backendCfg.Type,
					Name:            backendCfg.Name,
					Hardware:        backendCfg.Hardware,
					Enabled:         backendCfg.Enabled,
					PowerWatts:      backendCfg.Characteristics.PowerWatts,
					AvgLatencyMs:    backendCfg.Characteristics.AvgLatencyMs,
					Priority:        backendCfg.Characteristics.Priority,
					ModelCapability: modelCap,
				},
				Endpoint: backendCfg.Endpoint,
			})
			if err != nil {
				logging.Logger.Error("Failed to create backend",
					zap.String("backend_id", backendCfg.ID),
					zap.Error(err),
				)
				continue
			}

			// Start backend
			if err := backend.Start(ctx); err != nil {
				logging.Logger.Warn("Backend failed to start, skipping registration",
					zap.String("backend_id", backendCfg.ID),
					zap.Error(err),
				)
				continue
			}

			logging.Logger.Info("Backend started successfully",
				zap.String("backend_id", backendCfg.ID),
				zap.String("hardware", backendCfg.Hardware),
				zap.String("endpoint", backendCfg.Endpoint),
			)

			if err := r.RegisterBackend(backend); err != nil {
				logging.Logger.Error("Failed to register backend",
					zap.String("backend_id", backendCfg.ID),
					zap.Error(err),
				)
				continue
			}

		case "openvino":
			// Build model capability
			var modelCap *backends.ModelCapability
			if backendCfg.ModelCapability.MaxModelSizeGB > 0 ||
				len(backendCfg.ModelCapability.SupportedModelPatterns) > 0 {
				modelCap = &backends.ModelCapability{
					MaxModelSizeGB:         backendCfg.ModelCapability.MaxModelSizeGB,
					SupportedModelPatterns: backendCfg.ModelCapability.SupportedModelPatterns,
					PreferredModels:        backendCfg.ModelCapability.PreferredModels,
					ExcludedPatterns:       backendCfg.ModelCapability.ExcludedPatterns,
				}
			}

			backend, err := openvino.NewOpenVINOLLMBackend(openvino.LLMConfig{
				BackendConfig: backends.BackendConfig{
					ID:              backendCfg.ID,
					Type:            backendCfg.Type,
					Name:            backendCfg.Name,
					Hardware:        backendCfg.Hardware,
					Enabled:         backendCfg.Enabled,
					PowerWatts:      backendCfg.Characteristics.PowerWatts,
					AvgLatencyMs:    backendCfg.Characteristics.AvgLatencyMs,
					Priority:        backendCfg.Characteristics.Priority,
					ModelCapability: modelCap,
				},
				Device:    backendCfg.Device,
				ModelPath: backendCfg.ModelPath,
				ModelName: backendCfg.ModelName,
			}, logging.Logger)
			if err != nil {
				logging.Logger.Error("Failed to create OpenVINO backend",
					zap.String("backend_id", backendCfg.ID),
					zap.Error(err),
				)
				continue
			}

			// Start backend
			if err := backend.Start(ctx); err != nil {
				logging.Logger.Warn("OpenVINO backend failed to start, skipping registration",
					zap.String("backend_id", backendCfg.ID),
					zap.Error(err),
				)
				continue
			}

			logging.Logger.Info("OpenVINO backend started successfully",
				zap.String("backend_id", backendCfg.ID),
				zap.String("hardware", backendCfg.Hardware),
				zap.String("device", backendCfg.Device),
				zap.String("model", backendCfg.ModelName),
			)

			if err := r.RegisterBackend(backend); err != nil {
				logging.Logger.Error("Failed to register OpenVINO backend",
					zap.String("backend_id", backendCfg.ID),
					zap.Error(err),
				)
				continue
			}

		default:
			logging.Logger.Warn("Unknown backend type",
				zap.String("type", backendCfg.Type),
			)
		}
	}

	// Initialize pipeline system
	// Note: Pipeline executor is always created for virtual device support
	var pipelineExecutor *pipeline.PipelineExecutor
	var pipelineLoader *pipeline.PipelineLoader

	// Create executor with all backends (required for virtual devices)
	allBackends := r.ListBackends()
	pipelineExecutor = pipeline.NewPipelineExecutor(allBackends)
	logging.Logger.Info("Pipeline executor initialized",
		zap.Int("backends", len(allBackends)),
	)

	if cfg.Pipelines.Enabled {
		logging.Logger.Info("Loading pipeline configurations",
			zap.String("config_file", cfg.Pipelines.ConfigFile),
		)

		// Load pipelines from config
		pipelineLoader = pipeline.NewPipelineLoader()
		if err := pipelineLoader.LoadFromFile(cfg.Pipelines.ConfigFile); err != nil {
			logging.Logger.Warn("Failed to load pipelines", zap.Error(err))
		} else {
			pipelineIDs := pipelineLoader.ListPipelines()
			logging.Logger.Info("Loaded pipelines",
				zap.Int("count", len(pipelineIDs)),
				zap.Strings("pipeline_ids", pipelineIDs),
			)
		}
	} else {
		logging.Logger.Info("Pipeline config loading disabled (executor still available)")
	}

	// Initialize virtual device manager
	var virtualDevMgr *virtual.VirtualDeviceManager
	if cfg.VirtualDevices.Enabled {
		vdm, err := virtual.NewVirtualDeviceManager(
			deviceManager,
			pipelineExecutor,
			&cfg.VirtualDevices,
			logging.Logger,
		)
		if err != nil {
			logging.Logger.Warn("Failed to initialize virtual device manager",
				zap.Error(err),
				zap.String("note", "Virtual audio/video devices unavailable"),
			)
		} else {
			virtualDevMgr = vdm

			// Collect backend information
			backendInfos := make([]virtual.BackendInfo, 0)
			for _, backendCfg := range cfg.Backends {
				if !backendCfg.Enabled {
					continue
				}
				backendInfos = append(backendInfos, virtual.BackendInfo{
					ID:       backendCfg.ID,
					Name:     backendCfg.Name,
					Hardware: backendCfg.Hardware,
				})
			}

			// Create audio devices for each backend
			for _, backendInfo := range backendInfos {
				if err := virtualDevMgr.CreateDevicesForBackend(
					backendInfo.ID,
					backendInfo.Name,
					backendInfo.Hardware,
				); err != nil {
					logging.Logger.Error("Failed to create virtual devices for backend",
						zap.String("backend", backendInfo.ID),
						zap.Error(err),
					)
				}
			}

			// Create virtual cameras (must be done together due to module loading)
			if cfg.VirtualDevices.Video.Enabled && cfg.VirtualDevices.Video.Camera.Enabled {
				if err := virtualDevMgr.CreateVirtualCameras(backendInfos); err != nil {
					logging.Logger.Error("Failed to create virtual cameras",
						zap.Error(err),
					)
				}
			}

			logging.Logger.Info("Virtual devices created",
				zap.Int("microphones", virtualDevMgr.GetSourceCount()),
				zap.Int("speakers", virtualDevMgr.GetSinkCount()),
				zap.Int("cameras", virtualDevMgr.GetCameraCount()),
			)

		// Register backends with virtual device manager for meeting bridge
		registeredBackends := r.ListBackends()
		for _, backend := range registeredBackends {
			// Register with virtual device manager
			virtualDevMgr.RegisterBackend(backend.ID(), backend)
			logging.Logger.Debug("Registered backend with virtual device manager",
				zap.String("backend_id", backend.ID()),
			)
		}

		// Auto-start meeting bridge for NPU backend
		if err := virtualDevMgr.StartMeetingBridge("ollama-npu"); err != nil {
			logging.Logger.Warn("Failed to auto-start meeting bridge",
				zap.String("backend", "ollama-npu"),
				zap.Error(err),
			)
		} else {
			logging.Logger.Info("Meeting audio bridge started successfully",
				zap.String("backend", "ollama-npu"),
				zap.String("description", "Google Meet AI Assistant active"),
			)
		}
		}
	} else {
		logging.Logger.Info("Virtual device management disabled")
	}

	// Create gRPC server (adapt router interface)
	var grpcRouter *router.Router
	if tr, ok := r.(*router.ThermalRouter); ok {
		grpcRouter = tr.Router
	} else {
		grpcRouter = r.(*router.Router)
	}

	// Create gRPC server with optional TLS
	var grpcServer *grpc.Server
	if cfg.Server.TLS.Enabled {
		cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		if err != nil {
			logging.Logger.Fatal("Failed to load TLS certificate", zap.Error(err))
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		// Optional mTLS if client CA provided
		if cfg.Server.TLS.ClientCAFile != "" {
			caCert, err := os.ReadFile(cfg.Server.TLS.ClientCAFile)
			if err != nil {
				logging.Logger.Fatal("Failed to load client CA", zap.Error(err))
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				logging.Logger.Fatal("Failed to parse client CA certificate")
			}

			tlsConfig.ClientCAs = caCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			logging.Logger.Info("gRPC mTLS enabled",
				zap.String("security_level", "mutual_tls"),
				zap.Bool("client_cert_required", true),
			)
		} else {
			logging.Logger.Info("gRPC TLS enabled",
				zap.String("security_level", "tls"),
			)
		}

		creds := credentials.NewTLS(tlsConfig)
		grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		grpcServer = grpc.NewServer()
		logging.Logger.Warn("gRPC TLS disabled",
			zap.String("warning", "not recommended for production"),
		)
	}

	// Pass forwarding router to server if enabled
	computeServer := server.NewComputeServer(grpcRouter)
	if forwardingRouter != nil {
		computeServer.SetForwardingRouter(forwardingRouter)
	}

	// Pass pipeline executor to server if enabled
	if pipelineExecutor != nil && pipelineLoader != nil {
		computeServer.SetPipelineExecutor(pipelineExecutor, pipelineLoader)
		logging.Logger.Info("Pipeline executor attached to gRPC server")
	}

	pb.RegisterComputeServiceServer(grpcServer, computeServer)

	// Register device service if device manager is enabled
	if deviceManager != nil {
		deviceGRPCService := device.NewGRPCService(deviceManager)
		devicev1.RegisterDeviceServiceServer(grpcServer, deviceGRPCService)
		logging.Logger.Info("Device gRPC service registered")
	}

	// Enable gRPC reflection for grpcurl
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logging.Logger.Fatal("Failed to listen on gRPC port",
			zap.String("address", grpcAddr),
			zap.Error(err),
		)
	}

	go func() {
		logging.Logger.Info("gRPC server listening",
			zap.String("address", grpcAddr),
			zap.Bool("tls", cfg.Server.TLS.Enabled),
			zap.Bool("reflection", true),
		)
		if err := grpcServer.Serve(lis); err != nil {
			logging.Logger.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// HTTP endpoints
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)

	// Liveness probe - Kubernetes style (is server alive?)
	http.HandleFunc("/healthz", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok\n")
	}))

	// Readiness probe - Kubernetes style (ready to serve traffic?)
	http.HandleFunc("/readyz", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backends := grpcRouter.ListBackends()
		healthyCount := 0
		for _, backend := range backends {
			if backend.IsHealthy() {
				healthyCount++
			}
		}

		w.Header().Set("Content-Type", "text/plain")
		if healthyCount > 0 {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "ready: %d/%d backends healthy\n", healthyCount, len(backends))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "not ready: 0/%d backends healthy\n", len(backends))
		}
	}))

	// Detailed health endpoint (legacy)
	http.HandleFunc("/health", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		healthResp, _ := computeServer.HealthCheck(r.Context(), &pb.HealthCheckRequest{})
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Status: %s\n", healthResp.Status)
		for backend, status := range healthResp.BackendHealth {
			fmt.Fprintf(w, "  %s: %s\n", backend, status)
		}
	}))

	// Backends endpoint
	http.HandleFunc("/backends", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendsResp, _ := computeServer.ListBackends(r.Context(), &pb.ListBackendsRequest{})
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Available Backends:\n\n")
		for _, b := range backendsResp.Backends {
			fmt.Fprintf(w, "ID: %s\n", b.Id)
			fmt.Fprintf(w, "  Name: %s\n", b.Name)
			fmt.Fprintf(w, "  Hardware: %s\n", b.Hardware)
			fmt.Fprintf(w, "  Status: %s\n", b.Status.State)
			fmt.Fprintf(w, "  Power: %.1fW\n", b.Metrics.PowerWatts)
			fmt.Fprintf(w, "  Avg Latency: %dms\n\n", b.Metrics.AvgLatencyMs)
		}
	}))

	// Thermal status endpoint
	if thermalMonitor != nil {
		http.HandleFunc("/thermal", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			states := thermalMonitor.GetAllStates()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(states)
		}))
	}

	// Efficiency mode endpoint
	if efficiencyMgr != nil {
		http.HandleFunc("/efficiency", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentMode := efficiencyMgr.GetMode()
			effectiveMode := efficiencyMgr.GetEffectiveMode()

			response := map[string]interface{}{
				"current_mode":   currentMode.String(),
				"effective_mode": effectiveMode.String(),
				"description":    efficiencyMgr.GetModeDescription(),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
	}

	// Initialize authentication middleware
	var authMiddleware func(http.Handler) http.Handler
	if cfg.Server.Auth.Enabled {
		authConfig := auth.Config{
			Enabled: true,
			APIKeys: make(map[string]auth.APIKeyInfo),
		}

		// Convert config API keys to auth.APIKeyInfo
		for key, keyInfo := range cfg.Server.Auth.APIKeys {
			authConfig.APIKeys[key] = auth.APIKeyInfo{
				Name:        keyInfo.Name,
				Permissions: keyInfo.Permissions,
				Enabled:     keyInfo.Enabled,
			}
		}

		authMiddleware = auth.APIKeyMiddleware(authConfig)
		logging.Logger.Info("API authentication enabled",
			zap.Int("api_keys_count", len(authConfig.APIKeys)),
		)
	} else {
		// No-op middleware when auth is disabled
		authMiddleware = func(next http.Handler) http.Handler {
			return next
		}
		logging.Logger.Info("API authentication disabled",
			zap.String("warning", "all requests will be accepted"),
		)
	}

	// Initialize rate limiting middleware
	var rateLimitMiddleware func(http.Handler) http.Handler
	if cfg.Server.RateLimit.Enabled {
		rateLimiter := ratelimit.NewIPRateLimiter(
			rate.Limit(cfg.Server.RateLimit.Rate),
			cfg.Server.RateLimit.Burst,
		)
		rateLimitMiddleware = rateLimiter.Middleware
		logging.Logger.Info("Rate limiting enabled",
			zap.Float64("rate_per_second", cfg.Server.RateLimit.Rate),
			zap.Int("burst", cfg.Server.RateLimit.Burst),
		)
	} else {
		// No-op middleware when rate limiting is disabled
		rateLimitMiddleware = func(next http.Handler) http.Handler {
			return next
		}
		logging.Logger.Info("Rate limiting disabled")
	}

	// Chain middleware: recovery first, then auth, then rate limiting
	applyMiddleware := func(handler http.HandlerFunc) http.Handler {
		return middleware.HTTPRecovery(authMiddleware(rateLimitMiddleware(handler)))
	}

	// OpenAI-compatible endpoints with middleware
	http.Handle("/v1/chat/completions", applyMiddleware(openaihttp.HandleChatCompletion(grpcRouter)))
	http.Handle("/v1/completions", applyMiddleware(openaihttp.HandleCompletion(grpcRouter)))
	http.Handle("/v1/embeddings", applyMiddleware(openaihttp.HandleEmbedding(grpcRouter)))
	http.Handle("/v1/models", applyMiddleware(openaihttp.HandleModels(grpcRouter)))

	// WebSocket endpoint for ultra-low latency streaming (with middleware)
	http.Handle("/v1/stream/ws", applyMiddleware(websockethttp.HandleWebSocketStream(grpcRouter)))

	// Version endpoint
	http.HandleFunc("/version", middleware.RecoveryHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versionInfo := map[string]string{
			"version":    Version,
			"git_commit": GitCommit,
			"build_time": BuildTime,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versionInfo)
	}))

	// Prometheus metrics endpoint
	if cfg.Monitoring.Enabled && cfg.Monitoring.PrometheusPort > 0 {
		metricsAddr := fmt.Sprintf(":%d", cfg.Monitoring.PrometheusPort)
		go func() {
			logging.Logger.Info("Prometheus metrics server listening",
				zap.String("address", metricsAddr),
				zap.String("endpoint", "/metrics"),
			)

			metricsMux := http.NewServeMux()
			metricsMux.Handle("/metrics", promhttp.Handler())

			if err := http.ListenAndServe(metricsAddr, metricsMux); err != nil {
				logging.Logger.Error("Metrics server failed", zap.Error(err))
			}
		}()
	}

	// pprof profiling endpoint
	if cfg.Monitoring.PprofEnabled && cfg.Monitoring.PprofPort > 0 {
		pprofAddr := fmt.Sprintf(":%d", cfg.Monitoring.PprofPort)
		go func() {
			logging.Logger.Info("pprof profiling server listening",
				zap.String("address", pprofAddr),
				zap.String("endpoints", "/debug/pprof/*"),
			)

			// pprof handlers are automatically registered via import _ "net/http/pprof"
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				logging.Logger.Error("pprof server failed", zap.Error(err))
			}
		}()
	}

	go func() {
		protocol := "http"
		if cfg.Server.TLS.Enabled {
			protocol = "https"
		}

		wsProtocol := "ws"
		if cfg.Server.TLS.Enabled {
			wsProtocol = "wss"
		}

		logging.Logger.Info("HTTP server listening",
			zap.String("address", httpAddr),
			zap.String("protocol", protocol),
			zap.Bool("tls", cfg.Server.TLS.Enabled),
		)
		logging.Logger.Info("HTTP endpoints available",
			zap.String("health", fmt.Sprintf("%s://%s/health", protocol, httpAddr)),
			zap.String("backends", fmt.Sprintf("%s://%s/backends", protocol, httpAddr)),
			zap.String("openai_api", fmt.Sprintf("%s://%s/v1/", protocol, httpAddr)),
			zap.String("websocket", fmt.Sprintf("%s://%s/v1/stream/ws", wsProtocol, httpAddr)),
			zap.Bool("thermal", thermalMonitor != nil),
			zap.Bool("efficiency", efficiencyMgr != nil),
		)

		if cfg.Server.TLS.Enabled {
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
			}

			httpServer := &http.Server{
				Addr:      httpAddr,
				TLSConfig: tlsConfig,
			}

			if err := httpServer.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
				logging.Logger.Fatal("Failed to serve HTTPS", zap.Error(err))
			}
		} else {
			if err := http.ListenAndServe(httpAddr, nil); err != nil {
				logging.Logger.Fatal("Failed to serve HTTP", zap.Error(err))
			}
		}
	}()

	// Start extended D-Bus services (backends, routing, thermal, system state)
	var backendsDBus *dbusPkg.BackendsService
	var routingDBus *dbusPkg.RoutingService
	var thermalDBus *dbusPkg.ThermalService
	var systemDBus *dbusPkg.SystemService

	if cfg.Efficiency.DBusEnabled {
		// Backends monitoring service
		backendsDBus, err = dbusPkg.NewBackendsService(grpcRouter)
		if err != nil {
			logging.Logger.Warn("Failed to create Backends D-Bus service", zap.Error(err))
		} else {
			if err := backendsDBus.Start(); err != nil {
				logging.Logger.Warn("Backends D-Bus service failed to start", zap.Error(err))
			} else {
				logging.Logger.Info("D-Bus Backends service started")
			}
		}

		// Routing statistics service
		routingDBus, err = dbusPkg.NewRoutingService(grpcRouter)
		if err != nil {
			logging.Logger.Warn("Failed to create Routing D-Bus service", zap.Error(err))
		} else {
			if err := routingDBus.Start(); err != nil {
				logging.Logger.Warn("Routing D-Bus service failed to start", zap.Error(err))
			} else {
				logging.Logger.Info("D-Bus Routing service started")
			}
		}

		// Thermal monitoring service
		if thermalMonitor != nil {
			thermalDBus, err = dbusPkg.NewThermalService(thermalMonitor)
			if err != nil {
				logging.Logger.Warn("Failed to create Thermal D-Bus service", zap.Error(err))
			} else {
				if err := thermalDBus.Start(); err != nil {
					logging.Logger.Warn("Thermal D-Bus service failed to start", zap.Error(err))
				} else {
					logging.Logger.Info("D-Bus Thermal service started")
				}
			}
		}

		// System state service
		if efficiencyMgr != nil {
			systemDBus, err = dbusPkg.NewSystemService(efficiencyMgr)
			if err != nil {
				logging.Logger.Warn("Failed to create System D-Bus service", zap.Error(err))
			} else {
				if err := systemDBus.Start(); err != nil {
					logging.Logger.Warn("System D-Bus service failed to start", zap.Error(err))
				} else {
					logging.Logger.Info("D-Bus System State service started")
				}
			}
		}
	}

	// Start background health checker
	go healthCheckLoop(ctx, grpcRouter)

	// Start thermal update loop (updates efficiency manager)
	if thermalMonitor != nil && efficiencyMgr != nil {
		go thermalUpdateLoop(thermalMonitor, efficiencyMgr)
	}

	// Print startup summary
	printStartupSummary(cfg, grpcRouter, thermalMonitor, efficiencyMgr, pipelineLoader, deviceManager)

	// Wait for interrupt signal or SIGHUP for config reload
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigChan

		if sig == syscall.SIGHUP {
			logging.Logger.Info("Received SIGHUP, reloading configuration")

			newCfg, err := loadConfig(*configPath)
			if err != nil {
				logging.Logger.Error("Failed to reload config", zap.Error(err))
				continue
			}

			configValidator := newCfg
			if err := config.ValidateConfig(configValidator); err != nil {
				logging.Logger.Error("Invalid configuration on reload", zap.Error(err))
				continue
			}

			// Apply CLI overrides
			if *logLevel != "" {
				newCfg.Monitoring.LogLevel = *logLevel
			}
			if *grpcPort > 0 {
				newCfg.Server.GRPCPort = *grpcPort
			}
			if *httpPort > 0 {
				newCfg.Server.HTTPPort = *httpPort
			}

			// Apply environment variable overrides
			configForEnv := newCfg
			config.ApplyEnvOverrides(configForEnv)

			// Update configuration
			cfg = newCfg
			logging.Logger.Info("Configuration reloaded successfully")

			// Note: Some configuration changes may require restart
			// This reload only updates runtime-changeable settings
			continue
		}

		// SIGTERM or SIGINT - shutdown
		break
	}

	logging.Logger.Info("Shutting down gracefully...")

	// Stop services
	if thermalMonitor != nil {
		thermalMonitor.Stop()
		logging.Logger.Info("Thermal monitor stopped")
	}
	if dbusSvc != nil {
		dbusSvc.Stop()
		logging.Logger.Info("D-Bus Efficiency service stopped")
	}

	// Stop device manager
	if deviceManager != nil {
		if err := deviceManager.Stop(); err != nil {
			logging.Logger.Error("Error stopping device manager", zap.Error(err))
		} else {
			logging.Logger.Info("Device manager stopped")
		}
	}

	// Stop virtual device manager
	if virtualDevMgr != nil {
		if err := virtualDevMgr.Stop(); err != nil {
			logging.Logger.Error("Error stopping virtual device manager", zap.Error(err))
		} else {
			logging.Logger.Info("Virtual device manager stopped")
		}
	}

	// Stop extended D-Bus services
	if backendsDBus != nil {
		backendsDBus.Stop()
		logging.Logger.Info("D-Bus Backends service stopped")
	}
	if routingDBus != nil {
		routingDBus.Stop()
		logging.Logger.Info("D-Bus Routing service stopped")
	}
	if thermalDBus != nil {
		thermalDBus.Stop()
		logging.Logger.Info("D-Bus Thermal service stopped")
	}
	if systemDBus != nil {
		systemDBus.Stop()
		logging.Logger.Info("D-Bus System service stopped")
	}

	grpcServer.GracefulStop()
	logging.Logger.Info("Shutdown complete")
}

func loadConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func healthCheckLoop(ctx context.Context, r *router.Router) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			backends := r.ListBackends()
			for _, backend := range backends {
				if err := backend.HealthCheck(ctx); err != nil {
					logging.Logger.Warn("Health check failed",
						zap.String("backend_id", backend.ID()),
						zap.String("backend_type", backend.Type()),
						zap.Error(err),
					)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// thermalUpdateLoop updates efficiency manager with thermal state
func thermalUpdateLoop(tm *thermal.ThermalMonitor, em *efficiency.EfficiencyManager) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Calculate average temperature and fan speed
		states := tm.GetAllStates()
		if len(states) == 0 {
			continue
		}

		var totalTemp float64
		var totalFan int
		count := 0

		for _, state := range states {
			if state != nil {
				totalTemp += state.Temperature
				totalFan += state.FanPercent
				count++
			}
		}

		if count > 0 {
			avgTemp := totalTemp / float64(count)
			avgFan := totalFan / count

			// Check time for quiet hours
			hour := time.Now().Hour()
			quietHours := (hour >= 22 || hour < 6)

			// Update efficiency manager state
			// Note: Battery info would come from system monitoring
			em.UpdateSystemState(
				100,         // batteryPercent - TODO: read from system
				false,       // onBattery - TODO: read from system
				avgTemp,     // avgTemp
				avgFan,      // avgFanSpeed
				quietHours,  // quietHours
			)
		}
	}
}

func printStartupSummary(cfg *config.Config, r *router.Router, tm *thermal.ThermalMonitor, em *efficiency.EfficiencyManager, pl *pipeline.PipelineLoader, dm *device.DeviceManager) {
	logging.Logger.Info("==================================================")
	logging.Logger.Info("OLLAMA COMPUTE PROXY - READY")
	logging.Logger.Info("==================================================")

	backends := r.ListBackends()
	logging.Logger.Info("Registered backends",
		zap.Int("count", len(backends)),
	)

	for _, b := range backends {
		healthy := b.IsHealthy()
		fields := []zap.Field{
			zap.String("backend_id", b.ID()),
			zap.String("hardware", b.Hardware()),
			zap.Bool("healthy", healthy),
			zap.Float64("power_watts", b.PowerWatts()),
			zap.Int32("avg_latency_ms", b.AvgLatencyMs()),
		}

		// Add thermal info if available
		if tm != nil {
			if state := tm.GetState(b.Hardware()); state != nil {
				fields = append(fields,
					zap.Float64("temperature_c", state.Temperature),
					zap.Int("fan_percent", state.FanPercent),
				)
			}
		}

		if healthy {
			logging.Logger.Info("Backend registered", fields...)
		} else {
			logging.Logger.Warn("Backend unhealthy", fields...)
		}
	}

	// Routing configuration
	routingFields := []zap.Field{
		zap.String("default_backend", cfg.Routing.DefaultBackend),
		zap.Bool("power_aware", cfg.Routing.PowerAware),
		zap.Bool("auto_optimize_latency", cfg.Routing.AutoOptimizeLatency),
		zap.Bool("thermal_monitoring", tm != nil),
	}

	if cfg.Routing.Forwarding.Enabled {
		routingFields = append(routingFields,
			zap.Bool("confidence_forwarding", true),
			zap.Float64("min_confidence", cfg.Routing.Forwarding.MinConfidence),
			zap.Strings("escalation_path", cfg.Routing.Forwarding.EscalationPath),
		)
	}

	logging.Logger.Info("Routing configuration", routingFields...)

	// Efficiency mode
	if em != nil {
		logging.Logger.Info("Efficiency mode",
			zap.String("current", em.GetMode().String()),
			zap.String("effective", em.GetEffectiveMode().String()),
		)
	}

	// Pipelines
	if pl != nil {
		pipelineIDs := pl.ListPipelines()
		logging.Logger.Info("Pipelines loaded",
			zap.Int("count", len(pipelineIDs)),
			zap.Strings("pipeline_ids", pipelineIDs),
		)
	}

	// Device management
	if dm != nil {
		devices, _ := dm.ListDevices("")
		logging.Logger.Info("Device manager",
			zap.String("dbus_service", "ie.fio.OllamaProxy.DeviceManager"),
			zap.Int("registered_devices", len(devices)),
			zap.Bool("auto_discovery", cfg.Devices.AutoDiscover),
		)
	}

	// API endpoints
	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GRPCPort)
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)

	logging.Logger.Info("API endpoints",
		zap.String("grpc", grpcAddr),
		zap.String("http_health", fmt.Sprintf("http://%s/health", httpAddr)),
		zap.String("http_backends", fmt.Sprintf("http://%s/backends", httpAddr)),
		zap.String("openai_chat", fmt.Sprintf("http://%s/v1/chat/completions", httpAddr)),
		zap.String("openai_completions", fmt.Sprintf("http://%s/v1/completions", httpAddr)),
		zap.String("openai_embeddings", fmt.Sprintf("http://%s/v1/embeddings", httpAddr)),
		zap.String("openai_models", fmt.Sprintf("http://%s/v1/models", httpAddr)),
		zap.Bool("thermal_endpoint", tm != nil),
		zap.Bool("efficiency_endpoint", em != nil),
	)

	// Example usage
	logging.Logger.Info("Example gRPC command",
		zap.String("command", fmt.Sprintf("grpcurl -plaintext localhost:%d list", cfg.Server.GRPCPort)),
	)
	logging.Logger.Info("Example HTTP health check",
		zap.String("command", fmt.Sprintf("curl http://localhost:%d/health", cfg.Server.HTTPPort)),
	)
	if tm != nil {
		logging.Logger.Info("Example thermal status",
			zap.String("command", fmt.Sprintf("curl http://localhost:%d/thermal", cfg.Server.HTTPPort)),
		)
	}
	if em != nil {
		logging.Logger.Info("Example efficiency mode",
			zap.String("command", fmt.Sprintf("curl http://localhost:%d/efficiency", cfg.Server.HTTPPort)),
		)
	}

	logging.Logger.Info("==================================================")
}
