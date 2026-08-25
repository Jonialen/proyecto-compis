# Design: Compiscript Semantic Analysis

## Technical Approach

Add one facade, `compiscript.Analyze(source)`, over a narrow ANTLR frontend and ordered semantic passes. The import direction is one-way:

`cmd/compiscript`, `cmd/ide` → `internal/compiscript` → `{frontend, semantic}` → `{ast, model}`; `frontend` alone imports `frontend/generated` and the ANTLR runtime. YALex/YAPar remain separate.

| Package | Responsibility |
|---|---|
| `internal/compiscript/ast` | Concrete nodes and ANTLR-independent half-open spans (byte offsets; one-based lines/columns). |
| `internal/compiscript/model` | Types, symbols, diagnostics, snapshots, AST view, report. |
| `internal/compiscript/frontend` | Error listeners and direct CST-to-AST mapping; malformed contexts become located `BadExpr`/`BadStmt`. |
| `internal/compiscript/semantic` | Concrete analyzer and passes; no adapter imports. |
| `internal/compiscript` | Shared facade that normalizes the final report. |

## Architecture Decisions

| Choice | Rejected alternative | Rationale |
|---|---|---|
| Project AST plus direct mapping | Analyze ANTLR contexts/listeners | Stable domain, tests, and consumers without a generic visitor framework. |
| ANTLR 4.13.2 tool/runtime, generated Go checked in | Runtime generation; force YAPar reuse | Meets the grammar requirement and keeps builds Java-free; YAPar has no CST/source-semantic contract. |
| Concrete structs and one facade | Plugin/compiler framework | KISS; the future SQL DSL only requires uncoupled boundaries, not SQL nodes, catalogs, or extension APIs. |
| Stable JSON-shaped report | Consumer-specific results | CLI and IDE cannot diverge semantically. |

`docs/semestre2/entrega1/Compiscript.g4` remains authoritative, adding only `float` syntax. `scripts/generate-compiscript.sh` fetches/verifies the pinned JAR, generates in a temporary directory, formats, then replaces `internal/compiscript/frontend/generated/`. `go generate` invokes it; generated files are committed. `go.mod` pins `github.com/antlr4-go/antlr/v4 v4.13.2`. `dependency_guard_test.go` gains a table-tested allowlist for that runtime and pinned `golang.org/x/exp` while rejecting every other external/GUI package.

## Data Flow

`source → lexer/parser diagnostics + CST → AST → semantic passes → normalized AnalysisReport → CLI/IDE`

Pass order is fixed: (1) predeclare functions/classes/member signatures, establish deterministic preorder scope IDs, resolve parent classes/cycles; (2) declare variables/constants/parameters in source order, resolve nearest names, closures, `this`, and catch-only exception bindings; (3) type expressions, assignments, calls/returns, homogeneous-per-level ragged lists, contextual empty lists, integer/provable bounds checks, constructors and inherited lookup/redeclaration; (4) validate boolean conditions, compatible unique non-fallthrough switch cases, transfer contexts, unreachable statements, and all-path non-null returns. `ErrorType` propagates through dependent operations without another diagnostic; independent siblings continue.

Lexer, syntax, then semantic diagnostics are sorted by `(startOffset,endOffset,phase,code,message)`; AST children, scopes, and symbols use source/name order, never map iteration. Recovered bad nodes yield `ErrorType`, preserving valid-region analysis. Empty collections serialize as `[]` for stable runs.

## File Changes

| Path | Action |
|---|---|
| `internal/compiscript/{ast,model,frontend,semantic}/`, `internal/compiscript/analyze.go` | Create domain, generated frontend, passes, facade, tests. |
| `scripts/generate-compiscript.sh`, `cmd/compiscript/{main.go,main_test.go}`, `testdata/compiscript/` | Create reproducible generation, thin JSON CLI, paired fixtures. |
| `docs/semestre2/entrega1/Compiscript.g4`, `go.mod`, `go.sum`, `dependency_guard_test.go` | Modify grammar/dependency policy. |
| `cmd/ide/{main.go,main_test.go}`, `web/{index.html,app.js,style.css}` | Add `POST /api/compiscript/analyze` with `{source}`; render the unchanged report. Keep `/api/process` intact. |

## Interfaces / Contracts

```go
type Type struct { Kind TypeKind; Name string; Element *Type; Params []Type; Result *Type }
type Symbol struct { Name string; Kind SymbolKind; Type Type; Mutable, Captured bool; Span ast.Span }
type ScopeSnapshot struct { ID, ParentID int; Kind ScopeKind; Span ast.Span; Symbols []Symbol }
type Diagnostic struct { Code string; Phase Phase; Message string; Span ast.Span }
type ASTView struct { Kind, Label string; Span ast.Span; Children []ASTView }
type AnalysisReport struct { AST ASTView; Diagnostics []Diagnostic; Scopes []ScopeSnapshot }
```

AST nodes are concrete structs satisfying only `Node.SourceSpan() ast.Span`.

## Testing Strategy

Strict TDD uses table-driven package tests and `testdata/compiscript/{valid,invalid}/<rule>.cps`: every rule gets located passing/failing evidence. Golden JSON proves AST/report ordering, recovery stability, and CLI/IDE equivalence; handler tests cover method/body/status. Regeneration tests compare checked-in output. `go test ./...` protects all YALex/YAPar tests and the dependency guard.

## Threat Matrix

| Boundary | Applicability | Safe/failure behavior and RED test |
|---|---|---|
| Documentation-like paths | N/A—no executable classification. | None. |
| Git repository selection | N/A—no VCS automation. | None. |
| Commit state | N/A—no commits. | None. |
| Push state | N/A—no pushes. | None. |
| PR commands | N/A—no PR automation. | None. |
| ANTLR subprocess | Applicable. | Fixed argv/root paths, checksum, temporary output; spaces work, missing Java/hash mismatch fails without replacing generated files. RED integration tests cover all three. |

## Migration / Rollout

No data migration. Later planning should preserve progressive prerequisites and auto-chain each over-budget unit: contracts/AST → frontend → names/types → functions/flow → lists → classes → CLI → IDE → acceptance evidence. Each slice includes its tests and can revert without touching predecessors; no child change is created by this design. Generated volume is reported separately from the 400 authored-line budget. Apply only scoped paths and never reset or overwrite unrelated worktree changes. Roll back by removing the Compiscript route/UI/command and new packages; existing YALex/YAPar behavior remains.

## Open Questions

None.
