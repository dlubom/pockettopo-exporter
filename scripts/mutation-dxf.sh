#!/usr/bin/env bash
# P10 native graphics DXF/lineage/options/CLI faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/exportdxf/dxf.go internal/exportdxf/writer.go internal/processing/graphics.go internal/nativefmt/general.go internal/cli/dxf.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/dxf-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exportdxf ./internal/processing ./internal/nativefmt ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original remaining changed status=0
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/exportdxf ./internal/processing ./internal/nativefmt ./internal/cli
 go test -json -count=1 -timeout=30s -run '^TestDXF|^TestGraphics|^TestGeneral' ./internal/exportdxf ./internal/processing ./internal/nativefmt ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestDXF|^TestGraphics|^TestGeneral"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED dxf mutation: %s\n' "$name"
}
x=internal/exportdxf/dxf.go
w=internal/exportdxf/writer.go
p=internal/processing/graphics.go
n=internal/nativefmt/general.go
c=internal/cli/dxf.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "pockettopo-1.372-graphics-dxf"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(d.Bytes())' 'sha256.Sum256(nil)'
mutate output-hash "$x" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate output-size "$x" 'report.OutputSize = len(b)' 'report.OutputSize = len(b)+1'
mutate source-size "$x" 'SourceSize: len(d.Bytes())' 'SourceSize: len(d.Bytes())+1'
mutate drawing-side "$x" 'name = "side"' 'name = "plan"'
mutate raw-row-flags "$x" 'm.FlagsRaw(), m.RollRaw()' '0, m.RollRaw()'
mutate raw-row-roll "$x" 'm.FlagsRaw(), m.RollRaw()' 'm.FlagsRaw(), 0'
mutate member-reversal "$x" 'key, member.Reversed}' 'key, false}'
mutate row-source "$x" 'Row{l.SourceIndex, l.RowIndex' 'Row{0, l.RowIndex'
mutate reference-key "$x" 'templateReference++' 'templateReference += 0'
mutate raw-row-span "$x" 'm.Offsets().Record, m.From().Raw()' 'source.Span{}, m.From().Raw()'
mutate raw-identity "$x" 'Identity{n.ID.Raw(), n.ID.NativeValue(), n.ID.String()}' 'Identity{0, n.ID.NativeValue(), n.ID.String()}'
mutate raw-reference "$x" 'ref.EastMM(), ref.NorthMM(), ref.AltitudeMM()' 'ref.NorthMM(), ref.EastMM(), ref.AltitudeMM()'
mutate polygon-color "$w" 'case 5:' 'case 8:'
mutate crlf "$w" 'w.b.WriteString("\r\n")' 'w.b.WriteString("\n")'
mutate output-budget "$w" 'len(s)+2 > w.lim.bytes-w.b.Len()' 'len(s)+2 >= w.lim.bytes-w.b.Len()'
mutate entity-budget "$w" 'len(w.report.Entities) >= w.lim.entities' 'len(w.report.Entities) > w.lim.entities'
mutate entity-first "$w" 'Entity{kind, w.owner, w.lineNumber, 0}' 'Entity{kind, w.owner, w.lineNumber+1, 0}'
mutate entity-last "$w" 'LastLine = w.lineNumber - 1' 'LastLine = w.lineNumber'
mutate entity-owner "$w" 'Entity{kind, w.owner, w.lineNumber, 0}' 'Entity{kind, "", w.lineNumber, 0}'
mutate coordinate-y "$w" 'float64(-(y + oy))' 'float64(y + oy)'
mutate coordinate-origin "$w" 'float64(x+ox)' 'float64(x)'
mutate coordinate-scale "$w" 'a/float64(w.opt.Scale)' 'a/500'
mutate coordinate-reference "$w" 'a += float64(w.view.Base.EastMM())' 'a += float64(w.view.Base.NorthMM())'
mutate circle-radius "$w" '50/float64(w.opt.Scale)' '500/float64(w.opt.Scale)'
mutate text-height "$w" 'w.pair(40, "1")' 'w.pair(40, "2")'
mutate shot-layer "$w" 'w.opt.Mask&(ShotLayer|RayLayer)' 'w.opt.Mask&ShotLayer'
mutate ray-layer "$w" 'return "XSect"' 'return "Shots"'
mutate label-layer "$w" 'layer = "Label"' 'layer = "Labels"'
mutate grid-layer "$w" 'return "Grid"' 'return "Sketch"'
mutate dashed-style "$w" 'w.pair(6, "DASH")' 'w.pair(6, "CONTINUOUS")'
mutate invisible "$w" 'n.Flags&4 == 0' 'true'
mutate plan-dash "$w" 'absIncl(n.Inclination) > 8192' 'absIncl(n.Inclination) >= 8192'
mutate side-dash "$w" 'dd*(dd>>3)+v*(v>>3)' 'dd*(dd>>3)+v*(v>>3)*0'
mutate native-circle "$w" 'if n.To != -1 {' 'if false {'
mutate label-distance "$w" 'n.Distance > 0' 'n.Distance < 0'
mutate grid-step "$w" 'step := int64(1000)' 'step := int64(1001)'
mutate feet-step "$w" 'step = 1520' 'step = 1524'
mutate grid-padding "$w" 'int64(lx)/step*step - 2*step' 'int64(lx)/step*step - step'
mutate grid-budget "$w" 'count > int64(w.lim.entities-len(w.report.Entities))' 'false'
mutate snapshot-reference-to "$p" 'To: -1, Node: n.node' 'To: 0, Node: n.node'
mutate snapshot-inclination "$p" 'v.Inclination = rows[n.head].Source.InclinationRaw()' 'v.Inclination = 0'
mutate snapshot-template "$p" 'if n.fixed {' 'if false {'
mutate bounds-invisible "$p" 'n.Flags&4 != 0' 'false'
mutate horizontal-ray "$p" 'x + c.Node.Vector.DX, y - c.Node.Vector.DY' 'x - c.Node.Vector.DX, y + c.Node.Vector.DY'
mutate directed-ray "$p" 'y - c.Node.Vector.DH' 'y + c.Node.Vector.DH'
mutate directed-dash "$p" 'magnitude*7 > c.Distance*5' 'magnitude*5 > c.Distance*7'
mutate absolute-ray "$p" 'magnitude = -magnitude' 'magnitude = magnitude'
mutate abs-overflow "$p" 'if dy == math.MinInt32 {' 'if false {'
mutate general-exponent "$n" 'exponent >= 15 || exponent < -4' 'exponent >= 15 || exponent < -5'
mutate general-significant "$n" 'places := 14 - exponent' 'places := 15 - exponent'
mutate general-trim "$n" 'digits = strings.TrimRight(digits, "0")' 'digits = digits'
mutate cli-scale "$c" 'opt.Scale = int32(n)' 'opt.Scale = 500'
mutate cli-mask "$c" 'opt.Mask = uint16(n)' 'opt.Mask = 14'
mutate cli-close "$c" 'opt.Geometry.CloseLoops = true' 'opt.Geometry.CloseLoops = false'
mutate cli-feet "$c" 'opt.Feet = value == "feet"' 'opt.Feet = false'
mutate cli-inventory "$c" 'Name: s.Name' 'Name: "wrong"'
mutate cli-duplicates "$c" 'if seen[key] {' 'if false {'
jq -s '.' mutants.jsonl >"$report"
