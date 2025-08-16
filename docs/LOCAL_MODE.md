# ApkHub CLI Local Mode Guide

## Overview

ApkHub CLI now supports improved local mode functionality for better integration with bucket-based workflows. This guide explains the new features and how to use them.

## New Features

### 1. Update Command (Alias for bucket update)

The `update` command is now available as a convenient alias for `bucket update`, similar to how Scoop works.

```bash
# Update all enabled buckets
apkhub update

# Update a specific bucket
apkhub update main

# Update all buckets (including disabled ones)
apkhub update --all
```

This is equivalent to:
```bash
apkhub bucket update
apkhub bucket update main
```

### 2. Enhanced Local Mode for Repository Generation

When generating repository manifests, you can now use improved local mode settings that work better with bucket clients.

#### Base URL Options

In your `apkhub.yaml` configuration, you can set `base_url` to:

1. **`"local"`** - Generates `file://` URLs based on the repository path
   ```yaml
   repository:
     base_url: "local"
   ```
   This will generate URLs like: `file:///absolute/path/to/repo/apks/app.apk`

2. **`"http://localhost:8080"`** - For local HTTP servers
   ```yaml
   repository:
     base_url: "http://localhost:8080"
   ```
   This will generate URLs like: `http://localhost:8080/apks/app.apk`

3. **`"file:///specific/path"`** - For specific file paths
   ```yaml
   repository:
     base_url: "file:///var/www/apks"
   ```
   This will generate URLs like: `file:///var/www/apks/apks/app.apk`

4. **`""`** (empty) - For relative paths only
   ```yaml
   repository:
     base_url: ""
   ```
   This will generate relative URLs like: `apks/app.apk`

#### Local Template

A new `local` template is available for quick setup:

```bash
apkhub repo init --template local
```

This creates a configuration optimized for local file-based repositories that can be used with bucket clients.

## Usage Examples

### Setting up a Local Repository for Bucket Use

1. Initialize a local repository:
   ```bash
   mkdir my-apk-repo
   cd my-apk-repo
   apkhub repo init --template local
   ```

2. Add APK files:
   ```bash
   apkhub repo scan /path/to/apk/files
   ```

3. The generated manifest will have `file://` URLs that bucket clients can use:
   ```json
   {
     "packages": {
       "com.example.app": {
         "versions": {
           "1.0": {
             "download_url": "file:///absolute/path/to/my-apk-repo/apks/com.example.app_1.0.apk"
           }
         }
       }
     }
   }
   ```

### Adding Local Repository as Bucket

1. Add the local repository as a bucket:
   ```bash
   apkhub bucket add local-repo /path/to/my-apk-repo "Local Repository"
   ```

2. Update the bucket:
   ```bash
   apkhub update local-repo
   # or
   apkhub bucket update local-repo
   ```

3. Search and install from the local bucket:
   ```bash
   apkhub search com.example.app
   apkhub install com.example.app
   ```

## Benefits

1. **Better Integration**: Local repositories now generate URLs that work seamlessly with bucket clients
2. **Flexible Deployment**: Support for various local deployment scenarios (file://, localhost, etc.)
3. **Scoop-like Experience**: The `update` command provides familiar workflow for users coming from Scoop
4. **Simplified Setup**: The `local` template provides optimal defaults for local development

## Migration from Old Local Mode

If you have existing repositories using `http://localhost:8080/` URLs, you can:

1. Update your configuration to use `base_url: "local"` for file:// URLs
2. Or keep using localhost URLs if you're running a local HTTP server
3. Regenerate your manifest with `apkhub repo scan` to apply the new URL scheme

The old configuration will continue to work, but the new options provide more flexibility.