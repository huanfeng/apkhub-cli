package apk

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apkhub/apkhub-cli/pkg/models"
)

// XAPKParser handles XAPK/APKM file parsing
type XAPKParser struct {
	workDir string
}

// NewXAPKParser creates a new XAPK parser
func NewXAPKParser(workDir string) *XAPKParser {
	return &XAPKParser{
		workDir: workDir,
	}
}

// XAPKInfo contains information about XAPK/APKM file
type XAPKInfo struct {
	*APKInfo
	IsXAPK        bool
	TotalSize     int64
	APKFiles      []string
	OBBFiles      []string
	ExpansionAPKs []string
}

// XAPKManifest represents the manifest.json in XAPK files
type XAPKManifest struct {
	PackageName      string   `json:"package_name"`
	Name             string   `json:"name"`
	VersionCode      int64    `json:"version_code"`
	VersionName      string   `json:"version_name"`
	MinSDKVersion    int      `json:"min_sdk_version"`
	TargetSDKVersion int      `json:"target_sdk_version"`
	APKsConfig       []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"split_apks"`
	ExpansionFiles []struct {
		Path     string `json:"install_path"`
		FileSize int64  `json:"file_size"`
	} `json:"expansions"`
}

// ParseXAPK parses an XAPK/APKM file
func (p *XAPKParser) ParseXAPK(xapkPath string) (*XAPKInfo, error) {
	// Open XAPK as zip file
	reader, err := zip.OpenReader(xapkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XAPK file: %w", err)
	}
	defer reader.Close()
	
	// Get file info
	fileInfo, err := os.Stat(xapkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat XAPK file: %w", err)
	}
	
	xapkInfo := &XAPKInfo{
		IsXAPK:    true,
		TotalSize: fileInfo.Size(),
		APKFiles:  []string{},
		OBBFiles:  []string{},
		ExpansionAPKs: []string{},
	}
	
	// Look for manifest files
	var manifestData []byte
	var baseAPKData io.ReadCloser
	
	for _, file := range reader.File {
		fileName := filepath.Base(file.Name)
		
		// Check for manifest files
		switch fileName {
		case "manifest.json", "info.json":
			rc, err := file.Open()
			if err != nil {
				continue
			}
			manifestData, _ = io.ReadAll(rc)
			rc.Close()
		}
		
		// Track APK files
		if strings.HasSuffix(file.Name, ".apk") {
			xapkInfo.APKFiles = append(xapkInfo.APKFiles, file.Name)
			
			// Find base APK
			if strings.Contains(file.Name, "base.apk") || file.Name == "base.apk" {
				baseAPKData, _ = file.Open()
			} else if baseAPKData == nil && !strings.Contains(file.Name, "config.") {
				// Use first non-config APK as base
				baseAPKData, _ = file.Open()
			}
		}
		
		// Track OBB files
		if strings.HasSuffix(file.Name, ".obb") {
			xapkInfo.OBBFiles = append(xapkInfo.OBBFiles, file.Name)
		}
	}
	
	// Parse manifest if found
	var manifest *XAPKManifest
	if manifestData != nil {
		manifest = &XAPKManifest{}
		json.Unmarshal(manifestData, manifest)
	}
	
	// Extract base APK info
	if baseAPKData != nil {
		defer baseAPKData.Close()
		
		// Create temporary file
		tempFile, err := os.CreateTemp("", "xapk_base_*.apk")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()
		
		// Copy base APK to temp file
		if _, err := io.Copy(tempFile, baseAPKData); err != nil {
			return nil, fmt.Errorf("failed to extract base APK: %w", err)
		}
		tempFile.Close()
		
		// Parse base APK
		parser := NewParser(p.workDir)
		apkInfo, err := parser.ParseAPK(tempFile.Name())
		if err != nil {
			// Try to use manifest info if APK parsing fails
			if manifest != nil {
				apkInfo = p.createAPKInfoFromManifest(manifest, xapkPath)
			} else {
				return nil, fmt.Errorf("failed to parse base APK: %w", err)
			}
		}
		
		xapkInfo.APKInfo = apkInfo
		
		// Override with manifest info if available
		if manifest != nil {
			if manifest.PackageName != "" {
				xapkInfo.PackageID = manifest.PackageName
			}
			if manifest.Name != "" {
				xapkInfo.AppName = map[string]string{"default": manifest.Name}
			}
			if manifest.VersionName != "" {
				xapkInfo.Version = manifest.VersionName
			}
			if manifest.VersionCode > 0 {
				xapkInfo.VersionCode = manifest.VersionCode
			}
			if manifest.MinSDKVersion > 0 {
				xapkInfo.MinSDK = manifest.MinSDKVersion
			}
			if manifest.TargetSDKVersion > 0 {
				xapkInfo.TargetSDK = manifest.TargetSDKVersion
			}
		}
	} else if manifest != nil {
		// No base APK found, use manifest info
		xapkInfo.APKInfo = p.createAPKInfoFromManifest(manifest, xapkPath)
	} else {
		return nil, fmt.Errorf("no base APK or manifest found in XAPK")
	}
	
	// Use XAPK total size
	xapkInfo.Size = xapkInfo.TotalSize
	
	// Update file path
	xapkInfo.FilePath = filepath.Base(xapkPath)
	
	return xapkInfo, nil
}

// createAPKInfoFromManifest creates APKInfo from XAPK manifest
func (p *XAPKParser) createAPKInfoFromManifest(manifest *XAPKManifest, xapkPath string) *APKInfo {
	// Calculate hash of XAPK file
	hashes, _ := p.calculateHashes(xapkPath)
	
	return &APKInfo{
		PackageID:   manifest.PackageName,
		AppName:     map[string]string{"default": manifest.Name},
		Version:     manifest.VersionName,
		VersionCode: manifest.VersionCode,
		MinSDK:      manifest.MinSDKVersion,
		TargetSDK:   manifest.TargetSDKVersion,
		SHA256:      hashes["sha256"],
		SignatureInfo: &models.SignatureInfo{}, // Empty for XAPK
		FilePath:    filepath.Base(xapkPath),
		ABIs:        p.extractABIsFromManifest(manifest),
	}
}

// extractABIsFromManifest extracts ABIs from XAPK manifest
func (p *XAPKParser) extractABIsFromManifest(manifest *XAPKManifest) []string {
	abiMap := make(map[string]bool)
	
	for _, apk := range manifest.APKsConfig {
		// Extract ABI from config APK names like "config.arm64_v8a.apk"
		if strings.HasPrefix(apk.Path, "config.") && strings.HasSuffix(apk.Path, ".apk") {
			abi := strings.TrimPrefix(apk.Path, "config.")
			abi = strings.TrimSuffix(abi, ".apk")
			abi = strings.ReplaceAll(abi, "_", "-")
			abiMap[abi] = true
		}
	}
	
	var abis []string
	for abi := range abiMap {
		abis = append(abis, abi)
	}
	
	return abis
}

// IsXAPKFile checks if the file is an XAPK or APKM file
func IsXAPKFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".xapk" || ext == ".apkm"
}

// ExtractXAPK extracts XAPK/APKM contents to a directory
func (p *XAPKParser) ExtractXAPK(xapkPath string, destDir string) error {
	reader, err := zip.OpenReader(xapkPath)
	if err != nil {
		return fmt.Errorf("failed to open XAPK: %w", err)
	}
	defer reader.Close()
	
	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Extract files
	for _, file := range reader.File {
		destPath := filepath.Join(destDir, file.Name)
		
		// Create directory if needed
		if file.FileInfo().IsDir() {
			os.MkdirAll(destPath, file.Mode())
			continue
		}
		
		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		
		// Extract file
		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		
		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()
		
		if _, err := io.Copy(outFile, rc); err != nil {
			return err
		}
	}
	
	return nil
}

// calculateHashes calculates the SHA256 hash of a file
func (p *XAPKParser) calculateHashes(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	
	return map[string]string{
		"sha256": hex.EncodeToString(hash.Sum(nil)),
	}, nil
}