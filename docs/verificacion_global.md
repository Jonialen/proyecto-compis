# Verificación global del repositorio

**Estado actual: pruebas y análisis estático en verde** tras normalizar de
forma reproducible la generación ANTLR. La corrección parte del commit
`63af21736ccc343acef6b12ebc522b4b7d2d613a`; la gramática y las versiones de
ANTLR/runtime no se modificaron. Este estado describe los comandos observados,
no certifica cualquier entrada posible del lenguaje.

## Resultado actual de la corrección

Los comandos se ejecutaron secuencialmente en una copia con espacios en la
ruta y caché aislada, con Go 1.27.1 efectivo y `GOTOOLCHAIN=local`, sin red.

| Orden | Código de salida | Resultado |
|---|---|---|
| `go test ./... -count=1` | 0 | 16 paquetes OK y 4 paquetes sin tests |
| `go vet ./...` | 0 | Sin diagnósticos |
| `go vet -a ./...` | 0 | Sin diagnósticos, con análisis forzado |

Dos regeneraciones conservaron los ocho artefactos byte por byte. Una
comparación estricta contra la base confirmó la eliminación exclusiva de los
46 saltos artificiales y las dos etiquetas sin uso, más los efectos de líneas
vacías de `gofmt`; se conservaron API, ATN, datos de tokens y saltos reales.
Siete casos públicos válidos e inválidos de Compiscript mantuvieron stdout,
stderr y código de salida exactamente iguales.

## Resultado histórico anterior a la corrección

El código probado en este primer cierre fue
`a533d50123e10630f1987c1b0bb3c87770d9cd48`.

Las órdenes se ejecutaron una vez cada una, secuencialmente y en una copia
aislada del repositorio, con las dependencias ya disponibles y sin descargas.

| Orden | Código de salida | Resultado |
|---|---|---|
| `go test ./internal/compiscript/frontend -run '^(TestGeneratedArtifactsArePortableAndReproducible|TestGrammarCompatibilityAndFloatLiterals)$' -count=1` | 0 | Ambos tests antes bloqueados por ANTLR pasaron |
| `go test ./... -count=1` | 0 | 15 paquetes OK, 0 fallidos y 4 paquetes sin tests |
| `go vet ./...` | 1 | 46 diagnósticos `unreachable code` en el parser generado |

No se usó `-short`. Go y Node estaban disponibles; la salida no verbosa no
permite afirmar un conteo individual de tests omitidos.

## Entorno y ANTLR

- Go: `go1.27.1 darwin/arm64`; `go.mod` declara `go 1.26.1`.
- Java: OpenJDK `17.0.20.1`; Node: `24.21.0`.
- Herramienta ANTLR: `4.13.2`; runtime Go declarado: `v4.13.1`.
- JAR restaurado desde <https://www.antlr.org/download/antlr-4.13.2-complete.jar>.
- SHA-256 verificado contra `tools/antlr/antlr-4.13.2-complete.jar.sha256`:
  `eae2dfa119a64327444672aff63e9ec35a20180dc5b8090b7a6ab85125df4d76`.
- El JAR es caché local ignorada por Git; no se agrega el binario al repositorio.

## Hallazgo histórico de vet

Los 46 diagnósticos se ubican en
`internal/compiscript/frontend/generated/compiscript_parser.go`, desde
`589:2` hasta `8662:2`. Por ejemplo, las líneas 588–589 contienen:

```go
return localctx
goto errorExit // Trick to prevent compiler error if the label is not used
```

El código generado conserva esos bytes en la revisión probada. No se editaron
sus saltos manualmente, no se cambiaron versiones y no se desactivó ningún
analizador para presentar un resultado verde. No se determinó si el diagnóstico
varía con otra versión de Go.

## Preservación del primer cierre

En aquella ejecución, los hashes de 243 archivos rastreados permanecieron
iguales antes y después de las pruebas, tanto en el checkout original como
en la copia aislada. La
gramática y los ocho artefactos generados no tuvieron diferencias tras la
regeneración. Los archivos sin seguimiento preexistentes se preservaron.

## Corrección reproducible de la generación

Se autorizó resolver la compatibilidad sin cambiar `Compiscript.g4`, la
herramienta ANTLR 4.13.2, el runtime Go declarado ni los analizadores de vet.
`scripts/generate-compiscript.sh` ahora ejecuta
`scripts/normalize-antlr-go.go` dentro del directorio temporal de generación,
antes de `gofmt` y de publicar los artefactos. Este paso requiere el ejecutable
`go`, además de Java y `gofmt`.

El normalizador usa el AST de Go y se limita al encabezado ANTLR 4.13.2 y al
patrón final anotado `return localctx` seguido del salto artificial. Conserva
los saltos de recuperación reales y solo retira una etiqueta si ya no tiene
referencias. Una forma inesperada del patrón hace fallar la generación antes
de publicar; no se reescribe código ordinario ni otra versión de ANTLR.

La regeneración real eliminó 46 saltos artificiales y dos etiquetas sin uso,
sin alterar la gramática. Las pruebas cubren preservación de recuperación,
idempotencia, rechazo seguro y publicación atómica ante una normalización
fallida.

La primera comprobación local sin forzar el análisis había devuelto éxito
incluso con el código original. Con la misma versión efectiva Go 1.27.1,
`go vet -a` reprodujo los 46 avisos. Por eso `TestGeneratedParserPassesVet`
ejecuta `go vet -a` sobre el paquete generado, sin depender de un resultado
previo de caché ni desactivar el analizador. La regresión fue observada en RED
y luego en GREEN tras regenerar mediante el script.

## Corrección separada: entrada malformada

La entrada exacta `if (true) { print(1);` ya no provoca un pánico. La prueba
pública `TestMissingClosingBrace` reprodujo primero el índice fuera de rango
en `mapper_statement.go:68` sobre la base `a51076d`. La recuperación del parser
puede producir un contexto `if` sin cuerpo; el mapper accedía a `bodies[0]`
antes de que `recoverProgram` pudiera procesar los diagnósticos.

El mapper ahora detecta ese contexto incompleto y reutiliza `recoverStatement`
con el span calculado por el builder. No inserta una llave ni modifica la
gramática. La prueba verifica exactamente los dos nodos `BadStmt` recuperados
y los dos diagnósticos sintácticos, incluidos mensajes, orden y spans vacíos
en EOF: offset 21, línea 1, columna 22.

La regresión pasó de RED (pánico) a GREEN. Cuatro controles válidos cubren el
caso cerrado, `else` con bloques, anidamiento y cuerpos sin llaves. En una copia
aislada, sin red y con Go 1.27.1, `go test ./... -count=1` terminó con 16 paquetes
OK y 4 sin tests; `go vet -a ./...` terminó sin diagnósticos. La CLI conservó
stdout, stderr y código de salida byte por byte frente a `a51076d` en esos
cuatro controles y un bloque independiente. Para la entrada malformada devuelve
el informe con ambos diagnósticos y stderr vacío; mantiene el código de salida
0 que la CLI ya utiliza para informes con errores del lenguaje.
La regresión HTTP incluye la misma entrada: conserva la respuesta 200 de la
API de análisis y verifica que el informe coincida con la fachada pública,
incluidos los diagnósticos, sin propagar el pánico.

La gramática, los ocho artefactos generados, las dependencias y los scripts de
generación permanecen sin cambios. Esta corrección se limita al fallo
reproducido; no certifica la recuperación de cualquier otra entrada malformada.
