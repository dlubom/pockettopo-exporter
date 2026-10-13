# Development review and real-sample verification

Review date: 2026-10-13. Implementation revision:
`9a48609b5a0c10da2da1e98d5a37b323ca3c1ff9` on main, matching remote main at review.
The starting tree was clean. The baseline review added documentation and ignored
local acceptance artifacts. The subsequent user-requested Walls averaging and
SRV/SVX layout correction is described separately below. Its delivery uses a
draft pull request; the baseline CI result does not validate that follow-up.

## Assessment

The current implementation is suitable for user acceptance testing and a bounded
future release-preparation increment. Version 1.0.0 was deferred by the user on
2026-10-13. **No confirmed actionable P1/P2 defect was
found in the inspected scope.** This is not exhaustive proof of every native
option or destination application. Release preparation and manual acceptance
remain necessary; the current focus is [SRV/SVX acceptance](PLAN.md#current-focus--srvsvx-user-acceptance).

Confirmed release gaps, distinct from regressions against the current dev profile:

1. CLI `--version`, inspection and exporter `tool_version`, and distribution
   `cli_version` still report `dev`. `internal/cli/run.go`, exporter report
   constructors and `scripts/distribution.py` need one consistent release identity.
2. CI uploads revision-bound validation ZIPs, but there is no tagged-release
   publication path. A release must bind version, tag, exact passing SHA and
   all three validated assets, without rebuilding an untested binary.
3. No tracked LICENSE, COPYING, NOTICE or CHANGELOG exists. The owner must choose
   the project distribution license and applicable attribution; this review does
   not make that choice or determine legal entitlement.
4. README instructions are mainly for developers running repository fixtures.
   Downloaded binaries need installation, checksum and practical export guidance.

## Verification actually performed

| Check | Result / evidence |
| --- | --- |
| Exact-SHA hosted CI | [Run 38393987305](https://github.com/dlubom/pockettopo-exporter/actions/runs/38393987305), all 30 jobs successful, including `CI complete`, all mutation groups, three platforms, corpus and consumers |
| Fresh bounded ordinary gates | `GOMAXPROCS=2 GOFLAGS=-p=1 bash scripts/check.sh` passed: validation controls, evidence, formatting, vet, Staticcheck, uncached race tests, coverage, build and smoke |
| Statement coverage | 98.2% overall; exact-count gate passed |
| Preserved evidence | 1462 earlier, 271 P14, 4 P15 and 8 P16 identities verified |
| Fresh complete pinned corpus | 262 inputs, 25 native-format profiles each, every profile run twice; 6504 successful and 46 explicit failed outcomes, matching the frozen baseline |
| Independent corpus text oracle | All 262 text exports matched original-program captures |
| Walls and Survex supplemental corpus | Each: 251 successful packages accepted by `cavern` 1.4.22; 11 invalid-input failures left no output; source/output report hashes checked |
| Pinned independent consumer gates | `validate-walls.py` and `validate-survex.py` passed analytic coordinates, repeats/splays/equations and malformed-input controls |
| Fresh macOS arm64 distribution | Clean exact-SHA build, embedded VCS identity, ZIP/checksum readback, empty-PATH smoke, six native goldens, SRV/SVX goldens, no-replace and malformed-input checks passed |
| Atschiloch sample | Nine complete packages, repeat-identical artifact/report bytes, valid source/output hashes, no-replace behavior; detailed checks below |
| Independent focused code review | No confirmed P1/P2; four additional literal zero/1 mm splay cases accepted by Survex with matching provenance |

The 46 native-profile failures are the existing 28 negative Toporobot-index
outcomes and 18 context/closure limits (12 work-limit and 6 iteration-limit
outcomes). These are not successful exports. The supplemental conversion checks
do not redefine the six-format frozen corpus baseline. The independent Walls
reader reports the known unsupported `Grid=0` option and uses its standalone zero
default; it is not a Walls GUI test. The real sample has no fixed references, so
the consumer explicitly anchors the first station at the origin.

Heavy mutations were verified through the exact-SHA hosted run, not rerun
locally. No full original-program oracle regeneration was performed.

## Acceptance follow-up: Walls repeated legs and readable output

The baseline SRV retained each repeat as a separate vector. Unlike Survex's
adjacent-repeat averaging, Walls admits these as redundant network edges. Its
[compiler](https://github.com/wallscavesurvey/walls/blob/master/Walls32/compile.cpp)
stores duplicate name pairs and its network code computes closure rank from
edges, stations and connected components. The 48 real triplets would therefore
contribute 96 additional closures. This is a conversion design gap identified
during user acceptance, not a successful native Walls GUI check.

The local correction exports one float64 ENU mean per consecutive positive
named repeated/reversed leg in Walls. It keeps every raw observation and links
all members to the output line. Splays and zero equations remain separate.
Survex keeps source readings because its own adjacent same-direction averaging
is [documented](https://survex.com/docs/manual/svxhowto.htm#repeated-readings) and
was verified on all 48 real triplets. Both formats retain source comments and
omit generated measurement labels/redundant declination directives.

Fresh full ordinary gates passed with 98.2% statement coverage. Both independent
consumer validators passed. The updated Walls corpus produced 251 accepted
packages and the same 11 invalid-input failures across 262 inputs. All original
frozen identities plus four final conversion expectations verify. The independent
review identified a half-micrometre rounding boundary that could create a zero
equation; the failing regression was reproduced and the inclusive guard fixed.

The new Atschiloch SRV has 48 averaged legs plus 239 splays, with all 383 raw
observations unchanged in its report. Every average was independently checked
against source TOP bytes (maximum vector-component rounding difference below
0.5 micrometres). All 49 named station coordinates decoded by the consumer match
the unchanged SVX result exactly at dump precision. Both compile with zero loops.

Current evidence and the replacement sample ZIP are in `build/walls-averaged/`.
The original all-format baseline package remains unchanged. All 14 newly added
faults were separately compiled and killed by named tests. The full 54/46-fault
Walls/Survex campaigns remain required in hosted CI for the final pushed commit;
its GitHub checks and delivery response provide that revision's result.
GUI acceptance remains pending after the previously reported Computer/Wine
limitations. A successful Survex SRV reader run does not establish native Walls
GUI compatibility. Version 1.0.0 is deferred at the user's request.

## Real acceptance sample

Source: `158-Atschiloch/2019-08-02/2019-08-02achiloch.top` in
[dlubom/test2 at 4c00c008a8dd441d70f9c62aa376c20b5914c7e0](https://github.com/dlubom/test2/tree/4c00c008a8dd441d70f9c62aa376c20b5914c7e0).

- 39,465 original bytes; SHA-256
  `5518ed096d5bfae36e91c8e5f52299b9535b3d9c50ae0b4581520f6939a2d275`.
- Complete supported TOP v3, 3 stored trips, 383 measurement rows, no references.
- 48 named leg groups, each containing exactly three source observations;
  239 splays. All 383 source rows remain represented.
- Plan: 312 ordered Polygons and 2448 points. Side: 119 Polygons and 1193 points.
  This sample has no XSections, so it does not demonstrate that separate feature.
- Isolated input, no template-directory context and no loop closure. Graphics
  DXF uses scale 500 and mask 510 (shots, rays, labels and separate layers/colors;
  no GEO/grid). Toporobot uses the explicit export clock `2026-10-13T12:00:00`;
  this is an output timestamp, not a claimed survey date.

The complete local package is
[`build/release-review-9a48609/sample-atschiloch.zip`](../build/release-review-9a48609/sample-atschiloch.zip).
It includes the original TOP, all eight formats with reports (both 2D DXF views),
the macOS arm64 test binary, reproducible export commands, checksums, source and
triple-group evidence, consumer output and a diagnostic PNG. It is a local test
package, not a tagged 1.0.0 distribution. Build artifacts are ignored by Git.

### Verified triple-measurement example

For `1.0 -> 1.1`, zero-based source rows **1, 2, 3** occupy byte ranges
`[65,85)`, `[85,105)`, `[105,125)`:

| Observation | Distance m | Bearing degrees | Inclination degrees |
| --- | --- | --- | --- |
| 1 | 6.605 | 203.829345703125 | 68.97216796875 |
| 2 | 6.632 | 205.1092529296875 | 68.79638671875 |
| 3 | 6.609 | 205.2960205078125 | 68.8128662109375 |

Text preserves the source rows at native display precision. SRV/SVX retain all
three observations with exact binary-angle decimal values. Native grouped
Therion produces `1.0 1.1 204.76 68.86 6.615 >`; its report links the output to
all three source spans. Grouping follows native fixed-point vector averaging,
not independent arithmetic means of the angles. Every triple group was checked
for three-member Therion lineage. Every SRV/SVX row's distance, bearing and
inclination was independently unpacked from TOP bytes and compared with the
actual exported line and report.

For all 431 drawing elements, independent little-endian reads checked the raw
Polygon record boundary, color, point count and complete ordered point list.
Every one of the 3641 coordinates matched both DXF and Therion with the documented
scale/Y-axis conversion. The preview was rendered from actual exported DXF
entities and visually inspected; it does not establish a CAD application's
import behavior and is not implementation of optional PNG/SVG exporters.

Stored trip dates include 2005, although the source filename contains 2019.
Native exports preserve stored dates. Neither value establishes the field date;
the acceptance package does not rewrite or reconcile them.

### Manual acceptance still pending

1. Open the original TOP and compare the two DXF views in the intended drawing
   application: stroke shape, colors, layers, scale and orientation. The 2D DXF
   header retains millimetres and coordinates are divided by scale 500; do not
   interpret them as unscaled survey metres. Check the supplied preview's scale.
2. Inspect the three `1.0 -> 1.1` rows in text/SRV/SVX and the grouped value in
   Therion/VisualTopo/Toporobot. Record application versions and diagnostics.
3. Import 3D DXF separately if it is part of the intended workflow. Native hulls
   are not a guarantee of a watertight cave surface. Therion `survey.txt` is
   PocketTopo interchange, not a standalone Therion project or TH2 drawing.

An initially considered Kitzgraben sample (`408-Kitzgraben/20k01/200801Kitzgraben.top`)
contains nine rows without a source station. Its six native formats export, but
Walls and Survex correctly reject row 0 (`raw 80000000`) without partial output.
The failed attempts remain in local evidence. This illustrates strict validation
of real data, not a repaired or silently filtered source.

## Review scope and local evidence

The independent reviewer examined CLI dispatch, input/context loading,
no-replace publication/cleanup, complete Walls and Survex writers/tests, source
station identity, document/table parsers, native text and shared grouping.
Native Trip/ID C# and IL were checked. This review also inspected native Polygon
read/write and DXF coordinate rules while validating the real sample.

Every C#/IL path of the six older exporters, actual Walls/CAD/VisualTopo/Toporobot
GUI import, signing/notarization and legal entitlement were not independently
established. Existing native goldens and current tests cover documented bounded
profiles; they do not remove these limits.

Detailed artifacts under `build/release-review-9a48609/`:
`ci.json`, `check.log`, `corpus-report.json`, `conversion-corpus.json`,
`walls-consumer.json`, `survex-consumer.json`, `distribution/`, `audit-sample.py`,
`sample-audit.log` and `sample-atschiloch/evidence/validation.json`.
The independent review is in `build/release-review-agent/review.txt`, with
`edge-case-summary.json` and its reproduction files. All original corpus bytes
and frozen repository evidence were preserved.

No commit, push, tag or release was created by this review.
