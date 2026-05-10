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

	fmt.Fprintf(logOut, "[*] Compiling lexer specification: %s\n", cfg.yalFile)
	lexResult, err := lexbuild.CompileYALFile(cfg.yalFile, false)
	if err != nil {
		return fmt.Errorf("compile .yal file: %w", err)
	}

	fmt.Fprintf(logOut, "[*] Tokenizing source file: %s\n", cfg.srcFile)
	tokens, lexicalErrors, err := lexbuild.TokenizeFile(lexResult.DFAEntries, cfg.srcFile)
	if err != nil {
		return fmt.Errorf("tokenize source file: %w", err)
	}
	if len(lexicalErrors) > 0 {
		return fmt.Errorf("lexical analysis failed:\n%s", strings.Join(lexicalErrors, "\n"))
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
	fs.StringVar(&method, "method", method, "parser method: slr, ll1, lalr")
	fs.BoolVar(&cfg.printTable, "table", false, "print the generated SLR(1) parsing table")
	fs.StringVar(&format, "format", format, "visualization output format: text, json, dot")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: yapar -yalp <parser.yalp> [-out <generated_parser.go>] [-yal <lexer.yal> -src <input>] [-table|-format <text|json|dot>]")
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
	parsedFormat, err := parseOutputFormat(format)
	if err != nil {
		return nil, err
	}
	cfg.format = parsedFormat
	if cfg.printTable {
		cfg.format = outputFormatText
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	return cfg, nil
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
