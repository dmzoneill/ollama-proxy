package main

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

const (
	dbusInterface = "com.anthropic.OllamaProxy.Efficiency"
	dbusPath      = "/com/anthropic/OllamaProxy/Efficiency"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to D-Bus: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	obj := conn.Object(dbusInterface, dbusPath)

	switch command {
	case "get":
		getMode(obj)
	case "set":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ai-efficiency set <mode>")
			os.Exit(1)
		}
		setMode(obj, os.Args[2])
	case "list":
		listModes(obj)
	case "info":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ai-efficiency info <mode>")
			os.Exit(1)
		}
		getModeInfo(obj, os.Args[2])
	case "status":
		showStatus(obj)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("AI Efficiency Mode Control")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ai-efficiency get               Get current mode")
	fmt.Println("  ai-efficiency set <mode>        Set efficiency mode")
	fmt.Println("  ai-efficiency list              List available modes")
	fmt.Println("  ai-efficiency info <mode>       Get mode information")
	fmt.Println("  ai-efficiency status            Show current status")
	fmt.Println()
	fmt.Println("Available modes:")
	fmt.Println("  Performance      - Maximum speed (NVIDIA GPU)")
	fmt.Println("  Balanced         - Smart routing (recommended)")
	fmt.Println("  Efficiency       - Low power (NPU/Intel GPU)")
	fmt.Println("  Quiet            - Minimal fan noise")
	fmt.Println("  Auto             - Automatic based on battery/thermal")
	fmt.Println("  UltraEfficiency  - Maximum battery life (NPU only)")
}

func getMode(obj dbus.BusObject) {
	var mode string
	err := obj.Call(dbusInterface+".GetMode", 0).Store(&mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get mode: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(mode)
}

func setMode(obj dbus.BusObject, mode string) {
	// Validate mode name
	validModes := map[string]bool{
		"Performance":     true,
		"Balanced":        true,
		"Efficiency":      true,
		"Quiet":           true,
		"Auto":            true,
		"UltraEfficiency": true,
	}

	if !validModes[mode] {
		fmt.Fprintf(os.Stderr, "Invalid mode: %s\n", mode)
		fmt.Fprintln(os.Stderr, "Use 'ai-efficiency list' to see available modes")
		os.Exit(1)
	}

	err := obj.Call(dbusInterface+".SetMode", 0, mode).Store()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set mode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ AI Efficiency mode set to: %s\n", mode)
}

func listModes(obj dbus.BusObject) {
	var modes []string
	err := obj.Call(dbusInterface+".ListModes", 0).Store(&modes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list modes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Available AI Efficiency Modes:")
	fmt.Println()

	for _, mode := range modes {
		var info map[string]dbus.Variant
		obj.Call(dbusInterface+".GetModeInfo", 0, mode).Store(&info)

		icon := "  "
		if iconVar, ok := info["icon"]; ok {
			icon = iconVar.Value().(string)
		}

		desc := ""
		if descVar, ok := info["description"]; ok {
			desc = descVar.Value().(string)
		}

		fmt.Printf("%s %-20s %s\n", icon, mode, desc)
	}
}

func getModeInfo(obj dbus.BusObject, mode string) {
	var info map[string]dbus.Variant
	err := obj.Call(dbusInterface+".GetModeInfo", 0, mode).Store(&info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get mode info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mode: %s\n", mode)
	if iconVar, ok := info["icon"]; ok {
		fmt.Printf("Icon: %s\n", iconVar.Value().(string))
	}
	if descVar, ok := info["description"]; ok {
		fmt.Printf("Description: %s\n", descVar.Value().(string))
	}
	if powerVar, ok := info["maxPower"]; ok {
		fmt.Printf("Max Power: %dW\n", powerVar.Value().(int))
	}
	if fanVar, ok := info["maxFan"]; ok {
		fmt.Printf("Max Fan: %d%%\n", fanVar.Value().(int))
	}
	if tempVar, ok := info["maxTemp"]; ok {
		fmt.Printf("Max Temp: %.1f°C\n", tempVar.Value().(float64))
	}
}

func showStatus(obj dbus.BusObject) {
	var currentMode, effectiveMode string

	err := obj.Call(dbusInterface+".GetMode", 0).Store(&currentMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
		os.Exit(1)
	}

	err = obj.Call(dbusInterface+".GetEffectiveMode", 0).Store(&effectiveMode)
	if err != nil {
		effectiveMode = currentMode
	}

	fmt.Println("AI Efficiency Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Current Mode:   %s\n", currentMode)

	if currentMode != effectiveMode {
		fmt.Printf("Effective Mode: %s (auto-selected)\n", effectiveMode)
	}

	// Get info about effective mode
	var info map[string]dbus.Variant
	obj.Call(dbusInterface+".GetModeInfo", 0, effectiveMode).Store(&info)

	if descVar, ok := info["description"]; ok {
		fmt.Printf("\n%s\n", descVar.Value().(string))
	}

	if powerVar, ok := info["maxPower"]; ok {
		maxPower := powerVar.Value().(int)
		if maxPower < 999 {
			fmt.Printf("\nLimits:\n")
			fmt.Printf("  Max Power: %dW\n", maxPower)
		}
	}
	if fanVar, ok := info["maxFan"]; ok {
		maxFan := fanVar.Value().(int)
		if maxFan < 100 {
			fmt.Printf("  Max Fan:   %d%%\n", maxFan)
		}
	}
}
