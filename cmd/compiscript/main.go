package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	compiscript "genanalex/internal/compiscript"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: compiscript <file.cps>")
		return exitUsage
	}

	path := args[0]
	if filepath.Ext(path) != ".cps" {
		fmt.Fprintf(stderr, "compiscript: expected a .cps file: %s\n", path)
		return exitUsage
	}

	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "compiscript: read %q: %v\n", path, err)
		return exitFailure
	}
	if !utf8.Valid(source) {
		fmt.Fprintf(stderr, "compiscript: source is not valid UTF-8: %s\n", path)
		return exitFailure
	}

	output, err := json.MarshalIndent(compiscript.Analyze(source), "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "compiscript: encode report: %v\n", err)
		return exitFailure
	}
	output = append(output, '\n')
	if _, err := stdout.Write(output); err != nil {
		fmt.Fprintf(stderr, "compiscript: write report: %v\n", err)
		return exitFailure
	}
	return exitOK
}
