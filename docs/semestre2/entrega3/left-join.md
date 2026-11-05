# LEFT JOIN

`LEFT JOIN` y `LEFT OUTER JOIN` son equivalentes. Conservan cada fila del lado izquierdo: si ninguna fila derecha cumple `ON`, las columnas derechas contienen `NULL`. Si hay varias coincidencias, se devuelve una fila por coincidencia. Una tabla derecha vacía también produce una fila extendida por cada fila izquierda.

```sql
SELECT e.nombre, i.nota
FROM estudiantes e
LEFT OUTER JOIN inscripciones i ON e.id = i.estudiante_id
ORDER BY e.nombre;
```

## ON y WHERE

`ON` decide las coincidencias antes de extender con `NULL`. `WHERE` filtra el resultado después de la extensión y conserva solo condiciones verdaderas.

```sql
-- Conserva estudiantes aunque no tengan una inscripción con nota >= 80.
SELECT e.nombre, i.nota
FROM estudiantes e
LEFT JOIN inscripciones i ON e.id = i.estudiante_id AND i.nota >= 80;

-- Excluye estudiantes sin una inscripción con nota >= 80.
SELECT e.nombre, i.nota
FROM estudiantes e
LEFT JOIN inscripciones i ON e.id = i.estudiante_id
WHERE i.nota >= 80;
```

Las comparaciones con `NULL` no son verdaderas; se usa `IS NULL` para detectar ausencia. Los joins encadenados se evalúan de izquierda a derecha y pueden combinarse con `JOIN` o `INNER JOIN`. `COUNT(*)` cuenta también las filas extendidas; `COUNT(columna)` y las otras agregaciones ignoran los valores `NULL`.

## Identificadores y ejecución

`LEFT` y `OUTER` son palabras contextuales, no reservadas globalmente: pueden ser nombres de tablas, columnas o alias. Para un alias `left` inmediatamente antes de `JOIN`, se debe escribir `AS left` para distinguirlo del inicio de `LEFT JOIN`.

El ejecutor reutiliza los bucles anidados y conserva el orden de `JOIN`/`INNER JOIN`. No usa índices de `WHERE` para reducir el lado derecho de un `LEFT JOIN`: quitar coincidencias antes de evaluar `ON` podría fabricar filas extendidas. Los índices del lado preservado siguen disponibles. `-noindex` permite comparar los resultados con un recorrido completo. No se modifican los archivos de datos ni el esquema al consultar.

Las regresiones públicas están en `internal/sql/left_join_test.go` y `cmd/sql/left_join_test.go`.
