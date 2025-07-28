package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	searchBucket   string
	searchMinSDK   int
	searchCategory string
	searchLimit    int
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

		// Create bucket manager and search engine
		bucketMgr := client.NewBucketManager(config)
		searchEngine := client.NewSearchEngine(bucketMgr)

		// Search options
		options := client.SearchOptions{
			Bucket:   searchBucket,
			MinSDK:   searchMinSDK,
			Category: searchCategory,
			Limit:    searchLimit,
		}

		fmt.Printf("Searching for '%s'...\n\n", query)

		// Perform search
		results, err := searchEngine.Search(query, options)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			fmt.Println("\nTry:")
			fmt.Println("  - Using a different search term")
			fmt.Println("  - Updating buckets with 'apkhub bucket update'")
			fmt.Println("  - Adding more buckets with 'apkhub bucket add'")
			return nil
		}

		// Display results
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PACKAGE ID\tNAME\tVERSION\tBUCKET\tDESCRIPTION")
		fmt.Fprintln(w, "----------\t----\t-------\t------\t-----------")

		for _, result := range results {
			// Truncate description
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

		w.Flush()

		// Show total and hints
		fmt.Printf("\n%d result(s) found.\n", len(results))

		if len(results) == searchLimit && searchLimit > 0 {
			fmt.Printf("(Showing top %d results. Use --limit to see more)\n", searchLimit)
		}

		fmt.Println("\nUse 'apkhub info <package-id>' for more details")
		fmt.Println("Use 'apkhub install <package-id>' to install")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Add flags
	searchCmd.Flags().StringVarP(&searchBucket, "bucket", "b", "", "Search in specific bucket only")
	searchCmd.Flags().IntVar(&searchMinSDK, "min-sdk", 0, "Minimum SDK version")
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "", "Filter by category")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Maximum number of results")
}
