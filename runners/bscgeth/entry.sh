#!/usr/bin/env bash
set -euo pipefail

# resolve folder containing this script
DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

cd "$DIR"
go run ./runner.go "$@"
