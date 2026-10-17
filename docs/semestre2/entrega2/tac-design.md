# Compiscript three-address code

This is the numeric foundation of Entrega 2, not the complete assignment.
The existing analysis command and IDE report remain unchanged. A separate
source compiler and CLI will consume this representation in subsequent units.

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

The source compiler will assign symbol names `s<scope-id>.<source-name>` from
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

The planned initial source slice supports initialized integer/float variables
and constants, identifier assignment, lexical blocks, grouping, unary minus,
and `+`, `-`, `*`, `/`. Unsupported statements, operators, expressions, types,
and missing initializers must be reported explicitly with source locations.
No partial TAC may escape after lexical, syntax, semantic, or lowering errors.
Original source bytes and the existing `Analyze` report must remain unchanged.
The source compiler and command are not available until their work units land.

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
go test ./internal/compiscript/tac -count=1
```

Current tests pin instruction formatting, deterministic program output, live
temporary uniqueness, release protection, type-changing reuse, and bounded
allocation across repeated expressions. Source-level and CLI tests will be
added with those behaviors, not inferred from these representation tests.
