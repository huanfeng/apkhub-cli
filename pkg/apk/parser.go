package apk

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/shogo82148/androidbinary/apk"
)

// Parser handles APK file parsing
type Parser struct {
	workDir string
}

// NewParser creates a new APK parser
func NewParser(workDir string) *Parser {
	return &Parser{
		workDir: workDir,
	}
}

// ParseAPK parses an APK file and extracts its information
func (p *Parser) ParseAPK(apkPath string) (*APKInfo, error) {
	// Open APK file
	pkg, err := apk.OpenFile(apkPath)
	if err != nil {
		// Try with aapt as fallback
		return p.parseWithAAPTFallback(apkPath, err)
	}
	defer pkg.Close()

	// Get file info
	fileInfo, err := os.Stat(apkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat APK file: %w", err)
	}

	// Parse manifest
	manifest := pkg.Manifest()

	// Calculate hashes
	hashes, err := p.calculateHashes(apkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hashes: %w", err)
	}

	// Extract signature info
	signatureInfo, err := p.extractSignatureInfo(pkg)
	if err != nil {
		// Non-fatal error, continue without signature info
		signatureInfo = nil
	}

	// Build APK info
	info := &APKInfo{
		PackageID:    manifest.Package.MustString(),
		AppName:      p.extractAppName(&manifest),
		Version:      manifest.VersionName.MustString(),
		VersionCode:  int64(manifest.VersionCode.MustInt32()),
		MinSDK:       p.extractMinSDK(&manifest),
		TargetSDK:    p.extractTargetSDK(&manifest),
		Size:         fileInfo.Size(),
		SHA256:       hashes["sha256"],
		SignatureInfo: signatureInfo,
		Permissions:  p.extractPermissions(&manifest),
		Features:     p.extractFeatures(&manifest),
		ABIs:         p.extractABIs(apkPath),
		ReleaseDate:  fileInfo.ModTime(),
	}

	// Calculate relative path if within work directory
	relPath, err := filepath.Rel(p.workDir, apkPath)
	if err == nil && !strings.HasPrefix(relPath, "..") {
		info.FilePath = relPath
	} else {
		info.FilePath = filepath.Base(apkPath)
	}

	return info, nil
}

// APKInfo contains parsed APK information
type APKInfo struct {
	PackageID     string
	AppName       map[string]string
	Version       string
	VersionCode   int64
	MinSDK        int
	TargetSDK     int
	Size          int64
	SHA256        string
	SignatureInfo *models.SignatureInfo
	Permissions   []string
	Features      []string
	ABIs          []string
	ReleaseDate   time.Time
	FilePath      string
}

// calculateHashes calculates various hashes of the APK file
func (p *Parser) calculateHashes(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	md5Hash := md5.New()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()

	// Create a multi-writer to calculate all hashes in one pass
	multiWriter := io.MultiWriter(md5Hash, sha1Hash, sha256Hash)
	
	if _, err := io.Copy(multiWriter, file); err != nil {
		return nil, err
	}

	return map[string]string{
		"md5":    hex.EncodeToString(md5Hash.Sum(nil)),
		"sha1":   hex.EncodeToString(sha1Hash.Sum(nil)),
		"sha256": hex.EncodeToString(sha256Hash.Sum(nil)),
	}, nil
}

// extractAppName extracts application name with multi-language support
func (p *Parser) extractAppName(manifest *apk.Manifest) map[string]string {
	names := make(map[string]string)
	
	// Try to get the default name
	if labelStr, err := manifest.App.Label.String(); err == nil && labelStr != "" {
		names["default"] = labelStr
	}
	
	// If no label found, use package name as fallback
	if len(names) == 0 {
		names["default"] = manifest.Package.MustString()
	}
	
	return names
}

// extractMinSDK extracts minimum SDK version
func (p *Parser) extractMinSDK(manifest *apk.Manifest) int {
	if minSDK, err := manifest.SDK.Min.Int32(); err == nil {
		return int(minSDK)
	}
	return 1 // Default minimum
}

// extractTargetSDK extracts target SDK version
func (p *Parser) extractTargetSDK(manifest *apk.Manifest) int {
	if targetSDK, err := manifest.SDK.Target.Int32(); err == nil {
		return int(targetSDK)
	}
	return 0
}

// extractPermissions extracts required permissions
func (p *Parser) extractPermissions(manifest *apk.Manifest) []string {
	var permissions []string
	for _, perm := range manifest.UsesPermissions {
		if permName, err := perm.Name.String(); err == nil && permName != "" {
			permissions = append(permissions, permName)
		}
	}
	return permissions
}

// extractFeatures extracts required features
func (p *Parser) extractFeatures(manifest *apk.Manifest) []string {
	// Note: The current Manifest struct doesn't have UsesFeatures field
	// This would need to be extracted from the raw XML if needed
	return []string{}
}

// extractABIs extracts supported ABIs from native libraries
func (p *Parser) extractABIs(apkPath string) []string {
	abiMap := make(map[string]bool)
	
	// Open APK as zip file to access entries
	reader, err := zip.OpenReader(apkPath)
	if err != nil {
		return []string{}
	}
	defer reader.Close()
	
	// Check lib directory
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "lib/") {
			parts := strings.Split(file.Name, "/")
			if len(parts) >= 2 {
				abi := parts[1]
				abiMap[abi] = true
			}
		}
	}
	
	// Convert map to slice
	var abis []string
	for abi := range abiMap {
		abis = append(abis, abi)
	}
	
	return abis
}

// extractSignatureInfo extracts APK signature information
func (p *Parser) extractSignatureInfo(pkg *apk.Apk) (*models.SignatureInfo, error) {
	// For now, return empty signature info
	// TODO: Implement proper signature extraction
	return &models.SignatureInfo{}, nil
}

// IsAPKFile checks if the file is an APK, XAPK, or APKM file
func IsAPKFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".apk", ".xapk", ".apkm":
		return true
	default:
		return false
	}
}

// parseWithAAPTFallback uses aapt command as fallback when androidbinary fails
func (p *Parser) parseWithAAPTFallback(apkPath string, originalErr error) (*APKInfo, error) {
	fmt.Printf("Warning: androidbinary failed to parse APK, trying aapt fallback...\n")
	
	// Try parsing with aapt
	basicInfo, aaptErr := TryParseWithAAPT(apkPath)
	if aaptErr != nil {
		return nil, fmt.Errorf("both parsers failed - androidbinary: %v, aapt: %v", originalErr, aaptErr)
	}
	
	// Get file info
	fileInfo, err := os.Stat(apkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat APK file: %w", err)
	}
	
	// Calculate hashes
	hashes, err := p.calculateHashes(apkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hashes: %w", err)
	}
	
	// Build APK info from aapt data
	info := &APKInfo{
		PackageID:    basicInfo.PackageID,
		AppName:      map[string]string{"default": basicInfo.AppName},
		Version:      basicInfo.VersionName,
		VersionCode:  basicInfo.VersionCode,
		MinSDK:       basicInfo.MinSDK,
		TargetSDK:    basicInfo.TargetSDK,
		Size:         fileInfo.Size(),
		SHA256:       hashes["sha256"],
		SignatureInfo: &models.SignatureInfo{}, // Empty signature for aapt fallback
		Permissions:  basicInfo.Permissions,
		Features:     basicInfo.Features,
		ABIs:         basicInfo.ABIs,
		ReleaseDate:  fileInfo.ModTime(),
	}
	
	// Calculate relative path if within work directory
	relPath, err := filepath.Rel(p.workDir, apkPath)
	if err == nil && !strings.HasPrefix(relPath, "..") {
		info.FilePath = relPath
	} else {
		info.FilePath = filepath.Base(apkPath)
	}
	
	return info, nil
}