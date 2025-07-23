#!/bin/bash

# ApkHub Client Demo Script
# This script demonstrates the client functionality of ApkHub

echo "=== ApkHub Client Demo ==="
echo

# Initialize client config
echo "1. Initializing client configuration..."
./apkhub bucket list
echo

# Add a demo bucket
echo "2. Adding a demo bucket..."
echo "   (You would run: ./apkhub bucket add main https://apk.example.com)"
echo

# Update buckets
echo "3. Updating bucket indexes..."
echo "   (You would run: ./apkhub bucket update)"
echo

# Search for apps
echo "4. Searching for applications..."
echo "   (You would run: ./apkhub search chrome)"
echo

# Show app info
echo "5. Getting app information..."
echo "   (You would run: ./apkhub info com.android.chrome)"
echo

# Download an app
echo "6. Downloading an application..."
echo "   (You would run: ./apkhub download com.android.chrome)"
echo

# Install an app
echo "7. Installing an application..."
echo "   (You would run: ./apkhub install com.android.chrome)"
echo

echo "=== Client Commands Summary ==="
echo
echo "Bucket Management:"
echo "  apkhub bucket list                    # List all buckets"
echo "  apkhub bucket add <name> <url>        # Add a new bucket"
echo "  apkhub bucket remove <name>           # Remove a bucket"
echo "  apkhub bucket update [name]           # Update bucket indexes"
echo
echo "Application Management:"
echo "  apkhub search <query>                 # Search for apps"
echo "  apkhub info <package-id>              # Show app details"
echo "  apkhub download <package-id>          # Download an app"
echo "  apkhub install <package-id>           # Install an app"
echo
echo "Install Options:"
echo "  --device, -s <id>    Target device"
echo "  --version, -v <ver>  Specific version"
echo "  --replace, -r        Replace existing"
echo "  --downgrade, -d      Allow downgrade"
echo
echo "=== Configuration Location ==="
echo "Config file: ~/.apkhub/config.yaml"
echo "Downloads:   ~/.apkhub/downloads/"
echo "Cache:       ~/.apkhub/cache/"