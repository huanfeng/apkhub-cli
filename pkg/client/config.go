package client

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the client configuration
type Config struct {
	DefaultBucket string             `yaml:"default_bucket"`
	Buckets       map[string]*Bucket `yaml:"buckets"`
	Client        ClientSettings     `yaml:"client"`
	ADB           ADBSettings        `yaml:"adb"`
}

// Bucket represents a repository source
type Bucket struct {
	Name        string    `yaml:"name"`
	URL         string    `yaml:"url"`
	Enabled     bool      `yaml:"enabled"`
	LastUpdated time.Time `yaml:"last_updated,omitempty"`
}

// ClientSettings contains client-specific settings
type ClientSettings struct {
	DownloadDir string `yaml:"download_dir"`
	CacheDir    string `yaml:"cache_dir"`
	CacheTTL    int    `yaml:"cache_ttl"` // seconds
}

// ADBSettings contains ADB configuration
type ADBSettings struct {
	Path          string `yaml:"path"`
	DefaultDevice string `yaml:"default_device"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	apkhubDir := filepath.Join(homeDir, ".apkhub")
	
	return &Config{
		DefaultBucket: "main",
		Buckets:       make(map[string]*Bucket),
		Client: ClientSettings{
			DownloadDir: filepath.Join(apkhubDir, "downloads"),
			CacheDir:    filepath.Join(apkhubDir, "cache"),
			CacheTTL:    3600, // 1 hour
		},
		ADB: ADBSettings{
			Path:          "adb",
			DefaultDevice: "",
		},
	}
}

// ConfigPath returns the default config file path
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".apkhub", "config.yaml")
}

// Load loads configuration from file
func Load() (*Config, error) {
	configPath := ConfigPath()
	
	// Create default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	
	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Expand paths
	config.expandPaths()
	
	return config, nil
}

// Save saves configuration to file
func (c *Config) Save() error {
	configPath := ConfigPath()
	
	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	
	return nil
}

// expandPaths expands ~ in paths
func (c *Config) expandPaths() {
	homeDir, _ := os.UserHomeDir()
	
	if c.Client.DownloadDir != "" && c.Client.DownloadDir[0] == '~' {
		c.Client.DownloadDir = filepath.Join(homeDir, c.Client.DownloadDir[1:])
	}
	if c.Client.CacheDir != "" && c.Client.CacheDir[0] == '~' {
		c.Client.CacheDir = filepath.Join(homeDir, c.Client.CacheDir[1:])
	}
}

// EnsureDirectories creates necessary directories
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Client.DownloadDir,
		c.Client.CacheDir,
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// AddBucket adds a new bucket to configuration
func (c *Config) AddBucket(name, url, displayName string) error {
	if c.Buckets == nil {
		c.Buckets = make(map[string]*Bucket)
	}
	
	if _, exists := c.Buckets[name]; exists {
		return fmt.Errorf("bucket %s already exists", name)
	}
	
	c.Buckets[name] = &Bucket{
		Name:    displayName,
		URL:     url,
		Enabled: true,
	}
	
	return c.Save()
}

// RemoveBucket removes a bucket from configuration
func (c *Config) RemoveBucket(name string) error {
	if _, exists := c.Buckets[name]; !exists {
		return fmt.Errorf("bucket %s not found", name)
	}
	
	delete(c.Buckets, name)
	
	// Update default bucket if necessary
	if c.DefaultBucket == name {
		c.DefaultBucket = ""
		for k := range c.Buckets {
			c.DefaultBucket = k
			break
		}
	}
	
	return c.Save()
}

// GetEnabledBuckets returns all enabled buckets
func (c *Config) GetEnabledBuckets() map[string]*Bucket {
	enabled := make(map[string]*Bucket)
	for name, bucket := range c.Buckets {
		if bucket.Enabled {
			enabled[name] = bucket
		}
	}
	return enabled
}