#!/usr/bin/env bash
# P05 text/provenance/publication faults; compile and named behavioral tests required.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/exporttext/text.go internal/nativefmt/fixed.go internal/cli/export.go internal/cli/run.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/text-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/... >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original changed remaining status=0
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/...
 go test -json -count=1 -timeout=30s -run '^TestText|^TestFixed|^TestExport|^TestRun' ./internal/... >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[]; .Action == "fail" and (.Test // "" | test("^TestText|^TestFixed|^TestExport|^TestRun"))) and
  all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >> mutants.jsonl
 printf 'KILLED text mutation: %s\n' "$name"
}
mutate profile internal/exporttext/text.go circle=360 circle=400
mutate update-deviation internal/exporttext/text.go 'Native Update geometry exceptions are not reproduced' 'Native Update reproduced'
mutate unsupported internal/exporttext/text.go 'if !d.Supported() {' 'if false {'
mutate empty-title internal/exporttext/text.go 'title == "" ||' 'false ||'
mutate invalid-title internal/exporttext/text.go '!utf8.ValidString(title) ||' 'false ||'
mutate control-title internal/exporttext/text.go 'strings.ContainsFunc(title, unicode.IsControl)' 'strings.ContainsFunc(title, func(r rune) bool { return unicode.IsControl(r) && false })'
mutate schema internal/exporttext/text.go 'Schema: 1,' 'Schema: 2,'
mutate source-hash internal/exporttext/text.go 'sha256.Sum256(raw)' 'sha256.Sum256(raw[:len(raw)/2])'
mutate source-size internal/exporttext/text.go 'SourceSize: len(raw)' 'SourceSize: len(raw)+1'
mutate source-version internal/exporttext/text.go 'SourceVersion: d.Version()' 'SourceVersion: 2'
mutate source-tail internal/exporttext/text.go 'Tail: d.TailKind()' 'Tail: source.TailAbsent'
mutate text-hash internal/exporttext/text.go 'sha256.Sum256(text)' 'sha256.Sum256(text[:len(text)/2])'
mutate text-size internal/exporttext/text.go 'report.TextSize = len(text)' 'report.TextSize = len(text)+1'
mutate newline internal/exporttext/text.go 'b.WriteString("\r\n")' 'b.WriteString("\n")'
mutate lineage-line internal/exporttext/text.go line++ 'line += 2'
mutate budget-bypass internal/exporttext/text.go 'len(s)+2 > maxTextBytes-b.Len()' false
mutate budget-exclusive internal/exporttext/text.go 'len(s)+2 > maxTextBytes-b.Len()' 'len(s)+2 >= maxTextBytes-b.Len()'
mutate budget-error internal/exporttext/text.go 'if overBudget {' 'if overBudget && false {'
mutate auto-declination internal/exporttext/text.go 'decl = 0' 'decl = 1'
mutate date-year internal/exporttext/text.go 'date.Year(), date.Month(), date.Day()' 'date.Year()+1, date.Month(), date.Day()'
mutate trip-label internal/exporttext/text.go 'i+1, date.Year()' 'i, date.Year()'
mutate sentinel-dist internal/exporttext/text.go 'm.DistanceMM() != math.MinInt32' true
mutate dist-unit internal/exporttext/text.go 'float64(m.DistanceMM())/1000, 3' 'float64(m.DistanceMM())/100, 3'
mutate dist-precision internal/exporttext/text.go 'float64(m.DistanceMM())/1000, 3' 'float64(m.DistanceMM())/1000, 2'
mutate unsigned-angle internal/exporttext/text.go 'v = int(uint16(value))' 'v = int(value)'
mutate angle-units internal/exporttext/text.go 360.0/65536 360.0/65535
mutate azimuth-signed internal/exporttext/text.go 'angle(m.AzimuthRaw(), true)' 'angle(m.AzimuthRaw(), false)'
mutate inclination-unsigned internal/exporttext/text.go 'angle(m.InclinationRaw(), false)' 'angle(m.InclinationRaw(), true)'
mutate negative-trip internal/exporttext/text.go 'm.TripIndexRaw() >= 0 {' 'm.TripIndexRaw() > 0 {'
mutate trip-index internal/exporttext/text.go 'int(m.TripIndexRaw())+1' 'int(m.TripIndexRaw())+2'
mutate comment-presence internal/exporttext/text.go 'if m.HasComment() {' 'if m.Comment() != "" {'
mutate ref-sentinel internal/exporttext/text.go 'if value != sentinel {' 'if value != 0 {'
mutate ref-unit internal/exporttext/text.go 'float64(value)/1000, 2' 'float64(value)/100, 2'
mutate id-alignment internal/exporttext/text.go 'dot+6-len(s)' 'dot+5-len(s)'
mutate ref-alignment internal/exporttext/text.go 'coordinate(r.EastMM(), math.MinInt64, 14)' 'coordinate(r.EastMM(), math.MinInt64, 13)'
mutate special-flag internal/exporttext/text.go 'if b&8 != 0 {' 'if b&32 != 0 {'
mutate invisible-flag internal/exporttext/text.go 'if b&4 != 0 {' 'if b&4 == 0 {'
mutate projected-flag internal/exporttext/text.go 'if b&16 != 0 {' 'if b&16 == 0 {'
mutate flipped-flag internal/exporttext/text.go 'if b&1 != 0 {' 'if b&1 == 0 {'
mutate comment-lf internal/exporttext/text.go 's = strings.ReplaceAll(s, "\n", "")' 's = strings.ReplaceAll(s, "\n", "\\n")'
mutate comment-quotes internal/exporttext/text.go 's = strings.ReplaceAll(s, "\"", "\\\"")' 's = strings.ReplaceAll(s, "\"", "")'
mutate comment-slashes internal/exporttext/text.go 's = strings.ReplaceAll(s, "\\", "\\\\")' 's = strings.ReplaceAll(s, "\\", "")'
mutate drawing-loss internal/exporttext/text.go 'len(p.Elements())+len(side.Elements())' 'len(p.Elements())+len(side.Elements())*0'
mutate raw-roll internal/exporttext/text.go 'm.RollRaw(), m.TripIndexRaw()' '0, m.TripIndexRaw()'
mutate raw-id internal/exporttext/text.go 'id.Raw(), id.NativeValue(), id.String()' '0, id.NativeValue(), id.String()'
mutate raw-trip internal/exporttext/text.go 'tripRaw{t.Ticks(), t.DeclinationRaw()}' 'tripRaw{0, t.DeclinationRaw()}'
mutate raw-ref internal/exporttext/text.go 'r.EastMM(), r.NorthMM(), r.AltitudeMM()' 'r.NorthMM(), r.EastMM(), r.AltitudeMM()'
mutate framework-precision internal/nativefmt/fixed.go 'shift := 14 - exponent' 'shift := 15 - exponent'
mutate midpoint internal/nativefmt/fixed.go 'digits[keep] >= '"'"'5'"'"'' 'digits[keep] > '"'"'5'"'"''
mutate carry internal/nativefmt/fixed.go 'b[i] == '"'"'9'"'"'' 'b[i] == '"'"'8'"'"''
mutate negative-zero internal/nativefmt/fixed.go 'value < 0 && rounded != ""' 'value < 0'
mutate decimal-place internal/nativefmt/fixed.go 'exponent + 1 + places' 'exponent + places'
mutate input-read-bound internal/cli/export.go 'int64(maxBytes)+1' 'int64(maxBytes)'
mutate filename-title internal/cli/export.go 'strings.TrimSuffix(base, filepath.Ext(base))' 'strings.TrimSuffix(base, "")'
mutate ignore-publication internal/cli/export.go 'if err = publish(output, result); err != nil {' 'if err = publish(output, result); false {'
mutate ignore-notification internal/cli/export.go 'if _, err = io.WriteString(stdout, "Exported survey.txt and report.json\n"); err != nil {' 'if _, err = io.WriteString(stdout, "Exported survey.txt and report.json\n"); false {'
mutate ignore-sync internal/cli/export.go 'err = f.Sync()' '_ = f.Sync()'
mutate ignore-close internal/cli/export.go 'if err == nil {
		err = closeErr
	}
	return err' 'if err == nil {
		err = closeErr; err = nil
	}
	return err'
mutate short-write internal/cli/export.go 'err == nil && n != len(b)' false
mutate cleanup-identity internal/cli/export.go 'if !os.SameFile(current, p.identity) {' 'if os.SameFile(current, p.identity) {'
mutate wrong-format internal/cli/run.go 'args[3] != "text"' 'args[3] == "text"'
jq -s '.' mutants.jsonl > "$report"
jq -e 'length == 61 and (map(.name)|unique|length)==61 and all(.[];.status=="KILLED")' "$report"
