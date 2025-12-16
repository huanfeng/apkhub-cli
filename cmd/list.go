package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/huanfeng/apkhub/internal/config"
	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/models"
	"github.com/huanfeng/apkhub/pkg/repo"
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
			return fmt.Errorf("%s: %w", i18n.T("cmd.list.errLoadConfig"), err)
		}

		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.list.errCreateRepo"), err)
		}

		// Load manifest
		manifest, err := repository.BuildManifestFromInfos()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.list.errLoadManifest"), err)
		}

		if len(manifest.Packages) == 0 {
			fmt.Println(i18n.T("cmd.list.noPackages"))
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

	listCmd.Flags().StringVarP(&listPackageID, "package", "p", "", i18n.T("cmd.list.flag.package"))
	listCmd.Flags().BoolVarP(&showVersions, "versions", "v", false, i18n.T("cmd.list.flag.versions"))
	listCmd.Flags().StringVarP(&sortBy, "sort", "s", "name", i18n.T("cmd.list.flag.sort"))
}

func listSinglePackage(manifest *models.ManifestIndex, packageID string) error {
	pkg, exists := manifest.Packages[packageID]
	if !exists {
		return fmt.Errorf(i18n.T("cmd.list.packageNotFound", map[string]interface{}{
			"id": packageID,
		}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.list.packageDetailsTitle"))
	fmt.Printf("%s\n", i18n.T("cmd.list.packageID", map[string]interface{}{"id": packageID}))
	fmt.Printf("%s\n", i18n.T("cmd.list.packageName", map[string]interface{}{"name": getDefaultName(pkg.Name)}))
	fmt.Printf("%s\n", i18n.T("cmd.list.packageVersionCount", map[string]interface{}{"count": len(pkg.Versions)}))
	fmt.Printf("%s\n\n", i18n.T("cmd.list.packageLatest", map[string]interface{}{"version": pkg.Latest}))

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

	fmt.Printf("%s\n", i18n.T("cmd.list.versionsTitle"))
	for _, v := range versions {
		fmt.Printf("\n%s\n", i18n.T("cmd.list.versionHeader", map[string]interface{}{
			"version": v.Version.Version,
			"code":    v.Version.VersionCode,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.list.versionSize", map[string]interface{}{
			"size": fmt.Sprintf("%.2f MB", float64(v.Version.Size)/(1024*1024)),
		}))
		fmt.Printf("%s\n", i18n.T("cmd.list.versionSDK", map[string]interface{}{
			"min":    v.Version.MinSDK,
			"target": v.Version.TargetSDK,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.list.versionSHA", map[string]interface{}{
			"sha": v.Version.SHA256,
		}))
		if v.Version.SignatureInfo != nil && v.Version.SignatureInfo.SHA256 != "" {
			fmt.Printf("%s\n", i18n.T("cmd.list.versionSignature", map[string]interface{}{
				"sha": v.Version.SignatureInfo.SHA256[:16],
			}))
		}
		if len(v.Version.ABIs) > 0 {
			fmt.Printf("%s\n", i18n.T("cmd.list.versionABI", map[string]interface{}{
				"abis": strings.Join(v.Version.ABIs, ", "),
			}))
		}
		fmt.Printf("%s\n", i18n.T("cmd.list.versionDownload", map[string]interface{}{
			"url": v.Version.DownloadURL,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.list.versionAdded", map[string]interface{}{
			"date": v.Version.ReleaseDate.Format("2006-01-02 15:04:05"),
		}))
		if v.Version.SignatureVariant != "" {
			fmt.Printf("%s\n", i18n.T("cmd.list.versionSignatureVariant"))
		}
	}

	return nil
}

func listAllPackages(manifest *models.ManifestIndex) error {
	type packageInfo struct {
		ID            string
		Package       *models.AppPackage
		LatestVersion *models.AppVersion
		TotalSize     int64
		LastUpdated   string
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
	fmt.Printf("%-40s %-20s %-10s %-12s %s\n",
		i18n.T("cmd.list.columns.packageID"),
		i18n.T("cmd.list.columns.name"),
		i18n.T("cmd.list.columns.versions"),
		i18n.T("cmd.list.columns.totalSize"),
		i18n.T("cmd.list.columns.lastUpdated"))
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
				fmt.Printf("%s\n", i18n.T("cmd.list.versionListItem", map[string]interface{}{
					"prefix":  prefix,
					"version": version.Version,
					"code":    version.VersionCode,
					"size":    formatSize(version.Size),
				}))
			}
		}
	}

	// Summary
	fmt.Printf("\n%s\n", i18n.T("cmd.list.summary", map[string]interface{}{
		"packages": len(packages),
		"apks":     manifest.TotalAPKs,
	}))

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
