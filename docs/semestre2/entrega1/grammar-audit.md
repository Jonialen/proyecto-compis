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
Generation runs from the grammar directory with the grammar basename, so generated
headers contain `Compiscript.g4` rather than an absolute checkout path. A focused
test regenerates from two distinct temporary checkout paths and compares both outputs
with the committed artifacts byte for byte.

## Recovery and Acceptance Evidence

`recovery_test.go` proves that a malformed declaration becomes one localized
`BadStmt` while its valid declaration siblings remain mapped, and that a malformed
print operand becomes `BadExpr` without replacing adjacent print statements. It also
parses two representative official `Especificaciones.md` examples with no diagnostics
or top-level bad statements. The canonical manifest fixes this exact ordered
eight-artifact scope:
`(cd "$tree" && for f in Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go; do sha256sum "$f"; done) | sha256sum | awk '{print $1}'`.
After portable regeneration changed only checkout-specific generated headers, the
current checked-in identity is
`79d82b69a89a6ef6b5d21a1eaf9ec9b5699b9db3048c2deaa244bbdc344fe2bb`, matching
the gate in `acceptance.md`.
