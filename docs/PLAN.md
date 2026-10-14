# Current state and delivery plan

This is the single current backlog and handoff. [SPEC](SPEC.md) defines product
contracts; [DEVELOPMENT](DEVELOPMENT.md) defines verification and delivery.
Historical PBI numbers remain traceable; they do not imply equal-sized outcomes.

## Current implementation — 2026-10-14

P04d started from P04c17 (`bfa73944e3d3baad00b07e2b6dd5a8c9f9a3682b`) plus
the documentation reset (`94a3f2e`). P05 builds on verified P04d
(`e3b3b3a395741757d1cf2527b192bfceeae5fa52`). P07 builds on verified P06
(`0e3e63b5e3057eed5c51df1a03048c60ca4b8c16`). P08 builds on verified P07
(`bd96ea94057ae7a6721caef94160dedf323aa662`). P09 builds on verified P08;
P05–P08 and existing prefix contracts remain unchanged.

| Area | Implemented behavior | Delivery state |
| --- | --- | --- |
| P00/P02 | Go skeleton, pinned tools and verified quality gates | Implemented |
| P03a–P03c2 | IDs, bounded v3 header/trips/measurements/references | Implemented; exposed by P04d inspection |
| P04a/P04b | Separate overview and plan mappings | Implemented |
| P04c1–P04c13 | Fixed-position marker/count/point/color prefix APIs | Implemented historical contracts; preserve regressions |
| P04c14/P04c15 | Bounded ordered plan Polygon/XSection loops, explicit stops | Implemented |
| P04c16/P04c17 | Side mapping and first raw side marker prefix APIs | Preserved historical contracts |
| P04d | Full supported v3 source document, side drawing, tail accounting, inspect JSON | Implemented; full local gates verified |
| P05 | Isolated native text CLI, native byte goldens, provenance/loss report, safe new-directory publication | Implemented; full local gates verified |
| P06 | Native fixed-point grouping, source-row lineage and average-direction processing/JSON | Implemented; full local gates verified |
| P07 | Isolated reference placement, plan/extended geometry and optional native loop closure | Implemented; full local gates verified |
| P08 | Explicit native template-directory geometry and per-source lineage | Implemented; full local gates verified |
| P09 | Native Therion DATA, drawings and embedded XSections with provenance | Implemented; local ordinary gates verified; exact-SHA hosted gates required |
| P10 | Native graphics DXF, plan/side/options, provenance and bounded native failures | Implemented; local ordinary gates and independent review verified; exact-SHA hosted gates required |
| P11 | Native VisualTopo 5.02, direction-selected passage dimensions, explicit context and provenance | Implemented; exact-SHA hosted gates required |
| P12 | Native Toporobot and dialog options, bounded existing-file merge and provenance | Implemented; exact-SHA hosted gates required |
| P13 | Native 3D DXF, source-linked points/entities and bounded native hulls | Implemented; exact-SHA hosted gates required |
| P14 | Native capability inventory, corpus validation and offline distributions | Implemented; exact-SHA hosted gates required |
| P15 | Walls SRV measurement export with P06 averages and individual splays | Delivered at `17c5960`; all 30 exact-SHA jobs passed |
| P16 | Survex SVX individual-reading export | Delivered at `d600cd0`; all 31 exact-SHA jobs passed |

### Verification and publication

P04d has independent native side/tail/writer evidence and a read-only fresh-agent
review of the reader/model/CLI and C#/IL, with no substantive implementation
findings. Its many-small-elements memory observation is addressed by a second
benchmark and explicit limits in SPEC. Tooling review identified a weakened weak-suite
cardinality check; the exact nine-mutant requirement is restored and verified.
Local verification passed: check.sh (including race, Staticcheck, build and 100%
statement coverage), 30-second document fuzzing (1,819,250 executions), all 482
Gremlins mutants and all 996 explicit faults in 24 scopes, plus negative and
dispatch controls. Existing frozen fixtures remain unchanged; P04d adds its own
native transcript and provenance.

Exact-SHA hosted CI must pass before publication is reported as complete. Commands
and full mutation scope remain authoritative in DEVELOPMENT/scripts; no earlier
SHA substitutes for the final pushed revision. The commit's GitHub checks and
delivery response provide publication evidence.

The P04d request authorized direct work, commit and push on `main`.
No branch/worktree/PR or exporter implementation was part of P04d.

## P04d — complete v3 source reading and inspection

**Status:** implemented and verified, including exact-SHA CI; preserved by P05.
**Outcome:** a user can inspect a TOP v3 file and know what was read, whether it
is complete/supported, and which bytes remain unexplained. This closes the
original P03 inspection and P04 complete-file goals.

This replaces the unimplemented P04c18 count-only proposal. Do not create a new
series for side count, points, color, next marker or the next Polygon.

**Dependencies and scope:** build on P04c17 on `main`; read the native analysis
map, `DataSet.Read/Write`, `Drawing.Read/Write`, `Polygon`, `XSection`, `Mapping`
and corresponding IL. Add a full-file source API, complete bounded side drawing
records and a minimal `inspect` CLI. Preserve existing APIs and golden evidence.
No geometry, export writer, old TOP versions or template-directory mode.

**Acceptance:**

- Read any bounded number of complete side Polygons/XSections in original order,
  including empty and mixed drawings, all raw colors and raw XSection fields.
  Establish the side cases with independent native probes and fixture readback.
- Preserve raw fields/bytes and exact spans; distinguish plan from side records.
  Bound allocations before reading points. Define and test per-drawing versus
  whole-document budgets, zero/exact limits and malformed/truncated cases.
- Preserve unknown markers and unread bytes without guessing payload length.
  Expose a structured unsupported/incomplete result. Never treat successful
  partial-prefix reading as a complete supported file or silently continue into
  the next drawing after an unsupported plan element.
- Account for every byte after the side terminator. Native writer output ends
  with Int32 zero, while the native reader ignores this boundary. Probe absent,
  partial, nonzero and additional trailing bytes; distinguish canonical writer
  form from native-reader acceptance and explicit unsupported-tail policy.
- Provide `inspect <input.top> --json` with a versioned deterministic summary:
  source SHA-256/size/version, table and drawing counts, completion/support state,
  consumed/unparsed byte accounting and structured diagnostics with offsets.
  Keep source values available through the full-file API; do not dump every
  vertex by default. Preserve source files; no implicit neighboring-file scan.
- For inspect, exit 0 only for a complete supported input under the documented
  policy; exit 1 for I/O, malformed, unsupported or incomplete input. Include
  structured status in JSON where input analysis is available; keep human
  diagnostics on stderr. Preserve existing help/version behavior.
- Test both supported native fixtures and synthetic boundary cases, input and
  accessor immutability, deterministic JSON, CLI failures and resource bounds.
  Avoid extending nested whole-prefix copies unnecessarily; assess memory on a
  representative bounded input before claiming scalability.
- Pass the applicable full gates in DEVELOPMENT. Update this status and native
  evidence. Add/rebalance mutation scheduling only as needed to keep full scope.

### P04d implementation sequence

- [x] Establish native side-record and full-file boundary evidence using the original
  1.372 assembly, independent literal streams, C# and IL; preserve earlier goldens.
- [x] Add `source.Document` and `top.ReadV3[WithLimits]`: flatten table records,
  share immutable drawing record types, retain complete raw input and field spans.
  Test mixed side records, unknown stops, every truncation, tail classes and budgets.
- [x] Add `inspect <input.top> --json`, deterministic schema 1, bounded input I/O,
  structured analysis failures and explicit exit status. Test file/output failures.
- [x] Extend full mutation scope without removing existing dispatch/failure controls;
  measure representative memory, fuzz the reader, run all gates, obtain an independent
  review and update current contracts.

**Publication requirement:** commit/push the verified P04d tree on main and confirm
every hosted CI job for that exact SHA before reporting delivery complete.

Routine representation decisions: drawing counts are per drawing; document ceilings
add a separate budget across both drawings. Malformed reads return an empty source
result and a stable ParseError (matching existing APIs); inspect reports unavailable
counts and all bytes as unparsed in that case. An absent trailer is supported but
noncanonical; partial/nonzero/extra tails are preserved and unsupported. Native
reader acceptance is established independently and does not override this policy.

**Refinement rule:** settle routine API names and representation in code review.
Record native boundary findings in SPEC/evidence before relying on them. A new
dependency that changes the outcome or adds geometry/export work requires
revising this PBI explicitly, not returning to arbitrary one-field tasks.

## P05 — native text export

**Status:** implemented; full local gates verified. **Dependency:** P04d.
Delivers a usable
`export` CLI for the native text format with exact original-program goldens,
explicit isolated-input context, deterministic output/provenance/loss reporting
and safe output publication. Research `Exporter.WriteText`, `Survey.WriteText`,
`Station.WriteText` and the text-relevant `Survey.Update` effects in C#/IL and
independent native probes before implementing. Promote only demonstrated P06/P07
prerequisites; do not assume all geometry is necessary or bypass Update silently.
Do not bundle other exporters or template-directory behavior.

### Delivered behavior and verification

`export <input.top> --format text --output <new-directory>` produces native
`survey.txt` plus deterministic schema-1 `report.json` in one explicit isolated
context. The fixed profile is metres/360 degrees/invariant culture/UTF-8 without
BOM/CRLF. Every source row survives, with byte-span/output-line lineage,
raw/native/display identities, source and output hashes, transformations and
omissions. P04d and prefix APIs remain unchanged. Existing output paths are
refused; files are synced and closed before no-replace hard-link publication,
report last. Cleanup retires temporary names and checks original file identity
while private hard links pin inodes through rollback. Once the complete report
is linked, cleanup errors preserve the complete package.

C# and IL plus seven independent successful native byte goldens establish text
formatting and Update independence. These include actual repeat/reversal across
trips, all 256 flags and 512 large Int64 references. The probe also preserves
native Update failure output: a first projected shot throws and leaves a header.
P05 explicitly elides geometry and does not reproduce native geometry exceptions;
its compatibility claim applies when native Update completes. No P06/P07
processing, other exporter or template-directory mode was needed or added.
See [SPEC](SPEC.md#isolated-native-text-export-p05) and
[native provenance](../internal/top/testdata/README.md#p05-native-text-export).

A fresh read-only reviewer checked native C#/IL, all new behavior and gates,
finding one substantive cleanup ownership defect. Recreated temporary/final
path regressions failed before the repair and pass afterward. Follow-up survivor
review strengthened input-read error preservation and exact-limit diagnostics.
Linux CI additionally exposed immediate inode reuse after removing the original
final link. The repair retains private ownership hard links through rollback;
independent review confirmed the ordering and post-commit failure policy. Tests
cover second-anchor conflicts and foreign file/directory anchor replacement after
commit, alongside the original rollback/recreated-path cases. Windows rejects
control-character filenames before export; only that filesystem fixture is
skipped there, while direct title validation remains cross-platform.

Final ordinary gates pass with **99.5%** total statement coverage. Gremlins
reports **619/626 KILLED (98.88%)**, seven reviewed survivors and no uncovered,
invalid, timed-out or skipped outcomes. Five survivors are equivalent zero-boundary
or valid-ID changes; two affect only cleanup diagnostics, not publication or
foreign-entry preservation. All **1063** explicit faults across **25** scopes are
KILLED, including 67 native-text faults and six ownership-anchor/commit faults.
Dispatch controls preserve exactly-once scheduling and failure propagation;
the weak-suite control has all 18 mutants LIVED and rejects the gate. Compiler
and setup failure controls return NOT VIABLE, never KILLED. The rebuilt CLI
matches the native drawings golden, preserves its source and produces the
verified source/text hashes and exactly two artifacts. All 16 P05 frozen evidence
hashes are verified. Independent follow-up review reports no substantive findings.
The pushed commit's GitHub checks and delivery response provide exact-SHA CI
publication evidence; no earlier SHA substitutes for it.

## P06 — native grouped measurement processing

**Status:** implemented; full local gates verified. **Dependency:** P05. **Boundary:** isolated complete
supported TOP v3 input; fixed-point vectors, consecutive native identity groups,
source-row lineage and forward/reverse native average directions. Preserve P05
text bytes and existing inspection schema. Placement, closure, directory context
and other exporters belong to later stages.

### Implementation and verification sequence

- [x] Capture original Angle.ToRect/sine table, isolated Survey.EvalData and
  Station.GetAvDirection goldens; confirm C#/IL rounding, Int32/Int64 wrapping,
  identity aliases, undefined endpoints, flags and per-row trip corrections.
- [x] Add `internal/processing` with a source-preserving grouped result and
  `process <input.top> --json` inspection; test repeat/reverse/trip boundaries,
  0/360, 1/2/3+ readings, separated pairs, degenerate/extreme values and lineage.
- [x] Rebalance measured mutation groups and add bounded local concurrency in
  independent disposable checkouts; preserve all scopes and failure controls.
- [x] Run full ordinary/mutation/negative/dispatch gates, obtain independent
  review, and update contracts and provenance.

**Publication requirement:** commit/push the verified P06 tree on main and confirm
every hosted CI job for the exact pushed SHA before reporting delivery complete.

### Delivered behavior and evidence

`process <input.top> --json` and `processing.GroupMeasurements` preserve every
raw source row/offset and expose consecutive matching/reversed native groups,
Q30 fixed-point vectors, per-row corrections and forward/reverse Float32 average
directions. Original assembly goldens cover six datasets, every sine-table entry
and 176 ToRect boundary outcomes. P05's 16 frozen evidence hashes are unchanged.
Placement, closure, directory context and other exporters remain outside P06.
See [SPEC](SPEC.md#native-grouped-measurement-processing-p06) and
[fixture provenance](../internal/top/testdata/README.md#p06-native-grouped-measurements).

A fresh read-only reviewer checked C#/IL, source lineage, processing/CLI,
fixtures and mutation orchestration, independently running the full Go suite and
focused race checks. No substantive behavior defect was found. The review exposed
Gremlins switch-condition coverage mapping; equivalent `if` branches now permit
all sites to run without reducing scope. Compiler-invalid pointer-address
mutations were removed by idiomatic `new` allocation. Targeted survivor checks
added true ID-alias/same-To reserved-ID cases, exact trip-count boundary evidence,
input-read error preservation and exact input-limit tests. All 34 targeted P06
faults now fail named behavioral tests. Dispatch controls prove private snapshots,
complete report collection, bounded overlapping workers and failure propagation.
The final source snapshot passed `check.sh` (formatting, vet, staticcheck, race,
coverage controls, full tests, build and smoke checks): statement coverage 99.5%,
processing 100%. All **1097** explicit faults across **26** scopes were KILLED.
The 22-group parallel campaign completed successfully with at most three private
workers. The four Gremlins shards killed **731/741** mutants (98.65%); their
disjoint site union exactly matches unsharded discovery. No uncovered, invalid,
timed-out or skipped outcomes were accepted. Ten survivors were reviewed: seven
preserved P05 equivalents/cleanup diagnostics and three equivalent quarter-wave
boundary comparisons (both branches return the same exact endpoint values).
The weak-suite negative control leaves exactly 25 mutants LIVED and is rejected;
build/setup-error controls remain NOT VIABLE and fail closed. All 33 documented
P05/P06 evidence hashes were verified. Publication remains subject to every
exact-SHA hosted CI job.

**Next useful increment after delivery:** P07 — reference placement and plan
geometry, refined at a complete behavior boundary. Not implemented in P06.

## P07 — isolated reference placement, geometry and closure

**Status:** implemented; full local gates verified. **Dependency:** P06.
One complete geometry behavior: native EvalRef/SetupShot/SetupXsect ordering,
plan and extended coordinates, optional native weighted loop correction.
No templates, neighboring-file discovery or new export format.

- [x] Capture original-assembly geometry and SideProjection goldens; inspect
  Survey/Station/Reference/Projection/Loop C# and IL and compare JKTZ prior art.
- [x] Test first, then add `processing.PlaceGeometry(Document, GeometryOptions)`
  with immutable source-linked nodes, explicit unplaced groups, corrections,
  loop residuals and bounded failure. Preserve `GroupMeasurements` and P05.
- [x] Add `geometry <input.top> --json [--close-loops]`, deterministic isolated
  schema, input hash, native coordinates and structured processing failures.
- [x] Add geometry mutation faults without removing existing scopes/controls;
  run full gates and obtain an independent review. Publication is required below.

Review focus: duplicate/undefined/alias reference IDs; delayed/reversed and
disconnected legs; splays before/after adjacent shots; projected zero-length
ancestor failures; intersecting/zero-weight/overflowing loops. Native arithmetic
wraps; explicit resource/failure diagnostics replace crashes and unbounded work.

### Delivered behavior and evidence

`geometry <input.top> --json [--close-loops]` and `processing.PlaceGeometry`
place an isolated supported document using native reference/shot ordering,
plan X/Y/H and extended D coordinates. Source-linked immutable nodes retain
raw head spans and derived native identities; undefined, zero-splay and
unplaced groups remain explicit. Optional closure uses native raw-head weights,
Q30 corrections, reverse loop order and the 10 mm residual tolerance. Checked
work/iteration ceilings and stable failures replace native crashes or unbounded
work without returning partial geometry. P05 text bytes, P06 grouping/directions,
source values and previous prefix/inspection contracts remain preserved.
See [SPEC](SPEC.md#isolated-native-geometry-p07) and
[native provenance](../internal/top/testdata/README.md#p07-native-geometry).

C#/IL and independent reflection into the original 1.372 assembly establish
22 datasets with raw/closed node goldens and independent loop endpoint/order/
residual goldens, plus 1960 SideProjection outcomes. These cover actual repeated
heads, aliases/undefined references, deferred placement competition, reverse
attachment priorities, projection chains, splays and intersecting/fixed/zero-
weight loops. Native exception cases are frozen separately. All 103 documented
P05/P06/P07 evidence hashes pass; all 68 new geometry artifacts and the transcript
match fresh original-program output.

A fresh read-only reviewer found a loop-mark cleanup defect when the second
ancestor-search arm reaches the common node. A failing regression and a new
independent native fixture established the repair. Follow-up review checked
C#/IL, source preservation, geometry/CLI, evidence hashes and mutation survivors,
and independently ran focused race tests; no substantive findings remain.
The ordinary gates pass with 99.3% total statement coverage, including full
race tests, vet, Staticcheck, coverage controls, build and CLI smoke checks.
The complete campaign passes all 22 groups with at most two private workers:
all **1141** explicit faults in **27** scopes are KILLED, including all 44 P07
faults. The four Gremlins shards kill **994/1018** mutants (97.64%); their
disjoint site union exactly matches unsharded discovery, with no uncovered,
invalid, timed-out or skipped outcomes. The 24 survivors were reviewed:
17 equivalent processing boundary/width/sign changes and the seven preserved
P05 equivalents/cleanup-diagnostic changes.
The weak-suite control leaves exactly 35 mutants LIVED and rejects the gate;
compiler/setup failures remain NOT VIABLE. Dispatch controls pass without
removing any earlier scope or failure control.

**Publication requirement:** commit/push the verified P07 tree on main and confirm
every hosted CI job for the exact pushed SHA before reporting delivery complete.
The commit's GitHub checks and delivery response provide publication evidence.

**Next useful increment:** P08 — explicit native template-directory context,
with deterministic inventory, ordering and isolation tests. Implemented below.

## P08 — explicit native template-directory context

**Status:** implemented; full local gates verified. **Dependency:** P07. Work directly on main; preserve
all isolated P05–P07 APIs and outputs. Only P08 is authorized in this delivery.

**Outcome/contract:** an explicit `context <input.top> --template-directory <dir>
--json [--close-loops]` command inventories immediate TOP neighbors, excludes
the current basename, records hashes/order/roles and evaluates native two-phase
template/current geometry with source-file and local-row lineage. No implicit
scan; no new exporters. Templates read only v3 tables, preserving ignored bytes.
Portable UTF-8 byte ordering and case-insensitive `.top` selection are explicit
choices; native `Array.Sort` is culture-sensitive and its old `*.top` wildcard
also selects `.topx`, as the original CLR probe confirms. Fail closed on bad templates,
symlinks, unsupported current sources and aggregate budgets rather than retaining
native partial reads. Default ceilings: 1024 templates, 64 MiB total input,
32767 trips, 1 million rows/references. Closure holds templates fixed (`all=false`).

Implementation plan (SPEC remains the durable contract):
- [x] Capture original-assembly C#/IL-backed templates, cross-file grouping,
  boundary separation, roots/references, priority, projections and closure goldens.
- [x] Add tests first, then explicit inventory/context, two-phase processing and
  CLI with deterministic provenance. Preserve raw IDs/fields/spans per source.
- [x] Add P08 targeted mutations to the existing complete schedule, retain all
  controls, run full gates/mutations and independently review the final change.
- [x] Verify frozen evidence and update contracts/status. Publication is required below.

### Delivered behavior and evidence

The explicit CLI and `processing.GroupContext` / `PlaceContextGeometry` preserve
per-source raw records while evaluating combined template groups, separate current
groups, native trip rebasing, reference bases, attachment priorities and current-only
closure with fixed template nodes. P05–P07 retain their isolated behavior.
See [SPEC](SPEC.md#explicit-template-directory-context-p08) and
[fixture provenance](../internal/top/testdata/README.md#p08-native-directory-context).

C#/IL and reflection into the unchanged original assembly establish 13 directory
cases with 26 raw/closed native geometry goldens, including an inclined loop with
153 mm raw height residual and 0 mm after closure. A separate original-CLR pattern
probe confirms legacy `.topx` selection. Exact `.top` selection, UTF-8 ordering,
strict input failures and resource ceilings are documented portable choices.
All 57 P08 evidence hashes and 103 preserved P05–P07 hashes verify; all 54 native
artifacts plus stdout match fresh native capture, with empty stderr.

A fresh read-only reviewer checked C#/IL, source preservation, processing/CLI,
native evidence, limits and mutation survivors. Follow-up reviews confirm the
UTF-8 validation, pre-allocation trip ceiling, key representation and stronger
trip/file/height cases; no substantive findings remain. Uncached focused race
tests also passed independently.

The final ordinary gates pass with **98.8%** statement coverage, full race tests,
vet, Staticcheck, coverage controls, build and smoke checks. The built CLI matches
the project JSON golden and independently captured inclined-loop result. All
**1175** explicit faults in **28** scopes are KILLED, including all 34 P08 faults.
The complete 22-group campaign used at most two private workers. Gremlins kills
**1113/1138** mutants (**97.80%**); the disjoint four-shard site union exactly
matches unsharded discovery, with no uncovered, invalid, timed-out or skipped
outcomes. The 25 survivors were independently reviewed: 17 preserved P07
equivalents, seven preserved P05 equivalents/cleanup diagnostics and one private
zero-budget diagnostic boundary that still rejects the input. The weak-suite
control leaves exactly 38 mutants LIVED and rejects the gate; build/setup failures
remain NOT VIABLE. All dispatch and failure controls pass unchanged.

**Publication requirement:** commit/push the verified P08 tree on main and confirm
every hosted CI job for the exact pushed SHA before reporting delivery complete.
The commit's GitHub checks and delivery response provide publication evidence.

**Next useful increment:** P09 — native Therion DATA and drawing sections,
including embedded XSections. Implemented below.

## P09 — native Therion DATA and drawing sections

**Status:** implemented; local ordinary gates verified. **Dependency:** P04d/P07/P08.
Only P09 is authorized. Work on main; preserve isolated text, grouped measurements,
geometry and explicit directory context. Final hosted mutation/CI confirmation is
required before reporting publication complete.

`export <input.top> --format therion --output <new-directory>
[--template-directory <dir>] [--close-loops]` publishes native Therion interchange
`survey.txt` and a deterministic provenance/loss report. It reproduces FIX,
TRIP/DATA, PLAN/ELEVATION, native label/shot order, colored Polylines and embedded
XSection labels/rays/CONNECT records. Existing P05 publication is reused. See
[SPEC](SPEC.md#native-therion-interchange-p09) and
[native provenance](../internal/top/testdata/README.md#p09-native-therion-interchange).

- [x] Capture original-assembly goldens and confirm C#/IL, including trip/head
  filtering, references, all colors, lookup/order, XSections, context and closure.
- [x] Test first, then implement separate processing projection, serialization
  and CLI; preserve raw values, local-file lineage and P05–P08 contracts.
- [x] Add 43 targeted P09 faults to the unchanged 22-group schedule; preserve
  all earlier scopes, thresholds, negative and scheduling controls.
- [x] Verify frozen evidence, obtain independent reviews, update current contracts
  and run bounded local ordinary gates. Publication requirement is below.

### Delivered behavior and evidence

There are **89 independent successful native byte goldens** and **99** frozen P09
probe/input/output/transcript hashes. All **160** documented P05–P08 hashes remain
unchanged. Cases include repeated/reversed heads across trips and the template
boundary, undefined-FROM rows before a nonzero origin, unplaced self-pairs,
invalid trip indices, aliases/duplicate IDs, all 256 colors, empty/singleton
Polygons, wrapped coordinates, nonzero plan/side Polygon origins, and unresolved,
empty, horizontal/directed and template-bound XSections. Successful outputs and
native exception cases are accounted for separately. Native interchange is not
a ready-made Therion project; input/output/work limits and omissions are explicit.

Independent review found and verified repairs for ray-budget stopping, repeated
trip-slice cloning, output-line ownership and synthetic-root/source identity JSON.
Follow-up review checked source preservation, C#/IL, hashes, native closure/ray
ordering, strict limits and survivors. Package-scoped native acceptance tests now
exercise derived directions, raw-azimuth rays and undefined-origin selection.

The completed pre-refinement local campaign passed all **22** groups with one
private worker: **1218/1218** targeted faults were KILLED; Gremlins killed
**1253/1296**, with all 43 survivors reviewed. Its disjoint shard union matched
unsharded discovery, and no uncovered, invalid, timed-out or skipped outcomes
were accepted. The review identified fixture/assertion gaps, subsequently filled
using fresh native evidence and package-level consumer assertions. These final
test/fixture refinements require a new hosted campaign; the older local counts
are not evidence for the final test tree. The weak-suite control left all **42**
mutants LIVED and rejected the gate; build/setup errors remained NOT VIABLE.
Scheduling controls retained all scopes and failure propagation.

**Publication requirement:** commit/push P09 on main and verify every hosted CI
job, including all 22 mutation groups and three platforms, for the exact pushed
SHA. Following the 2026-10-12 workstation request, final heavy mutations run on
GitHub Actions. The commit's checks and delivery response retain publication
proof; do not substitute an earlier SHA or local snapshot.

**Next useful increment:** P10 — native graphics DXF with plan/side options,
independent native bytes and coordinate checks. Implemented below. P09 remains preserved.

## P10 — native graphics DXF

**Status:** implemented; local ordinary gates and independent review verified.
Exact-SHA hosted gates are required before publication is reported complete.
**Dependency:** P04d/P07/P08. Only P10 is authorized;
work directly on main and preserve P05–P09. Stop after P10.

**Outcome:** `export <input.top> --format dxf --output <new-directory>
[--view plan|side] [--scale 1..100000] [--dxf-options <native-mask>]
[--grid-unit metres|feet] [--template-directory <dir>] [--close-loops]`
publishes `survey.dxf` and deterministic source/geometry/entity provenance and
loss reporting. One selected native drawing per invocation; default plan,
scale 500, shots/rays/labels enabled (mask 14), metre grid. The SIDE bit is
selected by `--view`; all other native graphics bits are supported independently.
No 3D DXF, VisualTopo, Toporobot or optional drawing formats.

Architecture: `internal/processing/graphics.go` captures a detached native
view and metadata from existing bounded geometry; `internal/exportdxf` owns
serialization, options, limits and entity lineage. `internal/cli/dxf.go` reuses
P08 inventory and P05 publication with a selected artifact filename. Existing
processing/export schemas and byte outputs retain their contracts.

Implementation and verification plan:
- [x] Confirm original DXFWriter/Polygon/XSection/Station/Survey/GraphicsForm
  C# and IL; capture independent plan/side/options/context/closure/color/number
  goldens and native failure cases without recompiling decompiled code.
- [x] Add failing DXF CLI/native byte and coordinate assertions; implement
  detached graphics view, bounded serializer and provenance, then verify the
  same tests pass. Test overflow/precision, invisible/dashed rays, origins,
  unresolved stations, template ownership, empty drawings and failures.
- [x] Extend existing mutation scopes/shard scheduling without losing earlier
  faults or controls; run bounded local ordinary gates, independent review,
  frozen evidence hash readback and coordinate/visual checks.
**Publication requirement:** Commit/push the final reviewed tree, run all ordinary platform gates,
explicit mutations, Gremlins and negative/dispatch controls in GitHub Actions,
review survivors and verify all jobs for the exact delivered SHA. The commit checks
and delivery response retain this proof; no earlier SHA substitutes for it.

Native partial writes are explicit errors before publication. Output bytes,
entities and grid work have ceilings; native potentially wrapping grid loops
must fail explicitly when they cannot terminate within bounds. No inferred CRS
or survey dates. Report raw fields separately from derived coordinates.

### Delivered behavior and verification

The separate graphics view and serializer reproduce all **261 complete native
byte goldens**; **7 native partial outputs** return explicit errors before
publication. The unchanged original assembly, C#/IL, all native option bits,
plan/side, source colors, XSections, scale/number formatting, metric/feet grids,
GEO with current/template references, directory contexts and optional closure
are covered. Independent coordinate/structure checks and a rendered native
plan/side preview supplement exact bytes; no destination CAD import is claimed.

Independent review identified table cloning per provenance row and last-entity
footer ownership; both were repaired and protected by regressions. The final
review also strengthened nonzero-roll/reversed-member and strict dash-boundary
fixtures, confirming the fixtures contain the cases asserted. No substantive
production findings remain. All **290** new checksums verify; **308** preexisting
raw/golden/probe files remain byte-for-byte unchanged. Local check.sh passes
with race, vet, Staticcheck, coverage controls, build/smoke and **98.7%** statement
coverage. All **46** weak-suite mutants LIVED and the quality gate rejected the
suite; compiler/setup controls return NOT VIABLE. Dispatch controls preserve
exactly-once scopes, report/error propagation and private worker limits.
P10 adds **60** explicit faults in the existing native-text group and assigns
new graphics production files to the output/processing Gremlins shards. The
complete heavy campaign runs on GitHub Actions, including all earlier scopes.
Its first run rejected nonviable generated string/address mutants and uncovered
global-initializer sites, as required. Equivalent operations now use the existing
P09 allocation/string idioms and literal native option constants. No operator,
exclusion or gate was weakened; a full dry discovery finds **1588 RUNNABLE** sites
and zero uncovered sites. The initial explicit side-dash fault also correctly
failed compilation rather than counting as KILLED; its replacement retains the
height variable while zeroing its contribution. The inclination fault likewise
retains its source-table read; all 60 targeted faults pass a serial bounded
compiler-only preflight. Package-owned tests and 44 new
native goldens strengthen default/boundary options, no-template-export masks,
nonzero origin fallback, label layers, exact ray thresholds and positive/negative
survey grids. Existing 224 P10 golden bytes remain unchanged. Independent review
classifies prior/equivalent survivors; final hosted results remain required.

**Next useful increment:** P11 and P12 are delivered below.

## P11 — native VisualTopo and passage dimensions

**Status:** implemented; local gates and independent review verified;
exact-SHA hosted gates are required before publication is reported complete.
**Dependency:** P06/P07/P08. P05–P10 and existing prefix APIs remain unchanged.
The P11 delivery ended after P11; P12 is delivered below.

`export <input.top> --format visualtopo --output <new-directory>` publishes native
`survey.tro` and a deterministic schema-1 provenance/loss report. The default
profile is metres/360-circle/Windows-1252 best fit/invariant culture/CRLF;
`--angle-unit grads`, explicit `--template-directory` and current-only
`--close-loops` are supported. Native sorted placed heads, reference headers,
trip fallback, Float32 formatting, Flip/comment conventions and LRUD selectivity
are established with original C#/IL and 194 independently checked native byte
goldens. Exhaustive native encoding/trim tables and 4105 Single-format goldens
pin runtime behavior. Duplicate 17/32/65-head fixtures verify .NET 2 sorting ties.
See [SPEC](SPEC.md#native-visualtopo-and-passage-dimensions-p11) and
[native provenance](../internal/top/testdata/README.md#p11-native-visualtopo-and-passage-dimensions).

Implementation plan (Go standard library; native behavior defines the contract):
- [x] Capture original WriteExport/WriteList/GetStations/WriteXSections evidence,
  both angular profiles, context/closure, dimensions, encoding and sorting.
- [x] Add detached VisualTopo processing and raw-ray LRUD, verified against
  independent native goldens; retain previous geometry/export contracts.
- [x] Add bounded serialization, current/template member and candidate-ray
  provenance, semantic omissions and existing safe no-replace publication.
- [x] Extend full Gremlins/explicit scope with 53 P11 faults and preserve
  exactly-once scheduling, exact weak-suite cardinality and failure controls.
- [x] Independent review; repair trip-table cloning and splay omission reasons;
  verify all native bytes, ordinary gates and local negative/dispatch controls.
**Publication gate:** commit/push on main and confirm every GitHub Actions job
for the exact SHA. The final handoff records the delivered commit and CI run.

Heavy mutation campaigns use the existing GitHub Actions matrix. The weak-suite
negative control now requires exactly 50 exercised live mutants, rejects its
quality score and preserves build/setup-failure controls. Local ordinary checks
use `GOMAXPROCS=2 GOFLAGS=-p=1`. Published CI results, including every mutation
scope/shard and the aggregate, must belong to the final delivered SHA.

Hosted mutation feedback additionally requires package-local native snapshot
goldens, nonzero raw trip-tick provenance, exact trip bounds/global indexes,
mixed/wrapped identity sorting, diagonal LRUD and exhaustive BMP conversion.
Compiler-invalid string/pointer mutations and initialization panics are avoided
with normal Go serialization/value updates and direct embedded native tables;
sort progress and range checks bound altered partition control flow. Gates,
operators, previous scope and thresholds remain unchanged.

**Next useful increment:** P12 — Toporobot and dialog options, delivered below.

## P12 — native Toporobot and dialog options

**Status:** implemented; local gates and independent review verified.
Exact-SHA hosted gates are required before publication is reported complete.
**Dependency:** P06/P07/P08. Work directly on main;
preserve P05–P11 and stop after P12. P13 remains planned only.

**Outcome:** native `survey.text` plus schema-1 provenance/loss report, isolated
or explicit template context and optional closure. Expose native dialog defaults:
code/trip minimum indexes 1 (range 1..100000), included splays, horizontal passage
sections, automatic/manual declination overrides, and read-only merge from an
existing native MacEncoding file. Degrees/grads are explicit. An export timestamp
is recorded and can be fixed for reproducibility; it is writer metadata, not a
survey date. All publication uses the existing new-directory no-replace contract.

**Architecture:** detached Toporobot processing view; separate bounded native
model/parser/writer in `internal/exporttoporobot`; CLI reuses P08 input inventory
and P05 publication. No new dependencies or changes to previous output schemas.

Implementation and verification plan:
- [x] Confirm original ToporobotExp/Toporobot/ToporobotForm/MacEncoding and IL;
  capture original-program byte goldens, merge/roundtrip and failure cases.
- [x] Write failing native-byte/CLI tests; implement detached geometry, native
  LRUD at radius 100 with Int32 wrap, series/trip/code selection and inline rays.
- [x] Implement bounded MacEncoding read/merge/write, UTF-16 truncation/comments,
  fixed timestamp option and detailed source/output lineage; test malformed files,
  holes, duplicate IDs, padding, no references, resource limits and preservation.
- [x] Extend explicit/Gremlins scope and negative/dispatch controls without
  removing prior faults/operators/thresholds; run bounded ordinary local gates.
- [x] Independent final review and regression repairs; final ordinary gates,
  native readback, evidence preservation and complete mutation discovery.
**Publication requirement:** commit/push on main, run the full GitHub Actions
campaign, review survivors and verify every job for the exact delivered SHA.

**Review focus:** merge rewrites/renumbering; timestamps and source dates;
reserved/series-zero IDs; overwritten output records and template starts;
Float32/Int32 overflow and Unicode/continuation boundaries.

Native C#/IL and original assembly probes establish 325 complete byte goldens,
54 native failures and 21 existing-reader/writer cases. MacEncoding is exhaustive;
context/closure, both circles, all 16 dialog combinations, terminal series links,
long Unicode/comments, date/index boundaries, deduplication and Int32 LRUD wrap
are covered. Existing nonfinite values, oversized Int32 units, incomplete BOM
code units and work limits fail explicitly. No consumer import is claimed.

Early independent review identified shared-start copying, retained-zero LRUD,
UTF-16 padding, comment-copy work and group lookup issues. Final review also
identified existing continuation provenance, signed decimal padding, null
reference names and terminal-link omission classification. Native readbacks and
regressions verify all repairs; no substantive review finding remains. The final
ordinary suite includes race/vet/Staticcheck, >=95% exact coverage, build/smoke;
the existing-file fuzz run passed 256,532 executions in 10 seconds. All 55 explicit
faults compile in preflight. Full discovery has 2,405 RUNNABLE sites, zero uncovered
sites and an exact disjoint five-shard union; no operator or threshold was weakened. Final ordinary gates and independent review are verified. Hosted mutations,
survivor review and exact-SHA publication remain required before delivery claims.
P12 adds 55 explicit faults without removing prior scopes. A fifth Gremlins shard
holds the new exporter/CLI; all production files remain exactly-once assigned.
The weak-suite control requires exactly 54 live mutants and retains setup/build
failure rejection. At P12 delivery, P13 remained planned only; the P12 request ended after P12.

Hosted feedback additionally drives assertions for raw trip/reference/template/
existing-file identities, output record indexes, fixed-field reader boundaries,
shared slot/work budgets, code reuse and normalized high-bit station numbers.
The native Single:F2 oracle from P11 also checks seven-significant-digit formatting.
Six new original-program boundary goldens cover cross-series station1 and
zero/positive successor LRUD, including merges. Refactoring invalid pointer
operator sites and guarding trip-comment progress keeps native bytes unchanged;
compiler failures and timeouts remain rejected rather than counted as kills.

**Next increment at P12 delivery:** P13 — native 3D DXF, implemented below.

## P13 — native 3D DXF

**Status:** implemented; ordinary gates and independent review verified;
exact-SHA hosted publication required. **Dependency:** P07/P08. Deliver only native ThreeD
DXF through `export --format dxf3d`, isolated or explicit template context, with
optional current-only closure. Preserve P05–P12 and stop before P14.

**Architecture:** separate `internal/exportdxf3d` consumes the detached graphics
geometry view. Reproduce original .NET station sorting, 16 mm quantization,
Int32 cross products/Int64 dot products, point identity, persistent previous-face
normal and ordered native hull traversal. Reuse bounded P08 input loading and
P05 no-replace publication. No dependencies or previous schema changes.

Implementation and verification sequence:
- [x] Confirm ThreeD, Survey.GetStations and ID.CompareTo C#/IL; capture original
  assembly full-file and direct stretch goldens, degenerate and overflow cases.
- [x] Write failing byte/lineage/boundary tests; implement native station order,
  endpoint ancestor fallback, attached splays, line/triangle/hull serialization.
- [x] Add CLI, source/output hashes, raw rows/reference identities, node/point/
  entity lineage and explicit omissions. Fail without partial publication on
  malformed/unsupported input, native geometry failure and bounded-work excess.
- [x] Extend complete mutation scope and negative/dispatch controls; run bounded
  local ordinary gates and independent review.

**Publication requirement:** commit/push on main, run all hosted gates/mutations,
review survivors and verify every CI job for the exact delivered SHA.

**Review focus:** collinear/coincident/coplanar point identity; equal station IDs
and native sort overflow; zero-distance ancestors and template splays; arithmetic
wrap and negative quantization; work/entity/byte limits and report line ownership.
Native hulls are compatibility output, not an inferred watertight cave surface.
P14–P16 are implemented below.

Original-assembly evidence has 135 complete byte goldens, 24 generic-sort cases
and 7 native failures; all 1,200 pre-existing evidence files match the baseline.
Independent review confirmed core C#/IL, 77 full-file byte comparisons, raw
row/reference readback and point/entity ownership. Follow-up review required an
exact 45-entry, unique-name/KILLED gate for the new explicit campaign; it is added.
The new exporter has 100% statement coverage. All 45 explicit faults compile;
full Gremlins discovery has 2,611 RUNNABLE sites, no uncovered sites, and an exact
disjoint five-shard union. No operators or thresholds were weakened. The bounded
hull fuzz passed 535,209 executions in 10 seconds; final ordinary gates include
race/vet/Staticcheck, exact coverage, build and smoke checks. Full
hosted mutations/survivor review and exact-SHA CI remain required.

Hosted feedback adds 24 original FindFace point-identity/normal goldens, killing
four previously-live tie faults. Invalid string arithmetic sites are avoided by
`strings.Join`; negative work and finite partition-progress controls prevent
counter/partition mutants from timing out. Their named-failure reproductions and
follow-up independent review pass; full final-SHA hosted gates remain required.

## P14 — native capability matrix and distribution validation

**Status:** implemented and independently reviewed; publication requires
the exact-SHA hosted gates below.
**Dependency:** P05/P09–P13 and the pinned test2 corpus.
Deliver a closed, evidence-linked inventory of the six native exporters and
their exposed profiles, a reproducible corpus regression gate, and validated
offline binary packages for Linux amd64, Windows amd64 and macOS arm64.
Preserve P05–P13; no new exporter, input version, consumer integration or P15/P16.

**Architecture:** standard-library Python development scripts validate the built
CLI and package it; application Go code and existing schemas stay unchanged.
The external corpus is pinned to test2 `4c00c008a8dd441d70f9c62aa376c20b5914c7e0`;
each of its 262 TOP paths retains size, SHA-256 and Git blob identity. Exact
native goldens establish compatibility; corpus outcomes are explicitly project
regressions, with separate supported/unsupported/failed classifications.

Implementation sequence (executed inline on the requested main checkout):
- [x] Inventory C#/IL entrypoints, global/dialog options, native captures and
  deliberate deviations in `docs/CAPABILITIES.md`; freeze prior evidence hashes.
- [x] Add `scripts/validate-corpus.py` and its negative tests: reject missing,
  added or changed inputs; inspect every source; exercise all six formats in
  isolated and explicit context/closure profiles; verify input/output hashes,
  complete packages, deterministic reports and failure without publication.
  Freeze inspect/export outcomes separately from original-program observations.
- [x] Add `scripts/distribution.py` and negative tests: create new ZIP packages
  with binary/docs/build identity/checksums; verify extraction and every payload;
  run the extracted binary offline against six native byte goldens and test
  no-replace/error behavior. Bind packages to the exact checkout SHA.
- [x] Extend Actions with Linux corpus validation and three native-platform
  distribution gates/artifacts, retaining the complete mutation matrix and
  controls; include every new job in `CI complete`.
- [x] Run bounded local ordinary/tooling checks and native golden verification;
  obtain independent review and update SPEC/PLAN/evidence.
- Publication gate: commit/push and require all hosted checks and mutation
  groups for the exact final SHA; the commit's CI run is the publication record.

Review focus: duplicate input paths versus bytes; missing/extra corpus entries;
native caught partial files; absolute-path-independent report hashes; archive
tampering, unexpected members and executable permissions; package source SHA;
failed exports leaving output; no unsupported capability called native parity.

**Milestone claim:** coverage of the documented bounded profiles only. Text
remains the P05 metres/360 isolated profile; native text feet/grads and arbitrary
machine code pages are explicitly outside current compatibility. Destination
application acceptance and optional drawing research remain separate results.
At P14 delivery, work stopped there and P15/P16 remained planned. P15 is now
implemented below; P16 is implemented after its separate request.

The local complete corpus run has 6550 profile outcomes: **6504 successful**
packages and **46 explicit failures**, each repeated with identical outcomes.
The 46 failures are 28 negative Toporobot indexes and 18 context-closure
failures (12 work-limit and 6 iteration-limit results across six views/formats).
All 262 text exports match independently captured original-assembly bytes.
The 262 source paths contain 258 distinct hashes; duplicate paths stay separate.
All 1462 earlier evidence identities remain unchanged. Bounded local ordinary
gates passed with race/vet/Staticcheck and 98.1% total statement coverage;
validation negative controls and all six distribution native-byte cases pass.
Hosted distribution, full mutation and corpus checks must pass for the final SHA.
Independent review verified the pinned Git tree, all native text comparisons,
the 1462/271 evidence inventories and preserved C#/IL/checksum manifests.
The reproduced archive symlink/file-type acceptance gap is fixed: every ZIP
member must be a regular file, with negative binary/document controls. Final
review found no unresolved actionable defect. No application Go, report schema,
mutation scope, operator, threshold or existing failure control was changed.

The first full P14 hosted run passed all 29 jobs. Survivor review then found
three uncovered mutation behaviors despite equal export bytes: a whole-turn
change to reverse forward-azimuth provenance, and extra singleton sort work
in VisualTopo/3D DXF. Fixed regression expectations now preserve the native
Int32 angle and inclusive work boundaries; all three mutated variants fail
named tests. Production behavior and frozen evidence remain unchanged. The
final test commit must repeat every hosted job; the earlier green SHA is not
the delivery SHA.

## P15 — Walls SRV

**Status:** implemented; exact-SHA hosted gates required.
**Dependencies:** P04d, P06 and existing export/report publication.
**Scope:** only P15, following the
[SRV/SVX contract](SPEC.md#walls-srv-and-survex-svx-design-p15p16).

**Acceptance:**

- Produce `survey.srv` + `report.json`: P06 averaged repeats, raw singleton/splay
  rows, reversible names, stateful declination and commented editable trip dates.
- Verify reverse/mixed-correction groups, 0/360 and vertical angles, zero ties,
  invalid-data failures, date templates and full original-row lineage.
- Validate unclosed vectors and actual Walls acceptance; record version/settings
  and anonymous-splay limitations. Survex reading SRV is a separate check.

### Delivered behavior and evidence

`export --format srv` writes new-directory `survey.srv` and `report.json` using
P06 consecutive spatial averages, head orientation/correction, raw singletons
and separate anonymous splays. It preserves all source-row spans/values, raw ID
aliases, original/native/emitted values, members/reversals, unresolved correction
sources and output-line ownership. Initial/change-only manual declination and
commented trip-date templates are separate directives; no generated averaging
comments, geographic fixes, geometry/closure or inferred LRUD are added.

Original PocketTopo independently read eight authored cases and captured raw vectors
and Float32 direction bits. Review found Walls rejects zero self-legs as well as
nonzero self-legs; both now fail before publication. Distinct-name zero ties are
retained. Review also added a nonfinite native-summary negative control. All
P05–P14 frozen evidence remains unchanged. The 58-fault Walls campaign joins the
existing native-text group and output shard without removing any previous scope,
operator, threshold or failure control; the exact weak-suite cardinality is 62.
Hosted output review exposed four nonviable string-arithmetic mutations; typed
formatting preserves exact SRV bytes while allowing the unweakened gate to run.
Seven substantive survivor test gaps (grouped inclination units, negative vertical
normalization, missing-trip boundary, exact comment chunk, drawing-loss count,
raw invalid inclination hidden by averaging and small negative bearing)
were reproduced as named failures with bounded local replay and closed with
independent native/output assertions. The weak-suite measured inventory is exactly 58 preserved sites plus four new
Walls dispatch sites; its gate also checks those four sites explicitly. Full
mutation evidence remains hosted.

The original Walls v2.3.0-beta.1 Windows gate installs the checksum-pinned official
runtime, compiles the reviewed SRV in the application's normal project UI path,
closes it normally and verifies native NTV vectors/final NTN statistics, commented
dates and an actual invalid-inclination negative control. Its exact-SHA artifact
retains projects, input/report hashes, native workfiles and window-state evidence.
The successful original snapshot is frozen under `internal/exportwalls/testdata/walls-native/`
with the runtime hash and acquisition record; independent controls confirm
6/2/2/2 named vectors and 21/9.964/1/2.5 m, excluding both anonymous wall shots.
A Survex import is not its acceptance oracle. Local native-window access was
unavailable because the Mac was locked; the hosted native gate is authoritative.
Local ordinary checks (race, Staticcheck, coverage/build and validation controls)
and complete dispatch controls passed with `GOMAXPROCS=2 GOFLAGS=-p=1`; full
mutations run in Actions. Incomplete local preflight stopped on disk exhaustion
and is not counted as mutation evidence.
See [provenance](../internal/top/testdata/README.md#p15-walls-srv) and DEVELOPMENT.

P15 delivery ended at `17c5960338ec94203dd2874dcdbd745b0c5df512`;
[CI 38460204713](https://github.com/dlubom/pockettopo-exporter/actions/runs/38460204713)
passed all 30 jobs including original Walls, every mutation and CI complete.
P16 follows the separately authorized request below.

## P16 — Survex SVX

**Status:** implemented and independently reviewed; final exact-SHA hosted gates
remain the publication requirement.
**Dependencies:** P04d, P15's shared policy and existing export/report publication.
**Scope:** only P16, following the same
[contract](SPEC.md#walls-srv-and-survex-svx-design-p15p16).

Implementation plan (inline on main; accepted SPEC is authoritative):
- [x] Add failing CLI/writer tests for ordered individual rows, reversal, stateful
  correction/date templates, literal identities, invalid input and report lineage.
- [x] Implement `internal/exportsvx.Build(source.Document) (Result, error)` and
  `internal/cli/svx.go`, retaining P06 and checked report-last publication.
- [x] Validate actual cavern/dump3d 1.4.22 geometry, counts, lengths and flags;
  cover interrupted pairs, changed corrections/dates, ties, verticals and edited
  automatic templates. Preserve inputs and record consumer provenance.
- [x] Extend complete mutation/dispatch/weak controls and offline packages, run
  bounded ordinary checks and independent review, publication requires commit/push and confirmation of every exact-SHA CI job.
  Stop after P16.

Review focus: raw aliases versus emitted identity; trip changes within repeat
runs; inherited calibration/flags/inference/accuracy; invalid raw rows hidden by
P06 averages; escaped comments and inclusive output budget.

**Acceptance:**

- Produce `survey.svx` + `report.json`: individual readings in source order,
  reverse-repeat normalization, literal station dots and individual splays.
- Verify adjacent-repeat merging in Survex 1.4.22, including reversed readings,
  correction/date changes and interrupted pairs; check counts and splay flags.
- Verify manual declination transitions, commented date/auto templates, zero ties,
  vertical angles, strict failures and original/emitted-row lineage.

Delivered implementation: `export --format svx` emits individual source-order
rows with native-identity P06 reversal normalization, per-row correction/date
transitions, literal station dots, anonymous splays and original/normalized/
emitted report lineage. Processing/parser/P05–P15 behavior and frozen evidence
remain unchanged. Both destination formats reject zero/self native identities;
distinct-name zero ties remain supported. Empty SVX packages preserve metadata
but are explicitly outside cavern acceptance (`No survey data`).

The original PocketTopo reads all four new cases; eight exact native vector/
Float32 snapshots pass. Actual cavern/dump3d 1.4.22 validates nine independent
control graphs, counts and length statistics, plus inherited setting reset/
restoration, active dates/IGRF and an actual semantic compiler failure. All 150
new evidence files are frozen separately. Bounded check.sh passed with race,
vet, Staticcheck, build and 98.0% total statement coverage; parser/P14 controls,
53-fault compile-only preflight, complete dispatch and measured weak66 controls
passed. Full Gremlins discovery has 2792 RUNNABLE sites, no uncovered sites and
an exact disjoint five-shard union. Full mutation scores remain hosted.

Independent core/final reviews confirmed raw/source/report fidelity and actual
consumer behavior. Review identified and verified explicit zero-self rejection,
the empty-input consumer limitation, and the required GDAL build dependency in
the pinned-source Actions job. No substantive finding remains. Publication must
verify every platform, corpus, original Walls, new Survex, mutation/control job
and CI complete for the exact pushed SHA. P17 is not started.

Hosted validation killed all 1542 explicit faults, including 53 SVX faults,
but correctly rejected one compiler-invalid Gremlins pointer-bitwise mutation.
A local row copy/writeback removes that invalid site without changing frozen
SVX/report bytes or any operator. Named-failure replay closed two survivor gaps:
canonical normalized report bearings for reversed heads and the exact 160-byte
quoted comment boundary. The remaining four new survivors are equivalent
singleton, Float32 threshold, zero-bearing and incomplete-argument checks.
Independent review verifies those cases and exact correspondence of all earlier
50 output, 33 processing and 24 Toporobot survivors with published P15 evidence.
Fresh bounded ordinary/consumer checks and disjoint 2792-site discovery pass;
the repaired SHA must repeat every hosted job before delivery.

### SVX lexical settings correction — 2026-10-14

Requested scope: fix inherited decimal-comma and hash-comment parsing without
changing measurement processing or source data. Keep resets inside the anonymous
scope, before any emitted comment or numeric directive; restore caller settings
on return. Preserve all frozen P16 evidence and the pinned 1.4.22 hosted gate.

- [x] Reproduce both failures with real Survex regression tests, including caller
  settings restoration and the combined configuration.
- [x] Reset comment to `x3b` and decimal to `.`, retain independent byte/report
  expectations in a separate fixture directory, and add faults to the existing
  SVX mutation scope without removing previous faults.
- [x] Extend the consumer gate with independent graph/statistics comparisons for
  lexical settings; allow an explicit pinned 1.4.23 local run.
- [x] Run bounded ordinary checks, focused mutation/negative controls, Survex
  1.4.23 and independent review.

Local verification: complete `check.sh` passed with race, vet, Staticcheck, build
and 98.0% total statement coverage. Both actual cavern/dump3d 1.4.22 and 1.4.23
passed all nine earlier cases, three lexical graph/restoration cases, the inherited
settings case and the semantic negative control. All 1,967 prior frozen identities
and eight new identities verify. Independent review found no actionable issue.
The complete 56-fault SVX attempt stopped on a disk-exhausted compiler/linker;
no score is accepted from that interrupted attempt. A separate complete Go-overlay
campaign killed exactly the three new faults with named behavioral test failures.
Dispatch controls passed. Commit and push on `main` are authorized. Delivery
requires every hosted platform, consumer, corpus, mutation/weak-suite job and
`CI complete` to succeed for the exact published SHA; commit checks and the
delivery response retain that publication evidence. P17 is not started.

For both items, preserve P05–P15 and source/processing contracts. Follow all gates
in DEVELOPMENT, retain mutation scopes/failure controls, obtain independent review
and update fixture provenance. Changes to grouping or magnetic policy require
refining the contract. Commit/push needs authorization and exact-SHA hosted CI.

## Release 1.0.0 — accepted design, 2026-10-14

**Status:** accepted for implementation on 2026-10-14; the user authorized MIT,
direct work/commit/push on `main` and publication after exact-SHA green CI.
Implementation and publication are in progress. This is the next increment; P17
remains optional. Keep the design and implementation checklist here rather than
creating another backlog.

**Intent:** distribute the existing offline exporter so agents can automate it
and people can use it without installing a development toolchain. The user
requires command-line use at minimum and is interested in Windows right-click
export. The user accepted the Windows integration and six-target matrix below.

**Verified starting point:** local and remote `main` both point to
`d600cd0ffe6192f6cbf2db4d5b940e001930a9ad`; the working tree was clean before
this proposal. [CI run 38530570117](https://github.com/dlubom/pockettopo-exporter/actions/runs/38530570117)
passed all 31 jobs, including `CI complete`. No remote tags or releases were
returned by the live checks. Current distributions cover Linux amd64, Windows
amd64 and macOS arm64 and identify the CLI as `dev`. No project LICENSE file
exists in this checkout.

### Portable command-line release

- Publish six versioned ZIPs: Windows amd64/arm64, macOS amd64/arm64 and Linux
  amd64/arm64. Human download labels explain x64, Intel and Apple Silicon.
  This defines mainstream 64-bit desktop coverage; mobile, Windows CE, BSD and
  32-bit systems need separately requested and validated targets.
- Reuse the existing distribution builder and archive readback/offline tests.
  Ship a native executable, a concise QUICKSTART, reference documentation,
  approved license/attribution, BUILD.json and SHA256SUMS. Add an external
  checksum list covering every release archive. No Go, Python, Wine or PocketTopo
  installation is required to run the executable.
- Release builds report `1.0.0` through `--version` and the existing inspection
  tool-version field; local unversioned builds retain `dev`. Keep existing JSON
  schemas, export bytes, arguments, exit codes and stderr diagnostics. Record
  the exact source revision and actual toolchain in BUILD.json and the binary.
- Keep the current pinned Go toolchain for this release unless a separately
  verified reason requires an update. Publish its OS minimums separately from
  the OS versions actually exercised by CI; a build is not proof of runtime
  compatibility. Set `CGO_ENABLED=0` for portable release binaries.
- Put downloads and a first successful export before developer instructions in
  README. QUICKSTART covers PowerShell/cmd and Unix shells, paths containing
  spaces, selecting the right download, output directories, report.json and
  troubleshooting. Describe the supported TOP v3/profile boundary and native
  Therion interchange clearly. Advanced options remain available through CLI.

### Optional Windows Explorer integration

"Optional" means opt-in installation by the end user. The recommended release
scope includes shipping and validating the integration in 1.0.0; the user can
instead select a CLI-only release before implementation.

Propose a classic static submenu for `.top`: **PocketTopo Exporter > Export
as...**, offering Text, Survex, Walls, Therion interchange, VisualTopo, Toporobot,
DXF plan, DXF side and DXF 3D. Each entry invokes the same exporter with its
documented defaults. Start with one selected file; template context, closure
and advanced export settings remain explicit command-line options.

- Provide opt-in install/uninstall actions scoped to the current Windows user.
  Installation must not need administrator rights or change the default TOP
  application. Resolve the installed executable using its absolute path;
  document that moving the portable folder requires re-registering the menu.
- A menu action writes a new sibling directory such as `cave-survex`; existing
  directories use a bounded numeric suffix. Retain the actual no-replace
  publication guard against races. Never overwrite an existing export or source.
- Launch the executable directly, with correctly quoted arguments, rather than
  interpolating a TOP path into shell code. Verify spaces, Unicode and shell
  metacharacters in source and executable paths. Reject `%` in the executable's
  installation path explicitly before registry writes: Explorer interprets such
  command-template tokens even inside quotes. Normal CLI paths are unrestricted;
  source TOP paths containing percent tokens must pass actual Explorer invocation.
- Show the output location and report after success, and keep an error readable
  after failure. A console status prompt is sufficient; the interactive menu
  action must be separate from the noninteractive agent CLI.
- Windows 11 exposes this classic integration under **Show more options**.
  Direct placement in its modern menu requires `IExplorerCommand` and package
  identity and is a later improvement. Do not change Explorer's global menu
  settings. Full GUI authoring is outside this release design.

### Verification and publication

- Preserve the existing full mutation matrix, corpus/consumer jobs and negative
  controls. Exercise archives on native CI runners for all six targets; where
  a tool such as the race detector is unsupported, keep the existing supported
  race jobs and run the supported ordinary/runtime checks explicitly.
- Add meaningful release tests for version/manifest agreement, archive identity,
  six-target completeness and offline native-byte export. Windows tests verify
  registration readback, command arguments, uninstall, repeated export and
  visible errors. Include an actual Windows Explorer smoke check before claiming
  the menu works; registry readback alone is not Explorer acceptance.
- The release implementer owns the Explorer smoke check on an isolated native
  Windows Actions runner, starting with the existing x64 runner used by the
  original Walls GUI gate. First prove that an Explorer session can be exercised
  there, then capture menu presence, invocation and output evidence. Existing
  Walls success does not prove Explorer automation is available. If the runner
  cannot support that check, report the environment limitation and obtain a
  scope decision or another Windows test environment before publishing; do not
  quietly mark the menu gate successful or remove it.
- Keep signing status explicit. macOS Developer ID/notarization and Windows
  signing depend on separately available credentials/certificates. Do not claim
  signed packages or warning-free downloads without verified signing. A release
  without signing includes accurate platform-specific launch guidance.
- Commit/push the accepted release implementation directly on `main`, then
  require every applicable exact-SHA CI job and `CI complete` to succeed.
  Release automation must verify that the tagged revision has passed these gates
  and that all six packaged executables passed native archive tests.
- Create annotated `v1.0.0` at that verified revision, assemble a draft GitHub
  Release with all six archives, checksums and concise release notes, verify asset
  hashes/readback, then publish as the latest stable release. An incomplete or
  failed build must leave the release unpublished. Never silently reuse packages
  from a different revision or replace a published version's artifacts.

**License decision:** MIT was explicitly accepted by the user. Project copyright
uses the repository author's name, Dariusz Lubomski; third-party reference and
fixture terms remain separate.

### Release implementation plan

**Goal:** publish verified 1.0.0 archives for six native desktop targets.
**Architecture:** keep the exporter core unchanged; inject build version into
the existing CLI, add Windows shell commands in platform-specific source files,
and extend the existing archive builder and CI. Keep execution in this plan.
**Tech stack:** Go 1.26.3 standard library, Python standard library, GitHub
Actions, native Windows registry/Explorer integration.

**Review focus:** preserve no-replace outputs under collisions; quote Unicode
and shell metacharacters without command injection; keep install/uninstall
scoped to owned per-user keys; reject incorrect/missing archive version or SHA;
require complete native six-target evidence before publication.

1. [x] **Versioned portable release:** tests first for injected CLI/inspection
   version and archive version/target/manifest integrity. Update `internal/cli/run.go`,
   `internal/cli/inspect.go`, `scripts/distribution.py`, `scripts/test_validation.py`;
   retain dev defaults and native byte smoke tests. Builder accepts `--version`
   with default `dev`; release passes `1.0.0` and linker flag
   `-X pockettopo-exporter/internal/cli.Version=1.0.0`.
2. [x] **Windows integration:** tests first for sibling output/no-replace behavior,
   argument handling, per-user registry installation and removal. Add
   `windows-menu install|uninstall` and `shell-export <input.top> --format <format>`
   (DXF optionally `--view plan|side`) using platform-specific CLI files and
   `scripts/validate-windows-menu.ps1`. Main dispatches these interactive commands
   separately from `cli.Run`; shell export uses existing `cli.Run` export logic.
3. [x] **Native CI and release assets:** extend `.github/workflows/ci.yml` with
   archive/runtime jobs for six target runners and a required Explorer check.
   Add a script to assemble/verify all six packages from one successful exact-SHA
   run, rejecting missing/duplicate/foreign/unvalidated assets. Keep all existing
   jobs, mutation scopes and aggregate controls.
4. [x] **Documentation and MIT:** add `LICENSE`, `QUICKSTART.md`, concise release
   notes, and update README/SPEC/DEVELOPMENT/CAPABILITIES in place. Explain download
   selection, unsigned signing status, shell installation/removal and scope.
5. [ ] **Verification and delivery:** run bounded local checks, targeted package
   tests and evidence verification; obtain an independent review, fix actionable
   findings, commit/push on main, wait for all exact-SHA gates, verify archives,
   tag `v1.0.0`, create/verify draft assets and publish. Record final SHA/run/release
   evidence in release notes and machine-readable publication artifacts without
   claiming pending jobs passed. The [tagged release](https://github.com/dlubom/pockettopo-exporter/releases/tag/v1.0.0)
   and its exact-SHA CI link provide the final publication record; this committed
   plan records the source preparation stage before that external publication.

Local preparation passed bounded `check.sh` (race, vet, Staticcheck, build,
98.0% statement coverage and all frozen identities), distribution/assembly/hosted
release-gate negative controls, command-package profile/path/collision tests,
dispatch controls and the exact 66-live-mutant weak-suite control including
compiler/setup rejection. A concatenated version string initially introduced
two nonviable arithmetic mutations; ordinary formatted output restores the
original viable inventory without changing mutation controls. Independent
release/menu reviews found two actionable issues: false-green post-publication
filtering and Explorer executable-path placeholders. Both have tested fixes.
Windows amd64 code/test compilation and vet passed; native registry/Explorer and
all six actual release archives still require the final exact-SHA hosted run.
The first hosted attempt found a Windows-only negative-fixture issue: repeated
rename to the same foreign ZIP name relied on Unix replacement behavior. Each
target now uses its own foreign filename, preserving the filename-rejection
control without overwriting another fixture. Publication requires a fresh run
for the corrected revision; earlier successful target packages are not reused.

**Research references:** [GitHub release management](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository),
[native runner labels](https://docs.github.com/en/actions/reference/runners/github-hosted-runners),
[static Windows menus](https://learn.microsoft.com/en-us/windows/win32/shell/creating-static-cascading-menus),
[per-user file registrations](https://learn.microsoft.com/en-us/windows/win32/shell/fa-file-types),
[Windows 11 modern menu requirements](https://learn.microsoft.com/en-us/windows/apps/desktop/modernize/integrate-packaged-app-with-file-explorer),
[Go OS minimums](https://go.dev/wiki/MinimumRequirements).

## Remaining original roadmap

P14 closes the required native capability-inventory/distribution stage.
The remaining native profile gaps are explicit in CAPABILITIES; no universal
option or consumer-acceptance claim is made. P01's capability/fixture inventory
is closed by P14, not an additional standalone implementation chat. P15 and P16 add two subsequent
stages for Walls and Survex. P17–P19 and R01–R07 are optional/conditional scope.

These are meaningful outcomes, not a guaranteed count of future chats. P04d
delivers a complete reader and usable inspection; P05 delivers a usable export.
Geometry and drawing exporters may need subdivision after native research, but
only into independently testable behaviors. Individual fields, each Polygon,
routine documentation and individual mutation groups are not separate PBIs.

| ID | Outcome | Dependencies / refinement |
| --- | --- | --- |
| P01 | Native exporter/options and fixture capability matrix | Closed by P14, including explicit unsupported settings |
| P04d | Complete supported v3 source read and inspect | Implemented above |
| P05 | Isolated native text export through CLI, exact goldens and provenance/loss reporting | Implemented above; Update has no text-field effect, and geometry failure differences are explicit |
| P06 | Native fixed-point angles, consecutive-shot grouping and average directions | Implemented above |
| P07 | Reference placement, plan geometry, extended projection and closure | Implemented above |
| P08 | Explicit native template-directory context | Implemented above |
| P09 | Native Therion DATA and drawing sections, embedded XSections | Implemented above |
| P10 | Native graphics DXF, plan/side and options | Implemented above; native byte, coordinate and visual evidence |
| P11 | Native VisualTopo and passage dimensions | Implemented above |
| P12 | Toporobot and dialog options | Implemented above |
| P13 | Native 3D DXF | Implemented above; degenerate and bounded-failure cases |
| P14 | Close native capability matrix and validate distribution | Implemented above; exact-SHA hosted gates required |
| P15 | Walls SRV with averaged repeated legs | Implemented above; original Walls gate and exact-SHA CI required |
| P16 | Survex SVX retaining individual readings | Implemented above; pinned Survex consumer and exact-SHA CI required |
| P17 | Optional layered SVG | Accepted R03 contract; can follow native drawings before SRV/SVX |
| P18 | Optional XVI | Accepted R04 contract and native drawing semantics |
| P19 | Conditional TH2, PNG or PDF | Separate behavior-sized items only after the corresponding research decision |

This is dependency order, not a requirement to finish every P06–P08 feature
before any useful export. Preserve P05 as the first export milestone. Native
compatibility is established incrementally; P14 closes the documented profile
inventory while retaining explicit capability gaps.

## Optional research retained, not scheduled

| Item | Question | Evidence state / dependency |
| --- | --- | --- |
| R01 | Native Therion interchange through Caveink, including `.the` naming | Not validated; capture controlled/native cases |
| R02 | Native DXF through Inkscape | Historical structural smoke only; coordinate/GUI validation pending |
| R03 | Layered SVG contract | Provisional recommendation; depends on drawing/interoperability evidence |
| R04 | Direct XVI value | Await native interchange and XTherion/consumer comparison |
| R05 | Direct TH2 value | Await R03/R04; no inferred stroke semantics |
| R06 | Separate cross-section selection | Optional; resolve stroke ownership/cropping without inference |
| R07 | PNG/PDF through a renderer | Deferred until an explicit need and validated SVG |

The [historical research record](HISTORY.md#preserved-contracts-and-evidence)
retains exact probes, hashes and acceptance questions. Optional work does not
delay native exports unless it reveals a shared correctness issue. A tool failing
to run is an environmental limitation, not proof of format incompatibility.
