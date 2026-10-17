package frontend

import (
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func TestFunctionWithoutBodyRecovery(t *testing.T) {
	const source = "function f()"
	program, diagnostics := Parse([]byte(source))
	want := ast.Program{
		Span:       testSpan(source, 0, len(source)),
		Statements: ast.Statements{ast.BadStmt{Span: testSpan(source, 0, len(source))}},
	}
	if !reflect.DeepEqual(program, want) {
		t.Fatalf("program = %#v, want %#v", program, want)
	}
	wantDiagnostics := model.Diagnostics{{
		Code: "syntax", Phase: model.PhaseSyntax,
		Message: "mismatched input '<EOF>' expecting {'{', ':'}",
		Span:    testSpan(source, len(source), len(source)),
	}}
	if !reflect.DeepEqual(diagnostics, wantDiagnostics) {
		t.Fatalf("diagnostics = %#v, want %#v", diagnostics, wantDiagnostics)
	}
}

func TestRecoveryInputMatrix(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		diagnostics  int
	}{
		{"if missing brace", "if (true) { print(1);", 2},
		{"if valid", "if (true) { print(1); }", 0},
		{"print missing parenthesis", "print(1;", 1},
		{"print valid", "print(1);", 0},
		{"if missing condition", "if () {}", 1},
		{"if empty block valid", "if (true) {}", 0},
		{"binary missing operand", "print(1 + );", 1},
		{"binary valid", "print(1 + 2);", 0},
		{"missing initializer", "let x = ;", 1},
		{"initializer valid", "let x = 1;", 0},
		{"assignment missing value", "let x = 1; x = ;", 1},
		{"assignment valid", "let x = 1; x = 2;", 0},
		{"while missing body", "while (true)", 1},
		{"while missing brace", "while (true) {", 1},
		{"while valid", "while (true) {}", 0},
		{"for missing body", "for (;;)", 1},
		{"for valid omitted expressions", "for (;;) {}", 0},
		{"do missing body", "do", 1},
		{"do missing condition", "do {} while ();", 1},
		{"do valid", "do {} while (false);", 0},
		{"foreach missing body", "foreach (x in [1])", 1},
		{"foreach valid", "foreach (x in [1]) {}", 0},
		{"function missing body", "function f()", 1},
		{"function valid", "function f() {}", 0},
		{"typed function missing body", "function f(x: integer): integer", 1},
		{"nested function missing body", "function outer() { function inner() }", 1},
		{"method missing body", "class C { function f() }", 1},
		{"typed function valid", "function f(x: integer): integer { return x; }", 0},
		{"nested function valid", "function outer() { function inner() {} }", 0},
		{"method valid", "class C { function f() {} }", 0},
		{"for valid omitted condition", "for (let i = 0;; i = i + 1) {}", 0},
		{"for valid omitted post", "for (let i = 0; i < 2;) {}", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			before := append([]byte(nil), source...)
			program, diagnostics := Parse(source)
			if !reflect.DeepEqual(source, before) {
				t.Fatal("Parse modified the source")
			}
			if len(diagnostics) != tt.diagnostics {
				t.Fatalf("diagnostics = %#v, want %d", diagnostics, tt.diagnostics)
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Code != "syntax" || diagnostic.Phase != model.PhaseSyntax || diagnostic.Message == "" ||
					diagnostic.Span.Start.Offset < 0 || diagnostic.Span.End.Offset < diagnostic.Span.Start.Offset ||
					diagnostic.Span.End.Offset > len(source) {
					t.Fatalf("invalid syntax diagnostic: %#v", diagnostic)
				}
			}
			if tt.diagnostics == 0 {
				if len(program.Statements) == 0 {
					t.Fatal("valid input produced no statements")
				}
				for _, statement := range program.Statements {
					if _, bad := statement.(ast.BadStmt); bad {
						t.Fatalf("valid input produced a bad statement: %#v", statement)
					}
				}
			}
		})
	}
}
