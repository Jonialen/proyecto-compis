# genAnaLex + YAPar

Toolkit en Go para construir y ejecutar analizadores léxicos con **YALex** y analizadores sintácticos con **YAPar**. El proyecto permite compilar especificaciones `.yal`, tokenizar entradas, construir tablas sintácticas y comparar distintos métodos de parsing.

**Repositorio:** [https://github.com/Jonialen/proyecto-compis](https://github.com/Jonialen/proyecto-compis)

## Estado actual

| Componente | Estado |
|---|---|
| YALex | Funcional: compila `.yal`, tokeniza archivos y genera lexer standalone. |
| YAPar LL(1) | Backend ejecutable disponible. |
| YAPar SLR(1) | Backend ejecutable disponible. |
| YAPar LALR | Backend ejecutable disponible. |
| YAPar LR(0) / LR(1) | Existen como opciones comparables, pero **no** como backends ejecutables. |
| Parser standalone | Disponible para el camino basado en tablas LR/SLR; no debe asumirse universal para todos los métodos. |
| IDE/interfaz gráfica | Disponible como app web servida desde `cmd/ide`. |
| Compiscript | Frontend ANTLR, análisis semántico, generación TAC para los tipos de construcción válidos del AST, símbolos extendidos e IDE con un núcleo compartido. Contratos y evidencia en el informe de Entrega 2. |

La referencia técnica principal está en [`docs/semestre1/documentacion_tecnica.md`](docs/semestre1/documentacion_tecnica.md).

## Compiscript: Entrega 2

La [guía de ejecución e índice](docs/semestre2/entrega2/README.md) reúne el [informe y matriz del PDF](docs/semestre2/entrega2/report.md), la [arquitectura](docs/semestre2/entrega2/architecture.md), la [referencia TAC](docs/semestre2/entrega2/tac-design.md), la [batería de aceptación](docs/semestre2/entrega2/acceptance.md) y el [manual del IDE](docs/semestre2/entrega2/ide.md). T9/T10 resolvieron las brechas iniciales de print, resto, ternarias, asignaciones con valor, almacenamiento indefinido/null, valores sin anotación y try/catch. Se documentan las guardas estáticas y los fallos dinámicos; no se promete corrección exhaustiva, una calificación ni certificación visual del navegador.

Desde la raíz del repositorio, generar TAC de un ejemplo y consultar el reporte JSON con símbolos y entornos:

```bash
go run ./cmd/compiscript-tac testdata/compiscript/tac/numeric.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/acceptance.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/values-effects.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/dynamic-exceptions.cps
go run ./cmd/compiscript testdata/compiscript/tac/acceptance.cps
```

El último comando conserva el análisis sin generación. La CLI **genera código, no lo ejecuta**: print y try/catch se traducen a IR, no se ejecutan en la interfaz. Para el IDE, iniciar el servidor y abrir `http://localhost:8080`; Compiscript permite cargar o escribir `.cps`, compilar y consultar TAC, errores ubicados y símbolos extendidos. Analyze y YAPar siguen disponibles:

```bash
go run ./cmd/ide
```

Prueba focalizada de aceptación y pruebas de conexiones de la interfaz:

```bash
go test ./internal/compiscript/frontend ./internal/compiscript ./cmd/ide -run 'Test(MalformedForMissingBody|MissingLoopBodies|AcceptanceAtomicRecovery|AcceptanceMixedExecution|AcceptanceRecoveryCLI|AcceptanceValidLoops|MixedAcceptanceAPI)$' -count=1
go test ./internal/compiscript ./cmd/ide -run '^Test(ValuesEffects|DynamicExceptions|IndependentT9|IndependentT10|MixedAcceptanceAPI)' -count=1
node --test web/*.test.cjs
```

La suite completa puede regenerar archivos ANTLR ignorados: ejecutarla en una copia aislada, siguiendo la [política de reproducción](docs/semestre2/entrega2/acceptance.md#reproducible-tests). Las pruebas de VM/DOM no certifican apariencia real del navegador; CodeMirror conserva dependencias CDN. Entrega 1 mantiene su [matriz histórica](docs/semestre2/entrega1/compliance.md).

## Requisitos

- Go **1.26.1+**
- Node.js para pruebas web; entorno observado: Go 1.27.2 y Node 24.21.0.
- Java para regenerar ANTLR; entorno observado: Java 17.0.20.1 y jar ANTLR 4.13.2 en caché. El runtime Go es v4.13.1 según `go.mod`.
- Graphviz opcional para renderizar archivos DOT generados

## Estructura del proyecto

```text
cmd/
  ide/     # Servidor web para la interfaz gráfica
  yalex/   # CLI del lexer
  yapar/   # CLI del parser y comparador

internal/
  dfa/        # Árbol sintáctico, followpos, DFA y minimización
  generator/  # Generadores standalone
  lexbuild/   # Pipeline reutilizable de YALex
  lexer/      # Simulador léxico
  regex/      # Normalización y postfix para regex
  shared/     # Tipos compartidos
  yalex/      # Parser y expansor de YALex
  yapar/      # Gramática, tablas, backends, visualización y comparación

docs/
testdata/
web/
  # Interfaz gráfica Furlantran
```

## Uso rápido

### 0. Ejecutar la interfaz gráfica

Levantar la IDE web local:

```bash
go run ./cmd/ide
```

Luego abrir:

```text
http://localhost:8080
```

Desde la interfaz se puede:

- cargar o editar archivos `.yal`, `.yalp` y entradas de prueba;
- seleccionar métodos `LL(1)`, `SLR(1)` y `LALR`;
- ver tokens, tablas, resumen de aceptación y autómata cuando el método lo permite.

### 1. Ejecutar YALex

Tokenizar una entrada con una especificación `.yal`:

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -src testdata/test.lisp
```

Generar un lexer standalone:

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -out lexer_gen.go
```

Generar el árbol en formato DOT:

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -tree
```

Notas importantes:

- `-yal` es obligatorio.
- Debe usarse al menos una salida: `-src`, `-out` o `-tree`.
- `-tree` escribe `tree.dot` en el directorio actual.

### 2. Ejecutar YAPar

Fixtures disponibles para probar el parser:

| Archivo | Uso |
|---|---|
| `testdata/yapar/arithmetic.yalp` | Especificación del parser. |
| `testdata/yapar/arithmetic.yal` | Especificación del lexer. |
| `testdata/yapar/valid.expr` | Entrada válida. |
| `testdata/yapar/invalid.expr` | Entrada inválida. |

Construir el parser con el método por defecto:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp
```

Mostrar tabla SLR en texto:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method slr -table
```

Exportar autómata LALR en DOT:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method lalr -format dot
```

Exportar tabla LL(1) en JSON:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method ll1 -format json
```

Tokenizar y parsear una entrada real:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -yal testdata/yapar/arithmetic.yal -src testdata/yapar/valid.expr
```

## Comparar métodos de parsing

YAPar puede comparar `ll1`, `lr0`, `slr`, `lr1` y `lalr` desde la CLI:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -compare
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -compare -format json
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -yal testdata/yapar/arithmetic.yal -src testdata/yapar/valid.expr -compare
```

> Si ves rutas como `mi_parser.yalp` o `entrada.txt` en notas históricas, trátalas como placeholders. Los comandos anteriores usan fixtures reales del repositorio.

## Flags principales de `cmd/yapar`

| Flag | Descripción | Notas |
|---|---|---|
| `-yalp` | Ruta del parser `.yalp`. | Obligatorio. |
| `-method` | Método solicitado. | `ll1`, `lr0`, `slr`, `lr1`, `lalr`. |
| `-compare` | Ejecuta comparación entre métodos. | Ignora `-method`; no permite `-out` ni `-format dot`. |
| `-format` | Formato de visualización. | `text`, `json`, `dot`. |
| `-table` | Atajo para tabla en texto. | Equivale a salida `text`. |
| `-yal` | Lexer `.yal`. | Debe usarse junto con `-src`. |
| `-src` | Archivo fuente a tokenizar y parsear. | Debe usarse junto con `-yal`. |
| `-out` | Genera parser standalone LL1, SLR o LALR. | No usar con `-compare`; requiere una gramática válida para el método. |

## Parser standalone

La CLI permite generar parsers **LL1, SLR y LALR**. LL1 usa un runtime predictivo;
SLR y LALR comparten el runtime LR. Los [ejemplos reproducibles](docs/semestre1/parte2/guia_demos_yapar.md#parser-standalone-ll1)
incluyen generación, compilación y ejecución con tokens JSON válidos e inválidos.

El parser generado con `-out` consume tokens en JSON:

El token de tipo `$` está reservado para el fin de entrada y lo agrega el
runtime. No debe incluirse en el JSON: los parsers integrados y standalone lo
rechazan con un diagnóstico de ubicación. Un lexema cuyo texto sea `$` no es
el marcador si su tipo de token es otro.

```bash
go run parser_gen.go -tokens tokens.json
```

Formato esperado:

```json
[
  { "type": "ID", "lexeme": "x", "line": 1 }
]
```

## Integración lexer-parser standalone

Los parsers generados también pueden leer fuente mediante
`-lexer <ejecutable> -src <archivo>`, sin preparar JSON manualmente. El lexer
generado ofrece `-format json`; su salida de texto sigue siendo el valor
predeterminado. Véase la [integración reproducible](docs/semestre1/parte2/guia_demos_yapar.md#integración-lexer-parser-standalone).

## Limitaciones conocidas

- `lr0` y `lr1` aparecen en la comparación y en `-method`, pero hoy son placeholders no implementados.
- `-format dot` no aplica al modo comparativo y no todos los métodos tienen autómata exportable.
- El parser standalone no garantiza soporte completo para todos los métodos.
- La interfaz gráfica depende del servidor local `cmd/ide` y de assets web en `web/`.

## Más documentación

- [`docs/verificacion_global.md`](docs/verificacion_global.md): verificación global, entorno ANTLR y corrección reproducible de los avisos de `go vet`.
- [`docs/semestre1/parte2/guia_demos_yapar.md`](docs/semestre1/parte2/guia_demos_yapar.md): casos YAPar reproducibles, comandos y resultados esperados en CLI e IDE.
- [`docs/semestre1/documentacion_tecnica.md`](docs/semestre1/documentacion_tecnica.md): arquitectura, estado real y detalle técnico.
- [`docs/semestre1/parte2/`](docs/semestre1/parte2/): planeación y backlogs históricos.
- [`docs/semestre2/entrega1/compliance.md`](docs/semestre2/entrega1/compliance.md): current Compiscript requirement status, source contradictions, and remaining work.
- [`docs/semestre2/entrega1/acceptance.md`](docs/semestre2/entrega1/acceptance.md): current Compiscript acceptance and generator gates.
- [`docs/semestre2/entrega1/evidence/contributions.md`](docs/semestre2/entrega1/evidence/contributions.md): contribution counts derived from Git history.
