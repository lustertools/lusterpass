#!/usr/bin/env bash
set -euo pipefail

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
OUT="build/lusterpass"

echo "Building lusterpass ${VERSION}..."
mkdir -p build
go build -trimpath -ldflags "-X main.version=${VERSION}" -o "${OUT}" .

echo "Installing to /usr/local/bin/lusterpass..."
sudo cp "${OUT}" /usr/local/bin/lusterpass

echo "Done: $(lusterpass --version)"
