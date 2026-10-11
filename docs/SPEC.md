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

## Native grouped measurement processing (P06)

`processing.GroupMeasurements(source.Document)` consumes a complete supported
v3 document from `top.ReadV3`, returning an immutable `Result` or an empty result
and error. `Rows()`, `Groups()` and `Group.Members()` return copies. Every row
retains its original immutable `source.Measurement`, source index and complete
field/record byte spans. Groups retain the head's raw From/To, correction and
all contiguous member indices with a per-member reversal flag. Raw aliases and
source flags/trips remain visible; no record or input byte is rewritten.

The context is one isolated input with zero native template boundaries.
Per-row correction follows standard 1.372 `Trip.Read/Evaluate/DeclCorrByIndex`:
valid manual trips use their signed stored correction; Auto (`-32768`), negative
or out-of-range indices use zero. Embedded date/reference data does not supply
a magnetic correction. Real azimuth is the recorded azimuth plus that row's
correction, narrowed to signed Int16. The quarter-wave sine table has 16,385
Q30 entries generated with ties-to-even rounding; native table bytes are tested
exhaustively. `Angle.ToRect` multiplies in Int64, adds `2^29`, arithmetic-shifts
by 30 and narrows to Int32. This rounds negative halves toward positive infinity.
Inclination produces DH/DD, then real azimuth and DD produce DX/DY (all mm).

Grouping reproduces the portion of `Survey.EvalData` before placement:

- Native ID value `-1` is undefined, including raw aliases. Undefined From rows
  are retained as individual `undefined_from` groups with zero evaluated vector.
  Their row vectors are independently calculated with native Angle arithmetic;
  native EvalData itself skips their vector calculation.
- Undefined To rows are individual `splay` groups, even when adjacent and equal.
  Every other endpoint, including distinct reserved IDs with empty display text,
  participates in `leg` groups by native identity equality.
- Only consecutive identical/reversed endpoint pairs join a leg. The head fixes
  orientation; self-pairs match forward first. Trip, flag, roll and comment
  changes do not split a group. An intervening row ends it; no proximity/outlier
  threshold or physical-value filtering is applied.
- DX/DY/DH contributions reverse sign for reversed rows; DD never does.
  Int32 accumulation and negation wrap exactly as unchecked IL `add/sub/neg`.
  Divide the wrapped totals by the member count, truncating toward zero. Only
  the head's Flip negates averaged DD, and only for legs. Retain zero, negative
  and sentinel distances without silently repairing them.

Forward/reverse directions call the equivalent of `Station.GetAvDirection` on
that averaged vector. Distance is the square root of the unchecked Int64 square
sum, narrowed to Float32, not an arithmetic mean of distances. Reverse negates
Int32 DX/DY/DH after computing distance. Azimuth is Float32 atan2(DX,DY), minus
Float32 radians of the head's trip correction, adding one Float32 `2*pi` if
negative. Inclination is Float32 atan2(DH,sqrt(DX²+DY²)). Zero/nonpositive/NaN
distance yields zero angles. No extra modulo or angular mean is substituted.
All frozen native direction Float32 bits are compared exactly.

Reference placement, SetupShot/SetupXsect projection, closure, geometry-derived
station orientation are absent from P06; P07 exposes them separately below.
Template-directory context and other exporters remain absent.
A reverse direction is an explicit orientation option, not an inferred native
placed station. This API therefore also processes inputs whose later native
Update would fail. P05 keeps its isolated text contract and exact bytes, emitting
all individual source rows without calling this processing API.

### Process CLI (schema 1)

`process <input.top> --json` reads only the named input, bounded by P04d's 64 MiB
ceiling plus one detection byte, and emits deterministic compact JSON plus LF.
The profile is `isolated_native_grouping`; `schema_version: 1` and `source`
(SHA-256, full size and TOP version) identify the input. `rows` preserves every
raw measurement field, raw/native/display IDs, source offsets, correction,
real azimuth and vector. `groups` holds head/kind/IDs, contiguous `members`
(`row_index`, `reversed`), averaged vector, correction and both directions.
Offsets retain the source API's field names and Start/End byte intervals.
Direction numbers are mm/radians; `float32_bits` stores unsigned IEEE-754 bits
in distance/azimuth/inclination order. Nonfinite numbers serialize as null,
retaining their bits. Empty rows/groups serialize as empty arrays.

Exit 0 requires a supported complete source and successful JSON output. Invalid
arguments, I/O/oversize/malformed/unsupported/incomplete inputs and output errors
exit 1, with diagnostics on stderr. Analysis failures produce no JSON; a failing
writer may have received a partial stream. Existing `inspect` schema 1 and P05
export bytes stay unchanged; help additionally lists the process command.

Processing has linear time/storage in row count under the inherited one-million
measurement ceiling. It materializes rows/groups and the CLI materializes JSON
metadata; there is no separate output-size ceiling or streaming-memory promise.
On macOS arm64 Go 1.26.3, grouping 100,000 pre-read rows allocated about 55.4 MB
for one group and 102 MB for singletons. These are total allocations excluding
parsing and JSON, not peak retained memory or a largest-input guarantee.
Reproduce with `go test ./internal/processing -run '^$' -bench '^BenchmarkGrouped' -benchmem`.

C#/IL references, original-program captures and fixture hashes are in
[testdata/README](../internal/top/testdata/README.md#p06-native-grouped-measurements).

## Isolated native geometry (P07)

`processing.PlaceGeometry(Document, GeometryOptions)` requires one complete
supported v3 source. It uses P06 groups without changing their vectors, source
rows or average directions. P05 text and schema-1 `process`/`inspect` output are
unchanged. There are no templates or neighboring-file scans.

`EvalRef` chooses the first reference whose native ID is not UNDEF as the base.
It subtracts signed Int64 east/north and Int32 altitude, then narrows differences
to native Int32 millimetres. Duplicate reference records survive; the first
native identity wins lookup. UNDEF references do not participate in geometry;
raw records remain in the source document. If no usable references exist, the
first measurement FROM other than UNDEF supplies a synthetic origin. No origin
is invented for another disconnected component. A synthetic node has key
`reference:-1`, no source span and `base_key=none`; it is not a geographic fix.

Each node keeps its source kind/index key, derived endpoint identity, head record
span, attachment key, status, vector, position and Int64 Q30 corrections.
The endpoint retains the selected FROM/TO/reference bits, except undefined FROM
uses canonical UNDEF. Raw identities and all source fields remain in Document
and P06 rows; endpoint identity never substitutes for source-row identity.
`group:<head-index>` refers to P06's group/head and member lineage; averaged
members are not separate placed stations. Nodes are returned in source-reference
then source-group order; synthetic origin precedes groups. Undefined FROM,
unplaced groups and zero-distance splays remain explicit. Accessors copy slices;
source records and bytes remain immutable.

Native coordinates are X east/right, Y south/down, H downward and D extended,
all Int32 millimetres. Geometry is relative to the chosen base, not a declared
CRS or field-verified tie. SetupShot preserves native first-pass reverse rules,
LIFO pending stacks and both retry phases. The first placed native identity
remains the lookup node; later occurrences create explicit loop endpoints.
Reverse placement negates DX/DY/DH but preserves DD. SetupXsect attaches splays
without putting them into station lookup. A zero-distance splay has no placed
endpoint. Invisible shots still place and close; no native display bounds or
survey-length statistics are claimed by this API.

Extended projection uses the head's raw azimuth, grouped DD and the actual
attachment orientation and Flip. The head raw distance supplies the zero-distance
azimuth sentinel and closure weight, not the projected length. It follows projected/zero-distance predecessors
when the native program does, and reproduces SideProjection's Int16 angular
narrowing, Q30 sine table, signed shifts and multiply/divide order. It does not
replace this with Cartesian length or P06's corrected average azimuth. Adjacent
nonprojected shots can revise earlier splays' D; final source-order node output
contains those revisions.

Closure is opt-in. Native Loop.Setup finds common ancestors/reference stops;
Loop.Close apportions Q30 X/Y/H corrections using each group head's **raw**
distance (not averaged length), unchecked Int32 weight and Int64 arithmetic.
Loop order is reverse insertion order. Corrections iterate against the initial
positions until the native reported maximum is <=10 mm, then placement is
rebuilt. D is not directly corrected. Zero-weight loops perform no correction.
Returned loops link both node keys, retain the final X/Y/H/D residual and report
a derived path weight. Inspection weights ignore temporary source mark bits;
actual closure retains their native effect, including failures.

The Go port replaces native null-ancestor/arithmetic exceptions and unbounded
work with `GeometryError{Code,Key}` and an empty result. Defaults allow 10,000,000
retry/ancestor/closure work steps and 10,000 closure iterations; zero chooses the
default, negative limits are invalid, positive limits are inclusive. These are
explicit portability limits, not native iteration counts. The native constant
10 is an error tolerance. All ordinary additions/negations/products still wrap
at native widths; checked Abs/division failures remain errors.

`geometry <input.top> --json [--close-loops]` emits deterministic schema 1 with
profile `isolated_native_geometry`, source SHA-256/size/version, base key,
closure selection/iterations, nodes and loops. Empty collections are arrays.
Exit 0 means geometry evaluation succeeded, including explicit unplaced nodes;
exit 1 means invalid arguments, source/native/resource/I/O/output failure, with
diagnostics on stderr and no analysis JSON on evaluation failure. The command
uses the same bounded source reader as P06. It does not publish an export package.
Original-assembly goldens, C#/IL references, captures and hashes are documented
in [fixture provenance](../internal/top/testdata/README.md#p07-native-geometry).
