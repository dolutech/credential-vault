#!/bin/sh
# Credential Vault install script
# Installs credential-vault system-wide or for the current user
# Works on: Linux (Arch, Debian, RHEL, etc.), macOS

set -e

BINARY_NAME="credential-vault"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Determine install location
if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ "$EUID" -ne 0 ] 2>/dev/null && [ -w "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    # Create user bin if it doesn't exist
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing $BINARY_NAME to $INSTALL_DIR..."

# Build if binary doesn't exist
if [ ! -f "$REPO_DIR/$BINARY_NAME" ]; then
    echo "Building from source..."
    cd "$REPO_DIR"
    go build -ldflags "-X credential-vault/internal/cli.Version=0.1.0" -o "$BINARY_NAME" ./cmd/credential-vault
fi

# Copy binary
cp "$REPO_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo ""
echo "Installed successfully!"
echo ""
echo "Binary location: $INSTALL_DIR/$BINARY_NAME"
echo ""
if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
    echo "Make sure $INSTALL_DIR is in your PATH:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo "Add this to your ~/.bashrc or ~/.zshrc"
fi
echo ""
echo "Next steps:"
echo "  1. $BINARY_NAME init"
echo "  2. $BINARY_NAME add \"Server PROD 1\""
echo "  3. Configure in your AI assistant (see docs/MCP_CONFIG.md)"