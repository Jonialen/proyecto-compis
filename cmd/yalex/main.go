package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"genanalex/internal/generator"
	"genanalex/internal/lexbuild"
)

type config struct {
	yalFile string
	srcFile string
	outFile string
	genTree bool
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	cfg, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "[*] Loading YALex specification: %s\n", cfg.yalFile)
	buildResult, err := lexbuild.CompileYALFile(cfg.yalFile, cfg.genTree)
	if err != nil {
		return fmt.Errorf("compile .yal file: %w", err)
	}
	fmt.Fprintf(stdout, "[+] Parsed %d macros and %d rules\n", len(buildResult.Macros), len(buildResult.Rules))

	if cfg.genTree && len(buildResult.DOTContents) > 0 {
		dotContent := ""
		for _, d := range buildResult.DOTContents {
			dotContent += d + "\n"
		}
		if err := os.WriteFile("tree.dot", []byte(dotContent), 0o644); err != nil {
			fmt.Fprintf(stderr, "Error writing tree.dot: %v\n", err)
		} else {
			fmt.Fprintln(stdout, "[+] Generated tree.dot for visualization")
		}
	}

	if cfg.outFile != "" {
		fmt.Fprintf(stdout, "[*] Generating standalone lexer: %s\n", cfg.outFile)
		if err := generator.GenerateSource(cfg.outFile, buildResult.DFAEntries); err != nil {
			return fmt.Errorf("generate lexer source: %w", err)
		}
		fmt.Fprintln(stdout, "[+] Lexer generation successful")
	}

	if cfg.srcFile == "" {
		return nil
	}

	fmt.Fprintf(stdout, "[*] Tokenizing source file: %s\n", cfg.srcFile)
	tokens, errors, err := lexbuild.TokenizeFile(buildResult.DFAEntries, cfg.srcFile)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	fmt.Fprintln(stdout, "\n--- Tokenization Results ---")
	for _, tok := range tokens {
		fmt.Fprintf(stdout, "[%d] %-12s %s\n", tok.Line, tok.Type, tok.Lexeme)
	}

	if len(errors) > 0 {
		fmt.Fprintln(stdout, "\n--- Lexical Errors ---")
		for _, lexicalErr := range errors {
			fmt.Fprintln(stdout, lexicalErr)
		}
		return fmt.Errorf("lexical analysis failed")
	}

	return nil
}

func parseFlags(args []string, stderr io.Writer) (*config, error) {
	fs := flag.NewFlagSet("yalex", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg := &config{}
	fs.StringVar(&cfg.yalFile, "yal", "", "path to the .yal lexer specification file")
	fs.StringVar(&cfg.srcFile, "src", "", "path to the source file to tokenize (optional if -out is used)")
	fs.StringVar(&cfg.outFile, "out", "", "path to the output .go file for the generated lexer (optional)")
	fs.BoolVar(&cfg.genTree, "tree", false, "generate tree.dot Graphviz file for syntax trees")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: yalex -yal <specification.yal> [-src <input_file>] [-out <output_lexer.go>] [-tree]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if cfg.yalFile == "" {
		fs.Usage()
		return nil, fmt.Errorf("missing required -yal flag")
	}
	if cfg.srcFile == "" && cfg.outFile == "" && !cfg.genTree {
		return nil, fmt.Errorf("you must provide at least one of -src, -out, or -tree")
	}

	return cfg, nil
}
