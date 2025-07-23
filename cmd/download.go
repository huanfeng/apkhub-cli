package cmd

import (
	"fmt"

	"github.com/apkhub/apkhub-cli/pkg/client"
	"github.com/spf13/cobra"
)

var (
	downloadVersion  string
	downloadForce    bool
	downloadNoVerify bool
	downloadNoInstall bool
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
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
		
		// Create managers
		bucketMgr := client.NewBucketManager(config)
		downloadMgr := client.NewDownloadManager(config, bucketMgr)
		
		// Download options
		options := client.DownloadOptions{
			Version:   downloadVersion,
			Force:     downloadForce,
			NoCVerify: downloadNoVerify,
		}
		
		// Download APK
		apkPath, err := downloadMgr.Download(packageID, options)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		
		// Install if requested (default behavior)
		if !downloadNoInstall {
			fmt.Println("\nReady to install. Use 'apkhub install' to install the downloaded APK.")
			fmt.Printf("APK path: %s\n", apkPath)
		}
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	
	// Add flags
	downloadCmd.Flags().StringVarP(&downloadVersion, "version", "v", "", "Download specific version")
	downloadCmd.Flags().BoolVarP(&downloadForce, "force", "f", false, "Force re-download even if exists")
	downloadCmd.Flags().BoolVar(&downloadNoVerify, "no-verify", false, "Skip checksum verification")
	downloadCmd.Flags().BoolVar(&downloadNoInstall, "no-install", false, "Download without prompting to install")
}