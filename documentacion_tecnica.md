# Documentación Técnica — YAPar

---

## 1. Visión general

YAPar es un generador de parsers SLR(1) escrito en Go. A partir de una gramática libre de contexto descrita en un archivo `.yalp`, construye las tablas de análisis sintáctico (ACTION y GOTO) y puede:

- Verificar si una cadena de tokens es aceptada por la gramática.
- Generar un parser standalone en Go (`.go`) listo para compilar y distribuir.
- Integrarse con YALex para tokenizar un archivo fuente y parsearlo en un solo paso.

El proyecto implementa de cero el algoritmo SLR(1) clásico:

```
Gramática → Aumentación → FIRST/FOLLOW → Colección canónica LR(0) → Tabla SLR(1) → Simulación
```

---

## 2. Estructura de archivos

```
proyecto-compis/
├── cmd/
│   └── yapar/
│       ├── main.go           # Punto de entrada del binario yapar
│       └── main_test.go      # Tests de integración del CLI
│
├── internal/
│   ├── yapar/                # Núcleo del parser generator
│   │   ├── parser.go         # Parser del archivo .yalp
│   │   ├── grammar.go        # Construcción de la gramática formal
│   │   ├── first_follow.go   # Cómputo de FIRST / FOLLOW / nullable
│   │   ├── items.go          # Ítems LR(0) y colección canónica
│   │   ├── table.go          # Construcción de la tabla SLR(1)
│   │   ├── simulator.go      # Simulador del parser LR en tiempo de ejecución
│   │   └── errors.go         # Tipos de error (SpecError, GrammarConflictError, SyntaxError)
│   │
│   ├── generator/
│   │   └── parser_gen.go     # Generador de código Go para el parser standalone
│   │
│   ├── lexbuild/
│   │   └── pipeline.go       # Puente entre YALex y YAPar
│   │
│   ├── shared/
│   │   └── token.go          # Tipo Token compartido entre lexer y parser
│   │
│   ├── yalex/                # Parser de especificación léxica (.yal)
│   ├── lexer/                # Motor de tokenización (DFAs)
│   ├── dfa/                  # Construcción y minimización de DFAs
│   └── regex/                # Manejo de expresiones regulares
│
└── testdata/
    ├── *.yal                 # Especificaciones léxicas de prueba
    ├── input_grammar*.txt    # Gramáticas de prueba
    └── test.lisp             # Fuente de ejemplo para pruebas
```

---

## 3. Formato de especificación `.yalp`

Un archivo `.yalp` tiene dos secciones separadas por `%%`.

### Sección de encabezado (antes de `%%`)

```
%token ID PLUS TIMES LPAREN RPAREN WS
IGNORE WS
```

- `%token` — declara los nombres de tokens que el lexer puede producir.
- `IGNORE` — lista tokens que el parser debe descartar antes de analizar (ej. espacios en blanco).

### Sección de producciones (después de `%%`)

```
expr : expr PLUS term | term ;
term : term TIMES factor | factor ;
factor : ID | LPAREN expr RPAREN ;
```

- Cada producción termina con `;`.
- Las alternativas se separan con `|`.
- El símbolo inicial es la cabeza de la **primera** producción.
- Los comentarios se escriben como `/* ... */`.

---

## 4. Flujo de datos completo

```
Archivo .yalp
      │
      ▼
 ParseFile()          →  YaparSpec  (tokens declarados, producciones crudas, start)
      │
      ▼
 BuildGrammar()       →  Grammar    (gramática aumentada, terminales, no-terminales)
      │
      ▼
 ComputeFirstFollow() →  FirstFollow (FIRST, FOLLOW, nullable para cada símbolo)
      │
      ▼
 BuildCanonicalCollection() →  []State + transitions  (colección LR(0))
      │
      ▼
 BuildSLRTable()      →  ParsingTable  (tablas ACTION y GOTO)
      │
      ├──────────────────────────────────────────────────────────────┐
      ▼                                                              ▼
 GenerateParserSource()                                  (si se pasan -yal y -src)
 Escribe parser.go standalone                                        │
                                                           CompileYALFile()
                                                           TokenizeFile()
                                                                     │
                                                                     ▼
                                                           ParseTokens()
                                                           → ParseResult {Accepted: bool}
```

---

## 5. Módulo: parser de especificación (`internal/yapar/parser.go`)

Responsable de leer y decodificar un archivo `.yalp` en estructuras Go.

### Tipos principales

```go
type RawProduction struct {
    Head   string     // Nombre del no-terminal (ej. "expr")
    Bodies [][]string // Lista de alternativas; cada alternativa es []string de símbolos
}

type YaparSpec struct {
    Tokens       []string          // Tokens declarados con %token
    IgnoreTokens map[string]bool   // Tokens marcados con IGNORE
    Productions  []RawProduction   // Todas las reglas de producción
    StartSymbol  string            // Cabeza de la primera producción
}
```

### Funciones

| Función | Descripción |
|---------|-------------|
| `ParseFile(path string) (*YaparSpec, error)` | Lee el archivo y llama a `Parse`. |
| `Parse(content string) (*YaparSpec, error)` | Orquesta el parsing del contenido. |

### Proceso interno de `Parse`

1. `stripBlockComments(content)` — elimina bloques `/* ... */`, preservando saltos de línea.
2. `splitSections(clean)` — divide en header / productions por el separador `%%`; error si falta o hay duplicados.
3. `parseHeader(header, spec)` — extrae directivas `%token` e `IGNORE`.
4. `parseProductions(productions, lineOffset, spec)` — usa un escáner interno (`specScanner`) y un parser recursivo (`productionParser`) para construir los `RawProduction`.

La sintaxis esperada por `productionParser`:

```
head : body1 | body2 | ... ;
```

---

## 6. Módulo: gramática formal (`internal/yapar/grammar.go`)

Convierte la especificación cruda en una gramática formal aumentada, lista para el algoritmo SLR.

### Tipos principales

```go
type Symbol struct {
    Name     string // "ID", "expr", etc.
    Terminal bool   // true si es un token
}

type Production struct {
    ID   int      // 0 = producción aumentada; 1+ = producciones del usuario
    Head string   // LHS no-terminal
    Body []Symbol // RHS con símbolos tipados
}

type Grammar struct {
    Start        string              // Símbolo de inicio original (ej. "expr")
    Augmented    string              // Símbolo aumentado (ej. "expr'")
    Terminals    map[string]bool     // Conjunto de todos los tokens
    NonTerminals map[string]bool     // Conjunto de todos los no-terminales
    Productions  []Production        // Producción 0 = aumentada; resto = usuario
    IgnoreSet    map[string]bool     // Tokens a descartar durante el parsing
}
```

### Función `BuildGrammar(spec *YaparSpec) (*Grammar, error)`

1. Valida que la especificación sea no-nula, tenga producciones y símbolo de inicio.
2. Recopila no-terminales de las cabezas de producción.
3. Recopila terminales de la lista `%token`.
4. Valida que todos los tokens de `IGNORE` estén declarados en `%token`.
5. Crea la producción aumentada `Augmented' → Start` con ID = 0.
6. Añade el marcador de fin `$` al conjunto de terminales.
7. Convierte cada `RawProduction.Bodies` en objetos `Production` con símbolos tipados.
8. Asigna IDs estables (1 en adelante) a las producciones del usuario.

### Métodos útiles

| Método | Descripción |
|--------|-------------|
| `Grammar.ProductionsFor(head string)` | Retorna todas las producciones cuya cabeza es `head`. |
| `Grammar.IsTerminal(name string)` | Consulta el mapa `Terminals`. |
| `Grammar.IsNonTerminal(name string)` | Consulta `NonTerminals` o compara con `Augmented`. |

---

## 7. Módulo: FIRST / FOLLOW / nullable (`internal/yapar/first_follow.go`)

Computa los conjuntos necesarios para construir la tabla SLR(1).

### Tipos principales

```go
type Set map[string]bool
// Métodos: Add, Has, Merge, Clone, Sorted

type FirstFollow struct {
    First    map[string]Set  // FIRST(símbolo)
    Follow   map[string]Set  // FOLLOW(no-terminal)
    Nullable map[string]bool // ¿Puede el símbolo derivar ε?
}
```

### Función `ComputeFirstFollow(g *Grammar) (*FirstFollow, error)`

**Inicialización:**
- `FIRST(terminal)` = `{terminal}`
- `FIRST(no-terminal)` = `{}` (se llena iterativamente)
- `FOLLOW(start')` = `{$}`
- `FOLLOW(demás)` = `{}`
- `Nullable(terminal)` = `false`

**Tres etapas de punto fijo:**

1. **`computeNullable`** — un no-terminal es nullable si existe al menos una producción donde todos los símbolos del cuerpo son nullable.
2. **`computeFirst`** — para cada producción `A → X₁ X₂ … Xₙ`: agrega `FIRST(Xᵢ)\{ε}` a `FIRST(A)`; si `Xᵢ` es nullable, continúa con `Xᵢ₊₁`. Si todos son nullable, agrega `ε`.
3. **`computeFollow`** — para cada `[A → α X β]`: agrega `FIRST(β)\{ε}` a `FOLLOW(X)`; si `β` es nullable (o vacío), agrega `FOLLOW(A)` a `FOLLOW(X)`.

### Función auxiliar `FirstOfSequence(seq []Symbol, ff *FirstFollow) Set`

Calcula `FIRST` de una secuencia de símbolos; útil para el cómputo de FOLLOW y para la construcción de la tabla.

---

## 8. Módulo: colección canónica LR(0) (`internal/yapar/items.go`)

Construye los estados del autómata LR(0) y las transiciones entre ellos.

### Tipos principales

```go
type Item struct {
    ProductionID int // ID de la producción a la que pertenece el ítem
    Dot          int // Posición del punto (0 = inicio, len(Body) = final)
}
// Ejemplo: A → α • X β  se representa como {ProductionID: k, Dot: i}

type State struct {
    ID    int    // Número de estado (0 = estado inicial)
    Items []Item // Conjunto de ítems LR(0) que forman el estado
}
```

### Funciones

#### `Closure(g *Grammar, items []Item) []Item`

Algoritmo de clausura:
1. Encola los ítems iniciales.
2. Para cada ítem `[A → α • B γ]` donde `B` es no-terminal:
   - Agrega todos los ítems `[B → • β]` para cada producción `B → β`.
3. Ordena por `(ProductionID, Dot)` y retorna.

#### `Goto(g *Grammar, items []Item, symbol string) []Item`

1. Para cada ítem `[A → α • X γ]` donde `X = symbol`: crea `[A → α X • γ]`.
2. Retorna `Closure(ítems_avanzados)`.
3. Retorna `nil` si ningún ítem coincide con `symbol`.

#### `BuildCanonicalCollection(g *Grammar) ([]State, map[int]map[string]int, error)`

Algoritmo BFS:
1. Estado inicial (ID=0): `Closure({[0, 0]})` — producción aumentada `Augmented' → • Start`.
2. Para cada estado en la cola:
   - Calcula todos los símbolos que aparecen después de un punto.
   - Para cada símbolo: `nextItems = Goto(state.Items, symbol)`.
   - Busca o crea el estado correspondiente (usando `stateSignature` para deduplicar).
   - Registra la transición `state × symbol → nextState`.
3. Retorna `[]State` y `transitions[stateID][symbol] = targetStateID`.

---

## 9. Módulo: tabla SLR(1) (`internal/yapar/table.go`)

Construye las tablas ACTION y GOTO a partir de la colección canónica.

### Tipos principales

```go
type ActionKind int
const (
    ActionError   ActionKind = iota // Sin acción (error de sintaxis)
    ActionShift                     // Desplazar: apilar estado
    ActionReduce                    // Reducir: aplicar producción
    ActionAccept                    // Aceptar: análisis exitoso
)

type Action struct {
    Kind         ActionKind
    TargetState  int // Solo para Shift: estado destino
    ProductionID int // Solo para Reduce: producción a aplicar
}

type ParsingTable struct {
    Action map[int]map[string]Action // [estado][terminal] → acción
    Goto   map[int]map[string]int    // [estado][no-terminal] → estado
}
```

### Función `BuildSLRTable(...) (*ParsingTable, error)`

**Reglas de construcción:**

| Condición | Acción |
|-----------|--------|
| Transición `(estado, terminal)` → `nextState` | `ACTION[estado][terminal] = Shift(nextState)` |
| Transición `(estado, no-terminal)` → `nextState` | `GOTO[estado][no-terminal] = nextState` |
| Ítem `[A → γ •]` en `estado`, `a ∈ FOLLOW(A)` | `ACTION[estado][a] = Reduce(producción)` |
| Ítem `[Augmented' → Start •]` en `estado` | `ACTION[estado][$] = Accept` |

**Detección de conflictos:** si se intenta asignar dos acciones distintas a la misma celda `(estado, símbolo)`, se retorna un `GrammarConflictError` indicando el tipo de conflicto (shift/reduce o reduce/reduce).

### Método `ParsingTable.ExpectedTokens(state int) []string`

Lista todos los terminales con acciones no-error en un estado dado; se usa para generar mensajes de error descriptivos.

---

## 10. Módulo: simulador del parser (`internal/yapar/simulator.go`)

Ejecuta el algoritmo LR clásico sobre una cadena de tokens.

### Tipos

```go
type ParseResult struct {
    Accepted bool // true = cadena aceptada por la gramática
}
```

### Función `ParseTokens(g *Grammar, table *ParsingTable, tokens []shared.Token) (*ParseResult, error)`

**Preprocesamiento:**
1. `FilterIgnoredTokens(tokens, g.IgnoreSet)` — elimina tokens cuyo tipo está en `IgnoreSet`.
2. Agrega el token de fin `$` al final del stream.

**Algoritmo de simulación:**

```
pilaEstados = [0]
índice = 0

bucle:
  estado = tope(pilaEstados)
  lookahead = tokens[índice]
  acción = ACTION[estado][lookahead.Type]

  caso Shift(nextState):
    push(pilaEstados, nextState)
    índice++

  caso Reduce(prodID):
    prod = productions[prodID]
    pop(pilaEstados, len(prod.Body) veces)
    nuevoEstado = GOTO[tope(pilaEstados)][prod.Head]
    push(pilaEstados, nuevoEstado)

  caso Accept:
    return {Accepted: true}

  caso Error:
    return SyntaxError con tokens esperados
```

### Función `FilterIgnoredTokens(tokens []shared.Token, ignoreSet map[string]bool) []shared.Token`

Filtra el slice de tokens eliminando aquellos cuyo `Type` aparece en `ignoreSet`.

---

## 11. Módulo: generador de código (`internal/generator/parser_gen.go`)

Genera un archivo `.go` con el parser embebido, listo para compilar sin dependencias externas.

### Función `GenerateParserSource(outputPath string, g *Grammar, table *ParsingTable) error`

1. Parsea una plantilla Go embebida en el binario.
2. Construye los datos de la plantilla a partir de `Grammar` y `ParsingTable`:
   - Lista de producciones con sus cuerpos.
   - Filas de la tabla ACTION (por estado).
   - Filas de la tabla GOTO (por estado).
3. Ejecuta la plantilla sobre un buffer.
4. Escribe el resultado en `outputPath`.

### Estructuras de datos de la plantilla

```go
type parserProductionData struct {
    ID   int
    Head string
    Body []parserSymbolData // {Name string, Terminal bool}
}

type parserActionRowData struct {
    State int
    Cells []parserActionCellData // {Symbol, Kind, TargetState, ProductionID}
}

type parserGotoRowData struct {
    State int
    Cells []parserGotoCellData // {Symbol, Target}
}
```

### Parser generado — capacidades

El archivo `.go` generado incluye:

| Elemento | Descripción |
|----------|-------------|
| `productions` | Slice con todas las producciones embebidas. |
| `actionTable` | Mapa `state → symbol → Action` embebido. |
| `gotoTable` | Mapa `state → symbol → state` embebido. |
| `parseTokens([]Token)` | Función de simulación LR standalone. |
| `loadTokens(path)` | Lee un archivo JSON con array de tokens. |
| `filterIgnoredTokens()` | Elimina tokens ignorados. |
| `main()` | CLI: acepta `-tokens <json_file>`. |

---

## 12. Módulo: integración con el lexer (`internal/lexbuild/pipeline.go`)

Coordina la compilación del lexer (YALex) y la tokenización de un archivo fuente.

### Tipo `Result`

```go
type Result struct {
    DFAEntries  []lexer.DFAEntry  // DFAs compilados, uno por regla léxica
    DOTContents []string          // Representaciones GraphViz DOT (opcional)
    Rules       []yalex.TokenRule // Reglas originales expandidas
    Macros      map[string]string // Definiciones de macros
}
```

### Funciones

#### `CompileYALFile(path string, captureDOT bool) (*Result, error)`

1. `yalex.ParseFile(path)` — parsea el archivo `.yal`.
2. Llama a `CompileParseResult(parseResult, captureDOT)`.

#### `CompileParseResult(parseResult *yalex.ParseResult, captureDOT bool) (*Result, error)`

Para cada regla léxica:
1. `yalex.Expand(macros, rules)` — expande macros en los patrones.
2. `regex.Normalize(pattern)` — normaliza la expresión regular.
3. `regex.BuildPostfix(normalized)` — algoritmo shunting-yard.
4. `dfa.BuildTree(postfix)` — árbol de sintaxis para la regex.
5. `dfa.BuildDFA(tree, ...)` — construcción NFA → DFA.
6. `dfa.Minimize(dfa)` — minimización por el algoritmo de Hopcroft.

#### `TokenizeFile(entries []lexer.DFAEntry, path string) ([]shared.Token, []string, error)`

1. `lexer.ReadSource(path)` — lee el archivo fuente.
2. `lexer.Tokenize(entries, src)` — ejecuta los DFAs sobre el código fuente.
3. Retorna `[]shared.Token` y mensajes de error léxico (si los hay).

---

## 13. Punto de entrada CLI (`cmd/yapar/main.go`)

### Flags disponibles

| Flag | Tipo | Requerido | Descripción |
|------|------|-----------|-------------|
| `-yalp FILE` | string | Sí | Archivo de especificación del parser (.yalp) |
| `-yal FILE` | string | No* | Archivo de especificación del lexer (.yal) |
| `-src FILE` | string | No* | Archivo fuente a parsear |
| `-out FILE` | string | No | Ruta del archivo `.go` generado |
| `-table` | bool | No | Imprime la tabla SLR(1) en stdout |

*`-yal` y `-src` deben aparecer juntos.

### Función `run(args []string, stdout, stderr io.Writer) error`

Orquesta el pipeline completo:

```
1. parseFlags()
2. yapar.ParseFile()          → YaparSpec
3. yapar.BuildGrammar()       → Grammar
4. yapar.ComputeFirstFollow() → FirstFollow
5. yapar.BuildCanonicalCollection() → States + transitions
6. yapar.BuildSLRTable()      → ParsingTable
7. (si -table)  formatParsingTable() → imprime en stdout
8. (si -out)    generator.GenerateParserSource()
9. (si -yal/-src)
   a. lexbuild.CompileYALFile()
   b. lexbuild.TokenizeFile()
   c. yapar.ParseTokens()     → ParseResult
```

### Función `formatParsingTable(grammar, table)`

Imprime la tabla ACTION/GOTO en formato legible para depuración, con terminales y no-terminales ordenados y alineados por columnas.

---

## 14. Manejo de errores

### `SpecError` (`internal/yapar/errors.go`)

Se produce durante el parsing del `.yalp` o la construcción de la gramática.

```go
type SpecError struct {
    Line    int    // 0 si no aplica
    Message string
}
// Error() → "yapar spec: <msg>" o "yapar spec line N: <msg>"
```

Ejemplos de causas: falta de `%%`, símbolo no declarado, token duplicado, producción sin cabeza.

### `GrammarConflictError`

Se produce en `BuildSLRTable` cuando la gramática no es SLR(1).

```go
type GrammarConflictError struct {
    State     int
    Symbol    string
    Kind      string // "shift/reduce", "reduce/reduce", etc.
    Current   Action
    New       Action
}
```

### `SyntaxError`

Se produce en `ParseTokens` cuando el parser llega a una celda de error.

```go
type SyntaxError struct {
    Line     int
    GotType  string
    Lexeme   string
    Expected []string // tokens esperados según ExpectedTokens()
}
// Error() → "syntax error at line N: got TOKEN 'lexeme', expected [a, b, ...]"
```

---

## 15. Pruebas

### Tests de integración (`cmd/yapar/main_test.go`)

Prueban el pipeline completo invocando `run()` directamente con flags simulados:
- Sin archivo fuente (solo construcción de tabla).
- Con `-yal` y `-src` (tokenización + parsing).
- Con `-out` (generación de código).

### Tests unitarios (`internal/yapar/*_test.go`)

| Archivo de test | Qué prueba |
|-----------------|------------|
| `parser_test.go` | Parsing de `.yalp`, casos de error, comentarios, producciones vacías |
| `grammar_test.go` | Aumentación, validación de tokens ignorados, asignación de IDs |
| `first_follow_test.go` | Nullable, FIRST, FOLLOW sobre gramáticas conocidas |
| `items_test.go` | Closure, Goto, construcción de la colección canónica |
| `table_test.go` | Acciones shift/reduce/accept, detección de conflictos |
| `simulator_test.go` | Cadenas aceptadas y rechazadas, filtrado de tokens ignorados |

### Datos de prueba (`testdata/`)

| Archivo | Propósito |
|---------|-----------|
| `slr-1.yal` … `slr-4.yal` | Especificaciones léxicas para distintos casos SLR |
| `input_grammar1.txt` … `input_grammar4.txt` | Gramáticas de distinta complejidad |
| `test.lisp` | Archivo fuente de ejemplo para pruebas end-to-end |

---

## 16. Constantes y símbolos reservados

| Constante | Valor | Uso |
|-----------|-------|-----|
| `EndMarker` | `"$"` | Marcador de fin de entrada en la gramática y tablas |
| `Epsilon` | `"ε"` | Representa derivación vacía (solo semántico, no en tablas) |
| `Augmented` | `Start + "'"` | Cabeza de la producción aumentada (ej. `"expr'"`) |

**Regla de IDs de producción:**
- ID `0` — siempre la producción aumentada `Augmented' → Start`.
- IDs `1+` — producciones del usuario en orden de aparición en el `.yalp`.

**Tipo `shared.Token`:**

```go
type Token struct {
    Type   string // Nombre del token (ej. "INT", "PLUS")
    Lexeme string // Texto exacto del fuente (ej. "42", "+")
    Line   int    // Número de línea en el fuente
}
```

Este tipo es el contrato entre el módulo léxico (YALex) y el sintáctico (YAPar).
