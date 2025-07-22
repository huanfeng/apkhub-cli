package apk

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"regexp"
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

// CheckAAPT checks if aapt2 is available
func (p *AAPTParser) CheckAAPT() error {
	cmd := exec.Command(p.aaptPath, "version")
	if err := cmd.Run(); err != nil {
		// Try aapt (older version)
		cmd = exec.Command("aapt", "version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("aapt2 or aapt not found in PATH")
		}
		p.aaptPath = "aapt"
	}
	return nil
}

// ParseAPKWithAAPT parses APK using aapt command
func (p *AAPTParser) ParseAPKWithAAPT(apkPath string) (*APKBasicInfo, error) {
	// Run aapt dump badging
	var cmd *exec.Cmd
	if p.aaptPath == "aapt2" {
		cmd = exec.Command(p.aaptPath, "dump", "badging", apkPath)
	} else {
		cmd = exec.Command(p.aaptPath, "dump", "badging", apkPath)
	}
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aapt command failed: %w", err)
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