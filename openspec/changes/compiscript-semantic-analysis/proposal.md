# Proposal: Compiscript Semantic Analysis

## Intent

Deliver Entrega 1: parse `.cps`, present a visual project-owned AST, report located diagnostics, and expose nested symbol environments. YALex/YAPar has no Compiscript semantic contract.

## Scope

### In Scope
- ANTLR parsing, AST, semantic analysis, `AnalysisReport`, CLI, integrated IDE, and per-rule evidence.
- Types, scopes, declarations, functions/closures, flow, lists, classes, single inheritance, constructors, `this`, and minimal `try/catch`.
- Rubric traceability: IDE (15), syntax/semantics/tree (60), symbol environments (25).

### Out of Scope
- Code generation, execution, overloads, runtime bounds detection, and `throw` modeling.
- YAPar semantic actions or changes to existing YALex/YAPar behavior.
- SQL-oriented DSL syntax, APIs, catalog/schema semantics, plugins, or generic compiler frameworks.

## Capabilities

### New Capabilities
- `compiscript-semantic-analysis`: Parsing, AST visualization, semantics, symbol environments, reports, and adapters.

### Modified Capabilities
- None.

## Approach

Keep one dependency direction: `ANTLR adapter -> project-owned AST -> semantic domain -> AnalysisReport -> CLI/IDE adapters`. Generated code owns no semantics; adapters own no rules. Preserve boundaries for a future underspecified SQL DSL without extension machinery.

Policies: `integer`/`float`; numeric `+` or `string + string`; boolean loop/`if` conditions and comparable `switch`; `break` in loops/switch and loop-only `continue`; initializer-only inference with restricted `null`; integer indices and only provable bounds errors; single inheritance; minimal catch scope/special exception type; current valid `for` grammar.

Linked units: (1) contracts/AST; (2) ANTLR adapter; (3) semantic foundations; (4) functions/flow; (5) lists/classes; (6) CLI/IDE; (7) acceptance documentation. Each depends on stable preceding contracts. After parent specs, design, and tasks establish shared contracts, create linked SDD changes progressively as prerequisites stabilize. Prioritize semantics and symbol environments before IDE integration.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/compiscript/` | New | AST, semantics, reports, tests |
| `cmd/compiscript/`, `cmd/ide/main.go`, `web/` | Modified | Separate internal contracts; UI may change |
| `docs/semestre2/entrega1/`, `testdata/compiscript/` | Modified/New | Grammar, traceability, fixtures |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| ANTLR workflow unavailable | Medium | Validate tooling in the frontend unit |
| Boundary or scope expansion | High | Enforce dependency direction and explicit non-goals |
| Oversized reviews | High | Auto-chain autonomous work units above 400 authored lines |

## Rollback Plan

Revert linked units independently; remove Compiscript endpoints/UI without changing YALex/YAPar. Preserve pre-existing worktree changes.

## Dependencies

- Official grammar, rubric, policies, stable contracts/AST, then ordered units.

## Success Criteria

- [ ] Every semantic rule has valid and invalid evidence with locations.
- [ ] CLI and integrated IDE expose the same AST, diagnostics, and environment report.
- [ ] All linked changes are implemented, verified, and rubric-traced before tracker closure.
- [ ] Existing YALex/YAPar workflows remain functional.
