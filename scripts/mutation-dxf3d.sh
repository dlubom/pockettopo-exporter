#!/usr/bin/env bash
# P13 native 3D DXF/lineage/hull/CLI faults, fail closed.
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
files=(internal/exportdxf3d/dxf3d.go internal/exportdxf3d/hull.go internal/cli/dxf3d.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/dxf3d-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exportdxf3d ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
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
 go build ./internal/exportdxf3d ./internal/cli
 if [[ $preflight == true ]]; then printf 'COMPILES dxf3d fault: %s\n' "$name"; return; fi
 go test -json -count=1 -timeout=30s -run '^TestDXF3D' ./internal/exportdxf3d ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestDXF3D"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED dxf3d mutation: %s\n' "$name"
}
x=internal/exportdxf3d/dxf3d.go
w=internal/exportdxf3d/hull.go
c=internal/cli/dxf3d.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "pockettopo-1.372-3d-dxf"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(raw)' 'sha256.Sum256(nil)'
mutate output-hash "$x" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate output-size "$x" 'r.OutputSize = len(b)' 'r.OutputSize = len(b)+1'
mutate source-size "$x" 'SourceSize: len(raw)' 'SourceSize: len(raw)+1'
mutate raw-row-flags "$x" 'Flags: v.FlagsRaw()' 'Flags: 0'
mutate raw-row-roll "$x" 'Roll: v.RollRaw()' 'Roll: 0'
mutate member-reversal "$x" 'Reversed: m.Reversed' 'Reversed: false'
mutate row-source "$x" 'Source: l.SourceIndex' 'Source: 0'
mutate reference-key "$x" 'refIndex++' 'refIndex += 0'
mutate row-span "$x" 'Span: v.Offsets().Record, From:' 'Span: source.Span{}, From:'
mutate raw-identity "$x" 'Raw: n.ID.Raw()' 'Raw: 0'
mutate raw-reference "$x" 'East: v.EastMM(), North: v.NorthMM()' 'East: v.NorthMM(), North: v.EastMM()'
mutate synthetic-root "$x" 's.Node.Key != "reference:-1"' 'true'
mutate templates "$x" 'for _, s := range v.Template {' 'for _, s := range v.Template[:0] {'
mutate invisible "$x" 's.Flags&4 != 0' 'false'
mutate positive-distance "$x" 's.Distance <= 0' 's.Distance < 0'
mutate ancestor-fallback "$x" 'len(n.Cross) == 0 && n.Distance == 0' 'false && n.Distance == 0'
mutate ancestor-cross "$x" 'len(n.Cross) == 0 && n.Distance == 0' 'n.Distance == 0'
mutate point-budget "$x" 'len(r.Points) >= w.lim.points' 'len(r.Points) > w.lim.points'
mutate point-key "$x" 'PointRecord{c.Node.Key,' 'PointRecord{"",'
mutate point-index "$x" 'p.index = len(r.Points)' 'p.index = 0'
mutate point-order "$x" 'pts[i], pts[j] = pts[j], pts[i]' 'pts[i], pts[j] = pts[i], pts[j]'
mutate stretch-point-range "$x" 'rec.PointCount = len(r.Points) - rec.FirstPoint' 'rec.PointCount = len(r.Points)'
mutate stretch-entity-range "$x" 'rec.EntityCount = len(r.Entities) - rec.FirstEntity' 'rec.EntityCount = len(r.Entities)'
mutate quantization-x "$w" 'p.X >> 4' 'p.X >> 3'
mutate quantization-y "$w" '(-p.Y) >> 4' '(p.Y) >> 4'
mutate quantization-h "$w" '(-p.H) >> 4' '(p.H) >> 4'
mutate cross-product "$w" 'u.y*v.z - u.z*v.y' 'u.y*v.z + u.z*v.y'
mutate dot-width "$w" 'int64(a.x)*int64(b.x)' 'int64(a.x*b.x)'
mutate crlf "$w" 'w.b.WriteString("\r\n")' 'w.b.WriteString("\n")'
mutate output-budget "$w" 'len(s)+2 > w.lim.bytes-w.b.Len()' 'len(s)+2 >= w.lim.bytes-w.b.Len()'
mutate entity-budget "$w" 'len(w.report.Entities) >= w.lim.entities' 'len(w.report.Entities) > w.lim.entities'
mutate work-budget "$w" 'w.work >= w.lim.work' 'w.work > w.lim.work'
mutate entity-first "$w" 'FirstLine: w.lineNumber' 'FirstLine: w.lineNumber+1'
mutate entity-last "$w" 'e.LastLine = w.lineNumber - 1' 'e.LastLine = w.lineNumber'
mutate entity-owner "$w" 'Owner: w.owner' 'Owner: ""'
mutate fourth-vertex "$w" 'w.emit("3DFACE", a, b, c, c)' 'w.emit("3DFACE", a, b, c, a)'
mutate face-normal "$w" 'w.prev = v' 'w.prev = vector{}'
mutate sort-wrap "$w" 'a[i].Node.ID.NativeValue()-pivot >= 0' 'int64(a[i].Node.ID.NativeValue())-int64(pivot) >= 0'
mutate sort-ties "$w" 'if i < j {' 'if false {'
mutate cli-close "$c" 'opt.Geometry.CloseLoops = true' 'opt.Geometry.CloseLoops = false'
mutate cli-inventory "$c" 'Name: s.Name' 'Name: "wrong"'
mutate cli-duplicates "$c" 'if seen[key] {' 'if false {'
if [[ $preflight != true ]]; then
 jq -s '.' mutants.jsonl >"$report"
 jq -e 'length == 45 and (map(.name) | unique | length) == 45 and
   all(.[]; (.name | type == "string" and length > 0) and .status == "KILLED")' "$report" >/dev/null
fi
