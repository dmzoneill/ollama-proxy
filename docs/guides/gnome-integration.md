# GNOME Integration Guide

Complete guide to integrating Ollama Proxy with the GNOME desktop environment.

---

## Overview

The Ollama Proxy provides deep GNOME integration through:
- **GNOME Shell Extension** - Quick Settings panel integration
- **D-Bus Services** - System-wide control and monitoring
- **GSettings** - GNOME-native configuration storage
- **Desktop Notifications** - Thermal events, mode changes
- **systemd User Service** - Automatic startup and management

---

## Components

### 1. GNOME Shell Extension

**Purpose:** Quick Settings panel integration for efficiency mode control

**Features:**
- Efficiency mode selector in Quick Settings
- Visual indicator with mode icon
- Backend status display
- Thermal state monitoring
- One-click mode switching

**Location:** `~/.local/share/gnome-shell/extensions/ollamaproxy@anthropic.com/`

### 2. D-Bus Services

**Purpose:** System-wide IPC for desktop integration

**Services:**
- `ie.fio.OllamaProxy.Efficiency` - Mode control
- `ie.fio.OllamaProxy.Backends` - Backend monitoring
- `ie.fio.OllamaProxy.Routing` - Routing statistics
- `ie.fio.OllamaProxy.Thermal` - Temperature monitoring
- `ie.fio.OllamaProxy.SystemState` - Battery/power state

### 3. GSettings Schema

**Purpose:** GNOME-native configuration storage

**Schema:** `ie.fio.ollamaproxy`

**Settings:**
- Display preferences
- Notification preferences
- Default modes
- Auto mode behavior

### 4. Desktop Entry

**Purpose:** Application launcher integration

**File:** `~/.local/share/applications/ie.fio.ollamaproxy.desktop`

**Provides:**
- Application menu entry
- Icon in application grid
- Launch with correct environment

---

## Installation

### Automated Installation

```bash
cd ~/src/ollama-proxy
./scripts/install-gnome-integration.sh
```

The script installs:
1. GSettings schema
2. Desktop entry
3. systemd user service
4. GNOME Shell extension

### Manual Installation

See [Installation Guide](installation.md#gnome-integration-optional) for manual steps.

---

## GNOME Shell Extension

### Extension Features

#### Quick Settings Integration

After installation, the extension adds an **"AI Efficiency"** toggle to the Quick Settings panel:

1. Click top-right corner (Quick Settings)
2. Look for "AI Efficiency" entry
3. Click to open mode selector

**Mode Selector:**
```
╔══════════════════════════════════════╗
║ AI Efficiency                        ║
╠══════════════════════════════════════╣
║ ⚡ Performance                        ║
║ ⚖️  Balanced             [Selected]  ║
║ 🔋 Efficiency                        ║
║ 🔇 Quiet                             ║
║ 🔄 Auto                              ║
║ 🪫 Ultra Efficiency                  ║
╠══════════════════════════════════════╣
║ 4 backends, 4 healthy                ║
║ CPU: 72°C, GPU: 68°C                 ║
╚══════════════════════════════════════╝
```

#### Visual Indicators

**Top Bar Indicator:**
- Icon changes based on mode
- Optional mode name display
- Optional backend count display

**Icons:**
- ⚡ Performance
- ⚖️ Balanced
- 🔋 Efficiency
- 🔇 Quiet
- 🔄 Auto
- 🪫 Ultra Efficiency

**Color Coding:**
- 🟢 Green: Normal temperature (<80°C)
- 🟡 Yellow: Elevated (80-85°C)
- 🟠 Orange: High (85-90°C)
- 🔴 Red: Critical (>90°C)

#### Backend Status

Shows current backend status:
```
Backends:
  ✅ NPU (3W) - Healthy, Queue: 0
  ✅ iGPU (12W) - Healthy, Queue: 2
  ⚠️ NVIDIA (55W) - Throttled (87°C)
  ❌ CPU (28W) - Unhealthy
```

### Extension Preferences

Open extension preferences:

```bash
# Via GNOME Extensions app
gnome-extensions prefs ollamaproxy@anthropic.com

# Or from command line
gnome-extensions prefs ollamaproxy@anthropic.com
```

**Settings:**

1. **Display**
   - Show indicator in top bar
   - Show mode name in indicator
   - Show backend count in indicator

2. **Notifications**
   - Notify when mode changes
   - Notify when backend fails
   - Notify when thermal throttling

3. **Mode Settings**
   - Remember last used mode
   - Default mode on startup

4. **Auto Mode**
   - Enable quiet hours
   - Quiet hours start (hour)
   - Quiet hours end (hour)

**Example preferences:**
```
Display:
  ☑ Show indicator in top bar
  ☑ Show mode name in indicator
  ☐ Show backend count in indicator

Notifications:
  ☑ Notify when mode changes
  ☑ Notify when backend fails
  ☑ Notify when thermal throttling

Mode Settings:
  ☑ Remember last used mode
  Default mode: Auto

Auto Mode:
  ☑ Enable quiet hours
  Start: 22 (10 PM)
  End: 7 (7 AM)
```

### Extension Code Structure

```
ollamaproxy@anthropic.com/
├── extension.js       # Main extension code
├── prefs.js          # Preferences UI
├── metadata.json     # Extension metadata
├── stylesheet.css    # Custom styles
└── schemas/
    └── org.gnome.shell.extensions.ollamaproxy.gschema.xml
```

---

## Desktop Notifications

The extension shows notifications for important events:

### Mode Change Notifications

```
╔══════════════════════════════════════╗
║ AI Proxy: Mode Changed               ║
╠══════════════════════════════════════╣
║ Switched to Efficiency mode          ║
║ Reason: Battery level 42%            ║
╚══════════════════════════════════════╝
```

### Thermal Event Notifications

```
╔══════════════════════════════════════╗
║ ⚠️ AI Proxy: High Temperature        ║
╠══════════════════════════════════════╣
║ GPU temperature: 87°C                ║
║ Switching to Quiet mode              ║
╚══════════════════════════════════════╝
```

### Backend Health Notifications

```
╔══════════════════════════════════════╗
║ ❌ AI Proxy: Backend Unhealthy       ║
╠══════════════════════════════════════╣
║ Backend ollama-nvidia is offline     ║
║ Routing to alternative backends      ║
╚══════════════════════════════════════╝
```

### Disable Notifications

**Via extension preferences:**
```
Notifications:
  ☐ Notify when mode changes
  ☐ Notify when backend fails
  ☐ Notify when thermal throttling
```

**Via GSettings:**
```bash
gsettings set ie.fio.ollamaproxy notify-on-mode-change false
gsettings set ie.fio.ollamaproxy notify-on-backend-failure false
gsettings set ie.fio.ollamaproxy notify-on-thermal-throttle false
```

---

## GSettings Configuration

### Schema Location

```bash
~/.local/share/glib-2.0/schemas/ie.fio.ollamaproxy.gschema.xml
```

### Available Settings

```bash
# List all settings
gsettings list-keys ie.fio.ollamaproxy
```

**Output:**
```
show-indicator
indicator-show-mode
indicator-show-backend-count
notify-on-mode-change
notify-on-backend-failure
notify-on-thermal-throttle
remember-last-mode
default-mode
auto-quiet-hours-enabled
auto-quiet-hours-start
auto-quiet-hours-end
```

### Get/Set Settings

**Get setting:**
```bash
gsettings get ie.fio.ollamaproxy default-mode
# Output: 'Auto'
```

**Set setting:**
```bash
gsettings set ie.fio.ollamaproxy default-mode 'Efficiency'
```

**Reset to default:**
```bash
gsettings reset ie.fio.ollamaproxy default-mode
```

### Common Settings

**Show mode in indicator:**
```bash
gsettings set ie.fio.ollamaproxy indicator-show-mode true
```

**Disable mode change notifications:**
```bash
gsettings set ie.fio.ollamaproxy notify-on-mode-change false
```

**Enable quiet hours:**
```bash
gsettings set ie.fio.ollamaproxy auto-quiet-hours-enabled true
gsettings set ie.fio.ollamaproxy auto-quiet-hours-start 22
gsettings set ie.fio.ollamaproxy auto-quiet-hours-end 7
```

### Monitor Changes

```bash
# Watch for changes
gsettings monitor ie.fio.ollamaproxy

# Make changes in extension preferences or via CLI
# Output:
default-mode: 'Balanced' → 'Efficiency'
```

---

## D-Bus Integration

### Query Current Mode

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

### Set Mode

```bash
busctl --user call ie.fio.OllamaProxy.Efficiency \
  /ie/fio/OllamaProxy/Efficiency \
  ie.fio.OllamaProxy.Efficiency \
  SetEfficiencyMode s "Efficiency"
```

### Monitor Mode Changes

```bash
busctl --user monitor ie.fio.OllamaProxy.Efficiency
```

**Output:**
```
signal sender=:1.123 -> destination=(null)
  path=/ie/fio/OllamaProxy/Efficiency
  interface=ie.fio.OllamaProxy.Efficiency
  member=ModeChanged
  string "Balanced"
  string "Efficiency"
  string "user_request"
```

See [D-Bus Services API](../api/dbus-services.md) for complete reference.

---

## systemd Integration

### Service File

**Location:** `~/.config/systemd/user/ie.fio.ollamaproxy.service`

**Contents:**
```ini
[Unit]
Description=Ollama Proxy Service
After=default.target

[Service]
Type=simple
WorkingDirectory=/home/YOUR_USERNAME/src/ollama-proxy
ExecStart=/usr/local/bin/ollama-proxy
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

### Service Control

```bash
# Start service
systemctl --user start ie.fio.ollamaproxy.service

# Stop service
systemctl --user stop ie.fio.ollamaproxy.service

# Restart service
systemctl --user restart ie.fio.ollamaproxy.service

# Enable auto-start
systemctl --user enable ie.fio.ollamaproxy.service

# Disable auto-start
systemctl --user disable ie.fio.ollamaproxy.service

# Check status
systemctl --user status ie.fio.ollamaproxy.service

# View logs
journalctl --user -u ie.fio.ollamaproxy.service -f
```

### Auto-Start on Login

```bash
# Enable
systemctl --user enable ie.fio.ollamaproxy.service

# Verify
systemctl --user is-enabled ie.fio.ollamaproxy.service
# Output: enabled
```

---

## Keyboard Shortcuts (Optional)

Add custom keyboard shortcuts for quick mode switching:

### Via GNOME Settings

1. Open **Settings** → **Keyboard** → **Keyboard Shortcuts**
2. Click **"+"** to add custom shortcut
3. Set:
   - **Name:** "AI Efficiency: Performance Mode"
   - **Command:** `busctl --user call ie.fio.OllamaProxy.Efficiency /ie/fio/OllamaProxy/Efficiency ie.fio.OllamaProxy.Efficiency SetEfficiencyMode s "Performance"`
   - **Shortcut:** Super+Shift+P

Repeat for other modes:
- Super+Shift+B → Balanced
- Super+Shift+E → Efficiency
- Super+Shift+Q → Quiet
- Super+Shift+A → Auto

### Via dconf

```bash
# Set keyboard shortcut for Performance mode
dconf write /org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/custom0/name "'AI Efficiency: Performance'"
dconf write /org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/custom0/command "'busctl --user call ie.fio.OllamaProxy.Efficiency /ie/fio/OllamaProxy/Efficiency ie.fio.OllamaProxy.Efficiency SetEfficiencyMode s \"Performance\"'"
dconf write /org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/custom0/binding "'<Super><Shift>p'"
```

---

## Troubleshooting

### Extension Not Showing

**Check if installed:**
```bash
ls ~/.local/share/gnome-shell/extensions/
# Should list: ollamaproxy@anthropic.com
```

**Check if enabled:**
```bash
gnome-extensions list
# Should show: ollamaproxy@anthropic.com
```

**Enable manually:**
```bash
gnome-extensions enable ollamaproxy@anthropic.com
```

**Restart GNOME Shell:**
- **X11:** Alt+F2, type `r`, press Enter
- **Wayland:** Log out and log back in

**Check for errors:**
```bash
journalctl -f /usr/bin/gnome-shell
```

### D-Bus Services Not Available

**Check if proxy is running:**
```bash
systemctl --user status ie.fio.ollamaproxy.service
```

**List D-Bus services:**
```bash
busctl --user list | grep ie.fio
```

**If not listed:**
- Service not running, or
- D-Bus registration failed

**Check logs:**
```bash
journalctl --user -u ie.fio.ollamaproxy.service | grep -i dbus
```

### GSettings Schema Not Found

**Check if schema is installed:**
```bash
gsettings list-schemas | grep ie.fio.ollamaproxy
```

**If not found:**
```bash
# Recompile schemas
glib-compile-schemas ~/.local/share/glib-2.0/schemas/

# Verify
gsettings list-schemas | grep ie.fio
```

### Notifications Not Showing

**Check notification settings:**
```bash
gsettings get ie.fio.ollamaproxy notify-on-mode-change
# Should be: true
```

**Enable notifications:**
```bash
gsettings set ie.fio.ollamaproxy notify-on-mode-change true
```

**Check GNOME notification settings:**
- Open **Settings** → **Notifications**
- Find **"Ollama Proxy"**
- Ensure notifications are enabled

### Extension Crashes

**View extension errors:**
```bash
journalctl -f /usr/bin/gnome-shell | grep -i ollama
```

**Disable extension:**
```bash
gnome-extensions disable ollamaproxy@anthropic.com
```

**Re-enable and check:**
```bash
gnome-extensions enable ollamaproxy@anthropic.com
journalctl -f /usr/bin/gnome-shell
```

**Common issues:**
- D-Bus service not running
- Malformed extension code
- GNOME Shell version incompatibility

---

## Advanced Customization

### Custom Extension Icons

Replace icons in extension:

```bash
cd ~/.local/share/gnome-shell/extensions/ollamaproxy@anthropic.com/icons/

# Add custom icons (SVG or PNG)
# Update extension.js to reference new icons
```

### Custom Notification Sounds

Add notification sound:

```bash
# Install notification sound
cp notification.ogg ~/.local/share/sounds/ollama-proxy/

# Reference in extension
```

### Multi-Monitor Support

Extension automatically adapts to multi-monitor setups. Indicator shows on primary monitor.

---

## Uninstallation

### Remove Extension

```bash
gnome-extensions disable ollamaproxy@anthropic.com
rm -rf ~/.local/share/gnome-shell/extensions/ollamaproxy@anthropic.com
```

### Remove GSettings Schema

```bash
rm ~/.local/share/glib-2.0/schemas/ie.fio.ollamaproxy.gschema.xml
glib-compile-schemas ~/.local/share/glib-2.0/schemas/
```

### Remove Desktop Entry

```bash
rm ~/.local/share/applications/ie.fio.ollamaproxy.desktop
update-desktop-database ~/.local/share/applications/
```

### Remove systemd Service

```bash
systemctl --user stop ie.fio.ollamaproxy.service
systemctl --user disable ie.fio.ollamaproxy.service
rm ~/.config/systemd/user/ie.fio.ollamaproxy.service
systemctl --user daemon-reload
```

---

## Related Documentation

- [Installation Guide](installation.md) - Installation instructions
- [D-Bus Services API](../api/dbus-services.md) - D-Bus API reference
- [Efficiency Modes](../features/efficiency-modes.md) - Mode descriptions
- [Troubleshooting](troubleshooting.md) - Common issues
