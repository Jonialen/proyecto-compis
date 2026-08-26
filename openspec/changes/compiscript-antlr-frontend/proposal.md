# Proposal: Compiscript ANTLR Frontend

## Intent

Deliver one verifiable ANTLR frontend from Compiscript UTF-8 source to the project AST. Add `float` and correct bounded grammar inconsistencies while preserving accepted programs; breakage requires demonstrated contradiction with official instructions, examples, or rubric.

## Scope

### In Scope
- Extend the grammar with `float` and compatibility-gated corrections.
- Reproducibly generate and check in ANTLR Tool/JAR 4.13.2 Go output with atomic replacement.
- Recover parsing, map CST directly to AST, and report ordered diagnostics with half-open UTF-8 byte spans.
- Add focused generation, grammar, mapping, recovery, and boundary tests.

### Out of Scope
- Semantic analysis, CLI or IDE integration, SQL DSL, plugins, and generic compiler frameworks.
- AST/model contract changes or unrelated compiler phases.

## Capabilities

### New Capabilities
- `compiscript-antlr-frontend`: Compatible parsing, recovery, AST mapping, located diagnostics, and reproducible generation.

### Modified Capabilities
- None.

## Approach

Keep `Compiscript.g4` authoritative. Pin ANTLR Tool/JAR 4.13.2 and Go runtime module `github.com/antlr4-go/antlr/v4` v4.13.1; confine runtime, generated tree, listeners, UTF-8 indexing, and mapping to `internal/compiscript/frontend/`. Preserve current acceptance, including `for` separator ownership. Generate with verified argv into a temporary sibling; replace output only after checksum, generation, and formatting succeed. Use normal recovery and `BadStmt`/`BadExpr` only for unmappable regions.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `docs/semestre2/entrega1/Compiscript.g4` | Modified | Float and bounded corrections |
| `internal/compiscript/frontend/` | New | Generated parser and authored adapter |
| `scripts/generate-compiscript.sh` | New | Reproducible atomic generation |
| `go.mod` | Modified | Pin Go runtime module `github.com/antlr4-go/antlr/v4` v4.13.1 |
| `go.sum` | New | Record dependency checksums |
| `dependency_guard_test.go` | Modified | Add the narrow runtime allowlist |
| `.gitignore` | Modified | Generated-output inclusion |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Grammar correction rejects valid input | Medium | Preserve acceptance; require official contradiction evidence for breakage |
| Incorrect multibyte spans or recovery loss | Medium | Index original bytes and test valid siblings around errors |
| Generation corrupts checked-in output | Low | Checksum and atomic replacement tests |
| Authored work exceeds 400 lines | Medium | Keep this one child coherent; auto-chain delivery if measured over budget |

## Rollback Plan

Revert this child's grammar, frontend, generator, dependency, guard, and ignore changes together. Atomic generation and absent downstream integration keep rollback isolated from AST/model and unrelated work.

## Dependencies

- Java for regeneration, pinned ANTLR Tool/JAR 4.13.2, Go runtime module `github.com/antlr4-go/antlr/v4` v4.13.1, and existing `ast`/`model` contracts.

## Success Criteria

- [ ] Existing grammar programs remain accepted unless a documented official contradiction authorizes a correction.
- [ ] Float, recovered parsing, direct AST mapping, stable diagnostics, and UTF-8 byte spans pass focused tests.
- [ ] Regeneration is deterministic and failures leave checked-in output unchanged.
- [ ] Full repository tests pass with ANTLR imports confined to the frontend.
