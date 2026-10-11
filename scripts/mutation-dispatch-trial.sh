#!/usr/bin/env bash
# Check complete scheduling and error propagation without running mutations.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
mkdir -p "$trial/scripts" "$trial/.tools/bin" "$trial/cmd" "$trial/internal"
cp go.mod "$trial/"
cp scripts/mutation.sh scripts/mutation-go.sh scripts/mutation-gate.jq "$trial/scripts/"
export DISPATCH_LOG="$trial/calls.log"
printf 'gremlins\n' >"$trial/expected.log"
for script in scripts/mutation-*.sh; do
  case "$script" in
    scripts/mutation-go.sh|scripts/mutation-trial.sh|scripts/mutation-dispatch-trial.sh) continue ;;
  esac
  scope=${script##*/}
  printf '%s\n' "$scope" >>"$trial/expected.log"
  cat >"$trial/$script" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
scope=${0##*/}
printf '%s\n' "$scope" >>"$DISPATCH_LOG"
if [[ "${DISPATCH_FAIL_SCOPE:-}" == "$scope" ]]; then exit 1; fi
SH
done
cat >"$trial/.tools/bin/gremlins" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
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
printf 'Mutation dispatch controls passed: every scope exactly once; group/report/pruning errors rejected.\n'
