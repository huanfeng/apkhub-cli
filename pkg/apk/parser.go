package apk

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apkhub/apkhub-cli/pkg/models"
)

// Parser handles APK file parsing
type Parser struct {
	// Add aapt2 path or other dependencies here
}

// NewParser creates a new APK parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseAPK parses an APK file and extracts its information
func (p *Parser) ParseAPK(apkPath string) (*models.AppVersion, error) {
	// Check if file exists
	fileInfo, err := os.Stat(apkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat APK file: %w", err)
	}

	// Calculate SHA256
	sha256Hash, err := p.calculateSHA256(apkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate SHA256: %w", err)
	}

	// TODO: Parse APK using aapt2 or androidbinary library
	// For now, return a placeholder
	version := &models.AppVersion{
		Version:     "1.0.0", // TODO: Extract from APK
		VersionCode: 1,       // TODO: Extract from APK
		Size:        fileInfo.Size(),
		SHA256:      sha256Hash,
		// Other fields to be populated
	}

	return version, nil
}

// IsAPKFile checks if the file is an APK, XAPK, or APKM file
func IsAPKFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".apk", ".xapk", ".apkm":
		return true
	default:
		return false
	}
}

// calculateSHA256 calculates the SHA256 hash of a file
func (p *Parser) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}