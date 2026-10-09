#!/usr/bin/env bash
# Boundary and negative controls for the shared statement-count gate.
set -euo pipefail
cd "$(dirname "$0")/.."
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT

expect_gate() {
  local name="$1" expected="$2" profile="$3" status=0
  printf '%s\n' "$profile" >"$trial/coverage.out"
  awk -f scripts/coverage-gate.awk "$trial/coverage.out" >"$trial/output" 2>&1 || status=$?
  if [[ "$status" != "$expected" ]]; then
    cat "$trial/output" >&2
    printf 'Coverage control %s: expected exit %s, got %s.\n' "$name" "$expected" "$status" >&2
    exit 1
  fi
}

for mode in set count atomic; do
  expect_gate "$mode-rounded-below" 1 "mode: $mode
example.go:1.1,2.1 9496 1
example.go:3.1,4.1 504 0"
  expect_gate "$mode-just-below" 1 "mode: $mode
example.go:1.1,2.1 9499 1
example.go:3.1,4.1 501 0"
  expect_gate "$mode-exactly-95" 0 "mode: $mode
example.go:1.1,2.1 19 1
example.go:3.1,4.1 1 0"
  expect_gate "$mode-above-95" 0 "mode: $mode
example.go:1.1,2.1 9501 1
example.go:3.1,4.1 499 0"
done
expect_gate full-coverage 0 'mode: atomic
example.go:1.1,2.1 1 1
example.go:3.1,4.1 0 0'
expect_gate execution-count-is-not-weight 1 'mode: count
example.go:1.1,2.1 1 1000000
example.go:3.1,4.1 1 0'
expect_gate filename-with-spaces-and-colons 0 'mode: set
C:/source files/example.go:1.1,2.1 19 1
C:/source files/example.go:3.1,4.1 1 0'
expect_gate merge-covered-block 0 'mode: atomic
example.go:1.1,2.1 19 0
example.go:1.1,2.1 19 1
example.go:3.1,4.1 1 0'
expect_gate duplicate-block-is-not-extra-coverage 1 'mode: count
example.go:1.1,2.1 10 1
example.go:1.1,2.1 10 2
example.go:3.1,4.1 1 0'
expect_gate inconsistent-duplicate 1 'mode: set
example.go:1.1,2.1 19 0
example.go:1.1,2.1 20 1'
expect_gate empty-profile 1 ''
expect_gate no-blocks 1 'mode: set'
expect_gate no-statements 1 'mode: set
example.go:1.1,2.1 0 1'
expect_gate invalid-header 1 'mode: unknown
example.go:1.1,2.1 1 1'
expect_gate invalid-count 1 'mode: set
example.go:1.1,2.1 1 invalid'
expect_gate missing-count 1 'mode: set
example.go:1.1,2.1 1'
expect_gate negative-statements 1 'mode: set
example.go:1.1,2.1 -1 1'
printf 'Coverage controls passed: exact 95%% boundary, rounding, counts and invalid profiles.\n'
