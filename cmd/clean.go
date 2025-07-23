package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apkhub/apkhub-cli/internal/config"
	"github.com/apkhub/apkhub-cli/pkg/apk"
	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/apkhub/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	dryRun      bool
	keepVersions int
	removeOrphans bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up repository by removing old versions and orphan files",
	Long:  `Clean up the repository by removing old versions based on keep_versions configuration and orphan files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		
		// Override keep versions if specified
		if cmd.Flags().Changed("keep") {
			cfg.Repository.KeepVersions = keepVersions
		} else {
			keepVersions = cfg.Repository.KeepVersions
		}
		
		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}
		
		fmt.Printf("=== Repository Cleanup ===\n")
		fmt.Printf("Repository: %s\n", repository.GetRootDir())
		if dryRun {
			fmt.Printf("Mode: DRY RUN (no files will be deleted)\n")
		}
		if keepVersions > 0 {
			fmt.Printf("Keep Versions: %d\n", keepVersions)
		}
		fmt.Printf("\n")
		
		// Load all APK infos
		infos, err := repository.LoadAllAPKInfos()
		if err != nil {
			return fmt.Errorf("failed to load APK infos: %w", err)
		}
		
		// Group by package ID
		packageGroups := make(map[string][]*models.APKInfo)
		for _, info := range infos {
			packageGroups[info.PackageID] = append(packageGroups[info.PackageID], info)
		}
		
		var totalRemoved int
		var totalSize int64
		var filesToRemove []string
		
		// Process each package
		if keepVersions > 0 {
			fmt.Printf("=== Version Cleanup ===\n")
			
			for packageID, versions := range packageGroups {
				if len(versions) <= keepVersions {
					continue
				}
				
				// Sort by version code (newest first)
				sort.Slice(versions, func(i, j int) bool {
					return versions[i].VersionCode > versions[j].VersionCode
				})
				
				// Mark old versions for removal
				fmt.Printf("\nPackage: %s\n", packageID)
				fmt.Printf("  Current versions: %d\n", len(versions))
				
				for i := keepVersions; i < len(versions); i++ {
					version := versions[i]
					fmt.Printf("  Remove: v%s (Code: %d) - %s\n", 
						version.Version, 
						version.VersionCode,
						formatSize(version.Size))
					
					// Add files to removal list
					apkPath := filepath.Join(repository.GetRootDir(), version.FilePath)
					infoPath := filepath.Join(repository.GetRootDir(), version.InfoPath)
					
					filesToRemove = append(filesToRemove, apkPath, infoPath)
					totalRemoved++
					totalSize += version.Size
				}
			}
		}
		
		// Find orphan files
		if removeOrphans {
			fmt.Printf("\n=== Orphan File Detection ===\n")
			
			// Check for orphan APKs
			apksDir := filepath.Join(repository.GetRootDir(), "apks")
			orphanAPKs, err := findOrphanFiles(apksDir, infos, "apks")
			if err != nil {
				fmt.Printf("Warning: failed to check orphan APKs: %v\n", err)
			} else if len(orphanAPKs) > 0 {
				fmt.Printf("\nOrphan APK files:\n")
				for _, orphan := range orphanAPKs {
					fmt.Printf("  - %s\n", orphan)
					filesToRemove = append(filesToRemove, orphan)
				}
			}
			
			// Check for orphan info files
			infosDir := filepath.Join(repository.GetRootDir(), "infos")
			orphanInfos, err := findOrphanInfoFiles(infosDir, infos)
			if err != nil {
				fmt.Printf("Warning: failed to check orphan infos: %v\n", err)
			} else if len(orphanInfos) > 0 {
				fmt.Printf("\nOrphan info files:\n")
				for _, orphan := range orphanInfos {
					fmt.Printf("  - %s\n", orphan)
					filesToRemove = append(filesToRemove, orphan)
				}
			}
		}
		
		// Summary
		fmt.Printf("\n=== Summary ===\n")
		fmt.Printf("Files to remove: %d\n", len(filesToRemove))
		fmt.Printf("APKs to remove: %d\n", totalRemoved)
		fmt.Printf("Space to free: %s\n", formatSize(totalSize))
		
		if len(filesToRemove) == 0 {
			fmt.Printf("\nNothing to clean up!\n")
			return nil
		}
		
		// Confirm deletion
		if !dryRun && !skipConfirm {
			fmt.Print("\nProceed with cleanup? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Cleanup cancelled.")
				return nil
			}
		}
		
		// Delete files
		if !dryRun {
			fmt.Printf("\n=== Removing Files ===\n")
			var removed int
			for _, file := range filesToRemove {
				if err := os.Remove(file); err != nil {
					fmt.Printf("Failed to remove %s: %v\n", file, err)
				} else {
					removed++
				}
			}
			
			fmt.Printf("\nRemoved %d files\n", removed)
			
			// Update manifest
			fmt.Printf("Updating manifest...\n")
			if err := repository.UpdateManifest(); err != nil {
				return fmt.Errorf("failed to update manifest: %w", err)
			}
			
			fmt.Printf("\n✓ Cleanup completed successfully!\n")
		}
		
		return nil
	},
}

func init() {
	repoCmd.AddCommand(cleanCmd)
	
	cleanCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	cleanCmd.Flags().IntVarP(&keepVersions, "keep", "k", 0, "Number of versions to keep (overrides config)")
	cleanCmd.Flags().BoolVar(&removeOrphans, "orphans", true, "Remove orphan files")
	cleanCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
}

// findOrphanFiles finds files that don't have corresponding info entries
func findOrphanFiles(dir string, infos []*models.APKInfo, subdir string) ([]string, error) {
	// Build map of known files
	knownFiles := make(map[string]bool)
	for _, info := range infos {
		if strings.HasPrefix(info.FilePath, subdir+"/") {
			filename := filepath.Base(info.FilePath)
			knownFiles[filename] = true
		}
	}
	
	var orphans []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return orphans, nil
		}
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filename := entry.Name()
		if subdir == "apks" && !apk.IsAPKFile(filename) {
			continue
		}
		
		if !knownFiles[filename] {
			orphans = append(orphans, filepath.Join(dir, filename))
		}
	}
	
	return orphans, nil
}

// findOrphanInfoFiles finds info files without corresponding APK files
func findOrphanInfoFiles(dir string, infos []*models.APKInfo) ([]string, error) {
	// Build map of valid info files
	validInfos := make(map[string]bool)
	for _, info := range infos {
		infoName := filepath.Base(info.InfoPath)
		validInfos[infoName] = true
	}
	
	var orphans []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return orphans, nil
		}
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		
		if !validInfos[entry.Name()] {
			orphans = append(orphans, filepath.Join(dir, entry.Name()))
		}
	}
	
	return orphans, nil
}