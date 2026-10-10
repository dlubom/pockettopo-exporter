# PocketTopo Exporter

A standalone project for exporting PocketTopo `.top` survey files to multiple
formats, based on the PocketTopo 1.372 decompilation.

## Status

Implementation authorized on 2026-10-09. **P02, P03a, P03b, P03c1, P03c2, P04a, P04b, P04c1, P04c2, P04c3, P04c4, P04c5, P04c6, P04c7, P04c8, P04c9, P04c10, P04c11, P04c12 and P04c13 are complete**:
minimal Go CLI and checks, immutable station IDs, and bounded v3 trip and
measurement/reference/overview/plan-mapping/first-marker/Polygon-count/Polygon-points/Polygon-color/next-marker/second-Polygon-count/second-Polygon-points/second-Polygon-color/following-marker/third-Polygon-count/third-Polygon-points/third-Polygon-color/third-next-marker prefix readers with raw fields, source offsets,
copied records and explicit malformed-input errors. All earlier contracts remain intact.
Native export support is not yet implemented or validated.

Confirmed priorities:

- Reproduce the native exporters of the standard PocketTopo **1.372** first.
- Preserve measurements, plan sketches, extended elevations, and cross sections.
- Add Walls SRV and Survex SVX after native export compatibility.
- Keep historical comparison and reconciliation **in JKTZ**, which will consume
  this tool's outputs. Do not build a comparison engine in this project.
- Keep the implementation and development process small and understandable.

### P00 decisions required for P02

- **Language/toolchain:** Go 1.26.3 (the installed and selected baseline), standard
  library runtime, gofmt, go vet, Staticcheck v0.8.1. Gremlins v0.6.0 is accepted
  for the bounded mutation scope validated below. Tool upgrades are explicit changes.
- **First useful export:** isolated-input native text, followed by native Therion
  interchange and graphics DXF. Full native 1.372 export coverage remains the
  milestone; optional R01–R07 work is not a P02 prerequisite.
- **Context/compatibility:** explicit single input by default; native directory
  context is a later explicit option. Target exact native bytes under recorded
  settings. Preserve embedded XSections in full views; separate section files
  remain optional. No compatibility claim from toolchain checks.
- **Platforms:** initial CI targets Linux amd64, Windows amd64 and macOS arm64.
  Other architectures and release packaging are deferred.
- **Gates:** at least 95% statement coverage of implemented behavior (exclude only
  the process entry-point wiring); at least 90% killed mutants in the selected
  scope, with every survivor reviewed and critical survivors fixed. Empty runs,
  errors and timeouts are not success. No parser/corpus gate before a parser exists.
- **Repository:** module path `pockettopo-exporter`. Public GitHub creation and
  push were subsequently authorized for `dlubom/pockettopo-exporter`; releases
  remain out of scope. P02 adds no exporter framework.

**Next ready implementation PBI: P04c14 — fourth plan Polygon signed Int32 point count only.** See the
bounded acceptance contract below. P03b stops after trips, P03c1 after
measurements, P03c2 after references, P04a after the overview mapping and
P04b after the plan mapping, P04c1 after its first marker byte and P04c2
after the first Polygon point count, P04c3 after its raw vertices and P04c4
after its raw color, P04c5 after the next marker byte and P04c6 after the
second Polygon count, P04c7 after its raw vertices and P04c8 after its raw color, P04c9 after the following marker byte, P04c10 after the third Polygon count, P04c11 after its raw points, P04c12 after its raw color, and P04c13 after the next marker;
each leaves the remaining tail unparsed.

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

Scope priority and implementation authorization are separate. P02 and P03a were
authorized on 2026-10-09; later increments require their own implementation request. A successful third-party
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
| Go, mutation tooling and numeric quality gates | Selected and locally verified | P00/P02 decisions and evidence below |

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

## Language and development commands

Go is selected. The runtime has no external dependencies; tools are installed
at exact versions by `scripts/tools.sh` and are not application dependencies.
`go.mod` pins Go 1.26.3. Install that toolchain before running these commands;
all scripts use `GOTOOLCHAIN=local`, so they do not silently download a compiler.
Bash is required (Git Bash on Windows); mutation scripts also require jq 1.6+.
GitHub's Linux runner provides jq. No Python, Make or Cucumber runner is required.

From the repository root:

```sh
bash scripts/tools.sh                 # download/build pinned development tools
bash scripts/check.sh                 # format check, vet, Staticcheck, race tests,
                                      # exact >=95% coverage and boundary controls, build, CLI smoke
bash scripts/mutation.sh              # >=90% Gremlins killed; station/trip/measurement/reference/overview/plan mapping/marker/count/point/color/next-marker/second-count/second-point/second-color faults;
                                      # reject incomplete/empty/invalid runs
bash scripts/mutation.sh --matrix     # CI groups from the same local scheduler
bash scripts/mutation-dispatch-trial.sh # complete scheduling and failure controls
bash scripts/mutation-trial.sh        # weak-test, build/setup-error controls,
                                      # macOS/Linux

gofmt -w cmd internal                 # apply formatting
go run ./cmd/pockettopo-exporter --help
go run ./cmd/pockettopo-exporter --version
```

The built executable is `build/pockettopo-exporter` (`.exe` on Windows).
It accepts exactly one of
`--help`, `-h`, or `--version`. Success writes to stdout and exits 0. Missing,
extra or unsupported arguments and output errors exit 1; diagnostics use stderr.
`inspect` and `export` are not implemented. The development version is `dev`,
not a release number. No file input is opened by this skeleton.

Go/Staticcheck caches and tool binaries stay in ignored `.cache/` and `.tools/`.
The application can build offline once Go is installed; installing development
tools requires network access. `coverage.out`, `mutation.json` and `build/` are
regenerated outputs. There is no `go.sum` because there are no module dependencies.

The coverage gate sums statement counts directly from `coverage.out` and checks
`100 * covered >= 95 * total`; the rounded percentage printed by `go tool cover`
is informational. `bash scripts/coverage-trial.sh`, also run by `check.sh` on all
three CI platforms, rejects 94.96% and 94.99%, accepts exactly 95% and above,
and checks execution-count weighting, merged blocks, filenames containing spaces
or colons, and rejection of empty profiles and malformed counters.
This corrects the earlier gate's rounding gap without changing the 95% threshold
or the measured 100% coverage of the completed P04c1 implementation.

### Parallel CI mutation campaigns (2026-10-10)

Ordinary checks still run on Linux, Windows and macOS. Linux mutation work is
split into five independent jobs, with separate checkouts, disposable source
copies and Go caches. The matrix comes from `scripts/mutation.sh --matrix`;
the default `bash scripts/mutation.sh` runs all the same groups locally.
An optional group argument runs exactly one group:

| Group | Complete scope at P04c13 |
| --- | --- |
| `gremlins` | All 276 Gremlins mutants; CI also runs weak-test and compiler/setup-error controls |
| `tables` | Station, trips, measurements and references: 93 explicit faults |
| `first-polygon` | Overview/plan mappings, first marker/count/points/color: 198 explicit faults |
| `second-polygon` | Second count/points/color: 141 explicit faults |
| `markers` | Next/following markers, third count/points/color and next marker: 305 explicit faults |

All 737 explicit faults remain mandatory. Gremlins still requires at least 90%
killed mutants; explicit campaigns still require every fault killed by a named
test after successful compilation. Empty/incomplete reports, unknown statuses,
timeouts and compiler/setup errors remain failures. Operators, worker count,
tool versions, test timeouts and coverage thresholds are unchanged.

The planning job checks that the groups schedule every explicit campaign script
exactly once and that failures propagate. The final **CI complete** job requires
all three platform checks, the plan and all mutation groups to succeed; failed,
canceled or skipped dependencies cannot produce a successful final gate. Use
**CI complete** as the required check when configuring branch protection.
Parallel jobs reduce elapsed time when runners are available; queueing can still
delay completion. Timing must be measured on the delivered commit.

For each later PBI, register its new explicit mutation campaign exactly once in
an existing `run_group` scope list in `scripts/mutation.sh`, preserving all
earlier campaigns. Update the documented group scopes and measured counts.
Run `bash scripts/mutation-dispatch-trial.sh` to verify complete scheduling and
failure propagation, alongside the full local mutation and control commands.
Confirm all matrix jobs and **CI complete** succeed for the exact pushed SHA.

Local verification passed the ordinary checks (100% statement coverage), the
full 252/252 Gremlins and 571/571 explicit campaigns, all negative controls,
Bash 3.2 dispatch/syntax checks and actionlint v1.7.12. Independent read-only
review found no remaining issues and checked rejection of failed, canceled and
skipped dependency results. Parser code, fixtures, individual mutation scripts
and acceptance gates are unchanged.

### P02 verification and tool limits (2026-10-09)

Local environment: Go 1.26.3, darwin/arm64; Staticcheck 2026.2.1 (v0.8.1);
Gremlins v0.6.0. `scripts/check.sh` passed gofmt, go vet, Staticcheck, uncached
race tests, ordinary coverage, build and actual executable smoke checks.
There are 12 CLI cases (10 argument cases and two writer-error cases).
Statement coverage is **100% of `internal/cli`**. The one-line `main` wiring is
excluded from the numeric coverage gate and exercised by the executable smoke.
No PocketTopo behavior, native fixture or corpus has been tested by P02.

The mutation run generated **2 mutants, both killed**, with 0 lived, uncovered,
timed-out, invalid or skipped mutants. They negate argument-count validation
and output-error handling in `internal/cli/run.go`. The negative-control script
runs the same source with a test that calls the CLI without asserting its result:
ordinary tests pass and **both mutants live**. The same JSON gate used by
`mutation.sh` then exits **1**, correctly rejecting the 90% threshold. The original checkout is untouched by both trials.
Reproduce with the two mutation commands above; inspect their per-mutant JSON
in `mutation.json` and `build/mutation-weak.json`.

The measured source Git blob is `021ec158f91731db52906e5e834597d918308572`
(`internal/cli/run.go`); the strong tests blob is
`cbcfed532b607dfb984b1930082211fb59ea0974` (`internal/cli/run_test.go`).
Use `git hash-object` to verify these independently of README/CI-only edits.
The planning baseline is local commit `6b53351`; the P02 implementation commit
is identified by `git log --oneline` (this file is part of that commit).

Gremlins initially panicked while copying the complete module with local tool
caches. Inspection of its `internal/engine/workdir/workdir.go` showed that it
copies ignored files too. `scripts/mutation.sh` therefore copies `go.mod`,
optional `go.sum`, `cmd/` and `internal/` into a disposable source-only directory.
Keep that explicit scope current when adding real packages/fixtures. The shared `scripts/mutation-gate.jq` rejects empty reports and any status other
than KILLED/LIVED, then checks the actual killed fraction. Native Gremlins
threshold flags are not used: the trial found they could return exit 0 despite
0% efficacy. The JSON gate is the authoritative CI check.
No equivalent or invalid mutants have been exempted in P02.

The first hosted run (`07b5bdc`, [run 37915394919](https://github.com/dlubom/pockettopo-exporter/actions/runs/37915394919))
passed all ordinary checks on three systems and killed 2/2 Linux mutants, but
correctly failed the negative control's expectation of Gremlins exit 10. Local
Bash 3.2 had not stopped on a false standalone `[[ ... ]]` despite `set -e`, so
that first local negative-control success message was invalid. The repaired
trial checks the shared JSON gate with explicit failure handling; executable
smoke assertions also now use explicit `if`/`exit` branches. This finding is why
neither a tool's configured threshold nor a success message alone proves a gate.

This trial establishes usefulness for conditional defects only. Gremlins' default
operators do not mutate every return value, string or switch case. Expand the
mutation scope/operators and re-evaluate on native arithmetic in P03a/P06;
2/2 on this skeleton is not evidence for a future parser or exporter.

One GitHub Actions workflow runs formatting, vet, Staticcheck, tests/coverage and
build/CLI smoke on Linux amd64, Windows amd64 and macOS arm64. Linux also runs
mutation testing and its negative control. Action revisions and tool versions
are pinned. [CI run 37915898340](https://github.com/dlubom/pockettopo-exporter/actions/runs/37915898340)
passed all three jobs for implementation/tooling commit
`3b68dda974483950cf55bcf5ef6b82cc7c4af7f2`. The remote `main` SHA was verified
against that commit. The final handoff edit only records these results in README;
consult [GitHub Actions](https://github.com/dlubom/pockettopo-exporter/actions)
for the check attached to any later documentation commit.
The repository is [public](https://github.com/dlubom/pockettopo-exporter).
No release workflow or exporter packages were added.

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

Selected initial numeric gates: at least 95% Go statement coverage for implemented
behavior and 90% killed mutants in the explicitly selected scope, with every
survivor reviewed and every critical behavioral survivor resolved. P02 records
the first small baseline above; do not weaken gates to turn a failing run green.
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

P00 decisions needed to start, P02, P03a–P03c2 and P04a/P04b/P04c1/P04c2/P04c3/P04c4/P04c5/P04c6/P04c7/P04c8/P04c9/P04c10 are complete. P04 is split below so
the next chat can deliver one small implementation increment. Remaining rows are
planned, not completed; refine each contract when its dependencies are ready.
IDs describe this project only. R01–R07 are not scheduled for this handoff.

| ID | Outcome and acceptance | Depends on |
| --- | --- | --- |
| P00 | Done for startup: Go, native-first scope, isolation, byte compatibility, platforms and gates selected above | This draft |
| P01 | Deferred full exporter/options inventory; capture the reference contract needed by each implementation slice within that slice | Required before each corresponding exporter; optional research only when necessary |
| P02 | Done: public Git repository, Go skeleton, pinned tools, passing three-platform CI and positive/negative mutation trial | Explicit implementation request, P00 |
| P03a | **Done:** source station ID decoding and display, retaining raw bits and internal identity; contract and evidence below | P02; ID-specific C#/IL contract inside the slice |
| P03b | **Done:** bounded v3 header/trip prefix, immutable source fields, offsets, limits and structured errors | P03a; reader C#/IL and native evidence below |
| P03c1 | **Done:** bounded measurement prefix, immutable raw fields, optional comments, offsets and native acceptance evidence | P03b; Station/Survey reader contract below |
| P03c2 | **Done:** bounded reference prefix, raw coordinates/comments and explicit unparsed tail; native evidence below | P03c1; Reference/Survey reader contract |
| P04a | **Done:** bounded v3 overview mapping, three raw Int32 fields and explicit unparsed drawing tail; native evidence below | P03c2 |
| P04b | **Done:** bounded plan/outline mapping with separate raw fields/spans and native evidence; stop before the first element marker | P04a |
| P04c1 | **Done:** preserve the first plan marker byte and its span; stop before any payload, even for marker 0 | P04b |
| P04c2 | **Done:** first plan Polygon signed Int32 point count only; stop before vertices/color, no point allocation | P04c1 |
| P04c3 | **Done:** first plan Polygon raw vertices only; stop before color, including zero points | P04c2 |
| P04c4 | **Done:** first plan Polygon raw color byte only; stop before the next marker | P04c3 |
| P04c5 | **Done:** next plan element marker byte after the first Polygon; stop before its payload or side mapping, even for 0 | P04c4 |
| P04c6 | **Done:** second plan Polygon signed Int32 point count only, when next marker is 1; stop before its vertices/color | P04c5 |
| P04c7 | **Done:** second Polygon raw vertices only; stop before its color, including zero points | P04c6 |
| P04c8 | **Done:** second Polygon raw color byte only; stop before the following marker | P04c7 |
| P04c9 | **Done:** following plan element marker after the second Polygon; stop before payload or side mapping, even for 0 | P04c8 |
| P04c10 | **Done:** third Polygon signed Int32 point count only when following marker is 1; stop before vertices/color | P04c9 |
| P04c11 | **Done:** third Polygon raw vertices only; stop before color, including zero points | P04c10 |
| P04c12 | **Done:** third Polygon raw color byte only; stop before a later marker | P04c11 |
| P04c13 | **Done:** next plan-element marker byte only; stop before its payload | P04c12 |
| P04c14 | **Next / ready:** fourth Polygon signed Int32 point count only; stop before vertices/color | P04c13 |
| P04c15+ | Later small slices: subsequent payloads/markers, XSections, plan termination, side mapping/elements, then complete-file/unsupported-content accounting; refine separately | P04c14 |
| P05 | Reproduce isolated-input native text export; native golden cases for units, flags, comments and record order | P03c2, P04; required native update behavior understood |
| P06 | Reproduce fixed-point angles, grouping and average directions with original-program evidence | P03c2 |
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

## Completed PBI: P03a — source station identifiers

**Outcome:** the source model can preserve a raw 32-bit station identifier,
expose its native internal value and display name, and compare internal identity
without conflating different reserved IDs that display as empty text. This is
an independently testable prerequisite for trustworthy source inspection.

**Dependencies:** P02; inspect the bounded ID portion of P01 in this slice.
The complete exporter/options inventory is not a prerequisite for this helper.
Reference `ID.cs`: `Read`, `ToString`, `op_Equality`, `Equals`, and their IL in
`decompiled/PocketTopo.il`. Begin again with `analysis/ANALYSIS.txt`; do not adopt
JKTZ's current station-name mapping without checking those native methods.

**Non-goals:** TOP file parsing, CLI inspect/export, station-name input parsing,
ID generation, grouping/geometry, drawings, optional formats, archive comparison,
and source rewriting. Do not start P03b in the same chat.

**Fixtures/settings:** small table-driven raw bit patterns; no file/directory
context or locale-dependent native exporter options needed for the core helper.
Check C# and IL constants/branches and record provenance beside test cases.
Locate existing independently captured native ID evidence if available; label
source-derived cases as such and do not manufacture native output using Go.

**Acceptance examples:**

- Raw `0x80000000` maps to internal `-1`; raw `0x800fffff` maps to `-2`.
  Both display as empty, remain distinct identities, and retain original bits.
- Exercise raw/internal boundaries around `int.MinValue`, `LIMIT = -1048576`,
  `RESVD = -256`, zero, and the positive major/minor representation. Determine
  expected values from `ID.Read` and its IL, including raw aliases; equality
  follows the internal value while raw source records remain distinct.
- Formatting must not mutate raw values. Negative/reserved branches and positive
  major/minor formatting are tested separately; display is never an identity key.

**Required checks:** the P02 checks; focused mutation runs for boundary/sign
changes, raw preservation, reserved-value distinction and identity assertions.
Review each survivor and report any operator limitation. Add only the smallest
source-model package needed by actual code. Keep all source records immutable
by convention/API; no processing or exporter abstraction is needed here.

**Handoff:** update this README with implemented contract, reference locations,
actual commands/results and one refined next PBI (P03b if ready), then stop.

### Implemented contract and provenance (2026-10-09)

`internal/source.StationID` stores only a private `uint32` raw bit pattern.
`NewStationID` and `Raw` preserve it; `NativeValue` and `String` derive values
without changing the record. `SameIdentity` compares internal values, including
reserved IDs. Go struct `==` compares the raw records; it must not be used for
native station identity. The zero value represents raw `0`, displayed as `0.0`.
There are no setters, file access, CLI changes, processing or exporter packages.

Given signed `r = int32(raw)`, `NativeValue` returns `-1` for raw `0x80000000`,
`r + 2146435071` for other negative `r`, and `r` otherwise. These additions fit
Int32. Given internal `v`, display is decimal `v >> 16`.`v & 0xffff` when
`v >= 0`, decimal `v + 1048576` when `v < -256`, and empty otherwise.
The equivalent native subtraction of `LIMIT = -1048576` is preserved as addition
in Go. Decimal ASCII formatting has no locale setting in this helper.

Raw `0x80000001` is plain `0`, distinct from raw `0` (`0.0`). The complete
reserved range `0x800fff01..0x80100000` covers internal `-256..-1`.
`0x80000000` and `0x80100000` are aliases for `-1`; `0x800fffff` is distinct
internal `-2`. Nonnegative aliases include raw `1` and `0x80100002` (both `0.1`),
and raw `0x7feffffe` and `0xffffffff` (both `32751.65534`). Source records retain
different raw bits even when `SameIdentity` returns true.

Reference: `../pockettopo-decompilation/decompiled/csharp/PocketTopo/ID.cs`,
`Read`, `ToString`, `op_Equality`, `Equals`; IL methods at RVA `0x22348`,
`0x22070`, `0x21f20`, `0x220f4`, respectively. The inspected IL uses signed
comparisons, `add`, arithmetic `shr` and `and`, and compares the private internal
value directly. Reference SHA-256 values:

| File | SHA-256 |
| --- | --- |
| `ID.cs` | `8737d23529d5c27bc7d7be7a9563b73cd76ed8738d95639dde64eef113790fe6` |
| `PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

The reference manifest identifies the original EXE hash in the compatibility
section above. The tests separate the C#/IL-derived boundary table from five
historical observations recorded by reflection against original PocketTopo 1.372
on 2026-10-08 in [JKTZ issue #135](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/issues/135).
Those observations cover raw `0x80000000`, `0x80000001`, `0x800fff00`,
`0x800fffff`, and `0x00010009`, including internal values and text. The issue
does not record the probe's assembly hash/runtime; it is supporting historical
evidence, not a fresh native run or a fully captured boundary campaign.

Earlier JKTZ `model.py`, station tests and format contract were inspected at
[PR #136 head `67d3c98`](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/tree/67d3c98fe3b1e627eb975ab5f65aba6748424bb7/src/jktz/pockettopo).
Its immutable raw records are useful. Its export policy suppresses identities
for unnamed endpoints; that is not native `ID.Equals` and is not adopted here.
The local older JKTZ mapping was also inspected and is insufficient for reserved
IDs/aliases. No JKTZ or decompilation files were modified.

### P03a verification and mutation scope

Local Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and CLI smoke. Both `internal/cli` and `internal/source` have
  **100% statement coverage**. Station tests cover 22 boundary patterns,
  ten identity pairs, all 256 reserved values, the zero value and the five
  historical native observations. Raw bits are checked after derived operations.
- `bash scripts/mutation.sh`: **18/18 Gremlins mutants killed** (16 source-ID,
  2 CLI), no lived, uncovered, invalid, skipped or timed-out mutants. In addition
  to default operators, `--invert-assignments --invert-bitwise` test offset sign,
  shift direction and masking. No exclusions or survivor exemptions were added.
- The same command runs `scripts/mutation-station.sh`: **6/6 additional valid
  source mutations killed**. These discard/hide raw bits, compare raw records
  instead of aliases, compare empty display names instead of internal IDs,
  expose reserved names, and omit the major/minor separator. They cover return,
  constructor and string changes Gremlins does not generate. Each must compile
  and fail a named station test; errors and timeouts are rejected. Reports are
  regenerated in `mutation.json` and `build/station-mutation.json`.
- `bash scripts/mutation-trial.sh`: weak ordinary tests pass, both CLI mutants
  survive and the shared gate exits 1. A deliberate type error and a missing
  temporary directory also prove the test adapter returns 2 (NOT VIABLE),
  not a behavioral kill.

Gremlins' executor maps `go test` exit 1 to KILLED even for compiler errors.
`scripts/mutation-go.sh` is therefore installed only in the disposable mutation
PATH: a failed run must contain a named test failure in `go test -json`; other
errors, including setup/output failures, return 2 and fail the existing JSON
gate. The early expanded trial also
found two uncovered mutants on a package-level negative constant. The final
helper uses the equivalent direct addition, with every generated mutant covered.
Positive formatting uses `fmt.Sprintf` to avoid invalid string-subtraction
mutants; a valid separator fault is tested explicitly. The gates remain 95%
coverage and 90% killed; the additional six faults require all six killed.

Measured Git blobs: `eb326a668a23a4bd268191e846f823069f60abab` for
`internal/source/station_id.go` and `4f72e44bed5b5b9324ec5eac73200a3e45c9ad41`
for its tests. Reproduce with `git hash-object` and the commands above. The P03a
commit is identified by `git log --oneline`; hosted CI on Linux, Windows and
macOS reruns ordinary checks, with both mutation commands on Linux. The exact
pushed SHA and its run are verified in the chat handoff; this README is included
in that commit. P03a does not establish full TOP or exporter compatibility.

## Completed PBI: P03b — bounded v3 header and trips

**Outcome:** a binary prefix reader takes explicit input bytes, validates the
v3 header and trip table, and returns immutable source records with byte offsets,
the consumed offset and the unparsed tail size. It explicitly reports a prefix,
not a complete TOP parse. It leaves the caller's bytes unchanged and never
opens neighboring files. **Dependencies:** P03a and the reader-specific P01
contract inspected within P03b.

**Reference:** begin with `analysis/ANALYSIS.txt`, then `DataSet.Read`,
`Survey.Read` (the initial `Trip.ReadList` call only), `Trip.ReadList`,
`Trip.Read`, `FileReader`, and their IL. Native header and trip behavior must
be distinguished from the stricter documented resource/error policy.

**Bounded contract:**

- Accept only `54 6f 70 03`; reject bad magic and every other version explicitly.
  Preserve the header/version and signed little-endian Int32 trip count.
- Read ordered trips: signed Int64 ticks, .NET 7-bit byte-length UTF-8 comment,
  signed Int16 declination. Preserve ticks without float/date inference,
  original comment bytes and raw declination. Expose Auto for `-32768`
  separately; never replace the source value with native derived zero.
- Preserve a copy of the consumed prefix and record/field offsets; do not expose
  mutable aliases of caller bytes or model collections. Stop immediately after
  the trip table. Do not require or interpret a shot count or the remaining tail.
- Errors have a stable code, field context and zero-based byte offset. Reject
  truncation at every field, negative counts, overflowing 7-bit lengths,
  malformed UTF-8 and ticks outside the native DateTime range. Verify the exact
  native date/string behavior before fixing expected cases; document strict
  UTF-8 rejection if the native decoder substitutes invalid bytes.
- Explicit default operational bounds: 64 MiB input, 1,000,000 trips,
  1 MiB per comment; allow lower caller-supplied nonnegative limits. Check
  bounds and available minimum bytes before allocating, including arithmetic
  overflow. These are tool limits, not native format limits.

**Fixtures/acceptance:** small hand-authored prefix bytes for zero trips,
multiple ordered trips, empty/Unicode comments, 127/128-byte length boundary,
signed declination boundaries and Auto; expected offsets recorded beside cases.
The eight-byte header/count prefix with zero trips succeeds as a prefix even
without a tail. An arbitrary tail stays unparsed with its size recorded.
Truncating any required byte fails at the documented field/offset; exact limit
values pass and the next value fails. Include immutable-copy assertions and a
bounded fuzz test. Locate one existing native v3 fixture, verify its provenance
and hash, and compare only its independently established header/trips; do not
derive a native oracle from the new Go reader.

**Non-goals:** measurements/references, drawings, full-file validation, source
rewriting, CLI inspect/export, directory context, geometry, timestamp correction
and exporters. Do not start P03c in the same chat.

**Required checks/handoff:** P02/P03a commands, at least 95% behavior coverage,
focused count/length/sign/boundary mutations with every survivor reviewed and
no incomplete/error runs accepted. Expand disposable mutation copies only as
needed for new packages/fixtures. Update README with native versus stricter
behavior, source evidence and results, commit and push, verify CI for the exact
SHA, refine P03c and stop after P03b.

### Implemented prefix API and error policy (2026-10-09)

`internal/top.ReadV3TripPrefix(data)` uses `DefaultLimits()`;
`ReadV3TripPrefixWithLimits(data, limits)` accepts explicit lower limits.
`Limits` fields are `MaxInputBytes`, `MaxTrips` and `MaxCommentBytes`. Zero is
an actual bound; negative values or values above a default return
`invalid_limit`. Defaults are returned by value, not mutable global state.
The input limit includes the unparsed tail. No file or directory is opened.

The result is `source.TripPrefix`: `Header`, `Version`, `TripCountRaw`,
`Trips`, `Bytes`, `Offsets`, `ConsumedOffset` and `UnparsedTailSize`.
`Bytes` contains only the consumed prefix. `source.Trip` exposes `Ticks`,
`Comment`, `CommentBytes`, `DeclinationRaw`, `AutoDeclination` and `Offsets`.
Private fields, string-backed exact comment bytes, copied byte slices and
copied trip slices prevent mutable input/output aliases. Constructors also copy
collections. `Span` uses zero-based `Start` and exclusive `End`; offset structs
are returned by value. Fixed spans: header `[0,4)`, magic `[0,3)`, version
`[3,4)`, trip count `[4,8)`. Every trip records its full span and spans for
ticks, encoded comment length, comment bytes and raw declination.

`ParseError` has stable `Code`, `Field`, `Offset` and a readable diagnostic.
Failures return an empty result, never a successful partial table. Validation
order is limits, input size, header, count, minimum table bytes, then ordered
trip fields. Before allocating trips, `count <= remaining_bytes / 11` is
required; division avoids multiplication overflow. A failed preflight reports
the trip-count field, even when a later field would also be truncated.
Subsequent reads check remaining bytes before advancing; a comment is bounded
and checked before conversion/copying. With capped limits, all lengths and
offset additions fit `int` on both 32- and 64-bit targets.

| Code | Field and offset |
| --- | --- |
| `invalid_limit` | `limits.max_input_bytes`, `limits.max_trips` or `limits.max_comment_bytes`, offset 0 |
| `resource_limit` | `input` at 0, `trip_count` at 4, or `trips[i].comment.length` at its first encoded byte |
| `bad_magic` | `header.magic`, offset 0 |
| `unsupported_version` | `header.version`, offset 3; every byte except 3 is rejected |
| `negative_count` | `trip_count`, offset 4 |
| `truncated` | Required field start; preflight uses `trip_count` at 4; a missing length byte uses that byte's offset |
| `ticks_out_of_range` | `trips[i].ticks`, its first byte; valid inclusive range `0..3155378975999999999` |
| `string_length_overflow` | `trips[i].comment.length`, first encoded byte; byte five must be at most 7 |
| `invalid_utf8` | `trips[i].comment`, the comment field start, rather than an inferred replacement position |

Nonminimal length encodings are accepted and preserved, including a five-byte
encoding of zero. Strings count UTF-8 bytes, not characters. NUL, BOM, valid
U+FFFD and supplementary characters are retained without normalization.
Ticks retain all 100 ns bits without conversion to floats or dates. Auto is
derived only from raw `-32768`; the raw signed value remains unchanged.
Neither embedded dates nor fixture labels establish a field survey date.

### C#/IL and native evidence

After the analysis map, inspected `DataSet.Read` (RVA `0x219c0`), the initial
`Survey.Read` call (`0x744c`), `Trip.ReadList` (`0x12abc`), `Trip.Read`
(`0x12c1c`) and `FileReader` constructor (`0x2174d`) in both C# and IL.
IL confirms signed Int32 count, signed Int64 passed to `DateTime(long)`,
`BinaryReader.ReadString`, signed Int16, the Auto equality test and native
reset of derived `declCorr` to zero. Reference hashes:

| File | SHA-256 |
| --- | --- |
| `DataSet.cs` | `18dc45a572a7fcd53c18eef93d5195ab3778a755485674237c07d7923a5d9940` |
| `Survey.cs` | `e164752db9560a455d5c441bf0bebf133b19a0c25cfc3428bd5f09fdae45d48f` |
| `Trip.cs` | `7c452162180db0407a46058695219660222f47d405632386ce407d42d43beddd` |
| `FileReader.cs` | `d34ab119d81c9ff0d14a4f3a74a385cafcf394eebd9fc8baa3303cd114a71e17` |
| `PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

The original EXE was verified against the reference hash above. The checked-in
[native probe](scripts/reference-trip-probe.cs) called the original private
`Trip.ReadList`/`Trip.Read` methods through reflection on 2026-10-09, under
Wine Staging 11.7, Microsoft .NET 2.0.50727.42 x86, macOS arm64. No GUI,
native file writer or Go reader was used by this probe. Native stdout and
the optional reproduction commands are retained with
[fixture provenance](internal/top/testdata/README.md).

Native behavior differs from this strict contract:

- Native header checking uses signed `header >> 24` and `version <= 3`;
  this reader requires exactly v3 and does not claim older-version support.
- Native `Trip.ReadList` treats a negative count as an empty loop. This reader
  rejects it. Native has none of the explicit operational limits above.
- `DateTime(long)` rejected -1 and max+1; inclusive min/max succeeded. Native
  Auto set `automatic=true` and `declCorr=0`; the source model retains -32768.
- In the observed .NET 2.0 reader, encoded `A FF B` became `AB`, and an
  incomplete `E2 82` became empty. Thus these probes **omit** bad bytes rather
  than substituting U+FFFD. This reader rejects all malformed UTF-8. The later
  [Microsoft BinaryReader reference source](https://github.com/microsoft/referencesource/blob/main/mscorlib/system/io/binaryreader.cs)
  and [DateTime source](https://github.com/microsoft/referencesource/blob/main/mscorlib/system/datetime.cs)
  supplement the evidence; they do not override this observed .NET 2.0 behavior.
- Native accepted nonminimal zero and also `80 80 80 80 10` as zero, rejected
  the negative length from `80 80 80 80 08`, and rejected six-byte encoding.
  This reader rejects any fifth byte greater than 7 to enforce nonnegative
  Int32 lengths without discarded high bits.

The 542-byte `api-trips-ids.top` was copied unchanged from JKTZ's frozen
native fixtures. SHA-256:
`adb83280b6d5a70f383b5542727f675b3ff8a5740058b601d3d49ed27b14895b`.
The original native API helper's literal inputs and frozen expectations are
independent of this Go reader. This pass also read all three trips through
the original EXE: ticks, comments, declination and Auto matched; native
position 167 excludes the four header bytes, so prefix consumption is 171
and the unparsed tail is 371 bytes. Only header/trips are compared here.
Neither the Go test nor the native trip probe validates that tail.

JKTZ's strict parser/immutable records were reviewed locally. Resource bounds,
field-context errors and immutable source records are useful ideas. The Go
reader additionally validates native DateTime range and preserves the exact
prefix, encoded-length spans and comment bytes. No JKTZ processing policy,
full-file/tail rule, UTF-8 leniency or station identity mapping was imported.
Reference artifacts and original archives were not modified.

### P03b checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1 and Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and CLI smoke; **100% statement coverage** in `internal/cli`,
  `internal/source` and `internal/top`. Tests cover every required-byte
  truncation, all unsupported versions, length/count/sign/tick boundaries,
  default and lower bounds, exact comments and immutable-copy behavior.
- `bash scripts/mutation.sh`: **73/73 Gremlins mutants killed**, no lived,
  uncovered, invalid, skipped or timed-out cases. Assignment and bitwise
  operators exercise count/length arithmetic, signs, shifts and masking.
  The same command kills **6/6** station faults and **19/19** additional
  prefix faults in disposable copies. Prefix faults cover retained fields,
  source/input/output aliases, header/count offsets, consumed/tail reporting,
  endian order and error codes. Each must compile and fail a named assertion;
  stale targets, survivors, compiler errors and timeouts fail the run.
- `bash scripts/mutation-trial.sh`: weak ordinary tests pass and both CLI
  mutants survive; the shared gate rejects the result. Compiler/setup errors
  return NOT VIABLE rather than a behavioral kill.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3TripPrefix$' -fuzztime=10s -parallel=2`: passed,
  464,022 executions. Fuzz limits are 4096 input bytes, 32 trips and 256 comment
  bytes; assertions check deterministic structured results, input preservation
  and exact consumed/tail accounting. Normal checks also run the seed corpus.

The initial expanded campaign correctly failed on three noncompiling mutants:
untyped `-32768` negation/subtraction outside Int16 and pointer-creation `&`
inversion. Widening the Auto comparison to Int32 and using `new(ParseError)`
produce valid faults without exclusions. Two Auto faults then survived because
Gremlins runs a mutated package's own tests, and true Auto was asserted only by
the reader package. A model-level sentinel test closed that gap. The final
campaign has no exemptions; gates remain 95% coverage and 90% killed mutants.

Reports regenerate in `coverage.out`, `mutation.json`,
`build/station-mutation.json` and `build/prefix-mutation.json`. The source-only
mutation copy includes `internal/top/testdata`; the station-only copy needs no
TOP fixtures. Three-platform CI repeats ordinary checks, and Linux repeats
both mutation commands. This README is part of the delivered commit; exact
pushed SHA and hosted CI are verified in the chat handoff. There is no claim
of complete TOP parsing or exporter compatibility. P03b ends here.

## Completed PBI: P03c1 — bounded v3 measurements

P03c is split to keep one independently verified table per chat. P03c1 extends
the source prefix through the measurement table only. P03c2 extends it through
references; both APIs keep the remaining tail uninterpreted.

**Dependencies/reference:** P03b; begin with `analysis/ANALYSIS.txt`, then the
measurement portion of `Survey.Read`, `Station.Read`, `Station.Flags`, `ID.Read`,
and their IL. Inspect native trip-index behavior before choosing stricter
validation. Earlier JKTZ's restriction to flag bits 1/2 is not a native oracle:
the decompilation declares other flags and `Station.Read` retains a whole byte.

**Bounded acceptance contract:**

- Add a separate reader entry point that reads v3 header/trips followed by the
  signed Int32 measurement count. Keep P03b's trip-only API and behavior intact.
  Preserve an immutable consumed prefix, ordered raw records and all field spans.
- Preserve raw From/To UInt32 patterns through `source.StationID`; signed Int32
  distance in mm; signed Int16 azimuth/inclination; UInt8 flags and roll;
  signed Int16 trip index. Comment presence follows `flags & 2`, and present
  comments retain encoded length and exact UTF-8 bytes. Distinguish absent
  comment from present empty comment. No derived geometry/angles or grouping.
- Verify the native meanings of every flag, negative distance and trip-index
  boundaries against C#/IL and original-program evidence. Retain all raw bits;
  document any stricter rejection separately instead of silently masking or
  applying JKTZ's policy. Preserve reserved IDs and aliases without inferring
  identity from displayed text. Do not rewrite source trip indices.
- Bound count and minimum bytes before allocating; use the current 64 MiB input,
  1 MiB comment and 1,000,000-record default ceilings with lower nonnegative
  caller limits. Reuse the verified length/UTF-8 error rules. Reject negative
  count, overflow, truncation and resource violations with stable field/offset
  errors. Stop after measurements; do not require or interpret reference count.
- Add small literal cases for zero/multiple records, flag/comment presence,
  signed/endian boundaries, ID aliases/reserved values and trip-index cases.
  Test every required-byte truncation, exact bounds, immutable copies and
  bounded fuzzing. Use the existing native fixture's independent measurement
  expectations and native reader probes where necessary; no Go-generated oracle.

**Non-goals:** references, drawings, CLI inspect/export, full-file validation,
directory context, measurement correction, grouping, geometry and exporters.
Do not implement P03c2 or P04 in the same chat.

**Required checks/handoff:** current check/mutation/negative-control commands,
at least 95% coverage and 90% killed mutants, every survivor reviewed and no
incomplete/error campaign accepted. Expand explicit mutations for raw fields,
comment presence, flags and trip-index semantics as needed. Update README with
native versus strict behavior, provenance and results; refine P03c2, commit,
push and confirm CI for the exact SHA; stop after P03c1.

### Implemented measurement API and source policy (2026-10-09)

`internal/top.ReadV3MeasurementPrefix(data)` and
`ReadV3MeasurementPrefixWithLimits(data, limits)` read header, trips and
measurements only. `MeasurementLimits` embeds the unchanged P03b `Limits` and
adds `MaxMeasurements`; `DefaultMeasurementLimits()` uses the existing defaults
plus 1,000,000 measurements. Zero is a real bound; negative or above-default
values return `invalid_limit`. No CLI, processing or exporter code changed.

The result is `source.MeasurementPrefix`, with `Header`, `Version`,
`TripCountRaw`, `Trips`, `MeasurementCountRaw`, `Measurements`, `Bytes`,
`Offsets`, `ConsumedOffset` and `UnparsedTailSize`. It reuses immutable trip
records and copies measurement collections and consumed bytes on construction
and access. `MeasurementPrefixOffsets` includes P03b's fixed spans and the
actual `MeasurementCount` span following the variable-length trip table.

`source.Measurement` exposes `From`/`To` as raw-preserving `StationID` values,
`DistanceMM` (Int32), `AzimuthRaw`/`InclinationRaw` (Int16),
`FlagsRaw`/`RollRaw` (bytes), `TripIndexRaw` (Int16), `HasComment`, `Comment`,
`CommentBytes` and `Offsets`. Field order is exactly From, To, distance,
azimuth, inclination, flags, roll, trip index, optional string. Fixed fields
occupy 20 bytes; little-endian signed conversions preserve every stored bit.
No angular conversion, grouping, distance correction or trip-index adjustment
is performed. Negative and out-of-table indices are retained, including
-32768 and 32767, even with zero trips; they are not validated links.

Comment presence follows only `flags & 2`. An absent comment has zero
`CommentLength`/`Comment` spans and `HasComment=false`; a present empty comment
has an encoded-length span, a zero-length byte span at its source position and
`HasComment=true`. Comments retain exact UTF-8 bytes and nonminimal encoded
lengths in the copied prefix. All record/field spans use zero-based starts and
exclusive ends. There are no mutable input/output collection aliases.

The new measurement limit is validated first, then P03b's limit/input/header/
trip validation runs unchanged. Before measurement allocation,
`count <= remaining_bytes / 20` is required, avoiding count multiplication
overflow. Errors return an empty result, including when trips or earlier
measurements were read successfully. Existing `ParseError` codes are reused:

| Code | New field/offset contexts |
| --- | --- |
| `invalid_limit` | `limits.max_measurements`, offset 0 |
| `negative_count` | `measurement_count`, at its actual start after trips |
| `resource_limit` | `measurement_count`, or `measurements[i].comment.length` at its first encoded byte |
| `truncated` | Required field start; minimum-byte preflight uses `measurement_count`; missing length bytes use their own position |
| `string_length_overflow` | `measurements[i].comment.length`, at its first encoded byte |
| `invalid_utf8` | `measurements[i].comment`, at the comment byte start |

Fixed field names are `measurements[i].from_id`, `.to_id`, `.distance_mm`,
`.azimuth_raw`, `.inclination_raw`, `.flags_raw`, `.roll_raw`, `.trip_index_raw`.
All P03b codes, offsets, defaults and strict string rules remain in force.
An exact 12-byte zero-trip/zero-measurement prefix succeeds without a reference
count. Arbitrary, malformed or missing tail bytes stay uninterpreted. The
P03b eight-byte zero-trip prefix still succeeds through its original entry point.

### Measurement C#/IL and original-program evidence

Inspected the analysis map, `Survey.Read` (RVA `0x744c`), `Station.Read`
(`0x1669c`), `Station.Flags`, `ID.Read` and their IL. Native `Survey.Read`
reads a signed Int32 count and loops while `i < count`; negative counts would
skip the loop. This tool rejects negative counts and applies operational bounds.
Native `Station.Read` retains all flags and signed fields without distance or
trip-membership validation. `Station.GetDist`/`WriteDist` treat Int32.MinValue
as a blank distance; source reading retains that sentinel and other negatives.

| Flag | Native declaration/use; P03c1 preserves the stored bit |
| --- | --- |
| `0x01` | `flipped`, native extended-view flip/`<` text marker |
| `0x02` | `hasComment`, controls the following string |
| `0x04` | `invisible`, suppresses native drawing lines/`~` text marker |
| `0x08` | `special`, native special drawing style/`.` text marker |
| `0x10` | `projected`, native projected-view state/`=` text marker |
| `0x20` | `mark`, temporary loop traversal state in `Loop.Setup` |
| `0x40` | Undeclared in the inspected enum; accepted and retained by `Station.Read` |
| `0x80` | `readOnly`, subsequently set or cleared by `Survey.Read` according to load context |

Native nonnegative trip indices add `(short)tripOffs` and wrap through IL
`conv.i2`; negative indices are unchanged. That offset is the prior native trip
list size, not source data. `Trip.ByIndex` returns null and `DeclCorrByIndex`
returns zero outside the list. P03c1 stores the raw index and raw flags before
these context/processing changes. This is deliberate source preservation,
not an implementation of template loading or native derived state.

Reference hashes supplement the previously recorded Survey/ID/IL provenance:
`Station.cs` SHA-256
`44726d9b81881474147865a629365b9e02f0cc3e42dcec0b921613e36826c1cb`;
the unchanged `PocketTopo.il` SHA-256 is
`ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.

The new [native measurement probe](scripts/reference-measurement-probe.cs)
invokes original `Station.Read` through reflection, without GUI, native TOP
writing, references or drawings. On 2026-10-09, the hash-verified original EXE
under Wine Staging 11.7 / Microsoft .NET 2.0.50727.42 x86 confirmed all 256 flag
bytes, five distance boundaries, seven signed trip-index cases at offsets 0/3,
absent/present-empty comments, strict-policy string differences and all four
fixture measurements. Its CRLF stdout and reproduction details are in
[fixture provenance](internal/top/testdata/README.md). The probe exited 0;
native compilation produced a usable executable but its Wine compiler processes
remained open and were stopped after the probe completed. This optional compiler
invocation is not an automated gate.

The unchanged native `api-trips-ids.top` and independently frozen helper
expectations establish four ordered record spans `[175,226)`, `[226,419)`,
`[419,460)`, `[460,496)`. Measurement count is `[171,175)`; consumed bytes are
496 and the unparsed tail is 46. Only header/trips/measurements are validated.
Native UTF-8 `A FF B` becomes `AB` and fifth length byte 16 is discarded;
this reader reuses P03b's strict rejection rather than losing source bytes.

Earlier JKTZ immutable records and bounded reading were reviewed locally and
against its GitHub `master` parser (blob `74965043ebb9600ad71089bb079e76eaad1268f3`).
Its `flags & ~3` rejection and `-1..trip_count-1` index restriction are not
adopted: they disagree with native source reading. No original archives or
generated decompilation artifacts were edited.

### P03c1 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and executable smoke. **100% statement coverage** in each of
  `internal/cli`, `internal/source` and `internal/top`, including the unchanged
  P03b behavior. Tests cover literal signed/endian fields, reserved/alias IDs,
  all 256 flag bytes, absent/empty comments, raw trip boundaries, every required
  byte truncation, exact input/count/comment limits and immutable copies.
- `bash scripts/mutation-measurements.sh`: **35/35 valid additional mutants
  killed**. Each compiled and failed a named measurement assertion; build errors
  and timeouts are rejected. Faults discard raw fields/spans, hide empty-comment
  presence, alias collections/bytes, reverse endian order, swap flag/roll fields,
  select the wrong comment bit, reject native-accepted fields, or rewrite indices.
- `bash scripts/mutation.sh`: the complete integrated command passed:
  **119/119 Gremlins mutants killed** (43 measurement reader, 3 measurement model,
  52 trip reader, 3 trip model, 16 station-ID, 2 CLI). No lived, uncovered,
  invalid, skipped or timed-out mutants. The same run killed **6/6 station-ID,
  19/19 trip-prefix and 35/35 measurement explicit mutants**. No exclusions.
- `bash scripts/mutation-trial.sh`: ordinary weak tests passed, both CLI mutants
  survived and the shared gate rejected the run with exit 1. Deliberate compiler
  and setup errors returned NOT VIABLE exit 2, never a behavioral kill.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3MeasurementPrefix$' -fuzztime=10s -parallel=2`:
  passed, 763,135 executions. The same command with `FuzzReadV3TripPrefix`
  passed, 180,003 executions. Bounded inputs, deterministic structured results,
  immutable input and exact consumed/tail accounting are asserted; ordinary
  checks also execute both seed corpora.

The initial expanded run correctly rejected insufficient direct model coverage
and two uncovered `HasComment` mutants; source-contract tests resolved them.
The new explicit-fault script also rejected mutation-target and compiler errors
while its literal Bash replacement was being corrected. None of these incomplete
campaigns counts as passed; no survivor exemption or lowered gate was added.

Measured Git blobs: `9d6a98a589ba305f601431bb82cfd490349f7ced` (measurement model),
`027a209e345227af20a108419156e34b4e92a204` (model tests),
`fedbc10a433ec91c8b6f90e2e075f8e9d7b8e62f` (reader),
`4607d31b19719b68ba4a1520b313b6741b4c147c` (reader tests).
Reports regenerate in `coverage.out`, `mutation.json`,
`build/measurement-mutation.json` and the existing station/trip reports. All
source packages and fixtures remain in the explicit disposable mutation copy.
An independent read-only review found no Critical or Important issues.
CI repeats ordinary checks on Linux, Windows and macOS and both mutation
commands on Linux. The exact delivered SHA
and hosted run are verified in the chat handoff; README belongs to that commit.
P03c1 ends here and makes no complete-file or exporter compatibility claim.

## Completed PBI: P03c2 — bounded v3 references

**Outcome:** a separate source reader extends the validated measurement prefix
through the reference table, records the start/size of the uninterpreted tail,
and still reports a prefix rather than a complete TOP parse. Keep the P03b and
P03c1 entry points, source semantics, limits and errors intact.

**Dependencies/reference:** P03c1; start with `analysis/ANALYSIS.txt`, then
the reference portion of `Survey.Read`, `Reference.Read` (RVA `0x17764`),
`MetricLocation`, `MetricGrid`, `ID.Read` and their IL. Confirm field order,
units, signed boundaries and native empty-comment handling through the original
assembly before claiming native evidence. No CRS or geographic tie inference.

**Bounded acceptance contract:**

- Read the signed little-endian Int32 reference count following measurements.
  Preserve ordered immutable references, the copied consumed prefix and all
  record/field spans. Reject negative counts; validate count/resource/minimum
  bytes before allocation. Minimum record size is 25 bytes: ID 4, east 8,
  north 8, altitude 4, at least one string-length byte.
- Retain station UInt32 bits through `StationID`, signed Int64 east/north and
  signed Int32 altitude as source mm without float conversion or coordinate
  interpretation. Preserve the always-encoded UTF-8 comment, including empty
  bytes and its length span; native `Reference.Read` converts empty text to null,
  but the source model must retain what was stored. No flag byte is present.
- Reuse the current 64 MiB input / 1 MiB comment ceilings, add a 1,000,000
  reference ceiling with lower nonnegative caller limits, and reuse strict
  7-bit/UTF-8 rules and structured field/offset errors. On failure return no
  partial successful table. Do not normalize source fields.
- Stop immediately after references. Report consumed offset and tail size;
  accept no drawing bytes and an arbitrary tail. Do not require/read mappings,
  drawing element markers or a trailer; do not label that tail fully validated.
- Test zero/multiple references, Int64/Int32 signed and endian boundaries,
  station aliases/reserved IDs, empty/Unicode/nonminimal comments, every required
  byte truncation, exact limits, immutable copies and bounded fuzzing. Reuse the
  existing zero-reference fixture plus an independently captured native case
  with nonempty references and native boundary probes; no Go-generated oracle.

**Non-goals:** mappings, polylines, XSections, drawings, complete-file validation,
CLI inspect/export, directory context, geometry, correction and exporters.
Do not implement P04 in the same chat.

**Required checks/handoff:** current check, mutation and negative-control
commands; at least 95% statement coverage and 90% killed mutants, every survivor
reviewed, no empty/incomplete/error run accepted. Add explicit raw-coordinate,
comment and copy mutations where token operators are insufficient. Update README
with native/strict distinctions, provenance and actual results; refine P04 into
one small ready slice if necessary; commit, push, confirm CI for the exact SHA,
then stop after P03c2 and provide the next prompt.

### Implemented reference API and source policy (2026-10-09)

`internal/top.ReadV3ReferencePrefix` and `ReadV3ReferencePrefixWithLimits`
read only the v3 header, trips, measurements and references. `ReferenceLimits`
embeds the unchanged `MeasurementLimits` and adds `MaxReferences`;
`DefaultReferenceLimits()` adds a 1,000,000-reference bound. All inherited
ceilings remain 64 MiB input (including tail), 1 MiB comment and 1,000,000
records per earlier table. Zero is a real bound; negative/above-default values
fail. The new bound is validated first, then the unchanged P03c1 reader runs.

`source.Reference` exposes `Station`, `EastMM`, `NorthMM`, `AltitudeMM`,
`Comment`, `CommentBytes` and `Offsets`. East/north are signed Int64 and
altitude is signed Int32; no float conversion, unit conversion, CRS, geometry
or coordinate normalization occurs. `StationID` retains the raw UInt32 bits
and existing native identity behavior. Ordered duplicate station references
remain separate source records. Comments are always encoded: an empty comment
retains its encoded-length span and a zero-length byte span at its source
position. There is no reference flag byte or optional-comment inference.

`source.ReferencePrefix` exposes all earlier header/trip/measurement accessors,
plus `ReferenceCountRaw`, `References`, `Bytes`, `Offsets`, `ConsumedOffset`
and `UnparsedTailSize`. `ReferencePrefixOffsets` embeds the earlier offsets
and adds the actual `ReferenceCount` span following measurements. Constructors
and accessors copy collections/consumed bytes; comment bytes and offset structs
have no mutable aliases. Spans remain zero-based with exclusive ends.
`ConsumedOffset` is also the start of the uninterpreted tail.

Before reference allocation, `count <= remaining_bytes / 25` is required;
division avoids count multiplication overflow. Failure returns the empty
`ReferencePrefix`, even after earlier records have succeeded. Error rules:

| Code | New field/offset contexts |
| --- | --- |
| `invalid_limit` | `limits.max_references`, offset 0 |
| `negative_count`, `resource_limit` | `reference_count`, at its actual start |
| `truncated` | Required field start; minimum-byte preflight uses `reference_count`; a missing length byte uses its own position |
| `resource_limit`, `string_length_overflow` | `references[i].comment.length`, at the first encoded byte |
| `invalid_utf8` | `references[i].comment`, at its byte start |

Fixed field names are `references[i].station_id`, `.east_mm`, `.north_mm`
and `.altitude_mm`. All inherited limits/errors remain intact. Nonminimal
valid string lengths and exact UTF-8 bytes are preserved. The 16-byte
zero-trip/zero-measurement/zero-reference prefix succeeds without any mapping,
drawing or trailer; arbitrary tail bytes also succeed without interpretation.
The earlier eight-byte and twelve-byte prefix contracts still succeed through
their original entry points. No CLI or exporter code changed.

### Reference C#/IL and original-program evidence

Inspected `Survey.Read` (RVA `0x744c`), `Reference.Read` (`0x17764`),
`MetricLocation`, `MetricGrid.GetE/GetN/GetH`, `ID.Read` and their IL, after the
analysis map. C#/IL establish the signed reference count and field order
ID / east / north / altitude / string. Native negative reference counts skip
the loop; this reader rejects them and applies explicit operational bounds.
That negative-count observation is from C#/IL, not a fresh native Survey probe.
`MetricGrid` displays coordinates divided by 1000 and treats Int64.MinValue
(east/north) and Int32.MinValue (altitude) as blank values. Those sentinels remain
unchanged in the source model; no native display or processing is implemented.

| Reference file | SHA-256 |
| --- | --- |
| `Reference.cs` | `e6b2b669d39d0d63440908130ed90d62e91fa4e8e021bbfffe5823d62e805614` |
| `MetricLocation.cs` | `58b988d77763932bb24c1b5fae9e7c8a136a3aa1a7949d6ea192588754c2603a` |
| `MetricGrid.cs` | `33936b25fa3aec0ecd2685b216bcd2cd3b8de61c87bba973ebc606141b3b0c22` |
| `PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

The [native reference probe](scripts/reference-reference-probe.cs) invokes the
hash-verified original assembly under Wine Staging 11.7 / Microsoft .NET
2.0.50727.42 x86. It confirmed seven Int64 cases including min/max and values
outside exact Float64 integer precision, five Int32 boundaries, five ID
patterns including aliases/reserved values, asymmetric little-endian fields,
empty/Unicode/nonminimal comments and the two preserved native fixtures.
The probe exited 0; compiler processes remained open after producing its usable
executable and only the two processes created for this compilation were stopped.
Frozen CRLF stdout, hashes and actual optional commands are in
[fixture provenance](internal/top/testdata/README.md).

Native empty comments become null; this reader retains their stored length and
empty bytes. Native malformed `A FF B` becomes `AB`, and fifth length byte 16
is discarded to yield zero. Go keeps the P03b/P03c1 strict rejection of malformed
UTF-8 and lengths whose fifth byte exceeds 7. Valid nonminimal lengths survive
unchanged; no bytes are silently repaired.

The existing native zero-reference fixture has count span `[496,500)`, consumed
prefix 500 and unparsed tail 42. The unchanged 248-byte native `api-references.top`
was copied from pinned JKTZ evidence, SHA-256
`9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f`.
Independent native-helper expectations and fresh original-reader readback agree:
count span `[83,87)`, records `[87,150)` / `[150,206)`, consumed 206, tail 42.
Both references retain the same raw station `0x80000001`; coordinates are
synthetic, with no geographic meaning. Drawing bytes were neither read nor
validated. Earlier JKTZ's reference field order was reviewed locally and against
GitHub parser blob `74965043ebb9600ad71089bb079e76eaad1268f3`, then checked against
C#/IL. No reference archives or generated decompilation files were edited.

### P03c2 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and CLI smoke; **100% statement coverage** in `internal/cli`,
  `internal/source` and `internal/top`. Tests cover signed/endian fields, native
  fixtures, aliases/reserved IDs, always-present empty/Unicode/nonminimal
  comments, every required-byte truncation, exact default/lower bounds, no
  partial results, immutable copies and the unchanged earlier prefix contracts.
- `bash scripts/mutation.sh`: Gremlins **153/153 killed** (33 reference reader,
  1 reference model, and the unchanged 119 earlier mutants), with no lived,
  uncovered, invalid, skipped or timed-out mutants. The same complete command
  killed **6/6 station-ID, 19/19 trip, 35/35 measurement and 33/33 reference
  explicit mutants**. Reference faults discard raw coordinates/comments/spans,
  alias input/output collections/bytes, reverse endian order, swap coordinates,
  convert Int64 through Float64/Int32, change mm units, or normalize blank
  sentinels. Every explicit mutant compiled and failed a named assertion; stale
  targets, errors and timeouts are rejected. No exemptions or weakened gates.
- `bash scripts/mutation-trial.sh`: ordinary weak tests passed and both CLI
  mutants survived; the shared gate rejected that run with exit 1. Deliberate
  compiler and setup errors returned NOT VIABLE exit 2, never a behavioral kill.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3ReferencePrefix$' -fuzztime=10s -parallel=2`:
  passed, **637,039 executions**. Inputs are bounded to 4096 bytes, 32 records
  per table and 256 comment bytes; deterministic structured results, unchanged
  inputs, empty failures and exact consumed/tail accounting are asserted.
  Ordinary checks execute the earlier and new seed corpora.

Measured Git blobs: `75465f57c2012562733fe0e7d1f76aab12f88d2a` (reference model),
`a4bca010c2c1c82ab7d8f6ebe885a2f43e2ba19b` (model tests),
`0476da3f186790a0ee262373e1ea40aaaffd743c` (reader),
`e19ec2b5f40b40490aa2db72d11faf05b861afe9` (reader tests).
Reports regenerate in `coverage.out`, `mutation.json`,
`build/reference-mutation.json` and the unchanged earlier explicit reports.
The disposable mutation copy includes all source packages and native fixtures.
CI repeats ordinary checks on Linux, Windows and macOS, with the complete
mutation and negative-control commands on Linux. README belongs to the delivered
commit; exact pushed SHA and hosted CI are verified in the chat handoff.
P03c2 ends here, with no complete-file or exporter compatibility claim.

## Completed PBI: P04a — bounded v3 overview mapping

P04 is split before implementation. P04a reads exactly one fixed-size record:
the overview mapping immediately after references. Drawing mappings, elements
and complete-file/trailer rules remain later, separately refined P04 slices.

**Outcome/dependencies:** a separate immutable prefix reader extends P03c2 by
12 bytes and leaves the entire drawing tail uninterpreted. Preserve the P03b,
P03c1 and P03c2 APIs, limits, error rules and stopping positions unchanged.

**References:** begin with `analysis/ANALYSIS.txt`, then `DataSet.Read`
(v3, active-input `mapMap.Read` path), `Mapping.Read` (RVA `0x14479`),
`Mapping.Write`, `PixPerMm` and their C#/IL. Native `Mapping.Read` divides the
stored scale by `PixPerMm`; the source model must preserve the stored Int32
before that derived operation. Verify this through the original assembly.

**Bounded acceptance contract:**

- Read exactly three signed little-endian Int32 values in order: x0, y0,
  stored scale. Preserve raw bits/values and all record/field spans, copied
  consumed prefix, consumed offset and unparsed tail size. No scale division,
  screen transform, geographic orientation or positivity normalization.
- Reuse P03c2's operational bounds and strict earlier validation. Reject every
  required-byte truncation with stable mapping field/offset errors and return
  an empty result on failure. Accept a prefix ending immediately after those
  12 bytes and an arbitrary remaining tail; require no drawing mapping, element
  marker or trailer. This is not complete-file validation.
- Test literal asymmetric endian and Int32 min/max/negative/zero cases,
  offsets after variable earlier tables, immutable copies, earlier API
  regressions and bounded fuzzing. Reuse native `api-references.top` plus the
  independently captured `api-drawings.top` overview mapping (nonzero origin)
  at the pinned JKTZ revision; verify hashes and native helper inputs. Probe
  stored versus native derived scale, including zero/negative/boundary values,
  without using Go-generated expectations.

**Non-goals:** drawing mappings, polylines, XSections, drawing ownership,
geometry, rendering, full-file validation, directory context, CLI inspect/export
and all exporters. Do not implement another P04 slice in the same chat.

**Required checks/handoff:** current check, mutation and negative-control
commands; at least 95% statement coverage and 90% killed mutants, every survivor
reviewed and no incomplete/error campaign accepted. Add explicit raw-field,
scale and copy faults as needed. Update README with native/strict distinctions,
provenance and actual results, refine the next single P04 slice, commit, push,
confirm CI for the exact SHA and stop after P04a. P04a implementation was
explicitly authorized on 2026-10-09; no further P04 slice is included here.

### Implemented overview API and source policy (2026-10-09)

`internal/top.ReadV3OverviewPrefix` and `ReadV3OverviewPrefixWithLimits`
extend the unchanged P03c2 reader by exactly 12 bytes. The latter takes the
existing `ReferenceLimits`; there is no new limit type or table allocation.
All earlier validation, bounds, error precedence and stopping positions remain
unchanged. The 64 MiB input ceiling includes the entire uninterpreted tail.
Lower caller bounds and zero bounds retain their earlier meaning.

`source.Mapping` exposes `X0Raw`, `Y0Raw`, `ScaleRaw` and `Offsets`. All three
values are signed Int32 decoded little-endian, with no unit conversion, scale
division, float conversion, sign normalization or geographic interpretation.
`MappingOffsets` records the whole 12-byte record plus `X0`, `Y0` and `Scale`.
Zero/negative scales and all Int32 bit patterns are accepted as source fields.

`source.OverviewPrefix` retains all previous header/trip/measurement/reference
accessors and adds `OverviewMapping`. Its `Bytes`, `ConsumedOffset`,
`UnparsedTailSize` and `Offsets` describe this longer prefix only.
`OverviewPrefixOffsets` embeds the reference offsets and adds `Overview`, a
`MappingOffsets` value. Constructors/accessors retain the immutable earlier
models, copy consumed bytes and return mapping/offset structs by value.
Spans are zero-based with exclusive ends. No drawing mapping or marker is read.

Every incomplete overview field returns `truncated` at its field start:
`overview.x0` at the reference-prefix end, `overview.y0` four bytes later and
`overview.scale` eight bytes later. All failures return the empty
`OverviewPrefix`, including failures after successful earlier tables or fields.
A 28-byte all-zero-table prefix succeeds without any drawing bytes or trailer;
an arbitrary remaining tail also succeeds without being validated. P03b's
8-byte, P03c1's 12-byte and P03c2's 16-byte contracts are unchanged.

### Mapping C#/IL and original-program evidence

After the analysis map, inspected `DataSet.Read`'s v3 active-input branch,
`Mapping.Read` (RVA `0x14479`), `Mapping.Write` (`0x1443a`), the static
initializer and `SetVga`, with matching IL. `DataSet.Read` calls `mapMap.Read`
after the survey and before `outline.Read`/`sideview.Read`; template inputs
skip these mappings/drawings. P04a reads a single active-input source prefix
and introduces no template-directory behavior.

| Reference file | SHA-256 |
| --- | --- |
| `DataSet.cs` | `18dc45a572a7fcd53c18eef93d5195ab3778a755485674237c07d7923a5d9940` |
| `Mapping.cs` | `364d6d34cab642863ea92be7210afe652147ff509d9eba0cd35e319837d04e65` |
| `PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

The [native mapping probe](scripts/reference-mapping-probe.cs) invokes the
hash-verified original 1.372 assembly under Wine Staging 11.7 / Microsoft .NET
2.0.50727.42 x86. It calls native `Mapping.Read` on literal records and on the
two pinned fixtures, stopping before drawing bytes. `Mapping.Write` writes
only that scalar record to a separate memory stream to expose quantization;
it never writes a TOP archive. No Go reader generates native expectations.

The final probe exited 0 with 61 lines of frozen CRLF stdout. Both `PixPerMm=5`
(static default) and `PixPerMm=10` (native `SetVga`) were tested: 20 scales
including Int32 min/max, negative/zero and division boundaries, five origin
boundaries, an asymmetric endian case and both fixtures per mode.
Native signed division truncates toward zero: stored -501 becomes -100 or -50,
and a native rewrite stores -500; stored -1 becomes zero. Native min/max scale
reads also succeed. Go preserves the stored Int32, never the derived quotient.
JKTZ's earlier parser bounds scale to 10..50000; that restriction was reviewed
and deliberately not adopted because C#/IL and native probes accept wider values.

Native fixture readback confirms:

| Fixture | Overview span | Raw x0 / y0 / scale | Consumed / unparsed tail |
| --- | --- | --- | --- |
| `api-references.top` (248 bytes) | `[206,218)` | `0 / 0 / 500` | `218 / 30` |
| `api-drawings.top` (680 bytes) | `[122,134)` | `-1234 / 5678 / 500` | `134 / 546` |

The new fixture is copied byte-for-byte from pinned JKTZ revision
`3e3daa4156c6e6e79dce5203d8bb8e36122d571f`, Git blob
`07f805f0438dd2f48e9bd18d6771bf125713e8fe`, SHA-256
`4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5`.
The helper's literal nonzero origin and independent expected JSON were verified
against that revision. Attribution, hashes, actual optional commands and frozen
stdout are in [fixture provenance](internal/top/testdata/README.md).
Source TOP and C#/IL bytes remain unchanged; no drawings, screen transforms,
rendering or export compatibility were implemented or claimed.

### P04a checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and unchanged CLI smoke; **100% statement coverage** in
  `internal/cli`, `internal/source` and `internal/top`. New assertions cover
  raw signed/endian boundaries, native fixtures, every required mapping byte,
  field offsets after variable earlier tables, unchanged errors/limits,
  absent/arbitrary drawing tails, no partial results and immutable copies.
- `bash scripts/mutation.sh`: the complete integrated command passed with
  Gremlins **161/161 killed** (8 new overview-reader mutants and the unchanged
  153 earlier mutants), no lived, uncovered, invalid, skipped or timed-out
  mutants. The same command killed **6/6 station-ID, 19/19 trip, 35/35
  measurement, 33/33 reference and 29/29 overview explicit mutants**. New
  faults discard/swap raw fields or spans, use big-endian decoding, divide or
  multiply the stored scale, convert through Float32, normalize/reject source
  scales, hardcode the mapping position, alias bytes, consume the drawing tail
  or alter input. Every explicit mutant compiled and failed a named assertion;
  stale targets, compiler errors and timeouts are rejected. No exemptions or
  lowered gates were added.
- `bash scripts/mutation-trial.sh`: ordinary weak tests passed, both CLI
  mutants survived and the shared gate rejected the run with exit 1. Deliberate
  compiler/setup faults returned NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3OverviewPrefix$' -fuzztime=10s -parallel=2`:
  passed, **1,153,103 executions**. Bounded inputs, deterministic values/errors,
  unchanged input, empty failures and exact mapping spans/prefix/tail accounting
  are asserted. Ordinary tests also execute all earlier fuzz seed corpora.

Measured Git blobs: `0129f1562827b4603087da12d0d2ab64434b28c6` (mapping/prefix model),
`9d90e79436f0647ebfe55b8c561c2344367767bf` (model tests),
`6a0aa60b3469551defa6f8dc9f695a2d684126ae` (reader),
`e7db918f68e506b7d5dcae987be7592766f193da` (reader tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/overview-mutation.json` and the unchanged earlier explicit reports.
The disposable mutation copy includes all source packages and native fixtures.
CI repeats ordinary checks on Linux, Windows and macOS, with the complete
mutation and negative-control commands on Linux. README belongs to the delivered
commit; exact pushed SHA and hosted CI are verified in the chat handoff.
P04a ends here with no complete-file or exporter compatibility claim.

## Completed PBI: P04b — bounded v3 plan drawing mapping

**Outcome/dependencies:** a separate immutable prefix reader extends P04a by
exactly the next 12 bytes, the first drawing's plan/outline mapping, and stops
before any element marker. Preserve all P03/P04a APIs, source semantics,
limits, errors and stopping positions. P04b was explicitly authorized on 2026-10-09.

**References:** analysis map, `DataSet.Read`'s `outline.Read`/`sideview.Read`
sequence, the v3 mapping-before-elements branch in `Drawing.Read`,
`Mapping.Read`/`Write`, `PixPerMm` and matching IL. The side mapping follows
variable-length plan elements, so it cannot be read as another adjacent mapping.

**Bounded acceptance contract:**

- Reuse `source.Mapping` and P03c2's existing limits. Preserve the plan mapping's
  three raw signed little-endian Int32 fields, every field/record span, copied
  consumed bytes and tail accounting; retain the overview mapping separately.
  No derived scale, transform, coordinate normalization or drawing semantics.
- Reject every required-byte truncation with stable `plan.mapping.x0`,
  `plan.mapping.y0` and `plan.mapping.scale` field/start-offset errors; return
  an empty result on every failure. Retain all inherited strict rules.
- Accept a prefix ending immediately after the plan mapping, including no
  element marker. Accept arbitrary remaining bytes without validation; read
  neither plan elements nor the side mapping, terminators or trailer.
- Verify original native `Mapping.Read` at the independently established plan
  offsets using the pinned helper and fixtures: `api-references.top` mapping
  `[218,230)` with origin `0 / 0`, and `api-drawings.top` mapping `[134,146)`
  with helper origin `-100 / 200`; stored scale 500 in both. These pinned helper/format values
  are now confirmed by the fresh P04b native reader probe below.
- Test literal endian/min/max/negative/zero values, mapping separation, offsets
  after variable earlier tables, immutable copies, P03/P04a regressions, exact
  inherited limits, every required byte and bounded fuzzing. Add native evidence
  without deriving expectations from the Go implementation.

**Non-goals:** plan elements/markers, polylines, XSections, side mapping/elements,
drawing ownership, geometry/rendering, full-file/trailer validation, directory
context, CLI inspect/export and exporters. No later P04 slice in the same chat.

**Required checks/handoff:** current check, complete mutation and negative-control
commands; at least 95% statement coverage and 90% killed mutants, every survivor
reviewed, no incomplete/error campaign accepted. Extend explicit raw-field,
mapping-separation and copy faults as needed. Update README with actual native
evidence/results, refine one next small P04 slice, commit, push, confirm CI for
the exact SHA, then stop after P04b and provide its successor prompt.

### Implemented plan mapping API and source policy (2026-10-09)

`internal/top.ReadV3PlanMappingPrefix` and
`ReadV3PlanMappingPrefixWithLimits` extend the unchanged P04a reader by exactly
12 bytes. The latter takes the existing `ReferenceLimits`; no new bounds,
allocating table, CLI operation or exporter was added. The inherited 64 MiB
input ceiling includes the whole unparsed tail. Validation/error precedence,
zero/lower bounds and all earlier stopping positions remain unchanged.

`source.PlanMappingPrefix` retains every header/trip/measurement/reference
accessor plus `OverviewMapping`, and adds `PlanMapping`. Both mappings reuse
`source.Mapping` and retain three signed little-endian Int32 values without
scale division, units, float conversion, normalization or interpretation.
`PlanMappingPrefixOffsets` embeds the overview offsets and adds `Plan` with
its record, X0, Y0 and Scale spans. Consumed bytes are copied on construction
and access; previous immutable models and mappings/offsets returned by value
have no mutable aliases. Spans remain zero-based with exclusive ends.

Incomplete fields return `truncated` at `plan.mapping.x0` (overview end),
`plan.mapping.y0` (four bytes later), or `plan.mapping.scale` (eight bytes
later). Every failure returns the empty `PlanMappingPrefix`, including earlier
errors and truncation after successful fields. The 40-byte zero-table prefix
succeeds immediately after this mapping with no marker, side mapping or trailer.
Every possible first tail byte is accepted unchanged and remains unparsed.
P03b/P03c1/P03c2/P04a still accept their original 8/12/16/28-byte prefixes.

### Plan mapping C#/IL and fresh native evidence

Inspected the analysis map, `DataSet.Read` (RVA `0x219c0`), `Drawing.Read`
(`0x10a30`), `Mapping.Read`/`Write` (`0x14479`/`0x1443a`), static pixel defaults,
`SetVga` and matching IL. In the v3 active-input branch, overview precedes
`outline.Read`, which calls `mapping.Read` before its first `ReadByte`.
`sideview.Read` follows the entire variable-length plan drawing. It cannot be
located by reading a third adjacent scalar record. Template inputs skip these
mappings/drawings; this reader introduces no template behavior.

All previously recorded reference hashes remain unchanged. `Drawing.cs`
SHA-256 is `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7`.
The pinned JKTZ helper's literal plan origin and default stored scale were
checked at revision `3e3daa4156c6e6e79dce5203d8bb8e36122d571f`; its mapping
order was reviewed against C#/IL. JKTZ's 10..50000 scale restriction and
complete-drawing/trailer validation are not adopted by this source prefix.

The separate [P04b native probe](scripts/reference-plan-mapping-probe.cs)
uses the hash-verified original 1.372 assembly under Wine Staging 11.7 /
Microsoft .NET 2.0.50727.42 x86. Original trip/measurement/reference readers
establish the overview position, then two original `Mapping.Read` calls reach
the plan end. The helper asserts the independently pinned plan offsets; it
never reads an element marker, calls a drawing reader, writes a TOP file or
uses Go output. Full fixture streams and in-memory copies ending at the plan
mapping were read in both native pixel modes (5 and 10).

| Fixture | Plan span | Raw x0 / y0 / stored scale | Consumed / unparsed tail |
| --- | --- | --- | --- |
| `api-references.top` (248 bytes) | `[218,230)` | `0 / 0 / 500` | `230 / 18` |
| `api-drawings.top` (680 bytes) | `[134,146)` | `-100 / 200 / 500` | `146 / 534` |

The original reader preserves origins and derives scale 100 / 50 from stored
500; Go preserves 500. All four prefix-only calls succeed with tail zero.
Final probe exit was 0 with empty stderr; its 25-line CRLF stdout is frozen
as `native-plan-mapping-read.txt`. Original assembly/runtime, C#/IL and all
three source TOP hashes were rechecked and remain unchanged. Attribution,
probe/output hashes and actual reproduction commands are in
[fixture provenance](internal/top/testdata/README.md).

### P04b checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and unchanged CLI smoke; **100% statement coverage** in each
  implemented package (`internal/cli`, `internal/source`, `internal/top`).
  Tests cover signed/endian/min/max/negative/zero fields, all 256 first tail
  bytes, no marker, mapping separation, variable earlier tables, every required
  mapping byte, empty failures, inherited errors/limits and immutable copies.
- `bash scripts/mutation.sh`: complete integrated campaign passed with
  Gremlins **169/169 killed** (8 new plan-reader mutants and the unchanged
  161 earlier mutants), no lived, uncovered, invalid, skipped or timed-out
  mutants. The same command killed **6/6 station-ID, 19/19 trip, 35/35
  measurement, 33/33 reference, 29/29 overview and 32/32 plan mapping explicit
  mutants**. Plan faults swap/divide/normalize raw fields, confuse mappings or
  spans, reread overview, hardcode position, replace caller limits, misname
  errors, expose aliases, alter input, consume the tail or require a marker.
  Every explicit mutant compiled and failed a named assertion; stale/ambiguous
  targets, compiler errors and timeouts are rejected. No lowered gates or
  exemptions. The final complete campaign uses the delivered code/tests.
- `bash scripts/mutation-trial.sh`: ordinary weak tests passed and both CLI
  mutants survived; the shared gate rejected the run with exit 1. Deliberate
  compiler/setup errors returned NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanMappingPrefix$' -fuzztime=10s -parallel=2`:
  passed, **980,721 executions**. Bounded inputs, deterministic values/errors,
  unchanged input, empty failures, raw bits, separation from overview and exact
  prefix/tail/spans are asserted. Ordinary checks run all earlier fuzz seeds.
- Independent read-only review found no Critical, Important or Minor issues;
  its race tests, fixture/output checks and scope review passed.

Measured Git blobs: `ce41032012e9e2c54345dead947978e7027222ad` (model),
`d4c53a7a291f0e9a6203985f43dee4e79f627c25` (model tests),
`5e729eaaa3e31af3de56c065dd5596ac74626082` (reader),
`340988162326571b0d3b6e913ebeab4a597f8e82` (reader tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-mapping-mutation.json` and the unchanged earlier reports. CI repeats
ordinary checks on Linux, Windows and macOS; Linux runs the complete mutation
and negative-control commands, including the new plan scope. Exact pushed SHA
and hosted CI are verified in the chat handoff; README is in that delivered
commit. P04b stops before the first element marker and makes no element,
complete-file or exporter compatibility claim.

### P04b revalidation on the completed checkout (2026-10-09)

Revalidation baseline: `a9702baeb309ff0d1199392d8aa04069176c4c2b`.
P04b was already implemented in `3ff9b56c940feddc976ee1170861175104533ad2`,
and the separate P04c1 reader was already present. The P04b reader, model,
tests, native probe and explicit mutation script are byte-identical to their
P04b implementation revision. This handoff updates documentation only;
all earlier contracts and the existing P04c1 implementation remain unchanged.

Rechecked the analysis map, `DataSet.Read`, `Drawing.Read`, `Mapping.Read`/
`Write`, pixel defaults and matching IL. They confirm that the plan mapping
occupies the next 12 bytes and precedes the first marker. Original EXE,
Mapping/Drawing/DataSet C#, IL, all three TOP fixtures, the P04b probe source
and its frozen native stdout hashes still match the recorded provenance.
The existing JKTZ parser's scale restriction and complete-drawing policy
were reviewed against this reference and remain outside the source-prefix
contract. The native Wine probe was not rerun in this revalidation.

With Go 1.26.3 darwin/arm64, Staticcheck v0.8.1 and Gremlins v0.6.0:

- `bash scripts/check.sh` passed all checks, uncached race tests and CLI smoke;
  statement coverage remains **100%** in each implemented package.
- `bash scripts/mutation.sh` passed the complete current-checkout campaign:
  **173/173 Gremlins mutants killed**, with no other statuses, plus **6/6
  station, 19/19 trip, 35/35 measurement, 33/33 reference, 29/29 overview,
  32/32 plan mapping and 25/25 existing marker explicit mutants killed**.
  Every explicit mutant compiled and failed a named behavioral assertion.
- `bash scripts/mutation-trial.sh` passed: both weak-test CLI mutants lived,
  the gate rejected them with exit 1, and compiler/setup controls returned
  NOT VIABLE exit 2. Gates and mutation scopes were unchanged.
- The documented 10-second `FuzzReadV3PlanMappingPrefix` command passed with
  **451,267 executions**, preserving input, raw fields, mappings, spans,
  deterministic errors, empty failures and consumed/tail accounting.
- Independent read-only P04b review found no Critical, Important or Minor
  issues; its targeted uncached race tests and bounded fuzzing also passed.

Reports regenerate in the existing ignored output paths. The final handoff
verifies the pushed SHA and its three-platform CI; the next unfinished PBI
is **P04c2**, whose contract below remains a separate implementation request.

## Completed PBI: P04c1 — first v3 plan element marker

**Outcome/dependencies:** a separate immutable source prefix extends P04b by
exactly one byte: the first plan drawing element marker. This is deliberately
split from variable-length payload parsing. Preserve every earlier API, mapping,
source field, limit, error and stopping position. P04c1 was explicitly
authorized on 2026-10-09 and includes no later payload slice.

**References:** analysis map, `DataSet.Read` ordering, `Drawing.Read`'s first
`BinaryReader.ReadByte` and marker dispatch, and matching IL. Original
`Mapping.Read` followed by `BinaryReader.ReadByte` on both pinned fixtures
provides the native marker/position evidence without parsing an element.

**Bounded acceptance contract:**

- Reuse `ReferenceLimits` and the P04b source model. Preserve the marker as a
  raw UInt8 plus its one-byte span, copied consumed bytes and tail accounting.
  Keep the overview and plan mappings separate. No element collection/model.
- Missing marker returns `truncated`, field `plan.elements[0].kind`, at the
  P04b consumed offset; every failure returns an empty new result. Retain all
  inherited validation/error precedence and bounds, including tail input size.
- Accept all 256 raw marker values as a marker-only prefix. Document native
  dispatch for 0 (terminator), 1 (Polygon), 3 (XSection) and other bytes from
  C#/IL; accepting the byte does not claim its payload is supported. Stop
  immediately after the byte, even when it is 0. Do not require or read payload,
  next marker, side mapping, terminator beyond this byte, or trailer.
- Test every byte, absent/exact/arbitrary tails, immutable copies, variable
  earlier-table offsets, structured errors, inherited limits and all earlier
  stopping contracts. Verify both pinned native marker positions independently
  of Go; add bounded fuzzing and explicit marker/span/copy/overread mutants.

**Non-goals:** Polygon/XSection payloads, plan element loops, side mapping or
side elements, geometry/rendering, full-file validation, directory context,
CLI inspect/export and exporters. Refine the next single payload slice after
P04c1; do not implement it in the same chat.

**Required checks/handoff:** current check, complete mutation and negative-control
commands; at least 95% statement coverage and 90% killed mutants, every survivor
reviewed and no incomplete/error run accepted. Record native evidence/results,
update README, commit, push, confirm CI for the exact SHA, stop after P04c1 and
provide one successor prompt.

### Implemented marker API and source policy (2026-10-09)

`internal/top.ReadV3PlanMarkerPrefix` and
`ReadV3PlanMarkerPrefixWithLimits` extend the unchanged P04b reader by one
`take(1, "plan.elements[0].kind")`. They reuse `ReferenceLimits` without new
bounds or allocations for drawing elements. Inherited validation/error precedence,
zero/lower caller limits and the 64 MiB ceiling including the tail are unchanged.
Every failure returns the empty `source.PlanMarkerPrefix`.

`source.PlanMarkerPrefix` retains all earlier header/table/mapping accessors
and adds `MarkerRaw` (UInt8). `PlanMarkerPrefixOffsets` embeds all P04b spans
and adds `Marker`, a one-byte span. The constructor and `Bytes` accessor copy
consumed bytes; mappings/spans are returned by value and earlier immutable
records remain separate. A missing marker reports `truncated` at
`plan.elements[0].kind`, offset equal to P04b's consumed end. The zero-table
marker prefix consumes 41 bytes; P04b still accepts 40 bytes with no marker.
P03b/P03c1/P03c2/P04a retain their 8/12/16/28-byte stopping contracts.

All 256 bytes are retained unchanged. No dispatch or payload validation occurs;
0 still consumes exactly one byte and stops before the side mapping. Markers
1/3 and every other value succeed without any following byte. Arbitrary tails
remain unparsed, with their size recorded separately from copied consumed bytes.
No source input is modified and no CLI operation was added.

### Marker C#/IL and fresh native evidence

After the analysis map, inspected `DataSet.Read` (RVA `0x219c0`) and
`Drawing.Read` (`0x10a30`) in C#/IL. The active v3 branch reads overview,
then outline, then sideview. Outline reads its mapping before the first
`BinaryReader.ReadByte` (`IL_003a`). Native dispatch is:

| Marker | Native `Drawing.Read` behavior after the byte |
| --- | --- |
| 0 | Exit the element loop; the caller later reads sideview |
| 1 | Construct `Polygon`, then call its payload reader |
| 3 | Construct `XSection`, then call its payload reader |
| Other nonzero values | Leave element null, call `ReadBytes(0)`, then read the next marker; no payload length is read |

The last row is confirmed by IL's zero local at `IL_0046`, `ReadBytes`
at `IL_00b6` and next `ReadByte` at `IL_00c3`. This is source inspection,
not a claim of robust native unknown-payload support. The Go prefix intentionally
stops before all those dispatch actions. Earlier JKTZ parser blob
`74965043ebb9600ad71089bb079e76eaad1268f3` was checked through GitHub and
reviewed locally: its rejection of kinds outside 0/1/3 and complete drawing
loop are not adopted by this marker-only API. The pinned native helper's empty
References drawing and first three-point Polygon in Drawings explain markers
0/1 independently of Go.

The separate [P04c1 native probe](scripts/reference-plan-marker-probe.cs)
uses original `Trip.ReadList`, `Station.Read` and `Reference.Read` to locate
the overview, calls original `Mapping.Read` twice, and reads exactly one byte
through the original .NET `BinaryReader` held by `FileReader`. It asserts the
pinned offsets and marker values; no `Drawing.Read`, element payload, side
mapping, writer or Go output participates. In-memory streams ending before
or just after the byte test missing and exact prefixes in both native pixel
modes (5/10).

| Fixture | Marker span / raw value | Consumed / unparsed full-file tail |
| --- | --- | --- |
| `api-references.top` (248 bytes) | `[230,231)` / 0 | `231 / 17` |
| `api-drawings.top` (680 bytes) | `[146,147)` / 1 | `147 / 533` |

Final native probe exit was 0 with empty stderr; its exact 15-line CRLF stdout
is `native-plan-marker-read.txt`. Full and exact-prefix reads agree. Missing
markers throw `EndOfStreamException` without advancing the stream. Environment
is Wine Staging 11.7 / Microsoft .NET 2.0.50727.42 x86, original assembly
1.3.7.0, macOS arm64. Original EXE/runtime, C#/IL and all three source TOP
hashes were checked before and after and remain unchanged. Probe/output hashes,
attribution and actual commands are in [fixture provenance](internal/top/testdata/README.md).

### P04c1 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0 (module version
verified with `go version -m`; its binary version banner reports `dev`):

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, uncached race
  tests, build and unchanged CLI smoke; **100% statement coverage** in each
  implemented package (`internal/cli`, `internal/source`, `internal/top`).
  Tests cover all 256 markers, no payload, exact/arbitrary tails, absent marker,
  inherited truncations/errors/limits, variable earlier tables, separate mappings,
  immutable copies and every earlier stopping contract.
- `bash scripts/mutation.sh`: the complete integrated campaign passed with
  Gremlins **173/173 killed** (4 new marker-reader mutants and the unchanged
  169 earlier mutants), no lived, uncovered, invalid, skipped or timed-out
  mutants. The same command killed **6/6 station-ID, 19/19 trip, 35/35
  measurement, 33/33 reference, 29/29 overview, 32/32 plan mapping and 25/25
  marker explicit mutants**. Marker faults discard/mask the raw byte, confuse
  spans/mappings, expose aliases, hardcode/reread offsets, replace caller limits,
  misname errors, alter input, consume tails, read two bytes, require a next
  marker/payload, reject unknown values, omit zero or return a partial failure.
  Every explicit mutant compiled and failed a named assertion. Stale/ambiguous
  targets, compiler errors and timeouts are rejected; no exemptions or weakened
  gates were added. All campaigns ran on the delivered code/tests.
- `bash scripts/mutation-trial.sh`: ordinary weak tests passed and both CLI
  mutants survived; the shared gate rejected the run with exit 1. Deliberate
  compiler/setup errors returned NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanMarkerPrefix$' -fuzztime=10s -parallel=2`:
  passed, **1,176,471 executions**. Bounded inputs, deterministic values/errors,
  unchanged input, empty failures, inherited mappings/spans, raw marker and exact
  prefix/tail accounting are asserted. Ordinary checks run all earlier fuzz seeds.

Measured Git blobs: `d0bd80d68ce50099d9b8be95c5fae246ffaa7f61` (model),
`2c6c3f0d3b0e1a075be16dae088f9a6a9cffe56f` (model tests),
`749c62276a897a00a44f3316fa556f8f96626a92` (reader),
`19c93728a21a31e39ef9efad21b4b09a19daff0e` (reader tests).

Independent read-only review found no Critical, Important or Minor issues;
its race tests, formatting/shell checks and native evidence/hash checks passed.

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-marker-mutation.json` and the unchanged earlier reports. CI repeats
ordinary checks on Linux, Windows and macOS; Linux runs the complete mutation
and negative-control commands, including the new marker scope. Exact pushed SHA
and hosted CI are verified in the chat handoff; README is in that delivered
commit. P04c1 stops before every payload and includes no later P04 implementation.

## Completed PBI: P04c2 — first plan Polygon point count only

**Outcome/dependencies:** a separate immutable prefix extends P04c1 by exactly
four bytes only when the first marker is 1: the Polygon's signed little-endian
Int32 point count. Keep every earlier API, field, mapping, span, limit, error
precedence and stopping contract. P04c2 was explicitly authorized on 2026-10-09;
P04c1 continues to include no point-count or other payload implementation.

**References:** analysis map, `Drawing.Read` marker 1 dispatch,
`Polygon.Read` (RVA `0xc4b8`, first `ReadInt32`/`newarr` and vertex/color order),
matching IL, pinned native helper and `api-drawings.top`. Verify count behavior
against original .NET before fixing negative/boundary expectations. Extend a
separate native probe by one `ReadInt32` after the marker, without invoking
`Polygon.Read` or reading vertices/color. The pinned helper's first Polygon
has three points; its count span is `[147,151)` independently of Go.

**Bounded acceptance contract:**

- Reuse the immutable P04c1 model and retain the raw signed count, its four-byte
  span, copied consumed bytes and tail accounting. No point collection or
  coordinate/color/geometry model. Overview and plan mappings remain separate.
- Marker 1 is the sole supported branch of this new count-only API. Other
  markers, including 0/3, return `unsupported_element` at
  `plan.elements[0].kind` and the marker start, without consuming their tails.
  P04c1 continues to accept all 256 bytes unchanged.
- A missing/incomplete count returns `truncated` at
  `plan.elements[0].point_count`, the marker end. Reject negative counts with
  `negative_count` there; preserve all nonnegative raw values within an explicit
  operational ceiling. Introduce only a count limit (default 1,000,000, lower
  nonnegative caller bounds; zero is real) alongside unchanged `ReferenceLimits`.
  Validate the new limit first, then inherited limits/fields, marker, count.
  Above-default/negative limits use `invalid_limit`; above-bound counts use
  `resource_limit`. Every failure returns an empty new result.
- Stop immediately after the count, even for zero. Do not allocate points or
  preflight/require their bytes, color, next marker, side mapping or trailer.
  An exact count-only prefix and arbitrary tail succeed within input bounds.
- Test signed/endian/count/limit boundaries, every required count byte,
  non-1 markers, variable earlier tables, immutable copies, inherited errors
  and all earlier stops. Confirm the pinned native count/position and original
  zero/negative/boundary behavior, add bounded fuzzing and explicit count/span/
  copy/overread mutants. Native allocation semantics may be documented from IL;
  do not trigger large allocations for a count-only probe.

**Non-goals:** vertices, colors, full Polygon/XSection payloads, drawing element
loops, side mapping/elements, geometry/rendering, complete-file validation,
directory context, CLI inspect/export and exporters. Refine one next small
payload slice; do not implement it in the same chat.

**Required checks/handoff:** current check, complete mutation and negative-control
commands; >=95% statement coverage, >=90% killed mutants, every survivor reviewed,
no incomplete/error run accepted. Record native evidence, update README, commit,
push, confirm exact-SHA CI, stop after P04c2 and provide one successor prompt.

### Implemented count API and source policy (2026-10-09)

`internal/top.ReadV3PlanPolygonCountPrefix` and
`ReadV3PlanPolygonCountPrefixWithLimits` extend the unchanged P04c1 reader
by exactly four bytes, solely for marker 1. `PolygonCountLimits` embeds the
unchanged `ReferenceLimits` and adds `MaxPoints`; `DefaultPolygonCountLimits()`
sets it to 1,000,000. Lower nonnegative values are accepted, including zero.
The input ceiling still includes the entire unparsed tail. No point collection,
coordinate/color model, derived geometry, CLI operation or exporter was added.

`source.PlanPolygonCountPrefix` reuses the immutable marker prefix and all its
header/table/mapping/marker accessors. `PointCountRaw` exposes signed Int32;
`PlanPolygonCountPrefixOffsets` embeds every earlier span and adds `PointCount`.
Consumed bytes are copied on construction and access, mappings/spans are values,
and inherited record collections remain copied. No raw field is normalized.
The model constructor preserves signed raw values; the reader enforces bounds.

Validation order is new count limit, inherited limits/input/header/tables/mappings,
marker availability, supported marker, count availability, signed count, ceiling.
Every failure returns the empty `PlanPolygonCountPrefix`. Error contexts:

| Code | Field / zero-based offset |
| --- | --- |
| `invalid_limit` | `limits.max_points` / 0, for negative or above-default bound |
| `unsupported_element` | `plan.elements[0].kind` / marker start, for all non-1 bytes including 0/3 |
| `truncated` | `plan.elements[0].point_count` / marker end, for 0–3 available count bytes |
| `negative_count` | `plan.elements[0].point_count` / marker end |
| `resource_limit` | `plan.elements[0].point_count` / marker end, for count above caller bound |

Inherited errors, precedence and stopping contracts remain unchanged. P04c1
still accepts all 256 markers with no payload. No available-point-byte preflight
or point allocation occurs in P04c2. The zero-table count prefix consumes 45
bytes; zero and 1,000,000 both succeed without a single following byte. Arbitrary
tails remain uninterpreted, including color, next marker, side mapping and trailer.

### Polygon count C#/IL and fresh native evidence

After the analysis map, inspected `Drawing.Read` (RVA `0x10a30`, marker 1
constructs Polygon), `Polygon.Read` (`0xc4b8`) and matching IL. `IL_0007`
reads signed Int32, `IL_000f` immediately allocates `System.Drawing.Point[]`,
then the loop reads signed X/Y Int32 pairs before the color byte. Native zero
skips that loop but still reaches color; negative counts reach `newarr` with a
negative length. These allocation/payload observations are from C#/IL, not a
full native Polygon execution. P04c2 intentionally stops before all of them.

The separate [count probe](scripts/reference-plan-polygon-count-probe.cs)
ran against hash-verified original 1.372 under Wine Staging 11.7 / Microsoft
.NET `2.0.50727.42` x86, macOS arm64. Original table/mapping methods locate
marker 1, then the `BinaryReader` held by original `FileReader` reads one count.
No `Drawing.Read`/`Polygon.Read`, point allocation, coordinate/color read, GUI,
TOP write or Go oracle participates. Final probe exit was 0 with empty stderr.

The pinned helper's first three-point Polygon and fresh native positions agree:
`api-drawings.top` has marker `[146,147)`, count `[147,151)` = 3, consumed 151
and full-file tail 529. Exact count-only streams agree in both pixel modes 5/10.
Literal little-endian count streams confirm 0, 1, 3, 66051, 1,000,000, 1,000,001,
Int32.MaxValue, -1 and Int32.MinValue, with no tail and arbitrary two-byte tails.
The scalar reader accepts those raw values; complete native Polygon acceptance
is not claimed. Native 0–3-byte truncated reads throw `EndOfStreamException`
after consuming available bytes; Go reports the stable field start and no result.

Reference `Polygon.cs` SHA-256 is
`63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`;
all earlier C#/IL and original EXE/runtime hashes remain unchanged. The pinned
helper, fixtures and earlier native evidence were preserved byte-for-byte.
Exact 33-line CRLF stdout, new probe/output hashes, provenance and actual optional
commands are in [fixture provenance](internal/top/testdata/README.md).
JKTZ's earlier signed/resource checks were reviewed against C#/IL; its point-byte
preflight, full Polygon/color reading and 2,000,000-point budget were not adopted.

### P04c2 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1 and Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke. **100% statement
  coverage** in each implemented package (`internal/cli`, `internal/source`,
  `internal/top`). Tests cover signed/endian/count/limit boundaries, zero,
  every required count byte, all non-1 markers, native positions, variable earlier
  tables, inherited errors/limits/stops and immutable copies. Count-only zero and
  1,000,000 prefixes have equal allocation counts; no point table is allocated.
- `bash scripts/mutation.sh`: final complete integrated campaign passed with
  Gremlins **186/186 killed** (13 new count-reader mutants and the unchanged
  173 earlier mutants), no lived, uncovered, invalid, skipped or timed-out cases.
  The same run killed **6/6 station-ID, 19/19 trip, 35/35 measurement, 33/33
  reference, 29/29 overview, 32/32 plan mapping, 25/25 marker and 42/42 count
  explicit mutants**. Count faults discard/narrow/normalize raw fields, confuse
  spans/mappings, expose aliases, misread endian order/position, replace bounds,
  reorder validation, change error contexts, accept non-1 markers, alter input,
  return partial failures or require point/color bytes. Every explicit mutant
  compiled and failed a named behavioral assertion. No gates or exclusions changed.
- `bash scripts/mutation-trial.sh`: ordinary weak tests passed, both CLI mutants
  survived and the shared gate rejected the run with exit 1. Deliberate compiler
  and setup errors returned NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanPolygonCountPrefix$' -fuzztime=10s -parallel=2`:
  passed, **1,002,644 executions**. Inputs are bounded to 4096 bytes, earlier
  tables to 32 records, comments to 256 bytes and point count to 32. Properties
  assert deterministic values/errors, unchanged input, empty failures, exact
  marker/count/error precedence, inherited mappings/spans and prefix/tail accounting.
  Ordinary checks run all earlier and new fuzz seeds.

An initial explicit attempt rejected a linker disk-space error; only the
repository's regenerable Go build cache was cleared. A later integrated attempt
rejected a compiler error caused by quoting in the final explicit-fault generator.
After fixing that generator, the entire integrated campaign above passed.
Neither incomplete attempt was counted as success or as a behavioral kill.
Production code/tests were unchanged by the generator correction.

Measured Git blobs: `051df23019b2e56bd9c9d6d19049e4c9ed68734e` (model), `1cf5e7ab569f89ddff2c92d2839cd699395b7227` (model tests),
`e85171bcdaca87c0d4bcc87f43a65ea5e1547740` (reader), `4b8ba28e1f55009f69c620d293286c85235a59e3` (reader tests).

Independent read-only review found no Critical, Important or Minor issues.
Its targeted uncached race tests, formatting/shell checks, native/reference hashes,
prior evidence preservation and final report/blob checks passed.

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-polygon-count-mutation.json` and all existing explicit reports.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands including the new count scope. Exact pushed SHA and hosted CI are
verified in the chat handoff; README is part of that delivered commit. P04c2
stops after the count and implements no points/color or later P04 slice.

## Completed PBI: P04c3 — first plan Polygon vertices only

**Outcome/dependencies:** a separate immutable prefix extends P04c2 through
only the first Polygon's ordered raw point table. Keep every existing API,
raw value, span, limit, error precedence and stopping contract. P04c3 was
explicitly authorized on 2026-10-09; P04c2 still reads no point or color bytes.

**References:** analysis map, `Polygon.Read`/`Write` signed X/Y order and matching
IL; unchanged pinned helper/`api-drawings.top`. Extend a separate original
`FileReader`/`BinaryReader` probe through the points, stopping before color.
No large native allocation or Go-generated oracle is needed. The first three native points
are `(-6000,-2000)`, `(-5500,-1000)`, `(-4500,-1500)` with point-table
span `[151,175)`, independently established by the helper's literal coordinates.

**Bounded acceptance contract:**

- Reuse P04c2 and `PolygonCountLimits`, adding no limit. Preserve ordered signed
  little-endian Int32 X/Y values without units/float/geometry conversion. Retain
  each point's record/X/Y spans, copied point collections, copied consumed bytes
  and all inherited records/mappings/spans. No color or full Polygon model.
- Before allocation, require `count <= remaining_bytes / 8`; avoid unchecked
  multiplication. Failed preflight returns `truncated` at
  `plan.elements[0].points`, the count end. All failures return an empty new
  result; inherited negative/resource/unsupported-element precedence is unchanged.
- Stop immediately after the points, before color. Zero points succeed at the
  P04c2 count end without reading color or another byte. Exact point-only prefixes
  and arbitrary tails succeed within input bounds. Require no next marker,
  side mapping or trailer. P04c2 still accepts large counts without point bytes.
- Test zero/one/multiple points, signed/endian boundaries, preserved order, every
  required-byte truncation, exact/lower limits, immutable copies, variable earlier
  tables and all earlier stops. Confirm the pinned native coordinates/positions;
  add bounded fuzzing and explicit coordinate/order/span/copy/overread mutants.

**Non-goals:** color, element loops, XSections, side mapping/elements,
geometry/rendering, complete-file validation, directory context, CLI operations
and exporters. Refine the next color-only slice after P04c3; do not implement
it in the same chat.

**Required checks/handoff:** current check, complete mutation and negative-control
commands; >=95% coverage, >=90% killed mutants, every survivor reviewed, no
incomplete/error run accepted. Independent review, unchanged original evidence,
README update, commit/push, exact-SHA green CI and one successor prompt.

### Implemented points API and source policy (2026-10-09)

`internal/top.ReadV3PlanPolygonPointsPrefix` and
`ReadV3PlanPolygonPointsPrefixWithLimits` extend the unchanged P04c2 count
reader through ordered raw X/Y pairs. Both reuse `PolygonCountLimits` and all
inherited bounds, including the input limit on the full tail. No existing
production API, reader, limit, error or stopping position was changed.

`source.PolygonPoint` exposes signed Int32 `XRaw`/`YRaw` and
`PolygonPointOffsets` (`Record`, `X`, `Y`). `PlanPolygonPointsPrefix` retains
all earlier records/mappings/accessors, adds copied `Points`, and exposes
`PlanPolygonPointsPrefixOffsets` embedding all earlier spans plus `Points`.
The constructor and accessors copy point collections and consumed bytes;
points and spans are values. No unit/float conversion, normalization,
bounding-box derivation, color or complete Polygon model is introduced.

After P04c2 succeeds, the reader checks `count <= remaining_bytes / 8` before
allocating points. This avoids unchecked count multiplication. Failure returns
an empty `PlanPolygonPointsPrefix` and `truncated` at `plan.elements[0].points`,
the count end. All earlier limit/header/table/mapping/marker/count validation
and error precedence remain inherited. A successful zero-point table has an
empty span at count end and consumes no further byte. Positive tables consume
exactly eight bytes per point and stop before color, accepting absent or
arbitrary tails. P04c2 still accepts count-only 1,000,000 with no point bytes.

### Polygon vertices C#/IL and fresh native evidence

Inspected the analysis map, `Polygon.Read`/`Write` and matching IL. Read uses
signed X/Y Int32 calls at `IL_0041`/`IL_004e` and stores each pair in order;
Write emits X/Y at `IL_003e`/`IL_005b`. Native `Polygon.Read` subsequently
derives a rectangle and reads color at `IL_00f0` even with zero points.
This prefix stops before both those operations.

The separate [points probe](scripts/reference-plan-polygon-points-probe.cs)
uses original table/mapping methods and the scalar `BinaryReader` held by
original `FileReader`. It does not call `Polygon.Read` or use Go output.
The hash-verified pinned helper independently establishes the three fixture
points `(-6000,-2000)`, `(-5500,-1000)`, `(-4500,-1500)`. Fresh native readback
confirms their table `[151,175)`, consumed 175 and full-file tail 505. Exact
prefixes agree in both pixel modes; every one of 24 required-byte truncations
throws `EndOfStreamException`. Literal zero/one/three-point streams verify
signed/endian boundaries, order, no-color stopping and arbitrary tails.
Native failed scalar reads consume available bytes; the Go preflight reports
the stable table start and returns no partial result.

Final probe exit was 0 with empty stderr under the installed original 1.372,
Wine Staging 11.7 / Microsoft .NET 2.0.50727.42 x86 on macOS arm64. Its 61-line
CRLF stdout, hashes and actual commands are recorded in
[fixture provenance](internal/top/testdata/README.md). Original EXE/runtime,
C#/IL, pinned helper and earlier fixtures/probes/stdout are unchanged.
The earlier JKTZ signed pair/minimum-byte checks were reviewed against C#/IL;
its whole drawing/color rules and aggregate point budget are not adopted.

### P04c3 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke. **100% statement
  coverage** in each implemented package. Tests cover zero/one/multiple points,
  Int32 and endian boundaries, order, point/table spans, all required-byte
  truncations, exact/lower/default limits, immutable copies, variable earlier
  tables, inherited errors and stops, native coordinates and all 256 tail bytes.
- `bash scripts/mutation.sh`: complete integrated campaign passed with
  **202/202 Gremlins mutants killed** (16 new points-reader mutants and the
  unchanged 186 earlier mutants), no lived, uncovered, invalid, skipped or
  timed-out cases. The same run killed **258/258 explicit mutants**: 6 station,
  19 trip, 35 measurement, 33 reference, 29 overview, 32 plan mapping, 25 marker,
  42 count and **37 points**. New faults discard/narrow/normalize coordinates,
  swap X/Y or point order, corrupt spans, alias collections/bytes, replace caller
  limits, change preflight/error contexts, mutate input, return partial results,
  consume tails or require color (including for zero). Every explicit mutant
  compiled and failed a named behavioral test. No gates or exclusions changed.
- `bash scripts/mutation-trial.sh`: passed. Ordinary weak tests passed, both CLI
  mutants survived, and the shared gate rejected them with exit 1. Deliberate
  compiler/setup failures returned NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanPolygonPointsPrefix$' -fuzztime=10s -parallel=2`:
  passed, **710,665 executions**. Limits bound inputs to 4096 bytes, earlier
  tables/points to 32 and comments to 256 bytes. Properties check deterministic
  values/errors, unchanged input, empty failures, inherited validation/spans,
  signed coordinates/order, table preflight and exact prefix/tail accounting.
  Ordinary checks also execute all earlier and new fuzz seeds.

Measured Git blobs: `bc2cebfbb3c87a61779ebbbb7460a3962265c7e9` (model),
`e013718eb444d8a63e26c2e58cbc31fb842677fd` (model tests),
`1cbefbc4bb4b6e48cb8302cb92974cb582e6d247` (reader),
`e43045364b449582d5812797bbe9a1d80c017fd8` (reader tests).

Independent read-only review found no Critical, Important or Minor issues.
The reviewer independently passed targeted uncached race tests/fuzz seeds,
formatting/shell/diff checks, verified 74 preserved reference/evidence hashes,
new native output/probe hashes and measured source/test blobs, and reviewed
this implementation, provenance and the successor contract. Full native
Polygon acceptance, derived geometry and later drawing/export work remain
outside this slice.

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-polygon-points-mutation.json` and all existing explicit reports.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands including the new points scope. Exact pushed SHA and green hosted CI
are verified in the chat handoff; README is part of that commit. P04c3 ends here.

## Completed PBI: P04c4 — first plan Polygon color byte only

**Outcome/dependencies:** a separate immutable prefix extends P04c3 by exactly
one raw color byte after the first Polygon's point table, including zero points.
P04c4 was explicitly authorized on 2026-10-09. Preserve every earlier API, raw value,
span, limit, error precedence, immutable-copy contract and stopping position.

**References:** analysis map, `Polygon.Read` color switch (`IL_00f0` onwards),
`Polygon.Write`, matching IL, unchanged pinned helper and `api-drawings.top`.
The first fixture color is black/code 1 at `[175,176)` according to the helper
and write layout; confirm this with a fresh bounded original scalar probe.
Extend a separate `FileReader`/`BinaryReader` probe by one byte, keeping all
prior evidence unchanged. Verify every byte without invoking GUI or writing TOP.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanPolygonPointsPrefixWithLimits` and `PolygonCountLimits`,
  adding no limit. Retain every inherited point/record/mapping/span and copied
  consumed bytes; add raw UInt8 color and its one-byte span to a separate
  prefix model. Do not introduce rendering, derived pen objects or color names
  into source fields.
- Require exactly one color byte even for zero points. Missing color returns
  `truncated` at `plan.elements[0].color`, the points end. Every failure returns
  an empty new result. Inherited validation and point-preflight precedence stay
  unchanged. P04c3 still succeeds without color, including zero points.
- Preserve all 256 raw byte values. Document native mapping from C#/IL: 1 black,
  2 gray, 3 brown, 4 blue, 5 red, 7 orange, default green (including 6 and unknown
  codes). Do not reject or normalize unknown codes to 6; source bytes and later
  rendering decisions remain separate.
- Stop immediately after color; accept exact prefixes and arbitrary tails.
  Require no next element marker, terminator, side mapping or trailer.
- Test all color bytes with zero/one/multiple points, missing/exact/arbitrary
  tails, variable earlier tables, input/count bounds, inherited errors/stops,
  raw/copy/span preservation and pinned native position. Add bounded fuzzing
  and explicit raw-color/span/copy/overread mutations.

**Non-goals:** next markers, element loops, XSections, side mapping/elements,
full-file validation, geometry/rendering, directory context, CLI and exporters.
Refine one successor slice after P04c4; do not implement it in the same chat.

**Required checks/handoff:** current check, complete mutation and negative-control
commands; >=95% coverage, >=90% killed mutants, every survivor reviewed and no
incomplete/error run accepted. Independent review, preserved original evidence,
README update, commit/push, exact-SHA green CI and one successor prompt.

### Implemented color API and source policy (2026-10-09)

`internal/top.ReadV3PlanPolygonColorPrefix` and
`ReadV3PlanPolygonColorPrefixWithLimits` extend the unchanged P04c3 reader
by one `take(1, "plan.elements[0].color")`. They reuse `PolygonCountLimits`
with no added limits. All inherited validation, point preflight, raw records,
spans, caller bounds and error precedence remain unchanged. The input ceiling
still includes the whole unparsed tail. Every earlier API stops as before.

`source.PlanPolygonColorPrefix` retains all earlier accessors and copied
points/records, and adds `ColorRaw` (UInt8). `PlanPolygonColorPrefixOffsets`
embeds all P04c3 spans and adds `Color`, the one-byte span. Constructors and
accessors copy consumed bytes; inherited collection accessors copy their data.
Offsets and raw point/mapping values are returned by value. There is no color
name, pen, derived bounding rectangle or normalization in the source model.

Missing color, including after zero points, returns `truncated` at
`plan.elements[0].color`, offset equal to the points end. Every failure returns
an empty `PlanPolygonColorPrefix`. All 256 bytes succeed and remain unchanged.
The reader consumes exactly one byte and stops: exact prefixes, malformed or
arbitrary tails require no next marker, termination, side mapping or trailer.
The zero-table/zero-point prefix consumes 46 bytes; P04c3 still succeeds at 45
with no color, and P04c2 still accepts large counts without any point bytes.

### Polygon color C#/IL and fresh native evidence

After the analysis map, inspected `Polygon.Read` (RVA `0xc4b8`),
`Polygon.Write` (`0xc350`) and matching IL. UInt8 color is read at `IL_00f0`
after the point loop and native derived rectangle, even for zero points.
Read selects 1 black, 2 gray, 3 brown, 4 blue, 5 red, 7 orange and default
green (including 6 and unknown codes). Write emits the named codes and default
6. These rendering/write decisions are documented from C#/IL; they are separate
from P04c4's raw preservation. Native rendering and rewriting were not run.

The separate [color probe](scripts/reference-plan-polygon-color-probe.cs)
uses original table/mapping methods and the scalar `BinaryReader` held by
original `FileReader`, stopping immediately after color. It calls no drawing
or Polygon reader, native writer or GUI, and uses no Go-generated oracle.
The unchanged hash-verified pinned helper's first Polygon is black; fresh
original readback confirms code 1 at `[175,176)`, consumed 176, full-file tail
504. Exact/missing color streams agree in native pixel modes 5/10. Missing
color throws `EndOfStreamException` without advancing.

Every raw color byte was read on independent zero/one/three-point scalar
streams, with exact prefixes and arbitrary two-byte tails; all consume one
byte. Missing color fails in each point-count case. Final native exit was 0,
stderr empty, stdout 1,548 exact CRLF lines under original assembly 1.3.7.0,
Wine Staging 11.7 / Microsoft .NET 2.0.50727.42 x86 on macOS arm64.
[Fixture provenance](internal/top/testdata/README.md) records the row checks,
new probe/output hashes and actual optional commands. Original EXE/runtime,
C#/IL, pinned helper, TOP fixtures and all earlier probes/stdout remain unchanged.
The hash-verified JKTZ parser's `1..7` color bound was reviewed against C#/IL
and not adopted. No reference archives or generated artifacts were edited.

### P04c4 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke. **100% statement
  coverage** in each implemented package. Tests cover all 256 colors with
  zero/one/multiple points, all 256 first tail bytes, missing/exact/arbitrary
  tails, variable earlier tables, immutable copies, inherited limits/errors/
  point preflight/stops, every required-byte truncation and native position.
- `bash scripts/mutation.sh`: complete integrated campaign passed with
  **206/206 Gremlins mutants killed** (four new color-reader mutants and the
  unchanged 202 earlier mutants), no lived, uncovered, invalid, skipped or
  timed-out cases. The same run killed **291/291 explicit mutants**: 6 station,
  19 trip, 35 measurement, 33 reference, 29 overview, 32 plan mapping, 25 marker,
  42 count, 37 points and **33 color**. Color faults discard/mask/normalize raw
  values, discard inherited records/points/spans, expose byte aliases, replace
  caller limits, bypass point preflight, corrupt positions/errors, return partial
  failures, omit zero-point color, reject unknown colors, mutate input, consume
  tails or require a next marker. Every explicit mutant compiled and failed a
  named behavioral test. No gates, exclusions or earlier scopes changed.
- `bash scripts/mutation-trial.sh`: passed. Weak ordinary tests passed and
  both CLI mutants survived; the shared gate rejected them with exit 1.
  Deliberate compiler/setup failures returned NOT VIABLE exit 2.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanPolygonColorPrefix$' -fuzztime=10s -parallel=2`:
  passed, **716,576 executions**. Bounds are 4096 input bytes, 32 earlier
  records/points and 256 comment bytes. Properties check deterministic values/
  errors, unchanged input, empty failures, inherited validation/point preflight/
  records/spans, raw color and exact one-byte prefix/tail accounting. Ordinary
  checks execute all earlier and new fuzz seeds.

Measured Git blobs: `07cc10e66592dfedc173eee1cd636a9319906a4c` (model),
`eda59880f03186f7823b8c09db5a5e8081a4bca3` (model tests),
`618dfc91986ec5eb40cf39b71ed13fa1a872f5c3` (reader),
`6f09cecc2e5c8c3e382d742bc0211f1dc629e277` (reader tests).

The first focused color-mutation attempt correctly rejected a noncompiling
wrong-byte fault (unused local). Only the generator expression was corrected;
production code/tests were unchanged. The focused rerun and final complete
integrated campaign passed. The rejected attempt was not counted as a kill
or successful campaign.

Independent read-only review found no Critical, Important or Minor issues.
The reviewer independently passed uncached race tests including earlier/new
fuzz seeds, formatting/shell/diff checks, all 1,548 native rows/CRLF, new
probe/output/binary hashes and the four measured source/test blobs. All 133
preserved paths in the 135-path source/evidence snapshot remained unchanged;
only the two planned infrastructure files changed. The reviewer also checked
C#/IL behavior, native evidence boundaries and the separate P04c5 contract.

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-polygon-color-mutation.json` and all earlier explicit reports.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands including the new color scope. The exact pushed SHA and green hosted
CI are verified in the chat handoff; README belongs to that commit.
P04c4 stops here and implements no next marker or later drawing/export work.

## Completed PBI: P04c5 — next plan element marker only

**Outcome/dependencies:** a separate immutable prefix extends P04c4 by exactly
one raw marker byte immediately after the first Polygon color. Implemented
on 2026-10-09. Preserve every earlier API, raw field, span, limit,
error precedence, copy contract and stopping position, including color-only
success without a next byte.

**References:** analysis map, `Drawing.Read`'s next `ReadByte` (`IL_00c3`),
its 0/1/3/unknown dispatch and matching IL; unchanged pinned helper and
`api-drawings.top`. The helper's next plan element is another Polygon; format
layout places its marker at `[176,177)`. Confirm code/position with a fresh
bounded original scalar probe extending only through this next byte. Preserve
all earlier probes/output. Do not invoke drawing readers, GUI or TOP writes.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanPolygonColorPrefixWithLimits` and `PolygonCountLimits`,
  adding no limit. Retain all raw fields/points/records/mappings/spans and copied
  consumed bytes; add a separate next-marker UInt8 accessor and one-byte span.
- Missing marker returns `truncated` at `plan.elements[1].kind`, the color
  end. Every failure returns an empty new result; all inherited validation and
  point/color error precedence remains unchanged.
- Accept and preserve all 256 marker bytes, including 0, 1, 3 and unknown
  values. Document native dispatch without executing it. Stop after this byte,
  even for 0; require no payload, later marker, side mapping or trailer.
- Test zero/one/multiple first-Polygon points, all marker/color bytes, missing/
  exact/arbitrary tails, variable earlier tables, inherited bounds/errors/stops,
  copies/spans/raw fields and pinned native position. Add bounded fuzzing and
  explicit marker/span/copy/overread mutations.

**Non-goals:** second payload, element loops, XSections, side mapping/elements,
full-file validation, geometry/rendering, directory context, CLI and exporters.
Refine one successor after P04c5; do not implement it in that same chat.

**Required checks/handoff:** current check, complete mutation and negative-control
scripts; >=95% coverage, >=90% killed mutants, every survivor reviewed and no
incomplete/error run accepted. Independent review, preserved original evidence,
README update, commit/push, exact-SHA green CI and one successor prompt.

### Implemented next-marker API and evidence (2026-10-09)

`internal/top.ReadV3PlanNextMarkerPrefix` and
`ReadV3PlanNextMarkerPrefixWithLimits` extend the unchanged color reader by
`take(1, "plan.elements[1].kind")`, reusing `PolygonCountLimits` without a new
limit. `source.PlanNextMarkerPrefix` retains every inherited accessor and adds
`NextMarkerRaw` (UInt8). `PlanNextMarkerPrefixOffsets` embeds the color spans
and adds `NextMarker`. Consumed bytes and collection accessors are copied;
fields and offsets are returned by value.

All 256 markers succeed unchanged, including 0. Missing marker returns an empty
new result with `truncated` at `plan.elements[1].kind`, the color end. Earlier
validation, limits, point preflight and error precedence remain intact. The
whole-input ceiling still includes the unparsed tail. The zero-table/zero-point
prefix ends at 47; the earlier color reader still succeeds at 46 without a next
byte. No payload, later marker, side mapping or trailer is required or read.

After the analysis map, inspected `Drawing.Read` (RVA `0x10a30`) and IL:
next `ReadByte` is `IL_00c3`, following the element read at `IL_007f`.
`IL_00ca..00d2` terminates on 0; 1 dispatches Polygon, 3 XSection, and unknown
nonzero markers execute `ReadBytes(0)` before the next byte. `Polygon.Read`
reads color at `IL_00f0`. This dispatch is documented, not executed by P04c5.
The hash-verified JKTZ parser blob `74965043ebb9600ad71089bb079e76eaad1268f3`
was reviewed: its unknown-marker rejection and complete drawing loop were not
adopted. The pinned helper appends another Polygon after the first, independently
establishing marker 1 at `[176,177)` in unchanged `api-drawings.top`.

The separate [native probe](scripts/reference-plan-next-marker-probe.cs) uses
original table/mapping methods and the `BinaryReader` held by original
`FileReader`, stopping after exactly that byte. It invokes no drawing reader,
GUI or TOP write and uses no Go-generated oracle. Full/exact/missing fixture
streams agree in native pixel modes 5/10: consumed 177, full-file tail 503;
missing marker throws `EndOfStreamException` at 176 without advancing.
Independent zero/one/three-point literal streams cover all marker bytes and all
complementary color bytes with exact and arbitrary tails. All **1,548 CRLF rows**
were independently checked; final probe exit was 0 and stderr empty. These are
bounded scalar observations, not full native drawing/export compatibility.
[Fixture provenance](internal/top/testdata/README.md) records hashes and commands.
Original EXE/runtime, C#/IL, helper, TOP fixtures and all earlier probes/stdout
remain unchanged; the snapshot differences are limited to the planned
`.gitattributes`, mutation orchestration and fixture documentation additions.

### P04c5 checks and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and CLI smoke; **100% statement coverage**
  in each implemented package. Tests cover zero/one/multiple points, all marker
  and color bytes, missing/exact/arbitrary tails, variable earlier tables,
  native positions, immutable copies, inherited limits/errors and earlier stops.
- `bash scripts/mutation.sh`: complete integrated campaign passed with
  **210/210 Gremlins mutants killed**, with no lived, uncovered, invalid, skipped
  or timed-out results. The same run killed **326/326 explicit mutants**: the
  unchanged 291 earlier faults and **35 new next-marker faults** covering raw
  marker preservation, inherited records/color/spans, byte copies, caller limits,
  missing-color precedence, positions/errors, partial failures, input mutation,
  tail consumption, payload overread and side-mapping reads for marker 0.
  Each compiled and failed a named behavioral test; earlier scopes/gates stayed.
- `bash scripts/mutation-trial.sh`: passed. Weak tests pass, both CLI mutants
  live and the shared gate rejects them with exit 1. Compiler/setup errors
  return NOT VIABLE exit 2.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanNextMarkerPrefix$' -fuzztime=10s -parallel=2`:
  passed, **1,003,245 executions**. Bounds: 4096 input bytes, 32 earlier records/
  points, 256 comment bytes. Properties verify deterministic results/errors,
  unchanged input, empty failures, inherited validation/fields/spans, raw marker
  and exact one-byte accounting. Ordinary checks execute all fuzz seeds.

Independent read-only review found no outstanding Critical, Important or Minor
issues. The reviewer passed uncached race tests, formatting/shell/diff checks,
all native rows/CRLF and new hashes, preserved-evidence checks, C#/IL inspection,
and the focused 35/35 mutation report. Two stale documentation scope summaries
were corrected and rechecked. Final delivery gates are verified separately.

Measured Git blobs: `fe3068f521ef18f514ebe89be4825442df9f24ed` (reader),
`60192e9f92fa53b159cd2c625065f8b6473b6f6c` (reader tests),
`0973cfe18196e5aac97d27ab3df70a5727bdf369` (model),
`1a62d12060d81a7a6e54dbaa42ce9f1641527ec3` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-next-marker-mutation.json` and earlier reports. Linux CI repeats both
complete mutation scripts; all three platforms run ordinary checks. Independent
review and the exact pushed SHA/green CI are verified in the chat handoff.
P04c5 stops here; no second payload or later drawing/export work is implemented.

## Completed PBI: P04c6 — second plan Polygon point count only

**Outcome/dependencies:** a separate immutable prefix extends P04c5 by one
signed little-endian Int32 count only when `NextMarkerRaw() == 1`. Implemented
on 2026-10-10. Preserve all earlier APIs, limits, errors, spans, copy contracts
and stopping positions, including P04c5 success for every marker with no payload.
No second-point allocation or second vertices/color reading.

**References:** analysis map, `Drawing.Read`/`Polygon.Read`/`Polygon.Write` and
matching IL; unchanged helper's second plan Polygon and `api-drawings.top`.
Its count is 3 at `[177,181)`. Confirm with a new bounded original scalar probe,
without changing earlier evidence or invoking drawing readers, GUI or TOP writes.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanNextMarkerPrefixWithLimits` and `PolygonCountLimits` with
  no new limit. Retain all earlier fields and add a distinct second-Polygon
  raw count accessor and four-byte span.
- Check the next marker before count: non-1 markers, including 0/3/unknown,
  return `unsupported_element` at `plan.elements[1].kind`, its marker start.
  Missing count returns `truncated` at `plan.elements[1].point_count`, marker
  end. Negative counts return `negative_count`; counts above `MaxPoints` return
  `resource_limit` at that same field/start. Every failure returns an empty new
  result; inherited validation and error precedence remain unchanged.
- Accept 0 through the inclusive caller `MaxPoints` (default/hard ceiling
  1,000,000), without requiring or allocating points or color. Apply the existing
  ceiling to this count separately; aggregate whole-drawing budgets are deferred.
  Stop after four count bytes, including zero. Preserve arbitrary tails.
- Test signed/endian/limit boundaries, every count truncation, all next markers,
  zero/one/multiple first points, inherited limits/errors/stops, variable earlier
  tables, copies/spans and native position. Add bounded fuzzing and explicit
  dispatch/count/span/copy/overread mutations.

**Non-goals:** second vertices/color, element loops, XSections, plan completion,
side mapping/elements, full-file validation, geometry, CLI and exporters.

**Required checks/handoff:** all three existing check scripts, >=95% coverage,
>=90% killed mutations with every survivor reviewed, independent review, README,
preserved evidence, commit/push, exact-SHA green CI and one next small PBI prompt.
Stop after P04c6; do not implement its successor in the same chat.

### Implemented second-count API and evidence (2026-10-10)

`internal/top.ReadV3PlanSecondPolygonCountPrefix` and
`ReadV3PlanSecondPolygonCountPrefixWithLimits` reuse the unchanged next-marker
reader and `PolygonCountLimits`. Only next marker 1 reaches `take(4,
"plan.elements[1].point_count")`, decoded as signed little-endian Int32.
`source.PlanSecondPolygonCountPrefix` retains every inherited accessor and adds
`SecondPointCountRaw`; its offsets embed `PlanNextMarkerPrefixOffsets` and add
`SecondPointCount`. `PointCountRaw` and `Points` still describe the first Polygon.
Consumed bytes and inherited collections are copied; offset structs are values.

Non-1 markers fail at `plan.elements[1].kind`, the next marker start, before
count availability is checked. Missing/incomplete, negative and above-limit
counts fail at `plan.elements[1].point_count`, its start, with `truncated`,
`negative_count` and `resource_limit` respectively. Every failure returns an
empty new result. Inherited validation, error precedence and limits remain
unchanged. The existing point ceiling applies to each count separately, with
zero and equality accepted; no aggregate budget is introduced. Exact prefixes
with second counts 0 through 1,000,000 succeed without second points or color.
The zero-table/zero-first-point prefix ends at 51; arbitrary tails remain unparsed.

Inspected the analysis map, Drawing.Read/Polygon.Read/Polygon.Write and matching
IL. Next marker ReadByte is IL_00c3; marker 1 selects Polygon and calls its Read
at IL_007f. Polygon.Read (RVA 0xc4b8) begins with signed ReadInt32 at IL_0007,
then native newarr at IL_000f. Polygon.Write (RVA 0xc350) emits marker 1 before
its Int32 length at IL_001c. P04c6 stops before second-point allocation/preflight,
vertices, derived bounds or color. Native allocation behavior is source evidence,
not a full native Polygon execution.

The unchanged pinned helper's second gray plan Polygon has three points;
format layout independently locates its count at `[177,181)`. The separate
[original scalar probe](scripts/reference-plan-second-polygon-count-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader. It reads
first Polygon scalars, next marker 1, then exactly one second count. No drawing
reader, GUI, TOP write or Go-generated oracle participates. Actual native results,
hashes and reproduction commands are recorded in
[fixture provenance](internal/top/testdata/README.md).

### P04c6 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke. **100% statement
  coverage** in each implemented package. Tests cover signed/endian/count/limit
  boundaries, every count truncation, all next markers, zero/one/multiple first
  points, variable earlier tables, inherited limits/errors/stops, copied fields/
  records/spans and native positions. Allocation counts for second counts 0 and
  1,000,000 are equal with no second payload; the two ceilings apply separately.
- `bash scripts/mutation.sh`: the complete integrated campaign passed with
  **219/219 Gremlins mutants killed** (nine new second-count reader mutants and
  the unchanged 210 earlier mutants), with no lived, uncovered, invalid, skipped
  or timed-out results. The same run killed **376/376 explicit mutants**: the
  unchanged 326 earlier faults and **50 new second-count faults**. New faults
  cover dispatch/count/endian/signed preservation, separate inclusive budgets,
  inherited fields/spans, limits/error precedence/contexts, input/output copies,
  empty failures, input mutation and second-point/color/tail overreads. Each
  compiled and failed a named behavioral test; setup errors and timeouts are
  rejected. No earlier scopes, gates or exclusions changed.
- `bash scripts/mutation-trial.sh`: passed. Weak ordinary tests pass and both CLI
  mutants live; the shared gate rejects them with exit 1. Compiler/setup-error
  controls return NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanSecondPolygonCountPrefix$' -fuzztime=10s
  -parallel=2`: passed, **1,069,772 executions**. Bounds: 4096 input bytes,
  32 earlier records/first points/second count, 256 comment bytes. Properties
  check deterministic values/errors, unchanged input, empty failures, inherited
  fields/spans/validation, dispatch/count precedence and exact four-byte
  consumed-prefix/tail accounting. Ordinary checks execute every fuzz seed.

Independent read-only review found no Critical, Important or Minor issues.
The reviewer independently passed fresh `go test -count=1 ./...` in isolated
temporary caches, checked C#/IL and the helper, all 87 native rows and CRLF,
new source/stdout/binary hashes, measured Go blobs, the 50-fault script and
its report, preserved evidence, API contracts and the separate P04c7 scope.
Only four planned existing files differ in the 153-path source/evidence
snapshot: README, fixture provenance, `.gitattributes` and mutation orchestration;
all 149 other paths remain unchanged. Original EXE/runtime, C#/IL, helper,
TOP fixtures and every prior probe/stdout remain byte-for-byte intact.

Measured Git blobs: `82e0dd8a6811d030e4e893bb4a6a54f92d2066f0` (reader), `cad0241c699e588e4636110f5b39dae50e1c5199` (reader tests),
`550ae29899976acd51762a94f2f95b9489b5709d` (model), `fbdda2708ed28b17a5d7a3f733cbaeb7a84d37f2` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-second-polygon-count-mutation.json` and all earlier reports.
All three CI platforms run ordinary checks; Linux runs both complete mutation
commands including the second-count scope. Exact pushed SHA and green hosted
CI are verified in the chat handoff; README belongs to that commit.
P04c6 stops here; second vertices/color and later drawing/export work remain
unimplemented.

## Completed PBI: P04c7 — second plan Polygon vertices only

**Outcome/dependencies:** a separate immutable prefix extends P04c6 through
only the second Polygon's ordered raw point table. Implemented on 2026-10-10.
Preserve every earlier API, limit, raw field, record, span, error
precedence, copy contract and stopping position. P04c6 still accepts its full
count range with no second-point bytes.

**References:** analysis map, Polygon.Read/Write signed X/Y order and matching
IL; unchanged pinned helper and `api-drawings.top`. The second gray Polygon's
points are `(-4000,-2000)`, `(-3500,-1000)`, `(-2500,-1500)`, table `[181,205)`.
Confirm through a new bounded original scalar probe stopping before second color;
preserve all earlier evidence and invoke no drawing readers, GUI or TOP writes.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanSecondPolygonCountPrefixWithLimits`, `PolygonCountLimits`
  and `source.PolygonPoint`, adding no limit. Add copied `SecondPoints` and a
  separate `SecondPoints` table span; existing `Points` continues to mean first
  Polygon points. Preserve ordered signed little-endian Int32 X/Y and each
  record/X/Y span without units, float conversion or geometry.
- Before allocating second points, require `second_count <= remaining_bytes / 8`.
  Failed preflight returns `truncated` at `plan.elements[1].points`, the second
  count end. Every failure returns an empty new result; inherited dispatch,
  negative/resource/limit/error precedence remains unchanged.
- Stop immediately after second vertices, before color. Zero second points
  succeed at the P04c6 count end without another byte. Exact prefixes and
  arbitrary tails succeed within inherited input bounds. Require no second
  color, next marker, side mapping or trailer; introduce no aggregate budget.
- Test zero/one/multiple points, signed/endian boundaries/order, every required
  byte, inclusive/lower limits, inherited errors/stops, variable earlier tables,
  copies/spans and pinned native position. Add bounded fuzzing and explicit
  coordinate/order/span/copy/preflight/overread mutations.

**Non-goals:** second color, element loops, XSections, plan completion, side
mapping/elements, full-file validation, geometry, CLI and exporters.

**Required checks/handoff:** all three existing check scripts, >=95% coverage,
>=90% killed mutations with every survivor reviewed, independent review, README,
preserved evidence, commit/push, exact-SHA green CI and one next small PBI prompt.
Stop after P04c7; do not implement its successor in the same chat.

### Implemented second-points API and evidence (2026-10-10)

`internal/top.ReadV3PlanSecondPolygonPointsPrefix` and
`ReadV3PlanSecondPolygonPointsPrefixWithLimits` extend the unchanged P04c6
reader through only ordered raw second X/Y pairs. Both reuse `PolygonCountLimits`;
all earlier bounds, dispatch, error precedence, source fields and stopping
positions remain intact. The existing ceiling applies separately to each
Polygon count; no aggregate budget or additional limit was introduced.

`source.PlanSecondPolygonPointsPrefix` retains every inherited accessor and
adds copied `SecondPoints`, using the unchanged `source.PolygonPoint`.
`PlanSecondPolygonPointsPrefixOffsets` embeds every P04c6 span and adds
`SecondPoints`. Existing `Points`, `PointCountRaw` and `ColorRaw` continue to
describe the first Polygon. Coordinates stay signed little-endian Int32;
record/X/Y/table spans stay zero-based with exclusive ends. Constructors and
accessors copy collections and consumed bytes; raw fields and spans are values.

After P04c6 succeeds, `second_count <= remaining_bytes / 8` is required before
allocating second points. Failed preflight returns the empty new result and
`truncated` at `plan.elements[1].points`, the second count end. Zero second
points succeed there with an empty table span and no additional byte. Positive
tables consume exactly eight bytes per point. Exact prefixes and arbitrary tails
succeed within inherited input limits, stopping before second color. P04c6
still accepts count-only 1,000,000 without second-point bytes.

After the analysis map, inspected `Polygon.Read` (RVA `0xc4b8`) and `Write`
(`0xc350`), `Drawing.Read` (`0x10a30`) and matching IL. Read uses signed X/Y
at `IL_0041`/`IL_004e`, in order; Write emits them at `IL_003e`/`IL_005b`.
Native rectangle derivation precedes color at `IL_00f0`, even with zero points.
P04c7 stops before those operations. The unchanged pinned helper independently
establishes second points `(-4000,-2000)`, `(-3500,-1000)`, `(-2500,-1500)`.
Earlier JKTZ pair reads/preflight were reviewed against this evidence; its full
Polygon/color reads, color restrictions and aggregate budget were not adopted.

The separate [original scalar probe](scripts/reference-plan-second-polygon-points-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader. It
reads first Polygon scalars, then the second marker/count/vertices, with no
drawing reader, GUI, TOP write, large native allocation or Go oracle. Full and
exact fixture streams in pixel modes 5/10 confirm second table `[181,205)`,
consumed 205 and full-file tail 475. All 24 required-byte truncations throw
EndOfStreamException. Independent zero/one/three-first/second-point streams
confirm signed/endian boundaries, order, zero/no-color stopping and arbitrary
tails at second-table starts 11/19/35. Final native exit was 0, stderr empty,
and all **73 exact CRLF rows** were independently checked. Environment, hashes
and actual optional commands are in [fixture provenance](internal/top/testdata/README.md).
These scalar observations do not prove full native Polygon/export compatibility.
Original EXE/runtime, C#/IL, helper, TOP and all prior probe/stdout bytes are preserved.

### P04c7 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke. **100% statement
  coverage** in each implemented package. Tests cover zero/one/multiple points,
  signed/endian boundaries/order, all required-byte truncations, inclusive/lower
  limits, inherited errors/stops, variable earlier tables, immutable copies,
  distinct point tables/spans, native positions and all 256 first tail bytes.
  Failed-preflight allocated-byte checks reject allocation proportional to count.
- `bash scripts/mutation.sh`: final complete integrated campaign passed with
  **235/235 Gremlins mutants killed** (16 new reader mutants and the unchanged
  219 earlier mutants), with no lived, uncovered, invalid, skipped or timed-out
  results. The same run killed **424/424 explicit mutants**: all 376 earlier
  faults and **48 new second-vertex faults**. New faults cover raw coordinates,
  signed/endian/order preservation, point/table spans, first/second separation,
  copied collections/bytes, inherited fields/limits/errors, empty failures,
  preflight before allocation, input mutation, tail consumption and second-color
  overreads including zero. Each compiled and failed a named test; setup errors
  and timeouts are rejected. No gates, earlier scopes or exclusions changed.
- `bash scripts/mutation-trial.sh`: passed. Ordinary weak tests pass, both CLI
  mutants survive, and the shared gate rejects them with exit 1. Deliberate
  compiler/setup failures return NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanSecondPolygonPointsPrefix$' -fuzztime=10s
  -parallel=2`: passed, **1,002,090 executions**. Bounds: 4096 input bytes,
  32 earlier records/first/second points, 256 comment bytes. Properties check
  deterministic results/errors, unchanged input, empty failures, inherited
  fields/spans/validation, ordered signed pairs, preflight and exact prefix/tail
  accounting. Ordinary checks execute all earlier/new fuzz seeds.

The first focused trial correctly rejected a surviving early-allocation fault:
allocation counts alone did not distinguish table sizes. Measuring allocated
bytes closed that gap; the focused and complete final campaigns kill the fault.
An earlier integrated attempt was rejected when removal of regenerable cache
entries interrupted an active build. After all builds stopped, the cache was
cleaned and the entire command rerun successfully. Neither incomplete attempt
is counted as a passed campaign or setup-error behavioral kill.

Independent read-only review found no Critical, Important or Minor issues.
It passed fresh targeted/full unit tests, full race tests, a separate bounded
fuzz run (383,727 executions), formatting/shell/diff checks, all 73 native rows
and new evidence hashes. The 159-path source/evidence snapshot preserves all
154 protected paths; only README, fixture provenance, mutation orchestration,
`.gitattributes` and the CI job timeout changed among earlier files. Original
EXE/runtime, C#/IL, helper, TOP and every prior probe/stdout remain intact.

Measured Git blobs: `b82c49ecba4b96ccfb9707728661741f0b176a2f` (reader), `a31de66d495311e0352d04d9998462904bf039f6` (reader tests),
`6b051290a59c71587fcad251fec694112b5fce55` (model), `d8cf93206e38bbb4e412fdb70b0457221c95ad38` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-second-polygon-points-mutation.json` and all earlier reports.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands including the new second-points scope. Exact pushed SHA and green CI
are verified in the chat handoff; README belongs to that delivered commit.
The first hosted attempt (`fb2947ffda342649eeeff2f3c1f40f319aaa9d1c`,
run `38029824715`) passed all ordinary platform checks and killed all 235
Gremlins mutants, but its Linux job hit the previous 15-minute timeout during
the explicit campaign. That incomplete run is not accepted. The CI job timeout
is now 25 minutes so the complete campaigns and trial controls can finish;
every check, mutation scope and acceptance gate remains enabled.
P04c7 stops before second color, including zero, and includes no later payload
or drawing/export implementation.

## Completed PBI: P04c8 — second plan Polygon color byte only

**Outcome/dependencies:** a separate immutable prefix extends P04c7 by exactly
one raw UInt8 second color after its vertices, including zero points. Implemented
on 2026-10-10. Preserve every earlier API, raw field, record,
span, limit, error precedence, copy contract and stopping position.

**References:** analysis map, Polygon.Read color at IL_00f0 and switch,
Polygon.Write and matching IL; unchanged pinned helper/api-drawings.top.
Second gray color/code 2 is at `[205,206)` by helper/write layout; confirm with
a fresh bounded original scalar probe preserving all earlier evidence.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanSecondPolygonPointsPrefixWithLimits` and `PolygonCountLimits`,
  adding no limit. Add distinct `SecondColorRaw` and `SecondColor` one-byte span;
  existing `ColorRaw` remains first color, and both point tables remain separate.
- Require one byte even for zero second points. Missing color returns
  `truncated` at `plan.elements[1].color`, the second-points end. Every failure
  returns an empty new result; inherited preflight/validation precedence stays.
- Preserve all 256 bytes without rejecting, masking or normalizing unknown
  colors. Native mapping is 1 black, 2 gray, 3 brown, 4 blue, 5 red, 7 orange,
  default green (including 6/unknown); rendering decisions stay separate.
- Stop immediately after second color. Exact prefixes and arbitrary tails
  succeed within inherited bounds; require no following marker, side mapping
  or trailer. P04c7 continues to succeed without second color, including zero.
- Test all color bytes with zero/one/multiple points in both tables, every
  required byte, missing/exact/arbitrary tails, variable earlier tables,
  inclusive/lower limits, inherited errors/stops, copied records/spans and
  pinned native position. Add bounded fuzzing and explicit raw-color/span/copy/
  separation/preflight/overread mutations.

**Non-goals:** following markers, element loops, XSections, plan completion,
side mapping/elements, full-file validation, geometry/rendering, CLI/exporters.

**Required checks/handoff:** all three existing scripts, >=95% coverage,
>=90% killed mutations with every survivor reviewed, independent review,
preserved evidence, README, commit/push, exact-SHA green CI and one next prompt.
Stop after P04c8; do not implement its successor in the same chat.

### Implemented second-color API and evidence (2026-10-10)

`internal/top.ReadV3PlanSecondPolygonColorPrefix` and
`ReadV3PlanSecondPolygonColorPrefixWithLimits` extend the unchanged P04c7
reader with `take(1, "plan.elements[1].color")`. Both reuse `PolygonCountLimits`;
no new bound, aggregate point budget or earlier reader change was introduced.
All inherited validation, preflight, error precedence and stopping positions
remain intact, including the whole-input ceiling on the unparsed tail.

`source.PlanSecondPolygonColorPrefix` retains every earlier accessor and adds
`SecondColorRaw` (UInt8). Its offsets embed `PlanSecondPolygonPointsPrefixOffsets`
and add `SecondColor`, a one-byte span. Existing `ColorRaw`/`Points` still mean
first Polygon color/points; `SecondPoints` remains separate and copied. All
consumed bytes and inherited collection accessors are copied; fields and offset
structs are values. No color names, pen objects, geometry or normalization were
added to the source model.

Missing second color, including after zero second points, returns an empty new
result and `truncated` at `plan.elements[1].color`, the second-points end.
All 256 raw bytes succeed unchanged. Exact prefixes and arbitrary tails succeed,
stopping immediately after that one byte without requiring a following marker,
side mapping or trailer. The zero-table/zero-first/second-point prefix consumes
52 bytes; P04c7 still succeeds at 51 without second color. Earlier APIs and
count-only large-count acceptance remain unchanged.

After the analysis map, inspected Polygon.Read/Write and matching IL. Native
ReadByte at IL_00f0 follows all vertices and the derived rectangle even for
zero points. Read selects 1 black, 2 gray, 3 brown, 4 blue, 5 red, 7 orange and
default green, including 6/unknown; Write emits those codes and default 6.
These rendering/write choices remain separate from raw source preservation.
The reviewed JKTZ 1..7 color restriction and aggregate budget were not adopted.

The unchanged pinned helper's second plan Polygon is gray. The separate
[original scalar probe](scripts/reference-plan-second-polygon-color-probe.cs)
confirms code 2 at `[205,206)`, consumed 206 and full-file tail 474 in pixel
modes 5/10. It uses original table/Mapping.Read methods and FileReader's
BinaryReader; it invokes no drawing reader, derived geometry, GUI or TOP writer.
Zero/one/three points in both tables cover all 256 colors with exact/arbitrary
tails, plus missing color in every combination. All **4,626 exact CRLF rows**
were checked independently against literal expectations; exit 0, empty stderr.
Environment, hashes and actual optional commands are in
[fixture provenance](internal/top/testdata/README.md). Original EXE/runtime,
C#/IL, helper, TOP and every prior probe/stdout remain byte-for-byte unchanged.
Scalar evidence does not establish full native Polygon/export compatibility.

### P04c8 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke. **100% statement
  coverage** in each implemented package. Tests cover all 256 colors across
  zero/one/multiple points in both tables, every required byte, missing/exact/
  arbitrary tails, all following byte values, separate inclusive point ceilings,
  lower inherited limits/errors/stops, variable earlier tables, immutable copies,
  distinct first/second fields/spans and native positions.
- `bash scripts/mutation-plan-second-polygon-color.sh`: **43/43 explicit faults
  killed**. Each compiled and failed a named test; errors/timeouts are rejected.
  Faults cover raw color masking/normalization/rejection, separate first/second
  fields/tables/spans, copied bytes, inherited limits/validation, point preflight,
  zero-point color omission, errors/empty failures, input mutation, tail
  consumption and following-marker overreads. All earlier scopes/gates stay.
- `bash scripts/mutation.sh`: complete integrated campaign passed with
  **239/239 Gremlins mutants killed** (four new reader faults and all 235 earlier
  faults), with no lived, uncovered, invalid, skipped or timed-out entries.
  The same run killed **467/467 explicit mutants** across all 14 scopes:
  all 424 earlier faults and the 43 new second-color faults above. Every explicit
  mutant compiled and failed a named behavioral assertion; no gates, exclusions
  or earlier scopes changed.
- `bash scripts/mutation-trial.sh`: passed. Ordinary weak tests pass, both CLI
  mutants survive and the shared gate rejects them with exit 1. Deliberate
  compiler/setup failures return NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanSecondPolygonColorPrefix$' -fuzztime=10s
  -parallel=2`: passed, **978,506 executions**. Bounds: 4096 input bytes,
  32 earlier records/first/second points, 256 comment bytes. Properties check
  deterministic results/errors, unchanged input, empty failures, inherited raw
  fields/spans/preflight, raw second color and exact one-byte prefix/tail
  accounting. Ordinary checks execute all earlier/new fuzz seeds.

Independent read-only review found no Critical, Important or Minor issues.
It independently passed fresh uncached unit and race tests for source/top,
a separate bounded fuzz run (178,333 executions), shell/diff checks, all 4,626
native rows/CRLF, new evidence hashes and all four measured Go blobs. The
166-path preservation snapshot confirms all 162 protected earlier paths
unchanged; only README, fixture provenance, mutation orchestration and
`.gitattributes` changed among earlier files. Full native geometry/rendering
and exporter acceptance remain outside this bounded scalar/source contract.

Measured Git blobs: `2ebd61c5605e97dada2f1c04bcbd3a38b27bb472` (reader), `dae2d943d1b2fdd5fba5c5fed0850a9b1dfb091d` (reader tests), `772f7b1a7e95e18ffefd0d8d8aa7536561cb5536` (model), `3d717fc15bf340b6614e579afc5dc951a36f85dc` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-second-polygon-color-mutation.json` and every earlier report.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands including the second-color scope. Exact pushed SHA and green CI are
verified in the chat handoff; README belongs to that delivered commit.
P04c8 stops before the following marker and adds no later drawing/export work.

## Completed PBI: P04c9 — following plan element marker byte only

**Outcome/dependencies:** a separate immutable prefix extends P04c8 by exactly
one raw UInt8 following marker after the second Polygon color. Implemented
on 2026-10-10. Preserve every earlier API, field, table, span, limit,
error precedence, copy contract and stopping position.

**References:** analysis map, Drawing.Read next ReadByte at IL_00c3 and
0/1/3/unknown dispatch, Polygon.Read/Write and matching IL; unchanged pinned
helper/api-drawings.top. The helper appends a third (brown) plan Polygon, placing
marker 1 at `[206,207)` by write layout; confirm with a fresh bounded original
scalar probe. Preserve all earlier evidence; invoke no drawing reader or TOP write.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanSecondPolygonColorPrefixWithLimits` and `PolygonCountLimits`,
  adding no limit. Add a distinct `FollowingMarkerRaw` byte and `FollowingMarker`
  one-byte span; retain `NextMarkerRaw`, both colors and both point tables.
- Missing marker returns `truncated` at `plan.elements[2].kind`, the second-color
  end. Every failure returns an empty new result; inherited validation and
  point/color/preflight error precedence remains unchanged.
- Preserve all 256 markers, including 0/1/3/unknown, without dispatching payload.
  Stop immediately after that byte, even for 0. Accept exact/arbitrary tails;
  require no payload, later marker, side mapping or trailer. P04c8 continues to
  succeed without this following byte, including zero points in both tables.
- Test all marker/color bytes, zero/one/multiple points in both tables,
  missing/exact/arbitrary tails, every required byte, inherited limits/errors/
  stops, variable earlier tables, copies/spans/separation and native position.
  Add bounded fuzzing and explicit raw-marker/span/copy/separation/preflight/
  overread mutations.

**Non-goals:** third payload/count/points/color, element loops, XSections,
plan completion, side mapping/elements, full-file validation, geometry/rendering,
CLI and exporters.

**Required checks/handoff:** all three existing scripts, >=95% coverage,
>=90% killed mutations with every survivor reviewed, independent review,
preserved evidence, README, commit/push, exact-SHA green CI and one next prompt.
Stop after P04c9; do not implement its successor in the same chat.

### Implemented following-marker API and evidence (2026-10-10)

`internal/top.ReadV3PlanFollowingMarkerPrefix` and
`ReadV3PlanFollowingMarkerPrefixWithLimits` reuse the unchanged P04c8 reader
and `PolygonCountLimits`, then `take(1, "plan.elements[2].kind")`. No new limit
or aggregate point budget was introduced. The source model
`source.PlanFollowingMarkerPrefix` retains every earlier accessor and adds
`FollowingMarkerRaw` and the one-byte `FollowingMarker` span. Its offsets embed
`PlanSecondPolygonColorPrefixOffsets`. First/second points, both colors and
`NextMarkerRaw` retain their distinct meanings and immutable copy contracts.

Every byte value, including 0/1/3/unknown, succeeds unchanged without dispatch.
Missing marker returns an empty new result and `truncated` at
`plan.elements[2].kind`, the second-color end. Exact prefixes and arbitrary tails
succeed within inherited bounds, stopping after one byte. Zero-table/zero-point
inputs consume 53 bytes; P04c8 still succeeds at 52 without a following marker.
Inherited input ceilings, validation/preflight/errors and all earlier stops stay.

After the analysis map, inspected Drawing.Read (RVA 0x10a30), Polygon.Read/Write
and matching IL. Drawing.Read's loop update calls ReadByte at IL_00c3, then
compares with zero at IL_00ca; marker 1 dispatches Polygon, 3 XSection, and
unknown nonzero markers take ReadBytes(0). Those native dispatch/termination
choices remain separate from this source prefix. The earlier JKTZ parser was
reviewed: its unknown-marker rejection, 1..7 color restriction and aggregate
point budget were not adopted. The unchanged pinned helper appends a third
brown Polygon after the second gray Polygon.

The separate [original scalar probe](scripts/reference-plan-following-marker-probe.cs)
confirms marker 1 at `[206,207)`, consumed 207 and full-file tail 473 in pixel
modes 5/10. It uses original table/Mapping.Read methods and FileReader's
BinaryReader, then first/second Polygon scalar fields and exactly one following
byte. No Drawing.Read/Polygon.Read, geometry, GUI or TOP writer is invoked.
Zero/one/three points in both tables cover all 256 markers with complementary
second colors, exact/arbitrary tails, and missing markers in all nine table
combinations. All **4,626 exact CRLF rows** match independent literal values
and positions; exit 0, empty stderr. Hashes/environment/actual optional commands
are in [fixture provenance](internal/top/testdata/README.md). Scalar evidence
does not establish complete native drawing/export compatibility.

### P04c9 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed format, vet, Staticcheck, coverage controls,
  uncached race tests, build and unchanged CLI smoke; **100% statement coverage**
  in each implemented package. New tests cover all marker/color bytes,
  zero/one/multiple points in both tables, missing/exact/arbitrary tails, every
  required byte, variable tables, copies/spans/separation, inclusive/lower
  inherited limits and errors/stops, and pinned native position.
- `bash scripts/mutation-plan-following-marker.sh`: **47/47 explicit faults
  killed**. Each compiled and failed a named behavioral test; setup/errors and
  timeouts fail closed. Raw marker masking/normalization/rejection, zero-marker
  side-mapping overread, payload requirements, both point tables/colors/markers,
  inherited spans/limits/preflight, immutable bytes, empty errors, offset/field
  faults and exact tail/one-byte stopping are exercised.
- `bash scripts/mutation.sh`: the complete rerun passed with **243/243 Gremlins
  mutants killed**, including four new reader faults and all 239 earlier faults;
  no lived, uncovered, invalid, skipped or timed-out entries. The same run killed
  **514/514 explicit faults** across all 15 scopes, including all 467 earlier
  faults and the 47 following-marker faults. Each compiled and failed a named
  behavioral test. An initial explicit-scope run stopped when disk space ran
  out and its gate rejected the run. The disposable repository Go build cache
  was cleared before rerunning the full campaign; no failure counted as a kill.
  Every earlier scope and gate remains enabled.
- `bash scripts/mutation-trial.sh`: passed; both weak-test CLI mutants survived
  and the gate rejected them with exit 1. Compiler/setup failures returned
  NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanFollowingMarkerPrefix$' -fuzztime=10s
  -parallel=2`: passed, **873,907 executions**. Bounds: 4096 input bytes,
  32 earlier records/first/second points, 256 comment bytes. Properties check
  determinism, unchanged input, empty failures, inherited fields/spans/errors,
  raw marker and exact one-byte prefix/tail accounting.

Independent read-only review found no Critical, Important or Minor issues.
It independently passed vet, Staticcheck, uncached race tests, 100% coverage
in each implemented package, a separate 5s bounded fuzz run (**525,208 executions**),
shell/diff checks, all 4,626 native CRLF rows and new evidence hashes. It checked
C#/IL, pinned helper order, all 47 uniquely targeted fail-closed faults, source
copies/separate fields and the preservation snapshot. All **167 protected earlier
paths** in the 171-file snapshot are unchanged; only README, fixture provenance,
mutation orchestration and `.gitattributes` changed among earlier files.
Full native drawing/export compatibility remains outside this scalar contract.

Measured Git blobs: `2024e3317425ed12c092ccaae7d2d43383cb8da6` (reader),
`d4ec893225dee80940a2ba592ee4c477dd584cc7` (reader tests),
`8bb33a8e95a068dbe01aad0340e41f84bc8852fd` (model),
`fe357961577f53f3e71470e1014bfbc359e46ec4` (model tests). Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-following-marker-mutation.json` and all earlier reports.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands, including the new scope. Exact pushed SHA and green CI are verified
in the chat handoff. P04c9 stops after the following marker byte.

## Completed PBI: P04c10 — third plan Polygon point count only

**Outcome/dependencies:** a separate immutable prefix extends P04c9 with only
the signed little-endian Int32 third Polygon point count when
`FollowingMarkerRaw` is 1. Implemented on 2026-10-10.

**References:** analysis map, Drawing.Read marker-1 dispatch, Polygon.Read's
first ReadInt32 and Polygon.Write/matching IL; unchanged pinned helper and
`api-drawings.top`. Third Polygon count 3 lies at `[207,211)` by write layout;
confirm with a fresh bounded original scalar probe, preserving earlier evidence.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanFollowingMarkerPrefixWithLimits` and `PolygonCountLimits`.
  Add distinct `ThirdPointCountRaw` and four-byte `ThirdPointCount` span;
  preserve all earlier APIs/fields/spans/copies/errors/limits/stops.
- Only marker 1 is supported by this new count reader. All other raw values,
  including 0/3/unknown, return `unsupported_element` at `plan.elements[2].kind`,
  its marker start, before requiring count bytes. P04c9 still accepts all 256.
- Require four count bytes; truncation points to `plan.elements[2].point_count`
  at the following-marker end. Reject negative counts with `negative_count`
  and values above the inclusive inherited `MaxPoints` with `resource_limit`.
  Counts 0 through the configured ceiling succeed without vertex/color bytes;
  no point preflight/allocation or aggregate point budget is introduced.
- Stop immediately after the count. Require no vertices, color, later marker,
  side mapping or trailer; accept arbitrary tails within inherited input bounds.
  Every failure returns an empty new result; inherited validation precedence stays.
- Test all markers, every count byte/truncation, signed/endian boundaries,
  zero/inclusive/lower limits, earlier errors/stops, variable positions,
  immutable copies/separation and native position. Add bounded fuzzing and
  explicit count/span/copy/dispatch/limit/overread mutations.

**Non-goals:** third vertices/color, element loops, XSections, plan completion,
side mapping/elements, full-file validation, geometry/rendering, CLI/exporters.

**Required checks/handoff:** all three scripts, >=95% coverage, >=90% killed
mutations with every survivor reviewed, independent review, preserved evidence,
README, commit/push, exact-SHA green CI and one next small prompt.
Stop after P04c10; do not implement its successor in the same chat.

### Implemented third-count API and evidence (2026-10-10)

`internal/top.ReadV3PlanThirdPolygonCountPrefix` and
`ReadV3PlanThirdPolygonCountPrefixWithLimits` reuse the unchanged P04c9 reader
and `PolygonCountLimits`, support only following marker 1, then read exactly
four bytes as a signed little-endian Int32. `source.PlanThirdPolygonCountPrefix`
retains every earlier accessor and embeds `PlanFollowingMarkerPrefixOffsets`,
adding distinct `ThirdPointCountRaw` and the four-byte `ThirdPointCount` span.
First/second counts, points, colors and markers retain their separate meanings.
Consumed bytes and inherited collections are copied; fields/spans are values.

Non-1 markers fail with `unsupported_element` at `plan.elements[2].kind`, the
marker start, before count truncation. Missing count bytes fail with `truncated`
at `plan.elements[2].point_count`, the following-marker end. Negative counts
fail with `negative_count`; counts above the inclusive caller `MaxPoints` fail
with `resource_limit` at the count start. All failures return an empty new model.
Counts zero through the ceiling require no vertices/color. No third point
preflight/allocation or aggregate budget is introduced. Exact/arbitrary tails
succeed within inherited input bounds. Zero-table/zero-point inputs consume 57
bytes; P04c9 still accepts all 256 markers and stops at 53 without the count.

After the analysis map, inspected Drawing.Read marker-1 dispatch at IL_0057,
Polygon.Read's initial ReadInt32 at IL_0007 and subsequent newarr at IL_000f,
and Polygon.Write's count Write(int32) at IL_001c. The allocation, point loop,
derived rectangle and color remain beyond this scalar reader. The reviewed
JKTZ aggregate point budget/preflight and color restriction were not adopted.
The pinned helper's third brown plan Polygon has three points.

The separate [original scalar probe](scripts/reference-plan-third-polygon-count-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader. It reads
first/second Polygon scalar fields, following marker 1, then only ReadInt32;
no Drawing.Read/Polygon.Read, third vertices/color, geometry, GUI or TOP write.
Full/exact streams in pixel modes 5/10 confirm count 3 at `[207,211)`, consumed
211 and full-file tail 469. Independent zero/one/three-point first/second tables
cover literal endian/signed counts, including negative and very large counts,
exact/FF 80 tails and all four partial-count lengths. All **231 exact CRLF rows**
were verified independently against literal expectations; native exit 0, empty
stderr. Native partial reads consume available bytes, while the Go prefix
reports the start and returns an empty result. Hashes and actual optional
commands are in [fixture provenance](internal/top/testdata/README.md).
This evidence establishes scalar decoding, not full native drawing acceptance.

### P04c10 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke; **100% statement
  coverage** in each implemented package. Tests cover every marker/count byte,
  signed/endian boundaries, count-only large values, zero/inclusive/lower limits,
  independent first/second point tables and colors, inherited errors/preflight/
  stops, variable earlier tables, immutable copies, separate fields/spans and
  native positions. Allocation checks compare zero/million third counts.
- `bash scripts/mutation-plan-third-polygon-count.sh`: **57/57 explicit faults
  killed**, each compiled and failed a named behavioral assertion. Covers raw
  signed counts/endian/narrowing, marker dispatch and error precedence, caller
  bounds, separate counts/colors/point tables/markers, spans/copies, inherited
  validation and empty errors, aggregate budgets, preflight, tail consumption,
  color overreads and changed input. Errors/timeouts fail closed.
- `bash scripts/mutation.sh`: the complete integrated run passed with **252/252
  Gremlins mutants killed** (nine new reader faults and all 243 earlier faults),
  with no lived, uncovered, invalid, skipped or timed-out entries. The same run
  killed **571/571 explicit faults** across all 16 scopes, including all 514
  earlier faults and the 57 new third-count faults. Each compiled and failed a
  named behavioral assertion. All earlier scopes and gates remain enabled.
- `bash scripts/mutation-trial.sh`: passed. Ordinary weak tests pass; both CLI
  faults survive and the gate rejects them with exit 1. Compiler/setup failures
  return NOT VIABLE exit 2, never behavioral kills.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanThirdPolygonCountPrefix$' -fuzztime=10s
  -parallel=2`: passed, **612,922 executions**. Bounds: 4096 input bytes,
  32 earlier records/first/second/third counts, 256 comment bytes. Properties
  check determinism, unchanged input, empty failures, inherited fields/spans/
  errors, marker dispatch, signed count validation and exact four-byte prefix/
  tail accounting. Ordinary checks run all earlier/new fuzz seeds.

Independent read-only review found no remaining Critical, Important or Minor
issues; its stale roadmap-label finding was fixed and rechecked. It independently
passed uncached race tests, vet, Staticcheck, 100% coverage per implemented
package, a separate 5s bounded fuzz run (**464,185 executions**), shell/diff
checks, C#/IL and pinned-helper order, all 231 native CRLF rows/new hashes and
all 57 uniquely targeted fail-closed mutations. The 128-entry preservation
snapshot covers all 122 earlier tracked paths plus six reference/runtime files:
123 entries are unchanged, with only README, fixture provenance, mutation
orchestration, `.gitattributes` and the Linux CI job deadline changed. All earlier
source and evidence remain byte-for-byte intact. Native full drawing/export compatibility is outside
this scalar contract.

Measured Git blobs: `e7223bb8ddc7c7020b9e7af4665d56d0e552de4c` (reader),
`5d92a8252cb2ca7f370daedbc20edcbf4a307fbb` (reader tests),
`c24c7af128af66cb096603a50cbb34fce6eb66fb` (model),
`da205e2286b1573a8ef8857cfbd9202c90d91cd0` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-third-polygon-count-mutation.json` and all earlier reports.
Three-platform CI repeats ordinary checks; Linux runs both complete mutation
commands including the new scope. The first hosted run completed all 252
Gremlins faults and all 514 earlier explicit faults, then its 25-minute job
deadline canceled the new scope's baseline before any new explicit faults.
This incomplete run was not accepted. Linux now has a bounded 40-minute job
deadline for the cumulative campaign; Windows/macOS retain 25 minutes. Mutation
thresholds, fail-closed gates and individual test timeouts are unchanged. The
follow-up deadline change was independently reviewed; all measured Go blobs
above and native evidence are unchanged. Exact pushed SHA and green CI are
verified in the chat handoff. P04c10 stops before third vertices/color.

## Completed PBI: P04c11 — third plan Polygon raw points only

**Outcome/dependencies:** a separate immutable prefix extends P04c10 with only
its counted ordered signed Int32 X/Y pairs. Implemented on 2026-10-10.

**References:** analysis map, Polygon.Read/Write and matching IL; unchanged
pinned helper and `api-drawings.top`. Third brown Polygon points occupy
`[211,235)`: `(-2000,-2000)`, `(-1500,-1000)`, `(-500,-1500)`. Confirm with a
fresh bounded original scalar probe, preserving all earlier evidence.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanThirdPolygonCountPrefixWithLimits` and `PolygonCountLimits`;
  add distinct copied `ThirdPoints` and `ThirdPoints` aggregate span with each
  `PolygonPointOffsets`. Keep first/second tables and all inherited APIs intact.
- Preflight `ThirdPointCountRaw * 8` against remaining bytes before allocating,
  using overflow-safe arithmetic. Insufficient bytes return `truncated` at
  `plan.elements[2].points`, the third-count end, with an empty new result.
- Preserve signed raw X/Y, order and per-coordinate spans; do not derive units,
  geometry, bounds, color or station identity. Use the existing point ceiling
  separately, with no aggregate budget. Zero points succeeds at the count end.
- Stop after the counted pairs, requiring no third color, later marker, side
  mapping or trailer. Accept exact/arbitrary tails within inherited bounds.
- Test zero/one/multiple points, signed/endian extremes, every required byte,
  preflight/limits/errors/stops, independent earlier tables, immutable copies,
  distinct spans and native positions. Add bounded fuzzing and explicit mutations.

**Non-goals:** third color, later markers/elements, loops, XSections, plan
completion, side drawing, full-file validation, geometry, CLI and exporters.

**Required checks/handoff:** `bash scripts/check.sh`, full
`bash scripts/mutation.sh`, `bash scripts/mutation-trial.sh` and
`bash scripts/mutation-dispatch-trial.sh`; >=95% coverage, >=90% Gremlins killed
with every survivor reviewed and 100% explicit faults killed. Register the new
campaign exactly once in the parallel scheduler, retaining all earlier scopes.
Obtain independent review, preserve evidence, update README, commit/push and
confirm all matrix jobs plus **CI complete** succeed for the exact pushed SHA.
Provide one next small prompt.
Stop after P04c11; do not implement its successor in the same chat.


### Implemented third-points API and evidence (2026-10-10)

`internal/top.ReadV3PlanThirdPolygonPointsPrefix` and its `WithLimits` variant
reuse unchanged P04c10 and `PolygonCountLimits`. The reader compares the validated
count with `(len(data)-start)/8` before allocation; neither count multiplication
nor an untrusted allocation precedes this overflow-safe preflight. Insufficient
bytes return an empty result and `truncated` at `plan.elements[2].points`, the
third-count end. Each point retains ordered signed little-endian Int32 X/Y and
record/X/Y spans. `source.PlanThirdPolygonPointsPrefix` copies `ThirdPoints` and
prefix bytes, embeds the earlier offsets and adds the distinct aggregate
`ThirdPoints` span. All first/second fields, tables, colors, markers, spans,
limits, error precedence and stops retain their contracts.

The point ceiling applies separately to each table. Zero points succeeds at the
count end (57 for empty earlier tables); exact prefixes need no third color.
Arbitrary tails remain unparsed within the inherited input ceiling. No units,
geometry, bounds, color interpretation or station identity are derived.

After the analysis map, inspected Polygon.Read's X/Y ReadInt32 calls at IL_0041
and IL_004e and assignments at IL_0064/IL_0079; Polygon.Write writes X/Y at
IL_003e/IL_005b. Native allocation, rectangle calculation and subsequent color
read are outside this prefix. The pinned helper's third brown plan Polygon is
`(-2000,-2000)`, `(-1500,-1000)`, `(-500,-1500)`. JKTZ's signed point decoding
was reviewed against C#/IL; its aggregate budget and color restriction were
not adopted.

The fresh [original scalar probe](scripts/reference-plan-third-polygon-points-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
first/second Polygon scalars, following marker/count and only third X/Y pairs.
It calls no Drawing.Read/Polygon.Read, third-color read, geometry, GUI or TOP
writer. Full/exact streams in pixel modes 5/10 confirm `[211,235)`, consumed 235
and full-file tail 445. Every partial third-point byte length throws
EndOfStreamException after consuming available bytes. Independent literal
streams cover zero/one/three points in all three tables, signed/endian extremes,
exact/FF 80 tails and every partial third-point length. All **397 exact CRLF
rows** match independent literal expectations; native exit 0, empty stderr.
Go reports the table start with an empty result on truncation. Hashes and actual
optional commands are in [fixture provenance](internal/top/testdata/README.md).
This is scalar evidence; full native drawing/export compatibility remains unvalidated.

### P04c11 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke; **100% statement
  coverage** in each implemented package. Tests cover signed/endian extremes,
  zero/one/multiple points, every required byte, separate inclusive/lower limits,
  preflight allocation, inherited errors/stops, variable records, all three
  independent point tables, immutable copies and distinct native spans.
- `bash scripts/mutation-plan-third-polygon-points.sh`: **56/56 explicit faults
  killed**, each compiled and failed a named behavioral test. Faults cover
  preflight/allocation, scalar endian/order/narrowing/normalization, all three
  tables/counts, inherited fields/spans/limits, copies, empty errors, coordinate
  spans, tail accounting and third-color overreads including zero points.
- `bash scripts/mutation.sh`: the full integrated run passed with **268/268
  Gremlins mutants killed** (16 new reader faults plus all 252 earlier faults),
  with no lived, uncovered, invalid, skipped or timed-out entries. The same run
  killed **627/627 explicit faults** across all 17 scopes: all 571 earlier
  faults plus the 56 new third-point faults. All reports have the expected
  counts and unique fault names, each compiled and failed a named behavioral
  test. Every earlier scope/gate remains enabled; the new campaign is registered
  exactly once in the existing `markers` group. Measured group totals are
  tables 93, first-polygon 198, second-polygon 141 and markers 195.
- `bash scripts/mutation-trial.sh`: passed. Both weak-test CLI faults survive
  and the gate rejects them; compiler/setup failures return NOT VIABLE exit 2.
- `bash scripts/mutation-dispatch-trial.sh`: passed, checking every scope
  exactly once through both default and matrix dispatch, failure propagation,
  invalid arguments and rejection of empty Gremlins reports.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanThirdPolygonPointsPrefix$' -fuzztime=10s
  -parallel=2`: passed, **868,912 executions**. Bounds: 4096 input bytes,
  32 earlier records/points per table, 256 comment bytes. Properties check
  determinism, unchanged input, empty failures, inherited fields/spans/errors,
  independent scalar decoding and exact point-prefix/tail accounting.

Independent read-only review found no remaining Critical, Important or Minor
issues; its stale roadmap-label finding was fixed and rechecked. It independently
passed source/top race tests, uncached 100% coverage in every implemented package,
the exact coverage gate, a separate 5s bounded fuzz run (**451,455 executions**),
dispatch controls, C#/IL/pinned-helper checks, all 397 native CRLF rows and seven
new/reference/helper hashes. It verified all 268 Gremlins entries and 56 uniquely
named focused faults are KILLED. The preservation snapshot has 135 entries:
131 unchanged, with only README, fixture provenance, mutation orchestration and
`.gitattributes` changed. All earlier source and native evidence are unchanged.

Measured Git blobs: `caa3bbcd10a91ddc5f6e5a8111831b94556c2d7b` (reader),
`356edf5756e1527c6a70baea8b75ef80f05b5b66` (reader tests),
`7dc85de71d4e2bb8895cc656fe28b8af93efe8ca` (model),
`96303c14ab0033d3e1a8c440cd2f3b175ab53f99` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-third-polygon-points-mutation.json` and all earlier reports.
Three-platform CI repeats ordinary checks; all five mutation groups and
**CI complete** must succeed for the exact pushed SHA. P04c11 stops before
third color, including zero points.

## Completed PBI: P04c12 — third plan Polygon raw color byte only

**Outcome/dependencies:** extend P04c11 with exactly one uninterpreted third
Polygon color byte after its counted points. Implementation authorized on 2026-10-10.

**References:** analysis map, Polygon.Read/Write and matching IL; unchanged
pinned helper and `api-drawings.top`. Third brown color is 3 at `[235,236)`.
Confirm with a fresh bounded original scalar probe, preserving earlier evidence.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanThirdPolygonPointsPrefixWithLimits` and `PolygonCountLimits`.
  Add distinct `ThirdColorRaw` and one-byte `ThirdColor` span; preserve every
  earlier API, table, raw field, span, copy, error, limit and stop.
- Read one byte after third points, including zero points. Preserve all 256 raw
  values without masking, normalization, validation or interpretation.
- Missing byte returns an empty new result and `truncated` at
  `plan.elements[2].color`, the third-point end. Exact prefixes and arbitrary
  tails succeed within inherited bounds, stopping immediately after that byte.
- Require no later marker, payload, side mapping or trailer. Test all color
  bytes, zero/one/multiple points in independent tables, every required byte,
  inherited preflight/limits/errors/stops, copies/spans and native position.
  Add bounded fuzzing and explicit mutations.

**Non-goals:** later markers/elements, loops, XSections, plan completion,
side drawing, full-file validation, geometry, CLI and exporters.

**Required checks/handoff:** `bash scripts/check.sh`, full
`bash scripts/mutation.sh`, `bash scripts/mutation-trial.sh` and
`bash scripts/mutation-dispatch-trial.sh`; >=95% coverage, >=90% Gremlins killed
with every survivor reviewed and 100% explicit faults killed. Register the new
campaign exactly once in an existing parallel scheduler group, retaining all
earlier scopes. Obtain independent review, preserve evidence, update README and
group scopes/measured counts, commit/push, confirm every platform check, every
mutation group and **CI complete** for the exact pushed SHA. Provide the next
small prompt. Stop after P04c12.

### P04c12 implementation and native evidence

`ReadV3PlanThirdPolygonColorPrefix` and its `WithLimits` variant reuse the
P04c11 point reader, then take exactly one raw byte, including after zero points.
The copied `source.PlanThirdPolygonColorPrefix` adds `ThirdColorRaw` and a
one-byte `ThirdColor` span, retaining all earlier accessors and offsets.
Missing color returns an empty new result with `truncated` at
`plan.elements[2].color`, the third-point end. Exact prefixes and arbitrary
tails within the inherited input bound stop after that byte. All 256 values
remain uninterpreted; no later marker, drawing completion or CLI change.

After the analysis map, inspected Polygon.Read/Write and matching IL:
`ReadByte()` at IL_00f0 follows all counted X/Y pairs, with no zero-count
exception. Its pen selection is a later native interpretation; the Go reader
preserves the byte. JKTZ's color restriction to 1..7 was reviewed against this
source and is not adopted. Native brown writes byte 3.

The fresh [original scalar probe](scripts/reference-plan-third-polygon-color-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
first/second Polygon scalars, following marker/count, third X/Y and one color.
It calls no Drawing.Read/Polygon.Read, later marker, geometry, GUI or TOP writer.
Full/exact streams in pixel modes 5/10 confirm color 3 at `[235,236)`, consumed
236 and full-file tail 444. Missing color throws EndOfStreamException at 235.
Independent literal streams cover zero/one/three points in all three tables,
signed/endian extremes, every color 0..255, exact/FF 80 tails and missing color
including zero points. All **13,860 exact CRLF rows** match independently
constructed literal expectations byte-for-byte; native exit 0, empty stderr.
Hashes and actual optional commands are in [fixture provenance](internal/top/testdata/README.md).
This is scalar evidence; full native drawing/export compatibility remains unvalidated.

### P04c12 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

The first full campaign stopped when compilation ran out of disk space;
the explicit gate correctly rejected that build failure as not killed. Its log
is retained at `build/p04c12-mutation-disk-full.log`. Cleared only this project's
rebuildable 12 GiB Go build cache, preserving source, probes, fixtures and evidence,
then restarted the full campaign without changing code or gates.

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke; **100% statement
  coverage** in every implemented package. Tests cover all 256 colors,
  independent 0/1/3-point tables, every required byte, separate inclusive/lower
  limits, inherited preflight/errors/stops, variable records, copies and native spans.
- `bash scripts/mutation-plan-third-polygon-color.sh`: **55/55 explicit faults
  killed**, each compiled and failed a named behavioral test. Faults cover raw
  preservation, normalization/rejection, zero-count skipping, limits/preflight,
  empty errors, all three tables/counts/colors, inherited spans/mappings/markers,
  byte copies, offsets, tail accounting and later-marker overreads.
- Full `bash scripts/mutation.sh`: the complete rerun passed with **272/272
  Gremlins mutants killed** (all 268 earlier faults plus four new reader faults),
  with no lived, uncovered, invalid, skipped or timed-out entries. The same run
  killed **682/682 explicit faults** across all 18 scopes: all 627 earlier
  faults plus 55 new color faults. Every report was freshly generated by this
  successful full run, with the expected count and unique names; each fault
  compiled and failed a named behavioral test. The new campaign is registered
  exactly once in the existing `markers` group; every earlier scope and gate
  remains enabled. Measured group totals: tables 93, first-polygon 198,
  second-polygon 141 and markers 250.
- `bash scripts/mutation-trial.sh`: passed; both weak-test CLI faults survive
  and are rejected by the gate; compiler/setup failures return NOT VIABLE exit 2.
- `bash scripts/mutation-dispatch-trial.sh`: passed, verifying every scope
  exactly once through default/matrix dispatch, failure propagation, invalid
  arguments and rejection of empty Gremlins reports.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanThirdPolygonColorPrefix$' -fuzztime=10s
  -parallel=2`: passed, **841,800 executions**. Bounds: 4096 input bytes,
  32 earlier records/points per table, 256 comment bytes. Properties check
  determinism, unchanged input, empty failures, inherited fields/spans/errors,
  raw color and exact one-byte prefix/tail accounting.

Independent read-only review found no remaining Critical, Important or Minor
issues after correcting the current scheduler table. It independently passed
uncached source/top race tests, 100% source/top coverage and the exact gate,
a separate 5s bounded fuzz run (**361,920 executions**), all 13,860 native CRLF
rows, pinned-helper/C#/IL checks, all new evidence and four Go blob hashes,
55 uniquely named focused faults and 272/272 Gremlins entries. The preservation
snapshot `build/p04c12-preservation.json` has 142 entries: 138 unchanged, with
only README, fixture provenance, mutation orchestration and `.gitattributes`
changed. All earlier source and native evidence remain byte-for-byte unchanged.

Measured Git blobs: `b505f4e4cc608abc0cd5ccb4365b8ac2a2df2b4d` (reader),
`376fa78f832ff3f0ebfcba2e76870d13814b1ee7` (reader tests),
`657ab796514b04a2689a31531352e1e151212f2e` (model),
`dcb3d0a510ce97e842e0041fde3ce864ba6c3483` (model tests).

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-third-polygon-color-mutation.json` and all earlier reports.
Three-platform CI, all five mutation groups, the plan and **CI complete**
must succeed for the exact pushed SHA. P04c12 stops after third color.

## Completed PBI: P04c13 — next plan-element marker byte only

**Outcome/dependencies:** extend P04c12 with exactly one uninterpreted next
plan-element marker at `plan.elements[3].kind`. Implementation authorized on 2026-10-10.

**References:** analysis map, Drawing.Read/Write and matching IL; unchanged
pinned helper and `api-drawings.top`. Inspect the next marker at `[236,237)`
and confirm it with a fresh bounded original scalar probe.

**Bounded acceptance contract:**

- Reuse `ReadV3PlanThirdPolygonColorPrefixWithLimits` and `PolygonCountLimits`.
  Add a distinct raw marker accessor and one-byte span; preserve every earlier
  API, table, raw field, span, copy, error, limit and stop.
- Preserve all 256 marker values, including zero, without dispatch or validation.
- Missing byte returns an empty new result and `truncated` at
  `plan.elements[3].kind`, the third-color end. Exact prefixes and arbitrary
  tails within inherited bounds stop immediately after that byte.
- Require no payload, later marker, side mapping or trailer. Test every marker,
  every required byte, independent tables/colors, inherited bounds/errors/stops,
  copies/spans and native position. Add bounded fuzzing and explicit mutations.

**Non-goals:** payloads, loops, XSections, plan completion, side drawing,
full-file validation, geometry, CLI and exporters.

**Required checks/handoff:** `bash scripts/check.sh`, full
`bash scripts/mutation.sh`, `bash scripts/mutation-trial.sh` and
`bash scripts/mutation-dispatch-trial.sh`; >=95% coverage, >=90% Gremlins killed
with every survivor reviewed and 100% explicit faults killed. Register the new
campaign exactly once in an existing parallel scheduler group, retaining all
earlier scopes. Obtain independent review, preserve evidence, update README and
group scopes/measured counts, commit/push, confirm every platform check, every
mutation group and **CI complete** for the exact pushed SHA. Provide the next
small prompt. Stop after P04c13.

### P04c13 implementation and native evidence

`ReadV3PlanThirdNextMarkerPrefix` and its `WithLimits` variant reuse unchanged
P04c12 and `PolygonCountLimits`, then read exactly one byte at
`plan.elements[3].kind`. `source.PlanThirdNextMarkerPrefix` adds
`ThirdNextMarkerRaw()` and a distinct `ThirdNextMarker` span, retaining all
inherited accessors, offsets and immutable copies. Every value 0..255 succeeds
without interpretation. Missing marker returns an empty result and `truncated`
at the third-color end. Exact prefixes and arbitrary tails stop after one byte.

After the analysis map, inspected Drawing.Read/Write and matching IL:
ReadByte at IL_00c3 follows Element.Read; dispatch and the zero-marker loop test
follow separately. Drawing.Write emits a terminal zero byte. JKTZ's byte read
was checked against this evidence; its unknown-kind rejection and full drawing
loop are outside this prefix and were not adopted.

The fresh [original scalar probe](scripts/reference-plan-third-next-marker-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
three Polygon scalar tables/colors and exactly one next marker. No Drawing.Read,
Polygon.Read, dispatch, payload, geometry, GUI or TOP writes. Full/exact streams
in pixel modes 5/10 confirm marker 1 at `[236,237)`, consumed 237 and full tail
443. Missing marker throws EndOfStreamException at 236. Literal streams cover
independent 0/1/3-point tables, signed/endian extremes, all 256 markers, exact
and FF 80 tails, and missing marker including empty tables. All **13,860 exact
CRLF rows** match independently constructed expectations; exit 0, empty stderr.
Hashes and actual optional commands are in [fixture provenance](internal/top/testdata/README.md).
This confirms scalar reading; full native drawing/export compatibility remains unvalidated.

### P04c13 verification and mutation scope

Go 1.26.3 darwin/arm64, Staticcheck v0.8.1, Gremlins v0.6.0:

- `bash scripts/check.sh`: passed formatting, vet, Staticcheck, exact coverage
  controls, uncached race tests, build and unchanged CLI smoke; **100% statement
  coverage** in every implemented package.
- `bash scripts/mutation-trial.sh` and `bash scripts/mutation-dispatch-trial.sh`:
  passed negative/build/setup controls and exact-once scheduling through default
  and matrix dispatch, failure propagation and empty-report rejection.
- `GOCACHE="$PWD/.cache/go-build" GOTOOLCHAIN=local go test ./internal/top
  -run '^$' -fuzz '^FuzzReadV3PlanThirdNextMarkerPrefix$' -fuzztime=10s
  -parallel=2`: passed, **879,201 executions**. Bounds: 4096 input bytes,
  32 earlier records/points per table, 256 comment bytes. Checks determinism,
  unchanged input, empty failures, inherited errors/fields/spans and raw marker,
  exact one-byte stop and prefix/tail accounting.

Independent read-only review found no outstanding Critical, Important or Minor
issues. A source-constructor test targeted the previous model; corrected it
and the reviewer independently retested it successfully. The reviewer passed
uncached source/top race tests, 100% source/top coverage, a separate 5s fuzz run
(**327,046 executions**), dispatch controls, C#/IL checks, hashes and all 13,860
native CRLF rows. The preservation snapshot `build/p04c13-preservation.json`
has 144 entries: 140 unchanged, with only README, fixture provenance, mutation
orchestration and `.gitattributes` changed. All earlier source and native evidence
are byte-for-byte unchanged.

Gremlins killed **276/276 mutants** on the final constructor test, including
a complete rerun after the cache failure and the separate post-review repeat. No lived, uncovered, invalid,
skipped or timed-out entries; the strict report gate passed. Full `bash scripts/mutation.sh` rerun passed with **737/737 explicit faults**
across all 19 scopes: 682 earlier plus **55/55 new marker faults**. Every report
was regenerated after that run's Gremlins result, has its expected count and
unique names, and records only KILLED entries after successful compilation and
a named behavioral failure. The new campaign is registered exactly once in
`markers`; all earlier scopes and gates remain enabled. Measured group totals:
tables 93, first-polygon 198, second-polygon 141 and markers 305.

The first full campaign was rejected after a live cache cleanup removed files
needed by an active compilation. Its failure log is retained in
`build/p04c13-mutation-cache-removal-failure.log`; compilation failures are not
counted as killed mutants. Once the failed process exited, cleared only the
project's rebuildable Go cache and restarted the complete campaign without
changing reader/model code or gates. No source, fixture, probe or report was
removed.
Measured Git blobs:

`336cf2eb247f6bdfa5cf25d29c230bdc01d9a947` (internal/top/plan_third_next_marker.go)

`564e4cada10952efaf4be7f781c7ab8da128ce7b` (internal/top/plan_third_next_marker_test.go)

`c6e55460956468ab2ec4c803f2d9b57e08a51b3e` (internal/source/plan_third_next_marker_prefix.go)

`8fa2af5e429a365d9b8a1c078caa6f4460d39afc` (internal/source/plan_third_next_marker_prefix_test.go)

Reports regenerate in `coverage.out`, `mutation.json`,
`build/plan-third-next-marker-mutation.json` and all earlier reports.
P04c13 stops after the next marker; no payload, loop, side drawing or CLI changes.

## Next ready PBI: P04c14 — fourth plan Polygon signed Int32 point count only

Requires a new implementation request. Reuse P04c13 and `PolygonCountLimits`;
only when `ThirdNextMarkerRaw() == 1`, read one signed little-endian Int32 at
`plan.elements[3].point_count`. Preserve raw count and distinct span, reject
other marker values before payload, negative counts and exceeded per-table
limits with established error precedence. Stop immediately after count,
including zero, before fourth points/color or later markers. Confirm native
position `[237,241)` with a fresh bounded scalar probe; retain every earlier
contract and evidence. Add tests, bounded fuzzing and explicit mutations;
run ordinary/full mutation/control checks, obtain independent review, update
README, commit/push and verify every CI job for the exact SHA. Stop after P04c14.

## Open issues and deferred work

- The TOP readers validate only the v3 trip/measurement/reference/overview/plan-mapping/first-marker/Polygon-count/Polygon-points/Polygon-color/next-marker/second-Polygon-count/second-Polygon-points/second-Polygon-color/following-marker/third-Polygon-count/third-Polygon-points/third-Polygon-color/third-next-marker prefixes.
  Later drawing payloads and side mappings, complete-file validation and native exporters remain
  unimplemented; the CLI still accepts only help/version.
- Native measurement arithmetic/export formatting fidelity remains unproven in Go.
- Gremlins is accepted only for the measured small scope; expand and reassess
  operator coverage as native logic arrives. CI rejects invalid/uncovered/time-out
  mutants rather than silently excluding them.
- The full P01 capability/fixture matrix, older TOP versions, corpus runs,
  release packaging and optional R01–R07 work remain deferred.
- Future full compatibility claims still require native evidence; the P02/P03a/P03b/P03c1/P03c2/P04a/P04b/P04c1/P04c2/P04c3/P04c4/P04c5/P04c6/P04c7/P04c8/P04c9/P04c10/P04c11/P04c12/P04c13
  checks validate only implemented behavior and their recorded reference cases.

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
[Gremlins v0.6.0](https://github.com/go-gremlins/gremlins/releases/tag/v0.6.0), and
[Gherkin reference](https://cucumber.io/docs/gherkin/reference/).
