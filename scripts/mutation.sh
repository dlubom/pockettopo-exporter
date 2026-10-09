#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod"
export GOTOOLCHAIN=local
# Gremlins copies the entire module, including ignored caches and .git.
# A source-only copy also keeps all mutation writes away from the checkout.
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
if [[ -f go.sum ]]; then cp go.sum "$trial/"; fi
cp -R cmd internal "$trial/"
rm -f mutation.json
cd "$trial"
"$root/.tools/bin/gremlins" unleash --workers 2 --exclude-files 'cmd/' \
  --output "$root/mutation.json"
# Fail closed on empty/incomplete runs, timeouts, invalid or unknown statuses.
# Gremlins' built-in efficacy excludes several of those statuses.
jq -e -f "$root/scripts/mutation-gate.jq" "$root/mutation.json"
