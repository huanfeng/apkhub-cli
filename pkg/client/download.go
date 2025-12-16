package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanfeng/apkhub/pkg/models"
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
	Version      string
	Force        bool // Force re-download even if exists
	NoVerify     bool // Skip checksum verification
	OutputPath   string
	MaxRetries   int
	Timeout      int // Timeout in seconds
	ShowProgress bool
}

// ProgressWriter wraps an io.Writer to report download progress
type ProgressWriter struct {
	writer       io.Writer
	total        int64
	written      int64
	lastUpdate   time.Time
	startTime    time.Time
	showProgress bool
}

// NewProgressWriter creates a new progress writer
func NewProgressWriter(writer io.Writer, total int64, showProgress bool) *ProgressWriter {
	return &ProgressWriter{
		writer:       writer,
		total:        total,
		startTime:    time.Now(),
		lastUpdate:   time.Now(),
		showProgress: showProgress,
	}
}

// Write implements io.Writer interface with progress reporting
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.written += int64(n)

	// Update progress every 500ms
	if pw.showProgress && time.Since(pw.lastUpdate) > 500*time.Millisecond {
		pw.updateProgress()
		pw.lastUpdate = time.Now()
	}

	return n, err
}

// updateProgress displays current download progress
func (pw *ProgressWriter) updateProgress() {
	if pw.total <= 0 {
		return
	}

	percentage := float64(pw.written) / float64(pw.total) * 100
	elapsed := time.Since(pw.startTime)

	// Calculate speed and ETA
	speed := float64(pw.written) / elapsed.Seconds()
	remaining := pw.total - pw.written
	eta := time.Duration(float64(remaining)/speed) * time.Second

	// Format sizes
	writtenMB := float64(pw.written) / (1024 * 1024)
	totalMB := float64(pw.total) / (1024 * 1024)
	speedMB := speed / (1024 * 1024)

	// Progress bar
	barWidth := 30
	filled := int(percentage * float64(barWidth) / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Printf("\r📥 [%s] %.1f%% (%.1f/%.1f MB) %.1f MB/s ETA: %v",
		bar, percentage, writtenMB, totalMB, speedMB, eta.Round(time.Second))
}

// Finish completes the progress display
func (pw *ProgressWriter) Finish() {
	if pw.showProgress {
		elapsed := time.Since(pw.startTime)
		totalMB := float64(pw.written) / (1024 * 1024)
		avgSpeed := float64(pw.written) / elapsed.Seconds() / (1024 * 1024)

		fmt.Printf("\r✅ Download completed: %.1f MB in %v (avg: %.1f MB/s)\n",
			totalMB, elapsed.Round(time.Second), avgSpeed)
	}
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
			if !options.NoVerify && version.SHA256 != "" {
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

	// Download with progress and retry
	fmt.Printf("📥 Downloading %s v%s...\n", packageID, version.Version)
	if options.ShowProgress {
		fmt.Printf("   URL: %s\n", downloadURL)
		fmt.Printf("   Size: %.2f MB\n", float64(version.Size)/(1024*1024))
		fmt.Println()
	}

	if err := d.downloadFileWithRetry(downloadURL, targetPath, version.Size, options); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	// Verify checksum
	if !options.NoVerify && version.SHA256 != "" {
		fmt.Print("🔍 Verifying checksum... ")
		if ok, err := d.verifyChecksum(targetPath, version.SHA256); !ok {
			os.Remove(targetPath)
			return "", fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("✅ OK")
	}

	fmt.Printf("✓ Downloaded to: %s\n", targetPath)
	return targetPath, nil
}

// downloadFileWithRetry downloads a file with retry mechanism and progress reporting
func (d *DownloadManager) downloadFileWithRetry(url, targetPath string, expectedSize int64, options DownloadOptions) error {
	// Check if this is a local file
	if strings.HasPrefix(url, "file://") {
		return d.copyLocalFile(url, targetPath, expectedSize, options.ShowProgress)
	}

	// Remote file download with retry
	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	timeout := time.Duration(options.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

			fmt.Printf("🔄 Retrying download in %v (attempt %d/%d)...\n", delay, attempt+1, maxRetries+1)
			time.Sleep(delay)
		}

		err := d.downloadFile(url, targetPath, expectedSize, timeout, options.ShowProgress)
		if err == nil {
			return nil
		}

		lastErr = err
		fmt.Printf("❌ Download attempt %d failed: %v\n", attempt+1, err)

		// Clean up partial file
		os.Remove(targetPath + ".tmp")
	}

	return fmt.Errorf("download failed after %d attempts: %w", maxRetries+1, lastErr)
}

// copyLocalFile copies a local file with progress reporting
func (d *DownloadManager) copyLocalFile(fileURL, targetPath string, expectedSize int64, showProgress bool) error {
	// Convert file:// URL to local path
	sourcePath := strings.TrimPrefix(fileURL, "file://")

	// Check if source file exists
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source file not accessible: %w", err)
	}

	if sourceInfo.IsDir() {
		return fmt.Errorf("source is a directory, not a file: %s", sourcePath)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	// Copy with progress
	fileSize := sourceInfo.Size()
	if expectedSize > 0 && fileSize != expectedSize {
		fmt.Printf("⚠️  Size mismatch: expected %d bytes, source has %d bytes\n", expectedSize, fileSize)
	}

	if showProgress {
		fmt.Printf("📁 Copying local file: %s\n", sourcePath)
	}

	progressWriter := NewProgressWriter(targetFile, fileSize, showProgress)
	written, err := io.Copy(progressWriter, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	progressWriter.Finish()

	if written != fileSize {
		return fmt.Errorf("incomplete copy: expected %d bytes, copied %d bytes", fileSize, written)
	}

	return nil
}

// downloadFile downloads a file with progress reporting
func (d *DownloadManager) downloadFile(url, targetPath string, expectedSize int64, timeout time.Duration, showProgress bool) error {
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

	// Create request with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		os.Remove(tempPath)
		return err
	}

	req.Header.Set("User-Agent", "ApkHub-CLI/1.0")

	// Download
	resp, err := d.client.Do(req)
	if err != nil {
		os.Remove(tempPath)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tempPath)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Get content length
	contentLength := resp.ContentLength
	if contentLength <= 0 && expectedSize > 0 {
		contentLength = expectedSize
	}

	// Create progress writer
	progressWriter := NewProgressWriter(out, contentLength, showProgress)

	// Copy with progress
	written, err := io.Copy(progressWriter, resp.Body)
	if err != nil {
		os.Remove(tempPath)
		return err
	}

	// Finish progress display
	progressWriter.Finish()

	// Close file before rename
	out.Close()

	// Verify size if expected
	if expectedSize > 0 && written != expectedSize {
		os.Remove(tempPath)
		return fmt.Errorf("size mismatch: expected %d bytes, got %d bytes", expectedSize, written)
	}

	// Rename temp to final
	if err := os.Rename(tempPath, targetPath); err != nil {
		os.Remove(tempPath)
		return err
	}

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
