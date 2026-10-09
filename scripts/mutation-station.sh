#!/usr/bin/env bash
# Six valid source mutations beyond Gremlins' token operators, in a copy.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod"
export GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
mkdir -p "$trial/internal/source" build
cp go.mod "$trial/"
cp internal/source/*.go "$trial/internal/source/"
original=$(cat internal/source/station_id.go)
report="$root/build/station-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" before="$2" after="$3" status=0 remaining
  # Require exactly one target; a stale/ambiguous mutation is a failed run.
  if [[ "$original" != *"$before"* ]]; then
    printf 'Missing mutation target: %s\n' "$name" >&2
    exit 1
  fi
  remaining=${original/$before/}
  if [[ "$remaining" == *"$before"* ]]; then
    printf 'Ambiguous mutation target: %s\n' "$name" >&2
    exit 1
  fi
  printf '%s\n' "${original/$before/$after}" >internal/source/station_id.go
  # Compile first: compilation errors cannot count as behavioral kills.
  go build ./internal/source
  go test -json -count=1 -timeout=30s ./internal/source >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | startswith("TestStationID"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a station assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED station mutation: %s\n' "$name"
}

mutate discard-source-bits 'StationID{raw: raw}' 'StationID{raw: 0}'
mutate hide-source-bits 'return id.raw' 'return 0'
mutate compare-raw-records 'id.NativeValue() == other.NativeValue()' 'id == other'
mutate compare-display-names 'id.NativeValue() == other.NativeValue()' 'id.String() == other.String()'
mutate show-reserved-id 'return ""' 'return "reserved"'
mutate omit-major-separator '"%d.%d"' '"%d%d"'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 6 and all(.[]; .status == "KILLED")' "$report"
