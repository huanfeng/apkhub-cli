package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/huanfeng/apkhub-cli/pkg/system"
	"github.com/spf13/cobra"
)

var (
	installDevice    string
	installReplace   bool
	installDowngrade bool
	installGrant     bool
	installLocalPath string
	installCheckDeps bool
)

var installCmd = &cobra.Command{
	Use:   "install <package-id|apk-path>",
	Short: "Install an application using adb",
	Long: `Install an application to a connected Android device using adb.
You can specify either a package ID to download and install, or a local APK path.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Check dependencies first
		if err := checkInstallDependencies(); err != nil {
			return err
		}

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		// Determine if target is a file or package ID
		var apkPath string

		if installLocalPath != "" {
			// Explicit local path provided
			apkPath = installLocalPath
		} else if _, err := os.Stat(target); err == nil {
			// Target is a file
			apkPath = target
		} else if filepath.Ext(target) == ".apk" {
			// Looks like a filename but doesn't exist
			return fmt.Errorf("APK file not found: %s", target)
		} else {
			// Target is a package ID, download it first
			fmt.Printf("Package ID detected: %s\n", target)

			bucketMgr := client.NewBucketManager(config)
			downloadMgr := client.NewDownloadManager(config, bucketMgr)

			downloadOptions := client.DownloadOptions{
				Version: downloadVersion, // Reuse flag from download command
			}

			fmt.Println("Downloading APK...")
			apkPath, err = downloadMgr.Download(target, downloadOptions)
			if err != nil {
				return fmt.Errorf("failed to download APK: %w", err)
			}
		}

		// Verify APK exists
		if _, err := os.Stat(apkPath); err != nil {
			return fmt.Errorf("APK file not found: %s", apkPath)
		}

		// Create ADB manager
		adbMgr := client.NewADBManager(config)

		// Select device
		deviceID := installDevice
		if deviceID == "" {
			deviceID, err = adbMgr.SelectDevice()
			if err != nil {
				return fmt.Errorf("failed to select device: %w", err)
			}
		}

		fmt.Printf("Installing to device: %s\n", deviceID)
		fmt.Printf("APK: %s\n", apkPath)

		// Install options
		options := client.InstallOptions{
			Replace:          installReplace,
			Downgrade:        installDowngrade,
			GrantPermissions: installGrant,
		}

		// Perform installation
		if err := adbMgr.Install(apkPath, deviceID, options); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		fmt.Println("✓ Installation successful!")

		// Get package ID from path if needed
		if target != apkPath {
			// Already have package ID
			fmt.Printf("\nTo uninstall: apkhub uninstall %s\n", target)
		}

		return nil
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

func init() {
	rootCmd.AddCommand(installCmd)

	// Add flags
	installCmd.Flags().StringVarP(&installDevice, "device", "s", "", "Target device ID")
	installCmd.Flags().BoolVarP(&installReplace, "replace", "r", true, "Replace existing application")
	installCmd.Flags().BoolVarP(&installDowngrade, "downgrade", "d", false, "Allow version downgrade")
	installCmd.Flags().BoolVarP(&installGrant, "grant", "g", true, "Grant all runtime permissions")
	installCmd.Flags().StringVarP(&downloadVersion, "version", "v", "", "Install specific version (when using package ID)")
	installCmd.Flags().StringVarP(&installLocalPath, "local", "l", "", "Force treating argument as local path")
	installCmd.Flags().BoolVar(&installCheckDeps, "check-deps", false, "Check dependencies before installation")
}
