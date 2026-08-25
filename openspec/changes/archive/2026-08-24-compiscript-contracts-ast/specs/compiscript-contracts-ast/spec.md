# Compiscript Contracts and AST Specification

## Purpose

Define dependency-free, source-located AST and deterministic analysis-report value contracts.

## Requirements

### Requirement: Source Locations

`Position` MUST contain byte `Offset`, one-based `Line`, and one-based `Column`. `Span` MUST contain `Start` and `End` positions and represent `[Start.Offset, End.Offset)`. Valid spans MUST have non-negative offsets, positive lines and columns, and `Start.Offset <= End.Offset`; an empty span with equal offsets MUST be valid.

#### Scenario: Multibyte half-open span
- GIVEN a UTF-8 source fragment and positions measured in bytes
- WHEN a span covers the fragment
- THEN its start is inclusive, its end is exclusive, and lines and columns are one-based

#### Scenario: Empty span
- GIVEN identical valid start and end positions
- WHEN the span is inspected
- THEN it represents a valid empty source range

### Requirement: Concrete AST Contracts

Every AST node MUST return its stored `Span` through `SourceSpan()`. The concrete categories MUST be: `Program`, `TypeRef`, `Parameter`, `SwitchCase`; statements `BlockStmt`, `VarDeclStmt`, `ConstDeclStmt`, `AssignStmt`, `FunctionDeclStmt`, `ClassDeclStmt`, `ExprStmt`, `PrintStmt`, `IfStmt`, `WhileStmt`, `DoWhileStmt`, `ForStmt`, `ForeachStmt`, `TryCatchStmt`, `SwitchStmt`, `BreakStmt`, `ContinueStmt`, `ReturnStmt`, `BadStmt`; and expressions `IdentifierExpr`, `LiteralExpr`, `ArrayExpr`, `ThisExpr`, `NewExpr`, `GroupExpr`, `UnaryExpr`, `BinaryExpr`, `TernaryExpr`, `AssignExpr`, `PropertyAssignExpr`, `CallExpr`, `IndexExpr`, `PropertyAccessExpr`, `BadExpr`.

#### Scenario: Current grammar representation
- GIVEN one construct from each current grammar alternative
- WHEN represented without parser contexts
- THEN a concrete source-located category exists for every construct

### Requirement: Minimal Node Markers

`Node` MUST expose only `SourceSpan() Span`. `Statement` and `Expression` MUST add only category markers and MUST be used solely for typed child fields. No visitor, plugin, generic node map, or semantic operation MAY be part of these contracts.

#### Scenario: Typed children
- GIVEN a compound statement or expression
- WHEN its children are constructed
- THEN statement and expression children accept only their corresponding category

### Requirement: Analysis Model Values

The model MUST provide these concrete JSON values: `Type{kind,name,element,params,result}`, `Symbol{name,kind,type,mutable,captured,span}`, `ScopeSnapshot{id,parentId,kind,span,symbols}`, `Diagnostic{code,phase,message,span}`, `ASTView{kind,label,span,children}`, and `AnalysisReport{ast,diagnostics,scopes}`. Required enum values MUST be:

| Enum | Values |
|---|---|
| `TypeKind` | `error`, `integer`, `float`, `boolean`, `string`, `null`, `list`, `class`, `function`, `exception` |
| `SymbolKind` | `variable`, `constant`, `parameter`, `function`, `class`, `field`, `method`, `catch` |
| `ScopeKind` | `global`, `class`, `function`, `block`, `catch` |
| `Phase` | `lexical`, `syntax`, `semantic` |

#### Scenario: Concrete report
- GIVEN values for every model contract and enum
- WHEN an analysis report is encoded
- THEN the documented fields and enum strings appear in the documented shape

### Requirement: Ordered Deterministic JSON

All caller-provided slices MUST preserve insertion order and MUST NOT be sorted. Repeated JSON encoding of equal values MUST be byte-identical. Every nil or empty slice MUST encode as `[]`, including AST statements, parameters, arguments, type parameters, symbols, diagnostics, scopes, and every nested `ASTView.children`.

#### Scenario: Ordered empty collections
- GIVEN caller-ordered values containing nil and empty slices at multiple AST-view depths
- WHEN encoded repeatedly
- THEN order is unchanged, outputs match byte-for-byte, and every empty collection is `[]`

### Requirement: Dependency and Behavior Boundary

`ast` and `model` MUST use only the Go standard library, with the sole project import direction `model -> ast`. They MUST NOT import ANTLR, generated, frontend, semantic, CLI, IDE, YAPar, or third-party packages, and MUST NOT parse, resolve, infer, validate, sort, or produce diagnostics.

#### Scenario: Pure contract packages
- GIVEN the package imports and exported behavior
- WHEN dependency and contract tests inspect them
- THEN only the permitted dependency exists and no semantic behavior is exposed

### Requirement: Contract and Regression Acceptance

Tests MUST use only Go's testing facilities, MUST cover the preceding contracts, and MUST preserve all existing repository behavior.

#### Scenario: Focused and full acceptance
- GIVEN strict RED/GREEN contract tests
- WHEN `go test ./internal/compiscript/ast ./internal/compiscript/model && go test ./...` runs
- THEN focused contract tests and the complete regression suite pass
