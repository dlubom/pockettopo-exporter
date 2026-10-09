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
mkdir "$trial/mutation-bin"
cp scripts/mutation-go.sh "$trial/mutation-bin/go"
chmod +x "$trial/mutation-bin/go"
export POCKETTOPO_MUTATION_REAL_GO
POCKETTOPO_MUTATION_REAL_GO=$(command -v go)
export PATH="$trial/mutation-bin:$PATH"
rm -f mutation.json
cd "$trial"
"$root/.tools/bin/gremlins" unleash --workers 2 --exclude-files 'cmd/' \
  --invert-assignments --invert-bitwise \
  --output "$root/mutation.json"
# Fail closed on empty/incomplete runs, timeouts, invalid or unknown statuses.
# Gremlins' built-in efficacy excludes several of those statuses.
jq -e -f "$root/scripts/mutation-gate.jq" "$root/mutation.json"
bash "$root/scripts/mutation-station.sh"
bash "$root/scripts/mutation-prefix.sh"
bash "$root/scripts/mutation-measurements.sh"
bash "$root/scripts/mutation-references.sh"
bash "$root/scripts/mutation-overview.sh"
bash "$root/scripts/mutation-plan-mapping.sh"
bash "$root/scripts/mutation-plan-marker.sh"
bash "$root/scripts/mutation-plan-polygon-count.sh"
bash "$root/scripts/mutation-plan-polygon-points.sh"
bash "$root/scripts/mutation-plan-polygon-color.sh"
