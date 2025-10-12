#!/bin/sh
# Ork installation script
# Usage: curl -sSL https://raw.githubusercontent.com/ork-cli/ork/main/install.sh | sh
# Or with custom location: curl -sSL https://ork.sh | sh

set -e

# Configuration
REPO="ork-cli/ork"
BINARY_NAME="ork"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    printf "${BLUE}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
    exit 1
}

warning() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="Linux";;
        Darwin*)    OS="Darwin";;
        MINGW*|MSYS*|CYGWIN*) OS="Windows";;
        *)          error "Unsupported operating system: $(uname -s)";;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64)     ARCH="x86_64";;
        amd64)      ARCH="x86_64";;
        arm64)      ARCH="arm64";;
        aarch64)    ARCH="arm64";;
        armv7l)     ARCH="armv7";;
        *)          error "Unsupported architecture: $ARCH";;
    esac
}

# Get latest release version
get_latest_version() {
    info "Fetching latest version..."
    VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version. Please check your internet connection or try again later."
    fi

    info "Latest version: $VERSION"
}

# Download and install binary
install_binary() {
    # Construct download URL
    ARCHIVE_EXT="tar.gz"
    if [ "$OS" = "Windows" ]; then
        ARCHIVE_EXT="zip"
    fi

    ARCHIVE_NAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.${ARCHIVE_EXT}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

    info "Downloading from: $DOWNLOAD_URL"

    # Create temporary directory
    TMP_DIR="$(mktemp -d)"
    trap "rm -rf '$TMP_DIR'" EXIT

    # Download archive
    if ! curl -sfL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"; then
        error "Failed to download $ARCHIVE_NAME"
    fi

    # Extract binary
    info "Extracting archive..."
    cd "$TMP_DIR"
    if [ "$ARCHIVE_EXT" = "zip" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi

    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        warning "Installation directory $INSTALL_DIR requires sudo privileges"
        SUDO="sudo"
    else
        SUDO=""
    fi

    # Install binary
    info "Installing $BINARY_NAME to $INSTALL_DIR..."
    if [ "$OS" = "Windows" ]; then
        $SUDO mv "$BINARY_NAME.exe" "$INSTALL_DIR/"
    else
        $SUDO mv "$BINARY_NAME" "$INSTALL_DIR/"
        $SUDO chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    success "$BINARY_NAME installed successfully!"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        INSTALLED_VERSION=$("$BINARY_NAME" --version 2>&1 || echo "unknown")
        success "Installation verified: $INSTALLED_VERSION"
        info ""
        info "Get started with:"
        info "  $BINARY_NAME --help"
    else
        warning "$BINARY_NAME was installed but is not in your PATH"
        info "Add $INSTALL_DIR to your PATH or run: $INSTALL_DIR/$BINARY_NAME"
    fi
}

# Main installation flow
main() {
    echo ""
    info "Installing Ork - Microservices Orchestration Tool"
    echo ""

    # Detect system
    detect_os
    detect_arch
    info "Detected: $OS $ARCH"

    # Get version
    get_latest_version

    # Install
    install_binary

    # Verify
    verify_installation

    echo ""
    success "Installation complete! 🚀"
    echo ""
}

# Run main function
main
