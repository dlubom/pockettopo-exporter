#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
groups=(gremlins tables first-polygon second-polygon markers)
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

run_gremlins() (
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
)

run_group() {
  local selected="$1" scope
  local scopes=()
  case "$selected" in
    gremlins) run_gremlins; return ;;
    tables) scopes=(station prefix measurements references) ;;
    first-polygon) scopes=(overview plan-mapping plan-marker plan-polygon-count plan-polygon-points plan-polygon-color) ;;
    second-polygon) scopes=(plan-second-polygon-count plan-second-polygon-points plan-second-polygon-color) ;;
    markers) scopes=(plan-next-marker plan-following-marker plan-third-polygon-count) ;;
    *) printf 'Missing mutation group implementation: %s\n' "$selected" >&2; exit 2 ;;
  esac
  for scope in "${scopes[@]}"; do
    bash "scripts/mutation-$scope.sh"
  done
}

if [[ "$group" == all ]]; then
  for candidate in "${groups[@]}"; do run_group "$candidate"; done
else
  run_group "$group"
fi
