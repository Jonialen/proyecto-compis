# Compiscript IDE API

The IDE analyzes browser-provided Compiscript without changing the YALex/YAPar `/api/process` workflow.

## Reproduce

Start the server, then submit source:

```bash
go run ./cmd/ide
curl --fail-with-body -sS \
  -H 'Content-Type: application/json' \
  --data '{"source":"let x: integer = 1;"}' \
  http://localhost:8080/api/compiscript/analyze
```

The `200` response is the exact JSON encoding of `model.AnalysisReport`: `ast`, `diagnostics`, and `scopes`. Diagnostics, including syntax recovery diagnostics, are report data and still return `200`. Empty source is valid.

The complete request body must be valid UTF-8 and no larger than 1 MiB (`1,048,576` bytes). The JSON object must contain exactly one string-valued `source`; duplicate keys are rejected rather than resolved last-value-wins. `application/json; charset=utf-8` is accepted.

## HTTP contract

| Status | Meaning |
|---:|---|
| `200` | The complete analysis report was emitted. |
| `400` | The body is invalid UTF-8; JSON is malformed/trailing; `source` is missing, duplicated, or not a string; or the object has unknown fields. |
| `405` | The request method is not `POST`; `Allow: POST` is returned. |
| `413` | The complete request body exceeds 1 MiB; no partial source is analyzed. |
| `415` | `Content-Type` is not `application/json`. |

Errors use `{"error":"<stable message>"}`. The browser's Compiscript editor renders the submitted source, nested visual AST, diagnostics, and nested environments/symbols with DOM text nodes rather than HTML injection.
