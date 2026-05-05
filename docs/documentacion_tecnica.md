# Documentacion Tecnica - Generador de Analizadores Lexicos

**Universidad del Valle de Guatemala**
**Facultad de Ingenieria - Departamento de Ciencias de la Computacion**
**Autores:**
- Luis Padilla
- Jonathan Díaz

**Repositorio:** [https://github.com/Jonialen/proyecto-compis](https://github.com/Jonialen/proyecto-compis)

**Video de presentacion:** Disponible en el repositorio de GitHub.

---

## 1. Introduccion

### 1.1 Descripcion del Proyecto

**genAnaLex** es un generador de analizadores lexicos implementado en Go. Recibe como entrada un archivo de especificacion escrito en el formato YALex (Yet Another Lex) y produce dos resultados:

1. **Modo Simulador:** Tokeniza directamente un archivo fuente utilizando los DFAs construidos en memoria.
2. **Modo Generador:** Genera un archivo fuente Go autonomo (`standalone`) que implementa el analizador lexico, compilable y ejecutable de forma independiente.

El sistema implementa el pipeline completo desde la lectura de la especificacion hasta la generacion de codigo, pasando por la construccion directa de DFAs mediante el metodo de Aho-Sethi-Ullman.

### 1.2 Objetivo

El problema que resuelve genAnaLex es la **generacion automatica de analizadores lexicos** a partir de una definicion formal de tokens. Dado un conjunto de expresiones regulares que describen los patrones lexicos de un lenguaje, el sistema produce un analizador capaz de:

- Reconocer tokens validos en un archivo fuente.
- Aplicar la estrategia **Maximal Munch** (avidez maxima) para desambiguacion.
- Resolver empates de longitud por **prioridad de reglas**.
- Reportar errores lexicos con informacion de linea.
- Recuperarse de errores y continuar el analisis.

### 1.3 Contexto Teorico

El proyecto se fundamenta en los siguientes pilares de la teoria de compiladores:

- **Expresiones regulares** como mecanismo de especificacion de patrones lexicos.
- **Automatas finitos deterministas (DFA)** como modelo computacional para el reconocimiento de tokens.
- **Construccion directa de DFA** (metodo Aho-Sethi-Ullman, Dragon Book seccion 3.9.5) como alternativa eficiente al camino Thompson NFA -> Subset Construction -> DFA.
- **Minimizacion de DFA** mediante el metodo Table-Filling (piramide) para optimizar el automata resultante.
- **YALex** como formato de especificacion lexica basado en OCamllex/Lex.

### 1.4 Alcance

**Incluye:**
- Parser completo de archivos `.yal` con macros, reglas, comentarios, header y trailer.
- Expansion recursiva de macros con deteccion de ciclos.
- Normalizacion de expresiones regulares: clases de caracteres, wildcards, escapes, complementos.
- Conversion a postfix via Shunting-Yard.
- Construccion de arbol sintactico aumentado.
- Calculo de nullable, firstpos, lastpos, followpos con memoizacion.
- Construccion directa de DFA.
- Minimizacion de DFA con Table-Filling y Union-Find.
- Simulacion de multiples DFAs en paralelo con Maximal Munch.
- Generacion de codigo Go standalone.
- Visualizacion de arboles sintacticos en formato Graphviz DOT.
- Suite de 139 tests automatizados.

**No incluye:**
- Operador de diferencia de conjuntos (`#`).
- Soporte para Unicode extendido (solo ASCII 32-126 + `\t` + `\r`).
- Generacion de codigo en otros lenguajes (solo Go).
- Integracion con un parser (YAPar).

---

## 2. Fundamentos Teoricos

### 2.1 Expresiones Regulares y su Papel en el Analisis Lexico

Las expresiones regulares son el mecanismo formal para describir los patrones que definen cada tipo de token en un lenguaje de programacion. Un analizador lexico toma una cadena de entrada y la descompone en una secuencia de tokens, donde cada token corresponde a un match con alguna expresion regular.

En YALex, las expresiones regulares soportan:
- **Literales:** `'a'`, `"abc"` - caracteres concretos.
- **Clases de caracteres:** `[a-z]`, `[0-9]` - rangos de caracteres.
- **Complemento:** `[^...]` - todos los caracteres excepto los listados.
- **Wildcards:** `.` y `_` - cualquier caracter del alfabeto.
- **Operadores:** `|` (alternancia), `*` (cerradura de Kleene), `+` (cerradura positiva), `?` (opcional).
- **Concatenacion:** implicita entre elementos adyacentes.
- **Agrupacion:** `(...)` para controlar precedencia.

### 2.2 Automatas Finitos Deterministas (DFA)

Un DFA es una quintupla `(Q, Sigma, delta, q0, F)` donde:
- `Q` es el conjunto finito de estados.
- `Sigma` es el alfabeto de entrada.
- `delta: Q x Sigma -> Q` es la funcion de transicion.
- `q0` es el estado inicial.
- `F` es el conjunto de estados aceptores.

Los DFA son el modelo computacional ideal para analisis lexico porque:
1. Garantizan reconocimiento en tiempo `O(n)` donde `n` es la longitud de la entrada.
2. No tienen ambiguedad en las transiciones (determinismo).
3. Son eficientes en espacio al minimizarlos.

### 2.3 Construccion Directa de DFA (Metodo Aho-Sethi-Ullman)

En lugar de seguir el camino tradicional `regex -> NFA (Thompson) -> DFA (Subset Construction)`, genAnaLex usa la **construccion directa**, que va de la expresion regular directamente al DFA mediante el arbol sintactico aumentado.

#### 2.3.1 Arbol Sintactico Aumentado

Dada una expresion regular `r`, se construye la expresion aumentada `(r)#` donde `#` es un marcador de fin especial. Esta expresion se convierte a un arbol sintactico donde:
- Las hojas representan simbolos del alfabeto, cada una con una posicion unica (1-indexed).
- Los nodos internos representan operaciones: concatenacion (`cat`), alternancia (`or`), cerradura (`star`), cerradura positiva (`plus`), opcional (`opt`).

#### 2.3.2 Funciones nullable, firstpos, lastpos, followpos

Estas cuatro funciones se calculan sobre el arbol sintactico:

**nullable(n)** - Determina si el nodo puede generar la cadena vacia:

| Nodo | Regla |
|------|-------|
| `leaf(epsilon)` | `true` |
| `leaf(c)` | `false` |
| `n1 \| n2` | `nullable(n1) OR nullable(n2)` |
| `n1 . n2` | `nullable(n1) AND nullable(n2)` |
| `n*` | `true` |
| `n+` | `nullable(n)` |
| `n?` | `true` |

> **Nota sobre `nullable(n+)`:** La regla correcta segun el Dragon Book es `nullable(n+) = nullable(n)`, no `false` incondicionalmente. La derivacion es: `n+ = n . n*`, entonces `nullable(n+) = nullable(n . n*) = nullable(n) AND nullable(n*) = nullable(n) AND true = nullable(n)`. Esto es relevante para patrones como `(a?)+` donde la expresion interna SI es nullable.

**firstpos(n)** - Conjunto de posiciones que pueden ser la primera posicion de alguna cadena generada:

| Nodo | Regla |
|------|-------|
| `leaf(epsilon)` | `{}` |
| `leaf(c, i)` | `{i}` |
| `n1 \| n2` | `firstpos(n1) U firstpos(n2)` |
| `n1 . n2` | Si `nullable(n1)`: `firstpos(n1) U firstpos(n2)`, sino: `firstpos(n1)` |
| `n*`, `n+`, `n?` | `firstpos(n)` |

**lastpos(n)** - Simetrico a firstpos, pero sobre la ultima posicion:

| Nodo | Regla |
|------|-------|
| `leaf(epsilon)` | `{}` |
| `leaf(c, i)` | `{i}` |
| `n1 \| n2` | `lastpos(n1) U lastpos(n2)` |
| `n1 . n2` | Si `nullable(n2)`: `lastpos(n1) U lastpos(n2)`, sino: `lastpos(n2)` |
| `n*`, `n+`, `n?` | `lastpos(n)` |

**followpos(i)** - Para cada posicion `i`, las posiciones que pueden seguirla:
- Para cada nodo `cat(n1, n2)`: Para todo `i` en `lastpos(n1)`, `followpos(i) U= firstpos(n2)`.
- Para cada nodo `star(n)` o `plus(n)`: Para todo `i` en `lastpos(n)`, `followpos(i) U= firstpos(n)`.
- Los nodos `opt(?)`, `or(|)` y las hojas **no** generan entradas en followpos.

#### 2.3.3 Construccion de Estados desde followpos

El algoritmo de construccion del DFA:
1. Estado inicial = `firstpos(root)`.
2. Para cada estado no procesado `S`:
   - Para cada simbolo `a` en el alfabeto que aparece en alguna posicion de `S`:
     - `U = union de followpos(i)` para todo `i` en `S` donde `posToSymbol[i] == a`.
     - Si `U` no existe como estado, crearlo y agregarlo a la lista de trabajo.
     - Agregar transicion `delta(S, a) = U`.
3. Un estado es aceptor si contiene la posicion del marcador `#`.

### 2.4 Minimizacion de DFA (Metodo Table-Filling / Piramide)

El algoritmo de minimizacion reduce el DFA eliminando estados equivalentes:

1. **Tabla triangular:** Para cada par de estados `(p, q)`, crear entrada no marcada.
2. **Inicializacion:** Marcar como distinguibles todos los pares donde uno es aceptor y el otro no.
3. **Propagacion iterativa:** Para cada par no marcado `(p, q)`, si existe un simbolo `a` tal que `delta(p, a)` y `delta(q, a)` llevan a un par ya marcado, entonces marcar `(p, q)`.
4. **Clases de equivalencia:** Los pares no marcados al final son estados equivalentes. Se agrupan usando Union-Find.
5. **Reconstruccion:** Se construye un DFA nuevo con un estado por clase de equivalencia.

**Complejidad:** `O(|Q|^2 x |Sigma|)` por pasada, `O(|Q|^2)` pasadas en el peor caso, total `O(|Q|^3 x |Sigma|)`.

### 2.5 Maximal Munch y Desambiguacion por Prioridad

La estrategia de tokenizacion sigue dos principios:

1. **Maximal Munch (avidez maxima):** Siempre se busca el match mas largo posible desde la posicion actual. Si multiples DFAs pueden consumir mas caracteres, se sigue avanzando.

2. **Prioridad por orden de reglas:** Cuando multiples DFAs aceptan el mismo lexema (misma longitud), se elige el DFA cuya regla aparece primero en el archivo `.yal`. Esto permite que keywords como `"let"` tengan prioridad sobre el patron de identificadores `ID`, siempre que se declaren antes.

**Ejemplo:**
- Input: `"let"` - tanto KEYWORD como ID aceptan en posicion 3. KEYWORD tiene prioridad menor (declarado antes), asi que gana.
- Input: `"letter"` - KEYWORD acepta en posicion 3 pero ID acepta en posicion 6. ID gana por Maximal Munch.

### 2.6 Formato YALex: Estructura, Macros, Reglas, Acciones

Un archivo YALex tiene la siguiente estructura general:

```
{ header }

let ident = regexp ...

rule entrypoint =
  | regexp { action }
  | regexp { action }
  ...

{ trailer }
```

- **Header/Trailer:** Bloques opcionales de codigo delimitados por `{ ... }`.
- **Macros:** Definiciones reutilizables con `let NOMBRE = expresion`.
- **Reglas:** Cada `| patron { accion }` define un par patron-accion.
- **Comentarios:** Delimitados por `(* ... *)`.
- **Acciones:** Strings que identifican el tipo de token (e.g., `INT`, `KEYWORD`, `skip`).

---

## 3. Arquitectura del Sistema

### 3.1 Vista General del Pipeline

genAnaLex implementa un pipeline lineal de 14 modulos que transforman la especificacion `.yal` en un analizador lexico funcional:

```
Archivo .yal
     |
     v
[1] YalexParser ---- macros + rules ---->
     |
     v
[2] MacroExpander -- expanded_rules ---->
     |
     v  (por cada regla expandida)
[3] RegexNormalizer -- tokens normalizados ---->
     |
     v
[4] RegexBuilder --- postfix tokens ---->
     |
     v
[5] SyntaxTreeBuilder -- root + posToSymbol ---->
     |
     v
[6] Nullable ----\
[7] FirstPos -----+--- funciones de posicion ---->
[8] LastPos -----/
     |
     v
[9] FollowPos ----- tabla followpos ---->
     |
     v
[10] DirectDFABuilder --- DFA crudo ---->
     |
     v
[11] DFAMinimizer --- DFA minimizado ---->
     |
     v
[12] InputReader ---\
                     +--- Tokenizacion --->
[13] LexerSimulator /
     |
     v
[14] Generator ---- archivo .go standalone
```

### 3.2 Diagrama de Flujo de Datos

El flujo de datos es estrictamente unidireccional: cada modulo recibe su entrada del modulo anterior y produce su salida para el siguiente. No hay retroalimentacion entre modulos. Este diseno permite:

- **Testeo independiente** de cada modulo con datos de entrada sinteticos.
- **Bajo acoplamiento** - cada modulo puede ser reemplazado sin afectar a los demas.
- **Depuracion predecible** - se puede inspeccionar la salida intermedia en cada punto del pipeline.

Los pasos 3-11 se ejecutan una vez **por cada regla** del archivo `.yal`. El resultado es un arreglo de `DFAEntry` que contiene todos los DFAs minimizados, cada uno etiquetado con su nombre de token y prioridad.

### 3.3 Estructura del Proyecto

```
genanalex/
|-- main.go                          # Punto de entrada CLI
|-- go.mod                           # Modulo Go (genanalex, go 1.26.1)
|-- integration_test.go              # Tests de integracion end-to-end
|-- internal/
|   |-- yalex/
|   |   |-- parser.go                # [1] Parser de archivos .yal
|   |   |-- expander.go              # [2] Expansion recursiva de macros
|   |   |-- yalex_test.go            # Tests del paquete yalex
|   |-- regex/
|   |   |-- normalizer.go            # [3] Tokenizacion y normalizacion
|   |   |-- builder.go               # [4] Augmentacion y Shunting-Yard
|   |   |-- regex_test.go            # Tests del paquete regex
|   |-- dfa/
|   |   |-- tree.go                  # [5] Construccion de arbol sintactico
|   |   |-- positions.go             # [6-8] Nullable, FirstPos, LastPos
|   |   |-- followpos.go             # [9] Calculo de followpos
|   |   |-- builder.go               # [10] Construccion directa de DFA
|   |   |-- minimizer.go             # [11] Minimizacion Table-Filling
|   |   |-- dfa_test.go              # Tests del paquete dfa
|   |-- lexer/
|   |   |-- reader.go                # [12] Lectura de archivos fuente
|   |   |-- simulator.go             # [13] Simulador Maximal Munch
|   |   |-- lexer_test.go            # Tests del paquete lexer
|   |-- generator/
|       |-- generator.go             # [14] Generacion de codigo Go
|       |-- generator_test.go        # Tests del paquete generator
|-- testdata/
    |-- lexer.yal                    # Especificacion de ejemplo (Lisp/Scheme)
    |-- test.lisp                    # Archivo fuente de prueba
```

### 3.4 Dependencias entre Modulos

```
yalex.parser --> yalex.expander --> regex.normalizer --> regex.builder
    --> dfa.tree --> dfa.positions --> dfa.followpos --> dfa.builder
    --> dfa.minimizer --> lexer.simulator
                     \-> generator.generator

lexer.reader --> lexer.simulator (independiente del pipeline .yal)
```

El paquete `dfa` tiene dependencia interna sobre `regex` (usa `regex.EndMarker` y `regex.ConcatOp`). Los demas paquetes solo dependen del paquete inmediatamente anterior en la cadena.

---

## 4. Diseno e Implementacion Detallada

### 4.0 Orquestacion del Pipeline (`main.go`)

#### Proposito

`main.go` es el punto de entrada del sistema. Orquesta la ejecucion completa del pipeline, desde la lectura de la especificacion `.yal` hasta la tokenizacion o generacion de codigo, segun los flags proporcionados por el usuario.

#### Parsing de Argumentos

El programa acepta cuatro flags a traves del paquete estandar `flag`:

```go
yalFile := flag.String("yal", "", "path to the .yal lexer specification file")
srcFile := flag.String("src", "", "path to the source file to tokenize")
outFile := flag.String("out", "", "path to the output .go file for the generated lexer")
genTree := flag.Bool("tree", false, "generate tree.dot Graphviz file for syntax trees")
```

La validacion requiere siempre `-yal` y al menos uno de `-src`, `-out` o `-tree`. Sin `-yal`, el programa imprime un mensaje de uso y termina con codigo 1. Sin ninguna de las tres opciones de salida, tambien termina con error.

#### Flujo de Control

El pipeline se ejecuta en 6 pasos secuenciales:

1. **Parsing del archivo `.yal`** - Invoca `yalex.ParseFile()` para obtener macros y reglas.
2. **Expansion de macros** - Invoca `yalex.Expand()` para sustituir todas las referencias a macros en los patrones.
3. **Construccion de DFAs** - Este es el nucleo del pipeline. Se itera sobre cada regla expandida y se ejecuta la subcadena completa:
   - `regex.Normalize()` - Tokenizacion y normalizacion del patron.
   - `regex.BuildPostfix()` - Conversion a postfix aumentado.
   - `dfa.BuildTree()` - Construccion del arbol sintactico.
   - (Opcional) `dfa.ToDOT()` - Si `-tree` esta activo, se genera la representacion Graphviz.
   - `dfa.BuildDFA()` - Construccion directa del DFA.
   - `dfa.Minimize()` - Minimizacion del DFA.
   - El resultado se acumula en un slice de `lexer.DFAEntry` con `TokenName` y `Priority`.
1. **Visualizacion de arboles** (opcional) - Si `-tree` esta activo, se escriben todos los DOTs acumulados al archivo `tree.dot`.
2. **Generacion de codigo** (opcional) - Si `-out` esta activo, se invoca `generator.GenerateSource()` con la ruta de salida y el slice de DFAs.
3. **Tokenizacion directa** (opcional) - Si `-src` esta activo, se lee el archivo fuente con `lexer.ReadSource()` y se tokeniza con `lexer.Tokenize()`. Los tokens se imprimen en formato `[linea] TIPO lexema` y los errores lexicos se reportan al final.

```go
for _, rule := range expandedRules {
    normalized, err := regex.Normalize(rule.Pattern)
    postfix, err := regex.BuildPostfix(normalized)
    root, posToSymbol, err := dfa.BuildTree(postfix)
    builtDFA := dfa.BuildDFA(root, posToSymbol, rule.Action)
    minimizedDFA := dfa.Minimize(builtDFA)
    dfaEntries = append(dfaEntries, lexer.DFAEntry{
        DFA: minimizedDFA, TokenName: rule.Action, Priority: rule.Priority,
    })
}
```

#### Notas de Diseno

- **Los pasos 4, 5 y 6 son independientes** entre si - pueden ejecutarse en cualquier orden (o todos juntos). Esto se refleja en la evaluacion secuencial de los tres `if` sin `else`.
- **Manejo de errores:** Cada paso del pipeline verifica errores y termina con `os.Exit(1)` en caso de fallo, excepto el paso de escritura de `tree.dot` que solo imprime un warning a stderr sin abortar.
- **El bucle del paso 3 es per-regla:** Cada regla del `.yal` produce un DFA independiente. No hay comparticion de estado entre iteraciones.

### 4.1 Modulo YalexParser (`internal/yalex/parser.go`)

#### Proposito y Responsabilidades

Leer un archivo `.yal`, eliminar comentarios, extraer header/trailer, recolectar macros y parsear las reglas de tokens.

#### Estructuras de Datos

```go
type TokenRule struct {
    Pattern  string // La expresion regular del patron
    Action   string // La accion asociada (nombre del token)
    Priority int    // Orden de aparicion (0-indexed)
}

type ParseResult struct {
    Macros map[string]string // Macros: nombre -> patron
    Rules  []TokenRule       // Reglas en orden de aparicion
}
```

#### Algoritmo de Parsing

El metodo `Parse(content string)` ejecuta los siguientes pasos:

**Paso 1 - Eliminacion de comentarios `(* ... *)`:**

```go
func removeComments(s string) string {
    var buf strings.Builder
    i := 0
    for i < len(s) {
        if i+1 < len(s) && s[i] == '(' && s[i+1] == '*' {
            end := strings.Index(s[i+2:], "*)")
            if end == -1 {
                break // Comentario sin cerrar: se descarta el resto
            }
            i = i + 2 + end + 2
        } else {
            buf.WriteByte(s[i])
            i++
        }
    }
    return buf.String()
}
```

El algoritmo recorre el string carácter a carácter. Cuando encuentra `(*`, busca el `*)` correspondiente y salta todo el contenido. Si el comentario no se cierra, se descarta todo lo restante.

**Paso 2 - Extraccion de header y trailer:**

Se identifican bloques `{ ... }` que aparecen antes de la seccion `rule` (header) y despues de la ultima regla (trailer). Se eliminan del contenido para que no interfieran con el parsing de macros y reglas.

La funcion `removeHeaderTrailer` divide el texto en dos partes usando la posicion de `"rule "` como punto de corte. Si no existe la palabra `rule`, todo el texto se trata como prefijo.

```go
func removeHeaderTrailer(s string) string {
    ruleIdx := strings.Index(s, "rule ")
    if ruleIdx == -1 {
        ruleIdx = len(s) // Sin seccion rule: todo es prefijo
    }
    prefix := s[:ruleIdx]
    suffix := s[ruleIdx:]
    prefix = removeTopLevelBraces(prefix)
    suffix = removeTrailingBraceBlock(suffix)
    return prefix + suffix
}
```

**`removeTopLevelBraces(s string)`** - Elimina todos los bloques `{ ... }` en el nivel superior del prefijo (antes de `rule`). Usa un contador de profundidad: incrementa con `{`, decrementa con `}`, y solo escribe caracteres cuando `depth == 0`. Esto remueve headers como `{ package main }` sin afectar llaves anidadas.

```go
func removeTopLevelBraces(s string) string {
    var buf strings.Builder
    depth := 0
    for _, c := range s {
        if c == '{' { depth++ }
        else if c == '}' { depth-- }
        else if depth == 0 { buf.WriteRune(c) }
    }
    return buf.String()
}
```

**`removeTrailingBraceBlock(s string)`** - Elimina el bloque trailer `{ ... }` al final de la seccion de reglas. Esta funcion es no trivial porque debe distinguir el trailer (un bloque `{ ... }` al final del archivo) de los bloques de accion `{ TOKEN }` dentro de las reglas.

El algoritmo usa un escaneo **hacia atras** (backward scan):

1. Busca el ultimo `}` en el string. Si no existe, no hay trailer.
2. Desde el ultimo caracter hacia el inicio, mantiene un contador `depth`: incrementa con `}` y decrementa con `{`.
3. Cuando `depth` llega a 0, ha encontrado el `{` correspondiente al ultimo `}` - este es el posible inicio del trailer.
4. **Validacion:** Verifica que el texto antes de este `{` (despues de hacer trim) termine en `}`. Si es asi, ese `}` es el cierre de la ultima regla y el bloque encontrado es efectivamente un trailer. Si no termina en `}`, el bloque es parte de una accion de regla y no se elimina.

```go
func removeTrailingBraceBlock(s string) string {
    lastClose := strings.LastIndex(s, "}")
    if lastClose == -1 { return s }

    runes := []rune(s)
    depth := 0
    trailerStart := -1
    for i := len(runes) - 1; i >= 0; i-- {
        if runes[i] == '}' { depth++ }
        else if runes[i] == '{' {
            depth--
            if depth == 0 { trailerStart = i; break }
        }
    }
    if trailerStart == -1 { return s }

    before := strings.TrimSpace(string(runes[:trailerStart]))
    if !strings.HasSuffix(before, "}") { return s }
    return before
}
```

La logica de escaneo hacia atras es necesaria porque un escaneo hacia adelante no podria distinguir el inicio del trailer del inicio de un bloque de accion sin parsear completamente las reglas primero.

**Paso 3 - Extraccion de macros:**

Se usa una expresion regular para encontrar lineas con el patron `let NOMBRE = patron`:

```go
letRe := regexp.MustCompile(`(?m)^[ \t]*let[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*=[ \t]*(.+)$`)
```

**Paso 4 - Parsing de reglas:**

Se localiza la seccion `rule entrypoint = ...` y se extrae el cuerpo de reglas. Este cuerpo se divide por pipes (`|`) al nivel superior usando `splitByPipe`.

#### Funcion `splitByPipe`

Esta funcion es critica porque debe ignorar pipes dentro de comillas simples, comillas dobles, corchetes, llaves y parentesis:

```go
func splitByPipe(s string) []string {
    var parts []string
    var cur strings.Builder
    inSingle := false
    inDouble := false
    inBracket := 0
    inBrace := 0
    inParen := 0
    // ...
    case c == '|' && !inSingle && !inDouble && inBracket == 0 &&
         inBrace == 0 && inParen == 0:
        parts = append(parts, cur.String())
        cur.Reset()
    // ...
}
```

Mantiene contadores de profundidad para corchetes, llaves y parentesis, y flags booleanos para comillas simples y dobles. Solo divide por `|` cuando todos los contadores estan en cero y no se esta dentro de comillas.

#### Funcion `extractPatternAction`

Separa un segmento de regla en su patron y accion. Busca la primera `{` al nivel superior (fuera de comillas y corchetes) y extrae el contenido entre `{` y su `}` correspondiente.

#### Manejo de Multi-Pipe Alternatives

Cuando se tienen reglas como:
```
| 'T' | 'F' { BOOLEAN }
```

El parser genera dos reglas separadas, cada una con el patron individual (`'T'` y `'F'`) pero compartiendo la misma accion (`BOOLEAN`). Esto se implementa acumulando segmentos sin accion en un buffer `pending` y, al encontrar un segmento con accion, aplicando esa accion a todos los pendientes.

#### Manejo de comillas escapadas

El patron `'\''` (comilla simple escapada) se maneja correctamente: cuando se detecta un `\` dentro del modo single-quote, se salta el siguiente caracter antes de buscar el cierre `'`.

### 4.2 Modulo MacroExpander (`internal/yalex/expander.go`)

#### Proposito

Sustituir todas las referencias a macros en los patrones de las reglas por sus expresiones expandidas completas.

#### Algoritmo de Expansion Recursiva con Deteccion de Ciclos

La expansion se realiza en dos fases:

**Fase 1 - Expansion de macros entre si:**

Las macros pueden referenciar a otras macros (e.g., `NUMBER = DIGIT+` donde `DIGIT = [0-9]`). Se usa un resolver recursivo con tres conjuntos:

```go
func expandMacros(macros map[string]string) (map[string]string, error) {
    expanded := make(map[string]string)   // Macros ya completamente expandidas
    visited := make(map[string]bool)       // Macros visitadas (para debug)
    inStack := make(map[string]bool)       // Macros en la pila de recursion (deteccion de ciclos)

    var expandOne func(name string) (string, error)
    expandOne = func(name string) (string, error) {
        if v, ok := expanded[name]; ok {
            return v, nil                  // Ya expandida: retornar cache
        }
        if inStack[name] {
            return "", fmt.Errorf("cyclic macro reference: %s", name)
        }
        // ...
        inStack[name] = true
        result, err := expandPatternWithResolver(val, expanded, expandOne)
        inStack[name] = false
        expanded[name] = result
        return result, nil
    }
}
```

**Fase 2 - Expansion de patrones de reglas:**

Cada identificador en el patron que coincida con un nombre de macro se reemplaza por su valor expandido, envuelto en parentesis.

#### Funcion `expandPatternWithResolver`

Esta es la funcion central de expansion. Recorre el patron caracter a caracter y:

1. **Comillas simples `'...'`:** Se copian literalmente sin expansion.
2. **Comillas dobles `"..."`:** Se copian literalmente sin expansion.
3. **Corchetes `[...]`:** Se copian literalmente sin expansion.
4. **Operadores `|`, `*`, `+`, `?`, `.`, `_`:** Se pasan directamente.
5. **Identificadores:** Se intenta resolver con el resolver. Si tiene exito, se escribe `(valor_expandido)`. Si el error contiene "undefined macro", se escribe el identificador tal cual (no es una macro).

#### Wrapping en Parentesis

Cuando una macro se expande, su valor se envuelve en parentesis. Esto es esencial para preservar la precedencia correcta. Por ejemplo:
```
DIGIT = [0-9]
Patron: DIGIT+
Expandido: ([0-9])+     ← El + aplica a toda la clase, no solo al 9
```

Sin parentesis, `[0-9]+` se interpretaria incorrectamente como `[0-9]` seguido de `+` aplicado al ultimo caracter.

#### Edge Cases

- **Macros ciclicas (`A=B, B=A`):** El conjunto `inStack` detecta la recursion y produce un error explicito.
- **Macros transitivas (`A=B, B=C, C=[0-9]`):** La recursion resuelve correctamente la cadena completa.
- **Identificadores indefinidos:** Se tratan como literales (no son macros), permitiendo que patrones contengan texto literal.

### 4.3 Modulo RegexNormalizer (`internal/regex/normalizer.go`)

#### Proposito

Transformar un patron YALex crudo en una secuencia de `RegexToken`, expandiendo clases de caracteres, resolviendo escapes e insertando operadores de concatenacion explicitos.

#### Estructura RegexToken

```go
type TokKind int

const (
    TokAtom  TokKind = iota // Caracter literal
    TokOp                    // Operador (|, *, +, ?, ConcatOp)
    TokOpen                  // Parentesis de apertura
    TokClose                 // Parentesis de cierre
)

type RegexToken struct {
    Kind TokKind
    Atom rune // Valido cuando Kind == TokAtom
    Op   rune // Valido cuando Kind == TokOp
}
```

Esta representacion estructurada elimina la ambiguedad entre caracteres literales y operadores. Por ejemplo, el caracter `|` puede ser tanto un operador de alternancia (`TokOp`) como un literal (`TokAtom`).

#### Constantes Sentinela

```go
const ConcatOp = '\x01'  // Operador de concatenacion explicita
const EndMarker = '\x00'  // Marcador de fin para augmentacion
```

Ambos valores estan fuera del alfabeto de entrada (ASCII 32-126 + `\t` + `\r`), garantizando que no haya colisiones con caracteres del usuario.

#### Tokenizacion de Patrones

La funcion `tokenize(pattern string)` procesa cada tipo de elemento:

**Literales single-quoted (`'a'`, `'\n'`, `'\''`):**
```go
case '\'':
    i++ // Skip opening quote
    r, newI, err := parseSingleQuoted(runes, i)
    // ...
    i = newI
    if i >= len(runes) || runes[i] != '\'' {
        return nil, fmt.Errorf("expected closing ' at position %d", i)
    }
    i++ // Skip closing quote
    tokens = append(tokens, atomTok(r))
```

La funcion `parseSingleQuoted` delega a `parseEscape` cuando encuentra un backslash dentro de las comillas.

#### Funcion `parseEscape` - Mapeo Completo de Secuencias de Escape

La funcion `parseEscape(c rune)` convierte un caracter escapado a su valor rune correspondiente. Es utilizada tanto por `parseSingleQuoted` (literales `'...'`) como por la tokenizacion de strings doble-comillas (`"..."`).

```go
func parseEscape(c rune) (rune, error) {
    switch c {
    case 'n':  return '\n', nil   // Salto de linea (0x0A)
    case 't':  return '\t', nil   // Tabulador horizontal (0x09)
    case 'r':  return '\r', nil   // Retorno de carro (0x0D)
    case '\\': return '\\', nil   // Backslash literal
    case '\'': return '\'', nil   // Comilla simple literal
    case '"':  return '"', nil    // Comilla doble literal
    case '0':  return '\x00', nil // Byte nulo (0x00)
    default:   return c, nil      // Fallback: caracter literal
    }
}
```

| Secuencia | Resultado | Valor | Uso tipico |
|-----------|-----------|-------|------------|
| `\n` | Salto de linea | `0x0A` | Detectar fin de linea en comentarios |
| `\t` | Tabulador | `0x09` | Whitespace en `[' ' '\t']` |
| `\r` | Retorno de carro | `0x0D` | Compatibilidad Windows |
| `\\` | Backslash literal | `0x5C` | Patron de strings con escapes |
| `\'` | Comilla simple | `0x27` | Literal `'\''` |
| `\"` | Comilla doble | `0x22` | Dentro de strings doble-comillas |
| `\0` | Byte nulo | `0x00` | Coincide con `EndMarker` (`\x00`) |
| `\x` (otro) | Caracter literal `x` | Mismo que `x` | Fallback para escapes no reconocidos |

**Nota sobre `\0`:** El mapeo de `\0` a `\x00` es significativo porque `\x00` es el mismo valor que `EndMarker`. Sin embargo, en la practica un usuario no deberia usar `\0` en patrones de tokens, ya que colisionaria con el marcador interno de fin de expresion.

**Nota sobre el fallback:** Las secuencias de escape no reconocidas (por ejemplo `\z`) no producen un error - simplemente retornan el caracter literal (`z`). Esto es una decision de tolerancia que evita fallos en archivos `.yal` con escapes no estandar.

**Strings double-quoted (`"abc"` -> `(a·b·c)`):**

Las cadenas se convierten en una secuencia de atomos concatenados dentro de un grupo. Un solo caracter se emite como atomo simple; una cadena vacia no produce tokens.

```go
case '"':
    // ...collect chars...
    if len(chars) == 1 {
        tokens = append(tokens, atomTok(chars[0]))
    } else {
        tokens = append(tokens, openTok())
        for j, r := range chars {
            tokens = append(tokens, atomTok(r))
            if j < len(chars)-1 {
                tokens = append(tokens, opTok(ConcatOp))
            }
        }
        tokens = append(tokens, closeTok())
    }
```

**Clases de caracteres (`[a-z]`, `[^ '\n']`):**

La funcion `expandCharClass` parsea la clase y devuelve un grupo de alternacion `(a|b|c|...)`:

```go
func expandCharClass(runes []rune, i int) ([]RegexToken, int, error) {
    complement := false
    if i < len(runes) && runes[i] == '^' {
        complement = true
        i++
    }
    // ...parsear rangos y literales...
    if complement {
        alphabet := buildAlphabet()
        // Incluir todos los caracteres del alfabeto EXCEPTO los especificados
    }
    return buildAlternationGroup(chars), i, nil
}
```

Soporta:
- Rangos: `a-z`, `0-9`, `A-Z`
- Literales entre comillas dentro de la clase: `[' ' '\t']`
- Escapes: `[\n]`
- Complemento: `[^ ...]`

**Wildcards (`.` y `_`):**

Ambos se expanden al grupo de alternacion del alfabeto completo:

```go
func buildAlphabet() []rune {
    var alphabet []rune
    alphabet = append(alphabet, '\t', '\r')
    for r := rune(32); r <= 126; r++ {
        alphabet = append(alphabet, r)
    }
    return alphabet
}
```

El alfabeto consta de 97 caracteres: `\t` (9), `\r` (13) y ASCII 32-126 (95 caracteres). Notablemente, `\n` (10) se excluye del alfabeto de wildcards - los saltos de linea se manejan explicitamente en las reglas del `.yal`.

#### Manejo de Whitespace en `tokenize`

El caso `default` de la funcion `tokenize` **descarta silenciosamente** los caracteres de espacio en blanco que aparecen entre elementos del patron:

```go
default:
    if unicode.IsSpace(c) {
        i++
        continue
    }
    tokens = append(tokens, atomTok(c))
    i++
```

Esto incluye espacios (` `), tabuladores (`\t`), saltos de linea (`\n`) y retornos de carro (`\r`) que aparecen **fuera** de comillas, corchetes o como parte de operadores. Es una decision de diseno que permite escribir patrones con formato legible en el archivo `.yal`:

```
DIGIT+ '.' DIGIT+       <- Los espacios entre DIGIT+, '.' y DIGIT+ se ignoran
[' ' '\t' '\n' '\r']    <- Los espacios dentro de [...] se manejan por expandCharClass
```

Sin este comportamiento, cada espacio en un patron se interpretaria como un atomo literal, produciendo resultados incorrectos. Los caracteres de espacio que SI deben ser literales se escriben entre comillas: `' '` o `'\t'`.

#### Insercion de Concatenacion Explicita (`insertConcat`)

La concatenacion es implicita en la sintaxis regex (`ab` significa `a` seguido de `b`). El normalizador la hace explicita insertando `ConcatOp` entre pares adyacentes que lo requieran:

```go
func needsConcat(left, right RegexToken) bool {
    leftOutput := left.Kind == TokAtom ||
        left.Kind == TokClose ||
        (left.Kind == TokOp && (left.Op == '*' || left.Op == '+' || left.Op == '?'))

    rightInput := right.Kind == TokAtom || right.Kind == TokOpen

    return leftOutput && rightInput
}
```

Se inserta `·` cuando el token izquierdo "termina" una expresion (atomo, cierre de parentesis o cuantificador) y el token derecho "comienza" una expresion (atomo o apertura de parentesis).

Ejemplos:
```
'a' 'b'      -> a · b
'a'* 'b'     -> a * · b
('a'|'b') c  -> (a|b) · c
```

### 4.4 Modulo RegexBuilder (`internal/regex/builder.go`)

#### Proposito

Aumentar la expresion regular con el marcador de fin `#` y convertirla de notacion infija a postfija usando el algoritmo Shunting-Yard.

#### Paso 1: Augmentacion

```go
func BuildPostfix(normalized []RegexToken) ([]RegexToken, error) {
    augmented := make([]RegexToken, 0, len(normalized)+4)
    augmented = append(augmented, openTok())
    augmented = append(augmented, normalized...)
    augmented = append(augmented, closeTok())
    augmented = append(augmented, opTok(ConcatOp))
    augmented = append(augmented, atomTok(EndMarker))

    return shuntingYard(augmented)
}
```

La expresion `regex` se transforma en `(regex) · #`, asegurando que el EndMarker sea la ultima posicion del arbol.

#### Paso 2: Shunting-Yard

La tabla de precedencias:

| Operador | Precedencia | Tipo |
|----------|-------------|------|
| `*`, `+`, `?` | 3 | Unario sufijo |
| `·` (ConcatOp) | 2 | Binario |
| `\|` | 1 | Binario |

**Tratamiento de operadores unarios:**

Los operadores unarios sufijos (`*`, `+`, `?`) se emiten **directamente al output** sin pasar por la pila:

```go
case TokOp:
    op := tok.Op
    if isUnary(op) {
        output = append(output, tok)
    } else {
        // Logica estandar de Shunting-Yard para binarios
        for len(opStack) > 0 {
            top := opStack[len(opStack)-1]
            if top.Kind == TokOpen { break }
            if precedence(topOp) >= precedence(op) {
                output = append(output, top)
                opStack = opStack[:len(opStack)-1]
            } else { break }
        }
        opStack = append(opStack, tok)
    }
```

Esta es la forma correcta de manejar operadores unarios postfijos: al encontrarlos, su operando ya esta en el output, asi que se emiten inmediatamente. Empujarlos a la pila causaria reordenamiento incorrecto en casos como `a*+` (que debe producir `a * +` en postfix, significando `(a*)+`).

### 4.5 Modulo SyntaxTreeBuilder (`internal/dfa/tree.go`)

#### Proposito

Construir el arbol sintactico a partir de la expresion en notacion postfija y asignar posiciones unicas a las hojas.

#### Estructura Node

```go
type NodeKind int

const (
    NodeLeaf    NodeKind = iota // Hoja con simbolo
    NodeEpsilon                  // Epsilon (cadena vacia)
    NodeCat                      // Concatenacion
    NodeOr                       // Alternancia
    NodeStar                     // Cerradura de Kleene
    NodePlus                     // Cerradura positiva
    NodeOpt                      // Opcional
)
```

> **Nota sobre `NodeEpsilon`:** Aunque `NodeKind` define `NodeEpsilon` y las funciones `Nullable`, `FirstPos` y `LastPos` incluyen reglas para manejarlo, la funcion `BuildTree` **nunca construye nodos epsilon**. Esto es por diseno: el operador `?` (opcional) se modela directamente como `NodeOpt` en lugar de transformarse en `(expr | epsilon)`. De manera similar, `*` se modela como `NodeStar` en lugar de usar una construccion con epsilon explicito. Las reglas para `NodeEpsilon` existen como salvaguarda de completitud y para soportar posibles extensiones futuras donde epsilon sea necesario (por ejemplo, si se implementara la transformacion `a? -> a | epsilon`).

#### Algoritmo de Construccion

Se usa un stack. Se recorre la secuencia postfix token por token:

```go
func BuildTree(postfix []regex.RegexToken) (*Node, map[int]rune, error) {
    var stack []*Node
    posCounter := 0
    posToSymbol := make(map[int]rune)

    for _, tok := range postfix {
        switch tok.Kind {
        case regex.TokAtom:
            posCounter++
            leaf := &Node{Kind: NodeLeaf, Symbol: tok.Atom, Pos: posCounter}
            posToSymbol[posCounter] = tok.Atom
            stack = append(stack, leaf)

        case regex.TokOp:
            switch tok.Op {
            case '|':
                r := stack[len(stack)-1]; l := stack[len(stack)-2]
                stack = stack[:len(stack)-2]
                stack = append(stack, &Node{Kind: NodeOr, Left: l, Right: r})
            case regex.ConcatOp:
                r := stack[len(stack)-1]; l := stack[len(stack)-2]
                stack = stack[:len(stack)-2]
                stack = append(stack, &Node{Kind: NodeCat, Left: l, Right: r})
            case '*':
                n := stack[len(stack)-1]; stack = stack[:len(stack)-1]
                stack = append(stack, &Node{Kind: NodeStar, Left: n})
            case '+':
                n := stack[len(stack)-1]; stack = stack[:len(stack)-1]
                stack = append(stack, &Node{Kind: NodePlus, Left: n})
            case '?':
                n := stack[len(stack)-1]; stack = stack[:len(stack)-1]
                stack = append(stack, &Node{Kind: NodeOpt, Left: n})
            }
        }
    }
    // Al final debe quedar exactamente un nodo en el stack (la raiz)
    return stack[0], posToSymbol, nil
}
```

Las posiciones se asignan secuencialmente (1-indexed) a cada hoja en el orden en que aparecen en el postfix.

#### Generacion de Graphviz DOT

La funcion `ToDOT(root *Node)` genera una representacion en formato DOT para visualizacion:

```go
func ToDOT(root *Node) string {
    var sb strings.Builder
    sb.WriteString("digraph syntaxtree {\n")
    sb.WriteString("  node [shape=circle];\n")
    // Recorrido recursivo asignando IDs y creando aristas
    // ...
    sb.WriteString("}\n")
    return sb.String()
}
```

### 4.6 Modulo Nullable (`internal/dfa/positions.go`)

#### Reglas para cada NodeKind

```go
func Nullable(n *Node) bool {
    if n == nil { return false }
    if n.nullableCache != nil { return *n.nullableCache }

    var result bool
    switch n.Kind {
    case NodeEpsilon: result = true
    case NodeLeaf:    result = false
    case NodeOr:      result = Nullable(n.Left) || Nullable(n.Right)
    case NodeCat:     result = Nullable(n.Left) && Nullable(n.Right)
    case NodeStar:    result = true
    case NodePlus:    result = Nullable(n.Left) // Dragon Book: n+ = n·n*
    case NodeOpt:     result = true
    }

    n.nullableCache = &result
    return result
}
```

**Nota sobre `nullable(n+) = nullable(n)`:** La derivacion desde el Dragon Book es `n+ = n · n*`, por lo tanto `nullable(n+) = nullable(n) AND nullable(n*) = nullable(n) AND true = nullable(n)`. Para la mayoria de patrones practicos (como `[0-9]+`), `nullable(n)` es `false`, pero para patrones como `(a?)+`, la expresion interna SI es nullable, haciendo que el plus tambien lo sea.

#### Memoizacion con `*bool`

Se usa un puntero a bool (`*bool`) en lugar de un bool directo porque `nil` indica "no calculado" y `false` indica "calculado como false". Un bool directo no puede distinguir estos dos estados (su zero-value es `false`).

### 4.7 Modulo FirstPos (`internal/dfa/positions.go`)

```go
func FirstPos(n *Node) map[int]bool {
    if n == nil { return map[int]bool{} }
    if n.firstPosCache != nil { return n.firstPosCache }

    var result map[int]bool
    switch n.Kind {
    case NodeEpsilon: result = map[int]bool{}
    case NodeLeaf:    result = map[int]bool{n.Pos: true}
    case NodeOr:      result = unionSets(FirstPos(n.Left), FirstPos(n.Right))
    case NodeCat:
        if Nullable(n.Left) {
            result = unionSets(FirstPos(n.Left), FirstPos(n.Right))
        } else {
            result = FirstPos(n.Left)
        }
    case NodeStar, NodePlus, NodeOpt:
        result = FirstPos(n.Left)
    }

    n.firstPosCache = result
    return result
}
```

La regla clave para `Cat` es: si el hijo izquierdo puede generar epsilon (`Nullable(n.Left) == true`), entonces las primeras posiciones incluyen tanto las del hijo izquierdo como las del hijo derecho.

### 4.8 Modulo LastPos (`internal/dfa/positions.go`)

Simetrico a FirstPos pero sobre el nodo derecho del Cat:

```go
func LastPos(n *Node) map[int]bool {
    // ...
    case NodeCat:
        if Nullable(n.Right) {
            result = unionSets(LastPos(n.Left), LastPos(n.Right))
        } else {
            result = LastPos(n.Right)
        }
    // ...
}
```

### 4.9 Modulo FollowPos (`internal/dfa/followpos.go`)

```go
func ComputeFollowPos(root *Node) map[int]map[int]bool {
    followPos := make(map[int]map[int]bool)
    computeFollow(root, followPos)
    return followPos
}

func computeFollow(n *Node, fp map[int]map[int]bool) {
    if n == nil { return }

    switch n.Kind {
    case NodeCat:
        // Para cada i en lastpos(n.Left): followpos(i) U= firstpos(n.Right)
        lp := LastPos(n.Left)
        fp1 := FirstPos(n.Right)
        for i := range lp {
            if fp[i] == nil { fp[i] = make(map[int]bool) }
            for j := range fp1 { fp[i][j] = true }
        }
        computeFollow(n.Left, fp)
        computeFollow(n.Right, fp)

    case NodeStar, NodePlus:
        // Para cada i en lastpos(n): followpos(i) U= firstpos(n)
        lp := LastPos(n.Left)
        fp1 := FirstPos(n.Left)
        for i := range lp {
            if fp[i] == nil { fp[i] = make(map[int]bool) }
            for j := range fp1 { fp[i][j] = true }
        }
        computeFollow(n.Left, fp)

    case NodeOr, NodeOpt:
        computeFollow(n.Left, fp)
        if n.Right != nil { computeFollow(n.Right, fp) }

    case NodeLeaf, NodeEpsilon:
        // Sin accion
    }
}
```

**Por que `Opt` (?) no genera entradas:** El operador `?` representa "cero o una ocurrencia". A diferencia de `*` y `+`, no crea un bucle (no se puede repetir la expresion), asi que no se agregan entradas en followpos. Los nodos `Or` y hojas tampoco generan entradas directamente.

### 4.10 Modulo DirectDFABuilder (`internal/dfa/builder.go`)

#### Estructura DFA

```go
type DFA struct {
    States      map[int]map[int]bool // ID estado -> conjunto de posiciones
    Transitions map[int]map[rune]int // ID estado -> (simbolo -> ID estado)
    Start       int                  // Estado inicial
    Accepting   map[int]bool         // Estados aceptores
    StateToken  map[int]string       // Estado -> nombre de token
}
```

#### Algoritmo Worklist (BFS)

```go
func BuildDFA(root *Node, posToSymbol map[int]rune, tokenName string) *DFA {
    followPos := ComputeFollowPos(root)

    // Encontrar la posicion del EndMarker
    endPos := -1
    for pos, sym := range posToSymbol {
        if sym == regex.EndMarker { endPos = pos }
    }

    // Estado inicial = firstpos(root)
    initial := FirstPos(root)
    stateID := 0
    keyToID := make(map[string]int)

    initialKey := setKey(initial)
    keyToID[initialKey] = stateID
    dfa.States[stateID] = initial

    if initial[endPos] {
        dfa.Accepting[stateID] = true
        dfa.StateToken[stateID] = tokenName
    }

    var worklist []int
    worklist = append(worklist, stateID)
    stateID++

    for len(worklist) > 0 {
        curr := worklist[0]
        worklist = worklist[1:]
        currState := dfa.States[curr]

        // Encontrar todos los simbolos en este estado
        symbols := make(map[rune]bool)
        for pos := range currState {
            sym := posToSymbol[pos]
            if sym != regex.EndMarker { symbols[sym] = true }
        }

        // Para cada simbolo, calcular la transicion
        for sym := range symbols {
            U := make(map[int]bool)
            for pos := range currState {
                if posToSymbol[pos] == sym {
                    for fp := range followPos[pos] { U[fp] = true }
                }
            }

            if len(U) == 0 { continue }

            key := setKey(U)
            nextID, exists := keyToID[key]
            if !exists {
                nextID = stateID; stateID++
                keyToID[key] = nextID
                dfa.States[nextID] = U
                worklist = append(worklist, nextID)
                if U[endPos] {
                    dfa.Accepting[nextID] = true
                    dfa.StateToken[nextID] = tokenName
                }
            }

            dfa.Transitions[curr][sym] = nextID
        }
    }
    return dfa
}
```

#### Deduplicacion de Estados con `setKey`

La funcion `setKey` convierte un conjunto de posiciones en un string canonico para usarlo como clave de mapa. Los enteros se ordenan y se concatenan con comas:

```go
func setKey(s map[int]bool) string {
    keys := sortedKeys(s)
    result := make([]byte, 0, len(keys)*4)
    for i, k := range keys {
        if i > 0 { result = append(result, ',') }
        result = appendInt(result, k)
    }
    return string(result)
}
```

#### Funciones Auxiliares de `positions.go`

**`sortedKeys(s map[int]bool) []int`** - Extrae las claves de un mapa y las ordena usando **insertion sort** con complejidad `O(n^2)`. Se implementa un sort manual en lugar de usar `sort.Ints()` del paquete `sort` para evitar agregar un import adicional (el paquete `dfa` solo importa `genanalex/internal/regex`). Para los tamanos tipicos de conjuntos de posiciones en un analizador lexico (decenas a cientos de elementos), la diferencia de rendimiento es despreciable.

```go
func sortedKeys(s map[int]bool) []int {
    keys := make([]int, 0, len(s))
    for k := range s { keys = append(keys, k) }
    for i := 0; i < len(keys); i++ {
        for j := i + 1; j < len(keys); j++ {
            if keys[j] < keys[i] { keys[i], keys[j] = keys[j], keys[i] }
        }
    }
    return keys
}
```

**`appendInt(b []byte, n int) []byte`** - Convierte un entero positivo a su representacion decimal y lo agrega a un slice de bytes. Se implementa manualmente (extrayendo digitos con `n%10` en un buffer temporal de 20 bytes) para evitar usar `strconv.Itoa()` o `fmt.Sprintf()`, lo que permite construir las claves `setKey` con **cero allocaciones** de strings intermedios. La funcion solo maneja enteros no-negativos, lo cual es suficiente ya que las posiciones son siempre >= 0.

```go
func appendInt(b []byte, n int) []byte {
    if n == 0 { return append(b, '0') }
    var tmp [20]byte
    i := len(tmp)
    for n > 0 {
        i--
        tmp[i] = byte('0' + n%10)
        n /= 10
    }
    return append(b, tmp[i:]...)
}
```

### 4.11 Modulo DFAMinimizer (`internal/dfa/minimizer.go`)

#### Algoritmo Table-Filling Completo

El minimizador implementa los 5 pasos del metodo de la piramide:

**Paso 1 - Preparacion:**

```go
stateList := sortedStateIDs(d) // Lista estable de IDs
n := len(stateList)
stateIndex := make(map[int]int, n) // Mapa de ID -> indice
```

**Paso 2 - Tabla triangular de distinguibilidad:**

```go
marked := make([][]bool, n)
for i := range marked {
    marked[i] = make([]bool, n)
}
```

Con funciones auxiliares `mark(i, j)` e `isMarked(i, j)` que normalizan los indices para la triangular (siempre `i < j`).

**Paso 3 - Inicializacion:**

```go
for i := 0; i < n; i++ {
    for j := i + 1; j < n; j++ {
        si, sj := stateList[i], stateList[j]
        if d.Accepting[si] != d.Accepting[sj] {
            mark(i, j) // Un aceptor y un no-aceptor son distinguibles
        }
    }
}
```

**Paso 4 - Propagacion iterativa:**

```go
changed := true
for changed {
    changed = false
    for i := 0; i < n; i++ {
        for j := i + 1; j < n; j++ {
            if isMarked(i, j) { continue }
            si, sj := stateList[i], stateList[j]

            for sym := range symbols {
                ri, hasRi := d.Transitions[si][sym]
                rj, hasRj := d.Transitions[sj][sym]

                if hasRi != hasRj {
                    mark(i, j); changed = true; break
                }
                if !hasRi && !hasRj { continue }
                if ri == rj { continue }

                if isMarked(stateIndex[ri], stateIndex[rj]) {
                    mark(i, j); changed = true; break
                }
            }
        }
    }
}
```

**Paso 5 - Union-Find con compresion de caminos:**

```go
parent := make([]int, n)
for i := range parent { parent[i] = i }

var find func(int) int
find = func(x int) int {
    if parent[x] != x {
        parent[x] = find(parent[x]) // Compresion de caminos
    }
    return parent[x]
}
union := func(x, y int) {
    px, py := find(x), find(y)
    if px != py { parent[px] = py }
}

// Unir pares no marcados (estados equivalentes)
for i := 0; i < n; i++ {
    for j := i + 1; j < n; j++ {
        if !isMarked(i, j) { union(i, j) }
    }
}
```

### 4.12 Modulo InputReader (`internal/lexer/reader.go`)

#### Estructura SourceFile

```go
type SourceFile struct {
    Path    string   // Ruta del archivo
    Content string   // Contenido completo normalizado
    Lines   []string // Lineas para error reporting
}
```

#### Normalizacion de Saltos de Linea

```go
func ReadSource(path string) (*SourceFile, error) {
    data, err := os.ReadFile(path)
    // ...
    content := string(data)
    content = strings.ReplaceAll(content, "\r\n", "\n") // Windows -> Unix
    content = strings.ReplaceAll(content, "\r", "\n")    // Old Mac -> Unix
    lines := strings.Split(content, "\n")
    return &SourceFile{Path: path, Content: content, Lines: lines}, nil
}
```

Primero se normalizan `\r\n` a `\n`, luego los `\r` sueltos restantes. El orden es importante para evitar convertir `\r\n` en `\n\n`.

### 4.13 Modulo LexerSimulator (`internal/lexer/simulator.go`)

#### Estructuras

```go
type Token struct {
    Type   string // Tipo de token (e.g., "INT", "ID")
    Lexeme string // Texto que matcheo
    Line   int    // Numero de linea (1-indexed)
}

type DFAEntry struct {
    DFA       *dfa.DFA // DFA minimizado
    TokenName string   // Nombre del token
    Priority  int      // Prioridad (menor = mayor prioridad)
}
```

#### Algoritmo Maximal Munch + Prioridad

```go
func Tokenize(dfas []DFAEntry, src *SourceFile) ([]Token, []string) {
    var tokens []Token
    var errors []string
    runes := []rune(src.Content)
    i := 0
    line := 1

    for i < len(runes) {
        // 1. Inicializar todos los DFAs en su estado inicial
        states := make([]dfaStatus, len(dfas))
        for k, entry := range dfas {
            states[k] = dfaStatus{entry: entry, current: entry.DFA.Start, active: true}
        }

        lastOKPos := -1
        var lastOKMatches []DFAEntry

        // 2. Lookahead loop (Maximal Munch)
        j := i
        for j < len(runes) {
            c := runes[j]
            anyActive := false

            for k := range states {
                if !states[k].active { continue }
                if nextState, ok := transitions[c]; ok {
                    states[k].current = nextState
                    anyActive = true
                } else {
                    states[k].active = false
                }
            }

            if !anyActive { break }
            j++

            // Verificar si algun DFA acepta en esta posicion
            var currentAccepting []DFAEntry
            for k := range states {
                if states[k].active && states[k].entry.DFA.Accepting[states[k].current] {
                    currentAccepting = append(currentAccepting, states[k].entry)
                }
            }
            if len(currentAccepting) > 0 {
                lastOKPos = j
                lastOKMatches = currentAccepting
            }
        }

        // 3. Resolver match o error
        if lastOKPos == -1 {
            // Error lexico: caracter no reconocido
            errors = append(errors, fmt.Sprintf("line %d: unrecognized character %q", line, runes[i]))
            if runes[i] == '\n' { line++ }
            i++ // Recovery: avanzar 1 caracter
            continue
        }

        // 4. Desambiguacion por prioridad
        lexeme := string(runes[i:lastOKPos])
        bestMatch := lastOKMatches[0]
        for _, m := range lastOKMatches[1:] {
            if m.Priority < bestMatch.Priority { bestMatch = m }
        }

        // 5. Emision o skip
        if bestMatch.TokenName != "skip" {
            tokens = append(tokens, Token{Type: bestMatch.TokenName, Lexeme: lexeme, Line: line})
        }

        // 6. Actualizar linea
        for _, r := range lexeme {
            if r == '\n' { line++ }
        }
        i = lastOKPos
    }
    return tokens, errors
}
```

**Puntos clave del algoritmo:**

1. **Simulacion paralela:** Todos los DFAs corren simultaneamente sobre el mismo input.
2. **Maximal Munch:** Se sigue avanzando mientras al menos un DFA este activo, guardando la ultima posicion donde algun DFA acepto.
3. **Desempate por prioridad:** Si multiples DFAs aceptan en la misma posicion mas larga, gana el de menor `Priority` (declarado antes en el `.yal`).
4. **Tokens "skip":** La accion especial `skip` descarta el lexema (util para whitespace y comentarios).
5. **Error recovery:** Si ningun DFA acepta ni un solo caracter, se reporta un error y se avanza 1 posicion para intentar recuperarse.
6. **Line tracking:** Se cuentan los `\n` tanto en lexemas matcheados como en errores.

### 4.14 Modulo Generator (`internal/generator/generator.go`)

#### Proposito

Generar un archivo fuente Go autonomo que implementa un analizador lexico completo, compilable y ejecutable de forma independiente.

#### Template-Based Code Generation

```go
func GenerateSource(outputPath string, entries []lexer.DFAEntry) error {
    tmpl, err := template.New("lexer").Parse(lexerTemplate)
    if err != nil { return fmt.Errorf("parsing template: %w", err) }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, LexerData{DFAs: entries}); err != nil {
        return fmt.Errorf("executing template: %w", err)
    }

    return os.WriteFile(outputPath, buf.Bytes(), 0644)
}
```

#### Serializacion de Tablas DFA

Las transiciones de cada DFA se serializan como mapas literales de Go dentro del template:

```go
Transitions: map[int]map[rune]int{
    {{- range $state, $trans := $entry.DFA.Transitions }}
    {{ $state }}: {
        {{- range $char, $next := $trans }}
        {{ printf "%d" $char }}: {{ $next }}, // {{ printf "%q" $char }}
        {{- end }}
    },
    {{- end }}
},
```

Cada transicion se serializa como `runeValue: targetState` con un comentario mostrando el caracter en formato quoted.

#### Estructura del Lexer Generado

El archivo generado contiene:
1. Definiciones de tipos: `Token`, `DFA`, `DFAEntry`.
2. Variable global `dfas` con todas las tablas serializadas.
3. Funcion `main()` que lee un flag `-src` y tokeniza el archivo.
4. Funcion `tokenize()` que implementa Maximal Munch (replica exacta del simulador).

#### Diferencias Estructurales entre el DFA Interno y el DFA Generado

La estructura `DFA` del codigo generado difiere intencionalmente de la estructura `dfa.DFA` interna utilizada durante la construccion:

```go
// DFA interno (internal/dfa/builder.go)
type DFA struct {
    States      map[int]map[int]bool // Conjuntos de posiciones por estado
    Transitions map[int]map[rune]int
    Start       int
    Accepting   map[int]bool
    StateToken  map[int]string
}

// DFA generado (template en generator.go)
type DFA struct {
    Transitions map[int]map[rune]int
    Start       int
    Accepting   map[int]bool
    TokenName   string
}
```

**Omision de `States`:** El campo `States` almacena los conjuntos de posiciones del arbol sintactico que forman cada estado del DFA. Esta informacion es necesaria durante la **construccion** (para calcular transiciones via followpos) pero es innecesaria para la **simulacion** - una vez construido el DFA, solo se necesitan las transiciones. Eliminarlo reduce significativamente el tamano del archivo generado.

**`TokenName` dentro del DFA en lugar de `StateToken`:** En el DFA interno, `StateToken` es un mapa `map[int]string` que asocia cada estado aceptor con su nombre de token. En el codigo generado, dado que cada DFA reconoce un solo tipo de token, se simplifica a un unico campo `TokenName string` directamente en la estructura `DFA`. El simulador generado accede al nombre del token via `bestMatch.DFA.TokenName` en lugar de buscarlo en un mapa por estado.

#### Ausencia de Normalizacion CRLF en el Lexer Generado

La funcion `ReadSource` del modulo `lexer/reader.go` normaliza los saltos de linea (`\r\n` -> `\n`, `\r` -> `\n`) antes de tokenizar. El lexer generado **no incluye esta normalizacion** - lee el archivo fuente directamente con `os.ReadFile` y pasa el contenido crudo a `tokenize()`:

```go
content, err := os.ReadFile(*srcPath)
// ...
tokens, errors := tokenize(string(content))
```

Esto significa que archivos con finales de linea Windows (`\r\n`) podrian comportarse de forma distinta entre el simulador (que normaliza) y el lexer generado (que no normaliza). En la practica, si el archivo `.yal` incluye reglas que manejan `\r` explicitamente (por ejemplo, `[' ' '\t' '\n' '\r'] { skip }`), ambos modos produciran los mismos resultados.

#### CLI del Lexer Generado

```
./generated_lexer -src <input_file>
```

El lexer generado imprime los tokens en formato `[linea] TIPO lexema` y reporta errores lexicos.

---

## 5. Especificacion del Archivo `.yal`

### 5.1 Formato Completo

El archivo `testdata/lexer.yal` define el analizador lexico para un subconjunto de Lisp/Scheme:

```yalex
(* Analizador lexico para Lisp/Scheme simplificado *)
(* Tokens segun Laboratorio 2 - Compiladores UVG     *)

let DIGIT   = [0-9]
let LETTER  = [a-zA-Z]
let ALNUM   = [a-zA-Z0-9_]

rule tokens =

  (* Espacios en blanco - se descartan *)
  | [' ' '\t' '\n' '\r']               { skip }

  (* Comentarios de linea: ; hasta fin de linea *)
  | ';' [^ '\n']*                      { COMMENT }

  (* Literales especiales - antes de ID para ganar por prioridad *)
  | "Nil"                              { NIL }
  | ('T' | 'F')                        { BOOLEAN }

  (* Numeros - FLOAT antes que INT *)
  | DIGIT+ '.' DIGIT+                  { FLOAT }
  | DIGIT+                             { INT }

  (* Strings con secuencias de escape: "hola \"mundo\"" *)
  | '"' ([^ '"' '\\'] | '\\' _)* '"'  { STRING }

  (* Operadores de dos caracteres ANTES que los de uno *)
  | "<="                               { OPERATOR }
  | ">="                               { OPERATOR }
  | '+'                                { OPERATOR }
  | '-'                                { OPERATOR }
  | '*'                                { OPERATOR }
  | '/'                                { OPERATOR }
  | '='                                { OPERATOR }
  | '<'                                { OPERATOR }
  | '>'                                { OPERATOR }

  (* Keywords - ANTES que ID para ganar empate de longitud *)
  | "If"                               { KEYWORD }
  | "defun"                            { KEYWORD }
  | "cond"                             { KEYWORD }
  | "let"                              { KEYWORD }

  (* Identificadores *)
  | LETTER ALNUM*                      { ID }

  (* Delimitadores *)
  | '('                                { DELIMITER }
  | ')'                                { DELIMITER }
  | '['                                { DELIMITER }
  | ']'                                { DELIMITER }
  | '{'                                { DELIMITER }
  | '}'                                { DELIMITER }
  | ','                                { DELIMITER }
  | '.'                                { DELIMITER }
  | ':'                                { DELIMITER }
```

### 5.2 Seccion por Seccion

#### Macros

```
let DIGIT   = [0-9]       -- Digitos 0-9
let LETTER  = [a-zA-Z]    -- Letras mayusculas y minusculas
let ALNUM   = [a-zA-Z0-9_] -- Alfanumericos y guion bajo
```

Las macros se expanden recursivamente. `DIGIT` se usa en los patrones de `INT` y `FLOAT`.

#### Regla de Whitespace

```
| [' ' '\t' '\n' '\r']  { skip }
```

Matchea cualquier caracter de espacio en blanco. La accion `skip` indica al simulador que descarte el lexema.

#### Regla de Comentarios

```
| ';' [^ '\n']*  { COMMENT }
```

Matchea un `;` seguido de cero o mas caracteres que no sean `\n`. Captura comentarios de linea al estilo Lisp.

#### Regla de Strings con Escapes

```
| '"' ([^ '"' '\\'] | '\\' _)* '"'  { STRING }
```

Esta regla merece una explicacion detallada:
- `'"'` - matchea la comilla doble de apertura.
- `([^ '"' '\\'] | '\\' _)*` - cero o mas ocurrencias de:
  - `[^ '"' '\\']` - cualquier caracter excepto comilla doble y backslash.
  - `'\\' _` - un backslash seguido de cualquier caracter (secuencia de escape).
- `'"'` - matchea la comilla doble de cierre.

Esto permite strings como `"hello"`, `"hello\"world"` y `"tab\there"`.

#### Orden de Prioridad

El orden de las reglas es significativo:
1. **FLOAT antes que INT:** `3.14` matchea como FLOAT (mas largo) y no como INT(`3`) + DOT(`.`) + INT(`14`).
2. **Operadores multi-caracter antes que los de un caracter:** `<=` matchea como un solo OPERATOR y no como `<` seguido de `=`.
3. **Keywords antes que ID:** `let` matchea como KEYWORD por prioridad cuando ID tambien lo matchea con la misma longitud.
4. **NIL y BOOLEAN antes que ID:** `Nil`, `T` y `F` ganan por prioridad sobre ID.

### 5.3 Archivo de Prueba (`testdata/test.lisp`)

```lisp
(let ((x 42)) (+ x 1.5))
(defun factorial (n)
  (cond ((= n 0) 1)
        (T (* n (factorial (- n 1))))))
; esto es un comentario
(let ((msg "hola mundo")) msg)
(cond ((>= x 0) Nil)
      (F "negativo"))
```

Este archivo ejercita todos los tipos de tokens: delimitadores, keywords, identificadores, enteros, flotantes, strings, booleanos, nil, comentarios y operadores (incluyendo multi-caracter como `>=`).

---

## 6. Trazas de Ejecucion

### 6.1 Traza con Macros - `INT` (Patron `DIGIT+`)

#### Modulo 1 - YalexParser
```
Entrada (.yal):
  let DIGIT = [0-9]
  | DIGIT+   { INT }

Salida:
  macros = { "DIGIT": "[0-9]" }
  rules  = [ ("DIGIT+", "INT") ]
```

#### Modulo 2 - MacroExpander
```
Regla: "DIGIT+"
Busca DIGIT en macros -> "[0-9]"
Sustituye -> "([0-9])+"

Salida: [ ("([0-9])+", "INT") ]
```

#### Modulo 3 - RegexNormalizer
```
Entrada: ([0-9])+
Expandir [0-9] -> (0|1|2|3|4|5|6|7|8|9)
+ es sufijo unario -> no requiere · adicional

Salida: (0|1|2|3|4|5|6|7|8|9)+
```

#### Modulo 4 - RegexBuilder
```
Aumentar -> ((0|1|2|3|4|5|6|7|8|9)+)·#

Shunting Yard (abreviado):
  01|2|3|4|5|6|7|8|9|   <- postfix del grupo
  + -> sufijo            -> 01|2|3|4|5|6|7|8|9|+
  ·# ->                  -> 01|2|3|4|5|6|7|8|9|+#·

Salida postfix: 01|2|3|4|5|6|7|8|9|+#·
```

#### Modulo 5 - SyntaxTreeBuilder
```
Posiciones: 0->1, 1->2, ..., 9->10, #->11

Arbol:
        · (cat)
       / \
      +   #(11)
      |
  or-tree(1..10)

pos_to_symbol: { 1:'0', 2:'1', ..., 10:'9', 11:'#' }
```

#### Modulo 6 - Nullable
```
or-tree -> false   plus -> false   # -> false   cat -> false
```

#### Modulo 7 - FirstPos
```
or-tree -> {1..10}   plus -> {1..10}   # -> {11}
cat: nullable(plus)=false -> firstpos(plus) = {1..10}

firstpos(root) = {1..10}
```

#### Modulo 8 - LastPos
```
or-tree -> {1..10}   plus -> {1..10}   # -> {11}
cat: nullable(#)=false -> lastpos(#) = {11}

lastpos(root) = {11}
```

#### Modulo 9 - FollowPos
```
Nodo plus:  Para todo i en lastpos(plus)={1..10}: followpos(i) U= firstpos(plus)={1..10}
Nodo cat:   Para todo i en lastpos(plus)={1..10}: followpos(i) U= firstpos(#)={11}

  pos 1..10 (digitos) -> {1,2,3,4,5,6,7,8,9,10,11}
  pos 11    (#)       -> {}
```

#### Modulo 10 - DirectDFABuilder
```
Estado A = {1..10}
  sym '0'..'9' -> followpos = {1..10, 11} -> Estado B={1..10,11}

Estado B = {1..10, 11}  <- contiene pos 11 = ACEPTOR (INT)
  sym '0'..'9' -> followpos = {1..10, 11} -> Estado B (bucle)

DFA:
          '0'..'9'       '0'..'9'
  -> A ----------> B <----------+  (B es aceptor: INT)
```

#### Modulo 11 - DFAMinimizer
```
Estados: A (no aceptor), B (aceptor)
Par unico: (A, B) -> A no en F, B en F -> MARCAR
No hay pares sin marcar -> DFA ya es minimo.
```

#### Modulo 13 - LexerSimulator
```
Input: "42"
  A ->'4'-> B (ok, lastOK=1)
  B ->'2'-> B (ok, lastOK=2)
  fin -> emit INT("42")
```

### 6.2 Traza Completa - `BOOLEAN` (Patron `('T'|'F')`)

#### Modulo 1 - YalexParser
```
Salida -> macros={},  rules=[("('T'|'F')", "BOOLEAN")]
```

#### Modulo 2 - MacroExpander
```
Sin macros.  Salida -> [("('T'|'F')", "BOOLEAN")]
```

#### Modulo 3 - RegexNormalizer
```
Entrada: ('T'|'F')
Tokenizacion: ( Atom(T) Op(|) Atom(F) )
Sin clases que expandir.
No hay concat implicita (el | ya separa).
Salida: ( T | F )
```

#### Modulo 4 - RegexBuilder
```
Aumentar -> (('T'|'F'))·#

Shunting Yard:
 Token  Stack     Output
  (     ( (
  T     ( (       T
  |     ( ( |     T
  F     ( ( |     T F
  )     (         T F |
  )               T F |
  ·     ·         T F |
  #     ·         T F | #
 fin              T F | # ·

Salida postfix: T F | # ·
```

#### Modulo 5 - SyntaxTreeBuilder
```
Arbol:
        · (cat)
       / \
      |   # (pos=3)
     / \
  T(1)  F(2)

pos_to_symbol: { 1:'T', 2:'F', 3:'#' }
```

#### Modulo 6 - Nullable
```
T(1) -> false, F(2) -> false, or -> false, #(3) -> false, cat -> false
```

#### Modulo 7 - FirstPos
```
firstpos(root) = {1,2}   <- estado inicial del DFA
```

#### Modulo 8 - LastPos
```
lastpos(root) = {3}
```

#### Modulo 9 - FollowPos
```
cat: Para todo i en lastpos(or)={1,2}: followpos(i) U= firstpos(#)={3}
  followpos(1) = {3}
  followpos(2) = {3}
  followpos(3) = {}
```

#### Modulo 10 - DirectDFABuilder
```
Estado A = {1,2}:
  sym 'T' -> followpos(1) = {3} -> Estado B={3}
  sym 'F' -> followpos(2) = {3} -> Estado B={3}

Estado B = {3}: contiene '#' -> ACEPTOR

DFA:
       T,F
  -> A ----> B (aceptor: BOOLEAN)
```

#### Modulo 11 - DFAMinimizer
```
Par (A,B): A no en F, B en F -> MARCAR
DFA ya es minimo, 2 estados.
```

### 6.3 Traza Compleja - `NIL` (Patron `"Nil"`)

#### Modulo 3 - RegexNormalizer
```
Entrada: "Nil" -> (N·i·l)  [string de 3 caracteres se convierte en concatenacion agrupada]
Insercion de ·: ya incluido por la expansion de double-quotes

Salida: (N · i · l)
```

#### Modulo 4 - RegexBuilder
```
Aumentar -> ((N·i·l))·#

Shunting Yard:
 Token  Stack     Output
  (     ( (
  N     ( (       N
  ·     ( ( ·     N
  i     ( ( ·     N i
  ·     ( ( ·     N i ·      <- pop · (misma prec) antes de push nuevo ·
  l     ( ( ·     N i · l
  )     (         N i · l ·  <- pop ·, descartar (
  )               N i · l ·
  ·     ·         N i · l ·
  #     ·         N i · l · #
 fin              N i · l · # ·

Salida postfix: N i · l · # ·
```

#### Modulo 5 - SyntaxTreeBuilder
```
Arbol:
            · (cat4)
           / \
        · (cat3)  # (pos=4)
        / \
     · (cat2)  l (pos=3)
     / \
  N(1)  i(2)

pos_to_symbol: { 1:'N', 2:'i', 3:'l', 4:'#' }
```

#### Modulo 9 - FollowPos
```
cat2 (N·i):  followpos(1) U= {2}
cat3 (·l):   followpos(2) U= {3}
cat4 (·#):   followpos(3) U= {4}

Tabla final:
  pos 1 (N) -> {2}
  pos 2 (i) -> {3}
  pos 3 (l) -> {4}
  pos 4 (#) -> {}
```

#### Modulo 10 - DirectDFABuilder
```
A={1} --N--> B={2} --i--> C={3} --l--> D={4} (ACEPTOR: NIL)

DFA lineal de 4 estados.
```

#### Modulo 11 - DFAMinimizer
```
Todos los pares marcados (transiciones distintas) -> 4 estados, ya minimo.
```

---

## 7. Suite de Tests

### 7.1 Resumen de Cobertura

El proyecto cuenta con **139 tests** distribuidos en **6 paquetes**, cubriendo desde funciones unitarias hasta el pipeline de integracion completo (incluyendo compilacion del lexer generado).

```
$ go test ./... -count=1
ok   genanalex                    0.720s
ok   genanalex/internal/dfa       0.003s
ok   genanalex/internal/generator 1.263s
ok   genanalex/internal/lexer     0.029s
ok   genanalex/internal/regex     0.004s
ok   genanalex/internal/yalex     0.004s
```

Todos los 139 tests pasan exitosamente.

### 7.2 Tests por Paquete

#### Paquete `yalex` (27 tests)

| Test | Que Verifica |
|------|-------------|
| `TestParse_BasicMacroAndRule` | Parsing basico de una macro y una regla |
| `TestParse_CommentsRemoved` | Eliminacion correcta de comentarios `(* ... *)` |
| `TestParse_MultilineComment` | Comentarios multilinea |
| `TestParse_UnclosedComment` | Comentario sin cerrar descarta el resto |
| `TestParse_NoRuleSection` | Archivo solo con macros (sin reglas) |
| `TestParse_HeaderAndTrailer` | Eliminacion de header y trailer `{ ... }` |
| `TestParse_MultiPipeAlternatives` | Multi-pipe: `\| 'T' \| 'F' { BOOLEAN }` genera 2 reglas |
| `TestParse_EscapedSingleQuote` | Comilla escapada `'\''` |
| `TestParse_PriorityOrder` | Prioridades asignadas en orden 0, 1, 2 |
| `TestSplitByPipe` (7 sub-tests) | Division por pipe respetando comillas, corchetes, llaves, parentesis |
| `TestExtractPatternAction` (3 sub-tests) | Extraccion de patron y accion, incluyendo llaves en comillas |
| `TestRemoveComments` | Eliminacion de comentarios `(* ... *)` como funcion aislada |
| `TestExpand_SimpleMacro` | Expansion basica: `DIGIT` -> `([0-9])` |
| `TestExpand_TransitiveMacro` | Expansion transitiva: `NUMBER` -> `(([0-9])+)` |
| `TestExpand_CyclicMacro` | Deteccion de ciclo directo: `A=A` |
| `TestExpand_CyclicMacro_MutualReference` | Deteccion de ciclo mutuo: `A=B, B=A` |
| `TestExpand_UndefinedMacro` | Macros indefinidas se tratan como literales |
| `TestExpand_NoMacros` | Patrones sin macros pasan sin cambios |
| `TestExpand_MacroInCharClass` | Macros dentro de `[...]` NO se expanden |
| `TestExpand_MacroInSingleQuotes` | Macros dentro de `'...'` NO se expanden |
| `TestExpand_MacroInDoubleQuotes` | Macros dentro de `"..."` NO se expanden |
| `TestExpand_UnderscorePassthrough` | `_` standalone pasa como operador, no como macro |
| `TestExpand_PreservesActionAndPriority` | Accion y prioridad se preservan tras expansion |
| `TestParse_EmptyInput` | Entrada vacia produce 0 macros y 0 reglas |
| `TestParse_OnlyComments` | Solo comentarios produce resultado vacio |
| `TestParse_MultipleMacros` | Multiples macros se almacenan correctamente |
| `TestParseAndExpand_Integration` | Pipeline Parse + Expand end-to-end |

#### Paquete `regex` (30 tests)

| Test | Que Verifica |
|------|-------------|
| `TestNormalize_SimpleLiteral` | Literal `'a'` produce un solo atomo |
| `TestNormalize_ConcatInsertion` | `'a' 'b'` inserta concatenacion explicita |
| `TestNormalize_AlternationNoConcatBetweenPipeAndOperands` | `'a' \| 'b'` NO inserta concat |
| `TestNormalize_CharClassRange` | `[a-c]` produce `(a\|b\|c)` |
| `TestNormalize_CharClassMultipleRanges` | `[a-cA-C]` produce 6 alternativas |
| `TestNormalize_ComplementCharClass` | `[^ 'a' 'b']` excluye a y b del alfabeto |
| `TestNormalize_DoubleQuotedString` | `"abc"` produce `(a·b·c)` |
| `TestNormalize_SingleCharDoubleQuoted` | `"x"` produce un solo atomo |
| `TestNormalize_EmptyDoubleQuoted` | `""` produce 0 tokens |
| `TestNormalize_WildcardDot` | `.` produce 97 alternativas |
| `TestNormalize_WildcardUnderscore` | `_` produce misma salida que `.` |
| `TestNormalize_EscapeSequences` (4 sub-tests) | `'\n'`, `'\t'`, `'\\'`, `'\''` |
| `TestNormalize_KleeneStarNoConcatAfter` | `'a'* 'b'` -> `a * · b` |
| `TestNormalize_PlusAndOptional` | `'a'+ 'b'?` -> `a + · b ?` |
| `TestNormalize_ComplexPattern` | Patron FLOAT: `[0-9]+ '.' [0-9]+` |
| `TestBuildPostfix_SimpleConcat` | `a · b` -> postfix `a b · # ·` |
| `TestBuildPostfix_Alternation` | `a \| b` -> postfix `a b \| # ·` |
| `TestBuildPostfix_StarPrecedence` | `a \| b *` -> postfix `a b * \| # ·` |
| `TestBuildPostfix_PlusPrecedence` | `a · b +` -> postfix `a b + · # ·` |
| `TestBuildPostfix_GroupedAlternation` | `( a \| b ) · c` -> postfix correcto |
| `TestBuildPostfix_UnmatchedOpenParen` | Error en parentesis sin cerrar |
| `TestBuildPostfix_UnmatchedCloseParen` | Error en parentesis sin abrir |
| `TestBuildPostfix_DoubleUnary` | `a * +` -> postfix `a * + # ·` |
| `TestBuildPostfix_EndMarkerPresent` (4 sub-tests) | Todo postfix termina en `# ·` |
| `TestIntegration_NormalizeThenBuild` (13 sub-tests) | Round-trip Normalize -> BuildPostfix |
| `TestNormalize_UnclosedSingleQuote` | Error en comilla sin cerrar |
| `TestNormalize_UnclosedDoubleQuote` | Error en comilla doble sin cerrar |
| `TestNormalize_EmptyCharClass` | Error en clase vacia `[]` |
| `TestBuildAlphabet_Size` | Alfabeto tiene 97 caracteres |
| `TestTokensToString` | Serializacion para debug |

#### Paquete `dfa` (40 tests)

| Test | Que Verifica |
|------|-------------|
| `TestBuildTree_SingleChar` | Arbol de `'a'` es `cat(leaf(a), leaf(#))` |
| `TestBuildTree_Alternation` | Arbol de `'a'\|'b'` con nodo `or` |
| `TestBuildTree_Concatenation` | Arbol de `'a''b'` con nodo `cat` interno |
| `TestBuildTree_KleeneStar` | Arbol de `'a'*` con nodo `star` |
| `TestBuildTree_Plus` | Arbol de `'a'+` con nodo `plus` |
| `TestBuildTree_Optional` | Arbol de `'a'?` con nodo `opt` |
| `TestNullable_Leaf` | Hoja no es nullable |
| `TestNullable_Star` | Star siempre nullable |
| `TestNullable_Plus_NonNullableChild` | `plus(leaf)` no es nullable |
| `TestNullable_Plus_NullableChild` | `plus(opt(leaf))` SI es nullable (Dragon Book) |
| `TestNullable_Opt` | Optional siempre nullable |
| `TestNullable_Or` | Or: false\|\|false=false, false\|\|true=true |
| `TestNullable_Cat` | Cat: false&&false=false, true&&true=true |
| `TestNullable_NilNode` | Nodo nil no es nullable |
| `TestFirstPos_Leaf` | firstpos(leaf(1)) = {1} |
| `TestFirstPos_Or` | firstpos(or(1,2)) = {1,2} |
| `TestFirstPos_Cat_NonNullableLeft` | firstpos = {1} cuando left no nullable |
| `TestFirstPos_Cat_NullableLeft` | firstpos = {1,2} cuando left nullable |
| `TestLastPos_Leaf` | lastpos(leaf(1)) = {1} |
| `TestLastPos_Cat_NonNullableRight` | lastpos = {2} cuando right no nullable |
| `TestLastPos_Cat_NullableRight` | lastpos = {1,2} cuando right nullable |
| `TestFollowPos_SimpleConcat` | followpos(1) contiene 2 en `'a''b'` |
| `TestFollowPos_Star` | followpos(1) contiene 1 en `'a'*` (bucle) |
| `TestFollowPos_Plus` | followpos(1) contiene 1 en `'a'+` (bucle) |
| `TestFollowPos_Opt_NoLoop` | followpos(1) NO contiene 1 en `'a'?` |
| `TestBuildDFA_SingleChar` | DFA de `'a'` tiene 2 estados |
| `TestBuildDFA_Alternation` | DFA de `'a'\|'b'` con transiciones para ambos |
| `TestBuildDFA_KleeneStar` | DFA de `'a'*` con start aceptor |
| `TestBuildDFA_Digits` | DFA de `[0-9]+` con 10 transiciones y bucle |
| `TestBuildDFA_Concat` | DFA de `'a''b'` con 3 estados |
| `TestBuildDFA_BooleanTF` | DFA de `'T'\|'F'` con 2 estados |
| `TestMinimize_AlreadyMinimal` | DFA ya minimo no cambia |
| `TestMinimize_RedundantStates` | Merge de estados equivalentes (4 -> 3) |
| `TestMinimize_PreservesAccepting` | DFA minimizado sigue aceptando "42" |
| `TestMinimize_EmptyDFA` | DFA vacio y de 1 estado no crashean |
| `TestDFA_AcceptsCorrectStrings` | `[0-9]+` acepta "42", "0", rechaza "", "abc" |
| `TestDFA_BooleanAccepts` | `'T'\|'F'` acepta "T", "F", rechaza "TF", "" |
| `TestDFA_NilAccepts` | `"Nil"` acepta "Nil", rechaza "nil", "Ni", "Nill" |
| `TestDFA_OptionalAccepts` | `'a'?` acepta "", "a", rechaza "aa" |
| `TestDFA_StarAccepts` | `'a'*` acepta "", "a", "aaa", rechaza "b" |

#### Paquete `lexer` (20 tests)

| Test | Que Verifica |
|------|-------------|
| `TestReadSource_NormalFile` | Lectura y splitting por lineas |
| `TestReadSource_CRLFNormalization` | `\r\n` se normaliza a `\n` |
| `TestReadSource_LoneCR` | `\r` solo se normaliza a `\n` |
| `TestReadSource_EmptyFile` | Archivo vacio produce content="" y Lines=[""] |
| `TestReadSource_NonexistentFile` | Archivo inexistente produce error |
| `TestTokenize_SingleToken` | Un solo INT("42") |
| `TestTokenize_MultipleTokens` | INT + ID separados por whitespace |
| `TestTokenize_SkipWhitespace` | Whitespace se descarta |
| `TestTokenize_MaximalMunch` | "123abc" -> INT("123") + ID("abc") |
| `TestTokenize_PriorityDisambiguation` | "let" -> KEYWORD (no ID) |
| `TestTokenize_LongerMatchWins` | "letter" -> ID (no KEYWORD) |
| `TestTokenize_UnrecognizedChar` | "@" produce error lexico |
| `TestTokenize_ErrorRecovery` | "42@abc" -> INT + error + ID |
| `TestTokenize_LineTracking` | Tokens en linea correcta tras `\n` |
| `TestTokenize_EmptySource` | Fuente vacia produce 0 tokens y 0 errores |
| `TestTokenize_Expression` | "(1 + 2)" -> 5 tokens |
| `TestTokenize_SingleCharTokens` | "()" -> LPAREN + RPAREN |
| `TestTokenize_AllWhitespace` | Solo whitespace -> 0 tokens |
| `TestTokenize_ConsecutiveErrors` | "@#$" -> 3 errores, uno por caracter |
| `TestTokenize_MultiLineError` | Errores reportan linea correcta |

#### Paquete `generator` (8 tests)

| Test | Que Verifica |
|------|-------------|
| `TestGenerateSource_CreatesFile` | El archivo de salida se crea y no esta vacio |
| `TestGenerateSource_ValidGoCode` | El codigo generado compila exitosamente |
| `TestGenerateSource_ProducesCorrectOutput` | El lexer generado tokeniza "42 abc 123" correctamente |
| `TestGenerateSource_WithRealYal` | Pipeline completo con `lexer.yal` + `test.lisp` reales |
| `TestGenerateSource_EmptyEntries` | Lexer con 0 DFAs compila sin errores |
| `TestGenerateSource_SpecialCharsInTransitions` | Transiciones con `\n`, `\t` se serializan correctamente |
| `TestGenerateSource_OutputPathCreation` | Error al escribir en directorio inexistente |
| `TestGenerateSource_SkipTokensNotEmitted` | Tokens "skip" no aparecen en la salida |

#### Paquete `main` - Integration Tests (14 tests)

| Test | Que Verifica |
|------|-------------|
| `TestIntegration_RealYalFile` | Pipeline completo con archivos reales: 73 tokens, 10 tipos |
| `TestIntegration_SimpleCalculator` | Calculadora: `(1 + 2) * 3` -> 7 tokens |
| `TestIntegration_KeywordVsIdentifier` | "if" = KEYWORD, "iffy" = ID |
| `TestIntegration_FloatVsIntVsDot` | "3.14" = FLOAT, "42" = INT, ".5" = DOT + INT |
| `TestIntegration_StringWithEscapes` | `"hello\"there"` = un solo STRING |
| `TestIntegration_ErrorRecovery` | "@#$" produce 3 errores, tokens validos se recuperan |
| `TestIntegration_EmptySource` | Fuente vacia -> 0 tokens, 0 errores |
| `TestIntegration_MultilineLineTracking` | "foo\nbar\nbaz" -> lineas 1, 2, 3 |
| `TestIntegration_CommentsInYal` | Comentarios en `.yal` no afectan el resultado |
| `TestIntegration_BooleanAndNil` | "Nile" -> ID (maximal munch), "T" -> BOOLEAN |
| `TestIntegration_OutFlag` | Flag `-out` genera archivo compilable |
| `TestIntegration_TreeFlagStandalone` | Flag `-tree` genera `tree.dot` |
| `TestIntegration_OutAndSrcTogether` | Flags `-out` y `-src` juntos funcionan |
| `TestIntegration_GeneratedLexerMatchesSimulator` | Salida del lexer generado = salida del simulador |

### 7.3 Edge Cases Cubiertos

- **`nullable(n+)` con hijo nullable:** `(a?)+` debe ser nullable - verifica la correccion del Dragon Book.
- **Macros ciclicas:** `A=B, B=A` produce error explicito "cyclic macro reference".
- **Maximal munch:** `"letter"` gana sobre keyword `"let"` (match mas largo).
- **Prioridad:** `"let"` gana sobre ID `"let"` (mismo largo, keyword declarado primero).
- **Strings con escapes:** `"hello\"there"` es un solo token STRING.
- **`"Nil"` vs `"Nile"`:** maximal munch elige ID("Nile") sobre NIL("Nil").
- **Operadores multi-caracter:** `"<="` gana sobre `"<"` seguido de `"="` por maximal munch.
- **Error recovery:** caracter no reconocido produce error; el lexer avanza y continua tokenizando.
- **Line tracking:** tokens emitidos incluyen numero de linea correcto, incluso despues de `\n` en lexemas skip.
- **DFA vacio o de 1 estado:** no crashea el minimizador.
- **Parentesis desbalanceados:** Shunting-Yard produce error claro.
- **Clases de caracteres vacias:** error explicito.
- **Archivos con CRLF:** se normalizan a LF.
- **Lexer generado identico al simulador:** test que compara token por token la salida de ambos modos.

### 7.4 Como Ejecutar los Tests

```bash
# Ejecutar todos los tests
go test ./... -count=1

# Con output verbose
go test ./... -count=1 -v

# Solo un paquete
go test ./internal/dfa -count=1 -v

# Solo un test especifico
go test ./internal/dfa -run TestNullable_Plus_NullableChild -v
```

---

## 8. Uso del Sistema

### 8.1 Requisitos Previos

- **Go 1.26+** (o compatible)
- Sistema operativo: Linux, macOS o Windows con WSL

### 8.2 Compilacion

```bash
# Desde el directorio raiz del proyecto
go build -o genanalex main.go
```

### 8.3 Modo Simulador: `-yal` + `-src`

Tokeniza directamente un archivo fuente usando los DFAs construidos en memoria:

```bash
./genanalex -yal testdata/lexer.yal -src testdata/test.lisp
```

Salida esperada:
```
[*] Loading YALex specification: testdata/lexer.yal
[+] Parsed 3 macros and 30 rules
[*] Expanding macros...
[*] Building DFAs from regular expressions...
[*] Tokenizing source file: testdata/test.lisp

--- Tokenization Results ---
[1] DELIMITER    (
[1] KEYWORD      let
[1] DELIMITER    (
[1] DELIMITER    (
[1] ID           x
[1] INT          42
...
```

### 8.4 Modo Generador: `-yal` + `-out`

Genera un archivo Go standalone que implementa el analizador lexico:

```bash
./genanalex -yal testdata/lexer.yal -out generated_lexer.go
```

El archivo generado se compila y ejecuta independientemente:

```bash
# Compilar el lexer generado
go build -o my_lexer generated_lexer.go

# Ejecutarlo
./my_lexer -src testdata/test.lisp
```

### 8.5 Modo Visualizacion: `-yal` + `-tree`

Genera un archivo `tree.dot` con los arboles sintacticos en formato Graphviz:

```bash
./genanalex -yal testdata/lexer.yal -tree
```

Para visualizar:
```bash
dot -Tpng tree.dot -o tree.png
```

### 8.6 Combinaciones de Flags

Los flags pueden combinarse:

```bash
# Generar lexer + tokenizar + visualizar arboles
./genanalex -yal testdata/lexer.yal -src testdata/test.lisp -out lexer.go -tree
```

### 8.7 Ejemplo Completo Paso a Paso

1. **Crear la especificacion `mi_lexer.yal`:**

```yalex
let DIGIT = [0-9]

rule tokens =
  | [' ' '\t' '\n']+ { skip }
  | DIGIT+ { INT }
  | '+' { PLUS }
  | '*' { TIMES }
  | '(' { LPAREN }
  | ')' { RPAREN }
```

2. **Crear un archivo fuente `expr.txt`:**

```
(1 + 2) * 3
```

3. **Ejecutar en modo simulador:**

```bash
./genanalex -yal mi_lexer.yal -src expr.txt
```

4. **Ejecutar en modo generador:**

```bash
./genanalex -yal mi_lexer.yal -out calc_lexer.go
go build -o calc_lexer calc_lexer.go
./calc_lexer -src expr.txt
```

---

## 9. Decisiones de Diseno

### 9.1 Por que Construccion Directa de DFA?

| Criterio | Thompson + NFA + Subset | Construccion Directa (elegida) |
|----------|------------------------|-------------------------------|
| Fases | 3 (NFA -> Subset -> DFA) | 1 (followpos -> DFA) |
| Compacidad | Puede tener estados redundantes | Generalmente mas compacto |
| Uso del `#` | No necesario | Natural como marcador de aceptacion |
| Debug | NFA puede ser grande | Solo posiciones enteras |

La construccion directa reduce las transformaciones intermedias. El marcador `#` hace que los estados aceptores sean trivialmente identificables (cualquier estado que contenga la posicion de `#`).

### 9.2 Por que Separar Nullable/FirstPos/LastPos/FollowPos?

Cada funcion es **pura** - dada la misma entrada siempre produce la misma salida, sin efectos secundarios:

- Se puede testear cada modulo con casos unitarios independientes.
- `FirstPos` y `LastPos` dependen de `Nullable` como funcion invocada, no como estado global.
- `FirstPos` y `LastPos` pueden ejecutarse en cualquier orden.
- `FollowPos` es el unico que recorre el arbol completo; los anteriores son recursivos en el nodo.

### 9.3 Por que Pipeline Lineal?

Para un generador lexico de esta escala:
- Es suficiente (no hay retroalimentacion entre modulos).
- Hace el flujo de datos predecible y facil de depurar.
- Cada modulo puede ser reemplazado o mejorado sin afectar a los demas (bajo acoplamiento).

### 9.4 Por que Go?

- **Tipado estatico** - errores de tipo se detectan en compilacion.
- **Compilacion rapida** - ciclo de desarrollo agil.
- **Manejo nativo de Unicode** - el tipo `rune` simplifica el manejo de caracteres.
- **Templates** - el paquete `text/template` facilita la generacion de codigo.
- **Testing integrado** - `go test` sin dependencias externas.
- **Zero dependencias externas** - el proyecto usa solo la biblioteca estandar.

### 9.5 Por que RegexToken en Lugar de Strings?

Trabajar con strings crudos introduce ambiguedades fundamentales: el caracter `|` puede ser un operador de alternancia o un literal. Con `RegexToken`:

```go
type RegexToken struct {
    Kind TokKind // TokAtom | TokOp | TokOpen | TokClose
    Atom rune    // Valido cuando Kind == TokAtom
    Op   rune    // Valido cuando Kind == TokOp
}
```

Un `|` como operador tiene `Kind: TokOp, Op: '|'`, mientras que un `|` literal tiene `Kind: TokAtom, Atom: '|'`. No hay ambiguedad posible.

### 9.6 Por que Sentinelas `\x01` y `\x00`?

- **`\x01` (ConcatOp):** El operador de concatenacion no tiene representacion en la sintaxis regex original (es implicito). Usar `\x01` como operador interno evita colisiones con cualquier caracter del alfabeto de entrada.

- **`\x00` (EndMarker):** El marcador de fin `#` es un simbolo especial del metodo Aho-Sethi-Ullman. Usar `\x00` asegura que no colisione con ningun caracter del alfabeto (ASCII 32-126 + `\t` + `\r`).

### 9.7 Por que Memoizacion con `*bool`?

El tipo `bool` en Go tiene zero-value `false`. Si usaramos `bool` directamente como cache, no podriamos distinguir entre "no calculado" y "calculado como false". El puntero `*bool` resuelve esto: `nil` = no calculado, `&false` = calculado como false, `&true` = calculado como true.

Para `FirstPos` y `LastPos` se usa `map[int]bool` donde `nil` indica "no calculado" (maps tienen zero-value `nil` en Go).

### 9.8 Por que Prioridad por Orden para Keywords vs ID?

El patron `ID = [a-zA-Z][a-zA-Z0-9_]*` matchea "let", "if", "cond", "defun" con la misma longitud que sus respectivos DFAs de keywords. Con Maximal Munch, ambos DFAs aceptan en la misma posicion, asi que se usa la prioridad por orden como desempate.

La solucion es declarar las keywords **antes** que ID en el archivo `.yal`. No es "first-match-wins" - siempre se busca el match mas largo primero. Solo cuando hay empate de longitud, la prioridad decide.

---

## 10. Interfaces entre Modulos

| Modulo | Entrada | Salida |
|--------|---------|--------|
| **YalexParser** | Archivo `.yal` (texto) | `ParseResult{Macros: map[string]string, Rules: []TokenRule}` |
| **MacroExpander** | `macros`, `rules` | `[]TokenRule` con patrones expandidos |
| **RegexNormalizer** | `string` (patron expandido) | `[]RegexToken` con concat explicita |
| **RegexBuilder** | `[]RegexToken` normalizado | `[]RegexToken` en postfix aumentado |
| **SyntaxTreeBuilder** | `[]RegexToken` postfix | `*Node` (raiz), `map[int]rune` (posToSymbol) |
| **Nullable** | `*Node` | `bool` |
| **FirstPos** | `*Node` | `map[int]bool` |
| **LastPos** | `*Node` | `map[int]bool` |
| **FollowPos** | `*Node` (raiz) | `map[int]map[int]bool` |
| **DirectDFABuilder** | `*Node`, `map[int]rune`, `string` (tokenName) | `*DFA` |
| **DFAMinimizer** | `*DFA` | `*DFA` minimizado |
| **InputReader** | Ruta del archivo fuente | `*SourceFile{Content, Lines}` |
| **LexerSimulator** | `[]DFAEntry`, `*SourceFile` | `[]Token`, `[]string` (errores) |
| **Generator** | Ruta de salida, `[]DFAEntry` | Archivo `.go` en disco |

---

## 11. Limitaciones Conocidas

### 11.1 Funcionalidad No Soportada

- **Operador de diferencia (`#`):** Especificado en YALex (`regexp1 # regexp2`) pero no implementado. No es comun en definiciones lexicas practicas.
- **Unicode extendido:** Solo soporta ASCII 32-126 mas `\t` y `\r`. Caracteres acentuados, emojis u otros caracteres fuera de este rango no son reconocidos.
- **`eof`:** La accion especial `eof` de YALex no esta implementada.
- **Header/Trailer funcional:** Se parsean y eliminan correctamente, pero su contenido no se incluye en el codigo generado.
- **Multiples entrypoints:** Solo se soporta un `rule entrypoint` por archivo.

### 11.2 Limitaciones Tecnicas

- **Alfabeto fijo:** El alfabeto de wildcards (`.`, `_`) y complementos (`[^...]`) esta hardcodeado a 97 caracteres. No es configurable.
- **Generacion solo en Go:** El generador produce exclusivamente codigo Go. Para otros lenguajes se requeriria escribir templates adicionales.
- **Sin optimizacion de tablas:** Las tablas de transicion se serializan como maps de Go. Para DFAs grandes, una representacion como arreglo bidimensional seria mas eficiente.
- **Minimizacion O(|Q|^3 x |Sigma|):** Para automatas muy grandes, el algoritmo de Hopcroft (O(n log n)) seria preferible. En la practica, los DFAs de un analizador lexico tipico son lo suficientemente pequenos para que esto no sea un problema.

### 11.3 Posibles Mejoras Futuras

- Soporte para el operador de diferencia `#`.
- Generacion de codigo para Python, Java, C.
- Integracion con YAPar (parser generator).
- Soporte para Unicode completo (UTF-8).
- Optimizacion de tablas de transicion con compresion.
- Soporte para contadores de columna ademas de linea.
- Modo interactivo (REPL) para probar patrones.

---

## 12. Referencias

1. **Aho, A. V., Sethi, R., & Ullman, J. D.** (1986). *Compilers: Principles, Techniques, and Tools* (1st ed.). Addison-Wesley. (Dragon Book) - Seccion 3.9: Construccion directa de DFA, funciones nullable, firstpos, lastpos, followpos.

2. **Aho, A. V., Lam, M. S., Sethi, R., & Ullman, J. D.** (2006). *Compilers: Principles, Techniques, and Tools* (2nd ed.). Pearson. - Actualizacion del metodo con derivaciones formales de nullable(n+).

3. **Leroy, X.** *The OCaml manual: Chapter 12 - Lexer and parser generators (ocamllex, ocamlyacc)*. INRIA. - Especificacion del formato YALex/OCamllex.

4. **Hopcroft, J. E., Motwani, R., & Ullman, J. D.** (2006). *Introduction to Automata Theory, Languages, and Computation* (3rd ed.). Pearson. - Minimizacion de DFA, equivalencia de estados, Table-Filling algorithm.

5. **Go Programming Language Specification.** https://go.dev/ref/spec - Referencia del lenguaje Go y su biblioteca estandar (`text/template`, `os`, `flag`).
