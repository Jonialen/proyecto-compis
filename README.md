# genAnaLex + YAPar

Toolkit en Go para la parte léxica y sintáctica del proyecto.

## Respuesta corta

- **YALex sí está funcional**: compila `.yal`, tokeniza archivos y puede generar lexer standalone.
- **YAPar ya es multi-método**: soporta backends **LL(1)**, **SLR(1)** y **LALR**.
- **La CLI de YAPar también tiene modo comparativo** con `-compare` para `ll1`, `lr0`, `slr`, `lr1`, `lalr`.
- **`lr0` y `lr1` NO están implementados como backends ejecutables**: hoy existen como placeholders comparables y reportan error explícito.
- **El standalone del parser NO debe asumirse universal**: el camino real sigue orientado a la tabla LR/SLR existente; `ll1` y `lalr` no están soportados para `-out`.

La referencia técnica principal está en `docs/documentacion_tecnica.md`.

## Estructura mínima

```text
cmd/
  yalex/   # CLI del lexer
  yapar/   # CLI del parser

internal/
  dfa/
  generator/
  lexbuild/
  lexer/
  regex/
  shared/
  yalex/
  yapar/

docs/
testdata/
```

## Requisitos

- Go **1.26.1+**
- Graphviz opcional si quieres renderizar DOT generado

## Uso rápido

### 1) YALex

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -src testdata/test.lisp
go run ./cmd/yalex -yal testdata/lexer.yal -out lexer_gen.go
go run ./cmd/yalex -yal testdata/lexer.yal -tree
```

Checklist real:

- `-yal` es obligatorio.
- Debes pasar al menos uno de `-src`, `-out` o `-tree`.
- `-tree` escribe `tree.dot` en el directorio actual.

### 2) YAPar — construir backend, visualizar, comparar o ejecutar

Fixtures reales listos para copiar/pegar:

- Parser: `testdata/yapar/arithmetic.yalp`
- Lexer: `testdata/yapar/arithmetic.yal`
- Entrada válida: `testdata/yapar/valid.expr`
- Entrada inválida: `testdata/yapar/invalid.expr`

#### Camino normal

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method slr -table
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method lalr -format dot
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method ll1 -format json
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -yal testdata/yapar/arithmetic.yal -src testdata/yapar/valid.expr
```

#### Modo comparativo

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -compare
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -compare -format json
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -yal testdata/yapar/arithmetic.yal -src testdata/yapar/valid.expr -compare
```

Si ves rutas como `mi_parser.yalp` o `entrada.txt` en notas históricas, trátalas como **placeholders**. Los comandos anteriores usan archivos **reales** que sí existen en el repo.

#### Flags relevantes de `cmd/yapar`

| Flag | Qué hace | Notas reales |
|---|---|---|
| `-yalp` | Ruta del parser `.yalp` | **Obligatorio** |
| `-method` | Método solicitado | `ll1`, `lr0`, `slr`, `lr1`, `lalr` |
| `-compare` | Ejecuta comparación entre métodos | Ignora `-method`; no permite `-out` ni `-format dot` |
| `-format` | Salida de visualización | `text`, `json`, `dot` |
| `-table` | Atajo para tabla en texto | Equivale a salida `text` |
| `-yal` | Lexer `.yal` | Debe usarse junto con `-src` |
| `-src` | Archivo fuente a tokenizar y parsear | Debe usarse junto con `-yal` |
| `-out` | Genera parser standalone | No usar con `-compare`; `ll1` y `lalr` no soportados |

## Qué sí exporta YAPar hoy

| Salida | Estado |
|---|---|
| Tabla en texto | Sí |
| Tabla en JSON | Sí |
| DOT del autómata cuando aplica | Sí |
| Comparación entre métodos | Sí |

## Limitaciones actuales IMPORTANTES

- La **IDE/interfaz gráfica** sigue pendiente.
- `lr0` y `lr1` aparecen en la comparación y en `-method`, pero hoy son **placeholders no implementados**.
- El **parser standalone** no representa soporte completo para todos los métodos.
- `-format dot` no aplica al modo comparativo y no todos los métodos tienen autómata exportable.

## Parser standalone

El parser generado con `-out` consume tokens por JSON:

```bash
go run parser_gen.go -tokens tokens.json
```

Contrato esperado:

```json
[
  { "type": "ID", "lexeme": "x", "line": 1 }
]
```

## Más detalle

- `docs/documentacion_tecnica.md`: documentación técnica principal y estado real.
- `docs/parte2/`: planeación y backlogs históricos.
