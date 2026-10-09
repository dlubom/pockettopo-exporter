#!/usr/bin/env bash
# Valid field/copy/endian faults beyond Gremlins' token operators, in a copy.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod"
export GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p build
model=$(cat internal/source/trip_prefix.go)
reader=$(cat internal/top/prefix.go)
report="$root/build/prefix-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining
  printf '%s\n' "$model" >internal/source/trip_prefix.go
  printf '%s\n' "$reader" >internal/top/prefix.go
  original=$(cat "$file")
  if [[ "$original" != *"$before"* ]]; then
    printf 'Missing mutation target: %s\n' "$name" >&2
    exit 1
  fi
  remaining=${original/$before/}
  if [[ "$remaining" == *"$before"* ]]; then
    printf 'Ambiguous mutation target: %s\n' "$name" >&2
    exit 1
  fi
  printf '%s\n' "${original/$before/$after}" >"$file"
  go build ./internal/source ./internal/top
  go test -json -count=1 -timeout=30s -run '^(TestPrefix|TestTripPrefix)' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^Test(Prefix|TripPrefix)"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a prefix assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED prefix mutation: %s\n' "$name"
}

model_file=internal/source/trip_prefix.go
reader_file=internal/top/prefix.go
mutate discard-ticks "$model_file" 'ticks: ticks' 'ticks: 0'
mutate discard-comment "$model_file" 'comment: comment' 'comment: ""'
mutate normalize-declination "$model_file" 'declination: declination' 'declination: 0'
mutate discard-offsets "$model_file" 'offsets: offsets' 'offsets: TripOffsets{}'
mutate hide-comment-bytes "$model_file" 'return []byte(t.comment)' 'return nil'
mutate discard-header "$model_file" 'header: header' 'header: [4]byte{}'
mutate discard-count "$model_file" 'count: count' 'count: 0'
mutate discard-trips "$model_file" 'trips: append([]Trip(nil), trips...)' 'trips: nil'
mutate alias-input-trips "$model_file" 'trips: append([]Trip(nil), trips...)' 'trips: trips'
mutate alias-output-trips "$model_file" 'return append([]Trip(nil), p.trips...)' 'return p.trips'
mutate alias-input-prefix "$model_file" 'prefix: bytes.Clone(prefix)' 'prefix: prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate big-endian-count "$reader_file" 'binary.LittleEndian.Uint32(countBytes)' 'binary.BigEndian.Uint32(countBytes)'
mutate big-endian-ticks "$reader_file" 'binary.LittleEndian.Uint64(ticksBytes)' 'binary.BigEndian.Uint64(ticksBytes)'
mutate big-endian-declination "$reader_file" 'binary.LittleEndian.Uint16(declinationBytes)' 'binary.BigEndian.Uint16(declinationBytes)'
mutate hide-error-code "$reader_file" 'err.Code, err.Field, err.Offset = code, field, offset' 'err.Code, err.Field, err.Offset = "unknown", field, offset'
mutate hide-count-offset "$model_file" 'TripCount: Span{Start: 4, End: 8}' 'TripCount: Span{}'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 19 and all(.[]; .status == "KILLED")' "$report"
