# ApkHub CLI Changelog

## Fixes - 2025-07-22

### Bug Fixes

1. **Fixed panic when signature SHA256 is empty**
   - Added length check before accessing signature SHA256 substring
   - Now displays "(extraction failed)" when signature cannot be extracted

2. **Added aapt/aapt2 fallback for APK parsing**
   - When androidbinary library fails to parse an APK, automatically tries aapt
   - Supports both aapt and aapt2 commands
   - Better compatibility with newer APK formats

### How the fallback works

1. First attempts to parse with built-in androidbinary library
2. If that fails, checks for aapt2 or aapt in system PATH
3. Uses aapt to extract APK information
4. Provides warning message when fallback is used

### Installation of aapt

The tool will work without aapt, but having it installed improves compatibility:

```bash
# Ubuntu/Debian
sudo apt-get install aapt

# macOS with Homebrew
brew install --cask android-commandlinetools

# Check if aapt is available
which aapt2 || which aapt
```

### Known Limitations

- Signature extraction is not implemented when using aapt fallback
- Some advanced APK features may not be fully parsed with aapt
- XAPK and APKM formats still require additional implementation