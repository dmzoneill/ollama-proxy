/* AI Efficiency Mode - GNOME Shell Extension
 * Adds AI Efficiency mode to Quick Settings menu
 */

const {GObject, Gio, St} = imports.gi;
const QuickSettings = imports.ui.quickSettings;
const QuickSettingsMenu = imports.ui.main.panel.statusArea.quickSettings;

// D-Bus interface
const AIEfficiencyInterface = `
<node>
  <interface name="com.anthropic.OllamaProxy.Efficiency">
    <method name="SetMode">
      <arg type="s" direction="in" name="mode"/>
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
    <signal name="ModeChanged">
      <arg type="s" name="oldMode"/>
      <arg type="s" name="newMode"/>
    </signal>
    <property name="CurrentMode" type="s" access="readwrite"/>
    <property name="EffectiveMode" type="s" access="read"/>
  </interface>
</node>`;

const AIEfficiencyProxy = Gio.DBusProxy.makeProxyWrapper(AIEfficiencyInterface);

// Mode icons
const ModeIcons = {
    'Performance': '🚀',
    'Balanced': '⚖️',
    'Efficiency': '🔋',
    'Quiet': '🔇',
    'Auto': '🤖',
    'UltraEfficiency': '🪫'
};

// Toggle for AI Efficiency
const AIEfficiencyToggle = GObject.registerClass(
class AIEfficiencyToggle extends QuickSettings.QuickMenuToggle {
    _init() {
        super._init({
            title: _('AI Efficiency'),
            iconName: 'ai-efficiency-symbolic',
            toggleMode: false,
        });

        // Connect to D-Bus
        this._proxy = new AIEfficiencyProxy(
            Gio.DBus.session,
            'com.anthropic.OllamaProxy.Efficiency',
            '/com/anthropic/OllamaProxy/Efficiency',
            this._onProxyReady.bind(this)
        );

        // Create menu
        this.menu.setHeader('ai-efficiency-symbolic', _('AI Efficiency'));
        this._addMenuItems();
    }

    _onProxyReady(proxy, error) {
        if (error) {
            log(`AI Efficiency Proxy error: ${error.message}`);
            return;
        }

        // Update current mode
        this._updateMode();

        // Listen for changes
        this._proxy.connectSignal('ModeChanged', () => {
            this._updateMode();
        });
    }

    _addMenuItems() {
        const modes = [
            'Performance',
            'Balanced',
            'Efficiency',
            'Quiet',
            'Auto',
            'UltraEfficiency'
        ];

        const descriptions = {
            'Performance': 'Maximum speed',
            'Balanced': 'Smart routing',
            'Efficiency': 'Low power',
            'Quiet': 'Minimal noise',
            'Auto': 'Automatic',
            'UltraEfficiency': 'Max battery'
        };

        this._modeItems = {};

        modes.forEach(mode => {
            const item = new QuickSettings.QuickMenuToggle({
                title: mode,
                subtitle: descriptions[mode],
                iconName: 'object-select-symbolic',
            });

            item.connect('clicked', () => {
                this._setMode(mode);
            });

            this.menu.addMenuItem(item);
            this._modeItems[mode] = item;
        });

        // Add separator
        this.menu.addMenuItem(new imports.ui.popupMenu.PopupSeparatorMenuItem());

        // Add settings button
        const settingsItem = this.menu.addAction(
            _('Settings'),
            () => {
                imports.misc.util.spawn(['gnome-control-center', 'power']);
            }
        );
    }

    _updateMode() {
        if (!this._proxy) return;

        try {
            const currentMode = this._proxy.CurrentMode;
            const effectiveMode = this._proxy.EffectiveMode;

            // Update subtitle
            let subtitle = currentMode;
            if (currentMode !== effectiveMode) {
                subtitle = `${currentMode} (${effectiveMode})`;
            }
            this.subtitle = subtitle;

            // Update menu items
            Object.keys(this._modeItems).forEach(mode => {
                const item = this._modeItems[mode];
                item.checked = (mode === currentMode);
            });

        } catch (e) {
            log(`Error updating AI Efficiency mode: ${e.message}`);
        }
    }

    _setMode(mode) {
        if (!this._proxy) return;

        try {
            this._proxy.SetModeRemote(mode, (result, error) => {
                if (error) {
                    log(`Error setting AI Efficiency mode: ${error.message}`);
                } else {
                    // Mode changed successfully
                    imports.ui.main.notify(
                        'AI Efficiency',
                        `Mode changed to ${mode}`
                    );
                }
            });
        } catch (e) {
            log(`Error calling SetMode: ${e.message}`);
        }
    }
});

class Extension {
    constructor() {
        this._indicator = null;
    }

    enable() {
        this._indicator = new AIEfficiencyToggle();
        QuickSettingsMenu.menu.addMenuItem(this._indicator);
    }

    disable() {
        if (this._indicator) {
            this._indicator.destroy();
            this._indicator = null;
        }
    }
}

function init() {
    return new Extension();
}
