# Compiscript three-address code

This is the numeric foundation of Entrega 2, not the complete assignment.
The separate `compiscript.Compile(source []byte)` library API generates typed,
source-located TAC for numeric programs. The existing analysis command and IDE
report remain unchanged. A separate CLI is deferred to T3 and is not available yet.

## Representation

An instruction has an opcode, destination, left operand, optional right operand,
and the originating AST source span. Operands distinguish literals, symbols,
and virtual temporaries and retain the semantic `integer` or `float` type.
Text renders an operand as `value:type`; structured JSON retains its kind.
This representation is not SSA: an assignment can update a symbol or reuse a
temporary. Spans locate source operations, not unique expression identities.

| Instruction | Text | Meaning |
| --- | --- | --- |
| Copy | `s1.x:integer = 1:integer` | Assign a value to a symbol or temporary |
| Add/subtract/multiply | `t0:integer = 2:integer * 3:integer` | Numeric operation |
| Divide | `t0:float = 6.0:float / 2.0:float` | Floating-point division |
| Negate | `t0:integer = neg s1.x:integer` | Numeric unary minus |
| Integer to float | `t0:float = i2f s1.x:integer` | Explicit numeric widening |

Each instruction reads all source operands before writing its destination.
Consequently, a destination may reuse the last-consumed operand's temporary.
An integer denotes an abstract exact integer; target-specific width and ABI
are not selected here. Floats denote IEEE-754 binary64 values; integer widening
may round. Mixed operations widen integers explicitly. Division widens integer
operands and produces a float, matching the semantic analyzer. No constant
folding is performed; division by zero is a runtime fault for a future backend,
not permission to omit instructions. There is no TAC executor in this unit.

## Binding and lifetimes

The source compiler assigns symbol names `s<scope-id>.<source-name>` from
the semantic scope inventory. These are logical identities, not byte offsets,
and are deterministic for the same source, not stable across source edits.
Declarations become visible only after their initializer is lowered. This
preserves outer-variable binding in a shadowing initializer. Function hoisting,
captures, and environment layout are not implemented by this numeric subset.

Temporaries start at `t0`. Allocation chooses the smallest available numeric
slot; release occurs only once the final consuming instruction is emitted.
Every occurrence retains its own type, so a free slot may be reused for a
different type. Live operands cannot overlap; repeated statements do not
reserve fresh slots indefinitely. `peakTemporaries` records peak live demand,
not physical register usage, stack size, or a claim of global optimality.

## Source-compiler boundary

The implemented source slice supports initialized integer/float variables
and constants, identifier assignment, lexical blocks, grouping, unary minus,
and `+`, `-`, `*`, `/`. Unsupported statements, operators, expressions, types,
and missing initializers must be reported explicitly with source locations.
No partial TAC may escape after lexical, syntax, semantic, or lowering errors.
Original source bytes and the existing `Analyze` report remain unchanged.
Numeric expression statements are supported; their final temporary is released
without a store. Assignment expressions (including property assignments),
ternaries, strings, booleans, lists, print, functions, and control flow are not
lowered. Unsupported input is not silently skipped.

### Library quick path

Inside this Go module, call `compiscript.Compile(source)` from
`genanalex/internal/compiscript`. The result is a `CompilationReport` with JSON
fields `tac`, `diagnostics`, and `scopes`. Parse and semantic analysis each run
once. Any frontend or semantic error preserves the same ordered diagnostics and
scope snapshots as `Analyze`, sets `tac` to `null`, and skips lowering entirely.
Lowering reports the first unsupported construct with code `TAC_UNSUPPORTED`,
phase `codegen`, a descriptive message, and its source span; `tac` is also `null`,
even when earlier statements were supported. An empty source succeeds with
`instructions: []`, `peakTemporaries: 0`, and `diagnostics: []`.

For example, `let x=1/2; x=x+3;` produces exactly:

```text
t0:float = i2f 1:integer
t1:float = i2f 2:integer
t0:float = t0:float / t1:float
s1.x:float = t0:float
t0:float = i2f 3:integer
t0:float = s1.x:float + t0:float
s1.x:float = t0:float
```

The peak is two temporaries: division consumes `t0` and `t1` before reusing
`t0` as its destination. Each JSON instruction also retains its source span and
operand kinds. No arithmetic is evaluated by the compiler.

For `let x=1; print(x);`, the result contains no TAC and reports
`unsupported statement in numeric TAC` at `print(x);`. For
`let x:integer;`, it reports `numeric declarations require an initializer`.
For `let x=missing;`, the existing semantic diagnostic is returned instead of
attempting code generation. These are library examples, not new CLI invocations.

## Remaining Entrega 2 work

- Logical/comparison instructions, short circuit, labels, branches, loops,
  switch, and correct break/continue targets.
- Calls, parameters, returns, recursion, nested functions and closures. Capture
  analysis must retain binding edges; a `Captured` flag alone is insufficient.
- Object allocation, constructors, fields, methods, `this`, lists and indexing.
- Target layout: symbol sizes/offsets, parameter/local/temporary/return slots,
  return addresses, control links, lexical access links and closure environments.
- Source-to-TAC output in the IDE and successful/failed end-to-end examples.

Future calls and memory access need distinct opcodes and explicit operands;
they must not be smuggled into numeric symbol strings. Activation records and
closure environments must be designed and tested before emitting their metadata.

## Verification

```sh
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test ./internal/compiscript ./internal/compiscript/tac -count=1
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go vet ./internal/compiscript ./internal/compiscript/tac
```

Current tests pin instruction formatting, deterministic program output, live
temporary uniqueness, release protection, type-changing reuse, and bounded
allocation across repeated expressions. Source-level tests pin precedence,
associativity, widening, division, lexical binding, exact typed/source-located
instructions, rejection without partial output, deterministic JSON, input
preservation, and compatibility with analysis diagnostics. CLI tests arrive
with T3; complete Entrega 2 coverage is not claimed by this numeric foundation.
