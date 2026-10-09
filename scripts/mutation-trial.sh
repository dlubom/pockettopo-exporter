#!/usr/bin/env bash
# Negative control: exercised code with all behavioral assertions removed.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -W 2>/dev/null || pwd)
export GOCACHE="$root/.cache/go-build" GOMODCACHE="$root/.cache/gomod"
export GOTOOLCHAIN=local
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
mkdir -p "$trial/internal/cli" build
cp go.mod "$trial/"
cp internal/cli/run.go "$trial/internal/cli/"
mkdir "$trial/mutation-bin"
cp scripts/mutation-go.sh "$trial/mutation-bin/go"
chmod +x "$trial/mutation-bin/go"
export POCKETTOPO_MUTATION_REAL_GO
POCKETTOPO_MUTATION_REAL_GO=$(command -v go)
export PATH="$trial/mutation-bin:$PATH"
cat >"$trial/internal/cli/run_test.go" <<'GO'
package cli_test

import (
    "io"
    "testing"
    "pockettopo-exporter/internal/cli"
)

func TestWithoutAssertions(t *testing.T) {
    cli.Run([]string{"--version"}, io.Discard, io.Discard)
}
GO
rm -f build/mutation-weak.json
cd "$trial"
if ! go test -count=1 ./... >baseline.log 2>&1; then
  cat baseline.log >&2
  exit 1
fi
"$root/.tools/bin/gremlins" unleash --workers 2 --output "$root/build/mutation-weak.json"
jq -e '.mutants_total == 2 and .mutants_lived == 2 and .mutants_killed == 0
  and ([.files[].mutations[]] | length) == 2
  and all(.files[].mutations[]; .status == "LIVED")' "$root/build/mutation-weak.json"
gate_status=0
jq -e -f "$root/scripts/mutation-gate.jq" "$root/build/mutation-weak.json" || gate_status=$?
if [[ "$gate_status" != 1 ]]; then
  printf 'Expected mutation gate rejection (exit 1), got %s.\n' "$gate_status" >&2
  exit 1
fi
printf 'Negative control passed: ordinary tests passed; both mutants survived; gate exited 1.\n'
# Validate the adapter using a real type error. Its exit must be NOT VIABLE,
# rather than the KILLED result Gremlins gives an unwrapped go test exit 1.
cat >"$trial/internal/cli/invalid.go" <<'GO'
package cli
var invalid int = "not an integer"
GO
compile_status=0
go test ./internal/cli >"$root/build/mutation-compile-error.log" 2>&1 || compile_status=$?
if [[ "$compile_status" != 2 ]]; then
  printf 'Expected build-error exit 2, got %s.\n' "$compile_status" >&2
  exit 1
fi
printf 'Build-error control passed: adapter returned NOT VIABLE exit 2.\n'
setup_status=0
TMPDIR="$trial/missing-directory" go test ./internal/cli \
  >"$root/build/mutation-setup-error.log" 2>&1 || setup_status=$?
if [[ "$setup_status" != 2 ]]; then
  printf 'Expected setup-error exit 2, got %s.\n' "$setup_status" >&2
  exit 1
fi
printf 'Setup-error control passed: adapter returned NOT VIABLE exit 2.\n'
