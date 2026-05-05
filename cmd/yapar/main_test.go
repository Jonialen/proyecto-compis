package main

import (
	"bytes"
	"os"
	"os/exec"
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

func TestRunGeneratesStandaloneParser(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("requires go compiler")
	}

	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	outPath := filepath.Join(dir, "generated_parser.go")

	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile(yalp) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-out", outPath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("generated parser file missing: %v", err)
	}

	buildDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte("module testparser\n\ngo 1.26.1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	generatedSource, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(generated parser) error = %v", err)
	}
	buildSource := filepath.Join(buildDir, "main.go")
	if err := os.WriteFile(buildSource, generatedSource, 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(buildDir, "parser_bin"), buildSource)
	cmd.Dir = buildDir
	buildOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated parser compilation failed: %v\n%s", err, string(buildOut))
	}

	out := stdout.String()
	if !strings.Contains(out, "Standalone parser generated successfully") {
		t.Fatalf("stdout = %q, want standalone parser success message", out)
	}
	if !strings.Contains(out, "No source provided") {
		t.Fatalf("stdout = %q, want no-source confirmation", out)
	}
}
