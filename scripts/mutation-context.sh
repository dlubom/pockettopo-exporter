#!/usr/bin/env bash
# P08 targeted inventory/grouping/geometry/lineage faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/processing/context.go internal/processing/grouped.go internal/processing/geometry.go internal/processing/closure.go internal/cli/context.go internal/cli/context_input.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/context-mutation.json"
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
 go test -json -count=1 -timeout=30s -run '^TestContext' ./internal/processing ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestContext"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED context mutation: %s\n' "$name"
}
g=internal/processing/context.go
p=internal/processing/geometry.go
l=internal/processing/closure.go
c=internal/cli/context.go
i=internal/cli/context_input.go
mutate template-version "$g" 'if t.Version() != 3 {' 'if false {'
mutate cumulative-trip-limit "$g" 'contextTablesWithinLimits(len(trips)+len(tt)+len(currentTrips), measurements, refs)' 'contextTablesWithinLimits(len(trips)+len(tt)+len(currentTrips)+1, measurements, refs)'
mutate final-trip-limit "$g" 'trips <= 32767' 'trips <= 32768'
mutate template-file-limit "$g" 'if len(templates) > 1024 {' 'if false {'
mutate trip-offset "$g" 'off += len(t.Trips())' 'off += 0'
mutate template-lineage "$g" 'ContextLineage{i, j}' 'ContextLineage{i+1, j}'
mutate current-lineage "$g" 'ContextLineage{len(templates), j}' 'ContextLineage{0, j}'
mutate templates-isolated "$g" 'r.Templates = groupRows(rows)' 'r.Templates = groupRows(nil)'
mutate template-phase "$g" 'if len(templates) > 0 {' 'if false {'
mutate template-references "$g" 'refs = append(refs, t.References()...)' 'refs = append(refs, t.References()[:0]...)'
mutate template-directory "$g" 'state.templates = state.dir' 'state.templates = nil'
mutate template-fixed "$g" 'n.fixed = true' 'n.fixed = false'
mutate template-key "$g" 'state.prefix = "template:"' 'state.prefix = ""'
mutate suppress-current-root "$g" 'state.suppressRoot = true' 'state.suppressRoot = false'
mutate current-only-closure "$g" 'if options.CloseLoops {' 'if false {'
mutate current-correction "$g" 'corrections[n.head] = n.node.Correction' 'corrections[n.head] = Correction{}'
mutate template-nodes "$g" 'append(templateNodes, state.stations...)' 'state.stations'
mutate carried-reference "$p" 'haveBase := s.baseRef != nil' 'haveBase := false'
mutate template-parent "$p" $'if parent == nil {\n\t\tparent = s.templates[n.from]\n\t}\n\tif parent == nil && reverse {' $'if parent == nil {\n\t\tparent = nil\n\t}\n\tif parent == nil && reverse {'
mutate fixed-first-arm "$l" '!a.reference && !a.fixed' '!a.reference'
mutate fixed-second-arm "$l" '!b.reference && !b.fixed' '!b.reference'
mutate exclude-current "$i" 'name == current ||' 'false ||'
mutate file-extension "$i" 'strings.EqualFold(filepath.Ext(name), ".top")' '(strings.Compare(filepath.Ext(name), ".top") == 0)'
mutate directory-order "$i" 'slices.Sort(names)' 'slices.Reverse(names)'
mutate template-symlink "$i" 'if !entry.Type().IsRegular() {' 'if false {'
mutate current-regular "$i" 'if !info.Mode().IsRegular() {' 'if info.Mode().IsRegular() {'
mutate filename-utf8 "$i" 'if !utf8.ValidString(filepath.Base(path)) {' 'if false {'
mutate inventory-hash "$i" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate inventory-ignored "$i" 'len(b) - consumed' '0'
mutate selected-file-limit "$i" 'len(names) >= maxContextFiles' 'len(names) > maxContextFiles'
mutate context-schema "$c" 'Schema: 1' 'Schema: 2'
mutate context-source-index "$c" 'l.SourceIndex, l.RowIndex, m.Reversed' '0, l.RowIndex, m.Reversed'
mutate context-member-reversed "$c" 'l.SourceIndex, l.RowIndex, m.Reversed' 'l.SourceIndex, l.RowIndex, false'
mutate context-output-failure "$c" 'if e = json.NewEncoder(stdout).Encode(s); e != nil {' 'if e = json.NewEncoder(stdout).Encode(s); false {'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length==34 and (map(.name)|unique|length)==34 and all(.[];.status=="KILLED")' "$report"
