# Current state and delivery plan

This is the single current backlog and handoff. [SPEC](SPEC.md) defines product
contracts; [DEVELOPMENT](DEVELOPMENT.md) defines verification and delivery.
Historical PBI numbers remain traceable; they do not imply equal-sized outcomes.

## Verified baseline — 2026-10-11

Implementation baseline: `bfa73944e3d3baad00b07e2b6dd5a8c9f9a3682b` (P04c17).
The documentation reset changes no implementation, tests, fixtures or CI scripts.

| Area | Implemented behavior | Delivery state |
| --- | --- | --- |
| P00/P02 | Go skeleton, pinned tools and verified quality gates | Implemented |
| P03a–P03c2 | IDs, bounded v3 header/trips/measurements/references | Implemented internally; original inspect goal still pending |
| P04a/P04b | Separate overview and plan mappings | Implemented |
| P04c1–P04c13 | Fixed-position marker/count/point/color prefix APIs | Implemented historical contracts; preserve regressions |
| P04c14/P04c15 | Bounded ordered plan Polygon/XSection loops, explicit stops | Implemented |
| P04c16/P04c17 | Side mapping and first raw side marker | Implemented; side payload remains unread |
| Complete source document / inspect | Full side drawing, tail accounting, usable CLI inspection | Not implemented; next item P04d |
| P05–P14 | Native exports, processing, context and compatibility milestone | Not implemented |
| P15/P16 | Walls SRV / Survex SVX | Subsequent scope |

Fresh local `bash scripts/check.sh` passed during this audit: formatting, vet,
Staticcheck, boundary controls, race tests, build/CLI smoke and **100% statement
coverage** in all three internal packages. Full mutations were not rerun for
this documentation-only reset. The implementation baseline has
[successful exact-SHA CI](https://github.com/dlubom/pockettopo-exporter/actions/runs/38121341947):
all 14 jobs, including all 9 mutation groups, succeeded. This is evidence for
`bfa7394`. Publication of this documentation revision requires a separate green
CI run for its exact commit; retrieve that result from GitHub Actions rather than
assuming the implementation baseline's run covers a later commit.

### Git handoff

The user selected **direct work on `main`** on 2026-10-11. Local `main` was
fast-forwarded to P04c17 without rewriting commits. The follow-up request
authorizes committing this documentation and pushing `main` with P04c17, then
verifying all CI jobs for the exact pushed commit. This is direct publication;
no PR is needed. Use Git history and the matching GitHub Actions run for the
resulting commit and CI status, keeping live delivery evidence out of the spec.

Old P04c15/P04c16 branches contain no changes missing from local `main`;
P04c17 is also included locally. They may be removed in a separately requested
branch cleanup. Future implementation should not resume on those branches.

## Next recommended item: P04d — complete v3 source reading and inspection

**Status:** specified for the next implementation request; not implemented here.
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

**Refinement rule:** settle routine API names and representation in code review.
Record native boundary findings in SPEC/evidence before relying on them. A new
dependency that changes the outcome or adds geometry/export work requires
revising this PBI explicitly, not returning to arbitrary one-field tasks.

**Ready handoff prompt:**

> Implement only P04d from docs/PLAN.md: complete TOP v3 source reading and
> inspection. Read AGENTS.md, docs/SPEC.md and docs/DEVELOPMENT.md first, and verify
> the current checkout. Work directly on main and preserve all existing prefix
> contracts and native evidence. Deliver complete bounded side drawings, explicit
> full-file/tail accounting and the specified inspect summary as one coherent
> increment. Use native C#/IL and independent probes, run the applicable full
> gates, review the result and update the plan. Do not implement exporters or
> split individual drawing fields into separate PBIs. Report local verification
> and publication state separately; follow the current request's commit/push
> authorization. Stop after P04d and identify the next useful step toward P05.

## Remaining original roadmap

There are **11 required delivery stages to the native milestone**: P04d and
P05–P14. P01 is a capability/fixture inventory maintained within those stages,
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
| P04d | Complete supported v3 source read and inspect | P04c17; next item above |
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
