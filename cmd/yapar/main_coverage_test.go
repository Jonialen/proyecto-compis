package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"genanalex/internal/yapar"
)

func TestRunHandlesNilWriters(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeParserSpec(t, dir, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)

	if err := run([]string{"-yalp", yalpPath}, nil, nil); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestParseFlagsValidationBranches(t *testing.T) {
	t.Run("missing yalp", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := parseFlags(nil, &stderr)
		if err == nil {
			t.Fatal("parseFlags() error = nil, want missing -yalp")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if !strings.Contains(stderr.String(), "Usage: yapar") {
			t.Fatalf("stderr = %q, want usage output", stderr.String())
		}
		if got, want := err.Error(), "missing required -yalp flag"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("yal and src must be paired", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-yal", "lexer.yal"}, &stderr)
		if err == nil {
			t.Fatal("parseFlags() error = nil, want pairing error")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if got, want := err.Error(), "-yal and -src must be provided together"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("unexpected positional args are rejected", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "extra"}, &stderr)
		if err == nil {
			t.Fatal("parseFlags() error = nil, want positional-args error")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if got, want := err.Error(), "unexpected positional arguments: extra"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("valid ll1 config keeps parsed method", func(t *testing.T) {
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-method", "ll1", "-table"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parseFlags() error = %v", err)
		}
		if cfg.method != yapar.MethodLL1 {
			t.Fatalf("cfg.method = %q, want %q", cfg.method, yapar.MethodLL1)
		}
		if !cfg.printTable {
			t.Fatal("cfg.printTable = false, want true")
		}
	})

	t.Run("format flag is parsed", func(t *testing.T) {
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-format", "json"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parseFlags() error = %v", err)
		}
		if got, want := cfg.format, outputFormatJSON; got != want {
			t.Fatalf("cfg.format = %q, want %q", got, want)
		}
	})

	t.Run("table alias forces text format", func(t *testing.T) {
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-format", "dot", "-table"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parseFlags() error = %v", err)
		}
		if got, want := cfg.format, outputFormatText; got != want {
			t.Fatalf("cfg.format = %q, want %q", got, want)
		}
	})

	t.Run("invalid format is rejected", func(t *testing.T) {
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-format", "xml"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("parseFlags() error = nil, want invalid format")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if !strings.Contains(err.Error(), "invalid output format") {
			t.Fatalf("error = %q, want invalid format message", err.Error())
		}
	})
}

func writeParserSpec(t *testing.T, dir, content string) string {
	t.Helper()
	path := dir + "/parser.yalp"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
