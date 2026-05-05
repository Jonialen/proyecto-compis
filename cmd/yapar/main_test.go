package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildsPipelineWithoutSource(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-table"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Parser pipeline ready") {
		t.Fatalf("stdout = %q, want pipeline summary", out)
	}
	if !strings.Contains(out, "SLR(1) Table") {
		t.Fatalf("stdout = %q, want rendered table", out)
	}
	if !strings.Contains(out, "No source provided") {
		t.Fatalf("stdout = %q, want no-source confirmation", out)
	}
}

func TestRunParsesTokenizedSource(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	yalPath := filepath.Join(dir, "lexer.yal")
	srcPath := filepath.Join(dir, "input.txt")

	if err := os.WriteFile(yalpPath, []byte(`%token INT PLUS WS
IGNORE WS
%%
expr : INT PLUS INT ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile(yalp) error = %v", err)
	}

	if err := os.WriteFile(yalPath, []byte(`let DIGIT = [0-9]

rule tokens =
  | [' ' '\t' '\n']+ { WS }
  | DIGIT+ { INT }
  | '+' { PLUS }
`), 0o644); err != nil {
		t.Fatalf("WriteFile(yal) error = %v", err)
	}

	if err := os.WriteFile(srcPath, []byte("12 + 34\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(src) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-yal", yalPath, "-src", srcPath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Tokenizing source file") {
		t.Fatalf("stdout = %q, want tokenization step", out)
	}
	if !strings.Contains(out, "Input accepted.") {
		t.Fatalf("stdout = %q, want accepted parse", out)
	}
}
