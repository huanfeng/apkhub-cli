package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Manage APK repository buckets",
	Long:  `Manage APK repository buckets (sources) for the client functionality.`,
}

var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured buckets",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(config.Buckets) == 0 {
			fmt.Println("No buckets configured. Use 'apkhub bucket add' to add one.")
			return nil
		}

		// Display buckets in table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDISPLAY NAME\tURL\tENABLED\tLAST UPDATED")
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
			fmt.Printf("\n* Default bucket: %s\n", config.DefaultBucket)
		}

		return nil
	},
}

var bucketAddCmd = &cobra.Command{
	Use:   "add <name> <url-or-path> [display-name]",
	Short: "Add a new bucket",
	Long: `Add a new bucket source. The source can be either:
  - HTTP/HTTPS URL: https://example.com/repo
  - Local directory path: /path/to/local/repo
  - Relative path: ./local-repo`,
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
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Detect and validate the source type
		bucketURL, bucketType, err := validateAndNormalizeBucketSource(urlOrPath)
		if err != nil {
			return fmt.Errorf("invalid bucket source: %w", err)
		}

		fmt.Printf("📦 Adding %s bucket '%s'...\n", bucketType, name)
		fmt.Printf("   Source: %s\n", bucketURL)

		// Add bucket
		if err := config.AddBucket(name, bucketURL, displayName); err != nil {
			return fmt.Errorf("failed to add bucket: %w", err)
		}

		fmt.Printf("✓ Added bucket '%s' (%s)\n", name, bucketURL)

		// Update bucket index
		fmt.Printf("Fetching bucket manifest...\n")
		bucketMgr := client.NewBucketManager(config)
		if _, err := bucketMgr.FetchManifest(name); err != nil {
			fmt.Printf("Warning: Failed to fetch manifest: %v\n", err)
			fmt.Println("You can try updating later with 'apkhub bucket update'")
		} else {
			fmt.Println("✓ Bucket manifest fetched successfully")
		}

		return nil
	},
}

var bucketRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Confirm removal
		if !skipConfirm {
			fmt.Printf("Remove bucket '%s'? [y/N]: ", name)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Remove bucket
		if err := config.RemoveBucket(name); err != nil {
			return fmt.Errorf("failed to remove bucket: %w", err)
		}

		fmt.Printf("✓ Removed bucket '%s'\n", name)

		// Clean up cache
		cacheFile := fmt.Sprintf("%s/%s.json", config.Client.CacheDir, name)
		os.Remove(cacheFile)

		return nil
	},
}

var bucketUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update bucket manifests",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		bucketMgr := client.NewBucketManager(config)

		if len(args) > 0 {
			// Update specific bucket
			name := args[0]
			fmt.Printf("Updating bucket '%s'...\n", name)
			if _, err := bucketMgr.FetchManifest(name); err != nil {
				return fmt.Errorf("failed to update bucket: %w", err)
			}
			fmt.Printf("✓ Updated bucket '%s'\n", name)
		} else {
			// Update all buckets
			if err := bucketMgr.UpdateAll(); err != nil {
				return err
			}
			fmt.Println("\n✓ All buckets updated")
		}

		return nil
	},
}

var bucketEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setBucketEnabled(args[0], true)
	},
}

var bucketDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setBucketEnabled(args[0], false)
	},
}

var bucketHealthCmd = &cobra.Command{
	Use:   "health [name]",
	Short: "Check bucket health status",
	Long:  `Check the health status of buckets. If no name is provided, checks all enabled buckets.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		bucketMgr := client.NewBucketManager(config)

		if len(args) > 0 {
			// Check specific bucket
			name := args[0]
			fmt.Printf("🔍 Checking health of bucket '%s'...\n", name)
			
			health, err := bucketMgr.CheckBucketHealth(name)
			if err != nil {
				fmt.Printf("❌ Health check failed: %v\n", err)
			}
			
			if health != nil {
				printSingleBucketHealth(health)
			}
		} else {
			// Check all buckets
			fmt.Println("🔍 Checking health of all enabled buckets...")
			bucketMgr.CheckAllHealth()
			bucketMgr.PrintHealthStatus()
			
			// Print summary
			summary := bucketMgr.GetHealthSummary()
			fmt.Printf("\n📊 Health Summary:\n")
			fmt.Printf("   ✅ Healthy: %d\n", summary["healthy"])
			fmt.Printf("   ⚠️  Degraded: %d\n", summary["degraded"])
			fmt.Printf("   ❌ Unhealthy: %d\n", summary["unhealthy"])
			fmt.Printf("   ❓ Unknown: %d\n", summary["unknown"])
		}

		return nil
	},
}

var bucketStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detailed bucket status and statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		bucketMgr := client.NewBucketManager(config)
		
		fmt.Println("📊 Bucket Status Report")
		fmt.Println("======================")
		
		// Show configuration summary
		totalBuckets := len(config.Buckets)
		enabledBuckets := len(config.GetEnabledBuckets())
		
		fmt.Printf("\n📋 Configuration:\n")
		fmt.Printf("   Total buckets: %d\n", totalBuckets)
		fmt.Printf("   Enabled buckets: %d\n", enabledBuckets)
		fmt.Printf("   Default bucket: %s\n", config.DefaultBucket)
		fmt.Printf("   Cache directory: %s\n", config.Client.CacheDir)
		fmt.Printf("   Cache TTL: %d seconds\n", config.Client.CacheTTL)
		
		// Check health of all buckets
		fmt.Printf("\n🏥 Performing health checks...\n")
		bucketMgr.CheckAllHealth()
		bucketMgr.PrintHealthStatus()
		
		// Show cache status
		fmt.Printf("\n💾 Cache Status:\n")
		showCacheStatus(config)
		
		return nil
	},
}

func setBucketEnabled(name string, enabled bool) error {
	// Load client config
	config, err := client.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	bucket, exists := config.Buckets[name]
	if !exists {
		return fmt.Errorf("bucket '%s' not found", name)
	}

	bucket.Enabled = enabled
	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	status := "enabled"
	if !enabled {
		status = "disabled"
	}
	fmt.Printf("✓ Bucket '%s' %s\n", name, status)

	return nil
}

// printSingleBucketHealth prints detailed health info for a single bucket
func printSingleBucketHealth(health *client.BucketHealth) {
	fmt.Printf("\n🏥 Health Report for '%s':\n", health.Name)
	fmt.Printf("   URL: %s\n", health.URL)
	
	statusIcon := "❓"
	switch health.Status {
	case "healthy":
		statusIcon = "✅"
	case "degraded":
		statusIcon = "⚠️ "
	case "unhealthy":
		statusIcon = "❌"
	}
	
	fmt.Printf("   Status: %s %s\n", statusIcon, health.Status)
	fmt.Printf("   Last Check: %s\n", health.LastCheck.Format("2006-01-02 15:04:05"))
	
	if !health.LastSuccess.IsZero() {
		fmt.Printf("   Last Success: %s\n", health.LastSuccess.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("   Last Success: Never\n")
	}
	
	if health.ResponseTime > 0 {
		fmt.Printf("   Response Time: %dms\n", health.ResponseTime)
	}
	
	fmt.Printf("   Error Count: %d\n", health.ErrorCount)
	fmt.Printf("   Consecutive Fails: %d\n", health.ConsecutiveFails)
	
	if health.LastError != "" {
		fmt.Printf("   Last Error: %s\n", health.LastError)
	}
}

// showCacheStatus shows cache file information
func showCacheStatus(config *client.Config) {
	cacheDir := config.Client.CacheDir
	
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Printf("   Cache directory does not exist: %s\n", cacheDir)
		return
	}
	
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		fmt.Printf("   Error reading cache directory: %v\n", err)
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
	
	fmt.Printf("   Cache files: %d\n", len(cacheFiles))
	fmt.Printf("   Total cache size: %s\n", formatBytes(totalSize))
	
	if len(cacheFiles) > 0 {
		fmt.Printf("   Files: %s\n", strings.Join(cacheFiles, ", "))
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
