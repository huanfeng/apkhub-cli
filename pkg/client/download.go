package client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apkhub/apkhub-cli/pkg/models"
)

// DownloadManager handles APK downloads
type DownloadManager struct {
	config    *Config
	bucketMgr *BucketManager
	client    *http.Client
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(config *Config, bucketMgr *BucketManager) *DownloadManager {
	return &DownloadManager{
		config:    config,
		bucketMgr: bucketMgr,
		client: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large APKs
		},
	}
}

// DownloadOptions contains download options
type DownloadOptions struct {
	Version   string
	Force     bool // Force re-download even if exists
	NoCVerify bool // Skip checksum verification
}

// Download downloads an APK by package ID
func (d *DownloadManager) Download(packageID string, options DownloadOptions) (string, error) {
	// Get merged manifest
	manifest, err := d.bucketMgr.GetMergedManifest()
	if err != nil {
		return "", fmt.Errorf("failed to get manifest: %w", err)
	}
	
	// Find package
	pkg, exists := manifest.Packages[packageID]
	if !exists {
		return "", fmt.Errorf("package '%s' not found", packageID)
	}
	
	// Select version
	var version *models.AppVersion
	
	if options.Version != "" {
		// Find specific version
		for _, ver := range pkg.Versions {
			if ver.Version == options.Version || ver.VersionCode == parseVersionCode(options.Version) {
				version = ver
				break
			}
		}
		if version == nil {
			return "", fmt.Errorf("version '%s' not found for package '%s'", options.Version, packageID)
		}
	} else {
		// Use latest version
		if pkg.Latest == "" {
			return "", fmt.Errorf("no versions available for package '%s'", packageID)
		}
		version = pkg.Versions[pkg.Latest]
	}
	
	// Construct filename
	filename := fmt.Sprintf("%s_%d.apk", packageID, version.VersionCode)
	targetPath := filepath.Join(d.config.Client.DownloadDir, filename)
	
	// Check if already downloaded
	if !options.Force {
		if info, err := os.Stat(targetPath); err == nil {
			// Verify checksum if file exists
			if !options.NoCVerify && version.SHA256 != "" {
				if ok, _ := d.verifyChecksum(targetPath, version.SHA256); ok {
					fmt.Printf("✓ Already downloaded: %s\n", targetPath)
					return targetPath, nil
				}
				fmt.Println("Checksum mismatch, re-downloading...")
			} else {
				fmt.Printf("✓ Already downloaded: %s (size: %.2f MB)\n", 
					targetPath, float64(info.Size())/(1024*1024))
				return targetPath, nil
			}
		}
	}
	
	// Download URL
	downloadURL := version.DownloadURL
	if downloadURL == "" {
		return "", fmt.Errorf("no download URL for version %s", version.Version)
	}
	
	// Download with progress
	fmt.Printf("Downloading %s v%s...\n", packageID, version.Version)
	fmt.Printf("URL: %s\n", downloadURL)
	fmt.Printf("Size: %.2f MB\n", float64(version.Size)/(1024*1024))
	
	if err := d.downloadFile(downloadURL, targetPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	
	// Verify checksum
	if !options.NoCVerify && version.SHA256 != "" {
		fmt.Print("Verifying checksum... ")
		if ok, err := d.verifyChecksum(targetPath, version.SHA256); !ok {
			os.Remove(targetPath)
			return "", fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("✓ OK")
	}
	
	fmt.Printf("✓ Downloaded to: %s\n", targetPath)
	return targetPath, nil
}

// downloadFile downloads a file with progress reporting
func (d *DownloadManager) downloadFile(url, targetPath string) error {
	// Ensure download directory exists
	downloadDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}
	
	// Create temp file
	tempPath := targetPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Download
	resp, err := d.client.Get(url)
	if err != nil {
		os.Remove(tempPath)
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		os.Remove(tempPath)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Copy with progress
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tempPath)
		return err
	}
	
	// Close file before rename
	out.Close()
	
	// Rename temp to final
	if err := os.Rename(tempPath, targetPath); err != nil {
		os.Remove(tempPath)
		return err
	}
	
	fmt.Printf("Downloaded %d bytes\n", written)
	return nil
}

// verifyChecksum verifies file checksum
func (d *DownloadManager) verifyChecksum(filePath, expectedSHA256 string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}
	
	actualSHA256 := hex.EncodeToString(hash.Sum(nil))
	return actualSHA256 == expectedSHA256, nil
}

// parseVersionCode parses version code from string
func parseVersionCode(s string) int64 {
	var code int64
	fmt.Sscanf(s, "%d", &code)
	return code
}

// GetPackageInfo retrieves detailed package information
func (d *DownloadManager) GetPackageInfo(packageID string) (*models.AppPackage, error) {
	// Get merged manifest
	manifest, err := d.bucketMgr.GetMergedManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	
	// Find package
	pkg, exists := manifest.Packages[packageID]
	if !exists {
		// Try case-insensitive search
		for pkgID, p := range manifest.Packages {
			if strings.EqualFold(pkgID, packageID) {
				return p, nil
			}
		}
		return nil, fmt.Errorf("package '%s' not found", packageID)
	}
	
	return pkg, nil
}