# Compilar Compiscript en el IDE

El IDE usa la misma fachada `compiscript.Compile` que la CLI. Muestra el TAC generado y los metadatos de almacenamiento estático; no ejecuta programas ni implementa un segundo compilador.

Por lo tanto, print, residuo, uniones perezosas, asignaciones, almacenamiento undefined/null, valores sin tipo protegidos y try/catch de T9/T10 usan la misma acción Compile sin un runtime web separado. Un reporte exitoso puede contener guardas en tiempo de ejecución: el IDE no imprime la salida del programa ni captura fallos ejecutando el TAC. Los diagnósticos sintácticos/semánticos reales o de inicialización estática siguen limpiando TAC/codegen de forma atómica.

## Ruta rápida

1. Desde la raíz del repositorio, ejecute `go run ./cmd/ide` y abra `http://localhost:8080`.
2. Seleccione **Compiscript** y escriba el código fuente, use **Load .cps** o elija **Load sample**. Los archivos subidos deben tener extensión `.cps` y un tipo MIME de texto plano o no especificado (el MIME binario genérico se acepta por compatibilidad entre plataformas).
3. Elija **Compile .cps**, o presione **Run / Ctrl+Enter** mientras el editor de Compiscript esté activo.
4. Lea **TAC**, **Diagnostics**, **Environments** y **Extended symbols**. **Analyze .cps (AST)** conserva el flujo previo de solo análisis y llena la pestaña AST.

**Clear .cps** limpia el editor e invalida los resultados. Editar, cargar otro archivo o cambiar entre Compiscript y YAPar invalida las solicitudes pendientes, de modo que las respuestas tardías no puedan reemplazar salidas más recientes. La lectura de un archivo captura su editor destino original; las ediciones o cargas posteriores en ese destino reemplazan la lectura. **Save** conserva la exportación existente a un archivo local. El lexer, la gramática, la entrada, los métodos y el flujo de análisis de YAPar siguen disponibles.

El servidor escucha en `:8080` (todas las interfaces) y sirve el directorio relativo `web`; use un entorno local de confianza y láncelo desde la raíz del repositorio. Ninguna verificación de la documentación inició un listener. Para los prerrequisitos y el ejemplo público de curl local, consulte la [guía de ejecución](README.md).

## Contrato HTTP

`POST /api/compiscript/compile` acepta exactamente `{"source":"let x = 1;"}` como `application/json` (se aceptan parámetros charset opcionales). El endpoint invoca `Compile` una vez y devuelve los campos completos y sin cambios de su `CompilationReport`, más `tacText`, renderizado en el servidor por `report.TAC.String()`. El navegador nunca da formato al TAC a partir del JSON. Analyze permanece en `POST /api/compiscript/analyze` con la forma de reporte anterior.

| Resultado | HTTP / cuerpo |
|---|---|
| Código fuente válido con soporte, incluido el código fuente vacío | 200; reporte del núcleo y texto TAC del servidor |
| Diagnóstico léxico, sintáctico, semántico o de codegen | 200; diagnósticos tipados ubicados, `tac: null`, `tacText` vacío, `codegen` ausente |
| Source faltante/null, JSON mal formado, llaves desconocidas/con variante de mayúsculas/duplicadas, JSON sobrante al final o UTF-8 inválido | 400; forma existente `{"error":"…"}` |
| Cuerpo de más de 1 MiB | 413 |
| Content type faltante/incorrecto | 415 |
| Método distinto de POST | 405, `Allow: POST` |
| Preflight de CORS | OPTIONS 204 |

Los fallos de transporte limpian el TAC previo, los símbolos y los demás paneles de Compiscript. El código fuente, los nombres de archivo, los errores y los metadatos usan nodos de texto en lugar de HTML. Los controles se restauran tras completarse o invalidarse la solicitud; Clear, la carga del ejemplo y la edición siguen disponibles durante una solicitud.

## Lectura de símbolos extendidos

El reporte opcional congelado `codegen` contiene `unit`, `symbols`, `spaces` y `labels`. La unidad es **abstract-cell**, no bytes ni direcciones físicas. Las entradas de símbolos conservan identidades semánticas, IDs de ámbito, metadatos de tipo/origen, ubicaciones (`space`, `offset`, `size`, `role`) y etiquetas. El panel muestra como texto las ranuras completas de los espacios de almacenamiento y los rangos de etiquetas, preservando el reporte determinante del núcleo en lugar de inferir disposiciones en JavaScript. Los globales, los temporales de main, los marcos de invocación, los entornos retenidos y las instancias tienen propiedad distinta. Las etiquetas no son celdas; las listas de ubicaciones vacías no implican una asignación inventada.

El tipo `error` de un símbolo semántico original puede marcar información de anotación ausente; no es en sí mismo un diagnóstico de compilación. Los contratos privados de almacenamiento de valores desconocidos y de excepciones se describen en la [referencia de TAC](tac-design.md#valores-protegidos-y-excepciones-síncronas-t10). En particular, un campo sin anotación sigue siendo desconocido aun cuando se inicialice con null, mientras que los campos anotados explícitamente conservan sus guardas de tipo/nominales. El navegador no infiere un tipo nuevo a partir de su valor inicial.

## Verificación y límites

Verificaciones focalizadas: `go test ./cmd/ide -count=1`, `node --test web/*.test.cjs` y `go vet ./cmd/ide`. El comando de Node ejecuta tanto las pruebas de la implementación como las pruebas VM independientes retenidas. Las pruebas HTTP comparan el reporte completo del núcleo y el texto TAC del servidor, y verifican la paridad estricta de transporte con Analyze. Las pruebas con la VM integrada de Node ejercitan los cuerpos de solicitud, el renderizado, los reinicios, el texto hostil, los fallos y el orden asíncrono sin un nuevo framework de navegador.

Estas pruebas establecen el cableado HTTP/DOM, no la apariencia en un navegador real. CodeMirror y Viz.js mantienen sus requisitos existentes de CDN. La evaluación visual/manual en el navegador sigue siendo aparte; la prueba de ejecución del compilador corresponde a las pruebas existentes de la CLI/compilador. No se ofrece ningún servicio de ejecución remota.
