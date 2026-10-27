# SQL interpreter, stages 1-4 (Entrega 3)
Objective: SQL lexer, parser, AST, semantic analysis, and CREATE/DROP/INSERT over file storage, integrated into this repo and its IDE.
Branch: main (user asked earlier for direct work on main). Delivery: single slice. Test runner: `go test ./...`.

## Specs
- S1: Lexer and parser for the SQL subset listed in the statement ("SQL Requerido"), no external dependency (dependency guard test forbids new ones).
- S2: AST for every required statement, plus a parse tree view.
- S3: Semantic analysis: "Tabla inexistente", "Columna inexistente", "Tipos incompatibles", "Valores incorrectos en INSERT", "Referencias ambiguas o inválidas", "Índice inexistente o duplicado", "Subconsulta escalar que retorna más de una columna". "Los errores deberán indicar claramente el problema y su ubicación."
- S4: Execution of CREATE TABLE, DROP TABLE, INSERT. Storage: only files and directories (`database/metadata.txt`, `<tabla>.csv`). Types INT, FLOAT, VARCHAR(n), BOOLEAN, DATE (YYYY-MM-DD) stored in the schema.
- S5: GUI: write SQL, view tokens, parse tree, AST, lexical/syntax/semantic errors, execute, view results.
- Out of scope (stages 5-8): executing SELECT, UPDATE, DELETE, indexes, joins, aggregates, subqueries. They are parsed and semantically checked only.

## Tasks
- [x] T1 (S1,S2) lexer + parser + AST + trees: inline; `go test ./internal/sql` green
- [x] T2 (S3,S4) catalog, semantic, storage, engine: inline; 38 semantic cases + persistence tests green
- [x] T3 (S5) HTTP endpoint, web page, CLI: inline; live server smoke test OK (sql.html, sql.js, /api/sql/run)
- [x] T4 docs (Spanish) in docs/semestre2/entrega3/README.md + README section
- Commits: none yet (user has not asked to commit)

## Log
- L1 (2026-10-26): "ayudame, hay que terminar la siguente entrega, la docu esta en docs/semestre2/entrega3/Proyecto_2_1__Compiladores-2.md"
- L2: "va integrado en este" ; "dale con las etapas 1 a 4"
