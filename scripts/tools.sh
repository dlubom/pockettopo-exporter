#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod"
export GOBIN="$root/.tools/bin" GOTOOLCHAIN=local
go install honnef.co/go/tools/cmd/staticcheck@v0.8.1
go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
