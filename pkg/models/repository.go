package models

import "time"

// RepositoryLayout defines the standard repository directory structure
type RepositoryLayout struct {
	RootDir      string
	APKsDir      string // apks/
	InfosDir     string // infos/
	ManifestFile string // apkhub_manifest.json
}

// APKInfo represents individual APK information stored in infos/ directory
type APKInfo struct {
	PackageID     string            `json:"package_id"`
	AppName       map[string]string `json:"app_name"`
	Version       string            `json:"version"`
	VersionCode   int64             `json:"version_code"`
	MinSDK        int               `json:"min_sdk"`
	TargetSDK     int               `json:"target_sdk"`
	Size          int64             `json:"size"`
	SHA256        string            `json:"sha256"`
	SignatureInfo *SignatureInfo    `json:"signature"`
	Permissions   []string          `json:"permissions,omitempty"`
	Features      []string          `json:"features,omitempty"`
	ABIs          []string          `json:"abis,omitempty"`
	AddedAt       time.Time         `json:"added_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	OriginalName  string            `json:"original_name"`
	FileName      string            `json:"file_name"`           // Normalized filename
	FilePath      string            `json:"file_path"`           // Relative path in apks/
	InfoPath      string            `json:"info_path"`           // Relative path in infos/
	IconPath      string            `json:"icon_path,omitempty"` // Relative path to icon in infos/
}

// ManifestIndex is the main index file (apkhub_manifest.json)
type ManifestIndex struct {
	Version     string                 `json:"version"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	UpdatedAt   time.Time              `json:"updated_at"`
	TotalAPKs   int                    `json:"total_apks"`
	TotalSize   int64                  `json:"total_size"`
	Packages    map[string]*AppPackage `json:"packages"`
	Signature   *ManifestSignature     `json:"signature,omitempty"`
}

// ManifestSignature captures the signing metadata for a manifest index
type ManifestSignature struct {
	PublicKeyFingerprint string    `json:"public_key_fingerprint,omitempty"`
	SignedAt             time.Time `json:"signed_at,omitempty"`
	Signer               string    `json:"signer,omitempty"`
}

// NewRepositoryLayout creates a new repository layout structure
func NewRepositoryLayout(rootDir string) *RepositoryLayout {
	return &RepositoryLayout{
		RootDir:      rootDir,
		APKsDir:      "apks",
		InfosDir:     "infos",
		ManifestFile: "apkhub_manifest.json",
	}
}
