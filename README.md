# PocketTopo Exporter

A standalone project for exporting PocketTopo `.top` survey files to multiple
formats, based on the PocketTopo 1.372 decompilation.

## Status

Planning draft, 2026-10-09. No implementation has been requested. This document
records the proposed scope, acceptance criteria, and delivery order; it is not a
claim of implemented support. No code, dependencies, tests, or workflows exist.

Confirmed priorities:

- Reproduce the native exporters of the standard PocketTopo **1.372** first.
- Preserve measurements, plan sketches, extended elevations, and cross sections.
- Add Walls SRV and Survex SVX after native export compatibility.
- Keep historical comparison and reconciliation **in JKTZ**, which will consume
  this tool's outputs. Do not build a comparison engine in this project.
- Keep the implementation and development process small and understandable.

Go is the current recommendation, not a selected implementation language.
The language, supported first-release formats, and executable commands remain
decisions for an explicit implementation request.

**Next handoff: R01, native Therion interchange → Caveink.** Start the bounded
research item below in a fresh chat. Do not start P02 or exporter implementation.

## Reference material

- [PocketTopo 1.372 decompilation](../pockettopo-decompilation/README.md):
  the main reference for file reading, station identifiers, measurements,
  sketches, geometry, and native export behavior. Start with
  [the code map](../pockettopo-decompilation/analysis/ANALYSIS.txt) and
  `../pockettopo-decompilation/decompiled/csharp/PocketTopo/`.
- [Earlier JKTZ experiments](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/tree/master/src/jktz/pockettopo):
  a Python parser, data model, Walls SRV and Survex SVX exporters, drawing export,
  and conversion reports. Treat these as prior work to review against the
  decompiled program, rather than an authoritative format specification.

## Product boundary

A portable, offline CLI for people and agents. It reads explicitly selected TOP
inputs, exports requested formats, and explains what was emitted, transformed,
or omitted. Running it does not require PocketTopo, Wine, or a graphical session.
The original application is a validation reference during development.

In scope: source inspection, native-compatible export, additional format export,
and a small machine-readable provenance/loss report. Out of scope: editing or
repairing archives, matching stations between historical surveys, deciding which
survey is correct, automatic date/declination correction, cave catalog logic,
Bluetooth, instrument calibration, a GUI, and rewriting TOP files.

Preserve embedded dates as recorded values. Independent survey dates belong to
the consuming project; neither a filename nor an embedded clock value proves
the field date. Do not infer a CRS, entrance, or geographic orientation.

## Native export inventory

These paths were found in the recovered 1.372 C# and UI resources. This is a
source inspection, not a completed native validation campaign. Extensions loaded
from additional DLLs and other PocketTopo versions are outside this baseline.

| Native export | Output | Scope and reference |
| --- | --- | --- |
| Text | `.txt` | Trips, references, measurement rows; `Exporter.WriteText`, `Survey.WriteText`, `Station.WriteText` |
| Therion interchange | `.txt` | PocketTopo interchange with DATA, PLAN, ELEVATION, stations, shots and sketch elements; `Therion.WriteExport` |
| Toporobot | `.text` | Survey export and its dialog options; `ToporobotExp`, `ToporobotForm` |
| VisualTopo | `.tro` | Survey export and calculated passage dimensions; `VisualTopoExp` |
| Graphics DXF | `.dxf` | Plan and/or extended elevation, drawings, shots, cross sections, labels, grid, layers and scale options; `GraphicsForm`, `DXFWriter` |
| 3D DXF | `.dxf` | Geometry built from survey legs and splays; `ThreeD.WriteDXF` |

The native Therion interchange is not a ready-made `.th`/`.th2` project. Direct
Therion project authoring would be a separate enhancement. Native 3D DXF is not
Survex `.3d`. Walls `.srv`, Survex `.svx`, SVG, and PNG are additional targets,
not exporters found in this baseline. The Inkscape research below recommends SVG
as the first additional drawing target, directly after native drawing validation.
PNG can follow if needed, with its rendering dependency outside the core.

"All native exports" includes each documented exporter option and relevant
input/context combination, not only one successful file per extension. Delivery
can be incremental, with an explicit capability matrix for each release.

### Inkscape interoperability research, 2026-10-09

This extends the planning scope; it does not authorize implementation. Historical
survey comparison remains in JKTZ. External importers are interoperability probes,
not replacements for PocketTopo as the behavioral reference.

| Route | Evidence and implication |
| --- | --- |
| TOP → Inkscape via [inkscape-speleo](https://github.com/speleo3/inkscape-speleo) | The extension invokes `topreader.py` to create SVG. Useful for studying layers and station annotations; its geometry rules differ from PocketTopo. |
| Native Therion TXT → Inkscape via [Caveink](https://github.com/mteg/caveink/blob/master/extensions/the_input.inx) | The importer recognizes `.the`. Validate that suffix with unchanged native interchange bytes in R01 before promising compatibility; it is not a separate format. |
| Native DXF → Inkscape | The installed 1.4.4 importer successfully processed two preserved native DXF fixtures in the smoke test below. It handles old POLYLINE/VERTEX/SEQEND records despite its stated R13+ scope. |
| Native Therion TXT → XTherion → XVI | [Therion documents this workflow](https://therion.speleo.sk/wiki/pockettopotherion). It creates a drawing background; it does not automatically identify semantic walls, pits or areas. |
| XVI/TH2 → Inkscape via inkscape-speleo | The project provides input extensions for both; TH2 also has an output extension. Test these as consumers of future exports. |

Candidate additions to the format plan, subject to the research decisions below:

- **Layered SVG: first additional drawing target.** Plain SVG geometry with
  Inkscape layer attributes; plan and extended elevation can be separate files.
  Separate sketch, legs, splays, stations/labels and cross-section connectors.
  Preserve original colors and vertex order, use explicit physical units and
  viewBox, and include all visible geometry in page bounds. Keep stable source
  element references. Inkscape is a consumer, not a runtime dependency.
- **XVI: next useful interchange target.** Export plan/extended backgrounds,
  station labels, survey lines, colored sketch lines and scale/grid information.
  Validate against XTherion and the inkscape-speleo reader. This is an additional
  exporter, not PocketTopo's native Therion TXT renamed to `.xvi`.
- **TH2: optional later authoring aid.** Decide whether to provide a station/scrap
  scaffold linked to an XVI background or raw lines with an explicit mapping.
  Do not interpret a source color as a wall/pit symbol without a user-supplied
  convention. A faithful drawing export is not a completed semantic cave map.
- **PNG/PDF: derived previews or print outputs.** Add through a renderer only when
  required; keep editable vectors as the primary output. Do not add a new core
  geometry implementation for either format.

SVG with Inkscape layers is one SVG output profile, not two separate formats.
Caveink's projected-elevation option derives a new view from native data; do not
label that result as PocketTopo's native extended elevation.

#### Checks performed now

Inspected inkscape-speleo at `64b13a199e7820b2caa620e254ec2a62344f96f7`
and Caveink at `87b182334435ef37da7f8d7cee68bbc529d77d7b`. Verified the tested
topreader source against Git blob `9835597a83ea664d68bba060b4e91ee88c5c2032`.
Its grouping combines matching pairs across the input rather than only adjacent
records; its averaging differs from the native integer-vector calculation.
Its SVG cross-section helper draws station-to-anchor connectors without the
native projected splay fans at those anchors.

Ran two upstream TOP→SVG smoke cases and two native DXF→SVG importer cases using
existing JKTZ fixtures, without changing inputs or installing extensions. The
DXF reader was the one bundled with local Inkscape 1.4.4, executed headlessly
with the Codex Python runtime and existing pure-Python Inkscape dependencies.
This tests the importer script, not the File/Open GUI or visual fidelity.

| Fixture | TOP→SVG | Native plan DXF→SVG |
| --- | --- | --- |
| `shadow/shadow.top` | Exit 0; valid SVG with planview/sideview layers | Exit 0; Sketch, Shots and XSect layers |
| `p05/native/xsection/xsection.top` | Exit 0; connectors present, native cross-section splay fans absent | Exit 0; 42 paths in XSect, matching the native DXF's 42 XSect LINE entities by count |

The native XSection case also has five connector LINEs on Sketch. Path/entity
counts are a structural check, not coordinate-by-coordinate equivalence. The
direct TOP SVG sets a 0..10 by 0..10 viewBox for this no-polyline fixture while
some transformed geometry falls outside it: empty-sketch bounds need regression
coverage. All four source files remained unchanged. Native PocketTopo itself was
not run in this research pass; these are preserved native exports.

Reproduction inputs are under
`../Jaskiniowy-Kataster-Tatr-Zachodnich/tests/fixtures/pockettopo/`:

| Input | SHA-256 |
| --- | --- |
| `shadow/shadow.top` | `a6554019285dcc50feadb672867c40e5bef0686551060da1670b49cde0bcfb4c` |
| `shadow/shadow-nativeP.dxf` | `f7af47fafcea84971fe9b03b18ed79a9bd1e24f3e192cbd1110a87c88585eabe` |
| `p05/native/xsection/xsection.top` | `4a591d3f42676d03e9578e016010991843c758c6462f7a22802bef10789c924f` |
| `p05/native/xsection/native-plan.dxf` | `dbead8750ca3343e09fba29202b2bec46e4ec2ff4bf79f650949fa726941942d` |

These preliminary results do not close any research item below. The next steps
have explicit boundaries and acceptance criteria in the research backlog.

### Scope and decision status

Scope priority and implementation authorization are separate. Confirmed scope
still requires an explicit implementation request. A successful third-party
import does not automatically promote an optional feature into required scope.

| Item | Scope status | Remaining decision/evidence |
| --- | --- | --- |
| Native 1.372 exports, including drawings and embedded XSections | Confirmed primary scope | Complete native contracts and fixture coverage; P01 and relevant research findings |
| Walls SRV and Survex SVX | Confirmed subsequent scope | Format-specific conversion/loss contracts and consumer checks before P15/P16 |
| Caveink `.the` interoperability | Candidate output naming/consumer support | R01; same native interchange bytes, no extra serializer |
| Inkscape support for native DXF | Candidate documented consumer workflow | R02; preserve the native writer even if an importer has limitations |
| Layered SVG | Recommended optional exporter | R03; P17 is conditional |
| XVI | Optional exporter | R04; P18 is conditional |
| Direct TH2 | Optional authoring aid | R05; no automatic interpretation of colored strokes |
| Separate files for individual cross sections | Optional selection feature | R06; not required to preserve embedded native XSections |
| PNG/PDF | Optional renderer outputs | R07; no new geometry engine |
| Cucumber runner | Deferred optional tooling | Revisit only if normal tests plus PBI scenarios are insufficient |
| Go, mutation tooling and numeric quality gates | Recommendation/proposals | P00 decision and P02 validation after implementation is requested |

Research outcomes are **recommend**, **defer**, or **reject**, with evidence and
the precise supported subset. An environmental blocker means **not validated**;
it is not proof of format incompatibility. Record a conditional recommendation
when remaining evidence is named. Optional work must not delay the native export
milestone unless it exposes a defect in shared native behavior.

### Bounded research backlog

Status on 2026-10-09: R01 is **ready / not started**; R02 is **ready / preliminary
smoke evidence only**; R03–R05 await the dependencies below; R06 is **ready /
optional, not scheduled**; R07 is **deferred**. Work on one item per chat. Research
may inspect C#/IL and existing importer code and run existing applications on
working copies. It does not authorize production code, new project dependencies,
an automated test harness, global extension installation, or repository setup.

| ID | Question and bounded work | Acceptance / decision | Depends on |
| --- | --- | --- | --- |
| R01 | Does the native Therion interchange work through Caveink? Export one controlled XSection fixture and one real test2 fixture from original 1.372; test unchanged bytes under `.txt`/`.the` names, plan and extended views. Inspect `Therion`, `Survey.WriteThSketch*`, `XSection`, `thetosvg.py` and its input descriptor. | Record capture settings, hashes and importer version. Check station labels, sketch vertices/colors, scale, XSection connectors and projected splays. Classify losses per case and recommend/defer the `.the` naming profile. No projected-elevation feature work. | Current native references and existing smoke findings; no language decision needed |
| R02 | Does the Inkscape DXF workflow preserve native drawings? Extend the two smoke cases to coordinate checks and actual Inkscape opening; add one native case with labels/grid/color options. | Compare scale, axes, layers, colors, labels, polyline vertices, XSection fans and connectors; distinguish parser success from visual/coordinate match. Record supported options or consumer limitations. Do not modernize native DXF bytes to hide importer defects. | Existing DXF fixtures and installed Inkscape |
| R03 | What is the smallest useful SVG contract? Use R01/R02 and inspected importer code to specify layers, units, labels, metadata, bounds, plan/extended separation and XSection rendering. | A concrete SVG acceptance matrix covering empty sketches, single-point strokes, all colors, off-origin sections and full page bounds; name the required native geometry dependency. Recommend/defer P17. No new SVG writer. | R01 and R02, or recorded blockers with an explicitly provisional contract |
| R04 | Is a direct XVI exporter worth adding? Obtain an XVI through the documented XTherion import of captured native Therion TXT and inspect it in XTherion and inkscape-speleo. | Record how stations, lines, colors, scale/grid and sections survive each route. Distinguish drawing background from editable semantic map. Recommend/defer P18 and state its minimal contract. | Native TXT capture from R01; existing XTherion/importer availability |
| R05 | Does TH2 add value beyond SVG/XVI? Inspect existing TH2 workflows using the same small case; compare a station/scrap scaffold with an XVI background against an explicit raw-line mapping. | Describe user work still required, metadata and supported geometry; recommend one narrow profile, defer, or reject. Any color-to-symbol mapping requires a stated convention, never inference. No semantic map generator. | R03 and R04 |
| R06 | Can individual cross sections be selected without inventing stroke ownership? Inspect overlapping/adjacent anchors and polylines in a controlled fixture and a real input. | Establish what is stored versus inferred; compare explicit element selection and user-defined crop. Recommend a selection contract or defer separate files. Full-view native XSections remain required regardless of outcome. | Native XSection semantics; no dependency on optional format implementation |
| R07 | Are raster/print outputs needed, and can an existing renderer produce them reliably? First record an explicit user need, then evaluate one renderer on validated SVG examples. | Confirm physical size/page bounds, fonts, colors, transparency, platform/dependency cost and headless operation; propose PNG and PDF separately. No renderer bundling before this decision. | User need and validated SVG output; deferred until available |

Each completed research item updates this README with: status; tested question;
input hashes and provenance; reference methods/importer revision; environment
and options; actual results versus expected behavior; unresolved differences;
recommend/defer/reject disposition; affected PBI dependencies; and exactly one
next ready item. Capture actual reproduction steps after running them, rather
than inventing project commands. Existing unverified findings remain labelled.
Do not depend on temporary output paths for the next chat: record fingerprints
and reconstruction steps here. An evidence archive can be introduced when project
structure is explicitly authorized.

#### R01 handoff details

- Controlled input: `p05/native/xsection/xsection.top`; real input:
  `shadow/shadow.top`, both relative to the JKTZ fixture directory listed above.
  Shadow's documented source is test2's
  `502-Schattenhohle/20120816-Shadow/shadow.top` at the pinned corpus revision.
- Verify current source hashes before testing. Read the fixture README for
  provenance. Use an isolated directory containing only the active working copy
  for each native capture; do not accidentally include neighboring templates.
- Confirm the native executable hash. Record relevant settings, locale and
  Wine/runtime versions. Capture both DATA and drawing sections without changing
  the original TOP. Check the input hash again afterwards.
- Check the chosen Caveink revision and options before running it. Test plan and
  extended modes; projected elevation is outside this item. Compare the source
  interchange coordinates/elements with imported SVG and inspect the visual
  result where the available environment permits.
- Report separately: extension recognition, importer execution, SVG structure,
  scale/coordinate agreement and visual checks. A missing application/module or
  unavailable GUI leaves the affected check unverified; complete independent
  source analysis and document the blocker without manufacturing a pass.
- End after recording R01's result and recommendation. R02 is the next intended
  item if ready; do not start it in the same chat. Do not implement P09 or P17.

## What compatibility means

The reference is the standard 1.372 executable with SHA-256:

```text
048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371
```

Use the original program's observed output as the behavioral reference, C# to
explain the implementation, and IL to resolve arithmetic/control-flow ambiguity.
Record disagreements between those sources instead of silently choosing a result.
Do not generalize these findings to the Swiss 1.472 build.

For every exporter, specify inputs, settings, record selection/grouping, numeric
conversion, ordering, text encoding, line endings, and representational losses.
Target byte-identical native files under the same controlled conditions. Where
that is not yet achieved, report semantic equivalence separately with a specific,
reviewed normalization rule; never erase rounding differences to obtain a pass.
Treat byte match, semantic match, and target-application acceptance as separate
results. A compiler accepting a file proves neither completeness nor identity.

Compatibility concerns output behavior for supported data. We will report I/O
errors and malformed/unsupported data even where the native program silently
catches exceptions. Resource bounds and non-convergence failures are explicit
deviations; the CLI must not hang or report a failed write as a success.

### Findings that shape the design

1. **Directory context is input.** `DataSet.Load` sorts and reads neighboring
   `*.top` files as templates before the active file. Use isolated inputs by
   default; support an explicit native-directory context with a recorded file
   inventory and ordering. Never scan arbitrary neighboring files implicitly.
   Claims of compatibility must name which context was used.
2. **Raw ID, internal ID, and displayed name differ.** `ID.Read`, `ToString`, and
   equality are separate operations. For example, raw `0x800fffff` decodes to
   internal `-2`, whose display is empty, but it is not `UNDEF` (`-1`). Preserve
   all three representations; never use display text as identity.
3. **Geometry uses native arithmetic.** `Survey.EvalData` groups consecutive
   matching/reversed station pairs and averages integer vector components.
   `Angle.ToRect` uses a fixed-point sine table. `Station.GetAvDirection` uses
   that geometry and float conversions. Signed division, shifts, overflow,
   float32 steps and .NET formatting require explicit compatibility tests.
   Exporters do not all serialize the same record representation.
4. **Native grouping is not a field assertion.** Reproduce the reference rules
   without requiring the user to prove repeated observations. Record the source
   rows contributing to the export. Do not add JKTZ's same-trip confirmation
   restrictions to native grouping without evidence from the relevant method.
5. **Projection and loop closure affect drawings.** `Projection`, `Survey`, and
   `Loop` must be studied together. The value 10 in the closure stopping
   condition is an error threshold, not a ten-iteration limit.
6. **Cross-section anchors are not sketch containers.** `XSection` stores a
   center, station ID, and direction; `-1` denotes the horizontal case in the
   inspected creation path. Surrounding polylines are separate drawing elements.
   Preserve native placement, projected splays and connectors in complete views.
   Exporting each complete hand-drawn cross section into its own file needs an
   explicit selection/cropping rule; proximity alone does not prove ownership.
7. **Automatic declination is version-specific.** In this standard build,
   `Trip.GetDeclination` returns the supplied declination rather than calculating
   it from date and position. Reproduce that behavior; do not add a magnetic
   model behind an existing flag.
8. **Old TOP versions need evidence.** `DataSet.Read` contains version-dependent
   branches. Start with a documented v3 support target; inventory older versions
   and add support deliberately. The native version condition alone is not proof
   that every possible old header is valid or that our parser supports it.

### Reuse from JKTZ

The local module was reviewed at commit
`dfac3a80e1757a8ea85ae0ad5f595e16b7eb53c7`, with no local module changes reported.
Useful ideas: immutable source records, input fingerprints, record provenance,
small fixtures, loss accounting, and protection against overwriting outputs.

Do not transplant the processing policy. Its `averaging.py` computes arithmetic
distance/inclination means and a circular azimuth mean; `grouping.py` requires
explicit repeat confirmations. Those serve a different contract from native
PocketTopo reproduction. Parser and drawing behavior also need reference checks
before reuse. Avoid importing the old mandatory multi-artifact package workflow:
one requested exporter should be usable independently.

## Architecture without a framework

Keep five responsibilities clear within one codebase:

1. A bounded binary reader and source model: original fields, ordered records,
   byte offsets, drawing elements, mappings, and source fingerprint.
2. Native processing: explicit context/settings and derived grouping, geometry,
   projection and loop closure. Never mutate the source model.
3. Format writers: choose the native source or derived representation required
   by that exporter. Keep format-specific selection and rounding visible.
4. A thin CLI: arguments, file access, output publication, and diagnostics.
5. A compact report: links from emitted elements to source records and known
   losses, including native omissions that must still be reproduced.

These are boundaries, not a requirement for five packages or many tiny files.
Start with ordinary modules and functions. Add an abstraction when multiple real
callers need it. Avoid a plugin system, dependency injection framework, database,
service layer, or separate implementation per platform. Concurrency is not needed
to establish correctness. Large batch conversion can be added after single-file
behavior is reliable.

## CLI contract to settle before implementation

Proposed operations are **inspect** and **export**. There is no compare/reconcile
operation. These are interface proposals, not existing runnable commands.

- Accept explicit input and output paths and a stable format identifier. One
  export per invocation is sufficient initially. Native plan/side DXF may
  naturally produce two files.
- Expose native options explicitly; document defaults and include resolved values
  in the report. Default to a single-file context; templates require an explicit
  context selection. Record active versus template records separately.
- Provide human-readable diagnostics and versioned JSON for agents. Keep machine
  output free of progress messages; send diagnostics to stderr.
- Preserve inputs byte-for-byte. Refuse input/output collisions and accidental
  overwrites. Write through temporary outputs and publish only completed results.
  Define the two-file/report failure behavior before claiming atomic export sets.
- Emit only requested artifacts. A requested sidecar report contains tool/schema
  version, source SHA-256, context hashes/order, options, counts, output hashes,
  source record references, transformations, and omissions. No absolute local
  paths or timestamps are needed in deterministic output by default.
- Identify a source record by source fingerprint, record kind and index/offset.
  Never replace this with a station name or generated geometry.
- Distinguish target limitations from unexpected omissions. For example, a survey
  format may intentionally lack sketches. Report this without requiring every
  export invocation to generate every other available format.
- Proposed exit meanings: 0 = requested export completed under its documented
  contract; 1 = failure/no completed result; 2 = explicitly requested partial
  result. Final values remain to be fixed with the CLI design. Reports separately
  describe native compatibility, source information loss, and validation status.
  Unexpected loss must not be hidden behind exit 0.

## Language decision

**Recommendation: Go for the standalone CLI**, conditional on proving numeric
compatibility and a usable mutation-testing workflow in the first implementation
slice. This is a distribution/maintenance choice, not a claim that Go inherently
produces more correct exporters. Do not build two production implementations.

| Consideration | Go | Python |
| --- | --- | --- |
| Distribution | A platform-specific executable is a natural delivery unit; aim for a pure-Go core | Requires an interpreter/environment or an additional packaging step |
| Native arithmetic | Fixed-width types help, but .NET rounding, shifts and float operations still need deliberate treatment | Binary reading is straightforward; unbounded integers and negative floor division require deliberate emulation |
| Testing | Standard testing, fuzzing and coverage tools; evaluate mutation tooling early | pytest, Hypothesis, coverage.py and mutmut form a strong testing option |
| Existing work | Rewrite verified behavior with a compact design | Existing JKTZ ideas can accelerate work, but its policy is not native compatibility |
| Drawing outputs | Text-based DXF/SVG do not require a GUI; defer rasterization | Broad analysis/rendering ecosystem; avoid making it a runtime requirement unnecessarily |

For Go, start with gofmt, go vet, Staticcheck and standard tests. Evaluate
go-mutesting or another maintained candidate against the selected Go version;
do not claim the mutation gate works until it detects seeded behavioral defects.
For Python, the alternative is Ruff formatting/linting, one strict type checker,
pytest, Hypothesis, coverage.py, and mutmut on Linux. Pin the selected tools when
implementation begins. Do not select both stacks or add a custom quality platform.

C# would reduce some semantic translation, but the recovered application is tied
to its old UI/runtime and is not a ready-made portable core. It remains an option
if numeric compatibility proves disproportionately difficult; no parallel port
is proposed now.

## Validation strategy

### Original PocketTopo as a behavioral reference

The installed executable at
`~/.local/share/pockettopo/app/PocketTopoV1372/PocketTopo.exe` was hashed on
2026-10-09 and matches the reference SHA-256 above. Its launcher is
`~/.local/bin/pockettopo`. No new native export or Wine execution was performed
for this planning draft.

Capture exports from working copies, with the original TOP bytes untouched.
Record executable hash, Wine/runtime version, relevant settings, locale, input
filename, template directory contents/order, export options, and output hashes.
Keep isolated-file and multi-file-context cases separate. Preserve exact native
bytes, including line endings. Ordinary CI compares against those captured
fixtures; it does not need to drive the GUI on every run. New behavior requires
new native evidence, not expected output generated by our own implementation.

### Small fixtures and a real corpus

The confirmed corpus is [dlubom/test2](https://github.com/dlubom/test2), inspected
at commit `4c00c008a8dd441d70f9c62aa376c20b5914c7e0` through a non-truncated Git
tree listing. It contains **262 TOP paths and 258 distinct Git blob IDs**, totaling
9,187,192 listed TOP bytes. This is an inventory, not a parser validation result.
There are also SRV, SVX, Therion and DXF files, but their adjacency or names do not
make them authenticated native exports. Do not use them as automatic ground truth.

Pin corpus revision and file hashes when obtaining fixtures. Classify files by
version and features before sampling. Use a small checked-in regression set and
a larger pinned corpus campaign; preserve duplicate paths as context evidence
even when identical bytes share a test fixture. Keep the original corpus external.

Small cases should isolate ID boundaries/reserved values; positive and negative
integer rounding; angle wrap; forward/backward groups; groups crossing trip
boundaries; zero-length records; splays; flags; comments/Unicode; references;
unknown elements; truncation/count limits; loops; disconnected components;
template collisions; and every drawing element, color and XSection direction.
Validate synthetic files with the native reader before using them as references.

### Test layers and gates

| Layer | Required evidence |
| --- | --- |
| Unit/contract | Binary fields and source offsets; native arithmetic boundaries; exporter selection/formatting; CLI exit/output contract |
| Native golden tests | Exact output comparison for recorded exporter/settings/input cases; explicit semantic diagnostics on mismatch |
| Property/fuzz tests | Bounded failure on malformed bytes, deterministic output, preserved input/source identity; properties must respect native order sensitivity |
| Drawing tests | Coordinates, transforms, layers, colors, labels, splays and connectors; rendered review supplements structural checks |
| Consumer checks | Where available, import using the relevant destination program; this supplements native byte comparisons |
| Corpus regression | Every pinned input is accounted for as supported, unsupported or failed; no silent skipping |
| Mutation tests | ID comparisons, flags, signs, units, grouping, rounding, record selection and omission paths are exercised by meaningful assertions |

Proposed initial numeric gates: at least 95% statement coverage for a Go core, or
95% line/90% branch coverage for a Python core. These metrics are not interchangeable.
Use 90% killed mutants in the agreed core scope as an initial proposal, with every
survivor reviewed and every critical behavioral survivor resolved. Confirm this
policy after a real baseline; do not weaken it simply to turn a failing run green.
Exclude provably equivalent/invalid mutants only with recorded reasoning. Missing
runs, crashes and timeouts are not killed mutants. Record scope, tool version,
denominator and commit SHA; do not combine evidence from different code revisions.

PR checks should cover formatting, static checks, unit/native regression tests,
coverage, and a small corpus on Linux, Windows and macOS. Focused mutation checks
can run on Linux. Longer fuzzing, full corpus runs and complete mutation campaigns
belong in a separate workflow before compatibility milestones/releases. Never
describe an unexecuted or incomplete check as passed. Pin CI dependencies and
associate evidence with the exact delivered commit. Once Git exists, start with
one ordinary CI workflow and add a second only for genuinely expensive work.

## Small work items and clean context

Use PBIs (Product Backlog Items) as small, independently reviewable behavior
changes. PBI is a work-item format; it does not require a methodology framework.
Use Given/When/Then acceptance examples where they clarify behavior. Gherkin is
the scenario language; Cucumber is execution tooling. Initially put scenarios
directly in the PBI and implement them in the language's normal test runner.
Add a Cucumber/BDD runner only if executable shared scenarios provide enough
value to justify step definitions and another dependency.

Keep this document as the initial source of scope and decisions. When development
starts, keep a compact backlog in one place; avoid duplicating live status across
README, issues, per-task plans, and progress logs. Split documents only when this
becomes difficult to navigate.

Each ready PBI needs: user-visible outcome; explicit non-goals; dependency IDs;
reference methods; fixtures/settings; acceptance examples; required checks; and
the expected handoff. One chat should implement one ready PBI and stop with a
reviewable change. If the item does not fit that boundary, split it first.

The handoff records the commit, files/behavior changed, checks actually run,
unresolved discrepancies and the next ready item. A fresh chat reads repository
instructions, this specification and that PBI, then verifies current state.
Conversation history must not be required to reconstruct a critical decision.

### Proposed sequence

Only planning/refinement is active. Every implementation item below is proposed,
not authorized. Later rows may be split further after their reference contract
is understood. IDs describe this project only.

| ID | Outcome and acceptance | Depends on |
| --- | --- | --- |
| P00 | Agree language, first release subset, context model and compatibility criteria; resolve the open decisions below | This draft |
| P01 | Specify the complete exporter/options inventory and fixture matrix from C#/IL; identify native capture cases and version gaps | Can be researched before P00; incorporate relevant R01/R02 evidence |
| P02 | Establish Git/remote if needed, minimal selected toolchain, CI and a verified mutation-tool trial; no exporter framework | Explicit implementation request, P00 |
| P03 | Implement bounded v3 reading and inspect for trips, IDs, references and measurements; raw values/offsets and malformed-input behavior verified | P01, P02 |
| P04 | Read mappings, polylines and XSections; account for the complete file and unsupported content | P03 |
| P05 | Reproduce isolated-input native text export; native golden cases for units, flags, comments and record order | P03, P04; required native update behavior understood |
| P06 | Reproduce fixed-point angles, grouping and average directions with original-program evidence | P03 |
| P07 | Reproduce reference placement and plan geometry, then extended projection and closure as separately reviewed slices | P06 |
| P08 | Support explicit template context with deterministic inventory/order; prove isolation and native multi-file cases | P07 |
| P09 | Reproduce Therion DATA and drawing sections, including XSections; separate data and drawing slices if needed | P04, P07, P08 |
| P10 | Reproduce graphics DXF plan/side with option coverage and visual review; deliver option groups in small slices | P04, P07, P08 |
| P11 | Reproduce VisualTopo and its passage-dimension rules | P06, P07, P08 |
| P12 | Reproduce Toporobot and its distinct dialog options | P06, P07, P08 |
| P13 | Reproduce native 3D DXF, with degenerate geometry and bounded-failure cases | P07, P08 |
| P14 | Close native capability matrix, complete pinned corpus/mutation campaigns, validate distribution artifacts on supported platforms | P05, P09–P13 |
| P15 | Add Walls SRV with an explicit conversion/loss contract and Walls validation | Native milestone; refined format contract |
| P16 | Add Survex SVX with an explicit conversion/loss contract and Survex validation | Native milestone; refined format contract |
| P17 | Optional: implement the agreed layered SVG contract; can follow P10 without waiting for SRV/SVX | R03 recommendation accepted, drawing milestone, explicit implementation request; R06 only if separate section files are requested |
| P18 | Optional: implement XVI backgrounds with checks in XTherion and inkscape-speleo | R04 recommendation accepted, native drawing semantics validated, explicit implementation request |
| P19 | Optional placeholder: create separate small implementation items for TH2, PNG or PDF only after the corresponding research decision | R05 or R07 recommendation accepted and explicit implementation request; never implement this combined placeholder as one PBI |

Native evidence is captured alongside each PBI, not postponed until P14. CLI
reporting and source provenance are acceptance criteria of each exporting slice,
not a cleanup item at the end. P05 is the proposed first useful vertical slice;
if native Update dependencies require P06/P07, promote those prerequisites rather
than bypassing native semantics. Do not advertise full native compatibility before
P14. Older TOP versions receive separate items after the corpus inventory.

Example acceptance scenarios (proposals, not executed tests):

```gherkin
Feature: Native-compatible export with traceable source records

  Scenario: Export a captured native case
    Given a TOP fixture with a recorded source hash
    And a PocketTopo 1.372 reference export with recorded settings and context
    When the corresponding exporter runs with those settings and context
    Then the output bytes match the reference
    And the source file is unchanged
    And the report identifies the contributing source records

  Scenario: Preserve distinct reserved identifiers
    Given records containing raw IDs 0x80000000 and 0x800fffff
    When the source is inspected
    Then their internal IDs are -1 and -2 respectively
    And both display names are empty
    And the records retain their distinct raw and internal identifiers

  Scenario: Avoid implicit directory context
    Given an active TOP file beside another TOP file
    When it is exported in isolated-input mode
    Then the neighboring file does not contribute records or geometry
    And the report identifies the isolated-input context

  Scenario: Refuse an incomplete parse as a successful export
    Given a truncated drawing record
    When a complete native export is requested
    Then a structured error identifies the input and byte offset
    And no completed export is published
    And the source file is unchanged
```

Definition of done for each slice: acceptance behavior implemented and tested;
reference method/fixture evidence recorded; no source mutation or unreported loss;
required checks passed for the delivered revision; discrepancies documented;
the code and handoff are understandable without the originating chat. Test count
and coverage alone do not establish compatibility.

## Open decisions for the next discussion

- Confirm Go or choose Python; keep one production language.
- Confirm TXT as the first narrow deliverable, followed by Therion and graphics
  DXF, while the complete native export set remains the milestone goal.
- Decide whether cross sections initially mean faithful placement within full
  views (recommended) or also separate user-selected section files.
- Select supported operating systems/architectures and the initial coverage and
  mutation gates before adding tool configuration.
- Decide where the Git repository/remote will live. The current directory had
  only README.md and AGENTS.md and no `.git` during this planning pass.

## Research references

Primary local source:
[analysis map](../pockettopo-decompilation/analysis/ANALYSIS.txt),
[C# directory](../pockettopo-decompilation/decompiled/csharp/PocketTopo/), and
[IL](../pockettopo-decompilation/decompiled/PocketTopo.il). This pass inspected
the export entry points, ID decoding, grouping, arithmetic, template loading and
XSection handling; it is not a complete line-by-line exporter specification.

External documentation consulted for the proposed workflow:
[PocketTopo release history](https://paperless.bheeb.ch/PocketTopo13.html),
[Go fuzzing](https://go.dev/doc/security/fuzz/),
[Go toolchain](https://pkg.go.dev/cmd/go),
[Staticcheck](https://staticcheck.dev/docs/),
[go-mutesting](https://github.com/zimmski/go-mutesting),
[Ruff](https://docs.astral.sh/ruff/formatter/),
[mutmut](https://mutmut.readthedocs.io/en/latest/), and
[Gherkin reference](https://cucumber.io/docs/gherkin/reference/).
