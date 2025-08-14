package client

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ADBManager handles ADB operations
type ADBManager struct {
	config *Config
}

// NewADBManager creates a new ADB manager
func NewADBManager(config *Config) *ADBManager {
	return &ADBManager{
		config: config,
	}
}

// Device represents an ADB device with detailed information
type Device struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	Model        string    `json:"model"`
	Product      string    `json:"product"`
	Device       string    `json:"device"`
	Transport    string    `json:"transport"`
	AndroidAPI   int       `json:"android_api"`
	AndroidVer   string    `json:"android_version"`
	Manufacturer string    `json:"manufacturer"`
	Brand        string    `json:"brand"`
	LastSeen     time.Time `json:"last_seen"`
	IsEmulator   bool      `json:"is_emulator"`
}

// DeviceStatus represents device connection status
type DeviceStatus struct {
	Online       []Device `json:"online"`
	Offline      []Device `json:"offline"`
	Unauthorized []Device `json:"unauthorized"`
	Total        int      `json:"total"`
}

// InstallResult represents the result of an installation
type InstallResult struct {
	Success      bool          `json:"success"`
	PackageID    string        `json:"package_id"`
	DeviceID     string        `json:"device_id"`
	Duration     time.Duration `json:"duration"`
	ErrorCode    string        `json:"error_code,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Suggestions  []string      `json:"suggestions,omitempty"`
}

// GetDevices returns list of connected devices with detailed information
func (a *ADBManager) GetDevices() ([]Device, error) {
	cmd := exec.Command(a.config.ADB.Path, "devices", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run adb devices: %w", err)
	}

	var devices []Device
	lines := strings.Split(string(output), "\n")

	for _, line := range lines[1:] { // Skip header
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		device := Device{
			ID:       parts[0],
			Status:   parts[1],
			LastSeen: time.Now(),
		}

		// Check if it's an emulator
		device.IsEmulator = strings.Contains(device.ID, "emulator-") ||
			strings.Contains(strings.ToLower(device.ID), "emulator")

		// Extract additional info from device list
		for _, part := range parts[2:] {
			if strings.HasPrefix(part, "model:") {
				device.Model = strings.TrimPrefix(part, "model:")
			} else if strings.HasPrefix(part, "product:") {
				device.Product = strings.TrimPrefix(part, "product:")
			} else if strings.HasPrefix(part, "device:") {
				device.Device = strings.TrimPrefix(part, "device:")
			} else if strings.HasPrefix(part, "transport_id:") {
				device.Transport = strings.TrimPrefix(part, "transport_id:")
			}
		}

		// Get additional device info if device is online
		if device.Status == "device" {
			a.enrichDeviceInfo(&device)
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// enrichDeviceInfo adds additional information to a device
func (a *ADBManager) enrichDeviceInfo(device *Device) {
	// Get Android version and API level
	if apiLevel, err := a.getDeviceProperty(device.ID, "ro.build.version.sdk"); err == nil {
		if api, parseErr := strconv.Atoi(strings.TrimSpace(apiLevel)); parseErr == nil {
			device.AndroidAPI = api
		}
	}

	if version, err := a.getDeviceProperty(device.ID, "ro.build.version.release"); err == nil {
		device.AndroidVer = strings.TrimSpace(version)
	}

	// Get manufacturer and brand
	if manufacturer, err := a.getDeviceProperty(device.ID, "ro.product.manufacturer"); err == nil {
		device.Manufacturer = strings.TrimSpace(manufacturer)
	}

	if brand, err := a.getDeviceProperty(device.ID, "ro.product.brand"); err == nil {
		device.Brand = strings.TrimSpace(brand)
	}

	// If model is empty, try to get it from properties
	if device.Model == "" {
		if model, err := a.getDeviceProperty(device.ID, "ro.product.model"); err == nil {
			device.Model = strings.TrimSpace(model)
		}
	}
}

// getDeviceProperty gets a system property from a device
func (a *ADBManager) getDeviceProperty(deviceID, property string) (string, error) {
	cmd := exec.Command(a.config.ADB.Path, "-s", deviceID, "shell", "getprop", property)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetDeviceStatus returns categorized device status
func (a *ADBManager) GetDeviceStatus() (*DeviceStatus, error) {
	devices, err := a.GetDevices()
	if err != nil {
		return nil, err
	}

	status := &DeviceStatus{
		Online:       []Device{},
		Offline:      []Device{},
		Unauthorized: []Device{},
		Total:        len(devices),
	}

	for _, device := range devices {
		switch device.Status {
		case "device":
			status.Online = append(status.Online, device)
		case "offline":
			status.Offline = append(status.Offline, device)
		case "unauthorized":
			status.Unauthorized = append(status.Unauthorized, device)
		}
	}

	return status, nil
}

// Install installs an APK to a device with enhanced error handling
func (a *ADBManager) Install(apkPath string, deviceID string, options InstallOptions) error {
	result, err := a.InstallWithResult(apkPath, deviceID, options)
	if err != nil {
		return err
	}

	if !result.Success {
		// Format error with suggestions
		errorMsg := fmt.Sprintf("installation failed: %s", result.ErrorMessage)
		if len(result.Suggestions) > 0 {
			errorMsg += "\n\n💡 Suggestions:"
			for _, suggestion := range result.Suggestions {
				errorMsg += fmt.Sprintf("\n   • %s", suggestion)
			}
		}
		return fmt.Errorf(errorMsg)
	}

	return nil
}

// InstallWithResult installs an APK and returns detailed result information
func (a *ADBManager) InstallWithResult(apkPath string, deviceID string, options InstallOptions) (*InstallResult, error) {
	startTime := time.Now()

	result := &InstallResult{
		DeviceID: deviceID,
		Duration: 0,
		Success:  false,
	}

	// Validate device is online
	if deviceID != "" {
		if err := a.validateDeviceOnline(deviceID); err != nil {
			result.ErrorMessage = err.Error()
			result.Suggestions = []string{
				"Check device connection with 'adb devices'",
				"Enable USB debugging on the device",
				"Try reconnecting the device",
			}
			return result, nil
		}
	}

	// Build command arguments
	args := []string{}

	// Add device selection if specified
	if deviceID != "" {
		args = append(args, "-s", deviceID)
	}

	args = append(args, "install")

	// Add install options
	if options.Replace {
		args = append(args, "-r")
	}
	if options.Downgrade {
		args = append(args, "-d")
	}
	if options.GrantPermissions {
		args = append(args, "-g")
	}

	args = append(args, apkPath)

	// Run adb install
	cmd := exec.Command(a.config.ADB.Path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Printf("🔧 Running: %s %s\n", a.config.ADB.Path, strings.Join(args, " "))

	err := cmd.Run()
	result.Duration = time.Since(startTime)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("command execution failed: %v", err)
		result.Suggestions = []string{
			"Check if ADB is properly installed",
			"Verify device is connected and authorized",
			"Try running 'adb kill-server && adb start-server'",
		}
		return result, nil
	}

	// Parse output
	output := stdout.String() + stderr.String()

	if strings.Contains(output, "Success") {
		result.Success = true
		return result, nil
	}

	// Parse and categorize errors
	result.ErrorCode, result.ErrorMessage, result.Suggestions = a.parseInstallError(output)

	return result, nil
}

// validateDeviceOnline checks if a device is online and accessible
func (a *ADBManager) validateDeviceOnline(deviceID string) error {
	devices, err := a.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get device list: %w", err)
	}

	for _, device := range devices {
		if device.ID == deviceID {
			switch device.Status {
			case "device":
				return nil
			case "offline":
				return fmt.Errorf("device %s is offline", deviceID)
			case "unauthorized":
				return fmt.Errorf("device %s is unauthorized - please allow USB debugging", deviceID)
			default:
				return fmt.Errorf("device %s has status: %s", deviceID, device.Status)
			}
		}
	}

	return fmt.Errorf("device %s not found", deviceID)
}

// parseInstallError parses ADB install error output and provides suggestions
func (a *ADBManager) parseInstallError(output string) (string, string, []string) {
	output = strings.ToUpper(output)

	// Define error patterns and their solutions
	errorPatterns := map[string]struct {
		code        string
		message     string
		suggestions []string
	}{
		"INSTALL_FAILED_ALREADY_EXISTS": {
			"ALREADY_EXISTS",
			"App already installed",
			[]string{
				"Use --replace flag to reinstall",
				"Uninstall the existing app first",
			},
		},
		"INSTALL_FAILED_VERSION_DOWNGRADE": {
			"VERSION_DOWNGRADE",
			"Cannot downgrade app version",
			[]string{
				"Use --downgrade flag to force downgrade",
				"Uninstall the existing app first",
				"Install a newer version instead",
			},
		},
		"INSTALL_FAILED_INSUFFICIENT_STORAGE": {
			"INSUFFICIENT_STORAGE",
			"Not enough storage space on device",
			[]string{
				"Free up storage space on the device",
				"Move apps to SD card if supported",
				"Clear app caches and data",
			},
		},
		"INSTALL_FAILED_INVALID_APK": {
			"INVALID_APK",
			"APK file is invalid or corrupted",
			[]string{
				"Re-download the APK file",
				"Verify APK file integrity",
				"Check if APK is compatible with device architecture",
			},
		},
		"INSTALL_FAILED_INCOMPATIBLE_SDK": {
			"INCOMPATIBLE_SDK",
			"APK requires higher Android version",
			[]string{
				"Update Android version on device",
				"Find a version compatible with your Android version",
			},
		},
		"INSTALL_FAILED_MISSING_SHARED_LIBRARY": {
			"MISSING_LIBRARY",
			"Required shared library not found",
			[]string{
				"Install required system libraries",
				"Check device compatibility",
			},
		},
		"INSTALL_FAILED_NO_MATCHING_ABIS": {
			"NO_MATCHING_ABIS",
			"APK architecture not compatible with device",
			[]string{
				"Download APK for correct architecture (ARM, x86, etc.)",
				"Use universal APK if available",
			},
		},
		"INSTALL_FAILED_PERMISSION_MODEL": {
			"PERMISSION_MODEL",
			"Permission model incompatibility",
			[]string{
				"Use --grant flag to grant permissions automatically",
				"Manually grant permissions after installation",
			},
		},
	}

	// Check for known error patterns
	for pattern, info := range errorPatterns {
		if strings.Contains(output, pattern) {
			return info.code, info.message, info.suggestions
		}
	}

	// Generic error handling
	if strings.Contains(output, "INSTALL_FAILED") {
		// Extract the specific error code
		re := regexp.MustCompile(`INSTALL_FAILED_([A-Z_]+)`)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			errorCode := matches[1]
			return errorCode, fmt.Sprintf("Installation failed: %s", errorCode), []string{
				"Check device logs for more details",
				"Try installing manually through device settings",
				"Verify APK compatibility with device",
			}
		}
	}

	// Unknown error
	return "UNKNOWN", fmt.Sprintf("Unknown installation error: %s", output), []string{
		"Check ADB connection",
		"Verify APK file is valid",
		"Try restarting ADB server",
		"Check device logs for more information",
	}
}

// InstallOptions contains install options
type InstallOptions struct {
	Replace          bool // Replace existing app
	Downgrade        bool // Allow version downgrade
	GrantPermissions bool // Grant all runtime permissions
}

// Uninstall uninstalls an app from device
func (a *ADBManager) Uninstall(packageID string, deviceID string) error {
	args := []string{}

	// Add device selection if specified
	if deviceID != "" {
		args = append(args, "-s", deviceID)
	}

	args = append(args, "uninstall", packageID)

	// Run adb uninstall
	cmd := exec.Command(a.config.ADB.Path, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("adb uninstall failed: %w\nOutput: %s", err, string(output))
	}

	if strings.Contains(string(output), "Success") {
		return nil
	}

	return fmt.Errorf("uninstall failed: %s", string(output))
}

// GetInstalledVersion gets the installed version of an app
func (a *ADBManager) GetInstalledVersion(packageID string, deviceID string) (string, int64, error) {
	args := []string{}

	// Add device selection if specified
	if deviceID != "" {
		args = append(args, "-s", deviceID)
	}

	args = append(args, "shell", "dumpsys", "package", packageID)

	cmd := exec.Command(a.config.ADB.Path, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get package info: %w", err)
	}

	// Parse output for version info
	var versionName string
	var versionCode int64

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "versionName=") {
			versionName = strings.TrimPrefix(line, "versionName=")
		} else if strings.HasPrefix(line, "versionCode=") {
			codeStr := strings.TrimPrefix(line, "versionCode=")
			// Extract first number (may have additional info)
			parts := strings.Fields(codeStr)
			if len(parts) > 0 {
				fmt.Sscanf(parts[0], "%d", &versionCode)
			}
		}
	}

	if versionName == "" && versionCode == 0 {
		return "", 0, fmt.Errorf("package not found or not installed")
	}

	return versionName, versionCode, nil
}

// SelectDevice prompts user to select a device with enhanced interface
func (a *ADBManager) SelectDevice() (string, error) {
	// Use default device if configured
	if a.config.ADB.DefaultDevice != "" {
		fmt.Printf("🔧 Using default device: %s\n", a.config.ADB.DefaultDevice)
		return a.config.ADB.DefaultDevice, nil
	}

	status, err := a.GetDeviceStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get device status: %w", err)
	}

	// Show device status overview
	a.printDeviceStatus(status)

	if len(status.Online) == 0 {
		return "", fmt.Errorf("no online devices available")
	}

	if len(status.Online) == 1 {
		device := status.Online[0]
		fmt.Printf("📱 Using device: %s\n", a.formatDeviceName(device))
		return device.ID, nil
	}

	// Multiple devices, show selection interface
	fmt.Println("\n📱 Multiple devices available:")
	fmt.Println("=" + strings.Repeat("=", 60))

	for i, device := range status.Online {
		fmt.Printf("%d. %s\n", i+1, a.formatDeviceDetails(device))
	}

	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Print("Select device [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	choice := 1 // default
	if trimmed := strings.TrimSpace(input); trimmed != "" {
		if parsed, parseErr := strconv.Atoi(trimmed); parseErr == nil {
			choice = parsed
		}
	}

	if choice < 1 || choice > len(status.Online) {
		choice = 1
	}

	selectedDevice := status.Online[choice-1]
	fmt.Printf("✅ Selected: %s\n", a.formatDeviceName(selectedDevice))

	return selectedDevice.ID, nil
}

// printDeviceStatus prints a formatted device status overview
func (a *ADBManager) printDeviceStatus(status *DeviceStatus) {
	fmt.Printf("📊 Device Status Overview:\n")
	fmt.Printf("   🟢 Online: %d\n", len(status.Online))

	if len(status.Offline) > 0 {
		fmt.Printf("   🔴 Offline: %d\n", len(status.Offline))
	}

	if len(status.Unauthorized) > 0 {
		fmt.Printf("   🔒 Unauthorized: %d\n", len(status.Unauthorized))
		fmt.Printf("       💡 Enable USB debugging and authorize this computer\n")
	}

	fmt.Printf("   📱 Total: %d\n", status.Total)
}

// formatDeviceName returns a user-friendly device name
func (a *ADBManager) formatDeviceName(device Device) string {
	if device.Model != "" {
		if device.IsEmulator {
			return fmt.Sprintf("%s (Emulator: %s)", device.Model, device.ID)
		}
		return fmt.Sprintf("%s (%s)", device.Model, device.ID)
	}

	if device.IsEmulator {
		return fmt.Sprintf("Emulator (%s)", device.ID)
	}

	return device.ID
}

// formatDeviceDetails returns detailed device information for selection
func (a *ADBManager) formatDeviceDetails(device Device) string {
	details := a.formatDeviceName(device)

	// Add Android version info
	if device.AndroidVer != "" {
		details += fmt.Sprintf(" - Android %s", device.AndroidVer)
		if device.AndroidAPI > 0 {
			details += fmt.Sprintf(" (API %d)", device.AndroidAPI)
		}
	}

	// Add manufacturer/brand info
	if device.Manufacturer != "" && device.Brand != "" && device.Manufacturer != device.Brand {
		details += fmt.Sprintf(" - %s %s", device.Manufacturer, device.Brand)
	} else if device.Brand != "" {
		details += fmt.Sprintf(" - %s", device.Brand)
	} else if device.Manufacturer != "" {
		details += fmt.Sprintf(" - %s", device.Manufacturer)
	}

	return details
}

// GetDeviceInfo returns detailed information about a specific device
func (a *ADBManager) GetDeviceInfo(deviceID string) (*Device, error) {
	devices, err := a.GetDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.ID == deviceID {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device %s not found", deviceID)
}

// WaitForDevice waits for a device to come online
func (a *ADBManager) WaitForDevice(deviceID string, timeout time.Duration) error {
	fmt.Printf("⏳ Waiting for device %s to come online...\n", deviceID)

	start := time.Now()
	for time.Since(start) < timeout {
		if err := a.validateDeviceOnline(deviceID); err == nil {
			fmt.Printf("✅ Device %s is now online\n", deviceID)
			return nil
		}

		time.Sleep(2 * time.Second)
		fmt.Print(".")
	}

	fmt.Println()
	return fmt.Errorf("timeout waiting for device %s to come online", deviceID)
}
