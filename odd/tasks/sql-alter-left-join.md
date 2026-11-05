# SQL ALTER TABLE and LEFT JOIN
Objective: implement the missing SQL operations authorized below. Tracker: `feat/sql-alter-left-join`, based on main `d478d67`.
Delivery: auto-chain / feature-branch-chain; forecast 800–1200 authored lines, coherent behavior slices; runner: `go test ./internal/sql ./cmd/sql` and SQL API tests.

## Specs
S1. Original request: "ayudame, en la rama main deje la feature de sql, pero necesito que pruebes casos de crecion de tablas, joins alteres, edicion, eleimnacion edicion, que las tablas si se eliminen y asi". Follow-up authorization: "dale lo que falta". Context authorized by that follow-up: implement the reported missing `ALTER TABLE` operations (ADD COLUMN, DROP COLUMN, RENAME TO), preserving existing records and indexes except deliberately removed columns and their indexes. Check via the public engine/CLI on isolated databases and reopening.
S2. "dale lo que falta" also refers to the reported unsupported `LEFT JOIN`. Implement LEFT JOIN and standard LEFT OUTER JOIN with NULL extension and correct ON versus WHERE semantics; preserve existing INNER JOIN/JOIN results. Check matching, unmatched, empty-right, multi-join, aggregate and indexed cases through public queries.
S3. "Ramas encadenadas por feature (recomendado)". Keep implementation on feature branches from main, with code/tests/docs in each work unit; no push, PR, merge-to-main, or changes to the user's real `sqldata`.
Assumptions (implementation defaults, not extra user requirements): added nullable columns fill existing records with NULL; reject adding a NOT NULL/PRIMARY KEY column to nonempty data without a default (DEFAULT is outside this request); reject dropping the final column; drop indexes only for explicitly dropped columns, preserve/rebuild other indexes, and retarget indexes on rename. Preserve currently valid identifier names by parsing added words contextually rather than globally reserving them. Preflight and ordinary filesystem failures must not leave partial catalog/data changes; no new multi-process concurrency or cross-statement transaction guarantee.

## Tasks
T1 | S1-S3 | delegated parallel writers in isolated same-clone worktrees (ALTER and LEFT JOIN are independent multi-file behavior units) | in_progress | commits pending; each writer owns its tests/docs, observes RED/GREEN, self-reviews.
T2 | S1-S3 | inline integration and one independent high-risk verifier | pending | commit pending; verify data preservation, physical deletion, indexes, old behavior and public CLI outputs.
T3 | S1-S3 | inline closure and evidence | pending | commit pending; finalize branch chain and record every failed/skipped check.

## Log
L1. User original request verbatim: "ayudame, en la rama main deje la feature de sql, pero necesito que pruebes casos de crecion de tablas, joins alteres, edicion, eleimnacion edicion, que las tablas si se eliminen y asi"
L2. Follow-up verbatim: "dale lo que falta". The preceding response identified ALTER TABLE ADD/DROP COLUMN/RENAME and LEFT JOIN as unsupported.
L3. Delivery choice verbatim: "Ramas encadenadas por feature (recomendado)". Tracker from main; child 1 ALTER, child 2 LEFT JOIN. Parallel preparation uses isolated worktrees because AST/parser edit surfaces overlap. Parent integrates only after both return. Each behavior is a rollback boundary; commits may exceed the advisory 400-line heuristic to keep runnable behavior with tests/docs, reported honestly without code-golf or artificial splits.
L4. Baseline: 55 SQL tests and 5 SQL API tests passed on main; CLI rejected all three ALTER forms and LEFT JOIN with exit 1 syntax diagnostics. Existing verification evidence: `.sql-check-main.g7LD4x/`. Browser was not verified; web SQL test glob discovered zero tests.
L5. Risk: item 1 (ALTER rewrites existing stored records and schema), item 3 (shared parser/query helpers serve existing SQL). Independent verification required after self-checks; RDD is off, assessment only.
