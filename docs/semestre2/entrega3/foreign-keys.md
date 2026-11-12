# Claves foráneas restrictivas

Una columna puede referenciar la `PRIMARY KEY` de una tabla existente. Hay dos formas equivalentes:

```sql
CREATE TABLE padre (id INT PRIMARY KEY);
CREATE TABLE hijo (id INT PRIMARY KEY, padre_id INT REFERENCES padre(id));
-- Alternativa: FOREIGN KEY (padre_id) REFERENCES padre(id)
INSERT INTO padre VALUES (1);
INSERT INTO hijo VALUES (10, 1);
```

`DROP TABLE padre` falla mientras exista la tabla hija, aunque esté vacía. `DELETE FROM padre WHERE id = 1` falla mientras alguna fila hija lo referencie. Elimine primero las filas hijas y luego la fila padre; para eliminar ambas tablas, elimine primero la hija. Una clave primaria sin referencias no impide borrar filas ni tablas. No existe borrado automático en cascada.

`INSERT` y `UPDATE` comprueban las referencias; cambiar una clave primaria referenciada también se rechaza. Se comprueba toda la sentencia antes de escribir: una fila inválida rechaza el lote completo. `NULL` no necesita coincidencia, salvo que la columna sea `NOT NULL`. Solo se permiten claves de una columna que apunten a una `PRIMARY KEY` del mismo tipo; se permiten varias columnas referenciantes independientes. No se admiten claves compuestas, `CONSTRAINT nombre`, ni `ON DELETE`/`ON UPDATE` (incluidos `CASCADE` y `RESTRICT` explícitos).

`ALTER TABLE ... RENAME TO` actualiza las referencias; no se puede eliminar una columna referenciada. Eliminar la columna hija elimina su propia restricción. Una tabla puede referenciarse a sí misma: su propia restricción no impide `DROP TABLE`, y `DELETE` se evalúa contra las filas que quedan después de la sentencia (por ejemplo, borrar todas las filas es válido).

Las referencias se guardan como `REFERENCES tabla(columna)` en la línea `COLUMN` de `metadata.txt`. Sin claves foráneas el formato anterior no cambia. El nuevo lector abre metadatos anteriores; versiones anteriores rechazan la nueva marca en vez de ignorarla. No hay migración automática. Los errores semánticos impiden ejecutar todo el script; un error de ejecución detiene la sentencia sin deshacer sentencias anteriores.
