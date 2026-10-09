#!/usr/bin/env bash
# Valid first Polygon color, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_polygon_color_prefix.go)
reader=$(cat internal/top/plan_polygon_color.go)
report="$root/build/plan-polygon-color-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_polygon_color_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_polygon_color.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanPolygonColorPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanPolygonColorPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan Polygon color assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan Polygon color mutation: %s\n' "$name"
}

model_file=internal/source/plan_polygon_color_prefix.go
reader_file=internal/top/plan_polygon_color.go
mutate discard-base "$model_file" 'points: points' 'points: PlanPolygonPointsPrefix{}'
mutate discard-color "$model_file" 'color: color' 'color: 0'
mutate mask-color "$model_file" 'color: color' 'color: color & 7'
mutate narrow-color "$model_file" 'color: color' 'color: color & 127'
mutate normalize-unknown-color "$model_file" 'color: color' 'color: func() byte { if color < 1 || color > 7 { return 6 }; return color }()'
mutate alias-input-bytes "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-bytes "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate discard-color-span "$model_file" 'span: span' 'span: Span{}'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate wrong-plan "$model_file" 'return p.points.PlanMapping()' 'return p.points.OverviewMapping()'
mutate wrong-overview "$model_file" 'return p.points.OverviewMapping()' 'return p.points.PlanMapping()'
mutate wrong-count "$model_file" 'return p.points.PointCountRaw()' 'return 0'
mutate discard-points "$model_file" 'return p.points.Points()' 'return nil'
mutate wrong-inherited-offsets "$model_file" 'PlanPolygonPointsPrefixOffsets: p.points.Offsets()' 'PlanPolygonPointsPrefixOffsets: PlanPolygonPointsPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)' 'ReadV3PlanPolygonPointsPrefixWithLimits(data, DefaultPolygonCountLimits())'
mutate bypass-point-preflight "$reader_file" 'ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)' 'func(data []byte, limits PolygonCountLimits) (source.PlanPolygonPointsPrefix, error) { c, e := ReadV3PlanPolygonCountPrefixWithLimits(data, limits); if e != nil { return source.PlanPolygonPointsPrefix{}, e }; return source.NewPlanPolygonPointsPrefix(c, nil, source.Span{}, data[:c.ConsumedOffset()], len(data)-c.ConsumedOffset()), nil }(data, limits)'
mutate wrong-start "$reader_file" 'offset: points.ConsumedOffset()' 'offset: 45'
mutate reread-point "$reader_file" 'offset: points.ConsumedOffset()' 'offset: points.ConsumedOffset() - 1'
mutate wrong-error-field "$reader_file" '"plan.elements[0].color"' '"plan.color"'
mutate read-two-bytes "$reader_file" 'r.take(1,' 'r.take(2,'
mutate require-next-marker "$reader_file" 'span := source.Span' 'if _, err := r.take(1, "plan.elements[1].kind"); err != nil { return empty, err }; span := source.Span'
mutate omit-zero-color "$reader_file" 'r := reader' 'if points.PointCountRaw() == 0 { return source.NewPlanPolygonColorPrefix(points, 0, source.Span{}, points.Bytes(), points.UnparsedTailSize()), nil }; r := reader'
mutate reject-unknown-color "$reader_file" 'span := source.Span' 'if b[0] < 1 || b[0] > 7 { return empty, failure("unsupported_color", "plan.elements[0].color", start) }; span := source.Span'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate mutate-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate wrong-color-byte "$reader_file" 'points, b[0], span' 'points, b[0] ^ data[start] ^ data[start-1], span'
mutate shift-color-span "$reader_file" 'Start: start, End: r.offset' 'Start: start-1, End: r.offset'
mutate shorten-color-span "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: start'
mutate wrong-error-offset "$reader_file" 'b, err := r.take(1, "plan.elements[0].color")' 'b, err := r.take(1, "plan.elements[0].color"); if e, ok := err.(*ParseError); ok { e.Offset++ }'
mutate wrong-error-code "$reader_file" 'b, err := r.take(1, "plan.elements[0].color")' 'b, err := r.take(1, "plan.elements[0].color"); if e, ok := err.(*ParseError); ok { e.Code = "resource_limit" }'
mutate partial-color-failure "$reader_file" 'b, err := r.take(1, "plan.elements[0].color")' 'b, err := r.take(1, "plan.elements[0].color"); if err != nil { return source.NewPlanPolygonColorPrefix(points, 0, source.Span{}, points.Bytes(), points.UnparsedTailSize()), err }'
mutate partial-inherited-failure "$reader_file" 'points, err := ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)' 'points, err := ReadV3PlanPolygonPointsPrefixWithLimits(data, limits); if err != nil { return source.NewPlanPolygonColorPrefix(points, 0, source.Span{}, nil, 1), err }'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 33 and all(.[]; .status == "KILLED")' "$report"
