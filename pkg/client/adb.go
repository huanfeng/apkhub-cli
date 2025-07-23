package client

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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

// Device represents an ADB device
type Device struct {
	ID     string
	Status string
	Model  string
}

// GetDevices returns list of connected devices
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
			ID:     parts[0],
			Status: parts[1],
		}
		
		// Extract model if available
		for _, part := range parts[2:] {
			if strings.HasPrefix(part, "model:") {
				device.Model = strings.TrimPrefix(part, "model:")
				break
			}
		}
		
		devices = append(devices, device)
	}
	
	return devices, nil
}

// Install installs an APK to a device
func (a *ADBManager) Install(apkPath string, deviceID string, options InstallOptions) error {
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
	
	fmt.Printf("Running: %s %s\n", a.config.ADB.Path, strings.Join(args, " "))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("adb install failed: %w\nOutput: %s\nError: %s", 
			err, stdout.String(), stderr.String())
	}
	
	output := stdout.String()
	if strings.Contains(output, "Success") {
		return nil
	}
	
	// Parse common errors
	if strings.Contains(output, "INSTALL_FAILED_ALREADY_EXISTS") {
		return fmt.Errorf("app already installed. Use --replace to reinstall")
	}
	if strings.Contains(output, "INSTALL_FAILED_VERSION_DOWNGRADE") {
		return fmt.Errorf("cannot downgrade app. Use --downgrade to force")
	}
	if strings.Contains(output, "INSTALL_FAILED_INSUFFICIENT_STORAGE") {
		return fmt.Errorf("insufficient storage on device")
	}
	
	return fmt.Errorf("installation failed: %s", output)
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

// SelectDevice prompts user to select a device if multiple are connected
func (a *ADBManager) SelectDevice() (string, error) {
	// Use default device if configured
	if a.config.ADB.DefaultDevice != "" {
		return a.config.ADB.DefaultDevice, nil
	}
	
	devices, err := a.GetDevices()
	if err != nil {
		return "", err
	}
	
	// Filter only online devices
	var onlineDevices []Device
	for _, device := range devices {
		if device.Status == "device" {
			onlineDevices = append(onlineDevices, device)
		}
	}
	
	if len(onlineDevices) == 0 {
		return "", fmt.Errorf("no devices connected")
	}
	
	if len(onlineDevices) == 1 {
		return onlineDevices[0].ID, nil
	}
	
	// Multiple devices, prompt user
	fmt.Println("Multiple devices connected:")
	for i, device := range onlineDevices {
		name := device.ID
		if device.Model != "" {
			name = fmt.Sprintf("%s (%s)", device.Model, device.ID)
		}
		fmt.Printf("%d. %s\n", i+1, name)
	}
	
	fmt.Print("Select device [1]: ")
	var choice int
	fmt.Scanln(&choice)
	
	if choice < 1 || choice > len(onlineDevices) {
		choice = 1
	}
	
	return onlineDevices[choice-1].ID, nil
}