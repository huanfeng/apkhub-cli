package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/huanfeng/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	dryRun        bool
	keepVersions  int
	removeOrphans bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: i18n.T("cmd.clean.short"),
	Long:  i18n.T("cmd.clean.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.clean.errLoadConfig"), err)
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
			return fmt.Errorf("%s: %w", i18n.T("cmd.clean.errCreateRepo"), err)
		}

		fmt.Printf("%s\n", i18n.T("cmd.clean.title"))
		fmt.Printf("%s\n", i18n.T("cmd.clean.repoPath", map[string]interface{}{"path": repository.GetRootDir()}))
		if dryRun {
			fmt.Printf("%s\n", i18n.T("cmd.clean.modeDryRun"))
		}
		if keepVersions > 0 {
			fmt.Printf("%s\n", i18n.T("cmd.clean.keepVersions", map[string]interface{}{"count": keepVersions}))
		}
		fmt.Printf("\n")

		// Load all APK infos
		infos, err := repository.LoadAllAPKInfos()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.clean.errLoadInfos"), err)
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
			fmt.Printf("%s\n", i18n.T("cmd.clean.versionCleanup"))

			for packageID, versions := range packageGroups {
				if len(versions) <= keepVersions {
					continue
				}

				// Sort by version code (newest first)
				sort.Slice(versions, func(i, j int) bool {
					return versions[i].VersionCode > versions[j].VersionCode
				})

				// Mark old versions for removal
				fmt.Printf("\n%s\n", i18n.T("cmd.clean.package", map[string]interface{}{"id": packageID}))
				fmt.Printf("%s\n", i18n.T("cmd.clean.packageVersions", map[string]interface{}{"count": len(versions)}))

				for i := keepVersions; i < len(versions); i++ {
					version := versions[i]
					fmt.Printf("%s\n", i18n.T("cmd.clean.removeVersion", map[string]interface{}{
						"version": version.Version,
						"code":    version.VersionCode,
						"size":    formatSize(version.Size),
					}))

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
			fmt.Printf("\n%s\n", i18n.T("cmd.clean.orphanTitle"))

			// Check for orphan APKs
			apksDir := filepath.Join(repository.GetRootDir(), "apks")
			orphanAPKs, err := findOrphanFiles(apksDir, infos, "apks")
			if err != nil {
				fmt.Printf("%s\n", i18n.T("cmd.clean.orphanWarnAPK", map[string]interface{}{"error": err}))
			} else if len(orphanAPKs) > 0 {
				fmt.Printf("\n%s\n", i18n.T("cmd.clean.orphanAPKTitle"))
				for _, orphan := range orphanAPKs {
					fmt.Printf("  - %s\n", orphan)
					filesToRemove = append(filesToRemove, orphan)
				}
			}

			// Check for orphan info files
			infosDir := filepath.Join(repository.GetRootDir(), "infos")
			orphanInfos, err := findOrphanInfoFiles(infosDir, infos)
			if err != nil {
				fmt.Printf("%s\n", i18n.T("cmd.clean.orphanWarnInfo", map[string]interface{}{"error": err}))
			} else if len(orphanInfos) > 0 {
				fmt.Printf("\n%s\n", i18n.T("cmd.clean.orphanInfoTitle"))
				for _, orphan := range orphanInfos {
					fmt.Printf("  - %s\n", orphan)
					filesToRemove = append(filesToRemove, orphan)
				}
			}
		}

		// Summary
		fmt.Printf("\n%s\n", i18n.T("cmd.clean.summaryTitle"))
		fmt.Printf("%s\n", i18n.T("cmd.clean.summaryFiles", map[string]interface{}{"count": len(filesToRemove)}))
		fmt.Printf("%s\n", i18n.T("cmd.clean.summaryAPKs", map[string]interface{}{"count": totalRemoved}))
		fmt.Printf("%s\n", i18n.T("cmd.clean.summarySpace", map[string]interface{}{"size": formatSize(totalSize)}))

		if len(filesToRemove) == 0 {
			fmt.Printf("\n%s\n", i18n.T("cmd.clean.nothing"))
			return nil
		}

		// Confirm deletion
		if !dryRun && !skipConfirm {
			fmt.Print("\n" + i18n.T("cmd.clean.confirm"))
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println(i18n.T("cmd.clean.cancel"))
				return nil
			}
		}

		// Delete files
		if !dryRun {
			fmt.Printf("\n%s\n", i18n.T("cmd.clean.removing"))
			var removed int
			for _, file := range filesToRemove {
				if err := os.Remove(file); err != nil {
					fmt.Printf("%s\n", i18n.T("cmd.clean.errRemove", map[string]interface{}{
						"file":  file,
						"error": err,
					}))
				} else {
					removed++
				}
			}

			fmt.Printf("\n%s\n", i18n.T("cmd.clean.removedFiles", map[string]interface{}{"count": removed}))

			// Update manifest
			fmt.Printf("%s\n", i18n.T("cmd.clean.updateManifest"))
			if err := repository.UpdateManifest(); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cmd.clean.errUpdateManifest"), err)
			}

			fmt.Printf("\n%s\n", i18n.T("cmd.clean.success"))
		}

		return nil
	},
}

func init() {
	repoCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolVar(&dryRun, "dry-run", false, i18n.T("cmd.clean.flag.dryRun"))
	cleanCmd.Flags().IntVarP(&keepVersions, "keep", "k", 0, i18n.T("cmd.clean.flag.keep"))
	cleanCmd.Flags().BoolVar(&removeOrphans, "orphans", true, i18n.T("cmd.clean.flag.orphans"))
	cleanCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, i18n.T("cmd.clean.flag.yes"))
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
