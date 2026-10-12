#!/usr/bin/env bash
# P09 native Therion serialization/lineage/CLI faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/exporttherion/therion.go internal/processing/therion.go internal/cli/therion.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/therion-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exporttherion ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original remaining changed status=0
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/exporttherion ./internal/cli
 go test -json -count=1 -timeout=30s -run '^TestTherion' ./internal/exporttherion ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestTherion"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED therion mutation: %s\n' "$name"
}
x=internal/exporttherion/therion.go
p=internal/processing/therion.go
c=internal/cli/therion.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "pockettopo-1.372-therion"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(d.Bytes())' 'sha256.Sum256(nil)'
mutate text-hash "$x" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate text-size "$x" 'report.TextSize = len(b)' 'report.TextSize = len(b)+1'
mutate crlf "$x" 'w.b.WriteString("\r\n")' 'w.b.WriteString("\n")'
mutate budget-inclusive "$x" 'len(s)+2 > w.limit-w.b.Len()' 'len(s)+2 >= w.limit-w.b.Len()'
mutate budget-error "$x" $'if w.failed {\n\t\treturn Result{},' $'if false {\n\t\treturn Result{},'
mutate coordinate-y "$x" 'fields(coord(x), coord(-y))' 'fields(coord(x), coord(y))'
mutate coordinate-unit "$x" 'float64(v)/1000, 3' 'float64(v)/100, 3'
mutate reference-order "$x" 'for i, r := range refs {' 'for i := range refs { r:=refs[len(refs)-1-i]'
mutate trip-auto "$x" 'decl = 0' 'decl = 1'
mutate trip-date "$x" 'date.Year(), date.Month(), date.Day()' 'date.Year()+1, date.Month(), date.Day()'
mutate trip-declination "$x" 'float64(decl)*360/65536' 'float64(decl)*400/65536'
mutate data-boundary "$x" $'if len(templates) > 0 {\n\t\tfor _, t := range templates' $'if false {\n\t\tfor _, t := range templates'
mutate data-first "$x" 'has && m.To().NativeValue() != -1 && samePair(m, prev)' 'has && m.To().NativeValue() != -1 && samePair(m, prev) && false'
mutate data-no-trip "$x" 'm.TripIndexRaw() < 0 || int(m.TripIndexRaw()) >= len(trips)' 'm.TripIndexRaw() < 0 || int(m.TripIndexRaw()) > len(trips)'
mutate zero-splay "$x" 'm.To().NativeValue() == -1 && m.DistanceMM() == 0' 'false'
mutate member-span "$x" 'rows[member.RowIndex].Offsets().Record' 'source.Span{}'
mutate member-source "$x" 'Member{len(templates), member.RowIndex' 'Member{0, member.RowIndex'
mutate data-direction "$x" 'direction := view.Heads[i].Direction' 'direction := gs[i].Forward'
mutate flip "$x" 'rows[g.HeadIndex].FlagsRaw()&1 != 0' 'false'
mutate angle-float32 "$x" 'float64(v*360)' 'float64(v)*360.001'
mutate sorted-comparer "$x" 'out[mid].Node.ID.NativeValue() - n.Node.ID.NativeValue()' 'n.Node.ID.NativeValue() - out[mid].Node.ID.NativeValue()'
mutate ordering-budget "$x" 'work += len(out) - lo + 1' 'work += 0'
mutate default-color "$x" 'return "GREEN"' 'return "BLACK"'
mutate all-red "$x" 'return "RED"' 'return "BLUE"'
mutate lookup-key "$x" 'lookup[n.Node.ID.NativeValue()] = n' 'lookup[0] = n'
mutate drawing-side "$x" $'if side {\n\t\treturn point(p.D, p.H)' $'if false {\n\t\treturn point(p.D, p.H)'
mutate origin-offset "$x" 'ox, oy := view.Origin.X, view.Origin.Y' 'ox, oy := int32(0),int32(0)'
mutate cross-tab "$x" 'fields(point(cx, cy), point(rx, ry), "")' 'fields(point(cx, cy), point(rx, ry))'
mutate connect "$x" 'w.write("POLYLINE CONNECT")' 'w.write("POLYLINE BLACK")'
mutate polygon-header "$x" 'w.write(fmt.Sprintf("POLYLINE %s", color(p.ColorRaw())))' 'w.write(color(p.ColorRaw()))'
mutate native-order "$p" 'n = n.nextActual' 'n = nil'
mutate template-origin "$p" 'rows = groups.Templates.rows' 'rows = nil'
mutate reverse-average "$p" 'n.to != -1 && n.node.ID.NativeValue() == n.from' 'false'
mutate horizontal-ray "$p" 'x + c.Node.Vector.DX, y - c.Node.Vector.DY' 'x - c.Node.Vector.DX, y + c.Node.Vector.DY'
mutate vertical-ray "$p" 'y - c.Node.Vector.DH' 'y + c.Node.Vector.DH'
mutate raw-ray-azimuth "$p" 'c.Azimuth-direction' 'direction-c.Azimuth'
mutate close-option "$c" 'CloseLoops: close' 'CloseLoops: false'
mutate duplicate-close "$c" 'if close {' 'if false {'
mutate inventory "$c" 'Name: s.Name' 'Name: "wrong"'
mutate notify-failure "$c" 'e != nil {
		return exportError(stderr, fmt.Errorf("export published;' 'false {
		return exportError(stderr, fmt.Errorf("export published;'
jq -s '.' mutants.jsonl >"$report"
