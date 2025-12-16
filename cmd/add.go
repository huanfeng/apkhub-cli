package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanfeng/apkhub/internal/config"
	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/apk"
	"github.com/huanfeng/apkhub/pkg/models"
	"github.com/huanfeng/apkhub/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	skipConfirm bool
	copyFile    bool
)

var addCmd = &cobra.Command{
	Use:   "add [apk-file]",
	Short: i18n.T("cmd.repoAdd.short"),
	Long:  i18n.T("cmd.repoAdd.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apkPath := args[0]

		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errLoadConfig"), err)
		}

		// Convert to absolute paths
		absAPKPath, err := filepath.Abs(apkPath)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errAPKPath"), err)
		}

		// Check if file exists
		if _, err := os.Stat(absAPKPath); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errAPKNotFound"), err)
		}

		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errCreateRepo"), err)
		}

		// Initialize repository structure
		if err := repository.Initialize(); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errInitRepo"), err)
		}

		// Parse APK
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.parsing", map[string]interface{}{
			"path": absAPKPath,
		}))
		parser := apk.NewParser(repository.GetRootDir())
		apkInfo, err := parser.ParseAPK(absAPKPath)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errParse"), err)
		}

		// Generate normalized filename
		normalizedName := repository.GenerateNormalizedFileName(apkInfo)

		// Display APK information
		fmt.Println("\n" + i18n.T("cmd.repoAdd.info.title"))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.package", map[string]interface{}{
			"id": apkInfo.PackageID,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.appName", map[string]interface{}{
			"name": getDefaultName(apkInfo.AppName),
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.version", map[string]interface{}{
			"version": apkInfo.Version,
			"code":    apkInfo.VersionCode,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.size", map[string]interface{}{
			"size": float64(apkInfo.Size) / (1024 * 1024),
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.sdk", map[string]interface{}{
			"min":    apkInfo.MinSDK,
			"target": apkInfo.TargetSDK,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.sha256", map[string]interface{}{
			"sha": apkInfo.SHA256,
		}))
		if apkInfo.SignatureInfo != nil && len(apkInfo.SignatureInfo.SHA256) >= 16 {
			fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.signature", map[string]interface{}{
				"sha": apkInfo.SignatureInfo.SHA256[:16],
			}))
		} else if apkInfo.SignatureInfo != nil {
			fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.signatureMissing"))
		}
		if len(apkInfo.ABIs) > 0 {
			fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.abis", map[string]interface{}{
				"abis": strings.Join(apkInfo.ABIs, ", "),
			}))
		}
		fmt.Printf("\n%s\n", i18n.T("cmd.repoAdd.info.original", map[string]interface{}{
			"name": filepath.Base(absAPKPath),
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.newName", map[string]interface{}{
			"name": normalizedName,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.info.target", map[string]interface{}{
			"path": filepath.Join("apks", normalizedName),
		}))

		// Confirm addition
		if !skipConfirm {
			fmt.Print("\n" + i18n.T("cmd.repoAdd.confirm"))
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println(i18n.T("cmd.repoAdd.cancel"))
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
			return fmt.Errorf(i18n.T("cmd.repoAdd.errDuplicate", map[string]interface{}{
				"name": normalizedName,
			}))
		}

		fmt.Printf("\n%s\n", i18n.T("cmd.repoAdd.copying"))
		if copyFile {
			if err := copyAPKFile(absAPKPath, targetPath); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errCopy"), err)
			}
		} else {
			// Move file
			if err := os.Rename(absAPKPath, targetPath); err != nil {
				// If rename fails (cross-device), fall back to copy
				if err := copyAPKFile(absAPKPath, targetPath); err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errMove"), err)
				}
				// Remove original after successful copy
				os.Remove(absAPKPath)
			}
		}

		// Save APK info with icon
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.saving"))
		if err := repository.SaveAPKInfoWithIcon(apkInfo, modelAPKInfo); err != nil {
			// Rollback: remove copied APK
			os.Remove(targetPath)
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errSaveInfo"), err)
		}

		// Update manifest
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.updating"))
		if err := repository.UpdateManifest(); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errUpdateManifest"), err)
		}

		fmt.Printf("\n%s\n", i18n.T("cmd.repoAdd.success"))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.successLocation", map[string]interface{}{
			"path": modelAPKInfo.FilePath,
		}))
		fmt.Printf("%s\n", i18n.T("cmd.repoAdd.successInfo", map[string]interface{}{
			"path": modelAPKInfo.InfoPath,
		}))

		return nil
	},
}

func init() {
	repoCmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, i18n.T("cmd.repoAdd.flag.yes"))
	addCmd.Flags().BoolVarP(&copyFile, "copy", "c", false, i18n.T("cmd.repoAdd.flag.copy"))
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
		return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errCreateDir"), err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errOpenSrc"), err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errCreateDst"), err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errCopyContent"), err)
	}

	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.repoAdd.errSync"), err)
	}

	return nil
}
