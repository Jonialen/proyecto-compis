# Evidencia de aceptación de la fase MIPS

La implementación fue verificada con generación real de ensamblador y ejecución en MARS. Esta evidencia cubre los escenarios y comandos indicados; no garantiza todos los programas posibles, terminación ni una calificación.

## Resultado final observado

La verificación global se ejecutó en una copia temporal de las fuentes finales, con HOME/GOCACHE/TMPDIR aislados, dependencias Go ya disponibles y `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`. Se copiaron los JAR locales verificados de MARS y ANTLR para no descargar herramientas ni regenerar archivos en el checkout original.

| Comprobación | Resultado |
|---|---|
| `go test -json ./... -count=1` | Salida 0; 21 paquetes aprobados, 2379 eventos de pruebas aprobadas —incluye tests padres y subtests—; aproximadamente 110 segundos |
| `TestMIPSPhase3Acceptance` dentro de la suite global | 150 escenarios aprobados mediante `.cps` → compilador → MIPS → MARS |
| `go vet -a ./...` | Salida 0, sin diagnósticos |
| `node --test web/*.test.cjs` | 41/41 aprobadas, cero fallidas y cero omitidas |
| `bash scripts/test-mars.sh` | Ejecución válida y rechazo de ensamblador inválido aprobados |
| `bash scripts/test-mips.sh` | Demo completa por CLI y MARS aprobada: stdout exacto, stderr de compilación vacío, stderr MARS una línea vacía y fuente sin cambios |
| Enlaces de las guías modificadas | Válidos |
| Hashes de fuentes originales antes/después | Sin cambios; regeneración ANTLR limitada a la copia temporal |

Los cinco eventos JSON `skip` sin campo `Test` corresponden a **paquetes sin archivos de prueba**, no a casos omitidos: `cmd/sql`, `cmd/yalex`, `internal/compiscript/frontend/generated`, `internal/lexbuild` e `internal/shared`. No se usó `-short`.

## Qué comprueba la matriz ampliada

[`mips_acceptance_test.go`](../../../internal/compiscript/mips_acceptance_test.go) reutiliza las matrices existentes de valores/efectos, valores dinámicos/excepciones, objetos y listas. Para resultados escalares agrega observaciones públicas con `print`, ordenadas por nombre; no lee memoria privada del simulador ni reemplaza MIPS con el oráculo TAC.

Cada ejecución comprueba compilación, inmutabilidad del fuente, stderr, código de salida y salida observable. Los programas con fallos conservan el prefijo exacto producido antes del fallo y la clase de error que la matriz anterior especificaba. Los casos con catch comprueban efectos posteriores, alias y contadores a través de la salida. Se imponen límites de tiempo e instrucciones para que una traducción que no termina falle la prueba.

La batería dedicada en `internal/compiscript/mips/` agrega pruebas positivas y negativas de control, aritmética, UTF-8/NUL, frames recursivos, presión de temporales/argumentos, closures, metadata inválida, objetos, herencia, listas, contratos dinámicos y handlers. Las pruebas CLI/API/DOM verifican el flujo del producto, no solamente el emisor.

## Fallos detectados y corregidos

- El primer corte truncaba el cociente de `%` a 32 bits y trataba strings como ASCII/NUL-terminados. Los casos de números grandes, UTF-8 y NUL reprodujeron los fallos; la reducción binaria y escritura por longitud los corrigieron.
- Los globals declarados sin valor se rechazaban antes de que una función pudiera escribirlos. Un sentinel de declaración separa ese caso de una declaración que todavía no se ejecutó; las lecturas sin valor siguen fallando.
- Una clase no utilizada eludía la validación de metadata. La validación ahora es incondicional y atómica.
- La primera aceptación ampliada pasó 149/150 escenarios: faltaba el descriptor de una excepción dentro de una lista. El caso conservado produjo `invalid value type "exception"`; agregar la etiqueta física ya existente corrigió la omisión. El recheck enfocado y luego los 150 escenarios de la suite global pasaron.

No se eliminaron ni debilitaron los ejemplos que encontraron esos fallos.

## Compatibilidad y verificación independiente

Una comparación diferencial temporal contra el corte anterior a la integración confirmó igualdad exacta de stdout, stderr y salida para Analyze y TAC —texto/JSON y rechazo de la nueva opción de formato en la CLI antigua—. Los endpoints anteriores también conservaron status, headers y body para éxito, error de fuente y campo desconocido. Ese script no forma parte de las pruebas versionadas ni depende de Git durante su ejecución normal.

Dos verificaciones independientes acotadas aprobaron:

1. Herramienta MARS local: checksum, ejecución, ensamblador inválido, syscall inválida, JAR ausente/corrupto y aislamiento de preferencias.
2. Integración CLI/API/IDE: cuatro paquetes Go focalizados, las 41 pruebas web y la demo pública CLI→MARS, en una copia aislada del commit de integración. Revisó S4/S5/S6 funcional/S7 sin bloqueadores.

## Límites y controles humanos

- No hubo herramienta de navegador disponible. Las comprobaciones DOM/HTTP son funcionales; **no se afirma validación visual en un navegador real**. Revisá carga, edición y descarga en el navegador de la presentación.
- Los aportes individuales reales de los tres integrantes deben comprobarse con el equipo. No se inventaron autores ni se asegura cumplimiento de ese requisito a partir de estos commits.
- MARS, binary64, memoria monotónica y límites de layouts/impresión se describen en la [guía de ejecución](README.md). No se afirma soporte de recursos ilimitados ni de todos los simuladores MIPS.
- La ejecución sin red aquí utilizó cachés locales existentes. Una máquina nueva necesita obtener previamente dependencias y herramientas; el IDE conserva sus dependencias CDN anteriores.
- Los cambios permanecen en una rama local. No se hizo push, PR ni merge.
