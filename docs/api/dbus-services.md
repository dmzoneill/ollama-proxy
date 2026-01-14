# D-Bus Services

The Ollama Proxy exposes **5 D-Bus services** for system-wide monitoring, control, and desktop integration.

---

## Overview

D-Bus services provide system-level integration allowing:
- **Desktop integration** - GNOME Shell extension control
- **System monitoring** - Battery, thermal, backend health
- **Mode control** - Efficiency mode switching
- **Service discovery** - Applications can find and control the proxy
- **Event notifications** - Real-time state changes

### Available Services

| Service | Interface | Purpose |
|---------|-----------|---------|
| **Efficiency** | `ie.fio.OllamaProxy.Efficiency` | Efficiency mode control |
| **Backends** | `ie.fio.OllamaProxy.Backends` | Backend monitoring |
| **Routing** | `ie.fio.OllamaProxy.Routing` | Routing statistics |
| **Thermal** | `ie.fio.OllamaProxy.Thermal` | Temperature monitoring |
| **SystemState** | `ie.fio.OllamaProxy.SystemState` | System state (battery, AC) |

---

## Connection

### Bus Type

Session bus (user-level services):

```bash
# Query session bus
busctl --user list | grep OllamaProxy
```

### Service Names

```
ie.fio.OllamaProxy.Efficiency
ie.fio.OllamaProxy.Backends
ie.fio.OllamaProxy.Routing
ie.fio.OllamaProxy.Thermal
ie.fio.OllamaProxy.SystemState
```

### Object Paths

```
/ie/fio/OllamaProxy/Efficiency
/ie/fio/OllamaProxy/Backends
/ie/fio/OllamaProxy/Routing
/ie/fio/OllamaProxy/Thermal
/ie/fio/OllamaProxy/SystemState
```

---

## Efficiency Service

Control efficiency modes via D-Bus.

### Interface

```
ie.fio.OllamaProxy.Efficiency
```

### Methods

#### SetEfficiencyMode

Set the current efficiency mode.

**Signature:** `s → ()`
**Parameters:**
- `mode` (string) - Mode name: "Performance", "Balanced", "Efficiency", "Quiet", "Auto", "UltraEfficiency"

**Example (busctl):**
```bash
busctl --user call ie.fio.OllamaProxy.Efficiency \
  /ie/fio/OllamaProxy/Efficiency \
  ie.fio.OllamaProxy.Efficiency \
  SetEfficiencyMode s "Balanced"
```

**Example (Python):**
```python
from gi.repository import Gio

bus = Gio.bus_get_sync(Gio.BusType.SESSION, None)
proxy = Gio.DBusProxy.new_sync(
    bus,
    Gio.DBusProxyFlags.NONE,
    None,
    'ie.fio.OllamaProxy.Efficiency',
    '/ie/fio/OllamaProxy/Efficiency',
    'ie.fio.OllamaProxy.Efficiency',
    None
)

proxy.call_sync('SetEfficiencyMode', GLib.Variant('(s)', ('Efficiency',)), 0, -1, None)
```

#### GetEfficiencyMode

Get the current efficiency mode.

**Signature:** `() → s`
**Returns:** Mode name (string)

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Efficiency \
  /ie/fio/OllamaProxy/Efficiency \
  ie.fio.OllamaProxy.Efficiency \
  GetEfficiencyMode
```

**Response:**
```
s "Balanced"
```

#### GetAvailableModes

List all available efficiency modes.

**Signature:** `() → as`
**Returns:** Array of mode names

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Efficiency \
  /ie/fio/OllamaProxy/Efficiency \
  ie.fio.OllamaProxy.Efficiency \
  GetAvailableModes
```

**Response:**
```
as 6 "Performance" "Balanced" "Efficiency" "Quiet" "Auto" "UltraEfficiency"
```

### Signals

#### ModeChanged

Emitted when efficiency mode changes.

**Signature:** `sss`
**Parameters:**
- `old_mode` (string) - Previous mode
- `new_mode` (string) - New mode
- `reason` (string) - Reason for change

**Monitor:**
```bash
busctl --user monitor ie.fio.OllamaProxy.Efficiency
```

**Example signal:**
```
signal sender=:1.123 -> destination=(null) serial=42
  path=/ie/fio/OllamaProxy/Efficiency
  interface=ie.fio.OllamaProxy.Efficiency
  member=ModeChanged
  string "Balanced"
  string "Efficiency"
  string "battery_level_42%"
```

**Python handler:**
```python
def on_mode_changed(proxy, sender, signal, parameters):
    old_mode, new_mode, reason = parameters.unpack()
    print(f"Mode changed: {old_mode} → {new_mode} ({reason})")

proxy.connect('g-signal', on_mode_changed)
```

### Properties

#### CurrentMode (readable)

**Type:** `s` (string)
**Description:** Current efficiency mode

```bash
busctl --user get-property ie.fio.OllamaProxy.Efficiency \
  /ie/fio/OllamaProxy/Efficiency \
  ie.fio.OllamaProxy.Efficiency \
  CurrentMode
```

#### AutoModeEnabled (readable)

**Type:** `b` (boolean)
**Description:** Whether auto mode is active

---

## Backends Service

Monitor backend health and status.

### Interface

```
ie.fio.OllamaProxy.Backends
```

### Methods

#### ListBackends

Get list of all backends.

**Signature:** `() → a(ssbiiddi)`
**Returns:** Array of backend info tuples:
- `id` (string)
- `name` (string)
- `healthy` (boolean)
- `avg_latency_ms` (int32)
- `power_watts` (int32)
- `queue_depth` (double)
- `cpu_temp_c` (double)
- `priority` (int32)

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Backends \
  /ie/fio/OllamaProxy/Backends \
  ie.fio.OllamaProxy.Backends \
  ListBackends
```

**Response:**
```
a(ssbiiddi) 4
  "ollama-npu" "Ollama NPU" true 800 3 0.0 65.5 3
  "ollama-igpu" "Ollama iGPU" true 400 12 2.0 70.2 2
  "ollama-nvidia" "Ollama NVIDIA" true 150 55 1.0 75.8 1
  "ollama-cpu" "Ollama CPU" false 2000 28 0.0 68.0 0
```

**Python:**
```python
result = proxy.call_sync('ListBackends', None, 0, -1, None)
backends = result.unpack()[0]

for backend in backends:
    backend_id, name, healthy, latency, power, queue, temp, priority = backend
    print(f"{name}: {'✓' if healthy else '✗'}, {latency}ms, {power}W, queue={queue}")
```

#### GetBackendDetails

Get detailed info for a specific backend.

**Signature:** `s → (ssbiiddii)`
**Parameters:**
- `backend_id` (string)

**Returns:** Backend details tuple

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Backends \
  /ie/fio/OllamaProxy/Backends \
  ie.fio.OllamaProxy.Backends \
  GetBackendDetails s "ollama-npu"
```

#### GetBackendHealth

Check if backend is healthy.

**Signature:** `s → b`
**Parameters:**
- `backend_id` (string)

**Returns:** `true` if healthy, `false` otherwise

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Backends \
  /ie/fio/OllamaProxy/Backends \
  ie.fio.OllamaProxy.Backends \
  GetBackendHealth s "ollama-nvidia"
```

### Signals

#### BackendHealthChanged

Emitted when backend health changes.

**Signature:** `sbs`
**Parameters:**
- `backend_id` (string)
- `healthy` (boolean)
- `reason` (string)

**Example:**
```
signal member=BackendHealthChanged
  string "ollama-nvidia"
  boolean false
  string "connection_timeout"
```

#### BackendAdded

Emitted when new backend is added.

**Signature:** `ss`
**Parameters:**
- `backend_id` (string)
- `backend_name` (string)

---

## Routing Service

Query routing statistics and decisions.

### Interface

```
ie.fio.OllamaProxy.Routing
```

### Methods

#### GetRoutingStats

Get routing statistics.

**Signature:** `() → (iia{si})`
**Returns:**
- `total_requests` (int32)
- `failed_requests` (int32)
- `backend_usage` (dict: backend_id → count)

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Routing \
  /ie/fio/OllamaProxy/Routing \
  ie.fio.OllamaProxy.Routing \
  GetRoutingStats
```

**Response:**
```
iia{si} 1543 12 4
  "ollama-npu" 450
  "ollama-igpu" 723
  "ollama-nvidia" 358
  "ollama-cpu" 12
```

**Python:**
```python
result = proxy.call_sync('GetRoutingStats', None, 0, -1, None)
total, failed, usage = result.unpack()

print(f"Total: {total}, Failed: {failed}")
for backend_id, count in usage.items():
    percentage = (count / total) * 100
    print(f"{backend_id}: {count} ({percentage:.1f}%)")
```

#### GetQueueDepths

Get current queue depth for all backends.

**Signature:** `() → a{si}`
**Returns:** Dict mapping backend_id → queue_depth

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Routing \
  /ie/fio/OllamaProxy/Routing \
  ie.fio.OllamaProxy.Routing \
  GetQueueDepths
```

**Response:**
```
a{si} 4
  "ollama-npu" 0
  "ollama-igpu" 3
  "ollama-nvidia" 5
  "ollama-cpu" 0
```

#### GetLastRoutingDecision

Get details of last routing decision.

**Signature:** `() → (ssis)`
**Returns:**
- `backend_id` (string)
- `reason` (string)
- `score` (int32)
- `timestamp` (string)

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Routing \
  /ie/fio/OllamaProxy/Routing \
  ie.fio.OllamaProxy.Routing \
  GetLastRoutingDecision
```

**Response:**
```
ssis "ollama-igpu" "balanced-mode-low-queue" 1450 "2025-01-11T14:30:25Z"
```

### Signals

#### RoutingDecisionMade

Emitted for each routing decision (if enabled).

**Signature:** `ssis`
**Parameters:**
- `backend_id` (string)
- `reason` (string)
- `score` (int32)
- `request_id` (string)

---

## Thermal Service

Monitor temperature and thermal events.

### Interface

```
ie.fio.OllamaProxy.Thermal
```

### Methods

#### GetThermalState

Get current thermal state.

**Signature:** `() → (ddis)`
**Returns:**
- `cpu_temp_c` (double)
- `gpu_temp_c` (double)
- `fan_speed_rpm` (int32)
- `thermal_state` (string) - "normal", "elevated", "high", "critical"

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Thermal \
  /ie/fio/OllamaProxy/Thermal \
  ie.fio.OllamaProxy.Thermal \
  GetThermalState
```

**Response:**
```
ddis 72.5 68.0 2300 "normal"
```

**Python:**
```python
result = proxy.call_sync('GetThermalState', None, 0, -1, None)
cpu_temp, gpu_temp, fan_speed, state = result.unpack()

print(f"CPU: {cpu_temp}°C, GPU: {gpu_temp}°C")
print(f"Fan: {fan_speed} RPM, State: {state}")
```

#### GetThermalHistory

Get temperature history (last N samples).

**Signature:** `i → a(ddt)`
**Parameters:**
- `num_samples` (int32) - Number of samples to return

**Returns:** Array of (cpu_temp, gpu_temp, timestamp) tuples

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Thermal \
  /ie/fio/OllamaProxy/Thermal \
  ie.fio.OllamaProxy.Thermal \
  GetThermalHistory i 10
```

#### GetThrottledBackends

Get list of thermally throttled backends.

**Signature:** `() → as`
**Returns:** Array of backend IDs

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.Thermal \
  /ie/fio/OllamaProxy/Thermal \
  ie.fio.OllamaProxy.Thermal \
  GetThrottledBackends
```

**Response:**
```
as 1 "ollama-nvidia"
```

### Signals

#### ThermalEvent

Emitted when thermal threshold exceeded.

**Signature:** `sdds`
**Parameters:**
- `event_type` (string) - "warning", "high", "critical"
- `cpu_temp_c` (double)
- `gpu_temp_c` (double)
- `action_taken` (string)

**Example:**
```
signal member=ThermalEvent
  string "high"
  double 87.5
  double 85.2
  string "switched_to_quiet_mode"
```

**Python handler:**
```python
def on_thermal_event(proxy, sender, signal, parameters):
    event_type, cpu_temp, gpu_temp, action = parameters.unpack()
    print(f"Thermal event: {event_type}")
    print(f"CPU: {cpu_temp}°C, GPU: {gpu_temp}°C")
    print(f"Action: {action}")

proxy.connect('g-signal', on_thermal_event)
```

#### ThermalRecovery

Emitted when temperature returns to normal.

**Signature:** `dd`
**Parameters:**
- `cpu_temp_c` (double)
- `gpu_temp_c` (double)

---

## SystemState Service

Monitor system power and battery state.

### Interface

```
ie.fio.OllamaProxy.SystemState
```

### Methods

#### GetBatteryState

Get battery level and power status.

**Signature:** `() → (ibi)`
**Returns:**
- `battery_level` (int32) - Percentage (0-100)
- `on_battery` (boolean) - True if on battery power
- `time_remaining_minutes` (int32) - Estimated time remaining

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.SystemState \
  /ie/fio/OllamaProxy/SystemState \
  ie.fio.OllamaProxy.SystemState \
  GetBatteryState
```

**Response:**
```
ibi 65 true 180
```

**Python:**
```python
result = proxy.call_sync('GetBatteryState', None, 0, -1, None)
level, on_battery, time_remaining = result.unpack()

print(f"Battery: {level}%")
print(f"Power: {'Battery' if on_battery else 'AC'}")
print(f"Time remaining: {time_remaining // 60}h {time_remaining % 60}m")
```

#### GetPowerProfile

Get current power profile.

**Signature:** `() → s`
**Returns:** "performance", "balanced", "power-saver"

**Example:**
```bash
busctl --user call ie.fio.OllamaProxy.SystemState \
  /ie/fio/OllamaProxy/SystemState \
  ie.fio.OllamaProxy.SystemState \
  GetPowerProfile
```

### Signals

#### BatteryStateChanged

Emitted when battery state changes.

**Signature:** `ib`
**Parameters:**
- `battery_level` (int32)
- `on_battery` (boolean)

**Example:**
```
signal member=BatteryStateChanged
  int32 45
  boolean true
```

#### PowerProfileChanged

Emitted when power profile changes.

**Signature:** `ss`
**Parameters:**
- `old_profile` (string)
- `new_profile` (string)

**Example:**
```
signal member=PowerProfileChanged
  string "performance"
  string "power-saver"
```

---

## Integration Examples

### Python (GLib)

```python
from gi.repository import Gio, GLib

# Connect to Efficiency service
bus = Gio.bus_get_sync(Gio.BusType.SESSION, None)

efficiency_proxy = Gio.DBusProxy.new_sync(
    bus,
    Gio.DBusProxyFlags.NONE,
    None,
    'ie.fio.OllamaProxy.Efficiency',
    '/ie/fio/OllamaProxy/Efficiency',
    'ie.fio.OllamaProxy.Efficiency',
    None
)

# Get current mode
result = efficiency_proxy.call_sync('GetEfficiencyMode', None, 0, -1, None)
current_mode = result.unpack()[0]
print(f"Current mode: {current_mode}")

# Set new mode
efficiency_proxy.call_sync(
    'SetEfficiencyMode',
    GLib.Variant('(s)', ('Efficiency',)),
    0,
    -1,
    None
)

# Listen for mode changes
def on_signal(proxy, sender, signal_name, parameters):
    if signal_name == 'ModeChanged':
        old_mode, new_mode, reason = parameters.unpack()
        print(f"Mode changed: {old_mode} → {new_mode} ({reason})")

efficiency_proxy.connect('g-signal', on_signal)

# Run event loop
loop = GLib.MainLoop()
loop.run()
```

### Shell Script

```bash
#!/bin/bash
# Battery-aware mode switcher

get_battery_level() {
    busctl --user call ie.fio.OllamaProxy.SystemState \
        /ie/fio/OllamaProxy/SystemState \
        ie.fio.OllamaProxy.SystemState \
        GetBatteryState | awk '{print $2}'
}

set_efficiency_mode() {
    local mode=$1
    busctl --user call ie.fio.OllamaProxy.Efficiency \
        /ie/fio/OllamaProxy/Efficiency \
        ie.fio.OllamaProxy.Efficiency \
        SetEfficiencyMode s "$mode"
}

# Main loop
while true; do
    battery=$(get_battery_level)

    if [ $battery -lt 20 ]; then
        set_efficiency_mode "UltraEfficiency"
    elif [ $battery -lt 50 ]; then
        set_efficiency_mode "Efficiency"
    else
        set_efficiency_mode "Balanced"
    fi

    sleep 60
done
```

### GNOME Shell Extension

```javascript
const Gio = imports.gi.Gio;

// Efficiency proxy
const EfficiencyProxy = Gio.DBusProxy.makeProxyWrapper(
  '<node>\
    <interface name="ie.fio.OllamaProxy.Efficiency">\
      <method name="SetEfficiencyMode">\
        <arg type="s" direction="in" name="mode"/>\
      </method>\
      <method name="GetEfficiencyMode">\
        <arg type="s" direction="out" name="mode"/>\
      </method>\
      <signal name="ModeChanged">\
        <arg type="s" name="old_mode"/>\
        <arg type="s" name="new_mode"/>\
        <arg type="s" name="reason"/>\
      </signal>\
    </interface>\
  </node>'
);

// Create proxy
this._efficiencyProxy = new EfficiencyProxy(
    Gio.DBus.session,
    'ie.fio.OllamaProxy.Efficiency',
    '/ie/fio/OllamaProxy/Efficiency',
    (proxy, error) => {
        if (error) {
            log(`Error creating proxy: ${error}`);
            return;
        }

        // Get current mode
        this._efficiencyProxy.GetEfficiencyModeRemote((result, error) => {
            if (!error) {
                let [mode] = result;
                log(`Current mode: ${mode}`);
            }
        });
    }
);

// Listen for mode changes
this._efficiencyProxy.connectSignal('ModeChanged', (proxy, sender, [oldMode, newMode, reason]) => {
    log(`Mode changed: ${oldMode} → ${newMode} (${reason})`);
    this._updateIndicator(newMode);
});

// Set mode
setMode(mode) {
    this._efficiencyProxy.SetEfficiencyModeRemote(mode, (result, error) => {
        if (error) {
            log(`Error setting mode: ${error}`);
        }
    });
}
```

---

## Introspection

### List All Services

```bash
busctl --user list | grep ie.fio.OllamaProxy
```

### Introspect Service

```bash
busctl --user introspect ie.fio.OllamaProxy.Efficiency \
  /ie/fio/OllamaProxy/Efficiency
```

**Output:**
```
NAME                                TYPE      SIGNATURE  RESULT/VALUE  FLAGS
ie.fio.OllamaProxy.Efficiency       interface -          -             -
.GetEfficiencyMode                  method    -          s             -
.SetEfficiencyMode                  method    s          -             -
.GetAvailableModes                  method    -          as            -
.ModeChanged                        signal    sss        -             -
.CurrentMode                        property  s          "Balanced"    emits-change
```

### Monitor All Signals

```bash
busctl --user monitor ie.fio.OllamaProxy.Efficiency
busctl --user monitor ie.fio.OllamaProxy.Thermal
```

---

## Best Practices

### 1. Cache Proxy Connections

Don't create a new proxy for each call:

```python
# ✅ Good: Reuse proxy
class ProxyClient:
    def __init__(self):
        bus = Gio.bus_get_sync(Gio.BusType.SESSION, None)
        self.efficiency = Gio.DBusProxy.new_sync(...)

    def set_mode(self, mode):
        self.efficiency.call_sync('SetEfficiencyMode', ...)

# ❌ Bad: New proxy each time
def set_mode(mode):
    bus = Gio.bus_get_sync(Gio.BusType.SESSION, None)
    proxy = Gio.DBusProxy.new_sync(...)
    proxy.call_sync('SetEfficiencyMode', ...)
```

### 2. Handle Service Unavailability

```python
try:
    result = proxy.call_sync('GetEfficiencyMode', None, 0, 1000, None)
except GLib.Error as e:
    if 'ServiceUnknown' in str(e):
        print("Proxy service not running")
    else:
        raise
```

### 3. Use Signals for Real-Time Updates

```python
# ✅ Good: Use signals
proxy.connect('g-signal', on_mode_changed)

# ❌ Bad: Poll with timer
def poll_mode():
    mode = proxy.call_sync('GetEfficiencyMode', ...)
    GLib.timeout_add_seconds(1, poll_mode)
```

### 4. Set Timeouts

```python
# Set timeout (milliseconds)
result = proxy.call_sync('GetEfficiencyMode', None, 0, 5000, None)
```

---

## Related Documentation

- [GNOME Integration](../guides/gnome-integration.md) - Desktop integration guide
- [Efficiency Modes](../features/efficiency-modes.md) - Mode descriptions
- [Thermal Monitoring](../features/thermal-monitoring.md) - Thermal management
- [Power Management](../features/power-management.md) - Power-aware routing
