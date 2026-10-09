#!/usr/bin/env bash
# Valid first plan marker, span, copy and overread faults in a disposable copy.
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
model=$(cat internal/source/plan_marker_prefix.go)
reader=$(cat internal/top/plan_marker.go)
report="$root/build/plan-marker-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_marker_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_marker.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanMarkerPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanMarkerPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan marker assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan marker mutation: %s\n' "$name"
}

model_file=internal/source/plan_marker_prefix.go
reader_file=internal/top/plan_marker.go
mutate discard-plan-prefix "$model_file" 'plan: plan' 'plan: PlanMappingPrefix{}'
mutate discard-marker "$model_file" 'marker: marker' 'marker: 0'
mutate mask-marker-high-bit "$model_file" 'marker: marker' 'marker: marker & 127'
mutate discard-marker-span "$model_file" 'span: span' 'span: Span{}'
mutate alias-input-prefix "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate confuse-marker-span "$model_file" 'Marker: p.span' 'Marker: p.plan.PlanMapping().Offsets().Scale'
mutate expose-overview-as-plan "$model_file" 'return p.plan.PlanMapping()' 'return p.plan.OverviewMapping()'
mutate expose-plan-as-overview "$model_file" 'return p.plan.OverviewMapping()' 'return p.plan.PlanMapping()'
mutate hardcoded-position "$reader_file" 'offset: plan.ConsumedOffset()' 'offset: 40'
mutate reread-scale-byte "$reader_file" 'offset: plan.ConsumedOffset()' 'offset: plan.ConsumedOffset() - 1'
mutate replace-caller-limits "$reader_file" 'ReadV3PlanMappingPrefixWithLimits(data, limits)' 'ReadV3PlanMappingPrefixWithLimits(data, DefaultReferenceLimits())'
mutate wrong-field-name "$reader_file" '"plan.elements[0].kind"' '"plan.marker"'
mutate shift-marker-start "$reader_file" 'Start: start, End: r.offset' 'Start: start + 1, End: r.offset'
mutate shift-marker-end "$reader_file" 'Start: start, End: r.offset' 'Start: start, End: r.offset + 1'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate change-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate read-two-bytes "$reader_file" 'r.take(1,' 'r.take(2,'
mutate require-next-marker "$reader_file" 'span := source.Span' 'if _, err := r.take(1, "plan.elements[1].kind"); err != nil { return empty, err }; span := source.Span'
mutate require-polygon-payload "$reader_file" 'span := source.Span' 'if b[0] == 1 { if _, err := r.take(4, "plan.elements[0].count"); err != nil { return empty, err } }; span := source.Span'
mutate reject-unknown-marker "$reader_file" 'span := source.Span' 'if b[0] != 0 && b[0] != 1 && b[0] != 3 { return empty, failure("unknown_element", "plan.elements[0].kind", start) }; span := source.Span'
mutate omit-terminator-byte "$reader_file" 'span := source.Span' 'if b[0] == 0 { r.offset-- }; span := source.Span'
mutate partial-failure "$reader_file" 'b, err := r.take' 'empty = source.NewPlanMarkerPrefix(plan, 0, source.Span{}, data[:start], len(data)-start); b, err := r.take'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 25 and all(.[]; .status == "KILLED")' "$report"
