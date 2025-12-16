package cmd

import (
	"fmt"
	"strings"

	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/client"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: i18n.T("cmd.cache.short"),
	Long:  i18n.T("cmd.cache.long"),
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: i18n.T("cmd.cache.stats.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.cache.errLoadConfig"), err)
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
	Short: i18n.T("cmd.cache.clean.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.cache.errLoadConfig"), err)
		}

		// Create cache manager
		cacheManager := client.NewCacheManager(config)

		fmt.Println(i18n.T("cmd.cache.clean.start"))

		// Clean expired entries
		removed, err := cacheManager.CleanExpired()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.cache.errClean"), err)
		}

		if removed > 0 {
			fmt.Printf(i18n.T("cmd.cache.clean.removed")+"\n", removed)
		} else {
			fmt.Println(i18n.T("cmd.cache.clean.none"))
		}

		// Show updated statistics
		fmt.Println()
		return cacheManager.PrintStats()
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: i18n.T("cmd.cache.clear.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.cache.errLoadConfig"), err)
		}

		// Create cache manager
		cacheManager := client.NewCacheManager(config)

		// Confirm clearing
		if !skipConfirm {
			fmt.Print(i18n.T("cmd.cache.clear.confirm"))
			var response string
			fmt.Scanln(&response)
			if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
				fmt.Println(i18n.T("cmd.cache.clear.cancel"))
				return nil
			}
		}

		fmt.Println(i18n.T("cmd.cache.clear.start"))

		// Clear cache
		if err := cacheManager.Clear(); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.cache.errClear"), err)
		}

		fmt.Println(i18n.T("cmd.cache.clear.success"))
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
	cacheClearCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, i18n.T("cmd.cache.flag.yes"))
}
