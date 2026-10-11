#!/usr/bin/env bash
# Bounded local full campaign, with private source/report/cache paths per group.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
jobs=${POCKETTOPO_MUTATION_JOBS:-3}
if [[ $# != 0 || ! "$jobs" =~ ^[1-9][0-9]?$ ]] || (( jobs > 32 )); then
  printf 'Expected no arguments and POCKETTOPO_MUTATION_JOBS=1..32.\n' >&2; exit 2
fi
matrix=$(bash scripts/mutation.sh --matrix)
groups=()
while IFS= read -r group; do groups+=("$group"); done < <(printf '%s\n' "$matrix" | jq -r '.group[]')
work=$(mktemp -d)
pids=()
running=()
cleanup() {
  # Never delete an active compiler's directory (also on interruption/error).
  if (( ${#pids[@]} > 0 )); then
    for pid in "${pids[@]}"; do wait "$pid" 2>/dev/null || true; done
  fi
  rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
reports="$root/build/mutation-parallel"
rm -rf "$reports"
mkdir -p "$reports"
run() (
  selected="$1"
  trap 'touch "$work/$selected.done"' EXIT
  checkout="$work/$selected"
  mkdir -p "$checkout" "$reports/$selected"
  cp go.mod "$checkout/"
  cp -R cmd internal scripts .tools "$checkout/"
  cd "$checkout"
  start=$SECONDS
  status=0
  POCKETTOPO_MUTATION_PRUNE_CACHE=${POCKETTOPO_MUTATION_PRUNE_CACHE:-true} bash scripts/mutation.sh "$selected" >"$reports/$selected/run.log" 2>&1 || status=$?
  printf '%s %ss exit=%s\n' "$selected" "$((SECONDS-start))" "$status" >"$reports/$selected/timing.txt"
  for report in mutation.json build/*-mutation.json; do
    if [[ -f "$report" ]]; then cp "$report" "$reports/$selected/"; fi
  done
  cat "$reports/$selected/timing.txt"
  if [[ "$status" != 0 ]]; then tail -n 30 "$reports/$selected/run.log" >&2; fi
  rm -rf "$checkout"
  exit "$status"
)
failed=0
reap() {
  local i pid
  local pending=() pending_groups=()
  for i in "${!pids[@]}"; do
    pid=${pids[$i]}
    if [[ -e "$work/${running[$i]}.done" ]] || ! kill -0 "$pid" 2>/dev/null; then
      wait "$pid" || failed=1
    else
      pending+=("$pid"); pending_groups+=("${running[$i]}")
    fi
  done
  if (( ${#pending[@]} > 0 )); then
    pids=("${pending[@]}"); running=("${pending_groups[@]}")
  else
    pids=(); running=()
  fi
}
for group in "${groups[@]}"; do
  while (( ${#pids[@]} >= jobs )); do
    reap
    if [[ "$failed" != 0 ]]; then exit 1; fi
    if (( ${#pids[@]} >= jobs )); then sleep 0.1; fi
  done
  run "$group" & pids+=("$!"); running+=("$group")
done
while (( ${#pids[@]} > 0 )); do
  reap
  if [[ "$failed" != 0 ]]; then exit 1; fi
  if (( ${#pids[@]} > 0 )); then sleep 0.1; fi
done
printf 'Full parallel campaign passed: %s groups, at most %s concurrent groups.\n' "${#groups[@]}" "$jobs"
