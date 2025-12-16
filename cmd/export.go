package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/huanfeng/apkhub-cli/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportFields []string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: i18n.T("cmd.export.short"),
	Long:  i18n.T("cmd.export.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.export.errLoadConfig"), err)
		}

		// Create repository instance
		repository, err := repo.NewRepository(workDir, cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.export.errCreateRepo"), err)
		}

		// Load manifest
		manifest, err := repository.BuildManifestFromInfos()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.export.errLoadManifest"), err)
		}

		// Determine output file
		if exportOutput == "" {
			exportOutput = fmt.Sprintf("apkhub_export.%s", exportFormat)
		}

		fmt.Printf("%s\n", i18n.T("cmd.export.start"))
		fmt.Printf("%s\n", i18n.T("cmd.export.format", map[string]interface{}{"format": exportFormat}))
		fmt.Printf("%s\n", i18n.T("cmd.export.output", map[string]interface{}{"output": exportOutput}))

		// Export based on format
		switch exportFormat {
		case "json":
			err = exportJSON(manifest, exportOutput)
		case "csv":
			err = exportCSV(manifest, exportOutput)
		case "md", "markdown":
			err = exportMarkdown(manifest, exportOutput)
		case "fdroid":
			err = exportFDroid(manifest, exportOutput)
		default:
			return fmt.Errorf("%s: %s", i18n.T("cmd.export.errUnsupported"), exportFormat)
		}

		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.export.errFailed"), err)
		}

		fmt.Printf("\n%s\n", i18n.T("cmd.export.success"))
		return nil
	},
}

func init() {
	repoCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", i18n.T("cmd.export.flag.format"))
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", i18n.T("cmd.export.flag.output"))
	exportCmd.Flags().StringSliceVar(&exportFields, "fields", []string{}, i18n.T("cmd.export.flag.fields"))
}

func exportJSON(manifest *models.ManifestIndex, output string) error {
	// Create output directory if needed
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.export.errOutputDir"), err)
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.export.errMarshalJSON"), err)
	}

	// Write to file
	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.export.errWriteFile"), err)
	}

	return nil
}

func exportCSV(manifest *models.ManifestIndex, output string) error {
	// Create output file
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.export.errCreateFile"), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Default fields if not specified
	if len(exportFields) == 0 {
		exportFields = []string{"package_id", "app_name", "version", "version_code", "size_mb", "min_sdk", "target_sdk", "sha256", "download_url"}
	}

	// Write header
	if err := writer.Write(exportFields); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.export.errWriteHeader"), err)
	}

	// Write data
	for packageID, pkg := range manifest.Packages {
		for _, version := range pkg.Versions {
			row := []string{}

			for _, field := range exportFields {
				switch field {
				case "package_id":
					row = append(row, packageID)
				case "app_name":
					row = append(row, getDefaultName(pkg.Name))
				case "version":
					row = append(row, version.Version)
				case "version_code":
					row = append(row, fmt.Sprintf("%d", version.VersionCode))
				case "size", "size_bytes":
					row = append(row, fmt.Sprintf("%d", version.Size))
				case "size_mb":
					row = append(row, fmt.Sprintf("%.2f", float64(version.Size)/(1024*1024)))
				case "min_sdk":
					row = append(row, fmt.Sprintf("%d", version.MinSDK))
				case "target_sdk":
					row = append(row, fmt.Sprintf("%d", version.TargetSDK))
				case "sha256":
					row = append(row, version.SHA256)
				case "download_url":
					row = append(row, version.DownloadURL)
				case "permissions":
					row = append(row, strings.Join(version.Permissions, ";"))
				case "abis":
					row = append(row, strings.Join(version.ABIs, ";"))
				case "release_date":
					row = append(row, version.ReleaseDate.Format("2006-01-02"))
				default:
					row = append(row, "")
				}
			}

			if err := writer.Write(row); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cmd.export.errWriteRow"), err)
			}
		}
	}

	return nil
}

func exportMarkdown(manifest *models.ManifestIndex, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.export.errCreateFile"), err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# %s\n\n", manifest.Name)
	fmt.Fprintf(file, "%s\n\n", manifest.Description)
	fmt.Fprintf(file, "Last Updated: %s\n\n", manifest.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Total Packages: %d | Total APKs: %d | Total Size: %.2f GB\n\n",
		len(manifest.Packages),
		manifest.TotalAPKs,
		float64(manifest.TotalSize)/(1024*1024*1024))

	// Write package list
	fmt.Fprintf(file, "## Packages\n\n")

	for packageID, pkg := range manifest.Packages {
		fmt.Fprintf(file, "### %s\n\n", packageID)
		fmt.Fprintf(file, "**Name:** %s\n", getDefaultName(pkg.Name))
		fmt.Fprintf(file, "**Versions:** %d\n", len(pkg.Versions))
		fmt.Fprintf(file, "**Latest:** %s\n\n", pkg.Latest)

		// Version table
		fmt.Fprintf(file, "| Version | Code | Size | Min SDK | SHA256 |\n")
		fmt.Fprintf(file, "|---------|------|------|---------|--------|\n")

		for _, version := range pkg.Versions {
			sha256Short := version.SHA256
			if len(sha256Short) > 16 {
				sha256Short = sha256Short[:16] + "..."
			}

			fmt.Fprintf(file, "| %s | %d | %.1f MB | %d | %s |\n",
				version.Version,
				version.VersionCode,
				float64(version.Size)/(1024*1024),
				version.MinSDK,
				sha256Short,
			)
		}

		fmt.Fprintf(file, "\n")
	}

	return nil
}

func exportFDroid(manifest *models.ManifestIndex, output string) error {
	// Create F-Droid compatible index structure
	fdroidIndex := map[string]interface{}{
		"repo": map[string]interface{}{
			"name":        manifest.Name,
			"description": manifest.Description,
			"timestamp":   manifest.UpdatedAt.Unix() * 1000, // F-Droid uses milliseconds
			"version":     21,                               // F-Droid index version
		},
		"apps":     []interface{}{},
		"packages": map[string][]interface{}{},
	}

	apps := []interface{}{}
	packages := make(map[string][]interface{})

	for packageID, pkg := range manifest.Packages {
		// Add app entry
		app := map[string]interface{}{
			"packageName": packageID,
			"name":        getDefaultName(pkg.Name),
			"added":       manifest.UpdatedAt.Unix() * 1000,
			"lastUpdated": manifest.UpdatedAt.Unix() * 1000,
		}
		apps = append(apps, app)

		// Add package versions
		versions := []interface{}{}
		for _, version := range pkg.Versions {
			ver := map[string]interface{}{
				"versionName":      version.Version,
				"versionCode":      version.VersionCode,
				"size":             version.Size,
				"minSdkVersion":    version.MinSDK,
				"targetSdkVersion": version.TargetSDK,
				"hash":             version.SHA256,
				"hashType":         "sha256",
				"added":            version.ReleaseDate.Unix() * 1000,
			}

			if len(version.Permissions) > 0 {
				ver["uses-permission"] = version.Permissions
			}

			versions = append(versions, ver)
		}
		packages[packageID] = versions
	}

	fdroidIndex["apps"] = apps
	fdroidIndex["packages"] = packages

	// Marshal to JSON
	data, err := json.MarshalIndent(fdroidIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal F-Droid index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
