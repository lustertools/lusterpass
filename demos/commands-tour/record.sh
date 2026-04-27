#!/usr/bin/env bash
# Render the lusterpass commands-tour demo GIF.
# Idempotent: installs vhs/ttyd via Homebrew if missing, then runs vhs demo.tape.

set -euo pipefail

cd "$(dirname "$0")"

if command -v brew >/dev/null 2>&1; then
  export PATH="$(brew --prefix)/bin:$PATH"
fi

for dep in vhs ttyd; do
  if ! command -v "$dep" >/dev/null 2>&1; then
    if ! command -v brew >/dev/null 2>&1; then
      echo "[record] $dep missing and Homebrew not available — install $dep manually" >&2
      exit 1
    fi
    echo "[record] $dep not found — installing via Homebrew"
    brew install "$dep"
  fi
done

export DEMO_DIR="$(pwd)"

echo "[record] rendering demo.tape -> commands-tour-demo.gif"
vhs demo.tape

echo "[record] done -> $(pwd)/commands-tour-demo.gif"
ls -lh commands-tour-demo.gif
