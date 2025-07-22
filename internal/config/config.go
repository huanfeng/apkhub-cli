package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apkhub/apkhub-cli/pkg/models"
	"github.com/spf13/viper"
)

var defaultConfig = models.Config{
	Repository: models.RepositoryConfig{
		Name:             "My APK Repository",
		Description:      "Private APK repository",
		BaseURL:          "",
		KeepVersions:     0,
		SignatureHandling: "mark",
	},
	Scanning: models.ScanningConfig{
		Recursive:      true,
		FollowSymlinks: false,
		IncludePattern: []string{"*.apk", "*.xapk", "*.apkm"},
		ExcludePattern: []string{},
		ParseAPKInfo:   true,
	},
}

// Load loads configuration from file and environment
func Load(configPath string) (*models.Config, error) {
	viper.SetConfigType("yaml")
	
	// Set defaults
	viper.SetDefault("repository.name", defaultConfig.Repository.Name)
	viper.SetDefault("repository.description", defaultConfig.Repository.Description)
	viper.SetDefault("repository.base_url", defaultConfig.Repository.BaseURL)
	viper.SetDefault("repository.keep_versions", defaultConfig.Repository.KeepVersions)
	viper.SetDefault("repository.signature_handling", defaultConfig.Repository.SignatureHandling)
	viper.SetDefault("scanning.recursive", defaultConfig.Scanning.Recursive)
	viper.SetDefault("scanning.follow_symlinks", defaultConfig.Scanning.FollowSymlinks)
	viper.SetDefault("scanning.include_pattern", defaultConfig.Scanning.IncludePattern)
	viper.SetDefault("scanning.exclude_pattern", defaultConfig.Scanning.ExcludePattern)
	viper.SetDefault("scanning.parse_apk_info", defaultConfig.Scanning.ParseAPKInfo)
	
	// Try to load config file
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Look for config in current directory and parent directories
		viper.SetConfigName("apkhub")
		viper.AddConfigPath(".")
		
		// Also check in user's home directory
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "apkhub"))
		}
	}
	
	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is not an error, we'll use defaults
	}
	
	// Bind environment variables
	viper.SetEnvPrefix("APKHUB")
	viper.AutomaticEnv()
	
	// Unmarshal configuration
	var config models.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// SaveTemplate saves a configuration template
func SaveTemplate(path string) error {
	templateContent := `# ApkHub Configuration File

repository:
  # Repository name
  name: "My APK Repository"
  
  # Repository description
  description: "Private APK repository for my applications"
  
  # Base URL for downloads (will be prepended to relative paths)
  # Leave empty to use relative paths only
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
  exclude_pattern: []
  
  # Parse APK information (slower but provides more details)
  parse_apk_info: true
`
	
	return os.WriteFile(path, []byte(templateContent), 0644)
}