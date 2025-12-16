package client

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/huanfeng/apkhub/pkg/models"
)

// OfflineManager handles offline mode detection and fallback strategies
type OfflineManager struct {
	config        *Config
	cacheManager  *CacheManager
	isOffline     bool
	lastCheck     time.Time
	checkInterval time.Duration
}

// NewOfflineManager creates a new offline manager
func NewOfflineManager(config *Config, cacheManager *CacheManager) *OfflineManager {
	return &OfflineManager{
		config:        config,
		cacheManager:  cacheManager,
		checkInterval: 30 * time.Second, // Check connectivity every 30 seconds
	}
}

// IsOffline checks if the system is currently offline
func (o *OfflineManager) IsOffline() bool {
	// Check if we need to refresh the offline status
	if time.Since(o.lastCheck) > o.checkInterval {
		o.checkConnectivity()
	}

	return o.isOffline
}

// checkConnectivity performs a connectivity check
func (o *OfflineManager) checkConnectivity() {
	o.lastCheck = time.Now()

	// Try to connect to a reliable host
	hosts := []string{
		"8.8.8.8:53",        // Google DNS
		"1.1.1.1:53",        // Cloudflare DNS
		"208.67.222.222:53", // OpenDNS
	}

	for _, host := range hosts {
		if o.canConnect(host) {
			o.isOffline = false
			return
		}
	}

	o.isOffline = true
}

// canConnect tests if we can connect to a specific host
func (o *OfflineManager) canConnect(host string) bool {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// GetOfflineManifest retrieves manifest in offline mode using cache
func (o *OfflineManager) GetOfflineManifest(bucketName string) (*models.ManifestIndex, error) {
	// Try to get from cache (allow stale)
	manifest, isStale, err := o.cacheManager.GetManifest(bucketName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached manifest: %w", err)
	}

	if manifest == nil {
		return nil, fmt.Errorf("no cached manifest available for bucket '%s'", bucketName)
	}

	if isStale {
		fmt.Printf("⚠️  Using stale cached data for bucket '%s' (offline mode)\n", bucketName)
	}

	return manifest, nil
}

// GetOfflineSearch performs search using cached data
func (o *OfflineManager) GetOfflineSearch(query string, options SearchOptions) ([]SearchResult, error) {
	fmt.Println("🔌 Operating in offline mode - using cached data")

	// Get merged offline manifest
	manifest, err := o.getMergedOfflineManifest()
	if err != nil {
		return nil, err
	}

	// Create a temporary search engine and perform search directly
	return o.performOfflineSearch(query, options, manifest)
}

// getMergedOfflineManifest creates a merged manifest from cached data
func (o *OfflineManager) getMergedOfflineManifest() (*models.ManifestIndex, error) {
	merged := &models.ManifestIndex{
		Version:     "1.0",
		Name:        "Offline Repository",
		Description: "Cached view of enabled buckets (offline mode)",
		UpdatedAt:   time.Now(),
		Packages:    make(map[string]*models.AppPackage),
	}

	enabledBuckets := o.config.GetEnabledBuckets()
	availableBuckets := 0

	for bucketName := range enabledBuckets {
		manifest, err := o.GetOfflineManifest(bucketName)
		if err != nil {
			fmt.Printf("⚠️  Skipping bucket '%s': %v\n", bucketName, err)
			continue
		}

		availableBuckets++

		// Merge packages (simplified version)
		for pkgID, pkg := range manifest.Packages {
			if existing, exists := merged.Packages[pkgID]; exists {
				// Merge versions
				for versionKey, version := range pkg.Versions {
					prefixedKey := fmt.Sprintf("%s_%s", bucketName, versionKey)
					existing.Versions[prefixedKey] = version
				}
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
				for versionKey, version := range pkg.Versions {
					prefixedKey := fmt.Sprintf("%s_%s", bucketName, versionKey)
					clonedVersion := *version
					clonedPkg.Versions[prefixedKey] = &clonedVersion
				}
				merged.Packages[pkgID] = clonedPkg
			}
		}

		merged.TotalAPKs += manifest.TotalAPKs
		merged.TotalSize += manifest.TotalSize
	}

	if availableBuckets == 0 {
		return nil, fmt.Errorf("no cached manifests available for offline mode")
	}

	fmt.Printf("📦 Loaded %d packages from %d cached bucket(s)\n", len(merged.Packages), availableBuckets)

	return merged, nil
}

// performOfflineSearch performs search on cached manifest data
func (o *OfflineManager) performOfflineSearch(query string, options SearchOptions, manifest *models.ManifestIndex) ([]SearchResult, error) {
	// Normalize query
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("empty search query")
	}

	var results []SearchResult

	// Search through packages (simplified version of search engine logic)
	for pkgID, pkg := range manifest.Packages {
		// Calculate relevance score
		score := o.calculateOfflineScore(query, pkgID, pkg, options.Exact)
		if score == 0 {
			continue
		}

		// Apply filters
		if options.Category != "" && !strings.EqualFold(pkg.Category, options.Category) {
			continue
		}

		// Get app name and latest version
		appName := getDefaultName(pkg.Name)
		latestVersion := ""
		var latestVersionInfo *models.AppVersion

		if pkg.Latest != "" {
			if version, exists := pkg.Versions[pkg.Latest]; exists {
				latestVersionInfo = version
				latestVersion = version.Version
			}
		}

		// Apply MinSDK filter
		if options.MinSDK > 0 && latestVersionInfo != nil {
			if latestVersionInfo.MinSDK < options.MinSDK {
				continue
			}
		}

		// Get bucket name from version key
		bucketName := ""
		if pkg.Latest != "" && strings.Contains(pkg.Latest, "_") {
			parts := strings.SplitN(pkg.Latest, "_", 2)
			bucketName = parts[0]
		}

		// Filter by bucket if specified
		if options.Bucket != "" && !strings.EqualFold(bucketName, options.Bucket) {
			continue
		}

		// Create result
		result := SearchResult{
			PackageID:   pkgID,
			AppName:     appName,
			Version:     latestVersion,
			Description: getDefaultName(pkg.Description),
			BucketName:  bucketName,
			Category:    pkg.Category,
			Score:       score,
		}

		// Add version-specific info if available
		if latestVersionInfo != nil {
			result.Size = latestVersionInfo.Size
			result.MinSDK = latestVersionInfo.MinSDK
			result.TargetSDK = latestVersionInfo.TargetSDK
		}

		results = append(results, result)
	}

	// Sort results
	o.sortOfflineResults(results, options.Sort)

	// Apply limit
	if options.Limit > 0 && len(results) > options.Limit {
		results = results[:options.Limit]
	}

	return results, nil
}

// calculateOfflineScore calculates relevance score for offline search
func (o *OfflineManager) calculateOfflineScore(query string, pkgID string, pkg *models.AppPackage, exact bool) float64 {
	if exact {
		return o.calculateExactOfflineScore(query, pkgID, pkg)
	}

	score := 0.0

	// Exact package ID match
	if strings.EqualFold(pkgID, query) {
		return 100.0
	}

	// Package ID contains query
	if strings.Contains(strings.ToLower(pkgID), query) {
		score += 50.0
	}

	// App name match
	appName := strings.ToLower(getDefaultName(pkg.Name))
	if appName == query {
		score += 80.0
	} else if strings.Contains(appName, query) {
		score += 40.0
	}

	// Description match
	desc := strings.ToLower(getDefaultName(pkg.Description))
	if strings.Contains(desc, query) {
		score += 10.0
	}

	return score
}

// calculateExactOfflineScore calculates score for exact matching
func (o *OfflineManager) calculateExactOfflineScore(query string, pkgID string, pkg *models.AppPackage) float64 {
	query = strings.ToLower(query)

	// Exact package ID match
	if strings.EqualFold(pkgID, query) {
		return 100.0
	}

	// Exact app name match
	appName := strings.ToLower(getDefaultName(pkg.Name))
	if appName == query {
		return 90.0
	}

	return 0.0
}

// sortOfflineResults sorts search results
func (o *OfflineManager) sortOfflineResults(results []SearchResult, sortBy string) {
	// Use the same sorting logic as the main search engine
	switch strings.ToLower(sortBy) {
	case "name":
		sort.Slice(results, func(i, j int) bool {
			return strings.ToLower(results[i].AppName) < strings.ToLower(results[j].AppName)
		})
	case "version":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Version > results[j].Version
		})
	case "size":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Size > results[j].Size
		})
	case "package":
		sort.Slice(results, func(i, j int) bool {
			return results[i].PackageID < results[j].PackageID
		})
	case "relevance":
		fallthrough
	default:
		sort.Slice(results, func(i, j int) bool {
			if results[i].Score == results[j].Score {
				return strings.ToLower(results[i].AppName) < strings.ToLower(results[j].AppName)
			}
			return results[i].Score > results[j].Score
		})
	}
}

// CheckOfflineCapabilities checks what operations are available offline
func (o *OfflineManager) CheckOfflineCapabilities() (*OfflineCapabilities, error) {
	capabilities := &OfflineCapabilities{
		IsOffline:        o.IsOffline(),
		AvailableBuckets: []string{},
		TotalPackages:    0,
		LastUpdate:       time.Time{},
	}

	// Check each enabled bucket for cached data
	enabledBuckets := o.config.GetEnabledBuckets()

	for bucketName := range enabledBuckets {
		if manifest, _, err := o.cacheManager.GetManifest(bucketName, true); err == nil && manifest != nil {
			capabilities.AvailableBuckets = append(capabilities.AvailableBuckets, bucketName)
			capabilities.TotalPackages += len(manifest.Packages)

			// Track most recent update
			if manifest.UpdatedAt.After(capabilities.LastUpdate) {
				capabilities.LastUpdate = manifest.UpdatedAt
			}
		}
	}

	return capabilities, nil
}

// OfflineCapabilities represents what's available in offline mode
type OfflineCapabilities struct {
	IsOffline        bool      `json:"is_offline"`
	AvailableBuckets []string  `json:"available_buckets"`
	TotalPackages    int       `json:"total_packages"`
	LastUpdate       time.Time `json:"last_update"`
}

// PrintOfflineStatus prints current offline status and capabilities
func (o *OfflineManager) PrintOfflineStatus() error {
	capabilities, err := o.CheckOfflineCapabilities()
	if err != nil {
		return err
	}

	fmt.Println("🔌 Offline Mode Status:")
	fmt.Println("========================")

	if capabilities.IsOffline {
		fmt.Println("   Status: 🔴 OFFLINE")
		fmt.Println("   Mode: Using cached data only")
	} else {
		fmt.Println("   Status: 🟢 ONLINE")
		fmt.Println("   Mode: Live data with cache fallback")
	}

	fmt.Printf("   Available buckets: %d\n", len(capabilities.AvailableBuckets))
	if len(capabilities.AvailableBuckets) > 0 {
		fmt.Printf("   Buckets: %s\n", fmt.Sprintf("%v", capabilities.AvailableBuckets))
		fmt.Printf("   Total packages: %d\n", capabilities.TotalPackages)

		if !capabilities.LastUpdate.IsZero() {
			fmt.Printf("   Last update: %s\n", capabilities.LastUpdate.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("   ⚠️  No cached data available")
		fmt.Println("   💡 Run 'apkhub bucket update' when online to cache data")
	}

	return nil
}
