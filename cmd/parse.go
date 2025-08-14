package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse [apk-file]",
	Short: "Parse a single APK file and display its information",
	Long:  `Parse an APK/XAPK/APKM file and display its metadata including package name, version, permissions, etc.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apkPath := args[0]

		// Convert to absolute path
		absPath, err := filepath.Abs(apkPath)
		if err != nil {
			return fmt.Errorf("failed to resolve APK path: %w", err)
		}

		// Get absolute work directory
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve work directory: %w", err)
		}

		parser := apk.NewParser(absWorkDir)
		apkInfo, err := parser.ParseAPK(absPath)
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
	repoCmd.AddCommand(parseCmd)
}
