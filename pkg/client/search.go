package client

import (
	"fmt"
	"sort"
	"strings"

	"github.com/huanfeng/apkhub-cli/pkg/models"
)

// SearchResult represents a search result
type SearchResult struct {
	PackageID   string  `json:"package_id"`
	AppName     string  `json:"app_name"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
	BucketName  string  `json:"bucket_name"`
	Category    string  `json:"category"`
	Score       float64 `json:"score"`
	Size        int64   `json:"size"`
	MinSDK      int     `json:"min_sdk"`
	TargetSDK   int     `json:"target_sdk"`
	IsInstalled bool    `json:"is_installed"`
}

// SearchEngine handles application searches
type SearchEngine struct {
	bucketMgr *BucketManager
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(bucketMgr *BucketManager) *SearchEngine {
	return &SearchEngine{
		bucketMgr: bucketMgr,
	}
}

// Search searches for applications across all enabled buckets
func (s *SearchEngine) Search(query string, options SearchOptions) ([]SearchResult, error) {
	// Get merged manifest
	manifest, err := s.bucketMgr.GetMergedManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Normalize query
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("empty search query")
	}

	var results []SearchResult

	// Search through packages
	for pkgID, pkg := range manifest.Packages {
		// Calculate relevance score
		var score float64
		if options.Exact {
			score = s.calculateExactScore(query, pkgID, pkg)
		} else {
			score = s.calculateScore(query, pkgID, pkg)
		}

		if score == 0 {
			continue
		}

		// Get latest version info
		var latestVersionInfo *models.AppVersion
		latestVersion := ""
		if pkg.Latest != "" {
			if version, exists := pkg.Versions[pkg.Latest]; exists {
				latestVersionInfo = version
				latestVersion = version.Version
			}
		}

		// Apply filters
		if options.MinSDK > 0 && latestVersionInfo != nil {
			if latestVersionInfo.MinSDK < options.MinSDK {
				continue
			}
		}

		if options.Category != "" && !strings.EqualFold(pkg.Category, options.Category) {
			continue
		}

		// Filter by bucket if specified
		bucketName := ""
		if pkg.Latest != "" && strings.Contains(pkg.Latest, "_") {
			parts := strings.SplitN(pkg.Latest, "_", 2)
			bucketName = parts[0]
		}

		if options.Bucket != "" && !strings.EqualFold(bucketName, options.Bucket) {
			continue
		}

		// Get app name
		appName := getDefaultName(pkg.Name)

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

		// Check installation status if requested
		if options.ShowInstalled {
			result.IsInstalled = s.checkInstallationStatus(pkgID)
		}

		results = append(results, result)
	}

	// Sort results based on options
	s.sortResults(results, options.Sort)

	// Apply limit
	if options.Limit > 0 && len(results) > options.Limit {
		results = results[:options.Limit]
	}

	return results, nil
}

// SearchOptions contains search options
type SearchOptions struct {
	Bucket        string
	MinSDK        int
	Category      string
	Limit         int
	Sort          string
	Exact         bool
	ShowInstalled bool
}

// calculateScore calculates relevance score for a package
func (s *SearchEngine) calculateScore(query string, pkgID string, pkg *models.AppPackage) float64 {
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
		// Bonus for word boundary match
		words := strings.Fields(appName)
		for _, word := range words {
			if strings.HasPrefix(word, query) {
				score += 10.0
				break
			}
		}
	}

	// Description match
	desc := strings.ToLower(getDefaultName(pkg.Description))
	if strings.Contains(desc, query) {
		score += 10.0
	}

	// Category match
	if strings.Contains(strings.ToLower(pkg.Category), query) {
		score += 5.0
	}

	return score
}

// getDefaultName returns the default name from a multi-language map
func getDefaultName(names map[string]string) string {
	if names == nil {
		return ""
	}

	// Try common language codes
	for _, lang := range []string{"default", "en", "en-US", "en_US"} {
		if name, ok := names[lang]; ok && name != "" {
			return name
		}
	}

	// Return first available
	for _, name := range names {
		if name != "" {
			return name
		}
	}

	return ""
}

// calculateExactScore calculates score for exact matching
func (s *SearchEngine) calculateExactScore(query string, pkgID string, pkg *models.AppPackage) float64 {
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

	// Exact word match in app name
	words := strings.Fields(appName)
	for _, word := range words {
		if word == query {
			return 80.0
		}
	}

	return 0.0
}

// sortResults sorts search results based on the specified criteria
func (s *SearchEngine) sortResults(results []SearchResult, sortBy string) {
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
		// Sort by score (descending), then by name
		sort.Slice(results, func(i, j int) bool {
			if results[i].Score == results[j].Score {
				return strings.ToLower(results[i].AppName) < strings.ToLower(results[j].AppName)
			}
			return results[i].Score > results[j].Score
		})
	}
}

// checkInstallationStatus checks if a package is installed (placeholder)
func (s *SearchEngine) checkInstallationStatus(packageID string) bool {
	// TODO: Implement actual installation status check
	// This would require ADB integration or checking local installation records
	return false
}
