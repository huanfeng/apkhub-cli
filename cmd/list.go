package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apkhub/apkhub-cli/internal/config"
	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/apkhub/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	listPackageID string
	showVersions  bool
	sortBy        string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages in the repository",
	Long:  `List all packages in the repository with optional filtering and sorting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		
		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}
		
		// Load manifest
		manifest, err := repository.BuildManifestFromInfos()
		if err != nil {
			return fmt.Errorf("failed to load manifest: %w", err)
		}
		
		if len(manifest.Packages) == 0 {
			fmt.Println("No packages found in repository.")
			return nil
		}
		
		// Filter by package ID if specified
		if listPackageID != "" {
			return listSinglePackage(manifest, listPackageID)
		}
		
		// List all packages
		return listAllPackages(manifest)
	},
}

func init() {
	repoCmd.AddCommand(listCmd)
	
	listCmd.Flags().StringVarP(&listPackageID, "package", "p", "", "Show details for specific package ID")
	listCmd.Flags().BoolVarP(&showVersions, "versions", "v", false, "Show all versions for each package")
	listCmd.Flags().StringVarP(&sortBy, "sort", "s", "name", "Sort by: name, size, versions, updated")
}

func listSinglePackage(manifest *models.ManifestIndex, packageID string) error {
	pkg, exists := manifest.Packages[packageID]
	if !exists {
		return fmt.Errorf("package %s not found", packageID)
	}
	
	fmt.Printf("=== Package Details ===\n")
	fmt.Printf("Package ID: %s\n", packageID)
	fmt.Printf("Name: %s\n", getDefaultName(pkg.Name))
	fmt.Printf("Versions: %d\n", len(pkg.Versions))
	fmt.Printf("Latest: %s\n\n", pkg.Latest)
	
	// List all versions
	type versionInfo struct {
		Key     string
		Version *models.AppVersion
	}
	
	var versions []versionInfo
	for k, v := range pkg.Versions {
		versions = append(versions, versionInfo{k, v})
	}
	
	// Sort by version code
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version.VersionCode > versions[j].Version.VersionCode
	})
	
	fmt.Printf("=== Versions ===\n")
	for _, v := range versions {
		fmt.Printf("\n%s (Code: %d):\n", v.Version.Version, v.Version.VersionCode)
		fmt.Printf("  Size: %.2f MB\n", float64(v.Version.Size)/(1024*1024))
		fmt.Printf("  Min SDK: %d, Target SDK: %d\n", v.Version.MinSDK, v.Version.TargetSDK)
		fmt.Printf("  SHA256: %s\n", v.Version.SHA256)
		if v.Version.SignatureInfo != nil && v.Version.SignatureInfo.SHA256 != "" {
			fmt.Printf("  Signature: %s...\n", v.Version.SignatureInfo.SHA256[:16])
		}
		if len(v.Version.ABIs) > 0 {
			fmt.Printf("  ABIs: %s\n", strings.Join(v.Version.ABIs, ", "))
		}
		fmt.Printf("  Download: %s\n", v.Version.DownloadURL)
		fmt.Printf("  Added: %s\n", v.Version.ReleaseDate.Format("2006-01-02 15:04:05"))
		if v.Version.SignatureVariant != "" {
			fmt.Printf("  ⚠️  Alternative signature variant\n")
		}
	}
	
	return nil
}

func listAllPackages(manifest *models.ManifestIndex) error {
	type packageInfo struct {
		ID           string
		Package      *models.AppPackage
		LatestVersion *models.AppVersion
		TotalSize    int64
		LastUpdated  string
	}
	
	var packages []packageInfo
	
	// Collect package information
	for id, pkg := range manifest.Packages {
		info := packageInfo{
			ID:      id,
			Package: pkg,
		}
		
		// Calculate total size and find latest version
		var latestTime int64
		for _, version := range pkg.Versions {
			info.TotalSize += version.Size
			if version.ReleaseDate.Unix() > latestTime {
				latestTime = version.ReleaseDate.Unix()
				info.LastUpdated = version.ReleaseDate.Format("2006-01-02")
			}
			if pkg.Latest != "" && version.Version == pkg.Latest {
				info.LatestVersion = version
			}
		}
		
		packages = append(packages, info)
	}
	
	// Sort packages
	switch sortBy {
	case "size":
		sort.Slice(packages, func(i, j int) bool {
			return packages[i].TotalSize > packages[j].TotalSize
		})
	case "versions":
		sort.Slice(packages, func(i, j int) bool {
			return len(packages[i].Package.Versions) > len(packages[j].Package.Versions)
		})
	case "updated":
		sort.Slice(packages, func(i, j int) bool {
			return packages[i].LastUpdated > packages[j].LastUpdated
		})
	default: // name
		sort.Slice(packages, func(i, j int) bool {
			return packages[i].ID < packages[j].ID
		})
	}
	
	// Display header
	fmt.Printf("%-40s %-20s %-10s %-12s %s\n", "Package ID", "Name", "Versions", "Total Size", "Last Updated")
	fmt.Println(strings.Repeat("-", 100))
	
	// Display packages
	for _, info := range packages {
		name := getDefaultName(info.Package.Name)
		if len(name) > 18 {
			name = name[:15] + "..."
		}
		
		fmt.Printf("%-40s %-20s %-10d %-12s %s\n",
			info.ID,
			name,
			len(info.Package.Versions),
			formatSize(info.TotalSize),
			info.LastUpdated,
		)
		
		// Show version details if requested
		if showVersions {
			for _, version := range info.Package.Versions {
				prefix := "  └─"
				if version.SignatureVariant != "" {
					prefix = "  └─ ⚠️"
				}
				fmt.Printf("%s %s (Code: %d) - %s\n",
					prefix,
					version.Version,
					version.VersionCode,
					formatSize(version.Size),
				)
			}
		}
	}
	
	// Summary
	fmt.Printf("\nTotal: %d packages, %d APKs\n", len(packages), manifest.TotalAPKs)
	
	return nil
}

func formatSize(size int64) string {
	mb := float64(size) / (1024 * 1024)
	if mb < 1 {
		kb := float64(size) / 1024
		return fmt.Sprintf("%.1f KB", kb)
	}
	return fmt.Sprintf("%.1f MB", mb)
}