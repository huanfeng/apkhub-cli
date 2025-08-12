package apk

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/shogo82148/androidbinary/apk"
)

// AndroidBinaryParser wraps the androidbinary library
type AndroidBinaryParser struct {
	workDir string
}

// NewAndroidBinaryParser creates a new AndroidBinary parser
func NewAndroidBinaryParser(workDir string) *AndroidBinaryParser {
	return &AndroidBinaryParser{
		workDir: workDir,
	}
}

// ParseAPK parses APK using androidbinary library
func (p *AndroidBinaryParser) ParseAPK(apkPath string) (*APKInfo, error) {
	// Open APK file
	pkg, err := apk.OpenFile(apkPath)
	if err != nil {
		return nil, err
	}
	defer pkg.Close()

	// Get file info
	fileInfo, err := os.Stat(apkPath)
	if err != nil {
		return nil, err
	}

	// Parse manifest
	manifest := pkg.Manifest()

	// Calculate hashes
	hashes, err := p.calculateHashes(apkPath)
	if err != nil {
		return nil, err
	}

	// Extract signature info
	signatureInfo, err := p.extractSignatureInfo(pkg)
	if err != nil {
		// Non-fatal error, continue without signature info
		signatureInfo = nil
	}

	// Extract icon
	iconExtractor := NewIconExtractor()
	iconData, iconExt, iconErr := iconExtractor.ExtractIcon(apkPath)
	// Icon extraction is non-fatal

	// Build APK info
	info := &APKInfo{
		PackageID:     manifest.Package.MustString(),
		AppName:       p.extractAppName(&manifest),
		Version:       manifest.VersionName.MustString(),
		VersionCode:   int64(manifest.VersionCode.MustInt32()),
		MinSDK:        p.extractMinSDK(&manifest),
		TargetSDK:     p.extractTargetSDK(&manifest),
		Size:          fileInfo.Size(),
		SHA256:        hashes["sha256"],
		SignatureInfo: signatureInfo,
		Permissions:   p.extractPermissions(&manifest),
		Features:      p.extractFeatures(&manifest),
		ABIs:          p.extractABIs(apkPath),
		ReleaseDate:   fileInfo.ModTime(),
	}

	// Add icon data if extraction was successful
	if iconErr == nil && iconData != nil {
		info.IconData = iconData
		info.IconExt = iconExt
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

// GetParserInfo returns information about this parser
func (p *AndroidBinaryParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:         "AndroidBinary",
		Version:      "1.0.5", // Version of the library we're using
		Capabilities: []string{"apk", "manifest", "resources", "icons"},
		Available:    true, // Always available as it's built-in
		Priority:     1,    // Highest priority
	}
}

// CanParse checks if this parser can handle the given file
func (p *AndroidBinaryParser) CanParse(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".apk"
}

// Helper methods (copied from original parser.go)
func (p *AndroidBinaryParser) calculateHashes(filePath string) (map[string]string, error) {
	// Implementation copied from original parser
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

func (p *AndroidBinaryParser) extractAppName(manifest *apk.Manifest) map[string]string {
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

func (p *AndroidBinaryParser) extractMinSDK(manifest *apk.Manifest) int {
	if minSDK, err := manifest.SDK.Min.Int32(); err == nil {
		return int(minSDK)
	}
	return 1 // Default minimum
}

func (p *AndroidBinaryParser) extractTargetSDK(manifest *apk.Manifest) int {
	if targetSDK, err := manifest.SDK.Target.Int32(); err == nil {
		return int(targetSDK)
	}
	return 0
}

func (p *AndroidBinaryParser) extractPermissions(manifest *apk.Manifest) []string {
	var permissions []string
	for _, perm := range manifest.UsesPermissions {
		if permName, err := perm.Name.String(); err == nil && permName != "" {
			permissions = append(permissions, permName)
		}
	}
	return permissions
}

func (p *AndroidBinaryParser) extractFeatures(manifest *apk.Manifest) []string {
	// Note: The current Manifest struct doesn't have UsesFeatures field
	// This would need to be extracted from the raw XML if needed
	return []string{}
}

func (p *AndroidBinaryParser) extractABIs(apkPath string) []string {
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

func (p *AndroidBinaryParser) extractSignatureInfo(pkg *apk.Apk) (*models.SignatureInfo, error) {
	// For now, return empty signature info
	// TODO: Implement proper signature extraction
	return &models.SignatureInfo{}, nil
}