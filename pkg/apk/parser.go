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

	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/shogo82148/androidbinary/apk"
)

// Parser handles APK file parsing using a chain of parsers
type Parser struct {
	workDir     string
	parserChain *ParserChain
}

// NewParser creates a new APK parser with parser chain
func NewParser(workDir string) *Parser {
	// Create parser chain with logger
	logger := &SimpleLogger{}
	chain := NewParserChain(logger)
	
	// Add parsers in priority order
	chain.AddParser(NewAndroidBinaryParser(workDir))
	chain.AddParser(NewAAPTParserWrapper(workDir))
	chain.AddParser(NewXAPKParserWrapper(workDir))
	
	return &Parser{
		workDir:     workDir,
		parserChain: chain,
	}
}

// ParseAPK parses an APK file and extracts its information using parser chain
func (p *Parser) ParseAPK(apkPath string) (*APKInfo, error) {
	// Use parser chain to parse APK (handles APK, XAPK, APKM)
	result, err := p.parserChain.ParseAPK(apkPath)
	if err != nil {
		return nil, err
	}

	// Log parsing result
	fmt.Printf("File parsed successfully using %s parser (took %v)\n", result.Parser, result.Duration)
	
	// Show warnings if any
	for _, warning := range result.Warnings {
		fmt.Printf("Warning: %s\n", warning)
	}

	return result.APKInfo, nil
}

// GetParserInfo returns information about available parsers
func (p *Parser) GetParserInfo() []ParserInfo {
	return p.parserChain.GetAvailableParsers()
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
	IconData      []byte // Icon data in PNG format
	IconExt       string // Icon file extension (.png)
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
	fmt.Printf("Warning: androidbinary failed to parse APK: %v\n", originalErr)
	fmt.Printf("Attempting aapt fallback parsing...\n")

	// Try parsing with aapt
	basicInfo, aaptErr := TryParseWithAAPT(apkPath)
	if aaptErr != nil {
		// Check if it's a tool not found error
		if strings.Contains(aaptErr.Error(), "not found") {
			fmt.Printf("Info: %v\n", aaptErr)
			fmt.Printf("Continuing with limited APK information from file analysis...\n")
			
			// Return basic info that we can extract without aapt
			return p.createBasicAPKInfo(apkPath, originalErr)
		}
		return nil, fmt.Errorf("both parsers failed - androidbinary: %v, aapt: %v", originalErr, aaptErr)
	}
	
	fmt.Printf("Success: APK parsed using aapt fallback\n")

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
		SignatureInfo: &models.SignatureInfo{}, // Empty signature for aapt fallback
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

// createBasicAPKInfo creates basic APK info when both parsers fail
func (p *Parser) createBasicAPKInfo(apkPath string, originalErr error) (*APKInfo, error) {
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

	// Extract basic info from filename if possible
	filename := filepath.Base(apkPath)
	packageID := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Try to extract ABIs from APK structure
	abis := p.extractABIs(apkPath)

	// Build minimal APK info
	info := &APKInfo{
		PackageID:     packageID, // Use filename as fallback
		AppName:       map[string]string{"default": packageID},
		Version:       "unknown",
		VersionCode:   0,
		MinSDK:        1,
		TargetSDK:     0,
		Size:          fileInfo.Size(),
		SHA256:        hashes["sha256"],
		SignatureInfo: &models.SignatureInfo{}, // Empty signature info
		Permissions:   []string{},
		Features:      []string{"parsing_limited"}, // Mark as limited parsing
		ABIs:          abis,
		ReleaseDate:   fileInfo.ModTime(),
	}

	// Calculate relative path if within work directory
	relPath, err := filepath.Rel(p.workDir, apkPath)
	if err == nil && !strings.HasPrefix(relPath, "..") {
		info.FilePath = relPath
	} else {
		info.FilePath = filepath.Base(apkPath)
	}

	fmt.Printf("Warning: Created limited APK info due to parsing failures\n")
	return info, nil
}

// parseXAPK handles XAPK/APKM file parsing
func (p *Parser) parseXAPK(xapkPath string) (*APKInfo, error) {
	xapkParser := NewXAPKParser(p.workDir)
	xapkInfo, err := xapkParser.ParseXAPK(xapkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XAPK: %w", err)
	}

	// Convert XAPKInfo to APKInfo
	apkInfo := xapkInfo.APKInfo

	// Add XAPK specific markers
	if len(xapkInfo.APKFiles) > 1 {
		// Add feature to indicate it's a split APK
		apkInfo.Features = append(apkInfo.Features, "split_apk")
	}
	if len(xapkInfo.OBBFiles) > 0 {
		apkInfo.Features = append(apkInfo.Features, "has_obb")
	}

	return apkInfo, nil
}
