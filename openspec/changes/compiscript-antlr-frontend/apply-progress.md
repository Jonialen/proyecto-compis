# Apply Progress: Slices 1–3C — Complete ANTLR Frontend Apply

## Completed Tasks
- [x] 1.1 Grammar compatibility and float RED tests.
- [x] 1.2 Atomic-generation failure RED tests.
- [x] 1.3 Compatible grammar and atomic generator GREEN implementation.
- [x] 1.4 Generated tree, runtime pin, dependency guard, and snapshot verification.
- [x] 2.1 RED: source index, diagnostic ordering/span, and exported Parse tests.
- [x] 2.2 GREEN: UTF-8 source index, shared diagnostics collector, and Parse boundary.

## Final Evidence Reconciliation (2026-08-26)

This is evidence-only reconciliation for the completed change. No implementation,
generated artifact, task checkbox, or routing outcome changed. The active route is
`sdd-verify` only.

### Canonical Generated-Tree Identity

The canonical manifest fixes both the eight-artifact scope and ordering: for each
listed basename it emits `sha256sum` from the tree root, then SHA-256 hashes those
eight ordered manifest lines. The exact command sequence was:

```bash
base=$(mktemp -d "$PWD/.compiscript-reconcile.XXXXXX")
trap 'rm -rf "$base"' EXIT
artifacts=(Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go)
manifest() { (cd "$1" && for f in "${artifacts[@]}"; do sha256sum "$f"; done) | sha256sum | awk '{print $1}'; }
./scripts/generate-compiscript.sh --output "$base/one"
./scripts/generate-compiscript.sh --output "$base/two"
diff -qr "$base/one" "$base/two"
diff -qr "$base/one" internal/compiscript/frontend/generated
printf 'canonical_one=%s\ncanonical_two=%s\ncanonical_checked_in=%s\n' "$(manifest "$base/one")" "$(manifest "$base/two")" "$(manifest internal/compiscript/frontend/generated)"
```

Both `diff -qr` commands were silent and exited `0`. The exact outputs were:

```text
canonical_one=34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010
canonical_two=34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010
canonical_checked_in=34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010
```

Therefore the one canonical current identity is
`34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010`.
The earlier Slice 1 `efa281361d53683aa5a3da0eac818be6376d0445acb5ad4145800fac3e1ce620`
claim has no recorded manifest-construction command or scope: its recorded command
only generated two outputs and ran `diff -qr`. No documented intervening generated
change or historical algorithm/scope distinction explains the different hash. It is
thus an unsupported, erroneous identity assertion rather than evidence of a different
generated tree. The historical silent `diff -qr` result remains determinism evidence;
the historical `efa281...` identity text below is retained only as explicitly
superseded at-the-time evidence.

### Superseded Historical Routing and Workload Guidance

The Next Route sections for Slice 2, Slice 3A, and Slice 3B below are retained only
as historical-at-the-time instructions. They are superseded by the final active
`sdd-verify` route. The historical recommendation to reacquire an 850-line cap is
rejected and superseded by the maintainer-authorized successor rescope of at most 400
authored lines for 3A, 3B, and 3C; there is no active 850-line execution objective.

### Reconciliation Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | N/A: this evidence-only reconciliation changed no production behavior or test and therefore ran no focused behavior test. |
| Runtime harness command/scenario and exact result | The canonical two-temporary-run generation, both `diff -qr` comparisons, and all three manifests above exited `0` with the identical `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010` identity. |
| Rollback boundary | Revert only this Final Evidence Reconciliation hunk and the grammar-audit canonical-manifest hunk; all implementation, generated artifacts, tasks, historical evidence, and final routing remain untouched. |

Strict TDD remains active for implementation tasks. This reconciliation adds no task,
production code, or test, so it creates no RED/GREEN/REFACTOR cycle and makes no such
claim.

## Maintainer-Authorized Authentic RED Reconstruction

- Work unit: `slice-1-authentic-red-reconstruction`
- Authority: maintainer reset revision `sha256:e493030935e9a0fdcb1066d1118f772ea3a9ab04927bbfd9a29ab3e35ef0bc43`; reason: authentic RED reconstruction after omitted transcripts.
- Baseline: detached disposable worktree at committed `HEAD` `2de60c8`, created at `/home/jonialen/Documents/uvg/s8/compis/proyecto-compis-worktrees/slice-1-authentic-red-reconstruction`.
- Isolation: only the current Slice 1 RED test sources were materialized in the disposable worktree. No grammar, generator, generated output, module, or guard change was copied into it.
- Primary candidate: no implementation, test, task, specification, design, proposal, tracker, or unrelated `.atl` file was modified by this reconstruction. This artifact is the sole primary-worktree file changed by the work unit.

### Authentic RED Transcripts

`go test ./internal/compiscript/frontend -run '^TestGenerationIsArgvSafeAndAtomic$' -count=1` executed in the detached baseline worktree and exited `1`:

```text
--- FAIL: TestGenerationIsArgvSafeAndAtomic (0.01s)
    --- FAIL: TestGenerationIsArgvSafeAndAtomic/spaced_paths (0.00s)
        generation_test.go:56: success=true, output=, error=fork/exec /home/jonialen/Documents/uvg/s8/compis/proyecto-compis-worktrees/slice-1-authentic-red-reconstruction/scripts/generate-compiscript.sh: no such file or directory
FAIL
FAIL	genanalex/internal/compiscript/frontend	0.014s
FAIL
EXIT_CODE=1
```

`go test ./internal/compiscript/frontend -run '^TestGrammarCompatibilityAndFloatLiterals$' -count=1` executed in the same detached baseline worktree and exited `1`:

```text
--- FAIL: TestGrammarCompatibilityAndFloatLiterals (0.00s)
    grammar_test.go:17: generate grammar: fork/exec /home/jonialen/Documents/uvg/s8/compis/proyecto-compis-worktrees/slice-1-authentic-red-reconstruction/scripts/generate-compiscript.sh: no such file or directory
FAIL
FAIL	genanalex/internal/compiscript/frontend	0.005s
FAIL
EXIT_CODE=1
```

The failures are authentic baseline executions: the pre-Slice-1 committed baseline lacks the required generator. The grammar test cannot generate the parser needed to execute its nine compatibility/float cases, and the generation test fails on its first required argv-safe spaced-path scenario. These transcripts prove the specified behavior was absent without importing GREEN sources into the baseline.

### Disposable Worktree Cleanup

```text
git worktree remove --force /home/jonialen/Documents/uvg/s8/compis/proyecto-compis-worktrees/slice-1-authentic-red-reconstruction
git worktree prune
WORKTREE_REMOVED=yes
git worktree list
/home/jonialen/Documents/uvg/s8/compis/proyecto-compis 2de60c8 [main]
```

## TDD Cycle Evidence

| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| 1.1 | `internal/compiscript/frontend/grammar_test.go` / subprocess integration | Isolated committed baseline; no production file was edited | Authentic baseline command above exited `1` because `scripts/generate-compiscript.sh` was absent | Current candidate: focused grammar/generation command exited `0` | 9 cases: existing integer, `floatValue`, float declaration, `for`, single-statement `if`, and four malformed floats | No source change in reconstruction; existing implementation retained |
| 1.2 | `internal/compiscript/frontend/generation_test.go` / subprocess integration | Isolated committed baseline; no production file was edited | Authentic baseline command above exited `1` at the spaced-path scenario because the generator was absent | Current candidate: focused grammar/generation command exited `0` | 5 cases: spaced paths, missing Java, checksum mismatch, generation failure, formatting failure | No source change in reconstruction; existing atomic staging retained |
| 1.3 | `generation_test.go` / subprocess integration | Evidence-only reconstruction; existing candidate was not edited | 1.2's authentic RED contract proves the absent generator behavior | Current focused grammar/generation command exited `0`; direct real-JAR harness passed twice | Success plus four distinct failure controls exercise the atomic generator contract | No refactor performed |
| 1.4 | `grammar_test.go`, `generation_test.go`, `dependency_guard_test.go` / integration | Evidence-only reconstruction; existing candidate was not edited | 1.1/1.2 authentic RED contracts prove the unavailable grammar/generation boundary | Dependency guard, full suite, build, and two eight-file real-JAR snapshots passed | Grammar cases, five generator controls, two snapshots, dependency boundary, and repository regression | No refactor performed |

The reconstruction establishes executable baseline RED evidence. It does not claim to recreate historical wall-clock authoring order; the current RED sources were run unchanged against the committed pre-Slice-1 baseline, then removed with that worktree.

## Current Candidate GREEN and Regression Evidence

| Check | Exact command / scenario | Result |
|---|---|---|
| Focused grammar and generator | `go test ./internal/compiscript/frontend -run 'Test(GrammarCompatibilityAndFloatLiterals|GenerationIsArgvSafeAndAtomic)' -count=1` | Exit `0`; `ok genanalex/internal/compiscript/frontend 1.184s` |
| Dependency guard | `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1` | Exit `0`; `ok genanalex 0.191s` |
| Real-JAR deterministic generation | Historical-at-the-time command: `base=$(mktemp -d "$PWD/.compiscript-realjar.XXXXXX"); ./scripts/generate-compiscript.sh --output "$base/one"; ./scripts/generate-compiscript.sh --output "$base/two"; diff -qr "$base/one" "$base/two"` | Exit `0`; each run produced 8 files and `diff -qr` was silent. The historical row asserted `efa281361d53683aa5a3da0eac818be6376d0445acb5ad4145800fac3e1ce620` without recording a manifest algorithm; that unsupported identity assertion is superseded by the canonical `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010` reconciliation above. |
| Repository tests | `go test ./...` | Exit `0`; 10 packages passed and 5 packages reported `[no test files]` |
| Repository build | `go build ./...` | Exit `0`; no output |
| Scoped changed Go formatting | `gofmt -l dependency_guard_test.go internal/compiscript/frontend/grammar_test.go internal/compiscript/frontend/generation_test.go internal/compiscript/frontend/generated/*.go` | Exit `0`; no output |
| Repository-wide formatting | `gofmt -l .` | Command exited `0` but reported only pre-existing `internal/yapar/parser_test.go`; this is outside Slice 1 and blocks only the repository-wide clean claim |
| Tracked-diff whitespace | `git diff --check` | Exit `0`; no output |

## Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/compiscript/frontend -run 'Test(GrammarCompatibilityAndFloatLiterals|GenerationIsArgvSafeAndAtomic)' -count=1` exited `0` with `ok genanalex/internal/compiscript/frontend 1.184s`. |
| Runtime harness command/scenario and exact result | Historical-at-the-time: two direct runs of the pinned real JAR generated 8 files each and `diff -qr` was silent. Its unrecorded-algorithm `efa281...` identity assertion is superseded by the canonical eight-artifact `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010` reconciliation. |
| Rollback boundary | Independently revert only the exact Slice 1 implementation paths below, the Slice 1 grammar/generation audit-content hunk in `docs/semestre2/entrega1/grammar-audit.md`, and the Slice 1-specific checkbox/evidence hunks in the two OpenSpec artifacts. Do not revert any cumulative artifact as a whole: later Slice 2 and Slice 3 checkboxes and evidence, including Slice 3C recovery/acceptance and final-reconciliation grammar-audit evidence, remain independently preserved. This removes float grammar compatibility, the atomic generator, its tests, the generated ANTLR snapshot, runtime pin/guard allowance, and Slice 1 evidence without touching AST/model, Slice 2, Slice 3, semantic analysis, CLI, IDE, or unrelated `.atl` paths. |

## Exact Slice 1 Changed Paths and Independent Rollback Boundary

The independently reversible Slice 1 path set is:

- `.gitignore`
- `dependency_guard_test.go`
- `docs/semestre2/entrega1/Compiscript.g4`
- `docs/semestre2/entrega1/grammar-audit.md` (Slice 1 grammar/generation audit-content hunk only)
- `go.mod`
- `go.sum`
- `internal/compiscript/frontend/generation_test.go`
- `internal/compiscript/frontend/grammar_test.go`
- `internal/compiscript/frontend/generated/compiscript_base_visitor.go`
- `internal/compiscript/frontend/generated/compiscript_lexer.go`
- `internal/compiscript/frontend/generated/compiscript_parser.go`
- `internal/compiscript/frontend/generated/compiscript_visitor.go`
- `internal/compiscript/frontend/generated/Compiscript.interp`
- `internal/compiscript/frontend/generated/Compiscript.tokens`
- `internal/compiscript/frontend/generated/CompiscriptLexer.interp`
- `internal/compiscript/frontend/generated/CompiscriptLexer.tokens`
- `openspec/changes/compiscript-antlr-frontend/tasks.md`
- `openspec/changes/compiscript-antlr-frontend/apply-progress.md`
- `scripts/generate-compiscript.sh`
- `tools/antlr/antlr-4.13.2-complete.jar.sha256`

For `docs/semestre2/entrega1/grammar-audit.md`,
`openspec/changes/compiscript-antlr-frontend/tasks.md`, and
`openspec/changes/compiscript-antlr-frontend/apply-progress.md`, this boundary is
hunk-scoped: revert only Slice 1 grammar/generation audit content, Slice 1
checkboxes, and Slice 1 evidence. It must preserve all later Slice 2 and Slice 3
records in the cumulative OpenSpec files and the later Slice 3C recovery/acceptance
and final-reconciliation evidence in the cumulative grammar audit.

## Delivery and Accounting

`auto-chain` / `stacked-to-main`; Slice 1 only. The authoritative Slice 1 accounting remains **371 authored lines** and **10,043 generated lines**. This maintainer-authorized evidence-only reconstruction changes no implementation or generated snapshot path, so it does not revise that accounting.

## Slice 2 — Parse Boundary and Diagnostics

### Exact RED Transcript

`go test ./internal/compiscript/frontend -run 'Test(Parse|SourceIndex)' -count=1` exited `1` before Slice 2 production files existed:

```text
# genanalex/internal/compiscript/frontend [genanalex/internal/compiscript/frontend.test]
internal/compiscript/frontend/diagnostics_test.go:11:20: undefined: Parse
internal/compiscript/frontend/diagnostics_test.go:50:22: undefined: Parse
internal/compiscript/frontend/frontend_test.go:11:26: undefined: Parse
internal/compiscript/frontend/source_index_test.go:11:11: undefined: newSourceIndex
internal/compiscript/frontend/source_index_test.go:46:11: undefined: newSourceIndex
FAIL	genanalex/internal/compiscript/frontend [build failed]
FAIL
EXIT_CODE=1
```

### TDD Cycle Evidence

| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| 2.1 | `source_index_test.go`, `diagnostics_test.go`, `frontend_test.go` / unit-integration boundary | Baseline `go test ./...` and `go build ./...` both exited `0`: 10 test packages passed, 5 reported `[no test files]` | Exact compile-failure transcript above; all five new symbols were undefined | After 2.2, focused command exited `0` | Two UTF-8 scalar cases; two coordinate cases; lexical→syntax order; deletion, insertion, EOF; valid exported Parse | No structural refactor needed; scoped gofmt and focused test stayed green |
| 2.2 | Same Slice 2 test set / integration boundary | New production files; Slice 1 frontend tests remained green in the baseline | 2.1's exact RED command exercised the absent Parse/index boundary before this code existed | `go test ./internal/compiscript/frontend -run 'Test(Parse|SourceIndex|Diagnostics)' -count=1` exited `0`: `ok genanalex/internal/compiscript/frontend 0.003s` | Parser recovery uses distinct deletion, insertion, and EOF fixtures; source indexing covers 2-byte and 4-byte UTF-8 scalars | No structural refactor needed; formatting preserved behavior |

### Slice 2 GREEN, Harness, and Regression Evidence

| Check | Exact command / scenario | Result |
|---|---|---|
| Focused tests | `go test ./internal/compiscript/frontend -run 'Test(Parse|SourceIndex|Diagnostics)' -count=1` | Exit `0`; `ok genanalex/internal/compiscript/frontend 0.003s`. |
| Runtime Parse harness | `go test ./internal/compiscript/frontend -run '^TestParseReportsLexerAndParserDiagnosticsInEncounterOrder$' -count=1 -v` | Exit `0`; one exported `Parse` scenario passed using `é\nlet value: integer = ;`, asserting a lexical `[0,2)` diagnostic before a syntax diagnostic. |
| Dependency guard | `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1` | Exit `0`; `ok genanalex 0.093s`. |
| Full tests | `go test ./...` | Exit `0`; 10 packages passed and 5 packages reported `[no test files]`. |
| Full build | `go build ./...` | Exit `0`; no output. |
| Scoped formatting | `gofmt -l internal/compiscript/frontend/source_index.go internal/compiscript/frontend/diagnostics.go internal/compiscript/frontend/frontend.go internal/compiscript/frontend/source_index_test.go internal/compiscript/frontend/diagnostics_test.go internal/compiscript/frontend/frontend_test.go` | Exit `0`; no output. |
| Diff whitespace | `git diff --check` | Exit `0`; no output. |

### Slice 2 Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/compiscript/frontend -run 'Test(Parse|SourceIndex|Diagnostics)' -count=1` exited `0` with `ok genanalex/internal/compiscript/frontend 0.003s`. |
| Runtime harness command/scenario and exact result | `go test ./internal/compiscript/frontend -run '^TestParseReportsLexerAndParserDiagnosticsInEncounterOrder$' -count=1 -v` exited `0`; it called exported `Parse` on a multibyte lexical error followed by a syntax error and passed one scenario. |
| Rollback boundary | Revert whole files only for `internal/compiscript/frontend/source_index.go`, `internal/compiscript/frontend/source_index_test.go`, `internal/compiscript/frontend/diagnostics.go`, and `internal/compiscript/frontend/diagnostics_test.go`. In shared `internal/compiscript/frontend/frontend.go` and `internal/compiscript/frontend/frontend_test.go`, revert only the Slice 2 base Parse/listener/index behavior hunks, preserving later Slice 3B Parse mapping wiring/adjusted Parse-expectation hunks and the Slice 3C recovery wrapper. Revert only the hunk-scoped 2.1/2.2 checkboxes in cumulative `tasks.md` and the Slice 2 section in cumulative `apply-progress.md`. This removes located Parse/index/diagnostic behavior without touching Slice 1 generation, Slice 3 mapper/recovery, AST/model, semantic analysis, CLI, IDE, or unrelated work. |

### Exact Slice 2 Changed Paths and Accounting

- `internal/compiscript/frontend/source_index.go`
- `internal/compiscript/frontend/diagnostics.go`
- `internal/compiscript/frontend/frontend.go`
- `internal/compiscript/frontend/source_index_test.go`
- `internal/compiscript/frontend/diagnostics_test.go`
- `internal/compiscript/frontend/frontend_test.go`
- `openspec/changes/compiscript-antlr-frontend/tasks.md`
- `openspec/changes/compiscript-antlr-frontend/apply-progress.md`

Slice 2 implementation and tests total **316 authored added lines** across the six frontend paths and **0 generated lines**. The two task-checkbox replacements add 2 lines and remove 2 lines. The slice remains within the 400 authored-line implementation budget; inherited Slice 1 files and unrelated pre-existing changes are excluded from this accounting.

## Historical Next Route (at Slice 2 completion; superseded)

Historical instruction: proceed to Slice 3 only: tasks **3.1–3.4** (direct CST-to-AST mapper, localized recovery, and acceptance). Do not start final verification from this work unit. It is superseded by the final `sdd-verify` route.

## Historical Slice 3 Workload Gate — Stopped Before Authoring (superseded)

- Native runtime authority remains retained by the orchestrator: `sha256:c3ea028c21cb59f2fd5c5983b8cd91efe13fc564916e5337ce5ff55f3608a3aa`.
- Delivery remains `auto-chain / stacked-to-main`; work unit `slice-3-cst-ast-mapping-recovery`.
- Strict TDD is active. The existing frontend-package safety net passed: `go test ./internal/compiscript/frontend -count=1` exited `0` with `ok genanalex/internal/compiscript/frontend 0.810s`.
- No Slice 3 RED test or production source was authored, so tasks 3.1–3.4 remain unchecked. This prevents an incomplete RED/GREEN cycle and preserves the required order: tests before production code.

### Reacquisition Forecast

The complete required Slice 3 scope cannot fit the currently authorized 400 authored changed-line cap. The direct CST-to-AST visitor must cover 18 statement alternatives, assignment/ternary/precedence expressions, three suffix alternatives, types, parameters, switch cases, synthetic blocks, and localized recovery; the required table-driven mapping and recovery tests must prove each contract.

| Planned authored path | Forecast lines |
|---|---:|
| `internal/compiscript/frontend/mapper.go` | 470 |
| `internal/compiscript/frontend/mapper_test.go` | 210 |
| `internal/compiscript/frontend/recovery_test.go` | 95 |
| `internal/compiscript/frontend/frontend.go` integration | 4 |
| `docs/semestre2/entrega1/grammar-audit.md` Slice 3 acceptance evidence | 18 |
| `openspec/changes/compiscript-antlr-frontend/tasks.md` checkboxes | 4 |
| **Slice 3 total authored forecast** | **801** |

No generated files are required for Slice 3. Historical at-the-time recommendation (rejected and superseded): reacquire with an explicit `size:exception` cap of at least **801 authored changed lines** (recommended operational cap: **850**) before starting RED work. The maintainer-authorized successor rescope instead bounded 3A, 3B, and 3C to at most 400 authored lines; no 850-line execution objective is active. This unsplit Slice 3 rollback forecast is obsolete and superseded by the actual independent Slice 3A, 3B, and 3C rollback boundaries recorded later in this artifact; it establishes no active rollback path. The final route remains `sdd-verify` only.

## Slice 3A — Expression and Type Mapping

### Completed Task

- [x] 3.1 `slice-3a-expression-type-mapping`: direct CST-to-AST mapping for every expression alternative, operator, suffix, type, and float literal.

### TDD Cycle Evidence

| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| 3.1 | `internal/compiscript/frontend/mapper_expression_test.go` / unit | `go test ./internal/compiscript/frontend -count=1` exited `0`: `ok genanalex/internal/compiscript/frontend 1.429s` before authoring | Test file was authored first; the exact focused command exited `1` because `newExpressionMapper` was undefined | Exact focused command exited `0`: `ok genanalex/internal/compiscript/frontend 0.011s` | 29 expression alternatives plus 10 float/type cases exercise operators, suffixes, literals, base/custom types, and dimensions | Kept a direct recursive CST mapper; shared binary folding, suffix application, and argument collection without an intermediate model; focused test stayed green |

### Exact RED Transcript

`go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1` exited `1` before `mapper_expression.go` existed:

```text
# genanalex/internal/compiscript/frontend [genanalex/internal/compiscript/frontend.test]
internal/compiscript/frontend/mapper_expression_test.go:105:9: undefined: newExpressionMapper
internal/compiscript/frontend/mapper_expression_test.go:113:9: undefined: newExpressionMapper
FAIL	genanalex/internal/compiscript/frontend [build failed]
FAIL
```

### GREEN and Regression Evidence

| Check | Exact command / scenario | Result |
|---|---|---|
| Focused GREEN | `go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1` | Exit `0`; `ok genanalex/internal/compiscript/frontend 0.011s`. |
| Runtime harness | `go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1 -v` | Exit `0`; 2 parent tests and 39 named cases passed: 29 expression/operator/suffix alternatives and 10 literal/type cases. `ok genanalex/internal/compiscript/frontend 0.018s`. |
| Full regression | `go test ./...` | Exit `0`; 11 packages passed and 5 packages reported `[no test files]`. |
| Full build | `go build ./...` | Exit `0`; no output. |
| Scoped changed-file formatting | `gofmt -l internal/compiscript/frontend/mapper_expression.go internal/compiscript/frontend/mapper_expression_test.go` | Exit `0`; no output. |
| Repository-wide formatting (informational) | `gofmt -l .` | Exit `0`; reported only pre-existing `internal/yapar/parser_test.go`. |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1` exited `0` with `ok genanalex/internal/compiscript/frontend 0.011s`. |
| Runtime harness command/scenario and exact result | `go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1 -v` exited `0`; it mapped 29 expression alternatives and 10 float/type cases directly from generated parser CST contexts. |
| Rollback boundary | Revert only `internal/compiscript/frontend/mapper_expression.go`, `internal/compiscript/frontend/mapper_expression_test.go`, the hunk-scoped task 3.1 checkbox in cumulative `tasks.md`, and the Slice 3A section in cumulative `apply-progress.md`. This removes expression/type mapping without touching Slice 1, Slice 2, statement mapping, recovery, official examples, acceptance, AST/model contracts, or unrelated work. |

### Exact Changed Paths and Accounting

- `internal/compiscript/frontend/mapper_expression.go`
- `internal/compiscript/frontend/mapper_expression_test.go`
- `openspec/changes/compiscript-antlr-frontend/tasks.md`
- `openspec/changes/compiscript-antlr-frontend/apply-progress.md`

The two Slice 3A frontend paths contain **262 authored added lines** (135 mapper lines and 127 test lines) and **0 generated lines**. Task completion replaces one checkbox line; prior Slice 1/Slice 2 and unrelated worktree changes are excluded. This native successor stays below the 400 authored-line cap inside the approved PR3 `size:exception` boundary.

## Historical Next Route (at Slice 3A completion; superseded)

Historical instruction: proceed only to task 3.2 `slice-3b-statement-mapping`; do not begin recovery, official examples, final acceptance, review, archive, commit, push, or PR work from Slice 3A. It is superseded by the final `sdd-verify` route.

## Slice 3B — Statement Mapping

### Completed Task

- [x] 3.2 `slice-3b-statement-mapping`: direct CST-to-AST mapping for all 18 statement alternatives through exported `Parse`, including parameters, switch cases, and synthetic `if` bodies.

### Exact RED Transcript

`go test ./internal/compiscript/frontend -run '^TestMapStatementAlternatives$' -count=1` exited `1` after `mapper_statement_test.go` was written and before statement mapping/wiring existed:

```text
--- FAIL: TestMapStatementAlternatives (0.01s)
    --- FAIL: TestMapStatementAlternatives/variable_declaration (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/constant_declaration (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/assignment (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/function (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/class (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/expression (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/print (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/block (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/if (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/while (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/do_while (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/for (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/foreach (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/try_catch (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/switch (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/break (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/continue (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/return (0.00s)
        mapper_statement_test.go:42: statements = 0, want 1
    --- FAIL: TestMapStatementAlternatives/parameters_cases_and_synthetic_blocks (0.00s)
        mapper_statement_test.go:53: program = ast.Program{Span:ast.Span{Start:ast.Position{Offset:0, Line:1, Column:1}, End:ast.Position{Offset:127, Line:1, Column:128}}, Statements:ast.Statements(nil)}, diagnostics = model.Diagnostics(nil)
FAIL
FAIL	genanalex/internal/compiscript/frontend	0.013s
FAIL
```

### TDD Cycle Evidence

| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| 3.2 | `internal/compiscript/frontend/mapper_statement_test.go` / integration boundary | `go test ./internal/compiscript/frontend -count=1` exited `0`: `ok genanalex/internal/compiscript/frontend 1.589s` before authoring | Exact RED transcript above: 18 mapped statement cases and the parameter/case/synthetic-block scenario observed the deferred empty program | Exact focused command exited `0`: `ok genanalex/internal/compiscript/frontend 0.015s` | 18 grammar alternatives plus parameter types, regular/default switch cases, and both synthetic `if` bodies passed through `Parse` | Added direct recursive CST helpers only; no behavior-preserving refactor was needed after GREEN |

### GREEN, Harness, and Regression Evidence

| Check | Exact command / scenario | Result |
|---|---|---|
| Focused GREEN | `go test ./internal/compiscript/frontend -run '^TestMapStatementAlternatives$' -count=1` | Exit `0`; `ok genanalex/internal/compiscript/frontend 0.015s`. |
| Verbose Parse runtime harness | `go test ./internal/compiscript/frontend -run '^TestMapStatementAlternatives$' -count=1 -v` | Exit `0`; 1 parent test and 19 named scenarios passed through exported `Parse` in `0.011s`. |
| Full regression | `go test ./...` | Exit `0`; 11 packages passed and 5 reported `[no test files]`. |
| Full build | `go build ./...` | Exit `0`; no output. |
| Dependency guard | `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1` | Exit `0`; `ok genanalex 0.160s`. |
| Scoped formatter and diff | `gofmt -l internal/compiscript/frontend/mapper_statement.go internal/compiscript/frontend/mapper_statement_test.go internal/compiscript/frontend/frontend.go internal/compiscript/frontend/frontend_test.go && git diff --check` | Exit `0`; no output. |
| Repository formatter (informational) | `gofmt -l .` | Exit `0`; reported only allowed pre-existing `internal/yapar/parser_test.go`. |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/compiscript/frontend -run '^TestMapStatementAlternatives$' -count=1` exited `0`; `ok genanalex/internal/compiscript/frontend 0.015s`. |
| Runtime harness command/scenario and exact result | The same command with `-v` exited `0`; all 18 alternatives plus the parameter/case/synthetic-block integration scenario entered `Parse` and passed. |
| Rollback boundary | Revert whole files only for `internal/compiscript/frontend/mapper_statement.go` and `internal/compiscript/frontend/mapper_statement_test.go`. In shared `internal/compiscript/frontend/frontend.go`, revert only the Slice 3B Parse mapping wiring hunk; in shared `internal/compiscript/frontend/frontend_test.go`, revert only the adjusted prior Parse-expectation hunk. Preserve the Slice 2 base Parse/listener/index behavior and the Slice 3C recovery wrapper. Revert only the hunk-scoped 3.2 checkbox in cumulative `tasks.md` and the Slice 3B section in cumulative `apply-progress.md`. This leaves Slices 1–3A, recovery, official examples, final acceptance, AST/model contracts, and unrelated work untouched. |

### Exact Changed Paths and Accounting

- `internal/compiscript/frontend/mapper_statement.go`
- `internal/compiscript/frontend/mapper_statement_test.go`
- `internal/compiscript/frontend/frontend.go`
- `internal/compiscript/frontend/frontend_test.go`
- `openspec/changes/compiscript-antlr-frontend/tasks.md`
- `openspec/changes/compiscript-antlr-frontend/apply-progress.md`

The four frontend paths contain **300 authored additions and 7 deletions (307 total)**: 226 mapper lines, 68 new test lines, 2 Parse-wiring additions/3 deletions, and 4 adjusted prior-Parse-test additions/4 deletions; **0 generated lines**. This successor remains below 400 authored implementation lines inside the retained PR3 **801-line `size:exception`** review boundary.

## Historical Next Route (at Slice 3B completion; superseded)

Historical instruction: proceed only to task 3.3 `slice-3c-recovery-acceptance`; do not begin localized recovery, official examples, final acceptance, review, archive, commits, pushes, or PR work in this slice. It is superseded by the final `sdd-verify` route.

## Slice 3C — Localized Recovery and Final Acceptance

### Completed Tasks

- [x] 3.3 `slice-3c-recovery-acceptance`: localized malformed declarations to `BadStmt`, normalized recovered missing operands to `BadExpr`, and retained valid sibling statements.
- [x] 3.4 `slice-3c-recovery-acceptance`: completed the official-example, dependency, regression, build, formatting, deterministic-regeneration, generated-identity, and grammar-audit acceptance checks.

### Exact RED Transcript

After adding `internal/compiscript/frontend/recovery_test.go` and before adding `mapper_recovery.go` or changing `frontend.go`, the exact command below exited `1`:

```text
$ go test ./internal/compiscript/frontend -run '^Test(LocalizedRecovery|OfficialExamples)$' -count=1
--- FAIL: TestLocalizedRecovery (0.00s)
    --- FAIL: TestLocalizedRecovery/bad_statement_preserves_valid_siblings (0.00s)
        recovery_test.go:22: statement[1] = ast.VarDeclStmt, want ast.BadStmt
    --- FAIL: TestLocalizedRecovery/bad_expression_stays_inside_its_statement (0.00s)
        recovery_test.go:41: broken operand = ast.UnaryExpr, want ast.BadExpr
FAIL
FAIL	genanalex/internal/compiscript/frontend	0.016s
FAIL
```

### TDD Cycle Evidence

| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| 3.3 | `internal/compiscript/frontend/recovery_test.go` / integration boundary | `go test ./internal/compiscript/frontend -count=1` exited `0`: `ok genanalex/internal/compiscript/frontend 1.712s` before test authoring | Exact command/transcript above exited `1` before recovery code existed; a recovered declaration remained `ast.VarDeclStmt` and a missing operand remained `ast.UnaryExpr` | Same exact command exited `0`: `ok genanalex/internal/compiscript/frontend 0.009s` after `mapper_recovery.go` and Parse wiring | Two localized-recovery scenarios plus two official-example sources: malformed declaration with valid declaration siblings, malformed print operand with valid print siblings, basic declarations/lists, and functions/classes/foreach control flow | Normalized only ANTLR's synthetic empty unary operand into `BadExpr`; no behavior-preserving refactor remained after the focused suite stayed green |
| 3.4 | Existing 3.3 suite / acceptance-only | 3.3 GREEN focused suite was green before acceptance commands | N/A: this task adds no production behavior; its acceptance inputs are the already-GREEN recovery and official-example tests | `go test ./...` and `go build ./...` both exited `0` | N/A: no new behavior or production branch exists in this command-only task | N/A: no production refactor |

### Final Acceptance Evidence

| Check | Exact command / scenario | Result |
|---|---|---|
| Focused recovery GREEN | `go test ./internal/compiscript/frontend -run '^Test(LocalizedRecovery|OfficialExamples)$' -count=1` | Exit `0`; `ok genanalex/internal/compiscript/frontend 0.009s`. |
| Official-example runtime harness | `go test ./internal/compiscript/frontend -run '^TestOfficialExamples$' -count=1 -v` | Exit `0`; one parent test passed in `0.011s`. Its two source scenarios came from `docs/semestre2/entrega1/Especificaciones.md` and produced no diagnostics or top-level bad statements. |
| Complete focused frontend suite | `go test ./internal/compiscript/frontend -count=1` | Exit `0`; `ok genanalex/internal/compiscript/frontend 1.608s`. `go test ./internal/compiscript/frontend -list .` listed 12 parent tests, including `TestLocalizedRecovery` and `TestOfficialExamples`. |
| Dependency guard | `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1` | Exit `0`; `ok genanalex 0.256s`. |
| Repository tests | `go test ./...` | Exit `0`; 11 packages passed and 5 packages reported `[no test files]`; frontend passed in `2.072s`. |
| Repository build | `go build ./...` | Exit `0`; no output. |
| Scoped formatter and whitespace | `gofmt -l internal/compiscript/frontend/mapper_expression.go internal/compiscript/frontend/mapper_expression_test.go internal/compiscript/frontend/mapper_statement.go internal/compiscript/frontend/mapper_statement_test.go internal/compiscript/frontend/mapper_recovery.go internal/compiscript/frontend/recovery_test.go internal/compiscript/frontend/frontend.go && git diff --check` | Exit `0`; no output. |
| Repository formatter (informational) | `gofmt -l .` | Exit `0`; reported only documented pre-existing `internal/yapar/parser_test.go`. This unrelated repository-wide caveat does not affect the empty scoped formatter result. |
| Two-run temporary regeneration | `base=$(mktemp -d "$PWD/.compiscript-regeneration.XXXXXX") && trap 'rm -rf "$base"' EXIT && ./scripts/generate-compiscript.sh --output "$base/one" && ./scripts/generate-compiscript.sh --output "$base/two" && diff -qr "$base/one" "$base/two"` | Exit `0`; each run produced 8 artifacts and `diff -qr` was silent. |
| Generated identity | Ordered SHA-256 manifests for both temporary outputs and `internal/compiscript/frontend/generated` | All three manifests: `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010`. The manifest covers `Compiscript.interp`, `Compiscript.tokens`, `CompiscriptLexer.interp`, `CompiscriptLexer.tokens`, `compiscript_base_visitor.go`, `compiscript_lexer.go`, `compiscript_parser.go`, and `compiscript_visitor.go`. |
| Grammar audit | `docs/semestre2/entrega1/grammar-audit.md` recovery-and-acceptance section | Recorded the localized bad-node scenarios, official-example acceptance, two-run eight-artifact regeneration result, and matching generated manifest. |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/compiscript/frontend -run '^Test(LocalizedRecovery|OfficialExamples)$' -count=1` exited `0`; `ok genanalex/internal/compiscript/frontend 0.009s`. |
| Runtime harness command/scenario and exact result | `go test ./internal/compiscript/frontend -run '^TestOfficialExamples$' -count=1 -v` exited `0`; the two official-source scenarios entered exported `Parse`, returned diagnostics-free programs, and had no top-level `BadStmt`. |
| Rollback boundary | Revert only `internal/compiscript/frontend/mapper_recovery.go`, `internal/compiscript/frontend/recovery_test.go`, and the recovery-wrapper hunk in `internal/compiscript/frontend/frontend.go`; only the Slice 3C recovery-and-acceptance and final-acceptance sections in `docs/semestre2/entrega1/grammar-audit.md`; the hunk-scoped 3.3/3.4 checkboxes in cumulative `tasks.md`; and the Slice 3C section in cumulative `apply-progress.md`. This removes only localized recovery and its acceptance evidence while retaining Slices 1–3B, AST/model contracts, and unrelated work. |

### Exact Changed Paths and Accounting

- `internal/compiscript/frontend/mapper_recovery.go`
- `internal/compiscript/frontend/recovery_test.go`
- `internal/compiscript/frontend/frontend.go` (one recovery wrapper replacement only)
- `docs/semestre2/entrega1/grammar-audit.md`
- `openspec/changes/compiscript-antlr-frontend/tasks.md`
- `openspec/changes/compiscript-antlr-frontend/apply-progress.md`

Slice 3C implementation/test content is **149 new frontend lines** (`80` recovery mapper + `69` tests) plus **1 addition and 1 deletion** in existing Parse wiring: **151 authored implementation changes**, **0 generated lines**. The grammar-audit acceptance record adds 9 authored documentation lines; task and progress records are operational evidence. Cumulative PR3 implementation accounting is **720 authored changes** (`262` Slice 3A + `307` Slice 3B + `151` Slice 3C), or **729** including the 3C grammar-audit acceptance record; both remain inside the retained **801-line `size:exception`** PR3 boundary, and Slice 3C remains below the native 400-line successor cap.

## Final Next Route (Active)

All change tasks are checked. Return to the orchestrator for verification only; do not commit, push, open a PR, review, or archive from this apply work unit.

## Bounded Unmanaged Remediation (2026-08-26)

This single correction batch is bounded to the two independent verification blockers.
It preserves all prior completed-task evidence above and changes no generated artifact,
proposal, specification, design, task checkbox, or failed verification report.

```yaml
schema: gentle-ai.remediation-result/v1
mode: unmanaged
lineage_id: unmanaged
generation: 1
fix_batch: unmanaged-1
native_acquire_token: sha256:80c3f48b89acb9b68ab285aa2cadbaaba5a9836094b4628a626305e53f1a5498
failed_evidence_revision: sha256:1e4cc41ab1708c5664913d2760d32521da381e1fca5e64f35433a2f8dd3e0d66
blockers_resolved: 2
correction_evidence_sha256: sha256:9430bf094ab8c8db44dbf5c806b6153ed060585cbf494edaff219c6a99fe7cce
```

```json
{"schema":"gentle-ai.remediation-evidence/v1","mode":"unmanaged","lineage_id":"unmanaged","generation":1,"fix_batch":"unmanaged-1","native_acquire_token":"sha256:80c3f48b89acb9b68ab285aa2cadbaaba5a9836094b4628a626305e53f1a5498","failed_evidence_revision":"sha256:1e4cc41ab1708c5664913d2760d32521da381e1fca5e64f35433a2f8dd3e0d66","correction_evidence":{"diagnostics":"mixed source-order RED then GREEN","mapping":"exact expression/type fields and spans RED then GREEN; statement contract assertions passed"}}
```

### TDD Cycle Evidence

| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| Mixed diagnostic source order | `diagnostics_test.go` / integration | `go test ./internal/compiscript/frontend -count=1` exited `0` | New mixed input failed: lexical `[23,24)` preceded syntax `[21,22)` | Exact focused test exited `0`; stable equal-offset ordering also passed | Mixed lexical/syntax ordering plus equal-offset tie preservation | Minimal stable final sort by start byte offset; focused suite remained green |
| Complete mapping proof | `mapper_expression_test.go`, `mapper_statement_test.go` / integration | Same frontend safety net exited `0` | Strengthened exact expression/type assertions exposed suffix spans that excluded their receiver | Expression/type focused suite exited `0`; all statement contract cases exited `0` | 29 expression alternatives, 10 literal/type alternatives, and 18 statement alternatives | Added `suffixSpan`; no statement-mapper production change was required |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/compiscript/frontend -run '^(TestParseOrdersMixedDiagnosticsBySourceEncounter|TestOrderDiagnosticsPreservesInputOrderForEqualOffsets)$' -count=1 -v` exited `0`; `go test ./internal/compiscript/frontend -run '^TestMap(ExpressionAlternatives|TypesAndFloatLiterals)$' -count=1 -v` and `-run '^TestMapStatementAlternatives$' -count=1 -v` exited `0`. |
| Runtime harness command/scenario and exact result | Exported `Parse` ran the mixed malformed source and all 18 statement alternatives; all focused harnesses exited `0`. |
| Rollback boundary | Revert only `diagnostics.go`, `frontend.go`, `diagnostics_test.go`, `mapper_expression.go`, `mapper_expression_test.go`, `mapper_statement_test.go`, and this remediation section. |

### Regression and Identity Evidence

- `go test ./...`, `go build ./...`, and `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1` exited `0`.
- `git diff --check` and scoped `gofmt -l` were empty. Repository `gofmt -l .` reported only pre-existing `internal/yapar/parser_test.go`.
- Temporary regeneration matched the checked-in eight-artifact tree and preserved canonical identity `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010`; temporary outputs were removed.
