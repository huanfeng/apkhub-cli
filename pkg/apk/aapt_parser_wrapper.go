package apk

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/huanfeng/apkhub-cli/pkg/models"
)

// AAPTParserWrapper wraps the AAPT parser for the parser chain
type AAPTParserWrapper struct {
	parser  *AAPTParser
	workDir string
}

// NewAAPTParserWrapper creates a new AAPT parser wrapper
func NewAAPTParserWrapper(workDir string) *AAPTParserWrapper {
	return &AAPTParserWrapper{
		parser:  NewAAPTParser(),
		workDir: workDir,
	}
}

// ParseAPK parses APK using AAPT
func (p *AAPTParserWrapper) ParseAPK(apkPath string) (*APKInfo, error) {
	// Parse with AAPT
	basicInfo, err := p.parser.ParseAPKWithAAPT(apkPath)
	if err != nil {
		return nil, err
	}

	// Get file info
	fileInfo, err := os.Stat(apkPath)
	if err != nil {
		return nil, err
	}

	// Calculate hashes
	hashes, err := p.calculateHashes(apkPath)
	if err != nil {
		return nil, err
	}

	// Extract icon
	iconExtractor := NewIconExtractor()
	iconData, iconExt, iconErr := iconExtractor.ExtractIcon(apkPath)
	// Icon extraction is non-fatal

	// Build APK info from aapt data
	info := &APKInfo{
		PackageID:     basicInfo.PackageID,
		AppName:       map[string]string{"default": basicInfo.AppName},
		Version:       basicInfo.VersionName,
		VersionCode:   basicInfo.VersionCode,
		MinSDK:        basicInfo.MinSDK,
		TargetSDK:     basicInfo.TargetSDK,
		Size:          fileInfo.Size(),
		SHA256:        hashes["sha256"],
		SignatureInfo: &models.SignatureInfo{}, // Empty signature for aapt
		Permissions:   basicInfo.Permissions,
		Features:      basicInfo.Features,
		ABIs:          basicInfo.ABIs,
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
func (p *AAPTParserWrapper) GetParserInfo() ParserInfo {
	// Check if AAPT is available
	available := p.parser.CheckAAPT() == nil

	capabilities := []string{"apk", "manifest", "permissions", "features"}
	if available {
		capabilities = append(capabilities, "native_abis", "densities")
	}

	return ParserInfo{
		Name:         "AAPT",
		Version:      "unknown", // We don't easily get AAPT version
		Capabilities: capabilities,
		Available:    available,
		Priority:     2, // Lower priority than AndroidBinary
	}
}

// CanParse checks if this parser can handle the given file
func (p *AAPTParserWrapper) CanParse(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".apk"
}

// calculateHashes calculates file hashes
func (p *AAPTParserWrapper) calculateHashes(filePath string) (map[string]string, error) {
	// Reuse the hash calculation from AndroidBinaryParser
	parser := NewAndroidBinaryParser(p.workDir)
	return parser.calculateHashes(filePath)
}
