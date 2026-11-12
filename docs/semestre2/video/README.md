# Guion de demostración: Compiscript, MIPS y SQL

**Jonathan Diaz y Luis Padilla · duración sugerida: 14 minutos · repositorio:** https://github.com/Jonialen/proyecto-compis

**Video (enlace provisional):** [Carpeta de entrega del video](https://uvggt-my.sharepoint.com/:f:/r/personal/dia23837_uvg_edu_gt/_layouts/15/onedrive.aspx?id=%2Fpersonal%2Fdia23837%5Fuvg%5Fedu%5Fgt%2FDocuments%2FVideo%20Compis&share=cgqR%5FSYPX8AxQ7OLibRUWzxiEgUCn10ckXxe%5F%2DjMxMiz8WWTaQ).

Este es un guion para ensayar y grabar; no afirma que el video haya sido grabado ni publicado. La parte de SQL es un intérprete separado: **SQL no se traduce a MIPS**. La ejecución usa una base aislada, nunca `sqldata/default`.

## Antes de comenzar

Desde la raíz, con Go compatible con `go.mod`, Java y el MARS local ya disponible, ejecuten:

```bash
bash scripts/demo-video.sh
```

El script imprime `Artefactos: .../.atl/demo-video.XXXXXX` al inicio y conserva ese directorio incluso si falla. Compila las tres herramientas allí, sin descargar dependencias; genera `program.tac` y `program.asm`, ejecuta **ese mismo** ASM con `scripts/mars.sh` y compara salida, error y exit. Después ejecuta cada paso SQL contra `db` dentro de ese directorio. Si falla, detengan el ensayo y revisen los archivos `*.stdout`, `*.stderr` y `*.exit` del paso; no continúen como si hubiera pasado. El wrapper MARS requiere Java y el JAR local fijado; no es necesario instalar nada durante la presentación.

Para inspeccionar con comandos manuales desde la raíz, sustituyan en la primera línea la ruta impresa por el ensayo. Esos comandos solo leen los artefactos ya generados; no repiten operaciones sobre la base:

```bash
D=.atl/demo-video.RUTA_IMPRESA
less testdata/video/mips.cps
less "$D/program.tac"
less "$D/program.asm"
less "$D/mars.stdout"
less testdata/video/sql/setup.sql
less "$D/delete-blocked.stderr"
cmp "$D/delete-blocked-before.json" "$D/delete-blocked-after.json"
less "$D/after-free.stdout"
less "$D/drop-empty-child-before.json"
```

Para repetir manualmente MARS sobre el archivo generado (sin volver a compilar):

```bash
bash scripts/mars.sh "$D/program.asm"
```

Para repetir SQL manualmente, usen **otra** base nueva dentro del mismo directorio de ensayo y estos comandos en orden. Cada archivo es una invocación independiente; los rechazos devuelven exit 1 y no son motivo para saltarse el siguiente comando. El script automático ejecuta esos mismos archivos con `-db` y además compara streams y estado público.

```bash
mkdir "$D/db-manual"
"$D/sql" -db "$D/db-manual" < testdata/video/sql/setup.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/duplicate.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/missing.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/delete-blocked.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/drop-blocked.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/delete-free.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/after-free.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/clear-child.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/after-clear.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/drop-blocked.sql
"$D/sql" -db "$D/db-manual" < testdata/video/sql/drop-all.sql
```

`db-manual` debe ser nuevo: si ya existe, hagan un ensayo nuevo y usen su ruta impresa. No borren bases previas. El mensaje final del script confirma todas las comparaciones; los archivos esperados están en `testdata/video/sql/` y `testdata/video/mips.stdout`.

## Guion para dos personas (14:00)

### 0:00–1:00 · Presentación

**Jonathan muestra** el repositorio y dice: «Somos Jonathan Diaz y Luis Padilla. En esta demostración presentamos la traducción y ejecución de Compiscript a MIPS y, por separado, las restricciones de nuestro intérprete SQL. El código está en github.com/Jonialen/proyecto-compis».

**Luis dice:** «Veremos el programa fuente, su código intermedio, el ensamblador generado y su ejecución. Después probaremos claves primarias y foráneas sobre una base temporal».

### 1:00–3:00 · Fuente y objetos

**Jonathan muestra** `testdata/video/mips.cps` con el primer comando `less` de arriba y dice: «Base declara el campo entero `n` con valor 2. Child hereda ese campo y sobrescribe `read`: suma 8. Creamos dos objetos Child. `alias` apunta al primero y cambia su campo a 4; el segundo conserva 2. También calculamos fib de 8 de forma recursiva».

**Luis señala** `print(box)` y los `print` de campos: «No mostramos solo resultados de funciones: imprimimos el objeto y consultamos sus campos. La representación esperada es `<object s1.Child>`; es una etiqueta opaca de clase, no un volcado de campos ni prueba de direcciones únicas. La evidencia de alias y de objetos separados son los valores de los campos».

### 3:00–5:00 · Traducción concreta

**Luis muestra** `program.tac` y dice: «El compilador recibe el mismo archivo `.cps`: el análisis léxico produce tokens; el sintáctico construye el AST y el semántico comprueba nombres y tipos. A partir de ahí se genera TAC, una representación de tres direcciones. Aquí buscamos las operaciones del campo, la llamada al método y las llamadas recursivas de fib».

**Jonathan muestra** `program.asm` y dice: «La herramienta MIPS vuelve a compilar esa fuente y baja el programa a ensamblador MIPS. Este archivo no es un ejemplo escrito a mano: es precisamente el que el ensayo entrega a MARS. Podemos comparar el campo y las llamadas de TAC con las instrucciones y rutinas generadas, sin confundir el código intermedio con la ejecución».

### 5:00–7:00 · Ejecutar el ensamblador

**Jonathan escribe** `bash scripts/mars.sh "$D/program.asm"` y espera a que termine.

**Luis dice:** «MARS ejecuta el ASM generado. La salida debe mostrar el objeto, después 2, 4 y 2 para inicial, alias e independiente; el método sobrescrito devuelve 12 y fib de 8 devuelve 21. El ensayo compara toda la salida con `testdata/video/mips.stdout`, exige el stderr exacto de `testdata/video/mips.stderr` (un salto de línea de MARS, sin diagnósticos) y exit cero».

**Pausa:** si la salida o el simulador no coincide, detenerse; conservar la ruta de artefactos para diagnóstico.

### 7:00–9:00 · SQL y preparación

**Luis muestra** `testdata/video/sql/setup.sql` y dice: «Esta parte no usa el backend MIPS. El intérprete SQL analiza y ejecuta sentencias contra su catálogo y archivos CSV en la base temporal. Creamos clientes con PK `id` y pedidos con una FK `cliente_id` que referencia esa clave. Insertamos clientes 1 y 2 y un pedido que referencia al 1».

**Jonathan muestra** `setup.stdout`: «La preparación crea ambas tablas e inserta dos clientes y un pedido. La PK exige unicidad y no admite NULL; una PK por sí sola no prohíbe borrar una fila. La FK impide dejar referencias sin padre».

### 9:00–11:00 · Rechazos y preservación

**Jonathan muestra** `duplicate.stderr`, `missing.stderr` y `delete-blocked.stderr` en los artefactos y dice: «El duplicado de la PK y el pedido con cliente 9 inexistente se rechazan. El DELETE selecciona los clientes 1 y 2: como el 1 tiene un pedido, se rechaza toda la sentencia y el cliente 2 también permanece».

**Luis escribe** `cmp "$D/delete-blocked-before.json" "$D/delete-blocked-after.json"` y dice: «Sin salida y con exit cero significa que los reportes públicos JSON coinciden byte a byte. Se consulta cada tabla en ambas capturas, con esquema, metadatos de FK, índices, conteos y filas. El ensayo hace esta comparación para cada rechazo, abriendo el CLI de nuevo cada vez. Los errores son invocaciones separadas porque el análisis semántico precede a la ejecución de todo el lote».

### 11:00–13:00 · Borrado permitido y dependencia del esquema

**Luis muestra** `after-free.stdout` y dice: «Borrar el cliente 2 sin pedidos sí funciona: queda el cliente 1 y su pedido. Esto demuestra que la PK sola no bloquea el borrado. Luego borramos las filas de pedidos y comprobamos que está vacía».

**Jonathan muestra** `drop-empty-child-before.json` y `drop-blocked.stderr`: «Aunque pedidos esté vacía, su esquema todavía referencia clientes, así que DROP de clientes se rechaza. Al eliminar primero pedidos y después clientes, ambas operaciones son permitidas. Es comportamiento restrictivo por defecto, no una cascada ni una nueva cláusula SQL».

### 13:00–14:00 · Cierre

**Jonathan dice:** «En resumen, vimos el recorrido de Compiscript hasta MARS, con objetos, alias, herencia y recursión». **Luis dice:** «Y comprobamos en SQL tanto la integridad de inserciones como el rechazo de borrados referenciados, sin modificar la base real. Gracias».

## Lista de ensayo

- [ ] Ensayar con cronómetro: 14 minutos, dentro del rango de 10 a 15.
- [ ] Ejecutar el ensayo completo; confirmar exit cero y conservar la ruta impresa.
- [ ] Revisar fuente, TAC, ASM, stdout MARS y los cinco pares de capturas JSON.
- [ ] Confirmar rechazos: PK duplicada, padre FK inexistente, DELETE completo y DROP antes y después de vaciar pedidos.
- [ ] Confirmar cliente 2 eliminado por separado, pedidos vacío y DROP hijo antes de padre.
- [ ] Practicar pausas, turnos y comandos; no presentar una falla como éxito.

Detalles técnicos: [TAC](../entrega2/tac-design.md), [MIPS](../mips/README.md) y [claves foráneas SQL](../entrega3/foreign-keys.md).
