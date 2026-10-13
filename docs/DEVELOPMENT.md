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
GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/check.sh # bounded local ordinary gates
bash scripts/mutation.sh                 # sequential full campaign
bash scripts/mutations-parallel.sh       # full local campaign; default 3 private workers
POCKETTOPO_MUTATION_JOBS=2 bash scripts/mutations-parallel.sh
bash scripts/mutation-dispatch-trial.sh
bash scripts/mutation-trial.sh

# On a space-constrained local disk, discard only regenerable Go build cache
# after each completed campaign; all reports/gates and CI groups are unchanged.
POCKETTOPO_MUTATION_PRUNE_CACHE=true bash scripts/mutation.sh

bash scripts/mutation.sh --matrix      # authoritative list of current CI groups
bash scripts/mutation.sh plan-elements # example focused campaign

gofmt -w cmd internal
go run ./cmd/pockettopo-exporter --help
go run ./cmd/pockettopo-exporter --version
go run ./cmd/pockettopo-exporter inspect internal/top/testdata/api-drawings.top --json
go run ./cmd/pockettopo-exporter export internal/top/testdata/api-drawings.top --format text --output build/drawings-text
go run ./cmd/pockettopo-exporter geometry internal/processing/testdata/geometry/intersecting.top --json --close-loops
go run ./cmd/pockettopo-exporter context internal/processing/testdata/context/crossfile/current.top --template-directory internal/processing/testdata/context/crossfile --json --close-loops
go run ./cmd/pockettopo-exporter export internal/exporttherion/testdata/features.top --format therion --output build/therion
go test ./internal/top -run '^$' -bench '^BenchmarkDocument' -benchmem
```

`check.sh` checks P14 validation negative controls and frozen evidence hashes,
formatting, vet, Staticcheck, coverage-boundary controls,
uncached race tests, coverage, build and CLI smoke behavior. Generated outputs
are `coverage.out`, `mutation.json` and `build/`; caches/tools are in ignored
`.cache/` and `.tools/`. There is no `go.sum` because there are no module
dependencies. The document group covers P04d source/accounting/inspect faults alongside the
unchanged earlier groups. The native-text group covers text, Therion, graphics DXF, VisualTopo, Toporobot and 3D DXF serialization, decimal
formatting, provenance and safe CLI publication. The weak-suite negative control deliberately stubs
inspection/export/processing/geometry/context and exercises version, inspect, process, geometry, context and text/Therion/DXF/VisualTopo/Toporobot/3D DXF export arguments without assertions;
all 58 generated mutants must live and the quality gate must reject them.
Run one sequential campaign or one parallel coordinator at a time. The parallel
runner snapshots only source/scripts/pinned tools into a private checkout per
group with private caches/reports. It limits active groups, collects reports and
runtimes in `build/mutation-parallel/`, fills a free worker immediately, stops
scheduling when a child failure is observed and waits for active children before
cleanup. It prunes each private cache after
completed campaigns by default; set `POCKETTOPO_MUTATION_PRUNE_CACHE=false` only
when disk space permits. This runner never changes Git branches/worktrees.
CI parallelism uses separate checkouts and caches.

## Workstation and hosted execution

Prefer bounded ordinary checks locally (`GOMAXPROCS=2 GOFLAGS=-p=1`) and the
existing GitHub Actions mutation matrix for heavy campaigns. This delivery follows
the 2026-10-12 request after a workstation freeze: no further local full mutation
runs are required. Retain the complete matrix, all targeted scopes, thresholds,
negative controls and aggregate gate. Before reporting publication complete,
verify every job for the exact pushed SHA; successful older local/hosted results
are not substitutes for final fixture/test changes. The local mutation commands
remain available for explicitly requested diagnosis.

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

CI runs ordinary checks and native-platform distribution verification on Linux
amd64, Windows amd64 and macOS arm64. A Linux corpus job checks every pinned input
and all 25 export profiles twice. Linux
mutation jobs come directly from `mutation.sh --matrix`. **CI complete** requires
all platform/distribution checks, corpus validation, the scheduling controls and
every mutation group to pass.
When configuring required checks, use this aggregate gate; a skipped/canceled
group is not a successful delivery. Do not claim branch protection is configured
merely because the workflow defines this job.

Keep parallel mutation scheduling. Group count is an execution detail, not the
number of PBIs or a target to preserve. Rebalance measured runtimes when needed;
do not automatically add one group for every field/task. Such a change must keep
the same complete scope, thresholds and controls.

### Mutation scheduling after P06

The previous P05 CI ran 11 mutation groups; the slowest `markers` group took
about 16 minutes and Gremlins about 13 minutes. P06 introduced 22 groups; P12 adds one Gremlins shard for its new exporter/CLI: expensive
second/third Polygon scopes are separated, mappings and marker pairs stay
cohesive, and processing has targeted P06, P07 and P08 campaigns in the same group. The unchanged prior 25 explicit
scopes plus P06 processing, P07 geometry and P08 context remain exactly-once; all prior faults and thresholds
are retained. Gremlins is partitioned into tables, drawings, output, Toporobot and processing
by source file, with all internal production files assigned to one shard (Therion, DXF and VisualTopo join output) and
unknown new packages rejected. P10 adds the dxf explicit scope to native-text; no earlier scope is removed. P11 adds VisualTopo and passage dimensions to output/processing Gremlins and the `visualtopo` explicit scope to native-text; P12 adds `toporobot` with 55 explicit faults and a separate Gremlins shard for `internal/exporttoporobot` and `internal/cli/toporobot.go`. All previous scopes remain. Every shard still discovers coverage with the
full suite, uses all prior operators, the fail-closed adapter and the same >=90%
JSON gate. Full/sharded mutation inventories are compared during delivery.
The scheduling controls check serial/matrix/parallel scope equality, private
source snapshots, report collection, real 2..3 overlap within a 3-worker budget,
child failures, invalid budgets, unknown packages and unchanged pruning controls.
CI's aggregate still requires every shard/group and every platform check.

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


P13 adds the `dxf3d` explicit scope to `native-text` and assigns
`internal/exportdxf3d` to the existing output Gremlins shard. It retains every
prior scope, all five shards, operators, >=90% gate and failure controls.
`GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/mutation-dxf3d.sh --preflight` checks that
all new faults compile without running a local mutation campaign or producing
a KILLED report. The actual full campaign and negative control run in Actions.
The native weak-suite control now requires exactly 58 live mutants.

## P14 corpus and offline distributions

P14 changes validation and distribution tooling; application Go, previous
contracts and every prior mutation scope/operator/threshold remain unchanged.
Hosted survivor review also adds fixed reverse-angle provenance and singleton
sort work-boundary regressions. They assert native Int32 values and concrete
acceptance ceilings; byte equality alone misses those mutant behaviors.
The standard-library Python scripts require Python 3.9+ only for development
and CI. The distributed executable requires no Python, Go, Wine or PocketTopo.

```sh
python3 scripts/test_validation.py
python3 scripts/verify-evidence.py
# Keep the original pinned corpus outside tracked runtime sources.
mkdir -p .cache/p14
curl --fail --location --silent --show-error https://codeload.github.com/dlubom/test2/tar.gz/4c00c008a8dd441d70f9c62aa376c20b5914c7e0 -o .cache/p14/test2.tar.gz
tar -xzf .cache/p14/test2.tar.gz -C .cache/p14
python3 scripts/validate-corpus.py --binary build/pockettopo-exporter --corpus .cache/p14/test2-4c00c008a8dd441d70f9c62aa376c20b5914c7e0 --report build/corpus-report.json
# Requires a clean committed checkout and a new output directory.
GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local GOMAXPROCS=2 GOFLAGS=-p=1 python3 scripts/distribution.py --output build/distribution
```

Corpus acquisition requires network access; validation runs offline after input
identity checks. Actions fetches only TOP members from the pinned archive.
`--record` is a deliberate baseline-authoring switch, never a CI repair or gate
option. Review changed results against source evidence before replacing a
baseline. Every input path remains distinct, including identical hashes.
All corpus text outputs compare against separate original-program goldens;
other corpus results are deterministic project regressions, not native truth.

ZIP artifacts and corpus reports are uploaded by the exact-SHA CI run.
Distribution validation builds its own binary, verifies VCS identity and payload
readback, extracts to a temporary unrelated directory, clears PATH and checks
six exact native byte cases, no-replace behavior and malformed input failures.
The package BUILD.json distinguishes the source SHA while preserving existing
`--version` and report schemas. No tag or GitHub Release is created by this gate.
CI complete explicitly requires the new corpus job and all three platform
checks (each containing distribution verification), alongside the unchanged
mutation matrix and failure controls.

## P15 Walls verification

P15 adds `internal/exportwalls` and `internal/cli/walls.go` to the existing output
Gremlins shard and a `walls` scope with **43 explicit faults** to `native-text`.
All previous scopes/operators/thresholds remain. The exercised weak-suite control
now has **62 mutants**, including Walls dispatch; all must survive and the gate
must reject them. Actions uploads every group's mutation reports for survivor
review. Dispatch controls still require exactly-once full-scope scheduling.

The new required `walls-consumer` Actions job builds a SHA-256-pinned Survex 1.4.22
source archive (development-only) and checks its Walls reader against independent
analytic coordinates and a malformed-input control. `CI complete` includes this
job. The three distribution jobs retain all six native byte cases and additionally
verify the Walls conversion golden, provenance, no-replace and malformed-input
behavior with an empty PATH. P14's 262-input / 25-profile baseline stays unchanged.

```sh
# Uses existing cavern/dump3d 1.4.22, no external tools needed by the exporter:
python3 scripts/validate-walls.py --binary build/pockettopo-exporter --report build/walls-consumer-report.json
# Linux development/CI installation, needs libproj-dev, libgdal-dev, libwxgtk3.2-dev:
bash scripts/install-walls-consumer.sh
```

The consumer deliberately allows exactly the two documented warnings: retained
compass on a plumb shot, and Survex ignoring `Grid=0` in an unreferenced standalone
context. Other errors/warnings fail the controlled consumer fixture. It is a
format/coordinate acceptance test, not a claim about Walls GUI or native P06
averaging. Regenerate P15 literal inputs/expected SRV only through deliberate
review of `scripts/walls-fixtures.py`; never update goldens during a gate.


## P16 Survex verification

P16 adds `internal/exportsurvex` and `internal/cli/survex.go` to the existing
output Gremlins shard, and `survex` with **43 explicit faults** to `native-text`.
All prior campaigns, operators and thresholds remain unchanged. The weak-suite
control now exercises Survex dispatch and requires exactly **66 live mutants**;
compiler/setup errors, timeouts and incomplete results still fail closed.
`mutation-survex.sh --preflight` checks literal targets and compiles all faults;
it does not execute a local mutation campaign or claim KILLED outcomes.

The required `walls-consumer` Actions job retains P15 validation and additionally
runs `validate-survex.py` against the same SHA-256-pinned Survex 1.4.22 build.
`check.sh` additionally runs independent Python regression controls for complete
anonymous endpoint decoding and per-endpoint flags. Its three literal SVX cases check numeric ENU fixes, signed declination, repeated
and reverse observations, consumer adjustment, independently anonymous non-wall
splays, zero equations and malformed-distance rejection. Only the redundant
self-equation case permits one specifically identified warning. The report is
uploaded with P15's report and the aggregate gate requires the combined job.

```sh
python3 scripts/validate-survex.py --binary build/pockettopo-exporter --report build/survex-consumer-report.json
GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/mutation-survex.sh --preflight
```

The exporter/distributed binary needs no Survex installation. All three platform
distribution jobs additionally exercise the P16 conversion golden, provenance,
no-replace and malformed-input contracts with empty PATH. The six native golden
cases, P15 offline case and P14's 262-input/25-profile baseline stay unchanged.
`p16-evidence.json` pins the eight new fixture/authoring identities; gates never
regenerate expected bytes.
