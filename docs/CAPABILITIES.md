# Native capability matrix

P14 closes the inventory for **standard PocketTopo 1.372**, under the bounded
profiles below. It does not claim every native setting, input version, host code
page or destination application. The original EXE hash is
`048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371`.
The [source contracts](SPEC.md), [fixture provenance](../internal/top/testdata/README.md)
and frozen `validation/` manifests are the evidence behind this table.

## Exporters and options

| Format / CLI | Verified profile and selection | Independent native evidence / executable checks |
| --- | --- | --- |
| Text / `text` | Isolated input, metres, circle 360; raw table order, original values, UTF-8/CRLF | `Exporter.WriteText`, `Survey.WriteText`, `Station.WriteText`, `Trip.WriteTxt`; `TestTextNativeGoldens`: 7 complete, 1 partial failure recorded separately; P14 real-corpus text captures |
| Therion interchange / `therion` | Metres/360; FIX/TRIP/DATA, both drawings, colored Polylines and embedded XSections; isolated/context, raw/closed | `Therion.WriteExport`; `TestTherionNativeGoldens`: 89 complete; 5 native geometry failures in transcript |
| Graphics DXF / `dxf` | Plan or side, scale 1..100000, every native mask bit, GEO, metres/feet grid; isolated/context, raw/closed | `GraphicsForm.Form_Closing`, `DXFWriter.WriteDXF`; `TestDXFNativeGoldens`: 261 complete plus 7 partial failures; independent native coordinate/structure checks and frozen previews |
| VisualTopo 5.02 / `visualtopo` | Metres, degrees/grads, native LRUD selection, Windows-1252/CRLF; isolated/context, raw/closed | `VisualTopoExp.WriteExport/WriteList`; `TestVisualTopoNativeGoldens`: 194 complete, 10 native geometry failures; 4105 Single-format observations and exhaustive 65536-unit encoding/trim maps |
| Toporobot / `toporobot` | Degrees/grads, code/trip 1..100000, auto/manual declination flags, include/exclude splays, vertical sections, existing-file merge, export clock; MacEncoding/CRLF; isolated/context, raw/closed | `ToporobotForm.Form_Closing`, `ToporobotExp`; `TestToporobotNativeGoldens`: 325 complete, 54 failures; 21 reader/writer cases (15 byte matches, 4 native failures, 2 finite-profile rejections), 66 numeric observations |
| 3D DXF / `dxf3d` | Native sorted centerline/hull output, millimetres, 16 mm quantization, no graphics settings; isolated/context, raw/closed | `ThreeD.WriteDXF/WriteDXF3D/WriteStretch/FindFace`, `Survey.GetStations`, `ID.CompareTo`; 135 byte outputs (77 complete files, 57 stretches, 1 sequence), 24 sorts, 7 failures; 24 additional face identity/normal outcomes |

Graphics masks expose `SIDE=1`, `ENS=2`, `ENX=4`, `ENL=8`, `ENA=16`,
`SEPS=32`, `SEPX=64`, `SEPL=128`, `SEPC=256`, `GEO=512`, `ENG=1024`,
`SEPG=2048`. `--view` selects SIDE independently of the mask. These represent
shots, rays, labels, grid, their separate layers, colors, georeferenced output
and all/template geometry respectively. Tests include individual bits, useful
combinations, both views, scale boundaries and feet grids. No exhaustive Cartesian
product of all options and all corpus files is claimed.

## Explicit differences and unsupported settings

- Text remains the P05 metres/360 isolated profile. Native text feet and circle
  400 are inventoried but unimplemented. Text elides geometry Update; when native
  Update fails, Go can still publish the raw tables, as reported in P05.
- Native GraphicsForm initially selects both views, scale 500, shots/rays,
  no labels, and separate layer flags (plan mask 2278, side mask 2279). CLI
  defaults select plan and mask 14. Native GEO radio forces scale 1; CLI scale
  and GEO are independent. Two exports select both views; native GUI P/S filename
  suffix customization is outside CLI packaging.
- Context is explicit and sorted by portable UTF-8 names. No neighboring files
  are discovered in isolated mode. Templates supply raw tables; current drawing
  comes from the selected file. Native all-files closure `CloseLoops(true,true)`
  is unimplemented; CLI closure is `CloseLoops(true,false)` with fixed templates.
- VisualTopo pins the captured Windows-1252 `Encoding.Default`; arbitrary machine
  code pages are unsupported. Toporobot pins MacEncoding, output basename
  `survey` and the supplied/default clock. `--add-to` reads the existing source
  without modifying it. Therion fixes metres/360 in its native writer.
- Only complete supported TOP v3 current documents are exported. Unknown drawing
  markers, unsupported tails, malformed data and bounded work/resource failures
  stay explicit. Prefix APIs and template-table acceptance retain their earlier
  contracts. Native caught partial files are failure evidence, never byte goldens.
- Source offsets/raw IDs/flags/trips and derived coordinates remain distinct.
  No CRS, survey-date correction, magnetic correction or stroke ownership is
  inferred. Native 3D hull output is not a watertightness guarantee.

## Corpus and distribution proof

The real corpus is [test2 at 4c00c008a8dd441d70f9c62aa376c20b5914c7e0](https://github.com/dlubom/test2/tree/4c00c008a8dd441d70f9c62aa376c20b5914c7e0).
`validation/corpus-inputs.json` preserves all 262 paths, byte sizes, SHA-256 and
Git blob identities. Identical bytes at different paths remain distinct context
cases. All are v3: 260 canonical zero trailers and 2 absent trailers; inspection
finds 291 trips, 38390 measurements and 26 references. The external original
corpus is not rewritten or copied into application runtime packages.

`scripts/validate-corpus.py` inspects every input and exercises 25 export profiles
per path: isolated text, and Therion, DXF plan/side, VisualTopo, Toporobot and 3D
DXF each isolated/context and raw/closed. Each run is repeated and both output
and report hashes must match. Complete packages must contain exactly the artifact
and report, with matching input/output identity; failures must leave no output.
Frozen project results distinguish successful, unsupported and failed outcomes.
Native text byte comparisons are separately identified. Other corpus outputs
are regression results, not independent native truth.

The frozen run has **6550 outcomes**: 6504 complete packages and 46 explicit
failures, each repeated identically. There are 28 negative Toporobot-index
failures across seven input paths and four profiles, plus 18 context-closure
failures across six format/view profiles (12 work limits and 6 iteration limits).
All 262 isolated text exports match original-program bytes. The 262 paths have
258 distinct input hashes. These failures are recorded limitations, not skipped
cases or successful native parity claims.

Linux, Windows and macOS on amd64 and arm64 build from the exact clean Git SHA.
Each ZIP includes the native executable, quick start, MIT/Go notices and contract documentation,
`BUILD.json` with full source revision/target/payload hashes, and `SHA256SUMS`.
An adjacent checksum covers the ZIP. The validator requires regular file
members (no directories or symlinks), exact membership,
checksums, executable mode and embedded Go VCS identity, then extracts to an
unrelated directory and runs with empty PATH: help/version, six exact native
byte cases, no-replace publication and malformed input failures. Go, Wine and
PocketTopo are development tools, not CLI runtime dependencies. Release builds
report the package version; unversioned developer builds remain `dev`. BUILD.json and
native validation bind version, target, source revision, toolchain and archive
hash. Six native release jobs and optional Windows menu acceptance are required
by `CI complete`; publication additionally validates all six artifact sets and
downloads the uploaded release assets for exact byte readback.

Byte compatibility, corpus regression, CAD/VisualTopo/Toporobot import acceptance
and rendered drawing fidelity are separate results. Optional R01/R02 consumer
checks and older TOP versions remain future scope. P15 now implements the
additional [Walls SRV measurement profile](SPEC.md#walls-srv-and-survex-svx-design-p15p16),
with P06 averages and original Walls validation. P16 implements individual-reading
SVX with pinned Survex 1.4.22 geometry/statistics validation. These destination
profiles are additional to native PocketTopo exporter parity.
The native milestone means the bounded profiles inventoried here, with these
limitations retained visibly.
