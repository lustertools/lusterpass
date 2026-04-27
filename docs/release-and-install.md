# Release and Install Guide

## How Releases Work

Releases are automated via GitHub Actions. Pushing a version tag triggers the full pipeline.

### Triggering a Release

```bash
git tag v0.1.0
git push origin v0.1.0
```

This runs `.github/workflows/release.yml` which:

1. **Builds** binaries for 4 platforms (CGO enabled, each on its native runner):
   - `lusterpass-linux-amd64`
   - `lusterpass-linux-arm64`
   - `lusterpass-darwin-arm64` (Apple Silicon; Intel Macs use this via Rosetta 2)
   - `lusterpass-windows-amd64.exe`

2. **Generates** SHA-256 checksums for all binaries

3. **Creates a release** on the [lusterpass repo](https://github.com/lustertools/lusterpass/releases) with all binaries + `checksums.txt` attached

### CI (Continuous Integration)

Every push to `main` and every PR runs `.github/workflows/ci.yml`:
- Build on ubuntu-latest + macos-latest
- Unit tests

## Installing lusterpass

### One-liner (recommended)

```bash
curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash
```

### Options

```bash
# Install a specific version
VERSION=v0.1.0 curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash

# Install to a custom directory
INSTALL_DIR=~/.local/bin curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash
```

### What the installer does

1. Detects OS (linux, darwin, windows) and architecture (amd64, arm64)
2. Fetches the latest version from `lustertools/lusterpass` releases
3. Downloads the correct binary
4. Installs to `/usr/local/bin` (or custom `INSTALL_DIR`)
5. Verifies the installation

### After install

```bash
lusterpass login            # set up access token + org ID
lusterpass migrate .envrc   # migrate existing secrets
lusterpass --help           # see all commands
```
