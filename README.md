# genAnaLex + YAPar

Toolkit en Go para construir y ejecutar analizadores de compiladores en dos capas:

- **YALex**: compila especificaciones `.yal`, genera DFA, tokeniza entradas y puede generar un lexer standalone.
- **YAPar**: compila especificaciones `.yalp`, construye gramática + tabla **SLR(1)**, simula parsing y puede generar un parser standalone.

La referencia técnica completa está en `docs/documentacion_tecnica.md`. Este README solo cubre el uso rápido real del repositorio.

## Estructura actual

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

- Go 1.26+
- Graphviz opcional, solo si quieres visualizar `tree.dot`

## Uso rápido

### 1) YALex: generar o simular lexer

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -src testdata/test.lisp
go run ./cmd/yalex -yal testdata/lexer.yal -out lexer_gen.go
go run ./cmd/yalex -yal testdata/lexer.yal -tree
```

Notas:

- `-yal` es obligatorio.
- Debes usar al menos uno de `-src`, `-out` o `-tree`.
- `-tree` genera `tree.dot` en el directorio actual.

### 2) YAPar: construir tabla, simular parser o generar parser standalone

```bash
go run ./cmd/yapar -yalp parser.yalp -table
go run ./cmd/yapar -yalp parser.yalp -out parser_gen.go
go run ./cmd/yapar -yalp parser.yalp -yal lexer.yal -src input.txt
```

Notas:

- `-yalp` es obligatorio.
- `-yal` y `-src` deben usarse juntos para ejecutar el flujo completo lexer → parser.
- Si usas solo `-yalp`, la CLI valida el pipeline sintáctico hasta la tabla SLR(1).

## Flujo básico del proyecto

```text
.yal  -> YALex -> tokens
.yalp -> YAPar -> tabla SLR(1)
tokens + tabla -> simulación sintáctica
```

Flujo de punta a punta desde las CLIs:

1. Compilas el lexer desde `.yal`.
2. Tokenizas el archivo fuente.
3. Compilas la gramática desde `.yalp`.
4. Ejecutas el parsing SLR(1) sobre esos tokens.

## Parser standalone

El parser generado con `-out` es autónomo y recibe tokens por JSON:

```bash
go run parser_gen.go -tokens tokens.json
```

Contrato esperado:

```json
[
  { "type": "ID", "lexeme": "x", "line": 1 }
]
```

Campos:

- `type`: token reconocido por la gramática
- `lexeme`: lexema original
- `line`: línea de origen

## Documentación adicional

- `docs/documentacion_tecnica.md`: estado técnico real del repositorio
- `docs/parte2/`: planeación e implementación de YAPar
