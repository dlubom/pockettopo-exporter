#!/usr/bin/env bash
# P12 native Toporobot/options/merge/lineage/CLI faults, fail closed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod" GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
cp go.mod "$trial/"
cp -R internal "$trial/"
mkdir -p "$trial/original" build
files=(internal/exporttoporobot/toporobot.go internal/exporttoporobot/model.go internal/exporttoporobot/writer.go internal/exporttoporobot/reader.go internal/processing/toporobot.go internal/cli/toporobot.go)
for file in "${files[@]}"; do cp "$file" "$trial/original/${file//\//_}"; done
report="$root/build/toporobot-mutation.json"
rm -f "$report"
cd "$trial"
go test -count=1 ./internal/exporttoporobot ./internal/processing ./internal/cli >baseline.log 2>&1 || { cat baseline.log >&2; exit 1; }
mutate() {
 local name="$1" file="$2" before="$3" after="$4" original remaining changed status=0
 for restore in "${files[@]}"; do cp "original/${restore//\//_}" "$restore"; done
 original=$(cat "$file")
 if [[ "$original" != *"$before"* ]]; then printf 'Missing mutation target: %s\n' "$name" >&2; exit 1; fi
 remaining=${original/"$before"/}
 if [[ "$remaining" == *"$before"* ]]; then printf 'Ambiguous mutation target: %s\n' "$name" >&2; exit 1; fi
 changed=${original/"$before"/"$after"}
 printf '%s\n' "$changed" >"$file"
 go build ./internal/exporttoporobot ./internal/processing ./internal/cli
 go test -json -count=1 -timeout=30s -run '^TestToporobot' ./internal/exporttoporobot ./internal/processing ./internal/cli >"$name.json" 2>&1 || status=$?
 if [[ "$status" != 1 ]] || ! jq -e -s '
  any(.[];.Action=="fail" and (.Test//""|test("^TestToporobot"))) and
  all(.[];((.Output//"")|contains("panic: test timed out"))|not)
 ' "$name.json" >/dev/null; then cat "$name.json" >&2; printf 'Fault not killed: %s\n' "$name" >&2; exit 1; fi
 jq -n --arg name "$name" '{name:$name,status:"KILLED"}' >>mutants.jsonl
 printf 'KILLED toporobot mutation: %s\n' "$name"
}
x=internal/exporttoporobot/toporobot.go
m=internal/exporttoporobot/model.go
w=internal/exporttoporobot/writer.go
r=internal/exporttoporobot/reader.go
p=internal/processing/toporobot.go
c=internal/cli/toporobot.go
mutate schema "$x" 'Schema: 1,' 'Schema: 2,'
mutate format "$x" 'Format: "pockettopo-1.372-toporobot"' 'Format: "text"'
mutate source-hash "$x" 'sha256.Sum256(d.Bytes())' 'sha256.Sum256(nil)'
mutate output-hash "$x" 'sha256.Sum256(b)' 'sha256.Sum256(nil)'
mutate size "$x" 'report.OutputSize = len(b)' 'report.OutputSize = len(b)+1'
mutate rawflags "$x" 'r.FlagsRaw(), r.RollRaw()' '0, r.RollRaw()'
mutate rawroll "$x" 'r.FlagsRaw(), r.RollRaw()' 'r.FlagsRaw(), 0'
mutate rawspan "$x" 'r.Offsets().Record, r.From().Raw()' 'source.Span{}, r.From().Raw()'
mutate trip-ticks "$x" 't.Ticks(), t.DeclinationRaw()' '0, t.DeclinationRaw()'
mutate reference-values "$x" 'r.EastMM(), r.NorthMM(), r.AltitudeMM()' 'r.NorthMM(), r.EastMM(), r.AltitudeMM()'
mutate template-hash "$x" 'sha256.Sum256(t.Bytes())' 'sha256.Sum256(nil)'
mutate existing-hash "$x" 'sha256.Sum256(opt.Existing)' 'sha256.Sum256(nil)'
mutate int32-series "$x" 'uint32(v) >> 16' 'uint32(v) >> 15'
mutate low-station "$x" 'uint32(v) & 65535' 'uint32(v) & 32767'
mutate decl-sign "$x" 'int32(decl)*-400' 'int32(decl)*400'
mutate decl-unit "$x" 'int32(decl)*-400' 'int32(decl)*-360'
mutate auto-priority "$x" 'opt.Auto || (src.AutoDeclination() && !opt.Manual)' 'opt.Auto && (src.AutoDeclination() && !opt.Manual)'
mutate code-limit "$x" 'lim = 0' 'lim = 100'
mutate mainlength "$x" 'dir.DistanceMM / 1000' 'dir.DistanceMM / 100'
mutate mainazimuth "$x" 'dir.AzimuthRadians * float32(opt.Circle) / float32(6.2831855)' 'dir.AzimuthRadians * float32(opt.Circle) / float32(6.28)'
mutate retain-zero-lrud "$x" 'if !applied {' 'if !applied && false {'
mutate inline-unit "$x" 'float32(ray.Distance) / 1000' 'float32(ray.Distance) / 100'
mutate include-lines "$x" 'if opt.ExcludeSplays {' 'if !opt.ExcludeSplays {'
mutate adjacency "$x" 's.Node.ID.NativeValue()+1' 's.Node.ID.NativeValue()+2'
mutate dims-quantization "$p" 'toRect(100, target[1])' 'toRect(1000, target[1])'
mutate dims-threshold "$p" 'best := int32(5000)' 'best := int32(10000)'
mutate dims-ties "$p" 'if dot > best {' 'if dot >= best {'
mutate dims-scale "$p" 'ray.Distance * dot / 10000' 'ray.Distance * dot / 1000'
mutate dims-key "$p" 'ray.Key, dot' '"", dot'
mutate dims-side "$p" 'az - 16384, 0' 'az + 16384, 0'
mutate forward-incl "$p" '-x.ForwardInclination' 'x.ForwardInclination'
mutate mac-supplementary "$m" 'mac[a], mac[z]' 'mac[a], byte(z)'
mutate utf16-limit "$m" 'u[:n]' 'u[:n-1]'
mutate exact-byte-limit "$w" 'len(b) > w.limit-len(w.b)' 'len(b) >= w.limit-len(w.b)'
mutate header-col "$w" 'prefix, ser, num, code, 1, trip, suffix' 'prefix, ser, num, code, 2, trip, suffix'
mutate trip-pad "$w" '14-len(units(s))' '14-len([]rune(s))'
mutate comment-budget "$w" 'budget := 254' 'budget := 255'
mutate single-precision "$w" 'p := 6 - e' 'p := 7 - e'
mutate date-range "$w" 't.year >= 1920 && t.year < 2020' 't.year >= 1920 && t.year < 2019'
mutate native-inline-space "$r" 'strings.SplitN(body[1:len(body)-1], " ", 3)' 'strings.Split(body[1:len(body)-1], " ")'
mutate read-month "$r" 't.month = integer(28, 2)' 't.month = integer(25, 2)'
mutate read-ref-east "$r" 'r.e = number(24, 12, 64)' 'r.e = number(36, 12, 64)'
mutate read-station-distance "$r" 't.dist = float32(number(24, 8, 32))' 't.dist = float32(number(32, 8, 32))'
mutate cli-circle "$c" 'opt.Circle = 400' 'opt.Circle = 360'
mutate cli-code "$c" 'opt.Code = v' 'opt.Code = 1'
mutate cli-trip "$c" 'opt.Trip = v' 'opt.Trip = 1'
mutate cli-exclude "$c" 'opt.ExcludeSplays = value == "false"' 'opt.ExcludeSplays = value == "true"'
mutate cli-context "$c" 'Name: s.Name' 'Name: "wrong"'
mutate cli-duplicate "$c" 'if seen[flag] {' 'if seen[flag] && false {'
mutate line-dedup "$m" 'math.Abs(float64(v.dist-l.dist)) < 0.1' 'math.Abs(float64(v.dist-l.dist)) < 0.01'
mutate crlf "$w" '[]string{s, "\r\n"}' '[]string{s, "\n"}'
mutate continuation-owner "$r" 'addOwner(actual, owner)
			var comment' 'addOwner(nil, owner)
			var comment'
mutate signed-digits "$w" 'digits-len(s)' 'digits-len(s)-len(sign)'
mutate null-reference "$w" 'if r.name == nil {' 'if r.name == nil && false {'
mutate terminal-omission "$x" 'record.Kind == "station" || record.Kind == "splay"' 'record.Kind == "station" || record.Kind == "splay" || record.Kind == "series"'
jq -s '.' mutants.jsonl >"$report"
jq -e 'length == 55 and all(.[]; .status=="KILLED")' "$report"
