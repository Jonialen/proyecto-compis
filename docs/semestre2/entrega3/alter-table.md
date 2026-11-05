# ALTER TABLE

Permite modificar una tabla existente sin perder sus registros restantes. Los nombres no distinguen mayúsculas de minúsculas y los cambios son visibles para las sentencias siguientes del mismo script.

```sql
ALTER TABLE estudiantes ADD [COLUMN] correo VARCHAR(80);
ALTER TABLE estudiantes DROP [COLUMN] correo;
ALTER TABLE estudiantes RENAME TO alumnos;
```

Los corchetes indican una palabra opcional; no se escriben en SQL. `ADD` reutiliza los tipos y las restricciones de `CREATE TABLE`: `INT`, `FLOAT`, `VARCHAR(n)`, `BOOLEAN`, `DATE`, `PRIMARY KEY` y `NOT NULL`. No hay `DEFAULT`.

- Una columna nueva sin restricciones recibe `NULL` en cada registro existente.
- Agregar `NOT NULL` o `PRIMARY KEY` requiere una tabla vacía; una clave primaria implica `NOT NULL` y solo puede haber una por tabla.
- No se permiten nombres de columna repetidos, longitudes `VARCHAR` no positivas ni eliminar la última columna.
- `DROP` elimina únicamente la columna indicada y sus índices. Los índices de las columnas restantes se reconstruyen con las nuevas posiciones del CSV.
- `RENAME TO` conserva el orden de la tabla en el catálogo, actualiza sus índices y elimina el CSV del nombre anterior. No sobrescribe otra tabla ni un archivo de destino existente. Un cambio solo de mayúsculas conserva la ruta normalizada del CSV.
- `ALTER`, `ADD`, `COLUMN`, `RENAME` y `TO` son palabras contextuales: siguen siendo identificadores válidos donde antes se admitían.

## Ejemplo aislado

```sh
go run ./cmd/sql -db /tmp/mibd-alter <<'SQL'
CREATE TABLE estudiantes (id INT PRIMARY KEY, nombre VARCHAR(50));
INSERT INTO estudiantes VALUES (1, 'Ana');
ALTER TABLE estudiantes ADD correo VARCHAR(80);
SELECT * FROM estudiantes;
ALTER TABLE estudiantes DROP COLUMN correo;
ALTER TABLE estudiantes RENAME TO alumnos;
SQL
```

Cada cambio exitoso devuelve un resultado `ALTER TABLE` con `rowsAffected = 0`: cambiar el esquema no cuenta como insertar, actualizar o eliminar registros. Los errores semánticos impiden ejecutar el script completo; un error de ejecución detiene las sentencias posteriores, pero conserva las anteriores.

## Persistencia y límites

Antes de modificar archivos se leen y validan los datos y las restricciones existentes. Se preparan el CSV, los índices y el catálogo en un directorio temporal dentro de la base. Se comprueban los destinos y se conservan los originales mediante renombrado hasta completar la instalación; un fallo ordinario de instalación restaura los archivos originales, incluidos sus bytes y sellos de índices. No se admiten directorios ni enlaces simbólicos como destinos de estos archivos.

`-analyze` solo comprueba el esquema en una copia del catálogo y nunca escribe. No valida restricciones dependientes de los registros: por ejemplo, agregar `NOT NULL` a una tabla con datos se rechaza durante la ejecución.

No se agregan garantías ante caída del proceso, fallos durante la propia recuperación, acceso concurrente entre procesos ni transacciones entre sentencias. Las pruebas usan bases temporales y no requieren acceder a `sqldata`.
