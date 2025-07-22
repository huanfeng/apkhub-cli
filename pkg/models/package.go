package models

import "time"

// PackageIndex represents the main index structure
type PackageIndex struct {
	Version     string                `json:"version"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Packages    map[string]*AppPackage `json:"packages"`
}

// AppPackage represents an application with all its versions
type AppPackage struct {
	PackageID   string                `json:"package_id"`
	Name        map[string]string     `json:"name"`        // Multi-language support
	Icon        string                `json:"icon,omitempty"`
	Category    string                `json:"category,omitempty"`
	Description map[string]string     `json:"description,omitempty"` // Multi-language support
	Versions    map[string]*AppVersion `json:"versions"`
	Latest      string                `json:"latest"`      // Latest version string
}

// AppVersion represents a specific version of an application
type AppVersion struct {
	Version      string    `json:"version"`
	VersionCode  int64     `json:"version_code"`
	MinSDK       int       `json:"min_sdk"`
	TargetSDK    int       `json:"target_sdk"`
	Size         int64     `json:"size"`
	SHA256       string    `json:"sha256"`
	SignatureInfo *SignatureInfo `json:"signature"`
	DownloadURL  string    `json:"download_url"`
	ReleaseDate  time.Time `json:"release_date"`
	Permissions  []string  `json:"permissions,omitempty"`
	Features     []string  `json:"features,omitempty"`
	ABIs         []string  `json:"abis,omitempty"`
	ScreenDPIs   []string  `json:"screen_dpis,omitempty"`
	Locales      []string  `json:"locales,omitempty"`
	SignatureVariant string `json:"signature_variant,omitempty"` // For different signatures
}

// SignatureInfo contains APK signature information
type SignatureInfo struct {
	SHA256     string `json:"sha256"`
	SHA1       string `json:"sha1"`
	MD5        string `json:"md5"`
	Issuer     string `json:"issuer,omitempty"`
	Subject    string `json:"subject,omitempty"`
}