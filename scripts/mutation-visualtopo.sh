#!/usr/bin/env bash
# P11 native VisualTopo/dimensions/lineage/CLI faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/exportvisualtopo/visualtopo.go internal/exportvisualtopo/encoding.go internal/processing/visualtopo.go internal/cli/visualtopo.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/visualtopo-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exportvisualtopo ./internal/processing ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original remaining changed status=0
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/exportvisualtopo ./internal/processing ./internal/cli
 go test -json -count=1 -timeout=30s -run '^TestVisualTopo' ./internal/exportvisualtopo ./internal/processing ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestVisualTopo"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED visualtopo mutation: %s\n' "$name"
}
x=internal/exportvisualtopo/visualtopo.go
p=internal/processing/visualtopo.go
e=internal/exportvisualtopo/encoding.go
c=internal/cli/visualtopo.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "pockettopo-1.372-visualtopo"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(d.Bytes())' 'sha256.Sum256(nil)'
mutate output-hash "$x" 'sha256.Sum256(w.b)' 'sha256.Sum256(nil)'
mutate output-size "$x" 'report.OutputSize = len(w.b)' 'report.OutputSize = len(w.b)+1'
mutate group-source "$x" 'Source: len(templates), Head:' 'Source: 0, Head:'
mutate raw-flags "$x" 'm.FlagsRaw(), m.RollRaw()' '0, m.RollRaw()'
mutate raw-roll "$x" 'm.FlagsRaw(), m.RollRaw()' 'm.FlagsRaw(), 0'
mutate raw-span "$x" 'm.Offsets().Record, m.From().Raw()' 'source.Span{}, m.From().Raw()'
mutate reference-values "$x" 'r.EastMM(), r.NorthMM(), r.AltitudeMM()' 'r.NorthMM(), r.EastMM(), r.AltitudeMM()'
mutate trip-values "$x" 't.Ticks(), t.DeclinationRaw()' '0, t.DeclinationRaw()'
mutate template-hash "$x" 'sha256.Sum256(t.Bytes())' 'sha256.Sum256(nil)'
mutate template-groups "$x" 'g.HeadIndex, g.From.Raw(), g.To.Raw(), g.Members()' 'g.HeadIndex, g.From.Raw(), g.To.Raw(), nil'
mutate identity-raw "$x" 'n.ID.Raw(), n.ID.NativeValue(), n.ID.String()' '0, n.ID.NativeValue(), n.ID.String()'
mutate identity-native "$x" 'n.ID.Raw(), n.ID.NativeValue(), n.ID.String()' 'n.ID.Raw(), 0, n.ID.String()'
mutate identity-display "$x" 'n.ID.Raw(), n.ID.NativeValue(), n.ID.String()' 'n.ID.Raw(), n.ID.NativeValue(), ""'
mutate group-line "$x" 'rec.Line = w.line' 'rec.Line = w.line+1'
mutate group-omission "$x" 'rec.Omission = ""' 'rec.Omission = "omitted"'
mutate splay-omission "$x" 'if n.Status != "placed" {' 'if true {'
mutate dimensions-report "$x" 'rec.Dimensions = dims' 'rec.Dimensions = [4]processing.PassageDimension{}'
mutate rays-report "$x" 'rec.Rays = s.Rays' 'rec.Rays = nil'
mutate crlf "$x" 'encode(s + "\r\n")' 'encode(s + "\n")'
mutate byte-bound "$x" 'len(b) > w.limit-len(w.b)' 'len(b) >= w.limit-len(w.b)'
mutate single-precision "$x" 'places := 6 - exponent' 'places := 7 - exponent'
mutate param-sign "$x" 'sign = "-"' 'sign = "+"'
mutate param-units "$x" '(360.0 / 65536)' '(400.0 / 65536)'
mutate data-length "$x" 'float64(dir.DistanceMM)/1000' 'float64(dir.DistanceMM)/100'
mutate data-angle "$x" 'single(dir.AzimuthRadians*float32(opt.Circle)/float32(6.2831855), 7)' 'single(dir.AzimuthRadians*float32(opt.Circle)/float32(6.28), 7)'
mutate dimension-float "$x" 'float64(float32(v.MM))/1000' 'float64(float32(v.MM))/100'
mutate next-adjacency "$x" 's.Node.ID.NativeValue()+1' 's.Node.ID.NativeValue()+2'
mutate next-distance "$x" 'next.Distance > 0' 'next.Distance < 0'
mutate flip-start "$x" 's.Flipped != s.StartFlipped' 's.Flipped'
mutate normalize-comment "$e" '";", ","' '";", ";"'
mutate encode-supplementary "$e" 'codepage[a], codepage[z]' 'codepage[a], byte(z)'
mutate avg-direction "$p" 'row.Declination, false' 'row.Declination, true'
mutate automatic-decl "$p" 'v.Declination = 0' 'v.Declination = -32768'
mutate synthetic-filter "$p" 'if i < 0 {' 'if i < -1 {'
mutate raw-to "$p" 'v.To = row.Source.To()' 'v.To = row.Source.From()'
mutate sort-order "$p" 'a[i].Node.ID.NativeValue()-pivot >= 0' 'a[i].Node.ID.NativeValue()-pivot <= 0'
mutate sort-ties "$p" 'if i < j {' 'if false {'
mutate bisector-shift "$p" 'int32(int16(b-a)) >> 1' 'int32(int16(b-a)) >> 2'
mutate bisector-wrap "$p" 'int32(int16(b-a))' '(b-a)'
mutate strict-threshold "$p" 'best := int32(500000)' 'best := int32(499999)'
mutate dimension-threshold "$p" 'best := int32(500000)' 'best := int32(1000000)'
mutate dimension-tie "$p" 'if dot > best {' 'if dot >= best {'
mutate dimension-targets "$p" 'axis - 16384, axis + 16384' 'axis + 16384, axis - 16384'
mutate dimension-sentinel "$p" 'if az == math.MinInt32 {' 'if axis == math.MinInt32 {'
mutate dimension-scale "$p" 'int64(dot) / 1000000' 'int64(dot) / 100000'
mutate dimension-key "$p" 'r.Key, dot' '"", dot'
mutate cli-grads "$c" 'circle = 400' 'circle = 360'
mutate cli-inventory "$c" 'Name: s.Name' 'Name: "wrong"'
mutate cli-circle-duplicate "$c" 'if seenCircle ||' 'if (seenCircle && false) ||'
mutate cli-close "$c" 'Geometry: processing.GeometryOptions{CloseLoops: close}' 'Geometry: processing.GeometryOptions{CloseLoops: false}'
jq -s '.' mutants.jsonl >"$report"
