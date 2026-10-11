#!/usr/bin/env bash
# Valid side mapping raw-field, scale, span and copy faults in a disposable copy.
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
model=$(cat internal/source/side_mapping_prefix.go)
reader=$(cat internal/top/side_mapping.go)
report="$root/build/side-mapping-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/side_mapping_prefix.go
  printf '%s\n' "$reader" >internal/top/side_mapping.go
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
  go test -json -count=1 -timeout=30s -run '^TestSideMappingPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestSideMappingPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by an side mapping assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED side mapping mutation: %s\n' "$name"
}

model_file=internal/source/side_mapping_prefix.go
reader_file=internal/top/side_mapping.go
mutate discard-plan "$model_file" 'PlanElementsPrefix: plan' 'PlanElementsPrefix: PlanElementsPrefix{}'
mutate discard-side "$model_file" 'side: side' 'side: Mapping{}'
mutate discard-presence "$model_file" 'present: present' 'present: false'
mutate alias-input "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-side "$model_file" 'return p.side, p.present' 'return p.OverviewMapping(), p.present'
mutate hide-side-spans "$model_file" 'Side: p.side.Offsets()' 'Side: MappingOffsets{}'
mutate hide-plan-spans "$model_file" 'PlanElementsPrefixOffsets: p.PlanElementsPrefix.Offsets()' 'PlanElementsPrefixOffsets: PlanElementsPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3PlanElementsPrefixWithLimits(data, limits)' 'ReadV3PlanElementsPrefixWithLimits(data, DefaultPlanElementLimits())'
mutate default-zero-limits "$reader_file" 'DefaultPlanElementLimits())' 'PlanElementLimits{})'
mutate reverse-stop "$reader_file" '== source.PlanStopUnsupportedElement' '!= source.PlanStopUnsupportedElement'
mutate read-after-unsupported "$reader_file" 'if plan.StopReason() == source.PlanStopUnsupportedElement {' 'if false {'
mutate unsupported-presence "$reader_file" 'source.Mapping{}, false' 'source.Mapping{}, true'
mutate unsupported-consume-tail "$reader_file" 'data[:plan.ConsumedOffset()]' 'data'
mutate unsupported-hide-tail "$reader_file" 'len(data)-plan.ConsumedOffset()' '0'
mutate inherited-partial "$reader_file" 'if err != nil {
		return empty, err
	}' 'if err != nil {
		return source.NewSideMappingPrefix(plan, source.Mapping{}, true, nil, len(data)), err
	}'
mutate scalar-partial "$reader_file" 'if err != nil {
			return empty, err
		}' 'if err != nil {
			return source.NewSideMappingPrefix(plan, source.Mapping{}, true, data[:r.offset], len(data)-r.offset), err
		}'
mutate big-endian "$reader_file" 'binary.LittleEndian.Uint32(b)' 'binary.BigEndian.Uint32(b)'
mutate swap-origins "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[1], fields[0], fields[2], offsets'
mutate swap-scale "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[2], fields[1], fields[0], offsets'
mutate divide-scale-5 "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], fields[2]/5, offsets'
mutate divide-scale-10 "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], fields[2]/10, offsets'
mutate float-scale "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0], fields[1], int32(float32(fields[2])), offsets'
mutate normalize-negative "$reader_file" 'offsets := source.MappingOffsets{' 'if fields[2]<0 { fields[2] = -fields[2] }; offsets := source.MappingOffsets{'
mutate normalize-zero "$reader_file" 'offsets := source.MappingOffsets{' 'if fields[2]==0 { fields[2] = 500 }; offsets := source.MappingOffsets{'
mutate reject-nonpositive "$reader_file" 'offsets := source.MappingOffsets{' 'if fields[2]<=0 {return empty, failure("invalid_scale", "side.mapping.scale", start+8)}; offsets := source.MappingOffsets{'
mutate scale-origin "$reader_file" 'fields[0], fields[1], fields[2], offsets' 'fields[0]/1000, fields[1]/1000, fields[2], offsets'
mutate hardcoded-offset "$reader_file" 'offset: plan.ConsumedOffset()' 'offset: 41'
mutate read-plan-again "$reader_file" 'offset: plan.ConsumedOffset()' 'offset: plan.PlanMapping().Offsets().Record.Start'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate change-input "$reader_file" 'start := r.offset' 'data[0] = 0; start := r.offset'
mutate wrong-x-field "$reader_file" '"side.mapping.x0"' '"plan.mapping.x0"'
mutate wrong-y-field "$reader_file" '"side.mapping.y0"' '"plan.mapping.y0"'
mutate wrong-scale-field "$reader_file" '"side.mapping.scale"' '"plan.mapping.scale"'
mutate require-side-marker "$reader_file" 'mapping := source.NewMapping' 'if _,err:=r.take(1,"side.elements[0].kind");err!=nil{return empty,err}; mapping := source.NewMapping'
mutate omit-record-span "$reader_file" 'Record: source.Span{Start: start, End: r.offset},' 'Record: source.Span{},'
mutate omit-x0-span "$reader_file" 'X0:     source.Span{Start: start, End: start + 4},' 'X0: source.Span{},'
mutate omit-y0-span "$reader_file" 'Y0:     source.Span{Start: start + 4, End: start + 8},' 'Y0: source.Span{},'
mutate omit-scale-span "$reader_file" 'Scale:  source.Span{Start: start + 8, End: r.offset},' 'Scale: source.Span{},'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 41 and ([.[].name] | unique | length) == 41 and all(.[]; .status == "KILLED")' "$report"
