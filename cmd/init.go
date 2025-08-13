package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/pkg/models"
	"github.com/spf13/cobra"
)

var (
	initForce       bool
	initInteractive bool
	initUpdate      bool
	initTemplate    string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new ApkHub repository configuration",
	Long: `Initialize a new ApkHub repository by creating a configuration file.
Supports interactive configuration, templates, and updating existing configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "apkhub.yaml"
		
		// Check if config already exists
		configExists := false
		if _, err := os.Stat(configPath); err == nil {
			configExists = true
		}

		if configExists && !initForce && !initUpdate {
			fmt.Printf("⚠️  Configuration file %s already exists\n", configPath)
			fmt.Println("\nOptions:")
			fmt.Println("  --force      Overwrite existing configuration")
			fmt.Println("  --update     Update existing configuration")
			fmt.Println("  --interactive Interactive configuration wizard")
			
			if !initInteractive {
				return fmt.Errorf("use --force to overwrite or --update to modify existing configuration")
			}
		}

		if initUpdate && configExists {
			return updateExistingConfig(configPath)
		}

		if initInteractive {
			return interactiveInit(configPath, configExists)
		}

		return createConfig(configPath, configExists)
	},
}

// createConfig creates a new configuration file
func createConfig(configPath string, exists bool) error {
	var templateContent string
	var err error

	switch initTemplate {
	case "minimal":
		templateContent = getMinimalTemplate()
	case "advanced":
		templateContent = getAdvancedTemplate()
	case "development":
		templateContent = getDevelopmentTemplate()
	default:
		templateContent = getDefaultTemplate()
	}

	if exists && initForce {
		fmt.Printf("🔄 Overwriting existing configuration: %s\n", configPath)
	} else {
		fmt.Printf("📝 Creating new configuration: %s\n", configPath)
	}

	if err = os.WriteFile(configPath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}

	// Create repository directory structure
	if err := createRepositoryStructure(); err != nil {
		fmt.Printf("⚠️  Warning: failed to create repository structure: %v\n", err)
	}

	fmt.Printf("✅ Configuration file created: %s\n", configPath)
	fmt.Printf("📁 Repository structure initialized\n")
	
	if initTemplate != "minimal" {
		fmt.Println("\n💡 Next steps:")
		fmt.Println("   1. Edit the configuration file to customize settings")
		fmt.Println("   2. Run 'apkhub repo scan <directory>' to add APK files")
		fmt.Println("   3. Run 'apkhub repo stats' to view repository status")
	}

	return nil
}

// updateExistingConfig updates an existing configuration
func updateExistingConfig(configPath string) error {
	fmt.Printf("🔄 Updating existing configuration: %s\n", configPath)

	// Load existing config
	existingConfig, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Validate and fix configuration
	if err := validateAndFixConfig(existingConfig); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	// Backup existing config
	backupPath := configPath + ".backup"
	if err := copyConfigFile(configPath, backupPath); err != nil {
		fmt.Printf("⚠️  Warning: failed to create backup: %v\n", err)
	} else {
		fmt.Printf("💾 Backup created: %s\n", backupPath)
	}

	// Save updated config
	if err := config.SaveConfig(existingConfig, configPath); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}

	fmt.Printf("✅ Configuration updated successfully\n")
	return nil
}

// interactiveInit runs interactive configuration wizard
func interactiveInit(configPath string, exists bool) error {
	fmt.Println("🧙 ApkHub Repository Configuration Wizard")
	fmt.Println("=========================================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	
	// Load existing config if available
	var existingConfig *models.Config
	if exists {
		var err error
		existingConfig, err = config.Load(configPath)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to load existing config: %v\n", err)
		}
	}

	// Repository settings
	fmt.Println("📦 Repository Settings")
	fmt.Println("---------------------")
	
	repoName := promptWithDefault(reader, "Repository name", getConfigValue(existingConfig, "name", "My APK Repository"))
	repoDesc := promptWithDefault(reader, "Repository description", getConfigValue(existingConfig, "description", "Private APK repository"))
	baseURL := promptWithDefault(reader, "Base URL (optional)", getConfigValue(existingConfig, "base_url", ""))
	
	fmt.Println()
	fmt.Println("🔧 Scanning Settings")
	fmt.Println("-------------------")
	
	recursive := promptBool(reader, "Scan directories recursively", getConfigBool(existingConfig, "recursive", true))
	parseInfo := promptBool(reader, "Parse detailed APK information", getConfigBool(existingConfig, "parse_apk_info", true))

	// Create configuration
	newConfig := &models.Config{
		Repository: models.RepositoryConfig{
			Name:              repoName,
			Description:       repoDesc,
			BaseURL:           baseURL,
			KeepVersions:      0,
			SignatureHandling: "mark",
		},
		Scanning: models.ScanningConfig{
			Recursive:      recursive,
			FollowSymlinks: false,
			IncludePattern: []string{"*.apk", "*.xapk", "*.apkm"},
			ExcludePattern: []string{},
			ParseAPKInfo:   parseInfo,
		},
	}

	// Save configuration
	if err := config.SaveConfig(newConfig, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create repository structure
	if err := createRepositoryStructure(); err != nil {
		fmt.Printf("⚠️  Warning: failed to create repository structure: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("✅ Interactive configuration completed: %s\n", configPath)
	fmt.Printf("📁 Repository structure initialized\n")

	return nil
}

// Helper functions
func promptWithDefault(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		return defaultValue
	}
	return input
}

func promptBool(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}
	
	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	
	if input == "" {
		return defaultValue
	}
	
	return input == "y" || input == "yes"
}

func getConfigValue(cfg *models.Config, key, defaultValue string) string {
	if cfg == nil {
		return defaultValue
	}
	
	switch key {
	case "name":
		if cfg.Repository.Name != "" {
			return cfg.Repository.Name
		}
	case "description":
		if cfg.Repository.Description != "" {
			return cfg.Repository.Description
		}
	case "base_url":
		return cfg.Repository.BaseURL
	}
	
	return defaultValue
}

func getConfigBool(cfg *models.Config, key string, defaultValue bool) bool {
	if cfg == nil {
		return defaultValue
	}
	
	switch key {
	case "recursive":
		return cfg.Scanning.Recursive
	case "parse_apk_info":
		return cfg.Scanning.ParseAPKInfo
	}
	
	return defaultValue
}

func createRepositoryStructure() error {
	dirs := []string{"apks", "infos"}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

func validateAndFixConfig(cfg *models.Config) error {
	// Validate repository settings
	if cfg.Repository.Name == "" {
		cfg.Repository.Name = "My APK Repository"
	}
	
	if cfg.Repository.Description == "" {
		cfg.Repository.Description = "Private APK repository"
	}
	
	if cfg.Repository.SignatureHandling == "" {
		cfg.Repository.SignatureHandling = "mark"
	}
	
	// Validate scanning settings
	if len(cfg.Scanning.IncludePattern) == 0 {
		cfg.Scanning.IncludePattern = []string{"*.apk", "*.xapk", "*.apkm"}
	}
	
	return nil
}

func copyConfigFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// Template functions
func getDefaultTemplate() string {
	return `# ApkHub Configuration File

repository:
  # Repository name
  name: "My APK Repository"
  
  # Repository description
  description: "Private APK repository for my applications"
  
  # Base URL for downloads (will be prepended to relative paths)
  # Leave empty to use relative paths only
  # Example: "https://example.com/apk-repo"
  base_url: ""
  
  # Number of versions to keep (0 = keep all)
  keep_versions: 0
  
  # How to handle different signatures:
  # - "mark": Mark versions with different signatures (default)
  # - "separate": Create separate entries for different signatures
  # - "reject": Reject APKs with different signatures
  signature_handling: "mark"

scanning:
  # Scan directories recursively
  recursive: true
  
  # Follow symbolic links
  follow_symlinks: false
  
  # Include patterns (glob)
  include_pattern:
    - "*.apk"
    - "*.xapk"
    - "*.apkm"
  
  # Exclude patterns (glob)
  exclude_pattern:
    - "*.tmp"
    - "backup/*"
  
  # Parse APK information (slower but provides more details)
  parse_apk_info: true
`
}

func getMinimalTemplate() string {
	return `# ApkHub Minimal Configuration

repository:
  name: "My APK Repository"
  description: "Private APK repository"

scanning:
  recursive: true
  parse_apk_info: true
`
}

func getAdvancedTemplate() string {
	return `# ApkHub Advanced Configuration

repository:
  name: "My APK Repository"
  description: "Private APK repository for my applications"
  base_url: ""
  keep_versions: 3
  signature_handling: "mark"

scanning:
  recursive: true
  follow_symlinks: false
  include_pattern:
    - "*.apk"
    - "*.xapk"
    - "*.apkm"
  exclude_pattern:
    - "*.tmp"
    - "*.bak"
    - "backup/*"
    - "test/*"
    - ".git/*"
  parse_apk_info: true
`
}

func getDevelopmentTemplate() string {
	return `# ApkHub Development Configuration

repository:
  name: "Development APK Repository"
  description: "Development and testing APK repository"
  base_url: "http://localhost:8080"
  keep_versions: 5
  signature_handling: "separate"

scanning:
  recursive: true
  follow_symlinks: true
  include_pattern:
    - "*.apk"
    - "*.xapk"
    - "*.apkm"
  exclude_pattern:
    - "*.tmp"
    - "*.debug"
    - "build/*"
    - "dist/*"
    - ".git/*"
    - "node_modules/*"
  parse_apk_info: true
`
}

func init() {
	repoCmd.AddCommand(initCmd)
	
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing configuration")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "Interactive configuration wizard")
	initCmd.Flags().BoolVarP(&initUpdate, "update", "u", false, "Update existing configuration")
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "default", "Configuration template (default, minimal, advanced, development)")
}