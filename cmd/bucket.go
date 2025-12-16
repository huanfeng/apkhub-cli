package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/client"
	"github.com/spf13/cobra"
)

var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: i18n.T("cmd.bucket.short"),
	Long:  i18n.T("cmd.bucket.long"),
}

var bucketVerifySignature bool

var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("cmd.bucket.list.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
		}

		if len(config.Buckets) == 0 {
			fmt.Println(i18n.T("cmd.bucket.list.empty"))
			return nil
		}

		// Display buckets in table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, i18n.T("cmd.bucket.list.header"))
		fmt.Fprintln(w, "----\t------------\t---\t-------\t------------")

		for name, bucket := range config.Buckets {
			enabled := "Yes"
			if !bucket.Enabled {
				enabled = "No"
			}

			lastUpdated := "Never"
			if !bucket.LastUpdated.IsZero() {
				lastUpdated = bucket.LastUpdated.Format("2006-01-02 15:04")
			}

			marker := ""
			if name == config.DefaultBucket {
				marker = "*"
			}

			fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n",
				name, marker, bucket.Name, bucket.URL, enabled, lastUpdated)
		}

		w.Flush()

		if config.DefaultBucket != "" {
			fmt.Printf("\n%s\n", i18n.T("cmd.bucket.list.default", map[string]interface{}{
				"name": config.DefaultBucket,
			}))
		}

		return nil
	},
}

var bucketAddCmd = &cobra.Command{
	Use:   "add <name> <url-or-path> [display-name]",
	Short: i18n.T("cmd.bucket.add.short"),
	Long:  i18n.T("cmd.bucket.add.long"),
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		urlOrPath := args[1]
		displayName := name
		if len(args) > 2 {
			displayName = args[2]
		}

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
		}

		// Detect and validate the source type
		bucketURL, bucketType, err := validateAndNormalizeBucketSource(urlOrPath)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errInvalidSource"), err)
		}

		fmt.Printf("%s\n", i18n.T("cmd.bucket.add.start", map[string]interface{}{
			"type": bucketType, "name": name,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.bucket.add.source", map[string]interface{}{
			"source": bucketURL,
		}))

		// Add bucket
		if err := config.AddBucket(name, bucketURL, displayName); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errAdd"), err)
		}

		fmt.Printf("%s\n", i18n.T("cmd.bucket.add.success", map[string]interface{}{
			"name": name, "source": bucketURL,
		}))

		// Update bucket index
		fmt.Printf("%s\n", i18n.T("cmd.bucket.add.fetch"))
		bucketMgr := client.NewBucketManager(config)
		bucketMgr.SetSignatureVerification(bucketVerifySignature && config.Security.VerifySignature)
		if _, err := bucketMgr.FetchManifest(name); err != nil {
			fmt.Printf("%s\n", i18n.T("cmd.bucket.add.fetchFail", map[string]interface{}{
				"error": err,
			}))
			fmt.Println(i18n.T("cmd.bucket.add.fetchRetry"))
		} else {
			fmt.Println(i18n.T("cmd.bucket.add.fetchSuccess"))
		}

		return nil
	},
}

var bucketRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: i18n.T("cmd.bucket.remove.short"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
		}

		// Confirm removal
		if !skipConfirm {
			fmt.Printf("%s", i18n.T("cmd.bucket.remove.confirm", map[string]interface{}{
				"name": name,
			}))
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println(i18n.T("cmd.bucket.remove.cancel"))
				return nil
			}
		}

		// Remove bucket
		if err := config.RemoveBucket(name); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errRemove"), err)
		}

		fmt.Printf("%s\n", i18n.T("cmd.bucket.remove.success", map[string]interface{}{
			"name": name,
		}))

		// Clean up cache
		cacheFile := fmt.Sprintf("%s/%s.json", config.Client.CacheDir, name)
		os.Remove(cacheFile)

		return nil
	},
}

var bucketUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: i18n.T("cmd.bucket.update.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
		}

		bucketMgr := client.NewBucketManager(config)
		bucketMgr.SetSignatureVerification(bucketVerifySignature && config.Security.VerifySignature)

		if len(args) > 0 {
			// Update specific bucket
			name := args[0]
			fmt.Printf("%s\n", i18n.T("cmd.bucket.update.single", map[string]interface{}{
				"name": name,
			}))
			if _, err := bucketMgr.FetchManifest(name); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errUpdate"), err)
			}
			fmt.Printf("%s\n", i18n.T("cmd.bucket.update.singleSuccess", map[string]interface{}{
				"name": name,
			}))
		} else {
			// Update all buckets
			if err := bucketMgr.UpdateAll(); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println(i18n.T("cmd.bucket.update.allSuccess"))
		}

		return nil
	},
}

var bucketEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: i18n.T("cmd.bucket.enable.short"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setBucketEnabled(args[0], true)
	},
}

var bucketDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: i18n.T("cmd.bucket.disable.short"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setBucketEnabled(args[0], false)
	},
}

var bucketHealthCmd = &cobra.Command{
	Use:   "health [name]",
	Short: i18n.T("cmd.bucket.health.short"),
	Long:  i18n.T("cmd.bucket.health.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
		}

		bucketMgr := client.NewBucketManager(config)

		if len(args) > 0 {
			// Check specific bucket
			name := args[0]
			fmt.Printf("%s\n", i18n.T("cmd.bucket.health.checking", map[string]interface{}{
				"name": name,
			}))

			health, err := bucketMgr.CheckBucketHealth(name)
			if err != nil {
				fmt.Printf("❌ Health check failed: %v\n", err)
				fmt.Printf("%s\n", i18n.T("cmd.bucket.health.checkFail", map[string]interface{}{
					"error": err,
				}))
			}

			if health != nil {
				printSingleBucketHealth(health)
			}
		} else {
			// Check all buckets
			fmt.Println(i18n.T("cmd.bucket.health.checkAll"))
			bucketMgr.CheckAllHealth()
			bucketMgr.PrintHealthStatus()

			// Print summary
			summary := bucketMgr.GetHealthSummary()
			fmt.Printf("\n%s\n", i18n.T("cmd.bucket.health.summaryTitle"))
			fmt.Printf(i18n.T("cmd.bucket.health.summaryHealthy")+"\n", summary["healthy"])
			fmt.Printf(i18n.T("cmd.bucket.health.summaryDegraded")+"\n", summary["degraded"])
			fmt.Printf(i18n.T("cmd.bucket.health.summaryUnhealthy")+"\n", summary["unhealthy"])
			fmt.Printf(i18n.T("cmd.bucket.health.summaryUnknown")+"\n", summary["unknown"])
		}

		return nil
	},
}

var bucketStatusCmd = &cobra.Command{
	Use:   "status",
	Short: i18n.T("cmd.bucket.status.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
		}

		bucketMgr := client.NewBucketManager(config)

		fmt.Println(i18n.T("cmd.bucket.status.title"))
		fmt.Println("======================")

		// Show configuration summary
		totalBuckets := len(config.Buckets)
		enabledBuckets := len(config.GetEnabledBuckets())

		fmt.Printf("\n%s\n", i18n.T("cmd.bucket.status.configTitle"))
		fmt.Printf(i18n.T("cmd.bucket.status.total")+"\n", totalBuckets)
		fmt.Printf(i18n.T("cmd.bucket.status.enabled")+"\n", enabledBuckets)
		fmt.Printf(i18n.T("cmd.bucket.status.default")+"\n", config.DefaultBucket)
		fmt.Printf(i18n.T("cmd.bucket.status.cacheDir")+"\n", config.Client.CacheDir)
		fmt.Printf(i18n.T("cmd.bucket.status.cacheTTL")+"\n", config.Client.CacheTTL)

		// Check health of all buckets
		fmt.Printf("\n%s\n", i18n.T("cmd.bucket.status.healthChecks"))
		bucketMgr.CheckAllHealth()
		bucketMgr.PrintHealthStatus()

		// Show cache status
		fmt.Printf("\n%s\n", i18n.T("cmd.bucket.status.cacheTitle"))
		showCacheStatus(config)

		return nil
	},
}

func setBucketEnabled(name string, enabled bool) error {
	// Load client config
	config, err := client.Load()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.bucket.errLoadConfig"), err)
	}

	bucket, exists := config.Buckets[name]
	if !exists {
		return fmt.Errorf(i18n.T("cmd.bucket.errNotFound", map[string]interface{}{
			"name": name,
		}))
	}

	bucket.Enabled = enabled
	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	status := "enabled"
	status = i18n.T("cmd.bucket.setStatus.enabled")
	if !enabled {
		status = i18n.T("cmd.bucket.setStatus.disabled")
	}
	fmt.Printf("%s\n", i18n.T("cmd.bucket.setStatus.success", map[string]interface{}{
		"name":   name,
		"status": status,
	}))

	return nil
}

// printSingleBucketHealth prints detailed health info for a single bucket
func printSingleBucketHealth(health *client.BucketHealth) {
	fmt.Printf("\n%s\n", i18n.T("cmd.bucket.health.reportTitle", map[string]interface{}{
		"name": health.Name,
	}))
	fmt.Printf(i18n.T("cmd.bucket.health.url")+"\n", health.URL)

	statusIcon := "❓"
	switch health.Status {
	case "healthy":
		statusIcon = "✅"
	case "degraded":
		statusIcon = "⚠️ "
	case "unhealthy":
		statusIcon = "❌"
	}

	fmt.Printf(i18n.T("cmd.bucket.health.status")+"\n", statusIcon, health.Status)
	fmt.Printf(i18n.T("cmd.bucket.health.lastCheck")+"\n", health.LastCheck.Format("2006-01-02 15:04:05"))

	if !health.LastSuccess.IsZero() {
		fmt.Printf(i18n.T("cmd.bucket.health.lastSuccess")+"\n", health.LastSuccess.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("%s\n", i18n.T("cmd.bucket.health.never"))
	}

	if health.ResponseTime > 0 {
		fmt.Printf(i18n.T("cmd.bucket.health.responseTime")+"\n", health.ResponseTime)
	}

	fmt.Printf(i18n.T("cmd.bucket.health.errorCount")+"\n", health.ErrorCount)
	fmt.Printf(i18n.T("cmd.bucket.health.consecutiveFails")+"\n", health.ConsecutiveFails)

	if health.LastError != "" {
		fmt.Printf(i18n.T("cmd.bucket.health.lastError")+"\n", health.LastError)
	}
}

// showCacheStatus shows cache file information
func showCacheStatus(config *client.Config) {
	cacheDir := config.Client.CacheDir

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Printf(i18n.T("cmd.bucket.cache.missing")+"\n", cacheDir)
		return
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		fmt.Printf(i18n.T("cmd.bucket.cache.readErr")+"\n", err)
		return
	}

	var totalSize int64
	var cacheFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".json") {
			cacheFiles = append(cacheFiles, entry.Name())

			if info, err := entry.Info(); err == nil {
				totalSize += info.Size()
			}
		}
	}

	fmt.Printf(i18n.T("cmd.bucket.cache.fileCount")+"\n", len(cacheFiles))
	fmt.Printf(i18n.T("cmd.bucket.cache.totalSize")+"\n", formatBytes(totalSize))

	if len(cacheFiles) > 0 {
		fmt.Printf(i18n.T("cmd.bucket.cache.files")+"\n", strings.Join(cacheFiles, ", "))
	}
}

// validateAndNormalizeBucketSource validates and normalizes bucket source
func validateAndNormalizeBucketSource(source string) (string, string, error) {
	// Check if it's a URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return source, "remote", validateRemoteURL(source)
	}

	// Check if it's a local path
	if isLocalPath(source) {
		normalizedPath, err := validateLocalPath(source)
		if err != nil {
			return "", "", err
		}

		// Convert to file:// URL for consistency
		fileURL := "file://" + normalizedPath
		return fileURL, "local", nil
	}

	return "", "", fmt.Errorf("source must be either a HTTP/HTTPS URL or a local directory path")
}

// isLocalPath checks if the source looks like a local path
func isLocalPath(source string) bool {
	// Absolute paths
	if strings.HasPrefix(source, "/") || strings.HasPrefix(source, "\\") {
		return true
	}

	// Windows drive letters
	if len(source) >= 3 && source[1] == ':' && (source[2] == '\\' || source[2] == '/') {
		return true
	}

	// Relative paths
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		return true
	}

	// Current directory reference
	if source == "." {
		return true
	}

	// Check if it exists as a directory
	if info, err := os.Stat(source); err == nil && info.IsDir() {
		return true
	}

	return false
}

// validateRemoteURL validates a remote URL
func validateRemoteURL(url string) error {
	// Basic URL validation is already done by the HTTP prefix check
	// Here we could add more sophisticated validation if needed
	return nil
}

// validateLocalPath validates and normalizes a local path
func validateLocalPath(path string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory does not exist: %s", absPath)
		}
		return "", fmt.Errorf("failed to access directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Check if it looks like an APK repository
	if err := validateRepositoryStructure(absPath); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
		fmt.Println("   The directory will be added anyway, but it may not work as expected.")
	}

	return absPath, nil
}

// validateRepositoryStructure checks if a directory looks like an APK repository
func validateRepositoryStructure(path string) error {
	// Check for manifest file
	manifestPath := filepath.Join(path, "apkhub_manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("no apkhub_manifest.json found in directory")
	}

	// Check for APKs directory
	apksPath := filepath.Join(path, "apks")
	if info, err := os.Stat(apksPath); err != nil || !info.IsDir() {
		return fmt.Errorf("no 'apks' directory found")
	}

	return nil
}

// formatBytes formats bytes in human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(bucketCmd)

	bucketCmd.PersistentFlags().BoolVar(&bucketVerifySignature, "verify-signature", true, "Verify bucket manifest signatures (set false to bypass)")

	// Add subcommands
	bucketCmd.AddCommand(bucketListCmd)
	bucketCmd.AddCommand(bucketAddCmd)
	bucketCmd.AddCommand(bucketRemoveCmd)
	bucketCmd.AddCommand(bucketUpdateCmd)
	bucketCmd.AddCommand(bucketEnableCmd)
	bucketCmd.AddCommand(bucketDisableCmd)
	bucketCmd.AddCommand(bucketHealthCmd)
	bucketCmd.AddCommand(bucketStatusCmd)

	// Add flags
	bucketRemoveCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
}
