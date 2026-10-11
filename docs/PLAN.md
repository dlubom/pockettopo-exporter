# Current state and delivery plan

This is the single current backlog and handoff. [SPEC](SPEC.md) defines product
contracts; [DEVELOPMENT](DEVELOPMENT.md) defines verification and delivery.
Historical PBI numbers remain traceable; they do not imply equal-sized outcomes.

## Current implementation — 2026-10-11

P04d started from P04c17 (`bfa73944e3d3baad00b07e2b6dd5a8c9f9a3682b`) plus
the documentation reset (`94a3f2e`). P05 builds on verified P04d
(`e3b3b3a395741757d1cf2527b192bfceeae5fa52`). Existing prefix contracts remain unchanged.

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
| P06–P14 | Processing, context, other native exports and compatibility milestone | Not implemented |
| P15/P16 | Walls SRV / Survex SVX | Subsequent scope |

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

**Status:** implemented and locally verified. **Dependency:** P04d.
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
report last. Cleanup retires temporary names and checks original file identity.

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
No substantive findings remain. Seven final Gremlins survivors were reviewed:
five equivalent padding/numeric branches and two noncritical cleanup-diagnostic
variants, with no critical survivor.

Final local verification passed: `check.sh` (race, vet, Staticcheck, coverage
controls, build and CLI checks; 99.5% statement coverage), all 621 Gremlins
mutants (614 killed, 7 lived; 98.87%; no invalid/uncovered/timed-out outcomes),
and all 1,057 explicit compile-and-test faults in 25 scopes. Full scope was
verified with the complete campaign plus focused native-text/Gremlins reruns
after correcting a noncompiling planned fault and strengthening tests. All 18
weak-suite mutants live and the gate rejects them; compiler/setup failure and
exactly-once dispatch/pruning controls also pass. Seven native byte goldens,
all 16 new evidence hashes, local documentation links and staged whitespace
checks pass. The built CLI independently matches the api-drawings text golden
and both report hashes. The commit's exact-SHA GitHub checks and delivery
response provide publication evidence; no earlier SHA substitutes for it.

### Next recommended item: P06 — native grouped measurement processing

Deliver native fixed-point vectors and consecutive matching/reversed-shot groups
with their complete source-row lineage and native average directions. Establish
real repeated/reversed/across-trip and 0/360-degree cases against original
EvalData/Angle/GetAvDirection in C#/IL and independent probes. Keep raw source
records separate from processing and preserve P05 text bytes; they deliberately
emit all individual source rows. Do not include placement, closure, template
context or another exporter. Finalize the useful API/inspection boundary in P06
without splitting arithmetic fields into separate PBIs. Stop after P06 when it
is explicitly requested; this P05 chat stops here after publication.

## Remaining original roadmap

After P05 there are **9 remaining required delivery stages to the native
milestone**: P06–P14. P01 is a capability/fixture inventory maintained within those stages,
not an additional standalone implementation chat. P15 and P16 add two subsequent
stages for Walls and Survex. P17–P19 and R01–R07 are optional/conditional scope.

These are meaningful outcomes, not a guaranteed count of future chats. P04d
delivers a complete reader and usable inspection; P05 delivers a usable export.
Geometry and drawing exporters may need subdivision after native research, but
only into independently testable behaviors. Individual fields, each Polygon,
routine documentation and individual mutation groups are not separate PBIs.

| ID | Outcome | Dependencies / refinement |
| --- | --- | --- |
| P01 | Native exporter/options and fixture capability matrix | Capture the relevant part with each exporter; finish full inventory by P14 |
| P04d | Complete supported v3 source read and inspect | Implemented above |
| P05 | Isolated native text export through CLI, exact goldens and provenance/loss reporting | Implemented above; Update has no text-field effect, and geometry failure differences are explicit |
| P06 | Native fixed-point angles, consecutive-shot grouping and average directions | Measurement source model; real repeated/reversed/boundary cases |
| P07 | Reference placement, plan geometry, extended projection and closure | P06; split by complete geometry behavior if necessary |
| P08 | Explicit native template-directory context | P07; deterministic inventory, ordering and isolation tests |
| P09 | Native Therion DATA and drawing sections, embedded XSections | P04d/P07/P08; complete data/drawing slices if required |
| P10 | Native graphics DXF, plan/side and options | P04d/P07/P08; native bytes plus coordinate/visual checks |
| P11 | VisualTopo and passage dimensions | P06/P07/P08 |
| P12 | Toporobot and dialog options | P06/P07/P08 |
| P13 | Native 3D DXF | P07/P08; degenerate and bounded-failure cases |
| P14 | Close native capability matrix and validate distribution | P05/P09–P13, pinned corpus and complete gates |
| P15/P16 | Walls SRV / Survex SVX | Native milestone; explicit conversions/losses and consumer validation |
| P17 | Optional layered SVG | Accepted R03 contract; can follow native drawings before SRV/SVX |
| P18 | Optional XVI | Accepted R04 contract and native drawing semantics |
| P19 | Conditional TH2, PNG or PDF | Separate behavior-sized items only after the corresponding research decision |

This is dependency order, not a requirement to finish every P06–P08 feature
before any useful export. Preserve P05 as the first export milestone. Native
compatibility is established incrementally; claim full coverage only at P14.

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
