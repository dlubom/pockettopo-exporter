#!/usr/bin/env bash
# P15 Walls grouping/lineage/units/CLI faults, fail closed.
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
mutate source-version "$x" 'SourceVersion: d.Version()' 'SourceVersion: 0'
mutate row-index "$x" 'Index: i, Span: m.Offsets().Record' 'Index: i+1, Span: m.Offsets().Record'
mutate row-span "$x" 'Span: m.Offsets().Record' 'Span: source.Span{}'
mutate row-distance "$x" 'Distance: m.DistanceMM()' 'Distance: 0'
mutate row-azimuth "$x" 'Azimuth: m.AzimuthRaw()' 'Azimuth: 0'
mutate row-inclination "$x" 'Inclination: m.InclinationRaw()' 'Inclination: 0'
mutate row-flags "$x" 'Flags: m.FlagsRaw()' 'Flags: 0'
mutate row-roll "$x" 'Roll: m.RollRaw()' 'Roll: 0'
mutate row-trip "$x" 'Trip: m.TripIndexRaw()' 'Trip: -1'
mutate row-comment "$x" 'Comment: m.Comment()' 'Comment: ""'
mutate row-correction "$x" 'Correction: row.Declination' 'Correction: 0'
mutate member-reversal "$x" 'Reversed = member.Reversed' 'Reversed = false'
mutate member-line "$x" 'Line = line' 'Line = line+1'
mutate member-head "$x" 'Head = g.HeadIndex' 'Head = g.HeadIndex+1'
mutate undefined-from "$x" 'if m.From().NativeValue() == -1 {' 'if false {'
mutate negative-distance "$x" 'if m.DistanceMM() < 0 {' 'if false {'
mutate invalid-inclination "$x" 'if m.InclinationRaw() < -16384 || m.InclinationRaw() > 16384 {' 'if false {'
mutate nonzero-self "$x" 'if m.From().SameIdentity(m.To()) {' 'if false {'
mutate mixed-zero "$x" 'if (m.Source.DistanceMM() == 0) != zero {' 'if false {'
mutate cancellation "$x" 'if !zero && !(values.Distance > 0) {' 'if false {'
mutate group-distance "$x" 'float64(g.Forward.DistanceMM) / 1000' 'float64(g.Forward.DistanceMM) / 100'
mutate group-azimuth "$x" 'float64(g.Forward.AzimuthRadians) * 180 / math.Pi' 'float64(g.Reverse.AzimuthRadians) * 180 / math.Pi'
mutate group-inclination "$x" 'float64(g.Forward.InclinationRadians) * 180 / math.Pi' 'float64(g.Reverse.InclinationRadians) * 180 / math.Pi'
mutate group-singleton "$x" 'if len(members) > 1 {' 'if len(members) >= 1 {'
mutate trip-date-transition "$x" 'if first || trip != previousTrip {' 'if first || trip == previousTrip {'
mutate decl-transition "$x" 'if first || g.Declination != previousDecl {' 'if first || g.Declination == previousDecl {'
mutate decl-duplicates "$x" 'if first || g.Declination != previousDecl {' 'if true || g.Declination != previousDecl {'
mutate decl-sign "$x" 'float64(g.Declination)*360/65536' '-float64(g.Declination)*360/65536'
mutate date-comment "$x" 'w.write(";#Date YYYY-MM-DD")' 'w.write("#Date YYYY-MM-DD")'
mutate output-anonymous "$x" 'v.Walls = "-"' 'v.Walls = "::"'
mutate native-id "$x" 'Native: id.NativeValue()' 'Native: int32(id.Raw())'
mutate raw-id "$x" 'Raw: id.Raw()' 'Raw: 0'
mutate name-fallback "$x" 'len(v.Walls) > 8' 'len(v.Walls) > 7'
mutate name-collision "$x" 'ok && owner != v.Native' 'ok && owner == v.Native'
mutate automatic "$x" 'if trips[index].AutoDeclination() {' 'if false {'
mutate derived-vertical "$x" 'math.Abs(v) > 90.00001' 'math.Abs(v) > 90'
mutate clamp-positive "$x" 'return 90, nil' 'return v, nil'
mutate clamp-negative "$x" 'return -90, nil' 'return v, nil'
mutate bearing-round-wrap "$x" 'if s == "360.000000" {' 'if false {'
mutate bearing-negative "$x" 'if v < 0 {' 'if v < -1 {'
mutate negative-zero "$x" 's = s[1:]' 's = s'
mutate comment-injection "$x" 'strconv.QuoteToASCII(s)' s
mutate crlf "$x" 'w.b.WriteString("\r\n")' 'w.b.WriteString("\n")'
mutate output-budget "$x" 'len(s)+2 > w.max-w.b.Len()' 'len(s)+2 >= w.max-w.b.Len()'
mutate reference-east "$x" 'ref.EastMM(), ref.NorthMM()' 'ref.NorthMM(), ref.EastMM()'
mutate trip-ticks "$x" 't.Ticks(), t.DeclinationRaw()' '0, t.DeclinationRaw()'
mutate limited-date "$x" 'limited = true' 'limited = false'
mutate distance-precision "$x" 'decimal(values.Distance, 3)' 'decimal(values.Distance, 2)'
mutate angle-precision "$x" 'decimal(values.Inclination, 6)' 'decimal(values.Inclination, 5)'
mutate cli-publication "$c" '"survey.srv"' '"survey.txt"'
mutate cli-source-limit "$c" 'top.DefaultDocumentLimits().MaxInputBytes' 100
mutate cli-options "$c" 'len(args) != 6' 'len(args) < 6'
if [[ $preflight != true ]]; then
 jq -s '.' mutants.jsonl >"$report"
 jq -e 'length == 58 and (map(.name) | unique | length) == 58 and
   all(.[]; (.name | type == "string" and length > 0) and .status == "KILLED")' "$report" >/dev/null
fi
