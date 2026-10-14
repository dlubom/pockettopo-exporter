# PocketTopo Exporter

A portable, offline CLI reproducing the native exporters of
**PocketTopo 1.372**, preserving source data and reporting transformations and
omissions. PocketTopo is a development reference, not a runtime dependency.

## Download and use

Get a ready-to-run program ZIP from [Releases](https://github.com/dlubom/pockettopo-exporter/releases/latest):
Windows, macOS and Linux each have x64 and ARM64 builds. Extract the whole ZIP;
no Go, Python, Wine or PocketTopo installation is needed. Read the
[quick start](QUICKSTART.md) for download selection and Windows PowerShell/cmd
examples. On macOS/Linux, from the unpacked folder:

```sh
./pockettopo-exporter export "My cave.top" --format svx --output "My cave-survex"
```

The examples below use the unpacked release binary and your own TOP files.
On Windows PowerShell, replace `./pockettopo-exporter` with
`.\pockettopo-exporter.exe`; in Command Prompt, use `pockettopo-exporter.exe`.
For template examples, `Surveys` is your folder of TOP files and `current.top`
is the file being exported. Every export output folder must not already exist.

The new output folder contains the export and `report.json`. Supported formats
are `text`, `therion`, `dxf` (plan/side), `visualtopo`, `toporobot`, `dxf3d`, `srv`
and `svx`. Inputs must be supported TOP v3. See [CAPABILITIES](docs/CAPABILITIES.md)
for profile limits and omissions.

Windows users can opt into a right-click menu for `.top` files by running
`pockettopo-exporter.exe windows-menu install`; remove it with
`windows-menu uninstall`. In Windows 11, use **Show more options**. Installation
is per user, needs no administrator rights and retains the default TOP app.
The program folder must stay in its registered location; re-run install after
moving it. Agents and scripts use the normal noninteractive CLI.

Project code/documentation uses [MIT](LICENSE). Packages include
[Go component notices](THIRD_PARTY_NOTICES.md). No Windows publisher certificate
or Apple Developer ID/notarization is supplied; see the quick start for launch
guidance and checksums.

## Where the project stands

The CLI reads supported **TOP v3** source data: trips, measurements, references,
three raw mappings and ordered plan/side Polygons and XSections. It preserves
original bytes, fields and offsets; unknown markers and unsupported tails remain
explicitly unparsed. It exports isolated native text with source lineage and loss
reporting. Native grouped measurement processing preserves every source row;
isolated reference placement, plan/extended geometry and optional loop closure
are available. Native Therion interchange includes DATA, plan/elevation drawings
and embedded XSections, with explicit template context and optional closure.

```sh
./pockettopo-exporter inspect "My cave.top" --json
```

Inspection emits deterministic schema-version-1 JSON with the input SHA-256,
size/version, counts, completion/support state, byte accounting and diagnostics.
Exit 0 means complete and supported; exit 1 covers malformed, unsupported,
incomplete, I/O and output failures. See [SPEC](docs/SPEC.md#complete-v3-source-document-p04d)
for the tail policy and source API. Inputs are never rewritten or scanned for
neighboring files.

Native text export uses metres, a 360-degree circle and exact UTF-8/CRLF bytes.
The output directory must be new; it contains `survey.txt` and `report.json`.

```sh
./pockettopo-exporter export "My cave.top" --format text --output "My cave-text"
```

See [the text contract](docs/SPEC.md#isolated-native-text-export-p05) for native
Update independence, its explicit failure deviation, publication and losses.

Start with [the current state and next work item](docs/PLAN.md). Work directly on
`main`, delivering one complete, reviewable behavior per increment. **P05**
adds usable native text export; **P06** adds native fixed-point consecutive and
reversed measurement groups with source lineage and average directions.

```sh
./pockettopo-exporter process "My cave.top" --json
```

Processing is isolated and precedes placement/closure. It does not average P05
text rows. See [the processing contract](docs/SPEC.md#native-grouped-measurement-processing-p06).
P07 adds isolated geometry without changing text or grouped processing output:

```sh
./pockettopo-exporter geometry "My cave.top" --json --close-loops
```

Coordinates use native millimetres (X east/right, Y south/down, H down,
D extended). Unplaced groups remain explicit; native failures and work limits
return errors. See [the geometry contract](docs/SPEC.md#isolated-native-geometry-p07).
P08 adds explicit template-directory geometry with a sorted source inventory,
hashes, ignored-template-byte accounting and per-file lineage:

```sh
./pockettopo-exporter context "Surveys/current.top" --template-directory "Surveys" --json --close-loops
```

Templates affect placement; current-only closure holds template geometry fixed.
See [the context contract](docs/SPEC.md#explicit-template-directory-context-p08)
for portable ordering, limits and native failure deviations. P09 adds native
Therion interchange (`survey.txt`) and a deterministic provenance/loss report:

```sh
./pockettopo-exporter export "My cave.top" --format therion --output "My cave-therion"
./pockettopo-exporter export "Surveys/current.top" --format therion --output "My cave-therion-context" --template-directory "Surveys" --close-loops
```

The native format contains FIX, TRIP/DATA, PLAN and ELEVATION, colored Polylines
and embedded XSection station/ray/CONNECT records. It is PocketTopo interchange,
not a standalone Therion project. See [the Therion contract](docs/SPEC.md#native-therion-interchange-p09)
for trip filtering, native ordering, strict limits and losses.
P10 adds native graphics DXF for a selected plan or extended side view:

```sh
./pockettopo-exporter export "My cave.top" --format dxf --output "My cave-dxf" --view plan --dxf-options 3582
./pockettopo-exporter export "Surveys/current.top" --format dxf --output "My cave-dxf-side" --view side --template-directory "Surveys" --close-loops
```

It publishes `survey.dxf` and `report.json`; default scale is 500 and shots,
rays and labels are enabled. All native graphics option bits, layers, GEO and
metre/feet grids are available. See [the DXF contract](docs/SPEC.md#native-graphics-dxf-p10)
for masks, provenance, native numeric behavior and explicit bounded failures.
P11 adds native VisualTopo 5.02 with direction-selected passage dimensions:

```sh
./pockettopo-exporter export "My cave.top" --format visualtopo --output "My cave-visualtopo"
./pockettopo-exporter export "Surveys/current.top" --format visualtopo --output "My cave-visualtopo-context" --angle-unit grads --template-directory "Surveys" --close-loops
```

It publishes `survey.tro` and `report.json`, with metres, degrees by default,
and a fixed native Windows-1252 profile. Dimensions select attached raw splays
by alignment, preserving native ties and zero cases. See [the VisualTopo
contract](docs/SPEC.md#native-visualtopo-and-passage-dimensions-p11) for ordering,
encoding, provenance and losses. P12 adds native Toporobot and its dialog options:

```sh
./pockettopo-exporter export "My cave.top" --format toporobot --output "My cave-toporobot" --export-time 2026-10-13T12:34:56
./pockettopo-exporter export "More readings.top" --format toporobot --output "My cave-toporobot-merge" --add-to "My cave-toporobot/survey.text" --angle-unit grads --vertical-sections
```

It publishes `survey.text` and `report.json`, reproducing native series/trip/code
selection, MacEncoding, LRUD and inline splays. Code/trip padding, declination
overrides, splays, vertical sections, merge, timestamp, explicit context and closure
are available. See [the Toporobot contract](docs/SPEC.md#native-toporobot-and-dialog-options-p12).
P13 adds native 3D DXF surfaces and centerline segments:

```sh
./pockettopo-exporter export "My cave.top" --format dxf3d --output "My cave-dxf3d"
./pockettopo-exporter export "Surveys/current.top" --format dxf3d --output "My cave-dxf3d-context" --template-directory "Surveys" --close-loops
```

It publishes `survey.dxf` and `report.json`, with native ordered `LINE`/`3DFACE`
entities, current/template geometry and 16 mm quantization. Unbounded native
hulls fail explicitly before publication. See [the 3D DXF contract](docs/SPEC.md#native-3d-dxf-p13)
for arithmetic, omissions and point/entity provenance. P14 closes the
[native capability matrix](docs/CAPABILITIES.md) with explicit profile limits,
the pinned 262-input real corpus and validated offline distributions. Release
1.0.0 extends native distribution validation to Linux, Windows and macOS on both
amd64 and arm64. Each exact-SHA CI run uploads checksummed
ZIP packages and the corpus validation report. Historical per-field prefix APIs
remain supported. P15 adds Walls measurement export:

```sh
./pockettopo-exporter export "My cave.top" --format srv --output "My cave-walls"
```

It publishes `survey.srv` and `report.json`, using P06 spatial averages for
consecutive repeated/reversed legs and retaining individual splays. Declination
is emitted initially and only at changes; editable date templates stay comments.
Original rows, corrections, native directions and output rounding remain in the
report. The measurement-only profile omits fixes and drawings. See the
[SRV contract](docs/SPEC.md#walls-srv-and-survex-svx-design-p15p16) and
[fixture provenance](internal/top/testdata/README.md#p15-walls-srv).
P16 adds Survex individual-reading export:

```sh
./pockettopo-exporter export "My cave.top" --format svx --output "My cave-survex-readings"
```

It publishes `survey.svx` and `report.json`, preserving every reading in source
order and normalizing reversed P06 members to the head direction. Survex owns
averaging and loop adjustment; literal station dots and individual splays remain.
Manual declination is initial/change-only; date and automatic/geographic
configuration are editable commented templates. The report separates originals,
normalized/emitted values, source lineage, corrections and omissions. See the
[SVX contract](docs/SPEC.md#walls-srv-and-survex-svx-design-p15p16) and
[P16 provenance](internal/top/testdata/README.md#p16-survex-svx).
P16 completes the requested scope. **P17: optional layered SVG** remains planned.

## Documentation

| Document | Purpose |
| --- | --- |
| [PLAN](docs/PLAN.md) | Authoritative delivery status, remaining roadmap and one next item |
| [SPEC](docs/SPEC.md) | Product scope, architecture, compatibility and source-preservation contracts |
| [DEVELOPMENT](docs/DEVELOPMENT.md) | Actual commands, quality gates and delivery workflow |
| [CAPABILITIES](docs/CAPABILITIES.md) | Native options, exact evidence, limitations, corpus and distribution proof |
| [HISTORY](docs/HISTORY.md) | Evidence behind the 2026-10-11 reset and access to earlier detailed contracts |
| [AGENTS](AGENTS.md) | Concise instructions for agents |
| [Native evidence](internal/top/testdata/README.md) | Fixture provenance, hashes, native probes and observed behavior |

## Development

See [DEVELOPMENT](docs/DEVELOPMENT.md) for build-from-source commands, development
tool requirements and quality gates. The application uses only Go's standard
library. To use a release package, follow the binary examples above or the
[quick start](QUICKSTART.md).

## Product direction and references

Native text comes first, then Therion interchange and graphics DXF; the native
milestone also includes VisualTopo, Toporobot and 3D DXF. Walls SRV and Survex
SVX follow native compatibility. SVG and other drawing conveniences are optional.
Historical archive comparison belongs in JKTZ, which will consume these outputs.

The primary behavioral reference is the original PocketTopo 1.372 program,
explained by the sibling [decompilation code map](../pockettopo-decompilation/analysis/ANALYSIS.txt)
and its recovered C#/IL. [Earlier JKTZ work](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/tree/master/src/jktz/pockettopo)
is prior art to verify against that reference. Source TOP files and reference
artifacts remain unchanged; successful parsing alone is not export compatibility.
