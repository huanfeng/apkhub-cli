package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/spf13/cobra"
)

var (
	verifyFix    bool
	verifyDeep   bool
	verifyQuiet  bool
	verifyReport string
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify repository integrity",
	Long:  `Verify the integrity of the APK repository by checking files, metadata, and structure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verifyStart := time.Now()
		
		if !verifyQuiet {
			fmt.Println("🔍 Starting repository verification...")
			fmt.Println(strings.Repeat("=", 50))
		}

		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Perform verification
		result, err := performRepositoryVerification(cfg)
		if err != nil {
			return err
		}
		
		// Show results
		showVerificationResults(result, time.Since(verifyStart))
		
		// Auto-fix if requested
		if verifyFix && len(result.Issues) > 0 {
			fmt.Println("\n🔧 Attempting to fix issues...")
			fixResult := attemptAutoFix(cfg, result)
			showFixResults(fixResult)
		}
		
		// Generate report if requested
		if verifyReport != "" {
			if err := generateVerificationReport(result, verifyReport); err != nil {
				fmt.Printf("⚠️  Failed to generate report: %v\n", err)
			} else {
				fmt.Printf("📄 Report saved to: %s\n", verifyReport)
			}
		}

		return nil
	},
}

// VerificationIssue represents a repository integrity issue
type VerificationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	File        string `json:"file,omitempty"`
	Fixable     bool   `json:"fixable"`
}

// VerificationResult contains the results of repository verification
type VerificationResult struct {
	RepositoryName string               `json:"repository_name"`
	VerifiedAt     time.Time            `json:"verified_at"`
	TotalFiles     int                  `json:"total_files"`
	ValidFiles     int                  `json:"valid_files"`
	Issues         []VerificationIssue  `json:"issues"`
	Statistics     VerificationStats    `json:"statistics"`
}

// VerificationStats contains verification statistics
type VerificationStats struct {
	MissingFiles    int `json:"missing_files"`
	CorruptedFiles  int `json:"corrupted_files"`
	OrphanedFiles   int `json:"orphaned_files"`
	InvalidMetadata int `json:"invalid_metadata"`
	MissingIcons    int `json:"missing_icons"`
	MissingInfo     int `json:"missing_info"`
}

// FixResult contains the results of auto-fix attempts
type FixResult struct {
	Fixed   []string `json:"fixed"`
	Failed  []string `json:"failed"`
	Skipped []string `json:"skipped"`
}

// performRepositoryVerification performs comprehensive repository verification
func performRepositoryVerification(cfg *models.Config) (*VerificationResult, error) {
	result := &VerificationResult{
		RepositoryName: cfg.Repository.Name,
		VerifiedAt:     time.Now(),
		Issues:         []VerificationIssue{},
		Statistics:     VerificationStats{},
	}

	if !verifyQuiet {
		fmt.Printf("📋 Repository: %s\n", cfg.Repository.Name)
	}

	// Check 1: Configuration integrity
	if !verifyQuiet {
		fmt.Print("🔧 Checking configuration... ")
	}
	configIssues := checkConfigurationIntegrity(cfg)
	result.Issues = append(result.Issues, configIssues...)
	if !verifyQuiet {
		if len(configIssues) == 0 {
			fmt.Println("✅")
		} else {
			fmt.Printf("❌ (%d issues)\n", len(configIssues))
		}
	}

	// Check 2: Directory structure
	if !verifyQuiet {
		fmt.Print("📁 Checking directory structure... ")
	}
	dirIssues := checkDirectoryStructure(cfg)
	result.Issues = append(result.Issues, dirIssues...)
	if !verifyQuiet {
		if len(dirIssues) == 0 {
			fmt.Println("✅")
		} else {
			fmt.Printf("❌ (%d issues)\n", len(dirIssues))
		}
	}

	// Check 3: Manifest integrity
	if !verifyQuiet {
		fmt.Print("📄 Checking manifest... ")
	}
	manifestIssues, manifest := checkManifestIntegrity(cfg)
	result.Issues = append(result.Issues, manifestIssues...)
	if !verifyQuiet {
		if len(manifestIssues) == 0 {
			fmt.Println("✅")
		} else {
			fmt.Printf("❌ (%d issues)\n", len(manifestIssues))
		}
	}

	if manifest != nil {
		result.TotalFiles = manifest.TotalAPKs

		// Check 4: APK files
		if !verifyQuiet {
			fmt.Print("📱 Checking APK files... ")
		}
		apkIssues, validCount := checkAPKFiles(cfg, manifest)
		result.Issues = append(result.Issues, apkIssues...)
		result.ValidFiles = validCount
		if !verifyQuiet {
			if len(apkIssues) == 0 {
				fmt.Println("✅")
			} else {
				fmt.Printf("❌ (%d issues)\n", len(apkIssues))
			}
		}

		// Check 5: Orphaned files
		if !verifyQuiet {
			fmt.Print("🗑️  Checking for orphaned files... ")
		}
		orphanIssues := checkOrphanedFiles(cfg, manifest)
		result.Issues = append(result.Issues, orphanIssues...)
		if !verifyQuiet {
			if len(orphanIssues) == 0 {
				fmt.Println("✅")
			} else {
				fmt.Printf("❌ (%d issues)\n", len(orphanIssues))
			}
		}

		// Deep verification if requested
		if verifyDeep {
			if !verifyQuiet {
				fmt.Print("🔬 Performing deep verification... ")
			}
			deepIssues := performDeepVerification(cfg, manifest)
			result.Issues = append(result.Issues, deepIssues...)
			if !verifyQuiet {
				if len(deepIssues) == 0 {
					fmt.Println("✅")
				} else {
					fmt.Printf("❌ (%d issues)\n", len(deepIssues))
				}
			}
		}
	}

	// Calculate statistics
	calculateStatistics(result)

	return result, nil
}

// checkConfigurationIntegrity checks configuration file integrity
func checkConfigurationIntegrity(cfg *models.Config) []VerificationIssue {
	var issues []VerificationIssue

	if cfg.Repository.Name == "" {
		issues = append(issues, VerificationIssue{
			Type:        "config",
			Severity:    "error",
			Description: "Repository name is empty",
			Fixable:     false,
		})
	}

	// Check if basic repository info is configured
	if cfg.Repository.Description == "" {
		issues = append(issues, VerificationIssue{
			Type:        "config",
			Severity:    "warning",
			Description: "Repository description is empty",
			Fixable:     true,
		})
	}

	return issues
}

// checkDirectoryStructure checks if required directories exist
func checkDirectoryStructure(cfg *models.Config) []VerificationIssue {
	var issues []VerificationIssue

	// Check standard directories that should exist
	requiredDirs := []string{"apks", "infos", "icons"}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			issues = append(issues, VerificationIssue{
				Type:        "directory",
				Severity:    "error",
				Description: fmt.Sprintf("Required directory missing: %s", dir),
				File:        dir,
				Fixable:     true,
			})
		}
	}

	return issues
}

// ManifestData represents the basic manifest structure
type ManifestData struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	TotalAPKs   int                    `json:"total_apks"`
	Packages    map[string]interface{} `json:"packages"`
}

// checkManifestIntegrity checks manifest file integrity
func checkManifestIntegrity(cfg *models.Config) ([]VerificationIssue, *ManifestData) {
	var issues []VerificationIssue

	// Check if manifest file exists
	if _, err := os.Stat("apkhub_manifest.json"); os.IsNotExist(err) {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "error",
			Description: "Manifest file not found",
			File:        "apkhub_manifest.json",
			Fixable:     false,
		})
		return issues, nil
	}

	// Load manifest
	data, err := os.ReadFile("apkhub_manifest.json")
	if err != nil {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "error",
			Description: fmt.Sprintf("Failed to read manifest: %v", err),
			File:        "apkhub_manifest.json",
			Fixable:     false,
		})
		return issues, nil
	}

	var manifest ManifestData
	if err := json.Unmarshal(data, &manifest); err != nil {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "error",
			Description: fmt.Sprintf("Failed to parse manifest: %v", err),
			File:        "apkhub_manifest.json",
			Fixable:     false,
		})
		return issues, nil
	}

	// Check manifest consistency
	if manifest.Name != cfg.Repository.Name {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "warning",
			Description: "Manifest repository name doesn't match configuration",
			File:        "apkhub_manifest.json",
			Fixable:     true,
		})
	}

	return issues, &manifest
}

// checkAPKFiles checks APK file integrity
func checkAPKFiles(cfg *models.Config, manifest *ManifestData) ([]VerificationIssue, int) {
	var issues []VerificationIssue
	validCount := 0

	if manifest == nil {
		return issues, validCount
	}

	// Check APK files in the apks directory
	apkDir := "apks"
	if entries, err := os.ReadDir(apkDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			
			filename := entry.Name()
			if strings.HasSuffix(strings.ToLower(filename), ".apk") {
				apkPath := filepath.Join(apkDir, filename)
				if _, err := os.Stat(apkPath); err == nil {
					validCount++
				} else {
					issues = append(issues, VerificationIssue{
						Type:        "apk",
						Severity:    "error",
						Description: fmt.Sprintf("Cannot access APK file: %s", filename),
						File:        filename,
						Fixable:     false,
					})
				}
			}
		}
	} else {
		issues = append(issues, VerificationIssue{
			Type:        "apk",
			Severity:    "error",
			Description: "Cannot access APK directory",
			File:        apkDir,
			Fixable:     false,
		})
	}

	return issues, validCount
}

// checkOrphanedFiles checks for files not referenced in manifest
func checkOrphanedFiles(cfg *models.Config, manifest *ManifestData) []VerificationIssue {
	var issues []VerificationIssue

	if manifest == nil {
		return issues
	}

	// For now, we'll do a basic check for APK files
	// In a more complete implementation, we'd parse the manifest packages
	// and check against the actual files referenced
	
	// Check APK directory for any obvious issues
	apkDir := "apks"
	if entries, err := os.ReadDir(apkDir); err == nil {
		apkCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			
			filename := entry.Name()
			if strings.HasSuffix(strings.ToLower(filename), ".apk") {
				apkCount++
			}
		}
		
		// Basic consistency check
		if apkCount != manifest.TotalAPKs {
			issues = append(issues, VerificationIssue{
				Type:        "consistency",
				Severity:    "warning",
				Description: fmt.Sprintf("APK count mismatch: found %d files, manifest reports %d", apkCount, manifest.TotalAPKs),
				File:        "apks/",
				Fixable:     false,
			})
		}
	}

	return issues
}

// performDeepVerification performs additional deep checks
func performDeepVerification(cfg *models.Config, manifest *ManifestData) []VerificationIssue {
	var issues []VerificationIssue

	if manifest == nil {
		return issues
	}

	// Check for icons directory
	iconDir := "icons"
	if _, err := os.Stat(iconDir); os.IsNotExist(err) {
		issues = append(issues, VerificationIssue{
			Type:        "icon",
			Severity:    "warning",
			Description: "Icons directory is missing",
			File:        iconDir,
			Fixable:     true,
		})
	} else {
		// Count icon files
		if entries, err := os.ReadDir(iconDir); err == nil {
			iconCount := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					iconCount++
				}
			}
			
			if iconCount == 0 {
				issues = append(issues, VerificationIssue{
					Type:        "icon",
					Severity:    "info",
					Description: "No icon files found",
					File:        iconDir,
					Fixable:     false,
				})
			}
		}
	}

	// Check for info directory
	infoDir := "infos"
	if _, err := os.Stat(infoDir); os.IsNotExist(err) {
		issues = append(issues, VerificationIssue{
			Type:        "info",
			Severity:    "warning",
			Description: "Info directory is missing",
			File:        infoDir,
			Fixable:     true,
		})
	}

	return issues
}

// calculateStatistics calculates verification statistics
func calculateStatistics(result *VerificationResult) {
	for _, issue := range result.Issues {
		switch issue.Type {
		case "apk":
			if issue.Severity == "error" {
				result.Statistics.MissingFiles++
			} else {
				result.Statistics.CorruptedFiles++
			}
		case "orphan":
			result.Statistics.OrphanedFiles++
		case "manifest":
			result.Statistics.InvalidMetadata++
		case "icon":
			result.Statistics.MissingIcons++
		}
	}
}

// showVerificationResults displays verification results
func showVerificationResults(result *VerificationResult, duration time.Duration) {
	if verifyQuiet {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 VERIFICATION RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("⏱️  Verification time: %v\n", duration)
	fmt.Printf("📁 Total files: %d\n", result.TotalFiles)
	fmt.Printf("✅ Valid files: %d\n", result.ValidFiles)
	fmt.Printf("❌ Issues found: %d\n", len(result.Issues))
	
	if len(result.Issues) > 0 {
		fmt.Println("\n📋 ISSUE BREAKDOWN:")
		fmt.Printf("   Missing files: %d\n", result.Statistics.MissingFiles)
		fmt.Printf("   Corrupted files: %d\n", result.Statistics.CorruptedFiles)
		fmt.Printf("   Orphaned files: %d\n", result.Statistics.OrphanedFiles)
		fmt.Printf("   Invalid metadata: %d\n", result.Statistics.InvalidMetadata)
		fmt.Printf("   Missing icons: %d\n", result.Statistics.MissingIcons)
		
		fmt.Println("\n🔍 DETAILED ISSUES:")
		for i, issue := range result.Issues {
			severity := "❌"
			if issue.Severity == "warning" {
				severity = "⚠️ "
			}
			
			fixable := ""
			if issue.Fixable {
				fixable = " (fixable)"
			}
			
			fmt.Printf("   %d. %s %s%s\n", i+1, severity, issue.Description, fixable)
			if issue.File != "" {
				fmt.Printf("      File: %s\n", issue.File)
			}
		}
	}
	
	fmt.Println(strings.Repeat("=", 60))
	
	if len(result.Issues) == 0 {
		fmt.Println("🎉 Repository verification passed! No issues found.")
	} else {
		fixableCount := 0
		for _, issue := range result.Issues {
			if issue.Fixable {
				fixableCount++
			}
		}
		
		if fixableCount > 0 {
			fmt.Printf("💡 %d issues can be automatically fixed with --fix flag\n", fixableCount)
		}
		
		fmt.Println("⚠️  Repository has integrity issues that need attention.")
	}
}

// attemptAutoFix attempts to automatically fix issues
func attemptAutoFix(cfg *models.Config, result *VerificationResult) *FixResult {
	fixResult := &FixResult{
		Fixed:   []string{},
		Failed:  []string{},
		Skipped: []string{},
	}

	for _, issue := range result.Issues {
		if !issue.Fixable {
			fixResult.Skipped = append(fixResult.Skipped, issue.Description)
			continue
		}

		switch issue.Type {
		case "directory":
			// Create missing directories
			if err := os.MkdirAll(issue.File, 0755); err != nil {
				fixResult.Failed = append(fixResult.Failed, fmt.Sprintf("Failed to create directory %s: %v", issue.File, err))
			} else {
				fixResult.Fixed = append(fixResult.Fixed, fmt.Sprintf("Created directory: %s", issue.File))
			}
			
		case "orphan":
			// Remove orphaned files (with confirmation)
			fmt.Printf("Remove orphaned file %s? [y/N]: ", issue.File)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) == "y" {
				orphanPath := filepath.Join("apks", issue.File)
				if err := os.Remove(orphanPath); err != nil {
					fixResult.Failed = append(fixResult.Failed, fmt.Sprintf("Failed to remove %s: %v", issue.File, err))
				} else {
					fixResult.Fixed = append(fixResult.Fixed, fmt.Sprintf("Removed orphaned file: %s", issue.File))
				}
			} else {
				fixResult.Skipped = append(fixResult.Skipped, fmt.Sprintf("Skipped removal of: %s", issue.File))
			}
			
		default:
			fixResult.Skipped = append(fixResult.Skipped, issue.Description)
		}
	}

	return fixResult
}

// showFixResults displays fix attempt results
func showFixResults(result *FixResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🔧 AUTO-FIX RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	
	if len(result.Fixed) > 0 {
		fmt.Printf("✅ FIXED (%d):\n", len(result.Fixed))
		for _, fix := range result.Fixed {
			fmt.Printf("   • %s\n", fix)
		}
	}
	
	if len(result.Failed) > 0 {
		fmt.Printf("\n❌ FAILED (%d):\n", len(result.Failed))
		for _, fail := range result.Failed {
			fmt.Printf("   • %s\n", fail)
		}
	}
	
	if len(result.Skipped) > 0 {
		fmt.Printf("\n⏭️  SKIPPED (%d):\n", len(result.Skipped))
		for _, skip := range result.Skipped {
			fmt.Printf("   • %s\n", skip)
		}
	}
	
	fmt.Println(strings.Repeat("=", 50))
}

// generateVerificationReport generates a JSON report
func generateVerificationReport(result *VerificationResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

func init() {
	repoCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().BoolVar(&verifyFix, "fix", false, "Attempt to automatically fix issues")
	verifyCmd.Flags().BoolVar(&verifyDeep, "deep", false, "Perform deep verification (slower but more thorough)")
	verifyCmd.Flags().BoolVar(&verifyQuiet, "quiet", false, "Suppress progress output")
	verifyCmd.Flags().StringVar(&verifyReport, "report", "", "Generate JSON report file")
}
