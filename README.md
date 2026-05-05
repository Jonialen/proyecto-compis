# Generador de Analizadores Léxicos (genAnaLex)

Este proyecto es un Generador de Analizadores Léxicos implementado en Go. Toma como entrada una especificación en lenguaje **YALex** y genera automáticamente un analizador léxico funcional basado en Autómatas Finitos Deterministas (DFA).

## Características

- **Soporte YALex:** Procesa macros (`let`), reglas (`rule`) y acciones asociadas.
- **Construcción de Árboles de Expresión:** Genera la representación visual de las expresiones regulares en formato Graphviz (`.dot`).
- **DFA Directo:** Implementa el algoritmo de construcción directa de DFA a partir del árbol de sintaxis.
- **Minimización de DFA:** Optimiza el autómata resultante para mayor eficiencia.
- **Generación de Código:** Produce un archivo fuente en Go autónomo que puede tokenizar archivos de texto.
- **Modo Simulación:** Permite tokenizar archivos directamente desde la herramienta principal sin necesidad de compilar el lexer generado.

## Requisitos

- [Go](https://golang.org/dl/) 1.18 o superior.
- [Graphviz](https://graphviz.org/download/) (opcional, para visualizar los árboles generados).

## Instalación

Clona el repositorio y compila el binario:

```bash
go build -o genanalex main.go
```

## Uso

La herramienta `genanalex` acepta varios parámetros para controlar el flujo de trabajo:

### 1. Generar Árboles de Expresión
Para visualizar la estructura interna de las expresiones regulares definidas en el `.yal`:

```bash
./genanalex -yal testdata/lexer.yal -tree
```
Esto generará un archivo `tree.dot` que puedes convertir a imagen con:
```bash
dot -Tpng tree.dot -o tree.png
```

### 2. Generar y Ejecutar el Analizador Léxico (Código Fuente)
Para generar el programa en Go que servirá como tu lexer personalizado:

```bash
./genanalex -yal testdata/lexer.yal -out mi_lexer.go
```

Luego puedes compilar y ejecutar este lexer generado:

```bash
go run mi_lexer.go -src testdata/test.lisp
```

### 3. Tokenización Directa (Modo Simulación)
Si deseas probar la especificación contra un archivo de entrada inmediatamente sin generar un archivo intermedio:

```bash
./genanalex -yal testdata/lexer.yal -src testdata/test.lisp
```

## Estructura del Proyecto

- `main.go`: Punto de entrada que coordina el flujo de generación y simulación.
- `internal/yalex/`: Parser y expansor de macros para el lenguaje YALex.
- `internal/regex/`: Normalizador de expresiones regulares y conversión a postfix.
- `internal/dfa/`: Construcción de árboles, cálculo de funciones (followpos) y construcción/minimización de DFA.
- `internal/generator/`: Motor de plantillas para la generación del código fuente del lexer.
- `internal/lexer/`: Simulador del lexer para pruebas rápidas y definiciones de estructuras de tokens.

## Ejemplo de Especificación (YALex)

```ocaml
let DIGIT = [0-9]
rule tokens =
  | [' ' '\t' '\n'] { skip }
  | DIGIT+          { INT }
  | ['a'-'z']+      { ID }
```

## Créditos
Desarrollado para el curso de Diseño de Lenguajes de Programación, Universidad del Valle de Guatemala.
