# Complex CLI regression programs

Seven source files exercise interacting compiler features, not just individual instructions. Four finish successfully, one intentionally faults during TAC execution, and two must be rejected before any TAC is exposed.

Run the reproducible battery from the repository root:

```sh
go test ./internal/compiscript -run '^TestComplexSourceFilesCLI$' -count=1 -v
```

The test builds the actual CLI, compiles each file in **both text and JSON**, checks exact report parity and exit/output channels, executes successful CLI JSON through the existing bounded test oracle, asserts expected values/output/faults, and verifies that source-file hashes do not change.

## Cases and expected results

| File | Main interactions | Compiler exit | Expected execution |
|---|---|---|---|
| [closures-ledger.cps](closures-ledger.cps) | Inherited constructor, virtual dispatch, escaped mutable captures, independent instances, bound method alias and list reduction | 0 | Output `13, 17, 22, 103, 29, 184`, one value per line; `result=184` |
| [matrix-transfers.cps](matrix-transfers.cps) | Ragged/empty lists, shared nested mutation, nested foreach, switch break, loop continue, for/do/while and remainder | 0 | Output `6, 20, 21, 32, 59`; `result=59` |
| [recursive-exceptions.cps](recursive-exceptions.cps) | Untyped recursion, caught division faults, nested handlers and a fault raised inside a catch body | 0 | Output `19, 2, 11, 720, 7, 738`; `result=738` |
| [evaluation-order.cps](evaluation-order.cps) | Receiver/index snapshots, effectful calls, lazy ternary, assignment values and changing unannotated field tags | 0 | Output `1234, 20, 17, changed`; `sum=17`, `x=5`, `y=6` |
| [uncaught-runtime-fault.cps](uncaught-runtime-fault.cps) | Loop/call effects before an uncaught guarded division | 0 | Output `50`, then `division by zero`; `attempts=2`, `total=50`; no completion message |
| [invalid-scope.cps](invalid-scope.cps) | Valid class/function/list prefix followed by a foreach-local binding escaping its scope | 1 | Located semantic diagnostics; null TAC, absent codegen |
| [invalid-list-argument.cps](invalid-list-argument.cps) | Valid typed calls/object mutation followed by an incompatible list argument | 1 | Located semantic diagnostics; null TAC, absent codegen |

An expected rejection or runtime fault is a **passing regression outcome**, not a claim that every file should finish execution.

## Inspect one file manually

```sh
go run ./cmd/compiscript-tac testdata/compiscript/stress/closures-ledger.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/stress/recursive-exceptions.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/stress/invalid-scope.cps
```

The CLI generates TAC; it does **not** execute `print` or runtime guards. The execution outputs above are asserted by the test-only oracle. The uncaught-fault program therefore compiles successfully, whereas the two semantic-error programs fail compilation atomically.

Exit values in the table describe the built CLI. `go run` can wrap a failing program's exit status and print `exit status 1`; use the automated test or a built binary when checking exact exit codes.
