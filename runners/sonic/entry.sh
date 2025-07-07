#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# build once, reuse on subsequent calls
if [ ! -f build/runner ]; then
  mkdir -p build
  go mod download
  go build -o build/runner ./...
fi

exec build/runner "$@"
