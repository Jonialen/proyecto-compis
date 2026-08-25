# Apply Progress: Compiscript Contracts and AST

## Status

Completed under the maintainer-approved child-only `size:exception`: normalized authored implementation files total 591 additions and 0 deletions, within the maximum of 650 with 59 lines headroom. Task 3.3 closed the failed JSON-evidence gap with test-only assertions; no tests or required contract behavior were weakened or removed.

## Completed Tasks

- [x] 1.1 AST RED test created and failed for undefined `Span`, `Position`, and `Program` contracts.
- [x] 1.2 AST GREEN implementation passed `go test ./internal/compiscript/ast`.
- [x] 2.1 Model RED test created and failed for undefined report/model contracts.
- [x] 2.2 Model GREEN implementation passed `go test ./internal/compiscript/model`.
- [x] 3.1 Scoped normalization, formatting, and dependency proofs passed.
- [x] 3.2 Focused, full-suite, vet, budget, and excluded-path acceptance passed.
- [x] 3.3 Remediation assertions prove every report JSON field/shape and all named collection nil/empty/order behavior under failed evidence revision `sha256:ad1bf6907a3d27f55be7a78817db2be5f076351f18797a9ab79f015a6632895a`.

## Pending Tasks

None.

## TDD Cycle Evidence

| Task | Test file | Layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|---|
| 1.1 | `internal/compiscript/ast/ast_test.go` | Unit | N/A (new package) | `go test ./internal/compiscript/ast` failed: undefined `Span`, `Position`, and `Program` | N/A — RED task | Three position cases plus every-node and typed-child cases | N/A — test-only task |
| 1.2 | `internal/compiscript/ast/ast_test.go` | Unit | N/A (new files) | Captured by 1.1 | `go test ./internal/compiscript/ast`: PASS | Covered multibyte, empty, producer-owned coordinates, all nodes, typed children | `gofmt -w` applied; final post-format test not run because budget gate blocked apply |
| 2.1 | `internal/compiscript/model/model_test.go` | Unit | N/A (new package) | `go test ./internal/compiscript/model` failed: undefined `AnalysisReport`, `ASTView`, and model enums | N/A — RED task | Report order, enums, repeated bytes, nil/empty arrays | N/A — test-only task |
| 2.2 | `internal/compiscript/model/model_test.go` | Unit | N/A (new files) | Captured by 2.1 | `go test ./internal/compiscript/model`: PASS | Multiple enum groups, ordered values, nested and AST empty arrays | `gofmt -w` applied; final post-format test not run because budget gate blocked apply |
| 3.1 | Existing AST/model contract tests | Unit | `go test ./internal/compiscript/ast ./internal/compiscript/model`: PASS (2 packages) before scoped normalization | N/A — verification-only task; no production behavior added | `gofmt -l internal/compiscript/ast internal/compiscript/model`: empty; import and module proofs passed | N/A — no new behavior or inputs | Scoped `gofmt -w` completed; final format check was clean |
| 3.2 | Existing AST/model contract tests | Unit | `go test ./internal/compiscript/ast ./internal/compiscript/model`: PASS (2 packages) after normalization | N/A — acceptance-only task; no production behavior added | `go test ./...`: PASS; `go vet ./...`: PASS | N/A — no new behavior or inputs | None needed; acceptance did not mutate contract behavior |
| 3.3 | `internal/compiscript/model/model_test.go` | Unit | `go test ./internal/compiscript/model -count=1`: PASS (3 top-level tests) before the new assertions | Failed verification bound `sha256:ad1bf6907a3d27f55be7a78817db2be5f076351f18797a9ab79f015a6632895a`; tests were written before any production change, which remained forbidden | `go test -v ./internal/compiscript/model -run 'TestAnalysisReportJSONIsOrderedAndDeterministic|TestNamedCollectionsEncodeArraysAndPreserveOrder' -count=1`: PASS (2 top-level tests, 9 collection subtests); no production correction | Exact report JSON tree plus 9 named collection nil/non-nil-empty/ordered cases and recursive nested `ASTView.children` | `gofmt -w internal/compiscript/model/model_test.go` followed by focused tests: PASS; test-only remediation required no further refactor |

## Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/compiscript/ast ./internal/compiscript/model -count=1` — PASS for 2 packages; remediation runtime command passed 2 top-level JSON tests and 9 collection subtests. |
| Runtime harness | `go test -v ./internal/compiscript/model -run 'TestAnalysisReportJSONIsOrderedAndDeterministic|TestNamedCollectionsEncodeArraysAndPreserveOrder' -count=1` — PASS; standard-library `json.Marshal` encoded the report tree and all named-slice scenarios. |
| Rollback boundary | Revert only task 3.3 additions in `internal/compiscript/model/model_test.go`; retain earlier six scoped files and unrelated worktree changes. |
| Formatting and dependency proof | `gofmt -l internal/compiscript/ast internal/compiscript/model` — empty. `go list -f '{{join .Imports " "}}' ./internal/compiscript/ast ./internal/compiscript/model` — `encoding/json` and `encoding/json genanalex/internal/compiscript/ast`. `go list -m all` — `genanalex`. |
| Full acceptance | `go test ./...` — PASS; `go vet ./...` — PASS. |
| Budget measurement | Per-file `git diff --no-index --numstat /dev/null <file>` produced 22, 296, 55, 95, 34, and 89 additions; `wc -l` totaled 591. Actual authored total: 591 additions + 0 deletions = 591, within the approved maximum of 650. |
| Excluded paths | The before/after `git status --porcelain=v1 --untracked-files=all` snapshots retained the pre-existing unrelated worktree changes; this apply batch changed no excluded path. |

## Next Action

Ready for re-`sdd-verify` against failed evidence revision `sha256:ad1bf6907a3d27f55be7a78817db2be5f076351f18797a9ab79f015a6632895a`; parent owns settlement. This child is complete under the parent-approved `size:exception`; do not create or implement `compiscript-antlr-frontend` from this apply batch.
