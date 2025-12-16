package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/huanfeng/apkhub-cli/internal/device"
	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	devicesFormat  string
	devicesWatch   bool
	devicesRefresh int
	devicesAll     bool
	devicesTargets []string

	devicesLogPackage string
	devicesLogLevel   string
	devicesLogOutput  string
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: i18n.T("cmd.devices.short"),
	Long:  i18n.T("cmd.devices.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errLoadConfig"), err)
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
	Short: i18n.T("cmd.devices.info.short"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errLoadConfig"), err)
		}

		// Create ADB manager
		adbMgr := client.NewADBManager(config)

		// Get device info
		device, err := adbMgr.GetDeviceInfo(deviceID)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errGetInfo"), err)
		}

		// Display device information
		showDeviceDetails(device)

		return nil
	},
}

var devicesWaitCmd = &cobra.Command{
	Use:   "wait <device-id>",
	Short: i18n.T("cmd.devices.wait.short"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errLoadConfig"), err)
		}

		// Create ADB manager
		adbMgr := client.NewADBManager(config)

		// Wait for device
		timeout := 60 * time.Second
		if err := adbMgr.WaitForDevice(deviceID, timeout); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errWait"), err)
		}

		return nil
	},
}

var devicesLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: i18n.T("cmd.devices.logs.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if devicesLogPackage == "" {
			return fmt.Errorf(i18n.T("cmd.devices.logs.packageRequired"))
		}

		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errLoadConfig"), err)
		}

		adbMgr := client.NewADBManager(config)
		deviceIDs, err := resolveTargetDevices(adbMgr, devicesTargets, devicesAll, true)
		if err != nil {
			return err
		}

		options := []device.Option[*client.LogCaptureResult]{}
		manager := device.NewManager[*client.LogCaptureResult](options...)
		results := manager.Run(context.Background(), deviceIDs, func(ctx context.Context, deviceID string) (*client.LogCaptureResult, error) {
			outputPath := buildLogOutputPath(deviceID, len(deviceIDs))
			return adbMgr.CaptureLogs(client.LogCaptureOptions{
				DeviceID:   deviceID,
				PackageID:  devicesLogPackage,
				Level:      devicesLogLevel,
				OutputPath: outputPath,
			})
		})

		return summarizeLogCaptures(results)
	},
}

// showDevices displays the list of devices
func showDevices(adbMgr *client.ADBManager) error {
	status, err := adbMgr.GetDeviceStatus()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errGetStatus"), err)
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
	fmt.Printf("%s\n", i18n.T("cmd.devices.default.title"))
	fmt.Printf("==================\n\n")

	if status.Total == 0 {
		fmt.Println(i18n.T("cmd.devices.default.none"))
		fmt.Println()
		fmt.Println(i18n.T("cmd.devices.default.troubleshoot"))
		fmt.Println(i18n.T("cmd.devices.default.tipUSB"))
		fmt.Println(i18n.T("cmd.devices.default.tipDebug"))
		fmt.Println(i18n.T("cmd.devices.default.tipAuth"))
		fmt.Println(i18n.T("cmd.devices.default.tipADB"))
		return nil
	}

	// Online devices
	if len(status.Online) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.devices.default.online", map[string]interface{}{"count": len(status.Online)}))
		for i, device := range status.Online {
			fmt.Printf("%d. %s\n", i+1, formatDeviceInfo(device))
		}
		fmt.Println()
	}

	// Offline devices
	if len(status.Offline) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.devices.default.offline", map[string]interface{}{"count": len(status.Offline)}))
		for i, device := range status.Offline {
			fmt.Printf("%d. %s\n", i+1, formatDeviceInfo(device))
		}
		fmt.Println(i18n.T("cmd.devices.default.tipReconnect"))
	}

	// Unauthorized devices
	if len(status.Unauthorized) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.devices.default.unauthorized", map[string]interface{}{"count": len(status.Unauthorized)}))
		for i, device := range status.Unauthorized {
			fmt.Printf("%d. %s\n", i+1, formatDeviceInfo(device))
		}
		fmt.Println(i18n.T("cmd.devices.default.tipAuthorize"))
	}

	// Summary
	fmt.Printf("%s\n", i18n.T("cmd.devices.default.summary", map[string]interface{}{
		"total": status.Total, "online": len(status.Online), "offline": len(status.Offline), "unauth": len(status.Unauthorized),
	}))

	return nil
}

// showDevicesTable displays devices in table format
func showDevicesTable(status *client.DeviceStatus) error {
	if status.Total == 0 {
		fmt.Println(i18n.T("cmd.devices.default.none"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cmd.devices.table.header"))
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
	fmt.Printf("\n%s\n", i18n.T("cmd.devices.table.total", map[string]interface{}{"count": status.Total}))
	fmt.Printf("%s\n", i18n.T("cmd.devices.table.breakdown", map[string]interface{}{
		"online": len(status.Online), "offline": len(status.Offline), "unauth": len(status.Unauthorized),
	}))
	return nil
}

// showDevicesJSON displays devices in JSON format
func showDevicesJSON(status *client.DeviceStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.devices.errMarshal"), err)
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
	fmt.Printf("%s\n", i18n.T("cmd.devices.details.title"))
	fmt.Printf("====================\n\n")

	fmt.Printf("%s\n", i18n.T("cmd.devices.details.id", map[string]interface{}{"id": device.ID}))
	fmt.Printf("%s\n", i18n.T("cmd.devices.details.status", map[string]interface{}{"status": device.Status}))

	if device.Model != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.model", map[string]interface{}{"model": device.Model}))
	}

	if device.Manufacturer != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.manufacturer", map[string]interface{}{"m": device.Manufacturer}))
	}

	if device.Brand != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.brand", map[string]interface{}{"brand": device.Brand}))
	}

	if device.Product != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.product", map[string]interface{}{"product": device.Product}))
	}

	if device.Device != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.device", map[string]interface{}{"device": device.Device}))
	}

	if device.AndroidVer != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.androidVer", map[string]interface{}{"ver": device.AndroidVer}))
	}

	if device.AndroidAPI > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.apiLevel", map[string]interface{}{"api": device.AndroidAPI}))
	}

	if device.Transport != "" {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.transport", map[string]interface{}{"transport": device.Transport}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.devices.details.type", map[string]interface{}{
		"type": map[bool]string{true: i18n.T("cmd.devices.details.typeEmu"), false: i18n.T("cmd.devices.details.typePhy")}[device.IsEmulator],
	}))

	if !device.LastSeen.IsZero() {
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.lastSeen", map[string]interface{}{"time": device.LastSeen.Format("2006-01-02 15:04:05")}))
	}

	// Status-specific information
	switch device.Status {
	case "device":
		fmt.Printf("\n%s\n", i18n.T("cmd.devices.details.online"))
	case "offline":
		fmt.Printf("\n%s\n", i18n.T("cmd.devices.details.offline"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixReconnect"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixRestart"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixADB"))
	case "unauthorized":
		fmt.Printf("\n%s\n", i18n.T("cmd.devices.details.unauthorized"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixAllowDebug"))
		fmt.Printf("%s\n", i18n.T("cmd.devices.details.fixAlwaysAllow"))
	}
}

func buildLogOutputPath(deviceID string, totalDevices int) string {
	if devicesLogOutput == "" {
		return ""
	}

	if totalDevices <= 1 {
		return devicesLogOutput
	}

	ext := filepath.Ext(devicesLogOutput)
	base := strings.TrimSuffix(filepath.Base(devicesLogOutput), ext)
	dir := filepath.Dir(devicesLogOutput)
	sanitized := strings.ReplaceAll(deviceID, ":", "_")

	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", base, sanitized, ext))
}

func summarizeLogCaptures(results []device.Result[*client.LogCaptureResult]) error {
	var successes []string
	var failures []string

	for _, res := range results {
		if res.Err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", res.DeviceID, res.Err))
			continue
		}

		if res.Value != nil {
			summary := fmt.Sprintf("%s -> %s (%d bytes)", res.DeviceID, res.Value.OutputPath, res.Value.SizeBytes)
			if res.Value.Note != "" {
				summary += fmt.Sprintf(" [%s]", res.Value.Note)
			}
			successes = append(successes, summary)
			continue
		}

		failures = append(failures, fmt.Sprintf("%s: no result", res.DeviceID))
	}

	fmt.Println("\n" + i18n.T("cmd.devices.logs.summary"))
	if len(successes) > 0 {
		fmt.Printf(i18n.T("cmd.devices.logs.captured")+"\n", len(successes))
		for _, s := range successes {
			fmt.Printf("      • %s\n", s)
		}
	}

	if len(failures) > 0 {
		fmt.Printf(i18n.T("cmd.devices.logs.failed")+"\n", len(failures))
		for _, f := range failures {
			fmt.Printf("      • %s\n", f)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf(i18n.T("cmd.devices.logs.errCapture", map[string]interface{}{
			"count": len(failures),
		}))
	}

	return nil
}

// watchDevices continuously monitors device status
func watchDevices(adbMgr *client.ADBManager) error {
	fmt.Printf("%s\n\n", i18n.T("cmd.devices.watch.start", map[string]interface{}{
		"seconds": devicesRefresh,
	}))

	for {
		// Clear screen (simple approach)
		fmt.Print("\033[2J\033[H")

		fmt.Printf("%s\n\n", i18n.T("cmd.devices.watch.updated", map[string]interface{}{
			"time": time.Now().Format("15:04:05"),
		}))

		if err := showDevices(adbMgr); err != nil {
			fmt.Printf("%s\n", i18n.T("cmd.devices.watch.err", map[string]interface{}{
				"error": err,
			}))
		}

		time.Sleep(time.Duration(devicesRefresh) * time.Second)
	}
}

func init() {
	rootCmd.AddCommand(devicesCmd)

	// Add subcommands
	devicesCmd.AddCommand(devicesInfoCmd)
	devicesCmd.AddCommand(devicesWaitCmd)
	devicesCmd.AddCommand(devicesLogsCmd)

	// Add flags
	devicesCmd.Flags().StringVar(&devicesFormat, "format", "default", i18n.T("cmd.devices.flag.format"))
	devicesCmd.Flags().BoolVar(&devicesWatch, "watch", false, i18n.T("cmd.devices.flag.watch"))
	devicesCmd.Flags().IntVar(&devicesRefresh, "refresh", 3, i18n.T("cmd.devices.flag.refresh"))
	devicesCmd.PersistentFlags().BoolVar(&devicesAll, "all-devices", false, i18n.T("cmd.devices.flag.allDevices"))
	devicesCmd.PersistentFlags().StringSliceVar(&devicesTargets, "devices", nil, i18n.T("cmd.devices.flag.devices"))

	devicesLogsCmd.Flags().StringVar(&devicesLogPackage, "package", "", i18n.T("cmd.devices.flag.logPackage"))
	devicesLogsCmd.Flags().StringVar(&devicesLogLevel, "level", "I", i18n.T("cmd.devices.flag.logLevel"))
	devicesLogsCmd.Flags().StringVar(&devicesLogOutput, "output", "", i18n.T("cmd.devices.flag.logOutput"))
}
