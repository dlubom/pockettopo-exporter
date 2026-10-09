#!/usr/bin/env bash
# Valid raw-field, presence, copy and native-acceptance faults in a source copy.
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
model=$(cat internal/source/measurement_prefix.go)
reader=$(cat internal/top/measurements.go)
report="$root/build/measurement-mutation.json"
rm -f "$report"
cd "$trial"
if ! go test -count=1 ./internal/source ./internal/top >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi

mutate() {
  local name="$1" file="$2" before="$3" after="$4" status=0 original remaining changed
  printf '%s\n' "$model" >internal/source/measurement_prefix.go
  printf '%s\n' "$reader" >internal/top/measurements.go
  original=$(cat "$file")
  if [[ "$original" != *"$before"* ]]; then
    printf 'Missing mutation target: %s\n' "$name" >&2
    exit 1
  fi
  # Quote the replacement pattern: field subscripts must be literal, not globs.
  remaining=${original/"$before"/}
  if [[ "$remaining" == *"$before"* ]]; then
    printf 'Ambiguous mutation target: %s\n' "$name" >&2
    exit 1
  fi
  changed=${original/"$before"/"$after"}
  printf '%s\n' "$changed" >"$file"
  go build ./internal/source ./internal/top
  go test -json -count=1 -timeout=30s -run '^TestMeasurementPrefix' \
    ./internal/source ./internal/top >"$name.json" 2>&1 || status=$?
  if [[ "$status" != 1 ]] || ! jq -e -s '
    any(.[]; .Action == "fail" and (.Test // "" | test("^TestMeasurementPrefix"))) and
    all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
  ' "$name.json" >/dev/null; then
    cat "$name.json" >&2
    printf 'Mutation was not killed by a measurement assertion: %s\n' "$name" >&2
    exit 1
  fi
  jq -n --arg name "$name" '{name: $name, status: "KILLED"}' >>mutants.jsonl
  printf 'KILLED measurement mutation: %s\n' "$name"
}

model_file=internal/source/measurement_prefix.go
reader_file=internal/top/measurements.go
mutate discard-from "$model_file" 'from: from' 'from: NewStationID(0)'
mutate discard-to "$model_file" 'to: to' 'to: NewStationID(0)'
mutate discard-distance "$model_file" 'distance: distance' 'distance: 0'
mutate discard-azimuth "$model_file" 'azimuth: azimuth' 'azimuth: 0'
mutate discard-inclination "$model_file" 'inclination: inclination' 'inclination: 0'
mutate mask-flags "$model_file" 'flags: flags' 'flags: flags & 3'
mutate discard-roll "$model_file" 'roll: roll' 'roll: 0'
mutate normalize-trip-index "$model_file" 'tripIndex: tripIndex' 'tripIndex: 0'
mutate discard-comment "$model_file" 'comment: comment' 'comment: ""'
mutate discard-offsets "$model_file" 'offsets: offsets' 'offsets: MeasurementOffsets{}'
mutate hide-comment-bytes "$model_file" 'return []byte(m.comment)' 'return nil'
mutate infer-comment-presence "$model_file" 'return m.flags&2 != 0' 'return m.comment != ""'
mutate discard-count "$model_file" 'count: count' 'count: 0'
mutate discard-trips "$model_file" 'trips: trips' 'trips: TripPrefix{}'
mutate discard-measurements "$model_file" 'measurements: append([]Measurement(nil), measurements...)' 'measurements: nil'
mutate alias-input-measurements "$model_file" 'measurements: append([]Measurement(nil), measurements...)' 'measurements: measurements'
mutate alias-output-measurements "$model_file" 'return append([]Measurement(nil), p.measurements...)' 'return p.measurements'
mutate alias-input-prefix "$model_file" 'bytes.Clone(prefix)' 'prefix'
mutate alias-output-prefix "$model_file" 'return bytes.Clone(p.prefix)' 'return p.prefix'
mutate hide-tail "$model_file" 'return p.tailBytes' 'return 0'
mutate hide-consumed "$model_file" 'return len(p.prefix)' 'return 0'
mutate discard-count-offset "$model_file" 'countOffset: countOffset' 'countOffset: 0'
mutate big-endian-count "$reader_file" 'binary.LittleEndian.Uint32(countBytes)' 'binary.BigEndian.Uint32(countBytes)'
mutate big-endian-from "$reader_file" 'binary.LittleEndian.Uint32(fields[0])' 'binary.BigEndian.Uint32(fields[0])'
mutate big-endian-to "$reader_file" 'binary.LittleEndian.Uint32(fields[1])' 'binary.BigEndian.Uint32(fields[1])'
mutate big-endian-distance "$reader_file" 'binary.LittleEndian.Uint32(fields[2])' 'binary.BigEndian.Uint32(fields[2])'
mutate big-endian-azimuth "$reader_file" 'binary.LittleEndian.Uint16(fields[3])' 'binary.BigEndian.Uint16(fields[3])'
mutate big-endian-inclination "$reader_file" 'binary.LittleEndian.Uint16(fields[4])' 'binary.BigEndian.Uint16(fields[4])'
mutate big-endian-trip-index "$reader_file" 'binary.LittleEndian.Uint16(fields[7])' 'binary.BigEndian.Uint16(fields[7])'
mutate swap-flags-roll "$reader_file" 'flags, roll := fields[5][0], fields[6][0]' 'flags, roll := fields[6][0], fields[5][0]'
mutate wrong-comment-bit "$reader_file" 'if flags&2 != 0' 'if flags&1 != 0'
mutate reject-negative-distance "$reader_file" 'azimuth := int16' 'if distance < 0 { return empty, failure("out_of_range", field, start) }; azimuth := int16'
mutate reject-extra-flags "$reader_file" 'tripIndex := int16' 'if flags & ^byte(3) != 0 { return empty, failure("unknown_flags", field, start) }; tripIndex := int16'
mutate reject-unassigned-index "$reader_file" 'measurements = append(measurements, measurement)' 'if measurement.TripIndexRaw() < -1 || int32(measurement.TripIndexRaw()) >= trips.TripCountRaw() { return empty, failure("out_of_range", "trip_index", r.offset) }; measurements = append(measurements, measurement)'
mutate rewrite-negative-index "$reader_file" 'offsets := source.MeasurementOffsets{' 'if tripIndex < 0 { tripIndex = -1 }; offsets := source.MeasurementOffsets{'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 35 and all(.[]; .status == "KILLED")' "$report"
