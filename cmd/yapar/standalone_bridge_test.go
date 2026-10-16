package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"genanalex/internal/generator"
	"genanalex/internal/lexbuild"
	"genanalex/internal/shared"
)

func buildStandaloneBinary(t *testing.T, source, binary string) {
	t.Helper()
	if output, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("compile standalone: %v\n%s", err, output)
	}
}

func standaloneResult(t *testing.T, binary string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, stdout.String(), stderr.String()
}

func TestStandaloneLexerParserBridge(t *testing.T) {
	if testing.Short() {
		t.Skip("bridge acceptance compiles generated lexer and parsers")
	}
	t.Setenv("BRIDGE_PROBE_FAILURE", "0")
	t.Setenv("BRIDGE_PROBE_EOF", "0")
	dir := t.TempDir()
	lexerSource, lexerBinary := filepath.Join(dir, "lexer.go"), filepath.Join(dir, "lexer binary")
	lex, err := lexbuild.CompileYALFile("../../testdata/yapar/arithmetic.yal", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := generator.GenerateSource(lexerSource, lex.DFAEntries); err != nil {
		t.Fatal(err)
	}
	buildStandaloneBinary(t, lexerSource, lexerBinary)
	badSource := writeTempFile(t, dir, "bad_lexer.go", `package main
import ("fmt"; "os")
func main() {
  if os.Getenv("BRIDGE_PROBE_EOF") == "1" {
    fmt.Print("[{\"type\":\"$\",\"line\":3}]")
    return
  }
  if os.Getenv("BRIDGE_PROBE_FAILURE") == "1" {
    fmt.Print("[]")
    fmt.Fprintln(os.Stderr, "intentional lexer failure")
    os.Exit(7)
  }
  fmt.Print("not-json")
}
`)
	badLexer := filepath.Join(dir, "bad lexer")
	buildStandaloneBinary(t, badSource, badLexer)

	t.Run("lexer JSON protocol", func(t *testing.T) {
		for _, input := range []string{"7", "", " \t\n"} {
			source := writeTempFile(t, dir, "input file.txt", input)
			code, out, diagnostics := standaloneResult(t, lexerBinary, "-src", source, "-format", "json")
			if code != 0 || diagnostics != "" {
				t.Fatalf("lexer exit=%d stderr=%q", code, diagnostics)
			}
			var tokens []shared.Token
			if err := json.Unmarshal([]byte(out), &tokens); err != nil {
				t.Fatalf("lexer stdout is not token JSON: %q, %v", out, err)
			}
			if input == "7" && (len(tokens) != 1 || tokens[0].Type != "INT" || tokens[0].Lexeme != "7" || tokens[0].Line != 1) {
				t.Fatalf("tokens=%v", tokens)
			}
			if input == "7" && !strings.Contains(out, `"type":"INT"`) {
				t.Fatalf("token JSON field names changed: %q", out)
			}
			if input != "7" && (len(tokens) != 0 || strings.TrimSpace(out) != "[]") {
				t.Fatalf("empty source JSON=%q, want []", out)
			}
		}
		source := writeTempFile(t, dir, "lexical failure.txt", "7@")
		code, out, diagnostics := standaloneResult(t, lexerBinary, "-src", source, "-format", "json")
		if code != 1 || out != "" || diagnostics == "" {
			t.Fatalf("failed lexing leaked partial JSON: (%d,%q,%q)", code, out, diagnostics)
		}
		code, out, diagnostics = standaloneResult(t, lexerBinary, "-src", source, "-format", "invalid")
		if code != 1 || out != "" || !strings.Contains(diagnostics, "invalid format") {
			t.Fatalf("invalid format result=(%d,%q,%q)", code, out, diagnostics)
		}
		defaultCode, defaultOut, defaultErr := standaloneResult(t, lexerBinary, "-src", source)
		explicitCode, explicitOut, explicitErr := standaloneResult(t, lexerBinary, "-src", source, "-format", "text")
		if defaultCode != 1 || explicitCode != defaultCode || defaultOut != explicitOut || defaultErr != explicitErr || !strings.Contains(defaultOut, "--- Tokens ---") {
			t.Fatal("explicit text mode differs from legacy default")
		}
	})

	for _, method := range []string{"ll1", "slr", "lalr"} {
		t.Run(method, func(t *testing.T) {
			parserSource, parserBinary := filepath.Join(dir, method+".go"), filepath.Join(dir, method+" parser")
			var stdout, stderr bytes.Buffer
			if err := run([]string{"-yalp", "../../testdata/yapar/demos/nullable.yalp", "-method", method, "-out", parserSource}, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			buildStandaloneBinary(t, parserSource, parserBinary)
			for _, tt := range []struct {
				name, input, message string
				code                 int
			}{
				{name: "valid", input: "7"},
				{name: "empty"},
				{name: "whitespace", input: " \t\n"},
				{name: "syntax rejection", input: "+", code: 1, message: "syntax error"},
				{name: "lexical rejection", input: "7@", code: 1, message: "lexer failed"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					source := writeTempFile(t, dir, "input file.txt", tt.input)
					code, out, diagnostics := standaloneResult(t, parserBinary, "-lexer", lexerBinary, "-src", source)
					if code != tt.code {
						t.Fatalf("exit=%d want %d stdout=%q stderr=%q", code, tt.code, out, diagnostics)
					}
					if tt.code == 0 {
						if !strings.Contains(out, "Input accepted") || diagnostics != "" {
							t.Fatalf("accepted stdout=%q stderr=%q", out, diagnostics)
						}
					} else if out != "" || !strings.Contains(diagnostics, tt.message) {
						t.Fatalf("rejected stdout=%q stderr=%q", out, diagnostics)
					}
					after, err := os.ReadFile(source)
					if err != nil || string(after) != tt.input {
						t.Fatalf("bridge changed source input: %v", err)
					}
				})
			}
			source := writeTempFile(t, dir, "input file.txt", "7")
			tokens := writeTempFile(t, dir, "tokens.json", "[]")
			for _, tt := range []struct {
				name, message string
				args          []string
			}{
				{name: "conflicting modes", args: []string{"-tokens", tokens, "-lexer", lexerBinary, "-src", source}, message: "-tokens cannot be combined"},
				{name: "missing source", args: []string{"-lexer", lexerBinary}, message: "-lexer and -src must be provided together"},
				{name: "missing lexer", args: []string{"-src", source}, message: "-lexer and -src must be provided together"},
				{name: "missing executable", args: []string{"-lexer", filepath.Join(dir, "missing"), "-src", source}, message: "lexer failed"},
				{name: "missing source file", args: []string{"-lexer", lexerBinary, "-src", filepath.Join(dir, "missing")}, message: "lexer failed"},
				{name: "invalid lexer JSON", args: []string{"-lexer", badLexer, "-src", source}, message: "decode lexer JSON"},
				{name: "failed lexer with partial JSON", args: []string{"-lexer", badLexer, "-src", source}, message: "intentional lexer failure"},
				{name: "lexer supplies reserved EOF", args: []string{"-lexer", badLexer, "-src", source}, message: "reserved for end of input at line 3"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					if tt.name == "failed lexer with partial JSON" {
						t.Setenv("BRIDGE_PROBE_FAILURE", "1")
					}
					if tt.name == "lexer supplies reserved EOF" {
						t.Setenv("BRIDGE_PROBE_EOF", "1")
					}
					code, out, diagnostics := standaloneResult(t, parserBinary, tt.args...)
					if code != 1 || out != "" || !strings.Contains(diagnostics, tt.message) {
						t.Fatalf("bridge error=(%d,%q,%q), want %q", code, out, diagnostics, tt.message)
					}
				})
			}
			stdout.Reset()
			if err := run([]string{"-yalp", "../../testdata/yapar/demos/required.yalp", "-method", method, "-out", parserSource}, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			buildStandaloneBinary(t, parserSource, parserBinary)
			empty := writeTempFile(t, dir, "input file.txt", "")
			code, out, diagnostics := standaloneResult(t, parserBinary, "-lexer", lexerBinary, "-src", empty)
			if code != 1 || out != "" || !strings.Contains(diagnostics, "syntax error") {
				t.Fatalf("non-nullable empty result=(%d,%q,%q)", code, out, diagnostics)
			}
		})
	}
}
