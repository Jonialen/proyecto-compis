# Tasks: Compiscript Contracts and AST

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated / observed lines | 374 design estimate; 591 authored additions observed |
| 400-line budget risk | High; maintainer exception accepted |
| Approved exception | Child-only `size:exception`, maximum 650; 59-line headroom |
| Chained PRs recommended | No; one child PR in the parent stack |
| Suggested split | Single `compiscript-contracts-ast` PR targeting `main` |
| Delivery strategy | `exception-ok`; later units retain the 400-line default |
| Chain strategy | `stacked-to-main` |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | AST/report contracts | Current child PR | `go test ./internal/compiscript/ast ./internal/compiscript/model` | N/A—data only | Delete the six `internal/compiscript/{ast,model}` files |

## Phase 1: AST RED/GREEN

- [x] 1.1 **RED** — Create `internal/compiscript/ast/ast_test.go` covering multibyte/empty spans, stored spans, sealed categories, typed children, and every node. **Prerequisite:** parent 1.1/design. **Evidence:** `go test ./internal/compiscript/ast` failed on missing contracts. **Lines:** 48. **Rollback:** delete test.
- [x] 1.2 **GREEN** — Create `internal/compiscript/ast/{span.go,nodes.go}` with exact concrete contracts, named slices, value `SourceSpan`, and markers only. **Prerequisite:** 1.1 RED. **Evidence:** `go test ./internal/compiscript/ast` passed. **Lines:** 170. **Rollback:** delete files.

## Phase 2: Model RED/GREEN

- [x] 2.1 **RED** — Create `internal/compiscript/model/model_test.go` covering enums, JSON, order, repeatable bytes, and nested empty arrays. **Prerequisite:** 1.2. **Evidence:** `go test ./internal/compiscript/model` failed on missing contracts. **Lines:** 52. **Rollback:** delete test.
- [x] 2.2 **GREEN** — Create `internal/compiscript/model/{model.go,json.go}` with six values, exact enums, `model -> ast`, and ordered nil normalization without maps/sorting. **Prerequisite:** 2.1 RED. **Evidence:** `go test ./internal/compiscript/model` passed. **Lines:** 104. **Rollback:** delete files.

## Phase 3: Proof and Completion

- [x] 3.1 Format/prove imports for the six scoped files. **Prerequisite:** 2.2. **Evidence:** scoped `gofmt -l` was empty; focused `go list` returned only `encoding/json` and `genanalex/internal/compiscript/ast`; `go list -m all` returned only `genanalex`. **Lines:** 0. **Rollback:** formatting only.
- [x] 3.2 Record acceptance in this file. **Prerequisite:** 3.1. **Evidence:** focused/full tests and `go vet ./...` passed; tracked numstat omitted six untracked files, while per-file no-index numstat plus `wc -l` reconciled 589 additions/0 deletions. Approved maximum 650 leaves 61 lines; no excluded path changed. **Lines:** 0. **Rollback:** delete six scoped files; preserve unrelated work.
- [x] 3.3 **Authorized remediation RED/GREEN** — Extended only `internal/compiscript/model/model_test.go`. **RED:** retained verifier evidence `sha256:ad1bf6907a3d27f55be7a78817db2be5f076351f18797a9ab79f015a6632895a`; added table-driven exact JSON-tree assertions for every documented report field/shape and nil, non-nil-empty, ordered cases for every named slice. **GREEN:** focused assertions passed without a production change, so no documented-behavior defect was exposed. **Evidence:** focused/full tests, `go vet ./...`, empty scoped `gofmt -l`, and no-index numstat plus `wc -l`; no excluded-path change and actual total `591 <= 650`. **Lines:** 2 net. **Rollback:** revert the test-only additions.
