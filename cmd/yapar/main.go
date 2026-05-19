package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"genanalex/internal/generator"
	"genanalex/internal/lexbuild"
	"genanalex/internal/shared"
	"genanalex/internal/yapar"
)

type outputFormat string

const (
	outputFormatNone outputFormat = ""
	outputFormatText outputFormat = "text"
	outputFormatJSON outputFormat = "json"
	outputFormatDOT  outputFormat = "dot"
)

type config struct {
	yalpFile   string
	yalFile    string
	outFile    string
	srcFile    string
	method     yapar.Method
	compare    bool
	printTable bool
	format     outputFormat
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
	logOut := stdout

	cfg, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}
	if cfg.format == outputFormatJSON || cfg.format == outputFormatDOT {
		logOut = stderr
	}

	fmt.Fprintf(logOut, "[*] Loading YAPar specification: %s\n", cfg.yalpFile)
	spec, err := yapar.ParseFile(cfg.yalpFile)
	if err != nil {
		return fmt.Errorf("parse .yalp file: %w", err)
	}

	fmt.Fprintln(logOut, "[*] Building formal grammar...")
	grammar, err := yapar.BuildGrammar(spec)
	if err != nil {
		return fmt.Errorf("build grammar: %w", err)
	}

	fmt.Fprintln(logOut, "[*] Computing FIRST/FOLLOW...")
	ff, err := yapar.ComputeFirstFollow(grammar)
	if err != nil {
		return fmt.Errorf("compute FIRST/FOLLOW: %w", err)
	}

	var tokens []shared.Token
	if cfg.srcFile != "" {
		fmt.Fprintf(logOut, "[*] Compiling lexer specification: %s\n", cfg.yalFile)
		lexResult, err := lexbuild.CompileYALFile(cfg.yalFile, false)
		if err != nil {
			return fmt.Errorf("compile .yal file: %w", err)
		}

		fmt.Fprintf(logOut, "[*] Tokenizing source file: %s\n", cfg.srcFile)
		tokenStream, lexicalErrors, err := lexbuild.TokenizeFile(lexResult.DFAEntries, cfg.srcFile)
		if err != nil {
			return fmt.Errorf("tokenize source file: %w", err)
		}
		if len(lexicalErrors) > 0 {
			return fmt.Errorf("lexical analysis failed:\n%s", strings.Join(lexicalErrors, "\n"))
		}
		tokens = tokenStream
	}

	if cfg.compare {
		return runCompare(cfg, grammar, ff, tokens, stdout, logOut)
	}

	fmt.Fprintf(logOut, "[*] Building parser backend (%s)...\n", cfg.method)
	parser, err := yapar.BuildParser(grammar, ff, cfg.method)
	if err != nil {
		return fmt.Errorf("build parser backend: %w", err)
	}

	table := parser.Table()
	fmt.Fprintf(logOut, "[+] Parser pipeline ready: %d productions, %d states\n", len(grammar.Productions), len(table.States()))
	if cfg.format != outputFormatNone {
		report, err := yapar.BuildVisReport(grammar, ff, cfg.method)
		if err != nil {
			return fmt.Errorf("build visualization report: %w", err)
		}
		if err := renderVisualization(stdout, report, cfg.method, cfg.format); err != nil {
			return err
		}
	}

	if cfg.outFile != "" {
		if cfg.method == yapar.MethodLL1 || cfg.method == yapar.MethodLALR {
			return fmt.Errorf("standalone parser generation is not supported for method %s", cfg.method)
		}
		fmt.Fprintf(logOut, "[*] Generating standalone parser: %s\n", cfg.outFile)
		if err := generator.GenerateParserSourceFromTableView(cfg.outFile, grammar, table); err != nil {
			return fmt.Errorf("generate standalone parser: %w", err)
		}
		fmt.Fprintln(logOut, "[+] Standalone parser generated successfully.")
	}

	if cfg.srcFile == "" {
		fmt.Fprintln(logOut, "[+] No source provided; syntax pipeline built successfully.")
		return nil
	}

	fmt.Fprintf(logOut, "[*] Parsing %d tokens...\n", len(tokens))
	result, err := parser.Parse(tokens)
	if err != nil {
		return fmt.Errorf("parse tokens: %w", err)
	}
	if result == nil || !result.Accepted {
		return fmt.Errorf("parse tokens: input was not accepted")
	}

	fmt.Fprintln(logOut, "[+] Input accepted.")
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
	format := string(outputFormatNone)
	fs.StringVar(&method, "method", method, "parser method: ll1, lr0, slr, lr1, lalr")
	fs.BoolVar(&cfg.compare, "compare", false, "run all supported parser methods and compare the results")
	fs.BoolVar(&cfg.printTable, "table", false, "print the generated SLR(1) parsing table")
	fs.StringVar(&format, "format", format, "visualization output format: text, json, dot")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: yapar -yalp <parser.yalp> [-out <generated_parser.go>] [-yal <lexer.yal> -src <input>] [-table|-format <text|json|dot>] [-compare]")
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
	parsedFormat, err := parseOutputFormat(format)
	if err != nil {
		return nil, err
	}
	cfg.format = parsedFormat
	if cfg.printTable {
		cfg.format = outputFormatText
	}
	if cfg.compare {
		if cfg.outFile != "" {
			return nil, fmt.Errorf("standalone parser generation is incompatible with -compare")
		}
		if cfg.format == outputFormatDOT {
			return nil, fmt.Errorf("DOT format incompatible with -compare")
		}
		if method != string(yapar.MethodSLR) {
			fmt.Fprintln(stderr, "warning: ignoring -method because -compare runs all methods")
		}
		cfg.method = yapar.MethodSLR
	} else {
		parsedMethod, err := yapar.ParseMethod(method)
		if err != nil {
			return nil, err
		}
		cfg.method = parsedMethod
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	return cfg, nil
}

func runCompare(cfg *config, g *yapar.Grammar, ff *yapar.FirstFollow, tokens []shared.Token, stdout, logOut io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if logOut == nil {
		logOut = io.Discard
	}

	methods := yapar.ValidMethods()
	var compareTokens []shared.Token
	if tokens != nil {
		compareTokens = make([]shared.Token, len(tokens))
		copy(compareTokens, tokens)
	}

	fmt.Fprintf(logOut, "[*] Building comparison report (%s)...\n", strings.Join(methodNames(methods), ", "))
	report := yapar.BuildComparisonReport(g, ff, compareTokens, methods)
	if err := renderComparison(stdout, report, cfg.format); err != nil {
		return err
	}
	if cfg.srcFile == "" {
		fmt.Fprintln(logOut, "[+] No source provided; comparison pipeline built successfully.")
	} else {
		fmt.Fprintf(logOut, "[+] Comparison completed for %d tokens.\n", len(tokens))
	}
	if report.HasSuccess() {
		return nil
	}
	return fmt.Errorf("comparison failed for all methods")
}

func renderComparison(stdout io.Writer, report *yapar.ComparisonReport, format outputFormat) error {
	if stdout == nil {
		stdout = io.Discard
	}

	switch format {
	case outputFormatJSON:
		payload, err := yapar.RenderComparisonJSON(report)
		if err != nil {
			return fmt.Errorf("render comparison json: %w", err)
		}
		_, err = stdout.Write(payload)
		return err
	case outputFormatText, outputFormatNone:
		_, err := io.WriteString(stdout, renderComparisonText(report))
		return err
	default:
		return fmt.Errorf("unsupported comparison format %q", format)
	}
}

func renderComparisonText(report *yapar.ComparisonReport) string {
	if report == nil {
		return "Comparison Summary\n<empty>\n"
	}

	var builder strings.Builder
	builder.WriteString("Comparison Summary\n")
	builder.WriteString("Method\tStatus\tDuration(ms)\n")
	for _, result := range report.Methods {
		builder.WriteString(strings.ToUpper(methodDisplayName(result.Method)))
		builder.WriteByte('\t')
		builder.WriteString(comparisonStatus(result))
		builder.WriteByte('\t')
		builder.WriteString(fmt.Sprintf("%.3f", float64(result.Duration)/float64(1e6)))
		builder.WriteByte('\n')
	}
	for _, result := range report.Methods {
		builder.WriteString("\n--- ")
		builder.WriteString(methodDisplayName(result.Method))
		builder.WriteString(" ---\n")
		if result.Error != "" {
			builder.WriteString("Error: ")
			builder.WriteString(result.Error)
			builder.WriteByte('\n')
			continue
		}
		if result.Accepted != nil {
			builder.WriteString(fmt.Sprintf("Accepted: %t\n", *result.Accepted))
		}
		builder.WriteString(yapar.RenderTableText(result.Report))
	}
	return builder.String()
}

func comparisonStatus(result yapar.MethodResult) string {
	if result.Error != "" {
		return "error"
	}
	if result.Accepted == nil {
		return "ok"
	}
	if *result.Accepted {
		return "accepted"
	}
	return "rejected"
}

func methodDisplayName(method yapar.Method) string {
	switch method {
	case yapar.MethodSLR:
		return "SLR(1)"
	case yapar.MethodLR0:
		return "LR0"
	case yapar.MethodLR1:
		return "LR1"
	case yapar.MethodLALR:
		return "LALR"
	case yapar.MethodLL1:
		return "LL1"
	default:
		return strings.ToUpper(string(method))
	}
}

func methodNames(methods []yapar.Method) []string {
	result := make([]string, len(methods))
	for i, method := range methods {
		result[i] = string(method)
	}
	sort.Strings(result)
	return result
}

func parseOutputFormat(raw string) (outputFormat, error) {
	format := outputFormat(strings.ToLower(strings.TrimSpace(raw)))
	switch format {
	case outputFormatNone, outputFormatText, outputFormatJSON, outputFormatDOT:
		return format, nil
	default:
		return outputFormatNone, fmt.Errorf("invalid output format %q: valid options are text, json, dot", raw)
	}
}

func renderVisualization(stdout io.Writer, report *yapar.VisualizationReport, method yapar.Method, format outputFormat) error {
	if stdout == nil {
		stdout = io.Discard
	}

	switch format {
	case outputFormatText:
		tableLabel := strings.ToUpper(string(method))
		if method == yapar.MethodSLR {
			tableLabel = "SLR(1)"
		}
		fmt.Fprintf(stdout, "\n--- %s Table ---\n", tableLabel)
		fmt.Fprint(stdout, yapar.RenderTableText(report))
		return nil
	case outputFormatJSON:
		payload, err := yapar.RenderTableJSON(report)
		if err != nil {
			return fmt.Errorf("render %s output: %w", format, err)
		}
		_, err = stdout.Write(payload)
		return err
	case outputFormatDOT:
		dot, err := yapar.RenderAutomatonDOT(report)
		if err != nil {
			return fmt.Errorf("render %s output for method %s: %w", format, method, err)
		}
		_, err = io.WriteString(stdout, dot)
		return err
	default:
		return nil
	}
}
