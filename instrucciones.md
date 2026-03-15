# Proyecto: Generador de Analizadores Léxicos

## Descripción

La presente actividad consiste en la implementación de un Generador de Analizadores Léxicos. El sistema deberá recibir como entrada un archivo escrito en YALex y generar automáticamente un analizador léxico capaz de reconocer los tokens definidos o detectar errores léxicos.

Se tomará como base el trabajo realizado en actividades anteriores. Para la correcta comprensión de YALex, se deberá revisar el documento:

Consideraciones de YALex.pdf

El trabajo podrá realizarse en grupos de hasta tres integrantes, en grupos menores o de forma individual, manteniendo las reglas de continuidad establecidas.

---

## Objetivos

### Objetivo General

Implementar un Generador de Analizadores Léxicos.

### Objetivos Específicos

* Aplicar la teoría de Analizadores Léxicos en la construcción de una herramienta generadora de dichos componentes.
* Implementar una herramienta capaz de generar Analizadores Léxicos funcionales a partir de la definición regular de tokens.
* Utilizar la teoría de Autómatas Finitos para implementar analizadores léxicos en un lenguaje de programación a elección.

---

## Especificación del Funcionamiento del Generador

### Entrada

Un archivo que contiene la especificación del analizador léxico, escrito en YALex.

### Salida

* Un Árbol de Expresión graficado que represente la definición regular de los tokens.
* Un programa fuente que implemente el analizador léxico generado.

---

## Especificación del Funcionamiento del Analizador Léxico Generado

### Entrada

Un archivo de texto plano con cadenas de caracteres a analizar.

### Salida

La impresión en pantalla de:

* Los tokens identificados.
* Los mensajes de errores léxicos detectados.

---

## Entregables

* Código fuente completo en un repositorio de GitHub.
* Un video no listado de YouTube que incluya:

  * Ejecución del sistema.
  * Explicación del funcionamiento.
* Un documento PDF en Canvas con:

  * Enlaces requeridos.
  * Documentación técnica.
  * Explicación del diseño e implementación.

---

# Consideraciones de YALex

## Descripción

YALex (Yet Another Lex) se encuentra basado en el estilo de Lex y toma inspiración en la implementación de ocamllex. Su objetivo principal es la generación de Analizadores Léxicos a partir de una Definición Regular o conjunto de Expresiones Regulares.

La llamada esperada es:

```
yalex lexer.yal -o thelexer
```

Donde lexer.yal es un archivo en lenguaje YALex que genera un archivo thelexer en el lenguaje elegido, el cual implementa el analizador léxico. Este archivo puede utilizarse junto con el parser generado por YAPar.

---

## Estructura de un Archivo YALex

La estructura general es:

```
{ header }

let ident = regexp ...

rule entrypoint [arg1 ... argn] =
regexp { action }
| regexp { action }
| ...

{ trailer }
```

---

## Consideraciones

* Se permiten comentarios delimitados por (* y *).
* {header} y {trailer} son opcionales.
* Se pueden definir expresiones reutilizables con:

```
let ident = regexp
```

* Si múltiples expresiones coinciden, se elige el lexema más largo. En caso de empate, se prioriza por orden de definición.

---

## Expresiones Regulares

```
'c' — símbolo constante.
_ — cualquier símbolo.
"string" — constante tipo cadena.
[set] — conjunto de símbolos.
[^set] — complemento del conjunto.
regexp1 # regexp2 — diferencia.
* — cerradura de Kleene.
+ — cerradura positiva.
? — opcional.
Concatenación.
| — alternancia.
```

---

## Precedencia

```
#
*, +, ?
Concatenación
|
```

---

## Acciones

El segmento {action} contiene instrucciones en el lenguaje elegido. Estas se ejecutan al encontrar una concordancia con la expresión regular asociada.

---

## Ejemplo (Generando Código en Python)

```
(* ejemplo.yal *)

{
import myToken
}

rule gettoken =
[' ' '\t'] { return lexbuf }
| ['\n'] { return EOL }
| ['0'-'9']+ { return int(lxm) }
| '+' { return PLUS }
| '-' { return MINUS }
| '*' { return TIMES }
| '/' { return DIV }
| '(' { return LPAREN }
| ')' { return RPAREN }
| eof { raise('Fin de buffer') }
```

