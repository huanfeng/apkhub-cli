package cmd

import (
	"github.com/spf13/cobra"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage APK repository",
	Long: `Repository management commands for ApkHub.
This includes initializing, scanning, maintaining, and exporting repository data.`,
}

func init() {
	rootCmd.AddCommand(repoCmd)

	// Repository management commands will be added as subcommands
	// These commands were previously at the root level
}
