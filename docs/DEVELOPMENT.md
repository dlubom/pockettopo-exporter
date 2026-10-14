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
all 66 generated mutants must live and the quality gate must reject them.
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

CI keeps full ordinary checks on Linux amd64, Windows amd64 and macOS arm64,
and runs native ordinary tests/versioned distribution verification on all six
Linux/Windows/macOS amd64/arm64 targets. Windows release jobs also exercise the
optional Explorer menu. A Linux corpus job checks every pinned input
and all 25 export profiles twice. Linux
mutation jobs come directly from `mutation.sh --matrix`. **CI complete** requires
all platform/distribution/release checks, corpus validation, the scheduling controls and
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
The native weak-suite control requires exactly 66 live mutants after P16.

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
report schemas. Ordinary packages retain `dev`; `--version 1.0.0` injects the
release version and validates it in help/version/inspection/build metadata.
No tag or GitHub Release is created by this builder.
CI complete explicitly requires the new corpus job and all three platform
checks (each containing distribution verification), alongside the unchanged
mutation matrix and failure controls.

## Release 1.0.0

The accepted release uses the current Go toolchain and six native targets.
`CGO_ENABLED=0` archives require no development runtime. Windows menu registration
is per user and its acceptance gate must run on native Windows. Keep existing
race checks on the three supported full-check runners; ARM Windows native tests
run without the unsupported race detector.

```sh
python3 scripts/test_validation.py
python3 scripts/test_assemble_release.py
python3 scripts/test_release_gate.py
GOTOOLCHAIN=local GOMAXPROCS=2 GOFLAGS=-p=1 python3 scripts/distribution.py --version 1.0.0 --output build/release-package
```

Distribution builds require clean committed sources. CI uploads
`release-<target>-<SHA>` for each target; the assembler accepts only complete,
same-version/same-SHA native ZIP/checksum/validation sets. Existing dev artifacts
are not release inputs. A native Windows release job extracts its tested package
and runs `validate-windows-menu.ps1` against that executable.

Publication is explicitly dispatched with an existing annotated `v1.0.0` and
the successful **main push CI run ID for that exact revision**:

```sh
gh workflow run release.yml -f tag=v1.0.0 -f ci_run_id=<verified-run-id>
```

The workflow checks every job, including the dynamic mutation inventory and six
native release jobs; it rejects skipped/canceled/foreign/incomplete evidence.
It verifies the tag, downloads that run's archives, assembles all six packages,
creates a complete draft, downloads its assets and compares exact membership and
hashes, then publishes as latest. Failure leaves publication incomplete; never
replace a published version's artifacts or force-move its tag. Publication records
are uploaded as `release-publication-v1.0.0`. No signing credentials are used;
packages have no Windows publisher signature or Apple Developer ID/notarization.

## P15 original Walls validation

P15 keeps every P05–P14 scope, operator, threshold and failure control. The
`walls` explicit campaign adds 58 faults to `native-text`; `exportwalls` is in
the existing output Gremlins shard. At P15 delivery, the native weak-suite required exact
cardinality: 62 live mutants; P16 extends that control to 66 below. Setup/compiler/timeouts/incomplete outcomes never
count as kills. Native compile-only preflight is available as
`GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/mutation-walls.sh --preflight`; it does not
produce a mutation score. Prefer the full Actions campaign after local disk or
resource pressure. Never aggregate incomplete preflight output as a passed run.

```sh
GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/check.sh
python3 scripts/test_walls_validation.py
bash scripts/mutation-dispatch-trial.sh
# Original application, on an isolated Windows runner with the pinned runtime:
# pwsh scripts/validate-walls-native.ps1 -Binary build/pockettopo-exporter.exe -Walls <Walls32.exe>
```

`walls-native` is a required Windows Actions job. It administratively extracts
(no installation changes to the user's Mac) official Walls v2.3.0-beta.1 from
MSI SHA-256 `11445a522760ef98876c3d025e90b734c2073d8a033717c7313a8da130168ce0`.
The owned runner process follows `ID_PRJ_COMPILEITEM`, then closes normally.
Validate exact native NTV layout/version, complete DBF headers, source-line/name
identity, unadjusted ENU vectors, final NTN survey count/length and an actual
invalid-inclination rejection. Merely starting Walls, creating a workfile or
reading SRV in Survex does not satisfy this gate. `test_walls_validation.py`
rejects malformed schema/header/date/fix/nonfinite/incomplete/trailing cases.
Native projects/workfiles/window evidence and `report.json` are uploaded as
`original-walls-<SHA>`; every mutation job also uploads its raw report for survivor
review. CI complete requires this native job in addition to all existing jobs.

Offline distribution validation adds one SRV package regression through a separate
P15 manifest. The six frozen P14 native byte cases and their hashes are unchanged.
The existing 262-input/25-profile native corpus baseline remains unchanged; P15
uses small independently read semantic cases and the original destination gate.

## P16 Survex validation

P16 preserves every P05–P15 scope/operator/threshold/control. Its 56-fault `svx`
campaign joins `native-text`; `exportsvx` joins the existing output Gremlins
shard. The measured weak inventory is 66 live sites: all 62 earlier sites plus
exactly four SVX dispatch sites. Compilation/setup/timeout/incomplete outcomes
fail closed. A compile-only preflight is available; it is not a mutation score.
Full mutations remain hosted, with bounded local ordinary checks.

```sh
GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/check.sh
GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/mutation-svx.sh --preflight
GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/mutation-trial.sh
bash scripts/mutation-dispatch-trial.sh
python3 scripts/validate-survex.py --binary build/pockettopo-exporter --output build/survex-native
python3 scripts/validate-survex.py --binary build/pockettopo-exporter --consumer-version 1.4.23 --output build/survex-1.4.23
```

`validate-survex.py` requires both cavern and dump3d **1.4.22** by default; an
explicit `--consumer-version 1.4.23` selects the other verified exact version.
Custom absolute paths use `--cavern` and `--dump3d`. It exports nine semantic cases, compiles them
and independently authored controls, compares `.3d` nodes/edges/flags at consumer
precision and exact leg/loop/length/plan/vertical statistics, and verifies source
immutability and package hashes. Active dates and a complete automatic profile
are deliberate edits using known synthetic test dates/coordinates. The default
export retains commented templates. A hostile include checks reset/restoration;
three additional includes cover decimal commas, hash comments and both together,
comparing the full graph with independently authored readings and a connected
caller leg that must still use its original lexical settings after the include.
An actual same-station error checks compiler failure. Parser tests reject
missing/nonfinite/malformed results and incomplete template editing.

The required Linux `survex-native` job builds official source archive
`https://survex.com/software/1.4.22/survex-1.4.22.tar.gz`, SHA-256
`a26962d888621cfd2043a7cc6a227fc905130dfbd14c450dc1c4c02fbfdd9dbc`.
It installs runner-only build dependencies and uploads raw SVX/report inputs,
`.3d`, logs/dumps, controls and structured comparisons as `original-survex-<SHA>`.
CI complete additionally requires this job. Local original consumer snapshots are
frozen in `internal/exportsvx/testdata/survex-native`; hash verification includes
`validation/p16-evidence.json`. P14 native goldens/corpus and P15 original Walls
evidence remain unchanged. The lexical correction adds eight separate byte/report
expectations under `internal/exportsvx/testdata/lexical`, pinned by
`validation/svx-lexical-evidence.json`; previous P16 snapshots stay frozen.
Its three new explicit faults cover decimal reset, comment reset and scope
placement, retaining all 53 earlier SVX faults. Optional Go consumer regressions
run when cavern 1.4.22/1.4.23 is installed; the dedicated CI consumer gate always
exercises all three lexical cases with pinned 1.4.22.
Offline distributions add one SVX destination case
without changing the six P14 native cases or their hashes.
