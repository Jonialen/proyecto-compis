
### Reconocer

Delimiadores: 

``` js
LPAREN      (
RPAREN      )
LBRACKET    [
RBRACKET    ]
LBRACE      {
RBRACE      }
COMMA       ,
DOT         .
COLON       :

```

Enteros:
`INT   [0-9]+`
``` js
INT   [0-9]+

```

Flotantes: 
`FLOAT   [0-9]+"."[0-9]+`

``` js
FLOAT   [0-9]+"."[0-9]+

```

Scaped Strings:
`STRING   \"([^"\\]|\\.)*\"`
``` js
STRING   \"([^"\\]|\\.)*\"

```

Booleanos
`BOOLEAN   T|F`
``` js
BOOLEAN   T|F

```

Nil

``` js
NIL   Nil

```

Ids
``` js
ID   [a-zA-Z]([a-zA-Z]0-9_])+

```

Palabras reservadas
``` js
If
defun
cond
let
```



Operadores

```js
<=
>=
+
-
*
/
=
<
>

```


comentarios
```js
;[^\n]*

```


---

<!-- v2: cambio -->
## Changelog v1 → v2

### Correcciones teóricas
1. **nullable(n+):** Cambiado de `false` a `nullable(n)` según el Dragon Book (n+ = n·n*)
2. **Shunting-Yard:** Operadores unarios sufijos (`*`, `+`, `?`) ahora se emiten directamente al output, no se apilan

### Correcciones de implementación
3. **MacroExpander:** Eliminada expansión optimista; siempre usa resolver recursivo con detección de ciclos
4. **YalexParser:** `splitByPipe` y `extractPatternAction` ahora manejan correctamente `'\''` (comilla simple escapada)
5. **Especificación .yal:** `| 'T' | 'F' { BOOLEAN }` cambiado a `| ('T' | 'F') { BOOLEAN }` (sintaxis YALex estándar); reglas multi-alternativa separadas en entradas individuales

### Nuevas secciones
6. **Implementación en Go:** Decisiones técnicas (RegexToken, sentinelas `\x01`/`\x00`, memoización)
7. **Suite de Tests:** 120+ tests con edge cases documentados

---

## Arquitectura del Generador


### Pipeline de módulos

![[diagrama1.png]]
![[Diagrama2.png]]
---

## Especificación de Módulos

### Módulo 1 - YalexParser

|               |                                                             |
| ------------- | ----------------------------------------------------------- |
| **Propósito** | Leer el archivo `.yal` y separar macros de reglas de tokens |
| **Entrada**   | Archivo `.yal` (texto plano)                                |
| **Salida**    | `macros: Dict[str, str]`, `rules: List[Tuple[str, str]]`    |
| **Recibe de** | Archivo en disco                                            |
| **Envía a**   | `MacroExpander`                                             |

**Estructuras internas:**
```
MacroDef  = { name: str, pattern: str }
TokenRule = { pattern: str, token_name: str }
```

- Distingue líneas de definición (`let DIGIT = [0-9]`) de reglas (`| pattern { ACTION }`)
- Ignora comentarios `(* ... *)`
- El orden de las reglas importa: **first match wins**
<!-- v2: cambio -->
- Manejo de header/trailer: elimina bloques `{ ... }` que aparecen antes de `rule` (header) y después de la última regla (trailer), conforme a la especificación YALex
- **Comillas simples escapadas:** `splitByPipe` y `extractPatternAction` manejan correctamente el literal `'\''` — cuando se detecta un `\` dentro del modo single-quote, se salta el siguiente carácter (es el carácter escapado) antes de buscar el cierre `'`

---

### Módulo 2 - MacroExpander

| | |
|---|---|
| **Propósito** | Sustituir referencias a macros por su expresión expandida |
| **Entrada** | `macros: Dict[str, str]`, `rules: List[Tuple[str, str]]` |
| **Salida** | `expanded_rules: List[Tuple[str, str]]` |
| **Recibe de** | `YalexParser` |
| **Envía a** | `RegexNormalizer` |

<!-- v2: cambio -->
- **Resolución recursiva con detección de ciclos:** el expander SIEMPRE usa un resolver recursivo (`expandOne`) que mantiene un conjunto `inStack` para detectar ciclos. No se usa una pasada optimista previa con macros ya expandidas — esa estrategia causaba que referencias a macros no resueltas se trataran silenciosamente como literales en lugar de activar el resolver recursivo.
- Cuando se expande un patrón, cada identificador se resuelve llamando al resolver directamente:
  - Si devuelve error "undefined macro", el identificador se trata como literal (no es una macro)
  - Si devuelve error "cyclic macro reference", se propaga el error
  - Si tiene éxito, se sustituye el identificador por `(valor_expandido)`
- Esto garantiza:
  - Macros cíclicas (A→B→A) producen un error explícito
  - Macros transitivas (A→B→C) se expanden completamente
  - Identificadores desconocidos se pasan como literales sin riesgo

---

### Módulo 3 - RegexNormalizer

| | |
|---|---|
| **Propósito** | Expandir clases `[a-z]` e insertar el operador de concatenación explícito `·` |
| **Entrada** | `str` - regex cruda |
| **Salida** | `str` - regex en infix con `·` explícito |
| **Recibe de** | `MacroExpander` |
| **Envía a** | `RegexBuilder` |

**Expansión de clases:**
```
[a-z]  →  (a|b|c|...|z)
[0-9]  →  (0|1|2|3|4|5|6|7|8|9)
[^"\n] →  todos los chars del alfabeto excepto " y \n
```

**Reglas de inserción de `·`** - se inserta entre posición `i` e `i+1` cuando:

| Izquierda (`i`)  | Derecha (`i+1`) |
|------------------|-----------------|
| símbolo literal  | símbolo literal |
| símbolo literal  | `(` |
| `)`, `*`, `+`, `?` | símbolo literal |
| `)`, `*`, `+`, `?` | `(` |

```
ab*c   →   a·b*·c
(a|b)c →   (a|b)·c
```

---

### Módulo 4 - RegexBuilder

| | |
|---|---|
| **Propósito** | Aumentar la regex con `#` y convertirla a postfix (Shunting Yard) |
| **Entrada** | `str` - regex normalizada en infix |
| **Salida** | `str` - postfix de la regex aumentada, e.g. `TF\|#·` |
| **Recibe de** | `RegexNormalizer` |
| **Envía a** | `SyntaxTreeBuilder` |

**Paso 1 - Aumentar:**
```
regex  →  (regex)·#
```
El `#` es el marcador de fin para la construcción directa de DFA.

<!-- v2: cambio -->
**Paso 2 - Shunting Yard:**

Tabla de precedencias:
```
Operador   Precedencia   Tipo
  *  +  ?       3        unario sufijo
  ·              2        binario
  |              1        binario
```

**Tratamiento de operadores unarios sufijos:** Los operadores `*`, `+` y `?` se emiten **directamente al output** sin pasar por la pila de operadores. Esto es el enfoque estándar para operadores unarios postfijos en Shunting-Yard: al encontrar un operador unario, ya tiene su operando a la izquierda en el output, así que se emite inmediatamente. Empujarlos a la pila de operadores causaría reordenamiento incorrecto cuando se apilan unarios (e.g. `a*+` debe producir `a * +` en postfix, significando `(a*)+`, no `a + *`).

Solo los operadores **binarios** (`·` y `|`) participan en la lógica de la pila (pop operadores de mayor o igual precedencia antes de push, asociatividad izquierda).

---

### Módulo 5 - SyntaxTreeBuilder

| | |
|---|---|
| **Propósito** | Construir árbol sintáctico del postfix y numerar posiciones de hojas |
| **Entrada** | `str` - regex en postfix |
| **Salida** | `root: Node`, `pos_to_symbol: Dict[int, str]` |
| **Recibe de** | `RegexBuilder` |
| **Envía a** | `Nullable`, `FirstPos`, `LastPos` |

**Estructura de nodo:**
```
Node {
    kind:   leaf | epsilon | cat | or | star | plus | opt
    symbol: string   -- solo para hoja
    pos:    int      -- solo para hoja (numeración global)
    left:   *Node
    right:  *Node
}
```

**Construcción:**
```
símbolo → push leaf(next_pos++, symbol)
'|'     → r=pop, l=pop, push or(l, r)
'·'     → r=pop, l=pop, push cat(l, r)
'*'/'+'/'?' → n=pop, push star/plus/opt(n)
```

---

### Módulo 6 - Nullable

|               |                                                                 |
| ------------- | --------------------------------------------------------------- |
| **Propósito** | Determinar si una sub-expresión puede generar la cadena vacía ε |
| **Entrada**   | `node: Node`                                                    |
| **Salida**    | `bool`                                                          |
| **Recibe de** | `SyntaxTreeBuilder`                                             |
| **Envía a**   | `FirstPos`, `LastPos`                                           |

<!-- v2: cambio -->
**Reglas (recurrente sobre el árbol):**
```
nullable(leaf ε)    = true
nullable(leaf c)    = false          (cualquier símbolo ≠ ε)
nullable(n1 | n2)   = nullable(n1) OR  nullable(n2)
nullable(n1 · n2)   = nullable(n1) AND nullable(n2)
nullable(n*)        = true           (cero o más)
nullable(n+)        = nullable(n)    (uno o más = n · n*)
nullable(n?)        = true           (cero o uno)
```

> **Nota sobre `nullable(n+)` — Dragon Book (Aho, Sethi, Ullman):** La definición correcta es `nullable(n+) = nullable(n)`, no `false` incondicionalmente. La derivación es directa: `n+ = n · n*`, por lo tanto `nullable(n+) = nullable(n · n*) = nullable(n) AND nullable(n*) = nullable(n) AND true = nullable(n)`. Esto importa para patrones como `(a?)+` donde la expresión interna ES nullable, haciendo que el plus también lo sea. Para la mayoría de patrones prácticos (como `[0-9]+`), `nullable(n)` es `false` y el resultado coincide con la regla simplificada original, pero la regla general es necesaria para corrección.

**Estructuras internas:** `cache: *bool` por nodo — memoización. El puntero permite distinguir "no calculado" (`nil`) de "calculado como `false`".

**Dependencias:** ninguna - es la base de la cadena de módulos de posición.

---

### Módulo 7 - FirstPos

|               |                                                                                                                          |
| ------------- | ------------------------------------------------------------------------------------------------------------------------ |
| **Propósito** | Calcular el conjunto de posiciones que pueden ser la **primera posición** de alguna cadena generada por la sub-expresión |
| **Entrada**   | `node: Node`, `nullable: Callable[[Node], bool]`                                                                         |
| **Salida**    | `Set[int]`                                                                                                               |
| **Recibe de** | `SyntaxTreeBuilder`, `Nullable`                                                                                          |
| **Envía a**   | `FollowPos`, `DirectDFABuilder` (como estado inicial)                                                                    |

**Reglas:**
```
firstpos(leaf ε)     = {}
firstpos(leaf c, i)  = {i}
firstpos(n1 | n2)    = firstpos(n1) ∪ firstpos(n2)
firstpos(n1 · n2)    = firstpos(n1) ∪ firstpos(n2)   si nullable(n1)
                     = firstpos(n1)                    si no nullable(n1)
firstpos(n*)         = firstpos(n)
firstpos(n+)         = firstpos(n)
firstpos(n?)         = firstpos(n)
```

**Estructuras internas:** `cache: map[int]bool` por nodo — memoización.

**Nota:** el estado inicial del DFA será `firstpos(root)`.

---

### Módulo 8 - LastPos

|               |                                                                                                                         |
| ------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Propósito** | Calcular el conjunto de posiciones que pueden ser la **última posición** de alguna cadena generada por la sub-expresión |
| **Entrada**   | `node: Node`, `nullable: Callable[[Node], bool]`                                                                        |
| **Salida**    | `Set[int]`                                                                                                              |
| **Recibe de** | `SyntaxTreeBuilder`, `Nullable`                                                                                         |
| **Envía a**   | `FollowPos`                                                                                                             |

**Estructuras internas:** `cache: map[int]bool` por nodo — memoización.

**Reglas (simétricas a FirstPos sobre `·`):**
```
lastpos(leaf ε)     = {}
lastpos(leaf c, i)  = {i}
lastpos(n1 | n2)    = lastpos(n1) ∪ lastpos(n2)
lastpos(n1 · n2)    = lastpos(n1) ∪ lastpos(n2)    si nullable(n2)
                    = lastpos(n2)                    si no nullable(n2)
lastpos(n*)         = lastpos(n)
lastpos(n+)         = lastpos(n)
lastpos(n?)         = lastpos(n)
```

---

### Módulo 9 - FollowPos

|               |                                                                                         |
| ------------- | --------------------------------------------------------------------------------------- |
| **Propósito** | Calcular qué posiciones pueden seguir a cada posición (necesario para construir el DFA) |
| **Entrada**   | `root: Node`, `firstpos: Callable`, `lastpos: Callable`                                 |
| **Salida**    | `followpos: Dict[int, Set[int]]`                                                        |
| **Recibe de** | `FirstPos`, `LastPos`                                                                   |
| **Envía a**   | `DirectDFABuilder`                                                                      |

**Reglas (solo dos casos generan followpos):**
```
nodo  n1 · n2:
  ∀ i ∈ lastpos(n1):  followpos(i) ∪= firstpos(n2)

nodo  n*  o  n+:
  ∀ i ∈ lastpos(n):   followpos(i) ∪= firstpos(n)
```

**Estructuras internas:** `table: map[int → Set[int]]` - indexada por posición de hoja.

Recorrido post-order sobre el árbol; los nodos `n?` y `|` y las hojas no generan entradas en followpos.

---

### Módulo 10 - DirectDFABuilder

|               |                                                                    |
| ------------- | ------------------------------------------------------------------ |
| **Propósito** | Construir DFA desde la tabla `followpos` (método Aho-Sethi-Ullman) |
| **Entrada**   | `followpos`, `pos_to_symbol`, `start = firstpos(root)`             |
| **Salida**    | `DFA = (Q, Σ, δ, q0, F)`                                           |
| **Recibe de** | `FollowPos`, `FirstPos`                                            |
| **Envía a**   | `DFAMinimizer`                                                     |

**Estructuras internas:**
```
DFA {
    states:      map[id → Set[int]]
    transitions: map[id → map[sym → id]]
    start:       id
    accepting:   Set[id]
    stateToken:  map[id → string]
}
```

Los estados aceptores son aquellos que contienen la posición del símbolo `#`.

---

### Módulo 11 - DFAMinimizer

|               |                                                                          |
| ------------- | ------------------------------------------------------------------------ |
| **Propósito** | Minimizar el DFA eliminando estados equivalentes (método de la pirámide) |
| **Entrada**   | `DFA = (Q, Σ, δ, q0, F)`                                                 |
| **Salida**    | `DFA minimizado`                                                         |
| **Recibe de** | `DirectDFABuilder`                                                       |
| **Envía a**   | `LexerSimulator`                                                         |

**Estructuras internas:**
```
tabla: map[(p, q) → bool]   -- true = par distinguible (marcado)
       triangular: solo pares con id(p) < id(q)

clases: UnionFind o map[estado → representante]
```

**Algoritmo - Método de la Pirámide (Table-Filling):**
```
Paso 1 - Construir tabla triangular:
  Para todo par (p, q) con p ≠ q → tabla[(p,q)] = false (no marcado)

Paso 2 - Inicialización:
  Para todo par (p, q):
    Si (p ∈ F) XOR (q ∈ F)  →  marcar tabla[(p,q)] = true
    [uno acepta y el otro no → claramente distinguibles]

Paso 3 - Propagación (repetir hasta convergencia):
  Para cada par (p, q) no marcado:
    Para cada símbolo a ∈ Σ:
      r = δ(p, a),  s = δ(q, a)
      Si r ≠ s  Y  tabla[(min(r,s), max(r,s))] == true:
        marcar tabla[(p,q)] = true
        break  ← ya no hace falta seguir con este par

Paso 4 - Construir clases de equivalencia:
  Pares no marcados → estados equivalentes
  Unir en una clase por cada grupo de equivalencia

Paso 5 - Construir DFA minimizado:
  Un estado por clase de equivalencia
  δ_min([p], a) = [δ(p, a)]
  q0_min = clase que contiene q0
  F_min  = clases que contienen algún estado de F
```

**Complejidad:** O(|Q|² × |Σ|) por pasada, O(|Q|²) pasadas en el peor caso -> O(|Q|³ × |Σ|) total.

---

### Módulo 12 - InputReader

| | |
|---|---|
| **Propósito** | Leer el archivo fuente a escanear y entregarlo como string al LexerSimulator |
| **Entrada** | Ruta del archivo fuente (`.lisp`, `.txt`, etc.) |
| **Salida** | `source: string`, `lines: []string` (para reportar número de línea en errores) |
| **Recibe de** | Archivo en disco (independiente del pipeline .yal) |
| **Envía a** | `LexerSimulator` |

**Estructuras internas:**
```
SourceFile {
    path:    string
    content: string       -- contenido completo
    lines:   []string     -- contenido dividido por \n (para error reporting)
}
```

**Responsabilidades:**
- Abrir y leer el archivo fuente
- Normalizar saltos de línea (`\r\n` → `\n`)
- Llevar un contador de línea actual para incluirlo en cada `Token`
- Reportar error si el archivo no existe o no se puede leer

---

### Módulo 13 - LexerSimulator

| | |
|---|---|
| **Propósito** | Tokenizar una cadena usando el DFA (maximal munch)  |
| **Entrada**   | `dfas: []DFAEntry`, `source: SourceFile`            |
| **Salida**    | `List[Token]` donde `Token = (tipo, lexema, línea)` |
| **Recibe de** | `DFAMinimizer` (los DFAs), `InputReader` (el fuente) |
| **Envía a**   | Parser / usuario                                    |

**Estructuras internas:**
```
Token         { type: string, lexeme: string, line: int }
DFAEntry      { dfa: DFA, tokenName: string, priority: int }  ← un por regla del .yal
```

**Algoritmo - Maximal Munch + Prioridad como desempate:**
```
Para cada posición i en el input:

  1. Correr TODOS los DFAs en paralelo desde su estado inicial.

  2. Avanzar carácter a carácter (j = i, i+1, ...):
       - Para cada DFA activo: seguir la transición con input[j].
       - Si un DFA no tiene transición → desactivarlo.
       - Si el conjunto de DFAs activos llegó a estados aceptores → guardar:
             lastOKPos   = j+1
             lastOKMatch = { todos los DFAs que aceptan en este j }

  3. Cuando no quedan DFAs activos (o se llegó al fin del input):
       - Si lastOKPos == -1 → error léxico.
       - Si lastOKMatch tiene más de un DFA → DESEMPATE por prioridad:
             token = DFA con menor índice en las reglas del .yal
       - Emitir Token(token, input[i:lastOKPos])
       - i = lastOKPos  (avanzar al siguiente lexema)
```

**Ejemplo de desempate:**
```
Input: "defun"   - ambos DFAs (KEYWORD y ID) aceptan en pos 5
  KEYWORD tiene priority=6   (definido en línea 6 del .yal)
  ID      tiene priority=7   (definido en línea 7 del .yal)
  → gana KEYWORD  ✓

Input: "definition"  - solo ID acepta en pos 10 (KEYWORD se desactivó en pos 5)
  → ID("definition")  ✓
```

---

<!-- v2: cambio -->
## Especificación `.yal` 

```
(* Analizador léxico para Lisp/Scheme simplificado *)
(* Tokens segun Laboratorio 2 - Compiladores UVG     *)

(* Macros *)
let DIGIT   = [0-9]
let LETTER  = [a-zA-Z]
let ALNUM   = [a-zA-Z0-9_]

(* ── Reglas (maximal munch; orden = prioridad en empate de longitud) ── *)
rule tokens =

  (* Espacios en blanco — se descartan *)
  | [' ' '\t' '\n' '\r']               { skip }

  (* Comentarios de linea: ; hasta fin de linea *)
  | ';' [^ '\n']*                      { COMMENT }

  (* Literales especiales — antes de ID para ganar por prioridad *)
  | "Nil"                              { NIL }
  | ('T' | 'F')                        { BOOLEAN }

  (* Numeros — FLOAT antes que INT *)
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

  (* Keywords — ANTES que ID para ganar empate de longitud *)
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

> **Nota sobre sintaxis YALex estándar:** En YALex (basado en OCamllex), cada `|` al nivel de regla introduce un nuevo par patrón-acción. Escribir `| 'T' | 'F' { BOOLEAN }` es ambiguo — crea dos reglas: `'T'` sin acción (error de sintaxis) y `'F' { BOOLEAN }`. Por ello, las alternativas dentro de un mismo patrón se agrupan con paréntesis: `('T' | 'F')` es un solo patrón que matchea T o F, con una única acción. Las reglas multi-alternativa que comparten acción (como operadores, keywords, delimitadores) se separan en entradas individuales con su propia acción `{ TOKEN }`, lo que además permite que el parser genere un `DFAEntry` independiente por cada una.

> **Nota:** las keywords (`If`, `defun`, `cond`, `let`) se declaran antes que `ID` para que el LexerSimulator las priorice por orden de reglas al usar maximal munch.

---

## Justificación de Decisiones Arquitectónicas

### ¿Por qué construcción directa de DFA y no Thompson + Subset?

| Criterio           | Thompson + NFA + Subset         | Construcción Directa (elegida)         |
| ------------------ | ------------------------------- | -------------------------------------- |
| Fases              | 3 (NFA → Subset → DFA)          | 1 (followpos → DFA)                    |
| Compacidad del DFA | Puede tener estados redundantes | Generalmente más compacto              |
| Uso del `#`        | No necesario                    | Natural como marcador de aceptación    |
| Facilidad de debug | NFA puede ser grande            | Solo se trabaja con posiciones enteras |

Se elige la construcción directa porque reduce las transformaciones intermedias y el marcador `#` hace que los estados aceptores sean trivialmente identificables (cualquier estado que contenga la posición de `#`).

### ¿Por qué separar Nullable, FirstPos, LastPos, FollowPos?

Cada función es pura - dada la misma entrada siempre produce la misma salida, sin efectos secundarios. Esto permite:
- Testear cada módulo con casos unitarios independientes
- `FirstPos` y `LastPos` dependen de `Nullable` como parámetro inyectado, no como global
- `FirstPos` y `LastPos` pueden ejecutarse en cualquier orden (no se bloquean entre sí)
- `FollowPos` es el único que necesita recorrer el árbol completo; los anteriores son recursivos en el nodo

### ¿Por qué prioridad por orden para keywords vs ID?

El patrón `ID` también matchea `defun`, `let`, `If`, `cond` con la misma longitud. Con maximal munch ambos DFAs (KEYWORD y ID) aceptan en la misma posición, entonces se usa la prioridad por orden de reglas como desempate: las keywords se declaran antes que ID en el `.yal`, por lo que ganan el empate. No es first-match-wins  primero siempre se busca el match más largo.

### ¿Por qué pipeline lineal y no grafo de dependencias?

Para un generador léxico de esta escala, un pipeline lineal:
- Es suficiente (no hay retroalimentación entre módulos)
- Hace el flujo de datos predecible y fácil de depurar
- Cada módulo puede ser reemplazado o mejorado sin afectar a los demás (bajo acoplamiento)

---

<!-- v2: cambio -->
## Implementación en Go — Decisiones de implementación

### Lenguaje y módulo

Implementado en **Go** (módulo `genanalex`). Se eligió Go por su tipado estático, compilación rápida y manejo nativo de Unicode (`rune`).

### Representación interna de tokens regex

Se usa `[]RegexToken` en lugar de strings crudos. La estructura:
```go
type RegexToken struct {
    Kind TokKind  // TokAtom | TokOp | TokOpen | TokClose
    Atom rune     // válido cuando Kind == TokAtom
    Op   rune     // válido cuando Kind == TokOp
}
```
Esto permite distinguir sin ambigüedad un `|` como operador (`TokOp`) de un `|` como literal (`TokAtom`), un problema frecuente cuando se trabaja con strings crudos donde el carácter pipe aparece tanto en el patrón como en la sintaxis regex.

### Sentinelas internos

- **Concatenación:** usa `\x01` (`ConcatOp`) como operador interno. Se inserta en la fase de normalización como token `TokOp` con `Op = '\x01'`.
- **Marcador de fin:** `#` usa `\x00` (`EndMarker`) como átomo. Esto evita colisiones con caracteres del alfabeto de entrada (ASCII 32-126 + `\t` + `\r`).

### Memoización

Nullable, FirstPos y LastPos usan caches almacenados directamente en los nodos del árbol:
- **Nullable:** `nullableCache *bool` — el puntero es necesario para distinguir "no calculado" (`nil`) de "calculado como `false`"
- **FirstPos/LastPos:** `firstPosCache map[int]bool` / `lastPosCache map[int]bool` — `nil` indica "no calculado"

### CLI

```
genanalex -yal lexer.yal -src input.lisp [-tree]
```

| Flag    | Descripción                                      |
|---------|--------------------------------------------------|
| `-yal`  | Archivo de especificación léxica (.yal)           |
| `-src`  | Archivo fuente a tokenizar                        |
| `-tree` | Genera `tree.dot` (formato Graphviz DOT) del árbol sintáctico |

---

## Traza con Macros - `INT {DIGIT}+`

> Ejerce YalexParser y MacroExpander con expansión real de macro, y RegexNormalizer con clase de caracteres.

### Módulo 1 - YalexParser
```
Entrada (.yal):
  let DIGIT = [0-9]
  | DIGIT+   { INT }

Salida:
  macros = { "DIGIT": "[0-9]" }
  rules  = [ ("{DIGIT}+", "INT"), ... ]
```

### Módulo 2 - MacroExpander
```
Regla: "{DIGIT}+"
Busca DIGIT en macros → "[0-9]"
Sustituye → "[0-9]+"

Salida: [ ("[0-9]+", "INT"), ... ]
```

### Módulo 3 - RegexNormalizer
```
Entrada : [0-9]+
Expandir [0-9] → (0|1|2|3|4|5|6|7|8|9)
+ es sufijo unario → no requiere · adicional

Salida : (0|1|2|3|4|5|6|7|8|9)+
```

### Módulo 4 - RegexBuilder
```
Aumentar → ((0|1|2|3|4|5|6|7|8|9)+)·#

Shunting Yard (interior del grupo, abreviado):
  01|2|3|4|5|6|7|8|9|   ← postfix del grupo (0|1|...|9)
  + → sufijo             → 01|2|3|4|5|6|7|8|9|+
  ·# →                   → 01|2|3|4|5|6|7|8|9|+#·

Salida postfix: 01|2|3|4|5|6|7|8|9|+#·
```

### Módulo 5 - SyntaxTreeBuilder
```
Posiciones: 0→1, 1→2, 2→3, ..., 9→10, #→11

Árbol:
        · (cat)
       / \
      +   #(11)
      |
  or-tree(1..10)

pos_to_symbol: { 1:'0', 2:'1', ..., 10:'9', 11:'#' }
```

### Módulo 6 - Nullable
```
or-tree → false   plus → false   # → false   cat → false
```
<!-- v2: cambio -->
> **Nota v2:** Aquí `nullable(plus) = nullable(or-tree) = false`, que coincide con la regla simplificada anterior `nullable(n+) = false`. La corrección `nullable(n+) = nullable(n)` no afecta esta traza porque la expresión interna (dígitos) no es nullable. Un caso donde SÍ importa: `(a?)+` → `nullable(opt) = true` → `nullable(plus) = true`.

### Módulo 7 - FirstPos
```
or-tree → {1..10}   plus → {1..10}   # → {11}
cat: nullable(plus)=false → firstpos(plus) = {1..10}

firstpos(root) = {1..10}
```

### Módulo 8 - LastPos
```
or-tree → {1..10}   plus → {1..10}   # → {11}
cat: nullable(#)=false → lastpos(#) = {11}

lastpos(root) = {11}
```

### Módulo 9 - FollowPos
```
Nodo plus:  ∀i ∈ lastpos(plus)={1..10}: followpos(i) ∪= firstpos(plus)={1..10}
Nodo cat:   ∀i ∈ lastpos(plus)={1..10}: followpos(i) ∪= firstpos(#)={11}

  pos 1..10 (dígitos) → {1,2,3,4,5,6,7,8,9,10,11}
  pos 11    (#)       → {}
```

### Módulo 10 - DirectDFABuilder
```
Estado A = {1..10}
  sym '0'..'9' → followpos = {1..10, 11} → Estado B={1..10,11}

Estado B = {1..10, 11}  ← contiene pos 11 = ACEPTOR (INT)
  sym '0'..'9' → followpos = {1..10, 11} → Estado B (bucle)

DFA:
          '0'..'9'       '0'..'9'
  → A ──────────► B ◄──────────┘  (B es aceptor: INT)

Maximal munch: B sigue consumiendo dígitos → reconoce el número completo.
```

### Módulo 11 - DFAMinimizer
```
Estados: A (no aceptor), B (aceptor)
Tabla triangular - único par: (A, B)

Paso 2 - Inicialización:
  A ∉ F, B ∈ F  →  MARCAR (A, B)

Paso 3 - Propagación:
  No hay pares sin marcar → termina.

Paso 4 - Clases de equivalencia:
  Ningún par sin marcar → no hay estados equivalentes.

DFA ya es mínimo. Sin cambios.
```

### Módulo 13 - LexerSimulator
```
Input: "42"
  A →'4'→ B (ok, lastOK=1)
  B →'2'→ B (ok, lastOK=2)
  fin → emit INT("42")

Input: "007"
  A →'0'→ B → '0'→ B → '7'→ B → fin → emit INT("007")
```

---

## Traza Completa - `BOOLEAN T|F`

### Módulo 1 - YalexParser
```
Salida → macros={},  rules=[("T|F", "BOOLEAN")]
```

### Módulo 2 - MacroExpander
```
Sin macros.  Salida → [("T|F", "BOOLEAN")]
```

### Módulo 3 - RegexNormalizer
```
Entrada : T|F
Sin clases [...] que expandir.
Inserción de ·: el | ya separa T y F, no hay concat implícita.
Salida  : T|F
```

### Módulo 4 - RegexBuilder
```
Aumentar  →  (T|F)·#

Shunting Yard:
 Token  Stack     Output
  (     (
  T     (         T
  |     ( |       T
  F     ( |       T F
  )               T F |
  ·     ·         T F |
  #     ·         T F | #
 fin              T F | # ·

Salida postfix : TF|#·
```

### Módulo 5 - SyntaxTreeBuilder
```
Postfix: T F | # ·

Stack paso a paso:
  T      →  [leaf(1,'T')]
  F      →  [leaf(1,'T'), leaf(2,'F')]
  |      →  [or(leaf(1,'T'), leaf(2,'F'))]
  #      →  [or(...), leaf(3,'#')]
  ·      →  [cat(or(leaf(1,'T'),leaf(2,'F')), leaf(3,'#'))]

Árbol:
        · (cat)
       / \
      |   # (pos=3)
     / \
  T(1)  F(2)

pos_to_symbol: { 1:'T', 2:'F', 3:'#' }
```

### Módulo 6 - Nullable
```
Árbol:  cat( or(T(1), F(2)), #(3) )

Nodo         Regla aplicada                   nullable?
──────────────────────────────────────────────────────
leaf T(1)    leaf(c) → false                  false
leaf F(2)    leaf(c) → false                  false
or           nullable(T) OR nullable(F)       false
leaf #(3)    leaf(c) → false                  false
cat (root)   nullable(or) AND nullable(#)     false
```

### Módulo 7 - FirstPos
```
Nodo         Regla aplicada                              firstpos
─────────────────────────────────────────────────────────────────
leaf T(1)    leaf(c,i) → {i}                            {1}
leaf F(2)    leaf(c,i) → {i}                            {2}
or           fp(T) ∪ fp(F)                              {1,2}
leaf #(3)    leaf(c,i) → {i}                            {3}
cat (root)   nullable(or)=false → fp(or)                {1,2}

firstpos(root) = {1,2}   ← estado inicial del DFA
```

### Módulo 8 - LastPos
```
Nodo         Regla aplicada                              lastpos
────────────────────────────────────────────────────────────────
leaf T(1)    leaf(c,i) → {i}                            {1}
leaf F(2)    leaf(c,i) → {i}                            {2}
or           lp(T) ∪ lp(F)                              {1,2}
leaf #(3)    leaf(c,i) → {i}                            {3}
cat (root)   nullable(#)=false → lp(#)                  {3}
```

### Módulo 9 - FollowPos
```
Recorrido post-order buscando nodos · y *:

  Nodo cat:  ∀i ∈ lastpos(or)={1,2}: followpos(i) ∪= firstpos(#)={3}
    followpos(1) = {3}
    followpos(2) = {3}
    followpos(3) = {}    ← # no tiene sucesor

Tabla final:
  pos 1 (T)  →  {3}
  pos 2 (F)  →  {3}
  pos 3 (#)  →  {}
```

### Módulo 10 - DirectDFABuilder
```
Estado inicial: A = {1,2}

Estado A = {1,2}:
  sym 'T' → posiciones con 'T': {1} → followpos(1) = {3} → Estado B={3}
  sym 'F' → posiciones con 'F': {2} → followpos(2) = {3} → Estado B={3}

Estado B = {3}:
  pos 3 tiene '#' → estado ACEPTOR
  sym '#' → followpos(3) = {} → estado muerto

DFA:
       T,F
  → A ────► B (aceptor: BOOLEAN)

δ(A,'T')=B,  δ(A,'F')=B,  q0=A,  F={B}
```

### Módulo 11 - DFAMinimizer
```
Estados: { A, B }    F = { B }
Tabla triangular (único par):

      B
  A [ ? ]

Paso 2 - Inicialización:
  (A, B): A ∉ F, B ∈ F  →  MARCAR ✗

      B
  A [ ✗ ]

Paso 3 - Propagación:
  No hay pares sin marcar → termina.

Conclusión: DFA ya es mínimo, 2 estados. Sin cambios.
```

### Módulo 13 - LexerSimulator
```
Input: "(T defun F)"

 i   DFA activo  char  estado  acepta?  acción
─────────────────────────────────────────────────
 0   DELIM       (     B       sí       LPAREN("(")
 1   BOOLEAN     T     B       sí       BOOLEAN("T")
 2   -           ' '   -       -        skip
 3   KEYWORD     d,e,  ...     sí       KEYWORD("defun")
      ...       f,u,n
 9   -           ' '   -       -        skip
10   BOOLEAN     F     B       sí       BOOLEAN("F")
11   DELIM       )     B       sí       RPAREN(")")

Salida:
  [ LPAREN("("), BOOLEAN("T"), KEYWORD("defun"), BOOLEAN("F"), RPAREN(")") ]
```

---

---

## Traza Compleja - `NIL Nil`

> Muestra concatenación explícita, árbol profundo y DFA lineal. Elegida porque es la forma más compacta del lenguaje que involucra concatenación de múltiples símbolos.

### Módulo 1 - YalexParser
```
Salida → macros={},  rules=[("Nil", "NIL")]
```

### Módulo 2 - MacroExpander
```
Sin macros.  Salida → [("Nil", "NIL")]
```

### Módulo 3 - RegexNormalizer
```
Entrada : Nil
Sin clases [...] que expandir.

Inserción de ·:
  N → i  (símbolo seguido de símbolo) → insertar ·
  i → l  (símbolo seguido de símbolo) → insertar ·

Salida : N·i·l
```

### Módulo 4 - RegexBuilder
```
Aumentar  →  (N·i·l)·#

Shunting Yard sobre (N·i·l)·#:
 Token  Stack     Output
  (     (
  N     (          N
  ·     ( ·        N
  i     ( ·        N i
  ·     ( ·        N i ·      ← · pop (misma prec, izq-asoc) antes de push nuevo ·
  l     ( ·        N i · l
  )                N i · l ·  ← pop ·, descartar (
  ·     ·          N i · l ·
  #     ·          N i · l · #
 fin               N i · l · # ·

Salida postfix : Ni·l·#·
```

### Módulo 5 - SyntaxTreeBuilder
```
Postfix: N i · l · # ·

Stack paso a paso:
  N  →  [leaf(1,'N')]
  i  →  [leaf(1,'N'), leaf(2,'i')]
  ·  →  [cat(leaf(1,'N'), leaf(2,'i'))]
  l  →  [cat(N,i),  leaf(3,'l')]
  ·  →  [cat(cat(N,i), leaf(3,'l'))]
  #  →  [cat(cat(N,i),l),  leaf(4,'#')]
  ·  →  [cat(cat(cat(N,i),l), leaf(4,'#'))]

Árbol:
            · (cat4)
           / \
        · (cat3)  # (pos=4)
        / \
     · (cat2)  l (pos=3)
     / \
  N(1)  i(2)

pos_to_symbol: { 1:'N', 2:'i', 3:'l', 4:'#' }
```

### Módulo 6 - Nullable
```
Nodo    Regla                                nullable?
──────────────────────────────────────────────────────
N(1)    leaf(c) → false                      false
i(2)    leaf(c) → false                      false
cat2    nullable(N) AND nullable(i)          false
l(3)    leaf(c) → false                      false
cat3    nullable(cat2) AND nullable(l)       false
#(4)    leaf(c) → false                      false
cat4    nullable(cat3) AND nullable(#)       false
```

### Módulo 7 - FirstPos
```
Nodo    Regla                                         firstpos
──────────────────────────────────────────────────────────────
N(1)    leaf(c,i) → {i}                              {1}
i(2)    leaf(c,i) → {i}                              {2}
cat2    nullable(N)=false → fp(N)                    {1}
l(3)    leaf(c,i) → {i}                              {3}
cat3    nullable(cat2)=false → fp(cat2)              {1}
#(4)    leaf(c,i) → {i}                              {4}
cat4    nullable(cat3)=false → fp(cat3)              {1}

firstpos(root) = {1}   ← estado inicial del DFA
```

### Módulo 8 - LastPos
```
Nodo    Regla                                         lastpos
─────────────────────────────────────────────────────────────
N(1)    leaf(c,i) → {i}                              {1}
i(2)    leaf(c,i) → {i}                              {2}
cat2    nullable(i)=false → lp(i)                    {2}
l(3)    leaf(c,i) → {i}                              {3}
cat3    nullable(l)=false → lp(l)                    {3}
#(4)    leaf(c,i) → {i}                              {4}
cat4    nullable(#)=false → lp(#)                    {4}

lastpos(root) = {4}
```

### Módulo 9 - FollowPos
```
Recorrido post-order - solo nodos · generan entradas:

  cat2 (N·i):   ∀i ∈ lastpos(N)={1}:    followpos(1) ∪= firstpos(i)={2}
  cat3 (·l):    ∀i ∈ lastpos(cat2)={2}: followpos(2) ∪= firstpos(l)={3}
  cat4 (·#):    ∀i ∈ lastpos(cat3)={3}: followpos(3) ∪= firstpos(#)={4}

Tabla final:
  pos 1 (N)  →  {2}
  pos 2 (i)  →  {3}
  pos 3 (l)  →  {4}
  pos 4 (#)  →  {}
```

### Módulo 10 - DirectDFABuilder
```
Estado inicial: A = {1}

A = {1}:  sym 'N' → pos con 'N': {1} → followpos(1)={2} → Estado B={2}
B = {2}:  sym 'i' → pos con 'i': {2} → followpos(2)={3} → Estado C={3}
C = {3}:  sym 'l' → pos con 'l': {3} → followpos(3)={4} → Estado D={4}
D = {4}:  contiene pos de '#' → ACEPTOR

DFA:
       N        i        l
  → A ────► B ────► C ────► D (aceptor: NIL)

δ: A→B por 'N',  B→C por 'i',  C→D por 'l'
q0=A,  F={D}
```

### Módulo 11 - DFAMinimizer
```
Estados: { A, B, C, D }    F = { D }
Pares (6 en total):

Tabla triangular inicial:
        B    C    D
   A  [ ]  [ ]  [ ]
   B       [ ]  [ ]
   C            [ ]

Paso 2 - Inicialización (un estado ∈ F, el otro ∉ F):
  (A,D): A∉F, D∈F  →  MARCAR ✗
  (B,D): B∉F, D∈F  →  MARCAR ✗
  (C,D): C∉F, D∈F  →  MARCAR ✗
  (A,B), (A,C), (B,C): ninguno mezcla F y no-F → sin marcar aún

        B    C    D
   A  [ ]  [ ]  [✗]
   B       [ ]  [✗]
   C            [✗]

Paso 3 - Propagación:

  Par (A,B): δ(A,'N')=B, δ(B,'N')=∅  →  B tiene transición, A no → distinguibles  →  MARCAR ✗
  Par (A,C): δ(A,'i')=∅, δ(C,'i')=∅ →  ambos sin transición en 'i'
             δ(A,'N')=B, δ(C,'N')=∅  →  B existe, ∅ no → distinguibles  →  MARCAR ✗
  Par (B,C): δ(B,'i')=C, δ(C,'i')=∅ →  C existe, ∅ no → distinguibles  →  MARCAR ✗

        B    C    D
   A  [✗]  [✗]  [✗]
   B       [✗]  [✗]
   C            [✗]

Paso 4: Todos los pares marcados → ningún estado equivalente.

Conclusión: DFA ya es mínimo, 4 estados. Sin cambios.
```

### Módulo 13 - LexerSimulator
```
Input: "(let ((x 42)) (+ x 1.5))"

Aplicando maximal munch con el conjunto de DFAs:

 pos  lexema   DFA que acepta  token
────────────────────────────────────────
  0   (        DELIM           LPAREN
  1   let      KEYWORD         KEYWORD
  4   (espacio)                skip
  5   (        DELIM           LPAREN
  6   (        DELIM           LPAREN
  7   x        ID              ID
  8   (espacio)                skip
  9   42       INT             INT
 11   )        DELIM           RPAREN
 12   )        DELIM           RPAREN
 13   (espacio)                skip
 14   (        DELIM           LPAREN
 15   +        OPERATOR        OPERATOR
 16   (espacio)                skip
 17   x        ID              ID
 18   (espacio)                skip
 19   1.5      FLOAT           FLOAT
 22   )        DELIM           RPAREN
 23   )        DELIM           RPAREN

Salida:
  [ LPAREN, KEYWORD("let"), LPAREN, LPAREN, ID("x"),
    INT("42"), RPAREN, RPAREN, LPAREN, OPERATOR("+"),
    ID("x"), FLOAT("1.5"), RPAREN, RPAREN ]
```

---

<!-- v2: cambio -->
## Suite de Tests

El proyecto cuenta con **120+ tests** distribuidos en 5 paquetes, cubriendo desde funciones unitarias hasta el pipeline de integración completo.

| Paquete | Archivo | Tests | Cobertura |
|---------|---------|-------|-----------|
| `yalex` | `yalex_test.go` | 26 | Parser: comentarios, header/trailer, multi-pipe, escapes. Expander: macros transitivos, cíclicos, indefinidos, `_` passthrough |
| `regex` | `regex_test.go` | 34 | Normalizer: clases `[a-z]`, complemento `[^...]`, wildcards `.`/`_`, escapes, inserción de concat. Builder: precedencia, paréntesis, unarios dobles |
| `dfa` | `dfa_test.go` | 40 | Árbol, nullable/firstpos/lastpos (incluyendo nullable(+) con hijo nullable), followpos, DFA building, minimización, acceptance end-to-end |
| `lexer` | `lexer_test.go` | 20 | Reader: normalización CRLF, archivos vacíos. Simulator: maximal munch, prioridad, skip, error recovery, line tracking |
| `main` | `integration_test.go` | 10 | Pipeline completo: .yal real, calculadora, keyword vs ID, float vs int, strings con escapes, recovery de errores, boolean/nil vs ID |

### Edge cases cubiertos por los tests

- **`nullable(n+)` con hijo nullable:** `(a?)+` debe ser nullable — verifica la corrección del Dragon Book
- **Macros cíclicas:** `A=B, B=A` debe producir error explícito "cyclic macro reference"
- **Maximal munch:** `"letter"` gana sobre keyword `"let"` (match más largo)
- **Prioridad:** `"let"` gana sobre ID `"let"` (mismo largo, keyword declarado primero)
- **Strings con escapes:** `"hello\"there"` es un solo token STRING
- **`"Nil"` vs `"Nile"`:** maximal munch elige ID("Nile") sobre NIL("Nil")
- **Operadores multi-carácter:** `"<="` gana sobre `"<"` seguido de `"="` por maximal munch
- **Error recovery:** carácter no reconocido produce error, el lexer avanza y continúa tokenizando
- **Line tracking:** tokens emitidos incluyen número de línea correcto, incluso después de `\n` en lexemas skip

---

## Interfaces

```
YalexParser       → macros: Dict[str,str],  rules: List[(str,str)]
MacroExpander     → expanded_rules: List[(str,str)]
RegexNormalizer   → normalized_infix: str              # por cada patrón
RegexBuilder      → postfix: str                       # por cada regex
SyntaxTreeBuilder → root: Node,  pos_to_sym: Dict[int,str]
Nullable          → Callable[[Node], bool]
FirstPos          → Callable[[Node], Set[int]]
LastPos           → Callable[[Node], Set[int]]
FollowPos         → followpos: Dict[int, Set[int]]
DirectDFABuilder  → DFA(Q, Σ, δ, q0, F)
DFAMinimizer      → DFA minimizado (Q', Σ, δ', q0', F')
InputReader       → SourceFile { content: string, lines: []string }
LexerSimulator    → List[Token(tipo, lexema, línea)]
```
