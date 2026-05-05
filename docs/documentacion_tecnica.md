# Documentación técnica — genAnaLex + YAPar

**Universidad del Valle de Guatemala**  
**Facultad de Ingeniería - Departamento de Ciencias de la Computación**

**Autores:**
- Luis Padilla
- Jonathan Díaz

**Repositorio:** [https://github.com/Jonialen/proyecto-compis](https://github.com/Jonialen/proyecto-compis)

---

## 1. Resumen ejecutivo

El proyecto ya implementa las dos partes del laboratorio:

- **Parte 1 — YALex:** generador y simulador de analizadores léxicos basados en DFA.
- **Parte 2 — YAPar:** parser de especificaciones `.yalp`, construcción de gramática formal, cálculo de **FIRST/FOLLOW**, colección **LR(0)**, tabla **SLR(1)**, simulador LR e **generador standalone** del parser.

La integración entre ambas partes ocurre mediante un **contrato de tokens compartido**, no mediante dependencias entre detalles internos del lexer y del parser.

---

## 2. Alcance actual del repositorio

### Incluye

- Parsing completo de archivos `.yal`.
- Expansión de macros y construcción directa de DFA.
- Minimización de DFA y tokenización con **Maximal Munch**.
- Generación de lexer standalone en Go.
- Parsing de archivos `.yalp` con `%token`, `IGNORE`, `%%`, `:`, `|` y `;`.
- Construcción de gramática aumentada.
- Cálculo de **nullable**, **FIRST** y **FOLLOW**.
- Construcción de colección canónica **LR(0)**.
- Construcción de tabla **SLR(1)** con detección de conflictos.
- Simulación sintáctica sobre `[]Token`.
- CLI separadas para lexer y parser: `cmd/yalex` y `cmd/yapar`.
- Generación de parser standalone en Go a partir de gramática y tabla LR.

### No incluye

- Construcción LR(1) completa o LALR.
- Resolución automática de conflictos shift/reduce o reduce/reduce.
- AST semántico o acciones semánticas embebidas en `.yalp`.
- Un formato automático de interoperabilidad entre **lexer standalone generado** y **parser standalone generado**; el parser standalone consume tokens vía JSON.

---

## 3. Estructura final del repositorio

```text
cmd/
  yalex/
    main.go                # CLI del lexer
  yapar/
    main.go                # CLI del parser

internal/
  dfa/                     # Árbol sintáctico, followpos, DFA, minimización
  generator/
    generator.go           # Generador standalone del lexer
    parser_gen.go          # Generador standalone del parser
  lexbuild/
    pipeline.go            # Pipeline reutilizable para compilar .yal y tokenizar
  lexer/                   # Simulador léxico sobre DFAEntries
  regex/                   # Normalización y postfix para regex
  shared/
    token.go               # Contrato compartido Token
  yalex/                   # Parser y expansor de YALex
  yapar/                   # Parser .yalp, gramática, FIRST/FOLLOW, LR(0), SLR, simulador

docs/
  documentacion_tecnica.md
  parte2/
    planeacion_tecnica_yapar.md
    backlog_tecnico_yapar.md
    structs_interfaces_base_yapar.md
```

---

## 4. Arquitectura general

### 4.1 Parte 1 — pipeline léxico

```text
.yal
  -> internal/yalex
  -> internal/regex
  -> internal/dfa
  -> DFAEntries
  -> internal/lexer (simulación)
  -> internal/generator/generator.go (standalone)
```

### 4.2 Parte 2 — pipeline sintáctico

```text
.yalp
  -> internal/yapar/parser.go
  -> internal/yapar/grammar.go
  -> internal/yapar/first_follow.go
  -> internal/yapar/items.go
  -> internal/yapar/table.go
  -> internal/yapar/simulator.go
  -> internal/generator/parser_gen.go
```

### 4.3 Integración lexer → parser

La integración real del repositorio sigue este flujo:

```text
.yal --CompileYALFile--> DFAEntries
src --TokenizeFile-----> []shared.Token
[]shared.Token --------> yapar.ParseTokens(grammar, table, tokens)
```

Punto CLAVE: el parser consume `[]shared.Token`, definido en `internal/shared/token.go`:

```go
type Token struct {
    Type   string
    Lexeme string
    Line   int
}
```

Esto mantiene desacoplamiento entre:

- implementación interna del lexer,
- runtime LR del parser,
- y código generado standalone.

---

## 5. Diseño de la Parte 2 (YAPar)

### 5.1 Parser de especificación `.yalp`

`internal/yapar/parser.go` interpreta:

- comentarios de bloque `/* ... */`,
- declaraciones `%token`,
- directiva `IGNORE`,
- separación por `%%`,
- producciones con `:`, `|` y `;`.

La salida base es:

```go
type YaparSpec struct {
    Tokens       []string
    IgnoreTokens map[string]bool
    Productions  []RawProduction
    StartSymbol  string
}
```

### 5.2 Modelo formal de gramática

`internal/yapar/grammar.go` transforma la especificación cruda en una gramática formal validada.

Puntos importantes:

- crea producción aumentada `S' -> S`,
- reserva `$` como fin de entrada,
- representa epsilon como **alternativa vacía**, no como símbolo explícito en la gramática fuente,
- prohíbe usar tokens ignorados dentro de producciones.

Estructuras principales:

```go
type Symbol struct {
    Name     string
    Terminal bool
}

type Production struct {
    ID   int
    Head string
    Body []Symbol
}

type Grammar struct {
    Start        string
    Augmented    string
    Terminals    map[string]bool
    NonTerminals map[string]bool
    Productions  []Production
    IgnoreSet    map[string]bool
}
```

### 5.3 FIRST, FOLLOW y nullable

`internal/yapar/first_follow.go` implementa iteración a punto fijo sobre la gramática formal.

Resultado principal:

```go
type FirstFollow struct {
    First    map[string]Set
    Follow   map[string]Set
    Nullable map[string]bool
}
```

Nota importante: **FOLLOW de gramática** y **followpos del lexer** NO son la misma cosa.

### 5.4 Colección LR(0)

`internal/yapar/items.go` implementa:

- `Closure`
- `Goto`
- `BuildCanonicalCollection`

La colección canónica se representa como estados con items LR(0) y un mapa de transiciones por símbolo.

### 5.5 Tabla SLR(1)

`internal/yapar/table.go` construye:

- **ACTION** para terminales,
- **GOTO** para no terminales,
- **accept** sobre la producción aumentada,
- **reduce** usando `FOLLOW(head)`.

Si la gramática no es SLR(1), el sistema devuelve `GrammarConflictError` con el estado, símbolo y tipo de conflicto detectado.

### 5.6 Simulador LR

`internal/yapar/simulator.go` ejecuta el parsing sobre `[]shared.Token`.

Comportamiento real:

1. filtra tokens declarados en `IGNORE`,
2. agrega el marcador `$`,
3. ejecuta acciones **shift / reduce / accept**,
4. reporta `SyntaxError` con línea, token recibido y esperados cuando falla.

---

## 6. CLIs actuales

### 6.1 CLI léxica — `cmd/yalex`

Uso actual:

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -src testdata/test.lisp
go run ./cmd/yalex -yal testdata/lexer.yal -out lexer_gen.go
go run ./cmd/yalex -yal testdata/lexer.yal -tree
```

Flags:

- `-yal` **obligatorio**
- `-src` tokeniza una entrada
- `-out` genera lexer standalone
- `-tree` genera `tree.dot`

### 6.2 CLI sintáctica — `cmd/yapar`

Uso actual:

```bash
go run ./cmd/yapar -yalp parser.yalp -table
go run ./cmd/yapar -yalp parser.yalp -out parser_gen.go
go run ./cmd/yapar -yalp parser.yalp -yal lexer.yal -src input.txt
```

Flags:

- `-yalp` **obligatorio**
- `-table` imprime la tabla SLR(1)
- `-out` genera parser standalone
- `-yal` y `-src` deben aparecer **juntos** para ejecutar análisis léxico + sintáctico de punta a punta

Si se usa solo `-yalp`, la CLI construye el pipeline sintáctico y valida que la gramática llegue hasta tabla SLR(1).

---

## 7. Generador standalone del parser

`internal/generator/parser_gen.go` serializa dentro del archivo generado:

- símbolo inicial,
- símbolo inicial aumentado,
- `IgnoreSet`,
- producciones,
- tabla ACTION,
- tabla GOTO.

El parser generado es autónomo y expone una CLI mínima:

```bash
go run parser_gen.go -tokens tokens.json
```

### 7.1 Contrato de entrada JSON

El archivo indicado por `-tokens` debe contener un JSON con `[]Token`:

```json
[
  { "type": "ID", "lexeme": "x", "line": 1 },
  { "type": "PLUS", "lexeme": "+", "line": 1 },
  { "type": "ID", "lexeme": "y", "line": 1 }
]
```

Campos esperados:

- `type`: nombre del token reconocido por la gramática
- `lexeme`: lexema original
- `line`: línea de origen

Comportamiento:

- los tokens incluidos en `IGNORE` se filtran antes del parsing,
- el parser agrega internamente `$`,
- si la entrada es válida imprime aceptación,
- si falla devuelve error sintáctico con línea y tokens esperados.

### 7.2 Implicación práctica

El **parser standalone** no recompila ni ejecuta el `.yal`; consume un stream de tokens ya materializado. Por eso, para conectarlo con un lexer externo o con el lexer standalone generado, se necesita un paso adaptador que produzca ese JSON.

---

## 8. Relación entre documentos

Los documentos de `docs/parte2/` quedan como soporte de diseño e implementación incremental.  
Este archivo describe el **estado final real del repositorio** y debe tomarse como referencia principal.

---

## 9. Estado de verificación

El repositorio contiene cobertura automatizada tanto para la parte léxica como para YAPar. A nivel de estructura actual destacan:

- tests para `cmd/yapar`,
- tests para parser `.yalp`, gramática, FIRST/FOLLOW, LR(0), tabla SLR(1) y simulador,
- tests para el generador standalone del parser,
- tests existentes de YALex, regex, DFA, lexer y generador léxico.

La base actual contiene **166 tests** detectables por nombre `func Test...` en archivos Go del repositorio.

---

## 10. Limitaciones y decisiones vigentes

- YAPar implementa hoy el camino **SLR(1)**, no LR(1) canónico.
- Los conflictos se detectan y reportan; no se resuelven automáticamente por precedencia.
- La integración más directa entre lexer y parser ocurre en memoria dentro de `cmd/yapar` mediante `internal/lexbuild` y `shared.Token`.
- El parser standalone trabaja con JSON de tokens porque su objetivo es desacoplar el runtime sintáctico del proceso de construcción léxica.

---

## 11. Conclusión

El proyecto ya no es solo un generador léxico. Su estado actual es un toolkit dividido en dos subsistemas:

1. **YALex** para construir y ejecutar analizadores léxicos.
2. **YAPar** para construir y ejecutar analizadores sintácticos SLR(1), incluyendo generación standalone.

La documentación queda alineada con la estructura real del repositorio, las CLIs actuales (`cmd/yalex`, `cmd/yapar`) y el pipeline sintáctico ya implementado en la Parte 2.
