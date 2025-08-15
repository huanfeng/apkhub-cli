package apk

import (
	"path/filepath"
	"strings"
)

// XAPKParserWrapper wraps the XAPK parser for the parser chain
type XAPKParserWrapper struct {
	parser  *XAPKParser
	workDir string
}

// NewXAPKParserWrapper creates a new XAPK parser wrapper
func NewXAPKParserWrapper(workDir string) *XAPKParserWrapper {
	return &XAPKParserWrapper{
		parser:  NewXAPKParser(workDir),
		workDir: workDir,
	}
}

// ParseAPK parses XAPK/APKM files
func (p *XAPKParserWrapper) ParseAPK(apkPath string) (*APKInfo, error) {
	xapkInfo, err := p.parser.ParseXAPK(apkPath)
	if err != nil {
		return nil, err
	}

	// Add XAPK specific features
	if xapkInfo.APKInfo != nil {
		// Mark as XAPK or APKM based on file extension
		ext := strings.ToLower(filepath.Ext(apkPath))
		if ext == ".xapk" {
			xapkInfo.APKInfo.Features = append(xapkInfo.APKInfo.Features, "xapk")
		} else if ext == ".apkm" {
			xapkInfo.APKInfo.Features = append(xapkInfo.APKInfo.Features, "apkm")
		}

		// Add split APK marker if multiple APKs
		if len(xapkInfo.APKFiles) > 1 {
			xapkInfo.APKInfo.Features = append(xapkInfo.APKInfo.Features, "split_apk")
		}

		// Add OBB marker if OBB files present
		if len(xapkInfo.OBBFiles) > 0 {
			xapkInfo.APKInfo.Features = append(xapkInfo.APKInfo.Features, "has_obb")
		}

		return xapkInfo.APKInfo, nil
	}

	return nil, err
}

// GetParserInfo returns information about this parser
func (p *XAPKParserWrapper) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:         "XAPK",
		Version:      "1.0",
		Capabilities: []string{"xapk", "apkm", "split_apk", "obb", "manifest"},
		Available:    true, // Always available
		Priority:     3,    // Lower priority than APK parsers
	}
}

// CanParse checks if this parser can handle the given file
func (p *XAPKParserWrapper) CanParse(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".xapk" || ext == ".apkm"
}
