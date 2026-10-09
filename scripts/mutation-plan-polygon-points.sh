#!/usr/bin/env bash
# Valid first Polygon points, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_polygon_points_prefix.go)
reader=$(cat internal/top/plan_polygon_points.go)
report="$root/build/plan-polygon-points-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_polygon_points_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_polygon_points.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanPolygonPointsPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanPolygonPointsPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan Polygon points assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan Polygon points mutation: %s\n' "$name"
}

model_file=internal/source/plan_polygon_points_prefix.go
reader_file=internal/top/plan_polygon_points.go
mutate discard-base "$model_file" 'count: count' 'count: PlanPolygonCountPrefix{}'
mutate discard-x "$model_file" 'x: x' 'x: 0'
mutate discard-y "$model_file" 'y: y' 'y: 0'
mutate swap-xy "$model_file" 'x: x, y: y' 'x: y, y: x'
mutate narrow-x "$model_file" 'x: x' 'x: int32(int16(x))'
mutate float-y "$model_file" 'y: y' 'y: int32(float32(y))'
mutate normalize-x "$model_file" 'x: x' 'x: x & 0x7fffffff'
mutate discard-point-spans "$model_file" 'offsets: offsets' 'offsets: PolygonPointOffsets{}'
mutate alias-input-points "$model_file" 'slices.Clone(points)' 'points'
mutate alias-output-points "$model_file" 'return slices.Clone(p.points)' 'return p.points'
mutate alias-input-bytes "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-bytes "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate discard-table-span "$model_file" 'span: span' 'span: Span{}'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate wrong-plan "$model_file" 'return p.count.PlanMapping()' 'return p.count.OverviewMapping()'
mutate wrong-overview "$model_file" 'return p.count.OverviewMapping()' 'return p.count.PlanMapping()'
mutate wrong-count "$model_file" 'return p.count.PointCountRaw()' 'return 0'
mutate replace-limits "$reader_file" 'ReadV3PlanPolygonCountPrefixWithLimits(data, limits)' 'ReadV3PlanPolygonCountPrefixWithLimits(data, DefaultPolygonCountLimits())'
mutate wrong-start "$reader_file" 'start := count.ConsumedOffset()' 'start := 45'
mutate reread-count "$reader_file" 'start := count.ConsumedOffset()' 'start := count.ConsumedOffset() - 4'
mutate preflight-off-by-one "$reader_file" 'if n > (len(data)-start)/8' 'if n > (len(data)-start)/8 + 1'
mutate reject-exact-preflight "$reader_file" 'if n > (len(data)-start)/8' 'if n >= (len(data)-start)/8'
mutate wrong-error-code "$reader_file" 'failure("truncated",' 'failure("resource_limit",'
mutate wrong-error-field "$reader_file" '"plan.elements[0].points"' '"plan.points"'
mutate wrong-error-offset "$reader_file" '"plan.elements[0].points", start' '"plan.elements[0].points", start - 4'
mutate big-endian-x "$reader_file" 'binary.LittleEndian.Uint32(data[offset : offset+4])' 'binary.BigEndian.Uint32(data[offset : offset+4])'
mutate big-endian-y "$reader_file" 'binary.LittleEndian.Uint32(data[offset+4 : offset+8])' 'binary.BigEndian.Uint32(data[offset+4 : offset+8])'
mutate reverse-order "$reader_file" 'points[i] =' 'points[len(points)-1-i] ='
mutate shift-record "$reader_file" 'Record: source.Span{Start: offset, End: offset + 8}' 'Record: source.Span{Start: offset+1, End: offset + 8}'
mutate wrong-x-span "$reader_file" 'X:      source.Span{Start: offset, End: offset + 4}' 'X: source.Span{Start: offset+4, End: offset+8}'
mutate wrong-y-span "$reader_file" 'Y:      source.Span{Start: offset + 4, End: offset + 8}' 'Y: source.Span{Start: offset, End: offset+4}'
mutate consume-tail "$reader_file" 'data[:offset]' 'data'
mutate mutate-input "$reader_file" 'offset := start' 'data[0] = 0; offset := start'
mutate require-color "$reader_file" 'points := make' 'if len(data)-start <= n*8 { return empty, failure("truncated", "plan.elements[0].color", start+n*8) }; points := make'
mutate read-color-zero "$reader_file" 'points := make' 'if n == 0 && len(data) == start { return empty, failure("truncated", "plan.elements[0].color", start) }; points := make'
mutate partial-preflight "$reader_file" 'return empty, failure("truncated",' 'return source.NewPlanPolygonPointsPrefix(count, nil, source.Span{}, data[:start], len(data)-start), failure("truncated",'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 37 and all(.[]; .status == "KILLED")' "$report"
