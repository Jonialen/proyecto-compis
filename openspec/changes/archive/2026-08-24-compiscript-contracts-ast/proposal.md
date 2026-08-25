# Proposal: Compiscript Contracts and AST

## Intent

Establish child unit 1.1 of `compiscript-semantic-analysis`: the dependency-free contracts required before `compiscript-antlr-frontend` can map grammar contexts or later semantic passes can publish stable results. These are current requirements—not speculative extension points—because the parent specification already requires a source-located project AST, deterministic reports, diagnostics, and scope snapshots. Generic extensibility would weaken those concrete downstream contracts.

## Scope

### In Scope
- In `internal/compiscript/ast`, define half-open byte-offset spans with one-based lines/columns, concrete grammar-aligned nodes, and only minimal `Node`, statement, and expression markers needed to type children.
- In `internal/compiscript/model`, define concrete `Type`, `Symbol`, `ScopeSnapshot`, `Diagnostic`, `ASTView`, and `AnalysisReport` contracts plus small enums.
- Preserve caller-provided slice order without internal sorting; serialize all empty collections as `[]`, including nested `ASTView.Children`.
- Use pure Go RED/GREEN tests and prove regression safety with focused package tests followed by `go test ./...`.
- Deliver 300–380 authored lines as one independently mergeable stacked-to-main PR within the 400-line budget.

### Out of Scope
- ANTLR runtime/generated code, grammar or float implementation, CST mapping, parsed fixtures, facade execution, or semantic behavior.
- CLI/IDE integration, SQL DSL work, generic visitors, plugins, frameworks, and unrelated changes.
- Creating or implementing `compiscript-antlr-frontend` or any later child.

## Capabilities

### New Capabilities
- `compiscript-contracts-ast`: Source-located concrete AST and deterministic analysis-report value contracts.

### Modified Capabilities
- None.

## Approach

Create two standard-library-only packages with one-way dependency `model -> ast`. Keep node shapes concrete and grammar-aligned. Store ordered data only in slices and use a narrow JSON normalization or marshaling boundary so zero values produce stable empty arrays without adding semantic logic.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/compiscript/ast/` | New | Spans, concrete nodes, markers, tests |
| `internal/compiscript/model/` | New | Report contracts, enums, deterministic JSON tests |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Premature or incomplete contract shape | Medium | Keep fields minimal, concrete, and tied to parent requirements and grammar |
| Nested empty slices emit `null` | Medium | Test zero-value and nested JSON explicitly |

## Rollback Plan

Revert the single child PR, removing only `internal/compiscript/{ast,model}` and their tests. No runtime, generated code, existing YALex/YAPar behavior, or unrelated worktree state is changed.

## Dependencies

- Parent `compiscript-semantic-analysis` proposal, specification, design, and task 1.1.
- This change is the prerequisite for `compiscript-antlr-frontend`; that child is not created here.

## Success Criteria

- [ ] Focused package tests prove spans, node categories, preserved ordering, and `[]` for every empty collection.
- [ ] `go test ./internal/compiscript/ast ./internal/compiscript/model && go test ./...` passes.
- [ ] The PR remains within 300–380 authored lines and contains no excluded dependencies or behavior.
