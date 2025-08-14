package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Name        string    `json:"name"`
	Required    bool      `json:"required"`
	Available   bool      `json:"available"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	UsedBy      []string  `json:"used_by"`
	LastChecked time.Time `json:"last_checked"`
	Error       string    `json:"error,omitempty"`
}

// DependencyManager manages system dependencies
type DependencyManager interface {
	CheckDependency(name string) DependencyStatus
	CheckForCommand(command string) []DependencyStatus
	CheckAll() map[string]DependencyStatus
	GetInstallInstructions(name string) []string
	ClearCache()
	SetCacheTTL(duration time.Duration)
}

// DefaultDependencyManager is the default implementation
type DefaultDependencyManager struct {
	cache    map[string]DependencyStatus
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() DependencyManager {
	return &DefaultDependencyManager{
		cache:    make(map[string]DependencyStatus),
		cacheTTL: 5 * time.Minute, // Default cache TTL
	}
}

// Dependency definitions
var dependencies = map[string]DependencyDefinition{
	"aapt2": {
		Name:        "aapt2",
		Required:    false,
		UsedBy:      []string{"repo scan", "repo add", "repo parse", "info"},
		Description: "Android Asset Packaging Tool 2 - for APK parsing",
		Executables: []string{"aapt2"},
		CommonPaths: getCommonAAPTPaths("aapt2"),
		VersionArgs: []string{"version"},
	},
	"aapt": {
		Name:        "aapt",
		Required:    false,
		UsedBy:      []string{"repo scan (fallback)", "repo add (fallback)", "repo parse (fallback)"},
		Description: "Android Asset Packaging Tool - fallback APK parser",
		Executables: []string{"aapt"},
		CommonPaths: getCommonAAPTPaths("aapt"),
		VersionArgs: []string{"version"},
	},
	"adb": {
		Name:        "adb",
		Required:    true,
		UsedBy:      []string{"install"},
		Description: "Android Debug Bridge - for device communication",
		Executables: []string{"adb"},
		CommonPaths: getCommonADBPaths(),
		VersionArgs: []string{"version"},
	},
}

// DependencyDefinition defines how to check for a dependency
type DependencyDefinition struct {
	Name        string
	Required    bool
	UsedBy      []string
	Description string
	Executables []string
	CommonPaths []string
	VersionArgs []string
}

// CheckDependency checks the status of a specific dependency
func (dm *DefaultDependencyManager) CheckDependency(name string) DependencyStatus {
	// Check cache first
	dm.cacheMu.RLock()
	if cached, exists := dm.cache[name]; exists {
		if time.Since(cached.LastChecked) < dm.cacheTTL {
			dm.cacheMu.RUnlock()
			return cached
		}
	}
	dm.cacheMu.RUnlock()

	// Get dependency definition
	def, exists := dependencies[name]
	if !exists {
		return DependencyStatus{
			Name:        name,
			Available:   false,
			Error:       "Unknown dependency",
			LastChecked: time.Now(),
		}
	}

	// Perform actual check
	status := dm.checkDependencyActual(def)

	// Update cache
	dm.cacheMu.Lock()
	dm.cache[name] = status
	dm.cacheMu.Unlock()

	return status
}

// checkDependencyActual performs the actual dependency check
func (dm *DefaultDependencyManager) checkDependencyActual(def DependencyDefinition) DependencyStatus {
	status := DependencyStatus{
		Name:        def.Name,
		Required:    def.Required,
		UsedBy:      def.UsedBy,
		Available:   false,
		LastChecked: time.Now(),
	}

	// Try to find the executable
	for _, executable := range def.Executables {
		// First try PATH
		if path, err := exec.LookPath(executable); err == nil {
			if version := dm.getToolVersion(path, def.VersionArgs); version != "" {
				status.Available = true
				status.Path = path
				status.Version = version
				return status
			}
		}

		// Then try common paths
		for _, commonPath := range def.CommonPaths {
			// Handle wildcard paths
			if strings.Contains(commonPath, "*") {
				matches := dm.expandWildcardPath(commonPath)
				for _, match := range matches {
					if version := dm.getToolVersion(match, def.VersionArgs); version != "" {
						status.Available = true
						status.Path = match
						status.Version = version
						return status
					}
				}
			} else {
				if _, err := os.Stat(commonPath); err == nil {
					if version := dm.getToolVersion(commonPath, def.VersionArgs); version != "" {
						status.Available = true
						status.Path = commonPath
						status.Version = version
						return status
					}
				}
			}
		}
	}

	status.Error = fmt.Sprintf("%s not found in PATH or common locations", def.Name)
	return status
}

// getToolVersion gets the version of a tool
func (dm *DefaultDependencyManager) getToolVersion(toolPath string, versionArgs []string) string {
	if len(versionArgs) == 0 {
		return "unknown"
	}

	cmd := exec.Command(toolPath, versionArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// Extract first line as version
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		version := strings.TrimSpace(lines[0])
		if version != "" {
			return version
		}
	}

	return "unknown"
}

// expandWildcardPath expands paths with wildcards
func (dm *DefaultDependencyManager) expandWildcardPath(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return []string{}
	}

	var validPaths []string
	for _, match := range matches {
		if _, err := os.Stat(match); err == nil {
			validPaths = append(validPaths, match)
		}
	}

	return validPaths
}

// CheckForCommand checks dependencies required for a specific command
func (dm *DefaultDependencyManager) CheckForCommand(command string) []DependencyStatus {
	var requiredDeps []string

	// Map commands to their dependencies
	commandDeps := map[string][]string{
		"install":    {"adb"},
		"repo scan":  {"aapt2", "aapt"}, // aapt as fallback
		"repo add":   {"aapt2", "aapt"},
		"repo parse": {"aapt2", "aapt"},
		"info":       {"aapt2", "aapt"},
		"search":     {}, // No external dependencies
		"download":   {}, // No external dependencies
		"bucket":     {}, // No external dependencies
	}

	if deps, exists := commandDeps[command]; exists {
		requiredDeps = deps
	}

	var statuses []DependencyStatus
	for _, dep := range requiredDeps {
		statuses = append(statuses, dm.CheckDependency(dep))
	}

	return statuses
}

// CheckAll checks all known dependencies
func (dm *DefaultDependencyManager) CheckAll() map[string]DependencyStatus {
	result := make(map[string]DependencyStatus)

	for name := range dependencies {
		result[name] = dm.CheckDependency(name)
	}

	return result
}

// GetInstallInstructions returns installation instructions for a dependency
func (dm *DefaultDependencyManager) GetInstallInstructions(name string) []string {
	switch name {
	case "aapt2", "aapt":
		return getAAPTInstallInstructions()
	case "adb":
		return getADBInstallInstructions()
	default:
		return []string{"Unknown dependency: " + name}
	}
}

// ClearCache clears the dependency cache
func (dm *DefaultDependencyManager) ClearCache() {
	dm.cacheMu.Lock()
	defer dm.cacheMu.Unlock()
	dm.cache = make(map[string]DependencyStatus)
}

// SetCacheTTL sets the cache time-to-live
func (dm *DefaultDependencyManager) SetCacheTTL(duration time.Duration) {
	dm.cacheTTL = duration
}

// Platform-specific path functions
func getCommonAAPTPaths(toolName string) []string {
	var paths []string

	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/bin/" + toolName,
			"/usr/local/bin/" + toolName,
			"/opt/android-sdk/build-tools/*/" + toolName,
		}

		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Android/Sdk/build-tools/*/"+toolName),
				filepath.Join(home, ".android-sdk/build-tools/*/"+toolName),
			)
		}

	case "darwin":
		paths = []string{
			"/usr/local/bin/" + toolName,
			"/opt/homebrew/bin/" + toolName,
		}

		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Library/Android/sdk/build-tools/*/"+toolName),
			)
		}

	case "windows":
		paths = []string{
			"C:\\Android\\Sdk\\build-tools\\*\\" + toolName + ".exe",
		}

		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			paths = append(paths, filepath.Join(localAppData, "Android\\Sdk\\build-tools\\*\\"+toolName+".exe"))
		}
	}

	return paths
}

func getCommonADBPaths() []string {
	var paths []string

	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/bin/adb",
			"/usr/local/bin/adb",
			"/opt/android-sdk/platform-tools/adb",
		}

		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Android/Sdk/platform-tools/adb"),
				filepath.Join(home, ".android-sdk/platform-tools/adb"),
			)
		}

	case "darwin":
		paths = []string{
			"/usr/local/bin/adb",
			"/opt/homebrew/bin/adb",
		}

		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Library/Android/sdk/platform-tools/adb"),
			)
		}

	case "windows":
		paths = []string{
			"C:\\Android\\Sdk\\platform-tools\\adb.exe",
		}

		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			paths = append(paths, filepath.Join(localAppData, "Android\\Sdk\\platform-tools\\adb.exe"))
		}
	}

	return paths
}

func getAAPTInstallInstructions() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{
			"Ubuntu/Debian: sudo apt-get install aapt",
			"Homebrew: brew install android-commandlinetools",
			"Manual: Download Android SDK Build Tools from https://developer.android.com/studio#command-tools",
		}
	case "darwin":
		return []string{
			"Homebrew: brew install --cask android-commandlinetools",
			"Manual: Download Android SDK Build Tools from https://developer.android.com/studio#command-tools",
		}
	case "windows":
		return []string{
			"Download Android SDK Build Tools from https://developer.android.com/studio#command-tools",
			"Extract to C:\\Android\\Sdk\\ and add build-tools to PATH",
		}
	default:
		return []string{
			"Download Android SDK Build Tools from https://developer.android.com/studio#command-tools",
		}
	}
}

func getADBInstallInstructions() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{
			"Ubuntu/Debian: sudo apt-get install adb",
			"Homebrew: brew install android-platform-tools",
			"Manual: Download Android SDK Platform Tools",
		}
	case "darwin":
		return []string{
			"Homebrew: brew install android-platform-tools",
			"Manual: Download Android SDK Platform Tools",
		}
	case "windows":
		return []string{
			"Download Android SDK Platform Tools",
			"Extract and add to PATH",
		}
	default:
		return []string{
			"Download Android SDK Platform Tools",
		}
	}
}
