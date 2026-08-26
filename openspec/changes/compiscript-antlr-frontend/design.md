# Design: Compiscript ANTLR Frontend

## Technical Approach

Keep `docs/semestre2/entrega1/Compiscript.g4` authoritative. An internal adapter maps UTF-8 source to established `ast` and `model.Diagnostic` contracts. Pin ANTLR Tool/JAR 4.13.2 and, independently, the available Go runtime `github.com/antlr4-go/antlr/v4 v4.13.1`. Generated/runtime imports remain confined to the frontend; semantic, CLI, IDE, AST/model, and framework work stays excluded.

## Architecture Decisions

| Decision | Rejected tradeoff | Choice and rationale |
|---|---|---|
| Boundary | Expose generated types; generic frontend | Export only `frontend.Parse`, minimizing coupling. |
| Mapping/recovery | Intermediate model; bail early | Visit CST directly while normal recovery preserves mappable siblings; bad nodes represent only unmappable regions. |
| Locations/order | Raw ANTLR indexes; post-sort | Convert against original bytes and append both listeners to one collector, preserving encounter order. |
| Generation | Local installs; in-place writes | Separately pin tool 4.13.2/runtime v4.13.1, stage, verify, and swap for reproducibility and failure isolation. |

## Grammar and Float Contract

`grammar-audit.md` records each delta, acceptance probe, and official evidence. Preserve existing acceptance and `forStatement`: declaration/assignment owns the initializer semicolon; `forStatement` owns the condition separator. Single-statement `if` remains the documented bounded correction. Other breakage requires official contradiction evidence.

Make exactly these float edits:

```antlr
baseType: 'boolean' | 'integer' | 'float' | 'string' | Identifier;
Literal: FloatLiteral | IntegerLiteral | StringLiteral;
FloatLiteral: [0-9]+ '.' [0-9]+;
IntegerLiteral: [0-9]+;
```

`literalExpr` still consumes `Literal`; `FloatLiteral` precedes the unchanged integer rule. ANTLR longest-match behavior makes `3.14` one `Literal`, while `3` remains one integer `Literal`; `.` belongs to a number only with digits on both sides. The mapper creates `ast.LiteralExpr` with unchanged token text and span for either form—no numeric conversion or AST change—and maps the type to `ast.TypeRef{Name: "float", Dimensions: n}`. Exact `float` becomes reserved, justified by the official type requirement; maximal munch keeps `floatValue` an identifier.

## Data Flow

    []byte -> sourceIndex -> lexer -> tokens -> parser -> visitor -> ast.Program
                  |           |                 |
                  +-------- listeners ----------+-> model.Diagnostics

ANTLR scalar start/inclusive-stop indexes convert through scalar-to-byte boundaries to half-open byte spans. Lines and scalar-columns are one-based; lexer faults cover one scalar, insertions are zero-width, deletions cover their token, and EOF positions clamp safely.

## File Changes

| File | Action | Description |
|---|---|---|
| `docs/semestre2/entrega1/{Compiscript.g4,grammar-audit.md}` | Modify/Create | Compatible grammar and evidence. |
| `internal/compiscript/frontend/*.go` | Create | Adapter, index, diagnostics, mapper, and tests. |
| `internal/compiscript/frontend/generated/*` | Create | Checked-in ANTLR output. |
| `scripts/generate-compiscript.sh`, `tools/antlr/antlr-4.13.2-complete.jar.sha256` | Create | Atomic pinned generation. |
| `go.mod` | Modified | Require `github.com/antlr4-go/antlr/v4 v4.13.1`. |
| `go.sum` | Create | Record module checksums. |
| `dependency_guard_test.go`, `.gitignore` | Modify | Narrow allowlist and cache exclusion. |

## Interfaces / Contracts

```go
func Parse(source []byte) (ast.Program, model.Diagnostics)
```

The visitor covers every alternative, preserves lexemes, folds operators/suffixes left-to-right, and wraps accepted single statements in synthetic blocks. The guard permits runtime v4.13.1 and required `golang.org/x/exp` only within `frontend`.

Generation verifies the 4.13.2 JAR checksum, invokes Java with an argv array (never `eval`) into a sibling temporary, runs `gofmt`, validates artifacts, and compares names and bytes. Equal output stays untouched; changed output swaps through a locked backup. Any prerequisite, checksum, generation, formatting, or swap failure restores prior bytes and removes temporaries.

## Testing Strategy

Strict TDD writes each scenario RED first. Table-driven tests prove `0.0`, `3.14`, and `let ratio: float = 3.14;` produce one `Literal` and exact `LiteralExpr` lexeme/span; `0`, `42`, and `floatValue` retain prior behavior; `.5`, `5.`, `1..2`, and `1.2.3` produce diagnostics without false float nodes. Other tests cover compatibility, every AST alternative, localized bad nodes, listener order, and multibyte/EOF spans.

Subprocess tests use spaced `t.TempDir()` paths and fake tools for argv integrity and unchanged-output failures; real-JAR determinism skips under `testing.Short()`. Run focused frontend/dependency/generation tests, then `go test ./...`, `go build ./...`, and require empty `gofmt -l .`. The runtime harness parses official examples and checks AST/diagnostics.

## Threat Matrix

| Boundary | Applicability | Safe/failure behavior and RED test |
|---|---|---|
| Documentation-like paths | N/A: no executable classification | None. |
| Git repository selection | N/A: no VCS commands | None. |
| Commit state | N/A: no commits | None. |
| Push state | N/A: no pushes | None. |
| PR commands | N/A: no PR automation | None. |
| Generation subprocess | Applicable | Preserve argv boundaries and prior output; test spaced paths and every stated failure. |

## Migration / Rollout

No migration. The original **700–1,000 authored-line forecast** is historical planning only. Final apply evidence records three PRs in the stacked-to-main architecture: PR1 **371**, PR2 **316**, and PR3 **720** authored implementation changes, totaling **1,407**. PR1 and PR2 stayed within 400 lines; PR3 itself exceeded 400 under the approved **801-line `size:exception`**. PR3 is **729** including its final nine grammar-audit lines, so the explicit cumulative implementation-plus-audit total is **1,416** (`371 + 316 + 729`); PR3 remains within its exception. Its native 3A, 3B, and 3C execution successors each stayed within 400 lines.

Generated files remain excluded only from the authored review threshold and included in complete snapshot identity and verification. Each PR carries its tests and file-scoped rollback. The rollout preserves the KISS scope: whole rollback removes frontend/generator files and reverts grammar, module, guard, and ignore changes without touching AST/model or unrelated work.

## Open Questions

None.
