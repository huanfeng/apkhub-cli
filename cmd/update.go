package cmd

import (
	"fmt"

	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/pkg/client"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command (alias for bucket update)
var updateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: i18n.T("cmd.update.short"),
	Long:  i18n.T("cmd.update.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.update.errLoadConfig"), err)
		}

		bucketMgr := client.NewBucketManager(config)
		bucketMgr.SetSignatureVerification(bucketVerifySignature && config.Security.VerifySignature)

		if len(args) > 0 {
			// Update specific bucket
			name := args[0]
			fmt.Printf("%s\n", i18n.T("cmd.update.single", map[string]interface{}{"name": name}))
			if _, err := bucketMgr.FetchManifest(name); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cmd.update.errUpdate"), err)
			}
			fmt.Printf("%s\n", i18n.T("cmd.update.singleSuccess", map[string]interface{}{"name": name}))
		} else {
			// Update all buckets
			if updateAll {
				// Update all buckets including disabled ones
				fmt.Println(i18n.T("cmd.update.allIncludingDisabled"))

				for name := range config.Buckets {
					fmt.Printf("%s\n", i18n.T("cmd.update.fetching", map[string]interface{}{"name": name}))
					if _, err := bucketMgr.FetchManifest(name); err != nil {
						fmt.Printf("%s\n", i18n.T("cmd.update.updateFail", map[string]interface{}{
							"name":  name,
							"error": err,
						}))
					} else {
						fmt.Printf("%s\n", i18n.T("cmd.update.updateSuccess", map[string]interface{}{"name": name}))
					}
				}
			} else {
				// Update only enabled buckets
				if err := bucketMgr.UpdateAll(); err != nil {
					return err
				}
			}
			fmt.Println()
			fmt.Println(i18n.T("cmd.update.completed"))
		}

		return nil
	},
}

var updateAll bool

func init() {
	rootCmd.AddCommand(updateCmd)

	// Add flags
	updateCmd.Flags().BoolVar(&updateAll, "all", false, i18n.T("cmd.update.flag.all"))
	updateCmd.Flags().BoolVar(&bucketVerifySignature, "verify-signature", true, i18n.T("cmd.update.flag.verifySignature"))
}
