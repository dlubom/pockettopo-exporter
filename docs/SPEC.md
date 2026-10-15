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
cave catalogue decisions belong in JKTZ. Optional Windows Explorer integration
adds an interactive shell adapter around the same exporter; the core CLI remains
offline and noninteractive. No full GUI, Bluetooth/calibration support,
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

The [P14 capability matrix](CAPABILITIES.md) closes that inventory for the
documented bounded profiles, including unimplemented native settings and
intentional deviations. It is not a claim of every native option or consumer
acceptance. P14 retains P05–P13 application behavior and report schemas.

P14 validation accounts for every path in the pinned test2 corpus using SHA-256,
size and Git blob identity, independently of duplicated bytes or neighboring
conversion files. Frozen corpus outcomes are project regressions; separately
captured original-assembly exports remain the native byte oracle. Every profile
must be deterministic, preserve inputs and verify complete output/report hashes,
or fail before publication with an explicit bounded/native diagnostic.

Distribution packages are ZIPs for Linux, Windows and macOS on amd64 and arm64,
built from a clean committed source tree. BUILD.json retains full Git revision,
target, Go toolchain, CLI version and payload hashes; SHA256SUMS covers every
other member including BUILD.json, and an adjacent checksum covers the ZIP.
Validation requires regular-file archive members, exact membership, byte hashes, executable mode and
matching embedded Go VCS revision, followed by offline native-byte CLI tests
from an unrelated directory. Release linker injection sets `cli.Version` to the
package version; `--version`, inspection and export report `tool_version` use it without changing JSON
schemas. Ordinary unversioned builds retain `dev`. Release manifests bind
version, target, SHA and `CGO_ENABLED=0`; native validation binds those fields to
the archive checksum and actual runtime tests. These CI packages become public
assets only after complete exact-SHA CI, six-target assembly validation, annotated
tag verification and uploaded asset byte readback. CI complete includes all six
native release jobs alongside existing platform/corpus/consumer/mutation gates.

### Windows Explorer adapter (1.0.0)

`windows-menu install|uninstall` registers/removes an owned `.top` static submenu
under the current user's Classes registry. The default file association is not
modified. Installation uses the absolute executable path; reinstall after moving
the program. A `%` anywhere in that executable path is rejected before registry
writes because Explorer interprets command-template placeholders; source paths
remain literal arguments, including percent tokens. Uninstall/reinstall must refuse foreign contents rather than delete
unowned registry data. Classic entries are under Show more options on Windows 11.

`shell-export <input.top> --format <format>` (DXF optionally `--view plan|side`)
uses existing export defaults and no-replace publication. New sibling directories
use human format names and bounded numeric suffixes. It displays output/report
location on success and readable diagnostics on failure; an actual console waits
for Enter. These commands are separate from noninteractive `cli.Run`. Shell paths
are passed directly as arguments, never interpolated into shell commands. Source,
export and provenance semantics remain those of the selected existing exporter.
Native Windows gates must exercise registry ownership, paths, collisions/errors
and actual Explorer menu invocation before publication.

## Walls SRV and Survex SVX design (P15/P16)

**Status:** P15 and P16 implemented. P15 exports `survey.srv`,
P16 exports `survey.svx`; both include `report.json`. Commands use
`export <input.top> --format srv|svx --output <new-directory>`.
One isolated TOP v3 input, measurement data only, without prior loop closure,
template context, drawings, derived LRUD or inferred geographic fixes.

### Measurement rules

| Case | Walls SRV | Survex SVX |
| --- | --- | --- |
| Consecutive repeated leg | One P06 spatial-vector average | All individual readings |
| Reverse member | Included in the average | Swap endpoints, bearing +180° modulo 360, negate inclination |
| Singleton or splay | Original measurement | Original measurement |
| Anonymous splay endpoint | `-` | `::` with colon separator |

- Use P06 identity/consecutive grouping and the head's direction, for any count.
  Do not sort, globally deduplicate, remove outliers or move intervening splays.
  Flip is not reversal. SVX repeats remain active, without `duplicate` flags.
- SRV averages use each member's correction before averaging and express the
  result relative to the head's correction. Do not replace P06 spatial averaging
  with separate length/angle means. SVX keeps each member's correction.
- Survex 1.4.22 does not merge opposite-direction repeats: a local probe produced
  two legs/one loop/20 m; normalizing direction produced one leg/no loop/10 m.
  A declination directive between same-direction repeats still allowed merging.
- Preserve station identity across trips, including `0` versus `0.0`. Walls names
  have at most eight characters; fallback is `p` plus unsigned native-ID base36,
  with a reversible, collision-checked map. SVX preserves literal dots using
  `*set separator :`, `*set names ._-`, `*case preserve`, `*truncate off`.
- Splays remain individual and outside cave-length statistics. Anonymous wall
  markers are a target convention, not proof of the physical point type.
- Metres/degrees, Distance/Azimuth/Inclination, three distance decimals and six
  angle decimals; rounding only at output. SRV: ASCII/CRLF, SVX: UTF-8/LF.
  Escape source text safely in comments; preserve originals in the UTF-8 report.
- Keep bearings in [0,360), including rounded 360 as zero. Derived P06 Float32
  inclinations within 0.00001° beyond ±90 normalize to that endpoint; preserve
  the native value in the report. Invalid raw inclinations are never clamped.
- All-zero named groups remain ties. Reject mixed zero/nonzero groups, nonzero
  cancellation, undefined From, name collisions, negative distances, raw
  inclinations outside ±90 and unsupported/non-finite results before publication.
  Both formats reject native-identical endpoints, including zero self-legs:
  Walls rejects duplicate names and Survex rejects self-legs with the declared
  inference profile. Distinct-name all-zero ties remain supported.
- The report retains source hashes, raw IDs/aliases, rows/spans, original/emitted
  values, grouping, corrections and omissions. Reuse existing safe publication
  and destination defaults; no guessed accuracy, team, entrance or flag semantics.

### Declination and editable dates

Default: emit manual declination before the first measurement (including zero),
then only at effective changes, including return to zero. Use separate lines:
`#Units Decl=...` / `*declination ... degrees`. A trip change alone does not
repeat the same correction. Manual trips use stored corrections; Auto/missing/
invalid trips use P06 effective zero, explicitly labelled unresolved in the report.

At each source-trip change, emit `;#Date YYYY-MM-DD` or `;*date YYYY-MM-DD`
for the user to fill and uncomment. Keep the recorded TOP date separately as
unverified metadata. For SRV averages, use the head's trip and report all members.
Date placeholders do not alter the default export.

To enable date-derived declination, comment out **all** manual declination lines:

- **Walls:** fill/uncomment dates, set the project's geographic reference and
  enable **Declinations — Derive from #Date**. Date and manual directives otherwise
  override one another in file order.
- **Survex:** fill/uncomment dates and geographic templates: input/output CRS,
  a known fixed station (or connection to a fixed survey), and `*declination auto`
  with a representative real location. A date alone does not enable correction.

SVX keeps individual readings available for recalculation. An SRV average spanning
multiple trips or corrections cannot be fully recalculated by editing one date;
report that limitation and recompute from original readings when needed. Geographic
position and verified dates are supplied by the user, never inferred by export.

### Examples

Three 10.020/10.000/9.980 m readings share bearing 90°, inclination/correction 0°.
A splay follows; the next trip uses correction +5°. Date templates remain comments.

```text
; survey.srv
#Units Reset Meters Order=DAV A=D V=D Case=Mixed
;#Date YYYY-MM-DD
#Units Decl=0.000000
1.0 1.1 10.000 90.000000 0.000000
1.1 -    2.500  0.000000 0.000000
;#Date YYYY-MM-DD
#Units Decl=5.000000
1.1 1.2  5.000 180.000000 0.000000
```

```text
; survey.svx
*require 1.4.22
*begin
*case preserve
*truncate off
*set separator :
*set names ._-
*units tape metres
*units compass clino declination degrees
;*cs <CRS>
;*cs out <OUTPUT_CRS>
;*fix <STATION> <X> <Y> <Z>
;*declination auto <X> <Y> <Z>
*data normal from to tape compass clino
;*date YYYY-MM-DD
*declination 0.000000 degrees
1.0 1.1 10.020 90.000000 0.000000
1.0 1.1 10.000 90.000000 0.000000
1.0 1.1  9.980 90.000000 0.000000
1.1 ::   2.500  0.000000 0.000000
;*date YYYY-MM-DD
*declination 5.000000 degrees
1.1 1.2  5.000 180.000000 0.000000
*end
```

Examples illustrate layout, not native TOP byte goldens. Production SVX resets
inherited calibration, flags, inference and accuracy inside the anonymous scope;
trips do not create separate station namespaces. Consumers own loop adjustment,
so equal closed SRV/SVX geometry is not required.

**References:** [P06](#native-grouped-measurement-processing-p06), original
`Survey.EvalData`, `Station.GetAvDirection` and corresponding C#/IL;
[Walls manual](https://texasspeleologicalsurvey.org/software/walls/Walls_manual.pdf),
[Survex commands](https://survex.com/docs/manual/datafile.htm#declination),
[Survex repeats](https://survex.com/docs/manual/svxhowto.htm#repeated-readings).
JKTZ is syntax/provenance prior art, not this project's averaging oracle.
P15 uses `internal/exportwalls.Build` and the existing checked no-replace,
report-last publication. Output has an inclusive 128 MiB ceiling; resource errors
return an empty result and publish nothing. Existing parser/table limits apply;
report lineage and source copies can require substantially more memory than the
input. Its deterministic schema-1 report includes raw trips/references/rows and
spans, source/output hashes, the reversible raw/native/name map, group members and
reversal, native P06 vectors/Float32 directions, original/emitted values, directive
line ownership, unresolved corrections and date-recalculation limitations.
References are retained as omitted records; no `#Fix` is emitted. Anonymous wall
shots remain individual `-` rows and are outside Walls cave-length statistics.
Original-assembly vector/Float32 captures and original Walls v2.3.0-beta.1
compilation/database validation are separate from the project SRV/report byte
regressions. The native gate uses manual correction, commented dates and no
geographic reference, and compares unadjusted NTV vectors and named lengths.
See [P15 provenance](../internal/top/testdata/README.md#p15-walls-srv).

P16 uses `internal/exportsvx.Build` and the same report-last no-replace
publication and inclusive 128 MiB output ceiling. Its deterministic schema-1
report retains raw measurements/trips/omitted references and record spans,
raw/native/display/emitted identities, original polar values, separately
normalized values, each emitted row and line, P06 members/reversals and native
summary vectors/directions, correction sources and directive line ownership.
P06 summaries validate the agreed failure boundaries and provide provenance;
they do not replace individual readings or control Survex's averaging.

The anonymous SVX scope starts before any comments or numeric directives. It
sets the comment character with `*set comment x3b` (semicolon) and the decimal
character with `*set decimal .`, so includes from decimal-comma or hash-comment
projects compile unchanged and restore the caller's lexical settings at `*end`.
The anonymous SVX scope resets units, calibration, accuracy, all three leg
flags and inference before data. It preserves literal dots and disables
truncation. Source comments are ASCII-escaped on bounded comment lines within
the UTF-8 file; the report preserves original Unicode. Manual correction is
emitted initially/change-only per row; date templates track every trip change,
including inside repeat groups. Geographic/automatic templates stay commented.
Empty inputs produce a metadata-only package; cavern rejects it with `No survey
data`, so empty packages are outside the consumer acceptance claim.

Pinned cavern/dump3d 1.4.22 compares nine compiled geometry/statistics cases with
independently authored literal SVX controls. These cover repetitions, reversed
heads/members, interrupted pairs, changed corrections/active dates, zero ties,
verticals, zero splays, aliases and an explicitly edited automatic profile.
Known synthetic UTM coordinates and test dates are supplied only in the consumer
test, never inferred by the exporter. A hostile include verifies resets and
scope restoration; an actual self-leg compiler failure is a negative control.
The checked `.3d` graph and length statistics are Survex results, distinct from
native integer P06 geometry. See [P16 provenance](../internal/top/testdata/README.md#p16-survex-svx).

Acceptance and implementation boundaries: [P15](PLAN.md#p15--walls-srv),
[P16](PLAN.md#p16--survex-svx).

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
  explicit P08 mode with deterministic file inventory, hashes and ordering.
- For each exporter fix settings, selection, grouping, arithmetic, ordering,
  encoding and line endings. For native P05–P13 formats, target exact native
  bytes under those conditions. P15/P16 are additional destination formats;
  their contracts require source fidelity and destination validation instead.
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

The deterministic JSON object contains `schema_version: 1`, `tool_version` (build version, or `"dev"` for unversioned builds),
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
including their arguments and exit semantics. P09–P16 exporters below are also
implemented; optional P17–P19 remain planned.

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
Other exporters remain absent; explicit template-directory context is separate below.
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

## Explicit template-directory context (P08)

`context <input.top> --template-directory <dir> --json [--close-loops]` is the
only directory-scanning command. Inspect, text export, process and geometry
retain their isolated P05–P07 behavior. Current input is a complete supported
v3 document; templates decode only v3 trips/measurements/references, matching
native `DataSet.Read(template=true)`. Preserve full original template bytes;
ignored drawings/tails are hashed and counted, never interpreted as supported
full documents. Source records, raw trip indices and source-local spans remain
unchanged. Embedded dates remain source values, not verified survey dates.

Inventory selects immediate regular files with case-insensitive `.top` suffix,
excludes the exact current basename even if the selected directory differs,
ignores subdirectories and other suffixes, and sorts names by UTF-8 byte order.
Names must be valid UTF-8. Reject selected symlinks/nonregular inputs; pin file
identity with Lstat/Open/Stat before bounded reads. This is a reproducible read
snapshot, not an atomic filesystem snapshot: concurrent file-content/directory
changes require a new capture. Hashes describe bytes actually read. A current
basename neighbor is excluded without reading it. Current input is last in the
reported inventory; its source index follows all sorted templates.

Native `DataSet.Load` uses Windows `*.top`, culture-sensitive `Array.Sort`, and
exact basename comparison. Its .NET 2 wildcard also selects `.topx`; the frozen
pattern probe confirms this legacy extension behavior. Exact `.top` selection
and portable byte ordering are explicit deviations. The invariant-culture ASCII
geometry fixtures agree. Native enumeration/read failures are ignored and may
retain partially read tables; this mode fails closed, emits no JSON on input/processing failure,
and returns no partial context. Only v3 templates are accepted. Ceilings include
1024 selected templates, 100000 scanned directory entries, 64 MiB combined input,
32767 combined trips and one million combined measurements/references. All are
inclusive; parser and P07 geometry work/iteration ceilings still apply. Limits
are operational bounds, not a low-memory guarantee; source tables and geometry
can retain several copies within them.

`processing.GroupContext` resolves the native Int16 global trip index against
the combined trip list, separately retaining raw source indices and per-file
lineage. It groups consecutive/reversed template rows across file boundaries,
then groups current rows separately. `PlaceContextGeometry` evaluates template
references/rows first, current references/rows second, carries a real template
reference base, uses native current-before-template attachment priority, and
uses the combined first defined From for synthetic template roots. Current
loops close with `all=false`: template nodes are fixed closure boundaries;
template loops are not closed. Native projections can still update splays on
shared template parents. P07 arithmetic, retry ordering and bounded failures
apply unchanged. Template keys have `template:` prefixes, current keys retain
P07 names; indices address each phase's concatenated table. Synthetic roots
have record index -1 and no byte span or source-file identity.

Deterministic schema-1 JSON has profile `native_template_directory`, ordering,
excluded basename, inventory (names, roles, SHA-256, sizes, consumed/ignored
bytes and table counts), groups with every member's source index/local row/span
and reversal, nodes with original identities/spans and source-file/local-record
lineage, base key, closure state/iterations, and current loop endpoints/weights/
residuals. Synthetic node lineage is null. No clock, absolute paths or inferred
source identities are serialized. Output errors exit 1; success exits 0 with
one compact object and final LF. This schema is a project inspection format,
not a native export. Other exporters and directory-aware text export are outside
P08. See [native evidence](../internal/top/testdata/README.md#p08-native-directory-context).

## Native Therion interchange (P09)

`export <input.top> --format therion --output <new-directory>
[--template-directory <dir>] [--close-loops]` publishes `survey.txt` and
`report.json`, using P05's checked no-replace publication. Options can follow
output in either order, each once. Default input is isolated; explicit context
reuses P08's complete inventory, byte/table/file ceilings and fail-closed reads.
Both Therion modes use the context processing ceiling of 32767 total trips and
one million total measurement/reference rows; this is stricter than the P04d
reader, while P05–P08 APIs retain their own existing ceilings.
The serializer `exporttherion.Build` takes the immutable current document,
explicit template table prefixes and P07 geometry options. Unsupported input,
native geometry failures and output/ordering limits return no partial result.
P05 text and P06/P07/P08 output schemas retain their existing behavior.

Exact profile: PocketTopo 1.372; metres/degrees; invariant decimal formatting;
UTF-8 without BOM; CRLF. Native export emits current raw FIX references, then
TRIP/DATE/DECLINATION/DATA for every current trip, then PLAN and ELEVATION.
Recorded dates remain unverified; Auto declination is zero. DATA selects raw
consecutive matching/reversed heads (`First(..., true)`), even across the last
template/current boundary, then filters by the head's trip. Negative/out-of-range
trip heads and zero splays emit no DATA row. Undefined-FROM rows have zero derived
direction; consecutive such rows can still be suppressed by First. Raw rows
remain linked to their local source indices and byte spans. Directions use the
placed grouped vector and original head declination, including native Float32
multiplication before degree conversion; the head Flip selects `<` or `>`.
Closure changes drawing placement, not raw source observations.

STATIONS traverses the native successful prepend order, selecting the latest
placed occurrence per identity, then inserts using SortedList's native unchecked
Int32 subtraction comparer. This can differ from mathematical ordering at extreme
IDs. SHOTS follows that same successful order, followed by each station's splay
attachment order. Templates are omitted from the current centerline, while
current legs may start at template coordinates. Directory lookup instead retains
the first current identity, falling back to the first template identity. These
ownership rules are separate and remain source-linked in the report.

Drawings run native phases 1, 2 and 0: XSection station labels after STATIONS,
XSection rays after SHOTS, then ordered Polylines/CONNECT records. The origin is
GetRef's first defined current FROM, or the first combined FROM when current
measurements are empty. The origin uses directory ownership, not the first FIX
or the label winner. Mappings/scales do not affect export coordinates. All point
additions/negations wrap in Int32 before conversion to three-decimal metres.
Colors 1/2/3/4/5/7 map to BLACK/GRAY/BROWN/BLUE/RED/ORANGE; every other raw value
maps to GREEN. Empty/singleton Polygons still emit their native header/points.

XSections resolve through the current/template directory. Unresolved stations
emit nothing and are reported. Resolved sections emit a label and CONNECT even
without rays. Direction -1 uses the station's attached grouped DX/DY; all other
Int32 directions use native ToRect(DD, raw head azimuth minus direction) and DH.
The final coordinate of each native ray has an intentional trailing tab. The
side drawing uses D/H for centerline and connectors, while horizontal embedded
rays retain their native DX/DY projection. Rebuilding current closure retains
template splays and replaces current splays without duplicate stale snapshots.

The byte ceiling is 128 MiB inclusive, checked during ray/point expansion; native
SortedList ordering has a 10 million combined binary-search and insertion-work
ceiling. Existing P07 work/
closure bounds apply before serialization. These explicit failures replace native
partial files, swallowed exceptions or unbounded work. Publication occurs only
after complete serialization. A terminal-notification failure preserves the
already published package.

Report schema 1 records tool/profile/options, current source/output hashes,
size, trip/reference source spans and output lines, DATA members/reversal and
omission reasons, drawing raw color/ID/direction/spans and every emitted line,
and geometry keys with local file/record lineage. Synthetic roots have source and
record indices -1. CLI input inventory hashes full files, including ignored
native-template bytes; library template entries additionally hash the exact
consumed table prefix. Native comments, roll, unrendered flags, mappings/trailer,
rounded directions, hidden station labels and default color mapping are explicit
losses. No CRS, entrance, stroke meaning or cross-section cropping is inferred.
Native byte identity does not establish destination-application acceptance or a
ready-made `.th`/`.th2` project. Independent capture and provenance are in
[testdata/README](../internal/top/testdata/README.md#p09-native-therion-interchange).

## Native graphics DXF (P10)

`export <input.top> --format dxf --output <new-directory> [--view plan|side]
[--scale 1..100000] [--dxf-options <mask>] [--grid-unit metres|feet]
[--template-directory <dir>] [--close-loops]` publishes `survey.dxf` and
schema-1 `report.json`. Defaults: plan, scale 500, native mask 14 (shots, rays,
labels), metre grid and isolated input. All option arguments are single-use;
unknown, duplicate, missing or invalid values fail. The decimal option mask
accepts 0..4094 excluding bit 1; `--view side` sets native SIDE. Options are
independent, including layer options when the corresponding geometry is disabled.

| Bit | Native option | Behavior |
| --- | --- | --- |
| 1 | SIDE | Selected by `--view`; extended D/H rather than plan X/Y |
| 2 | ENS | Survey legs/station circles; embedded XSection station geometry |
| 4 | ENX | Survey and embedded XSection rays |
| 8 | ENL | Labels of exported stations with positive raw head distance |
| 16 | ENA | Template stations and attached rays; includes template grid bounds |
| 32 | SEPS | Shot layer and fallback layer for rays/labels |
| 64 | SEPX | Ray layer; also puts ordinary shots on Shots, as native |
| 128 | SEPL | Label layer |
| 256 | SEPC | Sketch color layers, including LGray connectors |
| 512 | GEO | Add native reference east/north to coordinates |
| 1024 | ENG | Native survey grid |
| 2048 | SEPG | Grid layer |

The byte profile is original PocketTopo 1.372 graphics DXF: AC1002, literal
INSUNITS=4 at every scale, UTF-8 without BOM, CRLF and invariant culture.
Coordinates add drawing origin and negate Y as wrapped Int32 *before* Double
conversion. GEO adds the actual native reference Int64 east/north converted to
Double; even side coordinates use these fields. Divide by scale and use Framework
Double's general 15-significant-digit representation, trimmed fractions and
uppercase exponents. Circles have radius 50/scale; label height stays 1.
The native header does not describe rescaled real-world units or establish a CRS.

Current drawing elements retain source order and precede template/current survey
geometry. Singleton Polygons produce POINT, longer ones POLYLINE/VERTEX/SEQEND.
All raw colors map to the seven native pens, with unknown colors green. Empty
Polygons raise an explicit native-failure error. Resolved XSections always emit a
DASH connector from the offset sketch center to the already placed station; ENS
gates the embedded circle and ENX/ENL its rays/label. Lookup preserves native
current-first/template fallback ownership; unresolved XSections emit nothing.

Survey traversal uses native prepend order and cross attachment order. Invisible
legs/rays emit no line, while named leg circles and eligible labels remain native.
Reference/synthetic-root stations have native undefined TO and emit no circle.
Actual steep/projected rays use DASH; corresponding template rays with a null
native dashed pen are omitted. Dash tests retain raw inclination/distance and
wrapped Int32 products, while coordinates use grouped/placed vectors. Embedded
horizontal rays use raw vector DX/DY and inclination; directed rays use native
ToRect with raw head azimuth and grouped DD, with the original projection threshold.

Grid spacing is 1000 mm in metre mode or native 1520 mm in feet mode. Bounds are
visible placed legs and references, excluding splays and sketch vertices. Template
bounds are included only with ENA. Integer division truncates toward zero; expand
the bounds by two grid steps and emit vertical lines before horizontal lines.
Empty native bounds emit no grid lines. Wrapped/nonterminating grid arithmetic and
Math.Abs(Int32.MinValue) failures return explicit errors before publication.

`processing.PrepareGraphics` provides a detached view without changing P05–P09
outputs or schemas. `exportdxf.Build` validates supported input through the existing
bounded geometry/context path. Default output ceilings are 128 MiB and 1 million
entities (including VERTEX/SEQEND); exact limits are inclusive. Grid cardinality is
checked before iteration. A native missing-reference GEO failure is raised only
when a coordinate is emitted, so empty GEO drawings remain supported. Geometry
work/iteration limits and current-only closure retain P07/P08 contracts.

Report fields identify source SHA/size/tail, resolved options/view, current and
explicit template inventory/hashes/ignored bytes, every raw measurement and
reference with file/local index and span, group membership/reversal, detached
geometry with raw/native/display identities, each selected drawing element and
omission, and each entity's owner plus exact output line range. Footer/header lines
belong to no entity. Raw tables are read once when constructing provenance;
source fields are separate from grouped vectors, placement and Q30 corrections.
Losses include excluded drawings/template drawings, mapping and trip metadata,
comments/roll, native filters, color normalization, numeric rounding and native
failure deviations. Source bytes remain unchanged; no date/CRS is inferred.

The P05 checked new-directory publication contract also applies to `survey.dxf`:
sync/close each artifact, no-replace links and ownership anchors, report last.
Errors return no Build result and publish nothing; notification failures preserve
a completed package. Existing text/Therion artifact names and behavior remain.
See [native evidence](../internal/top/testdata/README.md#p10-native-graphics-dxf).
P11–P16 are implemented in their respective sections.

## Native VisualTopo and passage dimensions (P11)

`export <input.top> --format visualtopo --output <new-directory>` writes
`survey.tro` and schema-1 `report.json`. Optional `--angle-unit degrees|grads`
selects the native 360/400 circle (default degrees); distances always use metres.
`--template-directory <dir>` and `--close-loops` reuse P08 context and current-only
closure. There is no implicit directory discovery. P05–P10 contracts are unchanged.

`processing.PrepareVisualTopo` is a detached derived snapshot. It captures native
`GetStations(false)`: placed current heads and current references, excluding
unplaced/undefined heads, standalone splays, synthetic roots and template stations.
The actual-list input order and .NET 2 middle-pivot quicksort matter for duplicate
IDs. Comparisons retain unchecked native Int32 subtraction, and equal-key swaps
match the pinned runtime. Sorting consumes the existing geometry work budget;
invalid comparer scans return `native_failure` instead of a Go panic. Templates
can affect current placement and attached rays without appearing as output heads.

The header is `Version 5.02`. Only the earliest-index current reference produces
`Trou`/`Entree`, using native E/N divided by 1,000,000 (three decimals), altitude
integer metres truncated toward zero and the literal `SWISS`. That label does not
establish the source CRS. An empty reference comment becomes `?`; nonempty comments
are emitted verbatim, including embedded delimiters or newlines. IDs use native
display names with dots replaced by underscores; reserved empty labels and duplicate
names do not establish identity. No verified survey dates are inferred.

Data order includes sorted references for adjacency and trip fallback, although
references do not have data lines. A missing trip consults only the immediately
following sorted station's trip. Raw automatic-declination sentinel -32768 has
native effective declination zero. Parameter changes emit `Param Deca Degd/Gra
Clino Degd/Gra... Dir,Dir,Dir Inc Std`; declination is sign-reversed and converted
to truncated degrees/minutes/seconds even in the 400-circle profile. The first ID
is the endpoint opposite the placed station. Grouped placed vectors feed native
`GetAvDirection(rev=false)`, retaining native Float32 arithmetic, even for reversed
placement. Length is formatted as Double(Float32 millimetres)/1000; angle fields
retain the native Single expression and seven-significant-digit decimal formatting
before fixed two decimals. Negative zero is suppressed. The first `I`/`N` compares
head Flip with its start's Flip; the second flag is always `I`.

Passage dimensions are native derived LRUD, not independently measured wall
extents. The axis is `AngleBisector` of positive-distance head forward azimuth and
only the next sorted item when its native ID is exactly `head.num.Next()` and its
raw distance is positive. Forward azimuth uses raw head azimuth, adding 32768 for
reversed placement; the bisector narrows the delta to Int16 and shifts signed by
one. Left/right target axis minus/plus 16384; up/down target inclination plus/minus
16384. Each attached raw splay is projected to a 1000-unit direction with the
native Q30 sine table. The first largest dot product **strictly above 500000** wins;
length does not choose the ray, and equal alignment retains the first ray. The
reported dimension is wrapped Int32(Int64(raw distance)*dot/1,000,000), narrowed
to Float32 then formatted in metres with two decimals. No candidate emits zero.
With both forward directions unavailable, the Int32 sentinel wraps for left/right
but remains sentinel for up/down; this native asymmetry is preserved.

Native `Encoding.Default` is host-dependent. P11 pins the demonstrated original
.NET 2/Wine Windows-1252 best-fit/replacement profile with no BOM and exact CRLF.
Exhaustive original-runtime tables define BMP conversion and .NET 2 trimming;
supplementary characters produce two replacement bytes. This is not a claim of
matching every Windows locale. Shot comments are trimmed, CR/LF replaced by spaces
and semicolons by commas; present empty comments still emit `;;`. Unicode loss is
explicit and original decoded text remains in the report. Binary source bytes are
retained in the immutable document and identified by input hashes.

`exportvisualtopo.Build` runs bounded geometry before serialization and returns
no partial result on failure. Output is limited to 128 MiB, checked after native
encoding, with exact-limit acceptance. Geometry work/iteration and context/table
limits remain P07/P08 limits; original suppressed exceptions become explicit CLI
failures. Existing no-replace publication writes/syncs/closes `survey.tro`, then
publishes `report.json` last; notification failures preserve complete output.

The report includes source/output hashes, byte size, angular/context/closure
profile, sorted CLI input inventory with consumed/ignored bytes, original rows,
trips and references (source-local indexes, spans and raw values), current groups
and every grouped member/reversal, flattened template row lineage/group members
and table-prefix hashes. Current group nodes retain raw/native/display identity,
placement, wrapped vectors and corrections separately from source rows. Output
line/parameter changes, effective declination, passage axis, all candidate ray keys,
selected ray keys/dots/millimetres and semantic omission reasons are explicit.
Template group keys resolve through flattened template row indexes and their
source/local-row lineage. Zero output lines denote omissions. Source drawings,
mappings, roll, unrendered flags, trip comments/dates and TOP trailers have no
VisualTopo representation; losses are listed. The report does not reconstruct the
original TOP file, which remains the authoritative hashed input.

Original C#/IL, the pinned .NET sorting helper, **194 native byte goldens**,
**4105 Single-format cases**, exhaustive encoding/trim tables, hashes and the
unsuppressed native completion transcript are documented in
[fixture provenance](../internal/top/testdata/README.md#p11-native-visualtopo-and-passage-dimensions).

## Native Toporobot and dialog options (P12)

`export <input.top> --format toporobot --output <new-directory>` publishes
`survey.text` and schema-1 `report.json`. It preserves P05–P11 outputs, source
APIs, geometry/context contracts and no-replace report-last publication.
The separate `processing.PrepareToporobot` snapshot and `exporttoporobot.Build`
model/parser/writer use only the Go standard library.

| Option | Native dialog/default behavior |
| --- | --- |
| `--angle-unit degrees\|grads` | Full circle 360 by default, or 400 |
| `--code 1..100000` / `--trip 1..100000` | Default 1; pad table minimum indexes, reuse matching existing records |
| `--auto-declination` | Force automatic; wins when both overrides are present |
| `--manual-declination` | Force manual for automatic source trips; manual offset always uses -400/65536 |
| `--include-splays true\|false` | True by default; native inline ray records and strict Float32 deduplication |
| `--vertical-sections` | False by default; force vertical up/down basis and code limit zero |
| `--add-to <existing.text>` | Read-only native parse/merge/rewrite into the new output package |
| `--export-time YYYY-MM-DDTHH:MM:SS` | Fixed writer wall-clock metadata; CLI defaults to its current local wall clock |
| `--template-directory <dir>` / `--close-loops` | Explicit P08 context and current-only closure |

API callers supply `Options.WrittenAt`; its zero value is literal year 0001
metadata. With identical inputs and options, both artifacts are deterministic.
The native reference filename stem is `survey`, matching the fixed artifact
filename. Merge uses an empty cave name, as native WriteExport does. Every
existing input is hashed/inventoried and remains byte-for-byte unchanged.

Native GetStations(false) sorting, duplicate identity order, station-zero
creation, cross-series starts/terminal links, references, gaps and overwrites
are reproduced. Series zero can be modeled but is omitted by native Write;
reserved/undefined series -1 can cause explicit native failures. Trip selection
uses native global indexes, source date/name identity and first two nonempty
trimmed comment lines; matching trips share later overwritten settings.
Main lengths/directions use grouped placed vectors with Float32 operation order.
Dimensions use raw forward azimuth/inclination bisectors, radius-100 fixed-point
vectors, first dot strictly above 5000, and Int32 product wrap before /10000.
A zero-distance station with no positive successor retains existing LRUD.
Inline lines use raw distance/angles and deduplicate only when all three
Float32 differences are strictly below 0.1, without azimuth wrap.

The native custom MacEncoding table is captured exhaustively; it is not generic
Mac Roman. Unsupported UTF-16 units become question marks, including two per
supplementary character. Native UTF-16 limits/padding preserve 55-unit
reference/series names, 12-unit trip fields plus up to 53 continuation units,
and 254-unit trimmed comment budgets with 55-unit continuation wrapping.
Date formatting follows the native 1920..2019 compact interval. Field widths
are minima: large coordinates/indexes can expand columns and produce files the
native fixed-offset reader cannot reimport. Exact bytes do not imply consumer
acceptance; no destination Toporobot application validation is claimed.

Existing-file reads reproduce native fixed UTF-16 offsets, BOM detection
(UTF-8, UTF-16LE/BE, UTF-32LE), sequential station insertion ignoring station
header numbers, latest-record comment attachment, name continuations, inline
parsing and trip-name trim. UTF-32BE is rejected, matching the captured native
failure. Old Guest metadata is discarded and regenerated. The bounded portable
profile additionally rejects nonfinite existing numeric values, angular units
outside Int32, incomplete UTF-16/32 tails and resource exhaustion. Native
suppressed exceptions/partial output become explicit errors before publication.

Limits: merge input 32 MiB; output 128 MiB after encoding with exact-limit
acceptance; aggregate sparse slots 2,000,000, individual index <=1,000,000;
10,000,000 model work units (including searches, comment-copy units and LRUD
candidate work). P07/P08 geometry/input/context limits remain applicable.
Shared detached starts and report start tables are captured once; output work
uses indexed group lookup. These are acceptance ceilings, not promises of low
memory use at every maximum. Every invalid or duplicate CLI option fails.

The report separates raw source rows/trips/references and byte spans, current
member/reversal groups, flattened template row lineage and prefix hashes from
sorted geometry (raw/native/display IDs), start nodes, vectors/corrections,
LRUD axes/candidates/selection/wrap and applied/retained decisions. Every output
line has a Record range with current/template keys, existing line identities or
explicit generated/metadata ownership. Matching record keys retain contribution
order; native overwrite/normalization losses are listed. Source TOP files and
existing Toporobot files remain the authoritative hashed inputs; the report is
not an alternative lossless source archive. Drawings/mappings, roll, unused flags
and native omitted heads have explicit omissions; no CRS or survey date is inferred.

C#/IL and **325 native byte goldens**, **54 native failures**, **21 native
reader/writer cases**, exhaustive MacEncoding, 66 native numeric grammar observations and source-preserving CLI tests
are recorded in [P12 fixture provenance](../internal/top/testdata/README.md#p12-native-toporobot-and-dialog-options).
Two native-accepted nonfinite reader cases are intentionally rejected under
the finite merge profile; 15 successful roundtrips match exact native bytes.


## Native 3D DXF (P13)

`export <input.top> --format dxf3d --output <new-directory>
[--template-directory <dir>] [--close-loops]` publishes native `survey.dxf`
and schema-1 `report.json` under the P05 no-replace publication contract.
Input is complete supported TOP v3; P08 explicit input ordering, template table
prefixes, limits, symlink rejection and ignored-byte inventory apply. An explicit
empty template directory is still reported as explicit context. No automatic
neighbor discovery, graphics DXF options, scale or side selection is added.
The original ThreeD `side` parameter is unused. P05–P12 outputs/schemas remain
unchanged; the help listing gains this format.

The detached P07/P08 graphics geometry supplies native sorted stations:
current placed references/legs except the synthetic `reference:-1`, followed
by all placed template references/legs before original Framework 2.0 generic
`Array.Sort<Station>` partitioning. Compare IDs by unchecked Int32 subtraction,
including equal-ID partition order and wrap. Retain the actual template/current
boundary and current-only closure. Export a stretch only when its raw head
`dist > 0`, it has a start and its invisible bit (4) is clear. Unplaced,
undefined-from and unattached/zero splays are retained in provenance without
inventing a surface. No trip selection is applied.

Each endpoint walks its start chain while it has no cross list, zero distance
and a start. Add that resolved station, then all its attached splays in their
native order. Splay visibility does not filter this list. Prepend every point;
therefore the resulting list reverses both endpoint and cross order. Preserve
point identity independently of coordinates; do not deduplicate coincident
points. Two points yield a LINE, three a triangular 3DFACE, and four or more
use the original ordered edge-front hull traversal. Retain the previous normal
across searches and stretches, original tie branches, early duplicate-edge
termination and fourth face vertex equal to the third. Native degenerate faces
and non-watertight results are compatibility output, not inferred cave walls.

For geometry millimetres X/Y/H, internal points are `(X>>4, (-Y)>>4, (-H)>>4)`;
negation wraps Int32 before arithmetic shift. DXF coordinates are those Int32
values shifted left by four. This rounds toward negative infinity at 16 mm
spacing and preserves MinInt32 negation behavior. Differences, normal cross
products and seed additions wrap Int32; dot products widen each Int32 operand
to Int64 before multiplication and wrap their sum. Output uses decimal integers,
layer 0, UTF-8 without BOM and CRLF; AC1002 HEADER and ENTITIES sections have
no INSUNITS or CRS. No reference location offset is added. Plan sketches,
side sketches, XSections, mappings, labels, colors, raw comments/roll/trip metadata
and dates are not serialized in DXF; the loss report identifies this explicitly.

Limits are 128 MiB DXF, 1,000,000 entities, 1,000,000 allocated source-linked
points across stretches and 10,000,000 work steps for sorting, ancestor walking,
point assembly, hull scans/edge operations. Existing geometry ceilings apply
separately. The limits are inclusive; negative private test limits fail. Check
point budgets before allocation and string budgets before append. Nonterminating
native coincident/collinear hulls, geometry failures and budget exhaustion return
an error with an empty result; the CLI exits 1 without publishing a directory.
The original writer silently swallows exceptions and can leave partial files;
P13 deliberately fails before publication. Notification errors after completed
publication preserve the package, following P05.

The report identifies every input and its role/hash/consumed/ignored bytes,
current source hash/size/tail, output hash/size, options and context. All raw
measurements/references preserve source indexes, exact record spans, raw fields,
identity and group/reversal linkage. Geometry snapshots retain raw/native/display
IDs, source spans, placements, start keys, vectors and corrections. Sorted
stretches retain eligibility omissions, resolved endpoint keys, point and entity
ranges. Every allocated point links its geometry key and original position to
quantized output; every entity records its owner, exact one-based line span and
ordered point indexes. This separates repeated coordinates from source identity.
Raw comments/dates remain in the hashed original TOP, without inferred dates.

Original assembly C#/IL, 135 complete byte goldens, 24 generic-sort goldens,
24 finite FindFace identity/normal outcomes and 7 controlled native failures
establish this contract. Evidence and
reproduction commands are in [fixture provenance](../internal/top/testdata/README.md#p13-native-3d-dxf).
Consumer CAD import, a watertightness guarantee and the P14 compatibility matrix
remain outside P13.
