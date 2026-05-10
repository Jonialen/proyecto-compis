package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"genanalex/internal/generator"
	"genanalex/internal/lexbuild"
	"genanalex/internal/yapar"
)

type config struct {
	yalpFile   string
	yalFile    string
	outFile    string
	srcFile    string
	method     yapar.Method
	printTable bool
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

	fmt.Fprintf(stdout, "[*] Loading YAPar specification: %s\n", cfg.yalpFile)
	spec, err := yapar.ParseFile(cfg.yalpFile)
	if err != nil {
		return fmt.Errorf("parse .yalp file: %w", err)
	}

	fmt.Fprintln(stdout, "[*] Building formal grammar...")
	grammar, err := yapar.BuildGrammar(spec)
	if err != nil {
		return fmt.Errorf("build grammar: %w", err)
	}

	fmt.Fprintln(stdout, "[*] Computing FIRST/FOLLOW...")
	ff, err := yapar.ComputeFirstFollow(grammar)
	if err != nil {
		return fmt.Errorf("compute FIRST/FOLLOW: %w", err)
	}

	fmt.Fprintf(stdout, "[*] Building parser backend (%s)...\n", cfg.method)
	parser, err := yapar.BuildParser(grammar, ff, cfg.method)
	if err != nil {
		return fmt.Errorf("build parser backend: %w", err)
	}

	table := parser.Table()
	fmt.Fprintf(stdout, "[+] Parser pipeline ready: %d productions, %d states\n", len(grammar.Productions), len(table.States()))
	if cfg.printTable {
		tableLabel := strings.ToUpper(string(cfg.method))
		if cfg.method == yapar.MethodSLR {
			tableLabel = "SLR(1)"
		}
		fmt.Fprintf(stdout, "\n--- %s Table ---\n", tableLabel)
		fmt.Fprint(stdout, formatParsingTable(table))
	}

	if cfg.outFile != "" {
		if cfg.method == yapar.MethodLL1 || cfg.method == yapar.MethodLALR {
			return fmt.Errorf("standalone parser generation is not supported for method %s", cfg.method)
		}
		fmt.Fprintf(stdout, "[*] Generating standalone parser: %s\n", cfg.outFile)
		if err := generator.GenerateParserSourceFromTableView(cfg.outFile, grammar, table); err != nil {
			return fmt.Errorf("generate standalone parser: %w", err)
		}
		fmt.Fprintln(stdout, "[+] Standalone parser generated successfully.")
	}

	if cfg.srcFile == "" {
		fmt.Fprintln(stdout, "[+] No source provided; syntax pipeline built successfully.")
		return nil
	}

	fmt.Fprintf(stdout, "[*] Compiling lexer specification: %s\n", cfg.yalFile)
	lexResult, err := lexbuild.CompileYALFile(cfg.yalFile, false)
	if err != nil {
		return fmt.Errorf("compile .yal file: %w", err)
	}

	fmt.Fprintf(stdout, "[*] Tokenizing source file: %s\n", cfg.srcFile)
	tokens, lexicalErrors, err := lexbuild.TokenizeFile(lexResult.DFAEntries, cfg.srcFile)
	if err != nil {
		return fmt.Errorf("tokenize source file: %w", err)
	}
	if len(lexicalErrors) > 0 {
		return fmt.Errorf("lexical analysis failed:\n%s", strings.Join(lexicalErrors, "\n"))
	}

	fmt.Fprintf(stdout, "[*] Parsing %d tokens...\n", len(tokens))
	result, err := parser.Parse(tokens)
	if err != nil {
		return fmt.Errorf("parse tokens: %w", err)
	}
	if result == nil || !result.Accepted {
		return fmt.Errorf("parse tokens: input was not accepted")
	}

	fmt.Fprintln(stdout, "[+] Input accepted.")
	return nil
}

func parseFlags(args []string, stderr io.Writer) (*config, error) {
	fs := flag.NewFlagSet("yapar", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg := &config{}
	fs.StringVar(&cfg.yalpFile, "yalp", "", "path to the .yalp parser specification file")
	fs.StringVar(&cfg.yalFile, "yal", "", "path to the .yal lexer specification file")
	fs.StringVar(&cfg.outFile, "out", "", "path to the output .go file for the generated parser")
	fs.StringVar(&cfg.srcFile, "src", "", "path to the source file to tokenize and parse")
	method := string(yapar.MethodSLR)
	fs.StringVar(&method, "method", method, "parser method: slr, ll1, lalr")
	fs.BoolVar(&cfg.printTable, "table", false, "print the generated SLR(1) parsing table")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: yapar -yalp <parser.yalp> [-out <generated_parser.go>] [-yal <lexer.yal> -src <input>] [-table]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if cfg.yalpFile == "" {
		fs.Usage()
		return nil, fmt.Errorf("missing required -yalp flag")
	}
	if (cfg.yalFile == "") != (cfg.srcFile == "") {
		return nil, fmt.Errorf("-yal and -src must be provided together")
	}
	parsedMethod, err := yapar.ParseMethod(method)
	if err != nil {
		return nil, err
	}
	cfg.method = parsedMethod
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	return cfg, nil
}

func formatParsingTable(table yapar.TableView) string {
	if table == nil {
		return "<empty table>\n"
	}

	terms := table.Terminals()
	nonTerms := table.NonTerminals()
	states := table.States()

	var builder strings.Builder
	builder.WriteString("State\tACTION\tGOTO\n")
	for _, state := range states {
		builder.WriteString(fmt.Sprintf("%d\t%s\t%s\n", state, formatActionRow(table, state, terms), formatGotoRow(table, state, nonTerms)))
	}
	return builder.String()
}

func formatActionRow(table yapar.TableView, state int, terminals []string) string {
	parts := make([]string, 0, len(terminals))
	for _, symbol := range terminals {
		action := lookupAction(table, state, symbol)
		if action == "" {
			continue
		}
		parts = append(parts, symbol+"="+action)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func formatGotoRow(table yapar.TableView, state int, nonTerminals []string) string {
	parts := make([]string, 0, len(nonTerminals))
	for _, symbol := range nonTerminals {
		if target, ok := lookupGoto(table, state, symbol); ok {
			parts = append(parts, fmt.Sprintf("%s=%d", symbol, target))
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func lookupAction(table yapar.TableView, state int, symbol string) string {
	kind, value, ok := table.ActionAt(state, symbol)
	if !ok {
		return ""
	}
	switch kind {
	case yapar.ActionShift:
		return fmt.Sprintf("s%d", value)
	case yapar.ActionReduce:
		return fmt.Sprintf("r%d", value)
	case yapar.ActionAccept:
		return "acc"
	default:
		return ""
	}
}

func lookupGoto(table yapar.TableView, state int, symbol string) (int, bool) {
	return table.GotoAt(state, symbol)
}
