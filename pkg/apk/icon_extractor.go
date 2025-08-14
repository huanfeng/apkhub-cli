package apk

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
	"github.com/shogo82148/androidbinary/apk"
	"golang.org/x/image/webp"
)

const (
	// Standard icon size for ApkHub
	StandardIconSize = 144
)

// IconExtractor handles icon extraction from APK files
type IconExtractor struct {
	targetSize uint
}

// NewIconExtractor creates a new icon extractor
func NewIconExtractor() *IconExtractor {
	return &IconExtractor{
		targetSize: StandardIconSize,
	}
}

// ExtractIcon extracts the app icon from an APK file
func (e *IconExtractor) ExtractIcon(apkPath string) ([]byte, string, error) {
	// Try androidbinary library first
	pkg, err := apk.OpenFile(apkPath)
	if err == nil {
		defer pkg.Close()
		return e.extractIconFromPackage(pkg)
	}

	// Fallback to direct zip extraction
	return e.extractIconFromZip(apkPath)
}

// extractIconFromPackage extracts icon using androidbinary library
func (e *IconExtractor) extractIconFromPackage(pkg *apk.Apk) ([]byte, string, error) {
	// androidbinary library has limited API, fallback to zip extraction
	return nil, "", fmt.Errorf("androidbinary icon extraction not implemented")
}

// extractIconFromZip extracts icon directly from APK zip
func (e *IconExtractor) extractIconFromZip(apkPath string) ([]byte, string, error) {
	reader, err := zip.OpenReader(apkPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open APK: %w", err)
	}
	defer reader.Close()

	// Priority order for icon selection
	iconPriorities := []string{
		"res/mipmap-xxxhdpi/ic_launcher.png",
		"res/mipmap-xxhdpi/ic_launcher.png",
		"res/mipmap-xhdpi/ic_launcher.png",
		"res/mipmap-hdpi/ic_launcher.png",
		"res/drawable-xxxhdpi/ic_launcher.png",
		"res/drawable-xxhdpi/ic_launcher.png",
		"res/drawable-xhdpi/ic_launcher.png",
		"res/drawable-hdpi/ic_launcher.png",
		"res/mipmap-xxxhdpi/ic_launcher.webp",
		"res/mipmap-xxhdpi/ic_launcher.webp",
		"res/mipmap-xhdpi/ic_launcher.webp",
		"res/mipmap-hdpi/ic_launcher.webp",
	}

	// Try to find icon by priority
	for _, iconPath := range iconPriorities {
		for _, file := range reader.File {
			if file.Name == iconPath {
				rc, err := file.Open()
				if err != nil {
					continue
				}
				defer rc.Close()

				iconData, err := io.ReadAll(rc)
				if err != nil {
					continue
				}

				return e.processIcon(iconData, filepath.Ext(iconPath))
			}
		}
	}

	// If no standard icon found, search for any launcher icon
	for _, file := range reader.File {
		if strings.Contains(file.Name, "ic_launcher") &&
			(strings.HasSuffix(file.Name, ".png") || strings.HasSuffix(file.Name, ".webp")) &&
			!strings.Contains(file.Name, "_foreground") &&
			!strings.Contains(file.Name, "_background") {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			iconData, err := io.ReadAll(rc)
			if err != nil {
				continue
			}

			return e.processIcon(iconData, filepath.Ext(file.Name))
		}
	}

	return nil, "", fmt.Errorf("no launcher icon found in APK")
}

// processIcon processes and resizes the icon
func (e *IconExtractor) processIcon(iconData []byte, ext string) ([]byte, string, error) {
	// Decode image
	var img image.Image
	var err error

	if ext == ".webp" {
		img, err = webp.Decode(bytes.NewReader(iconData))
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode webp: %w", err)
		}
	} else {
		img, _, err = image.Decode(bytes.NewReader(iconData))
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode image: %w", err)
		}
	}

	// Resize to standard size
	resized := resize.Resize(e.targetSize, e.targetSize, img, resize.Lanczos3)

	// Encode as PNG (standardize to PNG format)
	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return nil, "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), ".png", nil
}
