#!/usr/bin/env bash
# P16 Survex individual-reading/lineage/units/CLI faults, fail closed.
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
files=(internal/exportsvx/svx.go internal/cli/svx.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/svx-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exportsvx ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
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
 go build ./internal/exportsvx ./internal/cli
 if [[ $preflight == true ]]; then printf 'COMPILES svx fault: %s\n' "$name"; return; fi
 go test -json -count=1 -timeout=30s -run '^TestSVX' ./internal/exportsvx ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestSVX"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED svx mutation: %s\n' "$name"
}
x=internal/exportsvx/svx.go
c=internal/cli/svx.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "survex-svx"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(raw)' 'sha256.Sum256(nil)'
mutate output-hash "$x" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate source-size "$x" 'SourceSize: len(raw)' 'SourceSize: len(raw)+1'
mutate output-size "$x" 'r.OutputSize = len(b)' 'r.OutputSize = len(b)+1'
mutate version "$x" 'SourceVersion: d.Version()' 'SourceVersion: 0'
mutate decimal-reset "$x" '"*set decimal ."' '"*set decimal ,"'
mutate comment-reset "$x" '"*set comment x3b"' '"*set comment x23"'
mutate lexical-scope "$x" '"*begin", "*set comment x3b", "*set decimal ."' '"*set comment x3b", "*set decimal .", "*begin"'
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
mutate undefined-from "$x" 'if m.From().NativeValue() == -1 {' 'if false {'
mutate negative-distance "$x" 'if m.DistanceMM() < 0 {' 'if false {'
mutate raw-inclination "$x" 'if m.InclinationRaw() < -16384 || m.InclinationRaw() > 16384 {' 'if false {'
mutate self-leg "$x" 'if m.From().SameIdentity(m.To()) {' 'if false {'
mutate mixed-zero "$x" '(rows[member.RowIndex].Source.DistanceMM() == 0) != zero' '(rows[member.RowIndex].Source.DistanceMM() == 0) == zero'
mutate cancel "$x" 'len(members) > 1 && !zero && !(g.Forward.DistanceMM > 0)' 'false'
mutate trip-date "$x" 'if first || trip != previousTrip {' 'if first || trip == previousTrip {'
mutate decl-transition "$x" 'if first || decl != previousDecl {' 'if first || decl == previousDecl {'
mutate decl-redundant "$x" 'if first || decl != previousDecl {' 'if true || decl != previousDecl {'
mutate decl-sign "$x" 'float64(decl)*360/65536' '-float64(decl)*360/65536'
mutate date-comment "$x" 'w.write(";*date YYYY-MM-DD")' 'w.write("*date YYYY-MM-DD")'
mutate member-reversal "$x" 'v.Reversed = member.Reversed' 'v.Reversed = false'
mutate member-head "$x" 'v.Head = g.HeadIndex' 'v.Head = g.HeadIndex+1'
mutate member-line "$x" 'v.Line = w.line' 'v.Line = w.line+1'
mutate reverse-bearing "$x" 'v.Values.Azimuth+180' 'v.Values.Azimuth+90'
mutate reverse-inclination "$x" 'v.Values.Inclination = -v.Values.Inclination' 'v.Values.Inclination = v.Values.Inclination'
mutate reverse-endpoints "$x" '[2]string{v.To.SVX, v.From.SVX}' '[2]string{v.From.SVX, v.To.SVX}'
mutate anonymous "$x" 'v.SVX = "::"' 'v.SVX = "-"'
mutate raw-id "$x" 'Raw: id.Raw()' 'Raw: 0'
mutate native-id "$x" 'Native: id.NativeValue()' 'Native: int32(id.Raw())'
mutate name-collision "$x" 'ok && owner != v.Native' 'ok && owner == v.Native'
mutate trip-bound "$x" 'int(index) >= len(trips)' 'int(index) > len(trips)'
mutate auto "$x" 'if trips[index].AutoDeclination() {' 'if false {'
mutate distance-precision "$x" 'decimal(v.Values.Distance, 3)' 'decimal(v.Values.Distance, 2)'
mutate inclination-precision "$x" 'decimal(v.Values.Inclination, 6)' 'decimal(v.Values.Inclination, 5)'
mutate bearing-wrap "$x" 'if s == "360.000000" {' 'if false {'
mutate bearing-negative "$x" 'if v < 0 {' 'if v < -1 {'
mutate negative-zero "$x" 's = s[1:]' 's = s'
mutate escape "$x" 'strconv.QuoteToASCII(s)' 's'
mutate lf "$x" 'w.b.WriteByte('\''\n'\'')' 'w.b.WriteByte('\''\r'\'')'
mutate budget "$x" 'len(s)+1 > w.max-w.b.Len()' 'len(s)+1 >= w.max-w.b.Len()'
mutate trip-ticks "$x" 't.Ticks(), t.DeclinationRaw()' '0, t.DeclinationRaw()'
mutate reference-east "$x" 'ref.EastMM(), ref.NorthMM()' 'ref.NorthMM(), ref.EastMM()'
mutate cli-publication "$c" '"survey.svx"' '"survey.txt"'
mutate cli-source-limit "$c" 'top.DefaultDocumentLimits().MaxInputBytes' 100
mutate cli-options "$c" 'len(args) != 6' 'len(args) < 6'
if [[ $preflight != true ]]; then
 jq -s '.' mutants.jsonl >"$report"
 jq -e 'length == 56 and (map(.name) | unique | length) == 56 and
   all(.[]; (.name | type == "string" and length > 0) and .status == "KILLED")' "$report" >/dev/null
fi
