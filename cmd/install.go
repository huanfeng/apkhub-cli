package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/device"
	"github.com/huanfeng/apkhub-cli/internal/errors"
	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/huanfeng/apkhub-cli/pkg/system"
	"github.com/huanfeng/apkhub-cli/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	installDevice     string
	installDeviceIDs  []string
	installAllDevices bool
	installReplace    bool
	installDowngrade  bool
	installGrant      bool
	installLocalPath  string
	installCheckDeps  bool
	installWorkers    int
)

var installCmd = &cobra.Command{
	Use:   "install <package-id|apk-path>",
	Short: "Install an application using adb",
	Long: `Install an application to a connected Android device using adb.
You can specify either a package ID to download and install, or a local APK path.`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // Don't show usage on errors
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.GetGlobalLogger()
		logger.Info("Starting installation process for: %s", args[0])

		target := args[0]

		// Check dependencies first
		if err := checkInstallDependencies(); err != nil {
			return errors.WrapError(err, errors.ErrorTypeDependency, "INSTALL_DEPS_MISSING",
				"Required dependencies are missing for installation").
				WithContext("target", target).
				WithSuggestion("Run 'apkhub doctor --fix' to install missing dependencies")
		}

		// Load client config
		config, err := client.Load()
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeConfiguration, "CONFIG_LOAD_FAILED",
				"Failed to load configuration").
				WithContext("config_file", cfgFile).
				WithSuggestion("Run 'apkhub init' to create a new configuration file")
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return errors.WrapError(err, errors.ErrorTypeFileSystem, "DIR_CREATE_FAILED",
				"Failed to create required directories").
				WithSuggestion("Check file permissions and disk space")
		}

		// Determine if target is a file or package ID
		var apkPath string
		var isLocalFile bool

		if installLocalPath != "" {
			// Explicit local path provided
			apkPath = installLocalPath
			isLocalFile = true
		} else if isLocalAPKFile(target) {
			// Target is a local APK file
			apkPath = target
			isLocalFile = true
		} else {
			// Target is a package ID, download it first
			fmt.Printf("📦 Package ID detected: %s\n", target)

			bucketMgr := client.NewBucketManager(config)
			downloadMgr := client.NewDownloadManager(config, bucketMgr)

			downloadOptions := client.DownloadOptions{
				Version: downloadVersion, // Reuse flag from download command
			}

			logger.Info("Downloading APK for package: %s", target)
			fmt.Println("📥 Downloading APK...")
			apkPath, err = downloadMgr.Download(target, downloadOptions)
			if err != nil {
				return errors.WrapError(err, errors.ErrorTypeNetwork, "DOWNLOAD_FAILED",
					"Failed to download APK").
					WithContext("package_id", target).
					WithSuggestions([]string{
						"Check your internet connection",
						"Verify the package ID is correct",
						"Try again later",
					})
			}
			isLocalFile = false
		}

		// Enhanced local file validation and info extraction
		if isLocalFile {
			logger.Debug("Validating local APK file: %s", apkPath)
			if err := validateAndShowLocalAPKInfo(apkPath); err != nil {
				return errors.WrapError(err, errors.ErrorTypeValidation, "APK_VALIDATION_FAILED",
					"Local APK file validation failed").
					WithContext("apk_path", apkPath).
					WithSuggestions([]string{
						"Verify the APK file is not corrupted",
						"Check file permissions",
						"Try with a different APK file",
					})
			}
		}

		// Verify APK exists
		if _, err := os.Stat(apkPath); err != nil {
			return errors.WrapError(err, errors.ErrorTypeNotFound, "APK_FILE_NOT_FOUND",
				"APK file not found").
				WithContext("apk_path", apkPath).
				WithSuggestion("Verify the file path is correct")
		}

		if installAllDevices && (installDevice != "" || len(installDeviceIDs) > 0) {
			return fmt.Errorf("--all-devices cannot be combined with --device or --devices")
		}

		// Perform unified installation process
		return performUnifiedInstall(config, apkPath, target, isLocalFile)
	},
}

// checkInstallDependencies checks if all required dependencies for install are available
func checkInstallDependencies() error {
	if installCheckDeps {
		fmt.Println("🔍 Checking dependencies for install command...")
	}

	depManager := system.NewDependencyManager()
	deps := depManager.CheckForCommand("install")

	var missingRequired []string
	for _, dep := range deps {
		if dep.Required && !dep.Available {
			missingRequired = append(missingRequired, dep.Name)
		}
	}

	if len(missingRequired) > 0 {
		fmt.Printf("❌ Required dependencies missing: %s\n", strings.Join(missingRequired, ", "))
		fmt.Println("\n💡 To fix this issue:")
		fmt.Println("   1. Run 'apkhub doctor' to see installation instructions")
		fmt.Println("   2. Run 'apkhub doctor --fix' to attempt automatic installation")
		fmt.Printf("   3. Run 'apkhub deps --command install' for detailed dependency info\n")
		return fmt.Errorf("missing required dependencies")
	}

	if installCheckDeps {
		fmt.Println("✅ All required dependencies are available")
		for _, dep := range deps {
			if dep.Available {
				fmt.Printf("   ✅ %s: %s\n", dep.Name, dep.Version)
			}
		}
		fmt.Println()
	}

	return nil
}

// validateAndShowLocalAPKInfo validates a local APK file and shows basic info
func validateAndShowLocalAPKInfo(apkPath string) error {
	// Check if file exists
	info, err := os.Stat(apkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("APK file not found: %s", apkPath)
		}
		return fmt.Errorf("cannot access APK file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", apkPath)
	}

	// Check file size
	if info.Size() == 0 {
		return fmt.Errorf("APK file is empty: %s", apkPath)
	}

	if info.Size() < 1024 {
		return fmt.Errorf("APK file is too small (likely corrupted): %s", apkPath)
	}

	fmt.Printf("📱 Local APK file detected:\n")
	fmt.Printf("   Path: %s\n", apkPath)
	fmt.Printf("   Size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Printf("   Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	// For XAPK/APKM files, show basic info only
	if isXAPKFile(apkPath) {
		fmt.Printf("   📦 Type: XAPK/APKM package\n")
		fmt.Printf("   📝 Will be extracted and installed automatically\n")
	} else {
		// Try to extract basic APK information using the parser
		if err := showLocalAPKDetails(apkPath); err != nil {
			fmt.Printf("   ⚠️  Could not extract APK details: %v\n", err)
			fmt.Printf("   📝 File will be installed as-is\n")
		}
	}

	fmt.Println()
	return nil
}

// showLocalAPKDetails extracts and shows detailed APK information
func showLocalAPKDetails(apkPath string) error {
	// Use the APK parser to extract information
	parser := apk.NewParser(".")
	apkInfo, err := parser.ParseAPK(apkPath)
	if err != nil {
		return err
	}

	fmt.Printf("   📋 APK Details:\n")
	fmt.Printf("      Package: %s\n", apkInfo.PackageID)
	fmt.Printf("      Version: %s (%d)\n", apkInfo.Version, apkInfo.VersionCode)
	fmt.Printf("      Min SDK: %d, Target SDK: %d\n", apkInfo.MinSDK, apkInfo.TargetSDK)

	if appName := getDefaultName(apkInfo.AppName); appName != "" {
		fmt.Printf("      App Name: %s\n", appName)
	}

	if len(apkInfo.Permissions) > 0 {
		fmt.Printf("      Permissions: %d\n", len(apkInfo.Permissions))
		if len(apkInfo.Permissions) <= 5 {
			for _, perm := range apkInfo.Permissions {
				fmt.Printf("        - %s\n", perm)
			}
		} else {
			for i, perm := range apkInfo.Permissions[:3] {
				fmt.Printf("        - %s\n", perm)
				if i == 2 {
					fmt.Printf("        ... and %d more\n", len(apkInfo.Permissions)-3)
					break
				}
			}
		}
	}

	return nil
}

// performUnifiedInstall handles the unified installation process for both local and remote APKs
func performUnifiedInstall(config *client.Config, apkPath, target string, isLocalFile bool) error {
	fmt.Println("🚀 Starting unified installation process...")

	// Initialize ADB manager
	adbMgr := client.NewADBManager(config)

	explicitTargets := append([]string{}, installDeviceIDs...)
	if installDevice != "" {
		explicitTargets = append(explicitTargets, installDevice)
	}

	deviceIDs, err := resolveTargetDevices(adbMgr, explicitTargets, installAllDevices, true)
	if err != nil {
		return fmt.Errorf("device selection failed: %w", err)
	}

	if err := performMultiDeviceInstall(adbMgr, deviceIDs, apkPath, target, isLocalFile); err != nil {
		return err
	}

	return nil
}

func performMultiDeviceInstall(adbMgr *client.ADBManager, deviceIDs []string, apkPath, target string, isLocalFile bool) error {
	fmt.Printf("📱 Targeting %d device(s)\n", len(deviceIDs))

	options := []device.Option[*client.InstallResult]{}
	if installWorkers > 0 {
		options = append(options, device.WithWorkerLimit[*client.InstallResult](installWorkers))
	}

	manager := device.NewManager[*client.InstallResult](options...)
	results := manager.Run(context.Background(), deviceIDs, func(ctx context.Context, deviceID string) (*client.InstallResult, error) {
		fmt.Printf("\n🔧 [%s] Preparing installation\n", deviceID)
		if err := validateSpecifiedDevice(adbMgr, deviceID); err != nil {
			return nil, err
		}

		if err := performPreInstallChecks(adbMgr, apkPath, deviceID); err != nil {
			return nil, fmt.Errorf("pre-installation checks failed: %w", err)
		}

		installOptions := client.InstallOptions{
			Replace:          installReplace,
			Downgrade:        installDowngrade,
			GrantPermissions: installGrant,
		}

		fmt.Printf("📦 [%s] Installing APK...\n", deviceID)
		result, err := adbMgr.InstallWithResult(apkPath, deviceID, installOptions)
		if err != nil {
			return nil, err
		}

		displayInstallationResult(result, target, isLocalFile)

		if result.Success {
			if err := performPostInstallVerification(adbMgr, result, deviceID); err != nil {
				fmt.Printf("⚠️  [%s] Post-installation verification failed: %v\n", deviceID, err)
				fmt.Println("   The app was installed but verification encountered issues")
			}
		}

		return result, nil
	})

	return summarizeInstallResults(results)
}

func summarizeInstallResults(results []device.Result[*client.InstallResult]) error {
	var successes []string
	var failures []string

	for _, res := range results {
		if res.Err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", res.DeviceID, res.Err))
			continue
		}

		if res.Value != nil && res.Value.Success {
			summary := res.DeviceID
			if res.Value.PackageID != "" {
				summary = fmt.Sprintf("%s (%s)", res.DeviceID, res.Value.PackageID)
			}
			successes = append(successes, summary)
			continue
		}

		if res.Value != nil {
			message := res.Value.ErrorMessage
			if message == "" {
				message = "installation failed"
			}
			failures = append(failures, fmt.Sprintf("%s: %s", res.DeviceID, message))
			continue
		}

		failures = append(failures, fmt.Sprintf("%s: unknown result", res.DeviceID))
	}

	fmt.Println("\n📊 Installation summary:")
	if len(successes) > 0 {
		fmt.Printf("   ✅ %d succeeded:\n", len(successes))
		for _, s := range successes {
			fmt.Printf("      • %s\n", s)
		}
	} else {
		fmt.Println("   ✅ 0 succeeded")
	}

	if len(failures) > 0 {
		fmt.Printf("   ❌ %d failed:\n", len(failures))
		for _, f := range failures {
			fmt.Printf("      • %s\n", f)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("installation failed on %d device(s)", len(failures))
	}

	return nil
}

// validateSpecifiedDevice validates that the specified device is available and online
func validateSpecifiedDevice(adbMgr *client.ADBManager, deviceID string) error {
	fmt.Printf("🔍 Validating device: %s\n", deviceID)

	device, err := adbMgr.GetDeviceInfo(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	if device.Status != "device" {
		return fmt.Errorf("device %s is not online (status: %s)", deviceID, device.Status)
	}

	fmt.Printf("✅ Device validated: %s\n", formatDeviceInfo(*device))
	return nil
}

// performPreInstallChecks performs various checks before installation
func performPreInstallChecks(adbMgr *client.ADBManager, apkPath, deviceID string) error {
	fmt.Println("🔍 Performing pre-installation checks...")

	// Check APK file integrity
	if err := validateAPKIntegrity(apkPath); err != nil {
		return fmt.Errorf("APK integrity check failed: %w", err)
	}

	// Check device storage space (if possible)
	if err := checkDeviceStorage(adbMgr, apkPath, deviceID); err != nil {
		fmt.Printf("⚠️  Storage check warning: %v\n", err)
		// Don't fail on storage check, just warn
	}

	// Check for existing installation
	if err := checkExistingInstallation(adbMgr, apkPath, deviceID); err != nil {
		fmt.Printf("ℹ️  Existing installation info: %v\n", err)
		// Don't fail, just inform
	}

	fmt.Println("✅ Pre-installation checks completed")
	return nil
}

// validateAPKIntegrity performs basic APK file integrity checks
func validateAPKIntegrity(apkPath string) error {
	// Check if file is readable
	file, err := os.Open(apkPath)
	if err != nil {
		return fmt.Errorf("cannot open APK file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to check if it's a valid ZIP file (APK is a ZIP)
	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("cannot read APK header: %w", err)
	}

	// Check ZIP signature (PK\x03\x04)
	if header[0] != 0x50 || header[1] != 0x4B || header[2] != 0x03 || header[3] != 0x04 {
		return fmt.Errorf("invalid APK file format (not a valid ZIP file)")
	}

	return nil
}

// checkDeviceStorage checks if device has enough storage space
func checkDeviceStorage(adbMgr *client.ADBManager, apkPath, deviceID string) error {
	// Get APK file size
	info, err := os.Stat(apkPath)
	if err != nil {
		return fmt.Errorf("cannot get APK file size: %w", err)
	}

	apkSizeMB := float64(info.Size()) / (1024 * 1024)

	// For now, just report the APK size
	// In a full implementation, we would check device storage via ADB
	fmt.Printf("   APK size: %.2f MB\n", apkSizeMB)

	if apkSizeMB > 100 {
		fmt.Printf("   ⚠️  Large APK detected (%.2f MB), ensure device has sufficient storage\n", apkSizeMB)
	}

	return nil
}

// checkExistingInstallation checks if the app is already installed
func checkExistingInstallation(adbMgr *client.ADBManager, apkPath, deviceID string) error {
	// For XAPK files, skip existing installation check to avoid duplicate parsing
	if isXAPKFile(apkPath) {
		fmt.Printf("   📦 XAPK package - installation check will be performed during installation\n")
		return nil
	}

	// Try to extract package ID from APK
	parser := apk.NewParser(".")
	apkInfo, err := parser.ParseAPK(apkPath)
	if err != nil {
		return fmt.Errorf("cannot extract package info: %w", err)
	}

	// Check if package is already installed
	versionName, versionCode, err := adbMgr.GetInstalledVersion(apkInfo.PackageID, deviceID)
	if err != nil {
		// Package not installed, which is fine
		return nil
	}

	fmt.Printf("   Package %s is already installed\n", apkInfo.PackageID)
	fmt.Printf("   Installed version: %s (%d)\n", versionName, versionCode)
	fmt.Printf("   APK version: %s (%d)\n", apkInfo.Version, apkInfo.VersionCode)

	if apkInfo.VersionCode < versionCode {
		fmt.Printf("   ⚠️  Installing older version (downgrade)\n")
	} else if apkInfo.VersionCode > versionCode {
		fmt.Printf("   ⬆️  Installing newer version (upgrade)\n")
	} else {
		fmt.Printf("   🔄 Installing same version (reinstall)\n")
	}

	return nil
}

// displayInstallationResult displays the installation result with detailed information
func displayInstallationResult(result *client.InstallResult, target string, isLocalFile bool) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 INSTALLATION RESULT")
	fmt.Println(strings.Repeat("=", 60))

	if isLocalFile {
		fmt.Printf("📁 Source: Local file (%s)\n", target)
	} else {
		fmt.Printf("📦 Source: Remote package (%s)\n", target)
	}

	fmt.Printf("📱 Device: %s\n", result.DeviceID)
	fmt.Printf("⏱️  Duration: %v\n", result.Duration)

	if result.Success {
		fmt.Printf("✅ Status: SUCCESS\n")
		if result.PackageID != "" {
			fmt.Printf("📋 Package: %s\n", result.PackageID)
		}
	} else {
		fmt.Printf("❌ Status: FAILED\n")
		if result.ErrorCode != "" {
			fmt.Printf("🔍 Error Code: %s\n", result.ErrorCode)
		}
		if result.ErrorMessage != "" {
			fmt.Printf("💬 Error: %s\n", result.ErrorMessage)
		}

		if len(result.Suggestions) > 0 {
			fmt.Println("\n💡 Suggestions:")
			for _, suggestion := range result.Suggestions {
				fmt.Printf("   • %s\n", suggestion)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

// performPostInstallVerification verifies the installation was successful
func performPostInstallVerification(adbMgr *client.ADBManager, result *client.InstallResult, deviceID string) error {
	if result.PackageID == "" {
		return fmt.Errorf("no package ID available for verification")
	}

	fmt.Printf("🔍 Verifying installation of %s...\n", result.PackageID)

	// Check if package is now installed
	versionName, versionCode, err := adbMgr.GetInstalledVersion(result.PackageID, deviceID)
	if err != nil {
		return fmt.Errorf("package not found after installation: %w", err)
	}

	fmt.Printf("✅ Verification successful:\n")
	fmt.Printf("   Package: %s\n", result.PackageID)
	fmt.Printf("   Version: %s (%d)\n", versionName, versionCode)

	return nil
}

// cleanupDownloadedFile removes the downloaded APK file
func cleanupDownloadedFile(apkPath string) error {
	fmt.Printf("🧹 Cleaning up downloaded file: %s\n", apkPath)
	return os.Remove(apkPath)
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Add flags
	installCmd.Flags().StringVarP(&installDevice, "device", "s", "", "Target device ID")
	installCmd.Flags().StringSliceVar(&installDeviceIDs, "devices", nil, "Comma-separated target device IDs")
	installCmd.Flags().BoolVar(&installAllDevices, "all-devices", false, "Install on all online devices")
	installCmd.Flags().IntVar(&installWorkers, "workers", 0, "Maximum concurrent installs (0 = auto)")
	installCmd.Flags().BoolVarP(&installReplace, "replace", "r", true, "Replace existing application")
	installCmd.Flags().BoolVarP(&installDowngrade, "downgrade", "d", false, "Allow version downgrade")
	installCmd.Flags().BoolVarP(&installGrant, "grant", "g", true, "Grant all runtime permissions")
	installCmd.Flags().StringVarP(&downloadVersion, "version", "v", "", "Install specific version (when using package ID)")
	installCmd.Flags().StringVarP(&installLocalPath, "local", "l", "", "Force treating argument as local path")
	installCmd.Flags().BoolVar(&installCheckDeps, "check-deps", false, "Check dependencies before installation")
}

// isXAPKFile checks if the file is an XAPK or APKM file
func isXAPKFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".xapk" || ext == ".apkm"
}
