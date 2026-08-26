# Compiscript Grammar Audit

## Compatibility

The original grammar remains authoritative. `forStatement` preserves initializer and
condition separator ownership, while official `if (n < 60) continue;` evidence
justifies accepting one existing statement in each `if` branch without narrowing syntax.

## Float Delta

Official type rules justify reserved `float`. `FloatLiteral` precedes `IntegerLiteral`,
requires digits on both sides, preserves `42` and `floatValue`, and rejects `.5`,
`5.`, `1..2`, and `1.2.3` with diagnostics.

## Generation Evidence

The generator verifies ANTLR 4.13.2, stages argv-safe output, formats Go, and swaps
only after success; focused tests cover spaced paths and every specified failure.

## Recovery and Acceptance Evidence

`recovery_test.go` proves that a malformed declaration becomes one localized
`BadStmt` while its valid declaration siblings remain mapped, and that a malformed
print operand becomes `BadExpr` without replacing adjacent print statements. It also
parses two representative official `Especificaciones.md` examples with no diagnostics
or top-level bad statements. On 2026-08-26, two temporary generator runs each produced
eight artifacts; `diff -qr` was silent and both temporary manifests plus the checked-in
generated manifest hashed to `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010`.
The canonical manifest fixes this exact ordered eight-artifact scope:
`(cd "$tree" && for f in Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go; do sha256sum "$f"; done) | sha256sum | awk '{print $1}'`.
Final evidence reconciliation reran that algorithm for both temporary outputs and the
checked-in tree; each produced the recorded `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010` identity.
