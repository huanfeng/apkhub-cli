package apk

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// AAPTParser uses aapt2 command line tool to parse APK
type AAPTParser struct {
	aaptPath string
}

// NewAAPTParser creates a new AAPT parser
func NewAAPTParser() *AAPTParser {
	return &AAPTParser{
		aaptPath: "aapt2", // Assume it's in PATH
	}
}

// SetAAPTPath sets custom aapt2 path
func (p *AAPTParser) SetAAPTPath(path string) {
	p.aaptPath = path
}

// CheckAAPT checks if aapt2 is available with enhanced detection
func (p *AAPTParser) CheckAAPT() error {
	// Try to find aapt2 first
	if path, err := p.findAAPTTool("aapt2"); err == nil {
		p.aaptPath = path
		return nil
	}
	
	// Try to find aapt as fallback
	if path, err := p.findAAPTTool("aapt"); err == nil {
		p.aaptPath = path
		return nil
	}
	
	// Neither found, return detailed error with installation hints
	return p.createAAPTNotFoundError()
}

// findAAPTTool attempts to find aapt tool in various locations
func (p *AAPTParser) findAAPTTool(toolName string) (string, error) {
	// First try PATH
	if path, err := exec.LookPath(toolName); err == nil {
		// Verify it works
		if err := exec.Command(path, "version").Run(); err == nil {
			return path, nil
		}
	}
	
	// Try common installation paths
	commonPaths := p.getCommonAAPTPaths(toolName)
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			// Verify it works
			if err := exec.Command(path, "version").Run(); err == nil {
				return path, nil
			}
		}
	}
	
	return "", fmt.Errorf("%s not found", toolName)
}

// getCommonAAPTPaths returns common installation paths for aapt tools
func (p *AAPTParser) getCommonAAPTPaths(toolName string) []string {
	var paths []string
	
	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/bin/" + toolName,
			"/usr/local/bin/" + toolName,
			"/opt/android-sdk/build-tools/*/aapt2",
			"/opt/android-sdk/build-tools/*/aapt",
			filepath.Join(os.Getenv("HOME"), "Android/Sdk/build-tools/*/"+toolName),
		}
	case "darwin":
		paths = []string{
			"/usr/local/bin/" + toolName,
			"/opt/homebrew/bin/" + toolName,
			filepath.Join(os.Getenv("HOME"), "Library/Android/sdk/build-tools/*/"+toolName),
			"/Applications/Android Studio.app/Contents/plugins/android/lib/build-tools/*/"+toolName,
		}
	case "windows":
		paths = []string{
			"C:\\Android\\Sdk\\build-tools\\*\\" + toolName + ".exe",
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Android\\Sdk\\build-tools\\*\\"+toolName+".exe"),
			filepath.Join(os.Getenv("PROGRAMFILES"), "Android\\Android Studio\\plugins\\android\\lib\\build-tools\\*\\"+toolName+".exe"),
		}
	}
	
	return paths
}

// createAAPTNotFoundError creates a detailed error message with installation hints
func (p *AAPTParser) createAAPTNotFoundError() error {
	var installHints []string
	
	switch runtime.GOOS {
	case "linux":
		installHints = []string{
			"Ubuntu/Debian: sudo apt-get install aapt",
			"Or install Android SDK Build Tools:",
			"  1. Download Android SDK Command Line Tools from https://developer.android.com/studio#command-tools",
			"  2. Extract and add build-tools directory to PATH",
			"  3. Or set ANDROID_HOME environment variable",
		}
	case "darwin":
		installHints = []string{
			"macOS: brew install --cask android-commandlinetools",
			"Or install Android SDK Build Tools:",
			"  1. Download Android SDK Command Line Tools from https://developer.android.com/studio#command-tools",
			"  2. Extract to ~/Library/Android/sdk/",
			"  3. Add build-tools directory to PATH",
		}
	case "windows":
		installHints = []string{
			"Windows: Install Android SDK Build Tools",
			"  1. Download Android SDK Command Line Tools from https://developer.android.com/studio#command-tools",
			"  2. Extract to C:\\Android\\Sdk\\",
			"  3. Add build-tools directory to PATH",
		}
	default:
		installHints = []string{
			"Install Android SDK Build Tools from https://developer.android.com/studio#command-tools",
		}
	}
	
	errorMsg := "aapt2 or aapt not found. APK parsing will use limited androidbinary library.\n\n"
	errorMsg += "To enable full APK parsing, install aapt2:\n"
	for _, hint := range installHints {
		errorMsg += "  " + hint + "\n"
	}
	
	return fmt.Errorf(errorMsg)
}

// ParseAPKWithAAPT parses APK using aapt command
func (p *AAPTParser) ParseAPKWithAAPT(apkPath string) (*APKBasicInfo, error) {
	// Verify APK file exists
	if _, err := os.Stat(apkPath); err != nil {
		return nil, fmt.Errorf("APK file not found: %w", err)
	}
	
	// Determine command based on tool
	var cmd *exec.Cmd
	toolName := filepath.Base(p.aaptPath)
	
	if strings.Contains(toolName, "aapt2") {
		cmd = exec.Command(p.aaptPath, "dump", "badging", apkPath)
	} else {
		cmd = exec.Command(p.aaptPath, "dump", "badging", apkPath)
	}
	
	// Execute command with better error handling
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr := string(exitError.Stderr)
			return nil, fmt.Errorf("aapt command failed (exit code %d): %s", exitError.ExitCode(), stderr)
		}
		return nil, fmt.Errorf("aapt command execution failed: %w", err)
	}
	
	if len(output) == 0 {
		return nil, fmt.Errorf("aapt produced no output for APK: %s", apkPath)
	}
	
	return p.parseBadgingOutput(string(output))
}

// APKBasicInfo contains basic APK information from aapt
type APKBasicInfo struct {
	PackageID    string
	VersionName  string
	VersionCode  int64
	MinSDK       int
	TargetSDK    int
	AppName      string
	Permissions  []string
	Features     []string
	ABIs         []string
	ScreenDensities []string
}

// parseBadgingOutput parses aapt dump badging output
func (p *AAPTParser) parseBadgingOutput(output string) (*APKBasicInfo, error) {
	info := &APKBasicInfo{
		Permissions: []string{},
		Features:    []string{},
		ABIs:        []string{},
		ScreenDensities: []string{},
	}
	
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Parse package info
		if strings.HasPrefix(line, "package:") {
			if matches := regexp.MustCompile(`name='([^']+)'`).FindStringSubmatch(line); len(matches) > 1 {
				info.PackageID = matches[1]
			}
			if matches := regexp.MustCompile(`versionCode='([^']+)'`).FindStringSubmatch(line); len(matches) > 1 {
				if code, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
					info.VersionCode = code
				}
			}
			if matches := regexp.MustCompile(`versionName='([^']+)'`).FindStringSubmatch(line); len(matches) > 1 {
				info.VersionName = matches[1]
			}
		}
		
		// Parse SDK versions
		if strings.HasPrefix(line, "sdkVersion:") {
			if matches := regexp.MustCompile(`'(\d+)'`).FindStringSubmatch(line); len(matches) > 1 {
				if sdk, err := strconv.Atoi(matches[1]); err == nil {
					info.MinSDK = sdk
				}
			}
		}
		
		if strings.HasPrefix(line, "targetSdkVersion:") {
			if matches := regexp.MustCompile(`'(\d+)'`).FindStringSubmatch(line); len(matches) > 1 {
				if sdk, err := strconv.Atoi(matches[1]); err == nil {
					info.TargetSDK = sdk
				}
			}
		}
		
		// Parse application label
		if strings.HasPrefix(line, "application-label:") {
			if matches := regexp.MustCompile(`'([^']+)'`).FindStringSubmatch(line); len(matches) > 1 {
				info.AppName = matches[1]
			}
		}
		
		// Parse permissions
		if strings.HasPrefix(line, "uses-permission:") {
			if matches := regexp.MustCompile(`name='([^']+)'`).FindStringSubmatch(line); len(matches) > 1 {
				info.Permissions = append(info.Permissions, matches[1])
			}
		}
		
		// Parse features
		if strings.HasPrefix(line, "uses-feature:") {
			if matches := regexp.MustCompile(`name='([^']+)'`).FindStringSubmatch(line); len(matches) > 1 {
				info.Features = append(info.Features, matches[1])
			}
		}
		
		// Parse native code
		if strings.HasPrefix(line, "native-code:") {
			parts := strings.Split(line, "'")
			if len(parts) >= 2 {
				abis := strings.Fields(parts[1])
				info.ABIs = append(info.ABIs, abis...)
			}
		}
		
		// Parse densities
		if strings.HasPrefix(line, "densities:") {
			parts := strings.Split(line, "'")
			if len(parts) >= 2 {
				densities := strings.Fields(parts[1])
				info.ScreenDensities = append(info.ScreenDensities, densities...)
			}
		}
	}
	
	if info.PackageID == "" {
		return nil, fmt.Errorf("failed to parse package information")
	}
	
	return info, nil
}

// GetManifestXML extracts AndroidManifest.xml using aapt
func (p *AAPTParser) GetManifestXML(apkPath string) (string, error) {
	var cmd *exec.Cmd
	if p.aaptPath == "aapt2" {
		cmd = exec.Command(p.aaptPath, "dump", "xmltree", apkPath, "--file", "AndroidManifest.xml")
	} else {
		cmd = exec.Command(p.aaptPath, "dump", "xmltree", apkPath, "AndroidManifest.xml")
	}
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to dump manifest: %w", err)
	}
	
	return string(output), nil
}

// ManifestData represents parsed manifest data
type ManifestData struct {
	XMLName     xml.Name `xml:"manifest"`
	Package     string   `xml:"package,attr"`
	VersionCode string   `xml:"versionCode,attr"`
	VersionName string   `xml:"versionName,attr"`
}

// TryParseWithAAPT attempts to parse APK using aapt if available
func TryParseWithAAPT(apkPath string) (*APKBasicInfo, error) {
	parser := NewAAPTParser()
	
	// Check if aapt is available
	if err := parser.CheckAAPT(); err != nil {
		return nil, err
	}
	
	return parser.ParseAPKWithAAPT(apkPath)
}