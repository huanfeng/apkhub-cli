package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/apk"
	"github.com/huanfeng/apkhub/pkg/client"
	"github.com/huanfeng/apkhub/pkg/models"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <package-id|apk-path>",
	Short: i18n.T("cmd.info.short"),
	Long:  i18n.T("cmd.info.long"),
	Args:  cobra.ExactArgs(1),
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
			return fmt.Errorf("%s: %w", i18n.T("cmd.info.errLoadConfig"), err)
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.info.errCreateDir"), err)
		}

		// Create managers
		bucketMgr := client.NewBucketManager(config)
		downloadMgr := client.NewDownloadManager(config, bucketMgr)

		// Get package info
		pkg, err := downloadMgr.GetPackageInfo(packageID)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.info.errGetInfo"), err)
		}

		// Display package information
		fmt.Printf("%s\n\n", i18n.T("cmd.info.title"))
		fmt.Printf("%s\n", i18n.T("cmd.info.packageID", map[string]interface{}{"id": pkg.PackageID}))
		fmt.Printf("%s\n", i18n.T("cmd.info.name", map[string]interface{}{"name": getDefaultName(pkg.Name)}))
		if desc := getDefaultName(pkg.Description); desc != "" {
			fmt.Printf("%s\n", i18n.T("cmd.info.description", map[string]interface{}{"desc": desc}))
		}
		if pkg.Category != "" {
			fmt.Printf("%s\n", i18n.T("cmd.info.category", map[string]interface{}{"category": pkg.Category}))
		}

		// Latest version info
		if pkg.Latest != "" {
			if latestVer, ok := pkg.Versions[pkg.Latest]; ok {
				fmt.Printf("\n%s\n", i18n.T("cmd.info.latest", map[string]interface{}{
					"version": latestVer.Version, "code": latestVer.VersionCode,
				}))
				fmt.Printf("%s\n", i18n.T("cmd.info.sizeMB", map[string]interface{}{
					"size": fmt.Sprintf("%.2f", float64(latestVer.Size)/(1024*1024)),
				}))
				fmt.Printf("%s\n", i18n.T("cmd.info.sdk", map[string]interface{}{
					"min": latestVer.MinSDK, "target": latestVer.TargetSDK,
				}))
			}
		}

		// Available versions
		fmt.Printf("\n%s\n\n", i18n.T("cmd.info.availableVersions"))

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
		fmt.Fprintln(w, i18n.T("cmd.info.table.header"))
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
		fmt.Println("\n" + i18n.T("cmd.info.hints.title"))
		fmt.Printf("%s\n", i18n.T("cmd.info.hints.download", map[string]interface{}{"id": pkg.PackageID}))
		fmt.Printf("%s\n", i18n.T("cmd.info.hints.install", map[string]interface{}{"id": pkg.PackageID}))
		if len(versions) > 1 {
			fmt.Printf("%s\n", i18n.T("cmd.info.hints.specific", map[string]interface{}{"id": pkg.PackageID}))
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
			return fmt.Errorf(i18n.T("cmd.info.errAPKNotFound", map[string]interface{}{
				"path": apkPath,
			}))
		}
		return fmt.Errorf("%s: %w", i18n.T("cmd.info.errAccessAPK"), err)
	}

	if info.IsDir() {
		return fmt.Errorf(i18n.T("cmd.info.errPathIsDir", map[string]interface{}{
			"path": apkPath,
		}))
	}

	fmt.Printf("%s\n\n", i18n.T("cmd.info.local.title"))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.path", map[string]interface{}{"path": apkPath}))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.sizeMB", map[string]interface{}{
		"size": fmt.Sprintf("%.2f", float64(info.Size())/(1024*1024)),
	}))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.modified", map[string]interface{}{
		"time": info.ModTime().Format("2006-01-02 15:04:05"),
	}))

	// Try to parse APK
	fmt.Println("\n" + i18n.T("cmd.info.local.analysis"))

	parser := apk.NewParser(".")
	apkInfo, err := parser.ParseAPK(apkPath)
	if err != nil {
		fmt.Printf("%s\n", i18n.T("cmd.info.local.parseFail", map[string]interface{}{"error": err}))
		fmt.Println()
		fmt.Println(i18n.T("cmd.info.local.parseHintsTitle"))
		fmt.Println(i18n.T("cmd.info.local.parseHintCorrupt"))
		fmt.Println(i18n.T("cmd.info.local.parseHintUnsupported"))
		fmt.Println(i18n.T("cmd.info.local.parseHintDeps"))
		fmt.Println()
		fmt.Println(i18n.T("cmd.info.local.parseDoctor"))
		return nil
	}

	// Display APK information
	fmt.Printf("%s\n", i18n.T("cmd.info.local.packageID", map[string]interface{}{
		"id": apkInfo.PackageID,
	}))
	if appName := getDefaultName(apkInfo.AppName); appName != "" {
		fmt.Printf("%s\n", i18n.T("cmd.info.local.appName", map[string]interface{}{
			"name": appName,
		}))
	}
	fmt.Printf("%s\n", i18n.T("cmd.info.local.version", map[string]interface{}{
		"version": apkInfo.Version, "code": apkInfo.VersionCode,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.sdk", map[string]interface{}{
		"min": apkInfo.MinSDK, "target": apkInfo.TargetSDK,
	}))

	if len(apkInfo.ABIs) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.info.local.abis", map[string]interface{}{
			"abis": strings.Join(apkInfo.ABIs, ", "),
		}))
	}

	// Permissions
	if len(apkInfo.Permissions) > 0 {
		fmt.Printf("\n%s\n\n", i18n.T("cmd.info.local.permissionsTitle", map[string]interface{}{
			"count": len(apkInfo.Permissions),
		}))

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
		fmt.Printf("%s\n\n", i18n.T("cmd.info.local.featuresTitle", map[string]interface{}{
			"count": len(apkInfo.Features),
		}))
		for _, feature := range apkInfo.Features {
			fmt.Printf("  • %s\n", feature)
		}
		fmt.Println()
	}

	// File analysis
	fmt.Printf("%s\n\n", i18n.T("cmd.info.local.fileAnalysis"))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.sha256", map[string]interface{}{
		"sha": apkInfo.SHA256,
	}))
	if apkInfo.SignatureInfo != nil && apkInfo.SignatureInfo.SHA256 != "" {
		fmt.Printf("%s\n", i18n.T("cmd.info.local.signatureSHA", map[string]interface{}{
			"sha": apkInfo.SignatureInfo.SHA256,
		}))
	}

	// Installation commands
	fmt.Printf("\n%s\n\n", i18n.T("cmd.info.local.installTitle"))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.installCmd", map[string]interface{}{
		"path": apkPath,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.info.local.installCmdDevice", map[string]interface{}{
		"path": apkPath,
	}))

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
		return i18n.T("cmd.info.perm.camera")
	} else if strings.Contains(perm, "location") || strings.Contains(perm, "gps") {
		return i18n.T("cmd.info.perm.location")
	} else if strings.Contains(perm, "storage") || strings.Contains(perm, "external") {
		return i18n.T("cmd.info.perm.storage")
	} else if strings.Contains(perm, "network") || strings.Contains(perm, "internet") {
		return i18n.T("cmd.info.perm.network")
	} else if strings.Contains(perm, "phone") || strings.Contains(perm, "call") || strings.Contains(perm, "sms") {
		return i18n.T("cmd.info.perm.phone")
	} else if strings.Contains(perm, "contact") {
		return i18n.T("cmd.info.perm.contacts")
	} else if strings.Contains(perm, "microphone") || strings.Contains(perm, "record") {
		return i18n.T("cmd.info.perm.microphone")
	} else if strings.Contains(perm, "calendar") {
		return i18n.T("cmd.info.perm.calendar")
	} else if strings.Contains(perm, "bluetooth") {
		return i18n.T("cmd.info.perm.bluetooth")
	} else if strings.Contains(perm, "notification") {
		return i18n.T("cmd.info.perm.notifications")
	} else {
		return i18n.T("cmd.info.perm.system")
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
