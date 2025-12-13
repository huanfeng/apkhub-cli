package models

// Config represents the application configuration
type Config struct {
	Repository RepositoryConfig `mapstructure:"repository" json:"repository"`
	Scanning   ScanningConfig   `mapstructure:"scanning" json:"scanning"`
}

// RepositoryConfig contains repository-related configuration
type RepositoryConfig struct {
	Name                  string   `mapstructure:"name" json:"name"`
	Description           string   `mapstructure:"description" json:"description"`
	BaseURL               string   `mapstructure:"base_url" json:"base_url"`
	KeepVersions          int      `mapstructure:"keep_versions" json:"keep_versions"`                     // 0 = keep all
	SignatureHandling     string   `mapstructure:"signature_handling" json:"signature_handling"`           // "mark", "separate", "reject"
	SigningKeyFingerprint string   `mapstructure:"signing_key_fingerprint" json:"signing_key_fingerprint"` // Fingerprint used when signing manifest
	Signer                string   `mapstructure:"signer" json:"signer"`                                   // Human-readable signer name
	TrustedKeys           []string `mapstructure:"trusted_keys" json:"trusted_keys"`
	SignaturePolicy       string   `mapstructure:"signature_policy" json:"signature_policy"` // "strict" or "lenient"
}

// ScanningConfig contains scanning-related configuration
type ScanningConfig struct {
	Recursive      bool     `mapstructure:"recursive" json:"recursive"`
	FollowSymlinks bool     `mapstructure:"follow_symlinks" json:"follow_symlinks"`
	IncludePattern []string `mapstructure:"include_pattern" json:"include_pattern"`
	ExcludePattern []string `mapstructure:"exclude_pattern" json:"exclude_pattern"`
	ParseAPKInfo   bool     `mapstructure:"parse_apk_info" json:"parse_apk_info"`
}
