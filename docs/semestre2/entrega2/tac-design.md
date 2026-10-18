# Compiscript three-address code

This is the scalar, control-flow and direct-global-function subset of Entrega 2,
not the complete assignment. Functions use explicit calls/returns and deterministic
abstract-cell activation records; recursion and closures remain deferred.
The separate `compiscript.Compile(source []byte)` library API generates typed,
source-located TAC for numeric and boolean programs with `if/else`, `while`,
`do-while`, `for`, `switch`, typed direct calls and returns. Existing analysis reports remain unchanged except for the
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
go run ./cmd/compiscript-tac testdata/compiscript/tac/functions.cps
go run ./cmd/compiscript-tac -format json testdata/compiscript/tac/functions.cps
go run ./cmd/compiscript-tac -h
```

The default format writes only TAC text to stdout. Rejected compilation writes
located diagnostics to stderr, no TAC to stdout, and exits 1. JSON format writes
the complete `CompilationReport` to stdout, including `tac: null` on rejection;
its exit code is still 1 when diagnostics exist. Success exits 0, including an
empty source. Invalid arguments or format exit 2; file/UTF-8/write failures
exit 1; help exits 0 and prints usage to stderr. Flags precede the source path.
`go run` can wrap a program's nonzero exit; build the binary when an automation
needs the command's exact status. Scope snapshots remain semantic metadata;
procedure frame layouts are separate code-generation metadata described below.

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
| Call | `t0:integer = call s1.f(2:integer)` | Evaluate caller-frame argument values, invoke the named procedure with fresh storage, resume and write its scalar result |
| Null-result call | `call s1.update()` | Invoke a procedure as a statement, without a result destination |
| Return | `return t0:integer` or `return` | Transfer a scalar result, or no value, to the saved caller continuation |

Label/jump instructions use `label`; branch-true also uses `left`. Their unused
legacy `target` operand is zero-valued; label/jump `left` is also zero-valued,
and `right` is omitted. These fields are not value reads or writes. Labels start
at `L0` and are allocated deterministically per instruction tape (main or procedure). Existing numeric
instruction text and JSON remain unchanged.

Each instruction reads all source operands before writing its destination.
Consequently, a destination may reuse the last-consumed operand's temporary.
An integer denotes an abstract exact integer; target-specific width and ABI
are not selected here. Floats denote IEEE-754 binary64 values; integer widening
may round. Mixed operations widen integers explicitly. Division widens integer
operands and produces a float, matching the semantic analyzer. No constant
folding is performed; division by zero is a runtime fault for a future backend,
not permission to omit instructions. There is no product TAC executor in this unit.

## Binding and lifetimes

The source compiler assigns symbol names `s<scope-id>.<source-name>` from
the semantic scope inventory. These are logical identities, not byte offsets,
and are deterministic for the same source, not stable across source edits.
Declarations become visible only after their initializer is lowered. This
preserves outer-variable binding in a shadowing initializer. Global function
identities/signatures are predeclared for forward calls; variable bindings are not.
Function scopes and activation records are described below; captures are not supported.

Temporaries start at `t0`. Allocation chooses the smallest available numeric
slot; release occurs only once the final consuming instruction is emitted.
Every occurrence retains its own type, so a free slot may be reused for a
different type. Live operands cannot overlap; repeated statements do not
reserve fresh slots indefinitely. `peakTemporaries` records peak live demand,
not physical register usage, stack size, or a claim of global optimality.

## Direct functions and abstract activation records

### Signatures, binding and validation

Only top-level global functions with explicitly typed `integer`, `float` or
`boolean` parameters are supported. Their result is a scalar type or the
analyzer's `null` result when omitted. For example:

```text
function increment(value: integer): integer {
    value = value + 1;
    return value;
}
let answer = increment(2);
```

Calls bind to an actual predeclared global function symbol through the active
lexical environment, never by a name-only registry lookup. Aliases, first-class
function values, indirect calls and shadowed function names are not callable.
The callee expression must be an identifier. Nested declarations/closures,
bodyless external AST functions and untyped/unsupported signatures are explicitly
rejected. A function uses the existing semantic function scope spanning its whole
declaration; its parameters and direct body declarations share that scope. Real
nested blocks and for scopes retain their own semantic IDs. Parameters are mutable
invocation-local values, not aliases to caller symbols.

Every body is lowered once in source order, including uncalled and unreachable
code, with an independent existing temporary pool and a fresh transfer stack.
All direct call edges are collected without recursively lowering callees. After
lowering, a deterministic graph walk rejects direct or mutual cycles, including
cycles in uncalled bodies. This slice does not execute or support recursion.

Scalar returns use existing type compatibility and explicit `i2f` widening;
there is no synthesized scalar default. Semantic missing-return diagnostics
remain authoritative. Null-result procedures accept bare return, return-null
(also grouped null), and fallthrough; fallthrough emits an explicit no-value
return. Null-result calls are statement-only, including grouped call statements;
they cannot be used as values, even as the operand of another null return. This
is a code-generation restriction, not a new `void` type or semantic policy.

### Evaluation and global initialization

Operands and arguments evaluate left to right, once. Before evaluating a later
call subtree, an earlier symbol operand is copied into a reserved temporary.
This applies to arithmetic, comparisons, argument lists and a switch selector
before effectful case tests. Literals and already-live temporaries need no copy.
For `let g=1; ... let x=g+bump();`, where `bump` increments and returns `g`, `x`
is 3, not 4. Similarly, `pair(g,bump())` receives 1 and 2, not 2 and 2. Widening
occurs before later arguments are evaluated. Logical `&&`/`||` retain runtime
short-circuit skipping while both operands are statically validated. Programs
without calls receive no extra snapshot instructions.

Function bodies retain declaration-time global visibility and switch-blocked
names, not the final variable inventory. Global reads and writes are allowed
when initialized; the absence of a semantic `Captured` flag on globals does not
imply closure support. Each procedure records direct global references, then
inherits its callees' dependencies transitively through the acyclic graph.
Each main call records the initialized global bindings available at its own
call site. After graph resolution, a missing binding rejects compilation with
`call requires initialized global binding s<scope>.<name>` at that call.
Reads **and writes**, including statically skipped references, conservatively
require initialization; no zero/default value is invented.

A caller's declaration need not see a callee's transitive globals: a forward
callee may be declared after those globals initialize. For example, a function
`first` declared before global `g` can call a later function `later` whose body
reads `g`; calling `first` after `g` initializes is safe. Calling it before that
initializer is rejected. A direct read of later `g` in `first` itself is still
subject to the existing semantic declaration-order visibility rule.

### Procedure and instruction metadata

`Program.instructions` remains the executable main tape. Function declarations
do not execute inline. Optional `procedures` is appended after the legacy Program
fields and omitted entirely for programs without functions. Each procedure
contains, in order: `label`, `scopeId`, `span`, ordered `parameters`, `resultKind`,
`frame`, `instructions`, `peakTemporaries`, and optional sorted transitive `globals`.
Labels use the global symbol identity (`s1.increment`); branch labels are local
to their tape. The procedure span is the declaration span, and all operations
retain their originating source spans.

Instruction fields retain their old order and tags, followed by optional
`callee`, `arguments` and `returnValue`. Call arguments are caller-frame operands
in formal order; a scalar call has a typed temporary `target`, and a null-result
call has a zero-valued unused target. A return uses `returnValue` when scalar;
unused legacy target/left fields are zero-valued and are not reads/writes.
Empty argument lists and absent return values are omitted in JSON.

All argument values must be read/copied **before** constructing or entering the
callee frame, even when caller and callee use the same `t0` name. The caller's
live cells remain reserved across calls. On return, read the result before
discarding the callee, restore the caller/continuation, and write the caller's
destination. Each invocation has fresh parameter/local/temporary storage;
only the global store is shared. Calls pass scalars by value.

### Cell layout and text rendering

`frame.cells` is the total cell count. `frame.slots` lists contiguous zero-based
`offset`, `size: 1`, `role`, optional owned `operand` and optional `dynamic` in
this exact order:

| Order | Role | Contents |
| --- | --- | --- |
| 0 | `continuation` | Return address / suspended caller instruction position |
| 1 | `controlLink` | Calling activation |
| 2 | `globalAccessLink` | Shared global store only, not a closure environment |
| next, scalar only | `result` | Scalar return value; absent for null results |
| next | `parameter` | Ordered formal operands, with fixed scalar types |
| next | `local` | All function-owned variables/constants, including block/for locals, sorted by declaration source offset with operand-identity tie-break |
| last | `temporary` | `t0` through `t<peak-1>`, dynamically typed cells |

All cell sizes and offsets are **abstract cells**, not bytes, machine words,
physical registers or a completed target ABI. Parameter/local operand names
encode semantic scope ownership. A temporary's slot operand has an empty type
and `dynamic: true`: its individual instruction occurrences carry their types,
because the existing pool can reuse a cell first as integer, then boolean or
float. No second allocator is introduced. Main does not acquire fictitious
procedure-frame metadata.

With procedures present, text begins `main:` and then prints main instructions,
followed by `procedure <label>(<formals>) -> <result> [scope=<id> frame=<n> cells]:`,
one `  cell <offset> <role> [operand] [dynamic]` line per slot, the procedure
instructions, and `end <label>`. Without procedures, old text and JSON (including
field ordering, temporary numbering and layout absence) are unchanged.

`functions.cps` terminates without print and would compute `total = 7` and
`result = 8.0`. The CLI emits its TAC, not execution output. Bounded test-only
frame execution checks independent invocation storage, continuation/control
restoration, global effects, defined slots, ownership and label/step/depth limits.

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
Any error discards the whole TAC program. Switch dispatch itself uses the existing
opcodes; its case expressions may now contain scalar direct calls.

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
and `+`, `-`, `*`, `/`, comparisons, boolean logic, `if/else`, the loops/switch/transfers above,
and the direct-function/call/return subset described above.
Unsupported statements, operators, expressions, types,
and missing initializers must be reported explicitly with source locations.
No partial TAC may escape after lexical, syntax, semantic, or lowering errors.
Original source bytes are preserved; Compile retains the matching Analyze diagnostics/scopes.
Numeric expression statements are supported; their final temporary is released
without a store. Except for the for-post case above, assignment expressions
(including property assignments), ternaries, strings, general null values, lists, objects/classes,
print, indirect/nested/recursive functions and foreach are not lowered. Transfers without an enclosing
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
- Recursion, indirect/first-class calls, nested functions and closures. Capture
  analysis must retain binding edges; a `Captured` flag alone is insufficient.
- Object allocation, constructors, fields, methods, `this`, lists and indexing.
- Target-machine byte sizes/offsets and calling ABI, main/global storage layout,
  non-global lexical access links and closure environments. Abstract function
  parameter/local/temporary/result/control cells are implemented, not this target layout.
- Source-to-TAC output in the IDE and successful/failed end-to-end examples.

Future memory access needs distinct opcodes and explicit operands, as direct calls
already have; operations must not be smuggled into numeric symbol strings.
Closure environments need their own binding-edge design and executable proof.
The inherited frontend panic for some malformed for headers is unchanged; this
slice adds no blanket recovery and does not claim all malformed input is covered.

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
Function tests execute separate tapes with fresh abstract frames, caller-value
freezing, mutable by-value parameters, widening, null/scalar returns, nested control
flow and transitive global effects. They pin deterministic layouts/JSON/text,
dynamic temporary cells, immutable source/AST/scopes and atomic function/graph/
initialization rejection, including uncalled bodies. The prior condition oracle
is unchanged; the new frame oracle is test-only and does not claim recursion support.
