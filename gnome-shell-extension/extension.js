/* extension.js
 *
 * Ollama Proxy GNOME Shell Extension
 * Adds a Quick Settings toggle for efficiency mode control
 */

const { GObject, St, Gio, GLib } = imports.gi;
const Main = imports.ui.main;
const QuickSettings = imports.ui.quickSettings;
const PopupMenu = imports.ui.popupMenu;

// D-Bus interface definitions
const EfficiencyInterface = `
<node>
  <interface name="ie.fio.OllamaProxy.Efficiency">
    <method name="SetMode">
      <arg type="s" direction="in" name="mode"/>
      <arg type="b" direction="out" name="success"/>
    </method>
    <method name="GetMode">
      <arg type="s" direction="out" name="mode"/>
    </method>
    <method name="GetEffectiveMode">
      <arg type="s" direction="out" name="mode"/>
    </method>
    <method name="ListModes">
      <arg type="as" direction="out" name="modes"/>
    </method>
    <property name="CurrentMode" type="s" access="read"/>
    <property name="EffectiveMode" type="s" access="read"/>
    <signal name="ModeChanged">
      <arg type="s" name="mode"/>
    </signal>
  </interface>
</node>
`;

const SystemInterface = `
<node>
  <interface name="ie.fio.OllamaProxy.SystemState">
    <method name="GetSystemState">
      <arg type="a{sv}" direction="out" name="state"/>
    </method>
    <property name="BatteryPercent" type="i" access="read"/>
    <property name="OnBattery" type="b" access="read"/>
  </interface>
</node>
`;

// Mode icons matching the system style
const MODE_INFO = {
    'Performance': { icon: 'power-profile-performance-symbolic', label: 'Performance' },
    'Balanced': { icon: 'power-profile-balanced-symbolic', label: 'Balanced' },
    'Efficiency': { icon: 'power-profile-power-saver-symbolic', label: 'Efficiency' },
    'Quiet': { icon: 'audio-volume-muted-symbolic', label: 'Quiet' },
    'Auto': { icon: 'emblem-system-symbolic', label: 'Auto' },
    'Ultra Efficiency': { icon: 'battery-level-10-symbolic', label: 'Ultra Efficiency' }
};

// D-Bus proxy wrapper
const EfficiencyProxy = Gio.DBusProxy.makeProxyWrapper(EfficiencyInterface);
const SystemProxy = Gio.DBusProxy.makeProxyWrapper(SystemInterface);

// Quick Settings Toggle for Ollama Proxy
const OllamaProxyToggle = GObject.registerClass(
class OllamaProxyToggle extends QuickSettings.QuickMenuToggle {
    _init() {
        super._init({
            title: 'AI Efficiency',
            iconName: 'power-profile-balanced-symbolic',
            toggleMode: true,
        });

        // Initialize D-Bus
        this._initDBus();

        // Create menu items
        this._buildMenu();

        // Update state
        this._updateState();

        // Set up periodic updates
        this._updateTimeout = GLib.timeout_add_seconds(GLib.PRIORITY_DEFAULT, 10, () => {
            this._updateState();
            return GLib.SOURCE_CONTINUE;
        });

        // Connect to toggle (for showing/hiding menu)
        this.connect('clicked', () => {
            this._updateState();
        });
    }

    _initDBus() {
        try {
            this._efficiencyProxy = new EfficiencyProxy(
                Gio.DBus.session,
                'ie.fio.OllamaProxy.Efficiency',
                '/com/anthropic/OllamaProxy/Efficiency',
                (proxy, error) => {
                    if (error) {
                        log(`Ollama Proxy: Failed to connect to Efficiency service: ${error}`);
                        this._showError();
                        return;
                    }
                    this._updateState();
                }
            );

            this._systemProxy = new SystemProxy(
                Gio.DBus.session,
                'ie.fio.OllamaProxy.SystemState',
                '/com/anthropic/OllamaProxy/SystemState',
                (proxy, error) => {
                    if (error) {
                        log(`Ollama Proxy: Failed to connect to System service: ${error}`);
                    }
                }
            );

            // Connect to signals
            this._efficiencyProxy?.connectSignal('ModeChanged', (proxy, sender, [mode]) => {
                this._onModeChanged(mode);
            });

        } catch (e) {
            log(`Ollama Proxy: D-Bus initialization error: ${e}`);
            this._showError();
        }
    }

    _buildMenu() {
        // Header showing current effective mode
        this._headerItem = new PopupMenu.PopupMenuItem('Loading...', {
            reactive: false,
            can_focus: false,
        });
        this._headerItem.label.set_style('font-weight: bold;');
        this.menu.addMenuItem(this._headerItem);

        this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

        // Mode selection items
        const modes = [
            'Performance',
            'Balanced',
            'Efficiency',
            'Quiet',
            'Auto',
            'Ultra Efficiency'
        ];

        this._modeItems = {};

        modes.forEach(mode => {
            const info = MODE_INFO[mode];
            const item = new PopupMenu.PopupImageMenuItem(info.label, info.icon);

            item.connect('activate', () => {
                this._setMode(mode);
            });

            this.menu.addMenuItem(item);
            this._modeItems[mode] = item;
        });

        this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

        // System state section (shown for Auto mode)
        this._systemStateItem = new PopupMenu.PopupMenuItem('', {
            reactive: false,
            can_focus: false,
        });
        this._systemStateItem.label.set_style('font-size: 9pt; color: #aaa;');
        this.menu.addMenuItem(this._systemStateItem);
    }

    _setMode(mode) {
        if (!this._efficiencyProxy) {
            log('Ollama Proxy: No connection to efficiency service');
            return;
        }

        try {
            this._efficiencyProxy.SetModeRemote(mode, (result, error) => {
                if (error) {
                    log(`Failed to set mode: ${error}`);
                    Main.notify('Ollama Proxy', `Failed to set mode: ${error.message}`);
                } else {
                    // Mode changed successfully
                    this._updateState();
                }
            });
        } catch (e) {
            log(`Error setting mode: ${e}`);
        }
    }

    _updateState() {
        if (!this._efficiencyProxy) {
            return;
        }

        try {
            // Get current mode
            this._efficiencyProxy.GetModeRemote((result, error) => {
                if (error) {
                    log(`Failed to get mode: ${error}`);
                    return;
                }
                const [mode] = result;
                this._currentMode = mode;
                this._updateDisplay();
            });

            // Get effective mode
            this._efficiencyProxy.GetEffectiveModeRemote((result, error) => {
                if (error) return;
                const [effectiveMode] = result;
                this._effectiveMode = effectiveMode;
                this._updateDisplay();
            });

            // Get system state for Auto mode
            if (this._systemProxy) {
                this._systemProxy.GetSystemStateRemote((result, error) => {
                    if (error) return;
                    const [state] = result;
                    this._systemState = state;
                    this._updateDisplay();
                });
            }
        } catch (e) {
            log(`Error updating state: ${e}`);
        }
    }

    _updateDisplay() {
        if (!this._currentMode) return;

        const info = MODE_INFO[this._currentMode];
        if (!info) return;

        // Update toggle title and icon
        this.title = `AI: ${info.label}`;
        this.iconName = info.icon;

        // Update header
        if (this._currentMode === 'Auto' && this._effectiveMode) {
            this._headerItem.label.text = `Auto Mode → ${this._effectiveMode}`;
        } else {
            this._headerItem.label.text = `Current Mode: ${this._currentMode}`;
        }

        // Update checkmarks
        Object.keys(this._modeItems).forEach(mode => {
            const item = this._modeItems[mode];
            if (mode === this._currentMode) {
                item.setOrnament(PopupMenu.Ornament.CHECK);
            } else {
                item.setOrnament(PopupMenu.Ornament.NONE);
            }
        });

        // Update system state text
        if (this._systemState) {
            const batteryPercent = this._systemState['battery_percent']?.unpack() || 0;
            const onBattery = this._systemState['on_battery']?.unpack() || false;
            const avgTemp = this._systemState['avg_temp']?.unpack() || 0;
            const avgFan = this._systemState['avg_fan_speed']?.unpack() || 0;

            const powerSource = onBattery ? 'Battery' : 'AC';
            const stateText = `${powerSource} ${batteryPercent}% • ${avgTemp.toFixed(0)}°C • Fan ${avgFan}%`;
            this._systemStateItem.label.text = stateText;
            this._systemStateItem.visible = true;
        } else {
            this._systemStateItem.visible = false;
        }

        // Toggle should always be "on" to show it's active
        this.checked = true;
    }

    _onModeChanged(mode) {
        this._currentMode = mode;
        this._updateState();

        // Show notification
        const info = MODE_INFO[mode];
        if (info) {
            Main.notify('Ollama Proxy', `Efficiency mode: ${info.label}`);
        }
    }

    _showError() {
        this.title = 'AI Efficiency';
        this.subtitle = 'Not Connected';
        this.iconName = 'dialog-error-symbolic';
        this.checked = false;

        if (this._headerItem) {
            this._headerItem.label.text = 'Ollama Proxy not running';
        }
    }

    destroy() {
        if (this._updateTimeout) {
            GLib.Source.remove(this._updateTimeout);
            this._updateTimeout = null;
        }
        super.destroy();
    }
});

// System indicator for additional status
const OllamaProxyIndicator = GObject.registerClass(
class OllamaProxyIndicator extends QuickSettings.SystemIndicator {
    _init() {
        super._init();

        // Create the indicator icon (shown in status area when important)
        this._indicator = this._addIndicator();
        this._indicator.iconName = 'power-profile-balanced-symbolic';
        this._indicator.visible = false; // Hidden by default, show on warnings

        // Create the toggle
        this.quickSettingsItems.push(new OllamaProxyToggle());

        // Add toggle to Quick Settings menu
        this.quickSettingsItems.forEach(item => {
            Main.panel.statusArea.quickSettings.addExternalIndicator(this);
            Main.panel.statusArea.quickSettings.menu.addMenuItem(item);
        });
    }

    destroy() {
        this.quickSettingsItems.forEach(item => item.destroy());
        super.destroy();
    }
});

class Extension {
    constructor() {
        this._indicator = null;
    }

    enable() {
        log('Ollama Proxy extension enabled');
        this._indicator = new OllamaProxyIndicator();
    }

    disable() {
        log('Ollama Proxy extension disabled');
        if (this._indicator) {
            this._indicator.destroy();
            this._indicator = null;
        }
    }
}

function init() {
    log('Ollama Proxy extension initialized');
    return new Extension();
}
