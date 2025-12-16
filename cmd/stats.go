package cmd

import (
	"fmt"
	"sort"

	"github.com/huanfeng/apkhub/internal/config"
	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/models"
	"github.com/huanfeng/apkhub/pkg/repo"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: i18n.T("cmd.stats.short"),
	Long:  i18n.T("cmd.stats.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.stats.errLoadConfig"), err)
		}

		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.stats.errCreateRepo"), err)
		}

		// Load manifest
		manifest, err := repository.BuildManifestFromInfos()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.stats.errLoadManifest"), err)
		}

		// Calculate statistics
		stats := calculateStats(manifest)

		// Display statistics
		fmt.Printf("%s\n\n", i18n.T("cmd.stats.title"))
		fmt.Printf("%s\n", i18n.T("cmd.stats.repo", map[string]interface{}{"path": repository.GetRootDir()}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.name", map[string]interface{}{"name": manifest.Name}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.description", map[string]interface{}{"desc": manifest.Description}))
		fmt.Printf("%s\n\n", i18n.T("cmd.stats.updated", map[string]interface{}{
			"time": manifest.UpdatedAt.Format("2006-01-02 15:04:05"),
		}))

		fmt.Printf("%s\n", i18n.T("cmd.stats.packageTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.stats.totalPackages", map[string]interface{}{"count": stats.TotalPackages}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.totalAPKs", map[string]interface{}{"count": stats.TotalAPKs}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.totalSizeMB", map[string]interface{}{
			"size": float64(stats.TotalSize) / (1024 * 1024),
		}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.avgSizeMB", map[string]interface{}{
			"size": stats.AverageSize / (1024 * 1024),
		}))

		fmt.Printf("\n%s\n", i18n.T("cmd.stats.versionTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.stats.multiVersion", map[string]interface{}{"count": stats.MultiVersionPackages}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.avgVersions", map[string]interface{}{"avg": stats.AverageVersionsPerPackage}))

		// Display top packages by version count
		if len(stats.TopPackagesByVersions) > 0 {
			fmt.Printf("\n%s\n", i18n.T("cmd.stats.topPackages"))
			for i, pkg := range stats.TopPackagesByVersions {
				if i >= 5 {
					break
				}
				fmt.Printf("%s\n", i18n.T("cmd.stats.topPackageItem", map[string]interface{}{
					"index": i + 1, "name": pkg.Name, "count": pkg.Count,
				}))
			}
		}

		fmt.Printf("\n%s\n", i18n.T("cmd.stats.sdkTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.stats.sdkMin"))
		displaySDKStats(stats.MinSDKDistribution)
		fmt.Printf("\n%s\n", i18n.T("cmd.stats.sdkTarget"))
		displaySDKStats(stats.TargetSDKDistribution)

		fmt.Printf("\n%s\n", i18n.T("cmd.stats.abiTitle"))
		displayDistribution(stats.ABIDistribution)

		fmt.Printf("\n%s\n", i18n.T("cmd.stats.signatureTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.stats.signatureUnique", map[string]interface{}{"count": stats.UniqueSignatures}))
		fmt.Printf("%s\n", i18n.T("cmd.stats.signatureMulti", map[string]interface{}{"count": stats.MultiSignaturePackages}))

		if len(stats.LargestAPKs) > 0 {
			fmt.Printf("\n%s\n", i18n.T("cmd.stats.largestTitle"))
			for i, apk := range stats.LargestAPKs {
				if i >= 5 {
					break
				}
				fmt.Printf("%s\n", i18n.T("cmd.stats.largestItem", map[string]interface{}{
					"index": i + 1, "id": apk.PackageID, "version": apk.Version,
					"size": float64(apk.Size) / (1024 * 1024),
				}))
			}
		}

		return nil
	},
}

type RepositoryStats struct {
	TotalPackages             int
	TotalAPKs                 int
	TotalSize                 int64
	AverageSize               float64
	MultiVersionPackages      int
	AverageVersionsPerPackage float64
	MinSDKDistribution        map[int]int
	TargetSDKDistribution     map[int]int
	ABIDistribution           map[string]int
	UniqueSignatures          int
	MultiSignaturePackages    int
	TopPackagesByVersions     []PackageCount
	LargestAPKs               []APKSize
}

type PackageCount struct {
	Name  string
	Count int
}

type APKSize struct {
	PackageID string
	Version   string
	Size      int64
}

func calculateStats(manifest *models.ManifestIndex) *RepositoryStats {
	stats := &RepositoryStats{
		MinSDKDistribution:    make(map[int]int),
		TargetSDKDistribution: make(map[int]int),
		ABIDistribution:       make(map[string]int),
	}

	signatures := make(map[string]bool)
	var totalSize int64
	var apkSizes []APKSize
	var packageVersionCounts []PackageCount

	for packageID, pkg := range manifest.Packages {
		stats.TotalPackages++
		versionCount := len(pkg.Versions)

		if versionCount > 1 {
			stats.MultiVersionPackages++
		}

		packageVersionCounts = append(packageVersionCounts, PackageCount{
			Name:  packageID,
			Count: versionCount,
		})

		packageSignatures := make(map[string]bool)

		for versionKey, version := range pkg.Versions {
			stats.TotalAPKs++
			totalSize += version.Size

			// SDK distribution
			if version.MinSDK > 0 {
				stats.MinSDKDistribution[version.MinSDK]++
			}
			if version.TargetSDK > 0 {
				stats.TargetSDKDistribution[version.TargetSDK]++
			}

			// ABI distribution
			for _, abi := range version.ABIs {
				stats.ABIDistribution[abi]++
			}

			// Signature tracking
			if version.SignatureInfo != nil && version.SignatureInfo.SHA256 != "" {
				signatures[version.SignatureInfo.SHA256] = true
				packageSignatures[version.SignatureInfo.SHA256] = true
			}

			// Track APK sizes
			apkSizes = append(apkSizes, APKSize{
				PackageID: packageID,
				Version:   versionKey,
				Size:      version.Size,
			})
		}

		if len(packageSignatures) > 1 {
			stats.MultiSignaturePackages++
		}
	}

	stats.TotalSize = totalSize
	if stats.TotalAPKs > 0 {
		stats.AverageSize = float64(totalSize) / float64(stats.TotalAPKs)
		stats.AverageVersionsPerPackage = float64(stats.TotalAPKs) / float64(stats.TotalPackages)
	}
	stats.UniqueSignatures = len(signatures)

	// Sort packages by version count
	sort.Slice(packageVersionCounts, func(i, j int) bool {
		return packageVersionCounts[i].Count > packageVersionCounts[j].Count
	})
	stats.TopPackagesByVersions = packageVersionCounts

	// Sort APKs by size
	sort.Slice(apkSizes, func(i, j int) bool {
		return apkSizes[i].Size > apkSizes[j].Size
	})
	stats.LargestAPKs = apkSizes

	return stats
}

func displaySDKStats(distribution map[int]int) {
	if len(distribution) == 0 {
		fmt.Println(i18n.T("cmd.stats.noData"))
		return
	}

	// Sort SDK versions
	var sdks []int
	for sdk := range distribution {
		sdks = append(sdks, sdk)
	}
	sort.Ints(sdks)

	// Display top SDKs
	for i, sdk := range sdks {
		if i >= 5 && i < len(sdks)-1 {
			if i == 5 {
				fmt.Printf("%s\n", i18n.T("cmd.stats.more", map[string]interface{}{"count": len(sdks) - 6}))
			}
			continue
		}
		fmt.Printf("%s\n", i18n.T("cmd.stats.sdkItem", map[string]interface{}{
			"sdk": sdk, "count": distribution[sdk],
		}))
	}
}

func displayDistribution(distribution map[string]int) {
	if len(distribution) == 0 {
		fmt.Println(i18n.T("cmd.stats.noData"))
		return
	}

	// Sort by count
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range distribution {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for _, item := range sorted {
		fmt.Printf("%s\n", i18n.T("cmd.stats.distributionItem", map[string]interface{}{
			"key": item.Key, "count": item.Value,
		}))
	}
}

func init() {
	repoCmd.AddCommand(statsCmd)
}
