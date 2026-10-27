# Entrega 3: intérprete SQL con almacenamiento en archivos

Implementa las ocho etapas de [Proyecto_2_1__Compiladores-2.md](Proyecto_2_1__Compiladores-2.md): lexer y parser, AST, análisis semántico con manejo de errores, `CREATE`/`DROP`/`INSERT`, `SELECT`/`WHERE`/`UPDATE`/`DELETE`, `JOIN`/agregaciones/`GROUP BY`, índices y subconsultas, e integración. El código está en `internal/sql`, el CLI en `cmd/sql` y la interfaz en `web/sql.html`. No se usa ningún DBMS ni dependencia externa: el lexer y el parser son manuales y solo usan la biblioteca estándar de Go.

## Cómo ejecutarlo

Desde la raíz del repositorio:

```sh
go run ./cmd/sql -db /tmp/mibd testdata/sql/ok/enunciado.sql      # analiza y ejecuta (ejemplos del enunciado)
go run ./cmd/sql -db /tmp/mibd -analyze testdata/sql/ok/enunciado.sql # solo analiza, no toca el disco
go run ./cmd/sql -db /tmp/mibd -noindex consulta.sql              # ignora los índices (para comparar el plan)
go run ./cmd/sql -format=json -analyze consulta.sql               # tokens, árbol, AST y errores
go run ./cmd/ide                                                   # interfaz en http://localhost:8080/sql.html
go test ./...                                                      # todas las pruebas
```

La interfaz trabaja sobre `sqldata/default/` (ignorado por Git). El servidor escucha en `:8080` y puede modificar esa carpeta: usarlo solo en un entorno local confiable. El cliente no elige la ruta de la base de datos y las ejecuciones se serializan.

## Flujo

`SQL → tokens (lexer) → árbol sintáctico y AST (parser) → análisis semántico → ejecución → archivos`

| Fase | Archivo | Resultado |
|---|---|---|
| Léxico | `internal/sql/lexer.go` | Tokens con línea/columna; errores léxicos (carácter no reconocido, cadena o comentario sin cerrar, número mal formado) |
| Sintáctico | `internal/sql/parser.go` | Árbol sintáctico y AST; recuperación por sentencia en el siguiente `;` |
| AST | `internal/sql/ast.go` | Nodos tipados y vista en árbol (`ASTTree`) |
| Semántico | `internal/sql/semantic.go` | Errores con línea y columna; los `CREATE`/`DROP` del mismo script son visibles para las sentencias siguientes |
| Consultas | `internal/sql/query.go` | Evaluador de expresiones, `JOIN`, `GROUP BY`/agregaciones, `ORDER BY`, subconsultas, elección de índice |
| Escrituras | `internal/sql/dml.go`, `engine.go` | `CREATE`/`DROP TABLE`, `INSERT`, `UPDATE`, `DELETE`, `CREATE`/`DROP INDEX` |
| Archivos | `internal/sql/storage.go`, `index.go`, `catalog.go` | `metadata.txt`, tablas CSV e índices |

Un script solo se ejecuta si no tiene errores léxicos, sintácticos ni semánticos; en ese caso no se escribe nada. Los errores de ejecución (clave primaria duplicada, división por cero, subconsulta escalar con más de un registro, desbordamiento) detienen el script en esa sentencia; las anteriores quedan aplicadas y la sentencia fallida no cambia ningún archivo.

## Almacenamiento

```text
database/
|-- metadata.txt          esquema: tablas, columnas, tipos, restricciones e índices
|-- estudiantes.csv       una fila de encabezado y un registro por línea
|-- cursos.csv
`-- idx_edad.idx          un archivo por índice
```

`metadata.txt`:

```text
TABLE estudiantes
COLUMN id INT PRIMARY_KEY NOT_NULL
COLUMN nombre VARCHAR(50) NOT_NULL
COLUMN nacimiento DATE
END
INDEX idx_edad ON estudiantes(edad)
```

- Tipos: `INT`, `FLOAT`, `VARCHAR(n)`, `BOOLEAN`, `DATE` (`YYYY-MM-DD`, validado como fecha real). El tipo se guarda en el esquema y se usa en el análisis semántico y en la ejecución.
- Restricciones: `PRIMARY KEY` (una por tabla, implica `NOT NULL`, se verifica la unicidad en `INSERT` y `UPDATE`) y `NOT NULL`.
- En el CSV, `NULL` se escribe `\N`, la cadena vacía `""`, y un texto que empieza con `\` lleva una barra extra para no confundirse con `NULL`.
- Las escrituras de `metadata.txt`, de cada CSV y de cada índice son atómicas (archivo temporal y renombrado). `INSERT`, `UPDATE` y `DELETE` modifican la tabla completa o nada.

## Índices

`CREATE INDEX nombre ON tabla (columna)` crea `nombre.idx`; `DROP INDEX nombre` lo elimina (y `DROP TABLE` elimina los de su tabla).

```text
#INDEX,idx_edad,estudiantes,edad,<sello>,5     encabezado: nombre, tabla, columna, sello del CSV, registros
17,27
18,162
20,54,189                                      clave ordenada, posiciones (bytes) de los registros en el CSV
```

- **Uso:** en un `WHERE` (también dentro de subconsultas, de un `JOIN` y de `UPDATE`/`DELETE`), las condiciones `columna = c`, `<`, `<=`, `>`, `>=` y `columna IN (c1, c2, …)` con constantes, unidas por `AND`, se resuelven con búsqueda binaria sobre el índice y se leen del CSV solo los registros que coinciden (por su posición en el archivo). Si hay varias columnas indexadas se prefiere la que tiene igualdad o `IN`. El resultado de cada `SELECT` incluye el **plan**, por ejemplo: `Índice "idx_edad" usado en "estudiantes" por edad >= 18 AND edad < 22: 3 de 5 registros leídos`; sin índice dice `Recorrido completo de "estudiantes": 5 registros`. En `UPDATE`/`DELETE` el archivo se reescribe de todos modos, así que el índice reduce los registros evaluados.
- **Mantenimiento:** después de cada `INSERT`, `UPDATE` o `DELETE` se reescriben los índices de la tabla, de modo que siempre coinciden con ella.
- **Robustez:** el encabezado guarda el tamaño y la fecha de modificación del CSV. Si no coinciden (se editó la tabla fuera del intérprete o la ejecución se interrumpió entre ambas escrituras) o el archivo falta o está dañado, el índice se reconstruye antes de usarse, nunca se confía en él.
- Los valores `NULL` no se indexan, porque ninguna comparación los selecciona.

## Semántica de ejecución

- Lógica de tres valores: comparar con `NULL` da `NULL`, y `WHERE` conserva solo las filas con resultado verdadero. `x NOT IN (…)` es `NULL` si la lista contiene `NULL`.
- Dos enteros operados dan un entero (la división trunca: `1 / 2` es `0`); si interviene un `FLOAT` el resultado es `FLOAT`. Los desbordamientos y la división por cero son errores de ejecución con su posición.
- Sin `ORDER BY` el resultado sale en el orden de la tabla, con o sin índice (el índice nunca cambia el orden). `ORDER BY` pone los `NULL` primero en orden ascendente (y últimos en descendente); sin `ORDER BY`, el orden es el de la tabla (en un `JOIN`, el de los bucles anidados).
- Agregaciones: `COUNT(*)` cuenta filas, las demás ignoran `NULL`; sobre una tabla vacía `COUNT` da `0` y las otras `NULL`. Las columnas fuera de una agregación deben estar en `GROUP BY` (error semántico en caso contrario).
- Subconsultas: `IN`, `NOT IN`, `EXISTS`, `NOT EXISTS` y escalares, anidadas y correlacionadas. Una subconsulta que no depende de la consulta exterior se evalúa una sola vez. Una escalar con más de una columna es error semántico; con más de un registro, error de ejecución.

## Errores

Formato: `línea:columna: error <fase>: mensaje`, con fase `lexical`, `syntax`, `semantic` o `execution`. La interfaz muestra los mismos datos en la pestaña Errores.

| Requisito del enunciado | Ejemplo de mensaje |
|---|---|
| Tabla inexistente | `La tabla "nada" no existe` |
| Columna inexistente | `La columna "zz" no existe` |
| Tipos incompatibles | `Tipos incompatibles: no se puede asignar un valor VARCHAR a la columna "id" de tipo INT` |
| Valores incorrectos en `INSERT` | cantidad de valores, `NOT NULL` sin valor, `VARCHAR` demasiado largo, fecha inválida, división por cero |
| Referencias ambiguas o inválidas | `Referencia ambigua: la columna "id" existe en más de una tabla; califíquela…` |
| Índice inexistente o duplicado | `El índice "i" no existe`, `Índice duplicado: la columna …` |
| Subconsulta escalar con más de una columna o registro | `Subconsulta escalar inválida: retorna 2 columnas…`, `La subconsulta escalar retornó 2 registros; debe retornar como máximo uno` |

## Ejemplos de verificación

Están en `testdata/sql/` y los ejecuta `go test ./internal/sql` (`examples_test.go`). Cada uno compara el resultado exacto contra un archivo guardado junto a él.

| Carpeta | Contenido | Se compara con |
|---|---|---|
| `ok/` | `enunciado.sql` (las tablas y las dos consultas del PDF del enunciado, tal cual, más las otras subconsultas, agregaciones e índice), `tienda.sql` (4 tablas, 21 consultas con `JOIN`, agregaciones, subconsultas e índices, y `UPDATE`/`DELETE`), `escuela.sql`, `tipos.sql` (límites de cada tipo y de la codificación CSV), `claves.sql`, `demo.sql` | `.golden`: resultado de cada sentencia, plan de lectura y contenido de todos los archivos de la base |
| `analyze/` | `consultas.sql`: 27 sentencias complejas sobre un esquema sin datos; solo se analizan | cero errores y 27 sentencias en el AST |
| `errors/` | Un archivo por fase: `lexico_*`, `sintaxis`, `semantico_*`, `ejecucion_*` | `.expected`: línea, columna, fase y mensaje de cada error |

Si cambia un mensaje a propósito, regenerar con `go test ./internal/sql -update` y revisar el diff a mano antes de confirmarlo. Un error léxico puede provocar un error sintáctico a continuación (por ejemplo, una cadena sin cerrar deja `VALUES (1,` incompleto): el token ilegal se descarta y el parser informa lo que falta.

Además, `query_test.go` calcula a mano los resultados de decenas de consultas (incluyendo `NULL`) y, con datos aleatorios, comprueba que 300 consultas con índice dan exactamente lo mismo que con recorrido completo y que, tras 40 escrituras aleatorias, cada índice coincide con lo que se reconstruiría desde su tabla.

## Matriz de requisitos

| Etapa | Requisito | Evidencia |
|---|---|---|
| 1 | Lexer y parser | `TestLex*`, `TestParse*` en `sql_test.go` |
| 2 | AST | `TestParseAllRequiredStatements`, pestaña AST de la interfaz |
| 3 | Semántico y manejo de errores | `TestSemanticErrors` (38 casos), `examples_test.go` (`errors/`) |
| 4 | `CREATE`, `DROP`, `INSERT` | `TestExecuteCreateInsertDropPersistsInFiles`, `TestPrimaryKeyViolationIsAtomic` |
| 5 | `SELECT`, `WHERE`, `UPDATE`, `DELETE` | `TestSelectWhereAndOrderWithNulls`, `TestUpdateAndDelete`, `TestRuntimeErrorsAreLocatedAndStopTheScript` |
| 6 | `JOIN`, agregaciones, `GROUP BY` | `TestJoinsAndAggregates`, `ok/tienda.sql` |
| 7 | Índices y subconsultas | `TestWhereUsesIndexInsteadOfScanningEveryRecord`, `TestInListUsesIndex`, `TestIndexesAreMaintainedByInsertUpdateDelete`, `TestStaleOrMissingIndexIsRebuiltInsteadOfTrusted`, `TestIndexedResultsEqualFullScanOnRandomData`, `TestSubqueries` |
| 8 | Integración | CLI `cmd/sql`, `cmd/ide/sql_test.go`, `web/sql.html` |

## Límites conocidos

- No hay `LIMIT`, `DISTINCT`, `LIKE`, `BETWEEN`, `LEFT JOIN` ni `ORDER BY` por posición: no los pide el enunciado.
- El índice no se usa con `OR`, `NOT`, `NOT IN`, `IN (subconsulta)` ni en la condición `ON` de un `JOIN`; esas consultas hacen recorrido completo y siguen dando el resultado correcto. Las columnas `PRIMARY KEY` no tienen índice automático.
- `UPDATE`, `DELETE` e `INSERT` reescriben el archivo de la tabla y sus índices (costo proporcional al tamaño de la tabla).
- No hay transacciones entre sentencias ni bloqueo entre procesos: dos procesos que escriban la misma base a la vez no están soportados (el servidor del IDE sí serializa sus ejecuciones).
- Un `JOIN` se ejecuta con bucles anidados, adecuado para bases pequeñas como las de este proyecto.
