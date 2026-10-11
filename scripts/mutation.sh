#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
groups=(gremlins-tables gremlins-drawings gremlins-output gremlins-processing
  tables mappings first-polygon second-count second-points second-color
  next-markers third-count third-points third-color third-marker
  plan-loop plan-elements side-mapping side-marker document native-text processing)
if [[ $# -gt 1 ]]; then
  printf 'Expected at most one mutation group.\n' >&2
  exit 2
fi
group=${1:-all}
if [[ "$group" == --matrix ]]; then
  printf '%s\n' "${groups[@]}" | jq -R . | jq -sc '{group: .}'
  exit 0
fi
if [[ "$group" != all ]]; then
  found=false
  for candidate in "${groups[@]}"; do
    if [[ "$group" == "$candidate" ]]; then found=true; fi
  done
  if [[ "$found" != true ]]; then
    printf 'Unknown mutation group: %s\n' "$group" >&2
    exit 2
  fi
fi

# Every production internal Go file belongs to exactly one shard. Unknown new
# packages fail closed until assigned; all shards still use the full test suite.
gremlins_shard() {
  case "$1" in
    internal/source/*|internal/top/prefix.go|internal/top/measurements.go|internal/top/references.go) printf tables ;;
    internal/top/*) printf drawings ;;
    internal/cli/*|internal/exporttext/*|internal/nativefmt/*) printf output ;;
    internal/processing/*) printf processing ;;
    *) printf 'Unassigned Gremlins source file: %s\n' "$1" >&2; return 2 ;;
  esac
}

run_gremlins() (
selected="$1"
exclusions=(--exclude-files 'cmd/')
for file in internal/*/*.go; do
  case "$file" in *_test.go) continue ;; esac
  shard=$(gremlins_shard "$file")
  if [[ "$shard" != "$selected" ]]; then
    exclusions+=(--exclude-files "^${file//./\\.}$")
  fi
done
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
"$root/.tools/bin/gremlins" unleash --workers 2 "${exclusions[@]}" \
  --invert-assignments --invert-bitwise \
  --output "$root/mutation.json"
# Fail closed on empty/incomplete runs, timeouts, invalid or unknown statuses.
# Gremlins' built-in efficacy excludes several of those statuses.
jq -e -f "$root/scripts/mutation-gate.jq" "$root/mutation.json"
)

# Optional local disk control; reports and source evidence are retained. Each
# cleanup runs only after a completed campaign, never alongside a compiler.
prune_cache() {
  if [[ "${POCKETTOPO_MUTATION_PRUNE_CACHE:-false}" == true ]]; then
    local root
    root=$(pwd -W 2>/dev/null || pwd)
    GOCACHE="$root/.cache/go-build" GOTOOLCHAIN=local go clean -cache
  fi
}

run_group() {
  local selected="$1" scope
  local scopes=()
  case "$selected" in
    gremlins-*) run_gremlins "${selected#gremlins-}"; prune_cache; return ;;
    tables) scopes=(station prefix measurements references) ;;
    mappings) scopes=(overview plan-mapping) ;;
    first-polygon) scopes=(plan-marker plan-polygon-count plan-polygon-points plan-polygon-color) ;;
    second-count) scopes=(plan-second-polygon-count) ;;
    second-points) scopes=(plan-second-polygon-points) ;;
    second-color) scopes=(plan-second-polygon-color) ;;
    plan-loop) scopes=(plan-polygons) ;;
    plan-elements) scopes=(plan-elements) ;;
    side-mapping) scopes=(side-mapping) ;;
    side-marker) scopes=(side-marker) ;;
    document) scopes=(document) ;;
    native-text) scopes=(text) ;;
    processing) scopes=(processing geometry) ;;
    next-markers) scopes=(plan-next-marker plan-following-marker) ;;
    third-count) scopes=(plan-third-polygon-count) ;;
    third-points) scopes=(plan-third-polygon-points) ;;
    third-color) scopes=(plan-third-polygon-color) ;;
    third-marker) scopes=(plan-third-next-marker) ;;
    *) printf 'Missing mutation group implementation: %s\n' "$selected" >&2; exit 2 ;;
  esac
  for scope in "${scopes[@]}"; do
    bash "scripts/mutation-$scope.sh"
    prune_cache
  done
}

if [[ "$group" == all ]]; then
  for candidate in "${groups[@]}"; do run_group "$candidate"; done
else
  run_group "$group"
fi
