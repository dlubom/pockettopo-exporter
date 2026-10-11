#!/usr/bin/env bash
# Valid raw marker, presence, span, bound and copy faults in a disposable copy.
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
model=$(cat internal/source/side_marker_prefix.go)
reader=$(cat internal/top/side_marker.go)
report="$root/build/side-marker-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/side_marker_prefix.go
  printf '%s\n' "$reader" >internal/top/side_marker.go
  original=$(cat "$file")
  if [[ "$original" != *"$before"* ]]; then
    printf 'Missing mutation target: %s\n' "$name" >&2
    exit 1
  fi
  remaining=${original/"$before"/}
  if [[ "$remaining" == *"$before"* ]]; then
    printf 'Ambiguous mutation target: %s\n' "$name" >&2
    exit 1
  fi
  changed=${original/"$before"/"$after"}
  printf '%s\n' "$changed" >"$file"
  go build ./internal/source ./internal/top
  go test -json -count=1 -timeout=30s -run '^TestSideMarkerPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestSideMarkerPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by an side marker assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED side marker mutation: %s\n' "$name"
}

model_file=internal/source/side_marker_prefix.go
reader_file=internal/top/side_marker.go
mutate discard-side "$model_file" 'SideMappingPrefix: side' 'SideMappingPrefix: SideMappingPrefix{}'
mutate discard-marker "$model_file" 'marker: marker' 'marker: 0'
mutate discard-presence "$model_file" 'present: present' 'present: false'
mutate discard-span "$model_file" 'span: span' 'span: Span{}'
mutate alias-input "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate model-hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-marker "$model_file" 'return p.marker, p.present' 'return p.StopMarkerRaw(), p.present'
mutate hide-presence "$model_file" 'return p.marker, p.present' 'return p.marker, true'
mutate hide-span "$model_file" 'SideMarker: p.span' 'SideMarker: Span{}'
mutate hide-side-spans "$model_file" 'SideMappingPrefixOffsets: p.SideMappingPrefix.Offsets()' 'SideMappingPrefixOffsets: SideMappingPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3SideMappingPrefixWithLimits(data, limits)' 'ReadV3SideMappingPrefixWithLimits(data, DefaultPlanElementLimits())'
mutate default-zero-limits "$reader_file" 'DefaultPlanElementLimits())' 'PlanElementLimits{})'
mutate reverse-stop "$reader_file" '!present {' 'present {'
mutate read-after-unsupported "$reader_file" '!present {' 'present && false {'
mutate unsupported-presence "$reader_file" 'side, 0, false' 'side, 0, true'
mutate unsupported-span "$reader_file" 'false, source.Span{}' 'false, source.Span{Start:1,End:2}'
mutate unsupported-marker "$reader_file" 'side, 0, false' 'side, 255, false'
mutate unsupported-consume-tail "$reader_file" 'data[:side.ConsumedOffset()]' 'data'
mutate unsupported-hide-tail "$reader_file" 'len(data)-side.ConsumedOffset()' '0'
mutate inherited-partial "$reader_file" 'side, err := ReadV3SideMappingPrefixWithLimits(data, limits)' 'empty = source.NewSideMarkerPrefix(source.SideMappingPrefix{}, 1, true, source.Span{}, nil, len(data)); side, err := ReadV3SideMappingPrefixWithLimits(data, limits)'
mutate marker-partial "$reader_file" 'b, err := r.take' 'empty = source.NewSideMarkerPrefix(side, 1, true, source.Span{}, data[:r.offset], len(data)-r.offset); b, err := r.take'
mutate wrong-offset "$reader_file" 'offset: side.ConsumedOffset()' 'offset: side.ConsumedOffset()-1'
mutate wrong-field "$reader_file" '"side.elements[0].kind"' '"plan.elements[0].kind"'
mutate extra-marker "$reader_file" 'r.take(1,' 'r.take(2,'
mutate reject-unknown "$reader_file" 'span := source.Span' 'if b[0] != 0 && b[0] != 1 && b[0] != 3 { return empty, failure("unknown_element", "side.elements[0].kind", start) }; span := source.Span'
mutate normalize-marker "$reader_file" 'side, b[0], true' 'side, b[0]&3, true'
mutate success-presence "$reader_file" 'side, b[0], true' 'side, b[0], false'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate reader-hide-tail "$reader_file" 'len(data)-r.offset' '0'
mutate span-start "$reader_file" 'Start: start, End: r.offset' 'Start: start+1, End: r.offset'
mutate span-end "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: r.offset+1'
mutate change-input "$reader_file" 'start := r.offset' 'data[0]=0; start := r.offset'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 34 and ([.[].name] | unique | length) == 34 and all(.[]; .status == "KILLED")' "$report"
