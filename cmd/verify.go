package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/spf13/cobra"
)

var (
	verifyFix        bool
	verifyDeep       bool
	verifyQuiet      bool
	verifyReport     string
	verifySignatures bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: i18n.T("cmd.verify.short"),
	Long:  i18n.T("cmd.verify.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		verifyStart := time.Now()

		if !verifyQuiet {
			fmt.Println(i18n.T("cmd.verify.start"))
			fmt.Println(strings.Repeat("=", 50))
		}

		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.verify.errLoadConfig"), err)
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
			fmt.Println("\n" + i18n.T("cmd.verify.fix.start"))
			fixResult := attemptAutoFix(cfg, result)
			showFixResults(fixResult)
		}

		// Generate report if requested
		if verifyReport != "" {
			if err := generateVerificationReport(result, verifyReport); err != nil {
				fmt.Printf("%s\n", i18n.T("cmd.verify.report.fail", map[string]interface{}{"error": err}))
			} else {
				fmt.Printf("%s\n", i18n.T("cmd.verify.report.saved", map[string]interface{}{"path": verifyReport}))
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
	RepositoryName string              `json:"repository_name"`
	VerifiedAt     time.Time           `json:"verified_at"`
	TotalFiles     int                 `json:"total_files"`
	ValidFiles     int                 `json:"valid_files"`
	Issues         []VerificationIssue `json:"issues"`
	Statistics     VerificationStats   `json:"statistics"`
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
		fmt.Printf("%s\n", i18n.T("cmd.verify.repo", map[string]interface{}{"name": cfg.Repository.Name}))
	}

	// Check 1: Configuration integrity
	if !verifyQuiet {
		fmt.Print(i18n.T("cmd.verify.check.config"))
	}
	configIssues := checkConfigurationIntegrity(cfg)
	result.Issues = append(result.Issues, configIssues...)
	if !verifyQuiet {
		if len(configIssues) == 0 {
			fmt.Println("✅")
		} else {
			fmt.Printf(i18n.T("cmd.verify.check.failCount")+"\n", len(configIssues))
		}
	}

	// Check 2: Directory structure
	if !verifyQuiet {
		fmt.Print(i18n.T("cmd.verify.check.dir"))
	}
	dirIssues := checkDirectoryStructure(cfg)
	result.Issues = append(result.Issues, dirIssues...)
	if !verifyQuiet {
		if len(dirIssues) == 0 {
			fmt.Println("✅")
		} else {
			fmt.Printf(i18n.T("cmd.verify.check.failCount")+"\n", len(dirIssues))
		}
	}

	// Check 3: Manifest integrity
	if !verifyQuiet {
		fmt.Print(i18n.T("cmd.verify.check.manifest"))
	}
	manifestIssues, manifest := checkManifestIntegrity(cfg)
	result.Issues = append(result.Issues, manifestIssues...)
	if !verifyQuiet {
		if len(manifestIssues) == 0 {
			fmt.Println("✅")
		} else {
			fmt.Printf(i18n.T("cmd.verify.check.failCount")+"\n", len(manifestIssues))
		}
	}

	if manifest != nil {
		result.TotalFiles = countManifestAPKs(manifest)

		// Check 4: APK files
		if !verifyQuiet {
			fmt.Print(i18n.T("cmd.verify.check.apks"))
		}
		apkIssues, validCount := checkAPKFiles(cfg, manifest)
		result.Issues = append(result.Issues, apkIssues...)
		result.ValidFiles = validCount
		if !verifyQuiet {
			if len(apkIssues) == 0 {
				fmt.Println("✅")
			} else {
				fmt.Printf(i18n.T("cmd.verify.check.failCount")+"\n", len(apkIssues))
			}
		}

		// Check 5: Orphaned files
		if !verifyQuiet {
			fmt.Print(i18n.T("cmd.verify.check.orphans"))
		}
		orphanIssues := checkOrphanedFiles(cfg, manifest)
		result.Issues = append(result.Issues, orphanIssues...)
		if !verifyQuiet {
			if len(orphanIssues) == 0 {
				fmt.Println("✅")
			} else {
				fmt.Printf(i18n.T("cmd.verify.check.failCount")+"\n", len(orphanIssues))
			}
		}

		// Deep verification if requested
		if verifyDeep {
			if !verifyQuiet {
				fmt.Print(i18n.T("cmd.verify.check.deep"))
			}
			deepIssues := performDeepVerification(cfg, manifest)
			result.Issues = append(result.Issues, deepIssues...)
			if !verifyQuiet {
				if len(deepIssues) == 0 {
					fmt.Println("✅")
				} else {
					fmt.Printf(i18n.T("cmd.verify.check.failCount")+"\n", len(deepIssues))
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
			Description: i18n.T("cmd.verify.issue.emptyName"),
			Fixable:     false,
		})
	}

	// Check if basic repository info is configured
	if cfg.Repository.Description == "" {
		issues = append(issues, VerificationIssue{
			Type:        "config",
			Severity:    "warning",
			Description: i18n.T("cmd.verify.issue.emptyDesc"),
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
				Description: i18n.T("cmd.verify.issue.missingDir", map[string]interface{}{"dir": dir}),
				File:        dir,
				Fixable:     true,
			})
		}
	}

	return issues
}

// checkManifestIntegrity checks manifest file integrity
func checkManifestIntegrity(cfg *models.Config) ([]VerificationIssue, *models.ManifestIndex) {
	var issues []VerificationIssue

	// Check if manifest file exists
	if _, err := os.Stat("apkhub_manifest.json"); os.IsNotExist(err) {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "error",
			Description: i18n.T("cmd.verify.issue.manifestMissing"),
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
			Description: i18n.T("cmd.verify.issue.manifestRead", map[string]interface{}{"error": err}),
			File:        "apkhub_manifest.json",
			Fixable:     false,
		})
		return issues, nil
	}

	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "error",
			Description: i18n.T("cmd.verify.issue.manifestParse", map[string]interface{}{"error": err}),
			File:        "apkhub_manifest.json",
			Fixable:     false,
		})
		return issues, nil
	}

	// Check manifest consistency
	if manifest.Name != "" && manifest.Name != cfg.Repository.Name {
		issues = append(issues, VerificationIssue{
			Type:        "manifest",
			Severity:    "warning",
			Description: i18n.T("cmd.verify.issue.nameMismatch"),
			File:        "apkhub_manifest.json",
			Fixable:     true,
		})
	}

	if verifySignatures {
		if issue := validateManifestSignature(&manifest, cfg); issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues, &manifest
}

// checkAPKFiles checks APK file integrity
func checkAPKFiles(cfg *models.Config, manifest *models.ManifestIndex) ([]VerificationIssue, int) {
	var issues []VerificationIssue
	validCount := 0

	if manifest == nil {
		return issues, validCount
	}

	strictPolicy := strings.ToLower(cfg.Repository.SignaturePolicy) == "strict"

	for pkgID, pkg := range manifest.Packages {
		if pkg == nil {
			continue
		}

		for versionKey, version := range pkg.Versions {
			if version == nil {
				continue
			}

			localPath, hasLocalPath := resolveLocalAPKPath(version.DownloadURL)
			if !hasLocalPath {
				continue
			}

			if _, err := os.Stat(localPath); err != nil {
				issues = append(issues, VerificationIssue{
					Type:     "apk",
					Severity: "error",
					Description: i18n.T("cmd.verify.issue.apkMissing", map[string]interface{}{
						"id": pkgID, "version": versionKey,
					}),
					File:    localPath,
					Fixable: false,
				})
				continue
			}

			if verifySignatures {
				if sigIssue := validateAPKSignature(pkgID, versionKey, version, cfg, strictPolicy); sigIssue != nil {
					issues = append(issues, *sigIssue)
				}

				expectedHash := strings.TrimSpace(version.SHA256)
				if expectedHash == "" {
					severity := "warning"
					if strictPolicy {
						severity = "error"
					}
					issues = append(issues, VerificationIssue{
						Type:     "manifest",
						Severity: severity,
						Description: i18n.T("cmd.verify.issue.missingSHA", map[string]interface{}{
							"id": pkgID, "version": versionKey,
						}),
						File:    "apkhub_manifest.json",
						Fixable: true,
					})
					continue
				}

				actualHash, err := calculateFileSHA256(localPath)
				if err != nil {
					issues = append(issues, VerificationIssue{
						Type:     "apk",
						Severity: "error",
						Description: i18n.T("cmd.verify.issue.hashFail", map[string]interface{}{
							"id": pkgID, "version": versionKey, "error": err,
						}),
						File:    localPath,
						Fixable: false,
					})
					continue
				}

				if actualHash != expectedHash {
					issues = append(issues, VerificationIssue{
						Type:     "apk",
						Severity: "error",
						Description: i18n.T("cmd.verify.issue.hashMismatch", map[string]interface{}{
							"id": pkgID, "version": versionKey, "expected": expectedHash, "actual": actualHash,
						}),
						File:    localPath,
						Fixable: false,
					})
					continue
				}
			}

			validCount++
		}
	}

	return issues, validCount
}

// checkOrphanedFiles checks for files not referenced in manifest
func checkOrphanedFiles(cfg *models.Config, manifest *models.ManifestIndex) []VerificationIssue {
	var issues []VerificationIssue

	if manifest == nil {
		return issues
	}

	// For now, we'll do a basic check for APK files against the reported total
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
		expected := countManifestAPKs(manifest)
		if expected > 0 && apkCount != expected {
			issues = append(issues, VerificationIssue{
				Type:     "consistency",
				Severity: "warning",
				Description: i18n.T("cmd.verify.issue.apkCountMismatch", map[string]interface{}{
					"found": apkCount, "expected": expected,
				}),
				File:    "apks/",
				Fixable: false,
			})
		}
	}

	return issues
}

// performDeepVerification performs additional deep checks
func performDeepVerification(cfg *models.Config, manifest *models.ManifestIndex) []VerificationIssue {
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
			Description: i18n.T("cmd.verify.issue.iconsMissing"),
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
					Description: i18n.T("cmd.verify.issue.iconsEmpty"),
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
			Description: i18n.T("cmd.verify.issue.infosMissing"),
			File:        infoDir,
			Fixable:     true,
		})
	}

	return issues
}

func validateManifestSignature(manifest *models.ManifestIndex, cfg *models.Config) *VerificationIssue {
	policyStrict := strings.ToLower(cfg.Repository.SignaturePolicy) == "strict"

	if manifest.Signature == nil || manifest.Signature.PublicKeyFingerprint == "" {
		if !verifySignatures {
			return nil
		}

		severity := "warning"
		if policyStrict {
			severity = "error"
		}

		return &VerificationIssue{
			Type:        "manifest",
			Severity:    severity,
			Description: i18n.T("cmd.verify.issue.manifestNoSignature"),
			File:        "apkhub_manifest.json",
			Fixable:     true,
		}
	}

	if len(cfg.Repository.TrustedKeys) > 0 && !containsString(cfg.Repository.TrustedKeys, manifest.Signature.PublicKeyFingerprint) {
		severity := "warning"
		if policyStrict {
			severity = "error"
		}

		return &VerificationIssue{
			Type:     "manifest",
			Severity: severity,
			Description: i18n.T("cmd.verify.issue.manifestSignerUntrusted", map[string]interface{}{
				"fingerprint": manifest.Signature.PublicKeyFingerprint,
			}),
			File:    "apkhub_manifest.json",
			Fixable: true,
		}
	}

	if manifest.Signature.SignedAt.IsZero() && verifySignatures {
		return &VerificationIssue{
			Type:        "manifest",
			Severity:    "warning",
			Description: i18n.T("cmd.verify.issue.manifestSignedAtMissing"),
			File:        "apkhub_manifest.json",
			Fixable:     true,
		}
	}

	return nil
}

func validateAPKSignature(pkgID, versionKey string, version *models.AppVersion, cfg *models.Config, strict bool) *VerificationIssue {
	if version.SignatureInfo == nil || version.SignatureInfo.SHA256 == "" {
		severity := "warning"
		if strict {
			severity = "error"
		}

		return &VerificationIssue{
			Type:     "signature",
			Severity: severity,
			Description: i18n.T("cmd.verify.issue.apkSignatureMissing", map[string]interface{}{
				"id": pkgID, "version": versionKey,
			}),
			File:    "apkhub_manifest.json",
			Fixable: true,
		}
	}

	if len(cfg.Repository.TrustedKeys) > 0 && !containsString(cfg.Repository.TrustedKeys, version.SignatureInfo.SHA256) {
		severity := "warning"
		if strict {
			severity = "error"
		}

		return &VerificationIssue{
			Type:     "signature",
			Severity: severity,
			Description: i18n.T("cmd.verify.issue.apkSignerUntrusted", map[string]interface{}{
				"fingerprint": version.SignatureInfo.SHA256, "id": pkgID, "version": versionKey,
			}),
			File:    "apkhub_manifest.json",
			Fixable: true,
		}
	}

	return nil
}

func resolveLocalAPKPath(downloadURL string) (string, bool) {
	if downloadURL == "" {
		return "", false
	}

	lowered := strings.ToLower(downloadURL)
	if strings.HasPrefix(lowered, "http://") || strings.HasPrefix(lowered, "https://") {
		return "", false
	}

	if strings.HasPrefix(lowered, "file://") {
		return strings.TrimPrefix(downloadURL, "file://"), true
	}

	return filepath.Clean(strings.TrimLeft(downloadURL, "/")), true
}

func calculateFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func countManifestAPKs(manifest *models.ManifestIndex) int {
	if manifest == nil {
		return 0
	}

	if manifest.TotalAPKs > 0 {
		return manifest.TotalAPKs
	}

	count := 0
	for _, pkg := range manifest.Packages {
		if pkg == nil {
			continue
		}
		count += len(pkg.Versions)
	}
	return count
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
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
		case "signature":
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
	fmt.Println(i18n.T("cmd.verify.results.title"))
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("%s\n", i18n.T("cmd.verify.results.time", map[string]interface{}{"duration": duration}))
	fmt.Printf("%s\n", i18n.T("cmd.verify.results.total", map[string]interface{}{"count": result.TotalFiles}))
	fmt.Printf("%s\n", i18n.T("cmd.verify.results.valid", map[string]interface{}{"count": result.ValidFiles}))
	fmt.Printf("%s\n", i18n.T("cmd.verify.results.issues", map[string]interface{}{"count": len(result.Issues)}))

	if len(result.Issues) > 0 {
		fmt.Println()
		fmt.Println(i18n.T("cmd.verify.results.breakdown"))
		fmt.Printf("%s\n", i18n.T("cmd.verify.results.missing", map[string]interface{}{"count": result.Statistics.MissingFiles}))
		fmt.Printf("%s\n", i18n.T("cmd.verify.results.corrupted", map[string]interface{}{"count": result.Statistics.CorruptedFiles}))
		fmt.Printf("%s\n", i18n.T("cmd.verify.results.orphaned", map[string]interface{}{"count": result.Statistics.OrphanedFiles}))
		fmt.Printf("%s\n", i18n.T("cmd.verify.results.invalid", map[string]interface{}{"count": result.Statistics.InvalidMetadata}))
		fmt.Printf("%s\n", i18n.T("cmd.verify.results.icons", map[string]interface{}{"count": result.Statistics.MissingIcons}))

		fmt.Println()
		fmt.Println(i18n.T("cmd.verify.results.details"))
		for i, issue := range result.Issues {
			severity := "❌"
			if issue.Severity == "warning" {
				severity = "⚠️ "
			}

			fixable := ""
			if issue.Fixable {
				fixable = " " + i18n.T("cmd.verify.results.fixable")
			}

			fmt.Printf("%s\n", i18n.T("cmd.verify.results.item", map[string]interface{}{
				"index": i + 1, "severity": severity, "desc": issue.Description, "fixable": fixable,
			}))
			if issue.File != "" {
				fmt.Printf("%s\n", i18n.T("cmd.verify.results.file", map[string]interface{}{"file": issue.File}))
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))

	if len(result.Issues) == 0 {
		fmt.Println(i18n.T("cmd.verify.results.pass"))
	} else {
		fixableCount := 0
		for _, issue := range result.Issues {
			if issue.Fixable {
				fixableCount++
			}
		}

		if fixableCount > 0 {
			fmt.Printf("%s\n", i18n.T("cmd.verify.results.fixableCount", map[string]interface{}{"count": fixableCount}))
		}

		fmt.Println(i18n.T("cmd.verify.results.attention"))
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
				fixResult.Failed = append(fixResult.Failed, fmt.Sprintf(i18n.T("cmd.verify.fix.errCreateDir"), issue.File, err))
			} else {
				fixResult.Fixed = append(fixResult.Fixed, fmt.Sprintf(i18n.T("cmd.verify.fix.createdDir"), issue.File))
			}

		case "orphan":
			// Remove orphaned files (with confirmation)
			fmt.Printf(i18n.T("cmd.verify.fix.confirmOrphan"), issue.File)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) == "y" {
				orphanPath := filepath.Join("apks", issue.File)
				if err := os.Remove(orphanPath); err != nil {
					fixResult.Failed = append(fixResult.Failed, fmt.Sprintf(i18n.T("cmd.verify.fix.errRemove"), issue.File, err))
				} else {
					fixResult.Fixed = append(fixResult.Fixed, fmt.Sprintf(i18n.T("cmd.verify.fix.removedOrphan"), issue.File))
				}
			} else {
				fixResult.Skipped = append(fixResult.Skipped, fmt.Sprintf(i18n.T("cmd.verify.fix.skipOrphan"), issue.File))
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
	fmt.Println(i18n.T("cmd.verify.fix.title"))
	fmt.Println(strings.Repeat("=", 50))

	if len(result.Fixed) > 0 {
		fmt.Printf(i18n.T("cmd.verify.fix.fixed")+"\n", len(result.Fixed))
		for _, fix := range result.Fixed {
			fmt.Printf("   • %s\n", fix)
		}
	}

	if len(result.Failed) > 0 {
		fmt.Printf("\n"+i18n.T("cmd.verify.fix.failed")+"\n", len(result.Failed))
		for _, fail := range result.Failed {
			fmt.Printf("   • %s\n", fail)
		}
	}

	if len(result.Skipped) > 0 {
		fmt.Printf("\n"+i18n.T("cmd.verify.fix.skipped")+"\n", len(result.Skipped))
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

	verifyCmd.Flags().BoolVar(&verifyFix, "fix", false, i18n.T("cmd.verify.flag.fix"))
	verifyCmd.Flags().BoolVar(&verifyDeep, "deep", false, i18n.T("cmd.verify.flag.deep"))
	verifyCmd.Flags().BoolVar(&verifyQuiet, "quiet", false, i18n.T("cmd.verify.flag.quiet"))
	verifyCmd.Flags().StringVar(&verifyReport, "report", "", i18n.T("cmd.verify.flag.report"))
	verifyCmd.Flags().BoolVar(&verifySignatures, "verify-signature", true, i18n.T("cmd.verify.flag.verifySignatures"))
}
