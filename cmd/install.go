package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/huanfeng/apkhub/internal/device"
	"github.com/huanfeng/apkhub/internal/errors"
	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/apk"
	"github.com/huanfeng/apkhub/pkg/client"
	"github.com/huanfeng/apkhub/pkg/system"
	"github.com/huanfeng/apkhub/pkg/utils"
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
	Use:          "install <package-id|apk-path>",
	Short:        i18n.T("cmd.install.short"),
	Long:         i18n.T("cmd.install.long"),
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // Don't show usage on errors
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.GetGlobalLogger()
		logger.Info(i18n.T("cmd.install.log.start"), args[0])

		target := args[0]

		// Check dependencies first
		if err := checkInstallDependencies(); err != nil {
			return errors.WrapError(err, errors.ErrorTypeDependency, "INSTALL_DEPS_MISSING",
				i18n.T("cmd.install.errDepsMissing")).
				WithContext("target", target).
				WithSuggestion(i18n.T("cmd.install.suggestDoctorFix"))
		}

		// Load client config
		config, err := client.Load()
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeConfiguration, "CONFIG_LOAD_FAILED",
				i18n.T("cmd.install.errLoadConfig")).
				WithContext("config_file", cfgFile).
				WithSuggestion(i18n.T("cmd.install.suggestInit"))
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return errors.WrapError(err, errors.ErrorTypeFileSystem, "DIR_CREATE_FAILED",
				i18n.T("cmd.install.errEnsureDirs")).
				WithSuggestion(i18n.T("cmd.install.suggestPermissions"))
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
			fmt.Printf("%s\n", i18n.T("cmd.install.packageDetected", map[string]interface{}{"id": target}))
			fmt.Printf("%s\n", i18n.T("cmd.install.detectPackage", map[string]interface{}{"id": target}))

			bucketMgr := client.NewBucketManager(config)
			downloadMgr := client.NewDownloadManager(config, bucketMgr)

			downloadOptions := client.DownloadOptions{
				Version: downloadVersion, // Reuse flag from download command
			}

			logger.Info(i18n.T("cmd.install.log.downloading", map[string]interface{}{"id": target}))
			fmt.Println(i18n.T("cmd.install.downloading"))
			apkPath, err = downloadMgr.Download(target, downloadOptions)
			if err != nil {
				return errors.WrapError(err, errors.ErrorTypeNetwork, "DOWNLOAD_FAILED",
					i18n.T("cmd.install.errDownload")).
					WithContext("package_id", target).
					WithSuggestions([]string{
						i18n.T("cmd.install.suggestNetwork"),
						i18n.T("cmd.install.suggestPackageID"),
						i18n.T("cmd.install.suggestRetry"),
					})
			}
			isLocalFile = false
		}

		// Enhanced local file validation and info extraction
		if isLocalFile {
			logger.Debug("Validating local APK file: %s", apkPath)
			if err := validateAndShowLocalAPKInfo(apkPath); err != nil {
				return errors.WrapError(err, errors.ErrorTypeValidation, "APK_VALIDATION_FAILED",
					i18n.T("cmd.install.errValidateLocal")).
					WithContext("apk_path", apkPath).
					WithSuggestions([]string{
						i18n.T("cmd.install.suggestCorrupt"),
						i18n.T("cmd.install.suggestFilePerm"),
						i18n.T("cmd.install.suggestTryAnother"),
					})
			}
		}

		// Verify APK exists
		if _, err := os.Stat(apkPath); err != nil {
			return errors.WrapError(err, errors.ErrorTypeNotFound, "APK_FILE_NOT_FOUND",
				i18n.T("cmd.install.errAPKMissing")).
				WithContext("apk_path", apkPath).
				WithSuggestion(i18n.T("cmd.install.suggestCheckPath"))
		}

		if installAllDevices && (installDevice != "" || len(installDeviceIDs) > 0) {
			return fmt.Errorf(i18n.T("cmd.install.errAllDevicesConflict"))
		}

		// Perform unified installation process
		return performUnifiedInstall(config, apkPath, target, isLocalFile)
	},
}

// checkInstallDependencies checks if all required dependencies for install are available
func checkInstallDependencies() error {
	if installCheckDeps {
		fmt.Println(i18n.T("cmd.install.checkDeps.start"))
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
		fmt.Printf("%s\n", i18n.T("cmd.install.checkDeps.missing", map[string]interface{}{
			"list": strings.Join(missingRequired, ", "),
		}))
		fmt.Println()
		fmt.Println(i18n.T("cmd.install.checkDeps.hintTitle"))
		fmt.Println(i18n.T("cmd.install.checkDeps.hintDoctor"))
		fmt.Println(i18n.T("cmd.install.checkDeps.hintDoctorFix"))
		fmt.Printf("%s\n", i18n.T("cmd.install.checkDeps.hintDepsCmd"))
		return fmt.Errorf(i18n.T("cmd.install.checkDeps.err"))
	}

	if installCheckDeps {
		fmt.Println(i18n.T("cmd.install.checkDeps.ok"))
		for _, dep := range deps {
			if dep.Available {
				fmt.Printf("%s\n", i18n.T("cmd.install.checkDeps.okItem", map[string]interface{}{
					"name":    dep.Name,
					"version": dep.Version,
				}))
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
			return fmt.Errorf(i18n.T("cmd.install.errLocalMissing", map[string]interface{}{
				"path": apkPath,
			}))
		}
		return fmt.Errorf(i18n.T("cmd.install.errLocalAccess", map[string]interface{}{
			"error": err,
		}))
	}

	if info.IsDir() {
		return fmt.Errorf(i18n.T("cmd.install.errLocalIsDir", map[string]interface{}{
			"path": apkPath,
		}))
	}

	// Check file size
	if info.Size() == 0 {
		return fmt.Errorf(i18n.T("cmd.install.errLocalEmpty", map[string]interface{}{
			"path": apkPath,
		}))
	}

	if info.Size() < 1024 {
		return fmt.Errorf(i18n.T("cmd.install.errLocalTooSmall", map[string]interface{}{
			"path": apkPath,
		}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.install.local.detected"))
	fmt.Printf("%s\n", i18n.T("cmd.install.local.path", map[string]interface{}{
		"path": apkPath,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.local.size", map[string]interface{}{
		"size": float64(info.Size()) / (1024 * 1024),
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.local.modified", map[string]interface{}{
		"time": info.ModTime().Format("2006-01-02 15:04:05"),
	}))

	// For XAPK/APKM files, show basic info only
	if isXAPKFile(apkPath) {
		fmt.Printf("%s\n", i18n.T("cmd.install.local.typeXapk"))
		fmt.Printf("%s\n", i18n.T("cmd.install.local.xapkNote"))
	} else {
		// Try to extract basic APK information using the parser
		if err := showLocalAPKDetails(apkPath); err != nil {
			fmt.Printf("%s\n", i18n.T("cmd.install.local.detailsError", map[string]interface{}{
				"error": err,
			}))
			fmt.Printf("%s\n", i18n.T("cmd.install.local.installAsIs"))
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

	fmt.Printf("%s\n", i18n.T("cmd.install.local.detailsTitle"))
	fmt.Printf("%s\n", i18n.T("cmd.install.local.package", map[string]interface{}{
		"id": apkInfo.PackageID,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.local.version", map[string]interface{}{
		"version": apkInfo.Version,
		"code":    apkInfo.VersionCode,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.local.sdk", map[string]interface{}{
		"min":    apkInfo.MinSDK,
		"target": apkInfo.TargetSDK,
	}))

	if appName := getDefaultName(apkInfo.AppName); appName != "" {
		fmt.Printf("%s\n", i18n.T("cmd.install.local.appName", map[string]interface{}{
			"name": appName,
		}))
	}

	if len(apkInfo.Permissions) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.install.local.permissions", map[string]interface{}{
			"count": len(apkInfo.Permissions),
		}))
		if len(apkInfo.Permissions) <= 5 {
			for _, perm := range apkInfo.Permissions {
				fmt.Printf("%s\n", i18n.T("cmd.install.local.permissionItem", map[string]interface{}{
					"permission": perm,
				}))
			}
		} else {
			for i, perm := range apkInfo.Permissions[:3] {
				fmt.Printf("%s\n", i18n.T("cmd.install.local.permissionItem", map[string]interface{}{
					"permission": perm,
				}))
				if i == 2 {
					fmt.Printf("%s\n", i18n.T("cmd.install.local.morePermissions", map[string]interface{}{
						"count": len(apkInfo.Permissions) - 3,
					}))
					break
				}
			}
		}
	}

	return nil
}

// performUnifiedInstall handles the unified installation process for both local and remote APKs
func performUnifiedInstall(config *client.Config, apkPath, target string, isLocalFile bool) error {
	fmt.Println(i18n.T("cmd.install.unifiedStart"))

	// Initialize ADB manager
	adbMgr := client.NewADBManager(config)

	explicitTargets := append([]string{}, installDeviceIDs...)
	if installDevice != "" {
		explicitTargets = append(explicitTargets, installDevice)
	}

	deviceIDs, err := resolveTargetDevices(adbMgr, explicitTargets, installAllDevices, true)
	if err != nil {
		return fmt.Errorf(i18n.T("cmd.install.errDeviceSelection", map[string]interface{}{
			"error": err,
		}))
	}

	if err := performMultiDeviceInstall(adbMgr, deviceIDs, apkPath, target, isLocalFile); err != nil {
		return err
	}

	return nil
}

func performMultiDeviceInstall(adbMgr *client.ADBManager, deviceIDs []string, apkPath, target string, isLocalFile bool) error {
	fmt.Printf("%s\n", i18n.T("cmd.install.targetDevices", map[string]interface{}{
		"count": len(deviceIDs),
	}))

	options := []device.Option[*client.InstallResult]{}
	if installWorkers > 0 {
		options = append(options, device.WithWorkerLimit[*client.InstallResult](installWorkers))
	}

	manager := device.NewManager[*client.InstallResult](options...)
	results := manager.Run(context.Background(), deviceIDs, func(ctx context.Context, deviceID string) (*client.InstallResult, error) {
		fmt.Printf("\n%s\n", i18n.T("cmd.install.prepareDevice", map[string]interface{}{
			"id": deviceID,
		}))
		if err := validateSpecifiedDevice(adbMgr, deviceID); err != nil {
			return nil, err
		}

		if err := performPreInstallChecks(adbMgr, apkPath, deviceID); err != nil {
			return nil, fmt.Errorf(i18n.T("cmd.install.errPreChecks", map[string]interface{}{
				"error": err,
			}))
		}

		installOptions := client.InstallOptions{
			Replace:          installReplace,
			Downgrade:        installDowngrade,
			GrantPermissions: installGrant,
		}

		fmt.Printf("%s\n", i18n.T("cmd.install.installing", map[string]interface{}{
			"id": deviceID,
		}))
		result, err := adbMgr.InstallWithResult(apkPath, deviceID, installOptions)
		if err != nil {
			return nil, err
		}

		displayInstallationResult(result, target, isLocalFile)

		if result.Success {
			if err := performPostInstallVerification(adbMgr, result, deviceID); err != nil {
				fmt.Printf("%s\n", i18n.T("cmd.install.verifyWarning", map[string]interface{}{
					"id":    deviceID,
					"error": err,
				}))
				fmt.Println(i18n.T("cmd.install.verifyNote"))
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
				message = i18n.T("cmd.install.summary.defaultFailure")
			}
			failures = append(failures, fmt.Sprintf("%s: %s", res.DeviceID, message))
			continue
		}

		failures = append(failures, fmt.Sprintf("%s: %s", res.DeviceID, i18n.T("cmd.install.summary.unknown")))
	}

	fmt.Println("\n" + i18n.T("cmd.install.summary.title"))
	if len(successes) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.install.summary.success", map[string]interface{}{
			"count": len(successes),
		}))
		for _, s := range successes {
			fmt.Printf("%s\n", i18n.T("cmd.install.summary.item", map[string]interface{}{
				"entry": s,
			}))
		}
	} else {
		fmt.Println(i18n.T("cmd.install.summary.none"))
	}

	if len(failures) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.install.summary.failure", map[string]interface{}{
			"count": len(failures),
		}))
		for _, f := range failures {
			fmt.Printf("%s\n", i18n.T("cmd.install.summary.item", map[string]interface{}{
				"entry": f,
			}))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf(i18n.T("cmd.install.errInstallFailed", map[string]interface{}{
			"count": len(failures),
		}))
	}

	return nil
}

// validateSpecifiedDevice validates that the specified device is available and online
func validateSpecifiedDevice(adbMgr *client.ADBManager, deviceID string) error {
	fmt.Printf("%s\n", i18n.T("cmd.install.validateDevice", map[string]interface{}{
		"id": deviceID,
	}))

	device, err := adbMgr.GetDeviceInfo(deviceID)
	if err != nil {
		return fmt.Errorf(i18n.T("cmd.install.errDeviceNotFound", map[string]interface{}{
			"error": err,
		}))
	}

	if device.Status != "device" {
		return fmt.Errorf(i18n.T("cmd.install.errDeviceOffline", map[string]interface{}{
			"id":     deviceID,
			"status": device.Status,
		}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.install.deviceValidated", map[string]interface{}{
		"info": formatDeviceInfo(*device),
	}))
	return nil
}

// performPreInstallChecks performs various checks before installation
func performPreInstallChecks(adbMgr *client.ADBManager, apkPath, deviceID string) error {
	fmt.Println(i18n.T("cmd.install.preChecks.start"))

	// Check APK file integrity
	if err := validateAPKIntegrity(apkPath); err != nil {
		return fmt.Errorf(i18n.T("cmd.install.preChecks.integrity", map[string]interface{}{
			"error": err,
		}))
	}

	// Check device storage space (if possible)
	if err := checkDeviceStorage(adbMgr, apkPath, deviceID); err != nil {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.storageWarn", map[string]interface{}{
			"error": err,
		}))
		// Don't fail on storage check, just warn
	}

	// Check for existing installation
	if err := checkExistingInstallation(adbMgr, apkPath, deviceID); err != nil {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.existingInfo", map[string]interface{}{
			"error": err,
		}))
		// Don't fail, just inform
	}

	fmt.Println(i18n.T("cmd.install.preChecks.done"))
	return nil
}

// validateAPKIntegrity performs basic APK file integrity checks
func validateAPKIntegrity(apkPath string) error {
	// Check if file is readable
	file, err := os.Open(apkPath)
	if err != nil {
		return fmt.Errorf(i18n.T("cmd.install.errOpenAPK", map[string]interface{}{
			"error": err,
		}))
	}
	defer file.Close()

	// Read first few bytes to check if it's a valid ZIP file (APK is a ZIP)
	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf(i18n.T("cmd.install.errReadHeader", map[string]interface{}{
			"error": err,
		}))
	}

	// Check ZIP signature (PK\x03\x04)
	if header[0] != 0x50 || header[1] != 0x4B || header[2] != 0x03 || header[3] != 0x04 {
		return fmt.Errorf(i18n.T("cmd.install.errInvalidAPK"))
	}

	return nil
}

// checkDeviceStorage checks if device has enough storage space
func checkDeviceStorage(adbMgr *client.ADBManager, apkPath, deviceID string) error {
	// Get APK file size
	info, err := os.Stat(apkPath)
	if err != nil {
		return fmt.Errorf(i18n.T("cmd.install.errAPKSize", map[string]interface{}{
			"error": err,
		}))
	}

	apkSizeMB := float64(info.Size()) / (1024 * 1024)

	// For now, just report the APK size
	// In a full implementation, we would check device storage via ADB
	fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.size", map[string]interface{}{
		"size": apkSizeMB,
	}))

	if apkSizeMB > 100 {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.largeAPK", map[string]interface{}{
			"size": apkSizeMB,
		}))
	}

	return nil
}

// checkExistingInstallation checks if the app is already installed
func checkExistingInstallation(adbMgr *client.ADBManager, apkPath, deviceID string) error {
	// For XAPK files, skip existing installation check to avoid duplicate parsing
	if isXAPKFile(apkPath) {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.xapkSkip"))
		return nil
	}

	// Try to extract package ID from APK
	parser := apk.NewParser(".")
	apkInfo, err := parser.ParseAPK(apkPath)
	if err != nil {
		return fmt.Errorf(i18n.T("cmd.install.preChecks.packageInfo", map[string]interface{}{
			"error": err,
		}))
	}

	// Check if package is already installed
	versionName, versionCode, err := adbMgr.GetInstalledVersion(apkInfo.PackageID, deviceID)
	if err != nil {
		// Package not installed, which is fine
		return nil
	}

	fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.installed", map[string]interface{}{
		"id": apkInfo.PackageID,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.installedVersion", map[string]interface{}{
		"version": versionName,
		"code":    versionCode,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.apkVersion", map[string]interface{}{
		"version": apkInfo.Version,
		"code":    apkInfo.VersionCode,
	}))

	if apkInfo.VersionCode < versionCode {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.downgrade"))
	} else if apkInfo.VersionCode > versionCode {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.upgrade"))
	} else {
		fmt.Printf("%s\n", i18n.T("cmd.install.preChecks.reinstall"))
	}

	return nil
}

// displayInstallationResult displays the installation result with detailed information
func displayInstallationResult(result *client.InstallResult, target string, isLocalFile bool) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println(i18n.T("cmd.install.result.title"))
	fmt.Println(strings.Repeat("=", 60))

	if isLocalFile {
		fmt.Printf("%s\n", i18n.T("cmd.install.result.sourceLocal", map[string]interface{}{
			"target": target,
		}))
	} else {
		fmt.Printf("%s\n", i18n.T("cmd.install.result.sourceRemote", map[string]interface{}{
			"target": target,
		}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.install.result.device", map[string]interface{}{
		"id": result.DeviceID,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.result.duration", map[string]interface{}{
		"duration": result.Duration,
	}))

	if result.Success {
		fmt.Printf("%s\n", i18n.T("cmd.install.result.statusSuccess"))
		if result.PackageID != "" {
			fmt.Printf("%s\n", i18n.T("cmd.install.result.package", map[string]interface{}{
				"id": result.PackageID,
			}))
		}
	} else {
		fmt.Printf("%s\n", i18n.T("cmd.install.result.statusFailed"))
		if result.ErrorCode != "" {
			fmt.Printf("%s\n", i18n.T("cmd.install.result.errorCode", map[string]interface{}{
				"code": result.ErrorCode,
			}))
		}
		if result.ErrorMessage != "" {
			fmt.Printf("%s\n", i18n.T("cmd.install.result.errorMsg", map[string]interface{}{
				"error": result.ErrorMessage,
			}))
		}

		if len(result.Suggestions) > 0 {
			fmt.Println("\n" + i18n.T("cmd.install.result.suggestions"))
			for _, suggestion := range result.Suggestions {
				fmt.Printf("%s\n", i18n.T("cmd.install.result.suggestionItem", map[string]interface{}{
					"suggestion": suggestion,
				}))
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

// performPostInstallVerification verifies the installation was successful
func performPostInstallVerification(adbMgr *client.ADBManager, result *client.InstallResult, deviceID string) error {
	if result.PackageID == "" {
		return fmt.Errorf(i18n.T("cmd.install.errNoPackageID"))
	}

	fmt.Printf("%s\n", i18n.T("cmd.install.verify.start", map[string]interface{}{
		"id": result.PackageID,
	}))

	// Check if package is now installed
	versionName, versionCode, err := adbMgr.GetInstalledVersion(result.PackageID, deviceID)
	if err != nil {
		return fmt.Errorf(i18n.T("cmd.install.verify.notFound", map[string]interface{}{
			"error": err,
		}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.install.verify.success"))
	fmt.Printf("%s\n", i18n.T("cmd.install.verify.package", map[string]interface{}{
		"id": result.PackageID,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.install.verify.version", map[string]interface{}{
		"version": versionName,
		"code":    versionCode,
	}))

	return nil
}

// cleanupDownloadedFile removes the downloaded APK file
func cleanupDownloadedFile(apkPath string) error {
	fmt.Printf("%s\n", i18n.T("cmd.install.cleanup", map[string]interface{}{
		"path": apkPath,
	}))
	return os.Remove(apkPath)
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Add flags
	installCmd.Flags().StringVarP(&installDevice, "device", "s", "", i18n.T("cmd.install.flag.device"))
	installCmd.Flags().StringSliceVar(&installDeviceIDs, "devices", nil, i18n.T("cmd.install.flag.devices"))
	installCmd.Flags().BoolVar(&installAllDevices, "all-devices", false, i18n.T("cmd.install.flag.allDevices"))
	installCmd.Flags().IntVar(&installWorkers, "workers", 0, i18n.T("cmd.install.flag.workers"))
	installCmd.Flags().BoolVarP(&installReplace, "replace", "r", true, i18n.T("cmd.install.flag.replace"))
	installCmd.Flags().BoolVarP(&installDowngrade, "downgrade", "d", false, i18n.T("cmd.install.flag.downgrade"))
	installCmd.Flags().BoolVarP(&installGrant, "grant", "g", true, i18n.T("cmd.install.flag.grant"))
	installCmd.Flags().StringVarP(&downloadVersion, "version", "v", "", i18n.T("cmd.install.flag.version"))
	installCmd.Flags().StringVarP(&installLocalPath, "local", "l", "", i18n.T("cmd.install.flag.local"))
	installCmd.Flags().BoolVar(&installCheckDeps, "check-deps", false, i18n.T("cmd.install.flag.checkDeps"))
}

// isXAPKFile checks if the file is an XAPK or APKM file
func isXAPKFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".xapk" || ext == ".apkm"
}
