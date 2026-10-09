#!/usr/bin/env bash
# Gremlins v0.6.0 mistakes go test's build-error exit 1 for a killed mutant.
# In the disposable PATH, require a named test failure before returning 1.
set -euo pipefail
if [[ $# == 0 || -z "${POCKETTOPO_MUTATION_REAL_GO:-}" ]]; then
  printf 'Missing mutation adapter arguments or Go executable.\n' >&2
  exit 2
fi
if [[ "$1" != test ]]; then exec "$POCKETTOPO_MUTATION_REAL_GO" "$@"; fi
log=$(mktemp) || exit 2
trap 'rm -f "$log" || true' EXIT
status=0
"$POCKETTOPO_MUTATION_REAL_GO" test -json "${@:2}" >"$log" 2>&1 || status=$?
cat "$log" || exit 2
if [[ "$status" == 0 ]]; then exit 0; fi
if [[ "$status" == 1 ]] && jq -e -s '
  any(.[]; .Action == "fail" and .Test != null) and
  all(.[]; ((.Output // "") | contains("panic: test timed out")) | not)
' "$log" >/dev/null 2>&1; then
  exit 1
fi
# Gremlins maps exit 2 to NOT VIABLE; the shared JSON gate rejects it.
exit 2
