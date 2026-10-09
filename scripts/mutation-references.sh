#!/usr/bin/env bash
# Valid raw-field, coordinate, comment, copy and native-acceptance faults in a source copy.
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
model=$(cat internal/source/reference_prefix.go)
reader=$(cat internal/top/references.go)
report="$root/build/reference-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/reference_prefix.go
  printf '%s\n' "$reader" >internal/top/references.go
  original=$(cat "$file")
  if [[ "$original" != *"$before"* ]]; then
    printf 'Missing mutation target: %s\n' "$name" >&2
    exit 1
  fi
  # Quote the replacement pattern: field subscripts must be literal, not globs.
  remaining=${original/"$before"/}
  if [[ "$remaining" == *"$before"* ]]; then
    printf 'Ambiguous mutation target: %s\n' "$name" >&2
    exit 1
  fi
  changed=${original/"$before"/"$after"}
  printf '%s\n' "$changed" >"$file"
  go build ./internal/source ./internal/top
  go test -json -count=1 -timeout=30s -run '^TestReferencePrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestReferencePrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a reference assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED reference mutation: %s\n' "$name"
}

model_file=internal/source/reference_prefix.go
reader_file=internal/top/references.go
mutate discard-station "$model_file" 'station: station' 'station: NewStationID(0)'
mutate discard-east "$model_file" 'east: east' 'east: 0'
mutate discard-north "$model_file" 'north: north' 'north: 0'
mutate discard-altitude "$model_file" 'altitude: altitude' 'altitude: 0'
mutate discard-comment "$model_file" 'comment: comment' 'comment: ""'
mutate discard-offsets "$model_file" 'offsets: offsets' 'offsets: ReferenceOffsets{}'
mutate hide-comment-bytes "$model_file" 'return []byte(r.comment)' 'return nil'
mutate discard-count "$model_file" 'count: count' 'count: 0'
mutate discard-measurements "$model_file" 'measurements: measurements' 'measurements: MeasurementPrefix{}'
mutate discard-references "$model_file" 'references: append([]Reference(nil), references...)' 'references: nil'
mutate alias-input-references "$model_file" 'references: append([]Reference(nil), references...)' 'references: references'
mutate alias-output-references "$model_file" 'return append([]Reference(nil), p.references...)' 'return p.references'
mutate alias-input-prefix "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate discard-count-offset "$model_file" 'countOffset: countOffset' 'countOffset: 0'
mutate big-endian-count "$reader_file" 'binary.LittleEndian.Uint32(countBytes)' 'binary.BigEndian.Uint32(countBytes)'
mutate big-endian-id "$reader_file" 'binary.LittleEndian.Uint32(fields[0])' 'binary.BigEndian.Uint32(fields[0])'
mutate big-endian-east "$reader_file" 'binary.LittleEndian.Uint64(fields[1])' 'binary.BigEndian.Uint64(fields[1])'
mutate big-endian-north "$reader_file" 'binary.LittleEndian.Uint64(fields[2])' 'binary.BigEndian.Uint64(fields[2])'
mutate big-endian-altitude "$reader_file" 'binary.LittleEndian.Uint32(fields[3])' 'binary.BigEndian.Uint32(fields[3])'
mutate swap-east-north "$reader_file" 'east, north, altitude, string(b)' 'north, east, altitude, string(b)'
mutate float-east "$reader_file" 'east, north, altitude, string(b)' 'int64(float64(east)), north, altitude, string(b)'
mutate float-north "$reader_file" 'east, north, altitude, string(b)' 'east, int64(float64(north)), altitude, string(b)'
mutate narrow-east "$reader_file" 'east, north, altitude, string(b)' 'int64(int32(east)), north, altitude, string(b)'
mutate narrow-north "$reader_file" 'east, north, altitude, string(b)' 'east, int64(int32(north)), altitude, string(b)'
mutate scale-altitude "$reader_file" 'east, north, altitude, string(b)' 'east, north, altitude / 1000, string(b)'
mutate normalize-blank-east "$reader_file" 'lengthStart := r.offset' 'if east == -9223372036854775808 { east = 0 }; lengthStart := r.offset'
mutate normalize-blank-north "$reader_file" 'lengthStart := r.offset' 'if north == -9223372036854775808 { north = 0 }; lengthStart := r.offset'
mutate normalize-blank-altitude "$reader_file" 'lengthStart := r.offset' 'if altitude == -2147483648 { altitude = 0 }; lengthStart := r.offset'
mutate omit-length-span "$reader_file" 'CommentLength: source.Span{Start: lengthStart, End: commentStart},' 'CommentLength: source.Span{},'
mutate omit-empty-comment-anchor "$reader_file" 'Comment:       source.Span{Start: commentStart, End: r.offset},' 'Comment: source.Span{},'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 33 and all(.[]; .status == "KILLED")' "$report"
