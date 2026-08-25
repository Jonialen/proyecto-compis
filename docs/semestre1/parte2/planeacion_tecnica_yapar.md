# Planeación técnica — Parte 2 (YAPar)

Esta planeación define una implementación **profesional, modular y mantenible** para la Parte 2 del proyecto: un generador de analizadores sintácticos en Go que se integra con la Parte 1 sin romper su arquitectura actual.

## Decisión arquitectónica

La Parte 2 se construirá como un subsistema separado sobre el lexer existente.

### Principios obligatorios

1. **Parser `.yalp` separado**
2. **Modelo de gramática separado**
3. **Algoritmos LR separados**
4. **Simulación separada**
5. **Generación separada**

> No se debe crear un archivo monolítico `yapar.go`. Esa estructura aumentaría el acoplamiento, dificultaría las pruebas y volvería frágil la evolución del proyecto.

---

## Objetivo de la Parte 2

Construir un pipeline sintáctico que tome una especificación `.yalp`, construya una gramática formal, derive una tabla LR y permita:

- validar secuencias de tokens producidas por la Parte 1,
- simular análisis sintáctico directamente,
- y generar un parser standalone en Go.

---

## Integración con la Parte 1

La Parte 1 ya resolvió el problema léxico. La Parte 2 debe reutilizar ese resultado a través del contrato de tokens, no a través de detalles internos de regex o DFA.

### Contrato de integración

```go
type Token struct {
    Type   string
    Lexeme string
    Line   int
}
```

El parser consumirá `[]Token` como entrada.

### Regla de diseño

- El parser **sí** depende del stream de tokens.
- El parser **no** debe depender del detalle interno de `regex`, `dfa` o `followpos` léxico.

---

## Estructura recomendada

```text
cmd/
  yalex/
    main.go
  yapar/
    main.go

internal/
  shared/
    token.go
  yapar/
    parser.go
    grammar.go
    first_follow.go
    items.go
    table.go
    simulator.go
    errors.go
  generator/
    lexer_gen.go
    parser_gen.go
```

---

## Responsabilidad por archivo

## 1. `cmd/yalex/main.go`

### Responsabilidad
Mantener la CLI actual del lexer.

### Alcance
- mover aquí el `main.go` actual,
- ajustar imports,
- dejar intacto el comportamiento actual.

### Restricción
No agregar lógica de parser aquí.

---

## 2. `cmd/yapar/main.go`

### Responsabilidad
Ser el nuevo punto de entrada de YAPar.

### Flujo principal
1. leer `.yalp`,
2. parsear especificación,
3. construir gramática,
4. calcular FIRST/FOLLOW,
5. construir colección LR,
6. construir tabla ACTION/GOTO,
7. simular parsing si hay `-src`,
8. generar parser standalone si hay `-out`.

### Flags sugeridos

```text
-yalp   ruta del archivo parser.yalp
-yal    ruta del archivo lexer.yal
-src    archivo fuente a analizar
-out    archivo de salida del parser generado
-table  imprimir tabla LR
```

---

## 3. `internal/shared/token.go`

### Responsabilidad
Definir el contrato compartido entre lexer y parser.

### Motivo
Evita acoplar el parser a la implementación interna del paquete `lexer`.

---

## 4. `internal/yapar/parser.go`

### Responsabilidad
Parsear el archivo `.yalp`.

### Debe resolver
- comentarios `/* ... */`,
- sección `%token`,
- sección `IGNORE`,
- separación con `%%`,
- producciones con `:`, `|`, `;`,
- detección de errores de formato.

### Salida sugerida

```go
type YaparSpec struct {
    Tokens       []string
    IgnoreTokens map[string]bool
    Productions  []RawProduction
    StartSymbol  string
}
```

### Observación
Este archivo solo interpreta la especificación. No debe calcular FIRST/FOLLOW ni construir tablas.

---

## 5. `internal/yapar/grammar.go`

### Responsabilidad
Convertir la especificación cruda en una gramática formal.

### Debe resolver
- terminales,
- no terminales,
- producciones enumeradas,
- símbolo inicial,
- gramática aumentada.

### Estructuras sugeridas

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

### Observación
Aquí se define el modelo formal. No se ejecuta parsing LR todavía.

---

## 6. `internal/yapar/first_follow.go`

### Responsabilidad
Calcular `FIRST`, `FOLLOW` y `nullable`.

### Debe implementar
- iteración a punto fijo,
- manejo correcto de epsilon,
- `FIRST` de secuencias,
- `FOLLOW` del símbolo inicial con `$`.

### Estructuras sugeridas

```go
type Set map[string]bool

type FirstFollow struct {
    First    map[string]Set
    Follow   map[string]Set
    Nullable map[string]bool
}
```

### Observación crítica
`FOLLOW` de gramáticas y `followpos` del lexer NO son la misma cosa.

---

## 7. `internal/yapar/items.go`

### Responsabilidad
Construir items LR y la colección canónica de estados.

### Enfoque recomendado
Comenzar con **items LR(0)** y construir una tabla **SLR(1)**.

### Estructuras sugeridas

```go
type Item struct {
    ProductionID int
    Dot          int
}

type State struct {
    ID    int
    Items []Item
}
```

### Debe implementar
- `closure`,
- `goto`,
- deduplicación de estados,
- colección canónica.

---

## 8. `internal/yapar/table.go`

### Responsabilidad
Construir la tabla `ACTION/GOTO`.

### Debe resolver
- acciones `shift`,
- acciones `reduce`,
- `accept`,
- transiciones `goto`,
- detección de conflictos.

### Estructuras sugeridas

```go
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

### Regla importante
Si aparece un conflicto `shift/reduce` o `reduce/reduce`, debe reportarse explícitamente.

---

## 9. `internal/yapar/simulator.go`

### Responsabilidad
Ejecutar el algoritmo LR sobre el stream de tokens.

### Debe implementar
- stack de estados,
- consulta de `ACTION`,
- operaciones `shift`, `reduce`, `accept`,
- filtrado de tokens `IGNORE`,
- error sintáctico con línea y expectativas.

### Meta mínima
Aceptar o rechazar correctamente con un mensaje útil.

---

## 10. `internal/yapar/errors.go`

### Responsabilidad
Modelar errores del dominio sintáctico.

### Tipos sugeridos
- error de conflicto de gramática,
- error sintáctico en ejecución,
- error de especificación `.yalp`.

---

## 11. `internal/generator/parser_gen.go`

### Responsabilidad
Generar un parser standalone en Go a partir de la gramática y su tabla LR.

### Debe serializar
- producciones,
- tabla `ACTION`,
- tabla `GOTO`,
- símbolo inicial,
- conjunto `IGNORE`.

### Restricción
No debe implementarse antes de tener el simulador funcionando correctamente en memoria.

---

## Dependencias entre módulos

```text
parser.go
   ↓
grammar.go
   ↓
first_follow.go
   ↓
items.go
   ↓
table.go
   ↓
simulator.go
   ↓
parser_gen.go
```

---

## Orden recomendado de implementación

### Etapa 0 — Base compartida
- `cmd/yalex/main.go`
- `cmd/yapar/main.go`
- `internal/shared/token.go`

### Etapa 1 — Parser de especificación
- `internal/yapar/parser.go`
- tests de `.yalp`

### Etapa 2 — Modelo formal
- `internal/yapar/grammar.go`
- `internal/yapar/first_follow.go`

### Etapa 3 — Núcleo LR
- `internal/yapar/items.go`
- `internal/yapar/table.go`

### Etapa 4 — Ejecución
- `internal/yapar/errors.go`
- `internal/yapar/simulator.go`

### Etapa 5 — Generación
- `internal/generator/parser_gen.go`

---

## Hitos técnicos

### Hito 1
El parser `.yalp` produce una estructura válida de especificación.

### Hito 2
Los conjuntos FIRST/FOLLOW funcionan para gramáticas simples.

### Hito 3
La colección de items LR se construye correctamente.

### Hito 4
La tabla SLR(1) se genera sin conflictos para la gramática base de pruebas.

### Hito 5
El parser acepta y rechaza entradas correctamente.

### Hito 6
Se genera un parser standalone.

---

## Riesgos técnicos

| Riesgo | Impacto | Mitigación |
|---|---|---|
| Conflictos SLR | Alto | Diagnóstico explícito y posible escalamiento a LALR |
| Reestructuración de imports | Medio | Hacerla primero y validar con tests |
| Error conceptual entre FOLLOW y followpos | Alto | Separar completamente los módulos |
| Parser monolítico | Alto | Mantener separación estricta por responsabilidades |
| Reducciones mal calculadas | Alto | Probar reduce/stack con casos pequeños controlados |

---

## Estrategia de pruebas

### Unitarias
- parser de `.yalp`,
- clasificación de símbolos,
- FIRST/FOLLOW,
- `closure`/`goto`,
- tabla LR,
- simulación de `shift/reduce/accept/error`.

### Integración
- `.yal` + `.yalp` + archivo fuente,
- filtrado de `IGNORE`,
- aceptación de gramática válida,
- error sintáctico con línea correcta.

### Gramática inicial recomendada
Expresiones aritméticas:

```text
E -> E + T | T
T -> T * F | F
F -> ( E ) | id
```

---

## Conclusión

La Parte 2 debe construirse como una extensión sintáctica limpia del proyecto actual, no como una mezcla improvisada dentro del lexer. La separación en parser de especificación, modelo formal, algoritmos LR, simulación y generación permite:

- menor acoplamiento,
- mejor testabilidad,
- diagnósticos más claros,
- y evolución futura hacia LALR o LR(1) si el proyecto lo exige.
