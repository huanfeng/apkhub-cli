package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apkhub/apkhub-cli/internal/config"
	"github.com/apkhub/apkhub-cli/pkg/apk"
	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/apkhub/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	skipConfirm bool
	copyFile    bool
)

var addCmd = &cobra.Command{
	Use:   "add [apk-file]",
	Short: "Add an APK file to the repository",
	Long:  `Add an APK file to the repository. The file will be analyzed, renamed according to naming conventions, and copied to the repository.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apkPath := args[0]
		
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		
		// Convert to absolute paths
		absAPKPath, err := filepath.Abs(apkPath)
		if err != nil {
			return fmt.Errorf("invalid APK path: %w", err)
		}
		
		// Check if file exists
		if _, err := os.Stat(absAPKPath); err != nil {
			return fmt.Errorf("APK file not found: %w", err)
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
		
		// Parse APK
		fmt.Printf("Parsing APK: %s\n", absAPKPath)
		parser := apk.NewParser(repository.GetRootDir())
		apkInfo, err := parser.ParseAPK(absAPKPath)
		if err != nil {
			return fmt.Errorf("failed to parse APK: %w", err)
		}
		
		// Generate normalized filename
		normalizedName := repository.GenerateNormalizedFileName(apkInfo)
		
		// Display APK information
		fmt.Println("\n=== APK Information ===")
		fmt.Printf("Package ID: %s\n", apkInfo.PackageID)
		fmt.Printf("App Name: %s\n", getDefaultName(apkInfo.AppName))
		fmt.Printf("Version: %s (Code: %d)\n", apkInfo.Version, apkInfo.VersionCode)
		fmt.Printf("Size: %.2f MB\n", float64(apkInfo.Size)/(1024*1024))
		fmt.Printf("Min SDK: %d, Target SDK: %d\n", apkInfo.MinSDK, apkInfo.TargetSDK)
		fmt.Printf("SHA256: %s\n", apkInfo.SHA256)
		if apkInfo.SignatureInfo != nil && len(apkInfo.SignatureInfo.SHA256) >= 16 {
			fmt.Printf("Signature SHA256: %s...\n", apkInfo.SignatureInfo.SHA256[:16])
		} else if apkInfo.SignatureInfo != nil {
			fmt.Printf("Signature: (extraction failed)\n")
		}
		if len(apkInfo.ABIs) > 0 {
			fmt.Printf("ABIs: %s\n", strings.Join(apkInfo.ABIs, ", "))
		}
		fmt.Printf("\nOriginal filename: %s\n", filepath.Base(absAPKPath))
		fmt.Printf("New filename: %s\n", normalizedName)
		fmt.Printf("Target location: %s\n", filepath.Join("apks", normalizedName))
		
		// Confirm addition
		if !skipConfirm {
			fmt.Print("\nAdd this APK to repository? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}
		
		// Create APK info structure
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
			UpdatedAt:     time.Now(),
			OriginalName:  filepath.Base(absAPKPath),
			FileName:      normalizedName,
			FilePath:      filepath.Join("apks", normalizedName),
		}
		
		// Copy or move APK to repository
		targetPath := repository.GetAPKPath(normalizedName)
		
		// Check if target already exists
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("APK with same name already exists in repository: %s", normalizedName)
		}
		
		fmt.Printf("\nCopying APK to repository...\n")
		if copyFile {
			if err := copyAPKFile(absAPKPath, targetPath); err != nil {
				return fmt.Errorf("failed to copy APK: %w", err)
			}
		} else {
			// Move file
			if err := os.Rename(absAPKPath, targetPath); err != nil {
				// If rename fails (cross-device), fall back to copy
				if err := copyAPKFile(absAPKPath, targetPath); err != nil {
					return fmt.Errorf("failed to move APK: %w", err)
				}
				// Remove original after successful copy
				os.Remove(absAPKPath)
			}
		}
		
		// Save APK info with icon
		fmt.Printf("Saving APK information...\n")
		if err := repository.SaveAPKInfoWithIcon(apkInfo, modelAPKInfo); err != nil {
			// Rollback: remove copied APK
			os.Remove(targetPath)
			return fmt.Errorf("failed to save APK info: %w", err)
		}
		
		// Update manifest
		fmt.Printf("Updating repository manifest...\n")
		if err := repository.UpdateManifest(); err != nil {
			return fmt.Errorf("failed to update manifest: %w", err)
		}
		
		fmt.Printf("\n✓ APK successfully added to repository!\n")
		fmt.Printf("  Location: %s\n", modelAPKInfo.FilePath)
		fmt.Printf("  Info: %s\n", modelAPKInfo.InfoPath)
		
		return nil
	},
}

func init() {
	repoCmd.AddCommand(addCmd)
	
	addCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	addCmd.Flags().BoolVarP(&copyFile, "copy", "c", false, "Copy file instead of moving")
}

// getDefaultName returns the default name from multi-language map
func getDefaultName(names map[string]string) string {
	if name, ok := names["default"]; ok {
		return name
	}
	if name, ok := names["en"]; ok {
		return name
	}
	// Return first available name
	for _, name := range names {
		return name
	}
	return "Unknown"
}

// copyAPKFile copies an APK file from source to destination
func copyAPKFile(src, dst string) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()
	
	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()
	
	// Copy content
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	
	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}
	
	return nil
}