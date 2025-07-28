package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/huanfeng/apkhub-cli/pkg/models"
)

// BucketManager manages repository buckets
type BucketManager struct {
	config *Config
}

// NewBucketManager creates a new bucket manager
func NewBucketManager(config *Config) *BucketManager {
	return &BucketManager{
		config: config,
	}
}

// FetchManifest fetches and caches a bucket's manifest
func (b *BucketManager) FetchManifest(bucketName string) (*models.ManifestIndex, error) {
	bucket, exists := b.config.Buckets[bucketName]
	if !exists {
		return nil, fmt.Errorf("bucket %s not found", bucketName)
	}

	if !bucket.Enabled {
		return nil, fmt.Errorf("bucket %s is disabled", bucketName)
	}

	// Check cache
	cachePath := filepath.Join(b.config.Client.CacheDir, bucketName+".json")
	if manifest, err := b.loadCachedManifest(cachePath, b.config.Client.CacheTTL); err == nil {
		return manifest, nil
	}

	// Fetch from remote
	manifestURL := bucket.URL + "/apkhub_manifest.json"
	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from %s: %w", manifestURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	// Read response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// Parse manifest
	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Save to cache
	if err := b.saveToCache(cachePath, data); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to cache manifest: %v\n", err)
	}

	// Update last updated time
	bucket.LastUpdated = time.Now()
	b.config.Save()

	return &manifest, nil
}

// UpdateAll updates all enabled buckets
func (b *BucketManager) UpdateAll() error {
	var errors []error

	for name, bucket := range b.config.GetEnabledBuckets() {
		fmt.Printf("Updating bucket '%s' from %s...\n", name, bucket.URL)
		if _, err := b.FetchManifest(name); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", name, err))
		} else {
			fmt.Printf("✓ Updated %s\n", name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some buckets failed to update: %v", errors)
	}

	return nil
}

// GetMergedManifest returns a merged manifest from all enabled buckets
func (b *BucketManager) GetMergedManifest() (*models.ManifestIndex, error) {
	merged := &models.ManifestIndex{
		Version:     "1.0",
		Name:        "Merged Repository",
		Description: "Combined view of all enabled buckets",
		UpdatedAt:   time.Now(),
		Packages:    make(map[string]*models.AppPackage),
	}

	// Load each bucket's manifest
	for name := range b.config.GetEnabledBuckets() {
		manifest, err := b.FetchManifest(name)
		if err != nil {
			fmt.Printf("Warning: failed to load bucket %s: %v\n", name, err)
			continue
		}

		// Merge packages
		for pkgID, pkg := range manifest.Packages {
			if existing, exists := merged.Packages[pkgID]; exists {
				// Merge versions
				for versionKey, version := range pkg.Versions {
					// Prefix version key with bucket name to avoid conflicts
					prefixedKey := fmt.Sprintf("%s_%s", name, versionKey)
					existing.Versions[prefixedKey] = version
					// Update base URL if needed
					if version.DownloadURL != "" && !isAbsoluteURL(version.DownloadURL) {
						version.DownloadURL = b.config.Buckets[name].URL + "/" + version.DownloadURL
					}
				}
				// Update latest if newer
				if pkg.Latest > existing.Latest {
					existing.Latest = pkg.Latest
				}
			} else {
				// Clone package
				clonedPkg := &models.AppPackage{
					PackageID:   pkg.PackageID,
					Name:        pkg.Name,
					Description: pkg.Description,
					Icon:        pkg.Icon,
					Category:    pkg.Category,
					Latest:      pkg.Latest,
					Versions:    make(map[string]*models.AppVersion),
				}
				// Clone versions with bucket prefix
				for versionKey, version := range pkg.Versions {
					prefixedKey := fmt.Sprintf("%s_%s", name, versionKey)
					clonedVersion := *version // Copy
					// Update base URL if needed
					if clonedVersion.DownloadURL != "" && !isAbsoluteURL(clonedVersion.DownloadURL) {
						clonedVersion.DownloadURL = b.config.Buckets[name].URL + "/" + clonedVersion.DownloadURL
					}
					clonedPkg.Versions[prefixedKey] = &clonedVersion
				}
				merged.Packages[pkgID] = clonedPkg
			}
		}

		merged.TotalAPKs += manifest.TotalAPKs
		merged.TotalSize += manifest.TotalSize
	}

	return merged, nil
}

// loadCachedManifest loads manifest from cache if not expired
func (b *BucketManager) loadCachedManifest(cachePath string, ttl int) (*models.ManifestIndex, error) {
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, err
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > time.Duration(ttl)*time.Second {
		return nil, fmt.Errorf("cache expired")
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	// Parse manifest
	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// saveToCache saves data to cache file
func (b *BucketManager) saveToCache(cachePath string, data []byte) error {
	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// isAbsoluteURL checks if a URL is absolute
func isAbsoluteURL(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")
}
