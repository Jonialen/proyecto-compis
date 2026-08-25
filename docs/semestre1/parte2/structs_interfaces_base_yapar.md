# Structs e interfaces base — Parte 2 (YAPar)

Este documento define los contratos base recomendados para implementar YAPar en Go sin acoplar el parser al detalle interno del lexer.

---

## Objetivo de diseño

Los contratos deben cumplir tres condiciones:

1. ser **pequeños y explícitos**,
2. separar **especificación, modelo formal, algoritmos y ejecución**,
3. permitir pruebas unitarias sin depender del pipeline completo.

---

## 1. Contrato compartido de token

**Archivo recomendado:** `internal/shared/token.go`

```go
package shared

type Token struct {
	Type   string
	Lexeme string
	Line   int
}
```

### Razón

El parser solo necesita saber:

- qué token llegó,
- qué lexema produjo,
- y en qué línea ocurrió.

No necesita conocer DFAs, regex ni detalles del simulador léxico.

---

## 2. Parser del archivo `.yalp`

**Archivo recomendado:** `internal/yapar/parser.go`

### Tipos base

```go
package yapar

type RawProduction struct {
	Head  string
	Bodies [][]string
}

type YaparSpec struct {
	Tokens       []string
	IgnoreTokens map[string]bool
	Productions  []RawProduction
	StartSymbol  string
}
```

### Funciones base

```go
func ParseFile(path string) (*YaparSpec, error)
func Parse(content string) (*YaparSpec, error)
```

### Responsabilidad

Solo interpreta el `.yalp`.

### Restricción

No debe construir gramática formal ni calcular FIRST/FOLLOW.

---

## 3. Modelo formal de gramática

**Archivo recomendado:** `internal/yapar/grammar.go`

### Tipos base

```go
package yapar

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

### Funciones base

```go
func BuildGrammar(spec *YaparSpec) (*Grammar, error)
func (g *Grammar) ProductionsFor(head string) []Production
func (g *Grammar) IsTerminal(name string) bool
func (g *Grammar) IsNonTerminal(name string) bool
```

### Responsabilidad

Convertir la especificación cruda en una gramática consistente y aumentada.

---

## 4. Sets y resultados de FIRST/FOLLOW

**Archivo recomendado:** `internal/yapar/first_follow.go`

### Tipos base

```go
package yapar

type Set map[string]bool

type FirstFollow struct {
	First    map[string]Set
	Follow   map[string]Set
	Nullable map[string]bool
}
```

### Funciones base

```go
func ComputeFirstFollow(g *Grammar) (*FirstFollow, error)
func FirstOfSequence(seq []Symbol, ff *FirstFollow) Set
```

### Helpers recomendados

```go
func NewSet(items ...string) Set
func (s Set) Add(v string) bool
func (s Set) Has(v string) bool
func (s Set) Merge(other Set) bool
func (s Set) Clone() Set
func (s Set) Sorted() []string
```

### Razón

Los sets son parte central del algoritmo. Si esa base es mala, TODO lo demás se contamina.

---

## 5. Items LR y estados

**Archivo recomendado:** `internal/yapar/items.go`

### Tipos base

```go
package yapar

type Item struct {
	ProductionID int
	Dot          int
}

type State struct {
	ID    int
	Items []Item
}
```

### Funciones base

```go
func Closure(g *Grammar, items []Item) []Item
func Goto(g *Grammar, items []Item, symbol string) []Item
func BuildCanonicalCollection(g *Grammar) ([]State, map[int]map[string]int, error)
```

### Nota de diseño

La función `BuildCanonicalCollection` puede devolver también el mapa de transiciones entre estados para no recalcularlo en la tabla.

---

## 6. Tabla ACTION/GOTO

**Archivo recomendado:** `internal/yapar/table.go`

### Tipos base

```go
package yapar

type ActionKind int

const (
	ActionError ActionKind = iota
	ActionShift
	ActionReduce
	ActionAccept
)

type Action struct {
	Kind         ActionKind
	TargetState  int
	ProductionID int
}

type ParsingTable struct {
	Action map[int]map[string]Action
	Goto   map[int]map[string]int
}
```

### Funciones base

```go
func BuildSLRTable(g *Grammar, ff *FirstFollow, states []State, transitions map[int]map[string]int) (*ParsingTable, error)
func (t *ParsingTable) ExpectedTokens(state int) []string
```

### Responsabilidad

Construir la tabla y detectar conflictos de gramática.

---

## 7. Errores del dominio

**Archivo recomendado:** `internal/yapar/errors.go`

### Tipos base

```go
package yapar

type SpecError struct {
	Line    int
	Message string
}

type GrammarConflictError struct {
	State  int
	Symbol string
	Kind   string
	Current Action
	New     Action
}

type SyntaxError struct {
	Line     int
	GotType  string
	Lexeme   string
	Expected []string
}
```

### Razón

Separar errores por nivel permite saber si falló:

- la especificación,
- la gramática,
- o la ejecución del parser.

---

## 8. Simulador LR

**Archivo recomendado:** `internal/yapar/simulator.go`

### Tipo resultado recomendado

```go
package yapar

type ParseResult struct {
	Accepted bool
}
```

### Funciones base

```go
func ParseTokens(g *Grammar, table *ParsingTable, tokens []shared.Token) (*ParseResult, error)
func FilterIgnoredTokens(tokens []shared.Token, ignoreSet map[string]bool) []shared.Token
```

### Responsabilidad

Ejecutar el algoritmo LR con stack de estados y reportar aceptación o error.

### Restricción

No debe saber cómo se construyó el lexer. Solo consume tokens.

---

## 9. Generación de parser standalone

**Archivo recomendado:** `internal/generator/parser_gen.go`

### Función base

```go
func GenerateParserSource(path string, g *yapar.Grammar, table *yapar.ParsingTable) error
```

### Responsabilidad

Serializar gramática, tabla y runtime mínimo del parser en un archivo Go autónomo.

### Restricción

No debe contener lógica de construcción de FIRST/FOLLOW ni del autómata. Solo emitir código a partir de estructuras ya resueltas.

---

## 10. Orquestación de CLI

**Archivo recomendado:** `cmd/yapar/main.go`

### Función base

```go
func main()
func run() error
```

### Flujo esperado

1. parsear flags,
2. leer `.yalp`,
3. construir gramática,
4. calcular FIRST/FOLLOW,
5. construir estados LR,
6. construir tabla,
7. ejecutar simulación o generar parser.

---

## 11. Interfaces opcionales recomendadas

Estas interfaces NO son obligatorias, pero ayudan a testear mejor.

### `TokenSource`

```go
type TokenSource interface {
	Tokens() ([]shared.Token, error)
}
```

Útil si luego quieres soportar:

- tokens desde archivo,
- tokens desde lexer en memoria,
- tokens sintéticos en tests.

### `SpecLoader`

```go
type SpecLoader interface {
	Load(path string) (*YaparSpec, error)
}
```

Solo vale la pena si el proyecto crece. Para el laboratorio, una función directa suele bastar.

---

## 12. Ajustes derivados del pull

Después del `pull`, hay dos observaciones relevantes para estos contratos:

1. La documentación ya vive bajo `docs/`, así que la estructura de planeación actual queda alineada con el repositorio.
2. Existen archivos en `testdata/first_test/` que ya sirven como base para construir fixtures de integración de YAPar. Conviene reutilizarlos en la fase de pruebas del simulador y del pipeline completo.

Esto **no cambia** la arquitectura modular propuesta. Solo mejora el punto de partida.

---

## Conclusión

Los contratos base de YAPar deben ser pequeños, explícitos y desacoplados. La clave no es escribir mucho código rápido; la clave es evitar contaminar el parser con responsabilidades que no le pertenecen.

La separación correcta sigue siendo:

1. parser `.yalp`,
2. gramática formal,
3. FIRST/FOLLOW,
4. items LR,
5. tabla,
6. simulación,
7. generación.
