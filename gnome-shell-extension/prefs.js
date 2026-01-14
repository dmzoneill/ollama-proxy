/* prefs.js
 *
 * Ollama Proxy Extension Preferences
 * Configuration UI for the extension
 */

const { Gtk, Gio, GObject } = imports.gi;
const ExtensionUtils = imports.misc.extensionUtils;

function init() {
}

function buildPrefsWidget() {
    // Load settings
    let settings = ExtensionUtils.getSettings('ie.fio.ollamaproxy');

    // Create main box
    let prefsWidget = new Gtk.Box({
        orientation: Gtk.Orientation.VERTICAL,
        spacing: 10,
        margin_top: 20,
        margin_bottom: 20,
        margin_start: 20,
        margin_end: 20
    });

    // Title
    let title = new Gtk.Label({
        label: '<b>Ollama Proxy Settings</b>',
        use_markup: true,
        halign: Gtk.Align.START
    });
    prefsWidget.append(title);

    // Separator
    prefsWidget.append(new Gtk.Separator({ orientation: Gtk.Orientation.HORIZONTAL }));

    // Display Settings Section
    let displayLabel = new Gtk.Label({
        label: '<b>Display</b>',
        use_markup: true,
        halign: Gtk.Align.START,
        margin_top: 10
    });
    prefsWidget.append(displayLabel);

    // Show indicator toggle
    let showIndicatorBox = createSettingRow(
        'Show indicator in top bar',
        'show-indicator',
        settings,
        'boolean'
    );
    prefsWidget.append(showIndicatorBox);

    // Show mode in indicator toggle
    let showModeBox = createSettingRow(
        'Show mode name in indicator',
        'indicator-show-mode',
        settings,
        'boolean'
    );
    prefsWidget.append(showModeBox);

    // Show backend count toggle
    let showBackendBox = createSettingRow(
        'Show backend count in indicator',
        'indicator-show-backend-count',
        settings,
        'boolean'
    );
    prefsWidget.append(showBackendBox);

    // Separator
    prefsWidget.append(new Gtk.Separator({ orientation: Gtk.Orientation.HORIZONTAL }));

    // Notification Settings Section
    let notifLabel = new Gtk.Label({
        label: '<b>Notifications</b>',
        use_markup: true,
        halign: Gtk.Align.START,
        margin_top: 10
    });
    prefsWidget.append(notifLabel);

    // Notify on mode change toggle
    let notifyModeBox = createSettingRow(
        'Notify when mode changes',
        'notify-on-mode-change',
        settings,
        'boolean'
    );
    prefsWidget.append(notifyModeBox);

    // Notify on backend failure toggle
    let notifyBackendBox = createSettingRow(
        'Notify when backend fails',
        'notify-on-backend-failure',
        settings,
        'boolean'
    );
    prefsWidget.append(notifyBackendBox);

    // Notify on thermal throttle toggle
    let notifyThermalBox = createSettingRow(
        'Notify when thermal throttling',
        'notify-on-thermal-throttle',
        settings,
        'boolean'
    );
    prefsWidget.append(notifyThermalBox);

    // Separator
    prefsWidget.append(new Gtk.Separator({ orientation: Gtk.Orientation.HORIZONTAL }));

    // Mode Settings Section
    let modeLabel = new Gtk.Label({
        label: '<b>Mode Settings</b>',
        use_markup: true,
        halign: Gtk.Align.START,
        margin_top: 10
    });
    prefsWidget.append(modeLabel);

    // Remember last mode toggle
    let rememberModeBox = createSettingRow(
        'Remember last used mode',
        'remember-last-mode',
        settings,
        'boolean'
    );
    prefsWidget.append(rememberModeBox);

    // Default mode dropdown
    let defaultModeBox = createModeDropdown(
        'Default mode on startup',
        'default-mode',
        settings
    );
    prefsWidget.append(defaultModeBox);

    // Separator
    prefsWidget.append(new Gtk.Separator({ orientation: Gtk.Orientation.HORIZONTAL }));

    // Auto Mode Settings Section
    let autoLabel = new Gtk.Label({
        label: '<b>Auto Mode</b>',
        use_markup: true,
        halign: Gtk.Align.START,
        margin_top: 10
    });
    prefsWidget.append(autoLabel);

    // Enable quiet hours toggle
    let quietHoursBox = createSettingRow(
        'Enable quiet hours',
        'auto-quiet-hours-enabled',
        settings,
        'boolean'
    );
    prefsWidget.append(quietHoursBox);

    // Quiet hours start
    let quietStartBox = createSpinButton(
        'Quiet hours start (hour)',
        'auto-quiet-hours-start',
        settings,
        0, 23, 1
    );
    prefsWidget.append(quietStartBox);

    // Quiet hours end
    let quietEndBox = createSpinButton(
        'Quiet hours end (hour)',
        'auto-quiet-hours-end',
        settings,
        0, 23, 1
    );
    prefsWidget.append(quietEndBox);

    // Show and return
    prefsWidget.show();
    return prefsWidget;
}

function createSettingRow(label, key, settings, type) {
    let box = new Gtk.Box({
        orientation: Gtk.Orientation.HORIZONTAL,
        spacing: 10,
        margin_top: 5
    });

    let labelWidget = new Gtk.Label({
        label: label,
        halign: Gtk.Align.START,
        hexpand: true
    });
    box.append(labelWidget);

    if (type === 'boolean') {
        let toggle = new Gtk.Switch({
            active: settings.get_boolean(key),
            halign: Gtk.Align.END
        });

        toggle.connect('notify::active', (widget) => {
            settings.set_boolean(key, widget.active);
        });

        box.append(toggle);
    }

    return box;
}

function createModeDropdown(label, key, settings) {
    let box = new Gtk.Box({
        orientation: Gtk.Orientation.HORIZONTAL,
        spacing: 10,
        margin_top: 5
    });

    let labelWidget = new Gtk.Label({
        label: label,
        halign: Gtk.Align.START,
        hexpand: true
    });
    box.append(labelWidget);

    let modes = ['Performance', 'Balanced', 'Efficiency', 'Quiet', 'Auto', 'UltraEfficiency'];
    let dropdown = new Gtk.ComboBoxText();

    modes.forEach(mode => {
        dropdown.append_text(mode);
    });

    let currentMode = settings.get_string(key);
    let currentIndex = modes.indexOf(currentMode);
    if (currentIndex >= 0) {
        dropdown.set_active(currentIndex);
    }

    dropdown.connect('changed', (widget) => {
        let index = widget.get_active();
        if (index >= 0) {
            settings.set_string(key, modes[index]);
        }
    });

    box.append(dropdown);
    return box;
}

function createSpinButton(label, key, settings, min, max, step) {
    let box = new Gtk.Box({
        orientation: Gtk.Orientation.HORIZONTAL,
        spacing: 10,
        margin_top: 5
    });

    let labelWidget = new Gtk.Label({
        label: label,
        halign: Gtk.Align.START,
        hexpand: true
    });
    box.append(labelWidget);

    let spinButton = new Gtk.SpinButton({
        adjustment: new Gtk.Adjustment({
            lower: min,
            upper: max,
            step_increment: step
        }),
        value: settings.get_int(key),
        halign: Gtk.Align.END
    });

    spinButton.connect('value-changed', (widget) => {
        settings.set_int(key, widget.get_value());
    });

    box.append(spinButton);
    return box;
}
