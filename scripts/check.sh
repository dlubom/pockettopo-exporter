#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod"
export STATICCHECK_CACHE="$root/.cache/staticcheck" GOTOOLCHAIN=local

unformatted=$(gofmt -l cmd internal)
if [[ -n "$unformatted" ]]; then
  printf 'Run gofmt on:\n%s\n' "$unformatted" >&2
  exit 1
fi
go vet ./...
.tools/bin/staticcheck ./...
go test -race -count=1 ./...
go test -count=1 -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out
go tool cover -func=coverage.out | awk '
  $1 == "total:" { found = 1; if ($3 + 0 < 95) exit 1 }
  END { if (!found) exit 1 }
'
binary=build/pockettopo-exporter
if [[ $(go env GOOS) == windows ]]; then binary+=.exe; fi
go build -o "$binary" ./cmd/pockettopo-exporter
[[ "$("$binary" --version)" == 'pockettopo-exporter dev' ]]
"$binary" --help
if "$binary" export >build/unsupported.out 2>build/unsupported.err; then
  printf 'Unsupported export unexpectedly succeeded.\n' >&2
  exit 1
fi
[[ ! -s build/unsupported.out && -s build/unsupported.err ]]
