# Compiscript three-address code

This is the numeric, conditional, loop and boolean-switch subset of Entrega 2, not the complete assignment.
The separate `compiscript.Compile(source []byte)` library API generates typed,
source-located TAC for numeric and boolean programs with `if/else`, `while`,
`do-while`, `for` and `switch`. Existing analysis reports remain unchanged except for the
corrected optional-for-header mapping described below. The separate `compiscript-tac` command compiles this
subset without changing the existing analysis command's interface or exit policy.

## Command quick path

Run the included numeric example (it computes the TAC for `x = 3.5`, without
executing the source program):

```sh
go run ./cmd/compiscript-tac testdata/compiscript/tac/numeric.cps
go run ./cmd/compiscript-tac -format json testdata/compiscript/tac/numeric.cps
go run ./cmd/compiscript-tac testdata/compiscript/tac/conditions.cps
go run ./cmd/compiscript-tac -format json testdata/compiscript/tac/conditions.cps
go run ./cmd/compiscript-tac testdata/compiscript/tac/loops.cps
go run ./cmd/compiscript-tac -format json testdata/compiscript/tac/loops.cps
go run ./cmd/compiscript-tac testdata/compiscript/tac/switch.cps
go run ./cmd/compiscript-tac -format json testdata/compiscript/tac/switch.cps
go run ./cmd/compiscript-tac -h
```

The default format writes only TAC text to stdout. Rejected compilation writes
located diagnostics to stderr, no TAC to stdout, and exits 1. JSON format writes
the complete `CompilationReport` to stdout, including `tac: null` on rejection;
its exit code is still 1 when diagnostics exist. Success exits 0, including an
empty source. Invalid arguments or format exit 2; file/UTF-8/write failures
exit 1; help exits 0 and prints usage to stderr. Flags precede the source path.
`go run` can wrap a program's nonzero exit; build the binary when an automation
needs the command's exact status. Scope snapshots are semantic metadata, not
the extended address/activation-record layout still required by Entrega 2.

## Representation

An instruction has an opcode, destination, left operand, optional right operand,
and the originating AST source span. The optional `label` field names a control-flow
destination; it is omitted from numeric JSON, preserving existing field order and
output. Operands distinguish literals, symbols, and virtual temporaries and retain
the semantic `integer`, `float` or `boolean` type.
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
| Comparison | `t0:boolean = 1:integer < 2:integer` | `==`, `!=`, `<`, `<=`, `>`, `>=`; boolean equality/inequality also supported |
| Logical not | `t0:boolean = ! s1.ready:boolean` | Boolean negation |
| Label | `L0:` | Control-flow destination |
| Jump | `goto L0` | Unconditional edge |
| Branch true | `if t0:boolean goto L0` | Jump when the boolean left operand is true; otherwise fall through |

Label/jump instructions use `label`; branch-true also uses `left`. Their unused
legacy `target` operand is zero-valued; label/jump `left` is also zero-valued,
and `right` is omitted. These fields are not value reads or writes. Labels start
at `L0` and are allocated deterministically per compilation. Existing numeric
instruction text and JSON remain unchanged.

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

## Conditions and control flow

Comparisons evaluate left to right and produce booleans; mixed numeric operands
receive explicit integer-to-float conversions. Arithmetic still requires numeric
operands, not booleans. Boolean literals, initialized declarations/constants,
identifier assignments, expression statements and unary `!` are supported.

`a && b` branches to `b` only when `a` is true; `a || b` branches to `b` only
when `a` is false. Grouping and nested negation retain these semantics. A logical
expression used as a value reserves a live result temporary before lowering its
operands, then writes true or false on the corresponding incoming edge before
joining. Recursive RHS operations cannot recycle that reserved result. A condition
can instead branch directly without materializing a logical value.

`if/else` lowers both bodies but executes only the selected body. Nested `if`,
`else if`, no-else, empty and unbraced bodies use the frontend's existing block
wrappers and matching semantic child scopes. Each declaration initializer reads
outer bindings before activating its own name; branch-local names do not escape.
Both sides of a logical expression and both branches are always statically
validated, even after a literal condition. Unsupported syntax or semantic errors
in unreachable source still reject the entire compilation.

The conditional example guards division by zero and selects `quotient = -1`.
The CLI emits TAC only; execution assertions use a bounded test-only interpreter,
not a product runtime or optimizer.

## Source-compiler boundary

### Loops and transfers

| Construct | Execution order | Nearest-loop `continue` destination |
| --- | --- | --- |
| `while` | Condition, body, repeat | Condition |
| `do-while` | Body, condition, repeat if true | Condition |
| `for` | Initializer once, condition, body, post, repeat | Post, then condition |

`break` exits the nearest loop or switch, skipping a for post when exiting that loop. An omitted for condition is
true; initializer and post are optional independently. Nested and consecutive
loops retain separate transfer targets. All bodies and posts are statically
validated even when a literal condition prevents execution; rejection emits no TAC.

A for initializer uses the existing semantic loop scope, with a child scope for
the body. While/do bodies use their block scopes; a do condition cannot see body
locals. Shadowing initializers read outer bindings before activating their names.
Header/body/post temporaries are consumed before backedges; runtime iterations
reuse emitted temporary identities without increasing static allocation demand.

Only a whole for post may be a simple identifier assignment expression, optionally
grouped: `for (;; (i=i+1)) { if(i>=3) break; }`. It uses the existing assignment
validation, widening and store path. Numeric/boolean post expressions may instead
be evaluated and discarded. Nested assignments, property/index assignments and
assignment expressions elsewhere remain unsupported.

The frontend now identifies condition/post by their actual header delimiters,
not expression count. Declaration/assignment initializers consume their own first
semicolon. The intentional compatibility exception is omitted-condition,
present-post headers, for every initializer variant: `for (;; i=i+1) { ... }`
now has no condition and an assignment post, rather than an assignment condition
that spuriously reports `SEM_CONDITION` / `condition must be boolean`. This also
corrects public AST/analysis reports for that family. Grammar, AST schema and
semantic policies are unchanged; other existing analysis reports are preserved.

The terminating loop example computes `total = 13` without print or a product
runtime. It exercises while-continue, do-while and for break/continue.

### Boolean switches

The selector must be boolean; cases retain the analyzer's compatible-type and
statically unique-case rules. Numeric selectors remain semantic errors, as required
by the team policy in `../entrega1/compliance.md`. The selector is evaluated once,
even for an empty or default-only switch. Case expressions execute in source order
until the first match; default runs only if none match. Every selected body exits
implicitly, including an empty case: there is no fallthrough.

Dispatch precedes all body labels. A selector temporary stays live and unchanged
through every comparison; each comparison uses a separate boolean result slot.
All tests and bodies are lowered and validated, including unselected source.
Any error discards the whole TAC program. No new opcodes or runtime are introduced.

Switch and loop frames share one transfer stack. `break` exits its nearest frame;
`continue` skips switch frames to the nearest loop's condition or for-post label.
A switch alone does not make `continue` legal.

Case statements share the incoming **semantic** scope and symbol IDs; only real
blocks create child scopes. TAC conservatively requires shared bindings to be
initialized before the switch. A case may initialize and use its own local
declaration, but later cases and code after the switch cannot read or assign that
case-only binding. This restriction also applies to default-only switches and to
declarations inside nested switches sharing the same scope. Such uses report
`TAC_UNSUPPORTED` in phase `codegen` with the located message
`switch binding is not guaranteed initialized on this path`.

This is a code-generation capability restriction, not a change to language scope
or analysis reports. Blocked names cannot silently fall back to an outer binding
shadowed by an earlier case declaration. A declaration's own initializer still
reads the outer name before activation. Initialized outer bindings remain readable
and writable across cases and nested blocks; no default values are invented.

The switch example computes `total = 6` without print. It demonstrates ordered
dispatch, implicit exit, switch break and continue through a switch to a for post.

### Supported source

The implemented source slice supports initialized integer/float/boolean variables
and constants, identifier assignment, lexical blocks, grouping, unary minus,
and `+`, `-`, `*`, `/`, comparisons, boolean logic, `if/else`, and the loops/switch/transfers above.
Unsupported statements, operators, expressions, types,
and missing initializers must be reported explicitly with source locations.
No partial TAC may escape after lexical, syntax, semantic, or lowering errors.
Original source bytes are preserved; Compile retains the matching Analyze diagnostics/scopes.
Numeric expression statements are supported; their final temporary is released
without a store. Except for the for-post case above, assignment expressions
(including property assignments), ternaries, strings, null, lists, objects/classes,
print, functions, calls and foreach are not lowered. Transfers without an enclosing
eligible construct are rejected. Unsupported input is not silently skipped.
Unsupported declarations report `only integer, float and boolean declarations are supported`.
Legacy unsupported-expression/statement and missing-initializer diagnostics retain
their numeric wording for compatibility; all supported scalar declarations require
an initializer.

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
attempting code generation. The default command renders the same TAC; JSON
exposes the complete report.

## Remaining Entrega 2 work

- Foreach lowering and stronger definite-initialization analysis for switch bindings.
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
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test ./cmd/compiscript-tac -count=1
```

Current tests pin instruction formatting, deterministic program output, live
temporary uniqueness, release protection, type-changing reuse, and bounded
allocation across repeated expressions. Source-level tests pin precedence,
associativity, widening, division, lexical binding, exact typed/source-located
instructions, rejection without partial output, deterministic JSON, input
preservation, and compatibility with analysis diagnostics. CLI tests execute
the built binary for exact TAC/report output, repeated runs, located rejection,
exit codes, argument/input errors, and unchanged files. Complete Entrega 2
coverage is not claimed by this subset. Conditional tests execute generated
instructions with a finite step budget, validate labels/targets and defined
temporaries, and check skipped/demanded division traps, truth tables, nested joins,
branch scopes, source preservation and deterministic bounded temporary allocation.
Loop tests use the same bounded interpreter for optional clauses, backedges,
nearest transfers, scope shadowing, numeric promotion and short-circuit traps.
Switch tests reuse that interpreter for dispatch order, selector lifetime, skipped
traps, implicit exit, nested transfers, case initialization restrictions and bounded
temporary pressure. CLI tests cover text/JSON success and atomic skipped-body rejection.
