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
	Name        string                `json:"name"`
	Icon        string                `json:"icon,omitempty"`
	Category    string                `json:"category,omitempty"`
	Description string                `json:"description,omitempty"`
	Versions    map[string]*AppVersion `json:"versions"`
}

// AppVersion represents a specific version of an application
type AppVersion struct {
	Version      string    `json:"version"`
	VersionCode  int       `json:"version_code"`
	MinSDK       int       `json:"min_sdk"`
	TargetSDK    int       `json:"target_sdk"`
	Size         int64     `json:"size"`
	SHA256       string    `json:"sha256"`
	Signature    string    `json:"signature"`
	DownloadURL  string    `json:"download_url"`
	ReleaseDate  time.Time `json:"release_date"`
	Permissions  []string  `json:"permissions,omitempty"`
	Features     []string  `json:"features,omitempty"`
	ABIs         []string  `json:"abis,omitempty"`
	ScreenDPIs   []string  `json:"screen_dpis,omitempty"`
}