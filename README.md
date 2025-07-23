# ApkHub CLI

A command-line tool for managing distributed APK repositories.

## Features

### Repository Management
- Scan directories for APK/XAPK/APKM files
- Parse APK metadata (package name, version, permissions, etc.)
- Extract and save app icons
- Generate repository index files (apkhub_manifest.json)
- Calculate SHA256 checksums
- Support for batch processing and incremental updates

### Client Features (NEW)
- Manage multiple APK repository sources (buckets)
- Search apps across all repositories
- Download APKs with checksum verification
- Install apps directly to Android devices via adb
- Scoop-like command line experience

## Installation

```bash
go install github.com/apkhub/apkhub-cli@latest
```

Or build from source:

```bash
git clone https://github.com/apkhub/apkhub-cli.git
cd apkhub-cli
go build -o apkhub
```

## Usage

### Repository Management

```bash
# Initialize a repository
apkhub repo init

# Scan a directory for APK files
apkhub repo scan /path/to/apks

# Add a single APK
apkhub repo add app.apk

# View repository statistics
apkhub repo stats

# Clean old versions
apkhub repo clean --keep 3
```

### Client Features

```bash
# Add a repository source
apkhub bucket add main https://apk.example.com

# Search for apps
apkhub search chrome

# View app details
apkhub info com.android.chrome

# Install an app
apkhub install com.android.chrome
```

### Quick Start

```bash
# 1. Add a repository
apkhub bucket add myrepo https://myapks.com

# 2. Search for an app
apkhub search telegram

# 3. Install it
apkhub install org.telegram.messenger
```

## Package.json Format

The generated `package.json` follows this structure:

```json
{
  "version": "1.0",
  "name": "My APK Repository",
  "updated_at": "2025-07-22T10:00:00Z",
  "packages": {
    "com.example.app": {
      "package_id": "com.example.app",
      "name": "Example App",
      "versions": {
        "1.0.0": {
          "version": "1.0.0",
          "version_code": 100,
          "min_sdk": 21,
          "size": 5242880,
          "sha256": "...",
          "download_url": "https://..."
        }
      }
    }
  }
}
```

## Requirements

### Basic Requirements
- Go 1.21+ (for building from source)

### APK Parsing

The tool uses two methods for parsing APK files:

1. **Primary**: Built-in Go library (github.com/shogo82148/androidbinary)
2. **Fallback**: aapt/aapt2 command line tool (recommended for better compatibility)

#### Installing aapt2

**Ubuntu/Debian:**
```bash
sudo apt-get install aapt
# or for newer versions
sudo apt-get install google-android-build-tools-installer
```

**macOS:**
```bash
# Install Android SDK command-line tools
brew install --cask android-commandlinetools
# aapt2 will be in: ~/Library/Android/sdk/build-tools/*/aapt2
```

**Manual Installation:**
1. Download Android SDK Build Tools from https://developer.android.com/studio#command-tools
2. Extract and add the build-tools directory to your PATH

## Development

### Build

```bash
go build -o apkhub
```

### Test

```bash
go test ./...
```

## License

MIT