# PocketTopo Exporter

A portable, offline Go CLI being developed to reproduce the native exporters of
**PocketTopo 1.372**, preserving source data and reporting transformations and
omissions. PocketTopo is a development reference, not a runtime dependency.

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
go run ./cmd/pockettopo-exporter inspect internal/top/testdata/api-drawings.top --json
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
go run ./cmd/pockettopo-exporter export internal/top/testdata/api-drawings.top --format text --output build/drawings-text
```

See [the text contract](docs/SPEC.md#isolated-native-text-export-p05) for native
Update independence, its explicit failure deviation, publication and losses.

Start with [the current state and next work item](docs/PLAN.md). Work directly on
`main`, delivering one complete, reviewable behavior per increment. **P05**
adds usable native text export; **P06** adds native fixed-point consecutive and
reversed measurement groups with source lineage and average directions.

```sh
go run ./cmd/pockettopo-exporter process internal/processing/testdata/repeats.top --json
```

Processing is isolated and precedes placement/closure. It does not average P05
text rows. See [the processing contract](docs/SPEC.md#native-grouped-measurement-processing-p06).
P07 adds isolated geometry without changing text or grouped processing output:

```sh
go run ./cmd/pockettopo-exporter geometry internal/processing/testdata/geometry/intersecting.top --json --close-loops
```

Coordinates use native millimetres (X east/right, Y south/down, H down,
D extended). Unplaced groups remain explicit; native failures and work limits
return errors. See [the geometry contract](docs/SPEC.md#isolated-native-geometry-p07).
P08 adds explicit template-directory geometry with a sorted source inventory,
hashes, ignored-template-byte accounting and per-file lineage:

```sh
go run ./cmd/pockettopo-exporter context internal/processing/testdata/context/crossfile/current.top --template-directory internal/processing/testdata/context/crossfile --json --close-loops
```

Templates affect placement; current-only closure holds template geometry fixed.
See [the context contract](docs/SPEC.md#explicit-template-directory-context-p08)
for portable ordering, limits and native failure deviations. P09 adds native
Therion interchange (`survey.txt`) and a deterministic provenance/loss report:

```sh
go run ./cmd/pockettopo-exporter export internal/exporttherion/testdata/features.top --format therion --output build/therion
go run ./cmd/pockettopo-exporter export internal/exporttherion/testdata/template-rays/current.top --format therion --output build/therion-context --template-directory internal/exporttherion/testdata/template-rays --close-loops
```

The native format contains FIX, TRIP/DATA, PLAN and ELEVATION, colored Polylines
and embedded XSection station/ray/CONNECT records. It is PocketTopo interchange,
not a standalone Therion project. See [the Therion contract](docs/SPEC.md#native-therion-interchange-p09)
for trip filtering, native ordering, strict limits and losses.
P10 adds native graphics DXF for a selected plan or extended side view:

```sh
go run ./cmd/pockettopo-exporter export internal/exportdxf/testdata/features.top --format dxf --output build/dxf --view plan --dxf-options 3582
go run ./cmd/pockettopo-exporter export internal/exportdxf/testdata/template-rays/current.top --format dxf --output build/dxf-side --view side --template-directory internal/exportdxf/testdata/template-rays --close-loops
```

It publishes `survey.dxf` and `report.json`; default scale is 500 and shots,
rays and labels are enabled. All native graphics option bits, layers, GEO and
metre/feet grids are available. See [the DXF contract](docs/SPEC.md#native-graphics-dxf-p10)
for masks, provenance, native numeric behavior and explicit bounded failures.
P11 adds native VisualTopo 5.02 with direction-selected passage dimensions:

```sh
go run ./cmd/pockettopo-exporter export internal/exportvisualtopo/testdata/dimensions.top --format visualtopo --output build/visualtopo
go run ./cmd/pockettopo-exporter export internal/exportvisualtopo/testdata/template-rays/current.top --format visualtopo --output build/visualtopo-context --angle-unit grads --template-directory internal/exportvisualtopo/testdata/template-rays --close-loops
```

It publishes `survey.tro` and `report.json`, with metres, degrees by default,
and a fixed native Windows-1252 profile. Dimensions select attached raw splays
by alignment, preserving native ties and zero cases. See [the VisualTopo
contract](docs/SPEC.md#native-visualtopo-and-passage-dimensions-p11) for ordering,
encoding, provenance and losses. P12 adds native Toporobot and its dialog options:

```sh
go run ./cmd/pockettopo-exporter export internal/exporttoporobot/testdata/topology.top --format toporobot --output build/toporobot --export-time 2026-10-13T12:34:56
go run ./cmd/pockettopo-exporter export internal/exporttoporobot/testdata/comments.top --format toporobot --output build/toporobot-merge --add-to build/toporobot/survey.text --angle-unit grads --vertical-sections
```

It publishes `survey.text` and `report.json`, reproducing native series/trip/code
selection, MacEncoding, LRUD and inline splays. Code/trip padding, declination
overrides, splays, vertical sections, merge, timestamp, explicit context and closure
are available. See [the Toporobot contract](docs/SPEC.md#native-toporobot-and-dialog-options-p12).
P13 adds native 3D DXF surfaces and centerline segments:

```sh
go run ./cmd/pockettopo-exporter export internal/exportdxf3d/testdata/fallback.top --format dxf3d --output build/dxf3d
go run ./cmd/pockettopo-exporter export internal/exportdxf/testdata/template-rays/current.top --format dxf3d --output build/dxf3d-context --template-directory internal/exportdxf/testdata/template-rays --close-loops
```

It publishes `survey.dxf` and `report.json`, with native ordered `LINE`/`3DFACE`
entities, current/template geometry and 16 mm quantization. Unbounded native
hulls fail explicitly before publication. See [the 3D DXF contract](docs/SPEC.md#native-3d-dxf-p13)
for arithmetic, omissions and point/entity provenance. P14 closes the
[native capability matrix](docs/CAPABILITIES.md) with explicit profile limits,
the pinned 262-input real corpus and validated offline distributions for Linux
amd64, Windows amd64 and macOS arm64. Each exact-SHA CI run uploads checksummed
ZIP packages and the corpus validation report. Historical per-field prefix APIs
remain supported. P15 adds a source-preserving Walls SRV conversion:

```sh
go run ./cmd/pockettopo-exporter export internal/exportwalls/testdata/features.top --format walls --output build/walls
```

It publishes `survey.srv` and `report.json`, retaining source-order measurements,
repeats, reverse shots, anonymous splays, stored declination and complete reference
fixes. Station names are mapped without collisions to the Walls eight-character
limit; comments, raw values, byte spans and omissions remain traceable. Dates,
CRS, LRUD and drawings are not inferred. This is a conversion profile, not a native
PocketTopo exporter. See [the Walls contract](docs/SPEC.md#walls-srv-conversion-p15)
for strict failures and independent consumer validation. **P16: Survex SVX** is
planned; no SVX exporter is implemented by P15.

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

## Build and check

Use **Go 1.26.3** and Bash (Git Bash on Windows). From the repository root:

```sh
bash scripts/tools.sh       # install pinned development tools; requires network
bash scripts/check.sh       # formatting, static checks, tests, coverage, build
go run ./cmd/pockettopo-exporter --help
go run ./cmd/pockettopo-exporter --version
```

The executable is written to `build/pockettopo-exporter` (`.exe` on Windows).
See [DEVELOPMENT](docs/DEVELOPMENT.md) for the complete mutation campaign and
its parallel CI groups and bounded local runner (`bash scripts/mutations-parallel.sh`).
There are no application module dependencies.

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
