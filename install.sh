#!/bin/bash
set -e

REPO="Lucu-lucuan-Lab/nolife-cli"
APP_NAME="nolife"
GITHUB_URL="https://github.com/$REPO"
API_URL="https://api.github.com/repos/$REPO/releases/latest"

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

write_step() { echo -e "\n${CYAN}[*] $1${NC}"; }
write_success() { echo -e "${GREEN}[+] $1${NC}"; }
write_warning() { echo -e "${YELLOW}[!] $1${NC}"; }
write_error() { echo -e "${RED}[X] $1${NC}"; }

assert_not_running() {
    if pgrep -x "$APP_NAME" > /dev/null; then
        write_error "$APP_NAME is currently running. Please close it before installing/updating."
        exit 1
    fi
}

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) write_error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    *) write_error "Unsupported OS: $OS"; exit 1 ;;
esac

write_step "Detecting system..."
echo "OS: $OS, Arch: $ARCH"

assert_not_running

# Determine install directory
if [ "$OS" == "darwin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

write_step "Fetching latest release from GitHub..."
RELEASE_DATA=$(curl -s "$API_URL")
VERSION=$(echo "$RELEASE_DATA" | grep -oE '"tag_name": "[^"]+"' | head -n 1 | cut -d'"' -f4)
echo "Version: $VERSION"

ASSET_NAME="${APP_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL=$(echo "$RELEASE_DATA" | grep -oE '"browser_download_url": "[^"]+'$ASSET_NAME'"' | head -n 1 | cut -d'"' -f4)

if [ -z "$DOWNLOAD_URL" ]; then
    write_error "Could not find download URL for $ASSET_NAME"
    exit 1
fi

TEMP_FILE="/tmp/${ASSET_NAME}"

write_step "Downloading $APP_NAME..."
curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL"

write_step "Installing to $INSTALL_DIR..."
chmod +x "$TEMP_FILE"

if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_FILE" "$INSTALL_DIR/$APP_NAME"
else
    write_warning "Need sudo to install to $INSTALL_DIR"
    sudo mv "$TEMP_FILE" "$INSTALL_DIR/$APP_NAME"
fi

if [ "$OS" == "darwin" ]; then
    write_step "Checking dependencies..."
    if [ ! -d "/Applications/Google Chrome.app" ] && ! command -v google-chrome &> /dev/null; then
        write_warning "Google Chrome not detected in /Applications."
        write_warning "Nolife needs Chrome/Chromium to work."
        write_warning "Please download it from: https://www.google.com/chrome/"
    else
        write_success "Google Chrome detected."
    fi
fi

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    write_warning "$INSTALL_DIR is not in your PATH."
    write_warning "Please add it to your .zshrc or .bashrc:"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\""
fi

write_success "Successfully installed $APP_NAME $VERSION!"
echo -e "\nUsage:"
echo "  $APP_NAME download [URL]"
echo "  $APP_NAME search [query]"
