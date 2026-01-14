# Troubleshooting Guide

Common issues and solutions for the Ollama Proxy.

---

## Service Issues

### Service Won't Start

**Symptoms:**
```bash
systemctl --user status ie.fio.ollamaproxy.service
# Shows: failed (code=exited, status=1)
```

**Check logs:**
```bash
journalctl --user -u ie.fio.ollamaproxy.service -n 50
```

#### Issue 1: Config File Not Found

**Error:**
```
failed to read config file: open config/config.yaml: no such file or directory
```

**Cause:** Service working directory not set correctly

**Solution:**
```bash
# Edit service file
nano ~/.config/systemd/user/ie.fio.ollamaproxy.service

# Add or update WorkingDirectory
[Service]
WorkingDirectory=/home/YOUR_USERNAME/src/ollama-proxy  # Update path

# Reload and restart
systemctl --user daemon-reload
systemctl --user restart ie.fio.ollamaproxy.service
```

#### Issue 2: Port Already in Use

**Error:**
```
bind: address already in use
```

**Find what's using the port:**
```bash
# Check port 8080
sudo lsof -i :8080

# Or
ss -tlnp | grep 8080
```

**Solutions:**

1. **Stop conflicting service:**
   ```bash
   # If another instance is running
   pkill ollama-proxy
   ```

2. **Change port in config:**
   ```yaml
   # config/config.yaml
   server:
     http_port: 8000  # Different port
   ```

#### Issue 3: Backend Unreachable

**Error:**
```
failed to connect to backend ollama-npu: connection refused
```

**Check if Ollama is running:**
```bash
systemctl status ollama
# Or
ps aux | grep ollama
```

**Start Ollama if needed:**
```bash
# System service
sudo systemctl start ollama

# Or run manually
ollama serve
```

**Test Ollama connection:**
```bash
curl http://localhost:11434/api/tags
```

#### Issue 4: Permission Denied

**Error:**
```
permission denied: /usr/local/bin/ollama-proxy
```

**Fix permissions:**
```bash
sudo chmod +x /usr/local/bin/ollama-proxy
```

**Check file ownership:**
```bash
ls -la /usr/local/bin/ollama-proxy
# Should show: -rwxr-xr-x
```

---

## API Issues

### 503 Service Unavailable

**Error:**
```json
{
  "error": {
    "message": "No healthy backends available",
    "type": "service_unavailable",
    "code": "no_available_backends"
  }
}
```

**Check backend health:**
```bash
curl http://localhost:8080/backends | jq '.[] | {id: .id, healthy: .health.healthy}'
```

**Common causes:**

1. **All backends unhealthy:**
   ```bash
   # Check Ollama
   systemctl status ollama
   curl http://localhost:11434/api/tags
   ```

2. **Constraints too strict:**
   ```bash
   # Example: X-Max-Latency-Ms: 50 filters out all backends
   # Remove constraint or increase threshold
   ```

3. **Model not supported:**
   ```bash
   # Check which backends support your model
   curl http://localhost:8080/backends | jq '.[] | {id: .id, models: .supported_models}'
   ```

### 404 Model Not Found

**Error:**
```json
{
  "error": {
    "message": "Model llama3.2:70b not found",
    "type": "invalid_request_error",
    "param": "model"
  }
}
```

**Check available models:**
```bash
# Via proxy
curl http://localhost:8080/v1/models

# Via Ollama directly
ollama list
```

**Pull missing model:**
```bash
ollama pull llama3.2:70b
```

**Note:** Large models (>10GB) may not fit on NPU/iGPU backends.

### Streaming Not Working

**Symptoms:**
- No tokens received
- Connection hangs

**Test streaming:**
```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

**Common issues:**

1. **Missing -N flag:**
   ```bash
   # ❌ Without -N (buffered)
   curl http://...

   # ✅ With -N (unbuffered)
   curl -N http://...
   ```

2. **Reverse proxy buffering:**
   ```nginx
   # If using nginx, disable buffering
   proxy_buffering off;
   proxy_request_buffering off;
   ```

3. **Backend not streaming:**
   ```bash
   # Test Ollama directly
   curl -N http://localhost:11434/api/generate \
     -d '{"model": "qwen2.5:0.5b", "prompt": "Hello", "stream": true}'
   ```

### High Latency

**Symptoms:**
- Requests taking >5 seconds
- Slow token generation

**Measure latency:**
```bash
time curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

**Check response headers:**
```bash
curl -i http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{...}'

# Look for:
# X-Backend-Used: ollama-npu (which backend was used?)
# X-Estimated-Latency-Ms: 800 (expected latency)
```

**Common causes:**

1. **Routed to slow backend:**
   ```bash
   # Force faster backend
   curl http://localhost:8080/v1/chat/completions \
     -H "X-Target-Backend: ollama-nvidia" \
     -d '{...}'
   ```

2. **Backend congested:**
   ```bash
   # Check queue depths
   curl http://localhost:8080/backends | jq '.[] | {id: .id, queue: .queue_depth}'
   ```

3. **Model loading time:**
   ```bash
   # First request loads model (slow)
   # Subsequent requests faster
   ```

---

## Routing Issues

### Wrong Backend Selected

**Symptoms:**
- Expected NVIDIA but got NPU
- Power-efficient mode routing to GPU

**Debug routing decision:**
```bash
curl -i http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Check headers:
# X-Backend-Used: ollama-npu
# X-Routing-Reason: power-efficient-mode
```

**Check efficiency mode:**
```bash
curl http://localhost:8080/efficiency
```

**Common issues:**

1. **Efficiency mode forcing backend:**
   ```bash
   # Set different mode
   curl -X POST http://localhost:8080/efficiency \
     -d '{"mode": "Performance"}'
   ```

2. **Backend unhealthy:**
   ```bash
   # Check backend health
   curl http://localhost:8080/backends | grep -A 5 ollama-nvidia
   ```

3. **Constraints excluding backend:**
   ```bash
   # Example: X-Max-Power-Watts: 10 excludes NVIDIA (55W)
   # Remove constraint or increase threshold
   ```

### Backends Not Balancing

**Symptoms:**
- All requests go to one backend
- Other backends idle

**Check routing stats:**
```bash
curl http://localhost:8080/backends | jq '.[] | {id: .id, request_count: .metrics.request_count}'
```

**Possible causes:**

1. **Priority differences:**
   ```yaml
   # config/config.yaml
   backends:
     - id: ollama-nvidia
       priority: 1  # Always preferred

     - id: ollama-npu
       priority: 0  # Never used unless nvidia fails
   ```

   **Solution:** Equalize priorities.

2. **Power/latency constraints:**
   ```bash
   # If all requests have X-Power-Efficient: true
   # Only NPU will be used
   ```

3. **Queue depth not considered:**
   ```yaml
   # config/config.yaml
   router:
     queue_depth_penalty_per_request: 0  # Disabled
   ```

   **Solution:** Enable queue depth penalty (default: 50).

---

## Performance Issues

### Low Throughput

**Symptoms:**
- Only handling 5-10 req/sec
- Expected 50+ req/sec

**Test concurrency:**
```bash
# Install wrk
sudo dnf install wrk  # Fedora
sudo apt install wrk  # Ubuntu

# Benchmark
wrk -t4 -c100 -d30s --latency \
  -s scripts/benchmark.lua \
  http://localhost:8080/v1/chat/completions
```

**Common bottlenecks:**

1. **Backend limit:**
   ```bash
   # Check Ollama concurrency
   # Ollama typically handles 1-4 concurrent requests
   ```

2. **Too few backends:**
   ```bash
   # Add more backends or increase concurrency per backend
   ```

3. **Connection limit:**
   ```yaml
   # config/config.yaml
   server:
     max_concurrent_streams: 10  # Too low

   # Increase:
   max_concurrent_streams: 100
   ```

### Memory Usage Growing

**Symptoms:**
- Memory usage increasing over time
- Eventually runs out of memory

**Monitor memory:**
```bash
watch -n 1 'systemctl --user status ie.fio.ollamaproxy.service | grep Memory'
```

**Check for leaks:**
```bash
# Enable Go memory profiling
GODEBUG=gctrace=1 ./ollama-proxy

# Watch for growing heap size
```

**Common causes:**

1. **Slow clients (backpressure issue):**
   ```bash
   # Check for stalled streams
   curl http://localhost:8080/debug/pprof/goroutine?debug=1 | grep -i stream
   ```

2. **Object pools not releasing:**
   ```bash
   # Check pool sizes
   curl http://localhost:8080/debug/pprof/heap
   ```

**Solution:** Restart service if memory leak suspected:
```bash
systemctl --user restart ie.fio.ollamaproxy.service
```

---

## GNOME Integration Issues

### Extension Not Showing

See [GNOME Integration Guide](gnome-integration.md#troubleshooting) for detailed steps.

**Quick fixes:**
```bash
# Check if installed
ls ~/.local/share/gnome-shell/extensions/ollamaproxy@anthropic.com

# Enable if disabled
gnome-extensions enable ollamaproxy@anthropic.com

# Restart GNOME Shell (X11)
Alt+F2, type 'r', Enter

# Check for errors
journalctl -f /usr/bin/gnome-shell | grep -i ollama
```

### D-Bus Services Not Available

**Check service status:**
```bash
systemctl --user status ie.fio.ollamaproxy.service
```

**List D-Bus services:**
```bash
busctl --user list | grep ie.fio
```

**If missing:**
- Service not running
- D-Bus registration failed (check logs)

### Notifications Not Showing

**Enable notifications:**
```bash
gsettings set ie.fio.ollamaproxy notify-on-mode-change true
gsettings set ie.fio.ollamaproxy notify-on-backend-failure true
gsettings set ie.fio.ollamaproxy notify-on-thermal-throttle true
```

**Check GNOME notification settings:**
- Settings → Notifications
- Find "Ollama Proxy"
- Ensure enabled

---

## Thermal Issues

### False Thermal Events

**Symptoms:**
- Thermal events triggered but temperature normal
- Frequent mode switching

**Check temperature reading:**
```bash
# Read raw sensor
cat /sys/class/thermal/thermal_zone0/temp
# Should be in millidegrees (e.g., 72000 = 72°C)

# Check proxy reading
curl http://localhost:8080/thermal | jq '.cpu_temperature_c'
```

**If values don't match:**
- Proxy may be reading wrong sensor
- Unit conversion issue

**Fix sensor path:**
```yaml
# config/config.yaml
thermal:
  sensors:
    cpu_thermal_zone: "/sys/class/thermal/thermal_zone2/temp"  # Correct sensor
```

### Temperature Not Monitored

**Check thermal monitoring:**
```bash
curl http://localhost:8080/thermal
```

**If null/error:**
```bash
# Check config
grep -A 5 "thermal:" config/config.yaml

# Ensure enabled
thermal:
  enabled: true
```

**Find thermal sensors:**
```bash
# List all thermal zones
for zone in /sys/class/thermal/thermal_zone*/type; do
  echo "$zone: $(cat $zone)"
done

# Update config with correct zone
```

### GPU Temperature Not Showing

**NVIDIA GPU:**
```bash
# Check nvidia-smi available
nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader

# If not found, install drivers
sudo dnf install nvidia-driver
```

**AMD GPU:**
```bash
# Find AMD sensor
find /sys/class/drm/card*/device/hwmon/hwmon*/temp1_input

# Update config
thermal:
  sensors:
    amd_gpu: "/sys/class/drm/card0/device/hwmon/hwmon0/temp1_input"
```

---

## Configuration Issues

### Config Not Loading

**Check file location:**
```bash
# Service working directory
grep WorkingDirectory ~/.config/systemd/user/ie.fio.ollamaproxy.service

# Config should be in: WorkingDirectory/config/config.yaml
ls /home/YOUR_USERNAME/src/ollama-proxy/config/config.yaml
```

**Validate config syntax:**
```bash
# YAML syntax error?
yamllint config/config.yaml

# Or test with proxy
./ollama-proxy --validate-config
```

### Changes Not Applied

**Restart required:**
```bash
# Config changes require restart
systemctl --user restart ie.fio.ollamaproxy.service

# Check logs
journalctl --user -u ie.fio.ollamaproxy.service -n 20
```

### Invalid Configuration

**Common syntax errors:**

1. **Indentation:**
   ```yaml
   # ❌ Wrong indentation
   server:
   grpc_port: 50051

   # ✅ Correct
   server:
     grpc_port: 50051
   ```

2. **Missing quotes:**
   ```yaml
   # ❌ String with spaces needs quotes
   name: Ollama NPU

   # ✅ Quoted
   name: "Ollama NPU"
   ```

3. **Wrong types:**
   ```yaml
   # ❌ String for integer
   grpc_port: "50051"

   # ✅ Integer
   grpc_port: 50051
   ```

**Validate with yamllint:**
```bash
sudo dnf install yamllint
yamllint config/config.yaml
```

---

## Network Issues

### Cannot Connect to Backend

**Test backend connectivity:**
```bash
# Direct connection
curl http://localhost:11434/api/tags

# If fails:
# - Ollama not running
# - Port blocked
# - Wrong endpoint
```

**Check firewall:**
```bash
# Fedora
sudo firewall-cmd --list-all

# Allow port if needed
sudo firewall-cmd --add-port=11434/tcp --permanent
sudo firewall-cmd --reload
```

### Proxy Not Accessible from Network

**If you want to access from other machines:**

1. **Change bind address:**
   ```yaml
   # config/config.yaml
   server:
     host: "0.0.0.0"  # Listen on all interfaces
   ```

2. **Open firewall:**
   ```bash
   sudo firewall-cmd --add-port=8080/tcp --permanent
   sudo firewall-cmd --reload
   ```

3. **Test from remote:**
   ```bash
   # From another machine
   curl http://YOUR_IP:8080/health
   ```

**Security note:** Only expose proxy on trusted networks.

---

## Logging and Debugging

### Enable Debug Logging

```yaml
# config/config.yaml
logging:
  level: "debug"  # More verbose logging

  stream_logging:
    enabled: true
    log_inter_token: true  # Log every token (performance impact)
```

**Restart and view logs:**
```bash
systemctl --user restart ie.fio.ollamaproxy.service
journalctl --user -u ie.fio.ollamaproxy.service -f
```

### View Specific Log Types

```bash
# Routing decisions
journalctl --user -u ie.fio.ollamaproxy.service | grep -i routing

# Thermal events
journalctl --user -u ie.fio.ollamaproxy.service | grep -i thermal

# Backend health
journalctl --user -u ie.fio.ollamaproxy.service | grep -i health

# Errors only
journalctl --user -u ie.fio.ollamaproxy.service | grep -i error
```

### Performance Profiling

**Enable pprof endpoint:**
```bash
# Access profiling data
curl http://localhost:8080/debug/pprof/

# CPU profile
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Memory profile
curl http://localhost:8080/debug/pprof/heap > mem.prof

# Analyze with go tool
go tool pprof cpu.prof
```

---

## Getting Help

### Collect Diagnostic Information

```bash
#!/bin/bash
# diagnostic-info.sh

echo "=== System Info ==="
uname -a
cat /etc/os-release

echo "=== Service Status ==="
systemctl --user status ie.fio.ollamaproxy.service

echo "=== Recent Logs ==="
journalctl --user -u ie.fio.ollamaproxy.service -n 50

echo "=== Configuration ==="
cat ~/src/ollama-proxy/config/config.yaml

echo "=== Backend Health ==="
curl -s http://localhost:8080/backends | jq

echo "=== Thermal State ==="
curl -s http://localhost:8080/thermal | jq

echo "=== Efficiency Mode ==="
curl -s http://localhost:8080/efficiency | jq
```

**Run and save output:**
```bash
chmod +x diagnostic-info.sh
./diagnostic-info.sh > diagnostic-$(date +%Y%m%d-%H%M%S).txt
```

### Report Issues

When reporting issues, include:

1. **System information:**
   - OS and version
   - GNOME version (if using extension)
   - Go version

2. **Error logs:**
   - Service logs (last 50 lines)
   - GNOME Shell logs (if extension issue)

3. **Configuration:**
   - config.yaml (remove sensitive data)

4. **Steps to reproduce:**
   - What you did
   - What you expected
   - What actually happened

5. **Diagnostic output:**
   - Output from diagnostic script above

**Submit to:**
- GitHub Issues: https://github.com/daoneill/ollama-proxy/issues

---

## Common Error Messages

| Error | Meaning | Solution |
|-------|---------|----------|
| `bind: address already in use` | Port conflict | Change port or stop conflicting service |
| `connection refused` | Backend not running | Start Ollama |
| `no healthy backends` | All backends unavailable | Check backend health |
| `model not found` | Model not pulled | Run `ollama pull <model>` |
| `permission denied` | File permissions wrong | `chmod +x` on binary |
| `config file not found` | Wrong working directory | Update service WorkingDirectory |
| `timeout` | Request took too long | Check backend performance |
| `service unknown` (D-Bus) | Proxy not running | Start service |

---

## Related Documentation

- [Installation Guide](installation.md) - Installation instructions
- [Configuration Guide](configuration.md) - Configuration reference
- [GNOME Integration](gnome-integration.md) - Desktop integration
- [OpenAI API](../api/openai-compatibility.md) - API usage
