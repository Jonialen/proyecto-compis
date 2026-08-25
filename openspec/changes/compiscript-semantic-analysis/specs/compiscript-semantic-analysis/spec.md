# Compiscript Semantic Analysis Specification

## Purpose

Define `.cps` parsing, semantic analysis, reporting, and consumers.

## Requirements

### Requirement: Parsing and Reports

The system MUST parse `.cps` into a source-located, project-owned AST, emit located lexical/syntax diagnostics, analyze recoverable regions, and return a deterministic `AnalysisReport` with visual AST, accumulated diagnostics, and scope snapshots.

#### Scenario: Valid and recoverable source
- GIVEN valid and recoverably malformed `.cps`
- WHEN analyzed repeatedly
- THEN valid output is stable and malformed output accumulates applicable located diagnostics

### Requirement: Type Compatibility

Types MUST include `integer`, `float`, `boolean`, `string`, `null`, and `ErrorType`. Mixed numeric arithmetic/comparison MUST promote to `float`; division MUST yield `float`. `+` MUST allow numeric pairs or two strings only; logic MUST require booleans. Equality MUST allow same types, mixed numerics, and permitted null pairs; ordering MUST require numerics. Assignments MUST allow same types, integer-to-float, and null-to-string/list/class. `ErrorType` MUST propagate and suppress dependent diagnostics. Constants MUST initialize and remain immutable.

#### Scenario: Compatible and incompatible expressions
- GIVEN valid promotion, concatenation, and null assignment beside invalid addition, logic, narrowing, and constant assignment
- WHEN checked
- THEN valid types follow these rules and independent violations are diagnosed without cascades

### Requirement: Environments and Names

The system MUST use nested environments, reject same-scope duplicates and unresolved names, permit nested shadowing, and deterministically snapshot each scope's symbols and types.

#### Scenario: Resolution policy
- GIVEN shadowing, a duplicate, and an unresolved reference
- WHEN resolved
- THEN nearest declarations win and both invalid names receive located diagnostics

### Requirement: Functions and Closures

Functions MUST enforce arity, positional types, return context/type, and all-path returns for non-null results. Recursion, nesting, and closure capture MUST be valid.

#### Scenario: Function validity
- GIVEN a recursive closure plus wrong arguments, external return, and missing return path
- WHEN analyzed
- THEN the closure passes and each violation is diagnosed

### Requirement: Control Flow

`if`/loop conditions MUST be boolean. `switch` cases MUST be compatible and statically unique, without fallthrough. `break` MUST occur only in loops/switches, `continue` in loops, and `return` in functions. Every statement after guaranteed transfer MUST receive an unreachable diagnostic. The grammar MUST accept `for (let i: integer = 0; i < 3; i = i + 1)`.

#### Scenario: Valid and invalid flow
- GIVEN valid `for`/switch constructs beside invalid conditions, cases, transfers, and unreachable statements
- WHEN analyzed
- THEN valid flow passes and every violation and unreachable statement is diagnosed

### Requirement: Collections and Classes

Lists/matrices MUST be homogeneous per level; ragged matrices MAY be valid; empty lists MUST receive contextual type. Indices MUST be integers; only provable bounds errors MUST be diagnosed. Classes MUST validate members, constructor arguments, `this`, single inheritance, and inherited lookup. Inherited names MUST NOT be redeclared; constructors MUST NOT be inherited.

#### Scenario: Aggregate checks
- GIVEN a ragged matrix and inherited lookup beside invalid collection/class uses
- WHEN analyzed
- THEN valid uses pass, provable violations are diagnosed, and uncertain bounds are ignored

### Requirement: Minimal Exception Semantics

`try/catch` MUST analyze both blocks and bind the catch identifier only inside its catch scope with the special exception type. The system MUST NOT model `throw`.

#### Scenario: Catch scope
- GIVEN a catch identifier referenced inside and outside its catch block
- WHEN analyzed
- THEN the inner reference has the exception type and the outer reference is unresolved

### Requirement: Consumers and Boundaries

CLI and IDE MUST expose equivalent reports through separate internal contracts and endpoints; the UI MAY change. Existing YALex/YAPar behavior MUST remain unchanged. Every semantic rule MUST have located valid and invalid evidence. SQL-oriented DSL behavior and speculative extension mechanisms MUST NOT be introduced.

#### Scenario: Acceptance boundary
- GIVEN one `.cps` corpus and existing YALex/YAPar workflows
- WHEN CLI, IDE, and acceptance evidence are verified
- THEN reports agree, every rule has paired evidence, prior behavior remains, and no SQL extension surface exists
