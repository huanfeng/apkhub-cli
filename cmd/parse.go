package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/apkhub/apkhub-cli/pkg/apk"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse [apk-file]",
	Short: "Parse a single APK file and display its information",
	Long:  `Parse an APK/XAPK/APKM file and display its metadata including package name, version, permissions, etc.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apkPath := args[0]
		
		parser := apk.NewParser()
		apkInfo, err := parser.ParseAPK(apkPath)
		if err != nil {
			return fmt.Errorf("failed to parse APK: %w", err)
		}
		
		// Pretty print the APK information
		jsonData, err := json.MarshalIndent(apkInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal APK info: %w", err)
		}
		
		fmt.Println(string(jsonData))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}