## Exploration: compiscript-semantic-analysis

### Current State

The repository is a Go 1.26.1 YALex/YAPar toolkit with a table-driven syntactic pipeline. Its IDE accepts `.yal`, `.yalp`, and a text input, then reports lexer/parser results; it does not accept `.cps`, build an AST, or expose semantic diagnostics. YAPar only validates token streams and intentionally has no semantic actions or CST/AST contract, so it is not an honest reuse point for Compiscript semantics.

Entrega 1 requires an ANTLR (or equivalent) Compiscript parser, visual syntax tree, semantic diagnostics with locations, nested symbol environments, and a functional IDE. The official grammar contains the syntax surface; the instructions define the minimum semantic checks; `Especificaciones.md` provides examples that expand or sometimes contradict them.

#### Confirmed semantic policies

| Area | Confirmed policy | Grammar or implementation consequence |
|---|---|---|
| Numeric types | Support `integer` and `float`. | Extend the grammar with `float` and float literals; preserve the rest of the supplied grammar. |
| `+` | Allow numeric addition or `string + string` concatenation only; no mixed coercion. | Expression typing selects one of those two valid operand pairs. |
| Conditions and `switch` | Require boolean conditions for `if` and loops; use a conventional comparable discriminant for `switch`. | Analyze `switch` case compatibility rather than treating its discriminant as boolean. |
| `break` | Permit `break` in a loop or `switch`; keep `continue` loop-only. | Flow context tracks loop and switch nesting independently. |
| Type inference and `null` | Infer a declaration's type only from its initializer; model `null` explicitly with restricted compatibility. | Declarations without an annotation or initializer remain invalid; assignments involving `null` use the defined restricted rule. |
| Lists | Require integer indices; emit bounds diagnostics only when an out-of-bounds access is statically provable. | Do not claim runtime bounds errors from static analysis. |
| Classes | Support simple single inheritance, inherited member lookup, and constructor validation. | Predeclare classes and resolve members through at most one parent chain. |
| `try/catch` | Provide a minimal catch scope and flow model with a special exception type; do not add a throw model. | Bind the catch identifier to the special type and analyze each block without speculative exception propagation. |
| `for` syntax | Preserve the current grammar. | `variableDeclaration` or statement-level `assignment` consumes the first separator, `forStatement` consumes the separator after the condition, and `expression` permits an assignment update without a trailing separator. `for (let i: integer = 0; i < 3; i = i + 1)` is valid. Punctuation ownership remains parser-adapter coupling, not a behavioral defect. |

### Affected Areas

- `docs/semestre2/entrega1/instrucciones.md` — rubric source for acceptance evidence.
- `docs/semestre2/entrega1/Compiscript.g4` — official syntax source; requires the focused `float` extension while preserving valid `for` separator ownership.
- `docs/semestre2/entrega1/Especificaciones.md` — examples that establish expected language behavior beyond the rubric.
- `internal/compiscript/` (new) — independent AST, source spans, semantic types, symbols, diagnostics, and analysis facade should live here without importing CLI/IDE code.
- `grammar/` or `internal/compiscript/antlr/` (new) — ANTLR-generated parser and a narrow CST-to-AST adapter; generated code must not own semantic rules.
- `cmd/compiscript/` (new) — thin `.cps` CLI adapter over the frontend/analyzer facade.
- `cmd/ide/main.go` — must add a separate Compiscript endpoint rather than alter the YAPar process contract.
- `web/index.html`, `web/app.js`, `web/style.css` — later IDE adapter for `.cps`, AST visualization, diagnostics, and symbol environments; current UI is YAPar-specific.
- `testdata/compiscript/` and `internal/compiscript/**/*_test.go` (new) — rule-focused valid/invalid fixtures and deterministic output tests.

### Approaches

1. **Independent Compiscript frontend and semantic domain** — Generate/use ANTLR only behind a parser adapter that maps the CST to a project-owned AST. Analyze that AST with project-owned `Diagnostic`, `Type`, `Scope`, and `Symbol` contracts.
   - Pros: Keeps ANTLR replaceable; makes CLI and IDE simple consumers; supports later compiler phases; gives testable semantic units; avoids pretending YAPar can supply an AST.
   - Cons: Requires initial AST mapping and parser-generation setup before semantic rules are visible.
   - Effort: High, but required by the rubric.

2. **Attach semantic listeners directly to ANTLR contexts and expose them to UI/CLI** — Run listeners over generated contexts and let adapters consume those structures.
   - Pros: Smaller initial code footprint.
   - Cons: Couples all consumers and tests to generated ANTLR APIs; makes visual output, diagnostics, and later phases unstable; violates the requested boundary direction.
   - Effort: Medium initially, High in follow-up changes.

3. **Extend YAPar with Compiscript productions and semantic actions** — Reuse the existing parser stack as the Compiscript frontend.
   - Pros: Reuses existing lexer/parser CLI concepts.
   - Cons: YAPar consumes token streams and exposes acceptance/tables, not CST/AST or source-level semantic hooks; forcing actions into it expands an unrelated subsystem and does not meet the ANTLR direction honestly.
   - Effort: High and unsuitable.

### Recommendation

Adopt approach 1 with one small dependency direction:

`ANTLR generated parser -> CST-to-AST adapter -> AST -> semantic analyzer -> analysis report -> CLI / IDE adapters`

The AST owns source spans and printable-tree data; the semantic domain owns type/symbol/diagnostic data; adapters only translate input/output. Do not introduce plugin registries, visitor frameworks beyond the direct AST traversal needed, or a generic compiler platform.

The minimal cohesive Entrega 1 product is: parse `.cps`; produce a stable visual AST; analyze names, scopes, declarations, types, calls/returns, loop/return legality and reachability, lists, classes/member/constructor/`this` behavior, single inheritance, and minimal `try/catch`; emit all diagnostics and scope snapshots; surface the same report in CLI and IDE. Code generation, runtime execution, a throw model, general overloads, and dynamic array bounds are future work.

Future work may require a SQL-oriented DSL rather than raw SQL DDL/DML support. Its syntax, embedded-versus-standalone form, catalog/schema model, DDL/DML subset, type semantics, and execution strategy are unknown. It is entirely out of Entrega 1 scope: do not implement SQL/DSL nodes, catalog abstractions, plugin registries, generic AST frameworks, or extension APIs. Preserve only the dependency direction and project-owned boundaries above so a later DSL frontend/domain can avoid coupling to ANTLR-generated contexts, CLI, or IDE; its architecture must be driven by requirements when they are known.

#### Preliminary implementation units

| Unit | Must stay together / prerequisite | Acceptance boundary | Review risk |
|---|---|---|---|
| 1. Core contracts and AST | Source locations, AST node model, diagnostic/type/symbol-report contracts. Prerequisite for every later unit. | Pure Go tests construct ASTs and assert deterministic printable tree and diagnostic/symbol serialization. | Medium, 250–400 lines. |
| 2. ANTLR frontend adapter | Grammar-generation setup, parse diagnostics, CST-to-AST mapping, and source-span preservation must remain one unit. Depends on 1. | `.cps` fixtures parse into the independent AST; syntax failures retain location; no semantic imports in generated/adapter package. | High, 400–700 lines; auto-chain. |
| 3. Semantic foundations | Scope stack, declarations, duplicate/unresolved-name checks, initializer-based inference, explicit `null`, and expression typing must remain coherent. Depends on 1–2. | Table-driven valid/invalid semantic tests report exact diagnostic categories and scope snapshots. | High, 450–700 lines; auto-chain. |
| 4. Functions and flow | Function signature predeclaration (recursion), parameter/local scopes, returns, closures, loop/switch legality, minimal `try/catch`, and dead-code flow analysis. Depends on 3. | Tests cover recursion, nested capture, call arity/type, return type/context, loop/switch `break`, loop-only `continue`, catch scope/type, and unreachable statements. | High, 500–800 lines; auto-chain. |
| 5. Arrays and classes | Array typing/integer-index policy; class predeclaration, single inheritance, inherited members, constructors, and `this`. Depends on 3; class methods reuse 4. | Tests cover homogeneous arrays, valid/static-invalid indices, inherited member lookup, constructor arguments, and `this` context. | High, 500–900 lines; split arrays/classes if necessary. |
| 6. CLI and IDE adapters | A stable `AnalysisReport` from 1–5; UI presentation must not define semantics. Depends on report contract. | CLI processes `.cps`; IDE submits `.cps` and renders AST, diagnostics, and scopes; endpoint tests cover request/response. | High, 450–800 lines; auto-chain, likely separate CLI then IDE slices. |
| 7. Documentation and full acceptance fixtures | Each behavioral unit owns its focused tests; this closes rubric traceability and execution instructions. Depends on implemented behavior. | Every rubric rule has success/failure evidence and documented commands. | Medium, 250–450 lines. |

Units 1–2 should not be split further because the frontend has no usable boundary without an AST. Unit 3 should not be split into separate "symbol table" and "types" changes because resolution and types jointly define declarations and expressions. Units 4–6 are independently deliverable after the report contract. Arrays and classes can be separated only after their common class/function APIs have stabilized.

Use multiple linked SDD changes rather than one parent implementation change: a lightweight parent/tracker can preserve the product narrative, while units 1–2, 3, 4, 5, and 6–7 are independently reviewable changes. This respects the 400-line auto-chain policy, allows honest prerequisites, and permits rollback per behavior. A single change would centralize decisions but will exceed the review budget and obscure frontend/domain/adapter boundaries.

Tooling note: Mason provides `gopls` v0.23.0 and `golangci-lint-langserver`; standalone `golangci-lint` is not available on the shell PATH or Mason bin. Quality gates therefore remain `gofmt`, `go test`, `go vet`, and the editor language server unless a standalone linter is intentionally added later.

### Risks

- The frontend, semantic core, and IDE each forecast over 400 authored lines; auto-chain must split by the listed acceptance boundaries, never by file type.
- ANTLR generation/runtime availability has not been established in the current Go module, which has no dependencies; choose and verify the generation workflow before committing frontend scope.
- The focused float grammar extension and the parser adapter's ownership of `for` punctuation need syntax-fixture coverage; the existing `for` grammar is valid and must not be rewritten.
- The future SQL-oriented DSL is underspecified; anticipating its syntax, schema model, semantics, or execution with extension mechanisms now would violate KISS, while coupling project-owned boundaries to ANTLR-generated contexts, CLI, or IDE would constrain it later.
- Existing worktree changes are unrelated and must remain untouched; this exploration created only this artifact.

### Ready for Proposal

Yes — the semantic policy decisions are resolved. The next phase may create a proposal that preserves the valid `for` grammar, scopes the float grammar extension, and starts with a tracker plus the core-contract/AST change; do not begin the IDE work first.
