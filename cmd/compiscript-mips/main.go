package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	compiscript "genanalex/internal/compiscript"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compiscript-mips", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "mips", "output format: mips assembly or json compilation report")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: compiscript-mips [-format mips|json] <file.cps>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}
	if *format != "mips" && *format != "json" {
		fmt.Fprintln(stderr, "compiscript-mips: format must be mips or json")
		return 2
	}
	path := flags.Arg(0)
	if filepath.Ext(path) != ".cps" {
		fmt.Fprintf(stderr, "compiscript-mips: expected a .cps file: %s\n", path)
		return 2
	}
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "compiscript-mips: read %q: %v\n", path, err)
		return 1
	}
	if !utf8.Valid(source) {
		fmt.Fprintf(stderr, "compiscript-mips: source is not valid UTF-8: %s\n", path)
		return 1
	}
	report := compiscript.CompileMIPS(source)
	var output []byte
	if *format == "json" {
		output, err = json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "compiscript-mips: encode report: %v\n", err)
			return 1
		}
		output = append(output, '\n')
	} else if len(report.Diagnostics) == 0 {
		output = []byte(report.MIPSText)
	} else {
		for _, diagnostic := range report.Diagnostics {
			fmt.Fprintf(stderr, "%s %s at %d:%d: %s\n", diagnostic.Phase, diagnostic.Code,
				diagnostic.Span.Start.Line, diagnostic.Span.Start.Column, diagnostic.Message)
		}
		return 1
	}
	if _, err := stdout.Write(output); err != nil {
		fmt.Fprintf(stderr, "compiscript-mips: write output: %v\n", err)
		return 1
	}
	if len(report.Diagnostics) > 0 {
		return 1
	}
	return 0
}
