#!/usr/bin/env bash
# Valid first Polygon count, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_polygon_count_prefix.go)
reader=$(cat internal/top/plan_polygon_count.go)
report="$root/build/plan-polygon-count-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_polygon_count_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_polygon_count.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanPolygonCountPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanPolygonCountPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan Polygon count assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan Polygon count mutation: %s\n' "$name"
}

model_file=internal/source/plan_polygon_count_prefix.go
reader_file=internal/top/plan_polygon_count.go
mutate discard-marker-prefix "$model_file" 'marker: marker' 'marker: PlanMarkerPrefix{}'
mutate discard-count "$model_file" 'count: count' 'count: 0'
mutate narrow-count "$model_file" 'count: count' 'count: int32(uint16(count))'
mutate normalize-signed-source "$model_file" 'count: count' 'count: count & 0x7fffffff'
mutate discard-count-span "$model_file" 'span: span' 'span: Span{}'
mutate alias-input-prefix "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate confuse-count-span "$model_file" 'PointCount: p.span' 'PointCount: p.marker.Offsets().Marker'
mutate expose-overview-as-plan "$model_file" 'return p.marker.PlanMapping()' 'return p.marker.OverviewMapping()'
mutate expose-plan-as-overview "$model_file" 'return p.marker.OverviewMapping()' 'return p.marker.PlanMapping()'
mutate hide-marker "$model_file" 'return p.marker.MarkerRaw()' 'return 0'
mutate hardcoded-position "$reader_file" 'offset: marker.ConsumedOffset()' 'offset: 41'
mutate reread-marker "$reader_file" 'offset: marker.ConsumedOffset()' 'offset: marker.ConsumedOffset() - 1'
mutate replace-caller-reference-limits "$reader_file" 'ReadV3PlanMarkerPrefixWithLimits(data, limits.ReferenceLimits)' 'ReadV3PlanMarkerPrefixWithLimits(data, DefaultReferenceLimits())'
mutate wrong-count-field "$reader_file" 'b, err := r.take(4, "plan.elements[0].point_count")' 'b, err := r.take(4, "plan.count")'
mutate wrong-marker-code "$reader_file" 'failure("unsupported_element",' 'failure("unknown_element",'
mutate wrong-marker-field "$reader_file" '"plan.elements[0].kind"' '"plan.marker"'
mutate wrong-marker-offset "$reader_file" 'marker.Offsets().Marker.Start)' 'marker.ConsumedOffset())'
mutate wrong-negative-code "$reader_file" 'failure("negative_count",' 'failure("resource_limit",'
mutate shift-count-start "$reader_file" 'Start: start, End: r.offset' 'Start: start + 1, End: r.offset'
mutate shift-count-end "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: r.offset + 1'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate change-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate big-endian-count "$reader_file" 'binary.LittleEndian.Uint32(b)' 'binary.BigEndian.Uint32(b)'
mutate discard-high-count-bytes "$reader_file" 'binary.LittleEndian.Uint32(b)' 'uint32(binary.LittleEndian.Uint16(b))'
mutate read-five-bytes "$reader_file" 'r.take(4,' 'r.take(5,'
mutate require-color "$reader_file" 'span := source.Span' 'if _, err := r.take(1, "plan.elements[0].color"); err != nil { return empty, err }; span := source.Span'
mutate preflight-points "$reader_file" 'span := source.Span' 'if int(count) > (len(data)-r.offset)/8 { return empty, failure("truncated", "plan.elements[0].points", r.offset) }; span := source.Span'
mutate reject-zero "$reader_file" 'if count < 0 {' 'if count <= 0 {'
mutate bypass-point-limit "$reader_file" 'if int(count) > limits.MaxPoints {' 'if false {'
mutate replace-caller-point-limit "$reader_file" 'if int(count) > limits.MaxPoints {' 'if int(count) > DefaultPolygonCountLimits().MaxPoints {'
mutate reject-limit-equality "$reader_file" 'if int(count) > limits.MaxPoints {' 'if int(count) >= limits.MaxPoints {'
mutate ignore-negative-limit "$reader_file" 'limits.MaxPoints < 0' 'limits.MaxPoints < -1'
mutate ignore-above-default-limit "$reader_file" 'limits.MaxPoints > DefaultPolygonCountLimits().MaxPoints' 'limits.MaxPoints > 2147483647'
mutate reject-zero-limit "$reader_file" 'limits.MaxPoints < 0' 'limits.MaxPoints <= 0'
mutate lower-default-ceiling "$reader_file" 'MaxPoints: 1_000_000' 'MaxPoints: 999999'
mutate accept-marker-zero "$reader_file" 'if marker.MarkerRaw() != 1 {' 'if marker.MarkerRaw() != 1 && marker.MarkerRaw() != 0 {'
mutate accept-marker-three "$reader_file" 'if marker.MarkerRaw() != 1 {' 'if marker.MarkerRaw() != 1 && marker.MarkerRaw() != 3 {'
mutate partial-count-failure "$reader_file" 'b, err := r.take' 'empty = source.NewPlanPolygonCountPrefix(marker, 0, source.Span{}, data[:start], len(data)-start); b, err := r.take'
# Delay the new limit until after inherited validation to test precedence.
limit_check='if limits.MaxPoints < 0 || limits.MaxPoints > DefaultPolygonCountLimits().MaxPoints {
		return empty, failure("invalid_limit", "limits.max_points", 0)
	}'
without_check=${reader/"$limit_check"/}
delayed_check='if limits.MaxPoints < 0 || limits.MaxPoints > DefaultPolygonCountLimits().MaxPoints { return empty, failure("invalid_limit", "limits.max_points", 0) }; r := reader'
delayed_reader=${without_check/"r := reader"/"$delayed_check"}
mutate delay-new-limit "$reader_file" "$reader" "$delayed_reader"
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 42 and all(.[]; .status == "KILLED")' "$report"
