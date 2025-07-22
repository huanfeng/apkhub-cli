package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apkhub/apkhub-cli/pkg/apk"
	"github.com/apkhub/apkhub-cli/pkg/models"
)

// Scanner handles directory scanning for APK files
type Scanner struct {
	config  *models.Config
	parser  *apk.Parser
	workDir string
}

// NewScanner creates a new scanner instance
func NewScanner(config *models.Config, workDir string) *Scanner {
	return &Scanner{
		config:  config,
		parser:  apk.NewParser(workDir),
		workDir: workDir,
	}
}

// ScanResult represents the result of scanning
type ScanResult struct {
	Index      *models.PackageIndex
	TotalFiles int
	ParsedAPKs int
	Errors     []error
}

// Scan scans the directory for APK files
func (s *Scanner) Scan(directory string) (*ScanResult, error) {
	result := &ScanResult{
		Index: &models.PackageIndex{
			Version:     "1.0",
			Name:        s.config.Repository.Name,
			Description: s.config.Repository.Description,
			UpdatedAt:   time.Now(),
			Packages:    make(map[string]*models.AppPackage),
		},
		Errors: []error{},
	}

	// Walk through directory
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error accessing %s: %w", path, err))
			return nil // Continue scanning
		}

		// Skip directories
		if info.IsDir() {
			// Check if we should skip this directory
			if !s.config.Scanning.Recursive && path != directory {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks if not following
		if info.Mode()&os.ModeSymlink != 0 && !s.config.Scanning.FollowSymlinks {
			return nil
		}

		// Check if file matches patterns
		if !s.matchesPattern(path) {
			return nil
		}

		result.TotalFiles++

		// Parse APK if enabled
		if s.config.Scanning.ParseAPKInfo {
			if err := s.processAPK(path, result); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("error processing %s: %w", path, err))
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return result, nil
}

// matchesPattern checks if file matches include/exclude patterns
func (s *Scanner) matchesPattern(path string) bool {
	filename := filepath.Base(path)

	// Check exclude patterns first
	for _, pattern := range s.config.Scanning.ExcludePattern {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return false
		}
		// Also check against full relative path
		if matched, _ := filepath.Match(pattern, path); matched {
			return false
		}
	}

	// Check include patterns
	for _, pattern := range s.config.Scanning.IncludePattern {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}

	return false
}

// processAPK processes a single APK file
func (s *Scanner) processAPK(apkPath string, result *ScanResult) error {
	apkInfo, err := s.parser.ParseAPK(apkPath)
	if err != nil {
		return err
	}

	result.ParsedAPKs++

	// Get or create package entry
	pkg, exists := result.Index.Packages[apkInfo.PackageID]
	if !exists {
		pkg = &models.AppPackage{
			PackageID:   apkInfo.PackageID,
			Name:        apkInfo.AppName,
			Versions:    make(map[string]*models.AppVersion),
		}
		result.Index.Packages[apkInfo.PackageID] = pkg
	}

	// Create version entry
	version := &models.AppVersion{
		Version:       apkInfo.Version,
		VersionCode:   apkInfo.VersionCode,
		MinSDK:        apkInfo.MinSDK,
		TargetSDK:     apkInfo.TargetSDK,
		Size:          apkInfo.Size,
		SHA256:        apkInfo.SHA256,
		SignatureInfo: apkInfo.SignatureInfo,
		DownloadURL:   s.buildDownloadURL(apkInfo.FilePath),
		ReleaseDate:   apkInfo.ReleaseDate,
		Permissions:   apkInfo.Permissions,
		Features:      apkInfo.Features,
		ABIs:          apkInfo.ABIs,
	}

	// Handle version with same version string but different signature
	existingVersion, versionExists := pkg.Versions[apkInfo.Version]
	if versionExists && s.config.Repository.SignatureHandling != "reject" {
		// Check if signatures differ
		if existingVersion.SignatureInfo != nil && apkInfo.SignatureInfo != nil &&
			existingVersion.SignatureInfo.SHA256 != apkInfo.SignatureInfo.SHA256 {
			
			switch s.config.Repository.SignatureHandling {
			case "mark":
				// Add signature variant suffix
				version.SignatureVariant = "alt-sig"
				versionKey := fmt.Sprintf("%s-altsig-%s", apkInfo.Version, apkInfo.SignatureInfo.SHA256[:8])
				pkg.Versions[versionKey] = version
			case "separate":
				// Create separate entry with signature hash suffix
				versionKey := fmt.Sprintf("%s-%s", apkInfo.Version, apkInfo.SignatureInfo.SHA256[:8])
				pkg.Versions[versionKey] = version
			}
		} else {
			// Same signature or no signature info, update if newer
			if version.ReleaseDate.After(existingVersion.ReleaseDate) {
				pkg.Versions[apkInfo.Version] = version
			}
		}
	} else {
		// No existing version or reject mode
		pkg.Versions[apkInfo.Version] = version
	}

	// Update latest version
	s.updateLatestVersion(pkg)

	return nil
}

// buildDownloadURL builds the download URL for an APK
func (s *Scanner) buildDownloadURL(filePath string) string {
	// Convert backslashes to forward slashes for URLs
	urlPath := strings.ReplaceAll(filePath, "\\", "/")
	
	if s.config.Repository.BaseURL != "" {
		// Ensure base URL doesn't end with slash and path doesn't start with slash
		baseURL := strings.TrimRight(s.config.Repository.BaseURL, "/")
		urlPath = strings.TrimLeft(urlPath, "/")
		return fmt.Sprintf("%s/%s", baseURL, urlPath)
	}
	
	// Return relative path
	return urlPath
}

// updateLatestVersion updates the latest version field for a package
func (s *Scanner) updateLatestVersion(pkg *models.AppPackage) {
	var latestVersion *models.AppVersion
	var latestVersionKey string

	for versionKey, version := range pkg.Versions {
		// Skip alternative signature versions when determining latest
		if version.SignatureVariant != "" {
			continue
		}
		
		if latestVersion == nil || version.VersionCode > latestVersion.VersionCode {
			latestVersion = version
			latestVersionKey = versionKey
		}
	}

	if latestVersionKey != "" {
		pkg.Latest = latestVersionKey
	}
}