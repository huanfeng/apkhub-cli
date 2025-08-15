# ApkHub CLI

 [English](README.md) | [简体中文](README_zh.md)

A distributed APK repository management tool, similar to Scoop for Windows, that enables you to create, maintain, and consume APK repositories with ease.

## 🎯 What is ApkHub?

ApkHub CLI is a **distributed APK repository system** that works like Scoop package manager:

- **🏗️ Repository Mode**: Create and maintain APK repositories (like creating a Scoop bucket)
- **📱 Client Mode**: Search, download, and install APKs from multiple repositories (like using Scoop)
- **🌐 Distributed**: No central server required - repositories can be hosted anywhere
- **🔄 Multi-format**: Supports APK, XAPK (APKPure), and APKM (APKMirror) formats

## 🚀 Key Features

### 🏗️ Repository Management (`apkhub repo`)
Create and maintain your own APK repositories:

- **Initialize**: Set up new repositories with customizable configurations
- **Scan & Parse**: Automatically discover and parse APK/XAPK/APKM files
- **Metadata Extraction**: Extract comprehensive app information (permissions, signatures, icons)
- **Index Generation**: Create standardized `apkhub_manifest.json` files
- **Integrity Verification**: SHA256 checksums and repository validation
- **Batch Operations**: Incremental updates and bulk processing
- **Export/Import**: Support multiple formats (JSON, CSV, Markdown, F-Droid)

### 📱 Client Operations (`apkhub bucket`, `apkhub search`, `apkhub install`)
Consume APK repositories like a package manager:

- **Multi-Repository**: Manage multiple APK sources (buckets)
- **Smart Search**: Find apps across all configured repositories
- **Direct Installation**: Install APKs directly to Android devices via ADB
- **Download Management**: Automatic verification and resume support
- **Offline Mode**: Work with cached data when network unavailable
- **Health Monitoring**: Track repository status and connectivity

### 🛠️ System Tools
- **Doctor Command**: Comprehensive diagnostics and auto-fix capabilities
- **Device Management**: Monitor and manage connected Android devices
- **Dependency Handling**: Automatic tool detection and installation

## 📦 Installation

### Pre-built Binaries
Download the latest release from [GitHub Releases](https://github.com/huanfeng/apkhub-cli/releases):

```bash
# Linux/macOS
curl -L https://github.com/huanfeng/apkhub-cli/releases/latest/download/apkhub-linux-x86_64.tar.gz -o apkhub.tar.gz
tar xzf apkhub.tar.gz
sudo mv apkhub /usr/local/bin/
```

### Package Managers

#### Homebrew (macOS/Linux)
```bash
brew tap huanfeng/tap
brew install apkhub
```

#### Scoop (Windows)
```bash
scoop bucket add apkhub https://github.com/huanfeng/apkhub-scoop-bucket
scoop install apkhub
```

### Build from Source
```bash
git clone https://github.com/huanfeng/apkhub-cli.git
cd apkhub-cli
go build -o apkhub
```

## 🛠️ Quick Start

### 1. System Health Check
```bash
# Check system dependencies and health
apkhub doctor

# Auto-fix common issues
apkhub doctor --fix
```

### 2. 🏗️ Repository Management (Create Your Own APK Repository)

```bash
# Initialize a new repository
apkhub repo init

# Scan directory for APK files
apkhub repo scan /path/to/apks

# Add a single APK to repository
apkhub repo add app.apk

# View repository statistics
apkhub repo stats

# Verify repository integrity
apkhub repo verify

# Export repository data
apkhub repo export --format csv
```

### 3. 📱 Client Operations (Use APK Repositories)

```bash
# Add a repository source (bucket)
apkhub bucket add myrepo https://example.com/apkhub_manifest.json

# List all configured repositories
apkhub bucket list

# Search for applications across all repositories
apkhub search telegram

# Get detailed app information
apkhub info org.telegram.messenger

# Download an APK
apkhub download org.telegram.messenger

# Install directly to Android device
apkhub install org.telegram.messenger

# Install local APK file
apkhub install /path/to/app.apk
```

### 4. 📱 Device Management

```bash
# List connected Android devices
apkhub devices

# Watch device status in real-time
apkhub devices --watch

# Install to specific device
apkhub install --device emulator-5554 app.apk
```

## 📋 Command Reference

### 🏗️ Repository Management Commands (`apkhub repo`)
Create and maintain APK repositories:

- `apkhub repo init` - Initialize a new repository with configuration
- `apkhub repo scan <directory>` - Scan directory for APK/XAPK/APKM files
- `apkhub repo add <apk-file>` - Add single APK to repository
- `apkhub repo clean` - Clean old versions and orphaned files
- `apkhub repo stats` - Show detailed repository statistics
- `apkhub repo verify` - Verify repository integrity and fix issues
- `apkhub repo export` - Export repository data (JSON/CSV/Markdown)
- `apkhub repo import` - Import from other formats (F-Droid, etc.)

### 📱 Client Commands (Consume Repositories)
Use APK repositories like a package manager:

#### Repository Sources Management
- `apkhub bucket list` - List all configured repository sources
- `apkhub bucket add <name> <url>` - Add a new repository source
- `apkhub bucket remove <name>` - Remove a repository source
- `apkhub bucket update` - Update all repository sources
- `apkhub bucket health` - Check repository health status

#### App Discovery & Installation
- `apkhub search <query>` - Search applications across all repositories
- `apkhub info <package-id>` - Show detailed application information
- `apkhub list` - List all available packages
- `apkhub download <package-id>` - Download APK files
- `apkhub install <package-id|apk-path>` - Install applications to device

#### Cache Management
- `apkhub cache` - Manage local repository cache

### 🛠️ System & Device Commands
- `apkhub doctor` - System diagnostics and auto-fix
- `apkhub devices` - List and manage Android devices
- `apkhub deps` - Check and install dependencies
- `apkhub version` - Show version information

## 🔧 Configuration

### Repository Configuration (`apkhub.yaml`)
```yaml
repository:
  name: "My APK Repository"
  description: "Personal APK collection"
  base_url: "https://example.com"
  
directories:
  apks: "./apks"
  icons: "./icons"
  info: "./info"

settings:
  icon_size: 512
  keep_versions: 3
  generate_thumbnails: true
```

### Client Configuration (`~/.apkhub/config.yaml`)
```yaml
default_bucket: "main"
buckets:
  main:
    name: "main"
    url: "https://apkhub.example.com/apkhub_manifest.json"
    enabled: true
  
client:
  download_dir: "~/Downloads/apkhub"
  cache_dir: "~/.apkhub/cache"
  cache_ttl: 3600

adb:
  path: "adb"
  default_device: ""
```

## 📊 Repository Format

The generated `apkhub_manifest.json` follows this structure:

```json
{
  "version": "1.0",
  "name": "My APK Repository",
  "description": "Personal APK collection",
  "updated_at": "2025-01-15T10:00:00Z",
  "total_apks": 150,
  "packages": {
    "com.example.app": {
      "package_id": "com.example.app",
      "name": {
        "en": "Example App",
        "zh": "示例应用"
      },
      "description": "An example application",
      "category": "productivity",
      "versions": {
        "1.0.0": {
          "version": "1.0.0",
          "version_code": 100,
          "min_sdk": 21,
          "target_sdk": 33,
          "size": 5242880,
          "sha256": "abc123...",
          "download_url": "https://example.com/apks/com.example.app-1.0.0.apk",
          "icon_url": "https://example.com/icons/com.example.app.png",
          "permissions": ["android.permission.INTERNET"],
          "features": ["android.hardware.camera"],
          "abis": ["arm64-v8a", "armeabi-v7a"],
          "signature": {
            "sha256": "def456...",
            "issuer": "CN=Example Corp",
            "subject": "CN=Example App"
          },
          "release_date": "2025-01-15T10:00:00Z"
        }
      }
    }
  }
}
```

## 🔍 System Requirements

### Basic Requirements
- Go 1.22+ (for building from source)
- 50MB+ free disk space

### APK Parsing Dependencies
The tool uses multiple parsing methods for maximum compatibility:

1. **Primary**: Built-in Go library (`github.com/shogo82148/androidbinary`)
2. **Fallback**: AAPT/AAPT2 command-line tools (recommended for full compatibility)

#### Installing AAPT2

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install aapt
# or for newer versions
sudo apt-get install google-android-build-tools-installer
```

**macOS:**
```bash
# Install Android SDK command-line tools
brew install --cask android-commandlinetools
# aapt2 will be available in: ~/Library/Android/sdk/build-tools/*/aapt2
```

**Windows:**
```bash
# Using Scoop
scoop bucket add extras
scoop install android-sdk

# Using Chocolatey
choco install android-sdk
```

### ADB for Device Installation
**Ubuntu/Debian:**
```bash
sudo apt-get install android-tools-adb
```

**macOS:**
```bash
brew install android-platform-tools
```

**Windows:**
```bash
# Using Scoop
scoop install adb

# Using Chocolatey
choco install adb
```

## 🚀 Advanced Usage

### 🏗️ Repository Management Workflows

#### Automated Repository Maintenance
```bash
# Full repository scan with progress
apkhub repo scan --recursive --progress /path/to/apks

# Incremental update (only new/changed files)
apkhub repo scan --incremental /path/to/apks

# Clean old versions (keep latest 3)
apkhub repo clean --keep 3

# Verify and auto-fix issues
apkhub repo verify --fix
```

#### Batch Operations
```bash
# Export repository data
apkhub repo export --format csv --output apps.csv
apkhub repo export --format markdown --output README.md

# Import from F-Droid
apkhub repo import --format fdroid https://f-droid.org/repo/index-v1.json
```

#### CI/CD Integration
```yaml
# GitHub Actions example
- name: Update APK Repository
  run: |
    apkhub repo scan ./apks
    apkhub repo verify --quiet
    git add apkhub_manifest.json
    git commit -m "Update repository index"
```

### 📱 Client Usage Workflows

#### Multi-Repository Setup
```bash
# Add multiple repository sources
apkhub bucket add official https://apkhub.example.com/apkhub_manifest.json
apkhub bucket add fdroid https://f-droid.org/repo/apkhub_manifest.json
apkhub bucket add personal https://my-repo.com/apkhub_manifest.json

# Search across all repositories
apkhub search "telegram"

# Install from any repository
apkhub install org.telegram.messenger
```

#### Bulk Installation
```bash
# Install multiple apps from list
cat app-list.txt | xargs -I {} apkhub install {}

# Install with specific options
apkhub install --device emulator-5554 --version 1.2.3 com.example.app
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [androidbinary](https://github.com/shogo82148/androidbinary) - APK parsing library
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management

## 📞 Support

- 📖 [Documentation](https://github.com/huanfeng/apkhub-cli/wiki)
- 🐛 [Issue Tracker](https://github.com/huanfeng/apkhub-cli/issues)
- 💬 [Discussions](https://github.com/huanfeng/apkhub-cli/discussions)