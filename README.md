# ApkHub CLI

A command-line tool for managing distributed APK repositories.

## Features

- Scan directories for APK/XAPK/APKM files
- Parse APK metadata (package name, version, permissions, etc.)
- Generate repository index files (package.json)
- Calculate SHA256 checksums
- Support for batch processing

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

### Scan a directory for APK files

```bash
# Scan current directory
apkhub scan .

# Scan with custom output
apkhub scan /path/to/apks -o index.json

# Scan without recursive search
apkhub scan /path/to/apks -r=false
```

### Parse a single APK

```bash
apkhub parse app.apk
```

### Generate repository index

```bash
apkhub index /path/to/repo
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