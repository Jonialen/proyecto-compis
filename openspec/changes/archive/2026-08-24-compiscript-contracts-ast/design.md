# Design: Compiscript Contracts and AST

## Technical Approach

Create standard-library-only `ast` and `model` data packages. `model` imports `ast`; neither adds behavior.

Import direction: `model → ast`; future consumers import either.

## Architecture Decisions

| Choice | Rejected alternative | Rationale |
|---|---|---|
| Concrete structs/typed children | Maps, visitors, plugins | Grammar shape without extension APIs. |
| Per-node methods | Base embedding | Explicit spans; no extra category. |
| Named slices/private helper | Node marshalers; generic list | Recursive `[]`; ordered concrete API. |
| Producer span invariant | Validation methods | Data-only; frontend later guarantees it. |

## Package and File Layout

| File | Lines | Unit |
|---|---:|---|
| `internal/compiscript/ast/span.go` | 16 | GREEN |
| `internal/compiscript/ast/nodes.go` | 154 | GREEN |
| `internal/compiscript/ast/ast_test.go` | 48 | RED |
| `internal/compiscript/model/model.go` | 76 | GREEN |
| `internal/compiscript/model/json.go` | 28 | GREEN |
| `internal/compiscript/model/model_test.go` | 52 | RED |
| **Total** | **374** | **One PR** |

## Interfaces and Contracts

```go
type Position struct { Offset, Line, Column int }
type Span struct { Start, End Position }
type Node interface { SourceSpan() Span }
type Statement interface { Node; isStatement() }
type Expression interface { Node; isExpression() }
```

Every node stores `Span Span`, has value-receiver `SourceSpan` and lower-camel JSON tags. Unexported markers seal categories. Slices: `Statements`, `Expressions`, `Parameters`, `SwitchCases`.

| Node | Fields besides `Span` |
|---|---|
| `Program` | `Statements Statements` |
| `TypeRef` | `Name string; Dimensions int` |
| `Parameter` | `Name string; Type *TypeRef` |
| `SwitchCase` | `Default bool; Value Expression; Statements Statements` |
| `BlockStmt` | `Statements Statements` |
| `VarDeclStmt`, `ConstDeclStmt` | `Name string; Type *TypeRef; Initializer Expression` / `Name string; Type *TypeRef; Value Expression` |
| `AssignStmt` | `Target, Value Expression` |
| `FunctionDeclStmt` | `Name string; Parameters Parameters; Result *TypeRef; Body *BlockStmt` |
| `ClassDeclStmt` | `Name, Parent string; Members Statements` |
| `ExprStmt`, `PrintStmt` | `Expression Expression` / `Value Expression` |
| `IfStmt` | `Condition Expression; Then, Else *BlockStmt` |
| `WhileStmt`, `DoWhileStmt` | `Condition Expression; Body *BlockStmt` |
| `ForStmt` | `Init Statement; Condition, Post Expression; Body *BlockStmt` |
| `ForeachStmt` | `Name string; Iterable Expression; Body *BlockStmt` |
| `TryCatchStmt` | `Try *BlockStmt; Name string; Catch *BlockStmt` |
| `SwitchStmt` | `Value Expression; Cases SwitchCases` |
| `ReturnStmt` | `Value Expression` |
| `BreakStmt`, `ContinueStmt`, `BadStmt` | none |
| `IdentifierExpr`, `LiteralExpr` | `Name string` / `Lexeme string` |
| `ArrayExpr`, `NewExpr` | `Elements Expressions` / `ClassName string; Arguments Expressions` |
| `ThisExpr`, `BadExpr` | none |
| `GroupExpr`, `UnaryExpr` | `Expression Expression` / `Operator string; Operand Expression` |
| `BinaryExpr` | `Left Expression; Operator string; Right Expression` |
| `TernaryExpr` | `Condition, Then, Else Expression` |
| `AssignExpr` | `Target, Value Expression` |
| `PropertyAssignExpr` | `Receiver Expression; Name string; Value Expression` |
| `CallExpr`, `IndexExpr` | `Callee Expression; Arguments Expressions` / `Collection, Index Expression` |
| `PropertyAccessExpr` | `Receiver Expression; Name string` |

Enums are exact: `TypeKind={error,integer,float,boolean,string,null,list,class,function,exception}`; `SymbolKind={variable,constant,parameter,function,class,field,method,catch}`; `ScopeKind={global,class,function,block,catch}`; `Phase={lexical,syntax,semantic}`. Structs:

```go
type Type struct { Kind TypeKind; Name string; Element *Type; Params Types; Result *Type }
type Symbol struct { Name string; Kind SymbolKind; Type Type; Mutable, Captured bool; Span ast.Span }
type ScopeSnapshot struct { ID, ParentID int; Kind ScopeKind; Span ast.Span; Symbols Symbols }
type Diagnostic struct { Code string; Phase Phase; Message string; Span ast.Span }
type ASTView struct { Kind, Label string; Span ast.Span; Children ASTViews }
type AnalysisReport struct { AST ASTView; Diagnostics Diagnostics; Scopes ScopeSnapshots }
```

JSON names follow the spec; collections are named slices.

## Deterministic JSON and Data Flow

Named-slice `MarshalJSON` changes nil to zero-length, then calls `encoding/json`. Recursion covers every nested collection without node marshalers. No maps or sorting means stable order and bytes.

## Testing Strategy

1. **RED AST:** table-test positions, empty spans, `SourceSpan`, typed categories, and every node.
2. **GREEN AST:** implement two AST files; run `go test ./internal/compiscript/ast`.
3. **RED model:** table-test enums, order, repeatable bytes, and nested nil/empty arrays.
4. **GREEN model:** implement values/slice marshalers; run `go test ./internal/compiscript/model`.
5. Run `gofmt -l internal/compiscript`, focused tests, and `go test ./...`.

Test-only tables cover invalid coordinates, multibyte byte ranges, and valid empties. Production has no constructor or validator.

Dependency proof: focused `go list` imports show only `encoding/json` and `genanalex/internal/compiscript/ast`; `go list -m all` shows only `genanalex`.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration, Delivery, and Rollback

No migration. One stacked-to-main PR forecasts 374 authored lines. Rollback deletes the six files; dependencies and YALex/YAPar remain unchanged.

## Open Questions

None.
