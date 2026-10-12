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
The next increment is **P10: native graphics DXF**, not implemented.
Historical per-field prefix APIs remain supported.

## Documentation

| Document | Purpose |
| --- | --- |
| [PLAN](docs/PLAN.md) | Authoritative delivery status, remaining roadmap and one next item |
| [SPEC](docs/SPEC.md) | Product scope, architecture, compatibility and source-preservation contracts |
| [DEVELOPMENT](docs/DEVELOPMENT.md) | Actual commands, quality gates and delivery workflow |
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
