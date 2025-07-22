package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apkhub",
	Short: "ApkHub CLI - A tool for managing APK repositories",
	Long: `ApkHub CLI is a command-line tool for managing distributed APK repositories.
It supports parsing APK files, generating repository indexes, and maintaining APK collections.`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}