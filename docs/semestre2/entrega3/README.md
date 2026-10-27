# Entrega 3: intérprete SQL con almacenamiento en archivos

Alcance de esta entrega: etapas 1 a 4 de [Proyecto_2_1__Compiladores-2.md](Proyecto_2_1__Compiladores-2.md) (lexer y parser, AST, análisis semántico con manejo de errores, y `CREATE`, `DROP` e `INSERT`). El código vive en `internal/sql`, el CLI en `cmd/sql` y la interfaz en `web/sql.html`. No se usa ningún DBMS ni dependencia externa: el lexer y el parser son manuales y solo usan la biblioteca estándar de Go.

## Cómo ejecutarlo

Desde la raíz del repositorio:

```sh
go run ./cmd/sql -db /tmp/mibd testdata/sql/ok/demo.sql        # analiza y ejecuta
go run ./cmd/sql -db /tmp/mibd -analyze testdata/sql/ok/demo.sql  # solo analiza, no toca el disco
go run ./cmd/sql -format=json -analyze testdata/sql/ok/demo.sql   # tokens, árbol, AST y errores
go run ./cmd/ide                                            # interfaz en http://localhost:8080/sql.html
```

La interfaz trabaja sobre `sqldata/default/` (ignorado por Git). El servidor escucha en `:8080` y puede modificar esa carpeta: usarlo solo en un entorno local confiable. El cliente no elige la ruta de la base de datos.

## Flujo

`SQL → tokens (lexer) → árbol sintáctico y AST (parser) → análisis semántico → ejecución → archivos`

| Fase | Archivo | Resultado |
|---|---|---|
| Léxico | `internal/sql/lexer.go` | Tokens con línea/columna; errores léxicos (carácter no reconocido, cadena o comentario sin cerrar, número mal formado) |
| Sintáctico | `internal/sql/parser.go` | Árbol sintáctico y AST; recuperación por sentencia en el siguiente `;` |
| AST | `internal/sql/ast.go` | Nodos tipados y vista en árbol (`ASTTree`) |
| Semántico | `internal/sql/semantic.go` | Errores con línea y columna; los `CREATE`/`DROP` del mismo script son visibles para las sentencias siguientes |
| Ejecución | `internal/sql/engine.go`, `storage.go` | `CREATE TABLE`, `DROP TABLE`, `INSERT` sobre archivos |

Un script solo se ejecuta si no tiene errores léxicos, sintácticos ni semánticos; en ese caso no se escribe nada. Los errores de ejecución (por ejemplo, clave primaria duplicada) detienen el script en esa sentencia y las anteriores quedan aplicadas.

## Almacenamiento

```text
database/
|-- metadata.txt        esquema: tablas, columnas, tipos, restricciones e índices
|-- estudiantes.csv     una fila de encabezado y un registro por línea
`-- cursos.csv
```

`metadata.txt`:

```text
TABLE estudiantes
COLUMN id INT PRIMARY_KEY NOT_NULL
COLUMN nombre VARCHAR(50) NOT_NULL
COLUMN nacimiento DATE
END
```

- Tipos: `INT`, `FLOAT`, `VARCHAR(n)`, `BOOLEAN`, `DATE` (`YYYY-MM-DD`, validado como fecha real). El tipo se guarda en el esquema y se usa en el análisis semántico y en la ejecución.
- Restricciones: `PRIMARY KEY` (una por tabla, implica `NOT NULL`, se verifica la unicidad al insertar) y `NOT NULL`.
- En el CSV, `NULL` se escribe `\N` y un texto que empieza con `\` se escribe con una barra extra para no confundirse con `NULL`.
- Las escrituras de `metadata.txt` y de cada CSV son atómicas (archivo temporal y renombrado). Un `INSERT` es todo o nada.

## Errores semánticos cubiertos

| Requisito del enunciado | Ejemplo de mensaje |
|---|---|
| Tabla inexistente | `La tabla "nada" no existe` |
| Columna inexistente | `La columna "zz" no existe` |
| Tipos incompatibles | `Tipos incompatibles: no se puede asignar un valor VARCHAR a la columna "id" de tipo INT` |
| Valores incorrectos en `INSERT` | cantidad de valores, `NOT NULL` sin valor, `VARCHAR` demasiado largo, fecha inválida, división por cero |
| Referencias ambiguas o inválidas | `Referencia ambigua: la columna "id" existe en más de una tabla; califíquela…` |
| Índice inexistente o duplicado | `El índice "i" no existe`, `Índice duplicado: la columna …` |
| Subconsulta escalar con más de una columna | `Subconsulta escalar inválida: retorna 2 columnas…` |

Formato de cada error: `línea:columna: error <fase>: mensaje`, con fase `lexical`, `syntax`, `semantic` o `execution`. La interfaz muestra los mismos datos en la pestaña Errores.

## Sintaxis reconocida

El parser reconoce todo el SQL requerido por el enunciado (`SELECT` con `WHERE`, `AND`, `OR`, `ORDER BY`, `INNER JOIN`, `COUNT/SUM/AVG/MIN/MAX`, `GROUP BY`, `HAVING`, `IN`, `NOT IN`, `EXISTS`, `NOT EXISTS`, subconsultas escalares, `UPDATE`, `DELETE`, `CREATE INDEX`, `DROP INDEX`) y el análisis semántico también lo revisa. Solo `CREATE TABLE`, `DROP TABLE` e `INSERT` se ejecutan; las demás sentencias terminan con un error de ejecución que lo indica. Eso corresponde a las etapas 5 a 7.

## Ejemplos de verificación

Los ejemplos están en `testdata/sql/` y los ejecuta `go test ./internal/sql` (`examples_test.go`). Cada uno compara el resultado exacto contra un archivo guardado junto a él.

| Carpeta | Contenido | Se compara con |
|---|---|---|
| `ok/` | Scripts que se ejecutan completos: `escuela.sql` (3 tablas, los 5 tipos, `NULL`, comillas escapadas, acentos, expresiones, columnas parciales), `tipos.sql` (límites de cada tipo y de la codificación CSV), `claves.sql` (`PRIMARY KEY`, nombres en distinta capitalización, `DROP` y recreación), `demo.sql` | `.golden`: contenido de `metadata.txt` y de cada CSV |
| `analyze/` | `consultas.sql`: 27 sentencias complejas (joins, `GROUP BY`/`HAVING`, `IN`/`EXISTS` y subconsultas anidadas o correlacionadas, `UPDATE`/`DELETE` con subconsultas, índices); solo se analizan | cero errores y 27 sentencias en el AST |
| `errors/` | Un archivo por fase: `lexico_*`, `sintaxis`, `semantico_insert`, `semantico_consultas`, `semantico_indices`, `ejecucion_pk`, `ejecucion_select` | `.expected`: línea, columna, fase y mensaje de cada error |

Si cambia un mensaje a propósito, regenerar con `go test ./internal/sql -update` y revisar el diff a mano antes de confirmarlo.

Los errores léxicos pueden provocar un error sintáctico a continuación (por ejemplo, una cadena sin cerrar deja `VALUES (1,` incompleto). Es esperado: el token ilegal se descarta y el parser informa lo que falta.

## Matriz de requisitos

| Etapa | Requisito | Evidencia |
|---|---|---|
| 1 | Lexer y parser | `TestLex*`, `TestParse*` en `internal/sql/sql_test.go` |
| 2 | AST | `TestParseAllRequiredStatements`, pestaña AST de la interfaz |
| 3 | Semántico y manejo de errores | `TestSemanticErrors` (38 casos), `examples_test.go`, `TestSemanticReportsLocation`, `TestSemanticAcceptsValidQueries` |
| 4 | `CREATE`, `DROP`, `INSERT` | `TestExecuteCreateInsertDropPersistsInFiles`, `TestPrimaryKeyViolationIsAtomic`, `TestExecutionRefusesScriptsWithErrorsAndRunsNothing` |
| GUI | Tokens, árbol, AST, errores, ejecutar, resultados | `cmd/ide/sql_test.go` y `web/sql.html` |

Verificación: `go test ./...`.

## Límites conocidos

- `SELECT`, `UPDATE`, `DELETE`, índices (archivos y uso en `WHERE`), `JOIN`, agregaciones y subconsultas no se ejecutan todavía (etapas 5 a 7). `CREATE INDEX` y `DROP INDEX` se analizan, pero no crean archivos.
- Que una subconsulta escalar devuelva más de un registro solo puede detectarse al ejecutar, así que queda para la etapa de ejecución de `SELECT`.
- La unicidad de `PRIMARY KEY` se verifica al ejecutar, no al analizar, porque depende de los datos.
