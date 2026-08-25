## Exploration: compiscript-contracts-ast

### Current State
`genanalex` is a Go 1.26.1 YALex/YAPar toolkit. There is no `internal/compiscript/` package, no module dependency, and no Compiscript runtime, CST mapping, or semantic implementation. The parent design already fixes the dependency direction: later frontend and semantic packages import `ast` and `model`; neither package imports ANTLR, YAPar, CLI, IDE, or semantic code.

Parent task 1.1 is a useful independent foundation, not a speculative framework: later parsing needs concrete source-located nodes, and later semantics/consumers need the six report value contracts. This slice can compile and be fully verified with pure Go tests before any parser exists.

### Affected Areas
- `internal/compiscript/ast/` — new concrete AST nodes and ANTLR-independent source spans.
- `internal/compiscript/model/` — new type, symbol, scope snapshot, diagnostic, AST view, and analysis report value contracts.
- `internal/compiscript/ast/*_test.go` — pure table-driven span and node-contract tests.
- `internal/compiscript/model/*_test.go` — pure JSON ordering and empty-slice serialization tests.

### Approaches
1. **Concrete AST and value-report contracts** — define one small `Node` span method, concrete statement/expression structs matching the grammar, and concrete model structs with ordered slices.
   - Pros: Meets the parent architecture, isolates ANTLR, gives later units stable construction targets, and has no runtime or third-party dependency.
   - Cons: Node fields must be deliberately named now; later frontend work must map each grammar context without changing the contract casually.
   - Effort: Medium.

2. **Generic tagged AST or visitor/plugin framework** — use a universal node map/tag structure and extensible traversal interfaces.
   - Pros: Fewer initial node declarations.
   - Cons: Loses compile-time shape, weakens tests and CST mapping, and introduces the explicitly rejected speculative abstraction surface.
   - Effort: Medium initially, High in later units.

### Recommendation
Use approach 1. Add only `internal/compiscript/{ast,model}` with concrete structs. `ast.Span` must use half-open byte offsets and one-based line/column positions. AST nodes should expose only `SourceSpan() ast.Span`; use statement/expression marker interfaces solely to type child fields, not as extension points.

`model` should contain concrete `Type`, `Symbol`, `ScopeSnapshot`, `Diagnostic`, `ASTView`, and `AnalysisReport` structs plus small enum types. Preserve source/name ordering by accepting slices in order and never storing report data in maps. JSON serialization must represent absent child/symbol/diagnostic/scope slices as `[]`, including nested `ASTView.Children`; a narrow normalization or marshaling boundary is justified so zero-value contracts cannot emit unstable `null` arrays. No sorting, name resolution, type compatibility, diagnostic production, or parser behavior belongs here.

### Acceptance Boundary
- New packages compile with no external dependencies and import only `model -> ast`.
- Pure Go table-driven tests construct representative concrete nodes and verify spans, node categories, report JSON shape, deterministic slice order, and `[]` for every empty serialized collection.
- `go test ./internal/compiscript/ast ./internal/compiscript/model && go test ./...` passes, preserving YALex/YAPar behavior.
- Excluded: ANTLR/runtime/generated code, grammar changes, CST mapping, semantic passes/rules, facade execution, CLI/IDE, SQL DSL, fixtures requiring parsing, generic visitors/plugins/frameworks, and later child units.

### Parent Relationship
This is child unit **1.1** of `compiscript-semantic-analysis`, targets `main`, and is the sole prerequisite for `compiscript-antlr-frontend` (1.2). It is independently mergeable because its tests prove a dependency-free public data boundary; rollback removes only `internal/compiscript/{ast,model}` and their tests.

### Estimated Authored Lines
300-380 lines including production contracts and their package tests. This remains within the 400-line review budget; one work-unit commit/PR is appropriate under `auto-chain` / `stacked-to-main`.

### Risks
- The parent design names the six model structs but does not freeze every AST field or enum spelling; this child must keep names minimal and grammar-aligned to avoid a follow-up breaking change.
- Custom JSON normalization must cover nested empty slices as well as top-level report slices; otherwise equivalent reports serialize as `null` versus `[]`.
- The official grammar currently lacks the parent-planned `float` syntax; this child must model `float` as a type without modifying grammar or introducing literal parsing.
- Existing worktree changes are unrelated and include deleted/untracked documentation; this child must not alter them.

### Open Questions
None. The existing parent contracts, ordering rule, and unit boundary are sufficient for proposal work.

### Ready for Proposal
Yes — create only the child proposal next. Keep its scope limited to the contracts and pure Go RED/GREEN tests above; do not pull frontend or semantic behavior into this change.
