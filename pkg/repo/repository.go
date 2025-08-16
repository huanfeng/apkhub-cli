package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/huanfeng/apkhub-cli/pkg/models"
)

// Repository manages the APK repository structure
type Repository struct {
	layout  *models.RepositoryLayout
	config  *models.Config
	parser  *apk.Parser
	rootDir string
}

// NewRepository creates a new repository instance
func NewRepository(rootDir string, config *models.Config) (*Repository, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root directory: %w", err)
	}

	return &Repository{
		layout:  models.NewRepositoryLayout(absRoot),
		config:  config,
		parser:  apk.NewParser(absRoot),
		rootDir: absRoot,
	}, nil
}

// Initialize creates the repository directory structure
func (r *Repository) Initialize() error {
	// Create directories
	dirs := []string{
		r.layout.RootDir,
		filepath.Join(r.layout.RootDir, r.layout.APKsDir),
		filepath.Join(r.layout.RootDir, r.layout.InfosDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create initial manifest if not exists
	manifestPath := filepath.Join(r.layout.RootDir, r.layout.ManifestFile)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifest := &models.ManifestIndex{
			Version:     "1.0",
			Name:        r.config.Repository.Name,
			Description: r.config.Repository.Description,
			UpdatedAt:   time.Now(),
			Packages:    make(map[string]*models.AppPackage),
		}

		if err := r.saveManifest(manifest); err != nil {
			return fmt.Errorf("failed to create initial manifest: %w", err)
		}
	}

	return nil
}

// GenerateNormalizedFileName generates a normalized filename for an APK
func (r *Repository) GenerateNormalizedFileName(info *apk.APKInfo) string {
	// Format: packageid_versioncode_signature[8]_variant.ext
	// Example: com.example.app_100_a1b2c3d4.apk

	filename := fmt.Sprintf("%s_%d", info.PackageID, info.VersionCode)

	// Add signature prefix if available
	if info.SignatureInfo != nil && info.SignatureInfo.SHA256 != "" {
		sigPrefix := info.SignatureInfo.SHA256[:8]
		filename = fmt.Sprintf("%s_%s", filename, sigPrefix)
	}

	// Add variant if needed (for different architectures or densities)
	if len(info.ABIs) > 0 {
		// Use first ABI as variant indicator
		abi := strings.ReplaceAll(info.ABIs[0], "-", "")
		filename = fmt.Sprintf("%s_%s", filename, abi)
	}

	// Check for XAPK features
	for _, feature := range info.Features {
		if feature == "split_apk" || feature == "has_obb" {
			// Check original file extension
			if strings.HasSuffix(strings.ToLower(info.FilePath), ".xapk") {
				return filename + ".xapk"
			} else if strings.HasSuffix(strings.ToLower(info.FilePath), ".apkm") {
				return filename + ".apkm"
			}
		}
	}

	return filename + ".apk"
}

// SaveAPKInfo saves individual APK information to infos directory
func (r *Repository) SaveAPKInfo(apkInfo *models.APKInfo) error {
	// Create info filename based on package ID only (without version)
	infoFileName := fmt.Sprintf("%s.json", apkInfo.PackageID)

	infoPath := filepath.Join(r.layout.RootDir, r.layout.InfosDir, infoFileName)
	apkInfo.InfoPath = filepath.Join(r.layout.InfosDir, infoFileName)

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(apkInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal APK info: %w", err)
	}

	// Write to file
	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write APK info: %w", err)
	}

	return nil
}

// SaveAPKInfoWithIcon saves APK info and icon from parsed APK data
func (r *Repository) SaveAPKInfoWithIcon(parsedInfo *apk.APKInfo, apkInfo *models.APKInfo) error {
	// Save APK info
	if err := r.SaveAPKInfo(apkInfo); err != nil {
		return err
	}

	// Save icon if available
	if parsedInfo.IconData != nil && len(parsedInfo.IconData) > 0 {
		iconFileName := fmt.Sprintf("%s%s", apkInfo.PackageID, parsedInfo.IconExt)
		iconPath := filepath.Join(r.layout.RootDir, r.layout.InfosDir, iconFileName)

		if err := os.WriteFile(iconPath, parsedInfo.IconData, 0644); err != nil {
			// Icon save failure is non-fatal, just log warning
			fmt.Printf("Warning: Failed to save icon for %s: %v\n", apkInfo.PackageID, err)
		} else {
			// Update icon path in APK info
			apkInfo.IconPath = filepath.Join(r.layout.InfosDir, iconFileName)
		}
	}

	return nil
}

// LoadAPKInfo loads APK information from infos directory
func (r *Repository) LoadAPKInfo(infoPath string) (*models.APKInfo, error) {
	fullPath := filepath.Join(r.layout.RootDir, infoPath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read APK info: %w", err)
	}

	var apkInfo models.APKInfo
	if err := json.Unmarshal(data, &apkInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal APK info: %w", err)
	}

	return &apkInfo, nil
}

// LoadAllAPKInfos loads all APK information files from infos directory
func (r *Repository) LoadAllAPKInfos() ([]*models.APKInfo, error) {
	infosDir := filepath.Join(r.layout.RootDir, r.layout.InfosDir)

	entries, err := os.ReadDir(infosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.APKInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read infos directory: %w", err)
	}

	var infos []*models.APKInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := r.LoadAPKInfo(filepath.Join(r.layout.InfosDir, entry.Name()))
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to load %s: %v\n", entry.Name(), err)
			continue
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// BuildManifestFromInfos builds the manifest index from individual APK info files
func (r *Repository) BuildManifestFromInfos() (*models.ManifestIndex, error) {
	infos, err := r.LoadAllAPKInfos()
	if err != nil {
		return nil, fmt.Errorf("failed to load APK infos: %w", err)
	}

	manifest := &models.ManifestIndex{
		Version:     "1.0",
		Name:        r.config.Repository.Name,
		Description: r.config.Repository.Description,
		UpdatedAt:   time.Now(),
		Packages:    make(map[string]*models.AppPackage),
	}

	var totalSize int64

	// Group APKs by package ID
	for _, info := range infos {
		totalSize += info.Size

		// Get or create package entry
		pkg, exists := manifest.Packages[info.PackageID]
		if !exists {
			pkg = &models.AppPackage{
				PackageID: info.PackageID,
				Name:      info.AppName,
				Versions:  make(map[string]*models.AppVersion),
			}
			manifest.Packages[info.PackageID] = pkg
		}

		// Create version entry
		version := &models.AppVersion{
			Version:       info.Version,
			VersionCode:   info.VersionCode,
			MinSDK:        info.MinSDK,
			TargetSDK:     info.TargetSDK,
			Size:          info.Size,
			SHA256:        info.SHA256,
			SignatureInfo: info.SignatureInfo,
			DownloadURL:   r.buildDownloadURL(info.FilePath),
			ReleaseDate:   info.AddedAt,
			Permissions:   info.Permissions,
			Features:      info.Features,
			ABIs:          info.ABIs,
		}

		// Use version string as key, but handle duplicates
		versionKey := info.Version
		if _, exists := pkg.Versions[versionKey]; exists {
			// Add signature suffix for duplicates
			if info.SignatureInfo != nil && info.SignatureInfo.SHA256 != "" {
				versionKey = fmt.Sprintf("%s_%s", info.Version, info.SignatureInfo.SHA256[:8])
			}
		}

		pkg.Versions[versionKey] = version

		// Update latest version
		r.updateLatestVersion(pkg)
	}

	manifest.TotalAPKs = len(infos)
	manifest.TotalSize = totalSize

	return manifest, nil
}

// buildDownloadURL builds the download URL for an APK
func (r *Repository) buildDownloadURL(filePath string) string {
	// Convert backslashes to forward slashes for URLs
	urlPath := strings.ReplaceAll(filePath, "\\", "/")

	if r.config.Repository.BaseURL != "" {
		baseURL := r.config.Repository.BaseURL
		
		// Handle local mode with special rules
		if r.isLocalMode(baseURL) {
			return r.buildLocalDownloadURL(baseURL, urlPath)
		}
		
		// Handle remote URLs
		baseURL = strings.TrimRight(baseURL, "/")
		urlPath = strings.TrimLeft(urlPath, "/")
		return fmt.Sprintf("%s/%s", baseURL, urlPath)
	}

	// Return relative path
	return urlPath
}

// isLocalMode checks if the base URL indicates local mode
func (r *Repository) isLocalMode(baseURL string) bool {
	return strings.HasPrefix(baseURL, "http://localhost") || 
		   strings.HasPrefix(baseURL, "http://127.0.0.1") ||
		   strings.HasPrefix(baseURL, "file://") ||
		   baseURL == "local"
}

// buildLocalDownloadURL builds download URL for local mode based on bucket path rules
func (r *Repository) buildLocalDownloadURL(baseURL, urlPath string) string {
	// If baseURL is "local", use file:// protocol with absolute path
	if baseURL == "local" {
		absPath := filepath.Join(r.rootDir, urlPath)
		return "file://" + absPath
	}
	
	// If it's already a file:// URL, resolve relative to the repository root
	if strings.HasPrefix(baseURL, "file://") {
		basePath := strings.TrimPrefix(baseURL, "file://")
		absPath := filepath.Join(basePath, urlPath)
		return "file://" + absPath
	}
	
	// For localhost URLs, keep the original behavior but ensure proper path joining
	baseURL = strings.TrimRight(baseURL, "/")
	urlPath = strings.TrimLeft(urlPath, "/")
	return fmt.Sprintf("%s/%s", baseURL, urlPath)
}

// updateLatestVersion updates the latest version field for a package
func (r *Repository) updateLatestVersion(pkg *models.AppPackage) {
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

// SaveManifest saves the manifest index to file
func (r *Repository) saveManifest(manifest *models.ManifestIndex) error {
	manifestPath := filepath.Join(r.layout.RootDir, r.layout.ManifestFile)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// UpdateManifest rebuilds and saves the manifest from current APK info files
func (r *Repository) UpdateManifest() error {
	manifest, err := r.BuildManifestFromInfos()
	if err != nil {
		return fmt.Errorf("failed to build manifest: %w", err)
	}

	if err := r.saveManifest(manifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	return nil
}

// GetAPKPath returns the full path for an APK in the repository
func (r *Repository) GetAPKPath(filename string) string {
	return filepath.Join(r.layout.RootDir, r.layout.APKsDir, filename)
}

// GetRootDir returns the repository root directory
func (r *Repository) GetRootDir() string {
	return r.rootDir
}
