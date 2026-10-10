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
model=$(cat internal/source/plan_polygons_prefix.go)
reader=$(cat internal/top/plan_polygons.go)
report="$root/build/plan-polygons-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/plan_polygons_prefix.go
  printf '%s\n' "$reader" >internal/top/plan_polygons.go
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
  go test -json -count=1 -timeout=30s -run '^TestPlanPolygons' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestPlanPolygons"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a plan Polygon loop assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED plan Polygon loop mutation: %s\n' "$name"
}

model_file=internal/source/plan_polygons_prefix.go
reader_file=internal/top/plan_polygons.go
mutate discard-base "$model_file" 'PlanMappingPrefix: base' 'PlanMappingPrefix: PlanMappingPrefix{}'
mutate discard-marker "$model_file" 'marker: marker' 'marker: 0'
mutate discard-count "$model_file" 'count: count' 'count: 0'
mutate discard-color "$model_file" 'color: color' 'color: 0'
mutate mask-color "$model_file" 'color: color' 'color: color & 7'
mutate alias-points-input "$model_file" 'points: slices.Clone(points)' 'points: points'
mutate alias-points-output "$model_file" 'return slices.Clone(p.points)' 'return p.points'
mutate alias-polygons-input "$model_file" 'polygons: slices.Clone(polygons)' 'polygons: polygons'
mutate alias-polygons-output "$model_file" 'return slices.Clone(p.polygons)' 'return p.polygons'
mutate alias-bytes-input "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-bytes-output "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate discard-offsets "$model_file" 'offsets: offsets' 'offsets: PlanPolygonOffsets{}'
mutate discard-stop "$model_file" 'stopMarker: marker' 'stopMarker: 0'
mutate mask-stop "$model_file" 'stopMarker: marker' 'stopMarker: marker & 3'
mutate discard-stop-span "$model_file" 'stopSpan: span' 'stopSpan: Span{}'
mutate wrong-reason "$model_file" 'reason: reason' 'reason: PlanStopTerminated'
mutate hide-total "$model_file" 'totalPoints: totalPoints' 'totalPoints: 0'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate wrong-offset-base "$model_file" 'PlanMappingPrefixOffsets: p.PlanMappingPrefix.Offsets()' 'PlanMappingPrefixOffsets: PlanMappingPrefixOffsets{}'
mutate replace-limits "$reader_file" 'ReadV3PlanMappingPrefixWithLimits(data, limits.ReferenceLimits)' 'ReadV3PlanMappingPrefixWithLimits(data, DefaultReferenceLimits())'
mutate invalid-lower "$reader_file" 'bound.value < 0' 'bound.value < -1'
mutate invalid-upper "$reader_file" 'bound.value > bound.maximum' 'bound.value > bound.maximum+1'
mutate wrong-start "$reader_file" 'offset: base.ConsumedOffset()' 'offset: 40'
mutate wrong-index "$reader_file" 'len(polygons))' '0)'
mutate read-two-markers "$reader_file" 'r.take(1, kindField)' 'r.take(2, kindField)'
mutate no-terminator "$reader_file" 'marker[0] != 1' 'marker[0] != 1 && marker[0] != 0'
mutate reject-unsupported "$reader_file" 'reason := source.PlanStopUnsupportedElement' 'return empty, failure("unsupported_element", kindField, start); reason := source.PlanStopUnsupportedElement'
mutate wrong-zero-reason "$reader_file" 'marker[0] == 0' 'marker[0] == 3'
mutate require-stop-payload "$reader_file" 'reason := source.PlanStopUnsupportedElement' 'if _, e := r.take(1, field+".payload"); e != nil {return empty,e}; reason := source.PlanStopUnsupportedElement'
mutate element-exclusive "$reader_file" 'len(polygons) >= limits.MaxElements' 'len(polygons) >= limits.MaxElements-1'
mutate element-over-limit "$reader_file" 'len(polygons) >= limits.MaxElements' 'len(polygons) > limits.MaxElements'
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
mutate wrong-count-endian "$reader_file" 'binary.LittleEndian.Uint32(b)' 'binary.BigEndian.Uint32(b)'
mutate wrong-record-start "$reader_file" 'Record: source.Span{Start: start, End: r.offset}' 'Record: source.Span{Start: countStart, End: r.offset}'
mutate wrong-count-span "$reader_file" 'Start: countStart, End: pointsStart' 'Start: countStart+1, End: pointsStart'
mutate wrong-point-span "$reader_file" 'Start: pointsStart, End: colorStart' 'Start: pointsStart+1, End: colorStart'
mutate wrong-color-span "$reader_file" 'Start: colorStart, End: r.offset' 'Start: colorStart-1, End: r.offset'
mutate consume-tail "$reader_file" 'data[:r.offset]' 'data'
mutate wrong-error-code "$reader_file" 'failure("negative_count",' 'failure("truncated",'
mutate partial-failure "$reader_file" 'var empty source.PlanPolygonsPrefix' 'empty := source.NewPlanPolygonsPrefix(source.PlanMappingPrefix{},nil,3,source.Span{},source.PlanStopUnsupportedElement,0,nil,1)'
mutate stop-after-three "$reader_file" 'if marker[0] != 1 {' 'if marker[0] != 1 || len(polygons)==3 {'
mutate reverse-order "$reader_file" 'total += n' 'if len(polygons)>1 {polygons[0],polygons[len(polygons)-1]=polygons[len(polygons)-1],polygons[0]}; total += n'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 58 and (map(.name)|unique|length)==58 and all(.[]; .status == "KILLED")' "$report"
