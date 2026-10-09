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
  --threshold-efficacy 90 --threshold-mcover 100 --output "$root/mutation.json"
# Fail closed on empty/incomplete runs, timeouts, invalid or unreviewed mutants.
# Gremlins' built-in efficacy excludes several of those statuses.
jq -e '
  [.files[].mutations[]] as $m |
  ($m | length) > 0 and
  ($m | length) == .mutants_total and
  all($m[]; .status == "KILLED" or .status == "LIVED") and
  ([ $m[] | select(.status == "KILLED") ] | length) / ($m | length) >= 0.9
' "$root/mutation.json"
