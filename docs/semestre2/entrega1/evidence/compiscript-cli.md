# Compiscript CLI

The CLI analyzes one `.cps` source file and writes the facade's complete analysis report as JSON.

## Run it

Valid source:

```bash
go run ./cmd/compiscript testdata/compiscript/valid/types.cps
```

Source with diagnostics:

```bash
go run ./cmd/compiscript testdata/compiscript/invalid/types.cps
```

Both commands exit `0`. Diagnostics are analysis data, not process failures.

## Output contract

`stdout` contains only `model.AnalysisReport` JSON, formatted with two-space indentation and one trailing newline. The top-level field order is `ast`, `diagnostics`, then `scopes`; nested field order and JSON names come directly from the model. Empty collections remain `[]`, never `null`.

`stderr` is empty when a report is emitted. Invocation, file-extension, read, report-encoding, or output errors are human-readable on `stderr`; human-readable messages are never mixed into `stdout`.

Invalid UTF-8 is rejected before analysis with exit `1`, empty `stdout`, and `stderr` beginning with `compiscript: source is not valid UTF-8`.

| Exit | Meaning |
|---:|---|
| `0` | A report was emitted, with or without diagnostics. |
| `1` | The source is not valid UTF-8, could not be read, or the report could not be encoded or written. |
| `2` | Usage is invalid or the input path does not end in `.cps`. |

The exact invocation is `compiscript <file.cps>`; extra or missing arguments print `usage: compiscript <file.cps>` to `stderr`.
