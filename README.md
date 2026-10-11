# PocketTopo Exporter

A portable, offline Go CLI being developed to reproduce the native exporters of
**PocketTopo 1.372**, preserving source data and reporting transformations and
omissions. PocketTopo is a development reference, not a runtime dependency.

## Where the project stands

The tested foundation exists: immutable source records, bounded TOP v3 readers,
native reference fixtures, and CI on Linux, Windows and macOS. Internal readers
reach complete supported plan drawings, the side mapping and its first marker.
Side drawing payloads and complete-file accounting remain unfinished.

**No exporter is implemented yet.** The CLI currently accepts only `--help`,
`-h` and `--version`; internal parser APIs are not exposed through the CLI.

Start with [the current state and next work item](docs/PLAN.md). Work directly on
`main`, delivering one complete, reviewable behavior per increment. The next
recommended increment is **P04d: complete v3 source reading and inspection**,
followed by **P05: the first native text export**. Single-field P04c tasks are
historical contracts, not the pattern for future work.

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
its parallel CI groups. There are no application module dependencies.

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
