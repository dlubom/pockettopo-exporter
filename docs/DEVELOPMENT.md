# Development and delivery

## Tools and actual commands

Use Go **1.26.3**, pinned by `go.mod`, Bash (Git Bash on Windows), and jq 1.6+
for mutation scripts. `scripts/tools.sh` installs Staticcheck **v0.8.1** and
Gremlins **v0.6.0** in `.tools/bin`. Scripts set `GOTOOLCHAIN=local`; install Go
first. Tool installation requires network access. The runtime uses only the
standard library. PocketTopo/Wine/.NET are optional native-evidence tools.

From the repository root:

```sh
bash scripts/tools.sh
bash scripts/check.sh
bash scripts/mutation.sh
bash scripts/mutation-dispatch-trial.sh
bash scripts/mutation-trial.sh

bash scripts/mutation.sh --matrix      # authoritative list of current CI groups
bash scripts/mutation.sh plan-elements # example focused campaign

gofmt -w cmd internal
go run ./cmd/pockettopo-exporter --help
go run ./cmd/pockettopo-exporter --version
```

`check.sh` checks formatting, vet, Staticcheck, coverage-boundary controls,
uncached race tests, coverage, build and CLI smoke behavior. Generated outputs
are `coverage.out`, `mutation.json` and `build/`; caches/tools are in ignored
`.cache/` and `.tools/`. There is no `go.sum` because there are no module
dependencies. Run one local mutation campaign at a time: reports/cache paths
are shared. CI parallelism uses separate checkouts and caches.

## Quality gates

The scripts and [CI workflow](../.github/workflows/ci.yml) are authoritative.

- Statement coverage is at least **95%** in the implemented internal packages.
  `coverage-gate.awk` checks exact counts, not rounded displayed percentages.
  Process-entry wiring is outside the coverage scope.
- Gremlins requires at least **90% killed**, a complete nonempty report, and
  only accepted `KILLED`/`LIVED` outcomes. Review every survivor; fix critical ones.
- Explicit mutation campaigns require **every selected fault** to compile and
  fail a named behavioral test. Compiler/setup failures, timeouts, empty or
  incomplete reports are failures, not killed mutants.
- The mutation adapter and negative controls are part of the gate. Retain
  every existing campaign when changing orchestration; dispatch controls require
  exactly-once scheduling and failure propagation.

CI runs ordinary checks on Linux amd64, Windows amd64 and macOS arm64. Linux
mutation jobs come directly from `mutation.sh --matrix`. **CI complete** requires
all platform checks, the scheduling controls and every mutation group to pass.
When configuring required checks, use this aggregate gate; a skipped/canceled
group is not a successful delivery. Do not claim branch protection is configured
merely because the workflow defines this job.

Keep parallel mutation scheduling. Group count is an execution detail, not the
number of PBIs or a target to preserve. Rebalance measured runtimes when needed;
do not automatically add one group for every field/task. Such a change must keep
the same complete scope, thresholds and controls.

## One complete behavior per increment

Use a lightweight ASDLC workflow: durable contracts in [SPEC](SPEC.md), one live
backlog in [PLAN](PLAN.md), normal Go tests and reviewable changes. No additional
tracker, Cucumber runner or process framework is required. This applies
[Living Specs](https://asdlc.io/practices/living-specs) and
[PBI Authoring](https://asdlc.io/practices/pbi-authoring) at this project's scale.

1. Read the current plan and verify the checkout. State the requested outcome,
   its boundaries and the relevant native references before implementation.
2. Define observable acceptance cases for a complete behavior. Research the
   native details needed by this slice; do not split known record fields into
   separate chats. If new semantics make the slice too large, split by behavior.
3. Implement and test locally. Use focused tests while iterating. Preserve raw
   source evidence and use independent expectations, not Go output as its oracle.
4. For parser/processing/export or mutation-tooling changes, run `check.sh`, the
   full mutation campaign and applicable negative/dispatch controls on the final
   local tree. Rerun checks invalidated by subsequent changes. Do not weaken
   gates to resolve slowness, disk exhaustion or runner failures.
5. For documentation-only edits, verify factual claims, local links, structure
   and `git diff --check`. A full mutation rerun is unnecessary when code, tests,
   fixtures and tooling are unchanged. State exactly which checks were run.
6. Review the change independently when it affects behavior or a substantial
   contract. Update the current spec/plan in place, with a concise evidence link.
7. When commit/push is authorized, work directly on `main`. Before pushing,
   check the remote relationship and preserve others' changes. Verify all CI
   jobs for the **exact pushed SHA**. A green older SHA is not evidence for a
   changed implementation. Do not create branches/worktrees/PRs by default.
8. Finish with behavior delivered, local checks, publication/integration state,
   remaining limits and one next useful item. Stop at the requested boundary.

Creating a plan is not authorization to execute all future items. Follow the
current user request; do not require another approval for already authorized,
necessary work. A merge, push or release should never be implied by the word
"complete" in a local test report.

## Evidence placement

The selected real corpus is [dlubom/test2](https://github.com/dlubom/test2/tree/4c00c008a8dd441d70f9c62aa376c20b5914c7e0),
pinned at `4c00c008a8dd441d70f9c62aa376c20b5914c7e0`. Its earlier inventory is
not a completed parser or compatibility run. Pin revision and input hashes,
preserve duplicate paths and their directory context, and account for every
input as supported, unsupported or failed. Neighboring SRV/SVX/DXF files are
comparison material, not authenticated native truth. Control isolated versus
template-directory inputs explicitly in native captures. Keep small independent
fixtures for boundary behavior; large corpus coverage does not replace them.
Classify versions/features before sampling, retain the original corpus externally,
and validate synthetic reference cases with the native reader. Use property/fuzz
checks for malformed-input bounds and determinism alongside exact native goldens;
destination imports and rendered drawing reviews supplement structural checks.

Keep fixture hashes, attribution, original stdout and actual native commands in
[`internal/top/testdata/README.md`](../internal/top/testdata/README.md). Preserve
raw CRLF/native bytes and `.gitattributes` rules. Research code stays in `scripts/`
or ignored local outputs, never in the sibling decompilation artifacts.

Keep status and the next PBI in PLAN. README is an entry point, not an append-only
session journal. Git commits and CI runs retain delivery history. Record a newly
discovered durable contract in SPEC, not in several competing handoff sections.
