# Repository Guidelines

## Current scope

Implementation was authorized on 2026-10-09. P02 establishes the minimal Go
skeleton and checks. Read README.md for completed scope and the next ready PBI.
Deliver one small, verified PBI per chat; split larger items before implementing.

## References

- Use `../pockettopo-decompilation/` as the primary research reference.
  Begin with `analysis/ANALYSIS.txt`, then inspect the relevant C# and IL.
- Review the earlier work at
  https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/tree/master/src/jktz/pockettopo
  for reusable ideas. Check behavior against the decompilation before adopting it.
- Keep experiments and new implementation in this project. Do not edit original
  archives or generated decompilation artifacts in the reference repository.

## Implementation

- Use English for project-authored documentation, code, comments, and messages.
- Preserve input `.top` bytes and raw field values; keep derived values separate.
- Separate parsing, the source model, processing decisions, and format exporters.
- Verify station identifiers, flags, units, rounding, grouping, and drawing
  semantics from the reference code. Document uncertainties and provenance.
- Make corrections and lossy conversions explicit; report unsupported or omitted
  data. Preserve embedded dates without assuming they prove the survey date.
- Test meaningful behavior with small fixtures and native PocketTopo exports
  where available. Successful decompilation alone does not prove compatibility.
- Use the selected Go toolchain and actual development commands in README.md.
  Native PocketTopo 1.372 exports take priority; do not invent working commands.
