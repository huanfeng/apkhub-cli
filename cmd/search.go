package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	searchBucket     string
	searchMinSDK     int
	searchCategory   string
	searchLimit      int
	searchSort       string
	searchFormat     string
	searchVerbose    bool
	searchExact      bool
	searchInstalled  bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for applications",
	Long:  `Search for applications across all enabled buckets.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		// Search options
		options := client.SearchOptions{
			Bucket:        searchBucket,
			MinSDK:        searchMinSDK,
			Category:      searchCategory,
			Limit:         searchLimit,
			Sort:          searchSort,
			Exact:         searchExact,
			ShowInstalled: searchInstalled,
		}

		// Create managers
		bucketMgr := client.NewBucketManager(config)
		cacheManager := client.NewCacheManager(config)
		offlineManager := client.NewOfflineManager(config, cacheManager)
		
		// Check if we're offline and handle accordingly
		if offlineManager.IsOffline() {
			fmt.Println("🔌 Network unavailable - searching cached data only")
			results, err := offlineManager.GetOfflineSearch(query, options)
			if err != nil {
				return fmt.Errorf("offline search failed: %w", err)
			}
			
			// Display results based on format
			switch searchFormat {
			case "json":
				return displayResultsJSON(results)
			case "table":
				return displayResultsTable(results, searchVerbose)
			default:
				return displayResultsDefault(results, searchVerbose)
			}
		}
		
		// Online mode - create search engine
		searchEngine := client.NewSearchEngine(bucketMgr)

		if !strings.Contains(searchFormat, "json") {
			fmt.Printf("🔍 Searching for '%s'", query)
			if searchBucket != "" {
				fmt.Printf(" in bucket '%s'", searchBucket)
			}
			if searchCategory != "" {
				fmt.Printf(" (category: %s)", searchCategory)
			}
			fmt.Println("...\n")
		}

		// Perform search
		results, err := searchEngine.Search(query, options)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		// Display results based on format
		switch searchFormat {
		case "json":
			return displayResultsJSON(results)
		case "table":
			return displayResultsTable(results, searchVerbose)
		default:
			return displayResultsDefault(results, searchVerbose)
		}

		return nil
	},
}

// displayResultsDefault displays search results in default format
func displayResultsDefault(results []client.SearchResult, verbose bool) error {
	if len(results) == 0 {
		fmt.Println("❌ No results found.")
		fmt.Println("\n💡 Try:")
		fmt.Println("   • Using a different search term")
		fmt.Println("   • Updating buckets with 'apkhub bucket update'")
		fmt.Println("   • Adding more buckets with 'apkhub bucket add'")
		fmt.Println("   • Using --exact for exact matches")
		return nil
	}

	fmt.Printf("📱 Found %d result(s):\n\n", len(results))

	for i, result := range results {
		fmt.Printf("%d. %s (%s)\n", i+1, result.AppName, result.PackageID)
		fmt.Printf("   Version: %s", result.Version)
		if result.BucketName != "" {
			fmt.Printf(" | Bucket: %s", result.BucketName)
		}
		if verbose && result.Score > 0 {
			fmt.Printf(" | Score: %.1f", result.Score)
		}
		fmt.Println()
		
		if result.Description != "" {
			desc := result.Description
			if len(desc) > 80 && !verbose {
				desc = desc[:77] + "..."
			}
			fmt.Printf("   %s\n", desc)
		}
		
		if verbose && result.Category != "" {
			fmt.Printf("   Category: %s\n", result.Category)
		}
		
		fmt.Println()
	}

	if len(results) == searchLimit && searchLimit > 0 {
		fmt.Printf("📄 Showing top %d results. Use --limit to see more\n\n", searchLimit)
	}

	fmt.Println("💡 Next steps:")
	fmt.Println("   • Use 'apkhub info <package-id>' for detailed information")
	fmt.Println("   • Use 'apkhub install <package-id>' to install an app")
	
	return nil
}

// displayResultsTable displays search results in table format
func displayResultsTable(results []client.SearchResult, verbose bool) error {
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	if verbose {
		fmt.Fprintln(w, "PACKAGE ID\tNAME\tVERSION\tBUCKET\tCATEGORY\tSCORE\tDESCRIPTION")
		fmt.Fprintln(w, "----------\t----\t-------\t------\t--------\t-----\t-----------")
		
		for _, result := range results {
			desc := result.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.1f\t%s\n",
				result.PackageID,
				result.AppName,
				result.Version,
				result.BucketName,
				result.Category,
				result.Score,
				desc,
			)
		}
	} else {
		fmt.Fprintln(w, "PACKAGE ID\tNAME\tVERSION\tBUCKET\tDESCRIPTION")
		fmt.Fprintln(w, "----------\t----\t-------\t------\t-----------")
		
		for _, result := range results {
			desc := result.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				result.PackageID,
				result.AppName,
				result.Version,
				result.BucketName,
				desc,
			)
		}
	}

	w.Flush()
	fmt.Printf("\n%d result(s) found.\n", len(results))
	
	return nil
}

// displayResultsJSON displays search results in JSON format
func displayResultsJSON(results []client.SearchResult) error {
	output := map[string]interface{}{
		"total_results": len(results),
		"results":       results,
	}
	
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	fmt.Println(string(data))
	return nil
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Add flags
	searchCmd.Flags().StringVarP(&searchBucket, "bucket", "b", "", "Search in specific bucket only")
	searchCmd.Flags().IntVar(&searchMinSDK, "min-sdk", 0, "Minimum SDK version")
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "", "Filter by category")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Maximum number of results (0 = no limit)")
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort results by: relevance, name, version, size")
	searchCmd.Flags().StringVar(&searchFormat, "format", "default", "Output format: default, table, json")
	searchCmd.Flags().BoolVarP(&searchVerbose, "verbose", "v", false, "Show detailed information")
	searchCmd.Flags().BoolVar(&searchExact, "exact", false, "Exact match search")
	searchCmd.Flags().BoolVar(&searchInstalled, "installed", false, "Show installation status")
}
