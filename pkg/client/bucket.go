package client

import (
	"context"
	"encoding/json"
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

// BucketHealth represents the health status of a bucket
type BucketHealth struct {
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	Status           string    `json:"status"` // "healthy", "degraded", "unhealthy", "unknown"
	LastCheck        time.Time `json:"last_check"`
	LastSuccess      time.Time `json:"last_success"`
	ResponseTime     int64     `json:"response_time_ms"`
	ErrorCount       int       `json:"error_count"`
	LastError        string    `json:"last_error,omitempty"`
	ConsecutiveFails int       `json:"consecutive_fails"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Timeout       time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Timeout:       30 * time.Second,
	}
}

// BucketManager manages repository buckets
type BucketManager struct {
	config          *Config
	retryConfig     RetryConfig
	healthMap       map[string]*BucketHealth
	httpClient      *http.Client
	cacheManager    *CacheManager
	verifySignature bool
}

// NewBucketManager creates a new bucket manager
func NewBucketManager(config *Config) *BucketManager {
	return &BucketManager{
		config:       config,
		retryConfig:  DefaultRetryConfig(),
		healthMap:    make(map[string]*BucketHealth),
		cacheManager: NewCacheManager(config),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		verifySignature: config.Security.VerifySignature,
	}
}

// NewBucketManagerWithRetry creates a bucket manager with custom retry config
func NewBucketManagerWithRetry(config *Config, retryConfig RetryConfig) *BucketManager {
	return &BucketManager{
		config:       config,
		retryConfig:  retryConfig,
		healthMap:    make(map[string]*BucketHealth),
		cacheManager: NewCacheManager(config),
		httpClient: &http.Client{
			Timeout: retryConfig.Timeout,
		},
		verifySignature: config.Security.VerifySignature,
	}
}

// SetSignatureVerification allows callers to override signature verification behavior
func (b *BucketManager) SetSignatureVerification(enabled bool) {
	b.verifySignature = enabled
}

// FetchManifest fetches and caches a bucket's manifest with retry and health monitoring
func (b *BucketManager) FetchManifest(bucketName string) (*models.ManifestIndex, error) {
	bucket, exists := b.config.Buckets[bucketName]
	if !exists {
		return nil, fmt.Errorf("bucket %s not found", bucketName)
	}

	if !bucket.Enabled {
		return nil, fmt.Errorf("bucket %s is disabled", bucketName)
	}

	// Initialize health tracking
	health := b.getOrCreateHealth(bucketName, bucket.URL)

	// Check cache first
	if manifest, isStale, err := b.cacheManager.GetManifest(bucketName, false); err == nil && manifest != nil {
		// Update health status for cache hit
		health.LastCheck = time.Now()
		if health.Status == "unknown" {
			health.Status = "healthy"
		}

		if !isStale {
			return manifest, nil
		}
		// Continue to fetch fresh data but keep stale as fallback
	}

	// Determine if this is a local or remote bucket
	var data []byte
	var err error

	if strings.HasPrefix(bucket.URL, "file://") {
		// Local bucket
		data, err = b.fetchLocalManifest(bucket.URL, bucketName)
	} else {
		// Remote bucket
		manifestURL := bucket.URL + "/apkhub_manifest.json"
		data, err = b.fetchWithRetry(manifestURL, bucketName)
	}

	if err != nil {
		b.updateHealthOnError(bucketName, err)

		// Try to return stale cache if available
		if manifest, _, cacheErr := b.cacheManager.GetManifest(bucketName, true); cacheErr == nil && manifest != nil {
			fmt.Printf("⚠️  Using stale cache for bucket '%s' due to error: %v\n", bucketName, err)
			return manifest, nil
		}

		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	// Parse manifest
	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		b.updateHealthOnError(bucketName, fmt.Errorf("parse error: %w", err))
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if b.verifySignature {
		if err := b.verifyManifestSignature(&manifest); err != nil {
			policy := strings.ToLower(b.config.Security.SignaturePolicy)
			if policy == "" {
				policy = "lenient"
			}

			if policy == "strict" {
				b.updateHealthOnError(bucketName, fmt.Errorf("signature verification failed: %w", err))

				// Try to use stale cache as a safe downgrade path
				if manifest, _, cacheErr := b.cacheManager.GetManifest(bucketName, true); cacheErr == nil && manifest != nil {
					fmt.Printf("⚠️  Using stale cache for bucket '%s' due to signature verification failure: %v\n", bucketName, err)
					return manifest, nil
				}

				return nil, fmt.Errorf("manifest signature verification failed: %w. Add the signer fingerprint to security.trusted_keys or rerun with --verify-signature=false.", err)
			}

			fmt.Printf("⚠️  Manifest signature verification failed: %v (continuing due to lenient policy)\n", err)
		}
	}

	// Save to cache
	cacheTTL := time.Duration(b.config.Client.CacheTTL) * time.Second
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour // Default 24 hours
	}

	if err := b.cacheManager.SetManifest(bucketName, &manifest, cacheTTL); err != nil {
		fmt.Printf("⚠️  Failed to cache manifest for '%s': %v\n", bucketName, err)
	}

	// Update success metrics
	b.updateHealthOnSuccess(bucketName)

	// Update last updated time
	bucket.LastUpdated = time.Now()
	b.config.Save()

	return &manifest, nil
}

// fetchWithRetry performs HTTP request with exponential backoff retry
func (b *BucketManager) fetchWithRetry(url, bucketName string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= b.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(b.retryConfig.InitialDelay) * math.Pow(b.retryConfig.BackoffFactor, float64(attempt-1)))
			if delay > b.retryConfig.MaxDelay {
				delay = b.retryConfig.MaxDelay
			}

			fmt.Printf("🔄 Retrying bucket '%s' in %v (attempt %d/%d)...\n", bucketName, delay, attempt+1, b.retryConfig.MaxRetries+1)
			time.Sleep(delay)
		}

		// Create request with context for timeout
		ctx, cancel := context.WithTimeout(context.Background(), b.retryConfig.Timeout)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}

		// Set user agent
		req.Header.Set("User-Agent", "ApkHub-CLI/1.0")

		// Record start time for response time measurement
		startTime := time.Now()

		// Perform request
		resp, err := b.httpClient.Do(req)
		responseTime := time.Since(startTime)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("network error: %w", err)
			b.updateResponseTime(bucketName, responseTime)
			continue
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			b.updateResponseTime(bucketName, responseTime)

			// Don't retry on client errors (4xx)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				break
			}
			continue
		}

		// Read response
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("read error: %w", err)
			continue
		}

		// Success - update response time
		b.updateResponseTime(bucketName, responseTime)
		return data, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", b.retryConfig.MaxRetries+1, lastErr)
}

// fetchLocalManifest fetches manifest from local file system
func (b *BucketManager) fetchLocalManifest(fileURL, bucketName string) ([]byte, error) {
	// Convert file:// URL to local path
	localPath := strings.TrimPrefix(fileURL, "file://")
	manifestPath := filepath.Join(localPath, "apkhub_manifest.json")

	// Record start time for response time measurement
	startTime := time.Now()

	// Check if manifest file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("manifest file not found: %s", manifestPath)
	}

	// Read manifest file
	data, err := os.ReadFile(manifestPath)
	responseTime := time.Since(startTime)

	// Update response time
	b.updateResponseTime(bucketName, responseTime)

	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	return data, nil
}

// getOrCreateHealth gets or creates health tracking for a bucket
func (b *BucketManager) getOrCreateHealth(name, url string) *BucketHealth {
	if health, exists := b.healthMap[name]; exists {
		return health
	}

	health := &BucketHealth{
		Name:             name,
		URL:              url,
		Status:           "unknown",
		LastCheck:        time.Now(),
		ConsecutiveFails: 0,
	}
	b.healthMap[name] = health
	return health
}

// updateHealthOnSuccess updates health metrics on successful operation
func (b *BucketManager) updateHealthOnSuccess(bucketName string) {
	health := b.healthMap[bucketName]
	if health == nil {
		return
	}

	health.LastCheck = time.Now()
	health.LastSuccess = time.Now()
	health.ConsecutiveFails = 0
	health.LastError = ""

	// Update status based on recent performance
	if health.ErrorCount == 0 {
		health.Status = "healthy"
	} else if health.ErrorCount < 3 {
		health.Status = "degraded"
	} else {
		health.Status = "healthy" // Recovered
		health.ErrorCount = 0     // Reset error count on recovery
	}
}

// updateHealthOnError updates health metrics on error
func (b *BucketManager) updateHealthOnError(bucketName string, err error) {
	health := b.healthMap[bucketName]
	if health == nil {
		return
	}

	health.LastCheck = time.Now()
	health.ErrorCount++
	health.ConsecutiveFails++
	health.LastError = err.Error()

	// Update status based on error count
	if health.ConsecutiveFails >= 5 {
		health.Status = "unhealthy"
	} else if health.ConsecutiveFails >= 2 {
		health.Status = "degraded"
	}
}

// updateResponseTime updates response time metrics
func (b *BucketManager) updateResponseTime(bucketName string, responseTime time.Duration) {
	health := b.healthMap[bucketName]
	if health != nil {
		health.ResponseTime = responseTime.Milliseconds()
	}
}

// loadStaleCache loads cache without TTL check (for fallback)
func (b *BucketManager) loadStaleCache(cachePath string) (*models.ManifestIndex, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// UpdateAll updates all enabled buckets with concurrent processing
func (b *BucketManager) UpdateAll() error {
	enabledBuckets := b.config.GetEnabledBuckets()
	if len(enabledBuckets) == 0 {
		fmt.Println("No enabled buckets to update")
		return nil
	}

	fmt.Printf("🔄 Updating %d bucket(s)...\n", len(enabledBuckets))

	// Use channels for concurrent updates
	type updateResult struct {
		name   string
		bucket *Bucket
		err    error
	}

	results := make(chan updateResult, len(enabledBuckets))

	// Start concurrent updates
	for name, bucket := range enabledBuckets {
		go func(n string, bkt *Bucket) {
			fmt.Printf("📡 Fetching '%s' from %s...\n", n, bkt.URL)
			_, err := b.FetchManifest(n)
			results <- updateResult{name: n, bucket: bkt, err: err}
		}(name, bucket)
	}

	// Collect results
	var errors []error
	var successful []string
	var failed []string

	for i := 0; i < len(enabledBuckets); i++ {
		result := <-results

		if result.err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", result.name, result.err))
			failed = append(failed, result.name)
			fmt.Printf("❌ Failed to update '%s': %v\n", result.name, result.err)
		} else {
			successful = append(successful, result.name)
			fmt.Printf("✅ Updated '%s'\n", result.name)
		}
	}

	// Print summary
	fmt.Printf("\n📊 Update Summary:\n")
	fmt.Printf("   ✅ Successful: %d (%s)\n", len(successful), strings.Join(successful, ", "))
	if len(failed) > 0 {
		fmt.Printf("   ❌ Failed: %d (%s)\n", len(failed), strings.Join(failed, ", "))
	}

	// Show health status
	b.PrintHealthStatus()

	if len(errors) > 0 {
		return fmt.Errorf("some buckets failed to update")
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
				var latestPrefixedKey string
				for versionKey, version := range pkg.Versions {
					// Prefix version key with bucket name to avoid conflicts
					prefixedKey := fmt.Sprintf("%s_%s", name, versionKey)
					existing.Versions[prefixedKey] = version
					// Update base URL if needed
					if version.DownloadURL != "" && !isAbsoluteURL(version.DownloadURL) {
						version.DownloadURL = b.resolveDownloadURL(name, version.DownloadURL)
					}

					// Track the prefixed key for the latest version
					if versionKey == pkg.Latest {
						latestPrefixedKey = prefixedKey
					}
				}
				// Update latest if newer (compare version strings or use the new one if existing is empty)
				if existing.Latest == "" || (latestPrefixedKey != "" && pkg.Latest > existing.Latest) {
					existing.Latest = latestPrefixedKey
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
						clonedVersion.DownloadURL = b.resolveDownloadURL(name, clonedVersion.DownloadURL)
					}
					clonedPkg.Versions[prefixedKey] = &clonedVersion

					// Update Latest key if this is the latest version
					if versionKey == pkg.Latest {
						clonedPkg.Latest = prefixedKey
					}
				}
				merged.Packages[pkgID] = clonedPkg
			}
		}

		merged.TotalAPKs += manifest.TotalAPKs
		merged.TotalSize += manifest.TotalSize
	}

	return merged, nil
}

func (b *BucketManager) verifyManifestSignature(manifest *models.ManifestIndex) error {
	if manifest == nil {
		return fmt.Errorf("manifest is empty")
	}

	sig := manifest.Signature
	if sig == nil {
		return fmt.Errorf("manifest missing signature metadata")
	}

	if sig.PublicKeyFingerprint == "" {
		return fmt.Errorf("manifest signature missing public key fingerprint")
	}

	if len(b.config.Security.TrustedKeys) > 0 && !containsString(b.config.Security.TrustedKeys, sig.PublicKeyFingerprint) {
		return fmt.Errorf("untrusted manifest signer: %s", sig.PublicKeyFingerprint)
	}

	if sig.SignedAt.IsZero() {
		return fmt.Errorf("manifest signature timestamp is missing")
	}

	return nil
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

// GetBucketHealth returns health status for a specific bucket
func (b *BucketManager) GetBucketHealth(bucketName string) *BucketHealth {
	return b.healthMap[bucketName]
}

// GetAllHealth returns health status for all tracked buckets
func (b *BucketManager) GetAllHealth() map[string]*BucketHealth {
	return b.healthMap
}

// PrintHealthStatus prints a formatted health status report
func (b *BucketManager) PrintHealthStatus() {
	if len(b.healthMap) == 0 {
		return
	}

	fmt.Printf("\n🏥 Bucket Health Status:\n")
	fmt.Printf("%-20s %-10s %-8s %-12s %-6s %s\n", "NAME", "STATUS", "RESP_MS", "LAST_SUCCESS", "FAILS", "LAST_ERROR")
	fmt.Printf("%s\n", strings.Repeat("-", 80))

	for name, health := range b.healthMap {
		status := health.Status
		statusIcon := "❓"
		switch status {
		case "healthy":
			statusIcon = "✅"
		case "degraded":
			statusIcon = "⚠️ "
		case "unhealthy":
			statusIcon = "❌"
		}

		lastSuccess := "Never"
		if !health.LastSuccess.IsZero() {
			lastSuccess = formatDuration(time.Since(health.LastSuccess))
		}

		responseTime := fmt.Sprintf("%dms", health.ResponseTime)
		if health.ResponseTime == 0 {
			responseTime = "-"
		}

		lastError := health.LastError
		if len(lastError) > 30 {
			lastError = lastError[:27] + "..."
		}

		fmt.Printf("%-20s %s%-9s %-8s %-12s %-6d %s\n",
			name, statusIcon, status, responseTime, lastSuccess, health.ConsecutiveFails, lastError)
	}
}

// CheckBucketHealth performs a health check on a specific bucket
func (b *BucketManager) CheckBucketHealth(bucketName string) (*BucketHealth, error) {
	bucket, exists := b.config.Buckets[bucketName]
	if !exists {
		return nil, fmt.Errorf("bucket %s not found", bucketName)
	}

	health := b.getOrCreateHealth(bucketName, bucket.URL)

	startTime := time.Now()
	var err error

	if strings.HasPrefix(bucket.URL, "file://") {
		// Local bucket health check
		err = b.checkLocalBucketHealth(bucket.URL)
	} else {
		// Remote bucket health check
		err = b.checkRemoteBucketHealth(bucket.URL)
	}

	responseTime := time.Since(startTime)
	b.updateResponseTime(bucketName, responseTime)

	if err != nil {
		b.updateHealthOnError(bucketName, err)
		return health, err
	}

	b.updateHealthOnSuccess(bucketName)
	return health, nil
}

// checkLocalBucketHealth checks health of a local bucket
func (b *BucketManager) checkLocalBucketHealth(fileURL string) error {
	localPath := strings.TrimPrefix(fileURL, "file://")
	manifestPath := filepath.Join(localPath, "apkhub_manifest.json")

	// Check if directory exists
	if info, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("directory not accessible: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", localPath)
	}

	// Check if manifest exists and is readable
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("manifest not accessible: %w", err)
	}

	// Try to read and parse manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest models.ManifestIndex
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("invalid manifest format: %w", err)
	}

	return nil
}

// checkRemoteBucketHealth checks health of a remote bucket
func (b *BucketManager) checkRemoteBucketHealth(url string) error {
	// Perform a simple HEAD request to check availability
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", url+"/apkhub_manifest.json", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "ApkHub-CLI/1.0")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// CheckAllHealth performs health checks on all enabled buckets
func (b *BucketManager) CheckAllHealth() map[string]*BucketHealth {
	results := make(map[string]*BucketHealth)

	for name := range b.config.GetEnabledBuckets() {
		health, _ := b.CheckBucketHealth(name)
		results[name] = health
	}

	return results
}

// GetHealthSummary returns a summary of bucket health
func (b *BucketManager) GetHealthSummary() map[string]int {
	summary := map[string]int{
		"healthy":   0,
		"degraded":  0,
		"unhealthy": 0,
		"unknown":   0,
	}

	for _, health := range b.healthMap {
		summary[health.Status]++
	}

	return summary
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// resolveDownloadURL resolves a relative download URL for a bucket
func (b *BucketManager) resolveDownloadURL(bucketName, relativeURL string) string {
	bucket := b.config.Buckets[bucketName]
	if bucket == nil {
		return relativeURL
	}

	// Handle file:// URLs (local buckets)
	if strings.HasPrefix(bucket.URL, "file://") {
		localPath := strings.TrimPrefix(bucket.URL, "file://")
		fullPath := filepath.Join(localPath, relativeURL)
		// Normalize path separators for URLs
		fullPath = strings.ReplaceAll(fullPath, "\\", "/")
		return "file://" + fullPath
	}

	// Handle local development URLs
	if b.isLocalDevelopmentURL(bucket.URL) {
		// Ensure proper URL path joining
		baseURL := strings.TrimRight(bucket.URL, "/")
		urlPath := strings.TrimLeft(relativeURL, "/")
		return fmt.Sprintf("%s/%s", baseURL, urlPath)
	}

	// For remote buckets, use HTTP URL
	baseURL := strings.TrimRight(bucket.URL, "/")
	urlPath := strings.TrimLeft(relativeURL, "/")
	return fmt.Sprintf("%s/%s", baseURL, urlPath)
}

// isLocalDevelopmentURL checks if URL is for local development
func (b *BucketManager) isLocalDevelopmentURL(url string) bool {
	return strings.HasPrefix(url, "http://localhost") ||
		strings.HasPrefix(url, "http://127.0.0.1")
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

// isAbsoluteURL checks if a URL is absolute
func isAbsoluteURL(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://" || url[:7] == "file://")
}
