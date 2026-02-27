#!/usr/bin/env bash
#
# Installs the latest wtw binary for your OS and architecture.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mehran/wtw/main/install.sh | bash
#
# To install to a custom directory:
#   INSTALL_DIR=~/.local/bin bash install.sh

set -euo pipefail

REPO="mehran/wtw"
BINARY="wtw"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)       ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Fetch the latest release tag
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | cut -d'"' -f4)

if [[ -z "$LATEST" ]]; then
  echo "Could not determine the latest release. Check your internet connection."
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$LATEST/${BINARY}_${OS}_${ARCH}.tar.gz"

echo "Installing wtw $LATEST ($OS/$ARCH) → $INSTALL_DIR/$BINARY"
curl -fsSL "$URL" | tar -xz -C /tmp "$BINARY"

# Use sudo only if needed
if [[ -w "$INSTALL_DIR" ]]; then
  mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
else
  sudo mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "Done. Run 'wtw --help' to get started."
