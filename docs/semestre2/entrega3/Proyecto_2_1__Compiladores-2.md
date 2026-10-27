# Proyecto: Intérprete SQL con Almacenamiento en Archivos

**Construcción de Compiladores**  
Septiembre, 2026

## Objetivo

Desarrollar un sistema capaz de interpretar y ejecutar sentencias SQL utilizando su propio mecanimso de almacenamiento. No se utilizará ningún DBMS existente para procesar o ejecutar SQL.

## Funcionamiento Esperado

1. SQL
2. Análisis léxico
3. Análisis sintáctico
4. AST
5. Análisis semántico
6. Interpretación
7. Archivos de texto
8. Resultado

Podrán utilizar herramientas como ANTLR para implementar el lexer y parser.

## Almacenamiento

Las bases de datos deberán representarse utilizando únicamente archivos y directorios. El equipo deberá definir cómo almacenar esquemas, tipos de datos, registros y restricciones necesarias.

```text
database/
|-- metadata.txt
|-- estudiantes.csv
|-- cursos.csv
|-- inscripciones.csv
```

**No se permite utilizar SQLite, MySQL, PostgreSQL u otro DBMS como mecanismo de almacenamiento o ejecución.**

## Tipos de Datos

El sistema deberá soportar, como mínimo, los siguientes tipos de datos:

- `INT`: números enteros.
- `FLOAT`: números de punto flotante.
- `VARCHAR(n)`: cadenas de texto de longitud máxima n.
- `BOOLEAN`: valores `TRUE` o `FALSE`.
- `DATE`: fechas en formato `YYYY-MM-DD`.

Los tipos de cada columna deberán almacenarse como parte del esquema de la tabla y ser utilizados durante el análisis semántico y la ejecución.

## SQL Requerido

- `CREATE TABLE`
- `DROP TABLE`
- `CREATE INDEX`
- `DROP INDEX`
- `INSERT`
- `UPDATE`
- `DELETE`
- `SELECT`
- `FROM`
- `WHERE`
- `AND`
- `OR`
- `ORDER BY`
- `INNER JOIN`
- `COUNT`
- `SUM`
- `AVG`
- `MIN`
- `MAX`
- `GROUP BY`
- `HAVING`
- `IN`
- `EXISTS`

## Índices

El sistema deberá permitir crear índices sobre columnas de una tabla mediante `CREATE INDEX`, y eliminarlos mediante `DROP INDEX`.

- Los índices deberán almacenarse en archivos, siguiendo el formato definido por el equipo.
- Al evaluar condiciones de `WHERE` sobre una columna indexada, el intérprete deberá utilizar el índice en lugar de recorrer todos los registros.
- Los índices deberán mantenerse actualizados al ejecutar `INSERT`, `UPDATE` y `DELETE`.

## Subconsultas

El sistema deberá permitir sentencias `SELECT` anidadas dentro de la cláusula `WHERE`:

- Subconsultas con `IN` y `NOT IN`.
- Subconsultas con `EXISTS` y `NOT EXISTS`.
- Subconsultas escalares utilizadas en comparaciones (`=`, `<`, `>`, etc.).

```sql
SELECT nombre
FROM estudiantes
WHERE id IN (SELECT estudiante_id
             FROM inscripciones
             WHERE curso_id = 3);
```

## Análisis Semántico

Antes de ejecutar una sentencia deberán verificarse errores como:

- Tabla inexistente.
- Columna inexistente.
- Tipos incompatibles.
- Valores incorrectos en `INSERT`.
- Referencias ambiguas o inválidas.
- Índice inexistente o duplicado.
- Subconsulta escalar que retorna más de una columna o más de un registro.

Los errores deberán indicar claramente el problema y su ubicación.

## Interfaz Gráfica

La aplicación deberá permitir:

- Escribir sentencias SQL.
- Visualizar tokens.
- Visualizar el árbol sintáctico.
- Visualizar el AST.
- Visualizar errores léxicos, sintácticos y semánticos.
- Ejecutar la sentencia.
- Visualizar los resultados.

## Ejemplo

```sql
SELECT nombre
FROM estudiantes
WHERE edad >= 18;
```

El sistema deberá interpretar la sentencia, consultar los archivos correspondientes, evaluar la condición y mostrar únicamente los nombres que cumplan con ella.

## Entregas

El proyecto se desarrollará incrementalmente:

1. Lexer y parser.
2. AST.
3. Análisis semántico y manejo de errores.
4. `CREATE`, `DROP` e `INSERT`.
5. `SELECT`, `WHERE`, `UPDATE` y `DELETE`.
6. `JOIN`, agregaciones y `GROUP BY`.
7. Índices y subconsultas.
8. Integración y entrega final.
