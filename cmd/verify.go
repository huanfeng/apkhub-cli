package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/huanfeng/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	fixIssues bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify repository integrity",
	Long:  `Verify the integrity of the repository by checking that all indexed APKs exist and their checksums match.`,
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

		fmt.Println("Verifying repository integrity...")
		fmt.Printf("Repository: %s\n\n", repository.GetRootDir())

		// Load all APK infos
		infos, err := repository.LoadAllAPKInfos()
		if err != nil {
			return fmt.Errorf("failed to load APK infos: %w", err)
		}

		if len(infos) == 0 {
			fmt.Println("No APKs found in repository.")
			return nil
		}

		var (
			totalAPKs   = len(infos)
			missingAPKs []string
			corruptAPKs []string
			orphanInfos []string
			validAPKs   int
		)

		// Verify each APK
		for i, info := range infos {
			fmt.Printf("Verifying [%d/%d]: %s\n", i+1, totalAPKs, info.FileName)

			apkPath := repository.GetAPKPath(info.FileName)

			// Check if APK file exists
			fileInfo, err := os.Stat(apkPath)
			if err != nil {
				if os.IsNotExist(err) {
					missingAPKs = append(missingAPKs, info.FilePath)
					orphanInfos = append(orphanInfos, info.InfoPath)
					fmt.Printf("  ❌ APK file missing\n")
					continue
				}
				return fmt.Errorf("failed to stat APK: %w", err)
			}

			// Check file size
			if fileInfo.Size() != info.Size {
				fmt.Printf("  ⚠️  Size mismatch: expected %d, got %d\n", info.Size, fileInfo.Size())
			}

			// Verify SHA256 checksum
			sha256Hash, err := calculateFileSHA256(apkPath)
			if err != nil {
				fmt.Printf("  ❌ Failed to calculate checksum: %v\n", err)
				corruptAPKs = append(corruptAPKs, info.FilePath)
				continue
			}

			if sha256Hash != info.SHA256 {
				fmt.Printf("  ❌ Checksum mismatch\n")
				fmt.Printf("     Expected: %s\n", info.SHA256)
				fmt.Printf("     Got:      %s\n", sha256Hash)
				corruptAPKs = append(corruptAPKs, info.FilePath)
				continue
			}

			fmt.Printf("  ✓ Valid\n")
			validAPKs++
		}

		// Check for orphan APK files (APKs without info files)
		fmt.Printf("\nChecking for orphan APK files...\n")
		apksDir := filepath.Join(repository.GetRootDir(), "apks")
		orphanAPKs, err := findOrphanAPKs(apksDir, infos)
		if err != nil {
			fmt.Printf("Warning: failed to check for orphan APKs: %v\n", err)
		}

		// Print summary
		fmt.Printf("\n=== Verification Summary ===\n")
		fmt.Printf("Total APKs indexed: %d\n", totalAPKs)
		fmt.Printf("Valid APKs: %d\n", validAPKs)

		if len(missingAPKs) > 0 {
			fmt.Printf("\nMissing APKs (%d):\n", len(missingAPKs))
			for _, path := range missingAPKs {
				fmt.Printf("  - %s\n", path)
			}
		}

		if len(corruptAPKs) > 0 {
			fmt.Printf("\nCorrupt APKs (%d):\n", len(corruptAPKs))
			for _, path := range corruptAPKs {
				fmt.Printf("  - %s\n", path)
			}
		}

		if len(orphanAPKs) > 0 {
			fmt.Printf("\nOrphan APKs (%d) - files without info:\n", len(orphanAPKs))
			for _, path := range orphanAPKs {
				fmt.Printf("  - %s\n", path)
			}
		}

		// Handle fixes if requested
		if fixIssues && (len(orphanInfos) > 0 || len(orphanAPKs) > 0) {
			fmt.Printf("\n=== Fixing Issues ===\n")

			// Remove orphan info files
			if len(orphanInfos) > 0 {
				fmt.Printf("Removing orphan info files...\n")
				for _, infoPath := range orphanInfos {
					fullPath := filepath.Join(repository.GetRootDir(), infoPath)
					if err := os.Remove(fullPath); err != nil {
						fmt.Printf("  Failed to remove %s: %v\n", infoPath, err)
					} else {
						fmt.Printf("  Removed: %s\n", infoPath)
					}
				}
			}

			// Remove orphan APK files
			if len(orphanAPKs) > 0 {
				fmt.Printf("Remove orphan APK files? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if response == "y" || response == "Y" {
					for _, apkPath := range orphanAPKs {
						fullPath := filepath.Join(apksDir, apkPath)
						if err := os.Remove(fullPath); err != nil {
							fmt.Printf("  Failed to remove %s: %v\n", apkPath, err)
						} else {
							fmt.Printf("  Removed: %s\n", apkPath)
						}
					}
				}
			}

			// Update manifest
			fmt.Printf("\nUpdating manifest...\n")
			if err := repository.UpdateManifest(); err != nil {
				return fmt.Errorf("failed to update manifest: %w", err)
			}
		}

		// Final status
		if len(missingAPKs) == 0 && len(corruptAPKs) == 0 && len(orphanAPKs) == 0 {
			fmt.Printf("\n✓ Repository is valid!\n")
		} else {
			fmt.Printf("\n⚠️  Repository has issues. ")
			if !fixIssues {
				fmt.Printf("Run with --fix to attempt repairs.\n")
			}
		}

		return nil
	},
}

func init() {
	repoCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().BoolVar(&fixIssues, "fix", false, "Attempt to fix issues found during verification")
}

// calculateFileSHA256 calculates the SHA256 hash of a file
func calculateFileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// findOrphanAPKs finds APK files that don't have corresponding info files
func findOrphanAPKs(apksDir string, infos []*models.APKInfo) ([]string, error) {
	// Create a map of known APK filenames
	knownAPKs := make(map[string]bool)
	for _, info := range infos {
		knownAPKs[info.FileName] = true
	}

	// Check all files in apks directory
	var orphans []string
	entries, err := os.ReadDir(apksDir)
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
		if !apk.IsAPKFile(filename) {
			continue
		}

		if !knownAPKs[filename] {
			orphans = append(orphans, filename)
		}
	}

	return orphans, nil
}
