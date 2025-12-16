package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	searchBucket    string
	searchMinSDK    int
	searchCategory  string
	searchLimit     int
	searchSort      string
	searchFormat    string
	searchVerbose   bool
	searchExact     bool
	searchInstalled bool
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
			return fmt.Errorf("%s: %w", i18n.T("cmd.search.errLoadConfig"), err)
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.search.errCreateDirs"), err)
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
			fmt.Println(i18n.T("cmd.search.offline"))
			results, err := offlineManager.GetOfflineSearch(query, options)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cmd.search.errOfflineSearch"), err)
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
			fmt.Printf("%s", i18n.T("cmd.search.searchingFor", map[string]interface{}{"query": query}))
			if searchBucket != "" {
				fmt.Printf(" %s", i18n.T("cmd.search.searchBucket", map[string]interface{}{"bucket": searchBucket}))
			}
			if searchCategory != "" {
				fmt.Printf(" %s", i18n.T("cmd.search.searchCategory", map[string]interface{}{"category": searchCategory}))
			}
			fmt.Println("...")
		}

		// Perform search
		results, err := searchEngine.Search(query, options)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.search.errSearch"), err)
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
	},
}

// displayResultsDefault displays search results in default format
func displayResultsDefault(results []client.SearchResult, verbose bool) error {
	if len(results) == 0 {
		fmt.Println(i18n.T("cmd.search.noResults"))
		fmt.Println()
		fmt.Println(i18n.T("cmd.search.suggestionTitle"))
		fmt.Println(i18n.T("cmd.search.suggestionTerm"))
		fmt.Println(i18n.T("cmd.search.suggestionUpdate"))
		fmt.Println(i18n.T("cmd.search.suggestionAdd"))
		fmt.Println(i18n.T("cmd.search.suggestionExact"))
		return nil
	}

	fmt.Printf("%s\n\n", i18n.T("cmd.search.foundResults", map[string]interface{}{
		"count": len(results),
	}))

	for i, result := range results {
		fmt.Printf("%s\n", i18n.T("cmd.search.resultHeader", map[string]interface{}{
			"index": i + 1,
			"name":  result.AppName,
			"id":    result.PackageID,
		}))
		fmt.Printf("%s", i18n.T("cmd.search.resultVersion", map[string]interface{}{
			"version": result.Version,
		}))
		if result.BucketName != "" {
			fmt.Printf(" %s", i18n.T("cmd.search.resultBucket", map[string]interface{}{
				"bucket": result.BucketName,
			}))
		}
		if verbose && result.Score > 0 {
			fmt.Printf(" %s", i18n.T("cmd.search.resultScore", map[string]interface{}{
				"score": result.Score,
			}))
		}
		fmt.Println()

		if result.Description != "" {
			desc := result.Description
			if len(desc) > 80 && !verbose {
				desc = desc[:77] + "..."
			}
			fmt.Printf("%s\n", i18n.T("cmd.search.resultDescription", map[string]interface{}{
				"description": desc,
			}))
		}

		if verbose && result.Category != "" {
			fmt.Printf("%s\n", i18n.T("cmd.search.resultCategory", map[string]interface{}{
				"category": result.Category,
			}))
		}

		fmt.Println()
	}

	if len(results) == searchLimit && searchLimit > 0 {
		fmt.Printf("%s\n\n", i18n.T("cmd.search.limitNotice", map[string]interface{}{
			"limit": searchLimit,
		}))
	}

	fmt.Println(i18n.T("cmd.search.nextStepsTitle"))
	fmt.Println(i18n.T("cmd.search.nextStepsInfo"))
	fmt.Println(i18n.T("cmd.search.nextStepsInstall"))

	return nil
}

// displayResultsTable displays search results in table format
func displayResultsTable(results []client.SearchResult, verbose bool) error {
	if len(results) == 0 {
		fmt.Println(i18n.T("cmd.search.noResults"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if verbose {
		fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s",
			i18n.T("cmd.search.columns.packageID"),
			i18n.T("cmd.search.columns.name"),
			i18n.T("cmd.search.columns.version"),
			i18n.T("cmd.search.columns.bucket"),
			i18n.T("cmd.search.columns.category"),
			i18n.T("cmd.search.columns.score"),
			i18n.T("cmd.search.columns.description"),
		))
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
		fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s",
			i18n.T("cmd.search.columns.packageID"),
			i18n.T("cmd.search.columns.name"),
			i18n.T("cmd.search.columns.version"),
			i18n.T("cmd.search.columns.bucket"),
			i18n.T("cmd.search.columns.description"),
		))
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
	fmt.Printf("\n%s\n", i18n.T("cmd.search.resultsCount", map[string]interface{}{
		"count": len(results),
	}))

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
	searchCmd.Flags().StringVarP(&searchBucket, "bucket", "b", "", i18n.T("cmd.search.flag.bucket"))
	searchCmd.Flags().IntVar(&searchMinSDK, "min-sdk", 0, i18n.T("cmd.search.flag.minSDK"))
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "", i18n.T("cmd.search.flag.category"))
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, i18n.T("cmd.search.flag.limit"))
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", i18n.T("cmd.search.flag.sort"))
	searchCmd.Flags().StringVar(&searchFormat, "format", "default", i18n.T("cmd.search.flag.format"))
	searchCmd.Flags().BoolVarP(&searchVerbose, "verbose", "v", false, i18n.T("cmd.search.flag.verbose"))
	searchCmd.Flags().BoolVar(&searchExact, "exact", false, i18n.T("cmd.search.flag.exact"))
	searchCmd.Flags().BoolVar(&searchInstalled, "installed", false, i18n.T("cmd.search.flag.installed"))
}
