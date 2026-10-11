# Current state and delivery plan

This is the single current backlog and handoff. [SPEC](SPEC.md) defines product
contracts; [DEVELOPMENT](DEVELOPMENT.md) defines verification and delivery.
Historical PBI numbers remain traceable; they do not imply equal-sized outcomes.

## Current implementation — 2026-10-11

P04d starts from P04c17 (`bfa73944e3d3baad00b07e2b6dd5a8c9f9a3682b`) plus
the documentation reset (`94a3f2e`). Existing prefix contracts remain unchanged.

| Area | Implemented behavior | Delivery state |
| --- | --- | --- |
| P00/P02 | Go skeleton, pinned tools and verified quality gates | Implemented |
| P03a–P03c2 | IDs, bounded v3 header/trips/measurements/references | Implemented; exposed by P04d inspection |
| P04a/P04b | Separate overview and plan mappings | Implemented |
| P04c1–P04c13 | Fixed-position marker/count/point/color prefix APIs | Implemented historical contracts; preserve regressions |
| P04c14/P04c15 | Bounded ordered plan Polygon/XSection loops, explicit stops | Implemented |
| P04c16/P04c17 | Side mapping and first raw side marker prefix APIs | Preserved historical contracts |
| P04d | Full supported v3 source document, side drawing, tail accounting, inspect JSON | Implemented; full local gates verified |
| P05–P14 | Native exports, processing, context and compatibility milestone | Not implemented |
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

The current user request authorizes direct work, commit and push on `main`.
No branch/worktree/PR or exporter implementation is part of P04d.

## P04d — complete v3 source reading and inspection

**Status:** implemented and locally verified; ready for P05 after exact-SHA CI.
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

## Next recommended item: P05 — native text export

**Status:** specified; not implemented. **Dependency:** P04d. Deliver a usable
`export` CLI for the native text format with exact original-program goldens,
explicit isolated-input context, deterministic output/provenance/loss reporting
and safe output publication. Research `Exporter.WriteText`, `Survey.WriteText`,
`Station.WriteText` and the text-relevant `Survey.Update` effects in C#/IL and
independent native probes before implementing. Promote only demonstrated P06/P07
prerequisites; do not assume all geometry is necessary or bypass Update silently.
Do not bundle other exporters or template-directory behavior.

**Ready handoff prompt:**

> Implement only P05 from docs/PLAN.md: a usable native PocketTopo 1.372 text
> export through the CLI, building on P04d. Read AGENTS.md, README.md, SPEC, PLAN
> and DEVELOPMENT; verify the checkout and work directly on main. Establish
> Exporter.WriteText/Survey.WriteText/Station.WriteText and the text-relevant
> Survey.Update semantics in original C#/IL plus independent native probes.
> Implement only demonstrated processing prerequisites, preserve raw source
> identities/bytes and all P04d/prefix contracts. Deliver exact isolated-input
> native goldens, safe explicit output publication and provenance/loss reporting,
> with normal/boundary/failure cases, full quality/mutation gates, independent
> review and updated documentation. Do not add other exporters or directory
> context. After verification, commit/push main and confirm green CI for its
> exact SHA. Stop after P05; report behavior, limitations and the next useful
> behavior-sized increment.

## Remaining original roadmap

After P04d there are **10 remaining required delivery stages to the native
milestone**: P05–P14. P01 is a capability/fixture inventory maintained within those stages,
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
| P05 | First usable isolated native text export through CLI, exact goldens and provenance/loss reporting | P04d; establish text-relevant `Survey.Update` effects and promote only demonstrated P06/P07 prerequisites |
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
