package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	devicesFormat  string
	devicesWatch   bool
	devicesRefresh int
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List and manage connected Android devices",
	Long:  `List connected Android devices with detailed information and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create ADB manager
		adbMgr := client.NewADBManager(config)

		if devicesWatch {
			return watchDevices(adbMgr)
		}

		return showDevices(adbMgr)
	},
}

var devicesInfoCmd = &cobra.Command{
	Use:   "info <device-id>",
	Short: "Show detailed information about a specific device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create ADB manager
		adbMgr := client.NewADBManager(config)

		// Get device info
		device, err := adbMgr.GetDeviceInfo(deviceID)
		if err != nil {
			return fmt.Errorf("failed to get device info: %w", err)
		}

		// Display device information
		showDeviceDetails(device)

		return nil
	},
}

var devicesWaitCmd = &cobra.Command{
	Use:   "wait <device-id>",
	Short: "Wait for a device to come online",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create ADB manager
		adbMgr := client.NewADBManager(config)

		// Wait for device
		timeout := 60 * time.Second
		if err := adbMgr.WaitForDevice(deviceID, timeout); err != nil {
			return fmt.Errorf("failed to wait for device: %w", err)
		}

		return nil
	},
}

// showDevices displays the list of devices
func showDevices(adbMgr *client.ADBManager) error {
	status, err := adbMgr.GetDeviceStatus()
	if err != nil {
		return fmt.Errorf("failed to get device status: %w", err)
	}

	switch devicesFormat {
	case "json":
		return showDevicesJSON(status)
	case "table":
		return showDevicesTable(status)
	default:
		return showDevicesDefault(status)
	}
}

// showDevicesDefault displays devices in default format
func showDevicesDefault(status *client.DeviceStatus) error {
	fmt.Printf("📱 Android Devices\n")
	fmt.Printf("==================\n\n")

	if status.Total == 0 {
		fmt.Println("❌ No devices found")
		fmt.Println("\n💡 Troubleshooting:")
		fmt.Println("   • Connect your Android device via USB")
		fmt.Println("   • Enable USB debugging in Developer Options")
		fmt.Println("   • Authorize this computer when prompted")
		fmt.Println("   • Try running 'adb devices' manually")
		return nil
	}

	// Online devices
	if len(status.Online) > 0 {
		fmt.Printf("🟢 Online Devices (%d):\n", len(status.Online))
		for i, device := range status.Online {
			fmt.Printf("%d. %s\n", i+1, formatDeviceInfo(device))
		}
		fmt.Println()
	}

	// Offline devices
	if len(status.Offline) > 0 {
		fmt.Printf("🔴 Offline Devices (%d):\n", len(status.Offline))
		for i, device := range status.Offline {
			fmt.Printf("%d. %s\n", i+1, formatDeviceInfo(device))
		}
		fmt.Println("   💡 Try reconnecting or restarting the device\n")
	}

	// Unauthorized devices
	if len(status.Unauthorized) > 0 {
		fmt.Printf("🔒 Unauthorized Devices (%d):\n", len(status.Unauthorized))
		for i, device := range status.Unauthorized {
			fmt.Printf("%d. %s\n", i+1, formatDeviceInfo(device))
		}
		fmt.Println("   💡 Allow USB debugging when prompted on the device\n")
	}

	// Summary
	fmt.Printf("📊 Summary: %d total, %d online, %d offline, %d unauthorized\n",
		status.Total, len(status.Online), len(status.Offline), len(status.Unauthorized))

	return nil
}

// showDevicesTable displays devices in table format
func showDevicesTable(status *client.DeviceStatus) error {
	if status.Total == 0 {
		fmt.Println("No devices found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEVICE ID\tSTATUS\tMODEL\tANDROID\tMANUFACTURER\tTYPE")
	fmt.Fprintln(w, "---------\t------\t-----\t-------\t------------\t----")

	// Combine all devices
	allDevices := append(status.Online, status.Offline...)
	allDevices = append(allDevices, status.Unauthorized...)

	for _, device := range allDevices {
		deviceType := "Device"
		if device.IsEmulator {
			deviceType = "Emulator"
		}

		androidInfo := ""
		if device.AndroidVer != "" {
			androidInfo = device.AndroidVer
			if device.AndroidAPI > 0 {
				androidInfo += fmt.Sprintf(" (API %d)", device.AndroidAPI)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			device.ID,
			device.Status,
			device.Model,
			androidInfo,
			device.Manufacturer,
			deviceType)
	}

	w.Flush()
	return nil
}

// showDevicesJSON displays devices in JSON format
func showDevicesJSON(status *client.DeviceStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// formatDeviceInfo formats device information for display
func formatDeviceInfo(device client.Device) string {
	info := device.ID

	if device.Model != "" {
		info = fmt.Sprintf("%s (%s)", device.Model, device.ID)
	}

	if device.IsEmulator {
		info += " [Emulator]"
	}

	if device.AndroidVer != "" {
		info += fmt.Sprintf(" - Android %s", device.AndroidVer)
		if device.AndroidAPI > 0 {
			info += fmt.Sprintf(" (API %d)", device.AndroidAPI)
		}
	}

	if device.Manufacturer != "" && device.Brand != "" && device.Manufacturer != device.Brand {
		info += fmt.Sprintf(" - %s %s", device.Manufacturer, device.Brand)
	} else if device.Brand != "" {
		info += fmt.Sprintf(" - %s", device.Brand)
	}

	return info
}

// showDeviceDetails displays detailed information about a single device
func showDeviceDetails(device *client.Device) {
	fmt.Printf("📱 Device Information\n")
	fmt.Printf("====================\n\n")

	fmt.Printf("Device ID: %s\n", device.ID)
	fmt.Printf("Status: %s\n", device.Status)

	if device.Model != "" {
		fmt.Printf("Model: %s\n", device.Model)
	}

	if device.Manufacturer != "" {
		fmt.Printf("Manufacturer: %s\n", device.Manufacturer)
	}

	if device.Brand != "" {
		fmt.Printf("Brand: %s\n", device.Brand)
	}

	if device.Product != "" {
		fmt.Printf("Product: %s\n", device.Product)
	}

	if device.Device != "" {
		fmt.Printf("Device: %s\n", device.Device)
	}

	if device.AndroidVer != "" {
		fmt.Printf("Android Version: %s\n", device.AndroidVer)
	}

	if device.AndroidAPI > 0 {
		fmt.Printf("API Level: %d\n", device.AndroidAPI)
	}

	if device.Transport != "" {
		fmt.Printf("Transport ID: %s\n", device.Transport)
	}

	fmt.Printf("Type: %s\n", map[bool]string{true: "Emulator", false: "Physical Device"}[device.IsEmulator])

	if !device.LastSeen.IsZero() {
		fmt.Printf("Last Seen: %s\n", device.LastSeen.Format("2006-01-02 15:04:05"))
	}

	// Status-specific information
	switch device.Status {
	case "device":
		fmt.Printf("\n✅ Device is online and ready for use\n")
	case "offline":
		fmt.Printf("\n🔴 Device is offline\n")
		fmt.Printf("💡 Try:\n")
		fmt.Printf("   • Reconnecting the USB cable\n")
		fmt.Printf("   • Restarting the device\n")
		fmt.Printf("   • Running 'adb kill-server && adb start-server'\n")
	case "unauthorized":
		fmt.Printf("\n🔒 Device is unauthorized\n")
		fmt.Printf("💡 To fix:\n")
		fmt.Printf("   • Allow USB debugging when prompted on the device\n")
		fmt.Printf("   • Check 'Always allow from this computer' if available\n")
	}
}

// watchDevices continuously monitors device status
func watchDevices(adbMgr *client.ADBManager) error {
	fmt.Printf("👀 Watching devices (refresh every %ds, press Ctrl+C to stop)...\n\n", devicesRefresh)

	for {
		// Clear screen (simple approach)
		fmt.Print("\033[2J\033[H")

		fmt.Printf("🕐 Last updated: %s\n\n", time.Now().Format("15:04:05"))

		if err := showDevices(adbMgr); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		time.Sleep(time.Duration(devicesRefresh) * time.Second)
	}
}

func init() {
	rootCmd.AddCommand(devicesCmd)

	// Add subcommands
	devicesCmd.AddCommand(devicesInfoCmd)
	devicesCmd.AddCommand(devicesWaitCmd)

	// Add flags
	devicesCmd.Flags().StringVar(&devicesFormat, "format", "default", "Output format: default, table, json")
	devicesCmd.Flags().BoolVar(&devicesWatch, "watch", false, "Watch device status continuously")
	devicesCmd.Flags().IntVar(&devicesRefresh, "refresh", 3, "Refresh interval in seconds for watch mode")
}
