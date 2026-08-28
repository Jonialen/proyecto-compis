# Tasks: Compiscript Semantic Analysis

Tracker only: create each child after its prerequisite is verified. Each slice targets `main`, preserves unrelated work/YALex/YAPar, and excludes SQL DSL work.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated authored lines | 2,850–3,350 total; 250–400 per unit |
| Generated ANTLR volume | 4,000–8,000 lines; budget-exempt |
| Suggested split | 9 ordered child changes/PRs |
| Delivery / chain | `auto-chain` / `stacked-to-main` |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

## Phase 1: Stable Contracts and Frontend

Every semantic item below requires RED valid/invalid located fixtures and GREEN implementation plus evidence in `testdata/compiscript/` and `docs/semestre2/entrega1/evidence/`.

- [x] 1.1 `compiscript-contracts-ast` (parent): source spans, concrete AST, stable `AnalysisReport`, ordered scopes/symbols, `[]` serialization in `internal/compiscript/{ast,model}`. **Status:** child verified and archived; 591 authored additions/0 deletions under the approved child-only `size:exception` (650 maximum, 59 headroom); parent tracker remains open. **Test:** `go test ./internal/compiscript/ast ./internal/compiscript/model && go test ./...`. **Harness:** N/A—contracts only. **Rollback:** packages/evidence.
- [x] 1.2 `compiscript-antlr-frontend` (1.1): delivered and independently verified under ordinary Git policy; its child SDD archive remains blocked by canonical Gentle AI issue #2651, with no SDD PASS/archive claimed. Valid/recoverable parsing, float/current `for`, located lexer/syntax errors, stable reruns, pinned ANTLR 4.13.2 generation, and dependency boundaries are present. **Test:** `go test ./internal/compiscript/frontend -run 'Test(Parse|Generate|Recovery)' && go test ./...`. **Harness:** `./scripts/generate-compiscript.sh`. **Rollback:** frontend/tool/runtime; generated tree separately.

## Phase 2: Semantic Behavior

- [x] 2.1 `compiscript-names-types` (1.2): nested/global resolution, shadowing, same-scope variable/function/parameter duplicates, unresolved names, function/block scopes; numeric promotion/float division, string `+`, arithmetic/logical/comparison/assignment rules, null, constants, invalid function operands, `ErrorType` suppression, exhaustive existing-node traversal, and deterministic visual-AST facade normalization. **Status:** completed under ordinary Git policy as two bounded direct units (398-line core + 387-line traversal/report) plus a verified 42-line correction rejecting function equality/null operands and globally ordering mixed diagnostics; no SDD PASS/archive claimed. **Test:** `go test ./internal/compiscript ./internal/compiscript/semantic -run 'Test(Analyze|Names|Types)' -count=1 && go test ./... -count=1 && go build ./...`. **Harness:** N/A—library. **Rollback:** `internal/compiscript/{analyze.go,view.go,analyze_test.go}`, `internal/compiscript/semantic/`, and the 2.1 status hunk.
- [x] 2.2 `compiscript-functions-flow` (2.1): recursion, forward calls, lexical closure capture, arity/positional/return types, nil-body AST external declarations, external/all-path returns; boolean conditions, valid `for`, compatible statically unique non-fallthrough switch, transfer contexts, every unreachable statement in `semantic/`. **Status:** completed under ordinary Git policy as one 335-line direct unit plus a verified 59-line correction for direct formal parameters, canonical constant-case keys, and constant-true loop returns; no SDD PASS/archive claimed. **Test:** `go test ./internal/compiscript/frontend -run 'TestMapStatementAlternatives' -count=1 && go test ./internal/compiscript/semantic -run 'Test(Functions|Flow)' -count=1 && go test ./... -count=1 && go build ./...`. **Harness:** N/A—library. **Rollback:** `internal/compiscript/frontend/{mapper_statement.go,mapper_statement_test.go}`, `internal/compiscript/semantic/{semantic.go,functions_flow_test.go,testdata/{functions_valid.cps,functions_invalid.cps,flow_valid.cps,flow_invalid.cps}}`, and the 2.2 status hunk.
- [ ] 2.3 `compiscript-collections` (2.2): homogeneous lists/matrices, ragged/contextual-empty validity, mixed-level failure; integer indices, provable bounds errors, uncertain-bounds silence in `semantic/`. **Lines:** 250–320. **Test:** `go test ./internal/compiscript/semantic -run TestCollections && go test ./...`. **Harness:** N/A—library. **Rollback:** rules/evidence.
- [ ] 2.4 `compiscript-classes-exceptions` (2.3): class scopes, members, constructor arguments, class-only `this`, inheritance/lookup/cycles, inherited-name ban, non-inherited constructors; catch-only exception binding, both blocks analyzed, no `throw` in `semantic/`. **Lines:** 360–400. **Test:** `go test ./internal/compiscript/semantic -run 'Test(Classes|Catch)' && go test ./...`. **Harness:** N/A—library. **Rollback:** pass/evidence.

## Phase 3: Consumers

- [ ] 3.1 `compiscript-cli` (2.4): `.cps` input and deterministic facade-equivalent JSON in `cmd/compiscript/`; document valid/invalid runs. **Lines:** 250–320. **Test:** `go test ./cmd/compiscript && go test ./...`. **Harness:** `go run ./cmd/compiscript testdata/compiscript/valid/types.cps`. **Rollback:** command/docs.
- [ ] 3.2 `compiscript-ide` (3.1): method/body/status and report-equivalence tests for `POST /api/compiscript/analyze`; render source, AST, diagnostics, environments while preserving `/api/process` in `cmd/ide/`, `web/`, docs. **Lines:** 350–400. **Test:** `go test ./cmd/ide && go test ./...`. **Harness:** run IDE; POST `{ "source": "let x: integer = 1;" }`. **Rollback:** route/UI/docs.

## Phase 4: Acceptance

- [ ] 4.1 `compiscript-acceptance-evidence` (3.2): map every spec/rubric rule to located valid/invalid evidence; golden CLI/IDE equality, deterministic recovery/tree/scopes, YALex/YAPar regression, runbook, no SQL/plugin surface in corpus, acceptance tests, docs. **Lines:** 250–330. **Test:** `go test ./...`. **Harness:** analyze identical CLI/IDE corpus; diff JSON. **Rollback:** corpus/docs/tests.
