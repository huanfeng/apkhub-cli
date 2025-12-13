package cmd

import (
	"fmt"

	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command (alias for bucket update)
var updateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update bucket manifests (alias for 'bucket update')",
	Long: `Update bucket manifests from their sources. This is an alias for 'apkhub bucket update'.

Examples:
  apkhub update              # Update all enabled buckets
  apkhub update main         # Update specific bucket named 'main'
  apkhub update --all        # Update all buckets (including disabled ones)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		bucketMgr := client.NewBucketManager(config)
		bucketMgr.SetSignatureVerification(bucketVerifySignature && config.Security.VerifySignature)

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
			if updateAll {
				// Update all buckets including disabled ones
				fmt.Println("🔄 Updating all buckets (including disabled)...")

				for name := range config.Buckets {
					fmt.Printf("📡 Fetching '%s'...\n", name)
					if _, err := bucketMgr.FetchManifest(name); err != nil {
						fmt.Printf("❌ Failed to update '%s': %v\n", name, err)
					} else {
						fmt.Printf("✅ Updated '%s'\n", name)
					}
				}
			} else {
				// Update only enabled buckets
				if err := bucketMgr.UpdateAll(); err != nil {
					return err
				}
			}
			fmt.Println("\n✓ Update completed")
		}

		return nil
	},
}

var updateAll bool

func init() {
	rootCmd.AddCommand(updateCmd)

	// Add flags
	updateCmd.Flags().BoolVar(&updateAll, "all", false, "Update all buckets including disabled ones")
	updateCmd.Flags().BoolVar(&bucketVerifySignature, "verify-signature", true, "Verify bucket manifest signatures during update")
}
