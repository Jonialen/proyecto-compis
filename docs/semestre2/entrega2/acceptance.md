# Mixed compiler acceptance

The final battery combines retained closures, object references, typed nested lists,
iteration, switch transfer and recursion through the existing compiler. It also
checks malformed input through Parse, Analyze, Compile, built CLIs and the HTTP
mux. This is executable compiler evidence, not a physical runtime or visual IDE
certification. The [technical report and PDF matrix](report.md) distinguish this
battery from complete assignment compliance and list concrete remaining gaps.

## Quick path

From the repository root, with the required Go dependencies already cached:

```sh
go run ./cmd/compiscript-tac testdata/compiscript/tac/acceptance.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/acceptance.cps
go run ./cmd/compiscript testdata/compiscript/tac/acceptance.cps
```

All three commands succeed. The first emits TAC text; the second emits a report
with non-null `tac`, empty `diagnostics` and `codegen.unit: "abstract-cell"`;
the third emits the legacy analysis report without compilation metadata.
The compiler emits code; it does **not** execute the program or print its values.

The test-only frame oracle executes both core TAC and CLI-emitted JSON:

| Global | Expected value |
|---|---:|
| `first` | 30 |
| `second` | 43 |
| `independent` | 7 |
| `continued` | 57 |
| `shared` | 37 |
| `total` | 10 |
| `result` | 147 |

`saved` retains its original invocation and shared basket after `next` is replaced.
The ragged list includes an empty row; switch `continue` skips value 2 within the
inner foreach. Recursive calls preserve caller temporaries. Valid while, do,
for and foreach loops also have separate executable preservation cases.

## Reproducible tests

```sh
go test ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide -run 'Test(MalformedForMissingBody|MissingLoopBodies|AcceptanceAtomicRecovery|AcceptanceMixedExecution|AcceptanceRecoveryCLI|AcceptanceValidLoops|MixedAcceptanceAPI)$' -count=1
go test ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide -count=1
go test ./internal/compiscript -run '^TestValuesEffects' -count=1
go vet ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide
node --test web/*.test.cjs
go test ./... -count=1
go build ./...
go vet -a ./...
```

Expected outcome: every command exits zero. Do not use `-short` for the built-CLI
proof: it deliberately skips external builds. Run full checks in an isolated
source snapshot, because existing ANTLR tests exercise generation. Offline proof
uses `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local`, an existing cached
`GOMODCACHE`, external `HOME`, `TMPDIR` and `GOCACHE`, and the cached ANTLR jar.
No installation or network download is part of this battery.

## Rejections and HTTP parity

The durable tests create their own temporary input files and assert exact stdout,
stderr and exit status against the public core report, then verify source hashes.
They exercise these failures without modifying the submitted input:

| Input | Required outcome |
|---|---|
| `for(let i=0; i<;i=i+1){` | Located syntax diagnostics, no panic |
| `while(true)`, `do`, `foreach(x in [1])`, `for(let i=0; i<2; i=i+1)` | Missing bodies rejected with located diagnostics |
| `const rows:integer[][]=[[1]];rows[0][0]=2;` | Semantic rejection; const-root checks remain authoritative |
| `let rows:integer[][]=[[1]];rows[0]=true;` | Semantic rejection of incompatible nested shape |
| `let a=[1];if(false){try{print(a[0]);}catch(e){}}` | Deferred handler codegen diagnostic even in an unexecuted branch; no silent omission |

For each rejected compilation, `tac` is null and `codegen` is absent. Built TAC
CLI text mode exits 1 with empty stdout and located diagnostics on stderr; JSON
mode exits 1 with the full report on stdout and empty stderr. Legacy Analyze
still exits 0 for language diagnostics. An invalid TAC `-format` remains a usage
error (exit 2), not a language error. These statuses describe the **built binary**;
`go run` wraps nonzero program statuses and is not the exit-policy oracle.

`TestMixedAcceptanceAPI` sends the successful fixture, valid loops and rejected
sources through `newMux` using httptest. Both `/api/compiscript/analyze` and
`/api/compiscript/compile` return HTTP 200 for language reports, exactly matching
their core JSON reports; compilation adds `tacText` from `Program.String()`.
Transport errors remain governed by the existing strict request decoder.
The existing Node VM suite separately tests request/render/reset and stale-response
wiring. No real-browser appearance or performance result is claimed here.

## Recovery choice and limits

ANTLR can recover an incomplete loop header before creating its required block.
The mapper previously indexed an empty block slice. The surgical KISS fix returns
nil only when that block is absent. Existing syntax-diagnostic recovery then
marks the statement bad, and the normal compilation pipeline rejects it atomically.
There is no blanket panic recovery, invented empty body, grammar change or new
compiler pass. The same helper serves all four loop constructs, so all four have
valid preservation and malformed-boundary proof.

Metadata offsets and sizes use abstract cells, not bytes or a target ABI. Runtime
guards are IR contracts exercised by the test oracle, not a product interpreter.
Unsupported static features must fail with diagnostics, including in dead code.

## Static values/effects battery (T9)

`values_effects_test.go` and `values_effects_cli_test.go` check core and built-CLI
JSON execution, exact output, values/fault effects, deterministic reports, unchanged
Analyze and source bytes. Cases include lazy nested joins, captures, nominal
shadowing, mutable targets/arguments, float remainder, null-result effects and
fresh undefined cells. The legacy `runFunctionFrames` oracle wrapper is preserved.

```sh
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/values-effects.cps
```

This emits TAC, not runtime output. The bounded oracle's expected example output
is `3`, `1`, `null`, each followed by a newline; `x=2`, `y=3`, `count=1`.
Unknown values/untyped signatures and try/catch still require T10. No full-PDF
completion or refreshed final matrix is claimed by this static behavior unit.
