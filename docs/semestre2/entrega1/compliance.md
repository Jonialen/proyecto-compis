# Entrega 1 current compliance and remaining work

This audit describes the repository at `97c394b` after `c8df850`, `92914dc`, and `97c394b`. The implementation and generated grammar are unchanged by this document. The supplied requirements remain authoritative: [`instrucciones.md`](instrucciones.md), [`Especificaciones.md`](Especificaciones.md), and [`Compiscript.g4`](Compiscript.g4).

## Result

The ANTLR frontend, generated Visitor traversal, project AST, semantic checks, symbol environments, CLI, browser IDE, portable generation, and test evidence are present. Three items remain unresolved: two supplied sources contradict each other, and Git demonstrates only two contributor identities rather than the required three.

## Requirement matrix

| Requirement | Status | Current evidence | Pending |
|---|---|---|---|
| ANTLR parser based on the supplied grammar; `.cps` input | Met | `Compiscript.g4`; `internal/compiscript/frontend/frontend.go`; `cmd/compiscript`; grammar and CLI tests | None |
| Traverse the ANTLR tree through a Listener or Visitor | Met | Generated `CompiscriptVisitor` dispatch and `Accept` calls are used by `mapper_statement.go` and `mapper_expression.go`; `frontend_test.go` guards against returning to CST type-switch dispatch | None |
| Build and visually represent a real syntax tree | Met | Project AST under `internal/compiscript/ast`; `internal/compiscript/view.go`; JSON `ast` output; browser tree rendering | None |
| Located lexical, syntax, and semantic diagnostics | Met | `frontend/diagnostics.go`; `semantic/semantic.go`; `AnalysisReport.diagnostics`; UTF-8 and recovery evidence in [`acceptance.md`](acceptance.md) | None |
| Arithmetic, logical, comparison, assignment, constant, and semantic-expression type checks | Met | `semantic.go`; operator, type, name, and acceptance fixtures cover valid and invalid polarity | None |
| Homogeneous lists, nested lists, integer indices, and provable bounds | Met | `semantic/collections_test.go`; `collections_valid.cps`; `collections_invalid.cps` | None |
| Local/global resolution, undeclared names, same-scope duplicates, nested access, and shadowing | Met | `semantic_test.go`; `valid.cps`; `invalid.cps` | None |
| Global, function, class, block, and catch symbol environments | Met | Ordered `ScopeSnapshot` output; semantic scope construction; CLI and IDE expose `scopes` | None |
| Positional function arguments, return types, recursion, nested functions, closures, and duplicate declarations | Met | `functions_flow_test.go`; `functions_valid.cps`; `functions_invalid.cps`; captured symbols are marked | None |
| Boolean `if`, `while`, `do-while`, and `for` conditions | Met | `semantic.go` condition checks; flow fixtures | None |
| `continue` only in loops; `return` only in functions; unreachable code detection | Met | Transfer-depth and termination checks in `semantic.go`; flow and foreach fixtures | None |
| Classes, members, constructors, inheritance, and `this` | Met | `classes_exceptions_test.go`; `classes_valid.cps`; `classes_invalid.cps` | None |
| Successful and failing tests for the semantic rules | Met | Focused semantic tests plus the hash-pinned acceptance manifest and requirement-evidence gate documented in [`acceptance.md`](acceptance.md) | None |
| IDE can write/load and compile Compiscript and show tree, diagnostics, and environments | Met | `web/`; `POST /api/compiscript/analyze`; IDE handler and acceptance tests; [`evidence/compiscript-ide.md`](evidence/compiscript-ide.md) | None |
| Execution and architecture documentation | Met | Compiscript pipeline and commands in the [repository README](../../../README.md); CLI, IDE, acceptance, grammar-audit, and contribution evidence in this directory | None |
| `switch` condition semantics | Conflict | `instrucciones.md` requires a boolean condition. `Especificaciones.md` demonstrates `switch (x)` with integer cases. Current code accepts a comparable discriminant and requires compatible case values. | A course authority must choose the governing interpretation; then implementation/tests may need to change. Do not conceal either source. |
| `break` placement | Conflict | `instrucciones.md` permits `break` only inside loops. The supplied grammar and `Especificaciones.md` include `switch`; current code permits `break` in loops or switches. | A course authority must decide whether `break` in `switch` is valid; then implementation/tests may need to change. |
| Groups of three with individual commits from every member | Not demonstrated | [`evidence/contributions.md`](evidence/contributions.md) records 80 commits at audited baseline `97c394b` and only two observable contributor identities after alias reconciliation. | A real third member must contribute identifiable individual commits if the roster requirement applies. Documentation cannot manufacture this evidence. |
| Correct operation during the presentation | Not yet verifiable | Current automated gates are documented and previously passed. | Must be demonstrated in the evaluation environment on presentation day. |

## Closed audit gaps

- **Visitor:** closed by `c8df850`; CST-to-AST mapping now uses generated ANTLR `Accept`/Visitor dispatch.
- **Portable generation:** closed by `92914dc`; generation runs from the grammar directory, committed headers contain no checkout-specific absolute path, and two distinct temporary checkout paths reproduce identical bytes.
- **README:** closed by `97c394b`; the root README now explains the Compiscript pipeline, CLI, IDE, verification commands, and evidence links.

## Remaining work, in priority order

1. Obtain one explicit course decision covering the two source contradictions before changing semantics.
2. Resolve the three-member contribution requirement with real repository history; do not add unsupported attribution.
3. Re-run the documented gates in the presentation environment. This is an operational verification, not a documentation or implementation change.

No other repository-visible Entrega 1 requirement gap was found in this audit.
