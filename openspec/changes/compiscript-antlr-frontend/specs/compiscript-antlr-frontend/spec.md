# Compiscript ANTLR Frontend Specification

## Purpose

Define compatible UTF-8 Compiscript parsing to the established AST, recovery diagnostics, and reproducible ANTLR generation.

## Requirements

### Requirement: Grammar compatibility gate

The frontend MUST accept every program accepted by the current authoritative grammar, including existing `for` separator ownership. A breaking correction MUST NOT be made unless documented evidence identifies a contradiction with official instructions, examples, or rubric.

#### Scenario: Existing program remains valid
- GIVEN source accepted before this change
- WHEN it is parsed
- THEN no new lexical or syntax rejection occurs

#### Scenario: Breaking correction is proposed
- GIVEN accepted syntax without documented official contradiction
- WHEN a correction would reject it
- THEN compatibility is preserved

### Requirement: Float syntax

The grammar MUST recognize `float` as a type and a floating-point literal as one or more digits, `.`, and one or more digits. Integer literal behavior MUST remain unchanged.

#### Scenario: Float declaration parses
- GIVEN `let ratio: float = 3.14;`
- WHEN it is parsed
- THEN the declaration and float literal are represented in the AST

### Requirement: Complete direct AST mapping

Every current valid grammar alternative MUST map directly from CST to its established AST node, preserving lexemes and structure without an intermediate generic model.

#### Scenario: Current alternatives map
- GIVEN examples covering every statement and expression alternative
- WHEN they are parsed
- THEN each construct has its established AST shape and no bad node

### Requirement: Localized recovery

Malformed input MUST preserve the maximum mappable AST and valid siblings. `BadStmt` or `BadExpr` MUST appear only for unmappable regions.

#### Scenario: Valid siblings surround an error
- GIVEN valid constructs surrounding one malformed construct
- WHEN recovery completes
- THEN valid siblings and mappable recovered nodes remain established nodes, while only unmappable regions become bad nodes

### Requirement: Deterministic located diagnostics

Lexical and syntax diagnostics MUST be returned deterministically in encounter order. Every AST node and diagnostic span MUST use half-open UTF-8 byte offsets and one-based lines and columns.

#### Scenario: Mixed diagnostics are stable
- GIVEN multiple lexical and syntax errors
- WHEN parsed repeatedly
- THEN diagnostic phase, message, span, and encounter order remain identical

#### Scenario: Multibyte source is located
- GIVEN an error after a multibyte character
- WHEN located
- THEN byte offsets bound `[start,end)` in the original bytes and line and column values start at one

### Requirement: Reproducible atomic generation

Generation MUST pin ANTLR 4.13.2 and its checksum, use argv-safe invocation, format generated Go, and check in deterministic output. Existing output MUST be replaced only after all steps succeed.

#### Scenario: Regeneration is reproducible
- GIVEN pinned inputs and successful tools
- WHEN generation runs twice
- THEN both runs produce byte-identical checked-in output

#### Scenario: Paths contain spaces
- GIVEN input or output paths containing spaces
- WHEN generation succeeds
- THEN each path remains intact and generation succeeds

#### Scenario: Generation prerequisite fails
- GIVEN missing Java, checksum mismatch, generation failure, or formatting failure
- WHEN existing output is regenerated
- THEN the command fails and that output remains byte-for-byte unchanged

### Requirement: Narrow frontend boundary

ANTLR runtime and generated-code dependencies MUST remain confined to the frontend, and the dependency guard MUST allow only the required narrow closure. This change MUST NOT add semantic analysis, CLI, IDE, SQL DSL, plugins, AST/model changes, or generic compiler abstractions.

#### Scenario: Dependency boundary is checked
- GIVEN repository package imports
- WHEN the dependency guard runs
- THEN required frontend dependencies pass and unrelated external or frontend-leaking imports fail

### Requirement: Regression acceptance

Acceptance MUST require focused grammar, mapping, recovery, span, generation, and dependency checks plus the full repository regression suite.

#### Scenario: Change is accepted
- GIVEN passing focused checks
- WHEN full repository tests and builds run
- THEN they pass without regressions before acceptance
