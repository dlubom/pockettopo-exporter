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
model=$(cat internal/source/plan_elements_prefix.go)
reader=$(cat internal/top/plan_elements.go)
report="$root/build/plan-elements-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_elements_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_elements.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanElements' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanElements"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan element loop assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan element loop mutation: %s\n' "$name"
}

model_file=internal/source/plan_elements_prefix.go
reader_file=internal/top/plan_elements.go
mutate discard-base "$model_file" 'PlanMappingPrefix: base' 'PlanMappingPrefix: PlanMappingPrefix{}'
mutate discard-xs-marker "$model_file" 'marker: marker' 'marker: 0'
mutate discard-xs-x "$model_file" 'x: x' 'x: 0'
mutate discard-xs-y "$model_file" 'y: y' 'y: 0'
mutate discard-xs-id "$model_file" 'station: station' 'station: StationID{}'
mutate discard-xs-direction "$model_file" 'direction: direction' 'direction: 0'
mutate mask-xs-direction "$model_file" 'direction: direction' 'direction: direction & 65535'
mutate discard-xs-spans "$model_file" 'offsets: offsets' 'offsets: PlanXSectionOffsets{}'
mutate polygon-variant "$model_file" 'kind: 1' 'kind: 3'
mutate xs-variant "$model_file" 'kind: 3' 'kind: 1'
mutate discard-polygon "$model_file" 'polygon: p' 'polygon: PlanPolygon{}'
mutate discard-xsection "$model_file" 'xsection: x' 'xsection: PlanXSection{}'
mutate alias-elements-in "$model_file" 'elements: slices.Clone(elements)' 'elements: elements'
mutate alias-elements-out "$model_file" 'return slices.Clone(p.elements)' 'return p.elements'
mutate alias-bytes-in "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-bytes-out "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate discard-stop "$model_file" 'stopMarker: marker' 'stopMarker: 0'
mutate mask-stop "$model_file" 'stopMarker: marker' 'stopMarker: marker & 3'
mutate discard-stop-span "$model_file" 'stopSpan: span' 'stopSpan: Span{}'
mutate discard-reason "$model_file" 'reason: reason' 'reason: PlanStopTerminated'
mutate discard-total "$model_file" 'totalPoints: totalPoints' 'totalPoints: 0'
mutate discard-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate discard-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate discard-base-spans "$model_file" 'PlanMappingPrefixOffsets: p.PlanMappingPrefix.Offsets()' 'PlanMappingPrefixOffsets: PlanMappingPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3PlanMappingPrefixWithLimits(data, limits.ReferenceLimits)' 'ReadV3PlanMappingPrefixWithLimits(data, DefaultReferenceLimits())'
mutate invalid-lower "$reader_file" 'bound.value < 0' 'bound.value < -1'
mutate invalid-upper "$reader_file" 'bound.value > bound.maximum' 'bound.value > bound.maximum+1'
mutate wrong-start "$reader_file" 'offset: base.ConsumedOffset()' 'offset: 40'
mutate wrong-index "$reader_file" 'len(elements))' '0)'
mutate read-two-markers "$reader_file" 'r.take(1, kindField)' 'r.take(2, kindField)'
mutate reject-unsupported "$reader_file" 'reason := source.PlanStopUnsupportedElement' 'return empty, failure("unsupported_element", kindField, start); reason := source.PlanStopUnsupportedElement'
mutate wrong-zero-reason "$reader_file" 'marker[0] == 0' 'marker[0] == 3'
mutate require-stop-payload "$reader_file" 'reason := source.PlanStopUnsupportedElement' 'if _, e := r.take(1, field+".payload"); e != nil {return empty,e}; reason := source.PlanStopUnsupportedElement'
mutate element-exclusive "$reader_file" 'len(elements) >= limits.MaxElements' 'len(elements) >= limits.MaxElements-1'
mutate element-over-limit "$reader_file" 'len(elements) >= limits.MaxElements' 'len(elements) > limits.MaxElements'
mutate negative-ignored "$reader_file" 'count < 0' 'count < -1'
mutate points-exclusive "$reader_file" 'n > limits.MaxPoints' 'n >= limits.MaxPoints'
mutate points-bypass "$reader_file" 'n > limits.MaxPoints' 'false'
mutate total-exclusive "$reader_file" 'n > limits.MaxTotalPoints-total' 'n >= limits.MaxTotalPoints-total'
mutate total-per-polygon "$reader_file" 'n > limits.MaxTotalPoints-total' 'n > limits.MaxTotalPoints'
mutate total-bypass "$reader_file" 'n > limits.MaxTotalPoints-total' 'false'
mutate reset-total "$reader_file" 'total += n' 'total = n'
mutate double-total "$reader_file" 'total += n' 'total += n*2'
mutate preflight-exclusive "$reader_file" 'n > (len(data)-r.offset)/8' 'n >= (len(data)-r.offset)/8'
mutate preflight-short "$reader_file" 'n > (len(data)-r.offset)/8' 'n > (len(data)-r.offset)/4'
mutate wrong-x-endian "$reader_file" 'binary.LittleEndian.Uint32(data[offset : offset+4])' 'binary.BigEndian.Uint32(data[offset : offset+4])'
mutate wrong-y-endian "$reader_file" 'binary.LittleEndian.Uint32(data[offset+4 : offset+8])' 'binary.BigEndian.Uint32(data[offset+4 : offset+8])'
mutate swap-xy "$reader_file" 'source.NewPolygonPoint(x, y,' 'source.NewPolygonPoint(y, x,'
mutate point-shift "$reader_file" 'Start: offset, End: offset + 8' 'Start: offset+1, End: offset + 8'
mutate omit-zero-color "$reader_file" 'color, err := r.take(1, colorField)' 'color, err := func() ([]byte,error) {if n==0 {return []byte{0},nil};return r.take(1,colorField)}()'
mutate read-two-colors "$reader_file" 'r.take(1, colorField)' 'r.take(2, colorField)'
mutate wrong-count-endian "$reader_file" 'count := int32(binary.LittleEndian.Uint32(b))' 'count := int32(binary.BigEndian.Uint32(b))'
mutate wrong-record-start "$reader_file" 'Record: source.Span{Start: start, End: r.offset}' 'Record: source.Span{Start: countStart, End: r.offset}'
mutate wrong-count-span "$reader_file" 'Start: countStart, End: pointsStart' 'Start: countStart+1, End: pointsStart'
mutate wrong-point-span "$reader_file" 'Start: pointsStart, End: colorStart' 'Start: pointsStart+1, End: colorStart'
mutate wrong-color-span "$reader_file" 'Start: colorStart, End: r.offset' 'Start: colorStart-1, End: r.offset'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate wrong-error-code "$reader_file" 'failure("negative_count",' 'failure("truncated",'
mutate partial-failure "$reader_file" 'var empty source.PlanElementsPrefix' 'empty := source.NewPlanElementsPrefix(source.PlanMappingPrefix{},nil,3,source.Span{},source.PlanStopUnsupportedElement,0,nil,1)'
mutate stop-at-xs "$reader_file" 'marker[0] != 1 && marker[0] != 3' 'marker[0] != 1'
mutate no-terminator "$reader_file" 'marker[0] != 1 && marker[0] != 3' 'marker[0] != 1 && marker[0] != 3 && marker[0] != 0'
mutate unknown-as-xs "$reader_file" 'marker[0] != 1 && marker[0] != 3' 'marker[0] != 1 && marker[0] != 3 && marker[0] != 2'
mutate overread-xs-field "$reader_file" 'r.take(4, fmt.Sprintf("%s.%s", field, name))' 'r.take(5, fmt.Sprintf("%s.%s", field, name))'
mutate xs-endian "$reader_file" 'raw[i] = binary.LittleEndian.Uint32(b)' 'raw[i] = binary.BigEndian.Uint32(b)'
mutate swap-xs-xy "$reader_file" 'int32(raw[0]), int32(raw[1])' 'int32(raw[1]), int32(raw[0])'
mutate wrong-id "$reader_file" 'source.NewStationID(raw[2])' 'source.NewStationID(raw[1])'
mutate wrong-direction "$reader_file" 'int32(raw[3])' 'int32(raw[2])'
mutate mask-id "$reader_file" 'source.NewStationID(raw[2])' 'source.NewStationID(raw[2] & 0x7fffffff)'
mutate mask-direction "$reader_file" 'int32(raw[3])' 'int32(raw[3] & 65535)'
mutate xs-point-budget "$reader_file" 'continue' 'total++; continue'
mutate xs-record "$reader_file" 'Record:    source.Span{Start: start, End: r.offset},' 'Record:    source.Span{Start: start + 1, End: r.offset},'
mutate xs-marker "$reader_file" 'Marker:    source.Span{Start: start, End: start + 1},' 'Marker:    source.Span{Start: start + 1, End: start + 1},'
mutate xs-x-span "$reader_file" 'CenterX:   source.Span{Start: start + 1, End: start + 5},' 'CenterX:   source.Span{Start: start + 1 + 1, End: start + 5},'
mutate xs-y-span "$reader_file" 'CenterY:   source.Span{Start: start + 5, End: start + 9},' 'CenterY:   source.Span{Start: start + 1 + 5, End: start + 9},'
mutate xs-id-span "$reader_file" 'StationID: source.Span{Start: start + 9, End: start + 13},' 'StationID: source.Span{Start: start + 1 + 9, End: start + 13},'
mutate xs-dir-span "$reader_file" 'Direction: source.Span{Start: start + 13, End: start + 17},' 'Direction: source.Span{Start: start + 1 + 13, End: start + 17},'
mutate stop-after-three "$reader_file" 'if marker[0] != 1 && marker[0] != 3 {' 'if marker[0] != 1 && marker[0] != 3 || len(elements)==3 {'
mutate reverse-order "$reader_file" 'field := fmt.Sprintf("plan.elements[%d]", len(elements))' 'if len(elements)>1 {elements[0],elements[len(elements)-1]=elements[len(elements)-1],elements[0]}; field := fmt.Sprintf("plan.elements[%d]", len(elements))'
mutate validation-order "$reader_file" '{"limits.max_points", limits.MaxPoints, defaults.MaxPoints},
		{"limits.max_elements", limits.MaxElements, defaults.MaxElements},' '{"limits.max_elements", limits.MaxElements, defaults.MaxElements},
		{"limits.max_points", limits.MaxPoints, defaults.MaxPoints},'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 79 and (map(.name)|unique|length)==79 and all(.[]; .status == "KILLED")' "$report"
