## Exploration: compiscript-antlr-frontend

### Current State
`compiscript-contracts-ast` is verified and archived: `ast` supplies concrete, source-located nodes (including `BadStmt` and `BadExpr`), while `model` supplies lexical and syntax diagnostic values. No frontend, generator script, ANTLR dependency, or generated tree exists; `go.mod` has no requirements and the dependency guard currently rejects every external package.

The official `Compiscript.g4` is the syntax authority. It lacks `float`; its current `for` rule is intentionally valid because its initializer alternatives own their separator and the rule owns the condition separator. The frontend must preserve that grammar shape, add only float syntax, collect ANTLR recovery diagnostics, and directly map recovered CST regions to the established AST without semantic work.

### Affected Areas
- `docs/semestre2/entrega1/Compiscript.g4` — canonical grammar; add `float` type and float literal/token handling while retaining the current `for` rule.
- `internal/compiscript/frontend/` — the sole home for ANTLR runtime use, generated parser tree, located error listeners, byte-offset conversion, direct CST-to-AST mapper, and focused tests.
- `scripts/generate-compiscript.sh` — reproducibly acquire/check ANTLR 4.13.2, invoke Java with argv-safe paths, generate to a temporary directory, format, and atomically replace generated output only after success.
- `go.mod`, `go.sum`, `dependency_guard_test.go` — pin the Go ANTLR 4.13.2 runtime and allow only its required dependency closure; retain rejection of unrelated external/GUI packages.
- `.gitignore` — ensure the checked-in generated frontend tree is not ignored; generation cache and temporary material stay outside the repository tree.

### Approaches
1. **Narrow generated ANTLR adapter** — keep the canonical grammar under `docs/`, generate checked-in Go code only under `internal/compiscript/frontend/generated`, and implement a concrete mapper there using source-index helpers and ANTLR's normal recovery.
   - Pros: Java is needed only for regeneration; ANTLR cannot leak into AST/model/semantic packages; direct mapping matches the verified node contracts; generated output can be reviewed separately from authored code.
   - Cons: The mapper must cover the grammar's concrete alternatives and account for recovered/missing tokens.
   - Effort: Medium

2. **Expose ANTLR CST/listener contexts to later phases** — parse now and defer AST conversion to semantics.
   - Pros: Less authored frontend code initially.
   - Cons: Violates the project-owned AST boundary, couples future consumers to generated APIs, and makes recovery/report stability harder to prove.
   - Effort: High overall

### Recommendation
Use approach 1 as one focused stacked child. Keep the canonical grammar in `docs/`; confine the ANTLR runtime imports, generated files, generation directive, listeners, source indexing, and mapper to `internal/compiscript/frontend`. Use the default ANTLR recovery strategy, append located lexical/syntax diagnostics in encounter order, and emit `BadStmt`/`BadExpr` only for unrecoverable mapped regions so valid siblings remain available. Convert ANTLR line/column/token boundaries through a private UTF-8 byte source index to satisfy the AST's half-open byte spans and make repeated parsing deterministic.

The generator should use fixed argument slices (never shell interpolation), a pinned 4.13.2 JAR URL and SHA-256, a temporary sibling output directory, and rename replacement only after download, checksum, generation, and formatting succeed. Tests should inject command/download paths so they can prove spaces, missing Java, and checksum mismatch without network dependence or mutation of an existing generated tree. Generated Go is committed but excluded from the 400 authored-line review measurement; its identity remains verified by regeneration output comparison.

### Risks
- Mapping every current grammar alternative plus recovery paths may exceed the 350–400 authored-line forecast; preserve one coherent frontend work unit and auto-chain only if measured authored code crosses the budget.
- ANTLR token columns are not AST byte offsets; failing to derive spans from the original UTF-8 source would violate the verified location contract.
- A grammar change that "fixes" `for` punctuation would reject the required example; retain the existing separator ownership exactly.
- Download, Java, or checksum failure must leave the checked-in generated tree byte-for-byte untouched; test this with temporary directories and dependency-injected subprocess boundaries.

### Ready for Proposal
Yes — propose the bounded frontend unit only: grammar float extension, reproducible 4.13.2 generation, checked-in generated tree, direct CST-to-AST mapping, located recovery diagnostics, and tests. Exclude semantic analysis, facade/CLI/IDE work, SQL DSL, plugins, and generic compiler abstractions.
