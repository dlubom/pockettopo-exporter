#!/usr/bin/env bash
# Valid plan mapping raw-field, scale, span and copy faults in a disposable copy.
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
model=$(cat internal/source/plan_mapping_prefix.go)
reader=$(cat internal/top/plan_mapping.go)
report="$root/build/plan-mapping-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_mapping_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_mapping.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanMappingPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanMappingPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by an plan mapping assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan mapping mutation: %s\n' "$name"
}

model_file=internal/source/plan_mapping_prefix.go
reader_file=internal/top/plan_mapping.go
mutate discard-overview "$model_file" 'overview: overview' 'overview: OverviewPrefix{}'
mutate discard-plan "$model_file" 'plan: plan' 'plan: Mapping{}'
mutate alias-input-prefix "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate big-endian-fields "$reader_file" 'binary.LittleEndian.Uint32(b)' 'binary.BigEndian.Uint32(b)'
mutate swap-origins "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[1], fields[0], fields[2], offsets'
mutate swap-origin-scale "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[2], fields[1], fields[0], offsets'
mutate divide-scale-5 "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], fields[2] / 5, offsets'
mutate divide-scale-10 "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], fields[2] / 10, offsets'
mutate multiply-scale "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], fields[2] * 5, offsets'
mutate float32-scale "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], int32(float32(fields[2])), offsets'
mutate normalize-negative-scale "$reader_file" 'offsets := source.MappingOffsets{' 'if fields[2] < 0 { fields[2] = -fields[2] }; offsets := source.MappingOffsets{'
mutate normalize-zero-scale "$reader_file" 'offsets := source.MappingOffsets{' 'if fields[2] == 0 { fields[2] = 500 }; offsets := source.MappingOffsets{'
mutate reject-nonpositive-scale "$reader_file" 'offsets := source.MappingOffsets{' 'if fields[2] <= 0 { return empty, failure("invalid_scale", "plan.mapping.scale", start + 8) }; offsets := source.MappingOffsets{'
mutate scale-x0 "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0] / 1000, fields[1], fields[2], offsets'
mutate scale-y0 "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1] / 1000, fields[2], offsets'
mutate hardcoded-position "$reader_file" 'offset: overview.ConsumedOffset()' 'offset: 28'
mutate omit-record-span "$reader_file" 'Record: source.Span{Start: start, End: r.offset},' 'Record: source.Span{},'
mutate omit-x0-span "$reader_file" 'X0:     source.Span{Start: start, End: start + 4},' 'X0: source.Span{},'
mutate omit-y0-span "$reader_file" 'Y0:     source.Span{Start: start + 4, End: start + 8},' 'Y0: source.Span{},'
mutate omit-scale-span "$reader_file" 'Scale:  source.Span{Start: start + 8, End: r.offset},' 'Scale: source.Span{},'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate change-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate expose-overview-as-plan "$model_file" 'return p.plan' 'return p.overview.OverviewMapping()'
mutate expose-plan-as-overview "$model_file" 'return p.overview.OverviewMapping()' 'return p.plan'
mutate confuse-mapping-offsets "$model_file" 'Plan: p.plan.Offsets()' 'Plan: p.overview.OverviewMapping().Offsets()'
mutate read-overview-again "$reader_file" 'offset: overview.ConsumedOffset()' 'offset: overview.ConsumedOffset() - 12'
mutate replace-caller-limits "$reader_file" 'ReadV3OverviewPrefixWithLimits(data, limits)' 'ReadV3OverviewPrefixWithLimits(data, DefaultReferenceLimits())'
mutate wrong-field-name "$reader_file" '"plan.mapping.x0"' '"overview.x0"'
mutate require-marker "$reader_file" 'mapping := source.NewMapping' 'if _, err := r.take(1, "plan.marker"); err != nil { return empty, err }; mapping := source.NewMapping'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 32 and all(.[]; .status == "KILLED")' "$report"
