package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/huanfeng/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	importFormat string
	importSource string
	downloadAPKs bool
	mapFields    map[string]string
)

var importCmd = &cobra.Command{
	Use:   "import [source]",
	Short: "Import APK metadata from other formats",
	Long:  `Import APK metadata from other repository formats such as F-Droid index or another ApkHub manifest.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		importSource = args[0]

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

		// Initialize repository structure
		if err := repository.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		fmt.Printf("=== Import APK Metadata ===\n")
		fmt.Printf("Source: %s\n", importSource)
		fmt.Printf("Format: %s\n", importFormat)
		fmt.Printf("Download APKs: %v\n\n", downloadAPKs)

		// Load source data
		var sourceData []byte
		if strings.HasPrefix(importSource, "http://") || strings.HasPrefix(importSource, "https://") {
			// Download from URL
			fmt.Printf("Downloading from URL...\n")
			resp, err := http.Get(importSource)
			if err != nil {
				return fmt.Errorf("failed to download: %w", err)
			}
			defer resp.Body.Close()

			sourceData, err = io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}
		} else {
			// Read from file
			sourceData, err = os.ReadFile(importSource)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
		}

		// Import based on format
		var importedPackages []*models.APKInfo

		switch importFormat {
		case "apkhub":
			importedPackages, err = importFromApkHub(sourceData)
		case "fdroid":
			importedPackages, err = importFromFDroid(sourceData)
		case "json":
			importedPackages, err = importFromGenericJSON(sourceData, mapFields)
		default:
			return fmt.Errorf("unsupported import format: %s", importFormat)
		}

		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		fmt.Printf("\nFound %d APKs to import\n", len(importedPackages))

		// Confirm import
		if !skipConfirm && len(importedPackages) > 0 {
			fmt.Printf("\nProceed with import? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Import cancelled.")
				return nil
			}
		}

		// Import APKs
		fmt.Printf("\n=== Importing APKs ===\n")
		var imported, skipped, failed int

		for _, apkInfo := range importedPackages {
			fmt.Printf("\nImporting: %s v%s\n", apkInfo.PackageID, apkInfo.Version)

			// Check if already exists
			existingInfos, _ := repository.LoadAllAPKInfos()
			exists := false
			for _, existing := range existingInfos {
				if existing.PackageID == apkInfo.PackageID &&
					existing.VersionCode == apkInfo.VersionCode {
					exists = true
					break
				}
			}

			if exists {
				fmt.Printf("  Skip: Already exists\n")
				skipped++
				continue
			}

			// Save APK info
			if err := repository.SaveAPKInfo(apkInfo); err != nil {
				fmt.Printf("  Failed: %v\n", err)
				failed++
				continue
			}

			// Download APK if requested and URL is available
			if downloadAPKs && apkInfo.FilePath != "" {
				fmt.Printf("  Downloading APK...\n")
				// TODO: Implement APK download
				fmt.Printf("  Download not implemented yet\n")
			}

			imported++
			fmt.Printf("  ✓ Imported successfully\n")
		}

		// Update manifest
		fmt.Printf("\nUpdating repository manifest...\n")
		if err := repository.UpdateManifest(); err != nil {
			return fmt.Errorf("failed to update manifest: %w", err)
		}

		// Summary
		fmt.Printf("\n=== Import Summary ===\n")
		fmt.Printf("Imported: %d\n", imported)
		fmt.Printf("Skipped: %d\n", skipped)
		fmt.Printf("Failed: %d\n", failed)
		fmt.Printf("\n✓ Import completed!\n")

		return nil
	},
}

func init() {
	repoCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importFormat, "format", "f", "apkhub", "Import format: apkhub, fdroid, json")
	importCmd.Flags().BoolVarP(&downloadAPKs, "download", "d", false, "Download APK files if URLs are provided")
	importCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	importCmd.Flags().StringToStringVar(&mapFields, "map", map[string]string{}, "Field mapping for generic JSON import")
}

// importFromApkHub imports from another ApkHub manifest
func importFromApkHub(data []byte) ([]*models.APKInfo, error) {
	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse ApkHub manifest: %w", err)
	}

	var apkInfos []*models.APKInfo

	for packageID, pkg := range manifest.Packages {
		for versionKey, version := range pkg.Versions {
			info := &models.APKInfo{
				PackageID:     packageID,
				AppName:       pkg.Name,
				Version:       version.Version,
				VersionCode:   version.VersionCode,
				MinSDK:        version.MinSDK,
				TargetSDK:     version.TargetSDK,
				Size:          version.Size,
				SHA256:        version.SHA256,
				SignatureInfo: version.SignatureInfo,
				Permissions:   version.Permissions,
				Features:      version.Features,
				ABIs:          version.ABIs,
				AddedAt:       time.Now(),
				UpdatedAt:     time.Now(),
				OriginalName:  fmt.Sprintf("%s_%s.apk", packageID, versionKey),
				FileName:      "", // Will be generated
				FilePath:      "", // Will be set during import
			}
			apkInfos = append(apkInfos, info)
		}
	}

	return apkInfos, nil
}

// importFromFDroid imports from F-Droid index format
func importFromFDroid(data []byte) ([]*models.APKInfo, error) {
	var fdroidIndex struct {
		Repo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"repo"`
		Apps []struct {
			PackageName string `json:"packageName"`
			Name        string `json:"name"`
		} `json:"apps"`
		Packages map[string][]struct {
			VersionName      string   `json:"versionName"`
			VersionCode      int64    `json:"versionCode"`
			Size             int64    `json:"size"`
			MinSdkVersion    int      `json:"minSdkVersion"`
			TargetSdkVersion int      `json:"targetSdkVersion"`
			Hash             string   `json:"hash"`
			HashType         string   `json:"hashType"`
			Added            int64    `json:"added"`
			Permissions      []string `json:"uses-permission"`
		} `json:"packages"`
	}

	if err := json.Unmarshal(data, &fdroidIndex); err != nil {
		return nil, fmt.Errorf("failed to parse F-Droid index: %w", err)
	}

	// Build app name map
	appNames := make(map[string]string)
	for _, app := range fdroidIndex.Apps {
		appNames[app.PackageName] = app.Name
	}

	var apkInfos []*models.APKInfo

	for packageID, versions := range fdroidIndex.Packages {
		appName := appNames[packageID]
		if appName == "" {
			appName = packageID
		}

		for _, version := range versions {
			info := &models.APKInfo{
				PackageID:     packageID,
				AppName:       map[string]string{"default": appName},
				Version:       version.VersionName,
				VersionCode:   version.VersionCode,
				MinSDK:        version.MinSdkVersion,
				TargetSDK:     version.TargetSdkVersion,
				Size:          version.Size,
				SHA256:        version.Hash,
				SignatureInfo: &models.SignatureInfo{},
				Permissions:   version.Permissions,
				AddedAt:       time.Unix(version.Added/1000, 0),
				UpdatedAt:     time.Now(),
				OriginalName:  fmt.Sprintf("%s_%d.apk", packageID, version.VersionCode),
			}
			apkInfos = append(apkInfos, info)
		}
	}

	return apkInfos, nil
}

// importFromGenericJSON imports from generic JSON with field mapping
func importFromGenericJSON(data []byte, fieldMap map[string]string) ([]*models.APKInfo, error) {
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		// Try single object
		var singleObj map[string]interface{}
		if err := json.Unmarshal(data, &singleObj); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: expected array or object")
		}
		rawData = []map[string]interface{}{singleObj}
	}

	// Default field mapping
	if len(fieldMap) == 0 {
		fieldMap = map[string]string{
			"package_id":   "package_id",
			"app_name":     "app_name",
			"version":      "version",
			"version_code": "version_code",
			"size":         "size",
			"sha256":       "sha256",
		}
	}

	var apkInfos []*models.APKInfo

	for _, item := range rawData {
		info := &models.APKInfo{
			AddedAt:       time.Now(),
			UpdatedAt:     time.Now(),
			AppName:       make(map[string]string),
			SignatureInfo: &models.SignatureInfo{},
		}

		// Map fields
		if v, ok := getFieldValue(item, fieldMap["package_id"]); ok {
			info.PackageID = fmt.Sprintf("%v", v)
		}
		if v, ok := getFieldValue(item, fieldMap["app_name"]); ok {
			info.AppName["default"] = fmt.Sprintf("%v", v)
		}
		if v, ok := getFieldValue(item, fieldMap["version"]); ok {
			info.Version = fmt.Sprintf("%v", v)
		}
		if v, ok := getFieldValue(item, fieldMap["version_code"]); ok {
			if code, err := parseNumber(v); err == nil {
				info.VersionCode = code
			}
		}
		if v, ok := getFieldValue(item, fieldMap["size"]); ok {
			if size, err := parseNumber(v); err == nil {
				info.Size = size
			}
		}
		if v, ok := getFieldValue(item, fieldMap["sha256"]); ok {
			info.SHA256 = fmt.Sprintf("%v", v)
		}

		// Set defaults
		if info.PackageID == "" {
			continue // Skip invalid entries
		}
		if info.AppName["default"] == "" {
			info.AppName["default"] = info.PackageID
		}

		info.OriginalName = fmt.Sprintf("%s_%d.apk", info.PackageID, info.VersionCode)

		apkInfos = append(apkInfos, info)
	}

	return apkInfos, nil
}

// getFieldValue gets a field value from a map, supporting nested paths
func getFieldValue(data map[string]interface{}, path string) (interface{}, bool) {
	if path == "" {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := current[part]
			return val, ok
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}

// parseNumber parses a number from various types
func parseNumber(v interface{}) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case string:
		var num int64
		_, err := fmt.Sscanf(val, "%d", &num)
		return num, err
	default:
		return 0, fmt.Errorf("cannot parse number from %T", v)
	}
}
