package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/config"
	"github.com/huanfeng/apkhub-cli/internal/i18n"
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
			fmt.Println(i18n.T("cmd.init.configExists", map[string]interface{}{
				"path": configPath,
			}))
			fmt.Println()
			fmt.Println(i18n.T("cmd.init.optionsTitle"))
			fmt.Println(i18n.T("cmd.init.optionForce"))
			fmt.Println(i18n.T("cmd.init.optionUpdate"))
			fmt.Println(i18n.T("cmd.init.optionInteractive"))

			if !initInteractive {
				return fmt.Errorf(i18n.T("cmd.init.configExistsAdvice"))
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
	case "local":
		templateContent = getLocalTemplate()
	default:
		templateContent = getDefaultTemplate()
	}

	if exists && initForce {
		fmt.Printf("%s\n", i18n.T("cmd.init.overwriteConfig", map[string]interface{}{
			"path": configPath,
		}))
	} else {
		fmt.Printf("%s\n", i18n.T("cmd.init.createConfig", map[string]interface{}{
			"path": configPath,
		}))
	}

	if err = os.WriteFile(configPath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.init.errCreateConfig"), err)
	}

	// Create repository directory structure
	if err := createRepositoryStructure(); err != nil {
		fmt.Printf("%s\n", i18n.T("cmd.init.errCreateRepoStructure", map[string]interface{}{
			"error": err,
		}))
	}

	fmt.Printf("%s\n", i18n.T("cmd.init.createdConfig", map[string]interface{}{
		"path": configPath,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.init.repoInitialized"))

	if initTemplate != "minimal" {
		fmt.Println()
		fmt.Println(i18n.T("cmd.init.nextSteps"))
		fmt.Println(i18n.T("cmd.init.nextStepsEdit"))
		fmt.Println(i18n.T("cmd.init.nextStepsScan"))
		fmt.Println(i18n.T("cmd.init.nextStepsStats"))
	}

	return nil
}

// updateExistingConfig updates an existing configuration
func updateExistingConfig(configPath string) error {
	fmt.Printf("%s\n", i18n.T("cmd.init.updateConfig", map[string]interface{}{
		"path": configPath,
	}))

	// Load existing config
	existingConfig, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.init.errLoadExisting"), err)
	}

	// Validate and fix configuration
	if err := validateAndFixConfig(existingConfig); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.init.errValidateConfig"), err)
	}

	// Backup existing config
	backupPath := configPath + ".backup"
	if err := copyConfigFile(configPath, backupPath); err != nil {
		fmt.Printf("%s\n", i18n.T("cmd.init.errCreateBackup", map[string]interface{}{
			"error": err,
		}))
	} else {
		fmt.Printf("%s\n", i18n.T("cmd.init.backupCreated", map[string]interface{}{
			"path": backupPath,
		}))
	}

	// Save updated config
	if err := config.SaveConfig(existingConfig, configPath); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.init.errSaveUpdated"), err)
	}

	fmt.Printf("%s\n", i18n.T("cmd.init.updateSuccess"))
	return nil
}

// interactiveInit runs interactive configuration wizard
func interactiveInit(configPath string, exists bool) error {
	fmt.Println(i18n.T("cmd.init.wizardTitle"))
	fmt.Println(strings.Repeat("=", len(i18n.T("cmd.init.wizardTitle"))))
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
	fmt.Println(i18n.T("cmd.init.repoSettings"))
	fmt.Println(strings.Repeat("-", len(i18n.T("cmd.init.repoSettings"))))

	repoName := promptWithDefault(reader, i18n.T("cmd.init.prompt.repoName"), getConfigValue(existingConfig, "name", "My APK Repository"))
	repoDesc := promptWithDefault(reader, i18n.T("cmd.init.prompt.repoDescription"), getConfigValue(existingConfig, "description", "Private APK repository"))
	baseURL := promptWithDefault(reader, i18n.T("cmd.init.prompt.baseURL"), getConfigValue(existingConfig, "base_url", ""))

	fmt.Println()
	fmt.Println(i18n.T("cmd.init.scanSettings"))
	fmt.Println(strings.Repeat("-", len(i18n.T("cmd.init.scanSettings"))))

	recursive := promptBool(reader, i18n.T("cmd.init.prompt.recursive"), getConfigBool(existingConfig, "recursive", true))
	parseInfo := promptBool(reader, i18n.T("cmd.init.prompt.parseInfo"), getConfigBool(existingConfig, "parse_apk_info", true))

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
		return fmt.Errorf("%s: %w", i18n.T("cmd.init.errSaveConfig"), err)
	}

	// Create repository structure
	if err := createRepositoryStructure(); err != nil {
		fmt.Printf("%s\n", i18n.T("cmd.init.errCreateRepoStructure", map[string]interface{}{
			"error": err,
		}))
	}

	fmt.Println()
	fmt.Printf("%s\n", i18n.T("cmd.init.wizardCompleted", map[string]interface{}{
		"path": configPath,
	}))
	fmt.Printf("%s\n", i18n.T("cmd.init.repoInitialized"))

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
			return fmt.Errorf("%s: %w", i18n.T("cmd.init.errCreateDirectory", map[string]interface{}{
				"dir": dir,
			}), err)
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
  # Options:
  # - "" (empty): Use relative paths only
  # - "local": Generate file:// URLs based on repository path (for local buckets)
  # - "http://localhost:8080": Local HTTP server
  # - "https://example.com/apk-repo": Remote HTTP server
  # - "file:///absolute/path": Specific file path
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
  # For local development, you can use:
  # - "local" for file:// URLs based on repository path
  # - "http://localhost:8080" for local HTTP server
  # - "file:///absolute/path" for specific file path
  base_url: "local"
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

func getLocalTemplate() string {
	return `# ApkHub Local Repository Configuration

repository:
  name: "Local APK Repository"
  description: "Local file-based APK repository"
  # Use "local" for file:// URLs based on repository path
  # This will generate file:// URLs that can be used by bucket clients
  base_url: "local"
  keep_versions: 0
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
    - ".git/*"
  parse_apk_info: true
`
}

func init() {
	repoCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, i18n.T("cmd.init.flag.force"))
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, i18n.T("cmd.init.flag.interactive"))
	initCmd.Flags().BoolVarP(&initUpdate, "update", "u", false, i18n.T("cmd.init.flag.update"))
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "default", i18n.T("cmd.init.flag.template"))
}
