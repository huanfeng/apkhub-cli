package cmd

import (
	"fmt"
	"os"

	"github.com/apkhub/apkhub-cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new ApkHub repository configuration",
	Long:  `Initialize a new ApkHub repository by creating a configuration file template.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "apkhub.yaml"
		
		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("configuration file %s already exists", configPath)
		}
		
		// Save template
		if err := config.SaveTemplate(configPath); err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}
		
		fmt.Printf("Configuration file created: %s\n", configPath)
		fmt.Println("Edit this file to customize your repository settings.")
		
		return nil
	},
}

func init() {
	repoCmd.AddCommand(initCmd)
}