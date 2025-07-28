package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <package-id>",
	Short: "Show detailed information about an application",
	Long:  `Show detailed information about an application including all available versions.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageID := args[0]

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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}