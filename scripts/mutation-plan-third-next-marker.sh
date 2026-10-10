#!/usr/bin/env bash
# Valid following plan marker, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_third_next_marker_prefix.go)
reader=$(cat internal/top/plan_third_next_marker.go)
report="$root/build/plan-third-next-marker-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_third_next_marker_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_third_next_marker.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanThirdNextMarkerPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanThirdNextMarkerPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan following marker assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan following marker mutation: %s\n' "$name"
}

model_file=internal/source/plan_third_next_marker_prefix.go
reader_file=internal/top/plan_third_next_marker.go
mutate discard-base "$model_file" 'color: color' 'color: PlanThirdPolygonColorPrefix{}'
mutate discard-marker "$model_file" 'marker: marker' 'marker: 0'
mutate mask-marker "$model_file" 'marker: marker' 'marker: marker & 3'
mutate narrow-marker "$model_file" 'marker: marker' 'marker: marker & 127'
mutate normalize-marker "$model_file" 'marker: marker' 'marker: func() byte { if marker != 0 && marker != 1 && marker != 3 { return 1 }; return marker }()'
mutate alias-input "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate discard-span "$model_file" 'span: span' 'span: Span{}'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate wrong-plan "$model_file" 'return p.color.PlanMapping()' 'return p.color.OverviewMapping()'
mutate wrong-overview "$model_file" 'return p.color.OverviewMapping()' 'return p.color.PlanMapping()'
mutate wrong-color "$model_file" 'return p.color.ColorRaw()' 'return p.marker'
mutate wrong-first-marker "$model_file" 'return p.color.MarkerRaw()' 'return p.marker'
mutate wrong-count "$model_file" 'return p.color.PointCountRaw()' 'return 0'
mutate discard-points "$model_file" 'return p.color.Points()' 'return nil'
mutate discard-offsets "$model_file" 'PlanThirdPolygonColorPrefixOffsets: p.color.Offsets()' 'PlanThirdPolygonColorPrefixOffsets: PlanThirdPolygonColorPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)' 'ReadV3PlanThirdPolygonColorPrefixWithLimits(data, DefaultPolygonCountLimits())'
mutate bypass-color "$reader_file" 'ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)' 'func(data []byte, limits PolygonCountLimits) (source.PlanThirdPolygonColorPrefix, error) { p, e := ReadV3PlanThirdPolygonPointsPrefixWithLimits(data, limits); if e != nil { return source.PlanThirdPolygonColorPrefix{}, e }; return source.NewPlanThirdPolygonColorPrefix(p, 0, source.Span{}, p.Bytes(), p.UnparsedTailSize()), nil }(data, limits)'
mutate wrong-start "$reader_file" 'offset: color.ConsumedOffset()' 'offset: 58'
mutate reread-color "$reader_file" 'offset: color.ConsumedOffset()' 'offset: color.ConsumedOffset()-1'
mutate wrong-field "$reader_file" '"plan.elements[3].kind"' '"plan.elements[0].kind"'
mutate read-two "$reader_file" 'r.take(1,' 'r.take(2,'
mutate require-payload "$reader_file" 'span := source.Span' 'if _, err := r.take(1, "plan.elements[3].payload"); err != nil { return empty, err }; span := source.Span'
mutate zero-reads-side-mapping "$reader_file" 'span := source.Span' 'if b[0] == 0 { if _, err := r.take(12, "side.mapping"); err != nil { return empty, err } }; span := source.Span'
mutate reject-unknown "$reader_file" 'span := source.Span' 'if b[0] != 0 && b[0] != 1 && b[0] != 3 { return empty, failure("unsupported_element", "plan.elements[3].kind", start) }; span := source.Span'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate mutate-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate wrong-byte "$reader_file" 'color, b[0], span' 'color, b[0] ^ data[start] ^ data[start-1], span'
mutate shift-span "$reader_file" 'Start: start, End: r.offset' 'Start: start-1, End: r.offset'
mutate shorten-span "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: start'
mutate wrong-error-offset "$reader_file" 'b, err := r.take(1, "plan.elements[3].kind")' 'b, err := r.take(1, "plan.elements[3].kind"); if e, ok := err.(*ParseError); ok { e.Offset++ }'
mutate wrong-error-code "$reader_file" 'b, err := r.take(1, "plan.elements[3].kind")' 'b, err := r.take(1, "plan.elements[3].kind"); if e, ok := err.(*ParseError); ok { e.Code = "resource_limit" }'
mutate partial-marker-failure "$reader_file" 'b, err := r.take(1, "plan.elements[3].kind")' 'b, err := r.take(1, "plan.elements[3].kind"); if err != nil { return source.NewPlanThirdNextMarkerPrefix(color, 0, source.Span{}, color.Bytes(), color.UnparsedTailSize()), err }'
mutate partial-inherited-failure "$reader_file" 'color, err := ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)' 'color, err := ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits); if err != nil { return source.NewPlanThirdNextMarkerPrefix(color, 0, source.Span{}, nil, 1), err }'
mutate discard-second-points "$model_file" 'return p.color.SecondPoints()' 'return nil'
mutate confuse-second-points "$model_file" 'return p.color.SecondPoints()' 'return p.color.Points()'
mutate confuse-first-points "$model_file" 'return p.color.Points()' 'return p.color.SecondPoints()'
mutate discard-second-count "$model_file" 'return p.color.SecondPointCountRaw()' 'return 0'
mutate confuse-second-count "$model_file" 'return p.color.SecondPointCountRaw()' 'return p.color.PointCountRaw()'
mutate confuse-first-count "$model_file" 'return p.color.PointCountRaw()' 'return p.color.SecondPointCountRaw()'
mutate discard-next-marker "$model_file" 'return p.color.NextMarkerRaw()' 'return 0'
mutate confuse-next-marker "$model_file" 'return p.color.NextMarkerRaw()' 'return p.marker'
mutate confuse-second-color "$model_file" 'return p.color.SecondColorRaw()' 'return p.marker'
mutate confuse-colors "$model_file" 'return p.color.SecondColorRaw()' 'return p.color.ColorRaw()'
mutate confuse-following-span "$model_file" 'ThirdNextMarker: p.span' 'ThirdNextMarker: p.color.Offsets().FollowingMarker'
mutate bypass-point-preflight "$reader_file" 'ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)' 'func(data []byte, limits PolygonCountLimits) (source.PlanThirdPolygonColorPrefix, error) { c, e := ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits); if e != nil { return source.PlanThirdPolygonColorPrefix{}, e }; p := source.NewPlanThirdPolygonPointsPrefix(c, nil, source.Span{}, c.Bytes(), c.UnparsedTailSize()); return source.NewPlanThirdPolygonColorPrefix(p, 0, source.Span{}, p.Bytes(), p.UnparsedTailSize()), nil }(data, limits)'
mutate discard-third-points "$model_file" 'return p.color.ThirdPoints()' 'return nil'
mutate confuse-third-points "$model_file" 'return p.color.ThirdPoints()' 'return p.color.SecondPoints()'
mutate discard-third-count "$model_file" 'return p.color.ThirdPointCountRaw()' 'return 0'
mutate confuse-third-count "$model_file" 'return p.color.ThirdPointCountRaw()' 'return p.color.SecondPointCountRaw()'
mutate discard-third-color "$model_file" 'return p.color.ThirdColorRaw()' 'return 0'
mutate confuse-third-color "$model_file" 'return p.color.ThirdColorRaw()' 'return p.marker'
mutate discard-following-marker "$model_file" 'return p.color.FollowingMarkerRaw()' 'return 0'
mutate confuse-following-marker "$model_file" 'return p.color.FollowingMarkerRaw()' 'return p.marker'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 55 and all(.[]; .status == "KILLED")' "$report"
