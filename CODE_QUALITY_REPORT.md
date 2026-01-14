# Code Quality Assessment & Improvement Plan
**Generated:** 2026-01-11
**Project:** ollama-proxy
**Total Go Files:** 41 (excluding generated code)
**Test Files:** 2
**Total Lines of Code:** ~12,621 lines

---

## Executive Summary

### Overall Rating: **Good (7/10)**

**Strengths:**
- Excellent architecture with clean separation of concerns
- Outstanding documentation (30+ markdown files)
- Minimal dependencies, well-chosen libraries
- Clean, idiomatic Go code

**Critical Weaknesses:**
- **Test coverage: ~5%** (2 test files out of 41 source files)
- Missing production-readiness features (auth, TLS, metrics)
- Limited observability (basic logging, no structured logs)
- Incomplete error handling and validation

---

## Detailed Quality Metrics

| Category | Current Score | Target | Status |
|----------|--------------|--------|---------|
| Test Coverage | ⭐⭐ (2/10) | 70%+ | ❌ Critical |
| Documentation | ⭐⭐⭐⭐⭐ (10/10) | 95% | ✅ Excellent |
| Error Handling | ⭐⭐⭐ (6/10) | 90% | ⚠️ Needs Work |
| Security | ⭐⭐ (4/10) | 80% | ❌ Critical |
| Observability | ⭐⭐⭐ (6/10) | 85% | ⚠️ Needs Work |
| Code Organization | ⭐⭐⭐⭐⭐ (9/10) | 90% | ✅ Good |
| Dependency Mgmt | ⭐⭐⭐⭐ (8/10) | 85% | ✅ Good |
| Concurrency Safety | ⭐⭐⭐⭐ (8/10) | 95% | ⚠️ Verify |

---

## PRIORITY 1: CRITICAL ISSUES (Must Fix Before Production)

### 1.1 Expand Test Coverage
**Current:** 5% coverage (2/41 files tested)
**Target:** Minimum 70% coverage
**Effort:** High (2-3 weeks)
**Impact:** Critical

#### Context:
Currently only these files have tests:
- `pkg/router/forwarding_router_test.go` (432 lines) - Tests forwarding router logic
- `pkg/confidence/estimator_test.go` (422 lines) - Tests confidence estimation

**Missing critical test coverage for:**

#### 1.1.1 Core Router (`pkg/router/router.go`)
**Files:** `pkg/router/router_test.go` (create new)
```go
// Test cases needed:
// - RouteRequest with various annotations
// - Backend scoring algorithm (lines 189-266)
// - Filter candidates with constraints (lines 159-186)
// - Queue depth penalty calculation
// - Fallback routing when primary fails
// - Concurrent routing requests
// - Edge cases: no backends, all unhealthy, conflicting constraints

// Example test structure:
func TestRouteRequest_LatencyCritical(t *testing.T) {
    // Setup: Create router with mock backends (fast + slow)
    // Execute: Route with LatencyCritical=true
    // Assert: Selects fastest backend
}

func TestRouteRequest_PowerEfficient(t *testing.T) {
    // Setup: Create router with mock backends (low power + high power)
    // Execute: Route with PreferPowerEfficiency=true
    // Assert: Selects lowest power backend
}

func TestRouteRequest_QueueDepthPenalty(t *testing.T) {
    // Setup: Create router, simulate queue on one backend
    // Execute: Route multiple requests
    // Assert: Load balances to avoid congested backend
}
```

#### 1.1.2 Backend Implementations
**Files:** Create test files for each backend

**`pkg/backends/ollama/ollama_test.go`** (create new)
```go
// Test cases needed:
// - Connection to Ollama endpoint
// - Health check success/failure
// - Generate request/response
// - Stream generation
// - Embedding generation
// - Model capability filtering (lines 278-291 in main.go)
// - Metric updates
// - Error handling for network failures

// Use httptest.Server to mock Ollama API responses
```

**`pkg/backends/openai/openai_test.go`** (create new)
**`pkg/backends/anthropic/anthropic_test.go`** (create new)

#### 1.1.3 HTTP Handlers (`pkg/http/openai/handlers.go`)
**Files:** `pkg/http/openai/handlers_test.go` (create new)
```go
// Test cases needed:
// - HandleChatCompletion with valid request
// - HandleChatCompletion with invalid JSON
// - HandleChatCompletion with missing model
// - HandleCompletion streaming response
// - HandleEmbedding request
// - HandleModels listing
// - Request header parsing (X-Target-Backend, X-Priority, etc.)
// - Error responses (400, 500, 503)

// Use httptest.ResponseRecorder for testing HTTP handlers
```

#### 1.1.4 Thermal Monitor (`pkg/thermal/monitor.go`)
**Files:** `pkg/thermal/monitor_test.go` (create new)
```go
// Test cases needed:
// - Temperature reading from mock sensors
// - Fan speed detection
// - Thermal state updates
// - Warning/critical/shutdown thresholds
// - Cooldown timing
// - GetState and GetAllStates methods
// - Concurrent access to thermal state

// Mock hwmon sysfs interface for testing
```

#### 1.1.5 Efficiency Manager (`pkg/efficiency/modes.go`)
**Files:** `pkg/efficiency/manager_test.go` (create new)
```go
// Test cases needed:
// - Mode switching (Performance, Balanced, Efficiency, etc.)
// - Auto mode behavior based on battery state
// - Thermal limit enforcement per mode
// - Power limit enforcement per mode
// - UpdateSystemState effects
// - Quiet hours detection
// - D-Bus service integration (if possible)
```

#### 1.1.6 Integration Tests
**Files:** `tests/integration/` (create new directory)
```go
// End-to-end test scenarios:
// 1. Start proxy → register backends → send request → verify response
// 2. Backend failure → automatic fallback → success
// 3. Thermal threshold exceeded → backend exclusion → cooler backend selected
// 4. Priority queue: critical request preempts normal request
// 5. OpenAI API compatibility: curl request → response matches OpenAI format
// 6. WebSocket streaming: connect → stream tokens → close
// 7. Confidence forwarding: low confidence → escalate to better backend

// Use Docker Compose to spin up test environment
```

#### 1.1.7 Benchmark Tests
**Files:** `pkg/router/router_bench_test.go`, `pkg/http/openai/handlers_bench_test.go`
```go
// Benchmark critical paths:
func BenchmarkRouteRequest(b *testing.B) {
    // Measure routing decision latency
    // Target: <100µs per routing decision
}

func BenchmarkStreamingOverhead(b *testing.B) {
    // Measure proxy overhead in streaming
    // Target: <1ms per token
}
```

**Action Items:**
1. Create test files for each package (listed above)
2. Add table-driven tests for comprehensive coverage
3. Use test fixtures for mock backends
4. Add CI/CD pipeline with coverage reporting
5. Set coverage threshold: `go test -coverprofile=coverage.out -covermode=atomic ./...`
6. Add coverage badge to README.md

---

### 1.2 Add Structured Logging
**Current:** Standard library `log` with emoji-heavy messages
**Target:** Structured JSON logging with levels
**Effort:** Medium (3-5 days)
**Impact:** High

#### Context:
Current logging issues in `cmd/proxy/main.go`:
- Line 122: `log.Println("🚀 Starting Ollama Compute Proxy with Thermal Monitoring...")`
- Line 152: `log.Println("🌡️  Thermal monitoring started")`
- Line 586: `log.Printf("⚠️  Health check failed for %s: %v", backend.ID(), err)`

**Problems:**
- Emojis don't parse in log aggregation systems (Splunk, ELK, CloudWatch)
- No log levels (can't filter debug vs error logs)
- No structured fields (hard to query/filter)
- No request IDs for tracing
- Printf-style formatting is error-prone

#### Implementation Plan:

**Step 1:** Add logging dependency
```bash
# Update go.mod
go get go.uber.org/zap
```

**Step 2:** Create logger package (`pkg/logging/logger.go`)
```go
package logging

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(level string, production bool) error {
    var config zap.Config

    if production {
        config = zap.NewProductionConfig()
    } else {
        config = zap.NewDevelopmentConfig()
    }

    // Parse log level
    switch level {
    case "debug":
        config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
    case "info":
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    case "warn":
        config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
    case "error":
        config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
    default:
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    }

    var err error
    Logger, err = config.Build()
    return err
}

func Sync() {
    if Logger != nil {
        Logger.Sync()
    }
}
```

**Step 3:** Update `cmd/proxy/main.go`
```go
// Replace line 122:
// OLD: log.Println("🚀 Starting Ollama Compute Proxy with Thermal Monitoring...")
// NEW:
logging.Logger.Info("Starting Ollama Compute Proxy",
    zap.String("version", version),
    zap.Bool("thermal_enabled", cfg.Thermal.Enabled),
    zap.Bool("efficiency_enabled", cfg.Efficiency.Enabled),
)

// Replace line 152:
// OLD: log.Println("🌡️  Thermal monitoring started")
// NEW:
logging.Logger.Info("Thermal monitoring started",
    zap.Duration("update_interval", updateInterval),
    zap.Float64("warning_temp", cfg.Thermal.Temperature.Warning),
)

// Replace line 586:
// OLD: log.Printf("⚠️  Health check failed for %s: %v", backend.ID(), err)
// NEW:
logging.Logger.Warn("Health check failed",
    zap.String("backend_id", backend.ID()),
    zap.String("backend_type", backend.Type()),
    zap.Error(err),
)
```

**Step 4:** Add request ID middleware (`pkg/middleware/request_id.go`)
```go
package middleware

import (
    "context"
    "github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

func WithRequestID(ctx context.Context) context.Context {
    requestID := uuid.New().String()
    return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(RequestIDKey).(string); ok {
        return id
    }
    return ""
}
```

**Step 5:** Update configuration (`config/config.yaml`)
```yaml
monitoring:
  enabled: true
  prometheus_port: 9090
  log_level: "info"  # debug, info, warn, error
  log_format: "json"  # json or console
```

**Files to Update:**
- `cmd/proxy/main.go` (lines 122-728 - replace all log statements)
- `pkg/router/router.go` (add structured logging)
- `pkg/backends/ollama/ollama.go` (add structured logging)
- `pkg/thermal/monitor.go` (add structured logging)
- All other packages with logging

**Action Items:**
1. Add zap dependency to go.mod
2. Create pkg/logging/logger.go
3. Update main.go to initialize logger
4. Replace all log.Printf/Println calls with structured logging
5. Add request ID to context in all handlers
6. Update documentation with logging configuration
7. Test log output in JSON format

---

### 1.3 Implement Production Error Handling
**Current:** Inconsistent error handling, some errors ignored
**Target:** Comprehensive error handling with recovery
**Effort:** Medium (4-6 days)
**Impact:** High

#### Context:
**Critical error handling issues found:**

#### Issue 1: Silent error in time.ParseDuration (`cmd/proxy/main.go:137`)
```go
// CURRENT (BAD):
updateInterval, _ = time.ParseDuration(cfg.Thermal.UpdateInterval)

// FIX:
updateInterval, err = time.ParseDuration(cfg.Thermal.UpdateInterval)
if err != nil {
    logging.Logger.Warn("Invalid thermal update interval, using default",
        zap.String("value", cfg.Thermal.UpdateInterval),
        zap.Duration("default", 5*time.Second),
        zap.Error(err),
    )
    updateInterval = 5 * time.Second
}
```

#### Issue 2: Failed backend still registered (`cmd/proxy/main.go:298-307`)
```go
// CURRENT (BAD):
if err := backend.Start(ctx); err != nil {
    log.Printf("⚠️  Backend %s failed health check: %v", backendCfg.ID, err)
} else {
    log.Printf("✅ Backend %s healthy (%s at %s)", ...)
}
// Backend registered even if unhealthy!
if err := r.RegisterBackend(backend); err != nil {
    log.Printf("❌ Failed to register backend %s: %v", backendCfg.ID, err)
}

// FIX:
if err := backend.Start(ctx); err != nil {
    logging.Logger.Error("Backend failed to start",
        zap.String("backend_id", backendCfg.ID),
        zap.Error(err),
    )
    continue // Skip registration if unhealthy
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
```

#### Issue 3: No request timeout in RouteRequest (`pkg/router/router.go:98`)
```go
// ADD timeout enforcement:
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

    // ... rest of routing logic
}
```

#### Issue 4: Missing config validation
**Create:** `pkg/config/validator.go`
```go
package config

import "fmt"

func ValidateConfig(cfg *Config) error {
    // Validate server ports
    if cfg.Server.GRPCPort < 1 || cfg.Server.GRPCPort > 65535 {
        return fmt.Errorf("invalid gRPC port: %d", cfg.Server.GRPCPort)
    }
    if cfg.Server.HTTPPort < 1 || cfg.Server.HTTPPort > 65535 {
        return fmt.Errorf("invalid HTTP port: %d", cfg.Server.HTTPPort)
    }

    // Validate at least one backend enabled
    enabledCount := 0
    for _, backend := range cfg.Backends {
        if backend.Enabled {
            enabledCount++
        }
    }
    if enabledCount == 0 {
        return fmt.Errorf("no backends enabled in configuration")
    }

    // Validate backend configurations
    for _, backend := range cfg.Backends {
        if backend.Enabled {
            if backend.ID == "" {
                return fmt.Errorf("backend missing ID")
            }
            if backend.Endpoint == "" {
                return fmt.Errorf("backend %s missing endpoint", backend.ID)
            }
            if backend.Characteristics.PowerWatts <= 0 {
                return fmt.Errorf("backend %s has invalid power_watts: %.2f",
                    backend.ID, backend.Characteristics.PowerWatts)
            }
        }
    }

    // Validate thermal thresholds
    if cfg.Thermal.Enabled {
        if cfg.Thermal.Temperature.Warning >= cfg.Thermal.Temperature.Critical {
            return fmt.Errorf("thermal warning temp must be less than critical")
        }
        if cfg.Thermal.Temperature.Critical >= cfg.Thermal.Temperature.Shutdown {
            return fmt.Errorf("thermal critical temp must be less than shutdown")
        }
    }

    return nil
}
```

**Update `cmd/proxy/main.go:125-128`:**
```go
cfg, err := loadConfig("config/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// ADD validation:
if err := config.ValidateConfig(cfg); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

#### Issue 5: Add circuit breaker for backends
**Create:** `pkg/circuit/breaker.go`
```go
package circuit

import (
    "sync"
    "time"
)

type State int

const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

type CircuitBreaker struct {
    mu sync.RWMutex
    state State
    failures int
    lastFailure time.Time

    maxFailures int
    timeout time.Duration
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        state: StateClosed,
        maxFailures: maxFailures,
        timeout: timeout,
    }
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = StateHalfOpen
        } else {
            return fmt.Errorf("circuit breaker open")
        }
    }

    err := fn()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()

        if cb.failures >= cb.maxFailures {
            cb.state = StateOpen
        }
        return err
    }

    // Success - reset
    cb.failures = 0
    cb.state = StateClosed
    return nil
}
```

**Integrate into backends:**
```go
// In pkg/backends/ollama/ollama.go
type OllamaBackend struct {
    // ... existing fields
    circuitBreaker *circuit.CircuitBreaker
}

func (o *OllamaBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
    var resp *backends.GenerateResponse
    var err error

    breakerErr := o.circuitBreaker.Call(func() error {
        resp, err = o.doGenerate(ctx, req)
        return err
    })

    if breakerErr != nil {
        return nil, breakerErr
    }

    return resp, err
}
```

#### Issue 6: Add panic recovery middleware
**Create:** `pkg/middleware/recovery.go`
```go
package middleware

import (
    "context"
    "fmt"
    "runtime/debug"

    "go.uber.org/zap"
    "github.com/daoneill/ollama-proxy/pkg/logging"
)

func RecoverPanic(ctx context.Context, handler func() error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            logging.Logger.Error("Panic recovered",
                zap.Any("panic", r),
                zap.String("stack", string(debug.Stack())),
            )
            err = fmt.Errorf("internal server error: panic recovered")
        }
    }()

    return handler()
}
```

**Action Items:**
1. Fix silent error at cmd/proxy/main.go:137
2. Fix unhealthy backend registration at cmd/proxy/main.go:298
3. Create pkg/config/validator.go with validation logic
4. Add timeout enforcement to RouteRequest
5. Create pkg/circuit/breaker.go
6. Integrate circuit breaker into all backends
7. Create pkg/middleware/recovery.go
8. Wrap all HTTP handlers with panic recovery
9. Add error wrapping with context throughout codebase

---

### 1.4 Add Input Validation
**Current:** Minimal input validation
**Target:** Comprehensive validation of all inputs
**Effort:** Low (2-3 days)
**Impact:** High (Security)

#### Context:
Prevent injection attacks, DoS, and invalid requests.

**Create:** `pkg/validation/validator.go`
```go
package validation

import (
    "fmt"
    "regexp"
    "strings"
)

var (
    modelNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.:]+$`)
    backendIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

const (
    MaxPromptLength = 100000  // 100KB
    MaxModelNameLength = 256
    MinTemperature = 0.0
    MaxTemperature = 2.0
    MinTopP = 0.0
    MaxTopP = 1.0
)

func ValidateModelName(model string) error {
    if model == "" {
        return fmt.Errorf("model name cannot be empty")
    }
    if len(model) > MaxModelNameLength {
        return fmt.Errorf("model name too long: %d chars (max %d)",
            len(model), MaxModelNameLength)
    }
    if !modelNameRegex.MatchString(model) {
        return fmt.Errorf("invalid model name format: %s", model)
    }
    return nil
}

func ValidatePrompt(prompt string) error {
    if prompt == "" {
        return fmt.Errorf("prompt cannot be empty")
    }
    if len(prompt) > MaxPromptLength {
        return fmt.Errorf("prompt too long: %d chars (max %d)",
            len(prompt), MaxPromptLength)
    }
    return nil
}

func ValidateGenerationOptions(opts *backends.GenerationOptions) error {
    if opts == nil {
        return nil
    }

    if opts.Temperature < MinTemperature || opts.Temperature > MaxTemperature {
        return fmt.Errorf("temperature %.2f out of range [%.1f, %.1f]",
            opts.Temperature, MinTemperature, MaxTemperature)
    }

    if opts.TopP < MinTopP || opts.TopP > MaxTopP {
        return fmt.Errorf("top_p %.2f out of range [%.1f, %.1f]",
            opts.TopP, MinTopP, MaxTopP)
    }

    if opts.MaxTokens < 0 {
        return fmt.Errorf("max_tokens cannot be negative")
    }

    return nil
}

func ValidateBackendID(id string) error {
    if id == "" || id == "auto" {
        return nil  // Empty or "auto" is valid
    }
    if !backendIDRegex.MatchString(id) {
        return fmt.Errorf("invalid backend ID format: %s", id)
    }
    return nil
}
```

**Update `pkg/http/openai/handlers.go`:**
```go
func HandleChatCompletion(router *router.Router) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ADD request size limit
        r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit

        var req OpenAIChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        // ADD validation
        if err := validation.ValidateModelName(req.Model); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        // Validate messages
        for _, msg := range req.Messages {
            if err := validation.ValidatePrompt(msg.Content); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
        }

        // Validate backend target if specified
        targetBackend := r.Header.Get("X-Target-Backend")
        if err := validation.ValidateBackendID(targetBackend); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        // ... rest of handler
    }
}
```

**Action Items:**
1. Create pkg/validation/validator.go
2. Update all HTTP handlers to validate inputs
3. Add request size limits (http.MaxBytesReader)
4. Validate all header values
5. Add rate limiting per client (future task)
6. Add tests for validation edge cases

---

## PRIORITY 2: HIGH (Should Do Soon)

### 2.1 Implement Metrics Export
**Current:** Prometheus port configured but not implemented
**Target:** Full Prometheus metrics export
**Effort:** Medium (3-4 days)
**Impact:** Medium-High

#### Context:
`config/config.yaml:87-88` configures Prometheus but it's not implemented.

**Add dependency:**
```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

**Create:** `pkg/metrics/metrics.go`
```go
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
            Help: "Total number of requests by backend and status",
        },
        []string{"backend_id", "model", "status"},
    )

    RequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ollama_proxy_request_duration_seconds",
            Help: "Request duration in seconds",
            Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
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

    // Queue metrics
    QueueDepth = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ollama_proxy_queue_depth",
            Help: "Current queue depth by backend and priority",
        },
        []string{"backend_id", "priority"},
    )

    // Routing metrics
    RoutingDecisionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ollama_proxy_routing_decisions_total",
            Help: "Total routing decisions by reason",
        },
        []string{"reason", "backend_id"},
    )

    // Efficiency mode
    EfficiencyMode = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ollama_proxy_efficiency_mode",
            Help: "Current efficiency mode (0=Performance, 1=Balanced, 2=Efficiency, 3=Quiet, 4=Auto, 5=Ultra)",
        },
        []string{"mode"},
    )
)
```

**Update `cmd/proxy/main.go`:** Add Prometheus endpoint
```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// After line 453 (after HTTP handlers are registered):
if cfg.Monitoring.Enabled && cfg.Monitoring.PrometheusPort > 0 {
    http.Handle("/metrics", promhttp.Handler())

    go func() {
        metricsAddr := fmt.Sprintf(":%d", cfg.Monitoring.PrometheusPort)
        logging.Logger.Info("Prometheus metrics server started",
            zap.String("address", metricsAddr),
        )
        if err := http.ListenAndServe(metricsAddr, nil); err != nil {
            logging.Logger.Error("Metrics server failed", zap.Error(err))
        }
    }()
}
```

**Update `pkg/router/router.go`:** Add metrics
```go
import "github.com/daoneill/ollama-proxy/pkg/metrics"

func (r *Router) RouteRequest(ctx context.Context, annotations *backends.Annotations) (*RoutingDecision, error) {
    start := time.Now()

    // ... existing routing logic ...

    // Record metrics
    metrics.RoutingDecisionsTotal.WithLabelValues(
        reason,
        selectedBackend.ID(),
    ).Inc()

    metrics.RequestDuration.WithLabelValues(
        selectedBackend.ID(),
        annotations.Model,
    ).Observe(time.Since(start).Seconds())

    return decision, nil
}
```

**Update thermal monitor to export metrics:**
```go
// In pkg/thermal/monitor.go, add periodic metric updates
func (tm *ThermalMonitor) updateMetrics() {
    states := tm.GetAllStates()
    for hw, state := range states {
        if state != nil {
            metrics.BackendTemperature.WithLabelValues(
                "", // backend_id unknown here
                hw,
            ).Set(state.Temperature)

            metrics.BackendFanSpeed.WithLabelValues(
                "",
                hw,
            ).Set(float64(state.FanPercent))
        }
    }
}
```

**Action Items:**
1. Add Prometheus dependencies
2. Create pkg/metrics/metrics.go with all metric definitions
3. Update router to record routing metrics
4. Update backends to record request metrics
5. Update thermal monitor to export temperature/fan metrics
6. Add Prometheus scrape config example in docs/
7. Create Grafana dashboard JSON in docs/grafana/

---

### 2.2 Configuration Improvements
**Current:** Hardcoded config path, no env overrides
**Target:** Flexible configuration with validation
**Effort:** Low (1-2 days)
**Impact:** Medium

#### Context:
Current issues:
- `cmd/proxy/main.go:125` hardcodes "config/config.yaml"
- No environment variable support
- No config hot-reload
- No secret management

**Update `cmd/proxy/main.go`:** Add CLI flags
```go
import "flag"

var (
    configPath = flag.String("config", "config/config.yaml", "Path to configuration file")
    logLevel = flag.String("log-level", "", "Log level (debug, info, warn, error) - overrides config")
    grpcPort = flag.Int("grpc-port", 0, "gRPC port - overrides config")
    httpPort = flag.Int("http-port", 0, "HTTP port - overrides config")
)

func main() {
    flag.Parse()

    // Load configuration
    cfg, err := loadConfig(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Apply CLI overrides
    if *logLevel != "" {
        cfg.Monitoring.LogLevel = *logLevel
    }
    if *grpcPort > 0 {
        cfg.Server.GRPCPort = *grpcPort
    }
    if *httpPort > 0 {
        cfg.Server.HTTPPort = *httpPort
    }

    // ... rest of main
}
```

**Create:** `pkg/config/env.go` for environment variable overrides
```go
package config

import (
    "os"
    "strconv"
)

func ApplyEnvOverrides(cfg *Config) {
    // Server overrides
    if val := os.Getenv("OLLAMA_PROXY_GRPC_PORT"); val != "" {
        if port, err := strconv.Atoi(val); err == nil {
            cfg.Server.GRPCPort = port
        }
    }
    if val := os.Getenv("OLLAMA_PROXY_HTTP_PORT"); val != "" {
        if port, err := strconv.Atoi(val); err == nil {
            cfg.Server.HTTPPort = port
        }
    }

    // Monitoring overrides
    if val := os.Getenv("OLLAMA_PROXY_LOG_LEVEL"); val != "" {
        cfg.Monitoring.LogLevel = val
    }

    // Backend endpoint overrides
    if val := os.Getenv("OLLAMA_NPU_ENDPOINT"); val != "" {
        for i := range cfg.Backends {
            if cfg.Backends[i].ID == "ollama-npu" {
                cfg.Backends[i].Endpoint = val
            }
        }
    }
}
```

**Add hot-reload on SIGHUP:**
```go
// In main()
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

        if err := config.ValidateConfig(newCfg); err != nil {
            logging.Logger.Error("Invalid configuration on reload", zap.Error(err))
            continue
        }

        // Apply new config (carefully - some changes may require restart)
        cfg = newCfg
        logging.Logger.Info("Configuration reloaded successfully")
        continue
    }

    // Handle shutdown
    break
}
```

**Action Items:**
1. Add CLI flags for common overrides
2. Create pkg/config/env.go for environment variables
3. Add SIGHUP handler for config reload
4. Document environment variables in README.md
5. Add example .env file
6. Consider HashiCorp Vault integration for secrets (future)

---

### 2.3 Enhance Security
**Current:** No authentication, no TLS
**Target:** Basic auth, TLS, rate limiting
**Effort:** Medium (5-7 days)
**Impact:** Critical for production

#### Context:
Currently all endpoints are completely open.

#### 2.3.1 Add TLS Support

**Update config:**
```yaml
server:
  grpc_port: 50051
  http_port: 8080
  host: "0.0.0.0"
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    client_ca_file: "/path/to/ca.pem"  # For mTLS
```

**Update `cmd/proxy/main.go`:** TLS for gRPC
```go
import (
    "crypto/tls"
    "crypto/x509"
    "google.golang.org/grpc/credentials"
)

// Replace line 345:
var grpcServer *grpc.Server

if cfg.Server.TLS.Enabled {
    cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
    if err != nil {
        log.Fatalf("Failed to load TLS certificate: %v", err)
    }

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }

    // mTLS if client CA provided
    if cfg.Server.TLS.ClientCAFile != "" {
        caCert, err := os.ReadFile(cfg.Server.TLS.ClientCAFile)
        if err != nil {
            log.Fatalf("Failed to load client CA: %v", err)
        }

        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)

        tlsConfig.ClientCAs = caCertPool
        tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
    }

    creds := credentials.NewTLS(tlsConfig)
    grpcServer = grpc.NewServer(grpc.Creds(creds))

    logging.Logger.Info("gRPC TLS enabled",
        zap.Bool("mtls", cfg.Server.TLS.ClientCAFile != ""),
    )
} else {
    grpcServer = grpc.NewServer()
}
```

**TLS for HTTP:**
```go
// Replace http.ListenAndServe at line 453:
httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)

if cfg.Server.TLS.Enabled {
    logging.Logger.Info("Starting HTTPS server", zap.String("address", httpAddr))

    tlsConfig := &tls.Config{
        MinVersion: tls.VersionTLS12,
    }

    server := &http.Server{
        Addr:      httpAddr,
        TLSConfig: tlsConfig,
    }

    if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
        log.Fatalf("Failed to serve HTTPS: %v", err)
    }
} else {
    if err := http.ListenAndServe(httpAddr, nil); err != nil {
        log.Fatalf("Failed to serve HTTP: %v", err)
    }
}
```

#### 2.3.2 Add API Key Authentication

**Update config:**
```yaml
server:
  auth:
    enabled: true
    type: "api_key"  # or "jwt", "oauth"
    api_keys:
      - key: "sk-..." # API key
        name: "client1"
        permissions: ["read", "write"]
```

**Create:** `pkg/auth/middleware.go`
```go
package auth

import (
    "net/http"
    "strings"
)

type Config struct {
    Enabled bool
    APIKeys map[string]APIKey // key -> metadata
}

type APIKey struct {
    Name        string
    Permissions []string
}

func APIKeyMiddleware(cfg Config) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !cfg.Enabled {
                next.ServeHTTP(w, r)
                return
            }

            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
                return
            }

            // Support "Bearer <key>" format
            key := strings.TrimPrefix(authHeader, "Bearer ")
            key = strings.TrimPrefix(key, "bearer ")

            if _, valid := cfg.APIKeys[key]; !valid {
                http.Error(w, "Invalid API key", http.StatusUnauthorized)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Apply to HTTP handlers:**
```go
// Wrap handlers with auth
authMiddleware := auth.APIKeyMiddleware(authConfig)

http.Handle("/v1/chat/completions",
    authMiddleware(http.HandlerFunc(openaihttp.HandleChatCompletion(grpcRouter))))
```

#### 2.3.3 Add Rate Limiting

**Add dependency:**
```bash
go get golang.org/x/time/rate
```

**Create:** `pkg/ratelimit/limiter.go`
```go
package ratelimit

import (
    "net/http"
    "sync"
    "golang.org/x/time/rate"
)

type IPRateLimiter struct {
    mu       sync.RWMutex
    limiters map[string]*rate.Limiter
    rate     rate.Limit
    burst    int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
    return &IPRateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     r,
        burst:    b,
    }
}

func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters[ip] = limiter
    }

    return limiter
}

func (rl *IPRateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        limiter := rl.getLimiter(ip)

        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

**Apply rate limiting:**
```go
// In main.go
rateLimiter := ratelimit.NewIPRateLimiter(10, 20) // 10 req/sec, burst 20

http.Handle("/v1/chat/completions",
    rateLimiter.Middleware(
        authMiddleware(
            http.HandlerFunc(openaihttp.HandleChatCompletion(grpcRouter)))))
```

**Action Items:**
1. Add TLS configuration to config.yaml
2. Implement TLS for gRPC and HTTP servers
3. Create pkg/auth/middleware.go for API key auth
4. Create pkg/ratelimit/limiter.go
5. Apply middleware to all HTTP endpoints
6. Generate self-signed certs for development
7. Document auth setup in docs/security.md

---

### 2.4 Improve Error Messages
**Current:** Generic error messages
**Target:** Detailed, actionable error messages
**Effort:** Low (1-2 days)
**Impact:** Medium (Developer Experience)

#### Context:
Better error messages help debugging and reduce support burden.

**Example improvements:**

**Before (pkg/router/router.go:120):**
```go
return nil, fmt.Errorf("no healthy backends available matching criteria")
```

**After:**
```go
var reasons []string
if annotations.MaxLatencyMs > 0 {
    reasons = append(reasons, fmt.Sprintf("latency < %dms", annotations.MaxLatencyMs))
}
if annotations.MaxPowerWatts > 0 {
    reasons = append(reasons, fmt.Sprintf("power < %dW", annotations.MaxPowerWatts))
}

return nil, fmt.Errorf("no healthy backends available (checked %d backends, constraints: %s)",
    len(r.backends), strings.Join(reasons, ", "))
```

**Create:** `pkg/errors/errors.go` for typed errors
```go
package errors

import "fmt"

type NoBackendsError struct {
    TotalBackends   int
    HealthyBackends int
    Constraints     []string
}

func (e *NoBackendsError) Error() string {
    return fmt.Sprintf("no backends available: %d total, %d healthy, constraints: %v",
        e.TotalBackends, e.HealthyBackends, e.Constraints)
}

type BackendUnhealthyError struct {
    BackendID string
    Reason    string
}

func (e *BackendUnhealthyError) Error() string {
    return fmt.Sprintf("backend %s is unhealthy: %s", e.BackendID, e.Reason)
}

// ... more typed errors
```

**Action Items:**
1. Create pkg/errors/errors.go with typed errors
2. Update all error returns to use typed errors
3. Add error context (file:line, operation, values)
4. Create error codes for client parsing
5. Document error codes in API docs

---

## PRIORITY 3: MEDIUM (Nice to Have)

### 3.1 Add Request Tracing
**Effort:** Medium (3-4 days)
**Impact:** Low-Medium

Add OpenTelemetry for distributed tracing.

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/jaeger
```

**Files to create:**
- `pkg/tracing/tracer.go` - Initialize OpenTelemetry
- Update all major operations to create spans

---

### 3.2 Health Check Improvements
**Effort:** Low (1 day)
**Impact:** Medium

**Create:** `pkg/health/checker.go`
```go
// Implement:
// - Liveness probe (is server running?)
// - Readiness probe (can it serve traffic?)
// - Deep health check (can backends do inference?)
```

**Add endpoints:**
- `/healthz` - Liveness (always returns 200 if server running)
- `/readyz` - Readiness (200 if at least one backend healthy)
- `/health/deep` - Deep check (actually runs inference)

---

### 3.3 Performance Profiling
**Effort:** Low (few hours)
**Impact:** Low

**Enable pprof:**
```go
import _ "net/http/pprof"

// In main.go, register pprof handlers on separate port
if cfg.Monitoring.PprofEnabled {
    go func() {
        http.ListenAndServe(":6060", nil)
    }()
}
```

**Usage:**
```bash
go tool pprof http://localhost:6060/debug/pprof/profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

### 3.4 Code Quality Tools
**Effort:** Low (1 day)
**Impact:** Medium

**Create:** `.golangci.yml`
```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - misspell
    - gocyclo
    - gofmt
    - gocritic
```

**Add to Makefile:**
```makefile
.PHONY: lint
lint:
	golangci-lint run

.PHONY: security
security:
	gosec ./...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
```

**Setup CI/CD:** `.github/workflows/test.yml`
```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: make test
      - run: make lint
      - run: make security
```

---

## Quick Wins (1-2 hours each)

### QW1: Add Version Info
**File:** `cmd/proxy/version.go`
```go
package main

var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildTime = "unknown"
)
```

**Build with:**
```bash
go build -ldflags "-X main.Version=1.0.0 -X main.GitCommit=$(git rev-parse HEAD)"
```

**Add endpoint:** `/version` returns JSON with version info

---

### QW2: Add .editorconfig
**File:** `.editorconfig`
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.{yaml,yml}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
```

---

### QW3: Add CONTRIBUTING.md
Document:
- How to set up dev environment
- How to run tests
- Code style guidelines
- How to submit PRs

---

### QW4: Add Docker Support
**Create:** `Dockerfile`
```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o ollama-proxy ./cmd/proxy

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/ollama-proxy /usr/local/bin/
COPY --from=builder /app/config /etc/ollama-proxy/config
EXPOSE 8080 50051
CMD ["ollama-proxy", "--config", "/etc/ollama-proxy/config/config.yaml"]
```

**Create:** `docker-compose.yml` for local development

---

## Summary & Roadmap

### Phase 1: Critical (Week 1-2)
- [ ] Fix silent errors (1.3, Issue 1-2)
- [ ] Add config validation (1.3, Issue 4)
- [ ] Add input validation (1.4)
- [ ] Setup structured logging (1.2)

### Phase 2: Testing (Week 3-4)
- [ ] Write router tests (1.1.1)
- [ ] Write backend tests (1.1.2)
- [ ] Write HTTP handler tests (1.1.3)
- [ ] Integration tests (1.1.6)

### Phase 3: Production Readiness (Week 5-6)
- [ ] Add circuit breakers (1.3, Issue 5)
- [ ] Implement metrics export (2.1)
- [ ] Add TLS support (2.3.1)
- [ ] Add authentication (2.3.2)
- [ ] Add rate limiting (2.3.3)

### Phase 4: Polish (Week 7-8)
- [ ] Config improvements (2.2)
- [ ] Better error messages (2.4)
- [ ] Health checks (3.2)
- [ ] CI/CD setup (3.4)
- [ ] Documentation updates

---

## Tracking Progress

Use this checklist format:
```markdown
- [ ] Task name (Priority, Estimated hours)
  - Context: Why needed
  - Files: What to create/modify
  - Status: Not started | In Progress | Blocked | Done
```

**Next Steps:**
1. Review this report with team
2. Prioritize tasks based on deployment timeline
3. Assign tasks to developers
4. Set up project board (GitHub Projects, Jira, etc.)
5. Create tracking issues for each major task
6. Begin with Priority 1 items

---

**Report End**
For questions or clarifications, refer to specific file:line references throughout this document.
