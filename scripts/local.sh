#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Building ob from source (clean)..."
go build -a -o ob .

echo "Running: ./ob $*"
exec ./ob "$@"
