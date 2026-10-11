# Documentation reset — 2026-10-11

This is a dated audit record, not another live backlog. [PLAN](PLAN.md) owns
current status and next work; [SPEC](SPEC.md) owns current contracts.

## Original intent and observed drift

The initial plan was a portable CLI reproducing PocketTopo 1.372 native exports,
including drawings and embedded cross sections. Walls/Survex came later;
historical archive comparison stayed in JKTZ. Small, complete, verified changes
were intended to make each chat manageable.

The [initial committed plan](https://github.com/dlubom/pockettopo-exporter/blob/6b53351/README.md)
defined P03 as bounded reading **and inspection**, P04 as drawings **and complete
file accounting**, and P05 as the first native text export. The implementation
retains valuable tested foundations, but those broader outcomes are unfinished.

Review of the accessible local Codex histories confirmed:

- The planning chat on 2026-10-09 selected native exports first and small tasks
  to manage context; it did not prescribe one task for every field.
- Assistant continuation prompts introduced and perpetuated per-field P04c
  increments; users then submitted those prompts as bounded implementation tasks.
- After P04c13, the user challenged advancing one Polygon at a time. The P04c14
  request explicitly replaced that sequence with a complete bounded loop and
  prohibited another count/points/color/marker sequence for subsequent Polygons.
- P04c15 introduced feature branches as an assistant workflow choice. P04c16
  continued that pattern. At P04c17 the user requested P04c16 on `main`; P04c15
  and P04c16 were integrated, then a separate P04c17 branch was created.
- The P04c17 handoff proposed a side point-count-only P04c18, repeating the
  granularity problem. This reset retires that unimplemented proposal in favor
  of P04d and records the user's current choice to work directly on `main`.

The 4,325-line README mixed durable scope, future work, historical acceptance
contracts, verification transcripts and current status. Its top said P04c17 was
complete, while the roadmap still described P04c15 as future work. Its CI section
described older group counts. These were documentation/process discrepancies,
not evidence that existing implementation or fixtures should be discarded.

No `GED`/`JED`-named Markdown file was found in the tracked file history or
current tracked tree. The reviewed historical human prompts did not identify
one. Its identity remains unresolved; no similarly named file was guessed at
or deleted. `AGENTS.md` is the actual repository instruction file and is retained.

## Git and CI observations before the reset

The history was linear. Before local consolidation:

| Ref | Commit | Relationship |
| --- | --- | --- |
| `main`, `p04c16-side-mapping` | `640d996` | P04c16 |
| `p04c15-plan-xsections` | `c860317` | Already an ancestor of main |
| `p04c17-side-marker` | `bfa7394` | Exactly one commit ahead of main |

GitHub reported no PRs. All jobs passed for
[P04c16/main](https://github.com/dlubom/pockettopo-exporter/actions/runs/38118660988)
and [P04c17](https://github.com/dlubom/pockettopo-exporter/actions/runs/38121341947).
The reset fast-forwarded local main to P04c17; it did not rewrite history or
delete old branches. Publication status is recorded in PLAN.

The original parallelization commit
[`70473aa`](https://github.com/dlubom/pockettopo-exporter/commit/70473aa)
used **four explicit mutation groups plus Gremlins**. P04c14–17 each added a
group, yielding nine mutation jobs at `bfa7394`, plus three platform checks,
mutation scheduling and CI complete. This explains the remembered four groups.
The parallel strategy and all gates remain intact.

## Preserved contracts and evidence

No parser, test, native fixture, probe or CI behavior changed in this reset.
The full earlier README is preserved in Git at the exact pre-reset commit:

- [Full detailed record, P02 through P04c17](https://github.com/dlubom/pockettopo-exporter/blob/bfa73944e3d3baad00b07e2b6dd5a8c9f9a3682b/README.md)
- [Native fixture evidence, maintained in place](../internal/top/testdata/README.md)
- [Current source models](../internal/source/)
- [Current readers and regression tests](../internal/top/)

Retrieve the exact old text locally, including all hashes, commands and earlier
prefix contracts, with:

```sh
git show bfa73944e3d3baad00b07e2b6dd5a8c9f9a3682b:README.md
```

The old README is historical evidence; its "next PBI" sections are superseded.
Use this index to consult specific contracts without loading the entire journal:

| Earlier README lines at `bfa7394` | Contents |
| --- | --- |
| 117–273 | Inkscape/Caveink smoke evidence and R01–R07 research contracts |
| 500–561 | Initial mutation/tool verification and limitations |
| 753–1496 | P03 source ID, trip, measurement and reference contracts |
| 1497–1861 | P04a/P04b mapping contracts |
| 1862–3717 | Historical P04c1–P04c13 per-field prefix contracts |
| 3718–4166 | Complete plan loops and side-mapping contracts |
| 4167–4284 | P04c17 side-marker contract and evidence |

Earlier local build logs referenced there are historical observations, not a
promise that ignored outputs still exist. Frozen native artifacts remain in
testdata with provenance; repeat new behavior checks against the current tree.

## Documentation responsibilities after the reset

| Former README content | Current home |
| --- | --- |
| Product purpose and quick start | README |
| Stable scope, architecture and compatibility | SPEC |
| Current status, next item, original roadmap and optional research status | PLAN |
| Real commands, quality gates and delivery rules | DEVELOPMENT |
| Detailed historical contracts and old verification campaigns | Git at `bfa7394`, indexed above |
| Fixture attribution, raw hashes and native commands | Existing testdata README |

Local conversation histories were used as read-only evidence. Their private
contents are not copied into this repository. Current work should be
understandable from these documents without reopening old chats.
