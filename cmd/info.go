package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <package-id|apk-path>",
	Short: "Show detailed information about an application",
	Long: `Show detailed information about an application including all available versions.
You can specify either a package ID to get info from repositories, or a local APK file path.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Check if target is a local APK file
		if isLocalAPKFile(target) {
			return showLocalAPKInfo(target)
		}

		// Target is a package ID
		packageID := target

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		// Create managers
		bucketMgr := client.NewBucketManager(config)
		downloadMgr := client.NewDownloadManager(config, bucketMgr)

		// Get package info
		pkg, err := downloadMgr.GetPackageInfo(packageID)
		if err != nil {
			return fmt.Errorf("failed to get package info: %w", err)
		}

		// Display package information
		fmt.Printf("=== Application Information ===\n\n")
		fmt.Printf("Package ID: %s\n", pkg.PackageID)
		fmt.Printf("Name: %s\n", getDefaultName(pkg.Name))
		if desc := getDefaultName(pkg.Description); desc != "" {
			fmt.Printf("Description: %s\n", desc)
		}
		if pkg.Category != "" {
			fmt.Printf("Category: %s\n", pkg.Category)
		}

		// Latest version info
		if pkg.Latest != "" {
			if latestVer, ok := pkg.Versions[pkg.Latest]; ok {
				fmt.Printf("\nLatest Version: %s (Code: %d)\n", latestVer.Version, latestVer.VersionCode)
				fmt.Printf("Size: %.2f MB\n", float64(latestVer.Size)/(1024*1024))
				fmt.Printf("Min SDK: %d, Target SDK: %d\n", latestVer.MinSDK, latestVer.TargetSDK)
			}
		}

		// Available versions
		fmt.Printf("\n=== Available Versions ===\n\n")

		// Sort versions by version code (descending)
		type versionEntry struct {
			key     string
			version *models.AppVersion
		}
		var versions []versionEntry
		for k, v := range pkg.Versions {
			versions = append(versions, versionEntry{k, v})
		}
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].version.VersionCode > versions[j].version.VersionCode
		})

		// Display versions in table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tCODE\tSIZE\tSDK\tBUCKET\tFEATURES")
		fmt.Fprintln(w, "-------\t----\t----\t---\t------\t--------")

		for _, entry := range versions {
			ver := entry.version

			// Extract bucket name from key
			bucketName := ""
			if strings.Contains(entry.key, "_") {
				parts := strings.SplitN(entry.key, "_", 2)
				bucketName = parts[0]
			}

			// Format features
			features := ""
			if len(ver.Features) > 0 {
				features = strings.Join(ver.Features[:min(2, len(ver.Features))], ",")
				if len(ver.Features) > 2 {
					features += "..."
				}
			}

			// Display version info
			fmt.Fprintf(w, "%s\t%d\t%.1fMB\t%d-%d\t%s\t%s\n",
				ver.Version,
				ver.VersionCode,
				float64(ver.Size)/(1024*1024),
				ver.MinSDK,
				ver.TargetSDK,
				bucketName,
				features)
		}
		w.Flush()

		// Commands hint
		fmt.Println("\nCommands:")
		fmt.Printf("  Download: apkhub download %s\n", pkg.PackageID)
		fmt.Printf("  Install:  apkhub install %s\n", pkg.PackageID)
		if len(versions) > 1 {
			fmt.Printf("  Specific: apkhub download %s --version <version>\n", pkg.PackageID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

// isLocalAPKFile checks if the target looks like a local APK file
func isLocalAPKFile(target string) bool {
	// Check if it's a file path with APK extension
	if strings.HasSuffix(strings.ToLower(target), ".apk") {
		return true
	}

	// Check if it's an existing file
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		return true
	}

	// Check for other APK-related extensions
	lowerTarget := strings.ToLower(target)
	if strings.HasSuffix(lowerTarget, ".xapk") || strings.HasSuffix(lowerTarget, ".apkm") {
		return true
	}

	return false
}

// showLocalAPKInfo displays detailed information about a local APK file
func showLocalAPKInfo(apkPath string) error {
	// Validate file
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

	fmt.Printf("=== Local APK File Information ===\n\n")
	fmt.Printf("File Path: %s\n", apkPath)
	fmt.Printf("File Size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	// Try to parse APK
	fmt.Println("\n=== APK Analysis ===")

	parser := apk.NewParser(".")
	apkInfo, err := parser.ParseAPK(apkPath)
	if err != nil {
		fmt.Printf("❌ Failed to parse APK: %v\n", err)
		fmt.Println("\nThis might indicate:")
		fmt.Println("  • Corrupted APK file")
		fmt.Println("  • Unsupported APK format")
		fmt.Println("  • Missing parsing dependencies")
		fmt.Println("\n💡 Try running 'apkhub doctor' to check dependencies")
		return nil
	}

	// Display APK information
	fmt.Printf("Package ID: %s\n", apkInfo.PackageID)
	if appName := getDefaultName(apkInfo.AppName); appName != "" {
		fmt.Printf("App Name: %s\n", appName)
	}
	fmt.Printf("Version: %s (Code: %d)\n", apkInfo.Version, apkInfo.VersionCode)
	fmt.Printf("Min SDK: %d, Target SDK: %d\n", apkInfo.MinSDK, apkInfo.TargetSDK)

	if len(apkInfo.ABIs) > 0 {
		fmt.Printf("Architectures: %s\n", strings.Join(apkInfo.ABIs, ", "))
	}

	// Permissions
	if len(apkInfo.Permissions) > 0 {
		fmt.Printf("\n=== Permissions (%d) ===\n\n", len(apkInfo.Permissions))

		// Group permissions by category
		permGroups := groupPermissions(apkInfo.Permissions)

		for category, perms := range permGroups {
			fmt.Printf("%s:\n", category)
			for _, perm := range perms {
				fmt.Printf("  • %s\n", perm)
			}
			fmt.Println()
		}
	}

	// Features
	if len(apkInfo.Features) > 0 {
		fmt.Printf("=== Features (%d) ===\n\n", len(apkInfo.Features))
		for _, feature := range apkInfo.Features {
			fmt.Printf("  • %s\n", feature)
		}
		fmt.Println()
	}

	// File analysis
	fmt.Printf("=== File Analysis ===\n\n")
	fmt.Printf("SHA256: %s\n", apkInfo.SHA256)
	if apkInfo.SignatureInfo != nil && apkInfo.SignatureInfo.SHA256 != "" {
		fmt.Printf("Signature SHA256: %s\n", apkInfo.SignatureInfo.SHA256)
	}

	// Installation commands
	fmt.Printf("\n=== Installation Commands ===\n\n")
	fmt.Printf("Install: apkhub install \"%s\"\n", apkPath)
	fmt.Printf("Install with options: apkhub install \"%s\" --device <device-id>\n", apkPath)

	return nil
}

// groupPermissions groups permissions by category for better display
func groupPermissions(permissions []string) map[string][]string {
	groups := make(map[string][]string)

	for _, perm := range permissions {
		category := categorizePermission(perm)
		groups[category] = append(groups[category], perm)
	}

	return groups
}

// categorizePermission categorizes a permission for display
func categorizePermission(permission string) string {
	perm := strings.ToLower(permission)

	if strings.Contains(perm, "camera") {
		return "📷 Camera"
	} else if strings.Contains(perm, "location") || strings.Contains(perm, "gps") {
		return "📍 Location"
	} else if strings.Contains(perm, "storage") || strings.Contains(perm, "external") {
		return "💾 Storage"
	} else if strings.Contains(perm, "network") || strings.Contains(perm, "internet") {
		return "🌐 Network"
	} else if strings.Contains(perm, "phone") || strings.Contains(perm, "call") || strings.Contains(perm, "sms") {
		return "📞 Phone & SMS"
	} else if strings.Contains(perm, "contact") {
		return "👥 Contacts"
	} else if strings.Contains(perm, "microphone") || strings.Contains(perm, "record") {
		return "🎤 Microphone"
	} else if strings.Contains(perm, "calendar") {
		return "📅 Calendar"
	} else if strings.Contains(perm, "bluetooth") {
		return "📶 Bluetooth"
	} else if strings.Contains(perm, "notification") {
		return "🔔 Notifications"
	} else {
		return "🔧 System"
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
