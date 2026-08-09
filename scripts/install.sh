#!/usr/bin/env bash
set -euo pipefail

# upp installer — downloads the correct binary for the current platform
# Usage: curl -fsSL https://raw.githubusercontent.com/JhnFrankz/upp/main/scripts/install.sh | bash

REPO="JhnFrankz/upp"
BINARY="upp"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors (if terminal supports it)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

info() {
    printf "${GREEN}%s${NC}\n" "$1"
}

warn() {
    printf "${YELLOW}%s${NC}\n" "$1"
}

error() {
    printf "${RED}%s${NC}\n" "$1" >&2
    exit 1
}

# Check for required tools
check_deps() {
    for cmd in curl uname; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "Required command '$cmd' not found. Please install it."
        fi
    done
}

# Detect platform and architecture
detect_platform() {
    local os arch

    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            os="windows"
            ;;
        *)
            error "Unsupported operating system: $os"
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l|armhf)
            arch="arm"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac

    echo "${os}/${arch}"
}

# Get download URL for the current platform
get_download_url() {
    local platform="$1"
    local os="${platform%/*}"
    local arch="${platform#*/}"
    local ext=""

    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    if [ "$VERSION" = "latest" ]; then
        # Use GitHub API to get latest release
        local api_url="https://api.github.com/repos/${REPO}/releases/latest"
        local download_url
        download_url=$(curl -fsSL "$api_url" | grep -o '"browser_download_url": "[^"]*upp-'${os}'-'${arch}${ext}"' | cut -d'"' -f4)
        
        if [ -z "$download_url" ]; then
            # Fallback to direct URL
            download_url="https://github.com/${REPO}/releases/latest/download/upp-${os}-${arch}${ext}"
        fi
        echo "$download_url"
    else
        echo "https://github.com/${REPO}/releases/download/${VERSION}/upp-${os}-${arch}${ext}"
    fi
}

# Download and install binary
install_binary() {
    local platform="$1"
    local os="${platform%/*}"
    local arch="${platform#*/}"
    local ext=""

    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    local download_url
    download_url=$(get_download_url "$platform")
    
    local tmp_dir
    tmp_dir=$(mktemp -d)
    local tmp_file="${tmp_dir}/upp-${os}-${arch}${ext}"
    
    info "Downloading upp for ${os}/${arch}..."
    info "URL: ${download_url}"
    
    if ! curl -fsSL -o "$tmp_file" "$download_url"; then
        error "Failed to download binary from ${download_url}"
    fi
    
    # Verify checksum if available
    local checksum_url="${download_url}.sha256"
    if curl -fsSL -o "${tmp_file}.sha256" "$checksum_url" 2>/dev/null; then
        info "Verifying checksum..."
        local expected_checksum
        expected_checksum=$(cut -d' ' -f1 "${tmp_file}.sha256")
        local actual_checksum
        actual_checksum=$(sha256sum "$tmp_file" | cut -d' ' -f1)
        
        if [ "$expected_checksum" != "$actual_checksum" ]; then
            error "Checksum mismatch! Expected: ${expected_checksum}, Got: ${actual_checksum}"
        fi
        info "Checksum verified."
    else
        warn "No checksum file found, skipping verification."
    fi
    
    # Make executable (Unix only)
    if [ "$os" != "windows" ]; then
        chmod +x "$tmp_file"
    fi
    
    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Install binary
    local install_path="${INSTALL_DIR}/${BINARY}${ext}"
    
    # Backup existing binary if present
    if [ -f "$install_path" ]; then
        local backup_path="${install_path}.backup.$(date +%s)"
        warn "Backing up existing binary to ${backup_path}"
        mv "$install_path" "$backup_path"
    fi
    
    mv "$tmp_file" "$install_path"
    info "Installed ${BINARY} to ${install_path}"
    
    # Clean up
    rm -rf "$tmp_dir"
}

# Verify installation
verify_installation() {
    local platform="$1"
    local os="${platform%/*}"
    local ext=""
    
    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi
    
    local install_path="${INSTALL_DIR}/${BINARY}${ext}"
    
    if [ -f "$install_path" ]; then
        info "Installation successful!"
        info "Run '${BINARY} --help' to get started."
    else
        error "Installation failed. Binary not found at ${install_path}"
    fi
}

# Main
main() {
    check_deps
    
    local platform
    platform=$(detect_platform)
    
    info "Detected platform: ${platform}"
    
    install_binary "$platform"
    verify_installation "$platform"
}

main "$@"