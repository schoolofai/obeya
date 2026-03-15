#!/bin/sh
# Obeya (ob) installer for Linux and macOS
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh -s -- --version 0.2.0
#   curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh -s -- --global

set -e

REPO="schoolofai/obeya"
BINARY="ob"
VERSION=""
INSTALL_DIR="$HOME/.local/bin"

usage() {
    echo "Usage: install.sh [--version VERSION] [--global] [--help]"
    echo ""
    echo "Options:"
    echo "  --version VERSION   Install a specific version (e.g. 0.2.0). Default: latest"
    echo "  --global            Install to /usr/local/bin (requires sudo)"
    echo "  --help              Show this help message"
    exit 0
}

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --global)
            INSTALL_DIR="/usr/local/bin"
            shift
            ;;
        --help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)
        echo "Error: Unsupported operating system: $OS"
        echo "This installer supports Linux and macOS. For Windows, use install.ps1."
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Get latest version from GitHub API if not specified
if [ -z "$VERSION" ]; then
    echo "Fetching latest version..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version. Specify one with --version."
        exit 1
    fi
fi

ARCHIVE="obeya_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

echo "Installing obeya v${VERSION} (${OS}/${ARCH})..."

# Create temp directory
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Download and extract
echo "Downloading ${URL}..."
if ! curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"; then
    echo "Error: Download failed. Check that version v${VERSION} exists at:"
    echo "  https://github.com/${REPO}/releases"
    exit 1
fi

tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Install binary
mkdir -p "$INSTALL_DIR"
if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$(id -u)" -ne 0 ]; then
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo install -m 755 "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    install -m 755 "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "Successfully installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

# Check if install dir is in PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*)
        echo "Run 'ob --help' to get started."
        ;;
    *)
        echo ""
        echo "NOTE: ${INSTALL_DIR} is not in your PATH."
        echo "Add it by running:"
        echo ""
        echo "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc"
        echo ""
        echo "Then restart your shell or run: source ~/.bashrc"
        ;;
esac
