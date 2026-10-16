# Entrega 1 current compliance and remaining work

This audit describes the repository at `789da4b`, updated after the course authority (the two-member team) resolved the two source contradictions previously tracked here. The implementation and generated grammar are otherwise unchanged by this document. The supplied requirements remain authoritative: [`instrucciones.md`](instrucciones.md), [`Especificaciones.md`](Especificaciones.md), and [`Compiscript.g4`](Compiscript.g4).

## Result

The ANTLR frontend, generated Visitor traversal, project AST, semantic checks, symbol environments, CLI, browser IDE, portable generation, and test evidence are present. Both source contradictions are resolved (see below); the team has confirmed it has two members rather than three, so the roster requirement is accepted as a known, non-technical gap rather than something the implementation can close.

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
| `switch` condition semantics | Met (decided) | Team decision: `instrucciones.md` governs; `switch` requires a boolean discriminant, matching `if`/`while`/`do-while`/`for`. `Especificaciones.md`'s `switch (x)` integer example is superseded for this project. `semantic.go` now emits `SEM_CONDITION` for a non-boolean `switch` value; case values must remain type-compatible with that boolean discriminant. `flow_valid.cps`/`flow_invalid.cps` and their tests were updated accordingly. | None |
| `break` placement | Met (decided) | Team decision: `break` is valid inside loops or `switch`, matching the supplied grammar and `Especificaciones.md`. No implementation change was needed; `semantic.go` already permitted this. | None |
| Groups of three with individual commits from every member | Accepted gap, not applicable | The team confirmed it has two members, not three. [`evidence/contributions.md`](evidence/contributions.md) records only two observable contributor identities, consistent with the actual roster. | None — this is a program-level constraint, not a defect; disclose the two-member roster to the course authority rather than fabricating a third contributor. |
| Correct operation during the presentation | Not yet verifiable | Current automated gates are documented and previously passed. | Must be demonstrated in the evaluation environment on presentation day. |

## Closed audit gaps

- **Visitor:** closed by `c8df850`; CST-to-AST mapping now uses generated ANTLR `Accept`/Visitor dispatch.
- **Portable generation:** closed by `92914dc`; generation runs from the grammar directory, committed headers contain no checkout-specific absolute path, and two distinct temporary checkout paths reproduce identical bytes.
- **README:** closed by `97c394b`; the root README now explains the Compiscript pipeline, CLI, IDE, verification commands, and evidence links.
- **`switch`/`break` source contradiction:** closed after the team's decision (see matrix above); `semantic.go` and the flow fixtures/tests were updated to require a boolean `switch` discriminant while keeping `break` valid inside `switch`.
- **Method override and constructor inheritance:** this row was previously recorded as `Met` in error. The semantic analyzer rejected the canonical inheritance example in [`Especificaciones.md`](Especificaciones.md) on two counts: `SEM_INHERITED_MEMBER` flagged every redeclaration of an inherited member, which forbade method override, and `new` resolved `constructor` only on the class itself, so a subclass without its own constructor could not be instantiated with the inherited parameters. `semantic.go` now accepts an override whose signature matches the inherited method, still rejects every other inherited-name conflict, and reports `SEM_OVERRIDE` for an override with an incompatible signature; `lookupMember` resolves `constructor` through the inheritance chain. `inheritance_valid.cps` and `inheritance_invalid.cps` cover both directions and are pinned in the acceptance manifest. The earlier `classes_valid.cps` and `classes_invalid.cps` fixtures had avoided both paths, which is why the gap survived the previous audit.

## Remaining work, in priority order

1. Re-run the documented gates in the presentation environment. This is an operational verification, not a documentation or implementation change.

No other repository-visible Entrega 1 requirement gap was found in this audit. The three-member roster item is a disclosed program constraint, not tracked as outstanding work here.
