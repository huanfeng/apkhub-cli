# ApkHub Client Design

## Overview

ApkHub client functionality provides a Scoop-like experience for managing Android applications.

## Command Structure

### Bucket Management
```bash
# List all buckets
apkhub bucket list

# Add a new bucket
apkhub bucket add main https://apk.example.com
apkhub bucket add fdroid https://f-droid.org/repo

# Remove a bucket
apkhub bucket remove fdroid

# Update bucket indexes
apkhub bucket update [name]
```

### Application Management
```bash
# Search for applications
apkhub search chrome
apkhub search "google chrome"
apkhub search --bucket main chrome

# Show application details
apkhub info com.android.chrome

# Download application
apkhub download com.android.chrome
apkhub download com.android.chrome --version 100.0.4896.127

# Install application (requires adb)
apkhub install com.android.chrome
apkhub install com.android.chrome --device emulator-5554

# Update installed applications
apkhub update
apkhub update com.android.chrome

# Uninstall application
apkhub uninstall com.android.chrome

# List installed applications
apkhub list --installed
```

## Configuration Structure

### User Config (~/.apkhub/config.yaml)
```yaml
default_bucket: main
buckets:
  main:
    name: "Main Repository"
    url: "https://apk.example.com"
    enabled: true
  fdroid:
    name: "F-Droid"
    url: "https://f-droid.org/repo"
    enabled: true
    
client:
  download_dir: "~/.apkhub/downloads"
  cache_dir: "~/.apkhub/cache"
  cache_ttl: 3600  # 1 hour
  
adb:
  path: "adb"  # or full path to adb
  default_device: ""  # empty for auto-detect
```

## Data Storage

```
~/.apkhub/
├── config.yaml           # User configuration
├── cache/               # Cached bucket manifests
│   ├── main.json
│   └── fdroid.json
├── downloads/           # Downloaded APKs
│   └── com.android.chrome_100.apk
└── installed.json       # Track installed apps
```

## Implementation Plan

1. **Client Configuration Manager**
   - Load/save user config
   - Manage bucket list
   - Cache management

2. **Bucket Manager**
   - Add/remove/list buckets
   - Fetch and cache manifests
   - Merge multiple bucket indexes

3. **Search Engine**
   - Search across buckets
   - Fuzzy matching
   - Filter by bucket

4. **Download Manager**
   - Download with progress
   - Resume support
   - Verify checksums

5. **ADB Integration**
   - Device detection
   - Install/uninstall
   - Version checking

## Command Examples

### Typical Workflow
```bash
# Add a bucket
apkhub bucket add myrepo https://myapks.com

# Search for an app
apkhub search telegram

# Show app details
apkhub info org.telegram.messenger

# Install the app
apkhub install org.telegram.messenger

# Update all apps
apkhub update

# Remove an app
apkhub uninstall org.telegram.messenger
```

### Advanced Usage
```bash
# Install specific version
apkhub install org.telegram.messenger@9.0.2

# Install to specific device
apkhub install org.telegram.messenger --device 192.168.1.100:5555

# Download without installing
apkhub download org.telegram.messenger --no-install

# Search with filters
apkhub search --min-sdk 21 --free messaging

# Export installed list
apkhub list --installed --export installed.txt
```

## Error Handling

- Network errors: Retry with exponential backoff
- ADB errors: Clear error messages with troubleshooting hints
- Checksum mismatches: Re-download option
- Version conflicts: Show clear upgrade/downgrade paths