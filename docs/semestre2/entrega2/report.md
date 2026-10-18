# Entrega 2 technical report

The implementation translates numeric/boolean control flow, recursive and retained lexical functions, objects/strings and lists into typed, source-located TAC. It exposes extended symbols and activation layouts through a shared CLI/IDE core. This report maps the four-page assignment to concrete evidence; it does **not** claim full assignment compliance or a grading percentage.

## Basis and outcome

Normative source: [Proyecto 2 PDF](Proyecto_2__Compiladores.pdf), pages 1–4, read in full. The source is a human-provided, untracked read-only file; this documentation does not add, export or modify it. Its SHA-256 at verification was `0ba21279ad758dd8ef69344adae9d31193d5ca4d064ce525257da6974c707bdc`. A fresh Git checkout may need the original supplied separately. The matrix below retains short Spanish quotations; page numbers refer to PDF pages, not repository line numbers.

The key result is one compiler with explicit ownership and atomic rejection. [Architecture](architecture.md) explains the decisions; [TAC design](tac-design.md) specifies the complete implemented IR, examples and assumptions. [Run instructions](README.md) and [acceptance](acceptance.md) provide the executable entry points. Known unsupported source constructs remain an actionable requirement gap, not evidence of complete language translation.

## PDF requirement and evidence matrix

Compiler files below are under `internal/compiscript`; lowering filenames are in its `tac` subdirectory. Paths starting with `cmd`, `testdata` or `web` are repository-root relative. Tests are concrete evidence entry points, not a claim of measured line coverage. “Implemented” means the named capability has implementation and executable tests; it is not a guarantee for every possible source program.

| Page / short verbatim requirement | Implementation and evidence | Outcome |
|---|---|---|
| 1: “traducen cada construcción de Compiscript” | `internal/compiscript/compile.go`; `TestCompileRejectsUnsupportedWithoutPartialTAC` contrasts successful Analyze with rejected TAC | **Gap:** not every accepted construction is lowered; see reproduction below |
| 1: “asignación, operaciones binarias y unarias, copia” | `tac/ir.go`, `expression.go`; `TestCompileNumeric`, `TestCompileInstructionMetadata`; numeric translation in [reference](tac-design.md#representation) | Implemented numeric core |
| 1: “saltos condicionales e incondicionales, etiquetas” | `tac/flow.go`, `loops.go`, `switch.go`; `conditions_test.go`, `loops_test.go`, `switch_test.go`; checked-in examples | Implemented |
| 1: “llamadas a procedimientos, acceso indexado” | `tac/calls.go`, `collections.go`; `functions_test.go`, `collections_test.go` | Implemented |
| 1: “consistente y estar documentada” | [Instruction reference](tac-design.md), typed text/JSON, formatting tests `tac/ir_test.go`, `procedures_test.go` | Documented implemented syntax |
| 1: “(+, -, *, /) respetando precedencia y asociatividad” | `tac/expression.go`, `TestCompileNumeric`; `numeric.cps`, explicit `i2f` and float division | Implemented named arithmetic |
| 1: “(&&, \|\|, !) y de comparación (==, !=, <, <=, >, >=)” | `tac/flow.go`, `conditions_test.go`; short-circuit execution/trap checks | Implemented named logic/comparisons |
| 1: “declaraciones y asignaciones de variables y constantes” | `tac/lower.go`; `compile_test.go`, semantic const tests, `TestCollectionAtomicRejection`, `TestValuesEffectsCompile` | Typed undefined declarations and static assignment values added in T9; genuinely unknown bindings remain T10 work |
| 2: “if/else, while, do-while, for y switch” | `flow.go`, `loops.go`, `switch.go`; named example files under `testdata/compiscript/tac`, `TestAcceptanceValidLoops` | Implemented within current semantic contracts (boolean switch) |
| 2: “break y continue” | Shared transfer stack; loop/switch/collection tests, mixed `acceptance.cps` | Implemented nearest eligible targets |
| 2: “paso de parámetros, llamadas y valores de retorno” | `tac/functions.go`, `calls.go`, `frames.go`; `TestFunctionFrameExecution`, `TestFunctionFrameMetadata` | Implemented typed signatures/results; unsupported untyped signatures explicitly reject |
| 2: “funciones recursivas” | Fixed-point dependencies and fresh frames; `recursion_test.go`, `recursion_dependencies_test.go`, `recursion.cps` | Implemented self/mutual recursion; termination not promised |
| 2: “funciones anidadas y closures” | `tac/closures.go`; `TestClosureExecution`, `closure_compositions_test.go`, `closures.cps` | Implemented retained shared cells and escaped aliases |
| 2: “creación de objetos e invocación del constructor” | `tac/objects.go`; `TestObjectExecution`, `TestObjectThreeLevelPolicyA`, `objects.cps` | Implemented explicit initialization policy A |
| 2: “acceso a atributos” / “incluyendo el uso de this” | Field loads/stores and bound methods; `objects_test.go`, `independent_r3_test.go` | Implemented nominal origins, receivers and inheritance |
| 2: “creación de listas” / “mediante índices” | `tac/collections.go`; `TestCollectionExecution`, `independent_t3_test.go`, `collections.cps` | Implemented empty/nested/ragged lists, reads/stores; foreach additionally supported |
| 2: “Algoritmo de asignación de temporales” | `tac/temps.go`; `tac/temps_test.go`, `TestCompileRepeatedExpressions` | Smallest free slot, independent pools per tape |
| 2: “Reciclaje de temporales una vez que su valor ha sido consumido” | Final-consumer release; live-range/reuse tests in `tac/temps_test.go`; [lifetime contract](tac-design.md#binding-and-lifetimes) | Implemented, no optimality claim |
| 2: “direcciones de memoria o desplazamientos, tamaños, etiquetas” | `tac/symbols.go`; `TestCompilationSymbols`, `tac/symbols_test.go`, `symbols_cli_test.go` | Deterministic abstract-cell offsets/sizes and actual labels, not physical addresses |
| 2: “parámetros, variables locales, temporales, valor y dirección de retorno, enlaces de control y de acceso” | `tac/frames.go`, retained environment layouts; `TestFunctionFrameMetadata`, `TestCompilationSymbols` | Explicit frame control/result cells and lexical links; [layout reference](tac-design.md#cell-layout-and-text-rendering) |
| 2: “interactuar con cada fase de la compilación” | AST/source spans -> semantic identities/scopes -> Generate -> codegen locations; `TestAcceptanceMixedExecution` immutability checks | Same identity inventory, separate non-mutating compilation report |
| 2: “extensión .cps” / “Código intermedio (TAC)” / “Reporte de errores” / “Estado de la tabla de símbolos” | CLI extension/UTF-8 checks; report `tac`, `diagnostics`, `scopes`, optional `codegen`; `cmd/compiscript-tac/main_test.go`, CLI symbol tests | Implemented text/JSON and IDE views |
| 3 step 1: “carga o escribe un archivo .cps a través del IDE” | `web` load/edit/save and compile controls; Node upload and stale-request tests; [IDE guide](ide.md) | Wiring tested; manual browser check pending |
| 3 steps 2–4: “(ANTLR) construye el árbol sintáctico” / “validando las reglas” / “Si no existen errores” | `frontend.Parse` -> `semantic.Analyze` -> `tac.Generate` in Compile; facade parity and atomic recovery tests | Implemented once per compilation; no second compiler |
| 3 step 5: “errores encontrados con su tipo y ubicación” | Typed phase/code/spans; CLI rendering and `tacText` on HTTP; `TestMixedAcceptanceAPI`, Node diagnostic rendering tests | Implemented; HTTP 200 language errors differ from transport 4xx |
| 3 step 6 and deliverable: “batería de pruebas” / “casos exitosos y fallidos” | [Runnable battery](acceptance.md#reproducible-tests), public CLI/API tests and bounded oracle execution | Passed focused checks; evaluation-day availability still human responsibility |
| 3: “repositorio de GitHub, con commits individuales” | Root repository link and audited commit history below | Local author records established; real contribution attestation and remote/evaluation access pending |
| 3: “Documentación de la arquitectura” / “cómo ejecutar el compilador” | [Architecture](architecture.md), [run guide](README.md) and root index | Versioned Markdown supplied |
| 3: “lenguaje intermedio diseñado, con ejemplos de traducción y los supuestos” | [TAC reference](tac-design.md): exact numeric example, frames, closures, objects/lists and runtime guards | Supplied; unsupported source explicitly disclosed |
| 3: “IDE funcional que permita escribir y compilar” | Shared core HTTP endpoint plus TAC/errors/symbol panels; Go httptest and 39 Node cases | Functional boundary tested; real visual evaluation pending |
| 4: “funcionar correctamente el día de la presentación” / “documentado y organizado” | Documentation index and reproducible tests; manual preparation checklist below | Cannot certify future presentation conditions |
| 4: “Diseño de CI” 25 / “Generación de TAC” 65 / “Tabla de Símbolos” 10 | Reference/design, executable generation battery, extended-symbol/layout report respectively | Grading weights totaling 100 points, **not completion percentages** |

## Observed verification and provenance

The parent-provided T1–T7/T4 records distinguish separate units and independent checks. They report final isolated T4 full Go tests (18 tested packages, four without tests), build and forced vet passing, plus 39 Node cases. Full proof preceded the final retained test-only seed addition; the parent reran affected core/IDE packages afterward. This documentation-only check did not rerun full generation/build/vet or claim a new full-suite result.

Historical evidence includes exact compatibility comparisons: T4 writer 45, independent verifier 60 plus nine panic-to-atomic changes; T6 39 comparisons stripping only the authorized additive `codegen` field and 36 repeated JSON checks; T7 writer 37 HTTP, seven DOM and 31 CLI comparisons, with independent API/UI checks. These are overlapping scoped sets, **not an additive grand total**. Original T7 RED evidence is a retained transcript (API 404; Node 14 failures/four passes), not a separately saved raw log artifact. Historical native approval applies to T1 only; later clone-local review was disabled/unmanaged, not approved.

Fresh documentation verification used offline Go with external HOME/TMPDIR/GOCACHE and existing module cache:

- Built TAC and Analyze binaries outside the repository; help, all ten checked-in TAC examples in text/JSON, rejected text/JSON, invalid-format status and legacy Analyze statuses were checked. All 20 successful example invocations exited 0 with empty stderr. No source changes or product execution occurred.
- [Focused mixed battery](acceptance.md#reproducible-tests), first command: all three packages passed. It checks CLI-emitted JSON execution: `first=30`, `second=43`, `independent=7`, `continued=57`, `shared=37`, `total=10`, `result=147`. The independent seed-2 regression's recorded result is 75, not another interpretation of the same source.
- `go test ./internal/compiscript -run '^TestCompileRejectsUnsupportedWithoutPartialTAC$' -count=1`: passed all eight source cases confirming the gap below.
- `node --test web/*.test.cjs`: 39 passed, zero failed/skipped. These are VM/DOM tests, not a real browser session.
- Markdown readback, local link/anchor/command-path checks, source PDF hash preservation and `git diff --check` form the documentation acceptance gate.

## Contribution audit

Read-only audit commands: `git log d70abf1..355804c --format='%h %ad %an %s' --date=short` and `git log --format='%h %ad %an %s' --date=short -- internal/compiscript cmd/compiscript-tac cmd/ide web docs/semestre2/entrega2`.

The seven implementation commits in the first range all record author name `jonialen` and date **2026-10-18**: `d592b9e` closures; `3a41adb` objects; `208d709` nominal field correction; `c242e03` lists/foreach; `40d6cf9` codegen report; `2d2fb4e` IDE compilation; `355804c` atomic malformed-loop rejection. Earlier TAC foundations are recorded on October 17–18 under the same author name. The path history also records earlier IDE work under `DijanU` (`e8c9324`, `cc26b6f`, `3cf44ec` on May 24, and `9ed0d12` on September 6). Those earlier commits are not evidence of a second person's contribution to the seven Entrega 2 units.

A Git author record is not an attestation of the human who performed each task. No teammate percentage, shared authorship, identity equivalence or individual contribution is inferred. Team members must verify their real contribution statements and the assignment's individual-commit requirement before submission. History was not rewritten and no authorship claim was manufactured.

## Remaining requirement gaps and presentation checks

**Concrete normative gap — accepted source without TAC lowering.** Page 1 says “traducen cada construcción de Compiscript”. The initial T8 audit found print, remainder, ternary, assignment-value, undefined/null storage and untyped-signature gaps. T9 adds the static values/effects subset with `TestValuesEffectsCompile` / `TestValuesEffectsCLI` and the [format/guard contract](tac-design.md#static-values-and-effects-t9). The old negative fixtures now use genuinely invalid or deferred constructs while retaining atomic and dead-code validation.

Remaining examples include `function f(a){return;}`, `let value;` and `try{let x=1;}catch(e){print(e);}`: unknown values/untyped signatures and handler/unwinding behavior are T10 work. Print is no longer a valid rejection fixture. Historical T8 verification above describes the baseline audit, not current rejection of those now-supported constructs. Final matrix refresh and full-assignment disposition remain pending T10 and parent verification; this unit does not claim the whole PDF complete.

**Human/evaluation checks still pending:** verify real individual contributions, GitHub visibility and availability of the untracked PDF; run the battery in the evaluation environment; open the real browser and check `.cps` load/edit/save, Compile/Analyze, located errors, TAC/symbol views, resets and unchanged YAPar behavior. Confirm CDN availability, layout/accessibility and presentation-day operation. Automated DOM proof does not discharge these steps.

**Not an explicit PDF gap:** physical byte layout/assembler, a product TAC interpreter and garbage collection are later backend concerns. Abstract cells and retained lexical/frame contracts are documented consistently. Runtime faults in dynamic calls, null/undefined fields and bounds are IR obligations, not promises that every accepted compilation executes without faults. No video, script or unrequested Word/PDF export is part of this report.
