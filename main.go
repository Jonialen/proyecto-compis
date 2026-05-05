// Package main es el punto de entrada para la herramienta genanalex.
//
// genanalex es un generador de analizadores lexicos basado en especificaciones YALex.
// Puede operar en dos modos:
//   - Motor lexico directo: lee un archivo fuente y lo tokeniza en tiempo real.
//   - Generador de lexer: produce un archivo fuente Go autonomo que implementa el lexer.
//
// Pipeline completo de procesamiento:
//  1. Parsear archivo .yal -> extraer macros y reglas de tokens
//  2. Expandir macros -> resolver referencias recursivas en los patrones
//  3. Para cada regla: normalizar regex -> convertir a postfix -> construir arbol sintactico
//  4. Calcular posiciones (followpos) -> construir DFA -> minimizar DFA
//  5. Opcionalmente: generar archivo .go con el lexer autonomo
//  6. Opcionalmente: tokenizar un archivo fuente directamente
package main

import (
	"flag"
	"fmt"
	"os"

	"genanalex/internal/dfa"
	"genanalex/internal/generator"
	"genanalex/internal/lexer"
	"genanalex/internal/regex"
	"genanalex/internal/yalex"
)

func main() {
	// --- Analisis de Argumentos de Linea de Comandos ---
	// Se definen las banderas (flags) que controlan el comportamiento de la herramienta:
	//   -yal: (obligatorio) ruta al archivo de especificacion lexica .yal
	//   -src: (opcional) archivo fuente a tokenizar directamente
	//   -out: (opcional) ruta del archivo .go generado con el lexer autonomo
	//   -tree: (opcional) genera un archivo tree.dot con los arboles sintacticos en formato Graphviz
	yalFile := flag.String("yal", "", "path to the .yal lexer specification file")
	srcFile := flag.String("src", "", "path to the source file to tokenize (optional if -out is used)")
	outFile := flag.String("out", "", "path to the output .go file for the generated lexer (optional)")
	genTree := flag.Bool("tree", false, "generate tree.dot Graphviz file for syntax trees")
	flag.Parse()

	// Validacion: se requiere al menos un archivo de especificacion YALex.
	if *yalFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: genanalex -yal <specification.yal> [-src <input_file>] [-out <output_lexer.go>] [-tree]")
		os.Exit(1)
	}

	// Validacion: se necesita al menos uno de -src (tokenizar), -out (generar), o -tree (visualizar).
	// Sin ninguno de estos, la herramienta no tiene trabajo util que realizar.
	if *srcFile == "" && *outFile == "" && !*genTree {
		fmt.Fprintln(os.Stderr, "Error: You must provide at least one of -src, -out, or -tree.")
		os.Exit(1)
	}

	// --- Paso 1: Parseo de la Especificacion YALex ---
	// Se lee y analiza el archivo .yal para extraer:
	//   - Macros: definiciones reutilizables de expresiones regulares (let IDENT = regex)
	//   - Reglas: pares patron-accion que definen los tokens del lenguaje (| patron { ACCION })
	fmt.Printf("[*] Loading YALex specification: %s\n", *yalFile)
	parseResult, err := yalex.ParseFile(*yalFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing .yal file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Parsed %d macros and %d rules\n", len(parseResult.Macros), len(parseResult.Rules))

	// --- Paso 2: Expansion de Macros ---
	// Las macros pueden referenciar a otras macros, por lo que se expanden recursivamente
	// en orden topologico. Cada referencia a macro en un patron se sustituye por su
	// definicion expandida, envuelta en parentesis para preservar la precedencia.
	fmt.Println("[*] Expanding macros...")
	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expanding macros: %v\n", err)
		os.Exit(1)
	}

	// --- Paso 3: Construccion de DFAs ---
	// Para cada regla expandida se ejecuta el pipeline completo de compilacion de regex:
	//   3.1: Normalizacion -> se insertan operadores de concatenacion explicitos
	//   3.2: Conversion a postfix -> algoritmo shunting-yard para notacion postfija
	//   3.3: Arbol sintactico -> se construye el arbol de sintaxis a partir del postfix
	//   3.4: DFA -> se construye el DFA usando el metodo de followpos, luego se minimiza
	fmt.Println("[*] Building DFAs from regular expressions...")
	var dfaEntries []lexer.DFAEntry // Coleccion de DFAs, uno por cada regla de token
	var dotContents []string        // Representaciones DOT de los arboles (para visualizacion)

	for _, rule := range expandedRules {
		// Paso 3.1: Normalizacion del patron regex.
		// Se insertan operadores de concatenacion explicitos (·) donde estan implicitos,
		// se procesan secuencias de escape, y se normalizan las clases de caracteres.
		normalized, err := regex.Normalize(rule.Pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error normalizing %q: %v\n", rule.Pattern, err)
			os.Exit(1)
		}

		// Paso 3.2: Conversion a notacion postfix (postfija).
		// Se usa el algoritmo shunting-yard para convertir la expresion infija
		// a postfix, respetando la precedencia de operadores: * > · > |
		postfix, err := regex.BuildPostfix(normalized)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building postfix for %q: %v\n", rule.Pattern, err)
			os.Exit(1)
		}

		// Paso 3.3: Construccion del arbol sintactico.
		// Se construye un arbol binario donde las hojas son simbolos del alfabeto
		// y los nodos internos son operadores (|, ·, *, +, ?).
		// Tambien se devuelve posToSymbol: mapeo de posiciones numericas a sus simbolos.
		root, posToSymbol, err := dfa.BuildTree(postfix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building syntax tree for %q: %v\n", rule.Pattern, err)
			os.Exit(1)
		}

		// Si se solicito visualizacion, se genera la representacion DOT del arbol.
		if *genTree {
			dot := dfa.ToDOT(root)
			dotContents = append(dotContents, fmt.Sprintf("// Rule %d: %s\n%s", rule.Priority, rule.Action, dot))
		}

		// Paso 3.4: Construccion y minimizacion del DFA.
		// BuildDFA usa el algoritmo de construccion directa (metodo de followpos):
		//   - Calcula nullable, firstpos, lastpos para cada nodo del arbol
		//   - Calcula followpos para cada posicion
		//   - Construye los estados del DFA a partir de conjuntos de posiciones
		// Minimize aplica el algoritmo de minimizacion de Hopcroft para reducir estados.
		builtDFA := dfa.BuildDFA(root, posToSymbol, rule.Action)
		minimizedDFA := dfa.Minimize(builtDFA)

		// Se agrega la entrada del DFA a la coleccion, incluyendo el nombre del token
		// y su prioridad (orden de aparicion en el .yal) para resolucion de conflictos.
		dfaEntries = append(dfaEntries, lexer.DFAEntry{
			DFA:       minimizedDFA,
			TokenName: rule.Action,
			Priority:  rule.Priority,
		})
	}

	// --- Paso 4: Visualizacion de Arboles (Opcional) ---
	// Si se uso la bandera -tree, se escribe un archivo tree.dot con todos los arboles
	// sintacticos en formato Graphviz DOT, util para depuracion y documentacion.
	if *genTree && len(dotContents) > 0 {
		dotContent := ""
		for _, d := range dotContents {
			dotContent += d + "\n"
		}
		if err := os.WriteFile("tree.dot", []byte(dotContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing tree.dot: %v\n", err)
		} else {
			fmt.Println("[+] Generated tree.dot for visualization")
		}
	}

	// --- Paso 5: Generacion de Lexer Autonomo (Opcional) ---
	// Si se proporciono -out, se genera un archivo fuente Go que contiene un lexer
	// completamente funcional con los DFAs embebidos. El archivo generado no necesita
	// dependencias externas y puede compilarse y ejecutarse independientemente.
	if *outFile != "" {
		fmt.Printf("[*] Generating standalone lexer: %s\n", *outFile)
		if err := generator.GenerateSource(*outFile, dfaEntries); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating lexer source: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[+] Lexer generation successful")
	}

	// --- Paso 6: Tokenizacion Directa (Opcional) ---
	// Si se proporciono -src, se lee el archivo fuente y se tokeniza utilizando los DFAs
	// construidos. Los tokens se imprimen con su numero de linea, tipo y lexema.
	// Si ocurren errores lexicos (caracteres no reconocidos), se reportan al final.
	if *srcFile != "" {
		fmt.Printf("[*] Tokenizing source file: %s\n", *srcFile)
		src, err := lexer.ReadSource(*srcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading source file: %v\n", err)
			os.Exit(1)
		}

		// Tokenize ejecuta todos los DFAs en paralelo sobre la entrada,
		// seleccionando el match mas largo (maximal munch) y usando la prioridad
		// para resolver empates entre reglas.
		tokens, errors := lexer.Tokenize(dfaEntries, src)

		fmt.Println("\n--- Tokenization Results ---")
		for _, tok := range tokens {
			fmt.Printf("[%d] %-12s %s\n", tok.Line, tok.Type, tok.Lexeme)
		}

		// Si hay errores lexicos, se reportan y se sale con codigo de error.
		if len(errors) > 0 {
			fmt.Println("\n--- Lexical Errors ---")
			for _, e := range errors {
				fmt.Println(e)
			}
			os.Exit(1)
		}
	}
}
