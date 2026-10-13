#!/usr/bin/env bash
# Check complete scheduling and error propagation without running mutations.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
mkdir -p "$trial/scripts" "$trial/.tools/bin" "$trial/cmd" "$trial/internal"
printf "expected source\n" > "$trial/internal/snapshot"
mkdir -p "$trial/internal/cli" "$trial/internal/source" "$trial/internal/top" "$trial/internal/processing"
for file in cli/run source/station_id top/document processing/grouped; do printf 'package stub\n' >"$trial/internal/$file.go"; done
cp go.mod "$trial/"
cp scripts/mutation.sh scripts/mutation-go.sh scripts/mutation-gate.jq scripts/mutations-parallel.sh "$trial/scripts/"
export DISPATCH_LOG="$trial/calls.log"
export DISPATCH_ORIGINAL_ROOT="$trial" DISPATCH_STATE="$trial/concurrency"
cat >"$trial/scripts/dispatch-record.sh" <<'SH'
#!/usr/bin/env bash
if [[ "${DISPATCH_PARALLEL_CHECK:-false}" == true ]]; then
 if [[ "$PWD" == "$DISPATCH_ORIGINAL_ROOT" ]] || ! cmp -s internal/snapshot "$DISPATCH_ORIGINAL_ROOT/internal/snapshot"; then
  printf 'Parallel worker did not isolate/preserve its source.\n' >&2;exit 1
 fi
 lock() { until mkdir "$DISPATCH_STATE.lock" 2>/dev/null; do sleep 0.01; done; }
 lock
 read -r active peak <"$DISPATCH_STATE"
 active=$((active+1)); if (( active>peak )); then peak=$active; fi
 printf '%s %s\n' "$active" "$peak" >"$DISPATCH_STATE"
 rmdir "$DISPATCH_STATE.lock"
 release() { lock; read -r active peak <"$DISPATCH_STATE"; printf '%s %s\n' "$((active-1))" "$peak" >"$DISPATCH_STATE"; rmdir "$DISPATCH_STATE.lock"; }
 trap release EXIT
 sleep 0.05
fi
SH
printf 'gremlins\ngremlins\ngremlins\ngremlins\ngremlins\n' >"$trial/expected.log"
for script in scripts/mutation-*.sh; do
  case "$script" in
    scripts/mutation-go.sh|scripts/mutation-trial.sh|scripts/mutation-dispatch-trial.sh) continue ;;
  esac
  scope=${script##*/}
  printf '%s\n' "$scope" >>"$trial/expected.log"
  cat >"$trial/$script" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
source scripts/dispatch-record.sh
scope=${0##*/}
printf '%s\n' "$scope" >>"$DISPATCH_LOG"
mkdir -p build
printf '[{"name":"stub","status":"KILLED"}]' >"build/$scope-mutation.json"
if [[ "${DISPATCH_FAIL_SCOPE:-}" == "$scope" ]]; then exit 1; fi
SH
done
cat >"$trial/.tools/bin/gremlins" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
source "$DISPATCH_ORIGINAL_ROOT/scripts/dispatch-record.sh"
printf 'gremlins\n' >>"$DISPATCH_LOG"
if [[ "${DISPATCH_FAIL_SCOPE:-}" == gremlins ]]; then exit 1; fi
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --output ]]; then
    if [[ "${DISPATCH_BAD_REPORT:-}" == true ]]; then
      printf '{"mutants_total":0,"files":[]}' >"$2"
    else
      printf '{"mutants_total":1,"files":[{"mutations":[{"status":"KILLED"}]}]}' >"$2"
    fi
    exit 0
  fi
  shift
done
exit 2
SH
chmod +x "$trial/.tools/bin/gremlins"

matrix=$(bash scripts/mutation.sh --matrix)
printf '%s\n' "$matrix" | jq -e '
  .group | length > 0 and length == (unique | length) and
  all(.[]; type == "string" and . != "all" and . != "--matrix")
' >/dev/null
groups=()
while IFS= read -r group; do groups+=("$group"); done < <(printf '%s\n' "$matrix" | jq -r '.group[]')
sort "$trial/expected.log" >"$trial/expected-sorted.log"

cd "$trial"
bash scripts/mutation.sh >/dev/null
sort "$DISPATCH_LOG" >actual.log
cmp expected-sorted.log actual.log
cp "$DISPATCH_LOG" full.log
rm "$DISPATCH_LOG"
for group in "${groups[@]}"; do bash scripts/mutation.sh "$group" >/dev/null; done
cmp full.log "$DISPATCH_LOG"

# Every group must propagate a failing campaign, without running later scopes.
for group in "${groups[@]}"; do
  rm "$DISPATCH_LOG"
  bash scripts/mutation.sh "$group" >/dev/null
  first=$(head -n 1 "$DISPATCH_LOG")
  rm "$DISPATCH_LOG"
  if DISPATCH_FAIL_SCOPE="$first" bash scripts/mutation.sh "$group" >failure.log 2>&1; then
    printf 'Mutation group swallowed failure: %s\n' "$group" >&2
    exit 1
  fi
  if [[ $(wc -l <"$DISPATCH_LOG") -ne 1 ]]; then
    printf 'Mutation group continued after failure: %s\n' "$group" >&2
    exit 1
  fi
done
rm "$DISPATCH_LOG"
if DISPATCH_BAD_REPORT=true bash scripts/mutation.sh >failure.log 2>&1; then
  printf 'Empty Gremlins report unexpectedly passed.\n' >&2
  exit 1
fi
if [[ $(cat "$DISPATCH_LOG") != gremlins ]]; then
  printf 'Full campaign continued after rejected Gremlins report.\n' >&2
  exit 1
fi
# Optional disk pruning must retain identical scheduling and target only this
# disposable checkout's regenerable cache. Cleanup failures must propagate.
mkdir -p fake-bin
cat >fake-bin/go <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" != 'clean -cache' || "$GOCACHE" != "$DISPATCH_PRUNE_ROOT/.cache/go-build" ]]; then exit 2; fi
printf 'prune\n' >>"$DISPATCH_PRUNE_LOG"
if [[ "${DISPATCH_FAIL_PRUNE:-false}" == true ]]; then exit 1; fi
SH
chmod +x fake-bin/go
export DISPATCH_PRUNE_ROOT
DISPATCH_PRUNE_ROOT=$(pwd -W 2>/dev/null || pwd)
export DISPATCH_PRUNE_LOG="$trial/prunes.log"
rm "$DISPATCH_LOG"
PATH="$trial/fake-bin:$PATH" POCKETTOPO_MUTATION_PRUNE_CACHE=true bash scripts/mutation.sh >/dev/null
cmp full.log "$DISPATCH_LOG"
if [[ $(wc -l <"$DISPATCH_PRUNE_LOG") -ne $(wc -l <full.log) ]]; then
  printf 'Optional pruning missed or duplicated a completed campaign.\n' >&2
  exit 1
fi
rm "$DISPATCH_LOG"
if PATH="$trial/fake-bin:$PATH" POCKETTOPO_MUTATION_PRUNE_CACHE=true DISPATCH_FAIL_PRUNE=true bash scripts/mutation.sh >failure.log 2>&1; then
  printf 'Optional cache cleanup failure was swallowed.\n' >&2
  exit 1
fi
if [[ $(cat "$DISPATCH_LOG") != gremlins ]]; then
  printf 'Campaign continued after cache cleanup failure.\n' >&2
  exit 1
fi
rm "$DISPATCH_LOG"
for args in unknown-group 'tables extra-argument'; do
  # Deliberate splitting supplies one unknown argument or two arguments.
  if bash scripts/mutation.sh $args >failure.log 2>&1; then
    printf 'Invalid mutation arguments unexpectedly passed: %s\n' "$args" >&2
    exit 1
  fi
done
if [[ -e "$DISPATCH_LOG" ]]; then
  printf 'Invalid mutation arguments ran a campaign.\n' >&2
  exit 1
fi

# A bounded parallel full run must visit the identical scope multiset in private
# roots, collect every report and propagate a failed child. SECONDS is not an
# assertion; the stub uses a lock to measure concurrency and private roots.
rm -f "$DISPATCH_LOG"
printf '0 0\n' >"$DISPATCH_STATE"
DISPATCH_PARALLEL_CHECK=true POCKETTOPO_MUTATION_JOBS=3 POCKETTOPO_MUTATION_PRUNE_CACHE=false bash scripts/mutations-parallel.sh >/dev/null
read -r active peak <"$DISPATCH_STATE"
if (( active!=0 || peak<2 || peak>3 )); then printf 'Parallel worker budget violated: active=%s peak=%s\n' "$active" "$peak" >&2;exit 1;fi
reports=0
for report in build/mutation-parallel/*/*mutation.json; do if [[ -f "$report" ]]; then reports=$((reports+1)); fi; done
if [[ "$reports" != "$(wc -l <full.log | tr -d ' ')" ]]; then printf 'Parallel runner missed reports.\n' >&2;exit 1;fi
sort "$DISPATCH_LOG" > actual.log
cmp expected-sorted.log actual.log
rm "$DISPATCH_LOG"
if POCKETTOPO_MUTATION_JOBS=2 POCKETTOPO_MUTATION_PRUNE_CACHE=false DISPATCH_FAIL_SCOPE=mutation-document.sh bash scripts/mutations-parallel.sh >failure.log 2>&1; then
 printf 'Parallel mutation runner swallowed a child failure.\n' >&2;exit 1
fi
# New internal packages cannot silently disappear from Gremlins discovery.
mkdir -p internal/unassigned
printf 'package unassigned\n' >internal/unassigned/new.go
if bash scripts/mutation.sh gremlins-tables >failure.log 2>&1; then printf 'Unassigned Gremlins package accepted.\n' >&2;exit 1;fi
rm -rf internal/unassigned
for jobs in 0 -1 nope 1.5 33; do
 if POCKETTOPO_MUTATION_JOBS="$jobs" bash scripts/mutations-parallel.sh >failure.log 2>&1; then
  printf 'Invalid parallel worker budget accepted: %s\n' "$jobs" >&2;exit 1
 fi
done
printf 'Mutation dispatch controls passed: every scope exactly once; group/report/pruning errors rejected.\n'
