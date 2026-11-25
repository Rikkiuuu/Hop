#!/bin/sh
set -eu

REPO="Rikkiuuu/hop"
VERSION="${1:-}"

if [ -z "$VERSION" ]; then
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)"
fi

if [ -z "$VERSION" ]; then
    VERSION="v1.0.0"
fi

VER="${VERSION#v}"
VERSION="v${VER}"

RAW_OS="$(uname -s)"
case "$RAW_OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)
        printf 'Unsupported operating system: %s\n' "$RAW_OS" >&2
        exit 1
        ;;
esac

RAW_ARCH="$(uname -m)"
case "$RAW_ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        printf 'Unsupported architecture: %s\n' "$RAW_ARCH" >&2
        exit 1
        ;;
esac

ARCHIVE="hop_${VER}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$URL" -o "$TMP_DIR/$ARCHIVE"
tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"

if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

cp "$TMP_DIR/hop" "$INSTALL_DIR/hop"
chmod +x "$INSTALL_DIR/hop"

if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$INSTALL_DIR/hop" 2>/dev/null || true
fi

printf 'Installed hop %s to %s/hop\n' "$VERSION" "$INSTALL_DIR"

case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) printf 'Warning: %s is not in your PATH\n' "$INSTALL_DIR" >&2 ;;
esac
