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
go test -count=1 ./...
status=0
"$root/.tools/bin/gremlins" unleash --workers 2 --threshold-efficacy 90 \
  --threshold-mcover 100 --output "$root/build/mutation-weak.json" || status=$?
[[ "$status" == 10 ]]
jq -e '.mutants_total == 2 and .mutants_lived == 2 and .mutants_killed == 0
  and all(.files[].mutations[]; .status == "LIVED")' "$root/build/mutation-weak.json"
printf 'Negative control passed: ordinary tests passed; both mutants survived; gate exited 10.\n'
