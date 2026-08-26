package frontend

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrammarCompatibilityAndFloatLiterals(t *testing.T) {
	root := repositoryRoot(t)
	output := filepath.Join(root, "internal", "compiscript", "frontend", "generated")
	command := exec.Command(filepath.Join(root, "scripts", "generate-compiscript.sh"), "--output", output)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate grammar: %v\n%s", err, output)
	}

	probe := writeGrammarProbe(t, root)
	for _, test := range []struct {
		name    string
		source  string
		literal string
		valid   bool
	}{
		{"existing integer", "let count: integer = 42;", "42", true},
		{"identifier beginning with float", "let floatValue: integer = 0;", "0", true},
		{"float declaration", "let ratio: float = 3.14;", "3.14", true},
		{"for owns initializer separator", "for (let i: integer = 0; i < 3; i = i + 1) { print(i); }", "0", true},
		{"single statement if", "if (ready) print(1);", "1", true},
		{"leading dot is rejected", "let ratio = .5;", "", false},
		{"trailing dot is rejected", "let ratio = 5.;", "", false},
		{"double dot is rejected", "let ratio = 1..2;", "", false},
		{"multiple decimal points are rejected", "let ratio = 1.2.3;", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command("go", "run", probe, test.literal)
			command.Dir = root
			command.Stdin = strings.NewReader(test.source)
			output, err := command.CombinedOutput()
			if test.valid && err != nil {
				t.Fatalf("valid source was rejected: %v\n%s", err, output)
			}
			if !test.valid && err == nil {
				t.Fatalf("malformed float was accepted: %s", test.source)
			}
		})
	}
}

func writeGrammarProbe(t *testing.T, root string) string {
	t.Helper()
	directory := filepath.Join(root, ".compiscript-grammar-probe")
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	const source = `package main

import (
	"fmt"
	"io"
	"os"

	"github.com/antlr4-go/antlr/v4"
	generated "genanalex/internal/compiscript/frontend/generated"
)

type errors struct { *antlr.DefaultErrorListener; count int }
func (e *errors) SyntaxError(antlr.Recognizer, interface{}, int, int, string, antlr.RecognitionException) { e.count++ }

func main() {
	input, _ := io.ReadAll(os.Stdin)
	errors := &errors{DefaultErrorListener: antlr.NewDefaultErrorListener()}
	lexer := generated.NewCompiscriptLexer(antlr.NewInputStream(string(input)))
	lexer.RemoveErrorListeners(); lexer.AddErrorListener(errors)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := generated.NewCompiscriptParser(tokens)
	parser.RemoveErrorListeners(); parser.AddErrorListener(errors); parser.Program()
	tokens.Fill(); matches := 0
	for _, token := range tokens.GetAllTokens() { if token.GetText() == os.Args[1] { matches++ } }
	if errors.count != 0 || (os.Args[1] != "" && matches != 1) { fmt.Fprintln(os.Stderr, errors.count, matches); os.Exit(1) }
}`
	path := filepath.Join(directory, "main.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return "./" + filepath.Base(directory)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("go.mod not found")
		}
		directory = parent
	}
}
