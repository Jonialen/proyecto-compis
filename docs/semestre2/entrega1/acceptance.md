# Entrega 1 acceptance runbook

Run from the repository root. The authoritative gate builds the current CLI and invokes the current IDE handler through `httptest`; it never depends on port `8080` or another process.

For requirement status, source contradictions, and remaining work, see [current compliance](compliance.md).

## Quick path

Prerequisites: the Go version in `go.mod`; Java only for generator identity; Node for browser syntax.

```bash
go test ./cmd/ide -run 'Test(CompiscriptAcceptanceCorpus|CompiscriptRecoveryPreservesValidSiblings|AcceptanceManifestRejectsUnsafeInput|AcceptanceManifestRejectsSymlinkEscape|AcceptanceRequirementEvidence)' -count=1
go test ./internal/compiscript/... ./cmd/compiscript ./cmd/ide -count=1
go test ./internal/compiscript/frontend -run 'Test(LocalizedRecovery|OfficialExamples)' -count=1
```

Expected: all exit `0`. `TestOfficialExamples` covers two representative combined parser programs; dependent source-document snippets are not claimed as standalone programs.

## Evidence matrix

`testdata/compiscript/acceptance/manifest.tsv` is sorted, path-confined, and hash-pinned for drift detection. Hashes do not prove correctness: tests assert exact diagnostics/locations, recovery, tree/array shape, ordered environments, and facade/CLI/IDE equality.

| Rule | Located valid / invalid `.cps` evidence |
|---|---|
| Parse, visual tree, recovery, UTF-8 | `types-valid`; `recoverable:2`; `multibyte:2`; `lexical-invalid:1`; `constant-initializer-invalid:1` |
| Names, nested scopes, shadowing, duplicates | `names-valid:1`; `names-invalid:1` |
| Arithmetic, logic, comparison, assignment, constants | `operators-valid:1`; `operators-invalid:1`; `names-valid:1`; `names-invalid:1` |
| Null and recovery sentinel | `null-valid:1`; `null-invalid:1`; `names-invalid:1` |
| Functions, recursion, closures, arguments, returns | `functions-valid:1-11`; `functions-invalid:1-9` |
| Conditions, `for`, `foreach`, switch, transfer, dead code | `flow-valid:1-11`; `flow-invalid:1-23`; `foreach-valid:1`; `foreach-invalid:1` |
| Homogeneous/ragged lists, indices, provable bounds | `collections-valid:1-17`; `collections-invalid:1-25` |
| Classes, constructors, `this`, inheritance | `classes-valid:1-17`; `classes-invalid:1-18` |
| Catch-only exception binding | `catch-valid:1`; `catch-invalid:1-2` |
| IDE (15), semantics/tree (60), environments (25) | Every row crosses all consumers; focused tests prove polarity |

Operators cover `%`, `||`, unary `!`, `!=`, `>`, and `>=`; null covers valid string/list/class and invalid integer/boolean targets. `ErrorType` is a recovery sentinel whose valid polarity is suppression of dependent diagnostics, not a source type. Missing constant initialization is parser-only invalid evidence because the grammar requires `=` and an expression.

## Full, candidate, and generator gates

```bash
go test ./internal/yalex ./internal/yapar ./cmd/yapar -count=1
go test . -run 'TestIntegration_' -count=1
go test ./... -count=1
go build ./...
go test . -run TestDependencyTreeHasNoExternalOrGUIPackages -count=1
node --check web/app.js

tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
export GIT_INDEX_FILE="$tmp/index"
git read-tree HEAD; git add -A; git diff --cached --check
test -z "$(git diff --cached --name-only -z --diff-filter=ACM -- '*.go' | xargs -0 -r gofmt -l)"
bad=$(git diff --cached --name-only | grep -Ev '^(cmd/ide/acceptance_test.go|docs/semestre2/entrega1/acceptance.md|openspec/changes/compiscript-semantic-analysis/tasks.md|testdata/compiscript/acceptance/.*)$' || true)
test -z "$bad"
rm -rf "$tmp"; unset GIT_INDEX_FILE; trap - EXIT

tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
./scripts/generate-compiscript.sh --output "$tmp/generated"
diff -qr internal/compiscript/frontend/generated "$tmp/generated"
identity=$(cd "$tmp/generated" && for f in Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go; do sha256sum "$f"; done | sha256sum | cut -d' ' -f1)
test "$identity" = 79d82b69a89a6ef6b5d21a1eaf9ec9b5699b9db3048c2deaa244bbdc344fe2bb
```

The temporary index covers untracked files without mutating the real index. Its path allowlist plus the dependency guard excludes production packages, imports, dependencies, and extension mechanisms. Explicit generator ordering is locale-independent and never replaces committed output.

## Troubleshooting and rollback

- Hash mismatch: inspect output; update only for an approved contract change.
- Generator mismatch: verify the pinned JAR checksum and exact artifact set.
- Rollback: remove acceptance test/runbook/corpus and restore only the 4.1 tracker line; no production behavior belongs to this unit.
