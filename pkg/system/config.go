package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigManager provides configuration management capabilities
type ConfigManager struct {
	logger Logger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger Logger) *ConfigManager {
	return &ConfigManager{
		logger: logger,
	}
}

// ConfigValidationResult contains the result of configuration validation
type ConfigValidationResult struct {
	Valid       bool                   `json:"valid"`
	Errors      []string               `json:"errors"`
	Warnings    []string               `json:"warnings"`
	Suggestions []string               `json:"suggestions"`
	Details     map[string]interface{} `json:"details"`
	ConfigPath  string                 `json:"config_path"`
	Version     string                 `json:"version,omitempty"`
}

// ConfigBackup represents a configuration backup
type ConfigBackup struct {
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Size      int64     `json:"size"`
}

// ConfigMigration represents a configuration migration
type ConfigMigration struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ValidateConfig validates a configuration file
func (cm *ConfigManager) ValidateConfig(configPath string) *ConfigValidationResult {
	result := &ConfigValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
		Details:     make(map[string]interface{}),
		ConfigPath:  configPath,
	}
	
	if cm.logger != nil {
		cm.logger.Debug("Validating configuration file: %s", configPath)
	}
	
	// Check if file exists
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Valid = false
			result.Errors = append(result.Errors, "Configuration file does not exist")
			result.Suggestions = append(result.Suggestions, "Run 'apkhub init' to create a configuration file")
			return result
		} else {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Cannot access configuration file: %v", err))
			result.Suggestions = append(result.Suggestions, "Check file permissions")
			return result
		}
	}
	
	result.Details["file_size"] = info.Size()
	result.Details["file_mode"] = info.Mode().String()
	result.Details["modified_time"] = info.ModTime().Format("2006-01-02 15:04:05")
	
	// Check file permissions
	if info.Mode().Perm()&0044 == 0 {
		result.Warnings = append(result.Warnings, "Configuration file is not readable by others")
	}
	
	// Read and parse the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Cannot read configuration file: %v", err))
		return result
	}
	
	// Try to parse as YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid YAML syntax: %v", err))
		result.Suggestions = append(result.Suggestions, 
			"Check YAML syntax using an online validator",
			"Ensure proper indentation and structure")
		return result
	}
	
	result.Details["config_keys"] = len(config)
	
	// Validate configuration structure
	cm.validateConfigStructure(config, result)
	
	// Check for deprecated settings
	cm.checkDeprecatedSettings(config, result)
	
	// Validate paths and directories
	cm.validateConfigPaths(config, result)
	
	// Check for security issues
	cm.checkSecurityIssues(config, result)
	
	return result
}

// validateConfigStructure validates the basic structure of the configuration
func (cm *ConfigManager) validateConfigStructure(config map[string]interface{}, result *ConfigValidationResult) {
	// Define required sections
	requiredSections := []string{"repositories", "cache", "adb"}
	
	for _, section := range requiredSections {
		if _, exists := config[section]; !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Missing recommended section: %s", section))
			result.Suggestions = append(result.Suggestions, fmt.Sprintf("Add %s section to configuration", section))
		}
	}
	
	// Validate repositories section
	if repos, exists := config["repositories"]; exists {
		if repoMap, ok := repos.(map[string]interface{}); ok {
			result.Details["repository_count"] = len(repoMap)
			
			for name, repo := range repoMap {
				if repoConfig, ok := repo.(map[string]interface{}); ok {
					cm.validateRepositoryConfig(name, repoConfig, result)
				}
			}
		} else {
			result.Errors = append(result.Errors, "repositories section must be a map")
			result.Valid = false
		}
	}
	
	// Validate cache section
	if cache, exists := config["cache"]; exists {
		if cacheConfig, ok := cache.(map[string]interface{}); ok {
			cm.validateCacheConfig(cacheConfig, result)
		} else {
			result.Errors = append(result.Errors, "cache section must be a map")
			result.Valid = false
		}
	}
	
	// Validate ADB section
	if adb, exists := config["adb"]; exists {
		if adbConfig, ok := adb.(map[string]interface{}); ok {
			cm.validateADBConfig(adbConfig, result)
		} else {
			result.Errors = append(result.Errors, "adb section must be a map")
			result.Valid = false
		}
	}
}

// validateRepositoryConfig validates a repository configuration
func (cm *ConfigManager) validateRepositoryConfig(name string, config map[string]interface{}, result *ConfigValidationResult) {
	// Check required fields
	requiredFields := []string{"url", "type"}
	
	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("Repository '%s' missing required field: %s", name, field))
			result.Valid = false
		}
	}
	
	// Validate URL format
	if url, exists := config["url"]; exists {
		if urlStr, ok := url.(string); ok {
			if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") && !strings.HasPrefix(urlStr, "file://") {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Repository '%s' URL may be invalid: %s", name, urlStr))
			}
		}
	}
	
	// Validate type
	if repoType, exists := config["type"]; exists {
		if typeStr, ok := repoType.(string); ok {
			validTypes := []string{"s3", "http", "local", "git"}
			valid := false
			for _, validType := range validTypes {
				if typeStr == validType {
					valid = true
					break
				}
			}
			if !valid {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Repository '%s' has unknown type: %s", name, typeStr))
			}
		}
	}
}

// validateCacheConfig validates cache configuration
func (cm *ConfigManager) validateCacheConfig(config map[string]interface{}, result *ConfigValidationResult) {
	// Check cache directory
	if dir, exists := config["directory"]; exists {
		if dirStr, ok := dir.(string); ok {
			if !filepath.IsAbs(dirStr) {
				result.Warnings = append(result.Warnings, "Cache directory should be an absolute path")
			}
			
			// Check if directory exists or can be created
			if err := os.MkdirAll(dirStr, 0755); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Cannot create cache directory: %v", err))
				result.Valid = false
			}
		}
	}
	
	// Check cache size limits
	if maxSize, exists := config["max_size"]; exists {
		if sizeStr, ok := maxSize.(string); ok {
			if !strings.HasSuffix(sizeStr, "MB") && !strings.HasSuffix(sizeStr, "GB") {
				result.Warnings = append(result.Warnings, "Cache max_size should include units (MB, GB)")
			}
		}
	}
}

// validateADBConfig validates ADB configuration
func (cm *ConfigManager) validateADBConfig(config map[string]interface{}, result *ConfigValidationResult) {
	// Check ADB path
	if path, exists := config["path"]; exists {
		if pathStr, ok := path.(string); ok {
			if !filepath.IsAbs(pathStr) {
				// Try to find in PATH
				if _, err := exec.LookPath(pathStr); err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("ADB path may not be valid: %s", pathStr))
				}
			} else {
				// Check if absolute path exists
				if _, err := os.Stat(pathStr); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("ADB path does not exist: %s", pathStr))
					result.Valid = false
				}
			}
		}
	}
	
	// Check default device
	if device, exists := config["default_device"]; exists {
		if deviceStr, ok := device.(string); ok {
			if deviceStr != "" {
				result.Details["default_device"] = deviceStr
			}
		}
	}
}

// checkDeprecatedSettings checks for deprecated configuration settings
func (cm *ConfigManager) checkDeprecatedSettings(config map[string]interface{}, result *ConfigValidationResult) {
	deprecatedSettings := map[string]string{
		"output_format": "Use 'format' instead",
		"verbose_mode": "Use command-line flags instead",
		"debug_mode":   "Use command-line flags instead",
	}
	
	for deprecated, replacement := range deprecatedSettings {
		if _, exists := config[deprecated]; exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Deprecated setting '%s': %s", deprecated, replacement))
			result.Suggestions = append(result.Suggestions, fmt.Sprintf("Remove '%s' from configuration", deprecated))
		}
	}
}

// validateConfigPaths validates paths in the configuration
func (cm *ConfigManager) validateConfigPaths(config map[string]interface{}, result *ConfigValidationResult) {
	pathFields := []string{"cache.directory", "adb.path", "temp_directory"}
	
	for _, field := range pathFields {
		if value := cm.getNestedValue(config, field); value != nil {
			if pathStr, ok := value.(string); ok && pathStr != "" {
				if strings.Contains(pathStr, "~") {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Path contains '~' which may not expand correctly: %s", field))
				}
				
				// Check for relative paths that might be problematic
				if !filepath.IsAbs(pathStr) && !strings.Contains(pathStr, "/") {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Relative path may cause issues: %s = %s", field, pathStr))
				}
			}
		}
	}
}

// checkSecurityIssues checks for potential security issues in configuration
func (cm *ConfigManager) checkSecurityIssues(config map[string]interface{}, result *ConfigValidationResult) {
	// Check for hardcoded credentials
	sensitiveFields := []string{"password", "token", "key", "secret"}
	
	cm.checkSensitiveFields(config, "", sensitiveFields, result)
	
	// Check file permissions
	info, _ := os.Stat(result.ConfigPath)
	if info != nil && info.Mode().Perm()&0077 != 0 {
		result.Warnings = append(result.Warnings, "Configuration file is readable/writable by others")
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("Run 'chmod 600 %s' to secure the file", result.ConfigPath))
	}
}

// checkSensitiveFields recursively checks for sensitive fields
func (cm *ConfigManager) checkSensitiveFields(obj interface{}, prefix string, sensitiveFields []string, result *ConfigValidationResult) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fullKey := key
			if prefix != "" {
				fullKey = prefix + "." + key
			}
			
			// Check if this key is sensitive
			for _, sensitive := range sensitiveFields {
				if strings.Contains(strings.ToLower(key), sensitive) {
					if strValue, ok := value.(string); ok && strValue != "" {
						result.Warnings = append(result.Warnings, fmt.Sprintf("Potential sensitive data in configuration: %s", fullKey))
						result.Suggestions = append(result.Suggestions, "Consider using environment variables for sensitive data")
					}
				}
			}
			
			// Recurse into nested objects
			cm.checkSensitiveFields(value, fullKey, sensitiveFields, result)
		}
	case []interface{}:
		for i, item := range v {
			cm.checkSensitiveFields(item, fmt.Sprintf("%s[%d]", prefix, i), sensitiveFields, result)
		}
	}
}

// getNestedValue gets a nested value from a map using dot notation
func (cm *ConfigManager) getNestedValue(config map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := config
	
	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	
	return nil
}

// BackupConfig creates a backup of the configuration file
func (cm *ConfigManager) BackupConfig(configPath string) (*ConfigBackup, error) {
	if cm.logger != nil {
		cm.logger.Debug("Creating backup of configuration: %s", configPath)
	}
	
	// Check if source file exists
	info, err := os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access config file: %w", err)
	}
	
	// Create backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, timestamp)
	
	// Copy file
	sourceData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}
	
	if err := os.WriteFile(backupPath, sourceData, info.Mode()); err != nil {
		return nil, fmt.Errorf("cannot create backup: %w", err)
	}
	
	backup := &ConfigBackup{
		Path:      backupPath,
		Timestamp: time.Now(),
		Size:      info.Size(),
	}
	
	if cm.logger != nil {
		cm.logger.Debug("Configuration backup created: %s", backupPath)
	}
	
	return backup, nil
}

// RestoreConfig restores a configuration from backup
func (cm *ConfigManager) RestoreConfig(backupPath, configPath string) error {
	if cm.logger != nil {
		cm.logger.Debug("Restoring configuration from backup: %s", backupPath)
	}
	
	// Check if backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}
	
	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("cannot read backup file: %w", err)
	}
	
	// Write to config path
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("cannot restore config file: %w", err)
	}
	
	if cm.logger != nil {
		cm.logger.Debug("Configuration restored successfully")
	}
	
	return nil
}

// ListBackups lists available configuration backups
func (cm *ConfigManager) ListBackups(configPath string) ([]ConfigBackup, error) {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	pattern := base + ".backup.*"
	
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	
	var backups []ConfigBackup
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		
		backup := ConfigBackup{
			Path:      match,
			Timestamp: info.ModTime(),
			Size:      info.Size(),
		}
		
		backups = append(backups, backup)
	}
	
	return backups, nil
}

// FormatValidationResult formats validation result for display
func (cm *ConfigManager) FormatValidationResult(result *ConfigValidationResult) string {
	if result == nil {
		return "No validation result available"
	}
	
	output := fmt.Sprintf("Configuration Validation Report:\n")
	output += fmt.Sprintf("File: %s\n", result.ConfigPath)
	
	if result.Valid {
		output += fmt.Sprintf("Status: ✅ Valid\n")
	} else {
		output += fmt.Sprintf("Status: ❌ Invalid\n")
	}
	
	if len(result.Errors) > 0 {
		output += fmt.Sprintf("\nErrors:\n")
		for i, err := range result.Errors {
			output += fmt.Sprintf("  %d. %s\n", i+1, err)
		}
	}
	
	if len(result.Warnings) > 0 {
		output += fmt.Sprintf("\nWarnings:\n")
		for i, warning := range result.Warnings {
			output += fmt.Sprintf("  %d. %s\n", i+1, warning)
		}
	}
	
	if len(result.Suggestions) > 0 {
		output += fmt.Sprintf("\nSuggestions:\n")
		for i, suggestion := range result.Suggestions {
			output += fmt.Sprintf("  %d. %s\n", i+1, suggestion)
		}
	}
	
	if len(result.Details) > 0 {
		output += fmt.Sprintf("\nDetails:\n")
		for key, value := range result.Details {
			output += fmt.Sprintf("  %s: %v\n", key, value)
		}
	}
	
	return output
}