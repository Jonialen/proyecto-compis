```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:5d4885685351ead5eb6667e0d66ce5b297eed46317a799506d8fb7ec903c15ba
verdict: fail
blockers: 1
critical_findings: 2
requirements: 6/8
scenarios: 10/12
test_command: go test ./...
test_exit_code: 0
test_output_hash: sha256:bf5f8e0b9f216253d1c09d0a13f3b89e11664797124f7be6e273727834dc24dd
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: `compiscript-antlr-frontend`
**Version**: N/A
**Mode**: Strict TDD
**Artifact store**: Hybrid (OpenSpec + Engram)
**Native status**: `nextRecommended: verify`; 10/10 tasks complete; no acquire or settle performed.

### Completeness

| Metric | Value |
|---|---:|
| Requirements | 8 |
| Scenarios | 12 |
| Tasks total | 10 |
| Tasks complete | 10 |
| Tasks incomplete | 0 |
| Requirements compliant | 6/8 |
| Scenarios compliant | 10/12 |

All required proposal, specification, design, tasks, and merged apply-progress artifacts were read from both OpenSpec and Engram. Their requirement, scenario, and task totals agree.

### Build and Runtime Evidence

| Check | Exact command | Exit | Exact output hash | Result |
|---|---|---:|---|---|
| Focused frontend suite | `go test ./internal/compiscript/frontend -count=1 -v` | 0 | `sha256:7c169865ed23e1d04331ad1a75b81c64a1ecbf70b86cbc3b1d04641b7b50b4e4` | 12 persistent parent tests and all named subtests passed. |
| Dependency confinement | `go test . -run '^TestDependencyTreeHasNoExternalOrGUIPackages$' -count=1 -v` | 0 | `sha256:005298600aee131aa09b11babbcb1ffbfdecb4c74fc903dcb682971ef30a2ab8` | Guard passed. |
| Repository tests (authoritative final run) | `go test ./...` | 0 | `sha256:bf5f8e0b9f216253d1c09d0a13f3b89e11664797124f7be6e273727834dc24dd` | All repository packages passed. |
| Repository build | `go build ./...` | 0 | `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Passed with empty output. |
| Tool checksum | `(cd tools/antlr && sha256sum -c antlr-4.13.2-complete.jar.sha256)` | 0 | `sha256:86f862898cb77fbf16f059e8490430cac37d5481dce605f0fbd05d37c98b5380` | `antlr-4.13.2-complete.jar: OK`. |
| Runtime module | `go list -m github.com/antlr4-go/antlr/v4` | 0 | `sha256:9310d1b059569d2cd922af23cb80742c51e67f09493268fcf9ac8fd81588cb60` | `github.com/antlr4-go/antlr/v4 v4.13.1`. |
| Scoped formatter | `gofmt -l dependency_guard_test.go internal/compiscript/frontend/grammar_test.go internal/compiscript/frontend/generation_test.go internal/compiscript/frontend/source_index.go internal/compiscript/frontend/source_index_test.go internal/compiscript/frontend/diagnostics.go internal/compiscript/frontend/diagnostics_test.go internal/compiscript/frontend/frontend.go internal/compiscript/frontend/frontend_test.go internal/compiscript/frontend/mapper_expression.go internal/compiscript/frontend/mapper_expression_test.go internal/compiscript/frontend/mapper_statement.go internal/compiscript/frontend/mapper_statement_test.go internal/compiscript/frontend/mapper_recovery.go internal/compiscript/frontend/recovery_test.go internal/compiscript/frontend/generated/*.go` | 0 | `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Empty output. |
| Repository formatter (informational) | `gofmt -l .` | 0 | `sha256:dd681921c29a681e612f681f966dc3845c84b440b89e2465ce067a41fb763d6e` | Only `internal/yapar/parser_test.go`; pre-existing and outside this change. |
| Whitespace | `git diff --check` | 0 | `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Empty output. |

An earlier `go test ./...` invocation was intentionally not used as authoritative evidence because it ran concurrently with the focused suite and collided with the generator lock. It exited 1 with hash `sha256:5561c6593e5615b4ce23b4289ae25a3ea3cec9bafa90c5e8608037a9e58dbe34`; the isolated sequential rerun above passed. This was verification-harness interference, not a candidate regression.

### Generation Identity and Failure Safety

The exact two-run temporary command generated into a path containing spaces, compared both temporary trees, compared one temporary tree with the checked-in tree, counted artifacts, and hashed the fixed ordered manifest:

```bash
base=$(mktemp -d "$PWD/.compiscript-verify.XXXXXX")
trap 'rm -rf "$base"' EXIT
artifacts=(Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go)
manifest() { (cd "$1" && for f in "${artifacts[@]}"; do sha256sum "$f"; done) | sha256sum | cut -d' ' -f1; }
./scripts/generate-compiscript.sh --output "$base/path with spaces/one"
./scripts/generate-compiscript.sh --output "$base/path with spaces/two"
diff -qr "$base/path with spaces/one" "$base/path with spaces/two"
diff -qr "$base/path with spaces/one" internal/compiscript/frontend/generated
```

Exit `0`; output hash `sha256:2863dc5c59ebad86a66e20d17caad4a2b435970ef60b7b105edc5c7d859acab5`. Both temporary trees contained exactly eight artifacts. Temporary one, temporary two, and checked-in canonical identities were all `34acd09aeeaa9ae8edf51480bd1e5f3aa5f080ec81ac950daf6df81f01101010`. Generated Go headers identify ANTLR Tool 4.13.2.

`TestGenerationIsArgvSafeAndAtomic` passed runtime cases for spaced paths, missing Java, checksum mismatch, generation failure, and formatting failure. Source inspection confirms staging occurs before replacement, argv boundaries are quoted, prior output is restored on swap failure, and temporary/lock cleanup is trapped.

### Spec Compliance Matrix

| Requirement | Scenario | Runtime evidence | Result |
|---|---|---|---|
| Grammar compatibility gate | Existing program remains valid | Grammar probes for integer, `floatValue`, existing `for`, single-statement `if`; two official examples | ✅ COMPLIANT |
| Grammar compatibility gate | Breaking correction is proposed | Existing `for` ownership remains accepted; audit documents only the additive bounded `if` correction | ✅ COMPLIANT |
| Float syntax | Float declaration parses | Grammar probe plus `TestMapTypesAndFloatLiterals` and statement mapping | ✅ COMPLIANT |
| Complete direct AST mapping | Current alternatives map | 29 expression cases, 10 type/literal cases, 18 statement alternatives, and one structural integration case passed, but most cases assert only concrete node type and not the established fields/spans | ⚠️ PARTIAL |
| Localized recovery | Valid siblings surround an error | `TestLocalizedRecovery` passed `BadStmt` and nested `BadExpr` cases with valid siblings | ✅ COMPLIANT |
| Deterministic located diagnostics | Mixed diagnostics are stable | Repeated-parse temporary test passed, but an independent source-order test failed because a lexical error at byte 23 preceded a syntax error at byte 21 | ❌ FAILING |
| Deterministic located diagnostics | Multibyte source is located | Source-index and diagnostics tests passed 2-byte/4-byte scalars, one-based coordinates, deletion, insertion, and EOF spans | ✅ COMPLIANT |
| Reproducible atomic generation | Regeneration is reproducible | Two real-JAR temporary runs and checked-in tree were byte-identical | ✅ COMPLIANT |
| Reproducible atomic generation | Paths contain spaces | Real output path and fake input/tool paths containing spaces passed | ✅ COMPLIANT |
| Reproducible atomic generation | Generation prerequisite fails | Missing Java, checksum, generation, and formatter cases passed without replacing prior output | ✅ COMPLIANT |
| Narrow frontend boundary | Dependency boundary is checked | Focused dependency guard passed; runtime imports are confined to frontend | ✅ COMPLIANT |
| Regression acceptance | Change is accepted | Focused suite, official examples, `go test ./...`, `go build ./...`, scoped formatting, and whitespace checks passed | ✅ COMPLIANT |

**Compliance summary**: 10/12 scenarios compliant; 1 partial; 1 failing.

### Independent Failing Diagnostic Evidence

A temporary verification-only Go test (removed immediately after execution) parsed:

```text
let first: integer = ; @ let second: integer = ;
```

Command: `go test ./internal/compiscript/frontend -run '^TestVerificationMixedDiagnosticsFollowSourceEncounterOrder$' -count=1 -v`
Exit: `1`
Output hash: `sha256:c7b627881c20b6a7d303854bddc010fde8ac9d374eed4c5d4f00f41081256d50`

Observed diagnostics began with lexical span `[23,24)`, followed by syntax span `[21,22)`, then syntax span `[47,48)`. Therefore callback append order is deterministic but is not source encounter order for mixed lexer/parser errors. A separate temporary repeated-parse test passed 20 identical runs (exit `0`, hash `sha256:1b25292ad1bcab4fe98dda5a36522b5586ca0ca8ac57b39ef0c41cb2cacb4e88`). Both temporary test files were deleted.

### Correctness (Static Evidence)

| Area | Status | Notes |
|---|---|---|
| Grammar/float/`for` | ✅ Implemented | Float rule requires digits on both sides; integer rule is unchanged; current `for` separator ownership is preserved. |
| CST-to-AST mapping | ⚠️ Test proof partial | Direct mappers cover grammar contexts statically, preserve literal text, and avoid a generic model; behavioral assertions do not fully prove every node's fields and spans. |
| Localized recovery | ✅ Implemented | Syntax spans are localized to matching statements; print/expression recovery can preserve a statement and insert `BadExpr`. |
| Diagnostics | ❌ Incorrect ordering | Shared append-only collector preserves callback order, not mixed source encounter order. UTF-8 half-open span conversion itself passed. |
| Generation | ✅ Implemented | Tool/checksum/runtime pins, staging, formatting, swap, cleanup, spaced paths, and canonical identity verified. |
| Boundary/exclusions | ✅ Implemented | No AST/model contract, semantic analysis, CLI, IDE, SQL DSL, plugin, or generic framework change is part of this candidate. |

### Design Coherence

| Decision | Followed? | Notes |
|---|---|---|
| Export only `frontend.Parse` | ✅ Yes | Generated types remain internal. |
| Direct CST mapping with localized recovery | ✅ Yes | No intermediate model was introduced. |
| Convert original UTF-8 bytes to half-open spans | ✅ Yes | Runtime span tests passed. |
| Preserve diagnostic encounter order through one collector | ❌ No | One collector exists, but lexer lookahead can append a later lexical position before an earlier parser error. |
| Stage, verify, and atomically swap generated output | ✅ Yes | Runtime and static evidence passed. |

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | Cumulative apply-progress contains task-specific RED/GREEN/safety-net/triangulation rows. |
| All tasks have tests or justified acceptance-only scope | ✅ | 10/10; task 3.4 is explicitly command-only after recovery GREEN. |
| RED confirmed | ✅ | Test files exist; recorded authentic RED transcripts cover all behavior-bearing slices. |
| GREEN confirmed | ✅ | Every reported persistent focused test currently passes. |
| Triangulation adequate | ⚠️ | Case counts are broad, but mapping assertions are structurally shallow. |
| Safety net for modified files | ✅ | Baselines are recorded for each slice; Slice 1 reconstruction used an isolated committed baseline. |

**TDD compliance**: 5/6 checks fully passed. Strict TDD history is documented, but assertion depth does not prove the complete mapping requirement.

### Test Layer Distribution

| Layer | Behavioral cases | Files | Tools |
|---|---:|---:|---|
| Unit | 5 | 1 | Go `testing` (`source_index_test.go`) |
| Integration | 81 | 7 | Go `testing`, generated parser, subprocesses |
| E2E | 0 | 0 | Not applicable |
| **Total** | **86** | **8** | |

The distribution counts table entries and official-example inputs, not only top-level Go test functions.

### Changed File Coverage

Command: `go test ./internal/compiscript/frontend -count=1 -coverprofile=<temp>` plus per-file profile aggregation. Exit `0`; aggregate frontend statement coverage `95.2%`. Raw coverage output hash: `sha256:146099a9dd150bfd50b36ba0b99be097f8a4351700fe7569c640bbfcde82c2cb`; per-file aggregation hash: `sha256:698f011a353e3fb454b1e9b01bcad80713339e2ac0dc6d421431d3471c7bb0e7`.

| Authored production file | Statement coverage | Rating |
|---|---:|---|
| `diagnostics.go` | 100.0% | ✅ Excellent |
| `frontend.go` | 100.0% | ✅ Excellent |
| `mapper_expression.go` | 95.2% | ✅ Excellent |
| `mapper_recovery.go` | 81.1% | ⚠️ Acceptable |
| `mapper_statement.go` | 98.2% | ✅ Excellent |
| `source_index.go` | 92.6% | ⚠️ Acceptable |

Generated code, shell code, and tests are not meaningfully represented by Go statement coverage. No changed authored Go production file is below 80%.

### Assertion Quality

| File | Lines | Assertion | Issue | Severity |
|---|---:|---|---|---|
| `mapper_expression_test.go` | 52–55 | `reflect.TypeOf(got) == reflect.TypeOf(wantType)` | Most alternatives prove only node type; receiver/operands/arguments/elements/name/span are not asserted | WARNING |
| `mapper_statement_test.go` | 44–46 | `reflect.TypeOf(statement) == reflect.TypeOf(want)` | Most alternatives prove only top-level type; statement fields and spans are not asserted | WARNING |

**Assertion quality**: 0 CRITICAL, 2 WARNING classes. No tautologies, ghost loops, assertions without production calls, or mock-heavy tests were found.

### Quality Metrics

**Linter**: ➖ No separate linter detected.
**Type checker/build**: ✅ `go build ./...` passed.
**Formatter**: ✅ Changed Go scope clean; repository-wide caveat is pre-existing.
**Whitespace**: ✅ `git diff --check` passed.

### Changed Paths, Accounting, and Rollback

Verified change paths are confined to `.gitignore`, `dependency_guard_test.go`, `docs/semestre2/entrega1/{Compiscript.g4,grammar-audit.md}`, `go.mod`, `go.sum`, `internal/compiscript/frontend/`, `scripts/generate-compiscript.sh`, `tools/antlr/antlr-4.13.2-complete.jar.sha256`, and the change's OpenSpec artifacts. The checked-in generated tree contains exactly eight artifacts and `git diff --no-index --numstat /dev/null <artifact>` totals exactly **10,043 generated additions**, matching apply evidence.

The current final-tree authored delta against `HEAD`, excluding generated files, OpenSpec operational artifacts, and unrelated workspace changes, is **1,343 additions/deletions**. Apply-progress instead reports historical cumulative slice churn of **1,407 implementation changes** and **1,416 including nine audit lines**; that larger cumulative value cannot be reconstructed exactly from a single uncommitted final tree because shared files changed across slices.

One recorded detail is demonstrably inconsistent with the current candidate: Slice 3A says `mapper_expression.go` has 135 added lines and Slice 3A totals 262, while current `git diff --no-index --numstat` reports 139 for `mapper_expression.go` plus 127 for its test, totaling **266**. This does not threaten the 400-line successor cap. Even a conservative current PR3 upper bound that includes all final mapper/recovery files, both shared frontend files, and nine audit lines is **769**, still below the approved **801-line PR3 `size:exception`**. PR1/PR2/PR3 historical arithmetic and the distinct Slice 2, 3A, 3B, and 3C rollback boundaries are otherwise internally consistent.

Unrelated workspace changes were excluded: `.atl/.skill-registry.cache.json`, `.atl/skill-registry.md`, and `openspec/changes/compiscript-semantic-analysis/tasks.md`. The repository formatter caveat `internal/yapar/parser_test.go` is unchanged in `git status` and therefore pre-existing. No temporary verification file, generation tree, lock, or coverage profile remains.

### Issues Found

**CRITICAL**

1. Mixed lexical/syntax diagnostics violate source encounter order. A later lexical error `[23,24)` is returned before an earlier syntax error `[21,22)`. Requirement `Deterministic located diagnostics` is not satisfied.
2. Complete direct AST mapping lacks passing assertion coverage for established fields and spans across every statement/expression alternative. Existing tests broadly prove concrete node types, but the scenario remains only partially covered under Strict TDD verification.

**WARNING**

1. Mapping tests rely heavily on type-only assertions; structural corruption can pass.
2. Slice 3A authored accounting records 262 lines, while the current two Slice 3A files contain 266 Git-counted additions. The PR3 exception remains safe.
3. `gofmt -l .` reports pre-existing `internal/yapar/parser_test.go`; changed Go files are clean.

**SUGGESTION**

None. Verification does not prescribe remediation.

### Verdict

**FAIL**

All declared persistent tests and builds pass, generation is reproducible, and the dependency boundary holds. Acceptance is blocked because independent runtime evidence contradicts the required mixed-diagnostic encounter order, and complete AST-shape coverage is only partial.
