package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/apkhub/apkhub-cli/internal/config"
	"github.com/apkhub/apkhub-cli/pkg/apk"
	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/apkhub/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	outputPath string
	pretty     bool
	fullScan   bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Short: "Scan directory for APK files and update repository",
	Long:  `Scan the specified directory for APK/XAPK/APKM files and update the repository index. By default performs incremental scan.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		directory := args[0]
		
		// Convert to absolute paths
		absDir, err := filepath.Abs(directory)
		if err != nil {
			return fmt.Errorf("invalid directory path: %w", err)
		}
		
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override config with command line flags
		if cmd.Flags().Changed("recursive") {
			recursive, _ := cmd.Flags().GetBool("recursive")
			cfg.Scanning.Recursive = recursive
		}

		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}
		
		// Initialize repository structure
		if err := repository.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		fmt.Printf("Scanning directory: %s\n", absDir)
		fmt.Printf("Repository: %s\n", repository.GetRootDir())
		fmt.Printf("Mode: %s\n\n", getScanMode())

		// Load existing APK infos for incremental scan
		existingInfos := make(map[string]*models.APKInfo)
		if !fullScan {
			infos, err := repository.LoadAllAPKInfos()
			if err != nil {
				fmt.Printf("Warning: failed to load existing infos, performing full scan: %v\n", err)
				fullScan = true
			} else {
				// Build map by file path for quick lookup
				for _, info := range infos {
					existingInfos[info.OriginalName] = info
				}
				fmt.Printf("Loaded %d existing APK entries\n", len(existingInfos))
			}
		}

		// Perform scan
		parser := apk.NewParser(repository.GetRootDir())
		
		var (
			scannedFiles  int
			newAPKs       int
			updatedAPKs   int
			unchangedAPKs int
			errors        []error
		)
		
		// Walk through directory
		err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				errors = append(errors, fmt.Errorf("error accessing %s: %w", path, err))
				return nil // Continue scanning
			}

			// Skip directories
			if info.IsDir() {
				// Check if we should skip this directory
				if !cfg.Scanning.Recursive && path != absDir {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip symlinks if not following
			if info.Mode()&os.ModeSymlink != 0 && !cfg.Scanning.FollowSymlinks {
				return nil
			}

			// Check if file is APK
			if !apk.IsAPKFile(path) {
				return nil
			}

			scannedFiles++
			
			// Check if APK needs processing (incremental scan)
			filename := filepath.Base(path)
			existingInfo, exists := existingInfos[filename]
			
			if !fullScan && exists {
				// Check if file has been modified
				if info.ModTime().Equal(existingInfo.UpdatedAt) || info.ModTime().Before(existingInfo.UpdatedAt) {
					unchangedAPKs++
					fmt.Printf("Skip (unchanged): %s\n", filename)
					return nil
				}
			}
			
			// Parse APK
			fmt.Printf("Processing: %s\n", filename)
			apkInfo, err := parser.ParseAPK(path)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to parse %s: %w", filename, err))
				return nil
			}
			
			// Generate normalized filename
			normalizedName := repository.GenerateNormalizedFileName(apkInfo)
			
			// Create APK info
			modelAPKInfo := &models.APKInfo{
				PackageID:     apkInfo.PackageID,
				AppName:       apkInfo.AppName,
				Version:       apkInfo.Version,
				VersionCode:   apkInfo.VersionCode,
				MinSDK:        apkInfo.MinSDK,
				TargetSDK:     apkInfo.TargetSDK,
				Size:          apkInfo.Size,
				SHA256:        apkInfo.SHA256,
				SignatureInfo: apkInfo.SignatureInfo,
				Permissions:   apkInfo.Permissions,
				Features:      apkInfo.Features,
				ABIs:          apkInfo.ABIs,
				AddedAt:       time.Now(),
				UpdatedAt:     info.ModTime(),
				OriginalName:  filename,
				FileName:      normalizedName,
				FilePath:      filepath.Join("apks", normalizedName),
			}
			
			// If existing, preserve original added time
			if exists {
				modelAPKInfo.AddedAt = existingInfo.AddedAt
				updatedAPKs++
			} else {
				newAPKs++
			}
			
			// Check if APK exists in repository
			targetPath := repository.GetAPKPath(normalizedName)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				// Copy APK to repository
				fmt.Printf("  Copying to repository as: %s\n", normalizedName)
				if err := copyAPKFile(path, targetPath); err != nil {
					errors = append(errors, fmt.Errorf("failed to copy %s: %w", filename, err))
					return nil
				}
			}
			
			// Save APK info
			if err := repository.SaveAPKInfo(modelAPKInfo); err != nil {
				errors = append(errors, fmt.Errorf("failed to save info for %s: %w", filename, err))
				return nil
			}
			
			return nil
		})

		if err != nil {
			return fmt.Errorf("error walking directory: %w", err)
		}

		// Update manifest
		fmt.Printf("\nUpdating repository manifest...\n")
		if err := repository.UpdateManifest(); err != nil {
			return fmt.Errorf("failed to update manifest: %w", err)
		}

		// Print results
		fmt.Printf("\n=== Scan Results ===\n")
		fmt.Printf("Files scanned: %d\n", scannedFiles)
		fmt.Printf("New APKs: %d\n", newAPKs)
		fmt.Printf("Updated APKs: %d\n", updatedAPKs)
		fmt.Printf("Unchanged APKs: %d\n", unchangedAPKs)
		
		if len(errors) > 0 {
			fmt.Printf("\nErrors encountered (%d):\n", len(errors))
			for _, err := range errors {
				fmt.Printf("  - %v\n", err)
			}
		}
		
		fmt.Printf("\n✓ Repository updated successfully!\n")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	
	scanCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Legacy option (ignored, manifest is always apkhub_manifest.json)")
	scanCmd.Flags().BoolP("recursive", "r", true, "Scan directories recursively")
	scanCmd.Flags().BoolVar(&pretty, "pretty", true, "Pretty print JSON output")
	scanCmd.Flags().BoolVar(&fullScan, "full", false, "Perform full scan instead of incremental")
	
	// Mark output as deprecated
	scanCmd.Flags().MarkDeprecated("output", "manifest is always saved as apkhub_manifest.json")
}

// getScanMode returns the current scan mode as string
func getScanMode() string {
	if fullScan {
		return "Full scan"
	}
	return "Incremental scan"
}