#!/bin/bash

# Script to download go2rtc binary
set -e

INSTALL_DIR="${1:-./data/go2rtc/bin}"
VERSION="latest"

# Create directory
mkdir -p "$INSTALL_DIR"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm"
        ;;
esac

# Construct filename
FILENAME="go2rtc_${OS}_${ARCH}"
if [ "$OS" = "windows" ]; then
    FILENAME="${FILENAME}.exe"
fi

# Download URL
DOWNLOAD_URL="https://github.com/AlexxIT/go2rtc/releases/latest/download/${FILENAME}"

echo "Downloading go2rtc..."
echo "OS: $OS"
echo "Architecture: $ARCH"
echo "URL: $DOWNLOAD_URL"

# Download binary
curl -L -o "$INSTALL_DIR/go2rtc" "$DOWNLOAD_URL"

# Make executable
chmod +x "$INSTALL_DIR/go2rtc"

echo "go2rtc downloaded successfully to $INSTALL_DIR/go2rtc" 