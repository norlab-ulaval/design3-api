#!/usr/bin/env bash
# Run the Go toolchain inside Docker (no Go needed on the host).
# Usage: ./go.sh test ./...   (race detector: CGO_ENABLED=1 ./go.sh test -race ./...)
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p .cache
exec docker run --rm -u "$(id -u):$(id -g)" \
  -e HOME=/tmp -e GOCACHE=/src/.cache/build -e GOMODCACHE=/src/.cache/mod \
  -e CGO_ENABLED="${CGO_ENABLED:-0}" \
  -v "$PWD":/src -w /src golang:1.25 go "$@"
