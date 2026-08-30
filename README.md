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
| Compiscript | ANTLR frontend, project AST, semantic analysis, JSON CLI report, and IDE integration are available. |

La referencia técnica principal está en [`docs/documentacion_tecnica.md`](docs/documentacion_tecnica.md).

## Compiscript quick start

Compiscript follows a one-way pipeline: the ANTLR lexer/parser and generated Visitor in `internal/compiscript/frontend` build the project AST in `internal/compiscript/ast`; `internal/compiscript/semantic` then resolves scopes, names, and types. Consumers use the facade in `internal/compiscript` rather than depending on generated CST types.

Analyze a checked-in example from the CLI. The command prints the AST view, diagnostics, and scopes as JSON:

```bash
go run ./cmd/compiscript testdata/compiscript/acceptance/operators_valid.cps
```

Run the IDE and open `http://localhost:8080`; the Compiscript UI calls `POST /api/compiscript/analyze`:

```bash
go run ./cmd/ide
```

Run the focused and complete verification suites:

```bash
go test ./internal/compiscript/... ./cmd/compiscript ./cmd/ide -count=1
go test ./... -count=1
go build ./...
```

See the [Entrega 1 acceptance runbook](docs/semestre2/entrega1/acceptance.md), the [Compiscript grammar](docs/semestre2/entrega1/Compiscript.g4), and the [contribution evidence](docs/semestre2/entrega1/evidence/contributions.md).

## Requisitos

- Go **1.26.1+**
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
| `-out` | Genera parser standalone. | No usar con `-compare`; `ll1` y `lalr` no soportan este flujo. |

## Parser standalone

El parser generado con `-out` consume tokens en JSON:

```bash
go run parser_gen.go -tokens tokens.json
```

Formato esperado:

```json
[
  { "type": "ID", "lexeme": "x", "line": 1 }
]
```

## Limitaciones conocidas

- `lr0` y `lr1` aparecen en la comparación y en `-method`, pero hoy son placeholders no implementados.
- `-format dot` no aplica al modo comparativo y no todos los métodos tienen autómata exportable.
- El parser standalone no garantiza soporte completo para todos los métodos.
- La interfaz gráfica depende del servidor local `cmd/ide` y de assets web en `web/`.

## Más documentación

- [`docs/documentacion_tecnica.md`](docs/documentacion_tecnica.md): arquitectura, estado real y detalle técnico.
- [`docs/parte2/`](docs/parte2/): planeación y backlogs históricos.
- [`docs/semestre2/entrega1/acceptance.md`](docs/semestre2/entrega1/acceptance.md): current Compiscript acceptance and generator gates.
- [`docs/semestre2/entrega1/evidence/contributions.md`](docs/semestre2/entrega1/evidence/contributions.md): contribution counts derived from Git history.
