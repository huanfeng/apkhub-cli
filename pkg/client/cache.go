package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanfeng/apkhub-cli/pkg/models"
)

// CacheManager handles caching operations
type CacheManager struct {
	config    *Config
	cacheDir  string
	defaultTTL time.Duration
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key        string      `json:"key"`
	Data       interface{} `json:"data"`
	CreatedAt  time.Time   `json:"created_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
	Size       int64       `json:"size"`
	AccessCount int        `json:"access_count"`
	LastAccess time.Time   `json:"last_access"`
}

// CacheStats contains cache statistics
type CacheStats struct {
	TotalEntries   int           `json:"total_entries"`
	TotalSize      int64         `json:"total_size"`
	HitRate        float64       `json:"hit_rate"`
	MissRate       float64       `json:"miss_rate"`
	OldestEntry    time.Time     `json:"oldest_entry"`
	NewestEntry    time.Time     `json:"newest_entry"`
	ExpiredEntries int           `json:"expired_entries"`
	CacheDir       string        `json:"cache_dir"`
	DefaultTTL     time.Duration `json:"default_ttl"`
}

// NewCacheManager creates a new cache manager
func NewCacheManager(config *Config) *CacheManager {
	cacheDir := config.Client.CacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "apkhub-cache")
	}

	defaultTTL := time.Duration(config.Client.CacheTTL) * time.Second
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour // Default 24 hours
	}

	return &CacheManager{
		config:     config,
		cacheDir:   cacheDir,
		defaultTTL: defaultTTL,
	}
}

// Get retrieves an item from cache
func (c *CacheManager) Get(key string, target interface{}) (bool, error) {
	cachePath := c.getCachePath(key)
	
	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return false, nil
	}

	// Load cache entry
	entry, err := c.loadCacheEntry(cachePath)
	if err != nil {
		return false, err
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		os.Remove(cachePath)
		return false, nil
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()
	c.saveCacheEntry(cachePath, entry) // Best effort, ignore errors

	// Unmarshal data to target
	data, err := json.Marshal(entry.Data)
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return false, err
	}

	return true, nil
}

// Set stores an item in cache
func (c *CacheManager) Set(key string, data interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	entry := &CacheEntry{
		Key:        key,
		Data:       data,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
		LastAccess: time.Now(),
	}

	cachePath := c.getCachePath(key)
	return c.saveCacheEntry(cachePath, entry)
}

// Delete removes an item from cache
func (c *CacheManager) Delete(key string) error {
	cachePath := c.getCachePath(key)
	return os.Remove(cachePath)
}

// Clear removes all cache entries
func (c *CacheManager) Clear() error {
	if _, err := os.Stat(c.cacheDir); os.IsNotExist(err) {
		return nil // Nothing to clear
	}

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}

	var errors []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".cache") {
			path := filepath.Join(c.cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to remove some cache files: %s", strings.Join(errors, ", "))
	}

	return nil
}

// CleanExpired removes expired cache entries
func (c *CacheManager) CleanExpired() (int, error) {
	if _, err := os.Stat(c.cacheDir); os.IsNotExist(err) {
		return 0, nil
	}

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return 0, err
	}

	removed := 0
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".cache") {
			cachePath := filepath.Join(c.cacheDir, entry.Name())
			
			cacheEntry, err := c.loadCacheEntry(cachePath)
			if err != nil {
				// Remove corrupted cache files
				os.Remove(cachePath)
				removed++
				continue
			}

			if now.After(cacheEntry.ExpiresAt) {
				os.Remove(cachePath)
				removed++
			}
		}
	}

	return removed, nil
}

// GetStats returns cache statistics
func (c *CacheManager) GetStats() (*CacheStats, error) {
	stats := &CacheStats{
		CacheDir:   c.cacheDir,
		DefaultTTL: c.defaultTTL,
	}

	if _, err := os.Stat(c.cacheDir); os.IsNotExist(err) {
		return stats, nil
	}

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var oldestTime, newestTime time.Time
	totalHits := 0
	totalAccess := 0

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".cache") {
			cachePath := filepath.Join(c.cacheDir, entry.Name())
			
			cacheEntry, err := c.loadCacheEntry(cachePath)
			if err != nil {
				continue
			}

			stats.TotalEntries++
			
			// Get file size
			if info, err := entry.Info(); err == nil {
				stats.TotalSize += info.Size()
			}

			// Check if expired
			if now.After(cacheEntry.ExpiresAt) {
				stats.ExpiredEntries++
			}

			// Track oldest and newest
			if oldestTime.IsZero() || cacheEntry.CreatedAt.Before(oldestTime) {
				oldestTime = cacheEntry.CreatedAt
			}
			if newestTime.IsZero() || cacheEntry.CreatedAt.After(newestTime) {
				newestTime = cacheEntry.CreatedAt
			}

			// Calculate hit rate
			totalHits += cacheEntry.AccessCount
			totalAccess += cacheEntry.AccessCount + 1 // +1 for initial creation
		}
	}

	stats.OldestEntry = oldestTime
	stats.NewestEntry = newestTime

	if totalAccess > 0 {
		stats.HitRate = float64(totalHits) / float64(totalAccess)
		stats.MissRate = 1.0 - stats.HitRate
	}

	return stats, nil
}

// GetManifest retrieves a cached manifest with fallback to stale cache
func (c *CacheManager) GetManifest(bucketName string, allowStale bool) (*models.ManifestIndex, bool, error) {
	key := fmt.Sprintf("manifest_%s", bucketName)
	
	var manifest models.ManifestIndex
	found, err := c.Get(key, &manifest)
	if err != nil {
		return nil, false, err
	}

	if found {
		return &manifest, false, nil // Fresh cache
	}

	// Try stale cache if allowed
	if allowStale {
		cachePath := c.getCachePath(key)
		if entry, err := c.loadCacheEntry(cachePath); err == nil {
			// Return stale data
			data, err := json.Marshal(entry.Data)
			if err == nil {
				if err := json.Unmarshal(data, &manifest); err == nil {
					return &manifest, true, nil // Stale cache
				}
			}
		}
	}

	return nil, false, nil // No cache found
}

// SetManifest stores a manifest in cache
func (c *CacheManager) SetManifest(bucketName string, manifest *models.ManifestIndex, ttl time.Duration) error {
	key := fmt.Sprintf("manifest_%s", bucketName)
	return c.Set(key, manifest, ttl)
}

// getCachePath generates cache file path for a key
func (c *CacheManager) getCachePath(key string) string {
	// Sanitize key for filename
	safeKey := strings.ReplaceAll(key, "/", "_")
	safeKey = strings.ReplaceAll(safeKey, "\\", "_")
	safeKey = strings.ReplaceAll(safeKey, ":", "_")
	
	return filepath.Join(c.cacheDir, safeKey+".cache")
}

// loadCacheEntry loads a cache entry from file
func (c *CacheManager) loadCacheEntry(cachePath string) (*CacheEntry, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// saveCacheEntry saves a cache entry to file
func (c *CacheManager) saveCacheEntry(cachePath string, entry *CacheEntry) error {
	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// PrintStats prints formatted cache statistics
func (c *CacheManager) PrintStats() error {
	stats, err := c.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("💾 Cache Statistics:")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("   Directory: %s\n", stats.CacheDir)
	fmt.Printf("   Total entries: %d\n", stats.TotalEntries)
	fmt.Printf("   Total size: %s\n", formatBytes(stats.TotalSize))
	fmt.Printf("   Default TTL: %v\n", stats.DefaultTTL)
	
	if stats.TotalEntries > 0 {
		fmt.Printf("   Hit rate: %.1f%%\n", stats.HitRate*100)
		fmt.Printf("   Miss rate: %.1f%%\n", stats.MissRate*100)
		fmt.Printf("   Expired entries: %d\n", stats.ExpiredEntries)
		
		if !stats.OldestEntry.IsZero() {
			fmt.Printf("   Oldest entry: %s\n", stats.OldestEntry.Format("2006-01-02 15:04:05"))
		}
		if !stats.NewestEntry.IsZero() {
			fmt.Printf("   Newest entry: %s\n", stats.NewestEntry.Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}

// formatBytes formats bytes in human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}