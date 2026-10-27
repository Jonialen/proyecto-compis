# SQL interpreter, all stages (Entrega 3)
Objective: SQL lexer, parser, AST, semantic analysis, and CREATE/DROP/INSERT over file storage, integrated into this repo and its IDE.
Branch: main (user asked earlier for direct work on main). Delivery: single slice. Test runner: `go test ./...`.

## Specs
- S1: Lexer and parser for the SQL subset listed in the statement ("SQL Requerido"), no external dependency (dependency guard test forbids new ones).
- S2: AST for every required statement, plus a parse tree view.
- S3: Semantic analysis: "Tabla inexistente", "Columna inexistente", "Tipos incompatibles", "Valores incorrectos en INSERT", "Referencias ambiguas o inválidas", "Índice inexistente o duplicado", "Subconsulta escalar que retorna más de una columna". "Los errores deberán indicar claramente el problema y su ubicación."
- S4: Execution of CREATE TABLE, DROP TABLE, INSERT. Storage: only files and directories (`database/metadata.txt`, `<tabla>.csv`). Types INT, FLOAT, VARCHAR(n), BOOLEAN, DATE (YYYY-MM-DD) stored in the schema.
- S5: GUI: write SQL, view tokens, parse tree, AST, lexical/syntax/semantic errors, execute, view results.
- S6 (stages 5-8, added by L3): execute SELECT, WHERE, AND, OR, ORDER BY, UPDATE, DELETE; INNER JOIN, COUNT/SUM/AVG/MIN/MAX, GROUP BY, HAVING; CREATE INDEX / DROP INDEX stored in files, used for WHERE on an indexed column ("el intérprete deberá utilizar el índice en lugar de recorrer todos los registros") and kept updated by INSERT, UPDATE and DELETE; subqueries in WHERE: IN, NOT IN, EXISTS, NOT EXISTS and scalar comparisons ("Subconsulta escalar que retorna más de una columna o más de un registro"); integration and final delivery.

## Tasks
- [x] T1 (S1,S2) lexer + parser + AST + trees: inline; `go test ./internal/sql` green
- [x] T2 (S3,S4) catalog, semantic, storage, engine: inline; 38 semantic cases + persistence tests green
- [x] T3 (S5) HTTP endpoint, web page, CLI: inline; live server smoke test OK (sql.html, sql.js, /api/sql/run)
- [x] T4 docs (Spanish) in docs/semestre2/entrega3/README.md + README section
- Commits: 6333fe3 (core), 3c2ba5e (IDE), 801715b (docs)
- [x] T5 (S6) SELECT/WHERE/ORDER BY/UPDATE/DELETE execution: inline
- [x] T6 (S6) JOIN + aggregates + GROUP BY/HAVING: inline
- [x] T7 (S6) index files, index use in WHERE, maintenance on DML; subqueries at runtime: inline (high-risk item 1: persisted format; tests compare indexed vs full scan)
- [x] T8 (S6) integration: CLI/IDE results, examples with verified goldens, docs; Chrome headless screenshot of sql.html OK

## Log
- L1 (2026-10-26): "ayudame, hay que terminar la siguente entrega, la docu esta en docs/semestre2/entrega3/Proyecto_2_1__Compiladores-2.md"
- L2: "va integrado en este" ; "dale con las etapas 1 a 4"
- L3 (2026-10-26): "vamos con el rerto de las etapas"
- Evidence (stages 5-8): go test ./... green; external cross-check of the tienda example (21 SELECT + UPDATE/DELETE counts), 3600 random SELECTs and 240 table states after random writes against an independent engine used only as an offline test oracle (outside the repo): 0 mismatches. Bug found and fixed by tests: empty strings in one-column tables were lost (blank CSV lines are skipped by readers); now written as "".
