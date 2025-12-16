package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse [apk-file]",
	Short: i18n.T("cmd.parse.short"),
	Long:  i18n.T("cmd.parse.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apkPath := args[0]

		// Convert to absolute path
		absPath, err := filepath.Abs(apkPath)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.parse.errAPKPath"), err)
		}

		// Get absolute work directory
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.parse.errWorkDir"), err)
		}

		parser := apk.NewParser(absWorkDir)
		apkInfo, err := parser.ParseAPK(absPath)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.parse.errParse"), err)
		}

		// Pretty print the APK information
		jsonData, err := json.MarshalIndent(apkInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("cmd.parse.errMarshal"), err)
		}

		fmt.Println(string(jsonData))
		return nil
	},
}

func init() {
	repoCmd.AddCommand(parseCmd)
}
