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

	"genanalex/internal/generator"
	"genanalex/internal/lexbuild"
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
	buildResult, err := lexbuild.CompileYALFile(*yalFile, *genTree)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling .yal file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Parsed %d macros and %d rules\n", len(buildResult.Macros), len(buildResult.Rules))

	// --- Paso 4: Visualizacion de Arboles (Opcional) ---
	// Si se uso la bandera -tree, se escribe un archivo tree.dot con todos los arboles
	// sintacticos en formato Graphviz DOT, util para depuracion y documentacion.
	if *genTree && len(buildResult.DOTContents) > 0 {
		dotContent := ""
		for _, d := range buildResult.DOTContents {
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
		if err := generator.GenerateSource(*outFile, buildResult.DFAEntries); err != nil {
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
		tokens, errors, err := lexbuild.TokenizeFile(buildResult.DFAEntries, *srcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading source file: %v\n", err)
			os.Exit(1)
		}

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
