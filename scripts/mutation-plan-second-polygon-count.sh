#!/usr/bin/env bash
# Valid second Polygon count, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_second_polygon_count_prefix.go)
reader=$(cat internal/top/plan_second_polygon_count.go)
report="$root/build/plan-second-polygon-count-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_second_polygon_count_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_second_polygon_count.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanSecondPolygonCountPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanSecondPolygonCountPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan second Polygon count assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan second Polygon count mutation: %s\n' "$name"
}

model_file=internal/source/plan_second_polygon_count_prefix.go
reader_file=internal/top/plan_second_polygon_count.go
mutate discard-marker-prefix "$model_file" 'next: next' 'next: PlanNextMarkerPrefix{}'
mutate discard-count "$model_file" 'count: count' 'count: 0'
mutate narrow-count "$model_file" 'count: count' 'count: int32(uint16(count))'
mutate normalize-signed-source "$model_file" 'count: count' 'count: count & 0x7fffffff'
mutate discard-count-span "$model_file" 'span: span' 'span: Span{}'
mutate alias-input-prefix "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate confuse-count-span "$model_file" 'SecondPointCount: p.span' 'SecondPointCount: p.next.Offsets().NextMarker'
mutate expose-overview-as-plan "$model_file" 'return p.next.PlanMapping()' 'return p.next.OverviewMapping()'
mutate expose-plan-as-overview "$model_file" 'return p.next.OverviewMapping()' 'return p.next.PlanMapping()'
mutate hide-marker "$model_file" 'return p.next.MarkerRaw()' 'return 0'
mutate hardcoded-position "$reader_file" 'offset: marker.ConsumedOffset()' 'offset: 47'
mutate reread-marker "$reader_file" 'offset: marker.ConsumedOffset()' 'offset: marker.ConsumedOffset() - 1'
mutate replace-caller-reference-limits "$reader_file" 'ReadV3PlanNextMarkerPrefixWithLimits(data, limits)' 'ReadV3PlanNextMarkerPrefixWithLimits(data, DefaultPolygonCountLimits())'
mutate wrong-count-field "$reader_file" 'b, err := r.take(4, "plan.elements[1].point_count")' 'b, err := r.take(4, "plan.count")'
mutate wrong-marker-code "$reader_file" 'failure("unsupported_element",' 'failure("unknown_element",'
mutate wrong-marker-field "$reader_file" '"plan.elements[1].kind"' '"plan.marker"'
mutate wrong-marker-offset "$reader_file" 'marker.Offsets().NextMarker.Start)' 'marker.ConsumedOffset())'
mutate wrong-negative-code "$reader_file" 'failure("negative_count",' 'failure("resource_limit",'
mutate shift-count-start "$reader_file" 'Start: start, End: r.offset' 'Start: start + 1, End: r.offset'
mutate shift-count-end "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: r.offset + 1'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate change-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate big-endian-count "$reader_file" 'binary.LittleEndian.Uint32(b)' 'binary.BigEndian.Uint32(b)'
mutate discard-high-count-bytes "$reader_file" 'binary.LittleEndian.Uint32(b)' 'uint32(binary.LittleEndian.Uint16(b))'
mutate read-five-bytes "$reader_file" 'r.take(4,' 'r.take(5,'
mutate require-color "$reader_file" 'span := source.Span' 'if _, err := r.take(1, "plan.elements[1].color"); err != nil { return empty, err }; span := source.Span'
mutate preflight-points "$reader_file" 'span := source.Span' 'if int(count) > (len(data)-r.offset)/8 { return empty, failure("truncated", "plan.elements[1].points", r.offset) }; span := source.Span'
mutate reject-zero "$reader_file" 'if count < 0 {' 'if count <= 0 {'
mutate bypass-point-limit "$reader_file" 'if int(count) > limits.MaxPoints {' 'if false {'
mutate replace-caller-point-limit "$reader_file" 'if int(count) > limits.MaxPoints {' 'if int(count) > DefaultPolygonCountLimits().MaxPoints {'
mutate reject-limit-equality "$reader_file" 'if int(count) > limits.MaxPoints {' 'if int(count) >= limits.MaxPoints {'
mutate accept-marker-zero "$reader_file" 'if marker.NextMarkerRaw() != 1 {' 'if marker.NextMarkerRaw() != 1 && marker.NextMarkerRaw() != 0 {'
mutate accept-marker-three "$reader_file" 'if marker.NextMarkerRaw() != 1 {' 'if marker.NextMarkerRaw() != 1 && marker.NextMarkerRaw() != 3 {'
mutate partial-count-failure "$reader_file" 'b, err := r.take' 'empty = source.NewPlanSecondPolygonCountPrefix(marker, 0, source.Span{}, data[:start], len(data)-start); b, err := r.take'
mutate hide-first-count "$model_file" 'return p.next.PointCountRaw()' 'return p.count'
mutate hide-next-marker "$model_file" 'return p.next.NextMarkerRaw()' 'return 0'
mutate hide-color "$model_file" 'return p.next.ColorRaw()' 'return 0'
mutate discard-points "$model_file" 'return p.next.Points()' 'return nil'
mutate discard-trips "$model_file" 'return p.next.Trips()' 'return nil'
mutate discard-measurements "$model_file" 'return p.next.Measurements()' 'return nil'
mutate discard-references "$model_file" 'return p.next.References()' 'return nil'
mutate discard-inherited-offsets "$model_file" 'PlanNextMarkerPrefixOffsets: p.next.Offsets()' 'PlanNextMarkerPrefixOffsets: PlanNextMarkerPrefixOffsets{}'
mutate dispatch-first-marker "$reader_file" 'if marker.NextMarkerRaw() != 1 {' 'if marker.MarkerRaw() != 1 {'
mutate use-first-count "$reader_file" 'count := int32(binary.LittleEndian.Uint32(b))' 'count := marker.PointCountRaw() + int32(binary.LittleEndian.Uint32(b)) * 0'
mutate aggregate-budget "$reader_file" 'if int(count) > limits.MaxPoints {' 'if int(count) + int(marker.PointCountRaw()) > limits.MaxPoints {'
mutate wrong-count-offset "$reader_file" 'failure("negative_count", "plan.elements[1].point_count", start)' 'failure("negative_count", "plan.elements[1].point_count", start+1)'
mutate partial-inherited-failure "$reader_file" 'marker, err := ReadV3PlanNextMarkerPrefixWithLimits(data, limits)' 'marker, err := ReadV3PlanNextMarkerPrefixWithLimits(data, limits); if err != nil { return source.NewPlanSecondPolygonCountPrefix(marker, 0, source.Span{}, nil, 1), err }'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 50 and all(.[]; .status == "KILLED")' "$report"
