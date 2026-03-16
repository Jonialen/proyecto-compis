// Package main is the entry point for the genanalex tool.
// It can operate as a lexer engine (direct tokenization) or as a lexer generator
// (producing a standalone Go source file).
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
	// --- Argument Parsing ---
	yalFile := flag.String("yal", "", "path to the .yal lexer specification file")
	srcFile := flag.String("src", "", "path to the source file to tokenize (optional if -out is used)")
	outFile := flag.String("out", "", "path to the output .go file for the generated lexer (optional)")
	genTree := flag.Bool("tree", false, "generate tree.dot Graphviz file for syntax trees")
	flag.Parse()

	// Validation: We need at least a YALex file.
	if *yalFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: genanalex -yal <specification.yal> [-src <input_file>] [-out <output_lexer.go>] [-tree]")
		os.Exit(1)
	}

	// Validation: We need either a source file to tokenize or an output file to generate.
	if *srcFile == "" && *outFile == "" {
		fmt.Fprintln(os.Stderr, "Error: You must provide either -src to tokenize a file or -out to generate a lexer.")
		os.Exit(1)
	}

	// --- Step 1: Parsing the YALex Specification ---
	fmt.Printf("[*] Loading YALex specification: %s\n", *yalFile)
	parseResult, err := yalex.ParseFile(*yalFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing .yal file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Parsed %d macros and %d rules\n", len(parseResult.Macros), len(parseResult.Rules))

	// --- Step 2: Macro Expansion ---
	fmt.Println("[*] Expanding macros...")
	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expanding macros: %v\n", err)
		os.Exit(1)
	}

	// --- Step 3: DFA Construction ---
	fmt.Println("[*] Building DFAs from regular expressions...")
	var dfaEntries []lexer.DFAEntry
	var dotContents []string

	for _, rule := range expandedRules {
		// 3.1: Normalization
		normalized, err := regex.Normalize(rule.Pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error normalizing %q: %v\n", rule.Pattern, err)
			os.Exit(1)
		}

		// 3.2: Postfix Conversion
		postfix, err := regex.BuildPostfix(normalized)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building postfix for %q: %v\n", rule.Pattern, err)
			os.Exit(1)
		}

		// 3.3: Syntax Tree Construction
		root, posToSymbol, err := dfa.BuildTree(postfix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building syntax tree for %q: %v\n", rule.Pattern, err)
			os.Exit(1)
		}

		if *genTree {
			dot := dfa.ToDOT(root)
			dotContents = append(dotContents, fmt.Sprintf("// Rule %d: %s\n%s", rule.Priority, rule.Action, dot))
		}

		// 3.4: DFA Construction & Minimization
		builtDFA := dfa.BuildDFA(root, posToSymbol, rule.Action)
		minimizedDFA := dfa.Minimize(builtDFA)

		dfaEntries = append(dfaEntries, lexer.DFAEntry{
			DFA:       minimizedDFA,
			TokenName: rule.Action,
			Priority:  rule.Priority,
		})
	}

	// --- Step 4: Tree Visualization (Optional) ---
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

	// --- Step 5: Lexer Generation (Optional) ---
	if *outFile != "" {
		fmt.Printf("[*] Generating standalone lexer: %s\n", *outFile)
		if err := generator.GenerateSource(*outFile, dfaEntries); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating lexer source: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[+] Lexer generation successful")
	}

	// --- Step 6: Direct Tokenization (Optional) ---
	if *srcFile != "" {
		fmt.Printf("[*] Tokenizing source file: %s\n", *srcFile)
		src, err := lexer.ReadSource(*srcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading source file: %v\n", err)
			os.Exit(1)
		}

		tokens, errors := lexer.Tokenize(dfaEntries, src)

		fmt.Println("\n--- Tokenization Results ---")
		for _, tok := range tokens {
			fmt.Printf("[%d] %-12s %s\n", tok.Line, tok.Type, tok.Lexeme)
		}

		if len(errors) > 0 {
			fmt.Println("\n--- Lexical Errors ---")
			for _, e := range errors {
				fmt.Println(e)
			}
			os.Exit(1)
		}
	}
}
