# Guion de demostración: Compiscript, MIPS y SQL

**Jonathan Diaz y Luis Padilla · duración sugerida: 14 minutos · repositorio:** https://github.com/Jonialen/proyecto-compis

**Video (enlace provisional):** [Carpeta de entrega del video](https://uvggt-my.sharepoint.com/:f:/r/personal/dia23837_uvg_edu_gt/_layouts/15/onedrive.aspx?id=%2Fpersonal%2Fdia23837%5Fuvg%5Fedu%5Fgt%2FDocuments%2FVideo%20Compis&share=cgqR%5FSYPX8AxQ7OLibRUWzxiEgUCn10ckXxe%5F%2DjMxMiz8WWTaQ).

Presenten el IDE en vivo y ejecuten el ASM descargado en MARS gráfico. SQL es un intérprete separado: **SQL no se traduce a MIPS**. Este guion no afirma que el video esté grabado o publicado.

## Preparación fuera de cámara

Necesitan Go y las dependencias disponibles, Java, `tools/mars/Mars4_5.jar` local y un navegador capaz de cargar los recursos CDN que usa el editor. Ensayen la interfaz una vez antes de grabar; no se garantiza funcionamiento sin conexión. El servidor es de desarrollo local: no lo expongan públicamente.

Desde la raíz del repositorio, preparen una instancia nueva y aislada:

```bash
ROOT=$(pwd)
mkdir -p "$ROOT/.atl"
D=$(mktemp -d "$ROOT/.atl/video-ide.XXXXXX")
go build -o "$D/ide" ./cmd/ide
cp -R "$ROOT/web" "$D/web"
(cd "$D" && "$D/ide")
```

Dejen esa terminal abierta fuera de cámara. El puerto es **8080 fijo**; si está ocupado, detengan manualmente su IDE anterior antes de comenzar. Confirmen que el proceso sigue activo sin errores: si termina o falla al abrir el puerto, no continúen con otra instancia ya abierta, porque podría usar la base real. El binario se construye desde el repositorio, pero sirve la copia de `web` desde `D`; SQL escribe solo en `D/sqldata/default`, no en la base del repositorio. Para otro ensayo, creen otro directorio nuevo: no borren ni reinicien bases existentes.

Abran `http://localhost:8080/` y tengan disponibles los archivos de `testdata/video/`. En otra terminal fuera de cámara, desde la raíz, abran MARS gráfico:

```bash
java -jar tools/mars/Mars4_5.jar
```

La pestaña inicial del editor es `.yalp`: seleccionen **Compiscript** o carguen el `.cps`. Abran SQL en una **nueva pestaña** con `http://localhost:8080/sql.html`, equivalente al enlace **SQL** del encabezado, para conservar el editor principal. SQL no tiene carga de archivos: abran los fixtures en un editor de texto y copien su contenido al editor SQL. Cada paso negativo va en una invocación separada: un error de análisis impide ejecutar el lote.

## Guion para dos personas (14:00)

### 0:00–1:00 · Presentación

**Jonathan muestra** el IDE y dice: «Somos Jonathan Diaz y Luis Padilla. Mostraremos Compiscript desde su fuente hasta MIPS y su ejecución; después probaremos SQL por separado. El repositorio es github.com/Jonialen/proyecto-compis».

**Luis dice:** «La base de esta demostración es temporal y aislada. El IDE genera ensamblador; MARS lo ejecuta».

### 1:00–3:00 · Fuente y objetos

**Jonathan hace clic** en **Load .cps** y selecciona `testdata/video/mips.cps`; muestra el código en **Compiscript**. «Base declara `n` con valor 2; Child hereda el campo y sobrescribe `read`, sumando 8. Creamos dos Child; `alias` modifica el primero a 4 y el segundo conserva 2. También calculamos fib de 8».

**Luis señala** `print(box)` y los accesos a campos: «Imprimimos el objeto además de resultados. `<object s1.Child>` es una etiqueta opaca de clase, no una dirección única ni un volcado de campos; los valores de los campos muestran el alias y la independencia».

### 3:00–5:00 · Análisis y código intermedio

**Luis hace clic** en **Analyze .cps (AST)** y muestra **AST** y **Diagnostics**. «El análisis léxico identifica tokens internamente, el sintáctico construye el árbol y los diagnósticos muestran problemas; este ejemplo debe analizarse sin errores». No usen la pestaña **Tokens** como evidencia de Compiscript: esta acción no llena ese panel.

**Jonathan hace clic** en **Compile .cps (TAC)** y muestra **TAC**, **Environments** y **Extended symbols**. «El TAC representa operaciones intermedias: busquemos la creación del objeto, enlace del método, acceso al campo y llamadas, incluida la recursiva. Los entornos y símbolos muestran información del análisis». El botón **Run** en Compiscript también compila a TAC: no ejecuta el programa.

### 5:00–7:00 · MIPS y ejecución gráfica

**Jonathan hace clic** en **Compile MIPS**, muestra la pestaña **MIPS** y relaciona el acceso al campo y la llamada al método en TAC con el ensamblador generado. Luego hace clic en **Download .asm**. Al cargar `mips.cps`, el nombre sugerido es `mips.asm`; usen el archivo realmente descargado, respetando cualquier cambio de nombre del navegador.

**Luis cambia a MARS**, usa **File → Open** para abrir ese archivo descargado y las acciones de menú para ensamblar y después ejecutar (**Run/Go**). Muestra **Run I/O**: se espera `objeto:`, `<object s1.Child>`, y los valores inicial `2`, alias `4`, independiente `2`, método `12` y fib(8) `21` con sus etiquetas. «La ejecución ocurre en MARS, no en un panel de salida del IDE web». Si no coincide, detengan el ensayo; no presenten la ejecución como exitosa.

### 7:00–9:00 · Preparar SQL

**Luis cambia** a la pestaña SQL aislada, copia `testdata/video/sql/setup.sql` al editor y señala `FOREIGN KEY (cliente_id) REFERENCES clientes(id)`. Pulsa **▶ Ejecutar** y muestra **Resultados** y **Esquema**. «Creamos clientes con PK `id` y pedidos con FK a clientes. La PK exige unicidad y no acepta NULL; la FK impide referencias a padres inexistentes». La vista **Esquema** muestra tablas y columnas; no hace falta atribuirle una insignia FK.

**Jonathan reemplaza** el editor con `testdata/video/sql/select.sql`, pulsa **▶ Ejecutar** y muestra en **Resultados** clientes `(1, Ana)`, `(2, Luis)` y pedido `(10, 1)`. **Analizar** solo valida; **▶ Ejecutar** aplica las sentencias y persiste cambios.

### 9:00–11:00 · Rechazos y estado conservado

**Jonathan pega y ejecuta por separado**, reemplazando el contenido cada vez:

1. `testdata/video/sql/duplicate.sql`: muestra el rechazo de PK duplicada en **Errores**.
2. `testdata/video/sql/missing.sql`: muestra el rechazo de la FK para el cliente inexistente 9.
3. `testdata/video/sql/delete-blocked.sql`: muestra el rechazo del DELETE que incluye al cliente 1 referenciado.
4. `testdata/video/sql/drop-blocked.sql`: muestra el rechazo de DROP de la tabla referenciada.

**Luis vuelve a pegar y ejecutar** `testdata/video/sql/select.sql` y muestra ambos clientes, incluido el 2 no referenciado, y el pedido `(10, 1)`. «Se rechazó el DELETE completo, no solo la fila referenciada. La FK es restrictiva, no CASCADE».

### 11:00–13:00 · Borrado permitido y dependencia del esquema

**Luis pega y ejecuta** `testdata/video/sql/delete-free.sql`, luego `testdata/video/sql/after-free.sql`: **Resultados** muestra solo cliente 1 y pedido `(10, 1)`. «Borrar el cliente 2 sin referencias sí funciona; la PK sola no bloquea el borrado».

**Jonathan pega y ejecuta** `testdata/video/sql/clear-child.sql` y después `testdata/video/sql/after-clear.sql`; muestra pedidos sin filas. Ejecuta otra vez `testdata/video/sql/drop-blocked.sql` y muestra el rechazo en **Errores**: «Aunque la tabla hija esté vacía, su esquema todavía referencia a clientes». Finalmente pega y ejecuta `testdata/video/sql/drop-all.sql`: elimina primero pedidos y después clientes; muestra **Resultados** y **Esquema** vacío.

### 13:00–14:00 · Cierre

**Jonathan dice:** «Vimos fuente, análisis, TAC, MIPS y ejecución en MARS, con objetos, alias, herencia y recursión». **Luis dice:** «En SQL mostramos restricciones de inserción y borrado, sobre una base aislada. Gracias».

## Lista de ensayo

- [ ] Ensayar con cronómetro dentro del rango de 10 a 15 minutos y comprobar que el IDE y MARS se ven bien.
- [ ] Preparar una instancia nueva aislada y verificar el puerto y los recursos del navegador antes de grabar.
- [ ] Abrir en MARS el ASM descargado de esta compilación y comprobar la salida, sin confundir compilación con ejecución.
- [ ] Mostrar los cuatro rechazos por separado, el SELECT posterior, el borrado permitido y el DROP bloqueado con hija vacía.
- [ ] Practicar los turnos; no presentar fallas como éxitos ni afirmar grabación o entrega anticipadamente.

Como regresión técnica opcional **fuera de cámara**, `bash scripts/demo-video.sh` ejecuta el ensayo automatizado con artefactos y MARS en modo headless; no sustituye la demostración gráfica.

Detalles técnicos: [TAC](../entrega2/tac-design.md), [MIPS](../mips/README.md) y [claves foráneas SQL](../entrega3/foreign-keys.md).
