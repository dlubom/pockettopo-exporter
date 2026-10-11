# Repository Guidelines

## Start here

- Read `README.md`, then `docs/PLAN.md` for current status and the requested PBI.
  Use `docs/SPEC.md` for durable contracts and `docs/DEVELOPMENT.md` for commands.
- Check the working tree, branch and local/remote relationship before editing.
  Preserve unrelated changes. Work directly on `main`, as requested on
  2026-10-11; create a branch or worktree only when the user requests one.
- Deliver one cohesive, verified behavior per chat. A complete drawing reader
  or a usable export is an appropriate unit. Individual counts, coordinates,
  colors and marker bytes are implementation steps, not separate future PBIs.
- Respect an explicitly bounded user request. If a task needs subdivision,
  split at meaningful behavior boundaries and explain why. Update the single
  current plan; do not derive the next task from the last field implemented.

## References and source integrity

- Use `../pockettopo-decompilation/` as the primary research reference. Begin
  with `analysis/ANALYSIS.txt`, then inspect the relevant C# and IL.
- Review [earlier JKTZ work](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/tree/master/src/jktz/pockettopo)
  for reusable ideas; verify behavior against the native reference before reuse.
- Keep experiments and new implementation here. Preserve original TOP bytes,
  archives, frozen evidence and generated decompilation artifacts.
- Preserve raw values, source identities and offsets separately from derived
  geometry and processing decisions. Embedded dates do not prove survey dates.
- Verify identifiers, flags, units, rounding, grouping and drawing semantics.
  Record uncertainties, unsupported data and lossy conversions explicitly.

## Implementation and delivery

- Use English for project-authored documentation, code, comments and messages.
- Keep parsing, the source model, processing and exporters separate. Prefer
  small functions and normal Go tests; add no speculative framework.
- Preserve existing prefix API/error/stopping contracts unless a requested
  change explicitly revises them. New full-file work need not copy the old
  per-field API structure.
- Test meaningful behavior using independent native evidence and small fixtures.
  A synthetic fixture must actually contain the semantic case it claims to test.
- Follow the selected tools and applicable gates in `docs/DEVELOPMENT.md`.
  Keep all mutation scopes and failure controls when changing their scheduling.
- Update current contracts and `docs/PLAN.md` in place. Put fixture provenance in
  `internal/top/testdata/README.md`; do not append full session logs to README.
- Report delivered behavior, checks actually run, commit/publication state and
  one next useful increment. Respect the user's authorization for commit/push;
  repository creation history is not a standing request to publish every chat.
