# Compile Compiscript in the IDE

The IDE uses the same `compiscript.Compile` facade as the CLI. It displays generated TAC and static storage metadata; it does not execute programs or implement a second compiler.

## Quick path

1. From the repository root, run `go run ./cmd/ide` and open `http://localhost:8080`.
2. Select **Compiscript** and write source, use **Load .cps**, or choose **Load sample**. Uploads must have a `.cps` extension and a plain-text or unspecified MIME type (generic binary MIME is accepted for platform compatibility).
3. Choose **Compile .cps**, or press **Run / Ctrl+Enter** while the Compiscript editor is active.
4. Read **TAC**, **Diagnostics**, **Environments**, and **Extended symbols**. **Analyze .cps (AST)** retains the prior analysis-only flow and populates the AST tab.

**Clear .cps** clears the editor and invalidates results. Editing, loading another file, or switching between Compiscript and YAPar invalidates pending requests so late responses cannot replace newer outputs. A file read captures its original editor target; subsequent edits or uploads to that target supersede the read. **Save** retains the existing local-file export. YAPar's lexer, grammar, input, methods and analysis flow remain available.

## HTTP contract

`POST /api/compiscript/compile` accepts exactly `{"source":"let x = 1;"}` as `application/json` (optional charset parameters are accepted). The endpoint invokes `Compile` once and returns its complete, unchanged `CompilationReport` fields plus `tacText`, rendered on the server by `report.TAC.String()`. The browser never formats TAC from JSON. Analyze remains at `POST /api/compiscript/analyze` with its previous report shape.

| Result | HTTP / body |
|---|---|
| Valid supported source, including empty source | 200; core report and server TAC text |
| Lexical, syntax, semantic or codegen diagnostic | 200; located typed diagnostics, `tac: null`, empty `tacText`, absent `codegen` |
| Missing/null source, malformed JSON, unknown/case-variant/duplicate keys, trailing JSON or invalid UTF-8 | 400; existing `{"error":"…"}` shape |
| Body over 1 MiB | 413 |
| Missing/wrong content type | 415 |
| Method other than POST | 405, `Allow: POST` |
| CORS preflight | OPTIONS 204 |

Transport failures clear previous TAC, symbols and other Compiscript panels. Source, filenames, errors and metadata use text nodes rather than HTML. Controls are restored after completion or invalidation; Clear, sample loading and editing remain usable during a request.

## Reading extended symbols

The frozen optional `codegen` report contains `unit`, `symbols`, `spaces` and `labels`. The unit is **abstract-cell**, not bytes or physical addresses. Symbol entries retain semantic identities, scope IDs, type/source metadata, locations (`space`, `offset`, `size`, `role`) and labels. The panel shows complete storage-space slots and label spans as text, preserving the authoritative core report rather than inferring layouts in JavaScript. Globals, main temporaries, invocation frames, retained environments and instances have distinct ownership. Labels are not cells; empty location lists do not imply an invented allocation.

## Verification and limits

Focused checks: `go test ./cmd/ide -count=1`, `node --test web/*.test.cjs`, and `go vet ./cmd/ide`. The Node command runs both the implementation and retained independent VM tests. HTTP tests compare the complete core report and server TAC text, and check strict transport parity with Analyze. Node's built-in VM tests exercise request bodies, rendering, resets, hostile text, failures and asynchronous ordering without a new browser framework.

These tests establish HTTP/DOM wiring, not real-browser appearance. CodeMirror and Viz.js retain their existing CDN requirements. Visual/manual browser evaluation remains separate; compiler execution proof belongs to the existing CLI/compiler tests. No remote execution service is provided.
