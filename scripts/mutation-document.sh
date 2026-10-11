#!/usr/bin/env bash
# Complete-file, drawing and inspection faults; compile and named tests required.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/top/document.go internal/source/document.go internal/cli/inspect.go internal/cli/run.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/document-mutation.json"
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
 go test -json -count=1 -timeout=30s -run '^TestDocument|^TestInspect' ./internal/... >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[]; .Action == "fail" and (.Test // "" | test("^TestDocument|^TestInspect"))) and
  all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >> mutants.jsonl
 printf 'KILLED document mutation: %s\n' "$name"
}
r=internal/top/document.go
s=internal/source/document.go
c=internal/cli/inspect.go
mutate validation-lower "$r" 'b.value < 0' 'b.value < -1'
mutate validation-upper "$r" 'b.value > b.max' 'b.value > b.max+1'
mutate bypass-table-limits "$r" 'data, l.ReferenceLimits' 'data, DefaultReferenceLimits()'
mutate continue-unknown-plan "$r" 'plan.StopReason() == source.PlanStopUnsupportedElement' 'false'
mutate ignore-side-error "$r" 'side, e := r.drawing("side", l, budget)' 'side, e := r.drawing("side", l, budget); e = nil'
mutate tail-partial "$r" 'n < 4' 'n < 3'
mutate tail-masked "$r" 'binary.LittleEndian.Uint32(data[r.offset:r.offset+4]) != 0' 'data[r.offset] != 0'
mutate require-trailer "$r" 'tail = source.TailAbsent' 'tail = source.TailPartial'
mutate ignore-extra "$r" 'tail = source.TailExtra' 'tail = source.TailZeroTrailer'
mutate consume-partial "$r" 'tail = source.TailPartial' 'tail = source.TailPartial; r.offset = len(data)'
mutate body-end "$r" 'bodyEnd := r.offset' 'bodyEnd := r.offset+1'
mutate raw-mapping-x "$r" 'values[0], values[1], values[2]' 'values[1], values[0], values[2]'
mutate normalize-scale "$r" 'values[0], values[1], values[2]' 'values[0], values[1], values[2]/5'
mutate element-exclusive "$r" 'len(elements) >= l.MaxElements' 'len(elements) >= l.MaxElements-1'
mutate element-over "$r" 'len(elements) >= l.MaxElements' 'len(elements) > l.MaxElements'
mutate document-element-bypass "$r" 'budget.elements == 0' 'false'
mutate document-element-reset "$r" 'budget.elements--' 'budget.elements = l.MaxDocumentElements'
mutate points-exclusive "$r" 'n > l.MaxPoints' 'n >= l.MaxPoints'
mutate points-bypass "$r" 'n > l.MaxPoints' 'false'
mutate drawing-points-bypass "$r" 'n > l.MaxTotalPoints-total' 'n > l.MaxTotalPoints'
mutate document-points-bypass "$r" 'n > budget.points' 'false'
mutate document-points-reset "$r" 'budget.points -= n' 'budget.points = l.MaxDocumentPoints'
mutate total-reset "$r" 'total += n' 'total = n'
mutate short-preflight "$r" '(len(r.data)-r.offset)/8' '(len(r.data)-r.offset)/4'
mutate negative-count "$r" 'count < 0' 'count < -1'
mutate stop-mask "$r" 'marker := b[0]' 'marker := b[0]&3'
mutate xs-direction "$r" 'int32(raw[3])' 'int32(raw[3]&65535)'
mutate color-mask "$r" 'points, color[0], offsets' 'points, color[0]&7, offsets'
mutate alias-input "$s" 'bytes.Clone(raw)' 'raw'
mutate alias-output "$s" 'return bytes.Clone(d.raw)' 'return d.raw'
mutate alias-tail "$s" 'return bytes.Clone(d.raw[d.consumed:])' 'return d.raw[d.consumed:]'
mutate alias-elements-in "$s" 'slices.Clone(elements)' 'elements'
mutate alias-elements-out "$s" 'return slices.Clone(d.elements)' 'return d.elements'
mutate alias-trips "$s" 'return slices.Clone(d.trips)' 'return d.trips'
mutate alias-measurements "$s" 'return slices.Clone(d.measurements)' 'return d.measurements'
mutate alias-references "$s" 'return slices.Clone(d.references)' 'return d.references'
mutate supported-incomplete "$s" 'd.Complete() &&' 'true &&'
mutate unsupported-tail-success "$s" 'd.tail == TailAbsent || d.tail == TailZeroTrailer' 'd.tail != TailUnreached'
mutate wrong-schema "$c" 'Schema: 1' 'Schema: 2'
mutate wrong-hash "$c" 'sha256.Sum256(b)' 'sha256.Sum256(b[:len(b)/2])'
mutate wrong-size "$c" 'Size: len(b)' 'Size: len(b)+1'
mutate bad-exit "$c" 'if !s.Supported {
		return 1' 'if false {
		return 1'
mutate unknown-complete "$c" 's.Complete = d.Complete()' 's.Complete = true'
mutate side-as-plan "$c" 's.Side = summarizeDrawing(side)' 's.Side = summarizeDrawing(p)'
mutate dropped-diagnostic "$c" 'diagnostic{pe.Code, pe.Field, pe.Offset}' 'diagnostic{pe.Code, pe.Field, 0}'
mutate wrong-polygon-count "$c" 's.Polygons++' 's.XSections++'
mutate ignored-write-error "$c" 'if e = json.NewEncoder(stdout).Encode(s); e != nil {' 'if e = json.NewEncoder(stdout).Encode(s); false {'
jq -s '.' mutants.jsonl > "$report"
jq -e 'length == 47 and (map(.name)|unique|length)==47 and all(.[];.status=="KILLED")' "$report"
