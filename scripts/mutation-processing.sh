#!/usr/bin/env bash
# P06 targeted grouping/lineage/arithmetic/inspection faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/processing/angle.go internal/processing/grouped.go internal/cli/process.go internal/cli/run.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/processing-mutation.json"
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
 go test -json -count=1 -timeout=30s -run '^TestNative|^TestGrouped|^TestProcess' ./internal/processing ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^Test(Native|Grouped|Process)"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED processing mutation: %s\n' "$name"
}
a=internal/processing/angle.go
g=internal/processing/grouped.go
c=internal/cli/process.go
mutate sine-units "$a" '9.587379924285257e-05' '9.587379924285257e-04'
mutate sine-fixed-point "$a" '* 1073741824' '* 1073741823'
mutate signed-angle "$a" 'a := int32(int16(angle))' 'a := angle & 32767'
mutate negative-rounding "$a" 'int64(length)*int64(sin) + 536870912' 'int64(length)*int64(sin) + 536870911'
mutate rect-cos "$a" 'int64(length)*int64(cos)' 'int64(length)*(int64(cos)+int64(sin))'
mutate unsupported "$g" 'if !d.Supported() {' 'if false {'
mutate auto-declination "$g" 'decl = 0' 'decl = -32768'
mutate trip-correction "$g" 'decl = trips[idx].DeclinationRaw()' 'decl = trips[0].DeclinationRaw()'
mutate row-vector "$g" 'r.rows[i] = Row{i, m, decl, real, Vector{dx, dy, dh, dd}}' 'r.rows[i] = Row{i, m, decl, real, Vector{dy, dx, dh, dd}}'
mutate row-lineage "$g" 'Row{i, m, decl, real,' 'Row{0, m, decl, real,'
mutate undefined-id "$g" 'g.From.NativeValue() == -1' 'g.From.String() == ""'
mutate splay-id "$g" 'g.To.NativeValue() == -1' 'g.To.String() == ""'
mutate raw-pair "$g" 'row.Source.From().SameIdentity(g.From)' 'row.Source.From() == g.From'
mutate display-pair "$g" 'row.Source.From().SameIdentity(g.From)' 'row.Source.From().String() == g.From.String()'
mutate disable-reversed "$g" 'reverse := !forward &&' 'reverse := false &&'
mutate self-pair-priority "$g" 'reverse := !forward &&' 'reverse := true &&'
mutate negate-horizontal "$g" 'g.Vector.DD += v.DD' 'if reverse {v.DD=-v.DD}; g.Vector.DD += v.DD'
mutate no-average "$g" 'n := int32(len(g.members))' 'n := int32(1)'
mutate no-head-flip "$g" 'head.Source.FlagsRaw()&1 != 0' 'false'
mutate member-lineage "$g" 'Member{i, reverse}' 'Member{0, reverse}'
mutate member-reversal "$g" 'Member{i, reverse}' 'Member{i, false}'
mutate rows-alias "$g" 'return slices.Clone(r.rows)' 'return r.rows'
mutate groups-alias "$g" 'return slices.Clone(r.groups)' 'return r.groups'
mutate members-alias "$g" 'return slices.Clone(g.members)' 'return g.members'
mutate average-azimuth "$g" 'math.Atan2(float64(x), float64(y))' 'math.Atan2(float64(y), float64(x))'
mutate average-declination "$g" 'az -= float32' 'az += float32'
mutate average-reversed "$g" $'if reverse {\n\t\tx = -x' $'if false {\n\t\tx = -x'
mutate average-length "$g" 'int64(h)*int64(h)' 'int64(h)*0'
mutate average-wrap "$g" 'az += float32(math.Pi) * 2' 'az += float32(math.Pi)'
mutate json-schema "$c" 'processed{1,' 'processed{2,'
mutate json-hash "$c" 'sha256.Sum256(b)' 'sha256.Sum256(b[:len(b)/2])'
mutate json-member "$c" 'processingMember{m.RowIndex, m.Reversed}' 'processingMember{m.RowIndex+1, m.Reversed}'
mutate nonfinite-bits "$c" 'math.Float32bits(d.DistanceMM)' '0'
mutate json-output-failure "$c" 'if e = json.NewEncoder(stdout).Encode(s); e != nil {' 'if e = json.NewEncoder(stdout).Encode(s); false {'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length==34 and (map(.name)|unique|length)==34 and all(.[];.status=="KILLED")' "$report"
