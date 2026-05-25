# Documentación técnica — genAnaLex + YAPar

**Universidad del Valle de Guatemala**  
**Facultad de Ingeniería - Departamento de Ciencias de la Computación**

**Autores:**
- Luis Padilla
- Jonathan Díaz

**Repositorio:** [https://github.com/Jonialen/proyecto-compis](https://github.com/Jonialen/proyecto-compis)

---

## 1. Estado real del proyecto

### Respuesta corta

El repositorio YA NO es solo YALex + un parser SLR aislado.

Hoy el estado real es este:

- **YALex está funcional** para compilar `.yal`, tokenizar entradas y generar lexer standalone.
- **YAPar soporta tres backends ejecutables**: **LL(1)**, **SLR(1)** y **LALR**.
- **La arquitectura de YAPar es multi-método y table-centric**: distintos backends exponen una vista común de tabla.
- **La visualización/exportación es común**: tabla en `text`, tabla en `json` y **DOT del autómata cuando aplica**.
- **Existe una CLI comparativa** (`-compare`) para contrastar `ll1`, `lr0`, `slr`, `lr1`, `lalr`.
- **`lr0` y `lr1` NO están implementados como parsers ejecutables**: son placeholders explícitos para comparación/reporting.
- **El generador standalone del parser sigue orientado al camino LR/SLR existente**; no debe asumirse cobertura completa para todos los métodos.

## 2. Alcance actual

### Incluye

#### YALex

- Parsing de archivos `.yal`.
- Expansión de macros.
- Construcción y minimización de DFA.
- Tokenización con estrategia **Maximal Munch**.
- Generación de lexer standalone en Go.

#### YAPar

- Parsing de archivos `.yalp`.
- Construcción de gramática formal aumentada.
- Cálculo de `nullable`, `FIRST` y `FOLLOW`.
- Construcción de colección **LR(0)**.
- Construcción de colección **LR(1)** como insumo para LALR.
- Backend **LL(1)**.
- Backend **SLR(1)**.
- Backend **LALR**.
- Simulación sintáctica sobre `[]shared.Token`.
- Visualización/exportación común en texto y JSON.
- Exportación **DOT** del autómata cuando el método provee estados/transiciones LR.
- CLI de comparación entre métodos.
- Generación de parser standalone basada en una `TableView`.

#### IDE web

- Servidor local en `cmd/ide`.
- Interfaz gráfica en `web/`.
- Edición/carga de `.yal`, `.yalp` y entrada de prueba.
- Visualización de tokens, tablas, aceptación y autómata cuando aplica.

### No incluye

- Backend ejecutable **LR(0)**.
- Backend ejecutable **LR(1)** canónico.
- Soporte standalone garantizado para **todos** los métodos disponibles en la CLI.
- Resolución general por precedencia/asociatividad para conflictos.
- AST semántico o acciones semánticas embebidas en `.yalp`.
- Interoperabilidad automática completa entre lexer standalone generado y parser standalone generado; el parser standalone consume tokens vía JSON.

## 3. Mapa rápido del repositorio

```text
cmd/
  ide/                      # Servidor web de la interfaz gráfica
  yalex/                    # CLI del lexer
  yapar/                    # CLI del parser y comparador

internal/
  dfa/                      # Árbol sintáctico, followpos, DFA, minimización
  generator/                # Generadores standalone
  lexbuild/                 # Pipeline reutilizable de YALex
  lexer/                    # Simulador léxico
  regex/                    # Normalización y postfix para regex
  shared/                   # Tipos compartidos (Token)
  yalex/                    # Parser y expansor de YALex
  yapar/                    # Gramática, tablas, backends, visualización, comparación

docs/
  documentacion_tecnica.md  # Este documento
  parte2/                   # Planeación e historial de implementación

web/                        # Interfaz gráfica Furlantran
```

## 4. Arquitectura

### 4.1 Vista general

```text
.yal   -> YALex -> DFA -> tokenización -> []shared.Token
.yalp  -> YAPar -> gramática -> backend -> tabla/vista -> parseo
```

### 4.2 Contrato de integración

La integración real entre lexer y parser ocurre mediante `internal/shared/token.go`:

```go
type Token struct {
    Type   string
    Lexeme string
    Line   int
}
```

Eso desacopla:

- la implementación interna del lexer,
- el runtime del parser,
- y el código standalone generado.

### 4.3 Arquitectura multi-método en YAPar

La idea central NO es “un parser SLR con extras”, sino una capa común para varios métodos.

Piezas clave:

| Pieza | Rol |
|---|---|
| `internal/yapar/method.go` | Registro de métodos (`ll1`, `lr0`, `slr`, `lr1`, `lalr`) y factory de backends |
| `ExecutableParser` | Contrato común para parsear tokens |
| `TableView` | Vista común para inspección, exportación y generación |
| `internal/yapar/visualizer.go` | Render común a texto/JSON/DOT |
| `internal/yapar/comparison.go` | Reporte comparativo entre métodos |
| `internal/generator/parser_gen.go` | Generador standalone a partir de `TableView` |

### 4.4 Por qué “table-centric” importa

El proyecto normaliza la salida de los backends hacia una interfaz compartida:

```go
type TableView interface {
    ActionAt(state int, symbol string) (ActionKind, int, bool)
    GotoAt(state int, symbol string) (int, bool)
    States() []int
    Terminals() []string
    NonTerminals() []string
}
```

Eso permite reutilizar una misma capa para:

- imprimir tablas,
- serializar JSON,
- generar standalone,
- y comparar métodos sin acoplar la CLI a una implementación concreta.

## 5. Pipeline actual de YAPar

### Happy path

1. `cmd/yapar` carga el `.yalp`.
2. `internal/yapar/parser.go` construye `YaparSpec`.
3. `internal/yapar/grammar.go` construye la gramática formal aumentada.
4. `internal/yapar/first_follow.go` calcula `nullable`, `FIRST` y `FOLLOW`.
5. `internal/yapar/method.go` selecciona backend según `-method` o `-compare`.
6. El backend expone `Parse(tokens)` y `Table()`.
7. `visualizer.go` o `comparison.go` producen la salida visible.

### Detalle por backend

| Método | Construcción principal | Runtime |
|---|---|---|
| `ll1` | `BuildLL1Table` | simulación predictiva con stack |
| `slr` | `BuildCanonicalCollection` + `BuildSLRTable` | simulación LR sobre ACTION/GOTO |
| `lalr` | `BuildLR1Collection` + `MergeLR1States` + `BuildLALRTable` | simulación LR sobre tabla fusionada |
| `lr0` | placeholder | no implementado |
| `lr1` | placeholder | no implementado |

## 6. Componentes técnicos principales

### 6.1 Parser de `.yalp`

`internal/yapar/parser.go` interpreta:

- comentarios de bloque `/* ... */`,
- declaraciones `%token`,
- directiva `IGNORE`,
- separador `%%`,
- producciones con `:`, `|` y `;`.

Salida base:

```go
type YaparSpec struct {
    Tokens       []string
    IgnoreTokens map[string]bool
    Productions  []RawProduction
    StartSymbol  string
}
```

### 6.2 Gramática formal

`internal/yapar/grammar.go`:

- crea producción aumentada,
- reserva `$` como fin de entrada,
- maneja epsilon como alternativa vacía,
- impide usar tokens ignorados dentro de producciones.

### 6.3 FIRST/FOLLOW

`internal/yapar/first_follow.go` implementa iteración a punto fijo para:

- `Nullable`
- `First`
- `Follow`

### 6.4 Backend LL(1)

`internal/yapar/ll1_backend.go`:

- construye tabla LL(1) con `BuildLL1Table`,
- detecta recursión izquierda inmediata,
- detecta conflictos LL(1),
- ejecuta parsing predictivo con stack,
- adapta la tabla a `TableView` para exportación común.

### 6.5 Backend SLR(1)

El camino SLR sigue siendo importante y estable:

- `BuildCanonicalCollection` construye estados LR(0),
- `BuildSLRTable` llena ACTION/GOTO usando `FOLLOW`,
- `ParseTokens` ejecuta shift/reduce/accept,
- la tabla SLR puede exportarse por la misma capa común.

### 6.6 Backend LALR

`internal/yapar/lalr_backend.go`:

- construye colección LR(1),
- fusiona estados con mismo core mediante `MergeLR1States`,
- genera tabla LALR con `BuildLALRTable`,
- reutiliza el runtime LR basado en tabla,
- expone `TableView` compatible con la capa común.

Nota importante: la implementación actual puede resolver algunos conflictos `reduce/reduce` conservando la producción de menor ID y registrando un warning. Eso NO equivale a un sistema general de precedencia.

## 7. Visualización y exportación

### Respuesta corta

La visualización ya NO está atada a SLR.

Hoy existe una capa común en `internal/yapar/visualizer.go`.

### Formatos soportados

| Formato | Estado | Observaciones |
|---|---|---|
| `text` | Sí | imprime tabla legible en consola |
| `json` | Sí | serializa método, terminales, no terminales y estados |
| `dot` | Sí, cuando aplica | requiere estados/transiciones LR en el reporte |

### Cuándo aplica DOT

- **SLR**: sí, usando la colección LR(0).
- **LALR**: sí, reutilizando la vista LR(0) para el autómata visible.
- **LL(1)**: no aplica como autómata LR; la capa devuelve operación no soportada.
- **`-compare`**: no admite `dot`.

## 8. CLI reales del repositorio

### Convención de paths en esta documentación

- `testdata/yapar/...` = ejemplo real que existe en el repositorio.
- `archivo.yalp`, `archivo.yal`, `entrada.txt` = placeholder genérico, solo cuando se diga explícitamente.

Fixtures reales disponibles:

| Archivo | Uso |
|---|---|
| `testdata/yapar/arithmetic.yalp` | Gramática aritmética compatible con LL(1), SLR(1) y LALR |
| `testdata/yapar/arithmetic.yal` | Lexer compatible con los tokens `INT`, `PLUS`, `TIMES`, `LPAREN`, `RPAREN` |
| `testdata/yapar/valid.expr` | Entrada válida para pruebas end-to-end |
| `testdata/yapar/invalid.expr` | Entrada inválida para validar rechazo sintáctico |

### 8.1 `cmd/yalex`

Uso práctico:

```bash
go run ./cmd/yalex -yal testdata/lexer.yal -src testdata/test.lisp
go run ./cmd/yalex -yal testdata/lexer.yal -out lexer_gen.go
go run ./cmd/yalex -yal testdata/lexer.yal -tree
```

Reglas verificadas en código:

- `-yal` es obligatorio.
- Debe existir al menos uno de `-src`, `-out` o `-tree`.
- `-tree` escribe `tree.dot`.

### 8.2 `cmd/yapar`

Uso práctico:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method slr -table
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method ll1 -format json
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -method lalr -format dot
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -yal testdata/yapar/arithmetic.yal -src testdata/yapar/valid.expr
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -compare
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -compare -format json
```

Prueba de rechazo real:

```bash
go run ./cmd/yapar -yalp testdata/yapar/arithmetic.yalp -yal testdata/yapar/arithmetic.yal -src testdata/yapar/invalid.expr
```

### Flags relevantes de `cmd/yapar`

| Flag | Tipo | Estado real |
|---|---|---|
| `-yalp` | string | ruta del parser `.yalp`; **obligatorio** |
| `-yal` | string | ruta del lexer `.yal`; debe ir junto con `-src` |
| `-src` | string | archivo fuente a tokenizar y parsear; debe ir junto con `-yal` |
| `-out` | string | genera parser standalone |
| `-method` | string | `ll1`, `lr0`, `slr`, `lr1`, `lalr` |
| `-compare` | bool | corre todos los métodos válidos y compara |
| `-table` | bool | fuerza visualización en texto |
| `-format` | string | `text`, `json`, `dot` |

### Reglas de compatibilidad importantes

| Combinación | Resultado |
|---|---|
| `-compare` + `-method` | `-method` se ignora con warning |
| `-compare` + `-out` | inválido |
| `-compare` + `-format dot` | inválido |
| `-yal` sin `-src` | inválido |
| `-src` sin `-yal` | inválido |
| `-method ll1` + `-out` | no soportado |
| `-method lalr` + `-out` | no soportado |

## 9. Comparación entre métodos

`internal/yapar/comparison.go` construye un reporte por método con:

- método,
- tabla/render asociado cuando existe,
- duración,
- aceptación/rechazo si hubo tokens,
- error si el backend no existe o falla.

Eso permite dos cosas útiles:

1. comparar comportamiento de backends implementados,
2. mantener visibles los placeholders `lr0` y `lr1` sin fingir soporte real.

## 10. Standalone del parser

### Qué hace hoy

`internal/generator/parser_gen.go` genera un parser autónomo a partir de:

- gramática,
- `IgnoreSet`,
- producciones,
- filas ACTION,
- filas GOTO.

La generación se apoya en `TableView`, pero la CLI `cmd/yapar` restringe explícitamente algunos métodos.

### Contrato real

El parser generado consume tokens por JSON:

```bash
go run parser_gen.go -tokens tokens.json
```

Ejemplo:

```json
[
  { "type": "ID", "lexeme": "x", "line": 1 },
  { "type": "PLUS", "lexeme": "+", "line": 1 },
  { "type": "ID", "lexeme": "y", "line": 1 }
]
```

### Límite importante

Aunque el generador recibe una `TableView`, la CLI actual NO ofrece soporte standalone completo para todos los métodos. En particular:

- `ll1` con `-out` devuelve error explícito.
- `lalr` con `-out` devuelve error explícito.
- el camino operativo sigue alineado con el runtime de tabla LR/SLR ya consolidado.

NO hay que documentar esto como “standalone multi-método completo”, porque hoy sería falso.

## 11. Limitaciones vigentes

### Técnicas

- `lr0` y `lr1` siguen como **placeholders compare-only**.
- No existe resolución general de conflictos por precedencia/asociatividad.
- El standalone del parser no cubre uniformemente todos los métodos visibles en CLI.
- El parser standalone requiere un adaptador externo si se quiere conectar con otro lexer por JSON.

### De producto

- La **IDE/interfaz gráfica** ya existe como aplicación web local.
- La experiencia visual depende de `cmd/ide` y de los assets estáticos en `web/`.

## 12. Verificación estática usada para esta documentación

Se verificó contra código fuente, no contra supuestos históricos.

### Evidencia revisada

| Archivo | Qué confirmó |
|---|---|
| `cmd/yapar/main.go` | flags reales, reglas de compatibilidad, `-compare`, `-format`, `-table`, restricciones de `-out` |
| `internal/yapar/method.go` | métodos válidos `ll1`, `lr0`, `slr`, `lr1`, `lalr` |
| `internal/yapar/ll1_backend.go` | backend LL(1) real |
| `internal/yapar/lalr_backend.go` | backend LALR real |
| `internal/yapar/visualizer.go` | exportación común text/json/dot y DOT solo cuando aplica |
| `internal/yapar/comparison.go` | reporte comparativo multi-método |
| `internal/generator/parser_gen.go` | generación standalone basada en `TableView` |
| `cmd/yalex/main.go` | uso real de la CLI léxica |
| `cmd/ide/main.go` | servidor web local, APIs `/api/process` y `/api/health`, integración lexer/parser para la IDE |
| `web/index.html`, `web/app.js`, `web/style.css` | interfaz gráfica, editores, carga de archivos, resultados y render de autómata |

## 13. Conclusión

El proyecto hoy puede describirse con precisión así:

1. **YALex**: funcional para construir, simular y generar un lexer.
2. **YAPar**: multi-método, con backends **LL(1)**, **SLR(1)** y **LALR**, visualización/export común y modo comparativo.
3. **IDE web**: disponible mediante `go run ./cmd/ide`, con carga de archivos, ejecución de análisis y visualización de resultados.
4. **Límites reales**: `lr0`/`lr1` son placeholders y el standalone del parser no debe venderse como soporte completo para todos los métodos.

Este documento reemplaza cualquier afirmación obsoleta de que YAPar “solo soporta SLR(1)” o “todavía no tiene LALR”.
