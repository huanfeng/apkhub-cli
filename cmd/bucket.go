package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/apkhub/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Manage APK repository buckets",
	Long:  `Manage APK repository buckets (sources) for the client functionality.`,
}

var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured buckets",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		if len(config.Buckets) == 0 {
			fmt.Println("No buckets configured. Use 'apkhub bucket add' to add one.")
			return nil
		}
		
		// Display buckets in table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDISPLAY NAME\tURL\tENABLED\tLAST UPDATED")
		fmt.Fprintln(w, "----\t------------\t---\t-------\t------------")
		
		for name, bucket := range config.Buckets {
			enabled := "Yes"
			if !bucket.Enabled {
				enabled = "No"
			}
			
			lastUpdated := "Never"
			if !bucket.LastUpdated.IsZero() {
				lastUpdated = bucket.LastUpdated.Format("2006-01-02 15:04")
			}
			
			marker := ""
			if name == config.DefaultBucket {
				marker = "*"
			}
			
			fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n",
				name, marker, bucket.Name, bucket.URL, enabled, lastUpdated)
		}
		
		w.Flush()
		
		if config.DefaultBucket != "" {
			fmt.Printf("\n* Default bucket: %s\n", config.DefaultBucket)
		}
		
		return nil
	},
}

var bucketAddCmd = &cobra.Command{
	Use:   "add <name> <url> [display-name]",
	Short: "Add a new bucket",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := args[1]
		displayName := name
		if len(args) > 2 {
			displayName = args[2]
		}
		
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// Add bucket
		if err := config.AddBucket(name, url, displayName); err != nil {
			return fmt.Errorf("failed to add bucket: %w", err)
		}
		
		fmt.Printf("✓ Added bucket '%s' (%s)\n", name, url)
		
		// Update bucket index
		fmt.Printf("Fetching bucket manifest...\n")
		bucketMgr := client.NewBucketManager(config)
		if _, err := bucketMgr.FetchManifest(name); err != nil {
			fmt.Printf("Warning: Failed to fetch manifest: %v\n", err)
			fmt.Println("You can try updating later with 'apkhub bucket update'")
		} else {
			fmt.Println("✓ Bucket manifest fetched successfully")
		}
		
		return nil
	},
}

var bucketRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// Confirm removal
		if !skipConfirm {
			fmt.Printf("Remove bucket '%s'? [y/N]: ", name)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		
		// Remove bucket
		if err := config.RemoveBucket(name); err != nil {
			return fmt.Errorf("failed to remove bucket: %w", err)
		}
		
		fmt.Printf("✓ Removed bucket '%s'\n", name)
		
		// Clean up cache
		cacheFile := fmt.Sprintf("%s/%s.json", config.Client.CacheDir, name)
		os.Remove(cacheFile)
		
		return nil
	},
}

var bucketUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update bucket manifests",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		bucketMgr := client.NewBucketManager(config)
		
		if len(args) > 0 {
			// Update specific bucket
			name := args[0]
			fmt.Printf("Updating bucket '%s'...\n", name)
			if _, err := bucketMgr.FetchManifest(name); err != nil {
				return fmt.Errorf("failed to update bucket: %w", err)
			}
			fmt.Printf("✓ Updated bucket '%s'\n", name)
		} else {
			// Update all buckets
			if err := bucketMgr.UpdateAll(); err != nil {
				return err
			}
			fmt.Println("\n✓ All buckets updated")
		}
		
		return nil
	},
}

var bucketEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setBucketEnabled(args[0], true)
	},
}

var bucketDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setBucketEnabled(args[0], false)
	},
}

func setBucketEnabled(name string, enabled bool) error {
	// Load client config
	config, err := client.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	bucket, exists := config.Buckets[name]
	if !exists {
		return fmt.Errorf("bucket '%s' not found", name)
	}
	
	bucket.Enabled = enabled
	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	status := "enabled"
	if !enabled {
		status = "disabled"
	}
	fmt.Printf("✓ Bucket '%s' %s\n", name, status)
	
	return nil
}

func init() {
	rootCmd.AddCommand(bucketCmd)
	
	// Add subcommands
	bucketCmd.AddCommand(bucketListCmd)
	bucketCmd.AddCommand(bucketAddCmd)
	bucketCmd.AddCommand(bucketRemoveCmd)
	bucketCmd.AddCommand(bucketUpdateCmd)
	bucketCmd.AddCommand(bucketEnableCmd)
	bucketCmd.AddCommand(bucketDisableCmd)
	
	// Add flags
	bucketRemoveCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
}