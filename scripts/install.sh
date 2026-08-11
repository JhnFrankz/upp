#!/usr/bin/env bash
set -euo pipefail

# upp installer — downloads the matching release archive and verifies its
# SHA-256 checksum against checksums.txt before installing.
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
    local os="$1"
    for cmd in curl uname tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "Required command '$cmd' not found. Please install it."
        fi
    done
    if [ "$os" = "windows" ] && ! command -v unzip >/dev/null 2>&1; then
        error "Required command 'unzip' not found. Please install it."
    fi
    if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
        error "Required command 'sha256sum' or 'shasum' not found. Please install it."
    fi
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

# Resolve the release tag: "latest" queries the GitHub API, otherwise the pinned VERSION is used.
resolve_tag() {
    if [ "$VERSION" != "latest" ]; then
        echo "$VERSION"
        return
    fi

    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local tag
    tag=$(curl -fsSL "$api_url" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)

    if [ -z "$tag" ]; then
        error "Could not determine the latest release tag."
    fi
    echo "$tag"
}

# Name of the release asset for this platform (see the Makefile `release` target).
asset_name() {
    local os="$1" arch="$2"
    if [ "$os" = "windows" ]; then
        echo "${BINARY}-${os}-${arch}.zip"
    else
        echo "${BINARY}-${os}-${arch}.tar.gz"
    fi
}

# Verify the downloaded file against the checksums.txt shipped with the release.
verify_checksum() {
    local file="$1" checksums="$2" name="$3"
    local expected actual

    expected=$(grep -F "${name}" "$checksums" 2>/dev/null | awk '{print $1}' | head -n1)
    if [ -z "$expected" ]; then
        warn "No checksum entry for ${name}, skipping verification."
        return 0
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    else
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    fi

    if [ "$expected" != "$actual" ]; then
        error "Checksum mismatch! Expected: ${expected}, Got: ${actual}"
    fi
    info "Checksum verified."
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

    local asset
    asset=$(asset_name "$os" "$arch")
    local tag
    tag=$(resolve_tag)
    local base_url="https://github.com/${REPO}/releases/download/${tag}"
    local download_url="${base_url}/${asset}"
    local checksum_url="${base_url}/checksums.txt"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    local tmp_asset="${tmp_dir}/${asset}"
    local tmp_checksums="${tmp_dir}/checksums.txt"

    info "Downloading ${asset} (${tag})..."
    info "URL: ${download_url}"

    if ! curl -fsSL -o "$tmp_asset" "$download_url"; then
        error "Failed to download ${download_url}"
    fi

    if curl -fsSL -o "$tmp_checksums" "$checksum_url"; then
        info "Verifying checksum..."
        verify_checksum "$tmp_asset" "$tmp_checksums" "$asset"
    else
        warn "No checksums.txt found, skipping verification."
    fi

    # Extract the binary from the archive
    local stage="${tmp_dir}/stage"
    mkdir -p "$stage"
    if [ "$os" = "windows" ]; then
        unzip -q "$tmp_asset" -d "$stage"
    else
        tar xzf "$tmp_asset" -C "$stage"
    fi

    local tmp_file="${stage}/${BINARY}-${os}-${arch}/${BINARY}${ext}"
    if [ ! -f "$tmp_file" ]; then
        error "Binary not found in archive: ${tmp_file}"
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
    local platform
    platform=$(detect_platform)

    check_deps "${platform%/*}"

    info "Detected platform: ${platform}"

    install_binary "$platform"
    verify_installation "$platform"
}

main "$@"
