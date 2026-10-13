#!/usr/bin/env bash
# P15 Walls conversion/provenance/CLI faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
preflight=false
if [[ ${1:-} == --preflight && $# == 1 ]]; then preflight=true; elif [[ $# != 0 ]]; then exit 2; fi
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/exportwalls/walls.go internal/cli/walls.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/walls-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exportwalls ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
seen_names=""
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original remaining changed status=0
 if [[ $seen_names == *"|$name|"* ]]; then printf 'Duplicate mutation name: %s\n' "$name" >&2; exit 1; fi
 seen_names+="|$name|"
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/exportwalls ./internal/cli
 if [[ $preflight == true ]]; then printf 'COMPILES walls fault: %s\n' "$name"; return; fi
 go test -json -count=1 -timeout=30s -run '^TestWalls' ./internal/exportwalls ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestWalls"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED walls mutation: %s\n' "$name"
}
x=internal/exportwalls/walls.go
c=internal/cli/walls.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "walls-srv"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(raw)' 'sha256.Sum256(nil)'
mutate output-hash "$x" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate source-size "$x" 'SourceSize: len(raw)' 'SourceSize: len(raw)+1'
mutate output-size "$x" 'r.OutputSize = len(b)' 'r.OutputSize = len(b)+1'
mutate unsupported "$x" 'if !d.Supported() {' 'if false {'
mutate row-offset "$x" 'Span: m.Offsets().Record' 'Span: source.Span{}'
mutate raw-flags "$x" 'Flags: m.FlagsRaw()' 'Flags: 0'
mutate raw-roll "$x" 'Roll: m.RollRaw()' 'Roll: 0'
mutate row-index "$x" 'Index: i, Span: m.Offsets().Record' 'Index: 0, Span: m.Offsets().Record'
mutate row-line "$x" 'v.Line = w.line
		w.write' 'v.Line = w.line + 1
		w.write'
mutate from-identity "$x" 'if v.From.Walls == "" {' 'if false {'
mutate reserved-to "$x" '} else if v.To.Walls == "" {' '} else if false {'
mutate splay "$x" 'v.To.Native == -1' 'v.To.Native == -2'
mutate distance "$x" 'v.Distance < 0' 'v.Distance < -1'
mutate inclination-positive "$x" 'v.Inclination > 16384' 'v.Inclination > 16385'
mutate inclination-negative "$x" 'v.Inclination < -16384' 'v.Inclination < -16385'
mutate zero-link "$x" 'v.Distance == 0 && v.Kind == "leg"' 'false'
mutate unsigned-azimuth "$x" 'angle(int(uint16(v.Azimuth)))' 'angle(int(v.Azimuth))'
mutate angle-scale "$x" 'float64(raw)*360/65536' 'float64(raw)*400/65536'
mutate negative-declination "$x" 'return trips[index].DeclinationRaw(), "stored-explicit"' 'return -trips[index].DeclinationRaw(), "stored-explicit"'
mutate auto "$x" 'if trips[index].AutoDeclination() {' 'if false {'
mutate trip-boundary "$x" 'int(index) >= len(trips)' 'int(index) > len(trips)'
mutate name-limit "$x" 'len(s) > 8' 'len(s) >= 8'
mutate name-alias "$x" 'uint32(id.NativeValue())' 'id.Raw()'
mutate east-sentinel "$x" 'v.East == math.MinInt64' 'false'
mutate north-sentinel "$x" 'v.North == math.MinInt64' 'false'
mutate altitude-sentinel "$x" 'v.Altitude == math.MinInt32' 'false'
mutate reference-name "$x" 'v.Station.Walls == ""' 'false'
mutate reference-axis "$x" 'metres(v.East), metres(v.North)' 'metres(v.North), metres(v.East)'
mutate fix-conflict "$x" 'if previous != coords {' 'if previous != coords && false {'
mutate comment-escape "$x" 'strconv.QuoteToASCII(s)' 'strconv.Quote(s)'
mutate comment-length "$x" 'len(q) > 160' 'len(q) > 170'
mutate output-boundary "$x" 'len(s)+2 > w.max-w.b.Len()' 'len(s)+2 >= w.max-w.b.Len()'
mutate crlf "$x" 'w.b.WriteString("\r\n")' 'w.b.WriteString("\n")'
mutate metres "$x" 'if u%1000 == 0 {
		return sign + strconv.FormatUint(u/1000, 10)' 'if u%1000 == 0 {
		return sign + strconv.FormatUint(u/100, 10)'
mutate cli-name "$c" '"survey.srv"' '"survey.txt"'
mutate cli-extra-options "$c" 'len(args) != 6' 'len(args) < 6'
mutate self-leg "$x" 'v.Distance > 0 && v.From.Native == v.To.Native' 'false'
mutate zero-constraints "$x" 'if !constraints.join(v.From.Native, v.To.Native) {' 'if !constraints.join(v.From.Native, v.To.Native) && false {'
mutate fixed-propagation "$x" 'if !oka && okb {' 'if false {'
mutate grid "$x" 'Case=Mixed Grid=0' 'Case=Mixed'
if [[ $preflight != true ]]; then
 jq -s '.' mutants.jsonl >"$report"
 jq -e 'length == 43 and (map(.name) | unique | length) == 43 and
   all(.[]; (.name | type == "string" and length > 0) and .status == "KILLED")' "$report" >/dev/null
fi
