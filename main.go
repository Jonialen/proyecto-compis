package main

import (
	"flag"
	"fmt"
	"os"

	"genanalex/internal/dfa"
	"genanalex/internal/lexer"
	"genanalex/internal/regex"
	"genanalex/internal/yalex"
)

func main() {
	yalFile := flag.String("yal", "", "path to the .yal lexer specification file")
	srcFile := flag.String("src", "", "path to the source file to tokenize")
	genTree := flag.Bool("tree", false, "generate tree.dot Graphviz file")
	flag.Parse()

	if *yalFile == "" || *srcFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: genanalex -yal lexer.yal -src input.lisp [-tree]")
		os.Exit(1)
	}

	// Step 1: Parse the .yal file
	parseResult, err := yalex.ParseFile(*yalFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing .yal file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %d macros, %d rules\n", len(parseResult.Macros), len(parseResult.Rules))

	// Step 2: Expand macros
	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expanding macros: %v\n", err)
		os.Exit(1)
	}

	// Step 3: For each rule, build a DFA
	var dfaEntries []lexer.DFAEntry
	var dotContents []string

	for i, rule := range expandedRules {
		tokenName := rule.Action
		pattern := rule.Pattern

		fmt.Printf("Rule %d: %q → %s\n", i, pattern, tokenName)

		// Normalize the pattern → token slice
		normalized, err := regex.Normalize(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error normalizing rule %d (%q): %v\n", i, pattern, err)
			os.Exit(1)
		}

		// Build postfix token slice
		postfix, err := regex.BuildPostfix(normalized)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building postfix for rule %d (%q): %v\n", i, pattern, err)
			os.Exit(1)
		}

		// Build syntax tree
		root, posToSymbol, err := dfa.BuildTree(postfix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building syntax tree for rule %d (%q): %v\n", i, pattern, err)
			os.Exit(1)
		}

		// Generate DOT if requested
		if *genTree {
			dot := dfa.ToDOT(root)
			dotContents = append(dotContents, fmt.Sprintf("// Rule %d: %s\n%s", i, tokenName, dot))
		}

		// Build DFA
		builtDFA := dfa.BuildDFA(root, posToSymbol, tokenName)

		// Minimize DFA
		minimizedDFA := dfa.Minimize(builtDFA)

		dfaEntries = append(dfaEntries, lexer.DFAEntry{
			DFA:       minimizedDFA,
			TokenName: tokenName,
			Priority:  rule.Priority,
		})
	}

	// Write tree DOT file if requested
	if *genTree && len(dotContents) > 0 {
		dotContent := ""
		for _, d := range dotContents {
			dotContent += d + "\n"
		}
		if err := os.WriteFile("tree.dot", []byte(dotContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing tree.dot: %v\n", err)
		} else {
			fmt.Println("Generated tree.dot")
		}
	}

	// Step 4: Read the source file
	src, err := lexer.ReadSource(*srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source file: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Tokenize
	tokens, errors := lexer.Tokenize(dfaEntries, src)

	// Print tokens
	fmt.Println("\n--- Tokens ---")
	for _, tok := range tokens {
		fmt.Printf("[%d] %-12s %s\n", tok.Line, tok.Type, tok.Lexeme)
	}

	// Print errors
	if len(errors) > 0 {
		fmt.Println("\n--- Errors ---")
		for _, e := range errors {
			fmt.Println(e)
		}
		os.Exit(1)
	}
}
