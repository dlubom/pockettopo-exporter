#!/usr/bin/env bash
# P07 targeted placement/projection/closure/inspection faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/processing/geometry.go internal/processing/projection.go internal/processing/closure.go internal/cli/geometry.go internal/cli/run.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/geometry-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/processing ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original remaining changed status=0
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/processing ./internal/cli
 go test -json -count=1 -timeout=30s -run '^TestGeometry' ./internal/processing ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestGeometry"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED geometry mutation: %s\n' "$name"
}
g=internal/processing/geometry.go
p=internal/processing/projection.go
l=internal/processing/closure.go
c=internal/cli/geometry.go
mutate nodes-alias "$g" 'return slices.Clone(r.nodes)' 'return r.nodes'
mutate loops-alias "$g" 'return slices.Clone(r.loops)' 'return r.loops'
mutate reference-east "$g" 'r.EastMM() - baseRef.EastMM()' 'baseRef.EastMM() - r.EastMM()'
mutate reference-north "$g" 'baseRef.NorthMM() - r.NorthMM()' 'r.NorthMM() - baseRef.NorthMM()'
mutate reference-altitude "$g" 'baseRef.AltitudeMM() - r.AltitudeMM()' 'r.AltitudeMM() - baseRef.AltitudeMM()'
mutate duplicate-reference "$g" 'if s.dir[r.Station().NativeValue()] == nil {' 'if true {'
mutate display-reference "$g" 'r.Station().NativeValue() == -1' 'r.Station().String() == ""'
mutate projected-flag "$g" 's.flags&16 != 0' 's.flags&8 != 0'
mutate flipped-flag "$g" 's.flags&1 != 0' 's.flags&4 != 0'
mutate synthetic-root "$g" 'if len(s.stations) == 0 {' 'if false {'
mutate reverse-start "$g" 'parent = s.dir[n.to]' 'parent = nil'
mutate reverse-horizontal "$g" 'n.node.Vector.DX = -n.node.Vector.DX' 'n.node.Vector.DX = n.node.Vector.DX'
mutate reverse-north "$g" 'n.node.Vector.DY = -n.node.Vector.DY' 'n.node.Vector.DY = n.node.Vector.DY'
mutate projected-axis "$g" 'n.fwAzimuth()-axis.fwAzimuth()' 'n.fwAzimuth()+axis.fwAzimuth()'
mutate correction-shift "$g" 'int32(c.X>>30)' 'int32(c.X>>29)'
mutate plan-north "$g" 'p.Y - v.DY + int32(c.Y>>30)' 'p.Y + v.DY + int32(c.Y>>30)'
mutate plan-height "$g" 'p.H - v.DH + int32(c.H>>30)' 'p.H + v.DH + int32(c.H>>30)'
mutate extended-flip "$g" 'p.D + v.DD' 'p.D - v.DD'
mutate adjacent-splay "$g" 'if adjacent && !n.projected() {' 'if adjacent && n.projected() {'
mutate projected-ancestor "$g" 'n = n.start' 'n = nil'
mutate zero-splay "$g" 'n.node.Status = "zero_splay"' 'n.node.Status = "placed"'
mutate splay-height "$g" 'p.H - v.DH, p.D + d' 'p.H + v.DH, p.D + d'
mutate retry-phase "$g" 'if !reverseAll {' 'if reverseAll {'
mutate retry-order "$g" 'slices.Reverse(pending)' '// removed retry reversal'
mutate close-switch "$g" 'if options.CloseLoops {' 'if false {'
mutate work-limit "$g" 's.work > s.maxWork' 's.work > s.maxWork+1'
mutate source-span "$g" 'SourceSpan: row.Offsets().Record' 'SourceSpan: source.Span{}'
mutate signed-sin "$p" 'a := int32(int16(angle))' 'a := int32(uint16(angle)) % 32768'
mutate cosine "$p" 'sinQ30(angle + 16384)' 'sinQ30(angle + 8192)'
mutate mixed-flips "$p" 'if flip1 != flip2 {' 'if false {'
mutate projection-nearest "$p" $'if flip1 != flip2 {\n\t\tif backAbs < forwardAbs {' $'if flip1 != flip2 {\n\t\tif backAbs > forwardAbs {'
mutate projection-width "$p" 'width = 4096' 'width = 4095'
mutate projection-div-order "$p" 'n = int32(q * int64(sinQ30(half-offset)) >> 30)' 'n = int32((q>>30) * int64(sinQ30(half-offset)))'
mutate closure-head-weight "$l" 'l.weight += n.dist' 'l.weight += 1'
mutate common-cleanup "$l" 'a = l.stop[0]' '// leave search cursor uncleared'
mutate closure-mark "$l" 'a.mark = true' 'a.mark = false'
mutate closure-residual "$l" 'c.X += int64(a.X-b.X) << 30' 'c.X -= int64(a.X-b.X) << 30'
mutate closure-correction "$l" 'sign := int64(-1)' 'sign := int64(1)'
mutate closure-threshold "$l" 'if largest <= 10 {' 'if largest <= 100 {'
mutate closure-order "$g" 'slices.Reverse(s.loops)' '// removed loop reversal'
mutate closure-rebuild "$g" 'corrections[n.head] = n.node.Correction' 'corrections[n.head] = Correction{}'
mutate json-schema "$c" 'geometrySummary{1,' 'geometrySummary{2,'
mutate json-source-hash "$c" 'sha256.Sum256(b)' 'sha256.Sum256(b[:len(b)/2])'
mutate json-output-failure "$c" 'if e = json.NewEncoder(stdout).Encode(summary); e != nil {' 'if e = json.NewEncoder(stdout).Encode(summary); false {'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length==44 and (map(.name)|unique|length)==44 and all(.[];.status=="KILLED")' "$report"
