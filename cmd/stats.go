package cmd

import (
	"fmt"
	"sort"

	"github.com/apkhub/apkhub-cli/internal/config"
	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/apkhub/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display repository statistics",
	Long:  `Display detailed statistics about the APK repository including package counts, version distributions, and storage usage.`,
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
		
		// Calculate statistics
		stats := calculateStats(manifest)
		
		// Display statistics
		fmt.Printf("=== ApkHub Repository Statistics ===\n\n")
		fmt.Printf("Repository: %s\n", repository.GetRootDir())
		fmt.Printf("Name: %s\n", manifest.Name)
		fmt.Printf("Description: %s\n", manifest.Description)
		fmt.Printf("Last Updated: %s\n\n", manifest.UpdatedAt.Format("2006-01-02 15:04:05"))
		
		fmt.Printf("=== Package Summary ===\n")
		fmt.Printf("Total Packages: %d\n", stats.TotalPackages)
		fmt.Printf("Total APKs: %d\n", stats.TotalAPKs)
		fmt.Printf("Total Size: %.2f MB\n", float64(stats.TotalSize)/(1024*1024))
		fmt.Printf("Average APK Size: %.2f MB\n", stats.AverageSize/(1024*1024))
		
		fmt.Printf("\n=== Version Distribution ===\n")
		fmt.Printf("Packages with Multiple Versions: %d\n", stats.MultiVersionPackages)
		fmt.Printf("Average Versions per Package: %.2f\n", stats.AverageVersionsPerPackage)
		
		// Display top packages by version count
		if len(stats.TopPackagesByVersions) > 0 {
			fmt.Printf("\nTop Packages by Version Count:\n")
			for i, pkg := range stats.TopPackagesByVersions {
				if i >= 5 {
					break
				}
				fmt.Printf("  %d. %s: %d versions\n", i+1, pkg.Name, pkg.Count)
			}
		}
		
		fmt.Printf("\n=== SDK Version Analysis ===\n")
		fmt.Printf("Min SDK Versions:\n")
		displaySDKStats(stats.MinSDKDistribution)
		fmt.Printf("\nTarget SDK Versions:\n")
		displaySDKStats(stats.TargetSDKDistribution)
		
		fmt.Printf("\n=== Architecture Distribution ===\n")
		displayDistribution(stats.ABIDistribution)
		
		fmt.Printf("\n=== Signature Analysis ===\n")
		fmt.Printf("Unique Signatures: %d\n", stats.UniqueSignatures)
		fmt.Printf("Packages with Multiple Signatures: %d\n", stats.MultiSignaturePackages)
		
		if len(stats.LargestAPKs) > 0 {
			fmt.Printf("\n=== Largest APKs ===\n")
			for i, apk := range stats.LargestAPKs {
				if i >= 5 {
					break
				}
				fmt.Printf("  %d. %s v%s: %.2f MB\n", i+1, apk.PackageID, apk.Version, float64(apk.Size)/(1024*1024))
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
		fmt.Println("  No data available")
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
				fmt.Printf("  ... (%d more)\n", len(sdks)-6)
			}
			continue
		}
		fmt.Printf("  API %d: %d APKs\n", sdk, distribution[sdk])
	}
}

func displayDistribution(distribution map[string]int) {
	if len(distribution) == 0 {
		fmt.Println("  No data available")
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
		fmt.Printf("  %s: %d APKs\n", item.Key, item.Value)
	}
}

func init() {
	repoCmd.AddCommand(statsCmd)
}