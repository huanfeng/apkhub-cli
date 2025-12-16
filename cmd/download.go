package cmd

import (
	"fmt"

	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	downloadVersion   string
	downloadForce     bool
	downloadNoVerify  bool
	downloadNoInstall bool
	downloadOutput    string
	downloadRetries   int
	downloadTimeout   int
	downloadProgress  bool
)

var downloadCmd = &cobra.Command{
	Use:   "download <package-id>",
	Short: "Download an application",
	Long:  `Download an application APK from configured buckets.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageID := args[0]

		// Load client config
		config, err := client.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.download.errLoadConfig"), err)
		}

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.download.errCreateDirs"), err)
		}

		// Create managers
		bucketMgr := client.NewBucketManager(config)
		downloadMgr := client.NewDownloadManager(config, bucketMgr)

		// Download options
		options := client.DownloadOptions{
			Version:      downloadVersion,
			Force:        downloadForce,
			NoVerify:     downloadNoVerify,
			OutputPath:   downloadOutput,
			MaxRetries:   downloadRetries,
			Timeout:      downloadTimeout,
			ShowProgress: downloadProgress,
		}

		// Download APK
		apkPath, err := downloadMgr.Download(packageID, options)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.download.errDownload"), err)
		}

		// Install if requested (default behavior)
		if !downloadNoInstall {
			fmt.Println()
			fmt.Println(i18n.T("cmd.download.nextInstall"))
			fmt.Printf("%s\n", i18n.T("cmd.download.apkPath", map[string]interface{}{
				"path": apkPath,
			}))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Add flags
	downloadCmd.Flags().StringVarP(&downloadVersion, "version", "v", "", i18n.T("cmd.download.flag.version"))
	downloadCmd.Flags().BoolVarP(&downloadForce, "force", "f", false, i18n.T("cmd.download.flag.force"))
	downloadCmd.Flags().BoolVar(&downloadNoVerify, "no-verify", false, i18n.T("cmd.download.flag.noVerify"))
	downloadCmd.Flags().BoolVar(&downloadNoInstall, "no-install", false, i18n.T("cmd.download.flag.noInstall"))
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", i18n.T("cmd.download.flag.output"))
	downloadCmd.Flags().IntVar(&downloadRetries, "retries", 3, i18n.T("cmd.download.flag.retries"))
	downloadCmd.Flags().IntVar(&downloadTimeout, "timeout", 1800, i18n.T("cmd.download.flag.timeout"))
	downloadCmd.Flags().BoolVar(&downloadProgress, "progress", true, i18n.T("cmd.download.flag.progress"))
}
