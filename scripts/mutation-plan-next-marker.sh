#!/usr/bin/env bash
# Valid next plan marker, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_next_marker_prefix.go)
reader=$(cat internal/top/plan_next_marker.go)
report="$root/build/plan-next-marker-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_next_marker_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_next_marker.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanNextMarkerPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanNextMarkerPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan next marker assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan next marker mutation: %s\n' "$name"
}

model_file=internal/source/plan_next_marker_prefix.go
reader_file=internal/top/plan_next_marker.go
mutate discard-base "$model_file" 'color: color' 'color: PlanPolygonColorPrefix{}'
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
mutate discard-offsets "$model_file" 'PlanPolygonColorPrefixOffsets: p.color.Offsets()' 'PlanPolygonColorPrefixOffsets: PlanPolygonColorPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3PlanPolygonColorPrefixWithLimits(data, limits)' 'ReadV3PlanPolygonColorPrefixWithLimits(data, DefaultPolygonCountLimits())'
mutate bypass-color "$reader_file" 'ReadV3PlanPolygonColorPrefixWithLimits(data, limits)' 'func(data []byte, limits PolygonCountLimits) (source.PlanPolygonColorPrefix, error) { p, e := ReadV3PlanPolygonPointsPrefixWithLimits(data, limits); if e != nil { return source.PlanPolygonColorPrefix{}, e }; return source.NewPlanPolygonColorPrefix(p, 0, source.Span{}, p.Bytes(), p.UnparsedTailSize()), nil }(data, limits)'
mutate wrong-start "$reader_file" 'offset: color.ConsumedOffset()' 'offset: 46'
mutate reread-color "$reader_file" 'offset: color.ConsumedOffset()' 'offset: color.ConsumedOffset()-1'
mutate wrong-field "$reader_file" '"plan.elements[1].kind"' '"plan.elements[0].kind"'
mutate read-two "$reader_file" 'r.take(1,' 'r.take(2,'
mutate require-payload "$reader_file" 'span := source.Span' 'if _, err := r.take(1, "plan.elements[1].payload"); err != nil { return empty, err }; span := source.Span'
mutate zero-reads-side-mapping "$reader_file" 'span := source.Span' 'if b[0] == 0 { if _, err := r.take(12, "side.mapping"); err != nil { return empty, err } }; span := source.Span'
mutate reject-unknown "$reader_file" 'span := source.Span' 'if b[0] != 0 && b[0] != 1 && b[0] != 3 { return empty, failure("unsupported_element", "plan.elements[1].kind", start) }; span := source.Span'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate mutate-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate wrong-byte "$reader_file" 'color, b[0], span' 'color, b[0] ^ data[start] ^ data[start-1], span'
mutate shift-span "$reader_file" 'Start: start, End: r.offset' 'Start: start-1, End: r.offset'
mutate shorten-span "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: start'
mutate wrong-error-offset "$reader_file" 'b, err := r.take(1, "plan.elements[1].kind")' 'b, err := r.take(1, "plan.elements[1].kind"); if e, ok := err.(*ParseError); ok { e.Offset++ }'
mutate wrong-error-code "$reader_file" 'b, err := r.take(1, "plan.elements[1].kind")' 'b, err := r.take(1, "plan.elements[1].kind"); if e, ok := err.(*ParseError); ok { e.Code = "resource_limit" }'
mutate partial-marker-failure "$reader_file" 'b, err := r.take(1, "plan.elements[1].kind")' 'b, err := r.take(1, "plan.elements[1].kind"); if err != nil { return source.NewPlanNextMarkerPrefix(color, 0, source.Span{}, color.Bytes(), color.UnparsedTailSize()), err }'
mutate partial-inherited-failure "$reader_file" 'color, err := ReadV3PlanPolygonColorPrefixWithLimits(data, limits)' 'color, err := ReadV3PlanPolygonColorPrefixWithLimits(data, limits); if err != nil { return source.NewPlanNextMarkerPrefix(color, 0, source.Span{}, nil, 1), err }'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 35 and all(.[]; .status == "KILLED")' "$report"
