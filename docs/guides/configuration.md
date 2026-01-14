# Configuration Guide

Complete configuration reference for the Ollama Proxy.

---

## Configuration File

### Location

The proxy looks for `config/config.yaml` in the working directory:

```bash
# Default location (when running from project directory)
~/src/ollama-proxy/config/config.yaml

# Alternative location (if specified)
~/.config/ollama-proxy/config.yaml
```

### Format

YAML format with hierarchical structure:

```yaml
server:
  # Server configuration

router:
  # Routing configuration

backends:
  # Backend definitions

efficiency_modes:
  # Efficiency mode settings

thermal:
  # Thermal monitoring

logging:
  # Logging configuration
```

---

## Server Configuration

```yaml
server:
  grpc_port: 50051
  http_port: 8080
  host: "0.0.0.0"
  read_timeout_seconds: 30
  write_timeout_seconds: 30
  max_concurrent_streams: 100
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `grpc_port` | integer | 50051 | gRPC server port |
| `http_port` | integer | 8080 | HTTP/REST server port |
| `host` | string | "0.0.0.0" | Bind address ("0.0.0.0" = all interfaces, "127.0.0.1" = localhost only) |
| `read_timeout_seconds` | integer | 30 | HTTP read timeout |
| `write_timeout_seconds` | integer | 30 | HTTP write timeout |
| `max_concurrent_streams` | integer | 100 | Max concurrent gRPC streams |

### Examples

**Localhost only:**
```yaml
server:
  host: "127.0.0.1"  # Only accept local connections
```

**Custom ports:**
```yaml
server:
  grpc_port: 9090
  http_port: 8000
```

**Longer timeouts for slow backends:**
```yaml
server:
  read_timeout_seconds: 120
  write_timeout_seconds: 120
```

---

## Router Configuration

```yaml
router:
  default_backend_id: "ollama-igpu"
  power_aware: true
  auto_optimize: true

  priority_scoring:
    critical_boost: 500.0
    high_boost: 200.0
    normal_boost: 0.0
    best_effort_penalty: -100.0

  queue_depth_penalty_per_request: 50.0
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `default_backend_id` | string | "" | Fallback backend if auto-selection fails |
| `power_aware` | boolean | true | Enable power-aware routing |
| `auto_optimize` | boolean | true | Enable automatic optimization |
| `priority_scoring.*` | float | varies | Scoring boost/penalty per priority level |
| `queue_depth_penalty_per_request` | float | 50.0 | Penalty points per pending request |

### Examples

**Disable power-aware routing:**
```yaml
router:
  power_aware: false  # Ignore power consumption in routing
```

**Adjust priority scoring:**
```yaml
router:
  priority_scoring:
    critical_boost: 1000.0   # Stronger boost for critical
    best_effort_penalty: -200.0  # Stronger penalty for best-effort
```

**Aggressive queue avoidance:**
```yaml
router:
  queue_depth_penalty_per_request: 100.0  # Avoid congested backends more aggressively
```

---

## Backend Configuration

```yaml
backends:
  - id: ollama-npu
    type: ollama
    name: "Ollama NPU"
    hardware: npu
    enabled: true
    endpoint: "http://localhost:11434"

    characteristics:
      power_watts: 3.0
      avg_latency_ms: 800
      priority: 3

    model_capability:
      max_model_size_gb: 8
      supported_model_patterns:
        - "*:0.5b"
        - "*:1b"
      preferred_models:
        - "qwen2.5:0.5b"
      excluded_patterns:
        - "*:70b"

    health_check:
      enabled: true
      interval_seconds: 30
      timeout_seconds: 5
      endpoint: "/api/tags"
```

### Backend Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Unique backend identifier |
| `type` | string | Yes | Backend type ("ollama") |
| `name` | string | Yes | Human-readable name |
| `hardware` | string | Yes | Hardware type ("npu", "igpu", "gpu", "cpu") |
| `enabled` | boolean | No | Enable/disable backend (default: true) |
| `endpoint` | string | Yes | Backend URL |

### Characteristics

| Parameter | Type | Description |
|-----------|------|-------------|
| `power_watts` | float | Power consumption in watts |
| `avg_latency_ms` | integer | Average latency in milliseconds |
| `priority` | integer | Backend priority (higher = preferred) |

### Model Capability

| Parameter | Type | Description |
|-----------|------|-------------|
| `max_model_size_gb` | integer | Maximum model size supported (GB) |
| `supported_model_patterns` | array | Glob patterns for supported models |
| `preferred_models` | array | Models this backend prefers |
| `excluded_patterns` | array | Models to exclude |

**Pattern Examples:**
- `*:0.5b` - Any model with 0.5B parameters
- `qwen2.5:*` - Any qwen2.5 variant
- `*` - All models

### Health Check

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | true | Enable health checks |
| `interval_seconds` | integer | 30 | Check interval |
| `timeout_seconds` | integer | 5 | Check timeout |
| `endpoint` | string | "/api/tags" | Health check endpoint |

### Examples

**Single backend (NPU only):**
```yaml
backends:
  - id: ollama-npu
    type: ollama
    name: "Ollama NPU"
    hardware: npu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 3.0
      avg_latency_ms: 800
      priority: 1
```

**Multiple backends (same Ollama instance):**
```yaml
backends:
  # NPU configuration
  - id: ollama-npu
    type: ollama
    name: "Ollama NPU"
    hardware: npu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 3.0
      avg_latency_ms: 800
      priority: 3

  # iGPU configuration (same endpoint, different characteristics)
  - id: ollama-igpu
    type: ollama
    name: "Ollama iGPU"
    hardware: igpu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 12.0
      avg_latency_ms: 400
      priority: 2

  # NVIDIA GPU configuration
  - id: ollama-nvidia
    type: ollama
    name: "Ollama NVIDIA"
    hardware: gpu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 55.0
      avg_latency_ms: 150
      priority: 1
```

**Remote backend:**
```yaml
backends:
  - id: ollama-remote
    type: ollama
    name: "Ollama Remote Server"
    hardware: gpu
    enabled: true
    endpoint: "http://192.168.1.100:11434"
    characteristics:
      power_watts: 200.0
      avg_latency_ms: 50
      priority: 1
```

**Model-specific backend:**
```yaml
backends:
  - id: ollama-npu-small
    type: ollama
    name: "NPU (Small Models Only)"
    hardware: npu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 3.0
      avg_latency_ms: 800
      priority: 3
    model_capability:
      max_model_size_gb: 2
      supported_model_patterns:
        - "*:0.5b"
        - "*:1b"
      excluded_patterns:
        - "*:3b"
        - "*:7b"
        - "*:70b"
```

---

## Efficiency Modes

```yaml
efficiency_modes:
  performance:
    power_budget_watts: null  # No limit
    latency_weight: 3.0
    power_weight: 0.0
    backend_preference: ["nvidia", "igpu", "npu"]

  balanced:
    power_budget_watts: 20
    latency_weight: 1.5
    power_weight: 1.0
    backend_preference: ["igpu", "nvidia", "npu"]

  efficiency:
    power_budget_watts: 15
    latency_weight: 0.5
    power_weight: 2.5
    backend_preference: ["npu", "igpu"]

  quiet:
    power_budget_watts: 15
    latency_weight: 0.5
    power_weight: 2.0
    thermal_aggressive: true
    backend_preference: ["npu", "igpu"]

  ultra_efficiency:
    power_budget_watts: 10
    latency_weight: 0.0
    power_weight: 3.0
    backend_preference: ["npu"]
    max_concurrency: 2

  auto:
    enabled: true
    battery_thresholds:
      ultra_efficiency: 20
      efficiency: 50
      balanced: 80
    thermal_threshold: 85
    quiet_hours:
      enabled: true
      start_hour: 22
      end_hour: 7
```

### Mode Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `power_budget_watts` | integer/null | Maximum power budget (null = no limit) |
| `latency_weight` | float | Weight for latency in scoring (0-3) |
| `power_weight` | float | Weight for power efficiency in scoring (0-3) |
| `backend_preference` | array | Preferred backend order by hardware type |
| `thermal_aggressive` | boolean | Aggressive thermal management |
| `max_concurrency` | integer | Maximum concurrent requests (ultra efficiency only) |

### Auto Mode Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `enabled` | boolean | Enable auto mode |
| `battery_thresholds.*` | integer | Battery % thresholds for mode switching |
| `thermal_threshold` | integer | Temperature (°C) to trigger quiet mode |
| `quiet_hours.enabled` | boolean | Enable quiet hours |
| `quiet_hours.start_hour` | integer | Quiet hours start (24h format) |
| `quiet_hours.end_hour` | integer | Quiet hours end (24h format) |

### Examples

**Custom performance mode (no power limit):**
```yaml
efficiency_modes:
  performance:
    power_budget_watts: null
    latency_weight: 5.0  # Extreme latency optimization
    power_weight: 0.0
```

**Battery-focused auto mode:**
```yaml
efficiency_modes:
  auto:
    battery_thresholds:
      ultra_efficiency: 30  # More conservative
      efficiency: 60
      balanced: 90
```

**Quiet hours for night usage:**
```yaml
efficiency_modes:
  auto:
    quiet_hours:
      enabled: true
      start_hour: 21  # 9 PM
      end_hour: 8     # 8 AM
```

---

## Thermal Monitoring

```yaml
thermal:
  enabled: true
  poll_interval_seconds: 5

  thresholds:
    warning: 80
    high: 85
    critical: 90
    recovery: 75

  sensors:
    cpu_thermal_zone: "/sys/class/thermal/thermal_zone0/temp"
    nvidia_gpu: "nvidia-smi"
    amd_gpu: "/sys/class/drm/card0/device/hwmon/hwmon0/temp1_input"
    fan_speed: "/sys/class/hwmon/hwmon1/fan1_input"

  actions:
    notify_user: true
    auto_switch_mode: true
    mark_throttled: true
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | true | Enable thermal monitoring |
| `poll_interval_seconds` | integer | 5 | Temperature check interval |
| `thresholds.warning` | integer | 80 | Warning temperature (°C) |
| `thresholds.high` | integer | 85 | High temperature (°C) |
| `thresholds.critical` | integer | 90 | Critical temperature (°C) |
| `thresholds.recovery` | integer | 75 | Recovery temperature (°C) |
| `sensors.*` | string | varies | Sensor file paths |
| `actions.notify_user` | boolean | true | Send desktop notifications |
| `actions.auto_switch_mode` | boolean | true | Auto switch to quiet mode |
| `actions.mark_throttled` | boolean | true | Mark hot backends as throttled |

### Examples

**Conservative thermal management:**
```yaml
thermal:
  thresholds:
    warning: 70  # Earlier warning
    high: 75     # Earlier action
    critical: 80
    recovery: 65
```

**Disable thermal monitoring:**
```yaml
thermal:
  enabled: false
```

**Custom sensor paths:**
```yaml
thermal:
  sensors:
    cpu_thermal_zone: "/sys/class/thermal/thermal_zone2/temp"
    amd_gpu: "/sys/class/drm/card1/device/hwmon/hwmon2/temp1_input"
```

---

## Logging

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "text" # text or json
  output: "stdout"

  # Stream logging (detailed per-token logging)
  stream_logging:
    enabled: false  # Enable for debugging only (performance impact)
    log_ttft: true
    log_inter_token: false

  # File output (optional)
  file:
    enabled: false
    path: "/var/log/ollama-proxy/proxy.log"
    max_size_mb: 100
    max_backups: 3
    max_age_days: 7
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `level` | string | "info" | Log level (debug, info, warn, error) |
| `format` | string | "text" | Log format (text, json) |
| `output` | string | "stdout" | Output destination |
| `stream_logging.enabled` | boolean | false | Enable detailed stream logging |
| `stream_logging.log_ttft` | boolean | true | Log time-to-first-token |
| `stream_logging.log_inter_token` | boolean | false | Log inter-token latency |

### Examples

**Debug logging:**
```yaml
logging:
  level: "debug"
  stream_logging:
    enabled: true
    log_inter_token: true
```

**JSON logging for parsing:**
```yaml
logging:
  format: "json"
  output: "stdout"
```

**File logging:**
```yaml
logging:
  output: "file"
  file:
    enabled: true
    path: "/var/log/ollama-proxy/proxy.log"
    max_size_mb: 100
    max_backups: 5
    max_age_days: 14
```

---

## Complete Example Configuration

```yaml
# Server configuration
server:
  grpc_port: 50051
  http_port: 8080
  host: "0.0.0.0"
  read_timeout_seconds: 60
  write_timeout_seconds: 60
  max_concurrent_streams: 100

# Router configuration
router:
  default_backend_id: "ollama-igpu"
  power_aware: true
  auto_optimize: true
  priority_scoring:
    critical_boost: 500.0
    high_boost: 200.0
    normal_boost: 0.0
    best_effort_penalty: -100.0
  queue_depth_penalty_per_request: 50.0

# Backends
backends:
  - id: ollama-npu
    type: ollama
    name: "Ollama NPU"
    hardware: npu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 3.0
      avg_latency_ms: 800
      priority: 3
    model_capability:
      max_model_size_gb: 8
      supported_model_patterns:
        - "*:0.5b"
        - "*:1b"
    health_check:
      enabled: true
      interval_seconds: 30

  - id: ollama-igpu
    type: ollama
    name: "Ollama iGPU"
    hardware: igpu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 12.0
      avg_latency_ms: 400
      priority: 2

  - id: ollama-nvidia
    type: ollama
    name: "Ollama NVIDIA"
    hardware: gpu
    enabled: true
    endpoint: "http://localhost:11434"
    characteristics:
      power_watts: 55.0
      avg_latency_ms: 150
      priority: 1

# Efficiency modes
efficiency_modes:
  performance:
    power_budget_watts: null
    latency_weight: 3.0
    power_weight: 0.0
    backend_preference: ["nvidia", "igpu", "npu"]

  balanced:
    power_budget_watts: 20
    latency_weight: 1.5
    power_weight: 1.0
    backend_preference: ["igpu", "nvidia", "npu"]

  efficiency:
    power_budget_watts: 15
    latency_weight: 0.5
    power_weight: 2.5
    backend_preference: ["npu", "igpu"]

  quiet:
    power_budget_watts: 15
    latency_weight: 0.5
    power_weight: 2.0
    thermal_aggressive: true
    backend_preference: ["npu", "igpu"]

  ultra_efficiency:
    power_budget_watts: 10
    latency_weight: 0.0
    power_weight: 3.0
    backend_preference: ["npu"]
    max_concurrency: 2

  auto:
    enabled: true
    battery_thresholds:
      ultra_efficiency: 20
      efficiency: 50
      balanced: 80
    thermal_threshold: 85
    quiet_hours:
      enabled: true
      start_hour: 22
      end_hour: 7

# Thermal monitoring
thermal:
  enabled: true
  poll_interval_seconds: 5
  thresholds:
    warning: 80
    high: 85
    critical: 90
    recovery: 75
  sensors:
    cpu_thermal_zone: "/sys/class/thermal/thermal_zone0/temp"
    nvidia_gpu: "nvidia-smi"
  actions:
    notify_user: true
    auto_switch_mode: true
    mark_throttled: true

# Logging
logging:
  level: "info"
  format: "text"
  output: "stdout"
  stream_logging:
    enabled: false
    log_ttft: true
    log_inter_token: false
```

---

## Environment Variables

Override configuration with environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `OLLAMA_PROXY_CONFIG` | Config file path | `/etc/ollama-proxy/config.yaml` |
| `OLLAMA_PROXY_HTTP_PORT` | HTTP port | `8000` |
| `OLLAMA_PROXY_GRPC_PORT` | gRPC port | `9090` |
| `OLLAMA_PROXY_LOG_LEVEL` | Log level | `debug` |

**Usage:**
```bash
export OLLAMA_PROXY_HTTP_PORT=8000
./ollama-proxy
```

---

## Validation

### Test Configuration

```bash
# Dry-run to validate config
./ollama-proxy --validate-config

# Expected output:
Configuration valid
```

### Common Validation Errors

**Missing required field:**
```
Error: backend ollama-npu missing required field: endpoint
```

**Invalid value:**
```
Error: invalid log level: debugg (must be: debug, info, warn, error)
```

**Port conflict:**
```
Error: grpc_port and http_port cannot be the same
```

---

## Related Documentation

- [Installation Guide](installation.md) - Installation instructions
- [Troubleshooting](troubleshooting.md) - Common configuration issues
- [Efficiency Modes](../features/efficiency-modes.md) - Mode details
- [Thermal Monitoring](../features/thermal-monitoring.md) - Thermal configuration
