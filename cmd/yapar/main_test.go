package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	err := run([]string{"-yalp", yalpPath, "-method", "slr", "-table"}, &stdout, &stderr)
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

func TestRunDefaultsToSLRMethod(t *testing.T) {
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
	err := run([]string{"-yalp", yalpPath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Parser pipeline ready") {
		t.Fatalf("stdout = %q, want pipeline summary", stdout.String())
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

func TestRunRejectsInvalidMethod(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "foo"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want invalid method error")
	}
	if !strings.Contains(err.Error(), "slr") || !strings.Contains(err.Error(), "ll1") || !strings.Contains(err.Error(), "lalr") {
		t.Fatalf("error = %q, want valid method options", err.Error())
	}
}

func TestRunBuildsLALRPipelineWithoutSource(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "lalr"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Parser pipeline ready") {
		t.Fatalf("stdout = %q, want pipeline summary", out)
	}
	if !strings.Contains(out, "No source provided") {
		t.Fatalf("stdout = %q, want no-source confirmation", out)
	}
}

func TestRunRejectsLL1StandaloneGeneration(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	outPath := filepath.Join(dir, "parser.go")
	if err := os.WriteFile(yalpPath, []byte(`%token A B
%%
s : A opt ;
opt : B | ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "ll1", "-out", outPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unsupported ll1 codegen error")
	}
	if !strings.Contains(err.Error(), "ll1") || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("error = %q, want ll1 not supported error", err.Error())
	}
}

func TestRunRejectsLALRStandaloneGeneration(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	outPath := filepath.Join(dir, "parser.go")
	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "lalr", "-out", outPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unsupported lalr codegen error")
	}
	if !strings.Contains(err.Error(), "lalr") || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("error = %q, want lalr not supported error", err.Error())
	}
}

func TestRunBuildsLL1PipelineWithoutSource(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token A B
%%
s : A opt ;
opt : B | ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "ll1", "-table"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Parser pipeline ready") {
		t.Fatalf("stdout = %q, want pipeline summary", out)
	}
	if !strings.Contains(out, "LL1 Table") {
		t.Fatalf("stdout = %q, want LL1 rendered table", out)
	}
}

func TestRunEmitsJSONVisualization(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-format", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	var payload struct {
		Method string `json:"method"`
		States []struct {
			Actions map[string]string `json:"actions"`
		} `json:"states"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if payload.Method != "slr" {
		t.Fatalf("method = %q, want slr", payload.Method)
	}
	if len(payload.States) == 0 {
		t.Fatal("states = 0, want populated JSON table")
	}
	if !strings.Contains(stderr.String(), "Parser pipeline ready") {
		t.Fatalf("stderr = %q, want pipeline summary", stderr.String())
	}
}

func TestRunEmitsDOTVisualizationForLALR(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token ID PLUS
%%
expr : ID PLUS ID ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "lalr", "-format", "dot"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "digraph LR0") {
		t.Fatalf("stdout = %q, want DOT graph", out)
	}
	if !strings.Contains(out, "->") {
		t.Fatalf("stdout = %q, want transitions", out)
	}
}

func TestRunRejectsDOTForLL1(t *testing.T) {
	dir := t.TempDir()
	yalpPath := filepath.Join(dir, "parser.yalp")
	if err := os.WriteFile(yalpPath, []byte(`%token A B
%%
s : A opt ;
opt : B | ;
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-method", "ll1", "-format", "dot"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unsupported DOT error")
	}
	if !strings.Contains(err.Error(), "not supported") || !strings.Contains(err.Error(), "ll1") {
		t.Fatalf("error = %q, want explicit unsupported ll1 message", err.Error())
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

func TestRunParsesTokenizedSourceWithLALR(t *testing.T) {
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
	err := run([]string{"-yalp", yalpPath, "-yal", yalPath, "-src", srcPath, "-method", "lalr", "-table"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "LALR Table") {
		t.Fatalf("stdout = %q, want LALR rendered table", out)
	}
	if !strings.Contains(out, "Input accepted.") {
		t.Fatalf("stdout = %q, want accepted parse", out)
	}
}

func TestRunCompareTextRendersAllMethods(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-compare", "-format", "text"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, label := range []string{"Comparison Summary", "LL1", "LR0", "SLR(1)", "LR1", "LALR"} {
		if !strings.Contains(out, label) {
			t.Fatalf("stdout = %q, want label %q", out, label)
		}
	}
}

func TestRunCompareJSONRendersMethodsArray(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-compare", "-format", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	var payload struct {
		Methods []struct {
			Method string `json:"method"`
		} `json:"methods"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	got := make([]string, len(payload.Methods))
	for i, result := range payload.Methods {
		got[i] = result.Method
	}
	if want := []string{"ll1", "lr0", "slr", "lr1", "lalr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("methods = %v, want %v", got, want)
	}
	if !strings.Contains(stderr.String(), "Building comparison report") {
		t.Fatalf("stderr = %q, want compare pipeline log", stderr.String())
	}
}

func TestRunCompareRejectsDOTFormat(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-compare", "-format", "dot"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want incompatible dot error")
	}
	if !strings.Contains(err.Error(), "DOT format incompatible") {
		t.Fatalf("error = %q, want DOT incompatibility", err.Error())
	}
}

func TestRunCompareWarnsAndIgnoresMethodFlag(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-compare", "-method", "ll1", "-format", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "ignoring -method because -compare runs all methods") {
		t.Fatalf("stderr = %q, want warning about ignored method", stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"method\":\"ll1\"") || !strings.Contains(stdout.String(), "\"method\":\"lr1\"") || !strings.Contains(stdout.String(), "\"method\":\"lalr\"") {
		t.Fatalf("stdout = %q, want all comparison methods", stdout.String())
	}
}

func TestRunCompareRejectsStandaloneGeneration(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
	outPath := filepath.Join(dir, "generated.go")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-compare", "-out", outPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want incompatible out error")
	}
	if !strings.Contains(err.Error(), "standalone parser generation is incompatible with -compare") {
		t.Fatalf("error = %q, want compare out incompatibility", err.Error())
	}
}

func TestRunCompareContinuesAfterLL1Failure(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token A B
%%
s : A | A B ;
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-compare", "-format", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ll1 conflict") {
		t.Fatalf("stdout = %q, want ll1 conflict entry", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"method\":\"slr\"") || !strings.Contains(stdout.String(), "\"method\":\"lr0\"") || !strings.Contains(stdout.String(), "\"method\":\"lalr\"") {
		t.Fatalf("stdout = %q, want mixed successful and not-implemented entries", stdout.String())
	}
}

func TestRunCompareReturnsExitOneWhenAllMethodsFail(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token INT PLUS WS
IGNORE WS
%%
expr : INT PLUS INT ;
`)
	yalPath := writeTempFile(t, dir, "lexer.yal", `let DIGIT = [0-9]

rule tokens =
  | [' ' '\t' '\n']+ { WS }
  | DIGIT+ { INT }
  | '+' { PLUS }
`)
	srcPath := writeTempFile(t, dir, "input.txt", "12\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-yalp", yalpPath, "-yal", yalPath, "-src", srcPath, "-compare", "-format", "json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want all-method failure")
	}
	if got := err.Error(); !strings.Contains(got, "comparison failed for all methods") {
		t.Fatalf("error = %q, want all-method failure message", got)
	}
	for _, needle := range []string{"\"method\":\"ll1\"", "\"method\":\"lr0\"", "\"method\":\"slr\"", "\"method\":\"lr1\"", "\"method\":\"lalr\""} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), needle)
		}
	}
}

func TestMainExitsOneWhenCompareFailsForAllMethods(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		os.Args = []string{
			"yapar",
			"-yalp", os.Getenv("GO_HELPER_YALP"),
			"-yal", os.Getenv("GO_HELPER_YAL"),
			"-src", os.Getenv("GO_HELPER_SRC"),
			"-compare",
			"-format", "json",
		}
		main()
		return
	}

	dir := t.TempDir()
	yalpPath := writeTempFile(t, dir, "parser.yalp", `%token INT PLUS WS
IGNORE WS
%%
expr : INT PLUS INT ;
`)
	yalPath := writeTempFile(t, dir, "lexer.yal", `let DIGIT = [0-9]

rule tokens =
  | [' ' '\t' '\n']+ { WS }
  | DIGIT+ { INT }
  | '+' { PLUS }
`)
	srcPath := writeTempFile(t, dir, "input.txt", "12\n")

	cmd := exec.Command(os.Args[0], "-test.run=^TestMainExitsOneWhenCompareFailsForAllMethods$")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"GO_HELPER_YALP="+yalpPath,
		"GO_HELPER_YAL="+yalPath,
		"GO_HELPER_SRC="+srcPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("helper process error = nil, want exit status 1\noutput=%s", output)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("helper process error type = %T, want *exec.ExitError", err)
	}
	if got, want := exitErr.ExitCode(), 1; got != want {
		t.Fatalf("helper process exit code = %d, want %d\noutput=%s", got, want, output)
	}
	if !strings.Contains(string(output), "comparison failed for all methods") {
		t.Fatalf("helper process output = %q, want all-method failure message", string(output))
	}
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", name, err)
	}
	return path
}
