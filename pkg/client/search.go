package client

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apkhub/apkhub-cli/pkg/models"
)

// SearchResult represents a search result
type SearchResult struct {
	PackageID   string
	AppName     string
	Version     string
	Description string
	BucketName  string
	Score       float64
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
		score := s.calculateScore(query, pkgID, pkg)
		if score == 0 {
			continue
		}
		
		// Apply filters
		if options.MinSDK > 0 || options.Category != "" {
			// Get latest version info
			if pkg.Latest != "" {
				if version, exists := pkg.Versions[pkg.Latest]; exists {
					if options.MinSDK > 0 && version.MinSDK < options.MinSDK {
						continue
					}
				}
			}
			if options.Category != "" && !strings.EqualFold(pkg.Category, options.Category) {
				continue
			}
		}
		
		// Get app name
		appName := getDefaultName(pkg.Name)
		
		// Get latest version
		latestVersion := ""
		if pkg.Latest != "" {
			if version, exists := pkg.Versions[pkg.Latest]; exists {
				latestVersion = version.Version
			}
		}
		
		// Get bucket name from version key
		bucketName := ""
		if pkg.Latest != "" && strings.Contains(pkg.Latest, "_") {
			parts := strings.SplitN(pkg.Latest, "_", 2)
			bucketName = parts[0]
		}
		
		results = append(results, SearchResult{
			PackageID:   pkgID,
			AppName:     appName,
			Version:     latestVersion,
			Description: getDefaultName(pkg.Description),
			BucketName:  bucketName,
			Score:       score,
		})
	}
	
	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Apply limit
	if options.Limit > 0 && len(results) > options.Limit {
		results = results[:options.Limit]
	}
	
	return results, nil
}

// SearchOptions contains search options
type SearchOptions struct {
	Bucket   string
	MinSDK   int
	Category string
	Limit    int
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