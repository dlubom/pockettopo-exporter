# Product and compatibility specification

This document holds durable intent and contracts. Delivery status belongs in
[PLAN](PLAN.md); executable checks and commands belong in
[DEVELOPMENT](DEVELOPMENT.md).

## Product boundary

Build a portable, offline CLI that reads explicitly selected PocketTopo TOP
inputs, exports requested formats, and explains what was emitted, transformed
or omitted. It must run without PocketTopo, Wine or a graphical session.
The original program is a development and validation reference.

Preserve measurements, plan sketches, extended elevations and embedded cross
sections. Start with TOP v3. Older versions require separately established
contracts and corpus evidence. Never edit inputs or rewrite TOP files.

Historical comparison, reconciliation, station matching between surveys and
cave catalogue decisions belong in JKTZ. No GUI, Bluetooth/calibration support,
automatic survey-date correction, inferred CRS/entrance, or implicit magnetic
correction is planned here.

## Native milestone and later formats

| Native PocketTopo 1.372 export | Output | Primary reference |
| --- | --- | --- |
| Text, first useful export | `.txt` | `Exporter.WriteText`, `Survey.WriteText`, `Station.WriteText` |
| Therion interchange | `.txt` | `Therion.WriteExport` |
| Graphics DXF, plan and extended elevation | `.dxf` | `GraphicsForm`, `DXFWriter` |
| VisualTopo | `.tro` | `VisualTopoExp` |
| Toporobot | `.text` | `ToporobotExp`, `ToporobotForm` |
| 3D DXF | `.dxf` | `ThreeD.WriteDXF` |

Native Therion interchange is not a ready-made `.th`/`.th2` project. Native 3D
DXF is not Survex `.3d`. Walls `.srv` and Survex `.svx` are confirmed subsequent
scope. Layered SVG, XVI, direct TH2, separate cross-section files and PNG/PDF
are conditional additions; they do not block native exports.

The native milestone covers documented options and relevant input/context
combinations, not just one file per extension. Maintain a tested capability
matrix as each exporter arrives; add no empty exporter framework in advance.

## Source and compatibility contract

The reference is standard PocketTopo **1.372**, executable SHA-256:

```text
048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371
```

Use original-program observations to establish behavior, recovered C# to
explain it, and IL to resolve arithmetic/control-flow ambiguity. Preserve and
report disagreement; do not generalize to Swiss 1.472. Start at the sibling
[`analysis/ANALYSIS.txt`](../../pockettopo-decompilation/analysis/ANALYSIS.txt).

- Preserve original bytes, raw fields, record ordering and byte spans. Copies
  returned by source accessors must not allow mutation of the retained source.
- Raw station bits, native internal identity and display names are distinct.
  Empty display text does not make all reserved IDs identical.
- Preserve embedded dates as recorded values. Their relationship to field dates
  belongs to independent provenance, not an assumption in this parser.
- Default to one isolated input. Native template/directory context is an
  explicit later mode with deterministic file inventory, hashes and ordering.
- For each exporter fix settings, selection, grouping, arithmetic, ordering,
  encoding and line endings. Target exact native bytes under those conditions.
  Report semantic equivalence only with an explicit normalization rule.
- Byte identity, semantic equivalence, destination-application acceptance and
  drawing/coordinate fidelity are separate results. Parsing and green CI alone
  prove none of the unimplemented export behaviors.
- Report malformed/unsupported data, resource exhaustion and output failures.
  Stricter bounded failures than the native program are explicit deviations.
  An unsupported successful prefix is not a complete supported document.

## Architecture and existing parser contracts

Keep responsibilities separate, introducing packages only as needed:

1. `internal/top`: bounded decoding, offsets, validation and structured errors.
2. `internal/source`: immutable raw records and original-byte provenance.
3. Future processing: native arithmetic, grouping, geometry and context, with
   source-row lineage retained separately from computed values.
4. Future exporters: format selection/serialization and explicit loss reporting.
5. `internal/cli`: argument, file, diagnostics and output-publication boundaries.

Existing prefix APIs retain their stopping positions, limits and error order.
The [historical contract index](HISTORY.md#preserved-contracts-and-evidence)
locates their detailed specifications; source comments and tests describe the
implemented contracts. Do not turn historical per-field APIs into a requirement
to add another family for every subsequent record.

[`ReadV3PlanElementsPrefix`](../internal/top/plan_elements.go) preserves ordered
complete Polygons (marker 1) and XSections (3). It consumes a terminator (0), or
an unknown marker and stops before its payload. Unsupported stops are successful
partial results. The side-mapping/marker APIs explicitly expose absence when
an unsupported plan stop prevents reaching the side drawing. Their byte spans,
consumed offsets and unparsed-tail sizes remain significant.

Limits are acceptance ceilings, not a promise of low memory use. Existing nested
prefix models retain multiple byte copies. The complete-file model should avoid
extending that chain unnecessarily and state plan/side aggregate budget rules.

## Complete v3 source document (P04d)

`top.ReadV3` / `ReadV3WithLimits` return an immutable `source.Document` with one
copy of the entire original input, flat table records, table/field spans,
overview mapping and separate `Plan()` / `Side()` drawings. They reuse existing
bounded table decoding without retaining its nested prefix objects. Drawings
reuse the immutable Polygon/XSection record types; their historical `Plan*`
names do not imply that side elements belong to the plan. No coordinates,
colors, IDs, scale or directions are normalized. `Bytes()` and record accessors
return copies. Empty/malformed results have neither drawing.

Drawing budgets are independent per drawing: at most 100,000 complete elements,
1,000,000 points per Polygon and 1,000,000 points in each drawing. Additional
whole-document budgets default to 100,000 elements and 1,000,000 points across
both drawings. Earlier input/table/comment defaults are unchanged (64 MiB input).
Every bound is inclusive; zero is a real budget; callers may lower bounds but
never raise defaults. Stop markers consume no element/point budget. Point counts
are checked against both budgets and remaining bytes before allocation. These
are operational limits, stricter than the native reader.

Malformed/limit failures return the zero document and stable `ParseError`, with
no partial success. Unknown plan markers stop before side mapping; unknown side
markers stop before their payload. Both return a source document with a consumed
raw stop marker, `Complete()==false`, `Supported()==false`, and `unreached` tail.
`BodyEnd()` then denotes the stop boundary, rather than a terminated body.
Original bytes after that boundary remain available through `UnparsedBytes()`.

For two terminated drawings, `Complete()` means the supported body was read;
`Supported()` additionally applies this explicit tail policy:

| Bytes after side terminator | Tail kind | Supported | Accounting |
| --- | --- | --- | --- |
| EOF | `absent` | Yes, noncanonical | All body bytes consumed |
| Exactly Int32 zero | `zero_trailer` | Yes, writer form | Four trailer bytes consumed |
| 1–3 bytes | `partial_trailer` | No | Entire tail unparsed |
| At least four bytes, first Int32 nonzero | `nonzero_trailer` | No | Entire tail unparsed |
| Int32 zero followed by more bytes | `extra_bytes` | No | Zero trailer consumed; extra bytes unparsed |

`ConsumedOffset()+UnparsedTailSize()==len(Bytes())` in every successful full-file
result. `BodyEnd()` precedes the trailer; no tail payload length is inferred.
Native `DataSet.Read` accepts all probed tails and never validates EOF. Native
`DataSet.Write` emits Int32 zero. Native unknown markers are skipped with zero
payload length, while this reader preserves the existing bounded unsupported
stop policy. Writer form is not a claim that arbitrary raw colors/scales would
round-trip unchanged through native derived pens/mappings.

Memory is bounded but can be substantial. On macOS arm64 Go 1.26.3, one 100,000
point Polygon (800,064 input bytes) allocated about 12 MB in total; 100,000 empty
Polygons (600,054 bytes) allocated about 180 MB in total. These are benchmark
allocation totals, not peak retained memory or a whole-corpus scalability claim.
The CLI also clones element/table slices for summary access; earlier prefix APIs
retain their existing higher-copy representation. Reproduce both assessments
with `go test ./internal/top -run '^$' -bench '^BenchmarkDocument' -benchmem`.

Native evidence, independent transcript expectations, fixture readback and
commands live in [testdata/README](../internal/top/testdata/README.md#p04d-complete-document).

## Inspect CLI (schema 1)

`inspect <input.top> --json` reads exactly one named input, bounded to 64 MiB plus
one detection byte. No neighboring-file scan, geometry, or vertex dump is done.
`--help`, `-h` and `--version` keep their existing success/exit behavior; help now
lists inspection. Export remains unavailable.

The deterministic JSON object contains `schema_version: 1`, `tool_version: "dev"`,
`source` (`sha256`, `size_bytes`, `top_version`), `status` (`supported`,
`unsupported`, `malformed`), `complete`, `supported`, `counts` (trip/measurement/
reference totals), `plan` and `side` (element/Polygon/XSection/point totals, stop
reason/raw marker/offset), `accounting` (`body_end`, `consumed_bytes`,
`unparsed_bytes`, `tail_kind`) and `diagnostics` (`code`, `field`, `offset`).
Key order, compact encoding and one final LF are fixed by golden tests. No path,
clock or timestamp is included. Diagnostics are an empty array on supported input.

Exit 0 requires complete supported input and successful stdout writing. All other
outcomes exit 1 with human diagnostics on stderr. If the bounded input was read,
analysis failures also produce JSON: malformed results have null counts/drawings,
zero body/consumed bytes, all bytes unparsed and the precise parse-error offset.
`top_version` is null if fewer than four header bytes exist. Unsupported versions
have status `unsupported`; unknown markers/tails have structured diagnostics.
Argument/I/O/oversized-input failures produce no JSON, since complete input
analysis is unavailable. An oversized prefix is never hashed as a complete file.

## Native semantics that future work must preserve

- `Survey.EvalData` groups consecutive matching/reversed pairs and averages
  integer vector components. Arithmetic means of lengths/angles or JKTZ's
  confirmation policy are not substitutes. Use actual repeated-shot fixtures.
- Fixed-point trigonometry, signed division/shifts, overflow, float32 steps and
  .NET text formatting require independent native cases.
- Projection and loop closure affect drawings. The closure constant 10 is an
  error threshold, not a ten-iteration limit; bounded failure needs a stated policy.
- An XSection stores an anchor, station ID and direction. Nearby strokes remain
  independent elements; proximity does not establish ownership. Preserve native
  splays/connectors in complete views. Separate files require an explicit crop
  or selection contract.
- Standard 1.372 declination handling does not calculate a magnetic correction
  from location/date behind the Auto flag.
- `DataSet.Write` emits an Int32 zero after the side drawing; `DataSet.Read`
  stops after reading the drawing and does not validate that trailer or EOF.
  Full byte accounting must distinguish native-reader acceptance from canonical
  writer output. Confirm the boundary cases with a native probe before defining
  the complete-file policy; preserve/classify extra bytes rather than inventing
  a native requirement for a zero trailer.
- `Survey.WriteText` calls `Update`; `Station.WriteText` writes individual rows
  with processed flag semantics. P05 must establish which update effects affect
  text output, rather than assuming all geometry is required or bypassing it.

## CLI behavior to introduce with its implementation

`inspect` is implemented as specified above. `export` remains planned; P05 fixes
its arguments and exit semantics.

Accept explicit paths and formats. Preserve inputs; reject input/output
collisions and accidental overwrites. Write temporary output and publish only
completed results. Keep diagnostics on stderr and machine output deterministic.
Emit only requested artifacts. Define multi-output failure behavior before
claiming atomic export sets.

A requested provenance/loss report should identify tool/schema version, source
SHA-256, context inventory/order, resolved settings, output hashes, contributing
source rows, transformations and omissions. Record identity is source hash plus
kind/index/offset, not a display name or matching geometry. Avoid unnecessary
absolute local paths and timestamps in deterministic output.
