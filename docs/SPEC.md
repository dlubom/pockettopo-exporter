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
3. Processing when required: native arithmetic, grouping, geometry and context, with
   source-row lineage retained separately from computed values.
4. `internal/exporttext` and future exporters: format selection/serialization and explicit loss reporting.
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
lists inspection. Help also lists native text export.

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

## Isolated native text export (P05)

`export <input.top> --format text --output <new-directory>` reads only the
explicit input, using P04d's 64 MiB input ceiling, complete-body and tail policy.
Malformed, unsupported, incomplete and oversized inputs fail before output
creation. No template directory or neighboring file is read. The fixed P05
profile is metres, a 360-degree circle, invariant culture, UTF-8 without BOM and
CRLF on every platform. Other native settings remain for capability expansion.
The title is the input basename without its last extension; empty/invalid UTF-8
or control-character titles are rejected as an explicit stricter boundary.

`exporttext.Build` keeps every trip, reference and measurement in original table
order. Repeated, reversed, disconnected, invisible and anonymous rows are not
filtered or averaged. Isolated trip/reference/data boundaries are zero. Positive
trip indices print as source index plus one even if out of range, with an
unresolved-index report entry; negative indices print blank. Reserved IDs use
native display semantics while raw bits and internal identity remain distinct
in the report. Distances are metres with three decimals; angles use the raw
unsigned azimuth or signed inclination with two decimals. Auto trip declination
prints zero and recorded ticks print a calendar date without asserting a field
date. Measurement distance Int32 minimum and reference coordinate minimum
sentinels become blanks. References use native MetricGrid metre values with two
decimals. The .NET Framework 2.0 format retains 15 significant decimal digits,
rounds decimal midpoints away from zero and suppresses negative zero; Go's
ordinary fixed formatting is not used as an oracle.

Station flag priority is invisible `~`, projected `=`, flipped `<`, special `.`,
then blank. Native comments replace quotes first, then double backslashes,
replace CR with literal `\r`, and remove LF. A present empty measurement comment
prints `""`; empty trip/reference comments are absent. These quirks are preserved,
not repaired. Independent original-program bytes cover empty sections, Unicode,
all 256 flag values, actual repeated/reversed rows, signed/rounding boundaries,
sentinels, large references, duplicate/anonymous identities and drawings.

### Text dependency on Survey.Update

Native `Survey.WriteText` calls `Update` before writing tables. C# and IL confirm
that Update's EvalRef, Trip.Evaluate, EvalData, SetupShot, SetupXsect and SetBox
modify derived identities, links, vectors, coordinates, bounds and cached
_declination_, while text reads source from/to/dist/azimuth/inclination/flags/
trip/comment, reference num/loc/comment and trip date/declCorr/automatic/comment.
The fields consumed by text are unchanged. The independent native probe compares
the entire table text before/after Update, including real grouped/reversed shots:
the first mixed row's derived vector is (-290, 330), but all eleven original rows
and their individual distances/angles remain in the golden.

P05 therefore elides derived geometry explicitly, without promoting P06/P07
arithmetic or geometry. It does **not** reproduce native Update execution failures:
`update-error.top` demonstrates a first projected shot causing a native
NullReferenceException without changing text fields. P05 can serialize those
preserved fields, and every report declares this deviation. Byte compatibility
is claimed for the fixed profile when native Update completes, not equivalence
of geometry or native failure behavior. There is no loop closure or magnetic
correction in text. This is an evidence-based text dependency analysis, not an
assumption that other exporters may bypass Update.

### Publication and report

The explicit output directory must not exist. `Mkdir` reserves it exclusively;
each artifact is created exclusively as a temporary file inside it, written,
synced and closed, then hard-linked to its final name without replacement.
`report.json` is published last. Filesystems lacking hard-link support fail
explicitly. Existing files/directories/symlinks and concurrent final-file entries
are never overwritten. On failure, remove only files this invocation created
and the directory if empty; retire removed temporary names and check recorded
file identities before cleanup. Exclusive private `.survey.txt.owner` and
`.report.json.owner` hard links pin the original inodes before temporary names
are retired. Keep these anchors through every rollback ownership check; remove
public owned paths first and anchors last. This prevents immediate inode reuse
from making a foreign replacement appear owned. Ordinary successful publication
removes both anchors and leaves exactly the two artifacts.

Publication is committed immediately after linking the complete report. Subsequent
temporary/anchor cleanup failures return an explicit `export published` error,
preserving both complete artifacts and remaining anchors. Report cleanup failures
and leave foreign entries, including replaced anchors.
Identity checking and unlink are separate filesystem operations; arbitrary
same-user pathname replacement during that interval or tampering with private
anchors is outside this guarantee.
The reserved output directory must not be actively modified during export.
No recursive removal or source rewrite is performed. This is not an atomic
multi-file transaction: process termination can leave an incomplete new directory;
absence of a complete report means the package is incomplete. No filesystem
crash-durability guarantee is claimed. A notification failure returns exit 1 but
keeps complete files and explicitly reports that publication already succeeded.

Exit 0 requires successful input analysis, serialization, publication and stdout
notification. All failures exit 1 with diagnostics on stderr. Inspect JSON,
version and historical prefix contracts are preserved; help lists export.

`survey.txt` contains exactly the native text bytes. Deterministic schema-1
`report.json` contains tool/format/profile/isolated-context/title, complete source
SHA-256/size/TOP version/tail classification, text SHA-256/size, the explicit
Update policy, and one lineage record for every source table row (zero-based
index, original byte span, one-based output line, relevant raw values and
raw/native/display identities). There are no clocks, absolute paths or output
paths. Counts/details report dates/declination and numeric precision, sentinel
blanks, unrendered roll/flag metadata, removed comment LF, hidden reserved IDs,
unresolved trip indices, omitted drawing elements, mappings and trailer.
Original comments/drawings remain in the hashed TOP source, not in a recovered
or rewritten source file. A source hash requires retaining that original file.

Text output has a separate inclusive 128 MiB ceiling. Source table allocations
and report lineage remain bounded by P04d input/table limits, but can use much
more memory than the input; this is not a low-memory whole-corpus guarantee.
See [native evidence](../internal/top/testdata/README.md#p05-native-text-export)
for commands, hashes, controlled context and independent goldens.

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
  with native flag semantics. P05's C#/IL and probe-backed analysis establishes
  text-field independence and the explicit geometry-failure deviation above.
  Other exporters must establish their own Update dependencies.

## CLI behavior

`inspect` and the isolated P05 text `export` are implemented as specified above,
including their arguments and exit semantics. Other exporters remain planned.

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
