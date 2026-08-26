# Tasks: Compiscript ANTLR Frontend

## Review Workload Forecast

| Field | Value |
|---|---|
| Historical authored forecast (planning only; not final actual) | 700–1,000 |
| Historical generated-volume forecast (planning only; not final actual) | ~5,000–10,000 checked-in lines; threshold-excluded, with complete snapshot identity preserved |
| Final actual authored implementation | PR1 371 + PR2 316 + PR3 720 = 1,407 |
| Final actual authored total including audit | PR1 371 + PR2 316 + PR3 729 (including nine PR3 grammar-audit lines) = 1,416 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 |
| Delivery / chain strategy | auto-chain / stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

PR1 and PR2 remain ≤400. PR3 is **729 including nine audit lines**, within the approved **801-line `size:exception` review boundary**; native successors 3A/3B/3C remain ≤400. All tasks are complete. Next: `sdd-verify` only; no 850-line execution objective.

Baseline: passing `go test ./...` and `go build ./...`. Run `gofmt -l .` informationally; permit only documented pre-existing `internal/yapar/parser_test.go`. Slice 3 changed Go files require empty scoped `gofmt -l`. Preserve unrelated changes, PR1→PR2→PR3 order, and RED-before-GREEN evidence.

| Unit | Goal | Focused command | Runtime harness | Rollback |
|---|---|---|---|---|
| 1 / PR 1 | Grammar/generation | `go test ./internal/compiscript/frontend -run 'Test(Grammar|Generation)'` | `go test ./internal/compiscript/frontend -run TestRealJARDeterminism` | Grammar/generator/generated/module/guard paths |
| 2 / PR 2 | Parse/locations | `go test ./internal/compiscript/frontend -run 'Test(Parse|SourceIndex|Diagnostics)'` | Multibyte/error fixtures through `frontend.Parse` | Parse/index/listener/diagnostic paths |
| 3 / PR 3 | 3A→3B→3C; 801-line exception; successors ≤400 | Commands below | `-v` Parse/example scenarios | Boundaries below; 3C includes acceptance |

## Phase 1: PR 1 — Grammar and Generator

- [x] 1.1 RED: add compatibility, `for` ownership, bounded `if`, float/integer/identifier, and malformed-float cases to `internal/compiscript/frontend/grammar_test.go`; record focused failure.
- [x] 1.2 RED: add `internal/compiscript/frontend/generation_test.go` subprocess cases for spaced paths, missing Java, checksum mismatch, generation failure, and formatting failure; assert existing output is unchanged after every failure.
- [x] 1.3 GREEN: update `docs/semestre2/entrega1/{Compiscript.g4,grammar-audit.md}` with grammar/evidence; implement atomic argv-safe `scripts/generate-compiscript.sh` and `tools/antlr/antlr-4.13.2-complete.jar.sha256`.
- [x] 1.4 GREEN: check in `internal/compiscript/frontend/generated/*`; pin runtime v4.13.1 in `go.mod`/`go.sum`, and narrow `.gitignore` plus `dependency_guard_test.go`; verify twice and compare snapshots.

## Phase 2: PR 2 — Parse Boundary and Diagnostics

- [x] 2.1 RED: add `internal/compiscript/frontend/{source_index,diagnostics,frontend}_test.go` for half-open UTF-8 spans, one-based coordinates, listener order, insertion/deletion/EOF spans, and exported `Parse` behavior. Rollback: whole-file `source_index_test.go` and `diagnostics_test.go`; hunk-scoped Slice 2 base Parse/listener/index behavior only in shared `frontend_test.go`, preserving later Slice 3B adjusted Parse-expectation and Slice 3C recovery test hunks; hunk-scoped task 2.1 checkbox and Slice 2 `apply-progress.md` section.
- [x] 2.2 GREEN: create `internal/compiscript/frontend/{source_index,diagnostics,frontend}.go`; share one listener collector and confine generated/runtime imports; run Unit 2 verification. Rollback: whole-file `source_index.go` and `diagnostics.go`; hunk-scoped Slice 2 base Parse/listener/index behavior only in shared `frontend.go`, preserving later Slice 3B Parse mapping wiring and Slice 3C recovery wrapper; hunk-scoped task 2.2 checkbox and Slice 2 `apply-progress.md` section.

## Phase 3: PR 3 — Mapping, Recovery, Acceptance

- [x] 3.1 `slice-3a-expression-type-mapping` (**262 actual authored; prior forecast: 325**) RED `mapper_expression_test.go`: all expressions/operators/suffixes/types/floats; fail `go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1`. GREEN `mapper_expression.go`; harness same command with `-v`. Rollback: only `internal/compiscript/frontend/mapper_expression.go`, `internal/compiscript/frontend/mapper_expression_test.go`, the hunk-scoped task 3.1 checkbox in this cumulative `tasks.md`, and the Slice 3A section in cumulative `apply-progress.md`; preserve all later slice records.
- [x] 3.2 `slice-3b-statement-mapping` (**307 actual authored; prior forecast: 339**, after 3A) RED `mapper_statement_test.go`: all 18 statements, parameters/cases, synthetic blocks; fail `go test ./internal/compiscript/frontend -run '^TestMapStatementAlternatives$' -count=1`. GREEN `mapper_statement.go`/`frontend.go`; harness same command with `-v` through `Parse`. Rollback: whole-file `internal/compiscript/frontend/mapper_statement.go` and `internal/compiscript/frontend/mapper_statement_test.go`; hunk-scoped Slice 3B Parse mapping wiring only in shared `internal/compiscript/frontend/frontend.go` and the adjusted prior Parse expectation only in shared `internal/compiscript/frontend/frontend_test.go`, preserving Slice 2 base Parse/listener/index behavior and the Slice 3C recovery wrapper; hunk-scoped task 3.2 checkbox and Slice 3B `apply-progress.md` section.
- [x] 3.3 `slice-3c-recovery-acceptance` (**151 actual implementation changes; 160 including nine grammar-audit lines; prior forecast: 137 authored including 3.4**, after 3B) RED `recovery_test.go`: valid siblings, localized `BadStmt`/`BadExpr`; fail `go test ./internal/compiscript/frontend -run '^Test(LocalizedRecovery|OfficialExamples)$' -count=1`. GREEN `mapper_recovery.go`; harness `go test ./internal/compiscript/frontend -run '^TestOfficialExamples$' -count=1 -v`. Rollback: only `internal/compiscript/frontend/mapper_recovery.go`, `internal/compiscript/frontend/recovery_test.go`, and the recovery-wrapper hunk in `internal/compiscript/frontend/frontend.go`; only the Slice 3C recovery-and-acceptance and final-acceptance sections in `docs/semestre2/entrega1/grammar-audit.md`; the hunk-scoped 3.3/3.4 checkboxes in this cumulative `tasks.md`; and the Slice 3C section in cumulative `apply-progress.md`; preserve all other content.
- [x] 3.4 `slice-3c-recovery-acceptance`, after recovery GREEN: run `go test ./...`, `go build ./...`, `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1`, regeneration `diff -qr`, and empty `gofmt -l internal/compiscript/frontend/{mapper_expression.go,mapper_expression_test.go,mapper_statement.go,mapper_statement_test.go,mapper_recovery.go,recovery_test.go,frontend.go}`. Run `gofmt -l .` informationally, allowing only `internal/yapar/parser_test.go`; record official-example/final `grammar-audit.md` evidence without commit/push, preserving generated identity. Rollback: only `internal/compiscript/frontend/mapper_recovery.go`, `internal/compiscript/frontend/recovery_test.go`, and the recovery-wrapper hunk in `internal/compiscript/frontend/frontend.go`; only the Slice 3C recovery-and-acceptance and final-acceptance sections in `docs/semestre2/entrega1/grammar-audit.md`; the hunk-scoped 3.3/3.4 checkboxes in this cumulative `tasks.md`; and the Slice 3C section in cumulative `apply-progress.md`; preserve all other content.
