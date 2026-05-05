package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"genanalex/internal/lexbuild"
	"genanalex/internal/yapar"
)

type config struct {
	yalpFile   string
	yalFile    string
	srcFile    string
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

	fmt.Fprintln(stdout, "[*] Building LR(0) canonical collection...")
	states, transitions, err := yapar.BuildCanonicalCollection(grammar)
	if err != nil {
		return fmt.Errorf("build LR(0) collection: %w", err)
	}

	fmt.Fprintln(stdout, "[*] Building SLR(1) parsing table...")
	table, err := yapar.BuildSLRTable(grammar, ff, states, transitions)
	if err != nil {
		return fmt.Errorf("build SLR(1) table: %w", err)
	}

	fmt.Fprintf(stdout, "[+] Parser pipeline ready: %d productions, %d states\n", len(grammar.Productions), len(states))
	if cfg.printTable {
		fmt.Fprintln(stdout, "\n--- SLR(1) Table ---")
		fmt.Fprint(stdout, formatParsingTable(grammar, table))
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
	result, err := yapar.ParseTokens(grammar, table, tokens)
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
	fs.StringVar(&cfg.srcFile, "src", "", "path to the source file to tokenize and parse")
	fs.BoolVar(&cfg.printTable, "table", false, "print the generated SLR(1) parsing table")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: yapar -yalp <parser.yalp> [-yal <lexer.yal> -src <input>] [-table]")
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
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	return cfg, nil
}

func formatParsingTable(grammar *yapar.Grammar, table *yapar.ParsingTable) string {
	if table == nil {
		return "<empty table>\n"
	}

	terms := sortedSymbols(grammar.Terminals)
	nonTerms := sortedNonTerminals(grammar)
	states := sortedStateIDs(table)

	var builder strings.Builder
	builder.WriteString("State\tACTION\tGOTO\n")
	for _, state := range states {
		builder.WriteString(fmt.Sprintf("%d\t%s\t%s\n", state, formatActionRow(table, state, terms), formatGotoRow(table, state, nonTerms)))
	}
	return builder.String()
}

func sortedSymbols(symbols map[string]bool) []string {
	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}
	sort.Strings(result)
	return result
}

func sortedNonTerminals(grammar *yapar.Grammar) []string {
	if grammar == nil {
		return nil
	}
	result := make([]string, 0, len(grammar.NonTerminals))
	for symbol := range grammar.NonTerminals {
		if symbol == grammar.Augmented {
			continue
		}
		result = append(result, symbol)
	}
	sort.Strings(result)
	return result
}

func sortedStateIDs(table *yapar.ParsingTable) []int {
	stateSet := make(map[int]bool)
	for state := range table.Action {
		stateSet[state] = true
	}
	for state := range table.Goto {
		stateSet[state] = true
	}
	result := make([]int, 0, len(stateSet))
	for state := range stateSet {
		result = append(result, state)
	}
	sort.Ints(result)
	return result
}

func formatActionRow(table *yapar.ParsingTable, state int, terminals []string) string {
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

func formatGotoRow(table *yapar.ParsingTable, state int, nonTerminals []string) string {
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

func lookupAction(table *yapar.ParsingTable, state int, symbol string) string {
	if table == nil || table.Action == nil {
		return ""
	}
	row := table.Action[state]
	if row == nil {
		return ""
	}
	action, ok := row[symbol]
	if !ok {
		return ""
	}
	switch action.Kind {
	case yapar.ActionShift:
		return fmt.Sprintf("s%d", action.TargetState)
	case yapar.ActionReduce:
		return fmt.Sprintf("r%d", action.ProductionID)
	case yapar.ActionAccept:
		return "acc"
	default:
		return ""
	}
}

func lookupGoto(table *yapar.ParsingTable, state int, symbol string) (int, bool) {
	if table == nil || table.Goto == nil {
		return 0, false
	}
	row := table.Goto[state]
	if row == nil {
		return 0, false
	}
	target, ok := row[symbol]
	return target, ok
}
