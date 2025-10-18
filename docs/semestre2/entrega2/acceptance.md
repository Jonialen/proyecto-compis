# Aceptación del compilador con casos mixtos

La batería final combina closures retenidos, referencias a objetos, listas anidadas
tipadas, iteración, transferencia en switch y recursión a través del compilador existente.
También verifica entradas mal formadas mediante Parse, Analyze, Compile, las CLI construidas
y el mux HTTP. Esta es evidencia ejecutable del compilador, no una certificación de un
runtime físico ni del IDE visual. T9/T10 extienden la batería a valores/efectos, valores
desconocidos protegidos y manejadores. El [informe técnico y la matriz del PDF](report.md)
registran la resolución de la brecha inicial de construcción, la procedencia de las pruebas
y los controles humanos pendientes.

## Ruta rápida

Desde la raíz del repositorio, con las dependencias de Go requeridas ya en caché:

```sh
go run ./cmd/compiscript-tac testdata/compiscript/tac/acceptance.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/acceptance.cps
go run ./cmd/compiscript testdata/compiscript/tac/acceptance.cps
```

Los tres comandos terminan con éxito. El primero emite texto TAC; el segundo emite un
reporte con `tac` no nulo, `diagnostics` vacío y `codegen.unit: "abstract-cell"`;
el tercero emite el reporte de análisis heredado, sin metadatos de compilación.
El compilador emite código; **no** ejecuta el programa ni imprime sus valores.

El oráculo de marcos, exclusivo de las pruebas, ejecuta tanto el TAC del núcleo como el
JSON emitido por la CLI:

| Global | Valor esperado |
|---|---:|
| `first` | 30 |
| `second` | 43 |
| `independent` | 7 |
| `continued` | 57 |
| `shared` | 37 |
| `total` | 10 |
| `result` | 147 |

`saved` conserva su invocación original y la canasta compartida después de reemplazar `next`.
La lista irregular incluye una fila vacía; el `continue` del switch omite el valor 2 dentro
del foreach interno. Las llamadas recursivas preservan los temporales del llamador. Los
ciclos while, do, for y foreach válidos también tienen casos ejecutables de preservación
por separado.

## Pruebas reproducibles

```sh
go test ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide -run 'Test(MalformedForMissingBody|MissingLoopBodies|AcceptanceAtomicRecovery|AcceptanceMixedExecution|AcceptanceRecoveryCLI|AcceptanceValidLoops|MixedAcceptanceAPI)$' -count=1
go test ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide -count=1
go test ./internal/compiscript -run '^TestValuesEffects' -count=1
go test ./internal/compiscript -run '^TestDynamicExceptions' -count=1
go test ./internal/compiscript ./cmd/ide -run '^Test(ValuesEffects|DynamicExceptions|IndependentT9|IndependentT10|MixedAcceptanceAPI)' -count=1
go vet ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide
node --test web/*.test.cjs
go test ./... -count=1
go build ./...
go vet -a ./...
```

Resultado esperado: todos los comandos terminan con código cero. No use `-short` para la
prueba con la CLI construida: omite deliberadamente las compilaciones externas. Ejecute las
verificaciones completas en una copia aislada del código fuente, porque las pruebas
existentes de ANTLR ejercitan la generación. La prueba sin conexión usa
`GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local`, un `GOMODCACHE` existente en caché,
`HOME`, `TMPDIR` y `GOCACHE` externos, y el jar de ANTLR en caché.
Ninguna instalación ni descarga de red forma parte de esta batería.

## Rechazos y paridad HTTP

Las pruebas permanentes crean sus propios archivos de entrada temporales y comparan
exactamente stdout, stderr y el código de salida contra el reporte público del núcleo;
luego verifican los hashes del código fuente. Ejercitan estos fallos sin modificar la
entrada entregada:

| Entrada | Resultado requerido |
|---|---|
| `for(let i=0; i<;i=i+1){` | Diagnósticos sintácticos ubicados, sin panic |
| `while(true)`, `do`, `foreach(x in [1])`, `for(let i=0; i<2; i=i+1)` | Cuerpos faltantes rechazados con diagnósticos ubicados |
| `const rows:integer[][]=[[1]];rows[0][0]=2;` | Rechazo semántico; las verificaciones de raíz const siguen siendo determinantes |
| `let rows:integer[][]=[[1]];rows[0]=true;` | Rechazo semántico de una forma anidada incompatible |
| `let a=[1];if(false){try{print(a[missing]);}catch(e){}}` | Diagnóstico semántico en código no ejecutado; los manejadores válidos ya no son archivos de prueba (fixtures) de rechazo |

Para cada compilación rechazada, `tac` es null y `codegen` está ausente. La CLI de TAC
construida, en modo texto, termina con 1, stdout vacío y diagnósticos ubicados en stderr;
el modo JSON termina con 1, con el reporte completo en stdout y stderr vacío. Analyze
heredado sigue terminando con 0 ante diagnósticos del lenguaje. Un `-format` de TAC
inválido sigue siendo un error de uso (salida 2), no un error del lenguaje. Estos códigos
describen el **binario construido**; `go run` envuelve los códigos de salida distintos de
cero del programa y no es el oráculo de la política de salida.

`TestMixedAcceptanceAPI` envía el archivo de prueba exitoso, los ciclos válidos y las
fuentes rechazadas a través de `newMux` usando httptest. Tanto `/api/compiscript/analyze`
como `/api/compiscript/compile` devuelven HTTP 200 para los reportes del lenguaje,
coincidiendo exactamente con sus reportes JSON del núcleo; la compilación agrega `tacText`
a partir de `Program.String()`. Los errores de transporte siguen gobernados por el
decodificador estricto de solicitudes existente. La suite existente de Node VM prueba por
separado el cableado de solicitud/renderizado/reinicio y de respuestas obsoletas. Aquí no
se afirma ningún resultado de apariencia ni de rendimiento en un navegador real.

## Elección de recuperación y límites

### Ejemplo dinámico y de excepciones

```sh
go run ./cmd/compiscript-tac testdata/compiscript/tac/dynamic-exceptions.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/dynamic-exceptions.cps
```

`TestDynamicExceptionsCompile` y
`TestDynamicExceptionsCLI` ejecutan su IR emitido con el oráculo acotado compartido:
`result=3`, `effects=11` y la salida `3\ndivision by zero\n11\n`.
Los efectos previos sobreviven al catch; la escritura inalcanzable de 100 no se ejecuta.
La misma suite cubre valores/llamadas sin tipo, fronteras tipadas, capturas retenidas,
operaciones inválidas estrictas, identidad de referencias y limpieza síncrona de manejadores.
`TestDynamicExceptionsFatalOracleFailures` demuestra por separado que los metadatos mal
formados y los presupuestos de instrucciones/profundidad omiten los manejadores del código
fuente. Consulte el
[contrato de guardas/manejadores](tac-design.md#valores-protegidos-y-excepciones-síncronas-t10).

Los archivos retenidos `independent_t10_verifier_test.go` e
`independent_t10_b1_recheck_test.go` también prueban contratos de campos desconocidos,
disposiciones heredadas y campos nominales anotados. Los campos sin anotación no se
estrechan a la etiqueta de su inicializador, incluido null; los valores ausentes permanecen
indefinidos. Los valores y tipos de retorno se verifican antes de la limpieza del manejador,
y un cuerpo catch no puede capturar su propio fallo con el manejador ya consumido. Estas son
pruebas actuales del núcleo y de la CLI construida, no comparaciones que requieran binarios
de referencia externos ni el historial de Git.

### Cuerpos de ciclo recuperados

ANTLR puede recuperar el encabezado incompleto de un ciclo antes de crear su bloque
requerido. Anteriormente, el mapeador indexaba una lista de bloques vacía. La corrección
quirúrgica KISS devuelve nil solo cuando ese bloque está ausente. La recuperación existente
de diagnósticos sintácticos marca entonces la sentencia como errónea, y el pipeline normal
de compilación la rechaza de forma atómica. No hay recuperación general de panics, cuerpo
vacío inventado, cambio en la gramática ni una nueva pasada del compilador. El mismo
auxiliar sirve a las cuatro construcciones de ciclo, por lo que las cuatro tienen pruebas
válidas de preservación y de frontera mal formada.

Los desplazamientos y tamaños de los metadatos usan celdas abstractas, no bytes ni una ABI
objetivo. Las guardas en tiempo de ejecución son contratos del IR ejercitados por el oráculo
de pruebas, no un intérprete del producto. Los errores sintácticos/semánticos reales y los
fallos de guardas estáticas de ámbito/inicialización descartan el TAC, incluso en código
muerto. En cambio, los operandos dinámicamente inválidos pueden generar IR protegido y
fallar en el oráculo; el IDE y la CLI no ejecutan ese IR.

## Batería de valores/efectos estáticos (T9)

`values_effects_test.go` y `values_effects_cli_test.go` verifican la ejecución JSON del
núcleo y de la CLI construida, la salida exacta, los efectos de valores/fallos, los reportes
deterministas, y que Analyze y los bytes del código fuente no cambien. Los casos incluyen
uniones anidadas perezosas, capturas, sombreado nominal, destinos/argumentos mutables,
residuo de flotantes, efectos con resultado null y celdas indefinidas nuevas. Se preserva el
envoltorio heredado del oráculo `runFunctionFrames`.

```sh
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/values-effects.cps
```

Esto emite TAC, no salida en tiempo de ejecución. La salida esperada del ejemplo en el
oráculo acotado es `3`, `1`, `null`, cada uno seguido de un salto de línea; `x=2`, `y=3`,
`count=1`. Los valores desconocidos/firmas sin tipo y try/catch están implementados por
T10; no son construcciones pendientes sin soporte. La política original de T9 de
estrechamiento de campos a null fue reemplazada por el contrato de campos desconocidos de
T10. La [matriz final](report.md) separa la cobertura de construcciones de las guardas en
tiempo de ejecución y de los controles de presentación.

## Procedencia de la verificación

En `633aad1`, los registros de implementación de T9/T10 reportan suites completas de Go
aisladas (18 paquetes probados, cuatro sin pruebas), build/vet forzado, 39 pruebas de Node
y verificaciones de compatibilidad exacta acotadas. La adición final de T10 de campos
nominales, **exclusiva de pruebas**, se realizó después de la prueba con la suite completa;
el agente principal volvió a ejecutar después los cuatro paquetes afectados, sin cambios en producción.
La actualización de la documentación vuelve a ejecutar solo las pruebas focalizadas
retenidas de T9/T10/API, Node y los ejemplos de la CLI externa; no fabrica un resultado más
reciente de suite completa/build/vet/ANTLR. Consulte [el informe](report.md#verificación-observada-y-procedencia)
para los conteos históricos separados; los conjuntos superpuestos no se suman.
