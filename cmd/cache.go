package cmd

import (
	"fmt"
	"strings"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cache operations",
	Long:  `Manage cache operations including viewing statistics, cleaning expired entries, and clearing cache.`,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create cache manager
		cacheManager := client.NewCacheManager(config)

		// Print statistics
		if err := cacheManager.PrintStats(); err != nil {
			return err
		}

		// Also show offline capabilities
		fmt.Println()
		offlineManager := client.NewOfflineManager(config, cacheManager)
		return offlineManager.PrintOfflineStatus()
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove expired cache entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create cache manager
		cacheManager := client.NewCacheManager(config)

		fmt.Println("🧹 Cleaning expired cache entries...")

		// Clean expired entries
		removed, err := cacheManager.CleanExpired()
		if err != nil {
			return fmt.Errorf("failed to clean cache: %w", err)
		}

		if removed > 0 {
			fmt.Printf("✅ Removed %d expired cache entries\n", removed)
		} else {
			fmt.Println("✅ No expired entries found")
		}

		// Show updated statistics
		fmt.Println()
		return cacheManager.PrintStats()
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cache entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create cache manager
		cacheManager := client.NewCacheManager(config)

		// Confirm clearing
		if !skipConfirm {
			fmt.Print("⚠️  This will remove all cached data. Continue? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
				fmt.Println("❌ Cancelled")
				return nil
			}
		}

		fmt.Println("🗑️  Clearing all cache entries...")

		// Clear cache
		if err := cacheManager.Clear(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}

		fmt.Println("✅ Cache cleared successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)

	// Add subcommands
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	// Add flags
	cacheClearCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
}
